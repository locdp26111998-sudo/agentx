package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const contextGoClawAgentKey = "agentx_goclaw_agent_key"

// ContextWithGoClawAgentKey gắn agent_key GoClaw vào context.
func ContextWithGoClawAgentKey(ctx context.Context, agentKey string) context.Context {
	return context.WithValue(ctx, contextGoClawAgentKey, agentKey)
}

func goclawAgentKeyFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextGoClawAgentKey).(string)
	return v
}

// GoClawConfig kết nối GoClaw HTTP API.
type GoClawConfig struct {
	BaseURL string
	APIKey  string
	UserID  string
}

// GoClaw adapter mỏng qua OpenAI-compatible API.
type GoClaw struct {
	Config     GoClawConfig
	RunTimeout time.Duration
	client     *http.Client
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (g *GoClaw) httpClient() *http.Client {
	if g.client != nil {
		return g.client
	}
	return http.DefaultClient
}

func (g *GoClaw) runTimeout() time.Duration {
	if g.RunTimeout > 0 {
		return g.RunTimeout
	}
	return DefaultRunTimeout
}

// Run gọi POST {base_url}/v1/chat/completions, model=goclaw:{agent_key}.
func (g *GoClaw) Run(ctx context.Context, userPrompt, systemPrompt string) (Reply, error) {
	agentKey := strings.TrimSpace(goclawAgentKeyFromContext(ctx))
	if agentKey == "" {
		return Reply{}, newRunError(KindParse, "thiếu goclaw agent_key", fmt.Errorf("agent_key rỗng"))
	}

	start := time.Now()
	reply, err := g.run(ctx, agentKey, userPrompt, systemPrompt)
	elapsed := time.Since(start)

	senderID := senderIDFromContext(ctx)
	if err != nil {
		log.Printf("engine goclaw: lỗi sau %v sender_id=%q agent_key=%q: %v",
			elapsed.Round(time.Millisecond), senderID, agentKey, err)
		return Reply{}, err
	}

	log.Printf("engine goclaw: ok sau %v sender_id=%q agent_key=%q",
		elapsed.Round(time.Millisecond), senderID, agentKey)
	return reply, nil
}

func (g *GoClaw) run(ctx context.Context, agentKey, userPrompt, systemPrompt string) (Reply, error) {
	ctx, cancel := context.WithTimeout(ctx, g.runTimeout())
	defer cancel()

	messages := make([]chatMessage, 0, 2)
	if s := strings.TrimSpace(systemPrompt); s != "" {
		messages = append(messages, chatMessage{Role: "system", Content: s})
	}
	messages = append(messages, chatMessage{Role: "user", Content: userPrompt})

	body := chatRequest{
		Model:    "goclaw:" + agentKey,
		Messages: messages,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return Reply{}, newRunError(KindParse, "không tạo được request GoClaw", err)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(g.Config.BaseURL), "/")
	url := baseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return Reply{}, newRunError(KindExit, "gọi GoClaw thất bại", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.Config.APIKey)
	req.Header.Set("X-GoClaw-User-Id", g.Config.UserID)

	resp, err := g.httpClient().Do(req)
	if err != nil {
		if isTimeout(ctx, err) {
			return Reply{}, newRunError(KindTimeout, "phản hồi quá lâu", err)
		}
		return Reply{}, newRunError(KindExit, "GoClaw không phản hồi", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Reply{}, newRunError(KindParse, "không đọc được kết quả từ GoClaw", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("GoClaw trả lỗi %d", resp.StatusCode)
		if s := extractGoClawError(respBody); s != "" {
			msg = s
		}
		return Reply{}, newRunError(KindExit, msg, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody))))
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Reply{}, newRunError(KindParse, "không đọc được kết quả từ GoClaw", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return Reply{}, newRunError(KindExit, parsed.Error.Message, fmt.Errorf("goclaw error"))
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return Reply{}, newRunError(KindParse, "GoClaw trả về rỗng", fmt.Errorf("choices rỗng"))
	}

	return Reply{Text: parsed.Choices[0].Message.Content}, nil
}

func extractGoClawError(body []byte) string {
	var obj struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &obj); err != nil {
		return ""
	}
	return strings.TrimSpace(obj.Error.Message)
}

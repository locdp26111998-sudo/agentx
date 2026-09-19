package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"agentx/internal/store"
)

const contextSenderIDKey = "agentx_sender_id"

// ContextWithSenderID gắn sender_id vào context trước khi gọi Run.
func ContextWithSenderID(ctx context.Context, senderID string) context.Context {
	return context.WithValue(ctx, contextSenderIDKey, senderID)
}

func senderIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextSenderIDKey).(string)
	return v
}

// ClaudeCode gọi Claude Code headless (không dùng --bare).
type ClaudeCode struct {
	Command    string
	RunTimeout time.Duration
	Store      store.SessionStore // nil → fallback map trong RAM (test)

	mu       sync.Mutex
	sessions map[string]string // fallback khi Store == nil
}

type claudeResponse struct {
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
}

func (c *ClaudeCode) command() string {
	if c.Command != "" {
		return c.Command
	}
	return "claude"
}

func (c *ClaudeCode) runTimeout() time.Duration {
	if c.RunTimeout > 0 {
		return c.RunTimeout
	}
	return DefaultRunTimeout
}

func (c *ClaudeCode) getSession(senderID string) string {
	if c.Store != nil {
		sessionID, err := c.Store.GetSession(senderID)
		if err != nil {
			log.Printf("engine claude: GetSession sender_id=%q: %v (coi như lượt mới)", senderID, err)
			return ""
		}
		return sessionID
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sessions == nil {
		return ""
	}
	return c.sessions[senderID]
}

func (c *ClaudeCode) persistSession(senderID, sessionID string) {
	if c.Store != nil {
		if err := c.Store.SaveSession(senderID, sessionID); err != nil {
			log.Printf("engine claude: SaveSession sender_id=%q: %v", senderID, err)
		}
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sessions == nil {
		c.sessions = make(map[string]string)
	}
	c.sessions[senderID] = sessionID
}

func (c *ClaudeCode) Run(ctx context.Context, userPrompt, systemPrompt string) (Reply, error) {
	senderID := senderIDFromContext(ctx)
	start := time.Now()
	reply, err := c.run(ctx, senderID, userPrompt, systemPrompt)
	elapsed := time.Since(start)

	sessionID := c.getSession(senderID)
	if err != nil {
		log.Printf("engine claude: lỗi sau %v sender_id=%q session_id=%q: %v",
			elapsed.Round(time.Millisecond), senderID, sessionID, err)
		return Reply{}, err
	}

	log.Printf("engine claude: ok sau %v sender_id=%q session_id=%q",
		elapsed.Round(time.Millisecond), senderID, sessionID)
	return reply, nil
}

func (c *ClaudeCode) run(ctx context.Context, senderID, userPrompt, systemPrompt string) (Reply, error) {
	ctx, cancel := context.WithTimeout(ctx, c.runTimeout())
	defer cancel()

	sessionID := c.getSession(senderID)
	if sessionID != "" {
		log.Printf("engine claude: resume sender_id=%q session_id=%q", senderID, sessionID)
	} else {
		log.Printf("engine claude: lượt mới sender_id=%q (chưa có session)", senderID)
	}

	args := c.buildArgs(userPrompt, systemPrompt, sessionID)
	cmd := exec.CommandContext(ctx, c.command(), args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if isTimeout(ctx, err) {
			return Reply{}, newRunError(KindTimeout, "phản hồi quá lâu", err)
		}
		if isCommandNotFound(err) {
			return Reply{}, newRunError(KindNotFound, "không tìm thấy lệnh claude", err)
		}
		msg := "Claude Code thoát với lỗi"
		if s := strings.TrimSpace(stderr.String()); s != "" {
			msg = fmt.Sprintf("Claude Code thoát với lỗi: %s", truncate(s, 300))
		}
		return Reply{}, newRunError(KindExit, msg, err)
	}

	out := bytes.TrimSpace(stdout.Bytes())
	if len(out) == 0 {
		return Reply{}, newRunError(KindParse, "không đọc được kết quả từ Claude Code",
			fmt.Errorf("stdout rỗng"))
	}

	var resp claudeResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return Reply{}, newRunError(KindParse, "không đọc được kết quả từ Claude Code",
			fmt.Errorf("parse JSON: %w", err))
	}

	if resp.Result == "" {
		return Reply{}, newRunError(KindParse, "không đọc được kết quả từ Claude Code",
			fmt.Errorf("JSON thiếu field result"))
	}

	if resp.SessionID != "" {
		c.persistSession(senderID, resp.SessionID)
	}

	return Reply{Text: resp.Result, SessionID: c.getSession(senderID)}, nil
}

func (c *ClaudeCode) buildArgs(userPrompt, systemPrompt, sessionID string) []string {
	args := []string{"-p", userPrompt}
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	} else if systemPrompt != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}
	args = append(args, "--output-format", "json")
	return args
}

func isCommandNotFound(err error) bool {
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "executable file not found") ||
		strings.Contains(s, "not found in %PATH%")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

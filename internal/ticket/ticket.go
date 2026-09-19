package ticket

import (
	"fmt"
	"os"
	"strings"

	"agentx/internal/guard"

	"gopkg.in/yaml.v3"
)

// Prompt tách kênh: hồn/kiến thức vào system, câu hỏi khách vào user.
type Prompt struct {
	SystemPrompt string
	UserPrompt   string
}

// MessengerChannel cấu hình page Messenger gắn agent.
type MessengerChannel struct {
	PageID string `yaml:"page_id"`
}

// ZaloChannel cấu hình OA Zalo gắn agent.
type ZaloChannel struct {
	OAID string `yaml:"oa_id"`
}

// Channels cấu hình kênh nhận tin của agent.
type Channels struct {
	Messenger []MessengerChannel `yaml:"messenger"`
	Zalo      []ZaloChannel      `yaml:"zalo"`
}

type guardYAML struct {
	MaxReplyLength int      `yaml:"max_reply_length"`
	ForbiddenWords []string `yaml:"forbidden_words"`
}

type messengerCredentialsYAML struct {
	PageAccessToken string `yaml:"page_access_token"`
	VerifyToken     string `yaml:"verify_token"`
}

type agentConfig struct {
	Name           string                   `yaml:"name"`
	Engine         string                   `yaml:"engine,omitempty"`
	GoClawAgentKey string                   `yaml:"goclaw_agent_key,omitempty"`
	Channels       Channels                 `yaml:"channels"`
	Soul      string                   `yaml:"soul"`
	Knowledge string                   `yaml:"knowledge"`
	Guard     guardYAML                `yaml:"guard"`
	Messenger messengerCredentialsYAML `yaml:"messenger"`
}

// MessengerSettings credential gửi tin Messenger (page_id lấy từ routing).
type MessengerSettings struct {
	PageAccessToken string
	VerifyToken     string
}

// Ticket soạn prompt từ config agent.
type Ticket struct {
	cfg agentConfig
}

// Load đọc file YAML agent.
func Load(path string) (*Ticket, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("đọc config: %w", err)
	}

	var cfg agentConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	return &Ticket{cfg: cfg}, nil
}

// Name trả tên hiển thị agent.
func (t *Ticket) Name() string {
	return t.cfg.Name
}

// Engine trả loại động cơ (claude-code, goclaw, ...).
func (t *Ticket) Engine() string {
	if e := strings.TrimSpace(t.cfg.Engine); e != "" {
		return e
	}
	return "claude-code"
}

// GoClawAgentKey trả agent_key trên GoClaw khi engine=goclaw.
func (t *Ticket) GoClawAgentKey() string {
	return strings.TrimSpace(t.cfg.GoClawAgentKey)
}

// MessengerChannels trả danh sách page_id Messenger.
func (t *Ticket) MessengerChannels() []MessengerChannel {
	return t.cfg.Channels.Messenger
}

// ZaloChannels trả danh sách oa_id Zalo.
func (t *Ticket) ZaloChannels() []ZaloChannel {
	return t.cfg.Channels.Zalo
}

// Compose ghép hồn + kiến thức (system) và câu hỏi khách (user).
func (t *Ticket) Compose(userText string) Prompt {
	var parts []string
	if s := strings.TrimSpace(t.cfg.Soul); s != "" {
		parts = append(parts, s)
	}
	if k := strings.TrimSpace(t.cfg.Knowledge); k != "" {
		parts = append(parts, "Kiến thức:\n"+k)
	}

	return Prompt{
		SystemPrompt: strings.Join(parts, "\n\n"),
		UserPrompt:   userText,
	}
}

// MessengerSettings trả credential Messenger từ YAML agent.
func (t *Ticket) MessengerSettings() MessengerSettings {
	return MessengerSettings{
		PageAccessToken: t.cfg.Messenger.PageAccessToken,
		VerifyToken:     t.cfg.Messenger.VerifyToken,
	}
}

// GuardConfig trả cấu hình guard từ YAML agent.
func (t *Ticket) GuardConfig() guard.AgentConfig {
	return guard.AgentConfig{
		MaxReplyLength: t.cfg.Guard.MaxReplyLength,
		ForbiddenWords: t.cfg.Guard.ForbiddenWords,
	}
}

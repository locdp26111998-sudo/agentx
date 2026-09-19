package admin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agentx/internal/store"

	"gopkg.in/yaml.v3"
)

const defaultEngine = "claude-code"

// AgentSummary thông tin agent cho danh sách admin.
type AgentSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Engine       string `json:"engine"`
	Channel      string `json:"channel"`
	Status       string `json:"status"`
	SessionCount int    `json:"session_count"`
}

// GuardConfig cấu hình guard trong form admin.
type GuardConfig struct {
	MaxReplyLength int      `json:"max_reply_length" yaml:"max_reply_length"`
	ForbiddenWords []string `json:"forbidden_words" yaml:"forbidden_words"`
}

// MessengerChannelItem page_id trong channels.messenger.
type MessengerChannelItem struct {
	PageID string `json:"page_id" yaml:"page_id"`
}

// ZaloChannelItem oa_id trong channels.zalo.
type ZaloChannelItem struct {
	OAID string `json:"oa_id" yaml:"oa_id"`
}

// ChannelsConfig cấu hình kênh routing.
type ChannelsConfig struct {
	Messenger []MessengerChannelItem `json:"messenger" yaml:"messenger"`
	Zalo      []ZaloChannelItem      `json:"zalo" yaml:"zalo"`
}

// MessengerCredentials token gửi tin Messenger.
type MessengerCredentials struct {
	PageAccessToken string `json:"page_access_token" yaml:"page_access_token"`
	VerifyToken     string `json:"verify_token" yaml:"verify_token"`
}

// AgentDetail chi tiết agent để xem/sửa.
type AgentDetail struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Engine    string               `json:"engine"`
	Channels  ChannelsConfig       `json:"channels"`
	Soul      string               `json:"soul"`
	Knowledge string               `json:"knowledge"`
	Guard     GuardConfig          `json:"guard"`
	Messenger MessengerCredentials `json:"messenger"`
}

type agentYAML struct {
	Name      string               `yaml:"name"`
	Engine    string               `yaml:"engine,omitempty"`
	Channels  ChannelsConfig       `yaml:"channels"`
	Soul      string               `yaml:"soul"`
	Knowledge string               `yaml:"knowledge"`
	Guard     GuardConfig          `yaml:"guard"`
	Messenger MessengerCredentials `yaml:"messenger"`
}

// AgentStore đọc/ghi file YAML agent trong configs/agents.
type AgentStore struct {
	Dir string
}

// List liệt kê agent từ thư mục config; status theo session SQLite.
func (s *AgentStore) List(st store.SessionStore) ([]AgentSummary, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("đọc thư mục agents: %w", err)
	}

	sessionCount := 0
	if counter, ok := st.(interface{ SessionCount() (int, error) }); ok {
		sessionCount, err = counter.SessionCount()
		if err != nil {
			return nil, fmt.Errorf("đếm session: %w", err)
		}
	}

	var out []AgentSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".yaml")
		cfg, err := s.loadFile(filepath.Join(s.Dir, e.Name()))
		if err != nil {
			return nil, err
		}

		engineName := cfg.Engine
		if engineName == "" {
			engineName = defaultEngine
		}

		status := "offline"
		if sessionCount > 0 {
			status = "online"
		}

		out = append(out, AgentSummary{
			ID:           id,
			Name:         cfg.Name,
			Engine:       engineName,
			Channel:      detectChannel(cfg),
			Status:       status,
			SessionCount: sessionCount,
		})
	}
	return out, nil
}

// Get đọc chi tiết agent theo id (tên file không .yaml).
func (s *AgentStore) Get(id string) (*AgentDetail, error) {
	if err := validateAgentID(id); err != nil {
		return nil, err
	}
	path := filepath.Join(s.Dir, id+".yaml")
	cfg, err := s.loadFile(path)
	if err != nil {
		return nil, err
	}
	return cfg.toDetail(id), nil
}

// Save ghi đè file YAML agent.
func (s *AgentStore) Save(id string, detail AgentDetail) error {
	if err := validateAgentID(id); err != nil {
		return err
	}
	path := filepath.Join(s.Dir, id+".yaml")

	engineName := strings.TrimSpace(detail.Engine)
	if engineName == "" {
		engineName = defaultEngine
	}

	cfg := agentYAML{
		Name:      strings.TrimSpace(detail.Name),
		Engine:    engineName,
		Channels:  detail.Channels,
		Soul:      detail.Soul,
		Knowledge: detail.Knowledge,
		Guard:     detail.Guard,
		Messenger: detail.Messenger,
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("ghi file: %w", err)
	}
	return nil
}

func (s *AgentStore) loadFile(path string) (*agentYAML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("đọc %s: %w", path, err)
	}
	var cfg agentYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse YAML %s: %w", path, err)
	}
	return &cfg, nil
}

func (c *agentYAML) toDetail(id string) *AgentDetail {
	engineName := c.Engine
	if engineName == "" {
		engineName = defaultEngine
	}
	return &AgentDetail{
		ID:        id,
		Name:      c.Name,
		Engine:    engineName,
		Channels:  c.Channels,
		Soul:      c.Soul,
		Knowledge: c.Knowledge,
		Guard:     c.Guard,
		Messenger: c.Messenger,
	}
}

func detectChannel(cfg *agentYAML) string {
	var parts []string
	if len(cfg.Channels.Messenger) > 0 {
		parts = append(parts, "messenger")
	}
	if len(cfg.Channels.Zalo) > 0 {
		parts = append(parts, "zalo")
	}
	if len(parts) == 0 {
		return "webhook"
	}
	return strings.Join(parts, ", ")
}

func validateAgentID(id string) error {
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("id agent không hợp lệ")
	}
	return nil
}

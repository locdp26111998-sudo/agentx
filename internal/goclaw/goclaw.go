package goclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultTimeout = 15 * time.Second

// Config kết nối GoClaw API.
type Config struct {
	BaseURL string
	APIKey  string
	UserID  string
}

type configFile struct {
	GoClaw struct {
		BaseURL string `yaml:"base_url"`
		APIKey  string `yaml:"api_key"`
		UserID  string `yaml:"user_id"`
	} `yaml:"goclaw"`
}

// AgentSummary agent GoClaw trả về dashboard (chỉ đọc).
type AgentSummary struct {
	DisplayName string `json:"display_name"`
	Model       string `json:"model"`
	Status      string `json:"status"`
	Frontmatter string `json:"frontmatter"`
}

type agentsResponse struct {
	Agents []rawAgent `json:"agents"`
}

type rawAgent struct {
	DisplayName string `json:"display_name"`
	Model       string `json:"model"`
	Status      string `json:"status"`
	Frontmatter string `json:"frontmatter"`
}

// Client gọi GoClaw HTTP API.
type Client struct {
	cfg    Config
	client *http.Client
}

// LoadConfig đọc configs/goclaw.yaml.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("đọc config GoClaw: %w", err)
	}

	var f configFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return Config{}, fmt.Errorf("parse YAML GoClaw: %w", err)
	}

	cfg := Config{
		BaseURL: strings.TrimRight(strings.TrimSpace(f.GoClaw.BaseURL), "/"),
		APIKey:  strings.TrimSpace(f.GoClaw.APIKey),
		UserID:  strings.TrimSpace(f.GoClaw.UserID),
	}
	if cfg.BaseURL == "" {
		return Config{}, fmt.Errorf("thiếu goclaw.base_url")
	}
	if cfg.APIKey == "" {
		return Config{}, fmt.Errorf("thiếu goclaw.api_key")
	}
	if cfg.UserID == "" {
		return Config{}, fmt.Errorf("thiếu goclaw.user_id")
	}
	return cfg, nil
}

// NewClient tạo client GoClaw.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// ListAgents gọi GET {base_url}/v1/agents.
func (c *Client) ListAgents(ctx context.Context) ([]AgentSummary, error) {
	url := c.cfg.BaseURL + "/v1/agents"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("tạo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("X-GoClaw-User-Id", c.cfg.UserID)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GoClaw không phản hồi: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("đọc response GoClaw: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GoClaw trả lỗi %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed agentsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse JSON GoClaw: %w", err)
	}

	out := make([]AgentSummary, 0, len(parsed.Agents))
	for _, a := range parsed.Agents {
		out = append(out, AgentSummary{
			DisplayName: a.DisplayName,
			Model:       a.Model,
			Status:      a.Status,
			Frontmatter: a.Frontmatter,
		})
	}
	return out, nil
}

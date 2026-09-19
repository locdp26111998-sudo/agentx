package goclaw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestListAgents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-GoClaw-User-Id") != "system" {
			http.Error(w, `{"error":"missing user id"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"agents":[{"display_name":"Ho tro","model":"gpt-4o-mini","status":"active","frontmatter":"Mo ta ngan"}]}`))
	}))
	defer srv.Close()

	client := NewClient(Config{
		BaseURL: srv.URL,
		APIKey:  "test-key",
		UserID:  "system",
	})

	agents, err := client.ListAgents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 {
		t.Fatalf("agents=%d", len(agents))
	}
	if agents[0].DisplayName != "Ho tro" || agents[0].Model != "gpt-4o-mini" {
		t.Fatalf("unexpected agent: %+v", agents[0])
	}
}

func TestListAgentsNoResponse(t *testing.T) {
	client := NewClient(Config{
		BaseURL: "http://127.0.0.1:1",
		APIKey:  "test-key",
		UserID:  "system",
	})
	_, err := client.ListAgents(context.Background())
	if err == nil {
		t.Fatal("expected error when GoClaw unreachable")
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "goclaw.yaml")
	content := `
goclaw:
  base_url: "https://example.com"
  api_key: "key-123"
  user_id: "system"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://example.com" || cfg.APIKey != "key-123" {
		t.Fatalf("cfg=%+v", cfg)
	}
}

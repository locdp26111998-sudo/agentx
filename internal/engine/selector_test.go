package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"agentx/internal/ticket"
)

func TestSelectorClaudeVsGoClaw(t *testing.T) {
	dir := t.TempDir()
	writeAgentYAML(t, dir, "a.yaml", "name: A\nengine: claude-code\nsoul: s\n")
	writeAgentYAML(t, dir, "b.yaml", "name: B\nengine: goclaw\ngoclaw_agent_key: ho-tro\nsoul: s\n")

	tktClaude, _ := ticket.Load(filepath.Join(dir, "a.yaml"))
	tktGoClaw, _ := ticket.Load(filepath.Join(dir, "b.yaml"))

	if tktClaude.Engine() != "claude-code" {
		t.Fatalf("engine=%q", tktClaude.Engine())
	}
	if tktGoClaw.GoClawAgentKey() != "ho-tro" {
		t.Fatalf("key=%q", tktGoClaw.GoClawAgentKey())
	}

	goclawSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer goclawSrv.Close()

	sel := &Selector{
		Claude: &ClaudeCode{Command: "claude-khong-ton-tai-12345"},
		GoClaw: &GoClaw{
			Config: GoClawConfig{BaseURL: goclawSrv.URL, APIKey: "test-key", UserID: "system"},
			client: goclawSrv.Client(),
		},
	}

	ctx := ContextWithGoClawAgentKey(context.Background(), "ho-tro")
	reply, err := sel.Run(ctx, tktGoClaw, "xin chao", "system")
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text == "" {
		t.Fatal("empty reply")
	}
}

func writeAgentYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

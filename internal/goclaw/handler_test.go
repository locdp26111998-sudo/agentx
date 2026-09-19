package goclaw

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandlerAgents(t *testing.T) {
	goclawSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"agents":[{"display_name":"Ho tro","model":"gpt-4o-mini","status":"active","frontmatter":"Mo ta"}]}`))
	}))
	defer goclawSrv.Close()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "goclaw.yaml")
	content := "goclaw:\n  base_url: \"" + goclawSrv.URL + "\"\n  api_key: \"key\"\n  user_id: \"system\"\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	h := &Handler{ConfigPath: cfgPath}
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/goclaw/agents", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Agents []AgentSummary `json:"agents"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Agents) != 1 || resp.Agents[0].Model != "gpt-4o-mini" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestHandlerAgentsUnreachable(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "goclaw.yaml")
	content := "goclaw:\n  base_url: \"http://127.0.0.1:1\"\n  api_key: \"key\"\n  user_id: \"system\"\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	h := &Handler{ConfigPath: cfgPath}
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/goclaw/agents", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d want 502", rec.Code)
	}
}

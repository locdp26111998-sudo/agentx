package router_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"agentx/internal/engine"
	"agentx/internal/guard"
	"agentx/internal/router"
)

func TestWebhookRoutingByPageID(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "spa-hoa.yaml", `
name: Spa Hoa
channels:
  messenger:
    - page_id: "111111111"
soul: Ban la Hoa
knowledge: Gio mo cua 8h
guard:
  max_reply_length: 1000
messenger:
  verify_token: test
`)
	writeYAML(t, dir, "phong-kham-minh.yaml", `
name: Phong kham Minh
channels:
  messenger:
    - page_id: "222222222"
soul: Ban la Minh
knowledge: Gio kham 7h30
guard:
  max_reply_length: 1000
messenger:
  verify_token: test
`)

	rt := router.New(dir)
	eng := &engine.ClaudeCode{Command: "claude-khong-ton-tai-12345"}
	grd := guard.New()

	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		handleTestWebhook(w, r, rt, eng, grd)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cases := []struct {
		pageID    string
		wantAgent string
		wantCode  int
	}{
		{"111111111", "spa-hoa", http.StatusInternalServerError},
		{"222222222", "phong-kham-minh", http.StatusInternalServerError},
		{"999999999", "", http.StatusNotFound},
	}

	for _, tc := range cases {
		body, _ := json.Marshal(map[string]string{
			"sender_id": "khach-1",
			"text":      "xin chao",
			"page_id":   tc.pageID,
		})
		resp, err := http.Post(srv.URL+"/webhook", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != tc.wantCode {
			t.Errorf("page_id=%s: status=%d body=%s", tc.pageID, resp.StatusCode, data)
		}

		logs := logBuf.String()
		if tc.wantAgent != "" {
			want := `→ agent="` + tc.wantAgent + `"`
			if !bytes.Contains([]byte(logs), []byte(want)) {
				t.Errorf("page_id=%s: log thiếu %q, logs=%q", tc.pageID, want, logs)
			}
		} else {
			if !bytes.Contains([]byte(logs), []byte("cảnh báo — không tìm thấy agent")) {
				t.Errorf("page_id=%s: thiếu log cảnh báo, logs=%q", tc.pageID, logs)
			}
		}
	}
}

func handleTestWebhook(w http.ResponseWriter, r *http.Request, rt *router.Router, eng *engine.ClaudeCode, grd guard.Guard) {
	var req struct {
		SenderID string `json:"sender_id"`
		Text     string `json:"text"`
		PageID   string `json:"page_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ag, ok := rt.Resolve(router.ChannelMessenger, req.PageID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_, _ = eng.Run(r.Context(), req.Text, ag.Ticket.Compose(req.Text).SystemPrompt)
	w.WriteHeader(http.StatusInternalServerError)
}

func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

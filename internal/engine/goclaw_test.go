package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoClawRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-GoClaw-User-Id") != "system" {
			http.Error(w, "missing user", http.StatusBadRequest)
			return
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Model != "goclaw:ho-tro" {
			http.Error(w, "wrong model", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Xin chao tu GoClaw"}}]}`))
	}))
	defer srv.Close()

	eng := &GoClaw{
		Config: GoClawConfig{
			BaseURL: srv.URL,
			APIKey:  "test-key",
			UserID:  "system",
		},
		client: srv.Client(),
	}

	ctx := ContextWithGoClawAgentKey(context.Background(), "ho-tro")
	reply, err := eng.Run(ctx, "xin chao", "ban la tro ly")
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text != "Xin chao tu GoClaw" {
		t.Fatalf("reply=%q", reply.Text)
	}
}

func TestGoClawMissingAgentKey(t *testing.T) {
	eng := &GoClaw{Config: GoClawConfig{BaseURL: "http://example.com", APIKey: "k", UserID: "system"}}
	_, err := eng.Run(context.Background(), "hi", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

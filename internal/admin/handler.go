package admin

import (
	"encoding/json"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"agentx/internal/store"
)

//go:embed web/*
var webFS embed.FS

// Handler gom API admin + static frontend.
type Handler struct {
	Agents   *AgentStore
	Sessions store.SessionStore
	Logs     *LogHub
	Auth     *Auth
}

// Register gắn route /admin và /api/admin/* lên mux.
func (h *Handler) Register(mux *http.ServeMux) {
	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("admin static: %v", err)
	}
	fileServer := http.FileServer(http.FS(webRoot))

	mux.HandleFunc("/admin/login", h.handleLoginPage)
	mux.HandleFunc("/api/admin/login", h.handleLoginAPI)
	mux.HandleFunc("/api/admin/logout", h.handleLogoutAPI)

	mux.HandleFunc("/admin", h.serveAdminIndex)
	mux.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/login" {
			h.handleLoginPage(w, r)
			return
		}
		if hasStaticExtension(r.URL.Path) {
			http.StripPrefix("/admin/", fileServer).ServeHTTP(w, r)
			return
		}
		h.serveAdminIndex(w, r)
	})

	mux.HandleFunc("/api/admin/agents", h.guardAdmin(h.handleAgents))
	mux.HandleFunc("/api/admin/agents/", h.guardAdmin(h.handleAgentByID))
	mux.HandleFunc("/api/admin/logs/stream", h.guardAdmin(h.handleLogStream))
}

func (h *Handler) guardAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.requireAdmin(w, r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "chưa đăng nhập"})
			return
		}
		next(w, r)
	}
}

func (h *Handler) serveAdminIndex(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}
	data, err := webFS.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "admin UI không tìm thấy", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func hasStaticExtension(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".js") ||
		strings.HasSuffix(lower, ".css") ||
		strings.HasSuffix(lower, ".ico") ||
		strings.HasSuffix(lower, ".png") ||
		strings.HasSuffix(lower, ".svg")
}

func (h *Handler) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "chỉ GET"})
		return
	}
	list, err := h.Agents.List(h.Sessions)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) handleAgentByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/agents/")
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "thiếu id agent"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		detail, err := h.Agents.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, detail)
	case http.MethodPut:
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "đọc body thất bại"})
			return
		}
		var detail AgentDetail
		if err := json.Unmarshal(body, &detail); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON không hợp lệ"})
			return
		}
		detail.ID = id
		if err := h.Agents.Save(id, detail); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		log.Printf("admin: đã lưu config agent %q", id)
		writeJSON(w, http.StatusOK, map[string]string{"ok": "đã lưu"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "chỉ GET hoặc PUT"})
	}
}

func (h *Handler) handleLogStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "chỉ GET", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming không hỗ trợ", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	history, live, unsubscribe := h.Logs.Subscribe()
	defer unsubscribe()

	for _, line := range history {
		fmt.Fprintf(w, "data: %s\n\n", sseEscape(line))
	}
	flusher.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-live:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", sseEscape(string(line)))
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func sseEscape(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

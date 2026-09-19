package goclaw

import (
	"encoding/json"
	"log"
	"net/http"
)

// Handler proxy danh sách agent GoClaw cho frontend admin.
type Handler struct {
	ConfigPath string
}

// Register gắn GET /api/goclaw/agents.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/goclaw/agents", h.handleAgents)
}

func (h *Handler) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "chỉ GET"})
		return
	}

	cfg, err := LoadConfig(h.ConfigPath)
	if err != nil {
		log.Printf("goclaw: config: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	client := NewClient(cfg)
	agents, err := client.ListAgents(r.Context())
	if err != nil {
		log.Printf("goclaw: list agents: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"agents": agents})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"agentx/internal/admin"
	"agentx/internal/engine"
	"agentx/internal/goclaw"
	"agentx/internal/guard"
	"agentx/internal/inbox"
	"agentx/internal/router"
	"agentx/internal/store"
)

const guardClientError = "agent không thể trả lời lúc này"

const (
	agentsDir      = "configs/agents"
	goclawConfig   = "configs/goclaw.yaml"
	dbPath         = "data/agentx.db"
)

type webhookRequest struct {
	SenderID string `json:"sender_id"`
	Text     string `json:"text"`
	PageID   string `json:"page_id"`
}

type webhookResponse struct {
	Reply   string `json:"reply"`
	AgentID string `json:"agent_id,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	logHub := admin.NewLogHub()
	admin.AttachLog(logHub)

	rt := router.New(agentsDir)

	st, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	claudeEng := newEngineFromEnv()
	claudeEng.Store = st

	gcCfg, err := goclaw.LoadConfig(goclawConfig)
	if err != nil {
		log.Fatalf("goclaw config: %v", err)
	}
	engSel := &engine.Selector{
		Claude: claudeEng,
		GoClaw: &engine.GoClaw{
			Config: engine.GoClawConfig{
				BaseURL: gcCfg.BaseURL,
				APIKey:  gcCfg.APIKey,
				UserID:  gcCfg.UserID,
			},
		},
	}

	grd := guard.New()

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		handleWebhook(w, r, rt, engSel, grd)
	})
	mux.HandleFunc("/webhook/messenger", inbox.MessengerHandler(rt, engSel, grd))

	adminHandler := &admin.Handler{
		Agents:   &admin.AgentStore{Dir: agentsDir},
		Sessions: st,
		Logs:     logHub,
	}
	adminHandler.Register(mux)

	goclawHandler := &goclaw.Handler{ConfigPath: goclawConfig}
	goclawHandler.Register(mux)

	addr := ":8090"
	log.Printf("agentx webhook lắng nghe http://127.0.0.1%s/webhook", addr)
	log.Printf("agentx messenger webhook http://127.0.0.1%s/webhook/messenger", addr)
	log.Printf("agentx admin http://127.0.0.1%s/admin", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func handleWebhook(w http.ResponseWriter, r *http.Request, rt *router.Router, engSel *engine.Selector, grd guard.Guard) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "chỉ chấp nhận POST"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "đọc body thất bại"})
		return
	}

	var req webhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "body JSON không hợp lệ"})
		return
	}
	if req.SenderID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "thiếu field sender_id"})
		return
	}
	if req.Text == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "thiếu field text"})
		return
	}
	if req.PageID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "thiếu field page_id"})
		return
	}

	ag, ok := rt.Resolve(router.ChannelMessenger, req.PageID)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "không tìm thấy agent cho page_id"})
		return
	}

	tkt := ag.Ticket
	p := tkt.Compose(req.Text)
	ctx := engine.ContextWithSenderID(r.Context(), req.SenderID)
	reply, err := engSel.Run(ctx, tkt, p.UserPrompt, p.SystemPrompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: engine.ClientMessage(err)})
		return
	}

	checked, err := grd.Check(reply.Text, tkt.GuardConfig())
	if err != nil {
		log.Printf("guard: agent=%q: %v", ag.ID, err)
		writeJSON(w, http.StatusOK, errorResponse{Error: guardClientError})
		return
	}

	writeJSON(w, http.StatusOK, webhookResponse{Reply: checked, AgentID: ag.ID})
}

// newEngineFromEnv tạo engine; biến môi trường chỉ để test nghiệm thu (không bắt buộc).
// AGENTX_ENGINE_TIMEOUT=2s  — test timeout
// AGENTX_ENGINE_COMMAND=claude-sai-ten — test lệnh không tồn tại
func newEngineFromEnv() *engine.ClaudeCode {
	eng := &engine.ClaudeCode{}
	if v := os.Getenv("AGENTX_ENGINE_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("AGENTX_ENGINE_TIMEOUT không hợp lệ %q: %v", v, err)
		}
		eng.RunTimeout = d
		log.Printf("engine test: timeout=%v", d)
	}
	if v := os.Getenv("AGENTX_ENGINE_COMMAND"); v != "" {
		eng.Command = v
		log.Printf("engine test: command=%q", v)
	}
	return eng
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

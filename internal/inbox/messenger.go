package inbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"agentx/internal/engine"
	"agentx/internal/guard"
	"agentx/internal/router"
)

// MessengerConfig credential gửi tin qua Graph API.
type MessengerConfig struct {
	PageAccessToken string
	PageID          string
}

type MessengerPayload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID        string      `json:"id"`
	Time      int64       `json:"time"`
	Messaging []Messaging `json:"messaging"`
}

type Messaging struct {
	Sender    MSUser `json:"sender"`
	Recipient MSUser `json:"recipient"`
	Timestamp int64  `json:"timestamp"`
	Message   *MSMsg `json:"message"`
}

type MSUser struct {
	ID string `json:"id"`
}

type MSMsg struct {
	MID    string `json:"mid"`
	Text   string `json:"text"`
	IsEcho bool   `json:"is_echo"`
}

type MSSendRequest struct {
	Recipient     MSRecipient `json:"recipient"`
	MessagingType string      `json:"messaging_type"`
	Message         MSText      `json:"message"`
}

type MSRecipient struct {
	ID string `json:"id"`
}

type MSText struct {
	Text string `json:"text"`
}

// MessengerHandler xử lý verify (GET) và tin nhắn (POST) từ Meta.
func MessengerHandler(rt *router.Router, engSel *engine.Selector, grd guard.Guard) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleMessengerVerify(w, r, rt)
		case http.MethodPost:
			handleMessengerPost(w, r, rt, engSel, grd)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleMessengerVerify(w http.ResponseWriter, r *http.Request, rt *router.Router) {
	q := r.URL.Query()
	token := q.Get("hub.verify_token")
	if q.Get("hub.mode") == "subscribe" && rt.VerifyToken(token) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(q.Get("hub.challenge")))
		return
	}
	http.Error(w, "forbidden", http.StatusForbidden)
}

func handleMessengerPost(w http.ResponseWriter, r *http.Request, rt *router.Router, engSel *engine.Selector, grd guard.Guard) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	log.Printf("messenger: POST body=%s", string(body))

	var payload MessengerPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	log.Printf("messenger: object=%q entries=%d", payload.Object, len(payload.Entry))

	for _, entry := range payload.Entry {
		pageID := entry.ID
		for _, msg := range entry.Messaging {
			log.Printf("messenger: page_id=%s sender=%s msg=%+v", pageID, msg.Sender.ID, msg.Message)
			if msg.Message == nil || msg.Message.IsEcho || msg.Message.Text == "" {
				continue
			}
			go processMessengerMessage(rt, engSel, grd, pageID, msg.Sender.ID, msg.Message.Text)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func processMessengerMessage(rt *router.Router, engSel *engine.Selector, grd guard.Guard, pageID, senderID, text string) {
	ag, ok := rt.Resolve(router.ChannelMessenger, pageID)
	if !ok {
		return
	}

	tkt := ag.Ticket
	ms := tkt.MessengerSettings()
	cfg := MessengerConfig{
		PageAccessToken: ms.PageAccessToken,
		PageID:          pageID,
	}

	p := tkt.Compose(text)
	ctx := engine.ContextWithSenderID(context.Background(), senderID)

	reply, err := engSel.Run(ctx, tkt, p.UserPrompt, p.SystemPrompt)
	if err != nil {
		log.Printf("messenger: engine agent=%q sender_id=%q: %v", ag.ID, senderID, err)
		return
	}

	checked, err := grd.Check(reply.Text, tkt.GuardConfig())
	if err != nil {
		log.Printf("messenger: guard agent=%q sender_id=%q: %v", ag.ID, senderID, err)
		return
	}

	if err := sendReply(cfg, senderID, checked); err != nil {
		log.Printf("messenger: sendReply agent=%q sender_id=%q: %v", ag.ID, senderID, err)
	}
}

func sendReply(cfg MessengerConfig, recipientID, text string) error {
	apiURL := fmt.Sprintf(
		"https://graph.facebook.com/v25.0/%s/messages?access_token=%s",
		url.PathEscape(cfg.PageID),
		url.QueryEscape(cfg.PageAccessToken),
	)

	reqBody := MSSendRequest{
		Recipient:     MSRecipient{ID: recipientID},
		MessagingType: "RESPONSE",
		Message:       MSText{Text: text},
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal send body: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("tạo request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("gọi Send API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Send API status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

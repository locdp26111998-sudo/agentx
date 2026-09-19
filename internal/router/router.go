package router

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"agentx/internal/ticket"
)

// Channel loại kênh nhận tin.
type Channel string

const (
	ChannelMessenger Channel = "messenger"
	ChannelZalo      Channel = "zalo"
)

// Agent agent đã load kèm id file config.
type Agent struct {
	ID     string
	Ticket *ticket.Ticket
}

// Router map kênh + ID → agent; quét configs/agents/*.yaml mỗi lần resolve.
type Router struct {
	dir string
}

// New tạo router trỏ tới thư mục config agent.
func New(dir string) *Router {
	return &Router{dir: dir}
}

// Resolve tìm agent theo kênh và ID (page_id hoặc oa_id).
func (r *Router) Resolve(channel Channel, id string) (*Agent, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("router: cảnh báo — thiếu ID kênh %s", channel)
		return nil, false
	}

	agents, messengerIdx, zaloIdx, err := r.scan()
	if err != nil {
		log.Printf("router: lỗi quét config: %v", err)
		return nil, false
	}

	var agentID string
	var ok bool
	switch channel {
	case ChannelMessenger:
		agentID, ok = messengerIdx[id]
	case ChannelZalo:
		agentID, ok = zaloIdx[id]
	default:
		log.Printf("router: cảnh báo — kênh không hỗ trợ %q", channel)
		return nil, false
	}

	if !ok {
		log.Printf("router: cảnh báo — không tìm thấy agent cho kênh=%s id=%s", channel, id)
		return nil, false
	}

	ag := agents[agentID]
	log.Printf("router: kênh=%s id=%s → agent=%q (%s)", channel, id, agentID, ag.Ticket.Name())
	return ag, true
}

// VerifyToken kiểm tra token Meta webhook có khớp agent nào không.
func (r *Router) VerifyToken(token string) bool {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		tkt, err := ticket.Load(filepath.Join(r.dir, e.Name()))
		if err != nil {
			continue
		}
		ms := tkt.MessengerSettings()
		if ms.VerifyToken != "" && ms.VerifyToken == token {
			return true
		}
	}
	return false
}

func (r *Router) scan() (agents map[string]*Agent, messengerIdx, zaloIdx map[string]string, err error) {
	agents = make(map[string]*Agent)
	messengerIdx = make(map[string]string)
	zaloIdx = make(map[string]string)

	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("đọc thư mục agents: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".yaml")
		path := filepath.Join(r.dir, e.Name())

		tkt, err := ticket.Load(path)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("load %s: %w", path, err)
		}

		agents[id] = &Agent{ID: id, Ticket: tkt}

		for _, ch := range tkt.MessengerChannels() {
			pageID := strings.TrimSpace(ch.PageID)
			if pageID == "" {
				continue
			}
			if prev, dup := messengerIdx[pageID]; dup {
				log.Printf("router: cảnh báo — page_id=%s trùng agent %q và %q", pageID, prev, id)
			}
			messengerIdx[pageID] = id
		}

		for _, ch := range tkt.ZaloChannels() {
			oaID := strings.TrimSpace(ch.OAID)
			if oaID == "" {
				continue
			}
			if prev, dup := zaloIdx[oaID]; dup {
				log.Printf("router: cảnh báo — oa_id=%s trùng agent %q và %q", oaID, prev, id)
			}
			zaloIdx[oaID] = id
		}
	}

	return agents, messengerIdx, zaloIdx, nil
}

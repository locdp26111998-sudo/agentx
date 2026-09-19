package admin

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"
)

const logHistorySize = 200

// LogHub phát log real-time cho admin (SSE).
type LogHub struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
	history     [][]byte
}

// NewLogHub tạo hub log; gọi AttachLog để gắn vào stdlib log.
func NewLogHub() *LogHub {
	return &LogHub{
		subscribers: make(map[chan []byte]struct{}),
	}
}

// AttachLog chuyển log package sang ghi cả stderr và hub.
func AttachLog(hub *LogHub) {
	log.SetOutput(io.MultiWriter(os.Stderr, hub))
}

func (h *LogHub) Write(p []byte) (int, error) {
	line := append([]byte(nil), p...)
	h.broadcast(line)
	return len(p), nil
}

func (h *LogHub) broadcast(line []byte) {
	h.mu.Lock()
	h.history = append(h.history, line)
	if len(h.history) > logHistorySize {
		h.history = h.history[len(h.history)-logHistorySize:]
	}
	subs := make([]chan []byte, 0, len(h.subscribers))
	for ch := range h.subscribers {
		subs = append(subs, ch)
	}
	h.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- line:
		default:
			// bỏ qua nếu client chậm
		}
	}
}

// Subscribe nhận log mới; trả history gần nhất + channel live.
func (h *LogHub) Subscribe() (history []string, live <-chan []byte, unsubscribe func()) {
	ch := make(chan []byte, 64)

	h.mu.Lock()
	for _, line := range h.history {
		history = append(history, string(bytes.TrimRight(line, "\r\n")))
	}
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()

	return history, ch, func() {
		h.mu.Lock()
		delete(h.subscribers, ch)
		close(ch)
		h.mu.Unlock()
	}
}

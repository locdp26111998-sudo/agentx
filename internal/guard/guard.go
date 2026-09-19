package guard

import (
	"encoding/json"
	"errors"
	"strings"
)

var (
	ErrEmptyReply       = errors.New("guard: reply rỗng")
	ErrCLIError         = errors.New("guard: CLI trả lỗi")
	ErrTooLong          = errors.New("guard: reply quá dài")
	ErrForbiddenContent = errors.New("guard: nội dung bị chặn")
)

// AgentConfig cấu hình guard theo agent.
type AgentConfig struct {
	MaxReplyLength int
	ForbiddenWords []string
}

// Guard soát reply trước khi trả khách.
type Guard interface {
	Check(reply string, cfg AgentConfig) (string, error)
}

type basicGuard struct{}

// New tạo guard mặc định.
func New() Guard {
	return &basicGuard{}
}

func (g *basicGuard) Check(reply string, cfg AgentConfig) (string, error) {
	// Lớp 1 — lỗi kỹ thuật
	if strings.TrimSpace(reply) == "" {
		return "", ErrEmptyReply
	}
	if isCLIErrorReply(reply) {
		return "", ErrCLIError
	}

	// Lớp 2 — quá dài
	if cfg.MaxReplyLength > 0 && len(reply) > cfg.MaxReplyLength {
		return "", ErrTooLong
	}

	// Lớp 3 — từ cấm
	if hasForbiddenWord(reply, cfg.ForbiddenWords) {
		return "", ErrForbiddenContent
	}

	return reply, nil
}

func isCLIErrorReply(reply string) bool {
	trimmed := strings.TrimSpace(reply)
	if !strings.HasPrefix(trimmed, "{") {
		return false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return false
	}
	_, ok := obj["error"]
	return ok
}

func hasForbiddenWord(reply string, words []string) bool {
	lower := strings.ToLower(reply)
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

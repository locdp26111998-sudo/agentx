package engine

import (
	"context"
	"time"
)

// DefaultRunTimeout là giới hạn chờ Claude Code trả lời mỗi lần gọi.
const DefaultRunTimeout = 90 * time.Second

// Reply là kết quả từ động cơ; SessionID dùng cho --resume ở GĐ2.
type Reply struct {
	Text      string
	SessionID string
}

// Engine gọi động cơ agent (CLI hoặc adapter khác).
type Engine interface {
	Run(ctx context.Context, userPrompt, systemPrompt string) (Reply, error)
}

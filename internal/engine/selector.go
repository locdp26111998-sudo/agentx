package engine

import (
	"context"
	"strings"

	"agentx/internal/ticket"
)

// Selector chọn engine theo field engine trong YAML agent.
type Selector struct {
	Claude *ClaudeCode
	GoClaw *GoClaw
}

// Run gọi engine phù hợp với ticket.
func (s *Selector) Run(ctx context.Context, tkt *ticket.Ticket, userPrompt, systemPrompt string) (Reply, error) {
	switch strings.ToLower(strings.TrimSpace(tkt.Engine())) {
	case "goclaw":
		key := tkt.GoClawAgentKey()
		ctx = ContextWithGoClawAgentKey(ctx, key)
		return s.GoClaw.Run(ctx, userPrompt, systemPrompt)
	default:
		return s.Claude.Run(ctx, userPrompt, systemPrompt)
	}
}

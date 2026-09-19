package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRun_CommandNotFound(t *testing.T) {
	c := &ClaudeCode{Command: "claude-xyz-khong-ton-tai", RunTimeout: 5 * time.Second}
	_, err := c.Run(context.Background(), "xin chao", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if ClientMessage(err) != "không tìm thấy lệnh claude" {
		t.Fatalf("client message = %q", ClientMessage(err))
	}
	var re *RunError
	if !errors.As(err, &re) || re.Kind != KindNotFound {
		t.Fatalf("kind = %v", err)
	}
}

func TestClientMessage_Timeout(t *testing.T) {
	err := newRunError(KindTimeout, "phản hồi quá lâu", context.DeadlineExceeded)
	if ClientMessage(err) != "phản hồi quá lâu" {
		t.Fatalf("got %q", ClientMessage(err))
	}
}

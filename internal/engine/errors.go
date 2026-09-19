package engine

import (
	"context"
	"errors"
	"fmt"
)

// Kind phân loại lỗi engine trả về.
type Kind int

const (
	KindTimeout  Kind = iota // (a) quá giờ
	KindExit                 // (b) claude thoát mã lỗi
	KindParse                // (c) không parse được output
	KindNotFound             // (d) không tìm thấy lệnh claude
)

// RunError mô tả lỗi có phân loại; Message dùng trả client.
type RunError struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *RunError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *RunError) Unwrap() error { return e.Cause }

func newRunError(kind Kind, message string, cause error) error {
	return &RunError{Kind: kind, Message: message, Cause: cause}
}

// ClientMessage trả thông báo gọn cho client; không panic nếu err nil hoặc lạ.
func ClientMessage(err error) string {
	if err == nil {
		return ""
	}
	var re *RunError
	if errors.As(err, &re) {
		return re.Message
	}
	return "gọi engine thất bại"
}

func isTimeout(ctx context.Context, err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return true
	}
	return false
}

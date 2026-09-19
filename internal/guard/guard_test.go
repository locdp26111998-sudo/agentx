package guard

import (
	"errors"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	g := New()
	cfg := AgentConfig{
		MaxReplyLength: 1000,
		ForbiddenWords: []string{"giết", "tự tử", "bom"},
	}

	long1200 := strings.Repeat("a", 1200)
	long5000 := strings.Repeat("b", 5000)

	tests := []struct {
		name    string
		reply   string
		cfg     AgentConfig
		want    string
		wantErr error
	}{
		{
			name:    "1 reply rỗng",
			reply:   "",
			cfg:     cfg,
			wantErr: ErrEmptyReply,
		},
		{
			name:    "2 reply chỉ space",
			reply:   "   ",
			cfg:     cfg,
			wantErr: ErrEmptyReply,
		},
		{
			name:    "3 CLI trả lỗi JSON",
			reply:   `{"error":"timeout"}`,
			cfg:     cfg,
			wantErr: ErrCLIError,
		},
		{
			name:    "4 reply bình thường",
			reply:   "Dạ em có thể giúp gì?",
			cfg:     cfg,
			want:    "Dạ em có thể giúp gì?",
		},
		{
			name:    "5 reply quá dài",
			reply:   long1200,
			cfg:     cfg,
			wantErr: ErrTooLong,
		},
		{
			name:    "6 chứa từ cấm",
			reply:   "...bom hẹn giờ...",
			cfg:     cfg,
			wantErr: ErrForbiddenContent,
		},
		{
			name:    "7 từ cấm viết hoa",
			reply:   "BOM",
			cfg:     cfg,
			wantErr: ErrForbiddenContent,
		},
		{
			name:    "8 MaxReplyLength = 0 không giới hạn",
			reply:   long5000,
			cfg:     AgentConfig{MaxReplyLength: 0, ForbiddenWords: cfg.ForbiddenWords},
			want:    long5000,
		},
		{
			name:    "9 ForbiddenWords rỗng",
			reply:   "Dạ em có thể giúp gì?",
			cfg:     AgentConfig{MaxReplyLength: 1000, ForbiddenWords: nil},
			want:    "Dạ em có thể giúp gì?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := g.Check(tt.reply, tt.cfg)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Check() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Check() unexpected err = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Check() = %q, want %q", got, tt.want)
			}
		})
	}
}

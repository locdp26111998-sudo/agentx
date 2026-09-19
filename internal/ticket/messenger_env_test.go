package ticket

import (
	"testing"
)

func TestResolveMessengerSettingsEnvFirst(t *testing.T) {
	t.Setenv(EnvMessengerVerifyToken, "env-verify")
	t.Setenv(PageAccessTokenEnvKey("demo"), "env-page-token")

	ms := ResolveMessengerSettings("demo", "YOUR_PAGE_ACCESS_TOKEN", "YOUR_VERIFY_TOKEN")
	if ms.VerifyToken != "env-verify" {
		t.Fatalf("verify: got %q", ms.VerifyToken)
	}
	if ms.PageAccessToken != "env-page-token" {
		t.Fatalf("page token: got %q", ms.PageAccessToken)
	}
}

func TestResolveMessengerSettingsYAMLFallback(t *testing.T) {
	t.Setenv(EnvMessengerVerifyToken, "")
	t.Setenv(PageAccessTokenEnvKey("spa-hoa"), "")

	ms := ResolveMessengerSettings("spa-hoa", "real-page", "real-verify")
	if ms.VerifyToken != "real-verify" || ms.PageAccessToken != "real-page" {
		t.Fatalf("yaml fallback: %+v", ms)
	}
}

func TestResolveMessengerSettingsPlaceholderIgnored(t *testing.T) {
	t.Setenv(EnvMessengerVerifyToken, "")
	t.Setenv(PageAccessTokenEnvKey("demo"), "")

	ms := ResolveMessengerSettings("demo", "YOUR_PAGE_ACCESS_TOKEN", "agentx-secret-2024")
	if ms.PageAccessToken != "" {
		t.Fatalf("placeholder page token should be empty, got %q", ms.PageAccessToken)
	}
	if ms.VerifyToken != "agentx-secret-2024" {
		t.Fatalf("verify: got %q", ms.VerifyToken)
	}
}

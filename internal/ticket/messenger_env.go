package ticket

import (
	"os"
	"strings"
)

const EnvMessengerVerifyToken = "MESSENGER_VERIFY_TOKEN"

// PageAccessTokenEnvKey tên biến env page token theo id agent (demo → AGENT_DEMO_PAGE_ACCESS_TOKEN).
func PageAccessTokenEnvKey(agentID string) string {
	id := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(agentID), "-", "_"))
	return "AGENT_" + id + "_PAGE_ACCESS_TOKEN"
}

// ResolveMessengerSettings ưu tiên .env; YAML chỉ dùng khi không placeholder và env trống.
func ResolveMessengerSettings(agentID, yamlPageToken, yamlVerifyToken string) MessengerSettings {
	verify := strings.TrimSpace(os.Getenv(EnvMessengerVerifyToken))
	if verify == "" && !isPlaceholderCredential(yamlVerifyToken) {
		verify = strings.TrimSpace(yamlVerifyToken)
	}

	page := strings.TrimSpace(os.Getenv(PageAccessTokenEnvKey(agentID)))
	if page == "" && !isPlaceholderCredential(yamlPageToken) {
		page = strings.TrimSpace(yamlPageToken)
	}

	return MessengerSettings{
		PageAccessToken: page,
		VerifyToken:     verify,
	}
}

func isPlaceholderCredential(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "YOUR_") {
		return true
	}
	if strings.HasPrefix(strings.ToLower(s), "fake-token") {
		return true
	}
	return false
}

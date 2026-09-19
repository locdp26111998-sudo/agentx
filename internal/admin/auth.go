package admin

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	envAdminUser = "ADMIN_USERNAME"
	envAdminPass = "ADMIN_PASSWORD"
	cookieName   = "agentx_admin_session"
	sessionTTL   = 7 * 24 * time.Hour
)

// Auth session cookie cho /admin (bật khi có đủ user + pass trong env).
type Auth struct {
	username string
	password string
	enabled  bool

	mu       sync.Mutex
	sessions map[string]time.Time // token → hết hạn
}

// NewAuthFromEnv đọc ADMIN_USERNAME / ADMIN_PASSWORD.
func NewAuthFromEnv() *Auth {
	user := strings.TrimSpace(os.Getenv(envAdminUser))
	pass := os.Getenv(envAdminPass)
	a := &Auth{
		username: user,
		password: pass,
		sessions: make(map[string]time.Time),
	}
	a.enabled = user != "" && pass != ""
	return a
}

// Enabled trả true khi bắt buộc đăng nhập.
func (a *Auth) Enabled() bool {
	return a != nil && a.enabled
}

func (a *Auth) validSession(r *http.Request) bool {
	if !a.Enabled() {
		return true
	}
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[c.Value]
	if !ok || time.Now().After(exp) {
		delete(a.sessions, c.Value)
		return false
	}
	return true
}

func (a *Auth) setSessionCookie(w http.ResponseWriter, r *http.Request) {
	token, err := newSessionToken()
	if err != nil {
		http.Error(w, "không tạo được session", http.StatusInternalServerError)
		return
	}
	a.mu.Lock()
	a.sessions[token] = time.Now().Add(sessionTTL)
	a.pruneLocked()
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cookieSecure(r),
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func (a *Auth) clearSession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		a.mu.Lock()
		delete(a.sessions, c.Value)
		a.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cookieSecure(r),
		MaxAge:   -1,
	})
}

func (a *Auth) pruneLocked() {
	now := time.Now()
	for tok, exp := range a.sessions {
		if now.After(exp) {
			delete(a.sessions, tok)
		}
	}
}

func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func cookieSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); strings.EqualFold(proto, "https") {
		return true
	}
	return false
}

func credentialsMatch(gotUser, gotPass, wantUser, wantPass string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(gotUser), []byte(wantUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(gotPass), []byte(wantPass)) == 1
	return userOK && passOK
}

func (h *Handler) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Auth != nil && h.Auth.validSession(r) {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	data, err := webFS.ReadFile("web/login.html")
	if err != nil {
		http.Error(w, "login UI không tìm thấy", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (h *Handler) handleLoginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "chỉ POST"})
		return
	}
	if h.Auth == nil || !h.Auth.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "admin auth chưa cấu hình"})
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "đọc body thất bại"})
		return
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON không hợp lệ"})
		return
	}

	if !credentialsMatch(body.Username, body.Password, h.Auth.username, h.Auth.password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Sai tên đăng nhập hoặc mật khẩu"})
		return
	}

	h.Auth.setSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "đăng nhập thành công"})
}

func (h *Handler) handleLogoutAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "chỉ POST"})
		return
	}
	if h.Auth != nil && h.Auth.Enabled() {
		h.Auth.clearSession(w, r)
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "đã đăng xuất"})
}

func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if h.Auth == nil || !h.Auth.Enabled() {
		return true
	}
	if h.Auth.validSession(r) {
		return true
	}
	return false
}

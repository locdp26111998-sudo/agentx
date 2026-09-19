package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAdminAuthLoginFlow(t *testing.T) {
	t.Setenv(envAdminUser, "testuser")
	t.Setenv(envAdminPass, "testpass")

	auth := NewAuthFromEnv()
	if !auth.Enabled() {
		t.Fatal("auth phải bật")
	}

	h := &Handler{Auth: auth, Agents: &AgentStore{Dir: t.TempDir()}}
	mux := http.NewServeMux()
	h.Register(mux)

	// Chưa login → /admin redirect login
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/admin/login" {
		t.Fatalf("GET /admin: status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}

	// Sai password
	body, _ := json.Marshal(map[string]string{"username": "testuser", "password": "wrong"})
	req = httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("login sai: status=%d", rec.Code)
	}

	// Đúng password
	body, _ = json.Marshal(map[string]string{"username": "testuser", "password": "testpass"})
	req = httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login đúng: status=%d body=%s", rec.Code, rec.Body.String())
	}
	cookie := rec.Result().Cookies()
	var session *http.Cookie
	for _, c := range cookie {
		if c.Name == cookieName {
			session = c
			break
		}
	}
	if session == nil || session.Value == "" {
		t.Fatal("thiếu cookie session")
	}

	// Có session → vào /admin
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin sau login: status=%d", rec.Code)
	}

	// Logout
	req = httptest.NewRequest(http.MethodPost, "/api/admin/logout", nil)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout: status=%d", rec.Code)
	}

	// Sau logout không vào API
	req = httptest.NewRequest(http.MethodGet, "/api/admin/agents", nil)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("API sau logout: status=%d", rec.Code)
	}
}

func TestAdminAuthDisabledWhenEnvEmpty(t *testing.T) {
	os.Unsetenv(envAdminUser)
	os.Unsetenv(envAdminPass)
	auth := NewAuthFromEnv()
	if auth.Enabled() {
		t.Fatal("auth phải tắt khi thiếu env")
	}
}

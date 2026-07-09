package console

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoginSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	sm := NewSessionManager(db, 8*3600)
	r := gin.New()
	r.POST("/api/console/login", LoginHandler("secret123", sm))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/console/login", nil)
	req.Header.Set("X-Admin-Token", "secret123")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var env struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if env.Code != 0 {
		t.Fatalf("envelope code=%d", env.Code)
	}
	if _, ok := env.Data["csrf_token"].(string); !ok {
		t.Fatalf("missing or non-string csrf_token: %v", env.Data["csrf_token"])
	}
	if sid, ok := env.Data["session_id"].(string); !ok || sid == "" {
		t.Fatalf("missing or empty session_id: %v", env.Data["session_id"])
	}
	if a, ok := env.Data["admin_id"].(string); !ok || a != "admin" {
		t.Fatalf("wrong admin_id: %v", env.Data["admin_id"])
	}

	cookies := w.Result().Cookies()
	var sessionCookie, csrfCookie *http.Cookie
	for _, ck := range cookies {
		switch ck.Name {
		case CookieSession:
			sessionCookie = ck
		case CookieCSRF:
			csrfCookie = ck
		}
	}
	if sessionCookie == nil {
		t.Fatalf("session cookie not set; cookies=%v", cookies)
	}
	if !sessionCookie.HttpOnly {
		t.Fatalf("%s must be HttpOnly", CookieSession)
	}
	if csrfCookie == nil {
		t.Fatalf("csrf cookie not set; cookies=%v", cookies)
	}
	if csrfCookie.HttpOnly {
		t.Fatalf("%s must NOT be HttpOnly", CookieCSRF)
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session SameSite=%v want Strict", sessionCookie.SameSite)
	}
	if csrfCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("csrf SameSite=%v want Strict", csrfCookie.SameSite)
	}
	if sessionCookie.MaxAge != 8*3600 {
		t.Fatalf("session MaxAge=%d want %d", sessionCookie.MaxAge, 8*3600)
	}
	if csrfCookie.MaxAge != 8*3600 {
		t.Fatalf("csrf MaxAge=%d want %d", csrfCookie.MaxAge, 8*3600)
	}
	if sessionCookie.Path != "/" || csrfCookie.Path != "/" {
		t.Fatalf("cookie path wrong: session=%s csrf=%s", sessionCookie.Path, csrfCookie.Path)
	}
}

func TestLoginBadToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	sm := NewSessionManager(db, 8*3600)
	r := gin.New()
	r.POST("/api/console/login", LoginHandler("secret123", sm))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/console/login", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token: status=%d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/console/login", nil)
	req2.Header.Set("X-Admin-Token", "wrong-token")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("bad token: status=%d body=%s", w2.Code, w2.Body.String())
	}
}

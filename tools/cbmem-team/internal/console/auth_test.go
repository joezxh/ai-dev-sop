package console

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAdminTokenMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	v := NewAdminVerifier("secret123")
	r := gin.New()
	r.Use(v.Middleware())
	r.GET("/", func(c *gin.Context) { OK(c, nil) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Admin-Token", "secret123")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != 401 {
		t.Fatalf("status=%d", w2.Code)
	}
}

func TestSessionLifecycle(t *testing.T) {
	db := newTestDB(t)
	sm := NewSessionManager(db, 8*time.Hour)
	r := gin.New()
	r.GET("/login", func(c *gin.Context) {
		sid, err := sm.Create(c.Request.Context(), "admin", "127.0.0.1", "UA")
		if err != nil {
			t.Fatal(err)
		}
		csrf := "csrf-token"
		c.SetCookie(CookieSession, sid, 3600*8, "/", "", false, true)
		c.SetCookie(CookieCSRF, csrf, 3600*8, "/", "", false, false)
		OK(c, gin.H{"csrf": csrf})
	})
	r.GET("/me", RequireSession(sm), func(c *gin.Context) {
		OK(c, gin.H{"uid": c.GetString("admin_id")})
	})
	r.POST("/logout", RequireSession(sm), func(c *gin.Context) {
		id, _ := c.Cookie(CookieSession)
		_ = sm.Drop(c.Request.Context(), id)
		c.SetCookie(CookieSession, "", -1, "/", "", false, true)
		OK(c, nil)
	})

	// 登录
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/login", nil))
	if w.Code != 200 {
		t.Fatalf("login=%d", w.Code)
	}

	// 凭 cookie 访 me
	cookies := w.Result().Cookies()
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/me", nil)
	for _, ck := range cookies {
		req2.AddCookie(ck)
	}
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("me=%d", w2.Code)
	}

	// 退出
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("POST", "/logout", nil)
	for _, ck := range cookies {
		req3.AddCookie(ck)
	}
	r.ServeHTTP(w3, req3)
	if w3.Code != 200 {
		t.Fatalf("logout=%d", w3.Code)
	}

	// 再访 me 应 401（cookie 已清空）
	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest("GET", "/me", nil)
	r.ServeHTTP(w4, req4)
	if w4.Code != 401 {
		t.Fatalf("me after logout=%d", w4.Code)
	}
}

func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(context.TODO()); err != nil {
		t.Fatal(err)
	}
	return db
}

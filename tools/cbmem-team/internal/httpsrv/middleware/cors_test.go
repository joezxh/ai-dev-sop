package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newRouter(origins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(origins))
	r.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
	return r
}

func TestCORS_AllowsWhitelistedOrigin(t *testing.T) {
	r := newRouter([]string{"https://docs.fin-ai.net"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://docs.fin-ai.net")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://docs.fin-ai.net" {
		t.Errorf("want ACAO=https://docs.fin-ai.net, got %q", got)
	}
	if w.Code != 200 {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestCORS_RejectsUnlistedOrigin(t *testing.T) {
	r := newRouter([]string{"https://docs.fin-ai.net"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("want empty ACAO, got %q", got)
	}
}

func TestCORS_WildcardAllowsAnyOrigin(t *testing.T) {
	r := newRouter([]string{"*"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://anything.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://anything.example" {
		t.Errorf("want ACAO=*, got %q", got)
	}
}

func TestCORS_OptionsPreflightReturns204(t *testing.T) {
	r := newRouter([]string{"https://docs.fin-ai.net"})
	req := httptest.NewRequest("OPTIONS", "/ping", nil)
	req.Header.Set("Origin", "https://docs.fin-ai.net")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("want 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("want ACAM header set")
	}
}

func TestCORS_SetsVaryHeader(t *testing.T) {
	r := newRouter([]string{"https://docs.fin-ai.net"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://docs.fin-ai.net")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Vary"); got != "Origin" {
		t.Errorf("want Vary=Origin, got %q", got)
	}
}

package console

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRBACRoleDefaultsToAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		if RoleOf(c) != RoleAdmin {
			t.Errorf("expected default role admin, got %s", RoleOf(c))
		}
		c.Status(200)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestRBACSetAndRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		SetRole(c, RoleViewer)
		c.JSON(200, gin.H{"role": RoleOf(c)})
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if !contains(w.Body.String(), "viewer") {
		t.Fatalf("body missing role: %s", w.Body.String())
	}
}

func TestRBACRequireRoleBlocksNonAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", func(c *gin.Context) {
		SetRole(c, RoleViewer)
		c.Next()
	}, RequireRole(RoleAdmin), func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	if !contains(w.Body.String(), "4030002") {
		t.Fatalf("expected error code 4030002 in body, got %s", w.Body.String())
	}
}

func TestRBACRequireRoleAllowsMatching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ops", func(c *gin.Context) {
		SetRole(c, RoleOperator)
		c.Next()
	}, RequireRole(RoleAdmin, RoleOperator), func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ops", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHasRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		SetRole(c, RoleOperator)
		if !HasRole(c, RoleAdmin, RoleOperator) {
			t.Errorf("HasRole(operator, admin, operator) should be true")
		}
		if HasRole(c, RoleAdmin) {
			t.Errorf("HasRole(operator, admin) should be false")
		}
		c.Status(200)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
}

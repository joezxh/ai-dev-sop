package console

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOKFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		OK(c, gin.H{"foo": 1})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env["code"].(float64) != 0 {
		t.Fatalf("code = %v", env["code"])
	}
	if env["data"].(map[string]any)["foo"].(float64) != 1 {
		t.Fatalf("data wrong: %v", env["data"])
	}
}

func TestFailFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		Fail(c, 401, 4010001, "missing token")
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 401 {
		t.Fatalf("status = %d", w.Code)
	}
	var env map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env["code"].(float64) != 4010001 || env["msg"] != "missing token" {
		t.Fatalf("payload: %v", env)
	}
}

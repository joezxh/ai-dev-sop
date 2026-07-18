package console

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

const captureMessageCallBody = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"messages_create","arguments":{"messages":[{"role":"user","content":"hi"}]}}}`

const captureNonMessageCallBody = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"/tmp/x"}}}`

func TestCaptureSimpleMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	r := gin.New()
	// CaptureSessions runs BEFORE the handler (it must read the body and
	// restore it before the downstream handler tries to read it). The handler
	// here only needs to set user_id (the JWT-equivalent middleware normally
	// does this in main.go) and emit a 200.
	r.POST("/mcp",
		func(c *gin.Context) {
			c.Set("user_id", "alice")
			c.JSON(200, gin.H{"jsonrpc": "2.0", "result": "ok"})
		},
		CaptureSessions(CaptureConfig{DB: db}),
	)

	req := httptest.NewRequest("POST", "/mcp?project=/tmp/test", bytes.NewBufferString(captureMessageCallBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	// capture runs in a goroutine; poll the table.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		_ = db.QueryRowContext(context.TODO(), `SELECT COUNT(*) FROM session_turns`).Scan(&n)
		if n > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	var cnt int
	if err := db.QueryRowContext(context.TODO(), `SELECT COUNT(*) FROM session_turns`).Scan(&cnt); err != nil {
		t.Fatalf("count turns: %v", err)
	}
	if cnt == 0 {
		t.Fatalf("no turn captured")
	}

	var sid, role, content string
	if err := db.QueryRowContext(context.TODO(),
		`SELECT session_id, role, content FROM session_turns ORDER BY id DESC LIMIT 1`,
	).Scan(&sid, &role, &content); err != nil {
		t.Fatalf("scan turn: %v", err)
	}
	if sid == "" {
		t.Fatalf("sid empty")
	}
	if !bytes.HasPrefix([]byte(sid), []byte("sess_alice_")) {
		t.Fatalf("sid=%q want prefix sess_alice_", sid)
	}
	if role != "user" {
		t.Fatalf("role=%q want user", role)
	}
	if content != "hi" {
		t.Fatalf("content=%q want hi", content)
	}
}

func TestCaptureNonMessageBypasses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	r := gin.New()
	r.POST("/mcp",
		func(c *gin.Context) {
			c.Set("user_id", "alice")
			c.JSON(200, gin.H{"jsonrpc": "2.0", "result": "ok"})
		},
		CaptureSessions(CaptureConfig{DB: db}),
	)

	req := httptest.NewRequest("POST", "/mcp?project=/tmp/test", bytes.NewBufferString(captureNonMessageCallBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}

	// give the goroutine ample time to (incorrectly) write
	time.Sleep(200 * time.Millisecond)

	var n int
	if err := db.QueryRowContext(context.TODO(), `SELECT COUNT(*) FROM session_turns`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected no captured turns, got %d", n)
	}
}
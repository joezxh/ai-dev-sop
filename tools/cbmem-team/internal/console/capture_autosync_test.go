package console

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/llm"
)

// mempalaceStub is a tiny in-process MemPalace compatible with
// llm.MemPalace.AddDrawer. It records every Drawer payload so tests can
// assert dedup, ordering, and content.
type mempalaceStub struct {
	mu     sync.Mutex
	hits   []map[string]string
	status int // HTTP status to return; defaults to 200
	failN  int32
}

func newMemPalaceStub() *mempalaceStub {
	return &mempalaceStub{status: 200}
}

func (s *mempalaceStub) record(payload map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hits = append(s.hits, payload)
}

func (s *mempalaceStub) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.hits)
}

func (s *mempalaceStub) start() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a transient failure for the first failN requests so
		// the retry path can be exercised.
		if atomic.LoadInt32(&s.failN) > 0 {
			atomic.AddInt32(&s.failN, -1)
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"error":"transient"}`))
			return
		}
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		s.record(map[string]string{
			"path":    r.URL.Path,
			"body":    string(buf),
		})
		w.WriteHeader(s.status)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
}

// startMemPalace spins up a stub that satisfies the llm.MemPalace HTTP
// contract (/api/drawers POST + /healthz GET) and returns the configured
// client + a teardown func.
func startMemPalace(t *testing.T, stub *mempalaceStub) (*llm.MemPalace, func()) {
	t.Helper()
	srv := stub.start()
	mp := llm.NewMemPalace(srv.URL, "")
	return mp, func() { srv.Close() }
}

const captureAutoSyncBody1 = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"messages_create","arguments":{"messages":[{"role":"user","content":"hello"},{"role":"assistant","content":"hi"}]}}}`

const captureAutoSyncBody2 = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"messages_create","arguments":{"messages":[{"role":"user","content":"follow-up question"}]}}}`

func TestCaptureAutoSyncPushesAllTurns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	stub := newMemPalaceStub()
	mp, stop := startMemPalace(t, stub)
	defer stop()

	r := gin.New()
	r.POST("/mcp",
		func(c *gin.Context) {
			c.Set("user_id", "bob")
			c.JSON(200, gin.H{"jsonrpc": "2.0", "result": "ok"})
		},
		CaptureSessions(CaptureConfig{
			DB:        db,
			MemPalace: mp,
			Wing:      "project_demo",
			Hall:      "events",
		}),
	)

	// First request: 2 turns
	req := httptest.NewRequest("POST", "/mcp?project=/tmp/demo", bytes.NewBufferString(captureAutoSyncBody1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	// Second request: 1 more turn
	req = httptest.NewRequest("POST", "/mcp?project=/tmp/demo", bytes.NewBufferString(captureAutoSyncBody2))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	// Wait until the watermark reflects all 3 turns.
	deadline := time.Now().Add(3 * time.Second)
	var synced int
	for time.Now().Before(deadline) {
		_ = db.QueryRowContext(context.TODO(),
			`SELECT mempalace_synced_turns FROM ai_sessions WHERE id = ?`,
			"sess_bob_"+shortHash("bob|/tmp/demo")).Scan(&synced)
		if synced == 3 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if synced != 3 {
		t.Fatalf("expected synced=3, got %d", synced)
	}
	if got := stub.count(); got != 3 {
		t.Fatalf("expected 3 drawer hits, got %d", got)
	}

	// Each hit must include the configured wing + hall.
	stub.mu.Lock()
	defer stub.mu.Unlock()
	for i, h := range stub.hits {
		if h["path"] != "/api/drawers" {
			t.Fatalf("hit[%d] path=%q", i, h["path"])
		}
		if !bytes.Contains([]byte(h["body"]), []byte(`"wing":"project_demo"`)) {
			t.Fatalf("hit[%d] body missing wing: %s", i, h["body"])
		}
		if !bytes.Contains([]byte(h["body"]), []byte(`"hall":"events"`)) {
			t.Fatalf("hit[%d] body missing hall: %s", i, h["body"])
		}
	}
}

func TestCaptureAutoSyncIdempotentOnRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	stub := newMemPalaceStub()
	// Force the first request to fail so the retry path runs once.
	atomic.StoreInt32(&stub.failN, 1)
	mp, stop := startMemPalace(t, stub)
	defer stop()

	r := gin.New()
	r.POST("/mcp",
		func(c *gin.Context) {
			c.Set("user_id", "carol")
			c.JSON(200, gin.H{"jsonrpc": "2.0", "result": "ok"})
		},
		CaptureSessions(CaptureConfig{
			DB:        db,
			MemPalace: mp,
			Wing:      "project_x",
			Hall:      "events",
			MaxRetries: 3,
		}),
	)

	req := httptest.NewRequest("POST", "/mcp?project=/tmp/x", bytes.NewBufferString(captureAutoSyncBody1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	// The stub records ONLY successful requests, so a single retry
	// should still produce 2 successful draws (one per turn). The first
	// attempt is rejected by the stub and not recorded.
	deadline := time.Now().Add(3 * time.Second)
	var synced int
	for time.Now().Before(deadline) {
		_ = db.QueryRowContext(context.TODO(),
			`SELECT mempalace_synced_turns FROM ai_sessions WHERE id = ?`,
			"sess_carol_"+shortHash("carol|/tmp/x")).Scan(&synced)
		if synced == 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if synced != 2 {
		t.Fatalf("expected synced=2 after retry, got %d", synced)
	}
	if got := stub.count(); got != 2 {
		t.Fatalf("expected 2 successful drawer hits, got %d", got)
	}

	// last_error must be NULL after a successful retry.
	var lastErr *string
	if err := db.QueryRowContext(context.TODO(),
		`SELECT mempalace_last_error FROM ai_sessions WHERE id = ?`,
		"sess_carol_"+shortHash("carol|/tmp/x")).Scan(&lastErr); err != nil {
		t.Fatalf("read last_error: %v", err)
	}
	if lastErr != nil && *lastErr != "" {
		t.Fatalf("expected last_error cleared, got %q", *lastErr)
	}
}

func TestCaptureAutoSyncDoesNotBlockOnOutage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	stub := newMemPalaceStub()
	// Simulate a permanent outage — every request fails. Use a tiny
	// number so the retry loop completes within the test budget.
	atomic.StoreInt32(&stub.failN, 50)
	mp, stop := startMemPalace(t, stub)
	defer stop()

	r := gin.New()
	r.POST("/mcp",
		func(c *gin.Context) {
			c.Set("user_id", "dave")
			c.JSON(200, gin.H{"jsonrpc": "2.0", "result": "ok"})
		},
		CaptureSessions(CaptureConfig{
			DB:         db,
			MemPalace:  mp,
			Wing:       "project_y",
			Hall:       "events",
			MaxRetries: 2,
		}),
	)

	start := time.Now()
	req := httptest.NewRequest("POST", "/mcp?project=/tmp/y", bytes.NewBufferString(captureAutoSyncBody1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	elapsed := time.Since(start)

	// The MCP request itself must return well under 1s even when
	// MemPalace is fully unreachable (retries happen async).
	if elapsed > time.Second {
		t.Fatalf("MCP request blocked on sync: %v", elapsed)
	}

	// Wait for the retries to drain. With 2 turns x 2 retries and a
	// 200ms initial backoff, the failure path completes in well under
	// a second; allow generous slack for CI scheduling.
	deadline := time.Now().Add(5 * time.Second)
	var synced int
	var lastErr *string
	for time.Now().Before(deadline) {
		_ = db.QueryRowContext(context.TODO(),
			`SELECT mempalace_synced_turns, mempalace_last_error FROM ai_sessions WHERE id = ?`,
			"sess_dave_"+shortHash("dave|/tmp/y")).Scan(&synced, &lastErr)
		if lastErr != nil && *lastErr != "" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// synced should remain 0 because all retries failed.
	if synced != 0 {
		t.Fatalf("expected synced=0 on outage, got %d", synced)
	}

	// last_error must be populated for ops visibility.
	if lastErr == nil || *lastErr == "" {
		t.Fatalf("expected last_error set on outage, got nil/empty")
	}
}

func TestCaptureNoMemPalaceSkipsSync(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	r := gin.New()
	r.POST("/mcp",
		func(c *gin.Context) {
			c.Set("user_id", "erin")
			c.JSON(200, gin.H{"jsonrpc": "2.0", "result": "ok"})
		},
		// MemPalace left nil — must not panic and must not write the
		// watermark column.
		CaptureSessions(CaptureConfig{DB: db}),
	)

	req := httptest.NewRequest("POST", "/mcp?project=/tmp/z", bytes.NewBufferString(captureAutoSyncBody1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	time.Sleep(200 * time.Millisecond)

	var synced sql.NullInt32
	_ = db.QueryRowContext(context.TODO(),
		`SELECT mempalace_synced_turns FROM ai_sessions WHERE id = ?`,
		"sess_erin_"+shortHash("erin|/tmp/z")).Scan(&synced)
	if synced.Valid && synced.Int32 != 0 {
		t.Fatalf("expected synced=0 with nil MemPalace, got %v", synced)
	}
}
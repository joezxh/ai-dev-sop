package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/console"
)

// invocation is the in-memory shape of one tool_invocation_logs row.
//
// Streamable (T1.2) and the upgraded CaptureSessions (this file) both
// construct an invocation, call markComplete once the upstream pool
// returns, and persist asynchronously. The struct is intentionally
// short-lived — never survives past the goroutine that persists it.
type invocation struct {
	InvocationID  string
	Transport     string // "stdio" | "streamable"
	UserID        string
	ProjectID     string
	ProjectPath   string
	ToolID        string
	ArgsJSON      string
	ResponseJSON  string
	StartedAt     time.Time
	EndedAt       time.Time
	LatencyMS     int
	ErrorCode     string
	TimeoutFlag   bool
	WorkflowRunID string
	BlastRadius   json.RawMessage
	ClientIP      string
	JWTSub        string
}

func newInvocation(transport, userID, project string, body []byte, c *gin.Context) *invocation {
	now := time.Now().UTC()
	return &invocation{
		InvocationID: newID("inv_"),
		Transport:    transport,
		UserID:       userID,
		ProjectPath:  project,
		ToolID:       jsonRPCMethod(body),
		ArgsJSON:     string(body),
		StartedAt:    now,
		ClientIP:     c.ClientIP(),
		JWTSub:       userID,
	}
}

// markComplete fills the latency / response / error fields. Invoked
// immediately after pool.Request returns.
func (i *invocation) markComplete(resp string, err error, latency time.Duration) {
	i.EndedAt = time.Now().UTC()
	i.LatencyMS = int(latency.Milliseconds())
	if err != nil {
		i.ErrorCode = "UPSTREAM_ERROR"
		// Truncate upstream error text into ResponseJSON so the row is
		// useful in dashboard debugging without overflowing MySQL TEXT.
		if len(err.Error()) > 4000 {
			i.ResponseJSON = err.Error()[:4000]
		} else {
			i.ResponseJSON = err.Error()
		}
		return
	}
	if len(resp) > 200_000 {
		// spec §11.1 calls for truncation at 200KB to keep the table lean.
		i.ResponseJSON = resp[:200_000] + "\n[truncated]"
	} else {
		i.ResponseJSON = resp
	}
	// Heuristic: protocol-level error codes appear as "error" JSON keys.
	if strings.Contains(resp, `"error":`) || strings.Contains(resp, `"code":`) {
		i.ErrorCode = "RPC_ERROR"
	}
}

// persist writes the invocation row to tool_invocation_logs. Dialect-aware:
// MySQL uses INSERT IGNORE so a retry on the same invocation_id is
// idempotent; SQLite uses INSERT OR IGNORE for the same effect.
func (i *invocation) persist(ctx context.Context, db *console.DB) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}
	stmt := "INSERT IGNORE INTO tool_invocation_logs (invocation_id, user_id, project_path, tool_id, transport, args_json, response_json, started_at, ended_at, latency_ms, error_code, timeout_flag, workflow_run_id, client_ip, jwt_sub) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	if db.Driver() == "sqlite" {
		stmt = strings.Replace(stmt, "INSERT IGNORE", "INSERT OR IGNORE", 1)
	}
	argsJSON := i.ArgsJSON
	if len(argsJSON) > 200_000 {
		argsJSON = argsJSON[:200_000] + "\n[truncated]"
	}
	_, err := db.ExecContext(ctx, stmt,
		i.InvocationID, i.UserID, i.ProjectPath, i.ToolID, i.Transport,
		argsJSON, i.ResponseJSON, i.StartedAt, i.EndedAt, i.LatencyMS,
		nullIfEmpty(i.ErrorCode), boolToInt(i.TimeoutFlag), nullIfEmpty(i.WorkflowRunID),
		i.ClientIP, i.JWTSub,
	)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// newID returns a random hex prefix suitable for an invocation id.
// 16 bytes (128 bits) of entropy is more than enough to avoid collisions
// at the v1 write rate.
func newID(prefix string) string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return prefix + hex.EncodeToString(b[:])
}

// CaptureInvocations is the v2 capture middleware. It writes the legacy
// session/session_turns rows (already done by CaptureSessions) AND a
// tool_invocation_logs row for every MCP /mcp POST. The streamable
// handler is self-persisting and signals it via the
// `skip_capture_invocations` context value so we don't double-write.
func CaptureInvocations(db *console.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()

		c.Next()

		// Streamable handler already persisted; skip.
		if _, skip := c.Get("skip_capture_invocations"); skip {
			return
		}
		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
			return
		}
		uidVal, ok := c.Get("user_id")
		if !ok {
			return
		}
		uid, _ := uidVal.(string)
		if uid == "" {
			return
		}
		body := capturedBody(c)
		// Body is nil only if CaptureSessions wasn't applied to this
		// route — in which case we have no payload to capture from.
		if len(body) == 0 {
			return
		}

		transport := "stdio"
		if c.Query("transport") == "streamable" {
			transport = "streamable"
		}
		inv := newInvocation(transport, uid, c.Query("project"), body, c)
		inv.EndedAt = time.Now().UTC()
		inv.LatencyMS = int(inv.EndedAt.Sub(started).Milliseconds())
		if v, ok := c.Get("invocation_response"); ok {
			if s, ok2 := v.(string); ok2 {
				inv.ResponseJSON = s
			}
		}
		if db == nil {
			return
		}
		go func() {
			if err := inv.persist(context.Background(), db); err != nil {
				// best-effort
			}
		}()
	}
}

// capturedBody returns the request body if CaptureSessions already
// replaced it (it puts the body bytes back via io.NopCloser). We read
// them again so the v1 capture is cheap and the v2 capture sees the
// same payload.
func capturedBody(c *gin.Context) []byte {
	if v, ok := c.Get("captured_body"); ok {
		if b, ok2 := v.([]byte); ok2 {
			return b
		}
	}
	return nil
}

// httpOK is a small helper that hides the cast — kept here so callers
// don't need to import "net/http" just to read c.Writer.Status().
func httpOK(c *gin.Context) int {
	if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
		return c.Writer.Status()
	}
	return c.Writer.Status()
}

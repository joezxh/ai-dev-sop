package console

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// capState serialises the SQLite writes triggered by CaptureSessions so a
// burst of MCP calls from the same user doesn't queue behind SQLite's
// "database is locked" errors. One buffered (size 1) channel per user id is
// enough: while a goroutine is writing for that user, the next request from
// the same user parks on the channel.
type capState struct {
	mu sync.Mutex
	ch map[string]chan struct{}
}

var capGlob = &capState{ch: map[string]chan struct{}{}}

func capChanFor(uid string) chan struct{} {
	capGlob.mu.Lock()
	defer capGlob.mu.Unlock()
	c, ok := capGlob.ch[uid]
	if !ok {
		c = make(chan struct{}, 1)
		capGlob.ch[uid] = c
	}
	return c
}

// shortHash returns a stable 8-char hex of s. Used to derive a session id
// from "<user>_<project>" — same pair always maps to the same id (rolling
// session model).
func shortHash(s string) string {
	h := uint32(0)
	for i := 0; i < len(s); i++ {
		h = h*31 + uint32(s[i])
	}
	return fmtHex(uint64(h))
}

// fmtHex renders n as 8 fixed-width lowercase hex chars (no leading-zero
// padding tricks). Using a hand-rolled formatter avoids pulling fmt.Sprintf
// into the hot path of every MCP request.
func fmtHex(n uint64) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		out[i] = hex[n&0xf]
		n >>= 4
	}
	return string(out)
}

// captureJSONRPC mirrors the bits of a JSON-RPC MCP request the capture
// middleware actually inspects. Defined here rather than imported because
// the shape is fixed by the protocol and we don't want a public type.
type captureJSONRPC struct {
	Method string `json:"method"`
	Params struct {
		Name      string `json:"name"`
		Arguments struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"arguments"`
	} `json:"params"`
}

type captureWriteReq struct {
	Params struct {
		Arguments struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"arguments"`
	} `json:"params"`
}

// CaptureSessions is a gin middleware that mirrors MCP /mcp requests into
// the console DB so admins can browse the conversation history of any user.
//
// The body is consumed BEFORE c.Next() (and replaced with a replay-able
// reader) because the downstream mcp.Handler will drain it. After c.Next()
// we check the response status and the user_id set by the JWT middleware,
// then spawn a goroutine to write the captured message turns. A per-user
// buffered channel serialises writes so the SQLite writer never sees a
// burst.
//
// Any error inside the write path is logged but never propagated back to
// the user: capturing is best-effort and must never break the MCP request
// that the user's IDE is waiting on.
func CaptureSessions(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			// Couldn't read the body; let the downstream handler see an
			// empty body rather than blocking the request.
			c.Next()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		// Stash a copy for downstream middlewares (e.g. M1's
		// CaptureInvocations needs the body to build tool_id / args_json).
		c.Set("captured_body", body)
		c.Next()

		if c.Writer.Status() != 200 {
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
		project := c.Query("project")
		if project == "" {
			project = c.GetHeader("X-Project-Path")
		}
		if !isMessageCall(body) {
			return
		}

		ch := capChanFor(uid)
		go func() {
			ch <- struct{}{}
			defer func() { <-ch }()
			if err := captureWrite(db, uid, project, body); err != nil {
				log.Printf("capture: write session failed uid=%s err=%v", uid, err)
			}
		}()
	}
}

// isMessageCall reports whether body is a JSON-RPC tools/call (or
// tools/invoke) that targets a *message*-like tool and carries a non-empty
// messages array.
func isMessageCall(body []byte) bool {
	var req captureJSONRPC
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}
	if req.Method != "tools/call" && req.Method != "tools/invoke" {
		return false
	}
	if !strings.Contains(strings.ToLower(req.Params.Name), "message") {
		return false
	}
	return len(req.Params.Arguments.Messages) > 0
}

// captureWrite writes one rolling session row + N turn rows for a captured
// MCP call. Rolling model: same uid+project always yields the same session
// id, so a long conversation accumulates into one row that keeps getting
// UPDATEd with the latest ended_at and turn_count.
func captureWrite(db *DB, uid, project string, body []byte) error {
	var req captureWriteReq
	if err := json.Unmarshal(body, &req); err != nil {
		return err
	}
	ctx := context.TODO()
	now := time.Now().UTC()
	sessID := "sess_" + uid + "_" + shortHash(uid+"|"+project)
	msgCount := len(req.Params.Arguments.Messages)

	if _, err := db.ExecContext(ctx,
		`INSERT OR IGNORE INTO sessions (id, user_id, project_id, project_path, started_at, turn_count, tool_count) VALUES (?,?,NULL,?,?,?,?)`,
		sessID, uid, project, now, 0, 0,
	); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE sessions SET ended_at=?, turn_count = turn_count + ? WHERE id=?`,
		now, msgCount, sessID,
	); err != nil {
		return err
	}

	// Compute the base turn_no AFTER the UPDATE so we don't race with
	// concurrent writes for the same user (the channel serialises, so the
	// UPDATE we just ran is committed before this SELECT).
	var base int
	if err := db.QueryRowContext(ctx,
		`SELECT turn_count FROM sessions WHERE id=?`, sessID,
	).Scan(&base); err != nil {
		return err
	}
	// turn_count now reflects the post-update total; new turns occupy
	// [base - msgCount + 1 .. base].
	base = base - msgCount

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for i, m := range req.Params.Arguments.Messages {
		content := m.Content
		if len(content) > 4000 {
			content = content[:4000]
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO session_turns (session_id, turn_no, role, content, tools_json, ts) VALUES (?,?,?,?,?,?)`,
			sessID, base+i+1, m.Role, content, nil, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

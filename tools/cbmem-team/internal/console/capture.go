package console

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/llm"
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

// CaptureConfig controls optional side-effects of CaptureSessions. All
// fields are optional; the zero value preserves the v1 behaviour of
// writing only to the local console DB.
type CaptureConfig struct {
	// DB is required.
	DB *DB
	// MemPalace, when non-nil, enables the background auto-sync that
	// mirrors new turns into MemPalace as Drawers. nil disables sync
	// (matching v1 behaviour) — used in unit tests and deployments
	// without MemPalace configured.
	MemPalace *llm.MemPalace
	// Wing is the MemPalace wing under which auto-synced drawers are
	// written. Empty string means "use wing derived from the project
	// path basename" (default). The console summariser still uses its
	// own target_wing — this only affects the streaming auto-sync.
	Wing string
	// Hall selects which MemPalace hall auto-sync writes to. Defaults
	// to "events" (the hall best matching "a stream of turns arriving
	// over time").
	Hall string
	// MaxRetries caps the per-turn retry count when MemPalace is
	// unreachable. Defaults to 3 if zero.
	MaxRetries int
	// Logger receives structured sync events. If nil, log.Printf is
	// used. The handler is called from the sync goroutine, so it must
	// be safe for concurrent use.
	Logger func(level, msg string, kv ...any)
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
// When CaptureConfig.MemPalace is non-nil the same goroutine additionally
// pushes any new turns (turns whose turn_no exceeds the per-session
// mempalace_synced_turns counter) into MemPalace as Drawers. Sync is
// fire-and-forget: errors are recorded on sessions.mempalace_last_error
// but never returned to the MCP client.
//
// Any error inside the write path is logged but never propagated back to
// the user: capturing is best-effort and must never break the MCP request
// that the user's IDE is waiting on.
func CaptureSessions(cfg CaptureConfig) gin.HandlerFunc {
	if cfg.DB == nil {
		panic("CaptureSessions: cfg.DB is required")
	}
	hall := cfg.Hall
	if hall == "" {
		hall = "events"
	}
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	logf := cfg.Logger
	if logf == nil {
		logf = func(level, msg string, kv ...any) {
			// Default logger formats the kvs as "k=v k=v" so the
			// existing log scrapers keep working.
			args := make([]any, 0, len(kv)+2)
			args = append(args, "capture-sync "+level+" "+msg)
			for i := 0; i+1 < len(kv); i += 2 {
				args = append(args, fmt.Sprintf("%s=%v", kv[i], kv[i+1]))
			}
			log.Println(args...)
		}
	}

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

		sessID := "sess_" + uid + "_" + shortHash(uid+"|"+project)
		ch := capChanFor(uid)
		go func() {
			ch <- struct{}{}
			defer func() { <-ch }()
			if err := captureWrite(cfg.DB, uid, project, body); err != nil {
				log.Printf("capture: write session failed uid=%s err=%v", uid, err)
				return
			}
			if cfg.MemPalace != nil {
				syncTurnsToMemPalace(syncTurnsParams{
					DB:         cfg.DB,
					MP:         cfg.MemPalace,
					SessionID:  sessID,
					UID:        uid,
					Project:    project,
					Wing:       cfg.Wing,
					Hall:       hall,
					MaxRetries: maxRetries,
					Logf:       logf,
				})
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
//
// It is the SQLite/MySQL-only path; the MemPalace sync is intentionally
// layered on top (see syncTurnsToMemPalace) so the v1 contract — "capture
// must never break the MCP request" — is preserved even if MemPalace is
// misconfigured.
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

// ----------------------------------------------------------------------------
// MemPalace auto-sync (added 2026-07-15)
// ----------------------------------------------------------------------------
//
// syncTurnsToMemPalace pushes any turns beyond sessions.mempalace_synced_turns
// to MemPalace as Drawers, then bumps the counter on success. The bookkeeping
// is per-session and per-row, so an interrupted sync resumes from exactly
// where it stopped — no duplicates, no gaps. Each turn becomes ONE Drawer
// keyed by (session_id, turn_no) so the dedup is at the Drawer level on the
// MemPalace side too (AddDrawer is naturally idempotent when callers pass a
// stable content string; the deterministic formatter guarantees that).
//
// Failure model:
//   - per-turn retry with exponential backoff up to MaxRetries
//   - on final failure the latest error is stored in sessions.mempalace_last_error
//     and the synced counter is NOT advanced (next request will retry
//     only the missing turns)
//   - the goroutine never panics or returns errors to the MCP client
//
// Cost: O(N) HTTP calls per capture where N = number of new turns. At the
// rolling-session cadence this is typically 1-3 calls per MCP round-trip
// which is well within the 30s MemPalace timeout budget.

type syncTurnsParams struct {
	DB         *DB
	MP         *llm.MemPalace
	SessionID  string
	UID        string
	Project    string
	Wing       string
	Hall       string
	MaxRetries int
	Logf       func(level, msg string, kv ...any)
}

func syncTurnsToMemPalace(p syncTurnsParams) {
	defer func() {
		if r := recover(); r != nil {
			p.Logf("error", "mempalace sync panic recovered",
				"session_id", p.SessionID, "panic", fmt.Sprintf("%v", r))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Read the current synced watermark.
	var syncedTurns int
	if err := p.DB.QueryRowContext(ctx,
		`SELECT COALESCE(mempalace_synced_turns, 0) FROM sessions WHERE id = ?`,
		p.SessionID,
	).Scan(&syncedTurns); err != nil {
		p.Logf("error", "read watermark failed",
			"session_id", p.SessionID, "err", err.Error())
		return
	}

	// 2. Stream turns newer than the watermark, ordered by turn_no so the
	//    Drawer timeline reads in conversation order. We bound the batch
	//    to 100 turns per call so a long-running sync can interleave with
	//    new MCP requests; the per-session channel serialises anyway.
	rows, err := p.DB.QueryContext(ctx,
		`SELECT turn_no, role, content FROM session_turns
		   WHERE session_id = ? AND turn_no > ?
		   ORDER BY turn_no ASC
		   LIMIT 100`,
		p.SessionID, syncedTurns,
	)
	if err != nil {
		p.Logf("error", "fetch unsynced turns failed",
			"session_id", p.SessionID, "err", err.Error())
		return
	}
	type pendingTurn struct {
		turnNo  int
		role    string
		content string
	}
	var batch []pendingTurn
	for rows.Next() {
		var t pendingTurn
		if err := rows.Scan(&t.turnNo, &t.role, &t.content); err != nil {
			_ = rows.Close()
			p.Logf("error", "scan turn failed",
				"session_id", p.SessionID, "err", err.Error())
			return
		}
		batch = append(batch, t)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		p.Logf("error", "rows iteration failed",
			"session_id", p.SessionID, "err", err.Error())
		return
	}
	_ = rows.Close()

	if len(batch) == 0 {
		return
	}

	wing := p.Wing
	if wing == "" {
		wing = deriveAutoSyncWing(p.Project, p.UID)
	}

	var lastErr error
	var successUpTo int
	for _, t := range batch {
		drawerContent := formatTurnAsDrawer(p.SessionID, t.turnNo, t.role, t.content)
		if err := pushDrawerWithRetry(ctx, p.MP, wing, "turns", p.Hall, drawerContent, p.MaxRetries, p.Logf, p.SessionID, t.turnNo); err != nil {
			lastErr = err
			break
		}
		successUpTo = t.turnNo
	}

	// 3. Advance the watermark only on success so partial failures are
	//    retried on the next capture. We store the latest error in
	//    mempalace_last_error for ops visibility.
	if successUpTo > 0 {
		if _, err := p.DB.ExecContext(ctx,
			`UPDATE sessions
			    SET mempalace_synced_turns = ?,
			        mempalace_last_synced_at = ?,
			        mempalace_last_error = NULL
			  WHERE id = ? AND mempalace_synced_turns < ?`,
			successUpTo, time.Now().UTC(), p.SessionID, successUpTo,
		); err != nil {
			p.Logf("error", "advance watermark failed",
				"session_id", p.SessionID, "turn_no", successUpTo, "err", err.Error())
		} else {
			p.Logf("info", "synced turns",
				"session_id", p.SessionID, "count", len(batch),
				"up_to", successUpTo)
		}
	}
	if lastErr != nil {
		errMsg := lastErr.Error()
		if len(errMsg) > 500 {
			errMsg = errMsg[:500]
		}
		if _, err := p.DB.ExecContext(ctx,
			`UPDATE sessions SET mempalace_last_error = ? WHERE id = ?`,
			errMsg, p.SessionID,
		); err != nil {
			p.Logf("error", "store last_error failed",
				"session_id", p.SessionID, "err", err.Error())
		}
		p.Logf("warn", "sync stopped mid-batch",
			"session_id", p.SessionID, "err", errMsg)
	}
}

// pushDrawerWithRetry calls MemPalace.AddDrawer with bounded exponential
// backoff. The retry budget is owned by MaxRetries; context cancellation
// (timeout) short-circuits the loop so a long MemPalace outage doesn't
// pile up goroutines.
func pushDrawerWithRetry(
	ctx context.Context,
	mp *llm.MemPalace,
	wing, room, hall, content string,
	maxRetries int,
	logf func(level, msg string, kv ...any),
	sessionID string,
	turnNo int,
) error {
	var lastErr error
	backoff := 200 * time.Millisecond
	for attempt := 0; attempt < maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := mp.AddDrawerOnce(ctx, wing, room, hall, content); err == nil {
			return nil
		} else {
			lastErr = err
			if attempt < maxRetries-1 {
				logf("debug", "mempalace retry",
					"session_id", sessionID, "turn_no", turnNo,
					"attempt", attempt+1, "backoff_ms", backoff.Milliseconds(),
					"err", err.Error())
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
				backoff *= 2
				if backoff > 5*time.Second {
					backoff = 5 * time.Second
				}
			}
		}
	}
	return lastErr
}

// formatTurnAsDrawer is the deterministic formatter that turns one
// session_turn into a MemPalace Drawer content string. Determinism matters:
// AddDrawer is naturally idempotent only when callers pass a stable
// payload, and the watermark relies on (session_id, turn_no) being unique
// so a re-run of the same batch produces the same content and MemPalace
// can dedup at its end.
func formatTurnAsDrawer(sessionID string, turnNo int, role, content string) string {
	// Cap the content at 8KB — Drawers are meant to be glanceable, and a
	// pathological 100KB tool output is the wrong shape for a Drawer.
	if len(content) > 8192 {
		content = content[:8192] + "\n…[truncated]"
	}
	return fmt.Sprintf("<!-- cbmem-team auto-sync session=%s turn=%d -->\n\n**%s**\n\n%s",
		sessionID, turnNo, role, content)
}

// deriveAutoSyncWing picks a sensible default wing when the caller did
// not configure one explicitly. Project basename is preferred; falls
// back to the user id when project is empty.
func deriveAutoSyncWing(project, uid string) string {
	if project != "" {
		// Take the last path component, strip common VCS / build suffixes.
		base := project
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		} else if i := strings.LastIndex(base, "\\"); i >= 0 {
			base = base[i+1:]
		}
		base = strings.TrimSuffix(base, ".git")
		if base != "" {
			return "project_" + base
		}
	}
	if uid != "" {
		return "user_" + uid
	}
	return "cbmem_team"
}
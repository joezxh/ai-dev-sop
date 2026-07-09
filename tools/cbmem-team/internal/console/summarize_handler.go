package console

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/llm"
)

// SummarizeHandler runs an LLM-driven summarization over the requested source
// sessions synchronously and stores the result in summarize_tasks. The
// caller receives both the row id and the result inline. The handler is the
// reference implementation for "synchronous task" — Task 12 (and beyond) builds
// on the same shape.
func SummarizeHandler(db *DB, provider llm.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SourceIDs  []string `json:"source_ids"`
			Depth      string   `json:"depth"`
			TargetWing string   `json:"target_wing"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000001, "invalid request: "+err.Error())
			return
		}
		if req.Depth == "" {
			req.Depth = "deep"
		}
		if len(req.SourceIDs) == 0 {
			Fail(c, http.StatusBadRequest, 4000002, "source_ids required")
			return
		}

		taskID := fmt.Sprintf("sum_%s", randomHex(6))
		now := time.Now().UTC()
		srcJSON, _ := json.Marshal(req.SourceIDs)
		if _, err := db.ExecContext(c.Request.Context(),
			`INSERT INTO summarize_tasks (id, user_id, source_ids, depth, target_wing, status, created_at)
			 VALUES (?,?,?,?,?,?,?)`,
			taskID, c.GetString("admin_id"), string(srcJSON), req.Depth, req.TargetWing, "running", now); err != nil {
			Fail(c, http.StatusInternalServerError, 5000001, "insert task: "+err.Error())
			return
		}

		transcript, err := loadTranscript(c.Request.Context(), db, req.SourceIDs)
		if err != nil {
			_, _ = db.Exec(
				`UPDATE summarize_tasks SET status='error', finished_at=? WHERE id=?`,
				time.Now().UTC(), taskID)
			Fail(c, http.StatusInternalServerError, 5000001, "load transcript: "+err.Error())
			return
		}

		prompt, err := loadPrompt(req.Depth)
		if err != nil {
			_, _ = db.Exec(
				`UPDATE summarize_tasks SET status='error', finished_at=? WHERE id=?`,
				time.Now().UTC(), taskID)
			Fail(c, http.StatusInternalServerError, 5000001, "load prompt: "+err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		resp, err := provider.Complete(ctx, llm.Request{
			System: prompt,
			Messages: []llm.Message{
				{Role: llm.RoleUser, Content: transcript},
			},
			Format: "json",
		})
		if err != nil {
			_, _ = db.Exec(
				`UPDATE summarize_tasks SET status='error', finished_at=? WHERE id=?`,
				time.Now().UTC(), taskID)
			Fail(c, http.StatusBadGateway, 5000001, "llm complete: "+err.Error())
			return
		}

		finishedAt := time.Now().UTC()
		if _, err := db.Exec(
			`UPDATE summarize_tasks SET status='done', result_json=?, finished_at=? WHERE id=?`,
			resp.Text, finishedAt, taskID); err != nil {
			Fail(c, http.StatusInternalServerError, 5000001, "update task: "+err.Error())
			return
		}

		OK(c, gin.H{
			"task_id":     taskID,
			"result":      json.RawMessage(resp.Text),
			"depth":       req.Depth,
			"target_wing": req.TargetWing,
		})
	}
}

// GetSummarizeHandler returns the row from summarize_tasks, including the
// LLM-emitted result_json (decoded as a raw JSON value so the front-end can
// pass it through unchanged). 4040001 if missing.
func GetSummarizeHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("task_id")
		var status string
		var result sql.NullString
		var created sql.NullTime
		var finished sql.NullTime
		var wing sql.NullString
		var depth sql.NullString
		err := db.QueryRowContext(c.Request.Context(),
			`SELECT status, result_json, created_at, finished_at, target_wing, depth
			 FROM summarize_tasks WHERE id=?`, id,
		).Scan(&status, &result, &created, &finished, &wing, &depth)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Fail(c, http.StatusNotFound, 4040001, "task not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000001, "read task: "+err.Error())
			return
		}
		var raw json.RawMessage
		if result.Valid && result.String != "" {
			raw = json.RawMessage(result.String)
		}
		OK(c, gin.H{
			"status":      status,
			"result":      raw,
			"created_at":  created.Time,
			"finished_at": finished.Time,
			"target_wing": wing.String,
			"depth":       depth.String,
		})
	}
}

// loadTranscript concatenates session turns as
// "[<session_id> #<turn_no> <role>] <content>\n" across the given source
// ids, ordered by session then turn. Missing source ids simply contribute
// no rows.
func loadTranscript(ctx context.Context, db *DB, ids []string) (string, error) {
	if len(ids) == 0 {
		return "", nil
	}
	q := `SELECT session_id, turn_no, role, content FROM session_turns
	      WHERE session_id IN (` + placeholders(len(ids)) + `)
	      ORDER BY session_id, turn_no ASC`
	args := make([]any, len(ids))
	for i, v := range ids {
		args[i] = v
	}
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var b strings.Builder
	for rows.Next() {
		var sid string
		var tn int
		var role, content string
		if err := rows.Scan(&sid, &tn, &role, &content); err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "[%s #%d %s] %s\n", sid, tn, role, content)
	}
	return b.String(), rows.Err()
}

// placeholders returns "?,?,?,..." with n question marks. Used to build
// safe IN-clauses without pulling in query-builder deps.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

// loadPrompt reads internal/llm/prompts/summarize_<depth>.txt relative to
// this source file. Using runtime.Caller avoids depending on the test or
// server working directory.
func loadPrompt(depth string) (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	promptsDir := filepath.Join(filepath.Dir(thisFile), "..", "llm", "prompts")
	b, err := os.ReadFile(filepath.Join(promptsDir, "summarize_"+depth+".txt"))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

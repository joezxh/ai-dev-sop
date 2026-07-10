package console

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/llm"
)

// distillRules captures the optional user-supplied filter for the distill
// call. Default values are applied in DistillHandler before the LLM is
// invoked so the prompt template substitution stays uniform.
type distillRules struct {
	MinScore   float64  `json:"min_value_score"`
	Dimensions []string `json:"dimensions"`
}

func defaultDistillRules() distillRules {
	return distillRules{MinScore: 0.7, Dimensions: []string{"fact", "decision", "discovery"}}
}

// DistillHandler synchronously runs an LLM-driven distillation pass over
// the requested source sessions and stores the JSON result on the
// distill_tasks row. The caller receives both the row id and the LLM
// output inline. Fragment/commit lifecycle is handled separately by
// CommitDistillHandler once a downstream system (MemPalace) is ready
// to receive the knowledge.
func DistillHandler(db *DB, provider llm.Provider, _ *llm.MemPalace) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SourceIDs []string     `json:"source_ids"`
			Rules     distillRules `json:"rules"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000001, "invalid request: "+err.Error())
			return
		}
		if req.Rules.MinScore == 0 {
			req.Rules.MinScore = 0.7
		}
		if len(req.Rules.Dimensions) == 0 {
			req.Rules.Dimensions = []string{"fact", "decision", "discovery"}
		}
		if len(req.SourceIDs) == 0 {
			Fail(c, http.StatusBadRequest, 4000002, "source_ids required")
			return
		}

		taskID := fmt.Sprintf("dis_%s", randomHex(6))
		now := time.Now().UTC()
		srcJSON, _ := json.Marshal(req.SourceIDs)
		rulesJSON, _ := json.Marshal(req.Rules)
		if _, err := db.ExecContext(c.Request.Context(),
			`INSERT INTO distill_tasks (id, user_id, source_ids, rules_json, status, created_at)
			 VALUES (?,?,?,?,?,?)`,
			taskID, c.GetString("admin_id"), string(srcJSON), string(rulesJSON), "running", now); err != nil {
			Fail(c, http.StatusInternalServerError, 5000001, "insert task: "+err.Error())
			return
		}

		transcript, err := loadTranscript(c.Request.Context(), db, req.SourceIDs)
		if err != nil {
			_, _ = db.Exec(
				`UPDATE distill_tasks SET status='error', finished_at=? WHERE id=?`,
				time.Now().UTC(), taskID)
			Fail(c, http.StatusInternalServerError, 5000001, "load transcript: "+err.Error())
			return
		}

		prompt, err := renderDistillPrompt(req.Rules)
		if err != nil {
			_, _ = db.Exec(
				`UPDATE distill_tasks SET status='error', finished_at=? WHERE id=?`,
				time.Now().UTC(), taskID)
			Fail(c, http.StatusInternalServerError, 5000001, "render prompt: "+err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
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
				`UPDATE distill_tasks SET status='error', finished_at=? WHERE id=?`,
				time.Now().UTC(), taskID)
			Fail(c, http.StatusBadGateway, 5000001, "llm complete: "+err.Error())
			return
		}

		finishedAt := time.Now().UTC()
		if _, err := db.Exec(
			`UPDATE distill_tasks SET status='done', result_json=?, finished_at=? WHERE id=?`,
			resp.Text, finishedAt, taskID); err != nil {
			Fail(c, http.StatusInternalServerError, 5000001, "update task: "+err.Error())
			return
		}

		OK(c, gin.H{
			"task_id": taskID,
			"result":  json.RawMessage(resp.Text),
		})
	}
}

// GetDistillHandler returns the distill row, including the LLM JSON
// result. 4040001 if missing.
func GetDistillHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("task_id")
		var status string
		var result sql.NullString
		var sync int
		var created sql.NullTime
		var finished sql.NullTime
		var wing sql.NullString
		err := db.QueryRowContext(c.Request.Context(),
			`SELECT status, result_json, mempalace_synced, created_at, finished_at, target_wing
			 FROM distill_tasks WHERE id=?`, id,
		).Scan(&status, &result, &sync, &created, &finished, &wing)
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
			"status":           status,
			"mempalace_synced": sync,
			"result":           raw,
			"created_at":       created.Time,
			"finished_at":      finished.Time,
			"target_wing":      wing.String,
		})
	}
}

// CommitDistillHandler pushes a previously-completed distill task's
// fragments and decisions into a MemPalace wing. Fragments are written
// as `room/hall=facts` drawers; decisions become `room/hall=advice`
// drawers prefixed with the decision title. After committing the row is
// marked `mempalace_synced=1` so the front-end can display the badge.
func CommitDistillHandler(db *DB, mp *llm.MemPalace) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("task_id")
		var body struct {
			TargetWing string `json:"target_wing"`
		}
		_ = c.ShouldBindJSON(&body)
		if body.TargetWing == "" {
			Fail(c, http.StatusBadRequest, 4000003, "target_wing required")
			return
		}

		var result sql.NullString
		err := db.QueryRowContext(c.Request.Context(),
			`SELECT result_json FROM distill_tasks WHERE id=?`, id).Scan(&result)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Fail(c, http.StatusNotFound, 4040001, "task not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000001, "read task: "+err.Error())
			return
		}
		if !result.Valid || result.String == "" {
			Fail(c, http.StatusNotFound, 4040001, "no result to commit")
			return
		}

		var data struct {
			Fragments []map[string]any `json:"knowledge_fragments"`
			Decisions []map[string]any `json:"decisions"`
		}
		if err := json.Unmarshal([]byte(result.String), &data); err != nil {
			Fail(c, http.StatusInternalServerError, 5000001, "parse result: "+err.Error())
			return
		}

		for _, f := range data.Fragments {
			content, _ := f["content"].(string)
			if err := mp.AddDrawer(c.Request.Context(), body.TargetWing, "room", "facts", content); err != nil {
				Fail(c, http.StatusBadGateway, 5000002, "mempalace fragment: "+err.Error())
				return
			}
		}
		for _, d := range data.Decisions {
			title, _ := d["title"].(string)
			decision, _ := d["decision"].(string)
			content := fmt.Sprintf("%s: %s", title, decision)
			if err := mp.AddDrawer(c.Request.Context(), body.TargetWing, "room", "advice", content); err != nil {
				Fail(c, http.StatusBadGateway, 5000002, "mempalace decision: "+err.Error())
				return
			}
		}

		if _, err := db.ExecContext(c.Request.Context(),
			`UPDATE distill_tasks SET mempalace_synced=1, target_wing=? WHERE id=?`,
			body.TargetWing, id); err != nil {
			Fail(c, http.StatusInternalServerError, 5000001, "update task: "+err.Error())
			return
		}
		OK(c, gin.H{
			"committed":        true,
			"fragments_count":  len(data.Fragments),
			"decisions_count":  len(data.Decisions),
		})
	}
}

// renderDistillPrompt loads internal/llm/prompts/distill_default.txt
// (resolved via runtime.Caller) and substitutes MinScore / Dimensions.
func renderDistillPrompt(r distillRules) (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	promptsDir := filepath.Join(filepath.Dir(thisFile), "..", "llm", "prompts")
	raw, err := os.ReadFile(filepath.Join(promptsDir, "distill_default.txt"))
	if err != nil {
		return "", err
	}
	tpl, err := template.New("distill").Parse(string(raw))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]any{
		"MinScore":   r.MinScore,
		"Dimensions": r.Dimensions,
	}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

package console

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/llm"
)

// =============================================================================
// v2 (M5) Summarize / Distill task HTTP wrappers.
// =============================================================================
//
// The actual LLM-driven work still lives in SummarizeHandler and
// DistillHandler (v1 admin-cookie path, kept untouched for
// compatibility). The v2 path stamps user_id from the JWT, persists
// the pending task row, then forwards the request body to the
// underlying handler so the LLM call stays in one place.
//
// Endpoints (all JWT-gated, mounted under /api/v2):
//
//   POST /summarize-tasks                       trigger summarize
//   GET  /summarize-tasks                       list recent (scoped to caller)
//   GET  /summarize-tasks/:id                   one task + status + result
//   POST /distill-tasks                         trigger distill
//   GET  /distill-tasks                         list recent (scoped to caller)
//   GET  /distill-tasks/:id                     one task
//
// Admin/Lead can list any user's tasks; developer/reporter only
// their own (filtered by user_id = JWT sub).
// =============================================================================

// SummarizeDistillHandlersV2 wraps the v2 storage + the v1 LLM
// handlers. It doesn't run the LLM itself; it just books the task
// row so the UI can poll for status, then delegates.
type SummarizeDistillHandlersV2 struct {
	DB       *DB
	Provider llm.Provider
	MP       *llm.MemPalace
}

// NewSummarizeDistillHandlersV2 builds a v2 handler.
func NewSummarizeDistillHandlersV2(db *DB, provider llm.Provider, mp *llm.MemPalace) *SummarizeDistillHandlersV2 {
	return &SummarizeDistillHandlersV2{DB: db, Provider: provider, MP: mp}
}

// CreateSummarize books a pending summarize task with user_id from
// the JWT context, then runs the existing SummarizeHandler logic
// to fill in the result. The v2 task row is visible via
// GET /summarize-tasks/:id while the LLM is running.
//
// The "run synchronously" choice matches the v1 contract so existing
// UI integrations keep working — but we go through the storage
// helper so the user_id is stamped from JWT (not the admin cookie)
// and so the row exists before the LLM call returns.
func (h *SummarizeDistillHandlersV2) CreateSummarize() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		uid := CurrentUserID(c)
		if uid == "" {
			Fail(c, http.StatusUnauthorized, 4010110, "no auth context")
			return
		}
		var req SummarizeTaskReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000110, "invalid request: "+err.Error())
			return
		}
		if len(req.SourceIDs) == 0 {
			Fail(c, http.StatusBadRequest, 4000111, "source_ids required")
			return
		}
		if req.Depth == "" {
			req.Depth = "deep"
		}
		if !validSummarizeDepth(req.Depth) {
			Fail(c, http.StatusBadRequest, 4000112, "depth must be shallow|deep|expert")
			return
		}
		taskID := "sum_" + randomHex(6)
		srcJSON, _ := json.Marshal(req.SourceIDs)
		t := &SummarizeTask{
			ID:         taskID,
			UserID:     uid,
			SourceIDs:  string(srcJSON),
			Depth:      req.Depth,
			TargetWing: req.TargetWing,
		}
		if err := h.DB.CreateSummarizeTask(ctx, t); err != nil {
			Fail(c, http.StatusInternalServerError, 5000130, "create summarize task: "+err.Error())
			return
		}
		// Run the v1 handler — it expects "admin_id" in the context,
		// so we stage that before invoking it. The handler also
		// re-INSERTs the row but our row was created with the same
		// id and will be UPDATEd in place to status='done'.
		c.Request = c.Request.WithContext(ctx)
		c.Set("admin_id", uid)
		SummarizeHandler(h.DB, h.Provider)(c)
	}
}

// GetSummarize returns one summarize task by id. RBAC: admin/lead
// see anything; everyone else only their own.
func (h *SummarizeDistillHandlersV2) GetSummarize() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		t, err := h.DB.GetSummarizeTask(ctx, id)
		if err != nil {
			if errors.Is(err, ErrMemoryNotFound) {
				Fail(c, http.StatusNotFound, 4040110, "task not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000131, "read task: "+err.Error())
			return
		}
		if !h.canSeeTask(c, t.UserID) {
			Fail(c, http.StatusNotFound, 4040111, "task not found")
			return
		}
		OK(c, summarizeTaskToDTO(t))
	}
}

// ListSummarize returns recent tasks. RBAC narrows the result set
// to the caller unless they're admin/lead.
func (h *SummarizeDistillHandlersV2) ListSummarize() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		uid := CurrentUserID(c)
		role := CurrentRole(c)
		limit, _ := strconv.Atoi(c.Query("limit"))
		// Admins/leads can request another user's list explicitly.
		scope := uid
		if role == RoleAdmin || role == RoleLead {
			if q := c.Query("user_id"); q != "" {
				scope = q
			}
		}
		rows, err := h.DB.ListSummarizeTasks(ctx, scope, limit)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000132, "list summarize: "+err.Error())
			return
		}
		out := make([]SummarizeTaskDTO, 0, len(rows))
		for _, t := range rows {
			out = append(out, summarizeTaskToDTO(t))
		}
		OK(c, gin.H{"list": out, "total": len(out)})
	}
}

// CreateDistill is the v2 wrapper around the v1 DistillHandler.
func (h *SummarizeDistillHandlersV2) CreateDistill() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		uid := CurrentUserID(c)
		if uid == "" {
			Fail(c, http.StatusUnauthorized, 4010111, "no auth context")
			return
		}
		var req DistillTaskReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000120, "invalid request: "+err.Error())
			return
		}
		if len(req.SourceIDs) == 0 {
			Fail(c, http.StatusBadRequest, 4000121, "source_ids required")
			return
		}
		rulesJSON, _ := json.Marshal(req.Rules)
		if string(rulesJSON) == "null" || len(rulesJSON) == 0 {
			rulesJSON = []byte(`{}`)
		}
		taskID := "dis_" + randomHex(6)
		t := &DistillTask{
			ID:         taskID,
			UserID:     uid,
			SourceIDs:  mustJSON(req.SourceIDs),
			RulesJSON:  marshalOrEmpty(req.Rules),
			TargetWing: req.TargetWing,
		}
		if err := h.DB.CreateDistillTask(ctx, t); err != nil {
			Fail(c, http.StatusInternalServerError, 5000140, "create distill task: "+err.Error())
			return
		}
		c.Request = c.Request.WithContext(ctx)
		c.Set("admin_id", uid)
		DistillHandler(h.DB, h.Provider, h.MP)(c)
	}
}

// GetDistill returns one distill task by id.
func (h *SummarizeDistillHandlersV2) GetDistill() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		t, err := h.DB.GetDistillTask(ctx, id)
		if err != nil {
			if errors.Is(err, ErrMemoryNotFound) {
				Fail(c, http.StatusNotFound, 4040120, "task not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000141, "read task: "+err.Error())
			return
		}
		if !h.canSeeTask(c, t.UserID) {
			Fail(c, http.StatusNotFound, 4040121, "task not found")
			return
		}
		OK(c, distillTaskToDTO(t))
	}
}

// ListDistill returns recent distill tasks.
func (h *SummarizeDistillHandlersV2) ListDistill() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		uid := CurrentUserID(c)
		role := CurrentRole(c)
		limit, _ := strconv.Atoi(c.Query("limit"))
		scope := uid
		if role == RoleAdmin || role == RoleLead {
			if q := c.Query("user_id"); q != "" {
				scope = q
			}
		}
		rows, err := h.DB.ListDistillTasks(ctx, scope, limit)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000142, "list distill: "+err.Error())
			return
		}
		out := make([]DistillTaskDTO, 0, len(rows))
		for _, t := range rows {
			out = append(out, distillTaskToDTO(t))
		}
		OK(c, gin.H{"list": out, "total": len(out)})
	}
}

// canSeeTask is the per-task RBAC: admin/lead see all; everyone
// else only their own.
func (h *SummarizeDistillHandlersV2) canSeeTask(c *gin.Context, ownerID string) bool {
	role := CurrentRole(c)
	uid := CurrentUserID(c)
	if role == RoleAdmin || role == RoleLead {
		return true
	}
	return uid != "" && uid == ownerID
}

// marshalOrEmpty returns the JSON encoding of v, or "{}" on error.
func marshalOrEmpty(v any) string {
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 {
		return "{}"
	}
	return string(b)
}

func summarizeTaskToDTO(t *SummarizeTask) SummarizeTaskDTO {
	ids := unmarshalSourceIDs(t.SourceIDs)
	dto := SummarizeTaskDTO{
		ID:         t.ID,
		UserID:     t.UserID,
		SourceIDs:  ids,
		Depth:      t.Depth,
		TargetWing: t.TargetWing,
		Status:     t.Status,
		Result:     rawJSON(t.ResultJSON),
		CreatedAt:  t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if t.FinishedAt != nil {
		s := t.FinishedAt.UTC().Format("2006-01-02T15:04:05Z")
		dto.FinishedAt = &s
	}
	return dto
}

func distillTaskToDTO(t *DistillTask) DistillTaskDTO {
	ids := unmarshalSourceIDs(t.SourceIDs)
	rules := json.RawMessage(t.RulesJSON)
	if len(rules) == 0 {
		rules = json.RawMessage(`{}`)
	}
	dto := DistillTaskDTO{
		ID:              t.ID,
		UserID:          t.UserID,
		SourceIDs:       ids,
		Rules:           rules,
		Status:          t.Status,
		Result:          rawJSON(t.ResultJSON),
		MemPalaceSynced: t.MemPalaceSynced,
		TargetWing:      t.TargetWing,
		CreatedAt:       t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if t.FinishedAt != nil {
		s := t.FinishedAt.UTC().Format("2006-01-02T15:04:05Z")
		dto.FinishedAt = &s
	}
	return dto
}

// unmarshalSourceIDs parses the JSON-encoded list into a Go slice.
// Malformed input returns nil; the UI treats that as "no sources".
func unmarshalSourceIDs(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

// rawJSON turns a stored JSON string into a json.RawMessage so the
// HTTP response emits it as a nested JSON value (not an escaped
// string). Empty input -> nil so the field is omitted via omitempty.
func rawJSON(s string) json.RawMessage {
	if s == "" {
		return nil
	}
	return json.RawMessage(s)
}
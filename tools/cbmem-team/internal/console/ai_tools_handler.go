package console

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ptrString is a shorthand for building *string literals inline.
// Exported so db_memory_test.go can reuse it.
func ptrString(s string) *string { return &s }

// =============================================================================
// v2 (M6) AI Tool HTTP handlers.
//
// Endpoints (all under /api/v2):
//
//   GET    /ai-tools                            list (team_id query, required)
//   POST   /ai-tools                            register a tool (admin/lead)
//   GET    /ai-tools/:id                        detail
//   PUT    /ai-tools/:id                        partial update (admin/lead)
//   DELETE /ai-tools/:id                        hard-delete (admin, only when unused)
//   POST   /ai-tools/:id/invoke                 run the tool (RBAC + project allow-list)
//   GET    /ai-tools/:id/invocations            recent audit log
//
// Invoke flow:
//   1. RBAC: caller role ≥ tool.required_role AND caller is a team member
//   2. Project allow-list: project_id must be in tool.allowed_projects (or all)
//   3. Module must be a leaf (sessions/memories attached → leaf)
//   4. Persist invocation (status=pending → running → success/error/timeout)
//   5. Dispatch: protocol=http → POST input as JSON; protocol=stdio → exec
//      with stdin JSON, capture stdout/stderr
//   6. Update invocation row with output/error and finish time
// =============================================================================

// AIToolHandlers wires DB to the AI tool HTTP routes.
type AIToolHandlers struct {
	DB *DB
	// HTTPClient is shared across invocations so tests can
	// override it. Defaults to http.DefaultClient.
	HTTPClient *http.Client
}

// NewAIToolHandlers builds a handler. HTTPClient defaults to
// http.DefaultClient with a 30s timeout if nil.
func NewAIToolHandlers(db *DB) *AIToolHandlers {
	return &AIToolHandlers{
		DB:         db,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ----------------------------------------------------------------------------
// Catalogue
// ----------------------------------------------------------------------------

// List returns all tools in a team (admin/lead) or just those the
// caller is allowed to invoke (member / developer).
func (h *AIToolHandlers) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		teamID := c.Query("team_id")
		if teamID == "" {
			Fail(c, http.StatusBadRequest, 4000200, "team_id required")
			return
		}
		role := CurrentRole(c)
		uid := CurrentUserID(c)
		if role == "" {
			Fail(c, http.StatusUnauthorized, 4010200, "no auth context")
			return
		}
		rows, err := h.DB.ListAITools(ctx, teamID)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000200, "list: "+err.Error())
			return
		}
		// Filter by RBAC if the caller isn't admin/lead. Each row
		// is checked independently so a team could mix tools of
		// different required_role values.
		visible := make([]*AITool, 0, len(rows))
		for _, t := range rows {
			ok, err := h.DB.AIToolAccessibleByUser(ctx, t.ID, uid, string(role))
			if err != nil {
				Fail(c, http.StatusInternalServerError, 5000201, "rbac: "+err.Error())
				return
			}
			if ok {
				visible = append(visible, t)
			}
		}
		out := make([]AIToolDTO, 0, len(visible))
		for _, t := range visible {
			out = append(out, aiToolToDTO(t))
		}
		OK(c, gin.H{"list": out, "total": len(out)})
	}
}

// Get returns one tool by id.
func (h *AIToolHandlers) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		t, err := h.DB.GetAIToolByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrAIToolNotFound) {
				Fail(c, http.StatusNotFound, 4040200, "tool not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000202, "read: "+err.Error())
			return
		}
		OK(c, aiToolToDTO(t))
	}
}

// Create registers a new tool. Admin/lead only.
func (h *AIToolHandlers) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		role := CurrentRole(c)
		if role != RoleAdmin && role != RoleLead {
			Fail(c, http.StatusForbidden, 4030200, "admin or lead role required")
			return
		}
		var req CreateAIToolReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000201, "invalid request: "+err.Error())
			return
		}
		if !validProtocols[req.Protocol] {
			Fail(c, http.StatusBadRequest, 4000202, "protocol must be http or stdio")
			return
		}
		if req.Protocol == "http" && strings.TrimSpace(req.Endpoint) == "" {
			Fail(c, http.StatusBadRequest, 4000202, "endpoint required for http protocol")
			return
		}
		if req.RequiredRole == "" {
			req.RequiredRole = "developer"
		}
		if !validRequiredRoles[req.RequiredRole] {
			Fail(c, http.StatusBadRequest, 4000203, "required_role must be admin|lead|developer")
			return
		}
		argsJSON, _ := json.Marshal(req.Args)
		allowedJSON, _ := json.Marshal(req.AllowedProjects)
		t := &AITool{
			ID:                    "ait_" + randomHex(6),
			TeamID:                req.TeamID,
			Name:                  req.Name,
			Slug:                  req.Slug,
			Description:           req.Description,
			Endpoint:              req.Endpoint,
			Protocol:              req.Protocol,
			Command:               req.Command,
			ArgsJSON:              string(argsJSON),
			RequiredRole:          req.RequiredRole,
			AllowedProjectsJSON:   string(allowedJSON),
			TimeoutSeconds:        req.TimeoutSeconds,
			Enabled:              true,
		}
		if err := h.DB.CreateAITool(ctx, t); err != nil {
			if errors.Is(err, ErrAIToolInvalidProtocol) || errors.Is(err, ErrAIToolInvalidRole) {
				Fail(c, http.StatusBadRequest, 4000204, err.Error())
				return
			}
			Fail(c, http.StatusInternalServerError, 5000203, "create: "+err.Error())
			return
		}
		// Refresh to load generated args_json (we sent empty
		// string and the DB defaults to "[]" — read it back so the
		// response is consistent with subsequent reads).
		if refreshed, err := h.DB.GetAIToolByID(ctx, t.ID); err == nil {
			t = refreshed
		}
		OK(c, aiToolToDTO(t))
	}
}

// Update applies a partial update. Admin/lead only.
func (h *AIToolHandlers) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		role := CurrentRole(c)
		if role != RoleAdmin && role != RoleLead {
			Fail(c, http.StatusForbidden, 4030201, "admin or lead role required")
			return
		}
		id := c.Param("id")
		var req UpdateAIToolReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000205, "invalid request: "+err.Error())
			return
		}
		upd := AIToolUpdate{
			Name:         req.Name,
			Description:  req.Description,
			Endpoint:     req.Endpoint,
			Command:      req.Command,
			RequiredRole: req.RequiredRole,
			TimeoutSeconds: req.TimeoutSeconds,
			Enabled:        req.Enabled,
		}
		// Marshal slice updates to JSON only when the caller
		// supplied them — nil = leave unchanged; an empty slice
		// = "set to [] (no args / no allowed projects)".
		if req.Args != nil {
			s, _ := json.Marshal(*req.Args)
			upd.ArgsJSON = ptrString(string(s))
		}
		if req.AllowedProjects != nil {
			if len(*req.AllowedProjects) == 0 {
				upd.AllowedProjectsJSON = ptrString("")
			} else {
				s, _ := json.Marshal(*req.AllowedProjects)
				upd.AllowedProjectsJSON = ptrString(string(s))
			}
		}
		if upd.Protocol != nil && !validProtocols[*upd.Protocol] {
			Fail(c, http.StatusBadRequest, 4000206, "protocol must be http or stdio")
			return
		}
		if err := h.DB.UpdateAITool(ctx, id, upd); err != nil {
			if errors.Is(err, ErrAIToolInvalidProtocol) || errors.Is(err, ErrAIToolInvalidRole) {
				Fail(c, http.StatusBadRequest, 4000207, err.Error())
				return
			}
			if errors.Is(err, ErrAIToolNotFound) {
				Fail(c, http.StatusNotFound, 4040201, "tool not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000204, "update: "+err.Error())
			return
		}
		t, _ := h.DB.GetAIToolByID(ctx, id)
		OK(c, aiToolToDTO(t))
	}
}

// Delete hard-deletes a tool. Admin only. The DB layer refuses if
// the tool has any invocation history so audits stay intact.
func (h *AIToolHandlers) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		role := CurrentRole(c)
		if role != RoleAdmin {
			Fail(c, http.StatusForbidden, 4030202, "admin role required")
			return
		}
		id := c.Param("id")
		if err := h.DB.DeleteAITool(ctx, id); err != nil {
			if errors.Is(err, ErrAIToolNotFound) {
				Fail(c, http.StatusNotFound, 4040202, "tool not found")
				return
			}
			Fail(c, http.StatusConflict, 4090200, err.Error())
			return
		}
		OK(c, gin.H{"id": id, "deleted": true})
	}
}

// ----------------------------------------------------------------------------
// Invoke + audit
// ----------------------------------------------------------------------------

// Invoke runs a registered AI tool. The invocation is logged in
// ai_tool_invocations with a state machine: pending → running →
// success/error/timeout. The HTTP response is the invocation row
// including the output and elapsed time.
func (h *AIToolHandlers) Invoke() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		role := CurrentRole(c)
		uid := CurrentUserID(c)
		if role == "" || uid == "" {
			Fail(c, http.StatusUnauthorized, 4010201, "no auth context")
			return
		}
		var req InvokeAIToolReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000208, "invalid request: "+err.Error())
			return
		}
		t, err := h.DB.GetAIToolByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrAIToolNotFound) {
				Fail(c, http.StatusNotFound, 4040203, "tool not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000205, "read: "+err.Error())
			return
		}
		if !t.Enabled {
			Fail(c, http.StatusBadRequest, 4000209, "tool is disabled")
			return
		}
		// RBAC
		ok, err := h.DB.AIToolAccessibleByUser(ctx, t.ID, uid, string(role))
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000206, "rbac: "+err.Error())
			return
		}
		if !ok {
			Fail(c, http.StatusForbidden, 4030203, "insufficient privileges for this tool")
			return
		}
		// Project allow-list
		ok, err = h.DB.AIToolProjectAllowed(ctx, t.ID, req.ProjectID)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000207, "allow-list: "+err.Error())
			return
		}
		if !ok {
			Fail(c, http.StatusForbidden, 4030204, "project not allowed for this tool")
			return
		}
		// Resolve working dir. The handler computes a default
		// from project.path if the caller didn't supply one,
		// making "click to run" feel native to the UI.
		workingDir := req.WorkingDir
		if workingDir == "" {
			var p string
			if err := h.DB.QueryRowContext(ctx,
				`SELECT path FROM pm_projects WHERE id = ? AND deleted = 0`, req.ProjectID,
			).Scan(&p); err == nil {
				workingDir = p
			}
		}
		// Validate project and module exist and are not deleted.
		// Without this the invocation INSERT fails with FK violations.
		var projCount, modCount int
		if err := h.DB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM pm_projects WHERE id = ? AND deleted = 0`, req.ProjectID,
		).Scan(&projCount); err != nil || projCount == 0 {
			Fail(c, http.StatusBadRequest, 4000210, "project not found")
			return
		}
		if err := h.DB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM pm_modules WHERE id = ? AND deleted = 0`, req.ModuleID,
		).Scan(&modCount); err != nil || modCount == 0 {
			Fail(c, http.StatusBadRequest, 4000211, "module not found")
			return
		}
		// Log invocation row at pending → running.
		inv := &AIToolInvocation{
			ID:         "inv_" + randomHex(6),
			ToolID:     t.ID,
			UserID:     uid,
			TeamID:     t.TeamID,
			ProjectID:  req.ProjectID,
			ModuleID:   req.ModuleID,
			WorkingDir: workingDir,
			Status:     "running",
			StartedAt:  time.Now().UTC(),
		}
		inputJSON, _ := json.Marshal(req.Input)
		inv.InputJSON = string(inputJSON)
		if err := h.DB.CreateInvocation(ctx, inv); err != nil {
			Fail(c, http.StatusInternalServerError, 5000208, "log invocation: "+err.Error())
			return
		}
		// Dispatch
		out, runErr := h.runTool(ctx, t, inv, req.Input)
		status := "success"
		errMsg := ""
		outputJSON := ""
		if runErr != nil {
			// Distinguish timeout from generic error.
			if errors.Is(runErr, context.DeadlineExceeded) {
				status = "timeout"
			} else {
				status = "error"
			}
			errMsg = runErr.Error()
		}
		if out != nil {
			b, _ := json.Marshal(out)
			outputJSON = string(b)
		}
		if updErr := h.DB.UpdateInvocationStatus(ctx, inv.ID, status, outputJSON, errMsg); updErr != nil {
			// Audit row is stuck at running; surface the error
			// but still return the response so the user knows
			// what happened. The invocation row will be cleaned
			// by a janitor in a future M7+.
			Fail(c, http.StatusInternalServerError, 5000209, "audit update failed: "+updErr.Error())
			return
		}
		inv.Status = status
		inv.OutputJSON = outputJSON
		inv.Error = errMsg
		now := time.Now().UTC()
		inv.FinishedAt = &now
		OK(c, gin.H{
			"invocation":  aiToolInvocationToDTO(inv),
			"output":      out,
			"status":      status,
			"error":       errMsg,
			"duration_ms": time.Since(inv.StartedAt).Milliseconds(),
		})
	}
}

// runTool dispatches based on protocol. Returns the parsed output
// (already JSON-shaped for stdio; raw bytes for HTTP) and a
// non-nil error on failure. context.DeadlineExceeded is returned
// verbatim so the caller can mark the row as 'timeout'.
func (h *AIToolHandlers) runTool(ctx context.Context, t *AITool, inv *AIToolInvocation, input map[string]any) (map[string]any, error) {
	timeout := time.Duration(t.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	switch t.Protocol {
	case "http":
		return h.runHTTP(runCtx, t, input)
	case "stdio":
		return h.runStdio(runCtx, t, input)
	default:
		return nil, fmt.Errorf("unsupported protocol %q", t.Protocol)
	}
}

// runHTTP POSTs the input as JSON to endpoint and returns the
// parsed JSON response. Non-JSON responses are wrapped in
// {"text": ...} so the audit row stays valid JSON.
func (h *AIToolHandlers) runHTTP(ctx context.Context, t *AITool, input map[string]any) (map[string]any, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	out := map[string]any{}
	if jerr := json.Unmarshal(respBody, &out); jerr != nil {
		// Non-JSON: wrap as text + status code.
		out = map[string]any{
			"text":    string(respBody),
			"status":  resp.StatusCode,
			"headers": resp.Header,
		}
	} else {
		out["status"] = resp.StatusCode
	}
	if resp.StatusCode >= 400 {
		return out, fmt.Errorf("tool returned HTTP %d", resp.StatusCode)
	}
	return out, nil
}

// runStdio spawns command + args, writes the input JSON to stdin,
// captures stdout/stderr. stdout is parsed as JSON; stderr is
// stored under the "stderr" key so it isn't lost.
func (h *AIToolHandlers) runStdio(ctx context.Context, t *AITool, input map[string]any) (map[string]any, error) {
	if t.Command == "" {
		return nil, errors.New("stdio tool requires command")
	}
	var args []string
	if t.ArgsJSON != "" && t.ArgsJSON != "[]" {
		if err := json.Unmarshal([]byte(t.ArgsJSON), &args); err != nil {
			return nil, fmt.Errorf("decode args: %w", err)
		}
	}
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}
	cmd := exec.CommandContext(ctx, t.Command, args...)
	cmd.Stdin = bytes.NewReader(body)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	out := map[string]any{
		"stdout": stdout.String(),
		"stderr": stderr.String(),
	}
	if jsonErr := json.Unmarshal(stdout.Bytes(), &out); jsonErr != nil {
		// Non-JSON stdout: keep raw under stdout, drop the
		// failed parse but don't blow up.
		out["stdout_parsed"] = false
	} else {
		out["stdout_parsed"] = true
	}
	if runErr != nil {
		return out, fmt.Errorf("stdio exec: %w", runErr)
	}
	return out, nil
}

// ListInvocations returns recent audit rows for a tool.
func (h *AIToolHandlers) ListInvocations() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		limit, _ := strconv.Atoi(c.Query("limit"))
		f := ListInvocationsFilter{ToolID: id, Limit: limit}
		if uid := c.Query("user_id"); uid != "" {
			f.UserID = uid
		}
		rows, err := h.DB.ListInvocations(ctx, f)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000210, "list: "+err.Error())
			return
		}
		out := make([]AIToolInvocationDTO, 0, len(rows))
		for _, inv := range rows {
			out = append(out, aiToolInvocationToDTO(inv))
		}
		OK(c, gin.H{"list": out, "total": len(out)})
	}
}

// GetInvocation returns one invocation row (e.g. for the
// "wait for result" polling path).
func (h *AIToolHandlers) GetInvocation() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("inv_id")
		inv, err := h.DB.GetInvocation(ctx, id)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				Fail(c, http.StatusNotFound, 4040204, "invocation not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000211, "read: "+err.Error())
			return
		}
		OK(c, aiToolInvocationToDTO(inv))
	}
}

// ----------------------------------------------------------------------------
// DTO conversion
// ----------------------------------------------------------------------------

func aiToolToDTO(t *AITool) AIToolDTO {
	if t == nil {
		return AIToolDTO{}
	}
	args := decodeStringSlice(t.ArgsJSON)
	allowed := decodeStringSlice(t.AllowedProjectsJSON)
	return AIToolDTO{
		ID:              t.ID,
		TeamID:          t.TeamID,
		Name:            t.Name,
		Slug:            t.Slug,
		Description:     t.Description,
		Endpoint:        t.Endpoint,
		Protocol:        t.Protocol,
		Command:         t.Command,
		Args:            args,
		RequiredRole:    t.RequiredRole,
		AllowedProjects: allowed,
		TimeoutSeconds:  t.TimeoutSeconds,
		Enabled:         t.Enabled,
		CreatedAt:       t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       t.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func aiToolInvocationToDTO(inv *AIToolInvocation) AIToolInvocationDTO {
	dto := AIToolInvocationDTO{
		ID:         inv.ID,
		ToolID:     inv.ToolID,
		UserID:     inv.UserID,
		TeamID:     inv.TeamID,
		ProjectID:  inv.ProjectID,
		ModuleID:   inv.ModuleID,
		WorkingDir: inv.WorkingDir,
		Status:     inv.Status,
		StartedAt:  inv.StartedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Error:      inv.Error,
	}
	if inv.FinishedAt != nil {
		s := inv.FinishedAt.UTC().Format("2006-01-02T15:04:05Z")
		dto.FinishedAt = &s
	}
	return dto
}

func decodeStringSlice(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}
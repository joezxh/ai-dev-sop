package console

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// v2 (M6) AI tool storage layer.
//
// ai_tools is the catalogue of registered CLI tools (cursor, qoder,
// superpowers, gstack, codebase-mem-mcp, custom) that the team has
// blessed for invocation. ai_tool_invocations is the audit log
// one row per execution, capturing input/output/working_dir.
//
// The HTTP layer (ai_tools_handler.go) is responsible for actually
// running the tool (HTTP call or stdio spawn). This file is the
// catalogue + audit DB only.
// =============================================================================

var (
	ErrAIToolNotFound        = errors.New("ai tool not found")
	ErrAIToolInvalidProtocol = errors.New("invalid ai_tool protocol")
	ErrAIToolInvalidRole     = errors.New("invalid ai_tool required_role")
)

// validProtocols matches the schema CHECK constraint.
var validProtocols = map[string]bool{
	"http":  true,
	"stdio": true,
}

// validRequiredRoles matches the schema CHECK constraint.
var validRequiredRoles = map[string]bool{
	"admin":     true,
	"lead":      true,
	"developer": true,
}

// ----------------------------------------------------------------------------
// Catalogue CRUD
// ----------------------------------------------------------------------------

// CreateAITool inserts a new tool row. id/slug uniqueness is
// enforced by the (team_id, slug) UNIQUE index. args_json and
// allowed_projects_json are validated at this layer so callers
// don't have to encode []string → "[]".
func (db *DB) CreateAITool(ctx context.Context, t *AITool) error {
	if t.ID == "" {
		return errors.New("CreateAITool: id required")
	}
	if t.TeamID == "" {
		return errors.New("CreateAITool: team_id required")
	}
	if strings.TrimSpace(t.Name) == "" || strings.TrimSpace(t.Slug) == "" {
		return errors.New("CreateAITool: name and slug required")
	}
	if !validProtocols[t.Protocol] {
		return fmt.Errorf("%w: %q", ErrAIToolInvalidProtocol, t.Protocol)
	}
	if t.RequiredRole == "" {
		t.RequiredRole = "developer"
	}
	if !validRequiredRoles[t.RequiredRole] {
		return fmt.Errorf("%w: %q", ErrAIToolInvalidRole, t.RequiredRole)
	}
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	// ArgsJSON is the raw stored value. Normalise to "[]" so the
	// column is never NULL and the UI can rely on parseable JSON.
	if t.ArgsJSON == "" {
		t.ArgsJSON = "[]"
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO ai_tools
            (id, team_id, name, slug, description, endpoint, protocol,
             command, args_json, required_role, allowed_projects_json,
             timeout_seconds, enabled, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.TeamID, t.Name, t.Slug, t.Description, t.Endpoint, t.Protocol,
		t.Command, t.ArgsJSON, t.RequiredRole, t.AllowedProjectsJSON,
		t.TimeoutSeconds, boolToInt64(t.Enabled), t.CreatedAt, t.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("CreateAITool: duplicate (team_id, slug)")
		}
		return fmt.Errorf("insert ai_tool: %w", err)
	}
	return nil
}

// GetAIToolByID returns one row.
func (db *DB) GetAIToolByID(ctx context.Context, id string) (*AITool, error) {
	const q = `SELECT id, team_id, name, slug, description, endpoint, protocol,
                      command, args_json, required_role, allowed_projects_json,
                      timeout_seconds, enabled, created_at, updated_at
                 FROM ai_tools WHERE id = ?`
	row := db.QueryRowContext(ctx, q, id)
	return scanAITool(row)
}

// GetAIToolBySlug is a convenience for the team-scoped UI route.
func (db *DB) GetAIToolBySlug(ctx context.Context, teamID, slug string) (*AITool, error) {
	const q = `SELECT id, team_id, name, slug, description, endpoint, protocol,
                      command, args_json, required_role, allowed_projects_json,
                      timeout_seconds, enabled, created_at, updated_at
                 FROM ai_tools WHERE team_id = ? AND slug = ?`
	row := db.QueryRowContext(ctx, q, teamID, slug)
	return scanAITool(row)
}

// ListAITools returns all rows for a team, ordered by name.
func (db *DB) ListAITools(ctx context.Context, teamID string) ([]*AITool, error) {
	q := `SELECT id, team_id, name, slug, description, endpoint, protocol,
                 command, args_json, required_role, allowed_projects_json,
                 timeout_seconds, enabled, created_at, updated_at
            FROM ai_tools`
	args := []any{}
	if teamID != "" {
		q += ` WHERE team_id = ?`
		args = append(args, teamID)
	}
	q += ` ORDER BY name ASC`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list ai_tools: %w", err)
	}
	defer rows.Close()
	out := make([]*AITool, 0)
	for rows.Next() {
		t, err := scanAITool(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateAITool applies a partial update. protocol and required_role
// (if non-empty) are re-validated.
func (db *DB) UpdateAITool(ctx context.Context, id string, fields AIToolUpdate) error {
	if fields.Protocol != nil && !validProtocols[*fields.Protocol] {
		return fmt.Errorf("%w: %q", ErrAIToolInvalidProtocol, *fields.Protocol)
	}
	if fields.RequiredRole != nil && !validRequiredRoles[*fields.RequiredRole] {
		return fmt.Errorf("%w: %q", ErrAIToolInvalidRole, *fields.RequiredRole)
	}
	// ArgsJSON / AllowedProjectsJSON: caller passes the raw JSON
	// string. nil → leave unchanged; "" → clear to "[]" or ""
	// respectively. We use sql.NullString for COALESCE.
	argsJSON := sql.NullString{}
	if fields.ArgsJSON != nil {
		v := *fields.ArgsJSON
		if v == "" {
			v = "[]"
		}
		argsJSON = sql.NullString{String: v, Valid: true}
	}
	allowedJSON := sql.NullString{}
	if fields.AllowedProjectsJSON != nil {
		allowedJSON = sql.NullString{String: *fields.AllowedProjectsJSON, Valid: true}
	}
	enabledInt := sql.NullInt64{}
	if fields.Enabled != nil {
		enabledInt = sql.NullInt64{Int64: boolToInt64(*fields.Enabled), Valid: true}
	}
	timeout := sql.NullInt64{}
	if fields.TimeoutSeconds != nil {
		timeout = sql.NullInt64{Int64: int64(*fields.TimeoutSeconds), Valid: true}
	}
	res, err := db.ExecContext(ctx,
		`UPDATE ai_tools SET
            name         = COALESCE(?, name),
            description  = COALESCE(?, description),
            endpoint     = COALESCE(?, endpoint),
            protocol     = COALESCE(?, protocol),
            command      = COALESCE(?, command),
            args_json    = COALESCE(?, args_json),
            required_role= COALESCE(?, required_role),
            allowed_projects_json = COALESCE(?, allowed_projects_json),
            timeout_seconds = COALESCE(?, timeout_seconds),
            enabled      = COALESCE(?, enabled),
            updated_at   = ?
          WHERE id = ?`,
		nullableString(fields.Name), nullableString(fields.Description),
		nullableString(fields.Endpoint), nullableString(fields.Protocol),
		nullableString(fields.Command), argsJSON, nullableString(fields.RequiredRole),
		allowedJSON, timeout, enabledInt, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("update ai_tool: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAIToolNotFound
	}
	return nil
}

// DeleteAITool hard-deletes the row. We don't soft-delete because
// invocations reference ai_tools(id) ON DELETE RESTRICT; if a tool
// has been used it must stay. Caller checks first.
func (db *DB) DeleteAITool(ctx context.Context, id string) error {
	// Count invocations referencing this tool; if > 0 refuse.
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ai_tool_invocations WHERE tool_id = ?`, id,
	).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("DeleteAITool: tool has %d invocation(s); disable instead", n)
	}
	res, err := db.ExecContext(ctx, `DELETE FROM ai_tools WHERE id = ?`, id)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("DeleteAITool: tool in use")
		}
		return fmt.Errorf("delete ai_tool: %w", err)
	}
	rn, _ := res.RowsAffected()
	if rn == 0 {
		return ErrAIToolNotFound
	}
	return nil
}

// AIToolUpdate is the partial-update struct. nil = "leave
// unchanged". ArgsJSON / AllowedProjectsJSON carry the raw JSON
// string (as stored in the column) so a caller can set either an
// explicit array ("[\"a\",\"b\"]") or clear to "[]" / "". The
// HTTP layer marshals from []string into these fields.
type AIToolUpdate struct {
	Name                *string
	Description         *string
	Endpoint            *string
	Protocol            *string
	Command             *string
	ArgsJSON            *string
	RequiredRole        *string
	AllowedProjectsJSON *string
	TimeoutSeconds      *int
	Enabled             *bool
}

// ----------------------------------------------------------------------------
// Invocations (audit log)
// ----------------------------------------------------------------------------

// CreateInvocation inserts a pending row at the start of execution.
// Caller flips status to running/success/error/timeout via
// UpdateInvocationStatus.
func (db *DB) CreateInvocation(ctx context.Context, inv *AIToolInvocation) error {
	if inv.ID == "" {
		return errors.New("CreateInvocation: id required")
	}
	if inv.ToolID == "" || inv.UserID == "" || inv.TeamID == "" ||
		inv.ProjectID == "" || inv.ModuleID == "" {
		return errors.New("CreateInvocation: tool_id, user_id, team_id, project_id, module_id required")
	}
	if inv.StartedAt.IsZero() {
		inv.StartedAt = time.Now().UTC()
	}
	if inv.Status == "" {
		inv.Status = "pending"
	}
	inputJSON := inv.InputJSON
	if inputJSON == "" {
		inputJSON = "{}"
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO ai_tool_invocations
            (id, tool_id, user_id, team_id, project_id, module_id, working_dir,
             input_json, output_json, status, started_at, finished_at, error)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?, NULL, '')`,
		inv.ID, inv.ToolID, inv.UserID, inv.TeamID, inv.ProjectID, inv.ModuleID,
		inv.WorkingDir, inputJSON, inv.Status, inv.StartedAt,
	); err != nil {
		return fmt.Errorf("insert invocation: %w", err)
	}
	return nil
}

// UpdateInvocationStatus flips status + finished_at + (optionally)
// output_json and error. status must be one of running/success/
// error/timeout.
func (db *DB) UpdateInvocationStatus(ctx context.Context, id, status, outputJSON, errMsg string) error {
	switch status {
	case "running", "success", "error", "timeout":
	default:
		return fmt.Errorf("UpdateInvocationStatus: invalid status %q", status)
	}
	if outputJSON == "" {
		outputJSON = ""
	}
	res, err := db.ExecContext(ctx,
		`UPDATE ai_tool_invocations SET
            status = ?, output_json = ?, finished_at = ?, error = ?
          WHERE id = ?`,
		status, outputJSON, time.Now().UTC(), errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("update invocation: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("UpdateInvocationStatus: %s not found", id)
	}
	return nil
}

// GetInvocation returns one invocation by id.
func (db *DB) GetInvocation(ctx context.Context, id string) (*AIToolInvocation, error) {
	const q = `SELECT id, tool_id, user_id, team_id, project_id, module_id, working_dir,
                      input_json, output_json, status, started_at, finished_at, error
                 FROM ai_tool_invocations WHERE id = ?`
	row := db.QueryRowContext(ctx, q, id)
	return scanInvocation(row)
}

// ListInvocationsFilter bundles the list query.
type ListInvocationsFilter struct {
	ToolID    string
	UserID    string
	TeamID    string
	ProjectID string
	ModuleID  string
	Status    string
	Limit     int
}

// ListInvocations returns rows matching the filter, newest first.
// limit defaults to 50.
func (db *DB) ListInvocations(ctx context.Context, f ListInvocationsFilter) ([]*AIToolInvocation, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	q := `SELECT id, tool_id, user_id, team_id, project_id, module_id, working_dir,
                 input_json, output_json, status, started_at, finished_at, error
            FROM ai_tool_invocations`
	var (
		clauses []string
		args    []any
	)
	if f.ToolID != "" {
		clauses = append(clauses, "tool_id = ?")
		args = append(args, f.ToolID)
	}
	if f.UserID != "" {
		clauses = append(clauses, "user_id = ?")
		args = append(args, f.UserID)
	}
	if f.TeamID != "" {
		clauses = append(clauses, "team_id = ?")
		args = append(args, f.TeamID)
	}
	if f.ProjectID != "" {
		clauses = append(clauses, "project_id = ?")
		args = append(args, f.ProjectID)
	}
	if f.ModuleID != "" {
		clauses = append(clauses, "module_id = ?")
		args = append(args, f.ModuleID)
	}
	if f.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, f.Status)
	}
	if len(clauses) > 0 {
		q += " WHERE " + strings.Join(clauses, " AND ")
	}
	q += ` ORDER BY started_at DESC LIMIT ?`
	args = append(args, f.Limit)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list invocations: %w", err)
	}
	defer rows.Close()
	out := make([]*AIToolInvocation, 0)
	for rows.Next() {
		inv, err := scanInvocation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// ----------------------------------------------------------------------------
// RBAC helpers
// ----------------------------------------------------------------------------

// AIToolAccessibleByUser returns (allowed, mustBeAtLeastRole).
// Admin/lead/owner/maintainer are always allowed; developer only
// when the tool's required_role is "developer".
func (db *DB) AIToolAccessibleByUser(ctx context.Context, toolID, userID, role string) (bool, error) {
	t, err := db.GetAIToolByID(ctx, toolID)
	if err != nil {
		if errors.Is(err, ErrAIToolNotFound) {
			return false, nil
		}
		return false, err
	}
	switch role {
	case "admin", "lead":
		return true, nil
	}
	// Otherwise: must be a team member AND role ≥ tool.required_role.
	isMember, err := db.IsTeamMember(ctx, t.TeamID, userID)
	if err != nil {
		return false, err
	}
	if !isMember {
		return false, nil
	}
	// Role gating: required_role is admin|lead|developer. Viewer
	// is not in the allowed set — they can't run tools.
	switch t.RequiredRole {
	case "admin":
		return role == "admin", nil
	case "lead":
		return role == "admin" || role == "lead", nil
	case "developer":
		return role == "admin" || role == "lead" || role == "developer", nil
	}
	return false, nil
}

// AIToolProjectAllowed checks the allowed_projects_json list.
// Empty list (or "") means "all projects in the team are allowed".
func (db *DB) AIToolProjectAllowed(ctx context.Context, toolID, projectID string) (bool, error) {
	t, err := db.GetAIToolByID(ctx, toolID)
	if err != nil {
		return false, err
	}
	allowed, err := decodeAllowedProjects(t.AllowedProjectsJSON)
	if err != nil {
		return false, err
	}
	if len(allowed) == 0 {
		return true, nil
	}
	for _, p := range allowed {
		if p == projectID {
			return true, nil
		}
	}
	return false, nil
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// encodeAllowedProjects is the inverse of decodeAllowedProjects.
// An empty/nil slice → "" (which means "all projects in the team"
// downstream).
func encodeAllowedProjects(p []string) (string, error) {
	if len(p) == 0 {
		return "", nil
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeAllowedProjects(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// optionalJSONArray converts a *[]string into a sql.NullString for
// use with COALESCE: nil → NULL (leave unchanged); non-nil → JSON
func scanAITool(s scanner) (*AITool, error) {
	t := &AITool{}
	var enabled int
	if err := s.Scan(
		&t.ID, &t.TeamID, &t.Name, &t.Slug, &t.Description, &t.Endpoint, &t.Protocol,
		&t.Command, &t.ArgsJSON, &t.RequiredRole, &t.AllowedProjectsJSON,
		&t.TimeoutSeconds, &enabled, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAIToolNotFound
		}
		return nil, err
	}
	t.Enabled = enabled != 0
	return t, nil
}

func scanInvocation(s scanner) (*AIToolInvocation, error) {
	inv := &AIToolInvocation{}
	var finished sql.NullTime
	if err := s.Scan(
		&inv.ID, &inv.ToolID, &inv.UserID, &inv.TeamID, &inv.ProjectID, &inv.ModuleID,
		&inv.WorkingDir, &inv.InputJSON, &inv.OutputJSON, &inv.Status,
		&inv.StartedAt, &finished, &inv.Error,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("invocation not found")
		}
		return nil, err
	}
	if finished.Valid {
		v := finished.Time.UTC()
		inv.FinishedAt = &v
	}
	inv.StartedAt = inv.StartedAt.UTC()
	return inv, nil
}

// boolToInt64 is the int64-returning variant used by ExecContext
// calls that want a nullable column.
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
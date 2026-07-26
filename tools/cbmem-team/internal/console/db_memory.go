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
// v2 (M5) Memory / Template / Summarize+Distill Task storage layer.
// =============================================================================
//
// Memories live under a leaf module (application-level guard via
// RequireLeafModule). Each memory has:
//   - team_id, project_id, module_id (FK chain)
//   - user_id        (creator)
//   - template_id    (optional; nullable FK; SET NULL on delete)
//   - hall           ('facts' | 'events' | 'discoveries' | 'preferences' | 'advice')
//   - tags_json      (JSON array of strings)
//
// Memory templates are seeded on migrate (5 built-ins). Summarize
// and distill tasks were part of v1 and stay compatible — the v2
// HTTP wrappers just stamp user_id from JWT instead of admin cookie.
// =============================================================================

var (
	ErrMemoryNotFound  = errors.New("memory not found")
	ErrTemplateMissing = errors.New("memory template not found")
	ErrInvalidHall     = errors.New("invalid hall")
)

// validHalls is the closed set of hall names. Anything else is
// rejected at the storage boundary so the CHECK constraint and the
// UI stay in sync.
var validHalls = map[string]bool{
	"facts":       true,
	"events":      true,
	"discoveries": true,
	"preferences": true,
	"advice":      true,
}

// Default hall when callers don't specify one.
const defaultHall = "facts"

// ----------------------------------------------------------------------------
// Memory templates (read-only for non-builtin rows; built-ins are
// immutable via the API so the v2 migration can re-seed safely).
// ----------------------------------------------------------------------------

// ListMemoryTemplates returns templates ordered with built-ins
// first, then by name. If builtinOnly is true, only the seeded
// templates are returned.
func (db *DB) ListMemoryTemplates(ctx context.Context, builtinOnly bool) ([]*MemoryTemplate, error) {
	q := `SELECT id, name, description, fields_json, body_template, is_builtin
            FROM ai_memories_templates`
	if builtinOnly {
		q += ` WHERE is_builtin = 1`
	}
	q += ` ORDER BY is_builtin DESC, name ASC`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()
	out := make([]*MemoryTemplate, 0)
	for rows.Next() {
		t, err := scanMemoryTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetMemoryTemplate returns one template by id.
func (db *DB) GetMemoryTemplate(ctx context.Context, id int64) (*MemoryTemplate, error) {
	const q = `SELECT id, name, description, fields_json, body_template, is_builtin
                 FROM ai_memories_templates WHERE id = ?`
	row := db.QueryRowContext(ctx, q, id)
	return scanMemoryTemplate(row)
}

// GetMemoryTemplateByName returns one template by its unique name.
func (db *DB) GetMemoryTemplateByName(ctx context.Context, name string) (*MemoryTemplate, error) {
	const q = `SELECT id, name, description, fields_json, body_template, is_builtin
                 FROM ai_memories_templates WHERE name = ?`
	row := db.QueryRowContext(ctx, q, name)
	return scanMemoryTemplate(row)
}

// CreateMemoryTemplate inserts a user-supplied template (non-builtin).
// The five seeded rows are owned by the migration and cannot be
// re-created here. The id is auto-generated (AUTO_INCREMENT); on
// success t.ID is populated with the new row id.
func (db *DB) CreateMemoryTemplate(ctx context.Context, t *MemoryTemplate) error {
	if t.Name == "" {
		return errors.New("CreateMemoryTemplate: name required")
	}
	if t.BodyTemplate == "" {
		return errors.New("CreateMemoryTemplate: body_template required")
	}
	res, err := db.ExecContext(ctx,
		`INSERT INTO ai_memories_templates (name, description, fields_json, body_template, is_builtin)
         VALUES (?, ?, ?, ?, 0)`,
		t.Name, t.Description, t.FieldsJSON, t.BodyTemplate,
	); if err != nil {
		if isUniqueViolation(err) {
			return ErrModuleExists
		}
		return fmt.Errorf("insert template: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}
	t.ID = id
	t.IsBuiltin = false
	return nil
}

// scanMemoryTemplate handles both *sql.Row and *sql.Rows since the
// SELECT list is the same in both call sites.
func scanMemoryTemplate(s scanner) (*MemoryTemplate, error) {
	t := &MemoryTemplate{}
	if err := s.Scan(&t.ID, &t.Name, &t.Description, &t.FieldsJSON, &t.BodyTemplate, &t.IsBuiltin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTemplateMissing
		}
		return nil, err
	}
	return t, nil
}

// ----------------------------------------------------------------------------
// Memories
// ----------------------------------------------------------------------------

// MemoryFilter bundles the list query. Zero value matches every
// non-deleted row.
type MemoryFilter struct {
	TeamID     string
	ProjectID  string
	ModuleID   string
	UserID     string
	Hall       string
	TemplateID int64
	Tag        string // optional — substring match against the JSON-encoded tags array
	Limit      int
	Offset     int
}

// NewMemoryFilter returns a filter with sensible defaults.
func NewMemoryFilter() MemoryFilter {
	return MemoryFilter{Limit: 50}
}

// CreateMemory inserts a memory. moduleID must reference a leaf
// module (caller validates first via RequireLeafModule). The
// template_id is optional; pass "" to skip.
//
// hall is normalised to 'facts' if empty. tags are JSON-encoded
// inline (empty slice → "[]").
func (db *DB) CreateMemory(ctx context.Context, m *Memory) error {
	if m.ID == "" {
		return errors.New("CreateMemory: id required")
	}
	if m.TeamID == "" || m.ProjectID == "" || m.ModuleID == "" {
		return errors.New("CreateMemory: team_id, project_id, module_id required")
	}
	if m.UserID == "" {
		return errors.New("CreateMemory: user_id required")
	}
	if strings.TrimSpace(m.Title) == "" {
		return errors.New("CreateMemory: title required")
	}
	if m.Hall == "" {
		m.Hall = defaultHall
	}
	if !validHalls[m.Hall] {
		return fmt.Errorf("%w: %q", ErrInvalidHall, m.Hall)
	}
	if err := db.RequireLeafModule(ctx, m.ModuleID); err != nil {
		return fmt.Errorf("CreateMemory: %w", err)
	}
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	tagsJSON, err := encodeTags(m.Tags)
	if err != nil {
		return fmt.Errorf("encode tags: %w", err)
	}
	// template_id is nullable. Zero value → NULL.
	var templateID sql.NullInt64
	if m.TemplateID != 0 {
		templateID = sql.NullInt64{Int64: m.TemplateID, Valid: true}
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO ai_memories
            (id, team_id, project_id, module_id, user_id, title, content,
             template_id, tags_json, hall, created_at, updated_at, deleted)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		m.ID, m.TeamID, m.ProjectID, m.ModuleID, m.UserID, m.Title, m.Content,
		templateID, tagsJSON, m.Hall, m.CreatedAt, m.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert memory: %w", err)
	}
	m.TagsJSON = tagsJSON
	return nil
}

// GetMemory returns one memory by id. Soft-deleted rows return
// ErrMemoryNotFound.
func (db *DB) GetMemory(ctx context.Context, id string) (*Memory, error) {
	const q = `SELECT id, team_id, project_id, module_id, user_id, title, content,
                      template_id, tags_json, hall, created_at, updated_at, deleted
                 FROM ai_memories WHERE id = ? AND deleted = 0`
	row := db.QueryRowContext(ctx, q, id)
	return scanMemory(row)
}

// ListMemories applies the filter and returns the rows + total.
// Pagination defaults: limit=50, offset=0.
func (db *DB) ListMemories(ctx context.Context, f MemoryFilter) ([]*Memory, int, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	where, args := memoriesWhere(f)
	q := `SELECT id, team_id, project_id, module_id, user_id, title, content,
                 template_id, tags_json, hall, created_at, updated_at, deleted
            FROM ai_memories` + where +
		` ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, f.Limit, f.Offset)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()
	out := make([]*Memory, 0)
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	cntQ := "SELECT COUNT(*) FROM ai_memories" + where
	var total int
	if err := db.QueryRowContext(ctx, cntQ, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count memories: %w", err)
	}
	return out, total, nil
}

// UpdateMemory applies a partial update. Fields can be nil to skip.
// module_id changes are rejected (move memories via create+delete so
// leaf semantics are re-checked at the storage layer).
func (db *DB) UpdateMemory(ctx context.Context, id string, title, content, hall *string, tags []string) error {
	if hall != nil && !validHalls[*hall] {
		return fmt.Errorf("%w: %q", ErrInvalidHall, *hall)
	}
	now := time.Now().UTC()
	var tagsJSON sql.NullString
	if tags != nil {
		j, err := encodeTags(tags)
		if err != nil {
			return err
		}
		tagsJSON = sql.NullString{String: j, Valid: true}
	}
	res, err := db.ExecContext(ctx,
		`UPDATE ai_memories
            SET title    = COALESCE(?, title),
                content  = COALESCE(?, content),
                hall     = COALESCE(?, hall),
                tags_json= COALESCE(?, tags_json),
                updated_at = ?
          WHERE id = ? AND deleted = 0`,
		nullableString(title), nullableString(content), nullableString(hall),
		tagsJSON, now, id,
	)
	if err != nil {
		return fmt.Errorf("update memory: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrMemoryNotFound
	}
	return nil
}

// SoftDeleteMemory flips deleted=1.
func (db *DB) SoftDeleteMemory(ctx context.Context, id string) error {
	res, err := db.ExecContext(ctx,
		`UPDATE ai_memories SET deleted = 1, updated_at = ? WHERE id = ? AND deleted = 0`,
		time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrMemoryNotFound
	}
	return nil
}

// CountMemoriesByModule counts non-deleted memories for a single
// module — used by the modules tree enrichment so memory_count
// stays fresh.
func (db *DB) CountMemoriesUnderModule(ctx context.Context, moduleID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ai_memories WHERE module_id = ? AND deleted = 0`,
		moduleID,
	).Scan(&n)
	return n, err
}

// MemoryAccessibleByUser mirrors ModuleAccessibleByUser: admin OR
// team member can read; team owner/lead/maintainer can write.
func (db *DB) MemoryAccessibleByUser(ctx context.Context, memoryID, userID, role string) (bool, error) {
	if role == string(RoleAdmin) {
		return true, nil
	}
	var teamID string
	err := db.QueryRowContext(ctx,
		`SELECT team_id FROM ai_memories WHERE id = ? AND deleted = 0`, memoryID,
	).Scan(&teamID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return db.IsTeamMember(ctx, teamID, userID)
}

// ----------------------------------------------------------------------------
// Summarize / Distill tasks (v1 schema; storage only — the v2 HTTP
// wrappers re-use SummarizeHandler / DistillHandler with a JWT-stamped
// user_id).
// ----------------------------------------------------------------------------

// CreateSummarizeTask inserts a pending summarize_tasks row. The
// existing SummarizeHandler synchronously executes the LLM call and
// flips status to done/error; this helper is just the initial INSERT
// for the JWT path.
func (db *DB) CreateSummarizeTask(ctx context.Context, t *SummarizeTask) error {
	if t.ID == "" {
		return errors.New("CreateSummarizeTask: id required")
	}
	if t.Depth == "" {
		t.Depth = "deep"
	}
	if !validSummarizeDepth(t.Depth) {
		return fmt.Errorf("invalid depth: %q", t.Depth)
	}
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.Status == "" {
		t.Status = "pending"
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO ai_summarize_tasks (id, user_id, source_ids, depth, target_wing, status, result_json, created_at)
         VALUES (?, ?, ?, ?, ?, ?, '', ?)`,
		t.ID, t.UserID, t.SourceIDs, t.Depth, t.TargetWing, t.Status, t.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert summarize task: %w", err)
	}
	return nil
}

// GetSummarizeTask returns one summarize_tasks row.
func (db *DB) GetSummarizeTask(ctx context.Context, id string) (*SummarizeTask, error) {
	const q = `SELECT id, IFNULL(user_id,''), source_ids, depth, target_wing, status,
                      result_json, created_at, finished_at
                 FROM ai_summarize_tasks WHERE id = ?`
	row := db.QueryRowContext(ctx, q, id)
	return scanSummarizeTask(row)
}

// ListSummarizeTasks returns the most recent n rows. limit defaults to 50.
func (db *DB) ListSummarizeTasks(ctx context.Context, userID string, limit int) ([]*SummarizeTask, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	q := `SELECT id, IFNULL(user_id,''), source_ids, depth, target_wing, status,
                  result_json, created_at, finished_at
             FROM ai_summarize_tasks`
	args := []any{}
	if userID != "" {
		q += ` WHERE user_id = ?`
		args = append(args, userID)
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*SummarizeTask, 0)
	for rows.Next() {
		t, err := scanSummarizeTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CreateDistillTask inserts a pending distill_tasks row.
func (db *DB) CreateDistillTask(ctx context.Context, t *DistillTask) error {
	if t.ID == "" {
		return errors.New("CreateDistillTask: id required")
	}
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.Status == "" {
		t.Status = "pending"
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO ai_distill_tasks (id, user_id, source_ids, rules_json, status, result_json, mempalace_synced, target_wing, created_at)
         VALUES (?, ?, ?, ?, ?, '', 0, ?, ?)`,
		t.ID, t.UserID, t.SourceIDs, t.RulesJSON, t.Status, t.TargetWing, t.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert distill task: %w", err)
	}
	return nil
}

// GetDistillTask returns one distill_tasks row.
func (db *DB) GetDistillTask(ctx context.Context, id string) (*DistillTask, error) {
	const q = `SELECT id, IFNULL(user_id,''), source_ids, rules_json, status, result_json,
                      mempalace_synced, target_wing, created_at, finished_at
                 FROM ai_distill_tasks WHERE id = ?`
	row := db.QueryRowContext(ctx, q, id)
	return scanDistillTask(row)
}

// ListDistillTasks returns the most recent n rows. limit defaults to 50.
func (db *DB) ListDistillTasks(ctx context.Context, userID string, limit int) ([]*DistillTask, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	q := `SELECT id, IFNULL(user_id,''), source_ids, rules_json, status, result_json,
                  mempalace_synced, target_wing, created_at, finished_at
             FROM ai_distill_tasks`
	args := []any{}
	if userID != "" {
		q += ` WHERE user_id = ?`
		args = append(args, userID)
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*DistillTask, 0)
	for rows.Next() {
		t, err := scanDistillTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func validSummarizeDepth(d string) bool {
	switch d {
	case "shallow", "deep", "expert":
		return true
	}
	return false
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

func memoriesWhere(f MemoryFilter) (string, []any) {
	var (
		clauses []string
		args    []any
	)
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
	if f.UserID != "" {
		clauses = append(clauses, "user_id = ?")
		args = append(args, f.UserID)
	}
	if f.Hall != "" {
		clauses = append(clauses, "hall = ?")
		args = append(args, f.Hall)
	}
	if f.TemplateID != 0 {
		clauses = append(clauses, "template_id = ?")
		args = append(args, f.TemplateID)
	}
	if f.Tag != "" {
		// tags_json is stored as a JSON array string; LIKE works for
		// the simple substring case (no JSON1 needed). The query is
		// O(n) per row but n is bounded by Limit so it's fine.
		clauses = append(clauses, "tags_json LIKE ?")
		args = append(args, "%"+escapeLike(f.Tag)+"%")
	}
	clauses = append(clauses, "deleted = 0")
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// escapeLike escapes SQL LIKE metacharacters in the input. Without
// it, a tag containing "%" or "_" would silently match more rows
// than intended.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

func encodeTags(tags []string) (string, error) {
	if tags == nil {
		return "[]", nil
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeTags(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}

func scanMemory(s scanner) (*Memory, error) {
	m := &Memory{}
	var templateID sql.NullInt64
	var tagsJSON string
	var deleted int
	if err := s.Scan(
		&m.ID, &m.TeamID, &m.ProjectID, &m.ModuleID, &m.UserID, &m.Title, &m.Content,
		&templateID, &tagsJSON, &m.Hall, &m.CreatedAt, &m.UpdatedAt, &deleted,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemoryNotFound
		}
		return nil, err
	}
	if templateID.Valid {
		m.TemplateID = templateID.Int64
	}
	m.Tags = decodeTags(tagsJSON)
	m.Deleted = deleted != 0
	return m, nil
}

func scanSummarizeTask(s scanner) (*SummarizeTask, error) {
	t := &SummarizeTask{}
	var result sql.NullString
	var finished sql.NullTime
	if err := s.Scan(
		&t.ID, &t.UserID, &t.SourceIDs, &t.Depth, &t.TargetWing, &t.Status,
		&result, &t.CreatedAt, &finished,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemoryNotFound
		}
		return nil, err
	}
	if result.Valid {
		t.ResultJSON = result.String
	}
	if finished.Valid {
		v := finished.Time.UTC()
		t.FinishedAt = &v
	}
	t.CreatedAt = t.CreatedAt.UTC()
	return t, nil
}

func scanDistillTask(s scanner) (*DistillTask, error) {
	t := &DistillTask{}
	var result sql.NullString
	var finished sql.NullTime
	var synced int
	if err := s.Scan(
		&t.ID, &t.UserID, &t.SourceIDs, &t.RulesJSON, &t.Status, &result,
		&synced, &t.TargetWing, &t.CreatedAt, &finished,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemoryNotFound
		}
		return nil, err
	}
	if result.Valid {
		t.ResultJSON = result.String
	}
	if finished.Valid {
		v := finished.Time.UTC()
		t.FinishedAt = &v
	}
	t.MemPalaceSynced = synced != 0
	t.CreatedAt = t.CreatedAt.UTC()
	return t, nil
}

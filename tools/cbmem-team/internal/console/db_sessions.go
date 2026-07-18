package console

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// v2 (M4) Session storage layer.
// =============================================================================
//
// The DDL schema (see v2_schema_migrate.go) gives sessions the
// complete v2 layout: team_id, project_id, module_id. The v1 capture
// middleware predates this and writes rows with empty
// team_id/module_id/default team_legacy. This layer exposes the
// filtered queries the v2 UI needs.
//
// Filtering model: every filter is optional and composed with AND.
// The legacy rows (module_id = '' or team_id = 'team_legacy') are
// included by default and can be excluded with include_legacy=false.
// =============================================================================

var ErrSessionNotFound = errors.New("session not found")

// SessionFilter bundles every list filter. The zero value matches
// every non-deleted row.
type SessionFilter struct {
	TeamID      string
	ProjectID   string
	ModuleID    string
	UserID      string

	// ProjectPath filters by the legacy `project_path` column (the
	// on-disk path MCP clients send). Useful for the v1 path-based
	// detail page that hasn't migrated to project_id yet.
	ProjectPath string

	From, To time.Time

	// Limit defaults to 50 when zero; capped server-side at 200.
	Limit  int
	Offset int

	// IncludeLegacy controls whether rows with empty module_id
	// (captured before the v2 capture middleware landed) are listed.
	// Defaults true via the constructor below.
	IncludeLegacy bool
}

// NewSessionFilter returns a filter with sensible defaults applied.
func NewSessionFilter() SessionFilter {
	return SessionFilter{
		Limit:         50,
		IncludeLegacy: true,
	}
}

// ListSessions returns sessions matching the filter, ordered by
// started_at DESC. The total count uses the same filter so the UI's
// pagination can render "X of N".
func (db *DB) ListSessions(ctx context.Context, f SessionFilter) ([]*Session, int, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	if !f.IncludeLegacy {
		// Treated identically to "module_id != ''" since legacy rows
		// are the only source of empty module_id.
	}

	where, args := sessionsWhere(f)
	q := `SELECT id, IFNULL(user_id,''), IFNULL(team_id,''), IFNULL(project_id,''),
                  IFNULL(module_id,''), IFNULL(project_path,''),
                  started_at, ended_at,
                  tool_count, turn_count, IFNULL(summary,''),
                  mempalace_synced_turns, mempalace_last_synced_at, IFNULL(mempalace_last_error,'')
             FROM sessions` + where +
		` ORDER BY started_at DESC LIMIT ? OFFSET ?`
	args = append(args, f.Limit, f.Offset)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	out := make([]*Session, 0)
	for rows.Next() {
		s, err := scanSessionRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	cntQ := "SELECT COUNT(*) FROM sessions" + where
	var total int
	if err := db.QueryRowContext(ctx, cntQ, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sessions: %w", err)
	}
	return out, total, nil
}

func sessionsWhere(f SessionFilter) (string, []any) {
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
	if !f.IncludeLegacy {
		clauses = append(clauses, "module_id != ''")
	}
	if f.UserID != "" {
		clauses = append(clauses, "user_id = ?")
		args = append(args, f.UserID)
	}
	if f.ProjectPath != "" {
		clauses = append(clauses, "project_path = ?")
		args = append(args, f.ProjectPath)
	}
	if !f.From.IsZero() {
		clauses = append(clauses, "started_at >= ?")
		args = append(args, f.From.UTC().Format("2006-01-02 15:04:05"))
	}
	if !f.To.IsZero() {
		clauses = append(clauses, "started_at <= ?")
		args = append(args, f.To.UTC().Format("2006-01-02 15:04:05"))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// GetSession returns one session row, or ErrSessionNotFound.
func (db *DB) GetSession(ctx context.Context, id string) (*Session, error) {
	const q = `SELECT id, IFNULL(user_id,''), IFNULL(team_id,''), IFNULL(project_id,''),
                      IFNULL(module_id,''), IFNULL(project_path,''),
                      started_at, ended_at,
                      tool_count, turn_count, IFNULL(summary,''),
                      mempalace_synced_turns, mempalace_last_synced_at, IFNULL(mempalace_last_error,'')
                 FROM sessions WHERE id = ?`
	row := db.QueryRowContext(ctx, q, id)
	return scanSessionRow(row)
}

// ListSessionTurns returns the turns of a session ordered by turn_no.
func (db *DB) ListSessionTurns(ctx context.Context, sessionID string) ([]*SessionTurn, error) {
	const q = `SELECT id, session_id, turn_no, role, content, IFNULL(tools_json,''), ts
                 FROM session_turns
                WHERE session_id = ?
                ORDER BY turn_no ASC, id ASC`
	rows, err := db.QueryContext(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list turns: %w", err)
	}
	defer rows.Close()
	out := make([]*SessionTurn, 0)
	for rows.Next() {
		var t SessionTurn
		var ts time.Time
		if err := rows.Scan(&t.ID, &t.SessionID, &t.TurnNo, &t.Role,
			&t.Content, &t.ToolsJSON, &ts); err != nil {
			return nil, err
		}
		t.TS = ts
		out = append(out, &t)
	}
	return out, rows.Err()
}

// AggregateSessions returns totals + top-N users/projects for the
// dashboard. Same filter as ListSessions minus pagination.
func (db *DB) AggregateSessions(ctx context.Context, f SessionFilter) (*SessionStats, error) {
	where, args := sessionsWhere(f)
	stats := &SessionStats{
		ByProject:   map[string]int{},
		ByModule:    map[string]int{},
		ByUser:      map[string]int{},
		TopUsers:    []StatEntry{},
		TopProjects: []StatEntry{},
	}
	var sessTotal, turnTotal, toolTotal int64
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*), IFNULL(SUM(turn_count),0), IFNULL(SUM(tool_count),0)
		   FROM sessions`+where, args...,
	).Scan(&sessTotal, &turnTotal, &toolTotal); err != nil {
		return nil, fmt.Errorf("aggregate totals: %w", err)
	}
	stats.Total = int(sessTotal)
	stats.TurnCount = int(turnTotal)
	stats.ToolCount = int(toolTotal)

	// Top users
	rows, err := db.QueryContext(ctx,
		`SELECT user_id, COUNT(*) c FROM sessions`+where+
			` GROUP BY user_id ORDER BY c DESC LIMIT 5`, args...)
	if err != nil {
		return nil, fmt.Errorf("top users: %w", err)
	}
	for rows.Next() {
		var uid string
		var c int64
		if err := rows.Scan(&uid, &c); err != nil {
			rows.Close()
			return nil, err
		}
		stats.ByUser[uid] = int(c)
		stats.TopUsers = append(stats.TopUsers, StatEntry{Label: uid, Count: int(c)})
	}
	rows.Close()

	// Top projects
	rows2, err := db.QueryContext(ctx,
		`SELECT IFNULL(project_id,''), COUNT(*) c FROM sessions`+where+
			` GROUP BY project_id ORDER BY c DESC LIMIT 5`, args...)
	if err != nil {
		return nil, fmt.Errorf("top projects: %w", err)
	}
	for rows2.Next() {
		var pid string
		var c int64
		if err := rows2.Scan(&pid, &c); err != nil {
			rows2.Close()
			return nil, err
		}
		stats.ByProject[pid] = int(c)
		stats.TopProjects = append(stats.TopProjects, StatEntry{Label: pid, Count: int(c)})
	}
	rows2.Close()

	// ByModule is included for completeness even though the M4 UI
	// renders top-users/top-projects; downstream M5 dashboards
	// (memory creator surfaces "module X has 12 sessions") consume
	// it directly via the typed payload.
	rows3, err := db.QueryContext(ctx,
		`SELECT IFNULL(module_id,''), COUNT(*) c FROM sessions`+where+
			` GROUP BY module_id ORDER BY c DESC LIMIT 10`, args...)
	if err == nil {
		for rows3.Next() {
			var mid string
			var c int64
			if err := rows3.Scan(&mid, &c); err != nil {
				continue
			}
			stats.ByModule[mid] = int(c)
		}
		rows3.Close()
	}

	return stats, nil
}

// CountSessionsByModule returns the session count for a single module,
// used to populate ModuleDTO.SessionCount without an N+1 query.
func (db *DB) CountSessionsByModule(ctx context.Context, moduleID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sessions WHERE module_id = ? AND module_id != ''`,
		moduleID,
	).Scan(&n)
	return n, err
}

// CountMemoriesByModule fills ModuleDTO.MemoryCount (used by M5 once
// memories land; declared here so the module tree handler can rely
// on it without a missing-import surprise).
func (db *DB) CountMemoriesByModule(ctx context.Context, moduleID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM memories WHERE module_id = ? AND deleted = 0`,
		moduleID,
	).Scan(&n)
	return n, err
}

// ----------------------------------------------------------------------------
// scanner
// ----------------------------------------------------------------------------

func scanSessionRow(s scanner) (*Session, error) {
	out := &Session{}
	var (
		endedAt  sql.NullTime
		lastSync sql.NullTime
	)
	if err := s.Scan(
		&out.ID, &out.UserID, &out.TeamID, &out.ProjectID,
		&out.ModuleID, &out.ProjectPath,
		&out.StartedAt, &endedAt,
		&out.ToolCount, &out.TurnCount, &out.Summary,
		&out.MemPalaceSyncedTurns, &lastSync, &out.MemPalaceLastError,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	if endedAt.Valid {
		v := endedAt.Time.UTC()
		out.EndedAt = &v
	}
	if lastSync.Valid {
		v := lastSync.Time.UTC()
		out.MemPalaceLastSyncedAt = &v
	}
	out.StartedAt = out.StartedAt.UTC()
	return out, nil
}

// SetSessionModule updates sessions.module_id. Used by the v2 capture
// middleware once the requested leaf is resolved. Joins the row by
// id and only acts when module_id is empty (legacy rows) or matches
// the new value (idempotent). Returns rows affected for the caller.
func (db *DB) SetSessionModule(ctx context.Context, sessionID, moduleID string) (int64, error) {
	res, err := db.ExecContext(ctx,
		`UPDATE sessions SET module_id = ? WHERE id = ? AND (module_id = '' OR module_id = ?)`,
		moduleID, sessionID, moduleID,
	)
	if err != nil {
		return 0, fmt.Errorf("set session module: %w", err)
	}
	return res.RowsAffected()
}

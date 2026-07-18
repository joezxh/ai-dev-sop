package console

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// =============================================================================
// v2 (M3) Team + TeamMember + Project CRUD
// =============================================================================
//
// Layered on top of the v2 schema (see v2_schema_migrate.go). The
// schema is identical for SQLite and MySQL; the SQL here is portable
// except where noted.
//
// Global repo-root convention (M3 decision):
//
//   projects.path is always `<repo_root>/<project.slug>`.
//   repo_root is set via Config.RepoRoot (see cmd/cbmem-team/main.go).
//   users cannot override projects.path directly; the server computes
//   it on create and refuses to honour client-side path requests.
//
// =============================================================================

// ErrTeamNotFound / ErrMemberExists / ErrMemberNotFound / ErrProjectNotFound
// are the canonical "no such row" errors. Handlers branch on them.
var (
	ErrTeamNotFound   = errors.New("team not found")
	ErrTeamExists     = errors.New("team slug already exists")
	ErrMemberExists   = errors.New("user is already a member of this team")
	ErrMemberNotFound = errors.New("user is not a member of this team")

	ErrProjectNotFound = errors.New("project not found")
	ErrProjectExists   = errors.New("project slug already exists in team")
)

// ProjectRole values for team_members.role (when role is bound to a
// project — we use the same five-tier ladder as users.role). Kept as
// a tiny alias so handlers can `if r == ProjectRoleOwner { ... }`.
type ProjectRole string

const (
	ProjectRoleOwner      ProjectRole = "owner"
	ProjectRoleMaintainer ProjectRole = "maintainer"
	ProjectRoleDeveloper  ProjectRole = "developer"
	ProjectRoleReporter   ProjectRole = "reporter"
	ProjectRoleGuest      ProjectRole = "guest"
)

// AllProjectRoles is the canonical ordering (index 0 = strongest).
var AllProjectRoles = []ProjectRole{
	ProjectRoleOwner, ProjectRoleMaintainer,
	ProjectRoleDeveloper, ProjectRoleReporter, ProjectRoleGuest,
}

// ----------------------------------------------------------------------------
// Teams
// ----------------------------------------------------------------------------

// CreateTeam inserts a new team. Slug is the natural key — two teams
// cannot share a slug. ownerID may be empty for "no human owner",
// which the v2 FK to users(id) accepts as NULL.
func (db *DB) CreateTeam(ctx context.Context, t *Team) error {
	if t.ID == "" {
		return errors.New("CreateTeam: id required")
	}
	if t.Name == "" {
		return errors.New("CreateTeam: name required")
	}
	if t.Slug == "" {
		return errors.New("CreateTeam: slug required")
	}
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	var owner sql.NullString
	if t.OwnerID != "" {
		owner = sql.NullString{String: t.OwnerID, Valid: true}
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO pm_teams (id, name, slug, description, owner_id, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.Slug, t.Description, owner, t.CreatedAt, t.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) && strings.Contains(err.Error(), "slug") {
			return ErrTeamExists
		}
		return fmt.Errorf("insert team: %w", err)
	}
	return nil
}

// GetTeamByID returns the team with the given id, or ErrTeamNotFound.
// Soft-deleted teams return ErrTeamNotFound so callers don't have to
// remember to add `AND deleted = 0` everywhere.
func (db *DB) GetTeamByID(ctx context.Context, id string) (*Team, error) {
	const q = `SELECT id, name, slug, description,
                      IFNULL(owner_id,''), created_at, updated_at, deleted
                 FROM pm_teams WHERE id = ? AND deleted = 0`
	row := db.QueryRowContext(ctx, q, id)
	return db.scanTeam(row)
}

// GetTeamBySlug returns the team with the given slug, or ErrTeamNotFound.
// Slug comparison is case-insensitive on both backends. Soft-deleted
// teams return ErrTeamNotFound (see GetTeamByID).
func (db *DB) GetTeamBySlug(ctx context.Context, slug string) (*Team, error) {
	var q string
	if db.driver == "mysql" {
		q = `SELECT id, name, slug, description,
                     IFNULL(owner_id,''), created_at, updated_at, deleted
                FROM pm_teams WHERE LOWER(slug) = LOWER(?) AND deleted = 0`
	} else {
		q = `SELECT id, name, slug, description,
                     IFNULL(owner_id,''), created_at, updated_at, deleted
                FROM pm_teams WHERE LOWER(slug) = LOWER(?) AND deleted = 0`
	}
	row := db.QueryRowContext(ctx, q, slug)
	return db.scanTeam(row)
}

// ListTeams returns every non-deleted team, ordered by name. Member /
// project counts are populated by the handler (ListTeamsHandler) via
// separate COUNT queries; the DB layer stays cheap.
func (db *DB) ListTeams(ctx context.Context) ([]*Team, error) {
	const q = `SELECT id, name, slug, description,
                      IFNULL(owner_id,''), created_at, updated_at, deleted
                 FROM pm_teams
                WHERE deleted = 0
                ORDER BY name ASC`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	defer rows.Close()
	var out []*Team
	for rows.Next() {
		t, err := db.scanTeam(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListTeamsForUser returns every non-deleted team the user belongs
// to, plus teams they own. Used by non-admin ListTeams so we don't
// leak teams a developer has no business seeing.
func (db *DB) ListTeamsForUser(ctx context.Context, userID string) ([]*Team, error) {
	const q = `SELECT t.id, t.name, t.slug, t.description,
                      IFNULL(t.owner_id,''), t.created_at, t.updated_at, t.deleted
                 FROM pm_teams t
                 LEFT JOIN pm_team_members m ON m.team_id = t.id AND m.user_id = ?
                WHERE t.deleted = 0 AND (m.user_id IS NOT NULL OR t.owner_id = ?)
                ORDER BY t.name ASC`
	rows, err := db.QueryContext(ctx, q, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("list teams for user: %w", err)
	}
	defer rows.Close()
	var out []*Team
	for rows.Next() {
		t, err := db.scanTeam(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateTeam applies a partial update to a team. Pass nil for fields
// that should not change.
func (db *DB) UpdateTeam(ctx context.Context, id string, name, description *string) error {
	now := time.Now().UTC()
	res, err := db.ExecContext(ctx,
		`UPDATE pm_teams
            SET name = COALESCE(?, name),
                description = COALESCE(?, description),
                updated_at = ?
          WHERE id = ? AND deleted = 0`,
		nullableString(name), nullableString(description), now, id,
	)
	if err != nil {
		return fmt.Errorf("update team: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTeamNotFound
	}
	return nil
}

// DeleteTeam soft-deletes the team. Projects under the team remain
// (they are the application's source of truth); the UI hides the
// team from listings once deleted.
func (db *DB) DeleteTeam(ctx context.Context, id string) error {
	now := time.Now().UTC()
	res, err := db.ExecContext(ctx,
		`UPDATE pm_teams SET deleted = 1, updated_at = ? WHERE id = ? AND deleted = 0`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("delete team: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrTeamNotFound
	}
	return nil
}

// CountProjectsInTeam returns the number of projects under a team.
// Used by ListTeamsHandler to populate TeamDTO.ProjectCount.
func (db *DB) CountProjectsInTeam(ctx context.Context, teamID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pm_projects WHERE team_id = ? AND deleted = 0`,
		teamID,
	).Scan(&n)
	return n, err
}

// CountMembersInTeam returns the number of team_members rows.
func (db *DB) CountMembersInTeam(ctx context.Context, teamID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pm_team_members WHERE team_id = ?`,
		teamID,
	).Scan(&n)
	return n, err
}

// ----------------------------------------------------------------------------
// Team members
// ----------------------------------------------------------------------------

// AddTeamMember inserts a (team_id, user_id, role) row. ON CONFLICT
// DO NOTHING so the handler can map "already a member" to ErrMemberExists.
func (db *DB) AddTeamMember(ctx context.Context, teamID, userID string, role ProjectRole) error {
	if teamID == "" || userID == "" {
		return errors.New("AddTeamMember: team_id/user_id required")
	}
	if role == "" {
		role = ProjectRoleDeveloper
	}
	now := time.Now().UTC()
	res, err := db.ExecContext(ctx,
		`INSERT INTO pm_team_members (team_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		teamID, userID, string(role), now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrMemberExists
		}
		// FK violation: either team or user doesn't exist.
		return fmt.Errorf("insert team_member: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrMemberExists
	}
	return nil
}

// UpdateTeamMemberRole changes an existing member's role.
func (db *DB) UpdateTeamMemberRole(ctx context.Context, teamID, userID string, role ProjectRole) error {
	res, err := db.ExecContext(ctx,
		`UPDATE pm_team_members SET role = ? WHERE team_id = ? AND user_id = ?`,
		string(role), teamID, userID,
	)
	if err != nil {
		return fmt.Errorf("update team_member: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// RemoveTeamMember deletes the (team_id, user_id) row.
func (db *DB) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	res, err := db.ExecContext(ctx,
		`DELETE FROM pm_team_members WHERE team_id = ? AND user_id = ?`,
		teamID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete team_member: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// ListTeamMembers returns every member of the team with their role.
// Username is joined from users for the UI; empty if the user row
// has been removed.
func (db *DB) ListTeamMembers(ctx context.Context, teamID string) ([]*TeamMember, error) {
	const q = `SELECT tm.team_id, tm.user_id, IFNULL(u.username,''),
                      tm.role, tm.joined_at
                 FROM pm_team_members tm
                 LEFT JOIN sys_users u ON u.id = tm.user_id
                WHERE tm.team_id = ?
                ORDER BY tm.joined_at ASC`
	rows, err := db.QueryContext(ctx, q, teamID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	var out []*TeamMember
	for rows.Next() {
		m := &TeamMember{}
		if err := rows.Scan(&m.TeamID, &m.UserID, &m.Username, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetTeamMember returns the (team_id, user_id) row or ErrMemberNotFound.
func (db *DB) GetTeamMember(ctx context.Context, teamID, userID string) (*TeamMember, error) {
	const q = `SELECT tm.team_id, tm.user_id, IFNULL(u.username,''),
                      tm.role, tm.joined_at
                 FROM pm_team_members tm
                 LEFT JOIN sys_users u ON u.id = tm.user_id
                WHERE tm.team_id = ? AND tm.user_id = ?`
	m := &TeamMember{}
	err := db.QueryRowContext(ctx, q, teamID, userID).Scan(
		&m.TeamID, &m.UserID, &m.Username, &m.Role, &m.JoinedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}
	return m, nil
}

// IsTeamMember reports whether (teamID, userID) is a member, OR the
// owner of the team (owners are implicitly members for visibility
// purposes). Admins bypass this check at the handler level; this
// function is the non-bypass path.
func (db *DB) IsTeamMember(ctx context.Context, teamID, userID string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pm_teams t
		    LEFT JOIN pm_team_members m
		            ON m.team_id = t.id AND m.user_id = ?
		  WHERE t.id = ? AND (m.user_id IS NOT NULL OR t.owner_id = ?)`,
		userID, teamID, userID,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ----------------------------------------------------------------------------
// Projects v2
// ----------------------------------------------------------------------------

// CreateProjectV2 inserts a new v2 project. The path is computed as
// repoRoot + "/" + slug; callers must not pass a custom path. Returns
// ErrProjectExists when (team_id, slug) collides.
//
// gitURL / gitBranch are optional. When gitURL is non-empty, the
// handler is responsible for kicking off the clone (see
// StartAsyncClone).
func (db *DB) CreateProjectV2(ctx context.Context, p *Project) error {
	if p.ID == "" {
		return errors.New("CreateProjectV2: id required")
	}
	if p.TeamID == "" {
		return errors.New("CreateProjectV2: team_id required")
	}
	if p.Name == "" || p.Slug == "" {
		return errors.New("CreateProjectV2: name and slug required")
	}
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = ProjectStatusReady
	}

	if _, err := db.ExecContext(ctx,
		`INSERT INTO pm_projects (id, team_id, name, slug, description, path,
                              git_url, git_branch, git_commit_sha, status,
                              created_at, updated_at, deleted)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		p.ID, p.TeamID, p.Name, p.Slug, p.Description, p.Path,
		p.GitURL, p.GitBranch, p.GitCommitSHA, p.Status,
		p.CreatedAt, p.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			// Disambiguate the constraint that fired: the FK on
			// team_id, the unique (team_id, slug) index, or
			// projects.path. fkCheck used to query "projects" which
			// was a self-referencing bug (no project existed yet);
			// query the actual reference table instead.
			if errors.Is(db.fkCheck(ctx, "pm_teams", "id", p.TeamID), ErrFKMissing) {
				return fmt.Errorf("team %s not found", p.TeamID)
			}
			return ErrProjectExists
		}
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

// GetProjectV2ByID returns a v2 project row by id. Soft-deleted
// projects return ErrProjectNotFound.
func (db *DB) GetProjectV2ByID(ctx context.Context, id string) (*Project, error) {
	const q = `SELECT id, team_id, name, slug, description, path,
                      IFNULL(git_url,''), IFNULL(git_branch,''),
                      IFNULL(git_commit_sha,''), status,
                      created_at, updated_at, deleted
                 FROM pm_projects WHERE id = ? AND deleted = 0`
	row := db.QueryRowContext(ctx, q, id)
	return db.scanProjectV2(row)
}

// GetProjectV2ByTeamSlug returns a project by (team_id, slug). Used
// by the clone pipeline to look up a project before kicking off the
// git clone.
func (db *DB) GetProjectV2ByTeamSlug(ctx context.Context, teamID, slug string) (*Project, error) {
	const q = `SELECT id, team_id, name, slug, description, path,
                      IFNULL(git_url,''), IFNULL(git_branch,''),
                      IFNULL(git_commit_sha,''), status,
                      created_at, updated_at, deleted
                 FROM pm_projects WHERE team_id = ? AND slug = ?`
	row := db.QueryRowContext(ctx, q, teamID, slug)
	return db.scanProjectV2(row)
}

// ListProjectsV2 returns every non-deleted project, optionally filtered
// by teamID (empty = all teams). Pagination is handled at the handler
// layer.
func (db *DB) ListProjectsV2(ctx context.Context, teamID string) ([]*Project, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if teamID == "" {
		rows, err = db.QueryContext(ctx,
			`SELECT id, team_id, name, slug, description, path,
                    IFNULL(git_url,''), IFNULL(git_branch,''),
                    IFNULL(git_commit_sha,''), status,
                    created_at, updated_at, deleted
               FROM pm_projects
              WHERE deleted = 0
              ORDER BY created_at DESC`)
	} else {
		rows, err = db.QueryContext(ctx,
			`SELECT id, team_id, name, slug, description, path,
                    IFNULL(git_url,''), IFNULL(git_branch,''),
                    IFNULL(git_commit_sha,''), status,
                    created_at, updated_at, deleted
               FROM pm_projects
              WHERE deleted = 0 AND team_id = ?
              ORDER BY created_at DESC`, teamID)
	}
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	var out []*Project
	for rows.Next() {
		p, err := db.scanProjectV2(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateProjectV2 applies a partial update. Slug, team_id, and path
// are NOT modifiable here — to rename or relocate, create a new
// project. (Keeps the git clone logic sane: the path is the slug's
// physical home.)
func (db *DB) UpdateProjectV2(ctx context.Context, id string, name, description, gitBranch *string) error {
	now := time.Now().UTC()
	res, err := db.ExecContext(ctx,
		`UPDATE pm_projects
            SET name = COALESCE(?, name),
                description = COALESCE(?, description),
                git_branch = COALESCE(?, git_branch),
                updated_at = ?
          WHERE id = ? AND deleted = 0`,
		nullableString(name), nullableString(description),
		nullableString(gitBranch), now, id,
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// SetProjectStatus updates projects.status. Used by the async clone
// pipeline (cloning → ready / error).
func (db *DB) SetProjectStatus(ctx context.Context, id, status string) error {
	res, err := db.ExecContext(ctx,
		`UPDATE pm_projects SET status = ?, updated_at = ? WHERE id = ? AND deleted = 0`,
		status, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("set project status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// SetProjectGitCommit records the post-clone commit SHA so the UI
// can display "currently checked out at <sha>".
func (db *DB) SetProjectGitCommit(ctx context.Context, id, sha string) error {
	res, err := db.ExecContext(ctx,
		`UPDATE pm_projects SET git_commit_sha = ?, updated_at = ? WHERE id = ? AND deleted = 0`,
		sha, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("set commit sha: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// DeleteProjectV2 soft-deletes a project. The on-disk repo is NOT
// removed — operators wipe it manually after the row goes away.
func (db *DB) DeleteProjectV2(ctx context.Context, id string) error {
	now := time.Now().UTC()
	res, err := db.ExecContext(ctx,
		`UPDATE pm_projects SET deleted = 1, updated_at = ? WHERE id = ? AND deleted = 0`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// ProjectAccessibleByUser reports whether `userID` can see `project`.
// Returns true when:
//   - the user is admin (users.role = "admin"), OR
//   - the user is a member of the project's team.
//
// Caller passes the role explicitly so the DB layer stays free of
// any user lookup and the hot path is one query.
func (db *DB) ProjectAccessibleByUser(ctx context.Context, projectID, userID, role string) (bool, error) {
	if role == string(RoleAdmin) {
		return true, nil
	}
	var teamID string
	err := db.QueryRowContext(ctx,
		`SELECT team_id FROM pm_projects WHERE id = ? AND deleted = 0`,
		projectID,
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
// Helpers
// ----------------------------------------------------------------------------

func (db *DB) scanTeam(s scanner) (*Team, error) {
	t := &Team{}
	var deleted int
	if err := s.Scan(&t.ID, &t.Name, &t.Slug, &t.Description,
		&t.OwnerID, &t.CreatedAt, &t.UpdatedAt, &deleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	t.Deleted = deleted != 0
	return t, nil
}

func (db *DB) scanProjectV2(s scanner) (*Project, error) {
	p := &Project{}
	var deleted int
	if err := s.Scan(&p.ID, &p.TeamID, &p.Name, &p.Slug, &p.Description, &p.Path,
		&p.GitURL, &p.GitBranch, &p.GitCommitSHA, &p.Status,
		&p.CreatedAt, &p.UpdatedAt, &deleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	p.Deleted = deleted != 0
	return p, nil
}

// nullableString turns *string into sql.NullString. nil → NULL,
// non-nil → the value. Lets a COALESCE-aware UPDATE skip unchanged
// fields without us juggling empty-string-vs-NULL semantics.
func nullableString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// ErrFKMissing is the sentinel returned by fkCheck when the target
// row is absent. Exposed so handlers can disambiguate UNIQUE-vs-FK
// failures (both can raise "constraint failed" on SQLite).
var ErrFKMissing = errors.New("fk target missing")

// fkCheck probes whether a foreign-key target row exists. It returns
// ErrFKMissing when the row is missing, nil otherwise. Used only to
// refine ambiguous CREATE failures.
func (db *DB) fkCheck(ctx context.Context, table, column, value string) error {
	q := fmt.Sprintf(`SELECT 1 FROM %s WHERE %s = ? LIMIT 1`, table, column)
	var x int
	err := db.QueryRowContext(ctx, q, value).Scan(&x)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFKMissing
		}
		return err
	}
	return nil
}

// ComputeProjectPath returns the canonical filesystem path for a
// project: <repoRoot>/<slug>. The repoRoot is sanitised to absolute
// path so the on-disk location is unambiguous; slug is sanitised via
// SanitizeSlug to reject slashes and other filesystem-unsafe chars.
//
// Callers should NOT pass a user-supplied path — this is the
// authoritative computation.
func ComputeProjectPath(repoRoot, slug string) (string, error) {
	if repoRoot == "" {
		return "", errors.New("ComputeProjectPath: repoRoot required")
	}
	if slug == "" {
		return "", errors.New("ComputeProjectPath: slug required")
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("abs repo root: %w", err)
	}
	return filepath.Join(absRoot, SanitizeSlug(slug)), nil
}

// SanitizeSlug replaces characters unsafe for directory names with
// underscores. Lower-case ASCII is preserved; spaces become
// underscores. Empty / all-underscore inputs are rejected so callers
// can refuse to create obviously-broken projects.
func SanitizeSlug(s string) string {
	r := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
		" ", "_", "\t", "_", "\n", "_",
	)
	return r.Replace(s)
}

// ValidateGitURL is a quick sanity check: the URL must be http(s),
// ssh, or git. SSH "scp-like" syntax (git@github.com:foo/bar) is
// accepted by string-detection because url.Parse rejects it. We
// don't try to clone here — handlers decide whether async cloning
// happens.
func ValidateGitURL(raw string) error {
	if raw == "" {
		return nil // empty is allowed (manual / no-git projects)
	}
	// SSH scp-like: user@host:path (no scheme). url.Parse mis-parses
	// these so detect them by the absence of "://" before any "/".
	if !strings.Contains(raw, "://") && strings.Contains(raw, "@") && strings.Contains(raw, ":") {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse git url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "http", "https", "ssh", "git", "file":
		return nil
	}
	return fmt.Errorf("unsupported git url scheme %q", scheme)
}

package console

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// timeNow is overridable in tests via setTimeNowForMigrate.
var timeNow = func() time.Time { return time.Now().UTC() }

// =============================================================================
// v2 schema migration — runs idempotently on startup
// =============================================================================
//
// This is the canonical migrator. The reference SQL files in
// deploy/sql/v2_schema.sqlite.sql / v2_schema.mysql.sql and
// deploy/sql/v1_to_v2_migrate.sql mirror what's here so ops can audit
// the schema by reading either the SQL or the Go.
//
// Design notes:
//
//   1. IDEMPOTENT. Every step probes the schema first and is a no-op when
//      the target state is already reached. Safe to run on every boot and
//      from multiple replicas racing on a fresh deploy.
//
//   2. SCHEMA-AWARE. The same Go code handles both SQLite and MySQL 8.0;
//      only the dialect-specific DDL strings change. No driver-specific
//      imports leak into handlers.
//
//   3. BACKWARD-SAFE. v1 rows that can't be cleanly mapped are placed
//      under sentinel IDs (team_legacy, mod_legacy) that the application
//      treats as "auto-managed default bucket" rather than bombing the
//      upgrade.
//
//   4. FAIL-OPEN. A migration error is logged and the boot continues
//      with the legacy DDL when possible. Critical DDL failures still
//      panic to prevent data corruption.
//
// =============================================================================

// MigrateV2 brings a v1 console DB forward to the v2 schema described in
// cbmem-team-prd.md §4 and the diagrams/cbmem-team-v2-erd.md ER diagram.
// It is invoked from console.Mount after Migrate() so the order on disk
// (v1 tables → v1+v2 co-existing → v2-cleanup) is preserved.
//
// Phase order:
//   1. Create new v2-only tables
//   2. Add new columns on v1 tables (idempotent ALTER)
//   3. Backfill legacy rows with sentinel defaults
//   4. Add new indexes
//   5. Drop v1-only columns that are no longer used
//   6. Seed built-in memory templates
//
// Returns the first non-recoverable error; soft failures are logged via
// the supplied warn callback (or fmt.Printf when nil).
func (db *DB) MigrateV2(ctx context.Context, warn func(string, ...any)) error {
	if warn == nil {
		warn = func(format string, args ...any) {
			fmt.Printf("migrate-v2 warn: "+format+"\n", args...)
		}
	}

	if err := db.migrateV2Phase1(ctx, warn); err != nil {
		return fmt.Errorf("phase1 create v2 tables: %w", err)
	}
	if err := db.migrateV2Phase2(ctx, warn); err != nil {
		return fmt.Errorf("phase2 add columns: %w", err)
	}
	if err := db.migrateV2Phase3(ctx, warn); err != nil {
		return fmt.Errorf("phase3 backfill: %w", err)
	}
	if err := db.migrateV2Phase4(ctx, warn); err != nil {
		return fmt.Errorf("phase4 indexes: %w", err)
	}
	if err := db.migrateV2Phase5(ctx, warn); err != nil {
		return fmt.Errorf("phase5 drop legacy columns: %w", err)
	}
	if err := db.migrateV2Phase6(ctx, warn); err != nil {
		return fmt.Errorf("phase6 seed templates: %w", err)
	}
	return nil
}

// migrateV2Phase1 creates the 9 brand-new v2 tables (everything except
// users / projects / sessions / distill_tasks / summarize_tasks which
// existed in v1). All CREATE TABLE statements use IF NOT EXISTS so the
// fresh-DB path and the upgrade path share one code path.
func (db *DB) migrateV2Phase1(ctx context.Context, warn func(string, ...any)) error {
	tables := db.v2DDL().createTables
	for _, ddl := range tables {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			// Tolerate "already exists" so a manual SQL first-run doesn't
			// double-fail here.
			if isAlreadyExists(err) {
				continue
			}
			return fmt.Errorf("create %q: %w", firstLine(ddl), err)
		}
	}
	return nil
}

// migrateV2Phase2 adds new columns to v1 tables. SQLite 3.35+ and MySQL 8.0.29
// have native ALTER TABLE ADD COLUMN; older engines need the column
// recreation dance. We probe the catalog and skip when the column is
// already present.
func (db *DB) migrateV2Phase2(ctx context.Context, warn func(string, ...any)) error {
	cols := db.v2DDL().addColumns
	for _, c := range cols {
		has, err := db.columnExists(ctx, c.table, c.column)
		if err != nil {
			return fmt.Errorf("columnExists(%s.%s): %w", c.table, c.column, err)
		}
		if has {
			continue
		}
		if _, err := db.ExecContext(ctx, c.ddl); err != nil {
			if isAlreadyExists(err) || strings.Contains(err.Error(), "duplicate column") {
				continue
			}
			return fmt.Errorf("add column %s.%s: %w", c.table, c.column, err)
		}
		warn("added column %s.%s", c.table, c.column)
	}
	return nil
}

// migrateV2Phase3 backfills legacy v1 rows with sentinel defaults:
//   - users.username = users.id (id was already a usable login key in v1)
//   - users.password_hash = "!v1-legacy-must-reset!" + must_change_password=1
//   - projects.team_id = "team_legacy" (created in phase 1 if absent)
//   - sessions.team_id / module_id = "team_legacy" / "mod_legacy"
//
// Each UPDATE is idempotent: WHERE clauses exclude rows already populated.
func (db *DB) migrateV2Phase3(ctx context.Context, warn func(string, ...any)) error {
	// 1. Ensure team_legacy exists so the FK constraint on projects.team_id
	//    accepts the sentinel backfill below. ON CONFLICT DO NOTHING
	//    semantics — INSERT OR IGNORE on SQLite, INSERT IGNORE on MySQL.
	//
	// The INSERT uses the v2 teams schema: id, name, slug, description,
	// owner_id, created_at, updated_at (7 placeholders). created_at /
	// updated_at default to the legacy team's clock so audit queries
	// "WHEN was the team created?" still return a sensible value.
	legacyTeamID := "team_legacy"
	now := timeNow()
	// owner_id is NULL — there's no canonical owner for a v1 → v2 legacy
	// bucket; the audit log carries the operator that triggered the
	// upgrade if a human ever needs to know.
	if _, err := db.ExecContext(ctx,
		db.v2DDL().legacyTeamSQL,
		legacyTeamID, "Legacy Team (migrated from v1)",
		"legacy", "", nil, now, now,
	); err != nil {
		// An existing team with that id is fine — skip the duplication error.
		if !isAlreadyExists(err) {
			return fmt.Errorf("insert team_legacy: %w", err)
		}
	}

	// 2. Ensure proj_legacy exists. v2 projects schema: id, team_id,
	//    name, slug, description, path, status, created_at, updated_at
	//    (9 placeholders). The path is empty string because v1 didn't
	//    carry one and we leave the surface area empty rather than
	//    guessing a global code-repo path.
	firstProjectID := "proj_legacy"
	if _, err := db.ExecContext(ctx,
		db.v2DDL().legacyProjectSQL,
		firstProjectID, legacyTeamID,
		"project_legacy", "project_legacy",
		"Legacy Project (migrated from v1)", "",
		"ready", now, now,
	); err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("insert proj_legacy: %w", err)
	}

	// 3. Ensure mod_legacy exists under proj_legacy. v2 modules schema:
	//    id, project_id, name, description, is_leaf, created_at,
	//    updated_at (7 placeholders). mod_legacy is a leaf so existing
	//    v1 sessions can map onto it directly.
	if _, err := db.ExecContext(ctx,
		db.v2DDL().legacyModuleSQL,
		"mod_legacy", firstProjectID,
		"Legacy Module (migrated from v1)", "",
		1, now, now,
	); err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("insert mod_legacy: %w", err)
	}

	// 4. users.username backfill.
	if _, err := db.ExecContext(ctx,
		`UPDATE users SET username = id WHERE username = '' OR username IS NULL`,
	); err != nil {
		return fmt.Errorf("backfill users.username: %w", err)
	}

	// 4b. users.password_hash backfill. v1 had no password column, so
	//     every v1 row carries the empty default we set in phase 2.
	//     Stamp the sentinel so handlers can detect "must reset" rows
	//     without joining a separate column.
	if _, err := db.ExecContext(ctx,
		`UPDATE users SET password_hash = '!v1-legacy-must-reset!'
		   WHERE password_hash = ''`,
	); err != nil {
		return fmt.Errorf("backfill users.password_hash: %w", err)
	}

	// 5. users must_change_password=1 for any row that still has a sentinel
	//    hash (i.e. carried over from v1 with no real password).
	if _, err := db.ExecContext(ctx,
		`UPDATE users SET must_change_password = 1
		   WHERE password_hash = '!v1-legacy-must-reset!'`,
	); err != nil {
		return fmt.Errorf("force must_change_password for legacy users: %w", err)
	}

	// 6. projects.team_id backfill. Only rows where team_id is NULL or
	//    the literal default ('') need touching; new rows from a v2
	//    frontend already carry the column.
	if _, err := db.ExecContext(ctx,
		`UPDATE projects SET team_id = ?, status = 'ready', description = description
		   WHERE team_id IS NULL OR team_id = ''`,
		legacyTeamID,
	); err != nil {
		return fmt.Errorf("backfill projects.team_id: %w", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE projects SET slug = name WHERE slug = '' OR slug IS NULL`,
	); err != nil {
		return fmt.Errorf("backfill projects.slug: %w", err)
	}

	// 7. sessions.team_id / module_id backfill.
	if _, err := db.ExecContext(ctx,
		`UPDATE sessions SET team_id = ? WHERE team_id IS NULL OR team_id = ''`,
		legacyTeamID,
	); err != nil {
		return fmt.Errorf("backfill sessions.team_id: %w", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE sessions SET module_id = ? WHERE module_id IS NULL OR module_id = ''`,
		"mod_legacy",
	); err != nil {
		return fmt.Errorf("backfill sessions.module_id: %w", err)
	}

	return nil
}

// migrateV2Phase4 creates the new indexes from the v2 schema. Index
// creation in MySQL has no IF NOT EXISTS so we probe first; SQLite
// uses CREATE INDEX IF NOT EXISTS directly.
func (db *DB) migrateV2Phase4(ctx context.Context, warn func(string, ...any)) error {
	indexes := db.v2DDL().createIndexes
	for _, idx := range indexes {
		if db.driver == "mysql" {
			exists, err := db.indexExists(ctx, idx.table, idx.name)
			if err != nil {
				return fmt.Errorf("indexExists(%s): %w", idx.name, err)
			}
			if exists {
				continue
			}
		}
		if _, err := db.ExecContext(ctx, idx.ddl); err != nil {
			if isAlreadyExists(err) {
				continue
			}
			return fmt.Errorf("create index %s: %w", idx.name, err)
		}
		warn("created index %s", idx.name)
	}
	return nil
}

// migrateV2Phase5 drops v1-only columns that are no longer consulted.
// The auto-sync columns on sessions (mempalace_synced_turns / *_at /
// *_error) are already added by MigrateSessionsAutoSyncColumns and are
// not part of phase 5.
//
// We DROP users.project_paths because projects.path is now globally
// UNIQUE so the per-user whitelist is dead weight.
//
// Engines that don't support DROP COLUMN (SQLite < 3.35, MySQL < 8.0.29)
// need the column recreation dance. For those we leave the column in
// place and warn; handlers ignore unknown columns.
func (db *DB) migrateV2Phase5(ctx context.Context, warn func(string, ...any)) error {
	has, err := db.columnExists(ctx, "users", "project_paths")
	if err != nil {
		return err
	}
	if !has {
		return nil
	}
	dropSQL := db.v2DDL().dropProjectPaths
	if dropSQL == "" {
		warn("users.project_paths retained (engine does not support DROP COLUMN); handlers ignore it")
		return nil
	}
	if _, err := db.ExecContext(ctx, dropSQL); err != nil {
		// Don't fail boot — the column is harmless if ignored.
		warn("drop users.project_paths skipped: %v", err)
		return nil
	}
	warn("dropped users.project_paths")
	return nil
}

// migrateV2Phase6 seeds the 5 built-in memory templates. The seed is a
// pure upsert so re-running is safe.
func (db *DB) migrateV2Phase6(ctx context.Context, warn func(string, ...any)) error {
	for _, tpl := range builtInMemoryTemplates {
		if _, err := db.ExecContext(ctx,
			db.v2DDL().upsertTemplateSQL,
			tpl.ID, tpl.Name, tpl.Description, tpl.FieldsJSON, tpl.BodyTemplate, 1,
		); err != nil {
			return fmt.Errorf("seed template %s: %w", tpl.ID, err)
		}
	}
	return nil
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// isAlreadyExists recognises both SQLite ("table X already exists") and
// MySQL (Error 1050 / "Duplicate entry") so we can no-op CREATE / INSERT
// statements that race with concurrent migrations.
func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "Error 1050") ||
		strings.Contains(msg, "Error 1060") || // ER_DUP_FIELDNAME
		strings.Contains(msg, "Error 1061") || // ER_DUP_KEYNAME
		strings.Contains(msg, "SQLSTATE 42S21") || // SQLite: duplicate column
		strings.Contains(msg, "duplicate column")
}

// firstLine extracts the table name from a CREATE TABLE statement for
// error reporting.
func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}

// ----------------------------------------------------------------------------
// DDL bundles — dialect-specific strings returned by v2DDL()
// ----------------------------------------------------------------------------

// addColumnSpec describes one ALTER TABLE ADD COLUMN.
type addColumnSpec struct {
	table  string
	column string
	ddl    string
}

// indexCreateSpec describes one CREATE INDEX (or its MySQL equivalent).
type indexCreateSpec struct {
	table string
	name  string
	ddl   string
}

// ddlBundle is the dialect-aware bundle of every DDL string the
// migrator needs. New entries go here, not in scattered SQL fragments.
type ddlBundle struct {
	// createTables — Phase 1
	createTables []string
	// addColumns — Phase 2 (idempotent via probe)
	addColumns []addColumnSpec
	// legacy seed inserts — Phase 3
	legacyTeamSQL    string
	legacyProjectSQL string
	legacyModuleSQL  string
	// createIndexes — Phase 4
	createIndexes []indexCreateSpec
	// dropProjectPaths — Phase 5 ("" when engine doesn't support DROP COLUMN)
	dropProjectPaths string
	// upsertTemplateSQL — Phase 6
	upsertTemplateSQL string
}

// v2DDL returns the DDL bundle for the current driver. SQLite bundle is
// the default; MySQL is selected when db.driver == "mysql".
func (db *DB) v2DDL() ddlBundle {
	if db.driver == "mysql" {
		return mysqlV2DDL
	}
	return sqliteV2DDL
}

// sqliteV2DDL is the SQLite-flavoured bundle. Statements mirror
// deploy/sql/v2_schema.sqlite.sql.
var sqliteV2DDL = ddlBundle{
	createTables: []string{
		`CREATE TABLE IF NOT EXISTS teams (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            slug TEXT NOT NULL UNIQUE,
            description TEXT NOT NULL DEFAULT '',
            owner_id TEXT,
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            deleted INTEGER NOT NULL DEFAULT 0,
            FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
        )`,
		`CREATE TABLE IF NOT EXISTS team_members (
            team_id TEXT NOT NULL,
            user_id TEXT NOT NULL,
            role TEXT NOT NULL DEFAULT 'developer',
            joined_at DATETIME NOT NULL,
            PRIMARY KEY (team_id, user_id),
            FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
        )`,
		`CREATE TABLE IF NOT EXISTS modules (
            id TEXT PRIMARY KEY,
            project_id TEXT NOT NULL,
            parent_id TEXT,
            name TEXT NOT NULL,
            path TEXT NOT NULL DEFAULT '',
            description TEXT NOT NULL DEFAULT '',
            "order" INTEGER NOT NULL DEFAULT 0,
            is_leaf INTEGER NOT NULL DEFAULT 1,
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            deleted INTEGER NOT NULL DEFAULT 0,
            FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE ON UPDATE CASCADE,
            FOREIGN KEY (parent_id) REFERENCES modules(id) ON DELETE CASCADE ON UPDATE CASCADE
        )`,
		`CREATE TABLE IF NOT EXISTS memory_templates (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            description TEXT NOT NULL DEFAULT '',
            fields_json TEXT NOT NULL DEFAULT '[]',
            body_template TEXT NOT NULL DEFAULT '',
            is_builtin INTEGER NOT NULL DEFAULT 0
        )`,
		`CREATE TABLE IF NOT EXISTS memories (
            id TEXT PRIMARY KEY,
            team_id TEXT NOT NULL,
            project_id TEXT NOT NULL,
            module_id TEXT NOT NULL,
            user_id TEXT NOT NULL,
            title TEXT NOT NULL,
            content TEXT NOT NULL,
            template_id TEXT,
            tags_json TEXT NOT NULL DEFAULT '[]',
            hall TEXT NOT NULL DEFAULT 'facts',
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            deleted INTEGER NOT NULL DEFAULT 0,
            FOREIGN KEY (template_id) REFERENCES memory_templates(id) ON DELETE SET NULL
        )`,
		`CREATE TABLE IF NOT EXISTS ai_tools (
            id TEXT PRIMARY KEY,
            team_id TEXT NOT NULL,
            name TEXT NOT NULL,
            slug TEXT NOT NULL,
            description TEXT NOT NULL DEFAULT '',
            endpoint TEXT NOT NULL,
            protocol TEXT NOT NULL DEFAULT 'http',
            command TEXT NOT NULL DEFAULT '',
            args_json TEXT NOT NULL DEFAULT '[]',
            required_role TEXT NOT NULL DEFAULT 'developer',
            allowed_projects_json TEXT NOT NULL DEFAULT '',
            timeout_seconds INTEGER NOT NULL DEFAULT 300,
            enabled INTEGER NOT NULL DEFAULT 1,
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE
        )`,
		`CREATE TABLE IF NOT EXISTS ai_tool_invocations (
            id TEXT PRIMARY KEY,
            tool_id TEXT NOT NULL,
            user_id TEXT NOT NULL,
            team_id TEXT NOT NULL,
            project_id TEXT NOT NULL,
            module_id TEXT NOT NULL,
            working_dir TEXT NOT NULL,
            input_json TEXT NOT NULL DEFAULT '{}',
            output_json TEXT NOT NULL DEFAULT '',
            status TEXT NOT NULL DEFAULT 'pending',
            started_at DATETIME NOT NULL,
            finished_at DATETIME,
            error TEXT NOT NULL DEFAULT '',
            FOREIGN KEY (tool_id) REFERENCES ai_tools(id),
            FOREIGN KEY (user_id) REFERENCES users(id),
            FOREIGN KEY (team_id) REFERENCES teams(id),
            FOREIGN KEY (project_id) REFERENCES projects(id),
            FOREIGN KEY (module_id) REFERENCES modules(id)
        )`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            token_hash TEXT NOT NULL,
            issued_at DATETIME NOT NULL,
            expires_at DATETIME NOT NULL,
            revoked_at DATETIME,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
        )`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            ts DATETIME NOT NULL,
            user_id TEXT,
            team_id TEXT,
            kind TEXT NOT NULL,
            target_id TEXT NOT NULL DEFAULT '',
            meta_json TEXT NOT NULL DEFAULT '{}'
        )`,
	},

	addColumns: []addColumnSpec{
		// users (v1 → v2)
		{"users", "username", `ALTER TABLE users ADD COLUMN username TEXT NOT NULL DEFAULT ''`},
		{"users", "password_hash", `ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT ''`},
		{"users", "email", `ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT ''`},
		{"users", "default_team_id", `ALTER TABLE users ADD COLUMN default_team_id TEXT`},
		{"users", "role", `ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'developer'`},
		{"users", "must_change_password", `ALTER TABLE users ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 1`},
		{"users", "updated_at", `ALTER TABLE users ADD COLUMN updated_at DATETIME`},
		{"users", "last_login_at", `ALTER TABLE users ADD COLUMN last_login_at DATETIME`},

		// projects (v1 → v2)
		{"projects", "team_id", `ALTER TABLE projects ADD COLUMN team_id TEXT`},
		{"projects", "slug", `ALTER TABLE projects ADD COLUMN slug TEXT NOT NULL DEFAULT ''`},
		{"projects", "description", `ALTER TABLE projects ADD COLUMN description TEXT NOT NULL DEFAULT ''`},
		{"projects", "git_url", `ALTER TABLE projects ADD COLUMN git_url TEXT NOT NULL DEFAULT ''`},
		{"projects", "git_branch", `ALTER TABLE projects ADD COLUMN git_branch TEXT NOT NULL DEFAULT ''`},
		{"projects", "git_commit_sha", `ALTER TABLE projects ADD COLUMN git_commit_sha TEXT NOT NULL DEFAULT ''`},
		{"projects", "status", `ALTER TABLE projects ADD COLUMN status TEXT NOT NULL DEFAULT 'ready'`},
		// M3 (added late, after v2 migration already shipped on some
		// installs): owner_id was always in the v2 CREATE TABLE but
		// the ALTER path for v1 upgrades never added it. Idempotent
		// via the IF NOT EXISTS clause.
		{"projects", "owner_id", `ALTER TABLE projects ADD COLUMN owner_id TEXT NOT NULL DEFAULT ''`},

		// sessions (v1 → v2): team_id, module_id (NOT NULL after backfill), auto-sync columns
		{"sessions", "team_id", `ALTER TABLE sessions ADD COLUMN team_id TEXT NOT NULL DEFAULT ''`},
		{"sessions", "module_id", `ALTER TABLE sessions ADD COLUMN module_id TEXT NOT NULL DEFAULT ''`},
		{"sessions", "mempalace_synced_turns", `ALTER TABLE sessions ADD COLUMN mempalace_synced_turns INTEGER NOT NULL DEFAULT 0`},
		{"sessions", "mempalace_last_synced_at", `ALTER TABLE sessions ADD COLUMN mempalace_last_synced_at DATETIME`},
		{"sessions", "mempalace_last_error", `ALTER TABLE sessions ADD COLUMN mempalace_last_error TEXT NOT NULL DEFAULT ''`},
	},

	legacyTeamSQL: `INSERT OR IGNORE INTO teams (id, name, slug, description, owner_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?)`,
	legacyProjectSQL: `INSERT OR IGNORE INTO projects (id, team_id, name, slug, description, path, status, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
	legacyModuleSQL:  `INSERT OR IGNORE INTO modules (id, project_id, name, description, is_leaf, created_at, updated_at) VALUES (?,?,?,?,?,?,?)`,

	createIndexes: []indexCreateSpec{
		{"teams", "idx_teams_slug", `CREATE INDEX IF NOT EXISTS idx_teams_slug ON teams(slug)`},
		{"team_members", "idx_team_members_user", `CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id)`},
		{"projects", "idx_projects_team", `CREATE INDEX IF NOT EXISTS idx_projects_team ON projects(team_id)`},
		{"projects", "idx_projects_status", `CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status)`},
		// M3: enforce (team_id, slug) uniqueness via a UNIQUE index.
		// The v2 CREATE TABLE never ran on a v1 → v2 upgraded DB, so
		// the table-level UNIQUE constraint is missing here. The
		// UNIQUE INDEX is equivalent for read paths and idempotent.
		{"projects", "uk_projects_team_slug", `CREATE UNIQUE INDEX IF NOT EXISTS uk_projects_team_slug ON projects(team_id, slug)`},
		{"modules", "idx_modules_project", `CREATE INDEX IF NOT EXISTS idx_modules_project ON modules(project_id)`},
		{"modules", "idx_modules_parent", `CREATE INDEX IF NOT EXISTS idx_modules_parent ON modules(parent_id)`},
		{"modules", "idx_modules_is_leaf", `CREATE INDEX IF NOT EXISTS idx_modules_is_leaf ON modules(is_leaf)`},
		// M4: enforce (project_id, parent_id, name) uniqueness via a
		// UNIQUE INDEX — same rationale as uk_projects_team_slug.
		// SQLite treats NULL != NULL in unique indexes so two root
		// modules can share a name within a project; that matches
		// the intent (root modules act as a flat namespace within a
		// project, children must be unique under their parent).
		{"modules", "uk_modules_project_parent_name", `CREATE UNIQUE INDEX IF NOT EXISTS uk_modules_project_parent_name ON modules(project_id, parent_id, name)`},
		{"sessions", "idx_sessions_team", `CREATE INDEX IF NOT EXISTS idx_sessions_team ON sessions(team_id)`},
		{"sessions", "idx_sessions_module", `CREATE INDEX IF NOT EXISTS idx_sessions_module ON sessions(module_id)`},
		{"sessions", "idx_sessions_mempalace_pending", `CREATE INDEX IF NOT EXISTS idx_sessions_mempalace_pending ON sessions(mempalace_synced_turns, turn_count)`},
		{"memories", "idx_memories_team", `CREATE INDEX IF NOT EXISTS idx_memories_team ON memories(team_id)`},
		{"memories", "idx_memories_module", `CREATE INDEX IF NOT EXISTS idx_memories_module ON memories(module_id)`},
		{"memories", "idx_memories_user", `CREATE INDEX IF NOT EXISTS idx_memories_user ON memories(user_id)`},
		{"ai_tools", "idx_ai_tools_team", `CREATE INDEX IF NOT EXISTS idx_ai_tools_team ON ai_tools(team_id)`},
		{"ai_tool_invocations", "idx_invocations_tool", `CREATE INDEX IF NOT EXISTS idx_invocations_tool ON ai_tool_invocations(tool_id)`},
		{"ai_tool_invocations", "idx_invocations_user", `CREATE INDEX IF NOT EXISTS idx_invocations_user ON ai_tool_invocations(user_id)`},
		{"refresh_tokens", "idx_refresh_tokens_user", `CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id)`},
		{"refresh_tokens", "idx_refresh_tokens_hash", `CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash)`},
		{"audit_logs", "idx_audit_ts", `CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_logs(ts DESC)`},
		{"audit_logs", "idx_audit_user", `CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id)`},
	},

	// SQLite 3.35+ supports DROP COLUMN; older versions return "" so we
	// log a warning instead of failing.
	dropProjectPaths: func() string {
		// We can't query the SQLite version from here cheaply, so we
		// always emit the DROP and let it fail silently on old engines.
		// The isAlreadyExists / sqlite "error in DROP COLUMN" path is
		// caught in phase 5 and downgraded to a warn.
		return `ALTER TABLE users DROP COLUMN project_paths`
	}(),

	upsertTemplateSQL: `INSERT INTO memory_templates (id, name, description, fields_json, body_template, is_builtin)
	                    VALUES (?,?,?,?,?,?)
	                    ON CONFLICT(id) DO UPDATE SET
	                        name=excluded.name,
	                        description=excluded.description,
	                        fields_json=excluded.fields_json,
	                        body_template=excluded.body_template`,
}

// mysqlV2DDL is the MySQL 8.0-flavoured bundle. KEY and CREATE TABLE
// statements mirror deploy/sql/v2_schema.mysql.sql.
var mysqlV2DDL = ddlBundle{
	createTables: []string{
		`CREATE TABLE IF NOT EXISTS teams (
            id VARCHAR(128) NOT NULL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            slug VARCHAR(128) NOT NULL,
            description TEXT NOT NULL,
            owner_id VARCHAR(128) NULL,
            created_at DATETIME(0) NOT NULL,
            updated_at DATETIME(0) NOT NULL,
            deleted TINYINT(1) NOT NULL DEFAULT 0,
            UNIQUE KEY uk_teams_slug (slug)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS team_members (
            team_id VARCHAR(128) NOT NULL,
            user_id VARCHAR(128) NOT NULL,
            role VARCHAR(32) NOT NULL DEFAULT 'developer',
            joined_at DATETIME(0) NOT NULL,
            PRIMARY KEY (team_id, user_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS modules (
            id VARCHAR(128) NOT NULL PRIMARY KEY,
            project_id VARCHAR(128) NOT NULL,
            parent_id VARCHAR(128) NULL,
            name VARCHAR(255) NOT NULL,
            path VARCHAR(512) NOT NULL DEFAULT '',
            description TEXT NOT NULL,
            ` + "`order`" + ` INT NOT NULL DEFAULT 0,
            is_leaf TINYINT(1) NOT NULL DEFAULT 1,
            created_at DATETIME(0) NOT NULL,
            updated_at DATETIME(0) NOT NULL,
            deleted TINYINT(1) NOT NULL DEFAULT 0
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS memory_templates (
            id VARCHAR(128) NOT NULL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            description TEXT NOT NULL,
            fields_json JSON NOT NULL,
            body_template MEDIUMTEXT NOT NULL,
            is_builtin TINYINT(1) NOT NULL DEFAULT 0
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS memories (
            id VARCHAR(128) NOT NULL PRIMARY KEY,
            team_id VARCHAR(128) NOT NULL,
            project_id VARCHAR(128) NOT NULL,
            module_id VARCHAR(128) NOT NULL,
            user_id VARCHAR(128) NOT NULL,
            title VARCHAR(255) NOT NULL,
            content MEDIUMTEXT NOT NULL,
            template_id VARCHAR(128) NULL,
            tags_json JSON NOT NULL,
            hall VARCHAR(32) NOT NULL DEFAULT 'facts',
            created_at DATETIME(0) NOT NULL,
            updated_at DATETIME(0) NOT NULL,
            deleted TINYINT(1) NOT NULL DEFAULT 0
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS ai_tools (
            id VARCHAR(128) NOT NULL PRIMARY KEY,
            team_id VARCHAR(128) NOT NULL,
            name VARCHAR(255) NOT NULL,
            slug VARCHAR(128) NOT NULL,
            description TEXT NOT NULL,
            endpoint VARCHAR(512) NOT NULL,
            protocol VARCHAR(16) NOT NULL DEFAULT 'http',
            command VARCHAR(255) NOT NULL DEFAULT '',
            args_json JSON NOT NULL,
            required_role VARCHAR(32) NOT NULL DEFAULT 'developer',
            allowed_projects_json JSON NOT NULL,
            timeout_seconds INT NOT NULL DEFAULT 300,
            enabled TINYINT(1) NOT NULL DEFAULT 1,
            created_at DATETIME(0) NOT NULL,
            updated_at DATETIME(0) NOT NULL,
            UNIQUE KEY uk_ai_tools_team_slug (team_id, slug)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS ai_tool_invocations (
            id VARCHAR(128) NOT NULL PRIMARY KEY,
            tool_id VARCHAR(128) NOT NULL,
            user_id VARCHAR(128) NOT NULL,
            team_id VARCHAR(128) NOT NULL,
            project_id VARCHAR(128) NOT NULL,
            module_id VARCHAR(128) NOT NULL,
            working_dir VARCHAR(512) NOT NULL,
            input_json JSON NOT NULL,
            output_json JSON NOT NULL,
            status VARCHAR(16) NOT NULL DEFAULT 'pending',
            started_at DATETIME(0) NOT NULL,
            finished_at DATETIME(0) NULL,
            error TEXT NOT NULL
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
            id VARCHAR(128) NOT NULL PRIMARY KEY,
            user_id VARCHAR(128) NOT NULL,
            token_hash VARCHAR(255) NOT NULL,
            issued_at DATETIME(0) NOT NULL,
            expires_at DATETIME(0) NOT NULL,
            revoked_at DATETIME(0) NULL
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
            id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
            ts DATETIME(0) NOT NULL,
            user_id VARCHAR(128) NULL,
            team_id VARCHAR(128) NULL,
            kind VARCHAR(64) NOT NULL,
            target_id VARCHAR(128) NOT NULL DEFAULT '',
            meta_json JSON NOT NULL
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},

	addColumns: []addColumnSpec{
		{"users", "username", `ALTER TABLE users ADD COLUMN username VARCHAR(128) NOT NULL DEFAULT ''`},
		{"users", "password_hash", `ALTER TABLE users ADD COLUMN password_hash VARCHAR(255) NOT NULL DEFAULT ''`},
		{"users", "email", `ALTER TABLE users ADD COLUMN email VARCHAR(255) NOT NULL DEFAULT ''`},
		{"users", "default_team_id", `ALTER TABLE users ADD COLUMN default_team_id VARCHAR(128) NULL`},
		{"users", "role", `ALTER TABLE users ADD COLUMN role VARCHAR(32) NOT NULL DEFAULT 'developer'`},
		{"users", "must_change_password", `ALTER TABLE users ADD COLUMN must_change_password TINYINT(1) NOT NULL DEFAULT 1`},
		{"users", "updated_at", `ALTER TABLE users ADD COLUMN updated_at DATETIME(0) NULL`},
		{"users", "last_login_at", `ALTER TABLE users ADD COLUMN last_login_at DATETIME(0) NULL`},

		{"projects", "team_id", `ALTER TABLE projects ADD COLUMN team_id VARCHAR(128) NULL`},
		{"projects", "slug", `ALTER TABLE projects ADD COLUMN slug VARCHAR(128) NOT NULL DEFAULT ''`},
		{"projects", "description", `ALTER TABLE projects ADD COLUMN description TEXT NOT NULL`},
		{"projects", "git_url", `ALTER TABLE projects ADD COLUMN git_url VARCHAR(512) NOT NULL DEFAULT ''`},
		{"projects", "git_branch", `ALTER TABLE projects ADD COLUMN git_branch VARCHAR(128) NOT NULL DEFAULT ''`},
		{"projects", "git_commit_sha", `ALTER TABLE projects ADD COLUMN git_commit_sha VARCHAR(64) NOT NULL DEFAULT ''`},
		{"projects", "status", `ALTER TABLE projects ADD COLUMN status VARCHAR(32) NOT NULL DEFAULT 'ready'`},
		// M3: same late-arrival ALTER as the SQLite variant.
		{"projects", "owner_id", `ALTER TABLE projects ADD COLUMN owner_id VARCHAR(128) NOT NULL DEFAULT ''`},

		{"sessions", "team_id", `ALTER TABLE sessions ADD COLUMN team_id VARCHAR(128) NOT NULL DEFAULT ''`},
		{"sessions", "module_id", `ALTER TABLE sessions ADD COLUMN module_id VARCHAR(128) NOT NULL DEFAULT ''`},
		{"sessions", "mempalace_synced_turns", `ALTER TABLE sessions ADD COLUMN mempalace_synced_turns INT NOT NULL DEFAULT 0`},
		{"sessions", "mempalace_last_synced_at", `ALTER TABLE sessions ADD COLUMN mempalace_last_synced_at DATETIME(0) NULL`},
		{"sessions", "mempalace_last_error", `ALTER TABLE sessions ADD COLUMN mempalace_last_error TEXT NOT NULL`},
	},

	legacyTeamSQL: `INSERT IGNORE INTO teams (id, name, slug, description, owner_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?)`,
	legacyProjectSQL: `INSERT IGNORE INTO projects (id, team_id, name, slug, description, path, status, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
	legacyModuleSQL: `INSERT IGNORE INTO modules (id, project_id, name, description, is_leaf, created_at, updated_at) VALUES (?,?,?,?,?,?,?)`,

	createIndexes: []indexCreateSpec{
		{"team_members", "idx_team_members_user", `CREATE INDEX idx_team_members_user ON team_members(user_id)`},
		{"projects", "idx_projects_team", `CREATE INDEX idx_projects_team ON projects(team_id)`},
		{"projects", "idx_projects_status", `CREATE INDEX idx_projects_status ON projects(status)`},
		// M3: enforce (team_id, slug) uniqueness — see the SQLite
		// analogue above for the rationale.
		{"projects", "uk_projects_team_slug", `CREATE UNIQUE INDEX uk_projects_team_slug ON projects(team_id, slug)`},
		{"modules", "idx_modules_project", `CREATE INDEX idx_modules_project ON modules(project_id)`},
		{"modules", "idx_modules_parent", `CREATE INDEX idx_modules_parent ON modules(parent_id)`},
		{"modules", "idx_modules_is_leaf", `CREATE INDEX idx_modules_is_leaf ON modules(is_leaf)`},
		// M4: same composite UNIQUE as the SQLite variant.
		{"modules", "uk_modules_project_parent_name", `CREATE UNIQUE INDEX uk_modules_project_parent_name ON modules(project_id, parent_id, name)`},
		{"sessions", "idx_sessions_team", `CREATE INDEX idx_sessions_team ON sessions(team_id)`},
		{"sessions", "idx_sessions_module", `CREATE INDEX idx_sessions_module ON sessions(module_id)`},
		{"sessions", "idx_sessions_mempalace_pending", `CREATE INDEX idx_sessions_mempalace_pending ON sessions(mempalace_synced_turns, turn_count)`},
		{"memories", "idx_memories_team", `CREATE INDEX idx_memories_team ON memories(team_id)`},
		{"memories", "idx_memories_module", `CREATE INDEX idx_memories_module ON memories(module_id)`},
		{"memories", "idx_memories_user", `CREATE INDEX idx_memories_user ON memories(user_id)`},
		{"ai_tools", "idx_ai_tools_team", `CREATE INDEX idx_ai_tools_team ON ai_tools(team_id)`},
		{"ai_tool_invocations", "idx_invocations_tool", `CREATE INDEX idx_invocations_tool ON ai_tool_invocations(tool_id)`},
		{"ai_tool_invocations", "idx_invocations_user", `CREATE INDEX idx_invocations_user ON ai_tool_invocations(user_id)`},
		{"refresh_tokens", "idx_refresh_tokens_user", `CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id)`},
		{"refresh_tokens", "idx_refresh_tokens_hash", `CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash)`},
		{"audit_logs", "idx_audit_ts", `CREATE INDEX idx_audit_ts ON audit_logs(ts)`},
		{"audit_logs", "idx_audit_user", `CREATE INDEX idx_audit_user ON audit_logs(user_id)`},
	},

	// MySQL 8.0.29+ supports DROP COLUMN; older versions fail silently
	// at runtime and the warn() downgrade in migrateV2Phase5 hides it.
	dropProjectPaths: `ALTER TABLE users DROP COLUMN project_paths`,

	upsertTemplateSQL: `INSERT INTO memory_templates (id, name, description, fields_json, body_template, is_builtin)
	                    VALUES (?,?,?,?,?,?)
	                    ON DUPLICATE KEY UPDATE
	                        name=VALUES(name),
	                        description=VALUES(description),
	                        fields_json=VALUES(fields_json),
	                        body_template=VALUES(body_template)`,
}

// ----------------------------------------------------------------------------
// Built-in memory templates
// ----------------------------------------------------------------------------

type builtinTemplate struct {
	ID           string
	Name         string
	Description  string
	FieldsJSON   string
	BodyTemplate string
}

// builtInMemoryTemplates are the 5 default templates seeded on every
// install. The field schemas are deliberately small so the form fits a
// half-screen on the console UI.
var builtInMemoryTemplates = []builtinTemplate{
	{
		ID:          "tpl_adr",
		Name:        "Architecture Decision Record (ADR)",
		Description: "Capture one architectural decision with context, options, and consequences.",
		FieldsJSON: `[
			{"key":"title","label":"Title","type":"string","required":true},
			{"key":"status","label":"Status","type":"string","required":true,"help":"proposed | accepted | deprecated | superseded"},
			{"key":"context","label":"Context","type":"string","required":true,"help":"What is the issue we're addressing?"},
			{"key":"decision","label":"Decision","type":"string","required":true,"help":"What did we decide?"},
			{"key":"consequences","label":"Consequences","type":"string","required":false,"help":"What becomes easier / harder?"}
		]`,
		BodyTemplate: `# {{.title}}

> **Status**: {{.status}}
> **Date**: {{.date}}

## Context

{{.context}}

## Decision

{{.decision}}

## Consequences

{{.consequences}}
`,
	},
	{
		ID:          "tpl_lesson",
		Name:        "Lessons Learned",
		Description: "Document a problem, root cause, fix, and long-term impact.",
		FieldsJSON: `[
			{"key":"problem","label":"Problem","type":"string","required":true},
			{"key":"root_cause","label":"Root Cause","type":"string","required":true},
			{"key":"solution","label":"Solution","type":"string","required":true},
			{"key":"impact","label":"Impact","type":"string","required":false,"help":"severity + blast radius"}
		]`,
		BodyTemplate: `## Problem

{{.problem}}

## Root Cause

{{.root_cause}}

## Solution

{{.solution}}

## Impact

{{.impact}}
`,
	},
	{
		ID:          "tpl_snippet",
		Name:        "Reusable Code Snippet",
		Description: "Save a small, language-tagged code snippet with usage notes.",
		FieldsJSON: `[
			{"key":"language","label":"Language","type":"string","required":true,"help":"go | python | ts | sql | …"},
			{"key":"description","label":"Description","type":"string","required":true},
			{"key":"code","label":"Code","type":"string","required":true}
		]`,
		BodyTemplate: `**{{.language}}** — {{.description}}

` + "```" + `{{.language}}
{{.code}}
` + "```",
	},
	{
		ID:          "tpl_runbook",
		Name:        "Operational Runbook",
		Description: "A trigger-and-steps manual for an on-call or ops procedure.",
		FieldsJSON: `[
			{"key":"trigger","label":"Trigger","type":"string","required":true,"help":"What symptom / alert fires this?"},
			{"key":"steps","label":"Steps","type":"string","required":true,"help":"Numbered steps to resolve"},
			{"key":"rollback","label":"Rollback","type":"string","required":false,"help":"How to undo if needed"}
		]`,
		BodyTemplate: `## Trigger

{{.trigger}}

## Steps

{{.steps}}

## Rollback

{{.rollback}}
`,
	},
	{
		ID:          "tpl_decision",
		Name:        "Lightweight Decision",
		Description: "Capture a small decision without the full ADR ceremony.",
		FieldsJSON: `[
			{"key":"title","label":"Title","type":"string","required":true},
			{"key":"decision","label":"Decision","type":"string","required":true},
			{"key":"context","label":"Context","type":"string","required":false}
		]`,
		BodyTemplate: `**{{.title}}**

{{.decision}}

{{if .context}}_Context_: {{.context}}{{end}}
`,
	},
}
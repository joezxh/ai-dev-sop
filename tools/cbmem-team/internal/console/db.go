package console

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// DB wraps *sql.DB plus the driver name + DSN so handlers that need
// dialect-specific SQL (e.g. JSON_TABLE on MySQL) can branch on db.driver.
type DB struct {
	*sql.DB
	driver string // "sqlite" | "mysql"
	dsn    string // opaque, used for error messages only
}

// Driver returns "sqlite" or "mysql".
func (db *DB) Driver() string { return db.driver }

// Open opens a SQLite console DB. This is the legacy v1 path and is the
// default until `-mysql-dsn` is set. New code should call OpenEither so
// both backends share one migration entry point.
func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)",
		EscapePath(path))
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	sqlDB.SetMaxOpenConns(1)
	return &DB{DB: sqlDB, driver: "sqlite", dsn: path}, nil
}

func (db *DB) Conn() *sql.DB { return db.DB }
func (db *DB) Close() error  { return db.DB.Close() }

// Migrate creates the 7 legacy console tables. The DDL is dialect-aware:
// SQLite takes the legacy `CREATE INDEX IF NOT EXISTS` path; MySQL takes
// the `CreateMySQLSchema` path which uses information_schema lookups
// instead of IF NOT EXISTS (not supported by MySQL 8.0/9.0 for indexes).
//
// New v2 tables are added by their own Migrate* methods below so a
// SQLite-only build can opt out. M1 callers pass
// `WithToolDirectoryAndInvocationLogs()` to register M1's two new tables.
func (db *DB) Migrate(ctx context.Context, extra ...ExtraSchema) error {
	switch db.driver {
	case "mysql":
		return db.CreateMySQLSchema(ctx, extra...)
	default:
		if err := db.migrateConsole(ctx); err != nil {
			return err
		}
		for _, x := range extra {
			if x.ApplySQLite != nil {
				if err := x.ApplySQLite(ctx, db); err != nil {
					return fmt.Errorf("extra sqlite schema %s: %w", x.Name, err)
				}
			}
		}
		return nil
	}
}

func (db *DB) migrateConsole(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            display_name TEXT,
            project_paths TEXT,
            max_procs INTEGER,
            disabled INTEGER DEFAULT 0,
            created_at DATETIME,
            updated_at DATETIME
        )`,
		`CREATE TABLE IF NOT EXISTS projects (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            path TEXT NOT NULL UNIQUE,
            wing TEXT,
            mcp_bin TEXT,
            creator_id TEXT,
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            deleted INTEGER DEFAULT 0
        )`,
		`CREATE INDEX IF NOT EXISTS idx_projects_creator ON projects(creator_id)`,
`CREATE TABLE IF NOT EXISTS sessions (
           id TEXT PRIMARY KEY,
           user_id TEXT NOT NULL,
           project_id TEXT,
           project_path TEXT NOT NULL,
           started_at DATETIME NOT NULL,
           ended_at DATETIME,
           tool_count INTEGER DEFAULT 0,
           turn_count INTEGER DEFAULT 0,
           summary TEXT,
           mempalace_synced_turns INTEGER DEFAULT 0,
           mempalace_last_synced_at DATETIME,
           mempalace_last_error TEXT
       )`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_mempalace_pending ON sessions(mempalace_synced_turns, turn_count) WHERE mempalace_synced_turns < turn_count`,
		`CREATE TABLE IF NOT EXISTS session_turns (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            session_id TEXT NOT NULL,
            turn_no INTEGER NOT NULL,
            role TEXT NOT NULL,
            content TEXT NOT NULL,
            tools_json TEXT,
            ts DATETIME NOT NULL
        )`,
		`CREATE INDEX IF NOT EXISTS idx_session_turns_session ON session_turns(session_id)`,
		`CREATE TABLE IF NOT EXISTS summarize_tasks (
            id TEXT PRIMARY KEY,
            user_id TEXT,
            source_ids TEXT,
            depth TEXT,
            target_wing TEXT,
            status TEXT NOT NULL,
            result_json TEXT,
            created_at DATETIME NOT NULL,
            finished_at DATETIME
        )`,
		`CREATE TABLE IF NOT EXISTS distill_tasks (
            id TEXT PRIMARY KEY,
            user_id TEXT,
            source_ids TEXT,
            rules_json TEXT,
            status TEXT NOT NULL,
            result_json TEXT,
            mempalace_synced INTEGER DEFAULT 0,
            target_wing TEXT,
            created_at DATETIME NOT NULL,
            finished_at DATETIME
        )`,
		`CREATE TABLE IF NOT EXISTS console_sessions (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            created_at DATETIME NOT NULL,
            expires_at DATETIME NOT NULL,
            last_seen_at DATETIME,
            ip TEXT,
            ua TEXT
        )`,
		`CREATE INDEX IF NOT EXISTS idx_console_sessions_expires ON console_sessions(expires_at)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("migrate %q: %w", s[:40], err)
		}
	}
	return nil
}

// CreateMySQLSchema issues CREATE TABLE / CREATE INDEX statements rewritten
// for MySQL 8.0: ENGINE=InnoDB, DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci,
// AUTO_INCREMENT instead of AUTOINCREMENT, and the InnoDB replacement for
// INTEGER PRIMARY KEY (clustered index on PK is automatic).
//
// MySQL 8.0 does NOT support CREATE VIEW IF NOT EXISTS, so callers must
// invoke CreateMySQLViews separately after this returns. Likewise, MySQL
// rejects CREATE INDEX IF NOT EXISTS — the index DDL below is split into
// a check-and-create pair that produces identical results on a fresh DB
// and on a re-run.
//
// `extra` is the set of additional v2 tables to create in this run. It
// exists so M1–M4 can register their own DDL alongside the legacy 7
// tables without bloating this method.
func (db *DB) CreateMySQLSchema(ctx context.Context, extra ...ExtraSchema) error {
	if db.driver != "mysql" {
		return fmt.Errorf("CreateMySQLSchema called on non-mysql DB (driver=%s)", db.driver)
	}
	for _, ddl := range mysqlConsoleDDL {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("mysql ddl: %w\nstmt: %s", err, truncate(ddl, 200))
		}
	}
	for _, idx := range mysqlConsoleIndexes {
		exists, err := db.indexExists(ctx, idx.table, idx.name)
		if err != nil {
			return fmt.Errorf("indexExists %s: %w", idx.name, err)
		}
		if exists {
			continue
		}
		if _, err := db.ExecContext(ctx, idx.ddl); err != nil {
			return fmt.Errorf("create index %s: %w\nstmt: %s", idx.name, err, truncate(idx.ddl, 200))
		}
	}
	for _, x := range extra {
		if err := x.ApplyMySQL(ctx, db); err != nil {
			return fmt.Errorf("extra schema %s: %w", x.Name, err)
		}
	}
	return nil
}

// ExtraSchema lets the migration entry point add v2 tables without
// baking them into the legacy 7-table DDL.
type ExtraSchema struct {
	Name string
	// ApplyMySQL receives a MySQL DB and runs CREATE TABLE / INDEX.
	// The DDL is expected to be idempotent (CREATE TABLE IF NOT EXISTS
	// plus our indexExists probe pattern).
	ApplyMySQL func(ctx context.Context, db *DB) error
	// ApplySQLite is the SQLite counterpart; may be nil if the v2 table
	// is MySQL-only (the spec only requires v2 features on the production
	// MySQL path).
	ApplySQLite func(ctx context.Context, db *DB) error
}

// indexSpec describes one CREATE INDEX statement that MySQL 8.0 cannot
// express with IF NOT EXISTS.
type indexSpec struct {
	table string
	name  string
	ddl   string
}

func (db *DB) indexExists(ctx context.Context, table, name string) (bool, error) {
	const q = `SELECT COUNT(*) FROM information_schema.statistics
	            WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`
	var n int
	if err := db.QueryRowContext(ctx, q, table, name).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// mysqlConsoleIndexes is the ordered list of CREATE INDEX statements that
// accompany mysqlConsoleDDL. Adding a new table? Add its indexes here too.
var mysqlConsoleIndexes = []indexSpec{
	{"projects", "idx_projects_creator", `CREATE INDEX idx_projects_creator ON projects(creator_id)`},
	{"sessions", "idx_sessions_user", `CREATE INDEX idx_sessions_user ON sessions(user_id)`},
	{"sessions", "idx_sessions_project", `CREATE INDEX idx_sessions_project ON sessions(project_id)`},
	{"sessions", "idx_sessions_started", `CREATE INDEX idx_sessions_started ON sessions(started_at DESC)`},
	{"session_turns", "idx_session_turns_session", `CREATE INDEX idx_session_turns_session ON session_turns(session_id)`},
	{"console_sessions", "idx_console_sessions_expires", `CREATE INDEX idx_console_sessions_expires ON console_sessions(expires_at)`},
}

// mysqlConsoleDDL is the MySQL 8.0 rewrite of the SQLite DDL above.
//
// Differences vs SQLite:
//
//   - INTEGER PRIMARY KEY AUTOINCREMENT → BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY
//   - TEXT/INTEGER types stay compatible; MySQL is permissive about integer
//     width, so legacy INTEGER columns are accepted as-is.
//   - ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
//   - BOOLEAN stored as TINYINT(1) is fine in MySQL too; we keep INTEGER for
//     parity with the SQLite logical type.
//   - INDEX IF NOT EXISTS not supported in MySQL — we use plain CREATE INDEX
//     and rely on CREATE TABLE IF NOT EXISTS idempotency.
var mysqlConsoleDDL = []string{
	`CREATE TABLE IF NOT EXISTS users (
        id VARCHAR(128) NOT NULL PRIMARY KEY,
        display_name VARCHAR(255),
        project_paths TEXT,
        max_procs INT,
        disabled TINYINT(1) DEFAULT 0,
        created_at DATETIME,
        updated_at DATETIME
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS projects (
        id VARCHAR(128) NOT NULL PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        path VARCHAR(512) NOT NULL UNIQUE,
        wing VARCHAR(128),
        mcp_bin VARCHAR(512),
        creator_id VARCHAR(128),
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted TINYINT(1) DEFAULT 0
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
`CREATE TABLE IF NOT EXISTS sessions (
       id VARCHAR(128) NOT NULL PRIMARY KEY,
       user_id VARCHAR(128) NOT NULL,
       project_id VARCHAR(128),
       project_path VARCHAR(512) NOT NULL,
       started_at DATETIME NOT NULL,
       ended_at DATETIME,
       tool_count INT DEFAULT 0,
       turn_count INT DEFAULT 0,
       summary TEXT,
       mempalace_synced_turns INT DEFAULT 0,
       mempalace_last_synced_at DATETIME,
       mempalace_last_error TEXT
   ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS session_turns (
        id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
        session_id VARCHAR(128) NOT NULL,
        turn_no INT NOT NULL,
        role VARCHAR(32) NOT NULL,
        content TEXT NOT NULL,
        tools_json TEXT,
        ts DATETIME NOT NULL
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS summarize_tasks (
        id VARCHAR(128) NOT NULL PRIMARY KEY,
        user_id VARCHAR(128),
        source_ids TEXT,
        depth VARCHAR(32),
        target_wing VARCHAR(128),
        status VARCHAR(32) NOT NULL,
        result_json LONGTEXT,
        created_at DATETIME NOT NULL,
        finished_at DATETIME
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS distill_tasks (
        id VARCHAR(128) NOT NULL PRIMARY KEY,
        user_id VARCHAR(128),
        source_ids TEXT,
        rules_json LONGTEXT,
        status VARCHAR(32) NOT NULL,
        result_json LONGTEXT,
        mempalace_synced TINYINT(1) DEFAULT 0,
        target_wing VARCHAR(128),
        created_at DATETIME NOT NULL,
        finished_at DATETIME
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS console_sessions (
        id VARCHAR(128) NOT NULL PRIMARY KEY,
        user_id VARCHAR(128) NOT NULL,
        created_at DATETIME NOT NULL,
        expires_at DATETIME NOT NULL,
        last_seen_at DATETIME,
        ip VARCHAR(64),
        ua VARCHAR(512)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
}

// truncate returns the first n bytes of s plus "…" if s was longer.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// MigrationsContain reports whether table exists in the DB. Used by the
// ETL entry point to refuse a re-run that would silently double-write.
func (db *DB) MigrationsContain(ctx context.Context, table string) (bool, error) {
	// INFORMATION_SCHEMA.TABLES works on both SQLite 3.33+ (via schema)
	// and MySQL 8.0; we use the portable form.
	const q = `SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`
	var n int
	if err := db.QueryRowContext(ctx, q, table).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// MigrateSessionsAutoSyncColumns adds the mempalace_synced_turns /
// mempalace_last_synced_at / mempalace_last_error columns to a pre-existing
// `sessions` table. The CREATE TABLE in migrateConsole already includes them
// for fresh deployments; this ALTER is idempotent and runs at startup so
// v1 -> v1.1 upgrades get the auto-sync bookkeeping without a manual
// migrate step.
//
// Idempotent: each ALTER probes information_schema first and skips when
// the column is already present. Errors other than "already exists" are
// returned so a real schema problem isn't masked.
func (db *DB) MigrateSessionsAutoSyncColumns(ctx context.Context) error {
	cols := []struct {
		name string
		ddl  string
	}{
		{"mempalace_synced_turns", `ALTER TABLE sessions ADD COLUMN mempalace_synced_turns INTEGER DEFAULT 0`},
		{"mempalace_last_synced_at", `ALTER TABLE sessions ADD COLUMN mempalace_last_synced_at DATETIME`},
		{"mempalace_last_error", `ALTER TABLE sessions ADD COLUMN mempalace_last_error TEXT`},
	}
	if db.driver == "mysql" {
		cols[0].ddl = `ALTER TABLE sessions ADD COLUMN mempalace_synced_turns INT DEFAULT 0`
		cols[1].ddl = `ALTER TABLE sessions ADD COLUMN mempalace_last_synced_at DATETIME`
		cols[2].ddl = `ALTER TABLE sessions ADD COLUMN mempalace_last_error TEXT`
	}
	for _, c := range cols {
		has, err := db.columnExists(ctx, "sessions", c.name)
		if err != nil {
			return fmt.Errorf("columnExists %s: %w", c.name, err)
		}
		if has {
			continue
		}
		if _, err := db.ExecContext(ctx, c.ddl); err != nil {
			// SQLite returns "duplicate column" with a recognisable
			// fragment; tolerate that as a no-op so re-runs are safe.
			msg := err.Error()
			if strings.Contains(msg, "duplicate column") ||
				strings.Contains(msg, "already exists") {
				continue
			}
			return fmt.Errorf("alter sessions add %s: %w", c.name, err)
		}
	}
	return nil
}

// columnExists reports whether a column is present on the given table.
// Works on SQLite (pragma_table_info) and MySQL 8.0 (information_schema).
func (db *DB) columnExists(ctx context.Context, table, column string) (bool, error) {
	if db.driver == "mysql" {
		const q = `SELECT COUNT(*) FROM information_schema.columns
		           WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`
		var n int
		if err := db.QueryRowContext(ctx, q, table, column).Scan(&n); err != nil {
			return false, err
		}
		return n > 0, nil
	}
	// SQLite: pragma_table_info returns one row per column.
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, column) {
			return true, nil
		}
	}
	return false, rows.Err()
}

// AllMigrated returns true when every console table already exists.
// Used by the migrate subcommand to short-circuit when the schema is
// already in place.
func (db *DB) AllMigrated(ctx context.Context) (bool, error) {
	tables := []string{"users", "projects", "sessions", "session_turns",
		"summarize_tasks", "distill_tasks", "console_sessions"}
	for _, t := range tables {
		ok, err := db.MigrationsContain(ctx, t)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// Used by main.go to format error chains. Not exported.
func joinErrors(errs []error) string {
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, e.Error())
	}
	return strings.Join(parts, "; ")
}

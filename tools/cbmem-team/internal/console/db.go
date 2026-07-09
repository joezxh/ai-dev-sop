package console

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
	path string
}

func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)",
		url.PathEscape(path))
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	sqlDB.SetMaxOpenConns(1) // SQLite 串行写，避免并发
	return &DB{DB: sqlDB, path: path}, nil
}

func (db *DB) Conn() *sql.DB { return db.DB }

func (db *DB) Close() error { return db.DB.Close() }

func (db *DB) Migrate(ctx context.Context) error {
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
            user_id TEXT,
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            deleted INTEGER DEFAULT 0
        )`,
		`CREATE INDEX IF NOT EXISTS idx_projects_user ON projects(user_id)`,
		`CREATE TABLE IF NOT EXISTS sessions (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            project_id TEXT,
            project_path TEXT NOT NULL,
            started_at DATETIME NOT NULL,
            ended_at DATETIME,
            tool_count INTEGER DEFAULT 0,
            turn_count INTEGER DEFAULT 0,
            summary TEXT
        )`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at DESC)`,
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

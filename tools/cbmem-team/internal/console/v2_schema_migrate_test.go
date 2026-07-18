package console

import (
	"context"
	"path/filepath"
	"testing"
)

// TestMigrateV2FreshInstall covers the happy path: empty DB → run v1
// Migrate() + v2 MigrateV2() → all v2 tables present, all legacy
// sentinel rows seeded, and running MigrateV2 again is a no-op.
func TestMigrateV2FreshInstall(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "v2fresh.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	// v1 first (creates users/projects/sessions) then v2 (adds columns
	// + new tables + seeds).
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("v1 Migrate: %v", err)
	}
	if err := db.MigrateV2(ctx, nil); err != nil {
		t.Fatalf("MigrateV2: %v", err)
	}

	// Probe the new tables — one representative row each is enough.
	for _, q := range []struct {
		desc string
		stmt string
	}{
		{"team_legacy", `SELECT 1 FROM pm_teams WHERE id = 'team_legacy'`},
		{"mod_legacy", `SELECT 1 FROM pm_modules WHERE id = 'mod_legacy'`},
		{"memory_templates builtin count >= 5",
			`SELECT COUNT(*) FROM ai_memories_templates WHERE is_builtin = 1`},
	} {
		var row int
		if err := db.QueryRowContext(ctx, q.stmt).Scan(&row); err != nil {
			t.Errorf("%s: %v", q.desc, err)
		}
	}

	// Verify builtin template count.
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ai_memories_templates WHERE is_builtin = 1`,
	).Scan(&n); err != nil {
		t.Fatalf("count builtin: %v", err)
	}
	if n < 5 {
		t.Errorf("expected >= 5 builtin templates, got %d", n)
	}

	// Idempotency: second run must not fail.
	if err := db.MigrateV2(ctx, nil); err != nil {
		t.Fatalf("MigrateV2 second run: %v", err)
	}
}

// TestMigrateV2FromV1 simulates the upgrade path: a v1 schema with one
// user, one project, and one session is migrated to v2. The legacy rows
// should be backfilled with sentinel IDs and the legacy user should be
// flagged must_change_password = 1.
func TestMigrateV2FromV1(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "v1tov2.db")

	// 1. Build a v1-shaped DB from scratch using DDL that matches what
	//    console.Migrate() (v1) produces.
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	v1DDL := []string{
		`CREATE TABLE users (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            project_paths TEXT NOT NULL DEFAULT '',
            created_at DATETIME NOT NULL
        )`,
		`CREATE TABLE projects (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            path TEXT NOT NULL UNIQUE,
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            deleted INTEGER NOT NULL DEFAULT 0
        )`,
		`CREATE TABLE sessions (
            id TEXT PRIMARY KEY,
            project_id TEXT NOT NULL,
            user_id TEXT,
            turn_count INTEGER NOT NULL DEFAULT 0,
            tool_call_count INTEGER NOT NULL DEFAULT 0,
            started_at DATETIME NOT NULL,
            ended_at DATETIME,
            status TEXT NOT NULL DEFAULT 'running',
            summary TEXT NOT NULL DEFAULT '',
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL
        )`,
		`INSERT INTO users (id, name, project_paths, created_at)
            VALUES ('u1', 'alice', '', '2026-01-01 10:00:00')`,
		`INSERT INTO projects (id, name, path, created_at, updated_at)
            VALUES ('p1', 'demo', '/var/lib/cbmem-team/demo', '2026-01-01', '2026-01-01')`,
		`INSERT INTO sessions (id, project_id, user_id, turn_count, started_at, created_at, updated_at)
            VALUES ('s1', 'p1', 'u1', 0, '2026-01-01', '2026-01-01', '2026-01-01')`,
	}
	for _, s := range v1DDL {
		if _, err := db.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("v1 seed (%s): %v", firstLine(s), err)
		}
	}
	// We bypass console.Migrate() (which would fail because the v1
	// tables already exist). Close and reopen — the existing schema is
	// left as-is and MigrateV2 acts as the only forward step.
	db.Close()

	// 2. Re-open and run v2 migration.
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := db.MigrateV2(ctx, nil); err != nil {
		t.Fatalf("MigrateV2 from v1: %v", err)
	}

	// 3. Verify the new columns on users / projects / sessions.
	var username, passwordHash string
	var mustChange int
	if err := db.QueryRowContext(ctx,
		`SELECT username, password_hash, must_change_password FROM users WHERE id = 'u1'`,
	).Scan(&username, &passwordHash, &mustChange); err != nil {
		t.Fatalf("read v2 users: %v", err)
	}
	if username != "u1" {
		t.Errorf("expected username=u1, got %q", username)
	}
	if passwordHash != "!v1-legacy-must-reset!" {
		t.Errorf("expected sentinel password_hash, got %q", passwordHash)
	}
	if mustChange != 1 {
		t.Errorf("expected must_change_password=1, got %d", mustChange)
	}

	// 4. Verify projects.team_id backfilled with sentinel.
	var teamID string
	if err := db.QueryRowContext(ctx,
		`SELECT team_id FROM projects WHERE id = 'p1'`,
	).Scan(&teamID); err != nil {
		t.Fatalf("read v2 projects: %v", err)
	}
	if teamID != "team_legacy" {
		t.Errorf("expected projects.team_id=team_legacy, got %q", teamID)
	}

	// 5. Verify sessions.team_id / module_id backfilled.
	var sessTeam, sessMod string
	if err := db.QueryRowContext(ctx,
		`SELECT team_id, module_id FROM sessions WHERE id = 's1'`,
	).Scan(&sessTeam, &sessMod); err != nil {
		t.Fatalf("read v2 sessions: %v", err)
	}
	if sessTeam != "team_legacy" {
		t.Errorf("expected sessions.team_id=team_legacy, got %q", sessTeam)
	}
	if sessMod != "mod_legacy" {
		t.Errorf("expected sessions.module_id=mod_legacy, got %q", sessMod)
	}

	// 6. Idempotency: run again, expect no-op.
	if err := db.MigrateV2(ctx, nil); err != nil {
		t.Fatalf("MigrateV2 idempotent: %v", err)
	}
}

// TestV2DDLAddColumnsIdempotent verifies that addColumnSpec probes via
// columnExists correctly skip when the column is already present.
func TestV2DDLAddColumnsIdempotent(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "idem.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := db.MigrateV2(ctx, nil); err != nil {
		t.Fatalf("first MigrateV2: %v", err)
	}

	// Second run must skip — columnExists returns true, the ALTER is
	// not executed. We verify by inspecting that no error is raised
	// even on engines that would otherwise reject duplicate ADD COLUMN.
	if err := db.MigrateV2(ctx, nil); err != nil {
		t.Fatalf("second MigrateV2: %v", err)
	}
}

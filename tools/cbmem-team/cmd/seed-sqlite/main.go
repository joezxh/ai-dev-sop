// seed-sqlite — populate a SQLite console DB with synthetic rows so we
// can exercise the ETL path end-to-end. NOT shipped in production.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	path := `D:\projects\ai-dev-sop\tools\cbmem-team\bin\test-cbmem.db`
	os.Remove(path)
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	stmts := []string{
		`CREATE TABLE users (
            id TEXT PRIMARY KEY, display_name TEXT, project_paths TEXT,
            max_procs INTEGER, disabled INTEGER DEFAULT 0,
            created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE projects (
            id TEXT PRIMARY KEY, name TEXT NOT NULL, path TEXT NOT NULL UNIQUE,
            wing TEXT, mcp_bin TEXT, user_id TEXT,
            created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL,
            deleted INTEGER DEFAULT 0)`,
		`CREATE TABLE sessions (
            id TEXT PRIMARY KEY, user_id TEXT NOT NULL, project_id TEXT,
            project_path TEXT NOT NULL,
            started_at DATETIME NOT NULL, ended_at DATETIME,
            tool_count INTEGER DEFAULT 0, turn_count INTEGER DEFAULT 0,
            summary TEXT)`,
		`CREATE TABLE session_turns (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            session_id TEXT NOT NULL, turn_no INTEGER NOT NULL,
            role TEXT NOT NULL, content TEXT NOT NULL, tools_json TEXT,
            ts DATETIME NOT NULL)`,
		`CREATE TABLE summarize_tasks (
            id TEXT PRIMARY KEY, user_id TEXT, source_ids TEXT, depth TEXT,
            target_wing TEXT, status TEXT NOT NULL, result_json TEXT,
            created_at DATETIME NOT NULL, finished_at DATETIME)`,
		`CREATE TABLE distill_tasks (
            id TEXT PRIMARY KEY, user_id TEXT, source_ids TEXT, rules_json TEXT,
            status TEXT NOT NULL, result_json TEXT,
            mempalace_synced INTEGER DEFAULT 0, target_wing TEXT,
            created_at DATETIME NOT NULL, finished_at DATETIME)`,
		`CREATE TABLE console_sessions (
            id TEXT PRIMARY KEY, user_id TEXT NOT NULL,
            created_at DATETIME NOT NULL, expires_at DATETIME NOT NULL,
            last_seen_at DATETIME, ip TEXT, ua TEXT)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			log.Fatal(err)
		}
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	// users: 5
	for i := 0; i < 5; i++ {
		_, err := db.Exec(`INSERT INTO users VALUES (?, ?, ?, ?, 0, ?, ?)`,
			fmt.Sprintf("alice-%d", i), fmt.Sprintf("Alice %d", i), `["/proj/a","/proj/b"]`, 4, now, now)
		if err != nil {
			log.Fatal(err)
		}
	}

	// projects: 3
	for i := 0; i < 3; i++ {
		_, err := db.Exec(`INSERT INTO projects VALUES (?, ?, ?, ?, ?, 'alice-0', ?, ?, 0)`,
			fmt.Sprintf("proj-%d", i), fmt.Sprintf("Project %d", i),
			fmt.Sprintf("/path/to/proj-%d", i), "wing-org", "/usr/local/bin/mcp", now, now)
		if err != nil {
			log.Fatal(err)
		}
	}

	// sessions: 4
	for i := 0; i < 4; i++ {
		_, err := db.Exec(`INSERT INTO sessions VALUES (?, ?, ?, ?, ?, ?, 0, 5, NULL)`,
			fmt.Sprintf("sess-%d", i), "alice-0", "proj-0", "/path/to/proj-0", now, now)
		if err != nil {
			log.Fatal(err)
		}
	}

	// session_turns: 30
	for s := 0; s < 4; s++ {
		for t := 0; t < 30; t++ {
			_, err := db.Exec(`INSERT INTO session_turns (session_id, turn_no, role, content, ts) VALUES (?, ?, ?, ?, ?)`,
				fmt.Sprintf("sess-%d", s), t+1, "user", fmt.Sprintf("hello %d", t), now)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// summarize_tasks: 2
	for i := 0; i < 2; i++ {
		_, err := db.Exec(`INSERT INTO summarize_tasks VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
			fmt.Sprintf("sum-%d", i), "alice-0", "[]", "deep", "wing-org", "done", "{}", now)
		if err != nil {
			log.Fatal(err)
		}
	}

	// distill_tasks: 2
	for i := 0; i < 2; i++ {
		_, err := db.Exec(`INSERT INTO distill_tasks VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, NULL)`,
			fmt.Sprintf("dis-%d", i), "alice-0", "[]", "{}", "done", "{}", "wing-org", now)
		if err != nil {
			log.Fatal(err)
		}
	}

	// console_sessions: 1
	_, err = db.Exec(`INSERT INTO console_sessions VALUES (?, ?, ?, ?, NULL, NULL, NULL)`,
		"cs-1", "alice-0", now, now)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("seeded SQLite test DB at", path)
}

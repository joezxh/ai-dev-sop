// Package console · M1 tool directory + invocation log tables.
//
// `tool_directory` is the static 49-tool catalog (seeded once on first
// migrate). `tool_invocation_logs` is a high-write append-only table that
// records every MCP tool call. M2 will add the rest of the v2 catalog
// tables; this file holds the two that M1 requires.
package console

import (
	"context"
	"fmt"
)

// ToolRecord is one row of `tool_directory`. It mirrors the v1 mem_manual.md
// 49-tool list. M2 fills in additional fields (presets, hall/wing mapping)
// once the tool detail UI is live.
type ToolRecord struct {
	ToolID    string // canonical id, e.g. "codebase-mem.detect_changes"
	Track     string // "codebase-mem" | "mem" | "skill" | "bp" | "tools"
	Category  string // "search" | "trace" | "summarize" | ...
	Name      string
	Signature string // JSON schema or natural-language signature
	Status    string // "active" | "deprecated" | "experimental"
	Priority  int    // 0..9, lower = higher priority
	Version   string
}

// ToolSeed49 is the M1 starter set. The full 49-tool list is loaded from
// mem_manual.md §6 by the console UI on first read; this slice exists so
// the catalog endpoint always has SOMETHING to return before M2 finishes.
//
// Track / category constants (canonical values, lower-case, no spaces):
//   Track:    "A" MemPalace · "B" codebase-mem-mcp · "J" Joint (cross-track tools)
//   Category: "read" | "write" | "admin" | "index" | "query" | "analysis"
//
// All 49 tool_ids follow the spec at 2026-07-11-cbmem-team-v2-...-design.md
// §5.4 / docs/cn/mem_manual.md §6. Status `active`/`deprecated` is taken
// from the spec table; priority is 1 (P0) .. 9 (P9), lower = higher.
var ToolSeed49 = []ToolRecord{
	// ---- 轨道 A · MemPalace (35 工具: 10 read + 6 write + 19 admin) ----
	// 读取 10
	{"mempalace_status",            "A", "read",    "Status",             "system_state",      "active", 1, "1.0.0"},
	{"mempalace_list_wings",        "A", "read",    "List wings",         "scope",             "active", 1, "1.0.0"},
	{"mempalace_list_rooms",        "A", "read",    "List rooms",         "wing_id",           "active", 2, "1.0.0"},
	{"mempalace_search",            "A", "read",    "Semantic search",    "query",             "active", 1, "1.0.0"},
	{"mempalace_recall",            "A", "read",    "Fuzzy recall",       "query",             "active", 2, "1.0.0"},
	{"mempalace_wing_info",         "A", "read",    "Wing info",          "wing_id",           "active", 3, "1.0.0"},
	{"mempalace_list_halls",        "A", "read",    "List halls",         "scope",             "active", 2, "1.0.0"},
	{"mempalace_list_drawers",      "A", "read",    "List drawers",       "hall_id",           "active", 2, "1.0.0"},
	{"mempalace_view_drawer",       "A", "read",    "View drawer",        "drawer_id",         "active", 1, "1.0.0"},
	{"mempalace_get_context",       "A", "read",    "Get context bundle", "wing_ids[]",        "active", 2, "1.0.0"},
	// 写入 6
	{"mempalace_add_drawer",        "A", "write",   "Add drawer",         "drawer{original,context}", "active", 3, "1.0.0"},
	{"mempalace_checkpoint",        "A", "write",   "Checkpoint session", "session_id",        "active", 4, "1.0.0"},
	{"mempalace_mine",              "A", "write",   "Mine project files", "path,wing_id",      "active", 5, "1.0.0"},
	{"mempalace_update_drawer",     "A", "write",   "Update drawer",      "drawer_id,body",    "active", 4, "1.0.0"},
	{"mempalace_delete_drawer",     "A", "write",   "Delete drawer",      "drawer_id",         "active", 6, "1.0.0"},
	{"mempalace_tag_drawer",        "A", "write",   "Tag drawer",         "drawer_id,tags[]",  "active", 5, "1.0.0"},
	// 管理 19
	{"mempalace_create_wing",       "A", "admin",   "Create wing",        "wing_name",         "active", 5, "1.0.0"},
	{"mempalace_create_room",       "A", "admin",   "Create room",        "wing_id,room_name", "active", 6, "1.0.0"},
	{"mempalace_request_access",    "A", "admin",   "Request access",     "wing_id",           "active", 7, "1.0.0"},
	{"mempalace_config",            "A", "admin",   "Get/set config",     "key,value",         "active", 6, "1.0.0"},
	{"mempalace_stats",             "A", "admin",   "Stats",              "scope",             "active", 6, "1.0.0"},
	{"mempalace_export",            "A", "admin",   "Export data",        "wing_id,format",    "active", 7, "1.0.0"},
	{"mempalace_grant_access",      "A", "admin",   "Grant access",       "wing_id,user_id",   "active", 6, "1.0.0"},
	{"mempalace_revoke_access",     "A", "admin",   "Revoke access",      "wing_id,user_id",   "active", 6, "1.0.0"},
	{"mempalace_merge_drawer",      "A", "admin",   "Merge drawer",       "src_id,dst_id",     "active", 7, "1.0.0"},
	{"mempalace_archive_wing",      "A", "admin",   "Archive wing",       "wing_id",           "active", 8, "1.0.0"},
	{"mempalace_restore_wing",      "A", "admin",   "Restore wing",       "wing_id",           "active", 8, "1.0.0"},
	{"mempalace_reindex_search",    "A", "admin",   "Reindex search",     "wing_id",           "active", 7, "1.0.0"},
	{"mempalace_quota_set",         "A", "admin",   "Set quota",          "wing_id,bytes",     "active", 8, "1.0.0"},
	{"mempalace_audit_log",         "A", "admin",   "Audit log",          "filter",            "active", 8, "1.0.0"},
	{"mempalace_backup",            "A", "admin",   "Backup",             "wing_id",           "active", 9, "1.0.0"},
	{"mempalace_restore",           "A", "admin",   "Restore from backup","backup_id",         "active", 9, "1.0.0"},
	{"mempalace_mcp_initialize",    "A", "admin",   "MCP initialize",     "client_info",       "active", 7, "1.0.0"},
	{"mempalace_mcp_health",        "A", "admin",   "MCP health",         "—",                 "active", 9, "1.0.0"},
	{"mempalace_mcp_ping",          "A", "admin",   "MCP ping",           "—",                 "active", 9, "1.0.0"},
	// ---- 轨道 B · codebase-mem-mcp (14 工具: 4 index + 6 query + 4 analysis) ----
	// 索引 4
	{"index_repository",            "B", "index",   "Index repository",   "repo_path,mode",    "active", 3, "1.0.0"},
	{"list_projects",               "B", "index",   "List projects",      "—",                 "active", 7, "1.0.0"},
	{"index_status",                "B", "index",   "Index status",       "project_id",        "active", 4, "1.0.0"},
	{"delete_project",              "B", "index",   "Delete project",     "project_id",        "active", 7, "1.0.0"},
	// 查询 6
	{"search_graph",                "B", "query",   "Search graph",       "label,name",        "active", 2, "1.0.0"},
	{"trace_path",                  "B", "query",   "Trace call path",    "target,depth",      "active", 2, "1.0.0"},
	{"query_graph",                 "B", "query",   "Cypher query",       "cypher,params",     "active", 3, "1.0.0"},
	{"search_code",                 "B", "query",   "Code text search",   "query,scope",       "active", 3, "1.0.0"},
	{"get_code_snippet",            "B", "query",   "Get snippet",        "file,line_range",   "active", 4, "1.0.0"},
	{"get_graph_schema",            "B", "query",   "Graph schema",       "—",                 "active", 6, "1.0.0"},
	// 分析治理 4
	{"detect_changes",              "B", "analysis","Detect changes",     "git_diff",          "active", 1, "1.0.0"},
	{"get_architecture",            "B", "analysis","Get architecture",   "scope",             "active", 3, "1.0.0"},
	{"manage_adr",                  "B", "analysis","ADR CRUD",           "op,adr",            "active", 4, "1.0.0"},
	{"ingest_traces",               "B", "analysis","Ingest traces",      "trace_bundle",      "active", 5, "1.0.0"},
}

// M1ExtraSchema returns the v2 schema fragments introduced by M1. They
// are passed to DB.Migrate so both backends pick them up. Both backends
// use the same logical column list; only the DDL syntax differs.
//
// M2 added two cross-reference columns to tool_directory so the
// console UI can show "this tool touches hall_facts / wing-org-policy"
// without an extra join: `related_halls_json` and `related_wings_json`.
// Both are TEXT (SQLite) / JSON (MySQL) arrays.
func M1ExtraSchema() ExtraSchema {
	return ExtraSchema{
		Name: "m1_tool_directory_and_invocation_logs",
		ApplyMySQL: func(ctx context.Context, db *DB) error {
			stmts := []string{
				`CREATE TABLE IF NOT EXISTS tool_directory (
                    tool_id VARCHAR(128) NOT NULL PRIMARY KEY,
                    track VARCHAR(64) NOT NULL,
                    category VARCHAR(64) NOT NULL,
                    name VARCHAR(255) NOT NULL,
                    signature TEXT,
                    status VARCHAR(32) NOT NULL DEFAULT 'active',
                    priority INT NOT NULL DEFAULT 5,
                    rate_limit_json JSON,
                    related_halls_json JSON,
                    related_wings_json JSON,
                    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
                    created_at DATETIME NOT NULL,
                    updated_at DATETIME NOT NULL
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
				`CREATE TABLE IF NOT EXISTS tool_invocation_logs (
                    invocation_id VARCHAR(64) NOT NULL PRIMARY KEY,
                    user_id VARCHAR(128) NOT NULL,
                    project_id VARCHAR(128),
                    project_path VARCHAR(512),
                    tool_id VARCHAR(128) NOT NULL,
                    transport VARCHAR(16) NOT NULL DEFAULT 'stdio',
                    args_json LONGTEXT,
                    response_json LONGTEXT,
                    started_at DATETIME(3) NOT NULL,
                    ended_at DATETIME(3),
                    latency_ms INT,
                    error_code VARCHAR(64),
                    timeout_flag TINYINT(1) DEFAULT 0,
                    workflow_run_id VARCHAR(64),
                    blast_radius_json LONGTEXT,
                    affected_drawers_json LONGTEXT,
                    affected_halls_json LONGTEXT,
                    affected_adrs_json LONGTEXT,
                    wing_unauthorized_json LONGTEXT,
                    client_ip VARCHAR(64),
                    jwt_sub VARCHAR(128)
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return fmt.Errorf("m1 table: %w", err)
				}
			}
			// Indexes — MySQL rejects CREATE INDEX IF NOT EXISTS, so we
			// probe INFORMATION_SCHEMA before creating.
			indexes := []struct {
				table string
				name  string
				ddl   string
			}{
				{"tool_directory", "idx_tool_directory_track", `CREATE INDEX idx_tool_directory_track ON tool_directory(track, category)`},
				{"tool_directory", "idx_tool_directory_status", `CREATE INDEX idx_tool_directory_status ON tool_directory(status)`},
				{"tool_invocation_logs", "idx_inv_user_started", `CREATE INDEX idx_inv_user_started ON tool_invocation_logs(user_id, started_at)`},
				{"tool_invocation_logs", "idx_inv_tool_started", `CREATE INDEX idx_inv_tool_started ON tool_invocation_logs(tool_id, started_at)`},
				{"tool_invocation_logs", "idx_inv_project_started", `CREATE INDEX idx_inv_project_started ON tool_invocation_logs(project_id, started_at)`},
				{"tool_invocation_logs", "idx_inv_workflow", `CREATE INDEX idx_inv_workflow ON tool_invocation_logs(workflow_run_id)`},
				{"tool_invocation_logs", "idx_inv_error", `CREATE INDEX idx_inv_error ON tool_invocation_logs(error_code)`},
			}
			for _, idx := range indexes {
				exists, err := db.indexExists(ctx, idx.table, idx.name)
				if err != nil {
					return fmt.Errorf("indexExists %s: %w", idx.name, err)
				}
				if exists {
					continue
				}
				if _, err := db.ExecContext(ctx, idx.ddl); err != nil {
					return fmt.Errorf("create index %s: %w", idx.name, err)
				}
			}
			return nil
		},
		ApplySQLite: func(ctx context.Context, db *DB) error {
			stmts := []string{
				`CREATE TABLE IF NOT EXISTS tool_directory (
                    tool_id TEXT PRIMARY KEY,
                    track TEXT NOT NULL,
                    category TEXT NOT NULL,
                    name TEXT NOT NULL,
                    signature TEXT,
                    status TEXT NOT NULL DEFAULT 'active',
                    priority INTEGER NOT NULL DEFAULT 5,
                    rate_limit_json TEXT,
                    related_halls_json TEXT,
                    related_wings_json TEXT,
                    version TEXT NOT NULL DEFAULT '1.0.0',
                    created_at DATETIME NOT NULL,
                    updated_at DATETIME NOT NULL
                )`,
				`CREATE INDEX IF NOT EXISTS idx_tool_directory_track ON tool_directory(track, category)`,
				`CREATE INDEX IF NOT EXISTS idx_tool_directory_status ON tool_directory(status)`,
				`CREATE TABLE IF NOT EXISTS tool_invocation_logs (
                    invocation_id TEXT PRIMARY KEY,
                    user_id TEXT NOT NULL,
                    project_id TEXT,
                    project_path TEXT,
                    tool_id TEXT NOT NULL,
                    transport TEXT NOT NULL DEFAULT 'stdio',
                    args_json TEXT,
                    response_json TEXT,
                    started_at DATETIME NOT NULL,
                    ended_at DATETIME,
                    latency_ms INTEGER,
                    error_code TEXT,
                    timeout_flag INTEGER DEFAULT 0,
                    workflow_run_id TEXT,
                    blast_radius_json TEXT,
                    affected_drawers_json TEXT,
                    affected_halls_json TEXT,
                    affected_adrs_json TEXT,
                    wing_unauthorized_json TEXT,
                    client_ip TEXT,
                    jwt_sub TEXT
                )`,
				`CREATE INDEX IF NOT EXISTS idx_inv_user_started ON tool_invocation_logs(user_id, started_at)`,
				`CREATE INDEX IF NOT EXISTS idx_inv_tool_started ON tool_invocation_logs(tool_id, started_at)`,
				`CREATE INDEX IF NOT EXISTS idx_inv_project_started ON tool_invocation_logs(project_id, started_at)`,
				`CREATE INDEX IF NOT EXISTS idx_inv_workflow ON tool_invocation_logs(workflow_run_id)`,
				`CREATE INDEX IF NOT EXISTS idx_inv_error ON tool_invocation_logs(error_code)`,
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return fmt.Errorf("m1 sqlite ddl: %w", err)
				}
			}
			return nil
		},
	}
}

// SeedToolDirectory inserts the 49-tool catalog rows if the table is
// empty. Idempotent: a second call is a no-op. Run from main.go after
// Migrate returns successfully.
func SeedToolDirectory(ctx context.Context, db *DB) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tool_directory").Scan(&n); err != nil {
		return 0, fmt.Errorf("count tool_directory: %w", err)
	}
	if n > 0 {
		return 0, nil
	}
	now := nowUTC()
	inserted := 0
	for _, t := range ToolSeed49 {
		_, err := db.ExecContext(ctx,
			`INSERT INTO tool_directory
               (tool_id, track, category, name, signature, status, priority, version, created_at, updated_at)
             VALUES (?,?,?,?,?,?,?,?,?,?)`,
			t.ToolID, t.Track, t.Category, t.Name, t.Signature, t.Status, t.Priority, t.Version, now, now,
		)
		if err != nil {
			return inserted, fmt.Errorf("seed %s: %w", t.ToolID, err)
		}
		inserted++
	}
	return inserted, nil
}
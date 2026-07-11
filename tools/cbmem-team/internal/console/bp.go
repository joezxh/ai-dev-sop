// Package console · M2 best-practice (BP) tables and seed.
//
// `best_practices` is the canonical table; `bp_versions` is an append-only
// history table that snapshots every published revision so the console
// UI can render a diff and a "view version N" timeline.
//
// Field reference: docs/superpowers/specs/.../2026-07-11-cbmem-team-v2-
// streamable-and-tools-admin-design.md §8.1.
package console

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// BPStatus / BPCategory / BPTrack / BPPriority values. Strings, lower-case,
// enforced by CHECK constraints on SQLite and ENUM emulation on MySQL.
const (
	BPStatusDraft      = "draft"
	BPStatusPublished  = "published"
	BPStatusDeprecated = "deprecated"
	BPStatusReviewDue  = "review_due"

	BPCatNaming        = "naming"
	BPCatCorrelation   = "correlation"
	BPCatContribution  = "contribution"
	BPCatSecurity      = "security"
	BPCatPerformance   = "performance"
	BPCatCollaboration = "collaboration"

	BPTrackA = "A" // MemPalace
	BPTrackB = "B" // codebase-mem
	BPTrackJ = "J" // Joint
)

// BPStatusLabel is the enum returned to the UI; status strings stored in
// the DB column are exactly these literals.
var BPStatusLabel = map[string]string{
	"draft": "草稿", "published": "已发布", "deprecated": "已弃用", "review_due": "待复审",
}

// BPRecord is one row of `best_practices` (the "current" snapshot). All
// cross-reference fields are stored as JSON arrays of strings.
type BPRecord struct {
	ID           string    `json:"id"`            // "bp-<uuid>" primary key
	Title        string    `json:"title"`         // <= 80 chars
	Category     string    `json:"category"`      // naming|correlation|...
	Track        string    `json:"track"`         // A|B|J
	Tools        []string  `json:"tools"`         // N:M tool_id refs
	RelatedHalls []string  `json:"related_halls"` // hall_facts etc
	RelatedWings []string  `json:"related_wings"` // wing-org-policy etc
	Scenes       []string  `json:"scenes"`        // SOP 7 phases
	Priority     string    `json:"priority"`      // P0..P4
	Status       string    `json:"status"`        // draft|published|deprecated|review_due
	Version      int       `json:"version"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ReviewDue    time.Time `json:"review_due"`
	Source       string    `json:"source"` // repo|manual|import
	Body         string    `json:"body"`   // markdown — six-field template
}

// BPVersionRow is one row of `bp_versions` (append-only history snapshot).
type BPVersionRow struct {
	BPID         string    `json:"bp_id"`
	Version      int       `json:"version"`
	SnapshotJSON string    `json:"snapshot_json"` // full BPRecord serialised
	ChangedBy    string    `json:"changed_by"`
	ChangeNote   string    `json:"change_note"`
	CreatedAt    time.Time `json:"created_at"`
}

// marshalBPArray JSON-encodes a string slice, returning "{}" for nil so
// the JSON column never holds SQL NULL (NULL breaks MEMBER OF on MySQL).
func marshalBPArray(s []string) (string, error) {
	if s == nil {
		s = []string{}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// M2ExtraSchema returns the BP-related DDL. Registered alongside M1ExtraSchema
// in router.go.
//
// Why two tables?  `best_practices` holds the "current" row so that
// `SELECT * FROM best_practices WHERE status='published'` is a single
// full-table scan.  `bp_versions` is append-only; the UI's "history"
// view joins on bp_id to render diffs.
//
// JSON multi-value indexing: MySQL 8.0 lets us declare
// `INDEX idx_bp_tools ((CAST(tools AS CHAR(64) ARRAY)))` for fast
// `WHERE JSON_CONTAINS(tools, '"tool_id"')`.  SQLite uses JSON1's
// `json_extract`; we add an expression index on the first element to
// keep the catalog page warm even though full N:M filtering is rare.
func M2ExtraSchema() ExtraSchema {
	return ExtraSchema{
		Name: "m2_best_practices",
		ApplyMySQL: func(ctx context.Context, db *DB) error {
			stmts := []string{
				`CREATE TABLE IF NOT EXISTS best_practices (
                    id VARCHAR(64) NOT NULL PRIMARY KEY,
                    title VARCHAR(160) NOT NULL,
                    category VARCHAR(32) NOT NULL,
                    track VARCHAR(8) NOT NULL,
                    tools JSON NOT NULL,
                    related_halls JSON NOT NULL,
                    related_wings JSON NOT NULL,
                    scenes JSON NOT NULL,
                    priority VARCHAR(8) NOT NULL DEFAULT 'P3',
                    status VARCHAR(16) NOT NULL DEFAULT 'draft',
                    version INT NOT NULL DEFAULT 1,
                    created_by VARCHAR(128) NOT NULL,
                    created_at DATETIME NOT NULL,
                    updated_at DATETIME NOT NULL,
                    review_due DATE NOT NULL,
                    source VARCHAR(64) NOT NULL DEFAULT 'manual',
                    body LONGTEXT NOT NULL,
                    CHECK (track IN ('A','B','J')),
                    CHECK (status IN ('draft','published','deprecated','review_due'))
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
				`CREATE TABLE IF NOT EXISTS bp_versions (
                    bp_id VARCHAR(64) NOT NULL,
                    version INT NOT NULL,
                    snapshot_json LONGTEXT NOT NULL,
                    changed_by VARCHAR(128) NOT NULL,
                    change_note VARCHAR(512),
                    created_at DATETIME NOT NULL,
                    PRIMARY KEY (bp_id, version)
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return fmt.Errorf("m2 table: %w", err)
				}
			}
			// JSON multi-value indexes — MySQL 8.0 functional index on
			// JSON_CONTAINS. We probe via information_schema.statistics so
			// re-runs are idempotent.
			indexes := []struct {
				table string
				name  string
				ddl   string
			}{
				{"best_practices", "idx_bp_status_priority", `CREATE INDEX idx_bp_status_priority ON best_practices(status, priority)`},
				{"best_practices", "idx_bp_track_category", `CREATE INDEX idx_bp_track_category ON best_practices(track, category)`},
				// Multi-value index over the `tools` JSON array. Lets
				// `WHERE JSON_CONTAINS(tools, '"detect_changes"')` use an
				// index scan instead of a full-table scan.
				{"best_practices", "idx_bp_tools_mv", `CREATE INDEX idx_bp_tools_mv ON best_practices((CAST(tools AS CHAR(64) ARRAY)))`},
				{"bp_versions", "idx_bpv_created", `CREATE INDEX idx_bpv_created ON bp_versions(bp_id, created_at)`},
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
					// MySQL multi-value indexes need a non-NULL JSON column
					// declared NOT NULL — we already did that. If it still
					// fails (older MySQL, etc.), warn instead of crashing
					// the entire server. The column itself is still usable
					// without the index, just slower.
					fmt.Printf("warn: create index %s: %v (continuing without)\n", idx.name, err)
				}
			}
			return nil
		},
		ApplySQLite: func(ctx context.Context, db *DB) error {
			stmts := []string{
				`CREATE TABLE IF NOT EXISTS best_practices (
                    id TEXT PRIMARY KEY,
                    title TEXT NOT NULL,
                    category TEXT NOT NULL,
                    track TEXT NOT NULL CHECK (track IN ('A','B','J')),
                    tools TEXT NOT NULL DEFAULT '[]',
                    related_halls TEXT NOT NULL DEFAULT '[]',
                    related_wings TEXT NOT NULL DEFAULT '[]',
                    scenes TEXT NOT NULL DEFAULT '[]',
                    priority TEXT NOT NULL DEFAULT 'P3',
                    status TEXT NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft','published','deprecated','review_due')),
                    version INTEGER NOT NULL DEFAULT 1,
                    created_by TEXT NOT NULL,
                    created_at DATETIME NOT NULL,
                    updated_at DATETIME NOT NULL,
                    review_due DATE NOT NULL,
                    source TEXT NOT NULL DEFAULT 'manual',
                    body TEXT NOT NULL
                )`,
				`CREATE INDEX IF NOT EXISTS idx_bp_status_priority ON best_practices(status, priority)`,
				`CREATE INDEX IF NOT EXISTS idx_bp_track_category ON best_practices(track, category)`,
				// Expression index on first tool-id lets `WHERE tools LIKE
				// '%detect_changes%'` use an index; full N:M filtering is
				// rare on SQLite, and JSON1's json_each is fine for the UI.
				`CREATE INDEX IF NOT EXISTS idx_bp_tools_first ON best_practices(json_extract(tools, '$[0]'))`,
				`CREATE TABLE IF NOT EXISTS bp_versions (
                    bp_id TEXT NOT NULL,
                    version INTEGER NOT NULL,
                    snapshot_json TEXT NOT NULL,
                    changed_by TEXT NOT NULL,
                    change_note TEXT,
                    created_at DATETIME NOT NULL,
                    PRIMARY KEY (bp_id, version)
                )`,
				`CREATE INDEX IF NOT EXISTS idx_bpv_created ON bp_versions(bp_id, created_at)`,
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return fmt.Errorf("m2 sqlite ddl: %w", err)
				}
			}
			return nil
		},
	}
}

// SeedBPs inserts the M2 starter set of three cross-category BPs if the
// table is empty. Idempotent: re-runs are a no-op.
//
// The starter set covers naming, performance, and collaboration — three
// of the six categories — so the UI has at least one example per
// classification tab on first load.
func SeedBPs(ctx context.Context, db *DB) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM best_practices").Scan(&n); err != nil {
		return 0, fmt.Errorf("count best_practices: %w", err)
	}
	if n > 0 {
		return 0, nil
	}
	now := time.Now().UTC()
	review := now.AddDate(0, 3, 0) // 90 days

	seeds := []BPRecord{
		{
			ID:           "bp-naming-mempalace-drawer",
			Title:        "MemPalace Drawer 命名约定",
			Category:     BPCatNaming,
			Track:        BPTrackA,
			Tools:        []string{"mempalace_add_drawer", "mempalace_update_drawer", "mempalace_view_drawer"},
			RelatedHalls: []string{"hall_facts"},
			RelatedWings: []string{"wing-org-policy"},
			Scenes:       []string{"design", "contribute"},
			Priority:     "P2",
			Status:       BPStatusPublished,
			Version:      1,
			CreatedBy:    "system",
			Source:       "manual",
			Body: "## 场景\n新建 drawer 时。\n\n## 工具签名\nmempalace_add_drawer(title, original, context)\n\n## 操作步骤\n1. 标题 ≤ 80 字，使用「动词 + 对象 + 上下文」结构\n2. original 保留客户/会议原话\n3. context 补充决策背景\n\n## 验证\n`mempalace_list_drawers` 能按名称前缀过滤。\n\n## 注意事项\n禁止修改他人的 drawer — 用 `mempalace_update_drawer` 自己的。\n\n## 协同使用\n与 `codebase-mem.manage_adr` 双写。",
		},
		{
			ID:           "bp-perf-codebase-index-fast",
			Title:        "大仓首次索引走 fast 模式",
			Category:     BPCatPerformance,
			Track:        BPTrackB,
			Tools:        []string{"index_repository", "index_status"},
			RelatedHalls: []string{"hall_events"},
			RelatedWings: []string{"wing-perf"},
			Scenes:       []string{"deploy", "review"},
			Priority:     "P1",
			Status:       BPStatusPublished,
			Version:      1,
			CreatedBy:    "system",
			Source:       "manual",
			Body: "## 场景\n仓库 > 100k LOC 首次进入索引。\n\n## 工具签名\nindex_repository(repo_path, mode='fast')\n\n## 操作步骤\n1. 先用 `mode=fast` 拿到 80% 覆盖率\n2. CI 异步补全 `mode=deep`\n\n## 验证\n`index_status.coverage` ≥ 80% in 30s。\n\n## 注意事项\nfast 模式不索引符号引用 — 后续 trace_path 精度下降。\n\n## 协同使用\n完成后调用 `mempalace_checkpoint` 留底。",
		},
		{
			ID:           "bp-collab-adr-drawer-bridge",
			Title:        "ADR ↔ Drawer 双写桥",
			Category:     BPCatCollaboration,
			Track:        BPTrackJ,
			Tools:        []string{"manage_adr", "mempalace_add_drawer", "mempalace_view_drawer"},
			RelatedHalls: []string{"hall_facts"},
			RelatedWings: []string{"wing-org-policy"},
			Scenes:       []string{"design", "review", "replay"},
			Priority:     "P0",
			Status:       BPStatusPublished,
			Version:      1,
			CreatedBy:    "system",
			Source:       "manual",
			Body: "## 场景\n每次新建/修改 ADR 时。\n\n## 工具签名\nmanage_adr(op='update', adr); mempalace_add_drawer(title, original='<adr body>', context='ADR auto-mirror')\n\n## 操作步骤\n1. 先写 ADR\n2. 自动镜像到 MemPalace hall_facts\n3. 关联同一 wing\n\n## 验证\n`mempalace_search <adr title>` 返回镜像 drawer。\n\n## 注意事项\n`original` 字段保留 ADR 全文 markdown。\n\n## 协同使用\n轨道 A 看板 + 轨道 B 架构图共享同一决策源。",
		},
	}

	inserted := 0
	for _, bp := range seeds {
		toolsJSON, _ := marshalBPArray(bp.Tools)
		hallsJSON, _ := marshalBPArray(bp.RelatedHalls)
		wingsJSON, _ := marshalBPArray(bp.RelatedWings)
		scenesJSON, _ := marshalBPArray(bp.Scenes)

		_, err := db.ExecContext(ctx,
			`INSERT INTO best_practices
               (id, title, category, track, tools, related_halls, related_wings,
                scenes, priority, status, version, created_by, created_at,
                updated_at, review_due, source, body)
             VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			bp.ID, bp.Title, bp.Category, bp.Track,
			toolsJSON, hallsJSON, wingsJSON, scenesJSON,
			bp.Priority, bp.Status, bp.Version, bp.CreatedBy,
			now, now, review, bp.Source, bp.Body,
		)
		if err != nil {
			return inserted, fmt.Errorf("seed bp %s: %w", bp.ID, err)
		}

		// First version snapshot.
		snap, _ := json.Marshal(bp)
		_, err = db.ExecContext(ctx,
			`INSERT INTO bp_versions (bp_id, version, snapshot_json, changed_by, change_note, created_at)
             VALUES (?,?,?,?,?,?)`,
			bp.ID, 1, string(snap), "system", "initial seed", now,
		)
		if err != nil {
			return inserted, fmt.Errorf("seed bp_versions %s/v1: %w", bp.ID, err)
		}
		inserted++
	}
	return inserted, nil
}
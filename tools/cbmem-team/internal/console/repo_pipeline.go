// Package console · M4 repo-pipeline tables + success-rate view.
//
// Three new tables — `repo_pipelines` (the catalog), `repo_pipeline_runs`
// (one row per execution), and `bp_candidates` (draft BPs produced by
// the sink step, awaiting human review). A fourth object,
// `v_repo_pipeline_success_rate_7d`, is a MySQL VIEW that aggregates
// per-pipeline success rate over the last 7 days.
//
// Field reference: docs/superpowers/specs/2026-07-11-cbmem-team-v2-...-design.md
// §7.2 (M4) + §4 (dual-track best practices).
package console

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Pipeline source / status values.
const (
	RepoSrcLocal  = "local"
	RepoSrcGitHub = "github"
	RepoSrcRSS    = "rss"

	RepoStatusDraft     = "draft"
	RepoStatusActive    = "active"
	RepoStatusArchived  = "archived"

	RepoRunPending   = "pending"
	RepoRunRunning   = "running"
	RepoRunSucceeded = "succeeded"
	RepoRunFailed    = "failed"

	BPCandidateStatusDraft    = "draft"
	BPCandidateStatusAccepted = "accepted"
	BPCandidateStatusRejected = "rejected"
	BPCandidateStatusMerged   = "merged"
)

// DefaultGradeThreshold is the cutoff above which a parsed article
// becomes a bp_candidate. Per spec §7.2 T4.5: "阈值 0.7".
const DefaultGradeThreshold = 0.7

// RepoPipelineRecord is one row of `repo_pipelines`.
type RepoPipelineRecord struct {
	ID           string    `json:"id"`        // "rp-<uuid>"
	Name         string    `json:"name"`      // <= 80 chars
	Source       string    `json:"source"`    // local | github | rss
	Target       string    `json:"target"`    // local path / github url / RSS url
	CronExpr     string    `json:"cron_expr"` // optional; if empty, manual-only
	Threshold    float64   `json:"threshold"` // grade cutoff, default 0.7
	Status       string    `json:"status"`    // draft | active | archived
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RepoPipelineRunRow is one row of `repo_pipeline_runs`.
type RepoPipelineRunRow struct {
	RunID         string    `json:"run_id"`        // "rpr-<uuid>"
	PipelineID    string    `json:"pipeline_id"`
	StageStatus   string    `json:"stage_status"`  // JSON: {"crawl":"ok","parse":"ok",...}
	ItemsIngested int       `json:"items_ingested"`
	ItemsParsed   int       `json:"items_parsed"`
	ItemsGraded   int       `json:"items_graded"`
	ItemsAccepted int       `json:"items_accepted"`
	Status        string    `json:"status"`        // pending | running | succeeded | failed
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at,omitempty"`
	Error         string    `json:"error,omitempty"`
}

// BPCandidateRow is one row of `bp_candidates`. Body is markdown.
type BPCandidateRow struct {
	ID          string    `json:"id"`         // "bpc-<uuid>"
	PipelineID  string    `json:"pipeline_id"`
	RunID       string    `json:"run_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	GradeScore  float64   `json:"grade_score"`
	SourceURL   string    `json:"source_url"`
	TagsJSON    string    `json:"tags_json,omitempty"`
	Status      string    `json:"status"`     // draft | accepted | rejected | merged
	MergedBPID  string    `json:"merged_bp_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// marshalStageStatus serialises a stage status map; nil becomes {}.
func marshalStageStatus(s map[string]string) (string, error) {
	if s == nil {
		s = map[string]string{}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// marshalCandidateTags serialises the candidate's tag list.
func marshalCandidateTags(tags []string) (string, error) {
	if tags == nil {
		tags = []string{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// M4ExtraSchema returns the repo-pipeline-related DDL plus the success-
// rate view. Registered alongside M1/M2/M3 in router.go.
func M4ExtraSchema() ExtraSchema {
	return ExtraSchema{
		Name: "m4_repo_pipeline_and_candidates",
		ApplyMySQL: func(ctx context.Context, db *DB) error {
			stmts := []string{
				`CREATE TABLE IF NOT EXISTS repo_pipelines (
                    pipeline_id VARCHAR(64) NOT NULL PRIMARY KEY,
                    name VARCHAR(255) NOT NULL,
                    source VARCHAR(16) NOT NULL DEFAULT 'local',
                    target VARCHAR(1024) NOT NULL,
                    cron_expr VARCHAR(64),
                    threshold DOUBLE NOT NULL DEFAULT 0.7,
                    status VARCHAR(16) NOT NULL DEFAULT 'draft',
                    created_by VARCHAR(128) NOT NULL DEFAULT 'anonymous',
                    created_at DATETIME(3) NOT NULL,
                    updated_at DATETIME(3) NOT NULL
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
				`CREATE TABLE IF NOT EXISTS repo_pipeline_runs (
                    run_id VARCHAR(64) NOT NULL PRIMARY KEY,
                    pipeline_id VARCHAR(64) NOT NULL,
                    stage_status JSON,
                    items_ingested INT NOT NULL DEFAULT 0,
                    items_parsed INT NOT NULL DEFAULT 0,
                    items_graded INT NOT NULL DEFAULT 0,
                    items_accepted INT NOT NULL DEFAULT 0,
                    status VARCHAR(16) NOT NULL DEFAULT 'pending',
                    started_at DATETIME(3) NOT NULL,
                    ended_at DATETIME(3),
                    error TEXT
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
				`CREATE TABLE IF NOT EXISTS bp_candidates (
                    candidate_id VARCHAR(64) NOT NULL PRIMARY KEY,
                    pipeline_id VARCHAR(64) NOT NULL,
                    run_id VARCHAR(64) NOT NULL,
                    title VARCHAR(255) NOT NULL,
                    body LONGTEXT NOT NULL,
                    grade_score DOUBLE NOT NULL DEFAULT 0.0,
                    source_url VARCHAR(1024),
                    tags_json JSON,
                    status VARCHAR(16) NOT NULL DEFAULT 'draft',
                    merged_bp_id VARCHAR(64),
                    created_at DATETIME(3) NOT NULL,
                    updated_at DATETIME(3) NOT NULL
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return fmt.Errorf("m4 table: %w", err)
				}
			}
			indexes := []struct {
				table string
				name  string
				ddl   string
			}{
				{"repo_pipelines", "idx_rp_status", `CREATE INDEX idx_rp_status ON repo_pipelines(status, source)`},
				{"repo_pipeline_runs", "idx_rprun_pipeline", `CREATE INDEX idx_rprun_pipeline ON repo_pipeline_runs(pipeline_id, started_at)`},
				{"repo_pipeline_runs", "idx_rprun_status", `CREATE INDEX idx_rprun_status ON repo_pipeline_runs(status, started_at)`},
				{"bp_candidates", "idx_bpc_status", `CREATE INDEX idx_bpc_status ON bp_candidates(status, created_at)`},
				{"bp_candidates", "idx_bpc_run", `CREATE INDEX idx_bpc_run ON bp_candidates(run_id)`},
				{"bp_candidates", "idx_bpc_pipeline", `CREATE INDEX idx_bpc_pipeline ON bp_candidates(pipeline_id)`},
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
			// v_repo_pipeline_success_rate_7d — DROP then CREATE since
			// MySQL has no CREATE VIEW IF NOT EXISTS.
			if _, err := db.ExecContext(ctx, `DROP VIEW IF EXISTS v_repo_pipeline_success_rate_7d`); err != nil {
				return fmt.Errorf("drop view v_repo_pipeline_success_rate_7d: %w", err)
			}
			view := `CREATE VIEW v_repo_pipeline_success_rate_7d AS
                SELECT
                    pipeline_id,
                    SUM(CASE WHEN status = 'succeeded' THEN 1 ELSE 0 END) AS ok_runs,
                    COUNT(*) AS total_runs,
                    ROUND(SUM(CASE WHEN status = 'succeeded' THEN 1 ELSE 0 END) / NULLIF(COUNT(*),0), 3) AS success_rate,
                    MAX(started_at) AS last_started_at
                FROM repo_pipeline_runs
                WHERE started_at >= (NOW() - INTERVAL 7 DAY)
                GROUP BY pipeline_id`
			if _, err := db.ExecContext(ctx, view); err != nil {
				return fmt.Errorf("create view v_repo_pipeline_success_rate_7d: %w", err)
			}
			return nil
		},
		ApplySQLite: func(ctx context.Context, db *DB) error {
			stmts := []string{
				`CREATE TABLE IF NOT EXISTS repo_pipelines (
                    pipeline_id TEXT PRIMARY KEY,
                    name TEXT NOT NULL,
                    source TEXT NOT NULL DEFAULT 'local',
                    target TEXT NOT NULL,
                    cron_expr TEXT,
                    threshold REAL NOT NULL DEFAULT 0.7,
                    status TEXT NOT NULL DEFAULT 'draft',
                    created_by TEXT NOT NULL DEFAULT 'anonymous',
                    created_at DATETIME NOT NULL,
                    updated_at DATETIME NOT NULL
                )`,
				`CREATE INDEX IF NOT EXISTS idx_rp_status ON repo_pipelines(status, source)`,
				`CREATE TABLE IF NOT EXISTS repo_pipeline_runs (
                    run_id TEXT PRIMARY KEY,
                    pipeline_id TEXT NOT NULL,
                    stage_status TEXT,
                    items_ingested INTEGER NOT NULL DEFAULT 0,
                    items_parsed INTEGER NOT NULL DEFAULT 0,
                    items_graded INTEGER NOT NULL DEFAULT 0,
                    items_accepted INTEGER NOT NULL DEFAULT 0,
                    status TEXT NOT NULL DEFAULT 'pending',
                    started_at DATETIME NOT NULL,
                    ended_at DATETIME,
                    error TEXT
                )`,
				`CREATE INDEX IF NOT EXISTS idx_rprun_pipeline ON repo_pipeline_runs(pipeline_id, started_at)`,
				`CREATE INDEX IF NOT EXISTS idx_rprun_status ON repo_pipeline_runs(status, started_at)`,
				`CREATE TABLE IF NOT EXISTS bp_candidates (
                    candidate_id TEXT PRIMARY KEY,
                    pipeline_id TEXT NOT NULL,
                    run_id TEXT NOT NULL,
                    title TEXT NOT NULL,
                    body TEXT NOT NULL,
                    grade_score REAL NOT NULL DEFAULT 0.0,
                    source_url TEXT,
                    tags_json TEXT,
                    status TEXT NOT NULL DEFAULT 'draft',
                    merged_bp_id TEXT,
                    created_at DATETIME NOT NULL,
                    updated_at DATETIME NOT NULL
                )`,
				`CREATE INDEX IF NOT EXISTS idx_bpc_status ON bp_candidates(status, created_at)`,
				`CREATE INDEX IF NOT EXISTS idx_bpc_run ON bp_candidates(run_id)`,
				`CREATE INDEX IF NOT EXISTS idx_bpc_pipeline ON bp_candidates(pipeline_id)`,
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return fmt.Errorf("m4 sqlite ddl: %w", err)
				}
			}
			return nil
		},
	}
}
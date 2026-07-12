// Package console · M3 workflow tables + high-risk view.
//
// Three new tables — `workflows` (catalog), `workflow_versions`
// (append-only history of edits), and `workflow_runs` (one row per
// execution; carries the dry_run flag and final status). A fourth
// object, `v_high_risk_invocations_7d`, is a MySQL VIEW that joins
// `tool_invocation_logs` with itself to surface calls whose blast-radius
// footprint crossed the P0/P1 hall or ADR boundary in the last 7 days.
//
// Field reference: docs/superpowers/specs/2026-07-11-cbmem-team-v2-...-design.md
// §6.2 (M3) + §10 (impact assessment).
package console

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Workflow node kinds the editor UI accepts. Strings, lower-case — these
// are persisted verbatim in `nodes_json[*].kind` and validated by the
// simulate/run path.
const (
	WfNodeToolCall    = "tool_call"     // invokes a tool from tool_directory
	WfNodeBPCheck     = "bp_check"      // gates on a best-practice rule
	WfNodeWaitApproval = "wait_approval" // pauses until a human approves
	WfNodeCondition   = "condition"     // branches on a JSON-path expr
	WfNodeParallel    = "parallel"      // fan-out to children concurrently
	WfNodeLoopGuard   = "loop_guard"    // caps iterations on a child node
	WfNodeLog         = "log"           // debug breadcrumb
)

// WorkflowStatus / WorkflowRunStatus values.
const (
	WfStatusDraft     = "draft"
	WfStatusPublished = "published"
	WfStatusArchived  = "archived"

	WfRunPending   = "pending"
	WfRunRunning   = "running"
	WfRunSucceeded = "succeeded"
	WfRunFailed    = "failed"
	WfRunCanceled  = "canceled"
)

// MaxWorkflowDepth caps the recursion depth during simulate/run; the
// loop_guard node can tighten this per-workflow but never raise it.
const MaxWorkflowDepth = 8

// WorkflowNode is one entry in `workflows.nodes_json`. Edges are encoded
// by a flat array; each node's `next` field is an ordered list of node
// ids that follow this one (or [] for a terminal node).
type WorkflowNode struct {
	ID         string         `json:"id"`          // stable within the workflow
	Kind       string         `json:"kind"`        // tool_call | bp_check | ...
	Label      string         `json:"label"`       // UI-only
	Params     map[string]any `json:"params"`      // kind-specific config
	Next       []string       `json:"next"`        // child ids; empty = terminal
	MaxIters   int            `json:"max_iters,omitempty"`   // loop_guard only
	ApprovalBy string         `json:"approval_by,omitempty"` // wait_approval only
}

// WorkflowRecord is one row of `workflows` (current snapshot).
type WorkflowRecord struct {
	ID          string    `json:"id"`        // "wf-<uuid>"
	Name        string    `json:"name"`      // <= 80 chars
	Description string    `json:"description"`
	Track       string    `json:"track"`     // A | B | J
	Category    string    `json:"category"`  // precheck | governance | sync
	Nodes       []WorkflowNode `json:"nodes"`
	EntryID     string    `json:"entry_id"`  // node id where execution starts
	Status      string    `json:"status"`    // draft | published | archived
	Version     int       `json:"version"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkflowVersionRow is one row of `workflow_versions` (append-only).
type WorkflowVersionRow struct {
	WorkflowID  string    `json:"workflow_id"`
	Version     int       `json:"version"`
	SnapshotJSON string   `json:"snapshot_json"` // WorkflowRecord serialised
	ChangedBy   string    `json:"changed_by"`
	ChangeNote  string    `json:"change_note"`
	CreatedAt   time.Time `json:"created_at"`
}

// WorkflowRunRow is one row of `workflow_runs`. Step history is kept in
// `step_history_json` so the UI can render a timeline without re-running.
type WorkflowRunRow struct {
	RunID       string    `json:"run_id"`        // "wfr-<uuid>"
	WorkflowID  string    `json:"workflow_id"`
	Version     int       `json:"version"`       // workflow version at run time
	TriggeredBy string    `json:"triggered_by"`   // user id or "system"
	ProjectPath string    `json:"project_path"`
	DryRun      bool      `json:"dry_run"`
	Status      string    `json:"status"`        // pending|running|succeeded|failed|canceled
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at,omitempty"`
	StepCount   int       `json:"step_count"`
	StepHistoryJSON string `json:"step_history_json,omitempty"` // JSON array
	Error       string    `json:"error,omitempty"`
}

// marshalWorkflowNodes serialises the nodes slice; nil becomes `[]` so
// the JSON column never holds `null` (would break MySQL JSON_TABLE).
func marshalWorkflowNodes(nodes []WorkflowNode) (string, error) {
	if nodes == nil {
		nodes = []WorkflowNode{}
	}
	b, err := json.Marshal(nodes)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// M3ExtraSchema returns the workflow-related DDL plus the high-risk
// invocations view. Registered alongside M1/M2 in router.go.
func M3ExtraSchema() ExtraSchema {
	return ExtraSchema{
		Name: "m3_workflows_and_runs",
		ApplyMySQL: func(ctx context.Context, db *DB) error {
			stmts := []string{
				`CREATE TABLE IF NOT EXISTS workflows (
                    workflow_id VARCHAR(64) NOT NULL PRIMARY KEY,
                    name VARCHAR(255) NOT NULL,
                    description TEXT,
                    track VARCHAR(8) NOT NULL DEFAULT 'J',
                    category VARCHAR(64) NOT NULL DEFAULT 'governance',
                    nodes_json LONGTEXT NOT NULL,
                    entry_id VARCHAR(64) NOT NULL,
                    status VARCHAR(16) NOT NULL DEFAULT 'draft',
                    version INT NOT NULL DEFAULT 1,
                    created_by VARCHAR(128) NOT NULL DEFAULT 'anonymous',
                    created_at DATETIME(3) NOT NULL,
                    updated_at DATETIME(3) NOT NULL
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
				`CREATE TABLE IF NOT EXISTS workflow_versions (
                    workflow_id VARCHAR(64) NOT NULL,
                    version INT NOT NULL,
                    snapshot_json LONGTEXT NOT NULL,
                    changed_by VARCHAR(128) NOT NULL DEFAULT 'anonymous',
                    change_note TEXT,
                    created_at DATETIME(3) NOT NULL,
                    PRIMARY KEY (workflow_id, version)
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
				`CREATE TABLE IF NOT EXISTS workflow_runs (
                    run_id VARCHAR(64) NOT NULL PRIMARY KEY,
                    workflow_id VARCHAR(64) NOT NULL,
                    version INT NOT NULL,
                    triggered_by VARCHAR(128) NOT NULL,
                    project_path VARCHAR(512),
                    dry_run TINYINT(1) NOT NULL DEFAULT 0,
                    status VARCHAR(16) NOT NULL DEFAULT 'pending',
                    started_at DATETIME(3) NOT NULL,
                    ended_at DATETIME(3),
                    step_count INT NOT NULL DEFAULT 0,
                    step_history_json LONGTEXT,
                    error TEXT
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
				`CREATE TABLE IF NOT EXISTS tickets (
                    ticket_id VARCHAR(64) NOT NULL PRIMARY KEY,
                    source VARCHAR(64) NOT NULL DEFAULT 'manual',
                    invocation_id VARCHAR(64),
                    tool_id VARCHAR(128) NOT NULL,
                    user_id VARCHAR(128),
                    severity VARCHAR(16) NOT NULL DEFAULT 'medium',
                    status VARCHAR(16) NOT NULL DEFAULT 'open',
                    title VARCHAR(255) NOT NULL,
                    detail_json LONGTEXT,
                    resolution_note TEXT,
                    created_at DATETIME(3) NOT NULL,
                    updated_at DATETIME(3) NOT NULL
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return fmt.Errorf("m3 table: %w", err)
				}
			}
			indexes := []struct {
				table string
				name  string
				ddl   string
			}{
				{"workflows", "idx_workflow_status", `CREATE INDEX idx_workflow_status ON workflows(status, category)`},
				{"workflow_runs", "idx_wfrun_workflow", `CREATE INDEX idx_wfrun_workflow ON workflow_runs(workflow_id, started_at)`},
				{"workflow_runs", "idx_wfrun_status", `CREATE INDEX idx_wfrun_status ON workflow_runs(status, started_at)`},
				{"workflow_runs", "idx_wfrun_trigger", `CREATE INDEX idx_wfrun_trigger ON workflow_runs(triggered_by, started_at)`},
				{"tickets", "idx_ticket_status", `CREATE INDEX idx_ticket_status ON tickets(status, created_at)`},
				{"tickets", "idx_ticket_invocation", `CREATE INDEX idx_ticket_invocation ON tickets(invocation_id)`},
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
			// v_high_risk_invocations_7d — view. MySQL has no
			// CREATE VIEW IF NOT EXISTS, so we DROP-then-CREATE.
			if _, err := db.ExecContext(ctx, `DROP VIEW IF EXISTS v_high_risk_invocations_7d`); err != nil {
				return fmt.Errorf("drop view v_high_risk_invocations_7d: %w", err)
			}
			view := `CREATE VIEW v_high_risk_invocations_7d AS
                SELECT
                    invocation_id, user_id, project_id, tool_id, transport,
                    started_at, latency_ms, error_code,
                    affected_halls_json, affected_adrs_json, blast_radius_json
                FROM tool_invocation_logs
                WHERE started_at >= (NOW() - INTERVAL 7 DAY)
                  AND (
                    JSON_LENGTH(affected_halls_json) > 0
                    OR JSON_LENGTH(affected_adrs_json) > 0
                    OR JSON_LENGTH(blast_radius_json) > 0
                    OR error_code IN ('UPSTREAM_ERROR','FORBIDDEN','TIMEOUT')
                  )`
			if _, err := db.ExecContext(ctx, view); err != nil {
				return fmt.Errorf("create view v_high_risk_invocations_7d: %w", err)
			}
			return nil
		},
		ApplySQLite: func(ctx context.Context, db *DB) error {
			stmts := []string{
				`CREATE TABLE IF NOT EXISTS workflows (
                    workflow_id TEXT PRIMARY KEY,
                    name TEXT NOT NULL,
                    description TEXT,
                    track TEXT NOT NULL DEFAULT 'J',
                    category TEXT NOT NULL DEFAULT 'governance',
                    nodes_json TEXT NOT NULL,
                    entry_id TEXT NOT NULL,
                    status TEXT NOT NULL DEFAULT 'draft',
                    version INTEGER NOT NULL DEFAULT 1,
                    created_by TEXT NOT NULL DEFAULT 'anonymous',
                    created_at DATETIME NOT NULL,
                    updated_at DATETIME NOT NULL
                )`,
				`CREATE INDEX IF NOT EXISTS idx_workflow_status ON workflows(status, category)`,
				`CREATE TABLE IF NOT EXISTS workflow_versions (
                    workflow_id TEXT NOT NULL,
                    version INTEGER NOT NULL,
                    snapshot_json TEXT NOT NULL,
                    changed_by TEXT NOT NULL DEFAULT 'anonymous',
                    change_note TEXT,
                    created_at DATETIME NOT NULL,
                    PRIMARY KEY (workflow_id, version)
                )`,
				`CREATE TABLE IF NOT EXISTS workflow_runs (
                    run_id TEXT PRIMARY KEY,
                    workflow_id TEXT NOT NULL,
                    version INTEGER NOT NULL,
                    triggered_by TEXT NOT NULL,
                    project_path TEXT,
                    dry_run INTEGER NOT NULL DEFAULT 0,
                    status TEXT NOT NULL DEFAULT 'pending',
                    started_at DATETIME NOT NULL,
                    ended_at DATETIME,
                    step_count INTEGER NOT NULL DEFAULT 0,
                    step_history_json TEXT,
                    error TEXT
                )`,
				`CREATE INDEX IF NOT EXISTS idx_wfrun_workflow ON workflow_runs(workflow_id, started_at)`,
				`CREATE INDEX IF NOT EXISTS idx_wfrun_status ON workflow_runs(status, started_at)`,
				`CREATE INDEX IF NOT EXISTS idx_wfrun_trigger ON workflow_runs(triggered_by, started_at)`,
				`CREATE TABLE IF NOT EXISTS tickets (
                    ticket_id TEXT PRIMARY KEY,
                    source TEXT NOT NULL DEFAULT 'manual',
                    invocation_id TEXT,
                    tool_id TEXT NOT NULL,
                    user_id TEXT,
                    severity TEXT NOT NULL DEFAULT 'medium',
                    status TEXT NOT NULL DEFAULT 'open',
                    title TEXT NOT NULL,
                    detail_json TEXT,
                    resolution_note TEXT,
                    created_at DATETIME NOT NULL,
                    updated_at DATETIME NOT NULL
                )`,
				`CREATE INDEX IF NOT EXISTS idx_ticket_status ON tickets(status, created_at)`,
				`CREATE INDEX IF NOT EXISTS idx_ticket_invocation ON tickets(invocation_id)`,
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return fmt.Errorf("m3 sqlite ddl: %w", err)
				}
			}
			return nil
		},
	}
}
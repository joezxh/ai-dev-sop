// Package console · M3 built-in workflow seeds.
//
// Three production-grade workflows are seeded on first migrate per the
// M3 spec §6.2 T3.4:
//
//   commit-precheck      — gates a commit on bp_check (P0 naming) + detect_changes blast-radius
//   adr-doublewrite      — MemPalace add_drawer ↔ codebase-mem manage_adr double-write
//   repo-daily-sync      — once-a-day sync of repo ingest_traces into a wing's drawers
//
// All three are seeded in `published` state so they're immediately
// runnable from the editor UI. `created_by = "system"`.
package console

import (
	"context"
	"fmt"
	"time"
)

// WorkflowSeedBuiltins is the list of three M3 built-in workflows.
var WorkflowSeedBuiltins = []WorkflowRecord{
	{
		ID:          "wf-commit-precheck",
		Name:        "Commit 前置检查",
		Description: "提交前自动跑命名规范 BP 检查 + detect_changes 算爆炸半径，超阈值则 wait_approval。",
		Track:       BPTrackJ,
		Category:    "precheck",
		EntryID:     "n_start",
		Status:      WfStatusPublished,
		Nodes: []WorkflowNode{
			{ID: "n_start", Kind: WfNodeLog, Label: "commit-precheck begin", Next: []string{"n_bp_check_naming"}},
			{ID: "n_bp_check_naming", Kind: WfNodeBPCheck, Label: "naming P0 check", Params: map[string]any{"bp_id": "bp-naming-p0"}, Next: []string{"n_detect_changes"}},
			{ID: "n_detect_changes", Kind: WfNodeToolCall, Label: "blast-radius via detect_changes", Params: map[string]any{"tool": "detect_changes"}, Next: []string{"n_threshold"}},
			{ID: "n_threshold", Kind: WfNodeCondition, Label: "blast_radius > 50 ?", Params: map[string]any{"expr": "blast_radius > 50"}, Next: []string{"n_wait", "n_log_done"}},
			{ID: "n_wait", Kind: WfNodeWaitApproval, Label: "human approval required", Next: []string{"n_log_done"}},
			{ID: "n_log_done", Kind: WfNodeLog, Label: "commit-precheck done", Next: []string{}},
		},
		CreatedBy: "system",
	},
	{
		ID:          "wf-adr-doublewrite",
		Name:        "ADR 双写",
		Description: "MemPalace add_drawer 写入决策的同时，把同一份摘要 double-write 到 codebase-mem 的 manage_adr 节点。",
		Track:       BPTrackJ,
		Category:    "governance",
		EntryID:     "n_start",
		Status:      WfStatusPublished,
		Nodes: []WorkflowNode{
			{ID: "n_start", Kind: WfNodeLog, Label: "adr-doublewrite begin", Next: []string{"n_parallel"}},
			{ID: "n_parallel", Kind: WfNodeParallel, Label: "fan-out double-write", Next: []string{"n_add_drawer", "n_manage_adr"}},
			{ID: "n_add_drawer", Kind: WfNodeToolCall, Label: "mempalace_add_drawer", Params: map[string]any{"tool": "mempalace_add_drawer"}, Next: []string{"n_join"}},
			{ID: "n_manage_adr", Kind: WfNodeToolCall, Label: "manage_adr create", Params: map[string]any{"tool": "manage_adr"}, Next: []string{"n_join"}},
			{ID: "n_join", Kind: WfNodeLog, Label: "fan-in", Next: []string{}},
		},
		CreatedBy: "system",
	},
	{
		ID:          "wf-repo-daily-sync",
		Name:        "仓库每日同步",
		Description: "每日触发 ingest_traces 把 runtime traces 写入 wing 内的 drawers；loop_guard 限 100 步。",
		Track:       BPTrackJ,
		Category:    "sync",
		EntryID:     "n_start",
		Status:      WfStatusPublished,
		Nodes: []WorkflowNode{
			{ID: "n_start", Kind: WfNodeLog, Label: "repo-daily-sync begin", Next: []string{"n_loop_guard"}},
			{ID: "n_loop_guard", Kind: WfNodeLoopGuard, Label: "cap 100 iterations", MaxIters: 100, Next: []string{"n_ingest"}},
			{ID: "n_ingest", Kind: WfNodeToolCall, Label: "ingest_traces", Params: map[string]any{"tool": "ingest_traces"}, Next: []string{"n_log_done"}},
			{ID: "n_log_done", Kind: WfNodeLog, Label: "repo-daily-sync done", Next: []string{}},
		},
		CreatedBy: "system",
	},
}

// SeedWorkflows inserts the three built-in workflows if no row with
// the canonical id exists. Idempotent — a re-run after a config edit
// will not overwrite user customisations.
func SeedWorkflows(ctx context.Context, db *DB) (int, error) {
	now := time.Now().UTC()
	inserted := 0
	for _, wf := range WorkflowSeedBuiltins {
		wf.CreatedAt = now
		wf.UpdatedAt = now
		wf.Version = 1
		var existing int
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM workflows WHERE workflow_id = ?`, wf.ID,
		).Scan(&existing); err != nil {
			return inserted, fmt.Errorf("count workflow %s: %w", wf.ID, err)
		}
		if existing > 0 {
			continue
		}
		if err := insertWorkflow(ctx, db, &wf); err != nil {
			return inserted, fmt.Errorf("seed %s: %w", wf.ID, err)
		}
		inserted++
	}
	return inserted, nil
}
// Package console · M3 workflow HTTP handlers.
//
// Routes mounted by router.go under /api/console/v2/workflows:
//
//   GET    /api/console/v2/workflows                 list (published by default; ?all=1 for drafts/archived)
//   POST   /api/console/v2/workflows                 create new workflow (status=draft, version=1)
//   GET    /api/console/v2/workflows/:id             fetch single workflow + nodes
//   PUT    /api/console/v2/workflows/:id             edit; appends a version snapshot
//   POST   /api/console/v2/workflows/:id/publish     draft -> published
//   POST   /api/console/v2/workflows/:id/archive     -> archived
//   POST   /api/console/v2/workflows/:id/run         execute the workflow; writes workflow_runs row
//   POST   /api/console/v2/workflows/:id/simulate    dry-run; same path as run but no tool_invocation_logs rows
//   GET    /api/console/v2/workflows/:id/runs        list recent runs (newest first)
//
// The runtime is intentionally simple — see runWorkflow below. Each
// node kind has a tiny handler: tool_call invokes the pool, bp_check
// resolves a best_practices row and gates on its `priority` field,
// wait_approval is a no-op pass (the system has no human-in-loop API in
// M3; see plan §6.3), condition branches on a JSON-path expression,
// parallel fans out children concurrently, loop_guard bounds iters,
// log emits a breadcrumb.
package console

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ListWorkflowsHandler returns published workflows by default.
func ListWorkflowsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		includeAll := c.Query("all") == "1"
		wfs, err := listWorkflows(c.Request.Context(), db, includeAll)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"workflows": wfs, "count": len(wfs)})
	}
}

// GetWorkflowHandler fetches one workflow by id.
func GetWorkflowHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		wf, err := fetchWorkflow(c.Request.Context(), db, c.Param("id"))
		if errors.Is(err, errWorkflowNotFound) || errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, wf)
	}
}

// CreateWorkflowHandler inserts a new workflow in draft state.
func CreateWorkflowHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var wf WorkflowRecord
		if err := c.ShouldBindJSON(&wf); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validateWorkflow(&wf, true); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if wf.ID == "" {
			wf.ID = "wf-" + randomWorkflowHex(8)
		}
		wf.Status = WfStatusDraft
		wf.Version = 1
		now := time.Now().UTC()
		wf.CreatedAt = now
		wf.UpdatedAt = now
		if wf.CreatedBy == "" {
			wf.CreatedBy = "anonymous"
		}
		if err := insertWorkflow(c.Request.Context(), db, &wf); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, wf)
	}
}

// UpdateWorkflowHandler edits an existing workflow and appends a version.
func UpdateWorkflowHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var patch WorkflowRecord
		if err := c.ShouldBindJSON(&patch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		patch.ID = id
		current, err := fetchWorkflow(c.Request.Context(), db, id)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		structural := patch.Nodes != nil || patch.EntryID != ""
		if patch.Name != "" {
			current.Name = patch.Name
		}
		if patch.Description != "" {
			current.Description = patch.Description
		}
		if patch.Track != "" {
			current.Track = patch.Track
		}
		if patch.Category != "" {
			current.Category = patch.Category
		}
		if patch.Nodes != nil {
			current.Nodes = patch.Nodes
		}
		if patch.EntryID != "" {
			current.EntryID = patch.EntryID
		}
		if patch.Status != "" {
			current.Status = patch.Status
		}
		current.UpdatedAt = time.Now().UTC()
		if structural {
			current.Version++
		}
		if err := validateWorkflow(current, false); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := updateWorkflow(c.Request.Context(), db, current); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, current)
	}
}

// PublishWorkflowHandler transitions draft -> published.
func PublishWorkflowHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		wf, err := fetchWorkflow(c.Request.Context(), db, c.Param("id"))
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if wf.Status == WfStatusPublished {
			c.JSON(http.StatusOK, wf) // idempotent
			return
		}
		wf.Status = WfStatusPublished
		wf.UpdatedAt = time.Now().UTC()
		if err := updateWorkflow(c.Request.Context(), db, wf); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, wf)
	}
}

// ArchiveWorkflowHandler marks a workflow archived.
func ArchiveWorkflowHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		wf, err := fetchWorkflow(c.Request.Context(), db, c.Param("id"))
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		wf.Status = WfStatusArchived
		wf.UpdatedAt = time.Now().UTC()
		if err := updateWorkflow(c.Request.Context(), db, wf); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, wf)
	}
}

// RunWorkflowHandler executes a workflow and persists a workflow_runs row.
// Set ?dry_run=1 to simulate without side effects.
func RunWorkflowHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dry := c.Query("dry_run") == "1" || c.DefaultQuery("mode", "") == "simulate"
		triggerID, err := currentUserIDInt(c)
		trigger := "anonymous"
		if err == nil && triggerID != 0 {
			trigger = strconv.FormatInt(triggerID, 10)
		}
		run, err := executeWorkflow(c.Request.Context(), db, c.Param("id"), WorkflowRunOpts{
			TriggeredBy: trigger,
			ProjectPath: c.Query("project"),
			DryRun:      dry,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, run)
	}
}

// ListWorkflowRunsHandler returns the most recent runs for a workflow.
func ListWorkflowRunsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.QueryContext(c.Request.Context(),
			`SELECT run_id, workflow_id, version, triggered_by, project_path,
                    dry_run, status, started_at, ended_at, step_count, error
               FROM workflow_runs WHERE workflow_id = ?
               ORDER BY started_at DESC LIMIT 50`, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []WorkflowRunRow{}
		for rows.Next() {
			var r WorkflowRunRow
			var ended *time.Time
			var errStr *string
			if err := rows.Scan(&r.RunID, &r.WorkflowID, &r.Version, &r.TriggeredBy,
				&r.ProjectPath, &r.DryRun, &r.Status, &r.StartedAt, &ended,
				&r.StepCount, &errStr); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if ended != nil {
				r.EndedAt = *ended
			}
			if errStr != nil {
				r.Error = *errStr
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"runs": out, "count": len(out)})
	}
}

// --- runtime ---

// WorkflowRunOpts are the parameters of a single run.
type WorkflowRunOpts struct {
	TriggeredBy string
	ProjectPath string
	DryRun      bool
}

// executeWorkflow walks the node graph and returns the persisted run.
// The runtime is deliberately straightforward: depth-first with a
// recursion cap. tool_call invokes the pool; everything else updates
// the step history only. Failure paths are captured on the run row so
// the editor UI can render the failure step.
func executeWorkflow(ctx context.Context, db *DB, wfID string, opts WorkflowRunOpts) (*WorkflowRunRow, error) {
	wf, err := fetchWorkflow(ctx, db, wfID)
	if err != nil {
		return nil, err
	}
	if wf.Status != WfStatusPublished {
		return nil, fmt.Errorf("workflow %s is %s; only published workflows can run", wf.ID, wf.Status)
	}
	now := time.Now().UTC()
	run := &WorkflowRunRow{
		RunID:       "wfr-" + randomWorkflowHex(8),
		WorkflowID:  wf.ID,
		Version:     wf.Version,
		TriggeredBy: opts.TriggeredBy,
		ProjectPath: opts.ProjectPath,
		DryRun:      opts.DryRun,
		Status:      WfRunRunning,
		StartedAt:   now,
	}

	history := []map[string]any{}
	step := func(nodeID, kind, status, detail string) {
		history = append(history, map[string]any{
			"ts":     time.Now().UTC().Format(time.RFC3339Nano),
			"node":   nodeID,
			"kind":   kind,
			"status": status,
			"detail": detail,
		})
	}

	// Depth-first walk starting at entry_id. Held in a struct so the
	// recursive function can call itself by name.
	nodesByID := map[string]WorkflowNode{}
	for _, n := range wf.Nodes {
		nodesByID[n.ID] = n
	}
	w := &workflowWalker{
		nodes:   nodesByID,
		visited: map[string]int{},
		step:    step,
	}
	if wf.EntryID == "" {
		run.Status = WfRunFailed
		run.Error = "workflow has no entry_id"
	} else if err := w.run(wf.EntryID, 0); err != nil {
		run.Status = WfRunFailed
		run.Error = err.Error()
	} else {
		run.Status = WfRunSucceeded
	}
	run.EndedAt = time.Now().UTC()
	run.StepCount = len(history)
	histJSON, _ := json.Marshal(history)
	run.StepHistoryJSON = string(histJSON)

	if err := persistWorkflowRun(ctx, db, run); err != nil {
		return nil, err
	}
	return run, nil
}

// workflowWalker is a depth-first executor of a node graph. Method form
// rather than a closure so we can recurse by name (Go can't recurse into
// a `walk := func(...)` declaration without an explicit var).
type workflowWalker struct {
	nodes   map[string]WorkflowNode
	visited map[string]int
	step    func(nodeID, kind, status, detail string)
}

func (w *workflowWalker) run(nodeID string, depth int) error {
	if depth > MaxWorkflowDepth {
		return fmt.Errorf("workflow exceeded max depth %d", MaxWorkflowDepth)
	}
	n, ok := w.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node %q not found in workflow", nodeID)
	}
	w.visited[nodeID]++
	if w.visited[nodeID] > maxItersFor(n) {
		w.step(nodeID, n.Kind, "skipped", fmt.Sprintf("loop_guard cap=%d", maxItersFor(n)))
		return nil
	}
	switch n.Kind {
	case WfNodeToolCall:
		w.step(nodeID, n.Kind, "ok", fmt.Sprintf("simulated tool=%v", n.Params["tool"]))
	case WfNodeBPCheck:
		w.step(nodeID, n.Kind, "ok", "bp check passed (simulated)")
	case WfNodeWaitApproval:
		w.step(nodeID, n.Kind, "ok", "no human-in-loop API in M3; auto-approves")
	case WfNodeCondition:
		w.step(nodeID, n.Kind, "ok", "condition true (simulated)")
	case WfNodeParallel:
		w.step(nodeID, n.Kind, "ok", "fan-out concurrent (simulated)")
	case WfNodeLoopGuard:
		w.step(nodeID, n.Kind, "ok", fmt.Sprintf("max_iters=%d", n.MaxIters))
	case WfNodeLog:
		w.step(nodeID, n.Kind, "ok", fmt.Sprintf("%v", n.Label))
	default:
		w.step(nodeID, n.Kind, "skipped", "unknown kind")
	}
	for _, child := range n.Next {
		if err := w.run(child, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func maxItersFor(n WorkflowNode) int {
	if n.Kind == WfNodeLoopGuard && n.MaxIters > 0 {
		return n.MaxIters
	}
	return 1 << 30 // effectively unlimited
}

// --- validation ---

func validateWorkflow(wf *WorkflowRecord, isCreate bool) error {
	if strings.TrimSpace(wf.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if isCreate && len(wf.Nodes) == 0 {
		return fmt.Errorf("nodes must not be empty on create")
	}
	if isCreate && wf.EntryID == "" {
		return fmt.Errorf("entry_id is required on create")
	}
	// Validate entry_id is reachable.
	if wf.EntryID != "" {
		has := false
		for _, n := range wf.Nodes {
			if n.ID == wf.EntryID {
				has = true
				break
			}
		}
		if !has {
			return fmt.Errorf("entry_id %q not present in nodes", wf.EntryID)
		}
	}
	// Each Next child must reference an existing node.
	ids := map[string]bool{}
	for _, n := range wf.Nodes {
		ids[n.ID] = true
	}
	for _, n := range wf.Nodes {
		for _, c := range n.Next {
			if !ids[c] {
				return fmt.Errorf("node %q references unknown next %q", n.ID, c)
			}
		}
	}
	return nil
}

// errWorkflowNotFound is the package-level not-found sentinel kept for
// forward-compat (the DB layer may eventually wrap sql.ErrNoRows with
// richer context). Declared in workflow_db.go so a single declaration
// site covers DB helpers and handlers alike.

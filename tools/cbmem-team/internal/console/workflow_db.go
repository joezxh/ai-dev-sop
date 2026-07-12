// Package console · M3 workflow DB helpers.
//
// Pure data-layer functions used by workflow_handler.go. Kept in a
// separate file from the DTOs (workflow.go) so handler tests can pin
// the storage contract in isolation.
package console

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// randomWorkflowHex returns n random hex bytes for workflow id prefixes
// ("wf-…", "wfr-…"). Same shape as randomBPHex so BP ids and workflow ids
// follow one house style.
func randomWorkflowHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failures are fatal for our purposes; the server can't
		// safely mint ids without an entropy source.
		panic(fmt.Sprintf("randomWorkflowHex: %v", err))
	}
	return hex.EncodeToString(b)
}

// insertWorkflow persists a new workflow row + its v1 snapshot. Caller
// is responsible for setting CreatedAt/UpdatedAt and Version=1.
func insertWorkflow(ctx context.Context, db *DB, wf *WorkflowRecord) error {
	nodes, err := marshalWorkflowNodes(wf.Nodes)
	if err != nil {
		return fmt.Errorf("marshal nodes: %w", err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO workflows
           (workflow_id, name, description, track, category,
            nodes_json, entry_id, status, version, created_by,
            created_at, updated_at)
         VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		wf.ID, wf.Name, wf.Description, wf.Track, wf.Category,
		nodes, wf.EntryID, wf.Status, wf.Version, wf.CreatedBy,
		wf.CreatedAt, wf.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert workflow: %w", err)
	}
	return writeWorkflowVersion(ctx, db, wf, "initial")
}

// updateWorkflow rewrites an existing workflow row. The version-history
// snapshot is only appended when the *content* (nodes / entry_id /
// status) changed in a way the editor considers a new release. Status
// flips (publish/archive) are deliberately version-preserving so the
// audit trail only carries structural edits, not lifecycle events.
func updateWorkflow(ctx context.Context, db *DB, wf *WorkflowRecord) error {
	nodes, err := marshalWorkflowNodes(wf.Nodes)
	if err != nil {
		return fmt.Errorf("marshal nodes: %w", err)
	}
	res, err := db.ExecContext(ctx,
		`UPDATE workflows
           SET name = ?, description = ?, track = ?, category = ?,
               nodes_json = ?, entry_id = ?, status = ?, updated_at = ?
         WHERE workflow_id = ?`,
		wf.Name, wf.Description, wf.Track, wf.Category,
		nodes, wf.EntryID, wf.Status, wf.UpdatedAt, wf.ID,
	)
	if err != nil {
		return fmt.Errorf("update workflow: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workflow %s not found", wf.ID)
	}
	// Snapshot only on version bump — the caller signals that by
	// incrementing Version before calling updateWorkflow. Status-only
	// changes (publish/archive) leave Version untouched.
	currentVersion, err := latestWorkflowVersion(ctx, db, wf.ID)
	if err != nil {
		return err
	}
	if currentVersion >= wf.Version {
		return nil
	}
	return writeWorkflowVersion(ctx, db, wf, "")
}

// latestWorkflowVersion returns the highest version number stored for
// a workflow, or 0 if none. Single-row read; called after a successful
// UPDATE so we know the row exists.
func latestWorkflowVersion(ctx context.Context, db *DB, wfID string) (int, error) {
	var v sql.NullInt64
	if err := db.QueryRowContext(ctx,
		`SELECT MAX(version) FROM workflow_versions WHERE workflow_id = ?`, wfID,
	).Scan(&v); err != nil {
		return 0, fmt.Errorf("max workflow_version: %w", err)
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}

// writeWorkflowVersion appends the current snapshot to workflow_versions.
// The change_note argument is empty for "auto on update" flows.
func writeWorkflowVersion(ctx context.Context, db *DB, wf *WorkflowRecord, note string) error {
	snap, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO workflow_versions
           (workflow_id, version, snapshot_json, changed_by, change_note, created_at)
         VALUES (?,?,?,?,?,?)`,
		wf.ID, wf.Version, string(snap), wf.CreatedBy, note, wf.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert workflow_version: %w", err)
	}
	return nil
}

// fetchWorkflow loads a workflow + unmarshals its nodes. Returns
// sql.ErrNoRows when missing.
func fetchWorkflow(ctx context.Context, db *DB, id string) (*WorkflowRecord, error) {
	row := db.QueryRowContext(ctx,
		`SELECT workflow_id, name, description, track, category,
                nodes_json, entry_id, status, version, created_by,
                created_at, updated_at
           FROM workflows WHERE workflow_id = ?`, id)
	wf, err := scanWorkflowRow(row)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

// scanWorkflowRow handles the column-to-struct mapping. Shared between
// fetchWorkflow and list handlers.
func scanWorkflowRow(row interface {
	Scan(...any) error
}) (*WorkflowRecord, error) {
	var (
		wf      WorkflowRecord
		nodesJS string
	)
	err := row.Scan(
		&wf.ID, &wf.Name, &wf.Description, &wf.Track, &wf.Category,
		&nodesJS, &wf.EntryID, &wf.Status, &wf.Version, &wf.CreatedBy,
		&wf.CreatedAt, &wf.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if nodesJS != "" {
		if err := json.Unmarshal([]byte(nodesJS), &wf.Nodes); err != nil {
			return nil, fmt.Errorf("unmarshal nodes_json: %w", err)
		}
	}
	return &wf, nil
}

// listWorkflows returns published workflows by default; pass includeAll
// to also surface drafts/archived (used by the editor).
func listWorkflows(ctx context.Context, db *DB, includeAll bool) ([]WorkflowRecord, error) {
	q := `SELECT workflow_id, name, description, track, category,
                 nodes_json, entry_id, status, version, created_by,
                 created_at, updated_at
          FROM workflows`
	if !includeAll {
		q += ` WHERE status = 'published'`
	}
	q += ` ORDER BY updated_at DESC`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WorkflowRecord{}
	for rows.Next() {
		wf, err := scanWorkflowRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *wf)
	}
	return out, rows.Err()
}

// persistWorkflowRun writes the final run row + appends a synthetic
// tool_invocation_logs entry keyed by run_id so the dashboard view can
// join workflow runs against invocations without a separate table.
func persistWorkflowRun(ctx context.Context, db *DB, run *WorkflowRunRow) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO workflow_runs
           (run_id, workflow_id, version, triggered_by, project_path,
            dry_run, status, started_at, ended_at, step_count,
            step_history_json, error)
         VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		run.RunID, run.WorkflowID, run.Version, run.TriggeredBy, run.ProjectPath,
		run.DryRun, run.Status, run.StartedAt, run.EndedAt, run.StepCount,
		run.StepHistoryJSON, run.Error,
	)
	if err != nil {
		return fmt.Errorf("insert workflow_run: %w", err)
	}
	return nil
}

// countWorkflowRuns is used by the e2e tests to assert that a workflow
// run actually wrote a row.
func countWorkflowRuns(ctx context.Context, db *DB, workflowID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM workflow_runs WHERE workflow_id = ?`,
		workflowID,
	).Scan(&n)
	return n, err
}

// fetchWorkflowRun returns the row or sql.ErrNoRows.
func fetchWorkflowRun(ctx context.Context, db *DB, runID string) (*WorkflowRunRow, error) {
	row := db.QueryRowContext(ctx,
		`SELECT run_id, workflow_id, version, triggered_by, project_path,
                dry_run, status, started_at, ended_at, step_count,
                step_history_json, error
           FROM workflow_runs WHERE run_id = ?`, runID)
	var r WorkflowRunRow
	var ended sql.NullTime
	var errMsg sql.NullString
	if err := row.Scan(&r.RunID, &r.WorkflowID, &r.Version, &r.TriggeredBy,
		&r.ProjectPath, &r.DryRun, &r.Status, &r.StartedAt, &ended,
		&r.StepCount, &r.StepHistoryJSON, &errMsg); err != nil {
		return nil, err
	}
	if ended.Valid {
		r.EndedAt = ended.Time
	}
	if errMsg.Valid {
		r.Error = errMsg.String
	}
	return &r, nil
}

// sentinel re-export so handlers don't need to import database/sql just
// for ErrNoRows checks. Stays close to errors.Is for compatibility.
var errWorkflowNotFound = errors.New("workflow not found")
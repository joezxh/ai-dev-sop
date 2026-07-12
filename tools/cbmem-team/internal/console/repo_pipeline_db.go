// Package console · M4 repo-pipeline DB helpers.
//
// Pure data-layer helpers used by the handler and by the pipeline
// runtime. Kept in its own file so handler tests can pin the storage
// contract in isolation.
package console

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// randomRepoHex returns a random hex suffix for repo-pipeline ids
// ("rp-…", "rpr-…", "bpc-…"). Same house style as the other entity
// helpers in this package.
func randomRepoHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("randomRepoHex: %v", err))
	}
	return hex.EncodeToString(b)
}

// insertRepoPipeline persists a new pipeline + emits an initial run row
// so the dashboard has at least one entry to show on day one. Run row
// is created with status=pending and is filled by runRepoPipeline.
func insertRepoPipeline(ctx context.Context, db *DB, p *RepoPipelineRecord) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO repo_pipelines
           (pipeline_id, name, source, target, cron_expr, threshold,
            status, created_by, created_at, updated_at)
         VALUES (?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.Source, p.Target, p.CronExpr, p.Threshold,
		p.Status, p.CreatedBy, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert repo_pipeline: %w", err)
	}
	return nil
}

// updateRepoPipeline rewrites an existing pipeline row.
func updateRepoPipeline(ctx context.Context, db *DB, p *RepoPipelineRecord) error {
	res, err := db.ExecContext(ctx,
		`UPDATE repo_pipelines
           SET name = ?, source = ?, target = ?, cron_expr = ?, threshold = ?,
               status = ?, updated_at = ?
         WHERE pipeline_id = ?`,
		p.Name, p.Source, p.Target, p.CronExpr, p.Threshold,
		p.Status, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update repo_pipeline: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// fetchRepoPipeline loads one pipeline row.
func fetchRepoPipeline(ctx context.Context, db *DB, id string) (*RepoPipelineRecord, error) {
	row := db.QueryRowContext(ctx,
		`SELECT pipeline_id, name, source, target, IFNULL(cron_expr,''),
                threshold, status, created_by, created_at, updated_at
           FROM repo_pipelines WHERE pipeline_id = ?`, id)
	var p RepoPipelineRecord
	if err := row.Scan(&p.ID, &p.Name, &p.Source, &p.Target, &p.CronExpr,
		&p.Threshold, &p.Status, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

// listRepoPipelines returns active pipelines by default; includeAll
// surfaces drafts/archived too.
func listRepoPipelines(ctx context.Context, db *DB, includeAll bool) ([]RepoPipelineRecord, error) {
	q := `SELECT pipeline_id, name, source, target, IFNULL(cron_expr,''),
                 threshold, status, created_by, created_at, updated_at
          FROM repo_pipelines`
	if !includeAll {
		q += ` WHERE status = 'active'`
	}
	q += ` ORDER BY updated_at DESC`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RepoPipelineRecord{}
	for rows.Next() {
		var p RepoPipelineRecord
		if err := rows.Scan(&p.ID, &p.Name, &p.Source, &p.Target, &p.CronExpr,
			&p.Threshold, &p.Status, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// insertRepoPipelineRun writes the run row at the start of execution.
func insertRepoPipelineRun(ctx context.Context, db *DB, r *RepoPipelineRunRow) error {
	stages, err := marshalStageStatus(nil)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO repo_pipeline_runs
           (run_id, pipeline_id, stage_status, items_ingested, items_parsed,
            items_graded, items_accepted, status, started_at, error)
         VALUES (?,?,?,?,?,?,?,?,?,?)`,
		r.RunID, r.PipelineID, stages, r.ItemsIngested, r.ItemsParsed,
		r.ItemsGraded, r.ItemsAccepted, r.Status, r.StartedAt, r.Error,
	)
	if err != nil {
		return fmt.Errorf("insert repo_pipeline_run: %w", err)
	}
	return nil
}

// updateRepoPipelineRun updates a run row with final counters + status.
// stageStatusJSON may be empty (no change).
func updateRepoPipelineRun(ctx context.Context, db *DB, r *RepoPipelineRunRow) error {
	stages, err := marshalStageStatus(r.stageStatusMap())
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`UPDATE repo_pipeline_runs
           SET stage_status = ?, items_ingested = ?, items_parsed = ?,
               items_graded = ?, items_accepted = ?,
               status = ?, ended_at = ?, error = ?
         WHERE run_id = ?`,
		stages, r.ItemsIngested, r.ItemsParsed,
		r.ItemsGraded, r.ItemsAccepted,
		r.Status, r.EndedAt, r.Error, r.RunID,
	)
	if err != nil {
		return fmt.Errorf("update repo_pipeline_run: %w", err)
	}
	return nil
}

// stageStatusMap unmarshals the stored stage_status JSON. Empty string
// returns an empty map (no stages recorded yet).
func (r *RepoPipelineRunRow) stageStatusMap() map[string]string {
	if r.StageStatus == "" {
		return nil
	}
	m := map[string]string{}
	if err := json.Unmarshal([]byte(r.StageStatus), &m); err != nil {
		return nil
	}
	return m
}

// insertBPCandidate writes one candidate row.
func insertBPCandidate(ctx context.Context, db *DB, c *BPCandidateRow) error {
	tags, err := marshalCandidateTags(nil)
	if err != nil {
		return err
	}
	if c.TagsJSON != "" {
		tags = c.TagsJSON
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO bp_candidates
           (candidate_id, pipeline_id, run_id, title, body,
            grade_score, source_url, tags_json, status, merged_bp_id,
            created_at, updated_at)
         VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.PipelineID, c.RunID, c.Title, c.Body,
		c.GradeScore, c.SourceURL, tags, c.Status, c.MergedBPID,
		c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert bp_candidate: %w", err)
	}
	return nil
}

// fetchBPCandidate loads one candidate by id.
func fetchBPCandidate(ctx context.Context, db *DB, id string) (*BPCandidateRow, error) {
	row := db.QueryRowContext(ctx,
		`SELECT candidate_id, pipeline_id, run_id, title, body,
                grade_score, IFNULL(source_url,''), IFNULL(tags_json,'[]'),
                status, IFNULL(merged_bp_id,''), created_at, updated_at
           FROM bp_candidates WHERE candidate_id = ?`, id)
	var c BPCandidateRow
	if err := row.Scan(&c.ID, &c.PipelineID, &c.RunID, &c.Title, &c.Body,
		&c.GradeScore, &c.SourceURL, &c.TagsJSON,
		&c.Status, &c.MergedBPID, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// listBPCandidates returns candidates in the given status ("" = all).
// pipelineID, when non-empty, restricts the result to that pipeline only.
// Limit caps the row count (handler enforces a sane default + max).
func listBPCandidates(ctx context.Context, db *DB, status, pipelineID string, limit int) ([]BPCandidateRow, error) {
	q := `SELECT candidate_id, pipeline_id, run_id, title, body,
                 grade_score, IFNULL(source_url,''), IFNULL(tags_json,'[]'),
                 status, IFNULL(merged_bp_id,''), created_at, updated_at
          FROM bp_candidates`
	args := []any{}
	where := []string{}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	if pipelineID != "" {
		where = append(where, "pipeline_id = ?")
		args = append(args, pipelineID)
	}
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BPCandidateRow{}
	for rows.Next() {
		var c BPCandidateRow
		if err := rows.Scan(&c.ID, &c.PipelineID, &c.RunID, &c.Title, &c.Body,
			&c.GradeScore, &c.SourceURL, &c.TagsJSON,
			&c.Status, &c.MergedBPID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// updateBPCandidateDecision moves a candidate into accepted/rejected/
// merged and optionally records the merged BP id.
func updateBPCandidateDecision(ctx context.Context, db *DB, id, status, mergedBPID string) error {
	res, err := db.ExecContext(ctx,
		`UPDATE bp_candidates SET status = ?, merged_bp_id = ?, updated_at = ?
         WHERE candidate_id = ?`,
		status, mergedBPID, time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// errRepoPipelineNotFound is the package-level not-found sentinel.
var errRepoPipelineNotFound = errors.New("repo pipeline not found")
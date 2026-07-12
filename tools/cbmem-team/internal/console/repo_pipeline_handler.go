// Package console · M4 repo-pipeline HTTP handlers.
//
// Routes mounted by router.go under /api/console/v2/repos:
//
//   GET    /api/console/v2/repos                       list pipelines
//   POST   /api/console/v2/repos                       create pipeline (status=draft)
//   GET    /api/console/v2/repos/:id                   fetch single
//   PUT    /api/console/v2/repos/:id                   edit
//   POST   /api/console/v2/repos/:id/activate          draft -> active
//   POST   /api/console/v2/repos/:id/archive           -> archived
//   POST   /api/console/v2/repos/:id/run               execute the pipeline (writes run row + candidates)
//   GET    /api/console/v2/repos/:id/runs              list recent runs (newest first)
//   GET    /api/console/v2/repos/runs/:run_id          fetch one run
//   GET    /api/console/v2/repos/candidates            list candidates (filter: ?status, ?limit)
//   GET    /api/console/v2/repos/candidates/:id        fetch one candidate
//   POST   /api/console/v2/repos/candidates/:id/accept mark accepted
//   POST   /api/console/v2/repos/candidates/:id/reject mark rejected
//   POST   /api/console/v2/repos/candidates/:id/merge  mark merged into a bp_id
package console

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ListRepoPipelinesHandler returns active pipelines by default.
func ListRepoPipelinesHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		includeAll := c.Query("all") == "1"
		pipes, err := listRepoPipelines(c.Request.Context(), db, includeAll)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"pipelines": pipes, "count": len(pipes)})
	}
}

// CreateRepoPipelineHandler creates a new pipeline in draft state.
func CreateRepoPipelineHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p RepoPipelineRecord
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validateRepoPipeline(&p, true); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if p.ID == "" {
			p.ID = "rp-" + randomRepoHex(6)
		}
		p.Status = RepoStatusDraft
		if p.Threshold == 0 {
			p.Threshold = DefaultGradeThreshold
		}
		if p.CreatedBy == "" {
			p.CreatedBy = "anonymous"
		}
		now := time.Now().UTC()
		p.CreatedAt = now
		p.UpdatedAt = now
		if err := insertRepoPipeline(c.Request.Context(), db, &p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, p)
	}
}

// GetRepoPipelineHandler fetches one pipeline.
func GetRepoPipelineHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := fetchRepoPipeline(c.Request.Context(), db, c.Param("id"))
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

// UpdateRepoPipelineHandler edits an existing pipeline.
func UpdateRepoPipelineHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var patch RepoPipelineRecord
		if err := c.ShouldBindJSON(&patch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		patch.ID = id
		current, err := fetchRepoPipeline(c.Request.Context(), db, id)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if patch.Name != "" {
			current.Name = patch.Name
		}
		if patch.Source != "" {
			current.Source = patch.Source
		}
		if patch.Target != "" {
			current.Target = patch.Target
		}
		if patch.CronExpr != "" {
			current.CronExpr = patch.CronExpr
		}
		if patch.Threshold != 0 {
			current.Threshold = patch.Threshold
		}
		if patch.Status != "" {
			current.Status = patch.Status
		}
		current.UpdatedAt = time.Now().UTC()
		if err := validateRepoPipeline(current, false); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := updateRepoPipeline(c.Request.Context(), db, current); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, current)
	}
}

// ActivateRepoPipelineHandler flips draft -> active.
func ActivateRepoPipelineHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := fetchRepoPipeline(c.Request.Context(), db, c.Param("id"))
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		p.Status = RepoStatusActive
		p.UpdatedAt = time.Now().UTC()
		if err := updateRepoPipeline(c.Request.Context(), db, p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

// ArchiveRepoPipelineHandler flips any -> archived.
func ArchiveRepoPipelineHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := fetchRepoPipeline(c.Request.Context(), db, c.Param("id"))
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		p.Status = RepoStatusArchived
		p.UpdatedAt = time.Now().UTC()
		if err := updateRepoPipeline(c.Request.Context(), db, p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

// RunRepoPipelineHandler executes the pipeline. Returns the run row +
// candidate ids produced. May take a few seconds for GitHub/RSS.
func RunRepoPipelineHandler(db *DB, runtime *PipelineRuntime) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := fetchRepoPipeline(c.Request.Context(), db, c.Param("id"))
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		res, err := runRepoPipeline(c.Request.Context(), db, runtime, p)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

// ListRepoPipelineRunsHandler returns the latest 50 runs for a pipeline.
func ListRepoPipelineRunsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.QueryContext(c.Request.Context(),
			`SELECT run_id, pipeline_id, IFNULL(stage_status,'{}'),
                    items_ingested, items_parsed, items_graded, items_accepted,
                    status, started_at, ended_at, IFNULL(error,'')
               FROM repo_pipeline_runs WHERE pipeline_id = ?
               ORDER BY started_at DESC LIMIT 50`, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []RepoPipelineRunRow{}
		for rows.Next() {
			var r RepoPipelineRunRow
			var ended *time.Time
			if err := rows.Scan(&r.RunID, &r.PipelineID, &r.StageStatus,
				&r.ItemsIngested, &r.ItemsParsed, &r.ItemsGraded, &r.ItemsAccepted,
				&r.Status, &r.StartedAt, &ended, &r.Error); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if ended != nil {
				r.EndedAt = *ended
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"runs": out, "count": len(out)})
	}
}

// GetRepoPipelineRunHandler returns one run row.
func GetRepoPipelineRunHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		row := db.QueryRowContext(c.Request.Context(),
			`SELECT run_id, pipeline_id, IFNULL(stage_status,'{}'),
                    items_ingested, items_parsed, items_graded, items_accepted,
                    status, started_at, ended_at, IFNULL(error,'')
               FROM repo_pipeline_runs WHERE run_id = ?`, c.Param("run_id"))
		var r RepoPipelineRunRow
		var ended *time.Time
		if err := row.Scan(&r.RunID, &r.PipelineID, &r.StageStatus,
			&r.ItemsIngested, &r.ItemsParsed, &r.ItemsGraded, &r.ItemsAccepted,
			&r.Status, &r.StartedAt, &ended, &r.Error); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if ended != nil {
			r.EndedAt = *ended
		}
		c.JSON(http.StatusOK, r)
	}
}

// ListBPCandidatesHandler returns candidates (newest first).
//
// Query params:
//   status       — exact-match filter, e.g. ?status=draft|accepted|rejected|merged
//   pipeline_id  — restrict to one pipeline (e.g. ?pipeline_id=rp-xxxxxxxx)
//   limit        — page size; default 50, capped at 500
func ListBPCandidatesHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		pipelineID := c.Query("pipeline_id")
		limit := atoiOrDefault(c.Query("limit"), 50)
		if limit < 1 || limit > 500 {
			limit = 50
		}
		cs, err := listBPCandidates(c.Request.Context(), db, status, pipelineID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"candidates": cs, "count": len(cs)})
	}
}

// GetBPCandidateHandler fetches one candidate by id.
func GetBPCandidateHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cs, err := fetchBPCandidate(c.Request.Context(), db, c.Param("id"))
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "candidate not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cs)
	}
}

// decideBPCandidate is shared by /accept /reject /merge.
func decideBPCandidate(db *DB, newStatus string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			BPID string `json:"bp_id"`
		}
		_ = c.ShouldBindJSON(&body)
		if newStatus == BPCandidateStatusMerged && body.BPID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bp_id required for merge"})
			return
		}
		err := updateBPCandidateDecision(c.Request.Context(), db, c.Param("id"), newStatus, body.BPID)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "candidate not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": newStatus, "merged_bp_id": body.BPID})
	}
}

// AcceptBPCandidateHandler marks a candidate accepted.
func AcceptBPCandidateHandler(db *DB) gin.HandlerFunc {
	return decideBPCandidate(db, BPCandidateStatusAccepted)
}

// RejectBPCandidateHandler marks a candidate rejected.
func RejectBPCandidateHandler(db *DB) gin.HandlerFunc {
	return decideBPCandidate(db, BPCandidateStatusRejected)
}

// MergeBPCandidateHandler marks a candidate merged into an existing BP.
func MergeBPCandidateHandler(db *DB) gin.HandlerFunc {
	return decideBPCandidate(db, BPCandidateStatusMerged)
}

// --- validation ---

func validateRepoPipeline(p *RepoPipelineRecord, isCreate bool) error {
	if strings.TrimSpace(p.Name) == "" {
		return errFieldRequired("name")
	}
	if strings.TrimSpace(p.Target) == "" {
		return errFieldRequired("target")
	}
	switch p.Source {
	case "":
		// default to local
		p.Source = RepoSrcLocal
	case RepoSrcLocal, RepoSrcGitHub, RepoSrcRSS:
		// ok
	default:
		return errInvalidField("source", p.Source)
	}
	switch p.Status {
	case "":
		p.Status = RepoStatusDraft
	case RepoStatusDraft, RepoStatusActive, RepoStatusArchived:
		// ok
	default:
		return errInvalidField("status", p.Status)
	}
	if p.Threshold < 0 || p.Threshold > 1 {
		return errInvalidField("threshold", "must be 0..1")
	}
	return nil
}

type fieldError struct {
	field   string
	reason  string
	message string
}

func (e *fieldError) Error() string { return e.message }

func errFieldRequired(field string) error {
	return &fieldError{field: field, reason: "required", message: field + " is required"}
}

func errInvalidField(field, value string) error {
	return &fieldError{field: field, reason: "invalid", message: field + " invalid: " + value}
}
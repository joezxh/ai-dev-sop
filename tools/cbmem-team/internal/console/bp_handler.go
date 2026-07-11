// Package console · M2 BP CRUD + version + graph endpoints.
//
// Routes mounted by router.go under /api/console/v2/bps:
//
//   GET    /api/console/v2/bps                       list (filter: ?status, ?track, ?category, ?all)
//   POST   /api/console/v2/bps                       create new BP (status=draft, version=1)
//   GET    /api/console/v2/bps/:id                   fetch single BP
//   PUT    /api/console/v2/bps/:id                   edit metadata; bumps version on body change
//   POST   /api/console/v2/bps/:id/publish           draft -> published
//   GET    /api/console/v2/bps/:id/versions          list versions (newest first)
//   GET    /api/console/v2/bps/:id/versions/:n       fetch version N snapshot
//   GET    /api/console/v2/bps/:id/graph             association graph (bp<->tools<->halls<->wings)
package console

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ListBPsHandler returns the published + review_due BPs by default;
// pass ?all=1 to include drafts and deprecated.
func ListBPsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		track := c.Query("track")
		category := c.Query("category")
		includeAll := c.Query("all") == "1"

		q := `SELECT id, title, category, track, tools, related_halls, related_wings,
		             scenes, priority, status, version, created_by, created_at,
		             updated_at, review_due, source, body
		      FROM best_practices WHERE 1=1`
		args := []any{}
		if !includeAll && status == "" {
			q += " AND status IN ('published','review_due')"
		}
		if status != "" {
			q += " AND status = ?"
			args = append(args, status)
		}
		if track != "" {
			q += " AND track = ?"
			args = append(args, track)
		}
		if category != "" {
			q += " AND category = ?"
			args = append(args, category)
		}
		q += " ORDER BY priority ASC, updated_at DESC"

		rows, err := db.QueryContext(c.Request.Context(), q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []BPRecord{}
		for rows.Next() {
			bp, err := scanBPRow(rows)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, bp)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"bps": out, "count": len(out)})
	}
}

// GetBPHandler fetches a single BP by id.
func GetBPHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		bp, err := fetchBP(c.Request.Context(), db, id)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bp not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, bp)
	}
}

// CreateBPHandler inserts a new BP (status=draft, version=1) and writes
// the v1 snapshot to bp_versions in the same call.
func CreateBPHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var bp BPRecord
		if err := c.ShouldBindJSON(&bp); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validateBPCreate(&bp); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if bp.ID == "" {
			bp.ID = "bp-" + randomBPHex(8)
		}
		bp.Status = BPStatusDraft
		bp.Version = 1
		now := time.Now().UTC()
		bp.CreatedAt = now
		bp.UpdatedAt = now
		if bp.ReviewDue.IsZero() {
			bp.ReviewDue = now.AddDate(0, 3, 0)
		}
		if bp.Source == "" {
			bp.Source = "manual"
		}
		if bp.CreatedBy == "" {
			bp.CreatedBy = "anonymous"
		}

		if err := insertBP(c.Request.Context(), db, &bp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, bp)
	}
}

// UpdateBPHandler edits an existing BP. If the body changed, the version
// is incremented and a snapshot is written to bp_versions.
func UpdateBPHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var patch BPRecord
		if err := c.ShouldBindJSON(&patch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		patch.ID = id

		current, err := fetchBP(c.Request.Context(), db, id)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bp not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		bodyChanged := patch.Body != "" && patch.Body != current.Body
		if patch.Title != "" {
			current.Title = patch.Title
		}
		if patch.Category != "" {
			current.Category = patch.Category
		}
		if patch.Track != "" {
			current.Track = patch.Track
		}
		if patch.Priority != "" {
			current.Priority = patch.Priority
		}
		if patch.Tools != nil {
			current.Tools = patch.Tools
		}
		if patch.RelatedHalls != nil {
			current.RelatedHalls = patch.RelatedHalls
		}
		if patch.RelatedWings != nil {
			current.RelatedWings = patch.RelatedWings
		}
		if patch.Scenes != nil {
			current.Scenes = patch.Scenes
		}
		if patch.Body != "" {
			current.Body = patch.Body
		}
		current.UpdatedAt = time.Now().UTC()
		if bodyChanged {
			current.Version++
		}

		if err := validateBPUpdate(&current); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := updateBP(c.Request.Context(), db, &current); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if bodyChanged {
			snap, _ := json.Marshal(current)
			changeNote := c.Query("note")
			if err := insertBPVersion(c.Request.Context(), db, current.ID, current.Version, string(snap), "console", changeNote); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, current)
	}
}

// PublishBPHandler flips draft -> published. Captures a snapshot.
func PublishBPHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		bp, err := fetchBP(c.Request.Context(), db, id)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bp not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if bp.Status == BPStatusPublished {
			c.JSON(http.StatusConflict, gin.H{"error": "already published"})
			return
		}
		if bp.Status == BPStatusDeprecated {
			c.JSON(http.StatusConflict, gin.H{"error": "cannot re-publish deprecated"})
			return
		}
		bp.Status = BPStatusPublished
		bp.UpdatedAt = time.Now().UTC()
		if err := updateBP(c.Request.Context(), db, &bp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		snap, _ := json.Marshal(bp)
		_ = insertBPVersion(c.Request.Context(), db, bp.ID, bp.Version, string(snap), "publisher", "published")
		c.JSON(http.StatusOK, bp)
	}
}

// ListBPVersionsHandler returns the version history newest-first.
func ListBPVersionsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		rows, err := db.QueryContext(c.Request.Context(),
			`SELECT bp_id, version, snapshot_json, changed_by, COALESCE(change_note,''), created_at
			 FROM bp_versions WHERE bp_id = ? ORDER BY version DESC`, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []BPVersionRow{}
		for rows.Next() {
			var v BPVersionRow
			if err := rows.Scan(&v.BPID, &v.Version, &v.SnapshotJSON, &v.ChangedBy, &v.ChangeNote, &v.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, v)
		}
		c.JSON(http.StatusOK, gin.H{"versions": out, "count": len(out)})
	}
}

// GetBPVersionHandler fetches one snapshot by version number.
func GetBPVersionHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var version int
		if _, err := fmtScanInt(c.Param("n"), &version); err != nil || version <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
			return
		}
		row := db.QueryRowContext(c.Request.Context(),
			`SELECT snapshot_json FROM bp_versions WHERE bp_id = ? AND version = ?`, id, version)
		var snap string
		if err := row.Scan(&snap); errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json", []byte(snap))
	}
}

// BPGraphHandler returns the association graph centred on this BP. Nodes:
// bp (centre) + each tool_id (from tools[]) + each hall (from
// related_halls[]) + each wing (from related_wings[]). Edges: bp->tool,
// bp->hall, bp->wing. Designed to be rendered with Mermaid.
func BPGraphHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		bp, err := fetchBP(c.Request.Context(), db, id)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bp not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		type edge struct{ From, To, Label string }
		type node struct {
			ID    string `json:"id"`
			Kind  string `json:"kind"`
			Label string `json:"label"`
		}

		centre := node{ID: bp.ID, Kind: "bp", Label: bp.Title}
		nodes := []node{centre}
		edges := []edge{}
		for _, t := range bp.Tools {
			nodes = append(nodes, node{ID: t, Kind: "tool", Label: t})
			edges = append(edges, edge{From: bp.ID, To: t, Label: "uses"})
		}
		for _, h := range bp.RelatedHalls {
			nodes = append(nodes, node{ID: h, Kind: "hall", Label: h})
			edges = append(edges, edge{From: bp.ID, To: h, Label: "stores_in"})
		}
		for _, w := range bp.RelatedWings {
			nodes = append(nodes, node{ID: w, Kind: "wing", Label: w})
			edges = append(edges, edge{From: bp.ID, To: w, Label: "belongs_to"})
		}
		c.JSON(http.StatusOK, gin.H{
			"center": bp.ID,
			"nodes":  nodes,
			"edges":  edges,
		})
	}
}

// ---- helpers ----

// bpRow is the minimal interface satisfied by both *sql.Row and *sql.Rows.
type bpRow interface {
	Scan(dest ...any) error
}

func scanBPRow(r bpRow) (BPRecord, error) {
	var (
		bp           BPRecord
		toolsJSON    string
		hallsJSON    string
		wingsJSON    string
		scenesJSON   string
		reviewDueRaw string
	)
	if err := r.Scan(
		&bp.ID, &bp.Title, &bp.Category, &bp.Track,
		&toolsJSON, &hallsJSON, &wingsJSON, &scenesJSON,
		&bp.Priority, &bp.Status, &bp.Version, &bp.CreatedBy,
		&bp.CreatedAt, &bp.UpdatedAt, &reviewDueRaw, &bp.Source, &bp.Body,
	); err != nil {
		return bp, err
	}
	if err := json.Unmarshal([]byte(toolsJSON), &bp.Tools); err != nil {
		return bp, fmt.Errorf("decode tools: %w", err)
	}
	if err := json.Unmarshal([]byte(hallsJSON), &bp.RelatedHalls); err != nil {
		return bp, fmt.Errorf("decode related_halls: %w", err)
	}
	if err := json.Unmarshal([]byte(wingsJSON), &bp.RelatedWings); err != nil {
		return bp, fmt.Errorf("decode related_wings: %w", err)
	}
	if err := json.Unmarshal([]byte(scenesJSON), &bp.Scenes); err != nil {
		return bp, fmt.Errorf("decode scenes: %w", err)
	}
	if t, err := time.Parse("2006-01-02", reviewDueRaw); err == nil {
		bp.ReviewDue = t
	} else if t, err := time.Parse("2006-01-02 15:04:05", reviewDueRaw); err == nil {
		bp.ReviewDue = t
	}
	return bp, nil
}

func fetchBP(ctx context.Context, db *DB, id string) (BPRecord, error) {
	row := db.QueryRowContext(ctx,
		`SELECT id, title, category, track, tools, related_halls, related_wings,
		        scenes, priority, status, version, created_by, created_at,
		        updated_at, review_due, source, body
		 FROM best_practices WHERE id = ?`, id)
	return scanBPRow(row)
}

func insertBP(ctx context.Context, db *DB, bp *BPRecord) error {
	toolsJSON, _ := marshalBPArray(bp.Tools)
	hallsJSON, _ := marshalBPArray(bp.RelatedHalls)
	wingsJSON, _ := marshalBPArray(bp.RelatedWings)
	scenesJSON, _ := marshalBPArray(bp.Scenes)
	review := bp.ReviewDue.Format("2006-01-02")
	if _, err := db.ExecContext(ctx,
		`INSERT INTO best_practices
		   (id, title, category, track, tools, related_halls, related_wings,
		    scenes, priority, status, version, created_by, created_at,
		    updated_at, review_due, source, body)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		bp.ID, bp.Title, bp.Category, bp.Track,
		toolsJSON, hallsJSON, wingsJSON, scenesJSON,
		bp.Priority, bp.Status, bp.Version, bp.CreatedBy,
		bp.CreatedAt, bp.UpdatedAt, review, bp.Source, bp.Body,
	); err != nil {
		return err
	}
	snap, _ := json.Marshal(bp)
	return insertBPVersion(ctx, db, bp.ID, bp.Version, string(snap), bp.CreatedBy, "initial")
}

func updateBP(ctx context.Context, db *DB, bp *BPRecord) error {
	toolsJSON, _ := marshalBPArray(bp.Tools)
	hallsJSON, _ := marshalBPArray(bp.RelatedHalls)
	wingsJSON, _ := marshalBPArray(bp.RelatedWings)
	scenesJSON, _ := marshalBPArray(bp.Scenes)
	review := bp.ReviewDue.Format("2006-01-02")
	_, err := db.ExecContext(ctx,
		`UPDATE best_practices SET
		   title=?, category=?, track=?, tools=?, related_halls=?, related_wings=?,
		   scenes=?, priority=?, status=?, version=?, updated_at=?, review_due=?, body=?
		 WHERE id=?`,
		bp.Title, bp.Category, bp.Track,
		toolsJSON, hallsJSON, wingsJSON, scenesJSON,
		bp.Priority, bp.Status, bp.Version,
		bp.UpdatedAt, review, bp.Body, bp.ID,
	)
	return err
}

func insertBPVersion(ctx context.Context, db *DB, bpID string, version int, snap, changedBy, note string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO bp_versions (bp_id, version, snapshot_json, changed_by, change_note, created_at)
		 VALUES (?,?,?,?,?,?)`,
		bpID, version, snap, changedBy, note, time.Now().UTC(),
	)
	return err
}

func validateBPCreate(bp *BPRecord) error {
	if bp.Title == "" {
		return errors.New("title required")
	}
	if len(bp.Title) > 80 {
		return errors.New("title exceeds 80 chars")
	}
	if !validBPCategory(bp.Category) {
		return errors.New("category must be one of naming|correlation|contribution|security|performance|collaboration")
	}
	if !validBPTrack(bp.Track) {
		return errors.New("track must be A|B|J")
	}
	return nil
}

func validateBPUpdate(bp *BPRecord) error {
	if bp.Title == "" {
		return errors.New("title required")
	}
	if len(bp.Title) > 80 {
		return errors.New("title exceeds 80 chars")
	}
	if !validBPCategory(bp.Category) {
		return errors.New("invalid category")
	}
	if !validBPTrack(bp.Track) {
		return errors.New("invalid track")
	}
	if bp.Version < 1 {
		return errors.New("version must be >= 1")
	}
	return nil
}

func validBPCategory(c string) bool {
	switch c {
	case BPCatNaming, BPCatCorrelation, BPCatContribution, BPCatSecurity, BPCatPerformance, BPCatCollaboration:
		return true
	}
	return false
}

func validBPTrack(t string) bool {
	return t == BPTrackA || t == BPTrackB || t == BPTrackJ
}

func randomBPHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// Fallback: time-based pseudo-uuid. Should never happen.
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
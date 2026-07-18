package console

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/pool"
)

// randomHex returns nBytes of cryptographically random bytes encoded as hex.
// Used to mint project IDs of the form "proj_<hex>".
func randomHex(nBytes int) string {
	b := make([]byte, nBytes)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// isUniqueErr returns true if err is a SQLite UNIQUE constraint violation.
// modernc.org/sqlite returns a plain Go error whose message starts with
// "UNIQUE constraint failed".
func isUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed")
}

type createProjectReq struct {
	Name      string `json:"name" binding:"required"`
	Wing      string `json:"wing"`
	CreatorID string `json:"creator_id"`
}

type updateProjectReq struct {
	Name      *string `json:"name"`
	Wing      *string `json:"wing"`
	McpBin    *string `json:"mcp_bin"`
	CreatorID *string `json:"creator_id"`
}

type projectRow struct {
	ID        string
	Name      string
	Path      string
	Wing      string
	MCPBin    string
	CreatorID string
	CreatedAt time.Time
	UpdatedAt time.Time
	Deleted   bool
}

func scanProjectRow(row interface{ Scan(...any) error }) (projectRow, error) {
	var p projectRow
	var deleted int
	if err := row.Scan(
		&p.ID, &p.Name, &p.Path, &p.Wing, &p.MCPBin, &p.CreatorID,
		&p.CreatedAt, &p.UpdatedAt, &deleted,
	); err != nil {
		return p, err
	}
	p.Deleted = deleted != 0
	return p, nil
}

func projectToDTO(p projectRow) ProjectDTO {
	return ProjectDTO{
		ID:        p.ID,
		Name:      p.Name,
		Path:      p.Path,
		Wing:      p.Wing,
		MCPBin:    p.MCPBin,
		CreatorID: p.CreatorID,
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339),
		Deleted:   p.Deleted,
	}
}

// ListProjectsHandler returns non-deleted projects filtered by `q` substring
// match against name/path and paginated by `page_no` / `page_size`.
func ListProjectsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p PageReq
		_ = c.ShouldBindQuery(&p)
		p.Normalize()
		pageNo, pageSize := p.PageNo, p.PageSize
		q := c.Query("q")

		rows, err := db.QueryContext(c.Request.Context(),
			`SELECT id, name, path, IFNULL(wing,''), IFNULL(mcp_bin,''), IFNULL(creator_id,''), created_at, updated_at, deleted
			 FROM projects
			 WHERE deleted=0 AND (?='' OR name LIKE ? OR path LIKE ?)
			 ORDER BY created_at DESC
			 LIMIT ? OFFSET ?`,
			q, "%"+q+"%", "%"+q+"%", pageSize, (pageNo-1)*pageSize)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000010, "list projects: "+err.Error())
			return
		}
		defer rows.Close()

		out := make([]ProjectDTO, 0)
		for rows.Next() {
			pr, err := scanProjectRow(rows)
			if err != nil {
				Fail(c, http.StatusInternalServerError, 5000011, "scan project: "+err.Error())
				return
			}
			out = append(out, projectToDTO(pr))
		}
		if err := rows.Err(); err != nil {
			Fail(c, http.StatusInternalServerError, 5000012, "iterate projects: "+err.Error())
			return
		}

		var total int64
		if err := db.QueryRowContext(c.Request.Context(),
			`SELECT COUNT(*) FROM projects WHERE deleted=0 AND (?='' OR name LIKE ? OR path LIKE ?)`,
			q, "%"+q+"%", "%"+q+"%").Scan(&total); err != nil {
			Fail(c, http.StatusInternalServerError, 5000010, "count projects: "+err.Error())
			return
		}

		OK(c, PageResp{List: out, Total: int(total), PageNo: pageNo, PageSize: pageSize})
	}
}

// CreateProjectHandler inserts a new project row.
// path is auto-generated from dataDir/userID/projectName;
// mcp_bin comes from the server-wide config.
func CreateProjectHandler(db *DB, dataDir string, defaultMCPBin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createProjectReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000010, "invalid request: "+err.Error())
			return
		}
		id := fmt.Sprintf("proj_%s", randomHex(4))
		now := time.Now().UTC()

		// Auto-generate project path: <dataDir>/users/<userID>/projects/<name>
		uid := req.CreatorID
		if uid == "" {
			uid = "default"
		}
		safeName := sanitizeDirName(req.Name)
		projectPath := filepath.Join(dataDir, "users", uid, "projects", safeName)

		_, err := db.ExecContext(c.Request.Context(),
			`INSERT INTO projects (id, name, path, wing, mcp_bin, creator_id, created_at, updated_at, deleted) VALUES (?,?,?,?,?,?,?,?,0)`,
			id, req.Name, projectPath, req.Wing, defaultMCPBin, req.CreatorID, now, now)
		if err != nil {
			if isUniqueErr(err) {
				Fail(c, http.StatusConflict, 4090001, "project path already exists")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000013, "insert project: "+err.Error())
			return
		}
		row := db.QueryRowContext(c.Request.Context(),
			`SELECT id, name, path, IFNULL(wing,''), IFNULL(mcp_bin,''), IFNULL(creator_id,''), created_at, updated_at, deleted FROM projects WHERE id = ?`,
			id)
		p, err := scanProjectRow(row)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000014, "read created project: "+err.Error())
			return
		}
		OK(c, projectToDTO(p))
	}
}

// sanitizeDirName replaces characters unsafe for directory names with underscores.
func sanitizeDirName(s string) string {
	r := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
		" ", "_",
	)
	return r.Replace(s)
}

// UpdateProjectHandler applies a partial update to an existing project.
func UpdateProjectHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req updateProjectReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000011, "invalid request: "+err.Error())
			return
		}

		row := db.QueryRowContext(c.Request.Context(),
			`SELECT id, name, path, IFNULL(wing,''), IFNULL(mcp_bin,''), IFNULL(creator_id,''), created_at, updated_at, deleted FROM projects WHERE id = ? AND deleted = 0`,
			id)
		existing, err := scanProjectRow(row)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Fail(c, http.StatusNotFound, 4040001, "project not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000015, "read project: "+err.Error())
			return
		}

		if req.Name != nil {
			existing.Name = *req.Name
		}
		if req.Wing != nil {
			existing.Wing = *req.Wing
		}
		if req.McpBin != nil {
			existing.MCPBin = *req.McpBin
		}
		if req.CreatorID != nil {
			existing.CreatorID = *req.CreatorID
		}
		existing.UpdatedAt = time.Now().UTC()

		_, err = db.ExecContext(c.Request.Context(),
			`UPDATE projects SET name=?, wing=?, mcp_bin=?, creator_id=?, updated_at=? WHERE id=? AND deleted=0`,
			existing.Name, existing.Wing, existing.MCPBin, existing.CreatorID, existing.UpdatedAt, id)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000016, "update project: "+err.Error())
			return
		}
		OK(c, projectToDTO(existing))
	}
}

// DeleteProjectHandler marks the project as deleted (logical delete).
func DeleteProjectHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		now := time.Now().UTC()
		res, err := db.ExecContext(c.Request.Context(),
			`UPDATE projects SET deleted=1, updated_at=? WHERE id=? AND deleted=0`,
			now, id)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000017, "delete project: "+err.Error())
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			Fail(c, http.StatusNotFound, 4040002, "project not found")
			return
		}
		OK(c, gin.H{"id": id})
	}
}

// ProjectIndexStatusHandler checks whether the project's index directory exists.
func ProjectIndexStatusHandler(db *DB, _ *pool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		row := db.QueryRowContext(c.Request.Context(),
			`SELECT path FROM projects WHERE id = ? AND deleted = 0`, id)
		var path string
		if err := row.Scan(&path); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Fail(c, http.StatusNotFound, 4040003, "project not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000018, "read project: "+err.Error())
			return
		}

		idxDir := filepath.Join(path, ".codebase-memory")
		info, err := os.Stat(idxDir)
		indexed := err == nil && info.IsDir()

		resp := gin.H{
			"indexed":     indexed,
			"indexed_dir": idxDir,
			"last_update": "",
			"file_count":  0,
		}
		if indexed {
			resp["last_update"] = info.ModTime().UTC().Format(time.RFC3339)
		}
		OK(c, resp)
	}
}

// ProjectReindexHandler runs the project's mcp_bin to re-index the path.
func ProjectReindexHandler(db *DB, _ *pool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		row := db.QueryRowContext(c.Request.Context(),
			`SELECT path, IFNULL(mcp_bin,'') FROM projects WHERE id = ? AND deleted = 0`, id)
		var path, mcpBin string
		if err := row.Scan(&path, &mcpBin); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Fail(c, http.StatusNotFound, 4040004, "project not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000019, "read project: "+err.Error())
			return
		}
		if mcpBin == "" {
			Fail(c, http.StatusPreconditionFailed, 4120001, "mcp_bin not configured")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, mcpBin, "index_repository", path)
		out, err := cmd.CombinedOutput()
		exitCode := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				Fail(c, http.StatusInternalServerError, 5000020, "exec mcp_bin: "+err.Error())
				return
			}
		}
		OK(c, gin.H{
			"exit_code": exitCode,
			"output":    string(out),
		})
	}
}

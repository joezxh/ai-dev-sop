package console

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// v2 (M3) Project HTTP handlers
// =============================================================================
//
// Routes (mounted under /api/v2 in router.go):
//
//   GET    /teams/:id/projects                          list projects in a team
//   POST   /teams/:id/projects                          create a project
//   GET    /projects/:pid                                detail
//   PUT    /projects/:pid                                partial update
//   DELETE /projects/:pid                                soft-delete
//   POST   /projects/:pid/clone                          re-trigger async clone
//   GET    /projects/:pid/index-status                   AST index status
//   POST   /projects/:pid/reindex                       trigger async re-index
//
// RBAC:
//   - GET                              : any team member (or admin)
//   - POST / PUT / DELETE / clone      : admin OR owner of the team
//
// Repo path resolution:
//   projects.path is computed as RepoRoot() + "/" + slug, where
//   RepoRoot is injected by main.go (configurable via
//   `-repo-root <dir>`). User requests never override the path.
// =============================================================================

// ProjectHandlers groups the dependencies project endpoints need.
// The cfg.RepoRoot is the canonical base directory; all v2 project
// paths are computed as cfg.RepoRoot + "/" + slug. MCPBinary is the
// codebase-memory-mcp binary used for indexing.
type ProjectHandlers struct {
	DB        *DB
	RepoRoot  string
	MCPBinary string

	// cloneMu serialises git clone invocations so we don't fork N
	// simultaneous `git` subprocesses against the same RepoRoot.
	cloneMu sync.Mutex
}

// NewProjectHandlers builds a ProjectHandlers.
func NewProjectHandlers(db *DB, repoRoot, mcpBinary string) *ProjectHandlers {
	return &ProjectHandlers{DB: db, RepoRoot: repoRoot, MCPBinary: mcpBinary}
}

// SetRepoRoot updates the on-disk location of all v2 projects. Useful
// for ops who flip staging vs production roots without restarting.
func (h *ProjectHandlers) SetRepoRoot(root string) {
	h.cloneMu.Lock()
	defer h.cloneMu.Unlock()
	h.RepoRoot = root
}

// ----------------------------------------------------------------------------
// Projects under a team
// ----------------------------------------------------------------------------

// ListProjects returns every non-deleted project in a team.
func (h *ProjectHandlers) ListProjects() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		teamID := c.Param("id")
		uid := CurrentUserID(c)
		role := CurrentRole(c)
		if role != RoleAdmin {
			isMember, _ := h.DB.IsTeamMember(ctx, teamID, uid)
			if !isMember {
				Fail(c, http.StatusNotFound, 4040020, "team not found")
				return
			}
		}
		projects, err := h.DB.ListProjectsV2(ctx, teamID)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000030, "list projects: "+err.Error())
			return
		}
		out := make([]ProjectDTOv2, 0, len(projects))
		for _, p := range projects {
			out = append(out, projectToDTOv2(p))
		}
		OK(c, gin.H{"list": out, "total": len(out)})
	}
}

// CreateProject inserts a new project under a team. Computes the path
// from cfg.RepoRoot + slug; user-supplied paths are ignored.
func (h *ProjectHandlers) CreateProject() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		teamID := c.Param("id")
		if !h.canManageTeam(c, teamID) {
			return
		}

		var req CreateProjectReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000030, "invalid request: "+err.Error())
			return
		}
		slug := SanitizeSlug(req.Slug)
		if slug == "" {
			Fail(c, http.StatusBadRequest, 4000031, "slug required")
			return
		}
		if h.RepoRoot == "" {
			Fail(c, http.StatusPreconditionFailed, 4120030, "repo_root not configured")
			return
		}
		if err := ValidateGitURL(req.GitURL); err != nil {
			Fail(c, http.StatusBadRequest, 4000032, err.Error())
			return
		}
		path, err := ComputeProjectPath(h.RepoRoot, slug)
		if err != nil {
			Fail(c, http.StatusBadRequest, 4000033, err.Error())
			return
		}
		branch := req.GitBranch
		if branch == "" {
			if req.DefaultBranch != "" {
				branch = req.DefaultBranch
			} else {
				branch = "main"
			}
		}
		status := ProjectStatusReady
		if req.GitURL != "" {
			status = ProjectStatusCloning
		}
		p := &Project{
			ID:          "proj_" + randomHex(4),
			TeamID:      teamID,
			Name:        req.Name,
			Slug:        slug,
			Description: req.Description,
			Path:        path,
			GitURL:      req.GitURL,
			GitBranch:   branch,
			Status:      status,
		}
		if err := h.DB.CreateProjectV2(ctx, p); err != nil {
			if errors.Is(err, ErrProjectExists) {
				Fail(c, http.StatusConflict, 4090030, "project slug already exists in team")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000031, "create project: "+err.Error())
			return
		}
		// Fire async clone in the background. The UI polls status.
		if req.GitURL != "" {
			go h.runClone(p.ID, p.Path, req.GitURL, branch)
		}
		OK(c, projectToDTOv2(p))
	}
}

// ----------------------------------------------------------------------------
// Project-level routes
// ----------------------------------------------------------------------------

// GetProject returns one project. Admin or team member only.
func (h *ProjectHandlers) GetProject() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("pid")
		uid := CurrentUserID(c)
		role := CurrentRole(c)
		p, err := h.DB.GetProjectV2ByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrProjectNotFound) {
				Fail(c, http.StatusNotFound, 4040021, "project not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000032, "get project: "+err.Error())
			return
		}
		if role != RoleAdmin {
			isMember, _ := h.DB.IsTeamMember(ctx, p.TeamID, uid)
			if !isMember {
				Fail(c, http.StatusNotFound, 4040021, "project not found")
				return
			}
		}
		OK(c, projectToDTOv2(p))
	}
}

// UpdateProject applies a partial update. Slug, team_id, and path
// are NOT modifiable — create a new project instead.
func (h *ProjectHandlers) UpdateProject() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("pid")
		p, err := h.DB.GetProjectV2ByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrProjectNotFound) {
				Fail(c, http.StatusNotFound, 4040022, "project not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000033, "get project: "+err.Error())
			return
		}
		if !h.canManageTeam(c, p.TeamID) {
			return
		}
		var req UpdateProjectReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000034, "invalid request: "+err.Error())
			return
		}
		if err := h.DB.UpdateProjectV2(ctx, id, req.Name, req.Description, req.GitBranch); err != nil {
			Fail(c, http.StatusInternalServerError, 5000034, "update project: "+err.Error())
			return
		}
		updated, _ := h.DB.GetProjectV2ByID(ctx, id)
		OK(c, projectToDTOv2(updated))
	}
}

// DeleteProject soft-deletes a project.
func (h *ProjectHandlers) DeleteProject() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("pid")
		p, err := h.DB.GetProjectV2ByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrProjectNotFound) {
				Fail(c, http.StatusNotFound, 4040023, "project not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000035, "get project: "+err.Error())
			return
		}
		if !h.canManageTeam(c, p.TeamID) {
			return
		}
		if err := h.DB.DeleteProjectV2(ctx, id); err != nil {
			Fail(c, http.StatusInternalServerError, 5000036, "delete project: "+err.Error())
			return
		}
		OK(c, gin.H{"id": id})
	}
}

// CloneProject re-runs the async clone pipeline. Useful when the
// initial clone failed (network blip, bad credentials) and the
// user has now fixed the underlying issue.
func (h *ProjectHandlers) CloneProject() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("pid")
		p, err := h.DB.GetProjectV2ByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrProjectNotFound) {
				Fail(c, http.StatusNotFound, 4040024, "project not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000037, "get project: "+err.Error())
			return
		}
		if !h.canManageTeam(c, p.TeamID) {
			return
		}
		if p.GitURL == "" {
			Fail(c, http.StatusPreconditionFailed, 4120031, "project has no git_url")
			return
		}
		_ = h.DB.SetProjectStatus(ctx, id, ProjectStatusCloning)
		go h.runClone(p.ID, p.Path, p.GitURL, p.GitBranch)
		OK(c, gin.H{"id": id, "status": ProjectStatusCloning})
	}
}

// IndexStatus returns whether the project's .codebase-memory index directory
// exists and basic metadata about it. Any authenticated team member can call this.
func (h *ProjectHandlers) IndexStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("pid")

		p, err := h.DB.GetProjectV2ByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrProjectNotFound) {
				Fail(c, http.StatusNotFound, 4040030, "project not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000040, "read project: "+err.Error())
			return
		}

		// Check project access
		uid := CurrentUserID(c)
		role := CurrentRole(c)
		if role != RoleAdmin {
			isMember, _ := h.DB.IsTeamMember(ctx, p.TeamID, uid)
			if !isMember {
				Fail(c, http.StatusNotFound, 4040031, "project not found")
				return
			}
		}

		idxDir := filepath.Join(p.Path, ".codebase-memory")
		info, err := os.Stat(idxDir)

		resp := gin.H{
			"project_id": id,
			"indexed":    false,
			"index_dir":  idxDir,
			"status":     p.Status,
		}

		if err == nil && info.IsDir() {
			resp["indexed"] = true
			resp["last_update"] = info.ModTime().UTC().Format(time.RFC3339)
		} else if err != nil && !os.IsNotExist(err) {
			resp["error"] = err.Error()
		}

		OK(c, resp)
	}
}

// Reindex triggers an async re-index of the project's code via
// `codebase-memory-mcp index_repository`. The indexing runs in the
// background; the endpoint returns immediately with the task state.
// Only admin or team lead can trigger reindex.
func (h *ProjectHandlers) Reindex() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("pid")

		p, err := h.DB.GetProjectV2ByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrProjectNotFound) {
				Fail(c, http.StatusNotFound, 4040032, "project not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000041, "read project: "+err.Error())
			return
		}

		if !h.canManageTeam(c, p.TeamID) {
			return
		}

		if h.MCPBinary == "" {
			Fail(c, http.StatusPreconditionFailed, 4120040, "mcp_bin not configured")
			return
		}

		// Set status to indexing
		if err := h.DB.SetProjectStatus(ctx, id, ProjectStatusIndexing); err != nil {
			Fail(c, http.StatusInternalServerError, 5000042, "set status: "+err.Error())
			return
		}

		// Fire async indexing
		go h.runReindex(id, p.Path)

		OK(c, gin.H{
			"project_id": id,
			"status":     ProjectStatusIndexing,
			"message":    "indexing started in background",
		})
	}
}

// runReindex executes `codebase-memory-mcp index_repository <path>` in a
// detached context. On success sets status to "ready"; on failure sets "error".
func (h *ProjectHandlers) runReindex(projectID, path string) {
	if h.MCPBinary == "" {
		_ = h.DB.SetProjectStatus(context.Background(), projectID, ProjectStatusError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, h.MCPBinary, "index_repository", path)
	_, err := cmd.CombinedOutput()

	if err != nil {
		_ = h.DB.SetProjectStatus(ctx, projectID, ProjectStatusError)
		return
	}

	// Record the output SHA if we can read it
	if sha, shaErr := gitHeadSHA(path); shaErr == nil {
		_ = h.DB.SetProjectGitCommit(ctx, projectID, sha)
	}

	_ = h.DB.SetProjectStatus(ctx, projectID, ProjectStatusReady)
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// canManageTeam mirrors TeamHandlers.canManageTeam. Duplicated here
// because the two handlers' dependencies differ (this one holds
// RepoRoot) and we don't want a base-class glue layer just for one
// shared private method.
func (h *ProjectHandlers) canManageTeam(c *gin.Context, teamID string) bool {
	ctx := c.Request.Context()
	uid := CurrentUserID(c)
	if CurrentRole(c) == RoleAdmin {
		return true
	}
	t, err := h.DB.GetTeamByID(ctx, teamID)
	if err != nil {
		Fail(c, http.StatusNotFound, 4040025, "team not found")
		return false
	}
	if t.OwnerID != uid {
		Fail(c, http.StatusForbidden, 4030030, "only admin or team owner can manage this team")
		return false
	}
	return true
}

// runClone executes `git clone --branch <branch> <url> <path>` in a
// detached context. Status transitions: cloning → ready | error.
// SHA is recorded on success. Failures bump status to "error" so the
// UI can surface a banner; operators re-trigger via CloneProject.
//
// We hold h.cloneMu so two clones can't trample each other's
// filesystem state under the same RepoRoot.
func (h *ProjectHandlers) runClone(projectID, path, url, branch string) {
	h.cloneMu.Lock()
	defer h.cloneMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, path)

	cmd := exec.CommandContext(ctx, "git", args...)
	if _, err := cmd.CombinedOutput(); err != nil {
		_ = h.DB.SetProjectStatus(ctx, projectID, ProjectStatusError)
		return
	}
	if sha, shaErr := gitHeadSHA(path); shaErr == nil {
		_ = h.DB.SetProjectGitCommit(ctx, projectID, sha)
	}
	_ = h.DB.SetProjectStatus(ctx, projectID, ProjectStatusReady)
}

// gitHeadSHA returns HEAD's commit SHA for the on-disk repo. Empty
// string on any failure (caller treats that as non-fatal).
func gitHeadSHA(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s := string(out)
	if len(s) > 40 {
		s = s[:40]
	}
	return s, nil
}

// projectToDTOv2 converts the storage struct to the wire DTO.
func projectToDTOv2(p *Project) ProjectDTOv2 {
	rel, _ := filepath.Rel(filepath.Dir(p.Path), p.Path)
	return ProjectDTOv2{
		ID:           p.ID,
		TeamID:       p.TeamID,
		Name:         p.Name,
		Slug:         p.Slug,
		Description:  p.Description,
		Path:         p.Path,
		PathRel:      rel,
		GitURL:       p.GitURL,
		GitBranch:    p.GitBranch,
		GitCommitSHA: p.GitCommitSHA,
		Status:       p.Status,
		CreatedAt:    p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Deleted:      p.Deleted,
	}
}

// fmtErr is unused; left in case callers need a tiny inline helper.
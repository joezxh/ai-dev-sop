package console

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// v2 (M5) Memory HTTP handlers.
// =============================================================================
//
//   GET    /api/v2/memories                       list + filter
//   POST   /api/v2/memories                       create (leaf module required)
//   GET    /api/v2/memories/:id                   detail
//   PUT    /api/v2/memories/:id                   partial update
//   DELETE /api/v2/memories/:id                   soft delete
//   POST   /api/v2/memories/:id/tags              append a single tag
//   GET    /api/v2/modules/:id/memories           list under a specific leaf
//
// RBAC: read access requires admin OR membership in the memory's
// team. Writes additionally require admin/lead/owner (developer is
// read-only on memories). The leaf module guard is enforced in the
// storage layer so a caller can't sneak a non-leaf id through.
// =============================================================================

// MemoryHandlers groups dependencies for memory endpoints.
type MemoryHandlers struct {
	DB *DB
}

// NewMemoryHandlers builds a MemoryHandlers.
func NewMemoryHandlers(db *DB) *MemoryHandlers { return &MemoryHandlers{DB: db} }

// List returns memories matching the query filters. Pagination is
// limit/offset.
func (h *MemoryHandlers) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		f := NewMemoryFilter()
		f.TeamID = c.Query("team_id")
		f.ProjectID = c.Query("project_id")
		f.ModuleID = c.Query("module_id")
		f.UserID = c.Query("user_id")
		f.Hall = c.Query("hall")
		f.TemplateID = c.Query("template_id")
		f.Tag = c.Query("tag")
		f.Limit, _ = strconv.Atoi(c.Query("limit"))
		f.Offset, _ = strconv.Atoi(c.Query("offset"))
		rows, total, err := h.DB.ListMemories(ctx, f)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000100, "list memories: "+err.Error())
			return
		}
		out := make([]MemoryDTO, 0, len(rows))
		for _, m := range rows {
			out = append(out, memoryToDTO(m))
		}
		OK(c, PageResp{List: out, Total: total, PageNo: 1, PageSize: f.Limit})
	}
}

// ModuleMemories returns memories of a single module. ModuleID is
// the path parameter so the route matches the modules tree layout
// (the M4 /modules/:id/ subtree).
func (h *MemoryHandlers) ModuleMemories() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		moduleID := c.Param("id")
		m, err := h.DB.GetModuleByID(ctx, moduleID)
		if err != nil {
			if errors.Is(err, ErrModuleNotFound) {
				Fail(c, http.StatusNotFound, 4040090, "module not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000101, "read module: "+err.Error())
			return
		}
		// Reuse the modules read-access guard so we don't have two
		// slightly-different RBAC implementations for "I can see
		// this project's modules" vs "I can see this project's
		// memories".
		mods := NewModuleHandlers(h.DB)
		if !mods.assertProjectAccess(c, m.ProjectID) {
			return
		}
		f := NewMemoryFilter()
		f.ModuleID = moduleID
		f.Limit, _ = strconv.Atoi(c.Query("limit"))
		if f.Limit == 0 {
			f.Limit = 50
		}
		rows, total, err := h.DB.ListMemories(ctx, f)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000102, "list: "+err.Error())
			return
		}
		out := make([]MemoryDTO, 0, len(rows))
		for _, m := range rows {
			out = append(out, memoryToDTO(m))
		}
		OK(c, PageResp{List: out, Total: total, PageNo: 1, PageSize: f.Limit})
	}
}

// Create inserts a new memory. The module_id MUST reference a leaf;
// that's enforced both here (storage) and via RequireLeafModule so
// the operator gets a clear 4xx.
func (h *MemoryHandlers) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		uid := CurrentUserID(c)
		role := CurrentRole(c)
		if uid == "" || role == "" {
			Fail(c, http.StatusUnauthorized, 4010100, "no auth context")
			return
		}
		var req CreateMemoryReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000090, "invalid request: "+err.Error())
			return
		}
		if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
			Fail(c, http.StatusBadRequest, 4000091, "title and content required")
			return
		}
		// The leaf guard — module_id must be a real, non-deleted
		// leaf. RequireLeafModule also handles the missing-module
		// 404.
		if err := h.DB.RequireLeafModule(ctx, req.ModuleID); err != nil {
			switch {
			case errors.Is(err, ErrModuleNotFound):
				Fail(c, http.StatusNotFound, 4040091, "module not found")
			case errors.Is(err, ErrModuleNotLeaf):
				Fail(c, http.StatusBadRequest, 4000092, "module is not a leaf; pick a leaf descendant")
			default:
				Fail(c, http.StatusInternalServerError, 5000103, "validate module: "+err.Error())
			}
			return
		}
		// Look up team_id from the project — the schema requires it
		// but the API surface lets the caller supply just a module
		// id, so we derive the rest.
		var teamID string
		if err := h.DB.QueryRowContext(ctx,
			`SELECT p.team_id FROM modules m JOIN projects p ON p.id = m.project_id
			   WHERE m.id = ? AND m.deleted = 0 AND p.deleted = 0`,
			req.ModuleID,
		).Scan(&teamID); err != nil {
			Fail(c, http.StatusInternalServerError, 5000104, "resolve team: "+err.Error())
			return
		}
		if !h.assertMemoryWrite(c, teamID) {
			return
		}
		m := &Memory{
			ID:         "mem_" + randomHex(4),
			TeamID:     teamID,
			ProjectID:  deriveProjectIDFromModule(ctx, h.DB, req.ModuleID),
			ModuleID:   req.ModuleID,
			UserID:     uid,
			Title:      req.Title,
			Content:    req.Content,
			TemplateID: req.TemplateID,
			Tags:       req.Tags,
			Hall:       req.Hall,
		}
		if err := h.DB.CreateMemory(ctx, m); err != nil {
			if errors.Is(err, ErrInvalidHall) {
				Fail(c, http.StatusBadRequest, 4000093, err.Error())
				return
			}
			Fail(c, http.StatusInternalServerError, 5000105, "create memory: "+err.Error())
			return
		}
		OK(c, memoryToDTO(m))
	}
}

// Get returns one memory by id.
func (h *MemoryHandlers) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		m, err := h.DB.GetMemory(ctx, id)
		if err != nil {
			if errors.Is(err, ErrMemoryNotFound) {
				Fail(c, http.StatusNotFound, 4040092, "memory not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000106, "read memory: "+err.Error())
			return
		}
		uid := CurrentUserID(c)
		role := CurrentRole(c)
		ok, err := h.DB.MemoryAccessibleByUser(ctx, id, uid, string(role))
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000107, "access check: "+err.Error())
			return
		}
		if !ok {
			Fail(c, http.StatusNotFound, 4040093, "memory not found")
			return
		}
		OK(c, memoryToDTO(m))
	}
}

// Update applies a partial update.
func (h *MemoryHandlers) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		m, err := h.DB.GetMemory(ctx, id)
		if err != nil {
			if errors.Is(err, ErrMemoryNotFound) {
				Fail(c, http.StatusNotFound, 4040094, "memory not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000108, "read memory: "+err.Error())
			return
		}
		if !h.assertMemoryWrite(c, m.TeamID) {
			return
		}
		var req UpdateMemoryReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000094, "invalid request: "+err.Error())
			return
		}
		if err := h.DB.UpdateMemory(ctx, id, req.Title, req.Content, req.Hall, req.Tags); err != nil {
			switch {
			case errors.Is(err, ErrMemoryNotFound):
				Fail(c, http.StatusNotFound, 4040095, "memory not found")
			case errors.Is(err, ErrInvalidHall):
				Fail(c, http.StatusBadRequest, 4000095, err.Error())
			default:
				Fail(c, http.StatusInternalServerError, 5000109, "update memory: "+err.Error())
			}
			return
		}
		updated, _ := h.DB.GetMemory(ctx, id)
		OK(c, memoryToDTO(updated))
	}
}

// Delete soft-deletes a memory.
func (h *MemoryHandlers) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		m, err := h.DB.GetMemory(ctx, id)
		if err != nil {
			if errors.Is(err, ErrMemoryNotFound) {
				Fail(c, http.StatusNotFound, 4040096, "memory not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000110, "read memory: "+err.Error())
			return
		}
		if !h.assertMemoryWrite(c, m.TeamID) {
			return
		}
		if err := h.DB.SoftDeleteMemory(ctx, id); err != nil {
			if errors.Is(err, ErrMemoryNotFound) {
				Fail(c, http.StatusNotFound, 4040097, "memory not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000111, "delete memory: "+err.Error())
			return
		}
		OK(c, gin.H{"id": id})
	}
}

// AddTag appends a single tag to the memory's tag list. Duplicates
// are ignored so the operator can click the same tag twice without
// polluting the list.
func (h *MemoryHandlers) AddTag() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		m, err := h.DB.GetMemory(ctx, id)
		if err != nil {
			if errors.Is(err, ErrMemoryNotFound) {
				Fail(c, http.StatusNotFound, 4040098, "memory not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000112, "read memory: "+err.Error())
			return
		}
		if !h.assertMemoryWrite(c, m.TeamID) {
			return
		}
		var req AddTagReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000096, "invalid request: "+err.Error())
			return
		}
		if strings.TrimSpace(req.Tag) == "" {
			Fail(c, http.StatusBadRequest, 4000097, "tag required")
			return
		}
		newTags := append(m.Tags, req.Tag)
		seen := map[string]bool{}
		dedup := make([]string, 0, len(newTags))
		for _, t := range newTags {
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			dedup = append(dedup, t)
		}
		if err := h.DB.UpdateMemory(ctx, id, nil, nil, nil, dedup); err != nil {
			Fail(c, http.StatusInternalServerError, 5000113, "update tags: "+err.Error())
			return
		}
		updated, _ := h.DB.GetMemory(ctx, id)
		OK(c, memoryToDTO(updated))
	}
}

// ----------------------------------------------------------------------------
// RBAC helpers
// ----------------------------------------------------------------------------

// assertMemoryWrite is the write-side guard for memory endpoints.
// Admin/lead/team-owner/maintainer can write; developer is denied.
// Writes 4xx and returns false on rejection.
func (h *MemoryHandlers) assertMemoryWrite(c *gin.Context, teamID string) bool {
	ctx := c.Request.Context()
	role := CurrentRole(c)
	uid := CurrentUserID(c)
	if role == "" {
		Fail(c, http.StatusUnauthorized, 4010101, "no auth context")
		return false
	}
	if role == RoleAdmin || role == RoleLead {
		return true
	}
	if isMember, _ := h.DB.IsTeamMember(ctx, teamID, uid); !isMember {
		Fail(c, http.StatusForbidden, 4030090, "insufficient privileges on team")
		return false
	}
	// Read membership role from the join row to decide. We keep the
	// check simple: only owner/maintainer can write. Note that
	// tm.Role is a plain string from the schema; ProjectRole is a
	// typed alias of string so we cast for the comparison.
	tm, _ := h.DB.GetTeamMember(ctx, teamID, uid)
	teamRole := ProjectRole(tm.Role)
	if teamRole == ProjectRoleOwner || teamRole == ProjectRoleMaintainer {
		return true
	}
	Fail(c, http.StatusForbidden, 4030091, "developer role is read-only on memories")
	return false
}

// deriveProjectIDFromModule fetches the project_id for a module
// without leaking the implementation detail into the caller. Used
// during Create to populate the memory's required project_id from
// just the module_id.
func deriveProjectIDFromModule(ctx context.Context, db *DB, moduleID string) string {
	var pid string
	_ = db.QueryRowContext(ctx,
		`SELECT project_id FROM modules WHERE id = ? AND deleted = 0`, moduleID,
	).Scan(&pid)
	return pid
}

// memoryToDTO converts the storage row to the wire shape.
func memoryToDTO(m *Memory) MemoryDTO {
	if m == nil {
		return MemoryDTO{}
	}
	return MemoryDTO{
		ID:         m.ID,
		TeamID:     m.TeamID,
		ProjectID:  m.ProjectID,
		ModuleID:   m.ModuleID,
		UserID:     m.UserID,
		Title:      m.Title,
		Content:    m.Content,
		TemplateID: m.TemplateID,
		Tags:       m.Tags,
		Hall:       m.Hall,
		CreatedAt:  m.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  m.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
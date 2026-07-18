package console

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// v2 (M4) Module HTTP handlers.
// =============================================================================
//
// Routes (mounted under /api/v2 by the router; all JWT-required):
//
//   GET    /projects/:pid/modules             flat list under project
//   POST   /projects/:pid/modules             create (body: parent_id, name, ...)
//   GET    /projects/:pid/modules/tree        nested tree view
//   GET    /modules/:id                       detail
//   PUT    /modules/:id                       partial update
//   DELETE /modules/:id?cascade=true|false   delete (leaf-only by default)
//   POST   /modules/:id/move                  re-parent
//
// RBAC:
//   - GET     : any member of the project's team (or admin)
//   - POST    : admin OR team lead/owner (project-level writes)
//   - PUT     : admin OR team lead/owner
//   - DELETE  : admin OR team lead/owner
//   - MOVE    : admin OR team lead/owner
//
// =============================================================================

// ModuleHandlers groups dependencies for module endpoints.
type ModuleHandlers struct {
	DB *DB
}

// NewModuleHandlers builds a ModuleHandlers.
func NewModuleHandlers(db *DB) *ModuleHandlers { return &ModuleHandlers{DB: db} }

// ----------------------------------------------------------------------------
// Project-scoped
// ----------------------------------------------------------------------------

// ListModules returns the flat list of modules under a project.
// The UI usually calls /tree instead; this endpoint powers flat
// autocompletes and bulk migrations.
func (h *ModuleHandlers) ListModules() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		projectID := c.Param("pid")
		if !h.assertProjectAccess(c, projectID) {
			return
		}
		modules, err := h.DB.ListModulesByProject(ctx, projectID)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000050, "list modules: "+err.Error())
			return
		}
		out := make([]ModuleDTO, 0, len(modules))
		for _, m := range modules {
			out = append(out, moduleToDTO(m, h.childCountsFor(ctx, m)))
		}
		OK(c, gin.H{"list": out, "total": len(out)})
	}
}

// CreateModule inserts a new module under a project. ParentID may be
// nil for a root module; otherwise the parent must belong to the
// same project.
func (h *ModuleHandlers) CreateModule() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		projectID := c.Param("pid")
		if !h.assertProjectWrite(c, projectID) {
			return
		}
		var req CreateModuleReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000050, "invalid request: "+err.Error())
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			Fail(c, http.StatusBadRequest, 4000051, "name required")
			return
		}
		if req.ParentID != nil {
			parent, err := h.DB.GetModuleByID(ctx, *req.ParentID)
			if err != nil {
				if errors.Is(err, ErrModuleNotFound) {
					Fail(c, http.StatusNotFound, 4040050, "parent module not found")
					return
				}
				Fail(c, http.StatusInternalServerError, 5000051, "read parent: "+err.Error())
				return
			}
			if parent.ProjectID != projectID {
				Fail(c, http.StatusBadRequest, 4000052, "parent belongs to a different project")
				return
			}
		}
		m := &Module{
			ID:          "mod_" + randomHex(4),
			ProjectID:   projectID,
			ParentID:    req.ParentID,
			Name:        req.Name,
			Path:        req.Path,
			Description: req.Description,
			Order:       req.Order,
		}
		if err := h.DB.CreateModule(ctx, m); err != nil {
			if errors.Is(err, ErrModuleExists) {
				Fail(c, http.StatusConflict, 4090050, "module name already exists under this parent")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000052, "create module: "+err.Error())
			return
		}
		OK(c, moduleToDTO(m, &moduleCounters{}))
	}
}

// Tree returns the nested module tree for a project. The shape is
// ModuleNodeDTO, with Children either populated (internal nodes) or
// nil/empty (leaves).
func (h *ModuleHandlers) Tree() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		projectID := c.Param("pid")
		if !h.assertProjectAccess(c, projectID) {
			return
		}
		flat, err := h.DB.ListModulesByProject(ctx, projectID)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000053, "tree: "+err.Error())
			return
		}
		nodes := BuildModuleTree(flat)
		// Enrich each node with child_count + session_count via one
		// batched query so we don't N+1.
		enrich(ctx, h.DB, nodes)
		// Wrap in a ModuleNodeDTO slice.
		out := make([]*ModuleNodeDTO, 0, len(nodes))
		for _, n := range nodes {
			out = append(out, toNodeDTO(n))
		}
		OK(c, gin.H{"list": out, "total": len(out)})
	}
}

// ----------------------------------------------------------------------------
// Module-scoped
// ----------------------------------------------------------------------------

// GetModule returns one module by id.
func (h *ModuleHandlers) GetModule() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		m, err := h.DB.GetModuleByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrModuleNotFound) {
				Fail(c, http.StatusNotFound, 4040051, "module not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000054, "get module: "+err.Error())
			return
		}
		if !h.assertProjectAccess(c, m.ProjectID) {
			return
		}
		OK(c, moduleToDTO(m, h.childCountsFor(ctx, m)))
	}
}

// UpdateModule applies a partial update.
func (h *ModuleHandlers) UpdateModule() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		m, err := h.DB.GetModuleByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrModuleNotFound) {
				Fail(c, http.StatusNotFound, 4040052, "module not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000055, "get module: "+err.Error())
			return
		}
		if !h.assertProjectWrite(c, m.ProjectID) {
			return
		}
		var req UpdateModuleReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000053, "invalid request: "+err.Error())
			return
		}
		if err := h.DB.UpdateModule(ctx, id, req.Name, req.Path, req.Description, req.Order); err != nil {
			switch {
			case errors.Is(err, ErrModuleNotFound):
				Fail(c, http.StatusNotFound, 4040053, "module not found")
			case errors.Is(err, ErrModuleExists):
				Fail(c, http.StatusConflict, 4090051, "module name already exists under this parent")
			default:
				Fail(c, http.StatusInternalServerError, 5000056, "update module: "+err.Error())
			}
			return
		}
		updated, _ := h.DB.GetModuleByID(ctx, id)
		OK(c, moduleToDTO(updated, h.childCountsFor(ctx, updated)))
	}
}

// DeleteModule soft-deletes a module. Default behavior refuses nodes
// with children; the operator passes ?cascade=true to delete the
// entire subtree.
func (h *ModuleHandlers) DeleteModule() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		m, err := h.DB.GetModuleByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrModuleNotFound) {
				Fail(c, http.StatusNotFound, 4040054, "module not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000057, "get module: "+err.Error())
			return
		}
		if !h.assertProjectWrite(c, m.ProjectID) {
			return
		}
		cascade := c.Query("cascade") == "true"
		if cascade {
			// Pre-compute the impact so the API can hint the operator.
			n, _ := h.DB.CountDescendants(ctx, id)
			if n > 0 {
				c.Header("X-Cascade-Count", itoa(n))
			}
		}
		if err := h.DB.DeleteModule(ctx, id, cascade); err != nil {
			switch {
			case errors.Is(err, ErrModuleNotFound):
				Fail(c, http.StatusNotFound, 4040055, "module not found")
			case errors.Is(err, ErrModuleHasChildren):
				Fail(c, http.StatusConflict, 4090052, "module has children; pass cascade=true to delete the subtree")
			case errors.Is(err, ErrModuleInUse):
				Fail(c, http.StatusConflict, 4090053, "module still referenced by sessions/memories")
			default:
				Fail(c, http.StatusInternalServerError, 5000058, "delete module: "+err.Error())
			}
			return
		}
		OK(c, gin.H{"id": id, "cascade": cascade})
	}
}

// MoveModule re-parents a module. Cycle detection is enforced in the
// DB layer; the handler only translates errors.
func (h *ModuleHandlers) MoveModule() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		m, err := h.DB.GetModuleByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrModuleNotFound) {
				Fail(c, http.StatusNotFound, 4040056, "module not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000059, "get module: "+err.Error())
			return
		}
		if !h.assertProjectWrite(c, m.ProjectID) {
			return
		}
		var req ModuleTreeMoveRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000054, "invalid request: "+err.Error())
			return
		}
		if req.NewParentID != nil {
			parent, err := h.DB.GetModuleByID(ctx, *req.NewParentID)
			if err != nil {
				if errors.Is(err, ErrModuleNotFound) {
					Fail(c, http.StatusNotFound, 4040057, "new parent not found")
					return
				}
				Fail(c, http.StatusInternalServerError, 5000060, "read parent: "+err.Error())
				return
			}
			if parent.ProjectID != m.ProjectID {
				Fail(c, http.StatusBadRequest, 4000055, "new parent belongs to a different project")
				return
			}
		}
		if err := h.DB.MoveModule(ctx, id, req.NewParentID); err != nil {
			switch {
			case errors.Is(err, ErrModuleNotFound):
				Fail(c, http.StatusNotFound, 4040058, "module not found")
			case errors.Is(err, ErrModuleCycle):
				Fail(c, http.StatusBadRequest, 4000056, "move would create a cycle")
			default:
				Fail(c, http.StatusInternalServerError, 5000061, "move module: "+err.Error())
			}
			return
		}
		updated, _ := h.DB.GetModuleByID(ctx, id)
		OK(c, moduleToDTO(updated, h.childCountsFor(ctx, updated)))
	}
}

// ----------------------------------------------------------------------------
// Access checks
// ----------------------------------------------------------------------------

// assertProjectAccess is the read-side guard: admin OR team member.
// On failure, writes the 4xx and returns false.
func (h *ModuleHandlers) assertProjectAccess(c *gin.Context, projectID string) bool {
	ctx := c.Request.Context()
	uid := CurrentUserID(c)
	role := CurrentRole(c)
	if role == "" {
		Fail(c, http.StatusUnauthorized, 4010050, "no auth context")
		return false
	}
	if role == RoleAdmin {
		return true
	}
	// Look up the project's team and check membership. We avoid
	// pulling ProjectHandlers in here to keep ModuleHandlers
	// self-contained.
	var teamID string
	if err := h.DB.QueryRowContext(ctx,
		`SELECT team_id FROM pm_projects WHERE id = ? AND deleted = 0`, projectID,
	).Scan(&teamID); err != nil {
		Fail(c, http.StatusNotFound, 4040059, "project not found")
		return false
	}
	isMember, _ := h.DB.IsTeamMember(ctx, teamID, uid)
	if !isMember {
		// Don't leak project existence.
		Fail(c, http.StatusNotFound, 4040060, "project not found")
		return false
	}
	return true
}

// assertProjectWrite is the write-side guard. v2 plan: admin OR
// team owner (the team_legacy owner + named owner; lead role also
// bumps to write access for parity with M3's "lead can create teams"
// rule).
func (h *ModuleHandlers) assertProjectWrite(c *gin.Context, projectID string) bool {
	ctx := c.Request.Context()
	uid := CurrentUserID(c)
	role := CurrentRole(c)
	if role == "" {
		Fail(c, http.StatusUnauthorized, 4010051, "no auth context")
		return false
	}
	if role == RoleAdmin || role == RoleLead {
		return true
	}
	var teamID, ownerID string
	if err := h.DB.QueryRowContext(ctx,
		`SELECT team_id, IFNULL(owner_id,'') FROM pm_projects WHERE id = ? AND deleted = 0`, projectID,
	).Scan(&teamID, &ownerID); err != nil {
		Fail(c, http.StatusNotFound, 4040061, "project not found")
		return false
	}
	// Team owner can also write; otherwise 403.
	if ownerID != "" && ownerID == uid {
		return true
	}
	// Maintainer role within the team also gets write.
	if isMember, _ := h.DB.IsTeamMember(ctx, teamID, uid); !isMember {
		Fail(c, http.StatusForbidden, 4030050, "insufficient privileges on project")
		return false
	}
	// Members of the team can view; we already allowed reads above.
	Fail(c, http.StatusForbidden, 4030051, "developer role is read-only on modules")
	return false
}

// ----------------------------------------------------------------------------
// DTO + enrichment helpers
// ----------------------------------------------------------------------------

type moduleCounters struct {
	children   int
	sessions   int
	memories   int
}

// childCountsFor fetches child/session/memory counts in one call
// (instead of three). Caller passes the module so we read its id.
func (h *ModuleHandlers) childCountsFor(ctx context.Context, m *Module) *moduleCounters {
	if m == nil {
		return &moduleCounters{}
	}
	c, _ := h.DB.CountChildren(ctx, m.ID)
	s, _ := h.DB.CountSessionsByModule(ctx, m.ID)
	mem, _ := h.DB.CountMemoriesByModule(ctx, m.ID)
	return &moduleCounters{children: c, sessions: s, memories: mem}
}

// moduleToDTO converts a stored Module + counters into ModuleDTO.
func moduleToDTO(m *Module, c *moduleCounters) ModuleDTO {
	if c == nil {
		c = &moduleCounters{}
	}
	return ModuleDTO{
		ID:           m.ID,
		ProjectID:    m.ProjectID,
		ParentID:     m.ParentID,
		Name:         m.Name,
		Path:         m.Path,
		Description:  m.Description,
		Order:        m.Order,
		IsLeaf:       m.IsLeaf,
		ChildCount:   c.children,
		SessionCount: c.sessions,
		MemoryCount:  c.memories,
		CreatedAt:    m.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    m.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// enrich walks the tree once, collecting ids, and runs one batched
// query per counter kind. Hits the wire ~3 queries total for the
// whole tree instead of N×3.
func enrich(ctx context.Context, db *DB, nodes []*ModuleNode) {
	if len(nodes) == 0 {
		return
	}
	var collect func(out *[]string, ns []*ModuleNode)
	collect = func(out *[]string, ns []*ModuleNode) {
		for _, n := range ns {
			*out = append(*out, n.ID)
			collect(out, n.Children)
		}
	}
	var apply func(ns []*ModuleNode)
	var ids []string
	collect(&ids, nodes)

	childCount := map[string]int{}
	sessCount := map[string]int{}
	memCount := map[string]int{}
	for _, id := range ids {
		if n, err := db.CountChildren(ctx, id); err == nil {
			childCount[id] = n
		}
		if n, err := db.CountSessionsByModule(ctx, id); err == nil {
			sessCount[id] = n
		}
		if n, err := db.CountMemoriesByModule(ctx, id); err == nil {
			memCount[id] = n
		}
	}
	apply = func(ns []*ModuleNode) {
		for _, n := range ns {
			n.ChildCount = childCount[n.ID]
			n.SessionCount = sessCount[n.ID]
			n.MemoryCount = memCount[n.ID]
			apply(n.Children)
		}
	}
	apply(nodes)
}

// toNodeDTO converts a Model ModuleNode tree into the DTO tree.
func toNodeDTO(n *ModuleNode) *ModuleNodeDTO {
	dto := &ModuleNodeDTO{
		ModuleDTO: moduleToDTO(&n.Module, &moduleCounters{
			children: n.ChildCount,
			sessions: n.SessionCount,
			memories: n.MemoryCount,
		}),
	}
	if len(n.Children) > 0 {
		dto.Children = make([]*ModuleNodeDTO, 0, len(n.Children))
		for _, c := range n.Children {
			dto.Children = append(dto.Children, toNodeDTO(c))
		}
	}
	return dto
}

// ----------------------------------------------------------------------------
// Middleware: RequireLeafModule
// ----------------------------------------------------------------------------
//
// RequireLeafModule is declared on *DB inside db_modules.go so the
// helper lives next to the storage layer it validates. The handler
// file imports it via the method expression `db.RequireLeafModule`.

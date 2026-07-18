package console

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// v2 (M3) Team + TeamMember HTTP handlers
// =============================================================================
//
// Routes (mounted under /api/v2 in router.go):
//
//   GET    /teams                          list (member_count, project_count populated)
//   POST   /teams                          create (caller becomes owner of new team)
//   GET    /teams/:id                      detail
//   PUT    /teams/:id                      partial update (name, description)
//   DELETE /teams/:id                      soft-delete
//
//   GET    /teams/:id/members              list members
//   POST   /teams/:id/members              add member (body: {user_id, role})
//   PUT    /teams/:id/members/:uid         change role
//   DELETE /teams/:id/members/:uid         remove member
//
// All endpoints require JWT. RBAC:
//   - GET list/detail/members   : any authenticated user
//   - POST/PUT/DELETE team      : admin OR owner of the team
//   - POST/PUT/DELETE member    : admin OR owner of the team
//   - A non-owner, non-admin developer can only see teams they're a
//     member of (filter applied at the handler)
//
// =============================================================================

// TeamHandlers groups the dependencies team endpoints need.
type TeamHandlers struct {
	DB *DB
}

// NewTeamHandlers builds a TeamHandlers.
func NewTeamHandlers(db *DB) *TeamHandlers { return &TeamHandlers{DB: db} }

// ----------------------------------------------------------------------------
// Teams
// ----------------------------------------------------------------------------

// ListTeams returns every team the caller is allowed to see. Admins
// see all; non-admins see only teams they belong to.
func (h *TeamHandlers) ListTeams() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		uid := CurrentUserID(c)
		role := CurrentRole(c)

		var (
			teams []*Team
			err   error
		)
		if role == RoleAdmin {
			teams, err = h.DB.ListTeams(ctx)
		} else {
			teams, err = h.DB.ListTeamsForUser(ctx, uid)
		}
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000020, "list teams: "+err.Error())
			return
		}
		out := make([]TeamDTO, 0, len(teams))
		for _, t := range teams {
			dto := teamToDTO(t)
			if n, err := h.DB.CountMembersInTeam(ctx, t.ID); err == nil {
				dto.MemberCount = n
			}
			if n, err := h.DB.CountProjectsInTeam(ctx, t.ID); err == nil {
				dto.ProjectCount = n
			}
			out = append(out, dto)
		}
		OK(c, gin.H{"list": out, "total": len(out)})
	}
}

// GetTeam returns one team by id. Non-admin, non-member callers get 404
// rather than 403 to avoid leaking team existence.
func (h *TeamHandlers) GetTeam() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		uid := CurrentUserID(c)
		role := CurrentRole(c)

		t, err := h.DB.GetTeamByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrTeamNotFound) {
				Fail(c, http.StatusNotFound, 4040010, "team not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000021, "get team: "+err.Error())
			return
		}
		if role != RoleAdmin {
			isMember, _ := h.DB.IsTeamMember(ctx, id, uid)
			if !isMember {
				Fail(c, http.StatusNotFound, 4040010, "team not found")
				return
			}
		}
		dto := teamToDTO(t)
		if n, err := h.DB.CountMembersInTeam(ctx, id); err == nil {
			dto.MemberCount = n
		}
		if n, err := h.DB.CountProjectsInTeam(ctx, id); err == nil {
			dto.ProjectCount = n
		}
		OK(c, dto)
	}
}

// CreateTeam inserts a new team. Caller becomes a member with
// ProjectRoleOwner. Slug must be unique.
func (h *TeamHandlers) CreateTeam() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		uid := CurrentUserID(c)
		role := CurrentRole(c)

		// Only admins and leads can create teams (leads have to upgrade
		// by being invited; this keeps the namespace curated).
		if role != RoleAdmin && role != RoleLead {
			Fail(c, http.StatusForbidden, 4030020, "only admin/lead can create teams")
			return
		}

		var req CreateTeamReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000020, "invalid request: "+err.Error())
			return
		}
		t := &Team{
			ID:          "team_" + randomHex(4),
			Name:        req.Name,
			Slug:        SanitizeSlug(req.Slug),
			Description: req.Description,
			OwnerID:     req.OwnerID,
		}
		if t.Slug == "" {
			Fail(c, http.StatusBadRequest, 4000021, "slug required")
			return
		}
		if t.OwnerID == "" {
			t.OwnerID = uid
		}
		if err := h.DB.CreateTeam(ctx, t); err != nil {
			if errors.Is(err, ErrTeamExists) {
				Fail(c, http.StatusConflict, 4090010, "team slug already exists")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000022, "create team: "+err.Error())
			return
		}
		// Caller becomes the owner-member.
		_ = h.DB.AddTeamMember(ctx, t.ID, uid, ProjectRoleOwner)
		if t.OwnerID != uid {
			_ = h.DB.AddTeamMember(ctx, t.ID, t.OwnerID, ProjectRoleMaintainer)
		}
		OK(c, teamToDTO(t))
	}
}

// UpdateTeam applies a partial update to name / description. Owner or
// admin only.
func (h *TeamHandlers) UpdateTeam() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		if !h.canManageTeam(c, id) {
			return // response already written
		}
		var req UpdateTeamReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000022, "invalid request: "+err.Error())
			return
		}
		if err := h.DB.UpdateTeam(ctx, id, req.Name, req.Description); err != nil {
			if errors.Is(err, ErrTeamNotFound) {
				Fail(c, http.StatusNotFound, 4040011, "team not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000023, "update team: "+err.Error())
			return
		}
		t, _ := h.DB.GetTeamByID(ctx, id)
		OK(c, teamToDTO(t))
	}
}

// DeleteTeam soft-deletes the team. Owner or admin only.
func (h *TeamHandlers) DeleteTeam() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		if !h.canManageTeam(c, id) {
			return
		}
		if err := h.DB.DeleteTeam(ctx, id); err != nil {
			if errors.Is(err, ErrTeamNotFound) {
				Fail(c, http.StatusNotFound, 4040012, "team not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000024, "delete team: "+err.Error())
			return
		}
		OK(c, gin.H{"id": id})
	}
}

// ----------------------------------------------------------------------------
// Members
// ----------------------------------------------------------------------------

// ListMembers returns every member of a team, with their user_id,
// username (joined from users), and role. Any member can list.
func (h *TeamHandlers) ListMembers() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		uid := CurrentUserID(c)
		role := CurrentRole(c)
		if role != RoleAdmin {
			isMember, _ := h.DB.IsTeamMember(ctx, id, uid)
			if !isMember {
				Fail(c, http.StatusNotFound, 4040013, "team not found")
				return
			}
		}
		members, err := h.DB.ListTeamMembers(ctx, id)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000025, "list members: "+err.Error())
			return
		}
		out := make([]TeamMemberDTO, 0, len(members))
		for _, m := range members {
			out = append(out, teamMemberToDTO(m))
		}
		OK(c, gin.H{"list": out, "total": len(out)})
	}
}

// AddMember invites a user to a team. Owner or admin only. The user
// must exist (FK enforcement catches it, mapped to 404).
func (h *TeamHandlers) AddMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		if !h.canManageTeam(c, id) {
			return
		}
		var req AddTeamMemberReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000023, "invalid request: "+err.Error())
			return
		}
		role := ProjectRole(req.Role)
		if !validProjectRole(role) {
			Fail(c, http.StatusBadRequest, 4000024, "invalid role: "+req.Role)
			return
		}
		if err := h.DB.AddTeamMember(ctx, id, req.UserID, role); err != nil {
			switch {
			case errors.Is(err, ErrMemberExists):
				Fail(c, http.StatusConflict, 4090011, "user already a member")
			case errors.Is(err, ErrFKMissing):
				Fail(c, http.StatusNotFound, 4040014, "user or team not found")
			default:
				Fail(c, http.StatusInternalServerError, 5000026, "add member: "+err.Error())
			}
			return
		}
		m, _ := h.DB.GetTeamMember(ctx, id, req.UserID)
		OK(c, teamMemberToDTO(m))
	}
}

// UpdateMember changes an existing member's role.
func (h *TeamHandlers) UpdateMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		uid := c.Param("uid")
		if !h.canManageTeam(c, id) {
			return
		}
		var req UpdateTeamMemberReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000025, "invalid request: "+err.Error())
			return
		}
		role := ProjectRole(req.Role)
		if !validProjectRole(role) {
			Fail(c, http.StatusBadRequest, 4000026, "invalid role: "+req.Role)
			return
		}
		if err := h.DB.UpdateTeamMemberRole(ctx, id, uid, role); err != nil {
			if errors.Is(err, ErrMemberNotFound) {
				Fail(c, http.StatusNotFound, 4040015, "member not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000027, "update member: "+err.Error())
			return
		}
		m, _ := h.DB.GetTeamMember(ctx, id, uid)
		OK(c, teamMemberToDTO(m))
	}
}

// RemoveMember kicks a user from a team.
func (h *TeamHandlers) RemoveMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		uid := c.Param("uid")
		// Self-removal is always allowed (so users can leave a team
		// without admin help). Admin / owner removal otherwise.
		if CurrentUserID(c) != uid && !h.canManageTeam(c, id) {
			return
		}
		if err := h.DB.RemoveTeamMember(ctx, id, uid); err != nil {
			if errors.Is(err, ErrMemberNotFound) {
				Fail(c, http.StatusNotFound, 4040016, "member not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000028, "remove member: "+err.Error())
			return
		}
		OK(c, gin.H{"team_id": id, "user_id": uid})
	}
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// canManageTeam returns true when the caller is admin OR the owner of
// team `id`. Writes the 403 response and returns false otherwise; the
// caller should `return` immediately on false.
func (h *TeamHandlers) canManageTeam(c *gin.Context, id string) bool {
	ctx := c.Request.Context()
	uid := CurrentUserID(c)
	if CurrentRole(c) == RoleAdmin {
		return true
	}
	t, err := h.DB.GetTeamByID(ctx, id)
	if err != nil {
		Fail(c, http.StatusNotFound, 4040017, "team not found")
		return false
	}
	if t.OwnerID != uid {
		Fail(c, http.StatusForbidden, 4030021, "only admin or team owner can manage this team")
		return false
	}
	return true
}

// teamToDTO converts the storage struct to the wire DTO.
func teamToDTO(t *Team) TeamDTO {
	return TeamDTO{
		ID:           t.ID,
		Name:         t.Name,
		Slug:         t.Slug,
		Description:  t.Description,
		OwnerID:      t.OwnerID,
		MemberCount:  t.MemberCount,
		ProjectCount: t.ProjectCount,
		CreatedAt:    t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    t.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Deleted:      t.Deleted,
	}
}

// teamMemberToDTO converts the storage struct to the wire DTO.
func teamMemberToDTO(m *TeamMember) TeamMemberDTO {
	return TeamMemberDTO{
		TeamID:   m.TeamID,
		UserID:   m.UserID,
		Username: m.Username,
		Role:     m.Role,
		JoinedAt: m.JoinedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// validProjectRole reports whether r is one of the canonical values.
// Used by handlers to short-circuit typos before the DB rejects them.
func validProjectRole(r ProjectRole) bool {
	for _, x := range AllProjectRoles {
		if x == r {
			return true
		}
	}
	return false
}

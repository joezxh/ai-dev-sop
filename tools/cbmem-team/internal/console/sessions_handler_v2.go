package console

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// v2 (M4) Session HTTP handlers.
// =============================================================================
//
// Routes mounted by router.go:
//
//   GET    /api/v2/sessions                   list + filter (team/project/module/user/path)
//   GET    /api/v2/sessions/:id               detail (session + turns)
//   GET    /api/v2/sessions/stats             aggregate totals + top-N
//   GET    /api/v2/projects/:pid/sessions      project-scoped list
//
// All routes require JWT auth. The detail page returns the same
// gin.H{session, turns} envelope the existing v1 detail uses so the
// UI can adopt v2 transparently.
// =============================================================================

// SessionHandlersV2 wraps the v2 storage layer.
type SessionHandlersV2 struct {
	DB *DB
}

// NewSessionHandlersV2 builds a SessionHandlersV2.
func NewSessionHandlersV2(db *DB) *SessionHandlersV2 { return &SessionHandlersV2{DB: db} }

// ----------------------------------------------------------------------------
// ListSessions
// ----------------------------------------------------------------------------

// ListSessions returns sessions matching the filter from the query
// string. Pagination is limit/offset; defaults are sensible.
//
// Filter query params: team_id, project_id, module_id, user_id,
// project_path, from, to, include_legacy (default true).
func (h *SessionHandlersV2) ListSessions() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		f := NewSessionFilter()
		f.TeamID = c.Query("team_id")
		f.ProjectID = c.Query("project_id")
		f.ModuleID = c.Query("module_id")
		if uid := c.Query("user_id"); uid != "" {
			if v, err := strconv.ParseInt(uid, 10, 64); err == nil {
				f.UserID = strconv.FormatInt(v, 10)
			}
		}
		f.ProjectPath = c.Query("project_path")
		if raw := c.Query("include_legacy"); raw != "" {
			b, err := strconv.ParseBool(raw)
			if err != nil {
				Fail(c, http.StatusBadRequest, 4000060, "include_legacy must be bool")
				return
			}
			f.IncludeLegacy = b
		}
		f.Limit, _ = strconv.Atoi(c.Query("limit"))
		f.Offset, _ = strconv.Atoi(c.Query("offset"))
		if from := c.Query("from"); from != "" {
			if t, err := parseTimeParam(from); err == nil {
				f.From = t
			}
		}
		if to := c.Query("to"); to != "" {
			if t, err := parseTimeParam(to); err == nil {
				f.To = t
			}
		}

		// Permission: admin sees all; others are scoped to teams
		// they're members of. For now we accept the filter as-is and
		// rely on the v2 tree/team members endpoints to gate. The
		// v1 admin-cookie path is unchanged.
		rows, total, err := h.DB.ListSessions(ctx, f)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000070, "list sessions: "+err.Error())
			return
		}
		out := make([]SessionDTOv2, 0, len(rows))
		for _, s := range rows {
			out = append(out, sessionToDTOv2(s))
		}
		OK(c, PageResp{List: out, Total: total, PageNo: 1, PageSize: f.Limit})
	}
}

// ProjectSessions returns sessions of a single project. Used by the
// project-detail page to render the "recent conversations" tile.
func (h *SessionHandlersV2) ProjectSessions() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		projectID := c.Param("pid")
		// Reuse the same access model as modules.
		mods := NewModuleHandlers(h.DB)
		if !mods.assertProjectAccess(c, projectID) {
			return
		}
		f := NewSessionFilter()
		f.ProjectID = projectID
		f.IncludeLegacy = false // project page wants curated data only
		f.Limit, _ = strconv.Atoi(c.Query("limit"))
		if f.Limit == 0 {
			f.Limit = 20
		}
		f.Offset, _ = strconv.Atoi(c.Query("offset"))
		rows, total, err := h.DB.ListSessions(ctx, f)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000071, "list project sessions: "+err.Error())
			return
		}
		out := make([]SessionDTOv2, 0, len(rows))
		for _, s := range rows {
			out = append(out, sessionToDTOv2(s))
		}
		OK(c, PageResp{List: out, Total: total, PageNo: 1, PageSize: f.Limit})
	}
}

// ----------------------------------------------------------------------------
// Detail
// ----------------------------------------------------------------------------

// GetSession returns the session header + its ordered turns. Same
// shape as the existing v1 detail endpoint so the UI can adopt
// either; both endpoints share the layout.
func (h *SessionHandlersV2) GetSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		s, err := h.DB.GetSession(ctx, id)
		if err != nil {
			if errors.Is(err, ErrSessionNotFound) {
				Fail(c, http.StatusNotFound, 4040070, "session not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000072, "read session: "+err.Error())
			return
		}
		if s.ProjectID != "" {
			mods := NewModuleHandlers(h.DB)
			if !mods.assertProjectAccess(c, s.ProjectID) {
				return
			}
		}
		turns, err := h.DB.ListSessionTurns(ctx, id)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000073, "list turns: "+err.Error())
			return
		}
		outTurns := make([]TurnDTO, 0, len(turns))
		for _, t := range turns {
			outTurns = append(outTurns, turnToDTO(t))
		}
		OK(c, SessionDetailDTO{Session: sessionToDTOv2(s), Turns: outTurns})
	}
}

// ----------------------------------------------------------------------------
// Stats
// ----------------------------------------------------------------------------

// Stats returns aggregate counts for the dashboard. Honours the same
// filter as ListSessions so callers can scope the totals to a
// project / module / user without writing two endpoints.
func (h *SessionHandlersV2) Stats() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		f := NewSessionFilter()
		f.TeamID = c.Query("team_id")
		f.ProjectID = c.Query("project_id")
		f.ModuleID = c.Query("module_id")
		if uid := c.Query("user_id"); uid != "" {
			if v, err := strconv.ParseInt(uid, 10, 64); err == nil {
				f.UserID = strconv.FormatInt(v, 10)
			}
		}
		stats, err := h.DB.AggregateSessions(ctx, f)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000074, "aggregate: "+err.Error())
			return
		}
		OK(c, stats)
	}
}

// ----------------------------------------------------------------------------
// DTO helpers
// ----------------------------------------------------------------------------

func sessionToDTOv2(s *Session) SessionDTOv2 {
	dto := SessionDTOv2{
		ID:                    s.ID,
		UserID:                s.UserID,
		TeamID:                s.TeamID,
		ProjectID:             s.ProjectID,
		ModuleID:              s.ModuleID,
		ProjectPath:           s.ProjectPath,
		StartedAt:             s.StartedAt.UTC().Format(time.RFC3339),
		ToolCount:             s.ToolCount,
		TurnCount:             s.TurnCount,
		Summary:               s.Summary,
		MemPalaceSyncedTurns:  s.MemPalaceSyncedTurns,
		MemPalaceLastError:    s.MemPalaceLastError,
	}
	if s.EndedAt != nil {
		v := s.EndedAt.UTC().Format(time.RFC3339)
		dto.EndedAt = &v
	}
	if s.MemPalaceLastSyncedAt != nil {
		v := s.MemPalaceLastSyncedAt.UTC().Format(time.RFC3339)
		dto.MemPalaceLastSyncedAt = &v
	}
	return dto
}

func turnToDTO(t *SessionTurn) TurnDTO {
	return TurnDTO{
		ID:        int(t.ID),
		SessionID: t.SessionID,
		TurnNo:    t.TurnNo,
		Role:      t.Role,
		Content:   t.Content,
		ToolsJSON: t.ToolsJSON,
		TS:        t.TS.UTC().Format(time.RFC3339),
	}
}

// parseTimeParam accepts RFC3339 or "YYYY-MM-DD HH:MM:SS".
func parseTimeParam(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, nil
	}
	return time.Time{}, errors.New("unsupported time format: " + s)
}

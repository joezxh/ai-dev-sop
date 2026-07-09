package console

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ListSessionsHandler returns a paginated list of MCP conversation sessions
// captured by CaptureSessions. All filters are optional and composed with
// AND. Dates are compared as raw strings against the SQLite DATETIME column,
// which accepts "YYYY-MM-DD HH:MM:SS" or RFC3339.
func ListSessionsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p PageReq
		_ = c.ShouldBindQuery(&p)
		p.Normalize()
		pageNo, pageSize := p.PageNo, p.PageSize

		uid := c.Query("user_id")
		proj := c.Query("project")
		from := c.Query("from")
		to := c.Query("to")

		// SELECT — note IFNULL on ended_at so NULL → "" for stable scans.
		q := `SELECT id, user_id, IFNULL(project_id,''), project_path, started_at, IFNULL(ended_at,''), tool_count, turn_count
		      FROM sessions WHERE 1=1`
		args := []any{}
		if uid != "" {
			q += ` AND user_id = ?`
			args = append(args, uid)
		}
		if proj != "" {
			q += ` AND project_path = ?`
			args = append(args, proj)
		}
		if from != "" {
			q += ` AND started_at >= ?`
			args = append(args, from)
		}
		if to != "" {
			q += ` AND started_at <= ?`
			args = append(args, to)
		}
		q += ` ORDER BY started_at DESC LIMIT ? OFFSET ?`
		args = append(args, pageSize, (pageNo-1)*pageSize)

		rows, err := db.QueryContext(c.Request.Context(), q, args...)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000030, "list sessions: "+err.Error())
			return
		}
		defer rows.Close()

		out := make([]SessionDTO, 0)
		for rows.Next() {
			var d SessionDTO
			var started time.Time
			var endedStr string
			if err := rows.Scan(&d.ID, &d.UserID, &d.ProjectID, &d.ProjectPath,
				&started, &endedStr, &d.ToolCount, &d.TurnCount); err != nil {
				Fail(c, http.StatusInternalServerError, 5000031, "scan session: "+err.Error())
				return
			}
			d.StartedAt = started.UTC().Format(time.RFC3339)
			if endedStr != "" {
				// ended_at column stores either RFC3339 or "YYYY-MM-DD HH:MM:SS";
				// pass through the raw string to preserve the format SQLite
				// round-tripped. The console front-end normalises display.
				ended := endedStr
				d.EndedAt = &ended
			}
			out = append(out, d)
		}
		if err := rows.Err(); err != nil {
			Fail(c, http.StatusInternalServerError, 5000032, "iterate sessions: "+err.Error())
			return
		}

		// Separate COUNT for the filtered total — same WHERE clause.
		cntQ := `SELECT COUNT(*) FROM sessions WHERE 1=1`
		cntArgs := []any{}
		if uid != "" {
			cntQ += ` AND user_id = ?`
			cntArgs = append(cntArgs, uid)
		}
		if proj != "" {
			cntQ += ` AND project_path = ?`
			cntArgs = append(cntArgs, proj)
		}
		if from != "" {
			cntQ += ` AND started_at >= ?`
			cntArgs = append(cntArgs, from)
		}
		if to != "" {
			cntQ += ` AND started_at <= ?`
			cntArgs = append(cntArgs, to)
		}
		var total int64
		if err := db.QueryRowContext(c.Request.Context(), cntQ, cntArgs...).Scan(&total); err != nil {
			Fail(c, http.StatusInternalServerError, 5000033, "count sessions: "+err.Error())
			return
		}

		OK(c, PageResp{List: out, Total: int(total), PageNo: pageNo, PageSize: pageSize})
	}
}

// SessionDetailHandler returns one session row plus its ordered turns. The
// response is a compound `gin.H{session, turns}` rather than the standard
// envelope payload — that's an explicit exception in the brief because the
// detail view needs both shapes together.
func SessionDetailHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var d SessionDTO
		var started time.Time
		var endedAt sql.NullTime
		err := db.QueryRowContext(c.Request.Context(),
			`SELECT id, user_id, IFNULL(project_id,''), project_path, started_at, ended_at, tool_count, turn_count
			 FROM sessions WHERE id = ?`, id,
		).Scan(&d.ID, &d.UserID, &d.ProjectID, &d.ProjectPath,
			&started, &endedAt, &d.ToolCount, &d.TurnCount)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Fail(c, http.StatusNotFound, 4040010, "session not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000034, "read session: "+err.Error())
			return
		}
		d.StartedAt = started.UTC().Format(time.RFC3339)
		if endedAt.Valid {
			s := endedAt.Time.UTC().Format(time.RFC3339)
			d.EndedAt = &s
		}

		rows, err := db.QueryContext(c.Request.Context(),
			`SELECT id, session_id, turn_no, role, content, IFNULL(tools_json,''), ts
			 FROM session_turns WHERE session_id = ?
			 ORDER BY turn_no ASC, id ASC`, id,
		)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000035, "list turns: "+err.Error())
			return
		}
		defer rows.Close()

		turns := make([]TurnDTO, 0)
		for rows.Next() {
			var t TurnDTO
			var ts time.Time
			if err := rows.Scan(&t.ID, &t.SessionID, &t.TurnNo, &t.Role,
				&t.Content, &t.ToolsJSON, &ts); err != nil {
				Fail(c, http.StatusInternalServerError, 5000036, "scan turn: "+err.Error())
				return
			}
			t.TS = ts.UTC().Format(time.RFC3339)
			turns = append(turns, t)
		}
		if err := rows.Err(); err != nil {
			Fail(c, http.StatusInternalServerError, 5000037, "iterate turns: "+err.Error())
			return
		}

		OK(c, gin.H{"session": d, "turns": turns})
	}
}

// SessionsStatsHandler aggregates sessions table into the dashboard tiles:
// total sessions, total turns, total tool calls, plus top-5 users and
// top-5 projects by session count.
func SessionsStatsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var sessionTotal, turnTotal, toolTotal int64
		if err := db.QueryRowContext(c.Request.Context(),
			`SELECT COUNT(*) FROM sessions`,
		).Scan(&sessionTotal); err != nil {
			Fail(c, http.StatusInternalServerError, 5000038, "count sessions: "+err.Error())
			return
		}
		if err := db.QueryRowContext(c.Request.Context(),
			`SELECT IFNULL(SUM(turn_count), 0) FROM sessions`,
		).Scan(&turnTotal); err != nil {
			Fail(c, http.StatusInternalServerError, 5000039, "sum turns: "+err.Error())
			return
		}
		if err := db.QueryRowContext(c.Request.Context(),
			`SELECT IFNULL(SUM(tool_count), 0) FROM sessions`,
		).Scan(&toolTotal); err != nil {
			Fail(c, http.StatusInternalServerError, 5000040, "sum tool_count: "+err.Error())
			return
		}

		topUsers := make([]map[string]any, 0)
		rows, err := db.QueryContext(c.Request.Context(),
			`SELECT user_id, COUNT(*) AS c FROM sessions GROUP BY user_id ORDER BY c DESC LIMIT 5`,
		)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000041, "top users: "+err.Error())
			return
		}
		for rows.Next() {
			var uid string
			var cnt int64
			if err := rows.Scan(&uid, &cnt); err != nil {
				rows.Close()
				Fail(c, http.StatusInternalServerError, 5000042, "scan top user: "+err.Error())
				return
			}
			topUsers = append(topUsers, map[string]any{"user_id": uid, "count": cnt})
		}
		rows.Close()

		topProjects := make([]map[string]any, 0)
		rows2, err := db.QueryContext(c.Request.Context(),
			`SELECT project_path, COUNT(*) AS c FROM sessions GROUP BY project_path ORDER BY c DESC LIMIT 5`,
		)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000043, "top projects: "+err.Error())
			return
		}
		for rows2.Next() {
			var proj string
			var cnt int64
			if err := rows2.Scan(&proj, &cnt); err != nil {
				rows2.Close()
				Fail(c, http.StatusInternalServerError, 5000044, "scan top project: "+err.Error())
				return
			}
			topProjects = append(topProjects, map[string]any{"project_path": proj, "count": cnt})
		}
		rows2.Close()

		OK(c, gin.H{
			"session_total":   sessionTotal,
			"turn_total":      turnTotal,
			"tool_call_total": toolTotal,
			"top_users":       topUsers,
			"top_projects":    topProjects,
		})
	}
}
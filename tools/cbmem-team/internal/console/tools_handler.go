package console

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ListToolsHandler returns the 49-tool catalog. Optional `?track=` and
// `?category=` filters narrow the result. Sorted by track asc, then
// priority asc.
func ListToolsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		track := c.Query("track")
		category := c.Query("category")

		q := `SELECT tool_id, track, category, name, signature, status, priority, version, created_at, updated_at
		      FROM tool_directory WHERE status != 'deprecated'`
		args := []any{}
		if track != "" {
			q += " AND track = ?"
			args = append(args, track)
		}
		if category != "" {
			q += " AND category = ?"
			args = append(args, category)
		}
		q += " ORDER BY track ASC, priority ASC, name ASC"

		rows, err := db.QueryContext(c.Request.Context(), q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var (
				toolID, track, category, name, signature, status, version string
				priority                                                  int
				createdAt, updatedAt                                      time.Time
			)
			if err := rows.Scan(&toolID, &track, &category, &name, &signature, &status, &priority, &version, &createdAt, &updatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, gin.H{
				"tool_id": toolID, "track": track, "category": category,
				"name": name, "signature": signature, "status": status,
				"priority": priority, "version": version,
				"created_at": createdAt, "updated_at": updatedAt,
			})
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tools": out, "count": len(out)})
	}
}

// RecentInvocationsHandler returns the most recent N invocations. Default
// 50, max 500. v0 endpoint — full filters come in M2 when the tool
// detail UI is live.
func RecentInvocationsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 50
		if v := c.Query("limit"); v != "" {
			var n int
			if _, err := fmtScanInt(v, &n); err == nil && n > 0 && n <= 500 {
				limit = n
			}
		}

		rows, err := db.QueryContext(c.Request.Context(), `
			SELECT invocation_id, user_id, project_path, tool_id, transport,
			       started_at, ended_at, latency_ms, error_code, timeout_flag
			FROM tool_invocation_logs
			ORDER BY started_at DESC
			LIMIT ?`, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var (
				invID, uid, proj, toolID, transport string
				errCode                             *string
				started, ended                      *time.Time
				latency                             *int
				timeout                             *bool
			)
			if err := rows.Scan(&invID, &uid, &proj, &toolID, &transport,
				&started, &ended, &latency, &errCode, &timeout); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			row := gin.H{
				"invocation_id": invID,
				"user_id":       uid,
				"project_path":  proj,
				"tool_id":       toolID,
				"transport":     transport,
				"started_at":    started,
				"ended_at":      ended,
				"latency_ms":    latency,
				"error_code":    errCode,
				"timeout_flag":  timeout,
			}
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"invocations": out, "count": len(out)})
	}
}

// fmtScanInt is a tiny helper that avoids pulling strconv into the hot
// path. Returns (chars-consumed, nil) or (0, error).
func fmtScanInt(s string, dst *int) (int, error) {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + int(ch-'0')
	}
	*dst = n
	return len(s), nil
}

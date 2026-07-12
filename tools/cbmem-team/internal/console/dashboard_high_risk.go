// Package console · M3 impact-assessment dashboard.
//
// Adds three handlers under /api/console/v2/dashboard/:
//
//   GET /v2/dashboard/high-risk          list high-risk invocations in the last 7d
//   GET /v2/dashboard/high-risk/summary  grouped counts (by tool, by user, by error_code)
//   GET /v2/dashboard/blast-radius       tools whose blast_radius_json touched the most artefacts
//
// The "high-risk" definition comes from the v_high_risk_invocations_7d
// view created by M3ExtraSchema: an invocation is high-risk if (in the
// last 7 days) it touched ≥1 hall/ADR, has a non-empty blast_radius_json,
// OR produced one of the configured fatal error codes (UPSTREAM_ERROR /
// FORBIDDEN / TIMEOUT).
//
// SQLite path uses a portable WHERE clause; the MySQL path queries the
// view directly. The auto-ticket hook lives in tickets.go.
package console

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HighRiskInvocation is one row of the high-risk dashboard list.
type HighRiskInvocation struct {
	InvocationID       string    `json:"invocation_id"`
	UserID             string    `json:"user_id"`
	ProjectID          string    `json:"project_id"`
	ToolID             string    `json:"tool_id"`
	Transport          string    `json:"transport"`
	StartedAt          time.Time `json:"started_at"`
	LatencyMs          int       `json:"latency_ms"`
	ErrorCode          string    `json:"error_code"`
	AffectedHallsJSON  string    `json:"affected_halls_json"`
	AffectedADRsJSON   string    `json:"affected_adrs_json"`
	BlastRadiusJSON    string    `json:"blast_radius_json"`
}

// HighRiskSummary groups counts by tool, by user, and by error_code.
type HighRiskSummary struct {
	Window    string                 `json:"window"`
	Total     int                    `json:"total"`
	ByTool    []HighRiskCountRow     `json:"by_tool"`
	ByUser    []HighRiskCountRow     `json:"by_user"`
	ByError   []HighRiskCountRow     `json:"by_error_code"`
	Generated time.Time              `json:"generated_at"`
}

type HighRiskCountRow struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// HighRiskDashboardHandler returns the 7-day high-risk invocation list.
func HighRiskDashboardHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := atoiOrDefault(c.Query("limit"), 100)
		if limit < 1 || limit > 1000 {
			limit = 100
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// MySQL path: lean on the v_high_risk_invocations_7d view so the
		// filter lives in one place. SQLite path: use the inline form
		// (the test harness keeps SQLite minimal — no view creation).
		var q string
		if db.driver == "mysql" {
			q = `SELECT invocation_id, user_id, project_id, tool_id, transport,
                       started_at, latency_ms, error_code,
                       affected_halls_json, affected_adrs_json, blast_radius_json
                FROM v_high_risk_invocations_7d
                ORDER BY started_at DESC
                LIMIT ` + itoa(limit)
		} else {
			q = `SELECT invocation_id, user_id, project_id, tool_id, transport,
                       started_at, latency_ms, error_code,
                       affected_halls_json, affected_adrs_json, blast_radius_json
                FROM tool_invocation_logs
                WHERE started_at >= datetime('now','-7 days') AND (
                    JSON_LENGTH(affected_halls_json) > 0
                    OR JSON_LENGTH(affected_adrs_json) > 0
                    OR JSON_LENGTH(blast_radius_json) > 0
                    OR error_code IN ('UPSTREAM_ERROR','FORBIDDEN','TIMEOUT')
                )
                ORDER BY started_at DESC
                LIMIT ` + itoa(limit)
		}
		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []HighRiskInvocation{}
		for rows.Next() {
			var r HighRiskInvocation
			var projectID, errCode, halls, adrs, blast sql.NullString
			if err := rows.Scan(&r.InvocationID, &r.UserID, &projectID, &r.ToolID,
				&r.Transport, &r.StartedAt, &r.LatencyMs, &errCode,
				&halls, &adrs, &blast); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			r.ProjectID = projectID.String
			r.ErrorCode = errCode.String
			r.AffectedHallsJSON = halls.String
			r.AffectedADRsJSON = adrs.String
			r.BlastRadiusJSON = blast.String
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"invocations": out, "count": len(out), "window": "7d"})
	}
}

// HighRiskSummaryHandler returns the aggregated high-risk summary.
func HighRiskSummaryHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		summary := HighRiskSummary{Window: "7d", Generated: time.Now().UTC()}

		// total
		if db.driver == "mysql" {
			if err := db.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM v_high_risk_invocations_7d`,
			).Scan(&summary.Total); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			if err := db.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM tool_invocation_logs
                 WHERE started_at >= datetime('now','-7 days') AND (
                     JSON_LENGTH(affected_halls_json) > 0
                     OR JSON_LENGTH(affected_adrs_json) > 0
                     OR JSON_LENGTH(blast_radius_json) > 0
                     OR error_code IN ('UPSTREAM_ERROR','FORBIDDEN','TIMEOUT')
                 )`,
			).Scan(&summary.Total); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		// by tool / user / error_code — reuse the inline form on both
		// backends so the GROUP BY semantics match exactly.
		inlineWhere := `started_at >= datetime('now','-7 days') AND (
            JSON_LENGTH(affected_halls_json) > 0
            OR JSON_LENGTH(affected_adrs_json) > 0
            OR JSON_LENGTH(blast_radius_json) > 0
            OR error_code IN ('UPSTREAM_ERROR','FORBIDDEN','TIMEOUT')
        )`
		// MySQL: replace `datetime('now',...)` with `NOW() - INTERVAL 7 DAY`.
		if db.driver == "mysql" {
			inlineWhere = `started_at >= (NOW() - INTERVAL 7 DAY) AND (
                JSON_LENGTH(affected_halls_json) > 0
                OR JSON_LENGTH(affected_adrs_json) > 0
                OR JSON_LENGTH(blast_radius_json) > 0
                OR error_code IN ('UPSTREAM_ERROR','FORBIDDEN','TIMEOUT')
            )`
		}
		for _, target := range []struct {
			column string
			dest   *[]HighRiskCountRow
		}{
			{"tool_id", &summary.ByTool},
			{"user_id", &summary.ByUser},
			{"error_code", &summary.ByError},
		} {
			q := `SELECT ` + target.column + ` AS k, COUNT(*) AS cnt
                  FROM tool_invocation_logs WHERE ` + inlineWhere + `
                  GROUP BY ` + target.column + ` ORDER BY cnt DESC LIMIT 20`
			rows, err := db.QueryContext(ctx, q)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			for rows.Next() {
				var r HighRiskCountRow
				if err := rows.Scan(&r.Key, &r.Count); err != nil {
					rows.Close()
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				*target.dest = append(*target.dest, r)
			}
			rows.Close()
		}

		c.JSON(http.StatusOK, summary)
	}
}

// atoiOrDefault is a small helper for query-string ints.
//
// Returns def when:
//   - the string is empty
//   - any character is not 0-9
//
// Returns the parsed integer otherwise, including 0. The function does
// not check for overflow — callers that need an upper bound should clamp
// after this call (the M4 candidate handler clamps to [1,500]).
func atoiOrDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
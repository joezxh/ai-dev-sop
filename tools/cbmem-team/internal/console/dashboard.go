package console

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// DashboardSummary is the v0 dashboard payload: P50 / P95 / error-rate
// per tool, plus a top-line total invocation count. Computed with native
// SQL aggregate — no analytics store required at M1.
type DashboardSummary struct {
	Window           string       `json:"window"` // "1h" | "24h" | "7d"
	TotalInvocations int64        `json:"total_invocations"`
	ErrorRate        float64      `json:"error_rate"`
	ByTool           []ToolMetric `json:"by_tool"`
	GeneratedAt      time.Time    `json:"generated_at"`
}

type ToolMetric struct {
	ToolID     string `json:"tool_id"`
	Count      int64  `json:"count"`
	ErrorCount int64  `json:"error_count"`
	LatencyP50 int    `json:"latency_p50_ms"`
	LatencyP95 int    `json:"latency_p95_ms"`
	LatencyAvg int    `json:"latency_avg_ms"`
}

// DashboardSummaryHandler returns the v0 dashboard summary. Window is
// `1h`, `24h`, or `7d`. Defaults to `24h`.
func DashboardSummaryHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		window := c.DefaultQuery("window", "24h")
		since, ok := parseWindow(window)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid window (use 1h|24h|7d)"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		summary := DashboardSummary{Window: window, GeneratedAt: time.Now().UTC()}

		// Total invocations in window.
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM tool_invocation_logs WHERE started_at >= ?`, since,
		).Scan(&summary.TotalInvocations); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Error rate.
		var errCount int64
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM tool_invocation_logs WHERE started_at >= ? AND error_code IS NOT NULL`,
			since,
		).Scan(&errCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if summary.TotalInvocations > 0 {
			summary.ErrorRate = float64(errCount) / float64(summary.TotalInvocations)
		}

		// Per-tool metrics: count / error_count / P50 / P95 / avg.
		// MySQL 8.0 has PERCENT_RANK / CUME_DIST; for v0 we use a simple
		// ROW_NUMBER + COUNT approach that's portable across both SQLite
		// and MySQL by computing percentiles in Go after fetching latencies.
		rows, err := db.QueryContext(ctx, `
			SELECT tool_id,
			       COUNT(*) AS cnt,
			       SUM(CASE WHEN error_code IS NOT NULL THEN 1 ELSE 0 END) AS err_cnt,
			       GROUP_CONCAT(latency_ms ORDER BY latency_ms) AS latencies
			FROM tool_invocation_logs
			WHERE started_at >= ? AND latency_ms IS NOT NULL
			GROUP BY tool_id
			ORDER BY cnt DESC
			LIMIT 50`, since)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var (
				toolID          string
				cnt, errCnt     int64
				latenciesConcat string
			)
			if err := rows.Scan(&toolID, &cnt, &errCnt, &latenciesConcat); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			latencies := parseLatencyCSV(latenciesConcat)
			p50, p95, avg := percentile(latencies, 50), percentile(latencies, 95), avgInt(latencies)
			summary.ByTool = append(summary.ByTool, ToolMetric{
				ToolID: toolID, Count: cnt, ErrorCount: errCnt,
				LatencyP50: p50, LatencyP95: p95, LatencyAvg: avg,
			})
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, summary)
	}
}

func parseWindow(s string) (time.Time, bool) {
	now := time.Now().UTC()
	switch s {
	case "1h":
		return now.Add(-1 * time.Hour), true
	case "24h":
		return now.Add(-24 * time.Hour), true
	case "7d":
		return now.Add(-7 * 24 * time.Hour), true
	}
	return time.Time{}, false
}

// parseLatencyCSV splits a comma-separated latency string produced by
// GROUP_CONCAT(latency_ms ORDER BY latency_ms). Whitespace is trimmed.
func parseLatencyCSV(s string) []int {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// percentile returns the n-th percentile of sorted data using linear
// interpolation. Caller must pass already-sorted data.
func percentile(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := float64(p) / 100.0 * float64(len(sorted)-1)
	low := int(rank)
	high := low + 1
	if high >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := rank - float64(low)
	return sorted[low] + int(frac*float64(sorted[high]-sorted[low]))
}

func avgInt(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	sum := 0
	for _, x := range xs {
		sum += x
	}
	return sum / len(xs)
}

// Package console · M2 rate-limit admin endpoint.
//
// Mounted by router.go under /api/console/v2/tools/:id/rate-limit:
//
//   GET    /api/console/v2/tools/:id/rate-limit   fetch current config (or defaults)
//   PUT    /api/console/v2/tools/:id/rate-limit   overwrite (validates values)
//   DELETE /api/console/v2/tools/:id/rate-limit   reset to defaults (NULL row)
//   POST   /api/console/v2/tools/:id/rate-limit/reset  force-clear in-memory state
//
// The DELETE endpoint resets only the persisted row; the in-memory
// rateLimiter keeps the old entry until its cfgAt expires or the
// process restarts. The POST /reset endpoint fixes that immediately.
package console

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetRateLimitHandler returns the persisted JSON (or defaults if NULL).
func GetRateLimitHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		cfg, err := LoadRateLimit(c.Request.Context(), db, id)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tool_id": id, "config": cfg, "is_default": err != nil})
	}
}

// PutRateLimitHandler overwrites the persisted config. The next request
// through the middleware picks it up because Check calls LoadRateLimit
// every time.
func PutRateLimitHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var cfg RateLimitConfig
		if err := c.ShouldBindJSON(&cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validateRateLimit(cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		raw, _ := json.Marshal(cfg)
		_, err := db.ExecContext(c.Request.Context(),
			`UPDATE tool_directory SET rate_limit_json = ?, updated_at = ?
			 WHERE tool_id = ?`, string(raw), nowUTC(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tool_id": id, "config": cfg})
	}
}

// DeleteRateLimitHandler sets the column back to NULL (= use defaults).
func DeleteRateLimitHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		_, err := db.ExecContext(c.Request.Context(),
			`UPDATE tool_directory SET rate_limit_json = NULL, updated_at = ?
			 WHERE tool_id = ?`, nowUTC(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tool_id": id, "config": DefaultRateLimit(), "reset": true})
	}
}

// ResetRateLimitHandler drops in-memory entries for this tool. Useful
// after a hot edit when the operator wants the new config to apply
// to existing flows (not just new requests).
func ResetRateLimitHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		cleared := 0
		RateLimiter.mu.Lock()
		for k := range RateLimiter.entries {
			if len(k) > len(id) && k[:len(id)+1] == id+"|" {
				delete(RateLimiter.entries, k)
				cleared++
			}
		}
		RateLimiter.mu.Unlock()
		c.JSON(http.StatusOK, gin.H{"tool_id": id, "cleared_entries": cleared})
	}
}

// ListRateLimitsHandler returns a paginated view of every tool that
// has a non-default rate limit. Useful for the admin dashboard.
func ListRateLimitsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.QueryContext(c.Request.Context(),
			`SELECT tool_id, rate_limit_json FROM tool_directory
			 WHERE rate_limit_json IS NOT NULL AND rate_limit_json != ''
			 ORDER BY tool_id ASC`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var id, raw string
			if err := rows.Scan(&id, &raw); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			var cfg RateLimitConfig
			if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
				continue
			}
			out = append(out, gin.H{"tool_id": id, "config": cfg})
		}
		c.JSON(http.StatusOK, gin.H{"rate_limits": out, "count": len(out)})
	}
}

// validateRateLimit rejects clearly-broken values. We allow zero
// values individually (some operators may want only one of the three
// gates), but not all-zero (use DefaultRateLimit instead).
func validateRateLimit(cfg RateLimitConfig) error {
	if cfg.QPSPerUser < 0 || cfg.QPMPerUser < 0 || cfg.ConcurrencyGlobal < 0 {
		return errors.New("rate-limit values must be >= 0")
	}
	if cfg.CooldownMsAfterError < 0 {
		return errors.New("cooldown_ms_after_error must be >= 0")
	}
	if cfg.QPSPerUser == 0 && cfg.QPMPerUser == 0 && cfg.ConcurrencyGlobal == 0 {
		return errors.New("at least one of qps_per_user / qpm_per_user / concurrency_global must be > 0; use DELETE to reset")
	}
	switch cfg.QueuePriority {
	case "", "P0", "P1", "P2", "P3", "P4":
	default:
		return errors.New("queue_priority must be one of P0..P4")
	}
	return nil
}
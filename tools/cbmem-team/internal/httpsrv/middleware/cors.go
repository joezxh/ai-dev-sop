// Package middleware contains HTTP middlewares for cbmem-team.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a Gin middleware that implements a small, strict CORS policy.
//
// Semantics:
//   - allowed: origin whitelist. "*" matches any origin (dev only).
//     Empty slice disables CORS entirely (no response headers).
//   - On a permitted request: ACAO echoes the request Origin, Vary: Origin
//     is always set (per RFC 7234 to keep caches honest), and the
//     standard preflight headers are present so OPTIONS round-trips work.
//   - On an OPTIONS preflight: the request is short-circuited with 204
//     and headers are written even if the path has no handler.
func CORS(allowed []string) gin.HandlerFunc {
	set := make(map[string]struct{}, len(allowed))
	wildcard := false
	for _, o := range allowed {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			wildcard = true
			continue
		}
		set[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allow := wildcard || origin == ""
		if !allow {
			_, allow = set[origin]
		}
		if allow {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Admin-Token")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

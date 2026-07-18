package mcp

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/pool"
	"cbmem-team/internal/repos"
	"cbmem-team/internal/store"
)

// Handler routes a POST /mcp request to the per-user stdio subprocess.
//
// Access control happens in two layers:
//
//  1. JWT must be valid (set by middleware, user_id in context)
//  2. user must be registered and not disabled (store.Registry)
//  3. requested project path must match either the user's allow-list
//     (ProjectPaths) or any local clone registered for that user
//
// Once authorised, the raw JSON-RPC body is forwarded to the user's
// codebase-memory-mcp stdio process and the response is streamed back.
// We deliberately do NOT parse JSON here - keeping the wrapper dumb
// means we never have to keep up with upstream schema changes.
func Handler(p *pool.Pool, users *store.Registry, repos *repos.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("user_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no user_id in token"})
			return
		}
		userID, _ := v.(string)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "empty user_id"})
			return
		}
		u, exists := users.Get(userID)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "user not registered"})
			return
		}
		if u.Disabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "user disabled"})
			return
		}

		project := c.Query("project")
		if project == "" {
			project = c.GetHeader("X-Project-Path")
		}
		// Allow either explicit allow-list paths OR a clone we manage for this user.
		allowed := append([]string{}, u.ProjectPaths...)
		if repos != nil {
			for _, r := range repos.List(userID) {
				allowed = append(allowed, r.LocalPath)
			}
		}
		if len(allowed) > 0 && !anyMatch(allowed, project) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "project not in allow-list",
				"allowed_prefix": allowed,
			})
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := p.Request(userID, project, string(body))
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		// Notifications produce no response frame; ack per the MCP spec
		// with 202 Accepted and an empty body.
		if resp == "" {
			c.Status(http.StatusAccepted)
			return
		}
		c.Data(http.StatusOK, "application/json", []byte(resp))
	}
}

func anyMatch(prefixes []string, p string) bool {
	if p == "" {
		return false
	}
	for _, pre := range prefixes {
		if pre == "" {
			continue
		}
		// wildcard: "*" matches everything
		if pre == "*" {
			return true
		}
		// exact match
		if p == pre {
			return true
		}
		// prefix match (existing semantics)
		if len(pre) <= len(p) && strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

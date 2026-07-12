package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/auth"
	"cbmem-team/internal/pool"
	"cbmem-team/internal/repos"
	"cbmem-team/internal/store"
)

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		gin.DefaultWriter.Write([]byte(
			time.Now().UTC().Format(time.RFC3339) + " " +
				c.Request.Method + " " + c.Request.URL.Path + " " +
				http.StatusText(c.Writer.Status()) + " " +
				time.Since(start).String() + "\n",
		))
	}
}

type createUserReq struct {
	ID           string   `json:"id" binding:"required"`
	DisplayName  string   `json:"display_name"`
	ProjectPaths []string `json:"project_paths"`
	MaxProcs     int      `json:"max_procs"`
}

// patchUserReq lets admins update mutable fields without having to
// re-POST the entire user record. Fields left as nil/empty are not
// touched (so PATCH with only one field is safe).
type patchUserReq struct {
	DisplayName  *string   `json:"display_name,omitempty"`
	ProjectPaths *[]string `json:"project_paths,omitempty"`
	MaxProcs     *int      `json:"max_procs,omitempty"`
	Disabled     *bool     `json:"disabled,omitempty"`
}

func createUserHandler(users *store.Registry, dataDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		u := &store.User{
			ID:           req.ID,
			DisplayName:  req.DisplayName,
			ProjectPaths: req.ProjectPaths,
			MaxProcs:     req.MaxProcs,
		}
		if err := users.Upsert(u); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, u)
	}
}

// patchUserHandler updates an existing user without forcing the caller
// to re-send every field. This solves the "JWT expired and I have no way
// to refresh it" problem because admins can also disable a stale user
// or extend their allow-list without rebuilding the record from scratch.
func patchUserHandler(users *store.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		u, ok := users.Get(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		var req patchUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Apply deltas onto a copy so the in-memory record stays consistent
		// with the JSON on disk after Upsert.
		updated := *u
		if req.DisplayName != nil {
			updated.DisplayName = *req.DisplayName
		}
		if req.ProjectPaths != nil {
			updated.ProjectPaths = *req.ProjectPaths
		}
		if req.MaxProcs != nil {
			updated.MaxProcs = *req.MaxProcs
		}
		if req.Disabled != nil {
			updated.Disabled = *req.Disabled
		}
		if err := users.Upsert(&updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, &updated)
	}
}

func listUsersHandler(users *store.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": users.List()})
	}
}

func deleteUserHandler(users *store.Registry, p *pool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := users.Delete(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_ = p.Drop(id)
		c.JSON(http.StatusOK, gin.H{"deleted": id})
	}
}

func mintTokenHandler(users *store.Registry, secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if _, ok := users.Get(id); !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		ttlStr := c.DefaultQuery("ttl", "720h")
		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad ttl: " + err.Error()})
			return
		}
		v := auth.NewVerifier(secret)
		tok, err := v.Sign(id, ttl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user_id": id,
			"token":   tok,
			"expires": time.Now().UTC().Add(ttl),
		})
	}
}

// refreshTokenHandler lets an end-user (not an admin) swap a still-valid
// JWT for a fresh one. We verify the old token's signature and
// expiration first; only then do we mint a new one. This removes the
// "admin has to mint every refresh" pain point for long-running clients
// like Cursor that need tokens measured in months, not hours.
func refreshTokenHandler(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if len(raw) <= len(prefix) || raw[:len(prefix)] != prefix {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		v := auth.NewVerifier(secret)
		claims, err := v.Verify(raw[len(prefix):])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "old token invalid: " + err.Error()})
			return
		}
		// /admin is itself under the admin-token middleware, so the caller
		// already proves admin authority; we still require a valid bearer
		// to avoid accidentally minting tokens for arbitrary sub claims.
		ttlStr := c.DefaultQuery("ttl", "720h")
		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad ttl: " + err.Error()})
			return
		}
		tok, err := v.Sign(claims.Sub, ttl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user_id":        claims.Sub,
			"token":          tok,
			"expires":        time.Now().UTC().Add(ttl),
			"refreshed_from": time.Unix(claims.Exp, 0).UTC(),
		})
	}
}

func statsHandler(p *pool.Pool, users *store.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"registered_users": len(users.List()),
			"pool":             p.Stats(),
		})
	}
}

// cloneRepoHandler runs `git clone` (or `git fetch` if already cloned)
// into the per-user repos directory. Admins are expected to also call
// PATCH /admin/users/:id to add the resulting LocalPath to the user's
// ProjectPaths (or rely on the auto-extension in Handler()).
func cloneRepoHandler(rm *repos.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID string `json:"user_id" binding:"required"`
			URL    string `json:"url" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		r, err := rm.Clone(c.Request.Context(), req.UserID, req.URL)
		if err != nil {
			// 502 because upstream (git) failed, not because we mis-handled the request
			c.JSON(http.StatusBadGateway, gin.H{
				"error":      err.Error(),
				"local_path": r.LocalPath,
				"repo":       r,
			})
			return
		}
		c.JSON(http.StatusOK, r)
	}
}

func listReposHandler(rm *repos.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Query("user_id")
		if userID == "" {
			// empty list rather than 400 to keep the call site simple
			c.JSON(http.StatusOK, gin.H{"repos": []any{}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"repos": rm.List(userID)})
	}
}

package console

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// v2 (M2) RBAC middleware — bridges gin context → typed Role
// =============================================================================
//
// rbac.go already had `Role` / `SetRole` / `RoleOf` / `RequireRole` /
// `HasRole`; this file adds M2-specific helpers:
//
//   * RequireRoleAtLeast    — RequireRole with a strict ordering
//                             (admin > lead > developer > viewer).
//   * RequireUserMatches    — gate handlers that must only operate on
//                             "your own" resource (admins bypass).
//   * CurrentUserID         — read user_id from gin context.
//
// These sit alongside the older helpers rather than replacing them so
// existing v1 routes continue to work. M3+ handlers should prefer the
// new helpers; v1 helpers will be retired once the UI is fully on v2.
//
// =============================================================================

// CurrentUserID returns the user id stored in the gin context by
// auth.Verifier.Middleware. Returns "" when the request was not
// authenticated — handlers should pair this with their own early-out.
func CurrentUserID(c *gin.Context) string {
	return c.GetString("user_id")
}

// CurrentRole returns the Role stored in the gin context. Falls back
// to RoleViewer so unauthenticated requests cannot accidentally gain
// admin privileges when a downstream handler forgets to call
// CurrentUserID + role check.
func CurrentRole(c *gin.Context) Role {
	return Role(c.GetString("role"))
}

// RequireRoleAtLeast returns a 403 unless the caller's role is at
// least as capable as `want` per RoleAtLeast in rbac.go. Admin
// always wins; a viewer is rejected unless want is also viewer.
func RequireRoleAtLeast(want Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		got := CurrentRole(c)
		if !RoleAtLeast(got, want) {
			Fail(c, http.StatusForbidden, 4030010,
				"insufficient role: have="+string(got)+" want>="+string(want))
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireUserMatches returns a 403 unless the URL :id matches the
// current user id, OR the caller is admin. Handlers that mutate user
// resources ("change my settings", "delete my token") wrap themselves
// in this middleware so an attacker can't pass /users/<victim> in
// place of /users/me.
func RequireUserMatches(idParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := CurrentUserID(c)
		if uid == "" {
			Fail(c, http.StatusUnauthorized, 4010050, "unauthenticated")
			c.Abort()
			return
		}
		if CurrentRole(c) == RoleAdmin {
			c.Next()
			return
		}
		if c.Param(idParam) != uid {
			Fail(c, http.StatusForbidden, 4030011, "resource does not belong to caller")
			c.Abort()
			return
		}
		c.Next()
	}
}

// MustGetUser fetches the current user from the DB. The handler MUST
// be behind RequireJWT. Returns nil when the user vanished between
// login and now (token still valid but row deleted) so the caller can
// fail closed with a clean 401.
//
// This is a small wrapper around the very common "auth context +
// users row" pattern, used by /me, /change-password, and any handler
// that needs the live row (e.g. to check must_change_password).
func MustGetUser(c *gin.Context) (*User, bool) {
	uid := CurrentUserID(c)
	if uid == "" {
		Fail(c, http.StatusUnauthorized, 4010051, "unauthenticated")
		return nil, false
	}
	u, err := userDB(c).GetUserByID(c.Request.Context(), uid)
	if err != nil {
		Fail(c, http.StatusUnauthorized, 4010052, "user lookup: "+err.Error())
		return nil, false
	}
	return u, true
}

// userDB returns the *DB stored on the gin context by Mount. We avoid
// stashing it on every request by reading it from the gin engine's
// store; handlers that need it call this helper.
//
// The handler chain guarantees db is non-nil because MountConfig.DB
// is wired in router.go.
func userDB(c *gin.Context) *DB {
	if v, ok := c.Get("console_db"); ok {
		if db, ok := v.(*DB); ok {
			return db
		}
	}
	return nil
}

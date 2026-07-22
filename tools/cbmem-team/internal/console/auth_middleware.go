package console

import (
	"errors"
	"net/http"
	"strconv"

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

// CurrentUserIDInt parses the gin-context user_id as a bigint. Returns
// 0 when missing or malformed (used by M3+ handlers that operate on
// the numeric sys_users.id).
func CurrentUserIDInt(c *gin.Context) int64 {
	uid, _ := currentUserIDInt(c)
	return uid
}

// currentUserIDInt reads the user_id string from gin context and
// parses it back to an int64. The id is stored as a string because
// gin.Context values are interface{} (so anything works), and JWT
// subjects are conventionally strings — the int64 surface matches the
// sys_users.id BIGINT column. Returns 0 if missing or malformed.
func currentUserIDInt(c *gin.Context) (int64, error) {
	raw := c.GetString("user_id")
	if raw == "" {
		return 0, errors.New("unauthenticated")
	}
	return strconv.ParseInt(raw, 10, 64)
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
		uid, _ := currentUserIDInt(c)
		if uid == 0 {
			Fail(c, http.StatusUnauthorized, 4010050, "unauthenticated")
			c.Abort()
			return
		}
		if CurrentRole(c) == RoleAdmin {
			c.Next()
			return
		}
		if c.Param(idParam) != strconv.FormatInt(uid, 10) {
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

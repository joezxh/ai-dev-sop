package console

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Role identifies the capability level of a console session. The four
// values correspond to the roles in deploy/sql/v2_schema.*.sql (users.role
// CHECK constraint) and the models.User.Role field.
//
// Roles are intentionally ordered for the iterators in AllRoles below:
// higher in the list = more capable. Code that needs an "is at least X"
// check should use RoleAtLeast.
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleLead      Role = "lead"
	RoleDeveloper Role = "developer"
	RoleViewer    Role = "viewer"
)

// AllRoles is the canonical ordering used by iterators and tests.
var AllRoles = []Role{RoleAdmin, RoleLead, RoleDeveloper, RoleViewer}

// RoleAtLeast returns true when `r` is at least as capable as `want`,
// according to AllRoles (index 0 = strongest).
func RoleAtLeast(r, want Role) bool {
	for _, x := range AllRoles {
		if x == want {
			// r is "at least want" if r appears at or before want in
			// the canonical ordering. Anything after want (viewer at
			// the tail) is rejected.
			for _, y := range AllRoles {
				if y == r {
					return true
				}
				if y == want {
					return false
				}
			}
			return false
		}
	}
	return false
}

// SetRole injects the role into the gin context. Callers (login /
// session middleware) compute the role from underlying credentials
// and call this once per request.
func SetRole(c *gin.Context, r Role) { c.Set("role", string(r)) }

// RoleOf returns the role stored in the gin context. Defaults to
// admin so the present single-tier system remains unaffected;
// switching the default to a derived value is part of M5 follow-up.
func RoleOf(c *gin.Context) Role {
	if v, ok := c.Get("role"); ok {
		if rs, ok2 := v.(string); ok2 && rs != "" {
			return Role(rs)
		}
	}
	return RoleAdmin
}

// HasRole reports whether the current request's role appears in the
// allowed set. Helper for handlers that prefer a single conditional
// over chaining RequireRole middleware.
func HasRole(c *gin.Context, allowed ...Role) bool {
	got := RoleOf(c)
	for _, r := range allowed {
		if got == r {
			return true
		}
	}
	return false
}

// RequireRole returns middleware that 403s any request whose role is
// not in the allowed list. Compose with RequireSession /
// RequireCSRF. Example:
//
//	r.POST("/admin/users", RequireSession(sm), RequireCSRF(),
//	    RequireRole(RoleAdmin), CreateUserHandler(...))
func RequireRole(allowed ...Role) gin.HandlerFunc {
	allow := make(map[Role]struct{}, len(allowed))
	for _, r := range allowed {
		allow[r] = struct{}{}
	}
	return func(c *gin.Context) {
		got := RoleOf(c)
		if _, ok := allow[got]; !ok {
			Fail(c, http.StatusForbidden, 4030002,
				"role not permitted (required one of ["+joinRoles(allowed)+"], got "+string(got)+")")
			c.Abort()
			return
		}
		c.Next()
	}
}

func joinRoles(rs []Role) string {
	if len(rs) == 0 {
		return ""
	}
	out := string(rs[0])
	for i := 1; i < len(rs); i++ {
		out += "," + string(rs[i])
	}
	return out
}

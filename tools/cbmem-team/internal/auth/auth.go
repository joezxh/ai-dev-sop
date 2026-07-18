// Package auth handles authentication for two distinct audiences:
//
//   - JWT bearer tokens: end users (developers) issuing MCP requests
//   - Static admin token: operators using /admin endpoints
//   - Password login: console UI (and other password-aware clients)
//     trading username/password for a short-lived JWT + refresh pair
//
// JWTs are HS256 with a single shared secret. We do *not* require user
// registration in the claim - the user_id is derived from the URL path
// parameter (`?as=user_id`) to keep MCP clients simple. The presence of
// a valid signature is the gate; per-user allow-listing is enforced by
// the store.Registry.
//
// v2 (M2) extends the claim set with Role and TokenID so handlers can
// implement RBAC and so refresh tokens can be individually revoked by
// matching the jti claim against refresh_tokens.revoked_at.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Role is duplicated here as a tiny string alias so the auth package
// stays free of any /internal/console dependency. Console's rbac.go
// carries the typed const list; handlers convert with `auth.Role(u.Role)`.
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleLead      Role = "lead"
	RoleDeveloper Role = "developer"
	RoleViewer    Role = "viewer"
)

// Claims is the JWT payload. New fields are additive; existing
// verifiers ignore unknown fields, so rolling out the v2 fields does
// not require a coordinated client/server upgrade.
type Claims struct {
	Sub    string `json:"sub"`    // user id
	Exp    int64  `json:"exp"`    // unix seconds
	Iat    int64  `json:"iat"`    // unix seconds
	Jti    string `json:"jti,omitempty"` // refresh-token id; v2 only
	Role   string `json:"role,omitempty"` // users.role; v2 only
	Kind   string `json:"kind,omitempty"` // "access" | "refresh"; v2 only
}

type Verifier struct {
	secret []byte
}

func NewVerifier(secret []byte) *Verifier {
	return &Verifier{secret: secret}
}

func (v *Verifier) Sign(userID string, ttl time.Duration) (string, error) {
	return v.SignClaims(Claims{Sub: userID, Kind: "access"}, ttl)
}

// SignClaims signs an arbitrary claim set. jti is auto-populated when
// the caller leaves it empty so access tokens always carry a unique
// id (used to revoke on logout / password change).
func (v *Verifier) SignClaims(c Claims, ttl time.Duration) (string, error) {
	if len(v.secret) == 0 {
		return "", errors.New("jwt secret not configured")
	}
	if c.Exp == 0 {
		c.Exp = time.Now().UTC().Add(ttl).Unix()
	}
	if c.Iat == 0 {
		c.Iat = time.Now().UTC().Unix()
	}
	if c.Jti == "" {
		id, err := randomID(16)
		if err != nil {
			return "", fmt.Errorf("mint jti: %w", err)
		}
		c.Jti = id
	}
	if c.Kind == "" {
		c.Kind = "access"
	}
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(c)
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(hb) + "." + enc.EncodeToString(pb)
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(signing))
	sig := enc.EncodeToString(mac.Sum(nil))
	return signing + "." + sig, nil
}

// randomID returns a hex-encoded random string of nBytes bytes. Used
// for jti and refresh-token id; collision odds at 16 bytes are
// ~10^-38 which is comfortable for the lifetime of a single install.
func randomID(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (v *Verifier) Verify(token string) (*Claims, error) {
	return v.verifyKind(token, "")
}

// VerifyKind is Verify with an expected Kind claim. The empty string
// disables the check (preserving v1 behaviour); pass "access" or
// "refresh" to lock down an endpoint.
func (v *Verifier) VerifyKind(token, want string) (*Claims, error) {
	return v.verifyKind(token, want)
}

func (v *Verifier) verifyKind(token, want string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed jwt")
	}
	enc := base64.RawURLEncoding
	header, err := enc.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	payload, err := enc.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	sig, err := enc.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode sig: %w", err)
	}
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(mac.Sum(nil), sig) {
		return nil, errors.New("bad signature")
	}
	var h struct {
		Alg string `json:"alg"`
	}
	_ = json.Unmarshal(header, &h)
	if h.Alg != "HS256" {
		return nil, fmt.Errorf("unsupported alg %q", h.Alg)
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}
	if c.Exp > 0 && time.Now().UTC().Unix() >= c.Exp {
		return nil, errors.New("token expired")
	}
	// When the caller supplied an expected kind (e.g. "refresh"), reject
	// mismatches so /api/auth/refresh can't be fed an access token (or
	// vice-versa).
	if want != "" && c.Kind != "" && c.Kind != want {
		return nil, fmt.Errorf("wrong token kind: have=%q want=%q", c.Kind, want)
	}
	return &c, nil
}

// Middleware verifies the JWT and stashes the user id in the gin.Context
// under key "user_id". It does NOT enforce that the user exists in the
// registry - that is the handler's responsibility so that admin endpoints
// can still mint tokens for freshly-created users.
//
// v2 (M2) also stashes "role" (the users.role claim) and "jti" (the
// token id, used for refresh-token revocation). Handlers can read them
// via gin.Context.GetString.
//
// Optional extra verifiers (e.g. "ensure user is not disabled") are
// passed in via WithExtraCheck. The check runs after signature +
// expiry validation but before the handler, so a disabled user gets a
// clean 401 with no downstream side effects.
func (v *Verifier) Middleware(extra ...func(*Claims) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(raw, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := v.Verify(strings.TrimPrefix(raw, prefix))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		for _, check := range extra {
			if err := check(claims); err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}
		}
		c.Set("user_id", claims.Sub)
		c.Set("role", claims.Role)
		c.Set("jti", claims.Jti)
		c.Set("claims", claims)
		c.Next()
	}
}

type StaticToken struct {
	token string
}

func NewStaticToken(token string) *StaticToken {
	return &StaticToken{token: token}
}

func (s *StaticToken) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.token == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "admin token not configured"})
			return
		}
		got := c.GetHeader("X-Admin-Token")
		if got == "" {
			got = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		if subtleEqual(got, s.token) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin token"})
	}
}

func subtleEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

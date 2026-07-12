// Package auth handles authentication for two distinct audiences:
//
//   - JWT bearer tokens: end users (developers) issuing MCP requests
//   - Static admin token: operators using /admin endpoints
//
// JWTs are HS256 with a single shared secret. We do *not* require user
// registration in the claim - the user_id is derived from the URL path
// parameter (`?as=user_id`) to keep MCP clients simple. The presence of
// a valid signature is the gate; per-user allow-listing is enforced by
// the store.Registry.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Claims struct {
	Sub string `json:"sub"` // user id
	Exp int64  `json:"exp"` // unix seconds
	Iat int64  `json:"iat"`
}

type Verifier struct {
	secret []byte
}

func NewVerifier(secret []byte) *Verifier {
	return &Verifier{secret: secret}
}

func (v *Verifier) Sign(userID string, ttl time.Duration) (string, error) {
	if len(v.secret) == 0 {
		return "", errors.New("jwt secret not configured")
	}
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now().UTC()
	payload := Claims{
		Sub: userID,
		Iat: now.Unix(),
		Exp: now.Add(ttl).Unix(),
	}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(hb) + "." + enc.EncodeToString(pb)
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(signing))
	sig := enc.EncodeToString(mac.Sum(nil))
	return signing + "." + sig, nil
}

func (v *Verifier) Verify(token string) (*Claims, error) {
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
	return &c, nil
}

// Middleware verifies the JWT and stashes the user id in the gin.Context
// under key "user_id". It does NOT enforce that the user exists in the
// registry - that is the handler's responsibility so that admin endpoints
// can still mint tokens for freshly-created users.
func (v *Verifier) Middleware() gin.HandlerFunc {
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
		c.Set("user_id", claims.Sub)
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

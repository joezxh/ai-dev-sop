package console

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	CookieSession = "cbmem_console"
	CookieCSRF    = "cbmem_csrf"
	HeaderCSRF    = "X-CSRF-Token"
)

type AdminVerifier struct {
	token string
}

func NewAdminVerifier(token string) *AdminVerifier { return &AdminVerifier{token: token} }

func (v *AdminVerifier) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		got := c.GetHeader("X-Admin-Token")
		if got == "" {
			got = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		if got == "" || got != v.token {
			Fail(c, http.StatusUnauthorized, 4010001, "invalid admin token")
			c.Abort()
			return
		}
		c.Set("admin_id", "admin")
		c.Next()
	}
}

func randomID(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type SessionManager struct {
	db  *DB
	ttl time.Duration
}

func NewSessionManager(db *DB, ttl time.Duration) *SessionManager {
	return &SessionManager{db: db, ttl: ttl}
}

func (sm *SessionManager) Create(ctx context.Context, userID, ip, ua string) (string, error) {
	id, err := randomID(24)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	_, err = sm.db.ExecContext(ctx,
		`INSERT INTO console_sessions(id, user_id, created_at, expires_at, last_seen_at, ip, ua) VALUES(?,?,?,?,?,?,?)`,
		id, userID, now, now.Add(sm.ttl), now, ip, ua)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (sm *SessionManager) Lookup(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", errors.New("empty session id")
	}
	var userID string
	var expiresAt time.Time
	err := sm.db.QueryRowContext(ctx,
		`SELECT user_id, expires_at FROM console_sessions WHERE id = ?`, id).
		Scan(&userID, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("session not found")
		}
		return "", err
	}
	if time.Now().UTC().After(expiresAt) {
		return "", errors.New("session expired")
	}
	return userID, nil
}

func (sm *SessionManager) Touch(ctx context.Context, id string) error {
	_, err := sm.db.ExecContext(ctx,
		`UPDATE console_sessions SET last_seen_at = ? WHERE id = ?`, time.Now().UTC(), id)
	return err
}

func (sm *SessionManager) Drop(ctx context.Context, id string) error {
	_, err := sm.db.ExecContext(ctx, `DELETE FROM console_sessions WHERE id = ?`, id)
	return err
}

func RequireSession(sm *SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := c.Cookie(CookieSession)
		if err != nil || id == "" {
			Fail(c, http.StatusUnauthorized, 4010002, "session cookie missing")
			c.Abort()
			return
		}
		uid, err := sm.Lookup(c.Request.Context(), id)
		if err != nil {
			Fail(c, http.StatusUnauthorized, 4010003, "session expired or invalid")
			c.Abort()
			return
		}
		c.Set("admin_id", uid)
		c.Set("session_id", id)
		sm.Touch(c.Request.Context(), id)
		c.Next()
	}
}

func RequireCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}
		got := c.GetHeader(HeaderCSRF)
		want, err := c.Cookie(CookieCSRF)
		if got == "" || err != nil || got != want {
			Fail(c, http.StatusForbidden, 4030001, "csrf token mismatch")
			c.Abort()
			return
		}
		c.Next()
	}
}

package console

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const sessionMaxAgeSeconds = 8 * 3600

func LoginHandler(adminToken string, sm *SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Admin-Token")
		if token == "" {
			token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		if token == "" || token != adminToken {
			Fail(c, http.StatusUnauthorized, 4010001, "invalid admin token")
			return
		}

		sid, err := sm.Create(c.Request.Context(), "admin", c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000001, "create session")
			return
		}

		csrfBytes := make([]byte, 16)
		if _, err := rand.Read(csrfBytes); err != nil {
			Fail(c, http.StatusInternalServerError, 5000002, "mint csrf")
			return
		}
		csrf := hex.EncodeToString(csrfBytes)

		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie(CookieSession, sid, sessionMaxAgeSeconds, "/", "", false, true)
		c.SetCookie(CookieCSRF, csrf, sessionMaxAgeSeconds, "/", "", false, false)

		OK(c, gin.H{
			"admin_id":   "admin",
			"csrf_token": csrf,
			"session_id": sid,
		})
	}
}

func LogoutHandler(sm *SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if sid, err := c.Cookie(CookieSession); err == nil && sid != "" {
			_ = sm.Drop(c.Request.Context(), sid)
		}
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie(CookieSession, "", -1, "/", "", false, true)
		c.SetCookie(CookieCSRF, "", -1, "/", "", false, false)
		OK(c, nil)
	}
}

func MeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		OK(c, gin.H{"admin_id": c.GetString("admin_id")})
	}
}

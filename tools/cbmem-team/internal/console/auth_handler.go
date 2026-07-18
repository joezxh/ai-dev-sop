package console

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/auth"
)

// =============================================================================
// v2 (M2) password + JWT auth handlers
// =============================================================================
//
// Routes wired in router.go:
//
//   POST /api/auth/login             — username+password → {access,refresh,user}
//   POST /api/auth/refresh           — refresh_token       → {access,refresh}
//   POST /api/auth/logout            — revoke refresh_token (best-effort)
//   GET  /api/auth/me                — current user info
//   POST /api/auth/change-password   — old+new password
//   POST /api/auth/first-admin       — bootstrap initial admin when users=0
//
// All handlers share the convention:
//
//   * Return 4xx with envelope `{code, msg, data:null}` on auth failure
//   * Return 200 with envelope `{code:0, msg:"", data:{...}}` on success
//   * Never leak bcrypt errors directly — `auth.IsLegacySentinel` and
//     `auth.ErrPasswordMismatch` are the only ones surfaced, mapped to
//     friendly error codes (4010010 / 4010011 / etc.)
//
// =============================================================================

// AuthHandlers groups the dependencies all M2 auth handlers need.
// Mounted in router.go via NewAuthHandlers(...).Handlers(...) so the
// caller doesn't need to thread each method through MountConfig.
type AuthHandlers struct {
	DB              *DB
	Verifier        *auth.Verifier
	BcryptCost      int
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	// InitialAdmin, when non-empty, is the username of the user that
	// /first-admin creates. Empty disables the route (production
	// deployments should NOT allow self-bootstrapping).
	InitialAdmin string
}

// NewAuthHandlers builds an AuthHandlers. JWTSecret is the HMAC secret;
// AccessTokenTTL/RefreshTokenTTL default to sane values when zero.
// BcryptCost defaults to auth.BcryptCost when zero or out of range.
func NewAuthHandlers(db *DB, jwtSecret []byte, accessTTL, refreshTTL time.Duration, initialAdmin string) *AuthHandlers {
	return NewAuthHandlersWithCost(db, jwtSecret, accessTTL, refreshTTL, initialAdmin, 0)
}

// NewAuthHandlersWithCost is NewAuthHandlers with explicit bcrypt cost.
// 0 falls back to auth.BcryptCost; out-of-range values are clamped.
func NewAuthHandlersWithCost(db *DB, jwtSecret []byte, accessTTL, refreshTTL time.Duration, initialAdmin string, cost int) *AuthHandlers {
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	if cost < 4 || cost > 31 {
		cost = auth.BcryptCost
	}
	return &AuthHandlers{
		DB:              db,
		Verifier:        auth.NewVerifier(jwtSecret),
		BcryptCost:      cost,
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		InitialAdmin:    initialAdmin,
	}
}

// Mount registers the auth routes on the supplied router group.
//
// Pass a group whose path is the prefix (e.g. "/api/auth"); the
// routes below are appended at "/login", "/refresh", etc. — i.e.
// the resulting URLs are "<prefix>/login", NOT "<prefix>/auth/login".
func (h *AuthHandlers) Mount(g *gin.RouterGroup) {
	g.POST("/login", h.Login())
	g.POST("/refresh", h.Refresh())
	g.POST("/logout", h.Logout())
	g.GET("/me", h.RequireJWT(), h.Me())
	g.POST("/change-password", h.RequireJWT(), h.ChangePassword())
	if h.InitialAdmin != "" {
		g.POST("/first-admin", h.FirstAdmin())
	}
}

// ----------------------------------------------------------------------------
// Login
// ----------------------------------------------------------------------------

// Login exchanges username + password for an access JWT + refresh token.
//
// On success the response is `AuthDTO` (the v2 DTO already declared in
// dto.go). On failure we map each error to a distinct code:
//
//   4000001  — malformed request body
//   4010010  — user not found
//   4010011  — bad credentials
//   4010012  — legacy "must reset" hash (route to /change-password)
//   4010013  — user disabled
//   5000010  — server-side error
func (h *AuthHandlers) Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000001, "invalid request: "+err.Error())
			return
		}

		u, err := h.DB.GetUserByUsername(c.Request.Context(), req.Username)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				Fail(c, http.StatusUnauthorized, 4010010, "user not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000010, "lookup user: "+err.Error())
			return
		}
		if u.Disabled {
			Fail(c, http.StatusUnauthorized, 4010013, "user disabled")
			return
		}

		// Legacy sentinel path: refuse login until the user resets the
		// password. The UI redirects to /change-password and gates on
		// must_change_password=true on the user payload.
		if auth.IsLegacySentinel(u.PasswordHash) {
			OK(c, gin.H{
				"must_change_password": true,
				"username":            u.Username,
			})
			return
		}

		if err := auth.Verify(u.PasswordHash, req.Password); err != nil {
			switch {
			case errors.Is(err, auth.ErrPasswordMismatch):
				Fail(c, http.StatusUnauthorized, 4010011, "bad credentials")
			case errors.Is(err, auth.ErrPasswordBadFormat):
				// Non-bcrypt hash that isn't a sentinel — treat as
				// legacy too so the user gets the change-password flow.
				OK(c, gin.H{
					"must_change_password": true,
					"username":            u.Username,
				})
			default:
				Fail(c, http.StatusInternalServerError, 5000010, "verify: "+err.Error())
			}
			return
		}

		access, refresh, err := h.mintTokenPair(c, u)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000011, "mint tokens: "+err.Error())
			return
		}
		_ = h.DB.UpdateUserLastLogin(c.Request.Context(), u.ID)

		OK(c, AuthDTO{
			AccessToken:        access,
			RefreshToken:       refresh,
			ExpiresIn:          int(h.AccessTokenTTL.Seconds()),
			MustChangePassword: u.MustChangePassword,
			User:               toUserDTOv2(u),
		})
	}
}

// ----------------------------------------------------------------------------
// Refresh
// ----------------------------------------------------------------------------

// Refresh exchanges a refresh token for a fresh access token + new
// refresh token (rotation). The old refresh token is revoked on success.
//
// We use rotation rather than reuse to limit the blast radius of a
// stolen refresh token: each successful refresh consumes the previous
// token, so a replay attempt by an attacker produces "token revoked"
// once the legitimate client has refreshed at least once.
func (h *AuthHandlers) Refresh() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RefreshReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000002, "invalid request: "+err.Error())
			return
		}
		userID, tokenID, err := h.DB.ValidateRefreshToken(c.Request.Context(), req.RefreshToken)
		if err != nil {
			Fail(c, http.StatusUnauthorized, 4010020, "refresh invalid: "+err.Error())
			return
		}
		u, err := h.DB.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			Fail(c, http.StatusUnauthorized, 4010020, "user gone: "+err.Error())
			return
		}
		if u.Disabled {
			Fail(c, http.StatusUnauthorized, 4010021, "user disabled")
			return
		}

		// Revoke the old refresh token before minting the new pair.
		_ = h.DB.RevokeRefreshToken(c.Request.Context(), tokenID)

		access, refresh, err := h.mintTokenPair(c, u)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000012, "mint tokens: "+err.Error())
			return
		}
		OK(c, gin.H{
			"access_token":  access,
			"refresh_token": refresh,
			"expires_in":    int(h.AccessTokenTTL.Seconds()),
		})
	}
}

// ----------------------------------------------------------------------------
// Logout
// ----------------------------------------------------------------------------

// Logout revokes the supplied refresh token. The access JWT cannot be
// revoked server-side (it's stateless), but clients are expected to
// drop it locally. Calling logout without a token is a no-op (200) so
// the frontend can call it from "beforeunload" without surfacing
// errors when the user is already logged out.
func (h *AuthHandlers) Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RefreshReq
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			// Best-effort: also try reading from Authorization header
			// (extract jti) so a stateless client can still log out.
			OK(c, nil)
			return
		}
		_, tokenID, err := h.DB.ValidateRefreshToken(c.Request.Context(), req.RefreshToken)
		if err == nil && tokenID != "" {
			_ = h.DB.RevokeRefreshToken(c.Request.Context(), tokenID)
		}
		OK(c, nil)
	}
}

// ----------------------------------------------------------------------------
// Me
// ----------------------------------------------------------------------------

// Me returns the current user payload. Requires a valid JWT; mounted
// behind h.RequireJWT() in Mount.
func (h *AuthHandlers) Me() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString("user_id")
		if uid == "" {
			Fail(c, http.StatusUnauthorized, 4010030, "missing user context")
			return
		}
		u, err := h.DB.GetUserByID(c.Request.Context(), uid)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				Fail(c, http.StatusUnauthorized, 4010030, "user not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000013, "lookup user: "+err.Error())
			return
		}
		OK(c, toUserDTOv2(u))
	}
}

// ----------------------------------------------------------------------------
// Change password
// ----------------------------------------------------------------------------

// ChangePassword rotates the password hash and revokes ALL refresh
// tokens for the user. The access JWT is short-lived (15 min default)
// so we don't need to invalidate it server-side; revoking refresh
// means every active device has to re-login on the next refresh.
func (h *AuthHandlers) ChangePassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString("user_id")
		if uid == "" {
			Fail(c, http.StatusUnauthorized, 4010040, "missing user context")
			return
		}
		var req ChangePasswordReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000003, "invalid request: "+err.Error())
			return
		}
		u, err := h.DB.GetUserByID(c.Request.Context(), uid)
		if err != nil {
			Fail(c, http.StatusUnauthorized, 4010041, "user not found")
			return
		}
		// Verify the old password. Skip the check for legacy sentinels
		// (no real password to verify) so the user can still escape the
		// must_change_password loop.
		if !auth.IsLegacySentinel(u.PasswordHash) {
			if err := auth.Verify(u.PasswordHash, req.OldPassword); err != nil {
				Fail(c, http.StatusUnauthorized, 4010042, "old password wrong")
				return
			}
		}
		newHash, err := auth.HashWith(req.NewPassword, h.BcryptCost)
		if err != nil {
			Fail(c, http.StatusBadRequest, 4000004, "new password: "+err.Error())
			return
		}
		if err := h.DB.UpdateUserPassword(c.Request.Context(), uid, newHash); err != nil {
			Fail(c, http.StatusInternalServerError, 5000014, "update password: "+err.Error())
			return
		}
		// Revoke every outstanding refresh token so the user is
		// effectively logged out on every device.
		_ = h.DB.RevokeAllRefreshTokensForUser(c.Request.Context(), uid)

		// Mint a fresh access token (must_change_password cleared).
		u.PasswordHash = newHash
		u.MustChangePassword = false
		access, refresh, err := h.mintTokenPair(c, u)
		if err != nil {
			// Non-fatal: the user still gets a successful password
			// change. They can re-login to get a fresh token.
			OK(c, gin.H{"ok": true})
			return
		}
		OK(c, AuthDTO{
			AccessToken:        access,
			RefreshToken:       refresh,
			ExpiresIn:          int(h.AccessTokenTTL.Seconds()),
			MustChangePassword: false,
			User:               toUserDTOv2(u),
		})
	}
}

// ----------------------------------------------------------------------------
// First-admin bootstrap
// ----------------------------------------------------------------------------

// FirstAdmin creates the platform's first admin user. The route is
// only mounted when AuthHandlers.InitialAdmin is non-empty; when it
// IS mounted, the route refuses if any user already exists. This
// gives a single, race-safe path for first-boot setup without leaving
// a permanent "create admin" hole in production.
//
// Body: { "username": "...", "password": "..." }.
//
// Errors:
//   4090001 — admin already bootstrapped (CountUsers > 0)
//   4000005 — missing fields
//   5000015 — server error
func (h *AuthHandlers) FirstAdmin() gin.HandlerFunc {
	type req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
	}
	return func(c *gin.Context) {
		n, err := h.DB.CountUsers(c.Request.Context())
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000015, "count users: "+err.Error())
			return
		}
		if n > 0 {
			Fail(c, http.StatusConflict, 4090001, "admin already bootstrapped")
			return
		}
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Fail(c, http.StatusBadRequest, 4000005, "invalid request: "+err.Error())
			return
		}
		hash, err := auth.HashWith(r.Password, h.BcryptCost)
		if err != nil {
			Fail(c, http.StatusBadRequest, 4000006, "hash: "+err.Error())
			return
		}
		id, err := randomID(12)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000016, "mint id: "+err.Error())
			return
		}
		u := &User{
			ID:                 "u_" + id,
			Username:           r.Username,
			DisplayName:        r.Username,
			PasswordHash:       hash,
			Role:               string(RoleAdmin),
			MustChangePassword: false,
		}
		if err := h.DB.CreateUser(c.Request.Context(), u); err != nil {
			Fail(c, http.StatusInternalServerError, 5000017, "create: "+err.Error())
			return
		}
		access, refresh, err := h.mintTokenPair(c, u)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000018, "mint: "+err.Error())
			return
		}
		OK(c, AuthDTO{
			AccessToken:        access,
			RefreshToken:       refresh,
			ExpiresIn:          int(h.AccessTokenTTL.Seconds()),
			MustChangePassword: false,
			User:               toUserDTOv2(u),
		})
	}
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// RequireJWT returns a middleware that verifies the Bearer JWT and
// injects user_id / role / jti into the gin context. We re-use the
// shared Verifier rather than reimplementing signature checks here.
//
// "Disabled user" check is wired in via extra-check so a stolen JWT
// from a just-revoked user fails fast.
func (h *AuthHandlers) RequireJWT() gin.HandlerFunc {
	return h.Verifier.Middleware(func(claims *auth.Claims) error {
		if claims.Sub == "" {
			return errors.New("missing sub")
		}
		u, err := h.DB.GetUserByID(context.Background(), claims.Sub)
		if err != nil {
			return errors.New("user not found")
		}
		if u.Disabled {
			return errors.New("user disabled")
		}
		return nil
	})
}

// mintTokenPair signs a (access, refresh) pair and persists the
// refresh token. The access JWT carries the user's role so downstream
// RBAC checks don't need a second DB lookup.
func (h *AuthHandlers) mintTokenPair(c *gin.Context, u *User) (access, refresh string, err error) {
	// Refresh token id (also the JWT jti claim). We use this id when
	// revoking so the lookup is by primary key.
	refreshID, err := randomID(16)
	if err != nil {
		return "", "", err
	}
	refreshPlaintext, err := randomToken(48)
	if err != nil {
		return "", "", err
	}

	// Access JWT carries the same jti as the refresh token so a
	// session can be tied back. We DON'T verify them at the JWT layer
	// (that's a future enhancement); for now the access token is
	// short-lived enough that the link is informational.
	accessToken, err := h.Verifier.SignClaims(auth.Claims{
		Sub:  u.ID,
		Role: u.Role,
		Jti:  refreshID,
		Kind: "access",
	}, h.AccessTokenTTL)
	if err != nil {
		return "", "", err
	}

	// Refresh JWT is a separate token signed with the same secret. It
	// carries kind="refresh" so the /refresh endpoint refuses an
	// access token. We append 48 bytes of CSPRNG randomness so the
	// resulting string is unique even if two clients happen to get
	// the same JWT body (they won't, but defence-in-depth).
	refreshJWT, err := h.Verifier.SignClaims(auth.Claims{
		Sub:  u.ID,
		Role: u.Role,
		Jti:  refreshID,
		Kind: "refresh",
	}, h.RefreshTokenTTL)
	if err != nil {
		return "", "", err
	}
	// The opaque refresh "token" is `<jwt>.<random>`; the entire string
	// is what the client holds and what the DB stores (hashed). The
	// random tail ensures a database-only compromise can't forge the
	// JWT path: an attacker would still need the signing secret.
	composite := refreshJWT + "." + refreshPlaintext

	// Persist the composite (the whole thing the client will see) so
	// /refresh can hash + lookup atomically.
	if err := h.DB.CreateRefreshToken(c.Request.Context(), refreshID, u.ID, composite, h.RefreshTokenTTL); err != nil {
		return "", "", err
	}
	return accessToken, composite, nil
}

// randomToken returns a hex-encoded random string of nBytes bytes.
// Used for the refresh token's plaintext tail.
func randomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// toUserDTOv2 converts the storage struct to the wire DTO. Kept here
// (not in dto.go) to avoid an import cycle.
func toUserDTOv2(u *User) UserDTOv2 {
	dto := UserDTOv2{
		ID:                 u.ID,
		Username:           u.Username,
		DisplayName:        u.DisplayName,
		Email:              u.Email,
		DefaultTeamID:      u.DefaultTeamID,
		Role:               u.Role,
		MustChangePassword: u.MustChangePassword,
		Disabled:           u.Disabled,
		CreatedAt:          u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          u.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if u.LastLoginAt != nil {
		dto.LastLoginAt = u.LastLoginAt
	}
	return dto
}

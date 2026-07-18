package console

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/auth"
)

// authTestHarness wires up an in-memory-ish test rig: a sqlite DB, an
// AuthHandlers, and a gin router with the auth routes mounted. Tests
// reuse it via newAuthHarness(t).
type authTestHarness struct {
	DB       *DB
	Handlers *AuthHandlers
	Router   *gin.Engine
	Secret   []byte
}

func newAuthHarness(t *testing.T, withInitialAdmin bool) *authTestHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "authtest.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("v1 migrate: %v", err)
	}
	if err := db.MigrateV2(context.Background(), nil); err != nil {
		t.Fatalf("v2 migrate: %v", err)
	}

	secret := []byte("test-secret-32bytes-test-secret-32")
	handlers := NewAuthHandlersWithCost(
		db, secret,
		15*time.Minute, time.Hour,
		"admin", 4, // cost=4 for fast tests
	)
	r := gin.New()
	r.Use(gin.Recovery())
	g := r.Group("/api/auth")
	if withInitialAdmin {
		handlers.Mount(g)
	} else {
		// Disable InitialAdmin to test the production-mode behaviour.
		handlers.InitialAdmin = ""
		handlers.Mount(g)
	}

	return &authTestHarness{
		DB: db, Handlers: handlers, Router: r, Secret: secret,
	}
}

// doJSON performs a JSON request and decodes the response envelope.
// Optional `headers` map lets callers attach Authorization etc.
func (h *authTestHarness) doJSON(t *testing.T, method, path string, body any, headers ...map[string]string) (int, envelope, http.Header) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	for _, hdr := range headers {
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
	}
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)

	var env envelope
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &env)
	}
	return w.Code, env, w.Header()
}

// envelope mirrors wrap.go's type (same package, no need to redefine).
// We use it as the response decoder so tests can assert on code/msg/data.

// =============================================================================
// First-admin bootstrap
// =============================================================================

func TestAuthFirstAdminSuccess(t *testing.T) {
	h := newAuthHarness(t, true)

	code, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/first-admin",
		map[string]string{"username": "root", "password": "supersecret123"})

	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%+v)", code, env)
	}
	if env.Code != 0 {
		t.Fatalf("envelope code = %d, want 0", env.Code)
	}
	data, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("data not a map: %T", env.Data)
	}
	if data["access_token"] == nil || data["refresh_token"] == nil {
		t.Fatalf("expected access+refresh tokens, got %+v", data)
	}
}

func TestAuthFirstAdminRefusesWhenUserExists(t *testing.T) {
	h := newAuthHarness(t, true)

	// Create a user first so CountUsers > 0.
	hash, err := auth.HashWith("seedpass1234", 4)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := h.DB.CreateUser(context.Background(), &User{
		ID: "u_seed", Username: "seed", PasswordHash: hash, Role: string(RoleAdmin),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	code, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/first-admin",
		map[string]string{"username": "root", "password": "supersecret123"})

	if code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", code)
	}
	if env.Code != 4090001 {
		t.Fatalf("envelope code = %d, want 4090001", env.Code)
	}
}

// =============================================================================
// Login
// =============================================================================

func TestAuthLoginSuccess(t *testing.T) {
	h := newAuthHarness(t, true)
	// Seed an admin via first-admin.
	if _, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/first-admin",
		map[string]string{"username": "alice", "password": "alicepass123"}); env.Code != 0 {
		t.Fatalf("first-admin failed: %+v", env)
	}

	// Now login.
	code, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/login",
		map[string]string{"username": "alice", "password": "alicepass123"})

	if code != http.StatusOK {
		t.Fatalf("status = %d (env=%+v)", code, env)
	}
	data, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("data not map: %T", env.Data)
	}
	access, _ := data["access_token"].(string)
	refresh, _ := data["refresh_token"].(string)
	if access == "" || refresh == "" {
		t.Fatalf("missing tokens: %+v", data)
	}
	// Refresh should be a composite (jwt.tail).
	if !strings.Contains(refresh, ".") {
		t.Errorf("refresh token missing composite structure: %q", refresh)
	}

	// Verify the access token round-trips and carries the role claim.
	claims, err := auth.NewVerifier(h.Secret).Verify(access)
	if err != nil {
		t.Fatalf("verify access: %v", err)
	}
	if claims.Sub == "" {
		t.Errorf("claims.sub empty")
	}
	if claims.Role != string(RoleAdmin) {
		t.Errorf("claims.role = %q, want admin", claims.Role)
	}
}

func TestAuthLoginWrongPassword(t *testing.T) {
	h := newAuthHarness(t, true)
	if _, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/first-admin",
		map[string]string{"username": "alice", "password": "alicepass123"}); env.Code != 0 {
		t.Fatalf("first-admin: %+v", env)
	}

	code, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/login",
		map[string]string{"username": "alice", "password": "wrongpass"})

	if code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", code)
	}
	if env.Code != 4010011 {
		t.Fatalf("code = %d, want 4010011", env.Code)
	}
}

func TestAuthLoginUnknownUser(t *testing.T) {
	h := newAuthHarness(t, true)

	code, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/login",
		map[string]string{"username": "ghost", "password": "doesntmatter8"})

	if code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", code)
	}
	if env.Code != 4010010 {
		t.Fatalf("code = %d, want 4010010", env.Code)
	}
}

func TestAuthLoginDisabledUser(t *testing.T) {
	h := newAuthHarness(t, true)
	hash, _ := auth.HashWith("alicepass123", 4)
	if err := h.DB.CreateUser(context.Background(), &User{
		ID: "u_disabled", Username: "off", PasswordHash: hash,
		Role: string(RoleDeveloper), Disabled: true,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	code, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/login",
		map[string]string{"username": "off", "password": "alicepass123"})

	if code != http.StatusUnauthorized || env.Code != 4010013 {
		t.Fatalf("expected 401/4010013, got %d/%d", code, env.Code)
	}
}

func TestAuthLoginLegacySentinelRefused(t *testing.T) {
	h := newAuthHarness(t, false)
	// Manually insert a v1-migrated user with the sentinel hash.
	if err := h.DB.CreateUser(context.Background(), &User{
		ID: "u_legacy", Username: "legacy", PasswordHash: "!v1-legacy-must-reset!",
		Role: string(RoleDeveloper),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	code, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/login",
		map[string]string{"username": "legacy", "password": "anything"})

	// The legacy path returns 200 with must_change_password=true so the
	// UI knows to redirect to /change-password.
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	data, _ := env.Data.(map[string]any)
	if got, _ := data["must_change_password"].(bool); !got {
		t.Fatalf("expected must_change_password=true, got %+v", data)
	}
}

// =============================================================================
// Refresh rotation + revocation
// =============================================================================

func TestAuthRefreshRotationRevokesOld(t *testing.T) {
	h := newAuthHarness(t, true)
	_, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/first-admin",
		map[string]string{"username": "alice", "password": "alicepass123"})
	if env.Code != 0 {
		t.Fatalf("first-admin: %+v", env)
	}
	_, env, _ = h.doJSON(t, http.MethodPost, "/api/auth/login",
		map[string]string{"username": "alice", "password": "alicepass123"})
	oldRefresh, _ := env.Data.(map[string]any)["refresh_token"].(string)
	if oldRefresh == "" {
		t.Fatalf("no refresh on login: %+v", env)
	}

	// First refresh: should succeed and yield a new token.
	code, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/refresh",
		map[string]string{"refresh_token": oldRefresh})
	if code != http.StatusOK {
		t.Fatalf("first refresh failed: %d %+v", code, env)
	}
	data, _ := env.Data.(map[string]any)
	newRefresh, _ := data["refresh_token"].(string)
	if newRefresh == "" || newRefresh == oldRefresh {
		t.Fatalf("expected rotated refresh, got same=%v empty=%v",
			newRefresh == oldRefresh, newRefresh == "")
	}

	// Second refresh with the OLD token should now fail (revoked).
	code, env, _ = h.doJSON(t, http.MethodPost, "/api/auth/refresh",
		map[string]string{"refresh_token": oldRefresh})
	if code != http.StatusUnauthorized {
		t.Fatalf("replay attack succeeded! status=%d env=%+v", code, env)
	}
}

// =============================================================================
// Me + ChangePassword
// =============================================================================

func TestAuthMeAndChangePassword(t *testing.T) {
	h := newAuthHarness(t, true)
	_, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/first-admin",
		map[string]string{"username": "alice", "password": "alicepass123"})
	if env.Code != 0 {
		t.Fatalf("first-admin: %+v", env)
	}
	_, env, _ = h.doJSON(t, http.MethodPost, "/api/auth/login",
		map[string]string{"username": "alice", "password": "alicepass123"})
	access, _ := env.Data.(map[string]any)["access_token"].(string)

	// /me requires Authorization header.
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("/me status = %d", w.Code)
	}

	// Change password.
	code, env, _ := h.doJSON(t, http.MethodPost, "/api/auth/change-password",
		map[string]string{"old_password": "alicepass123", "new_password": "alicepass456"},
		map[string]string{"Authorization": "Bearer " + access})
	if code != http.StatusOK {
		t.Fatalf("change-password status = %d env=%+v", code, env)
	}

	// Old password should no longer work.
	code, env, _ = h.doJSON(t, http.MethodPost, "/api/auth/login",
		map[string]string{"username": "alice", "password": "alicepass123"})
	if code != http.StatusUnauthorized {
		t.Fatalf("expected old password rejected, got %d/%d", code, env.Code)
	}

	// New password works.
	code, env, _ = h.doJSON(t, http.MethodPost, "/api/auth/login",
		map[string]string{"username": "alice", "password": "alicepass456"})
	if code != http.StatusOK {
		t.Fatalf("expected new password accepted, got %d/%+v", code, env)
	}
}

// =============================================================================
// DB CRUD round-trips
// =============================================================================

func TestDBUserCRUDRoundTrip(t *testing.T) {
	h := newAuthHarness(t, false)
	hash, _ := auth.HashWith("mypassword123", 4)
	u := &User{
		ID: "u_test", Username: "tester", DisplayName: "Test User",
		Email: "t@example.com", PasswordHash: hash,
		Role: string(RoleDeveloper), DefaultTeamID: "team_x",
	}
	if err := h.DB.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := h.DB.GetUserByID(context.Background(), "u_test")
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Username != "tester" || got.Email != "t@example.com" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}

	got2, err := h.DB.GetUserByUsername(context.Background(), "TESTER") // case-insensitive
	if err != nil {
		t.Fatalf("get by username: %v", err)
	}
	if got2.ID != "u_test" {
		t.Errorf("case-insensitive lookup failed: %+v", got2)
	}

	if err := h.DB.UpdateUserPassword(context.Background(), "u_test", "$2a$10$abcdef"); err != nil {
		t.Fatalf("update password: %v", err)
	}
	got3, _ := h.DB.GetUserByID(context.Background(), "u_test")
	if got3.PasswordHash != "$2a$10$abcdef" {
		t.Errorf("password update failed: %q", got3.PasswordHash)
	}
	if got3.MustChangePassword {
		t.Errorf("update should clear must_change_password")
	}

	if err := h.DB.SetUserDisabled(context.Background(), "u_test", true); err != nil {
		t.Fatalf("disable: %v", err)
	}
	got4, _ := h.DB.GetUserByID(context.Background(), "u_test")
	if !got4.Disabled {
		t.Errorf("disable failed")
	}
}

func TestDBRefreshTokenLifecycle(t *testing.T) {
	h := newAuthHarness(t, false)
	// The FK on refresh_tokens.user_id requires a real users row, so
	// seed one first.
	hash, _ := auth.HashWith("seedseed1234", 4)
	if err := h.DB.CreateUser(context.Background(), &User{
		ID: "u_test", Username: "rtseed", PasswordHash: hash,
		Role: string(RoleDeveloper),
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	plain := "abcdefghijklmnop"
	if err := h.DB.CreateRefreshToken(context.Background(),
		"rt_1", "u_test", plain, time.Hour); err != nil {
		t.Fatalf("create refresh: %v", err)
	}

	uid, rtid, err := h.DB.ValidateRefreshToken(context.Background(), plain)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if uid != "u_test" || rtid != "rt_1" {
		t.Errorf("got uid=%q rtid=%q", uid, rtid)
	}

	// Revoke + verify refusal.
	if err := h.DB.RevokeRefreshToken(context.Background(), "rt_1"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, _, err := h.DB.ValidateRefreshToken(context.Background(), plain); err == nil {
		t.Errorf("revoked token still validated")
	}

	// Revoke-all sweeps every token for a user.
	if err := h.DB.CreateRefreshToken(context.Background(),
		"rt_2", "u_test", "second", time.Hour); err != nil {
		t.Fatalf("create rt_2: %v", err)
	}
	if err := h.DB.RevokeAllRefreshTokensForUser(context.Background(), "u_test"); err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	if _, _, err := h.DB.ValidateRefreshToken(context.Background(), "second"); err == nil {
		t.Errorf("rt_2 still valid after revoke-all")
	}
}

package console

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/auth"
)

// =============================================================================
// HTTP-layer integration tests for M3 teams + members endpoints.
//
// Mirrors the harness in auth_handler_test.go so handlers can be
// exercised end-to-end through the Gin router with real DB / JWT
// plumbing. Project-v2 routes live in projects_v2_handler_test.go.
// =============================================================================

// teamTestHarness is the test environment. Tests build a router with
// the JWT verifier + Teams handlers mounted under /api/v2.
type teamTestHarness struct {
	DB     *DB
	Router *gin.Engine
	Secret []byte

	// Mint a JWT for `userID` with `role`. Returns the signed token.
	MintToken func(userID, role string) string
}

func newTeamTestHarness(t *testing.T) *teamTestHarness {
	t.Helper()
	db := setupTestDB(t)
	secret := []byte("test-secret-32-bytes-padding!!")
	verifier := auth.NewVerifier(secret)
	teams := NewTeamHandlers(db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	v2 := r.Group("/api/v2")
	v2.Use(verifier.Middleware())
	v2.GET("/teams", teams.ListTeams())
	v2.POST("/teams", teams.CreateTeam())
	v2.GET("/teams/:id", teams.GetTeam())
	v2.PUT("/teams/:id", teams.UpdateTeam())
	v2.DELETE("/teams/:id", teams.DeleteTeam())

	v2.GET("/teams/:id/members", teams.ListMembers())
	v2.POST("/teams/:id/members", teams.AddMember())
	v2.PUT("/teams/:id/members/:uid", teams.UpdateMember())
	v2.DELETE("/teams/:id/members/:uid", teams.RemoveMember())

	h := &teamTestHarness{
		DB:     db,
		Router: r,
		Secret: secret,
	}
	h.MintToken = func(userID, role string) string {
		signer := auth.NewVerifier(secret)
		tok, err := signer.SignClaims(auth.Claims{Sub: userID, Role: role, Kind: "access"}, 15*time.Minute)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		return tok
	}
	return h
}

// seedUser creates a user with a real password hash and the given
// role. Used so `teams.owner_id` / team_members FKs can resolve.
func (h *teamTestHarness) seedUser(t *testing.T, id, role string) *User {
	t.Helper()
	hash, err := auth.Hash("password123")
	if err != nil {
		t.Fatal(err)
	}
	u := &User{
		ID:           id,
		Username:     "user_" + id,
		PasswordHash: hash,
		Role:         role,
	}
	if err := h.DB.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func doJSONHarness(t *testing.T, h *teamTestHarness, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != "" {
		buf = bytes.NewBufferString(body)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)
	return w
}

func decodeEnvelope(t *testing.T, body []byte) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, string(body))
	}
	return env
}

func TestTeamsCreateAndList(t *testing.T) {
	h := newTeamTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	tok := h.MintToken("u_admin", string(RoleAdmin))

	// Create
	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams", tok,
		`{"name":"Alpha","slug":"alpha","description":"first team"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create: status=%d body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	id, _ := env.Data.(map[string]interface{})["id"].(string)
	if !strings.HasPrefix(id, "team_") {
		t.Fatalf("id=%q", id)
	}

	// List
	w = doJSONHarness(t, h, http.MethodGet, "/api/v2/teams", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestTeamsCreateForbiddenForDeveloper(t *testing.T) {
	h := newTeamTestHarness(t)
	h.seedUser(t, "u_dev", string(RoleDeveloper))
	tok := h.MintToken("u_dev", string(RoleDeveloper))

	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams", tok,
		`{"name":"X","slug":"x"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTeamsCreateSlugConflict(t *testing.T) {
	h := newTeamTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	tok := h.MintToken("u_admin", string(RoleAdmin))

	body := `{"name":"A","slug":"same"}`
	if w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams", tok, body); w.Code != http.StatusOK {
		t.Fatalf("first: %d %s", w.Code, w.Body.String())
	}
	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams", tok, body)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTeamsAddMemberAndUpdateAndRemove(t *testing.T) {
	h := newTeamTestHarness(t)
	admin := h.seedUser(t, "u_admin", string(RoleAdmin))
	bob := h.seedUser(t, "u_bob", string(RoleDeveloper))
	tok := h.MintToken(admin.ID, admin.Role)

	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams", tok,
		`{"name":"Alpha","slug":"alpha","owner_id":"u_admin"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create team: %d %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	teamID, _ := env.Data.(map[string]interface{})["id"].(string)

	// Add bob as developer.
	w = doJSONHarness(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/members", tok,
		`{"user_id":"u_bob","role":"developer"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("add member: %d %s", w.Code, w.Body.String())
	}
	_ = bob

	// Update bob to maintainer.
	w = doJSONHarness(t, h, http.MethodPut, "/api/v2/teams/"+teamID+"/members/u_bob", tok,
		`{"role":"maintainer"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update member: %d %s", w.Code, w.Body.String())
	}

	// Bob should be visible in the listing.
	w = doJSONHarness(t, h, http.MethodGet, "/api/v2/teams/"+teamID+"/members", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list members: %d %s", w.Code, w.Body.String())
	}
	env = decodeEnvelope(t, w.Body.Bytes())
	list, _ := env.Data.(map[string]interface{})["list"].([]interface{})
	if len(list) < 2 {
		t.Fatalf("expected ≥2 members, got %d", len(list))
	}

	// Remove bob.
	w = doJSONHarness(t, h, http.MethodDelete, "/api/v2/teams/"+teamID+"/members/u_bob", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("remove: %d %s", w.Code, w.Body.String())
	}
}

func TestTeamsListForNonMember(t *testing.T) {
	h := newTeamTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	h.seedUser(t, "u_dev", string(RoleDeveloper))
	adminTok := h.MintToken("u_admin", string(RoleAdmin))
	devTok := h.MintToken("u_dev", string(RoleDeveloper))

	// Admin creates two teams; dev gets added to one.
	for i, slug := range []string{"first", "second"} {
		w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams", adminTok,
			`{"name":"T`+string(rune('A'+i))+`","slug":"`+slug+`"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("create %s: %d %s", slug, w.Code, w.Body.String())
		}
	}
	// Add dev to first team.
	w := doJSONHarness(t, h, http.MethodGet, "/api/v2/teams", adminTok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list teams: %d %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	list := env.Data.(map[string]interface{})["list"].([]interface{})
	firstTeamID, _ := list[0].(map[string]interface{})["id"].(string)
	w = doJSONHarness(t, h, http.MethodPost, "/api/v2/teams/"+firstTeamID+"/members", adminTok,
		`{"user_id":"u_dev","role":"developer"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("add dev: %d %s", w.Code, w.Body.String())
	}

	// Dev's list should be filtered to the team they're in (and the
	// team they own — they own none here, so exactly 1 team).
	w = doJSONHarness(t, h, http.MethodGet, "/api/v2/teams", devTok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("dev list: %d %s", w.Code, w.Body.String())
	}
	env = decodeEnvelope(t, w.Body.Bytes())
	visList := env.Data.(map[string]interface{})["list"].([]interface{})
	if len(visList) != 1 {
		t.Fatalf("dev should see 1 team, got %d body=%s", len(visList), w.Body.String())
	}
}
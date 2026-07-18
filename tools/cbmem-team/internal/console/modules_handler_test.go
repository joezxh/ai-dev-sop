package console

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/auth"
)

// =============================================================================
// HTTP-layer integration tests for M4 module endpoints.
//
// Reuses the team harness shape from M3 (teams_handler_test.go) but
// extends it with projects + modules so the project-scoped routes
// (`/api/v2/projects/:id/modules`) and the global module routes
// (`/api/v2/modules/:id`) are both reachable through real auth.
// =============================================================================

type modulesTestHarness struct {
	DB     *DB
	Router *gin.Engine
	Secret []byte

	MintToken func(userID, role string) string
}

func newModulesTestHarness(t *testing.T) *modulesTestHarness {
	t.Helper()
	db := setupTestDB(t)
	secret := []byte("test-secret-32-bytes-padding!!")
	verifier := auth.NewVerifier(secret)

	teams := NewTeamHandlers(db)
	projects := NewProjectHandlers(db, t.TempDir(), "codebase-memory-mcp")
	modules := NewModuleHandlers(db)
	sessionsV2 := NewSessionHandlersV2(db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	v2 := r.Group("/api/v2")
	v2.Use(verifier.Middleware())

	// M3 routes required so the project write access checks can
	// find a real team/project.
	v2.POST("/teams", teams.CreateTeam())
	v2.GET("/teams/:id", teams.GetTeam())

	v2.POST("/teams/:id/projects", projects.CreateProject())
	v2.GET("/projects/:pid", projects.GetProject())

	// M4 routes under test.
	v2.GET("/projects/:pid/modules", modules.ListModules())
	v2.POST("/projects/:pid/modules", modules.CreateModule())
	v2.GET("/projects/:pid/modules/tree", modules.Tree())
	v2.GET("/modules/:id", modules.GetModule())
	v2.PUT("/modules/:id", modules.UpdateModule())
	v2.DELETE("/modules/:id", modules.DeleteModule())
	v2.POST("/modules/:id/move", modules.MoveModule())

	// M4 sessions v2 list is also reachable so we can verify the
	// project-scoped filter end-to-end.
	v2.GET("/sessions", sessionsV2.ListSessions())
	v2.GET("/sessions/stats", sessionsV2.Stats())
	v2.GET("/sessions/:id", sessionsV2.GetSession())
	v2.GET("/projects/:pid/sessions", sessionsV2.ProjectSessions())

	h := &modulesTestHarness{DB: db, Router: r, Secret: secret}
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

// seedUser inserts a user with a real password hash.
func (h *modulesTestHarness) seedUser(t *testing.T, id, role string) {
	t.Helper()
	hash, err := auth.Hash("password123")
	if err != nil {
		t.Fatal(err)
	}
	u := &User{ID: id, Username: "user_" + id, PasswordHash: hash, Role: role}
	if err := h.DB.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

// seedTeamProject creates a team + project so the modules routes
// can resolve the project → team linkage required for RBAC.
func (h *modulesTestHarness) seedTeamProject(t *testing.T, teamSlug, projectSlug, ownerID string) (teamID, projectID string) {
	t.Helper()
	body := fmt.Sprintf(`{"name":"Team %s","slug":"%s"}`, teamSlug, teamSlug)
	w := doJSONHarness2(t, h, http.MethodPost, "/api/v2/teams", h.MintToken(ownerID, "admin"), body)
	if w.Code != http.StatusOK {
		t.Fatalf("create team: %d %s", w.Code, w.Body.String())
	}
	var env envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	teamID = string(env.Data.(map[string]any)["id"].(string))

	body = fmt.Sprintf(`{"name":"%s","slug":"%s"}`, projectSlug, projectSlug)
	w = doJSONHarness2(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", h.MintToken(ownerID, "admin"), body)
	if w.Code != http.StatusOK {
		t.Fatalf("create project: %d %s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	projectID = string(env.Data.(map[string]any)["id"].(string))
	return teamID, projectID
}

// doJSONHarness2 mirrors teams_handler_test.go's helper but works
// against modulesTestHarness so we don't have to expose the
// generic one across packages.
func doJSONHarness2(t *testing.T, h *modulesTestHarness, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
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

// =============================================================================
// Tests
// =============================================================================

func TestModulesCreateListTreeMoveDelete(t *testing.T) {
	h := newModulesTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	tok := h.MintToken("u_admin", string(RoleAdmin))
	_, projectID := h.seedTeamProject(t, "alpha", "p1", "u_admin")

	// Create root module.
	w := doJSONHarness2(t, h, http.MethodPost,
		"/api/v2/projects/"+projectID+"/modules", tok,
		`{"name":"frontend"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create root: %d %s", w.Code, w.Body.String())
	}
	var env envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	root := env.Data.(map[string]any)
	rootID := root["id"].(string)
	if root["is_leaf"].(bool) != true {
		t.Fatalf("expected root to be leaf initially")
	}

	// Create child under it.
	w = doJSONHarness2(t, h, http.MethodPost,
		"/api/v2/projects/"+projectID+"/modules", tok,
		fmt.Sprintf(`{"parent_id":"%s","name":"login-button"}`, rootID))
	if w.Code != http.StatusOK {
		t.Fatalf("create child: %d %s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	childID := env.Data.(map[string]any)["id"].(string)

	// List.
	w = doJSONHarness2(t, h, http.MethodGet,
		"/api/v2/projects/"+projectID+"/modules", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	list := env.Data.(map[string]any)["list"].([]any)
	if len(list) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(list))
	}

	// Tree.
	w = doJSONHarness2(t, h, http.MethodGet,
		"/api/v2/projects/"+projectID+"/modules/tree", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("tree: %d %s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	tree := env.Data.(map[string]any)["list"].([]any)
	if len(tree) != 1 {
		t.Fatalf("expected 1 root in tree, got %d", len(tree))
	}
	rootDTO := tree[0].(map[string]any)
	if rootDTO["is_leaf"].(bool) {
		t.Fatalf("root should be internal in tree view")
	}
	kids := rootDTO["children"].([]any)
	if len(kids) != 1 {
		t.Fatalf("root should have 1 child, got %d", len(kids))
	}

	// Move child up (promote to root).
	w = doJSONHarness2(t, h, http.MethodPost,
		"/api/v2/modules/"+childID+"/move", tok,
		`{"new_parent_id":null}`)
	if w.Code != http.StatusOK {
		t.Fatalf("move: %d %s", w.Code, w.Body.String())
	}

	// Create another module under child to make child an internal
	// node again, then verify cycle: trying to move child under its
	// own grandchild must be rejected.
	w = doJSONHarness2(t, h, http.MethodPost,
		"/api/v2/projects/"+projectID+"/modules", tok,
		fmt.Sprintf(`{"parent_id":"%s","name":"grandchild"}`, childID))
	if w.Code != http.StatusOK {
		t.Fatalf("create grandchild: %d %s", w.Code, w.Body.String())
	}
	var env2 envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env2)
	grandchildID := env2.Data.(map[string]any)["id"].(string)

	w = doJSONHarness2(t, h, http.MethodPost,
		"/api/v2/modules/"+childID+"/move", tok,
		fmt.Sprintf(`{"new_parent_id":"%s"}`, grandchildID))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 cycle (move child under grandchild), got %d body=%s",
			w.Code, w.Body.String())
	}

	// Delete root without cascade should succeed (it has no children now).
	w = doJSONHarness2(t, h, http.MethodDelete,
		"/api/v2/modules/"+rootID, tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("delete leaf root: %d %s", w.Code, w.Body.String())
	}
}

func TestModulesDeveloperCannotCreate(t *testing.T) {
	h := newModulesTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	h.seedUser(t, "u_dev", string(RoleDeveloper))
	_, projectID := h.seedTeamProject(t, "t", "p", "u_admin")

	devTok := h.MintToken("u_dev", string(RoleDeveloper))
	w := doJSONHarness2(t, h, http.MethodPost,
		"/api/v2/projects/"+projectID+"/modules", devTok,
		`{"name":"x"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	// Admin should succeed.
	adminTok := h.MintToken("u_admin", string(RoleAdmin))
	w = doJSONHarness2(t, h, http.MethodPost,
		"/api/v2/projects/"+projectID+"/modules", adminTok,
		`{"name":"admin-module"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("admin create: %d %s", w.Code, w.Body.String())
	}
}

func TestModulesRequiresJWT(t *testing.T) {
	h := newModulesTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	_, projectID := h.seedTeamProject(t, "noauth", "p", "u_admin")
	w := doJSONHarness2(t, h, http.MethodGet,
		"/api/v2/projects/"+projectID+"/modules", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestSessionsV2ListFiltersAndStats(t *testing.T) {
	h := newModulesTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	tok := h.MintToken("u_admin", string(RoleAdmin))
	_, projectID := h.seedTeamProject(t, "stats", "p", "u_admin")

	// Seed two sessions directly so we don't depend on the capture
	// middleware's async write path. Both belong to the same
	// project, one with a known module_id so the module filter is
	// testable.
	if _, err := h.DB.ExecContext(context.Background(),
		`INSERT INTO sessions (id, user_id, team_id, project_id, module_id, project_path, started_at, tool_count, turn_count)
         VALUES ('s1','u_admin','team_stats','`+projectID+`','mod_x','/path1','2026-07-15 10:00:00',3,2)`); err != nil {
		t.Fatalf("seed s1: %v", err)
	}
	if _, err := h.DB.ExecContext(context.Background(),
		`INSERT INTO sessions (id, user_id, team_id, project_id, module_id, project_path, started_at, tool_count, turn_count)
         VALUES ('s2','u_admin','team_stats','`+projectID+`','','/path2','2026-07-15 11:00:00',5,3)`); err != nil {
		t.Fatalf("seed s2: %v", err)
	}

	// Filter by project_id, include_legacy=false → 1 result.
	w := doJSONHarness2(t, h, http.MethodGet,
		"/api/v2/sessions?project_id="+projectID+"&include_legacy=false", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	var env envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	page := env.Data.(map[string]any)
	if int(page["total"].(float64)) != 1 {
		t.Fatalf("expected 1 (exclude legacy), got %v", page["total"])
	}

	// Include legacy (default) → 2.
	w = doJSONHarness2(t, h, http.MethodGet,
		"/api/v2/sessions?project_id="+projectID, tok, "")
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	page = env.Data.(map[string]any)
	if int(page["total"].(float64)) != 2 {
		t.Fatalf("expected 2 (include legacy), got %v", page["total"])
	}

	// Stats.
	w = doJSONHarness2(t, h, http.MethodGet,
		"/api/v2/sessions/stats?project_id="+projectID, tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("stats: %d %s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	stats := env.Data.(map[string]any)
	if int(stats["total"].(float64)) != 2 {
		t.Fatalf("expected 2 sessions in stats, got %v", stats["total"])
	}

	// Project-scoped endpoint should mirror include_legacy=false.
	w = doJSONHarness2(t, h, http.MethodGet,
		"/api/v2/projects/"+projectID+"/sessions", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("project sessions: %d %s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	page = env.Data.(map[string]any)
	if int(page["total"].(float64)) != 1 {
		t.Fatalf("expected 1, got %v", page["total"])
	}

	// Detail.
	w = doJSONHarness2(t, h, http.MethodGet, "/api/v2/sessions/s1", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("detail: %d %s", w.Code, w.Body.String())
	}
}
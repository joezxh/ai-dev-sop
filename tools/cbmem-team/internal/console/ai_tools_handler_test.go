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
// M6 (AI Tools) — HTTP handler tests.
// =============================================================================

type aiToolsTestHarness struct {
	DB       *DB
	Router   *gin.Engine
	MintToken func(userID string, role Role) string
	// Server is an optional httptest.Server used by invoke tests to
	// mock the external tool endpoint.
	Server *httptest.Server
}

func newAIToolsTestHarness(t *testing.T) *aiToolsTestHarness {
	t.Helper()
	// Use the SAME DB as setupTestDB() so the HTTP router and
	// direct DB calls share state.
	db := setupTestDB(t)
	secret := []byte("ai-tools-test-secret-32byte!!")
	verifier := auth.NewVerifier(secret)

	teams := NewTeamHandlers(db)
	projects := NewProjectHandlers(db, t.TempDir(), "codebase-memory-mcp")
	modules := NewModuleHandlers(db)
	aiTools := NewAIToolHandlers(db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	v2 := r.Group("/api/v2")
	v2.Use(verifier.Middleware())

	// M3 minimal routes needed for project resolution.
	v2.POST("/teams", teams.CreateTeam())
	v2.POST("/teams/:id/projects", projects.CreateProject())
	v2.POST("/projects/:pid/modules", modules.CreateModule())

	// M6 routes under test.
	v2.GET("/ai-tools", aiTools.List())
	v2.POST("/ai-tools", aiTools.Create())
	v2.GET("/ai-tools/:id", aiTools.Get())
	v2.PUT("/ai-tools/:id", aiTools.Update())
	v2.DELETE("/ai-tools/:id", aiTools.Delete())
	v2.POST("/ai-tools/:id/invoke", aiTools.Invoke())
	v2.GET("/ai-tools/:id/invocations", aiTools.ListInvocations())
	v2.GET("/ai-tools/invocations/:inv_id", aiTools.GetInvocation())

	h := &aiToolsTestHarness{DB: db, Router: r}
	h.MintToken = func(userID string, role Role) string {
		signer := auth.NewVerifier(secret)
		tok, _ := signer.SignClaims(
			auth.Claims{Sub: userID, Role: string(role), Kind: "access"},
			15*time.Minute)
		return tok
	}
	return h
}

func (h *aiToolsTestHarness) seedUser(t *testing.T, id string, role Role) {
	t.Helper()
	hash, _ := auth.Hash("password123")
	u := &User{ID: id, Username: "u_" + id, PasswordHash: hash, Role: string(role)}
	_ = h.DB.CreateUser(context.Background(), u)
}

func (h *aiToolsTestHarness) seedStack(t *testing.T, teamSlug, projectSlug, ownerID string) (teamID, projectID, leafID string) {
	t.Helper()
	// Create the team/project/module directly in DB so we bypass
	// HTTP routing and have deterministic IDs independent of handler
	// wiring quirks in the test environment.
	ctx := context.Background()
	tm := &Team{ID: "team_" + teamSlug, Name: "Team " + teamSlug, Slug: teamSlug, OwnerID: ownerID}
	if err := h.DB.CreateTeam(ctx, tm); err != nil {
		t.Fatalf("seed team: %v", err)
	}
	if err := h.DB.AddTeamMember(ctx, tm.ID, ownerID, ProjectRoleOwner); err != nil {
		t.Fatalf("seed owner member: %v", err)
	}
	proj := &Project{
		ID: "proj_" + projectSlug, TeamID: tm.ID, Slug: projectSlug,
		Name: projectSlug, Path: "/tmp/" + projectSlug, Status: "ready", OwnerID: ownerID,
	}
	if err := h.DB.CreateProjectV2(ctx, proj); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	// Leaf module: create as a root leaf first, then a child under it
	// so the parent becomes non-leaf (matches production CreateModule
	// semantics).
	internal := &Module{ID: "mod_int_" + projectSlug, ProjectID: proj.ID,
		Name: "Internal", IsLeaf: true}
	if err := h.DB.CreateModule(ctx, internal); err != nil {
		t.Fatalf("seed internal module: %v", err)
	}
	leaf := &Module{ID: "mod_leaf_" + projectSlug, ProjectID: proj.ID,
		ParentID: &internal.ID, Name: "Leaf", IsLeaf: true}
	if err := h.DB.CreateModule(ctx, leaf); err != nil {
		t.Fatalf("seed leaf module: %v", err)
	}
	return tm.ID, proj.ID, leaf.ID
}

// addMember adds a user to a team with the given role. Call this
// after seedStack for any user that needs team membership.
func (h *aiToolsTestHarness) addMember(t *testing.T, teamID, userID string, role ProjectRole) {
	t.Helper()
	if err := h.DB.AddTeamMember(context.Background(), teamID, userID, role); err != nil {
		t.Fatalf("add member %s->%s: %v", userID, teamID, err)
	}
}

func aiDoJSON(t *testing.T, h *aiToolsTestHarness, method, path, token, body string) *httptest.ResponseRecorder {
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

func aiExtractID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var env envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	return env.Data.(map[string]any)["id"].(string)
}

func aiAssertCode(t *testing.T, w *httptest.ResponseRecorder, want int) {
	if w.Code != want {
		t.Fatalf("code %d, want %d. body: %s", w.Code, want, w.Body.String())
	}
}

// ----------------------------------------------------------------------------
// Catalogue tests
// ----------------------------------------------------------------------------

func TestAIToolsCreateAndList(t *testing.T) {
	h := newAIToolsTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	h.seedUser(t, "u_dev", RoleDeveloper)
	teamID, _, _ := h.seedStack(t, "alpha", "p1", "u_admin")
	// Ensure developer is a team member so RBAC passes.
	h.addMember(t, teamID, "u_dev", ProjectRoleDeveloper)
	adminTok := h.MintToken("u_admin", RoleAdmin)

	// Admin creates a tool.
	body := fmt.Sprintf(`{
		"team_id":"%s",
		"name":"Cursor Ask",
		"slug":"cursor-ask",
		"endpoint":"http://localhost:9999/ask",
		"protocol":"http",
		"required_role":"developer",
		"timeout_seconds":60
	}`, teamID)
	w := aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools", adminTok, body)
	aiAssertCode(t, w, http.StatusOK)
	toolID := aiExtractID(t, w)

	// List: admin sees all.
	w = aiDoJSON(t, h, http.MethodGet, "/api/v2/ai-tools?team_id="+teamID, adminTok, "")
	aiAssertCode(t, w, http.StatusOK)
	if !bytes.Contains(w.Body.Bytes(), []byte("Cursor Ask")) {
		t.Fatalf("expected tool in list: %s", w.Body.String())
	}

	// Developer can see it too (required_role=developer ≤ developer).
	devTok := h.MintToken("u_dev", RoleDeveloper)
	w = aiDoJSON(t, h, http.MethodGet, "/api/v2/ai-tools?team_id="+teamID, devTok, "")
	aiAssertCode(t, w, http.StatusOK)
	if !bytes.Contains(w.Body.Bytes(), []byte("cursor-ask")) {
		t.Fatalf("developer should see tool: %s", w.Body.String())
	}

	// Lead-only tool: developer should NOT see it.
	body = fmt.Sprintf(`{
		"team_id":"%s","name":"Cursor Admin","slug":"cursor-admin",
		"endpoint":"http://localhost:9999/admin","protocol":"http",
		"required_role":"lead","timeout_seconds":60}`, teamID)
	w = aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools", adminTok, body)
	aiAssertCode(t, w, http.StatusOK)
	w = aiDoJSON(t, h, http.MethodGet, "/api/v2/ai-tools?team_id="+teamID, devTok, "")
	aiAssertCode(t, w, http.StatusOK)
	if bytes.Contains(w.Body.Bytes(), []byte("cursor-admin")) {
		t.Fatalf("developer should NOT see lead-only tool: %s", w.Body.String())
	}

	// Get one.
	w = aiDoJSON(t, h, http.MethodGet, "/api/v2/ai-tools/"+toolID, adminTok, "")
	aiAssertCode(t, w, http.StatusOK)

	// 404.
	w = aiDoJSON(t, h, http.MethodGet, "/api/v2/ai-tools/notfound", adminTok, "")
	aiAssertCode(t, w, http.StatusNotFound)

	// Update.
	w = aiDoJSON(t, h, http.MethodPut, "/api/v2/ai-tools/"+toolID, adminTok,
		`{"name":"Cursor Ask v2"}`)
	aiAssertCode(t, w, http.StatusOK)
	if !bytes.Contains(w.Body.Bytes(), []byte("Cursor Ask v2")) {
		t.Fatalf("update response missing new name: %s", w.Body.String())
	}

	// Disable.
	disabled := false
	bodyJSON, _ := json.Marshal(map[string]any{"enabled": disabled})
	w = aiDoJSON(t, h, http.MethodPut, "/api/v2/ai-tools/"+toolID, adminTok, string(bodyJSON))
	aiAssertCode(t, w, http.StatusOK)
}

func TestAIToolsCreateAdminLeadOnly(t *testing.T) {
	h := newAIToolsTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	h.seedUser(t, "u_dev", RoleDeveloper)
	teamID, _, _ := h.seedStack(t, "alpha", "p1", "u_admin")

	// Developer denied.
	devTok := h.MintToken("u_dev", RoleDeveloper)
	body := fmt.Sprintf(`{"team_id":"%s","name":"X","slug":"x","endpoint":"http://x","protocol":"http","required_role":"developer"}`, teamID)
	w := aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools", devTok, body)
	aiAssertCode(t, w, http.StatusForbidden)

	// Lead allowed (add as team lead member).
	lead := &User{ID: "u_lead", Username: "lead", PasswordHash: mustHash("pass")}
	_ = h.DB.CreateUser(context.Background(), lead)
	_ = h.DB.AddTeamMember(context.Background(), teamID, "u_lead", ProjectRoleMaintainer)
	leadTok := h.MintToken("u_lead", RoleLead)
	w = aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools", leadTok, body)
	aiAssertCode(t, w, http.StatusOK)
}

func TestAIToolsDeleteUnused(t *testing.T) {
	h := newAIToolsTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	teamID, _, _ := h.seedStack(t, "alpha", "p1", "u_admin")
	adminTok := h.MintToken("u_admin", RoleAdmin)

	body := fmt.Sprintf(`{"team_id":"%s","name":"Del","slug":"del","endpoint":"http://x","protocol":"http","required_role":"developer"}`, teamID)
	w := aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools", adminTok, body)
	aiAssertCode(t, w, http.StatusOK)
	toolID := aiExtractID(t, w)

	// Delete unused tool: allowed.
	w = aiDoJSON(t, h, http.MethodDelete, "/api/v2/ai-tools/"+toolID, adminTok, "")
	aiAssertCode(t, w, http.StatusOK)

	// 404 after delete.
	w = aiDoJSON(t, h, http.MethodGet, "/api/v2/ai-tools/"+toolID, adminTok, "")
	aiAssertCode(t, w, http.StatusNotFound)
}

func TestAIToolsInvokeHTTP(t *testing.T) {
	h := newAIToolsTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	teamID, projectID, leafID := h.seedStack(t, "alpha", "p1", "u_admin")
	adminTok := h.MintToken("u_admin", RoleAdmin)

	// Create a mock HTTP server that the tool will call.
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"answer":"42","model":"test"}`)
	}))
	defer mock.Close()

	body := fmt.Sprintf(`{
		"team_id":"%s","name":"Mock Tool","slug":"mock",
		"endpoint":"%s","protocol":"http",
		"required_role":"developer","timeout_seconds":10}`, teamID, mock.URL)
	w := aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools", adminTok, body)
	aiAssertCode(t, w, http.StatusOK)
	toolID := aiExtractID(t, w)

	// Invoke.
	invokeBody := fmt.Sprintf(`{
		"project_id":"%s","module_id":"%s","input":{"question":"what is 6*7?"}}`,
		projectID, leafID)
	w = aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools/"+toolID+"/invoke", adminTok, invokeBody)
	aiAssertCode(t, w, http.StatusOK)

	// Verify the response shape.
	var env envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	data := env.Data.(map[string]any)
	if data["status"] != "success" {
		t.Fatalf("expected success, got: %s", w.Body.String())
	}
	inv := data["invocation"].(map[string]any)
	if inv["tool_id"] != toolID {
		t.Errorf("tool_id mismatch: %v", inv["tool_id"])
	}

	// Audit log.
	w = aiDoJSON(t, h, http.MethodGet, "/api/v2/ai-tools/"+toolID+"/invocations", adminTok, "")
	aiAssertCode(t, w, http.StatusOK)
	if !bytes.Contains(w.Body.Bytes(), []byte(toolID)) {
		t.Fatalf("invocation should appear in audit: %s", w.Body.String())
	}
}

func TestAIToolsInvokeForbiddenForDisabledTool(t *testing.T) {
	h := newAIToolsTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	teamID, projectID, leafID := h.seedStack(t, "alpha", "p1", "u_admin")
	adminTok := h.MintToken("u_admin", RoleAdmin)

	body := fmt.Sprintf(`{"team_id":"%s","name":"X","slug":"x","endpoint":"http://x","protocol":"http","required_role":"developer","timeout_seconds":10}`, teamID)
	w := aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools", adminTok, body)
	aiAssertCode(t, w, http.StatusOK)
	toolID := aiExtractID(t, w)

	// Disable it.
	disabled := false
	bodyJSON, _ := json.Marshal(map[string]any{"enabled": disabled})
	w = aiDoJSON(t, h, http.MethodPut, "/api/v2/ai-tools/"+toolID, adminTok, string(bodyJSON))
	aiAssertCode(t, w, http.StatusOK)

	// Invoke disabled tool: 400.
	invokeBody := fmt.Sprintf(`{"project_id":"%s","module_id":"%s","input":{}}`, projectID, leafID)
	w = aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools/"+toolID+"/invoke", adminTok, invokeBody)
	aiAssertCode(t, w, http.StatusBadRequest)
}

func TestAIToolsInvokeUnauthorized(t *testing.T) {
	h := newAIToolsTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	teamID, _, leafID := h.seedStack(t, "alpha", "p1", "u_admin")
	adminTok := h.MintToken("u_admin", RoleAdmin)

	body := fmt.Sprintf(`{"team_id":"%s","name":"X","slug":"x","endpoint":"http://x","protocol":"http","required_role":"developer"}`, teamID)
	w := aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools", adminTok, body)
	toolID := aiExtractID(t, w)

	// No JWT → 401.
	w = aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools/"+toolID+"/invoke", "", "")
	aiAssertCode(t, w, http.StatusUnauthorized)

	// Bad project.
	invokeBody := fmt.Sprintf(`{"project_id":"notexist","module_id":"%s","input":{}}`, leafID)
	w = aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools/"+toolID+"/invoke", adminTok, invokeBody)
	aiAssertCode(t, w, http.StatusBadRequest)
}

func TestAIToolsStdioInvocation(t *testing.T) {
	h := newAIToolsTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	teamID, projectID, leafID := h.seedStack(t, "alpha", "p1", "u_admin")
	adminTok := h.MintToken("u_admin", RoleAdmin)

	// stdio tool: use cmd /c echo on Windows (echo is a shell builtin
	// on Unix). Args pass the /c flag so cmd runs the echo command.
	body := fmt.Sprintf(`{
		"team_id":"%s","name":"Echo","slug":"echo",
		"protocol":"stdio","command":"cmd",
		"args":["/c","echo","hello from stdio"],
		"required_role":"developer","timeout_seconds":5}`, teamID)
	w := aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools", adminTok, body)
	aiAssertCode(t, w, http.StatusOK)
	toolID := aiExtractID(t, w)

	invokeBody := fmt.Sprintf(`{"project_id":"%s","module_id":"%s","input":{}}`, projectID, leafID)
	w = aiDoJSON(t, h, http.MethodPost, "/api/v2/ai-tools/"+toolID+"/invoke", adminTok, invokeBody)
	aiAssertCode(t, w, http.StatusOK)

	var env envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	data := env.Data.(map[string]any)
	if data["status"] != "success" {
		t.Fatalf("expected success, got: %s", w.Body.String())
	}
}

// mustHash returns a bcrypt hash for test users.
func mustHash(pw string) string {
	h, _ := auth.Hash(pw)
	return h
}

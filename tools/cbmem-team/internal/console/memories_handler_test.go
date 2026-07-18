package console

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/auth"
)

// =============================================================================
// M5 — memory-templates + memories HTTP handler tests.
// =============================================================================
//
// Uses a standalone harness so these tests stay independent of
// modules/teams projects and can run quickly without an llm.Provider.
// =============================================================================

type memTestHarness struct {
	DB     *DB
	Router *gin.Engine
	MintToken func(userID string, role Role) string
}

func newMemTestHarness(t *testing.T) *memTestHarness {
	t.Helper()
	db := setupTestDB(t)
	secret := []byte("m5-test-secret-32-bytes-pad!!")
	verifier := auth.NewVerifier(secret)

	teams := NewTeamHandlers(db)
	projects := NewProjectHandlers(db, t.TempDir(), "codebase-memory-mcp")
	modules := NewModuleHandlers(db)
	tpls := NewMemoryTemplateHandlers(db)
	mems := NewMemoryHandlers(db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	v2 := r.Group("/api/v2")
	v2.Use(verifier.Middleware())

	// M3 — minimal team/project routes for memory creation to
	// resolve the team id and project id from the module id.
	v2.POST("/teams", teams.CreateTeam())
	v2.GET("/teams/:id", teams.GetTeam())
	v2.POST("/teams/:id/projects", projects.CreateProject())
	v2.GET("/projects/:pid", projects.GetProject())
	v2.POST("/projects/:pid/modules", modules.CreateModule())
	v2.GET("/modules/:id", modules.GetModule())

	// M5 routes under test
	v2.GET("/memory-templates", tpls.List())
	v2.GET("/memory-templates/:id", tpls.Get())
	v2.POST("/memory-templates/:id/render", tpls.Render())
	v2.POST("/memory-templates", tpls.Create())
	v2.GET("/memories", mems.List())
	v2.POST("/memories", mems.Create())
	v2.GET("/memories/:id", mems.Get())
	v2.PUT("/memories/:id", mems.Update())
	v2.DELETE("/memories/:id", mems.Delete())
	v2.POST("/memories/:id/tags", mems.AddTag())
	v2.GET("/modules/:id/memories", mems.ModuleMemories())

	h := &memTestHarness{DB: db, Router: r}
	h.MintToken = func(userID string, role Role) string {
		signer := auth.NewVerifier(secret)
		tok, err := signer.SignClaims(auth.Claims{Sub: userID, Role: string(role), Kind: "access"}, 15*time.Minute)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		return tok
	}
	return h
}

// seedUser inserts a user with a real password hash.
func (h *memTestHarness) seedUser(t *testing.T, id string, role Role) {
	t.Helper()
	hash, err := auth.Hash("password123")
	if err != nil {
		t.Fatal(err)
	}
	u := &User{ID: id, Username: "user_" + id, PasswordHash: hash, Role: string(role)}
	if err := h.DB.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

// seedStack creates team → project → leaf module chain with the
// given admin as owner. Returns (teamID, projectID, leafModuleID).
func (h *memTestHarness) seedStack(t *testing.T, teamSlug, projectSlug, ownerID string) (string, string, string) {
	t.Helper()
	body := fmt.Sprintf(`{"name":"Team %s","slug":"%s","owner_id":"%s"}`, teamSlug, teamSlug, ownerID)
	w := memDoJSON(t, h, http.MethodPost, "/api/v2/teams", h.MintToken(ownerID, RoleAdmin), body)
	if w.Code != http.StatusOK {
		t.Fatalf("create team: %d %s", w.Code, w.Body.String())
	}
	teamID := memExtractID(t, w)

	body = fmt.Sprintf(`{"name":"%s","slug":"%s"}`, projectSlug, projectSlug)
	w = memDoJSON(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", h.MintToken(ownerID, RoleAdmin), body)
	if w.Code != http.StatusOK {
		t.Fatalf("create project: %d %s", w.Code, w.Body.String())
	}
	projectID := memExtractID(t, w)

	body = `{"name":"leaf-leaf","is_leaf":true}`
	w = memDoJSON(t, h, http.MethodPost, "/api/v2/projects/"+projectID+"/modules", h.MintToken(ownerID, RoleAdmin), body)
	if w.Code != http.StatusOK {
		t.Fatalf("create module: %d %s", w.Code, w.Body.String())
	}
	leafID := memExtractID(t, w)
	return teamID, projectID, leafID
}

// memDoJSON issues a JSON request via httptest.
func memDoJSON(t *testing.T, h *memTestHarness, method, path, token, body string) *httptest.ResponseRecorder {
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

func memExtractID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var env envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	m, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("response %s has no data map", w.Body.String())
	}
	return m["id"].(string)
}

// ----------------------------------------------------------------------------
// Memory Templates
// ----------------------------------------------------------------------------

func TestMemoryTemplatesListAndGet(t *testing.T) {
	h := newMemTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	tok := h.MintToken("u_admin", RoleAdmin)

	// List — built-ins are seeded by MigrateV2.
	w := memDoJSON(t, h, http.MethodGet, "/api/v2/memory-templates", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	var env envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	m := env.Data.(map[string]any)
	list := m["list"].([]any)
	if len(list) != 5 {
		t.Fatalf("expected 5 templates, got %d", len(list))
	}
	// Get one
	w = memDoJSON(t, h, http.MethodGet, "/api/v2/memory-templates/tpl_adr", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}
	// Missing
	w = memDoJSON(t, h, http.MethodGet, "/api/v2/memory-templates/tpl_doesnotexist", tok, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing template, got %d", w.Code)
	}
}

func TestMemoryTemplatesRender(t *testing.T) {
	h := newMemTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	tok := h.MintToken("u_admin", RoleAdmin)

	body := `{"fields":{
		"title":"My Title",
		"status":"accepted",
		"context":"ctx here",
		"decision":"we decided X",
		"consequences":"easier to test"
	}}`
	w := memDoJSON(t, h, http.MethodPost, "/api/v2/memory-templates/tpl_adr/render", tok, body)
	if w.Code != http.StatusOK {
		t.Fatalf("render: %d %s", w.Code, w.Body.String())
	}
	var env envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	m := env.Data.(map[string]any)
	rendered := m["rendered"].(string)
	if !strings.Contains(rendered, "My Title") || !strings.Contains(rendered, "we decided X") {
		t.Fatalf("rendered missing substitutions: %s", rendered)
	}
	if !strings.Contains(rendered, "Status") {
		t.Fatalf("expected 'Status' in rendered template: %s", rendered)
	}
}

func TestMemoryTemplatesCreateAdminOnly(t *testing.T) {
	h := newMemTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	h.seedUser(t, "u_dev", RoleDeveloper)
	// Developer forbidden.
	body := `{"name":"Custom","body_template":"hello"}`
	w := memDoJSON(t, h, http.MethodPost, "/api/v2/memory-templates", h.MintToken("u_dev", RoleDeveloper), body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for developer, got %d", w.Code)
	}
	// Admin allowed.
	w = memDoJSON(t, h, http.MethodPost, "/api/v2/memory-templates", h.MintToken("u_admin", RoleAdmin), body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d %s", w.Code, w.Body.String())
	}
}

// ----------------------------------------------------------------------------
// Memories (CRUD + leaf guard + RBAC)
// ----------------------------------------------------------------------------

func TestMemoriesCreateAndList(t *testing.T) {
	h := newMemTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	_, _, leafID := h.seedStack(t, "alpha", "p1", "u_admin")
	tok := h.MintToken("u_admin", RoleAdmin)

	body := fmt.Sprintf(`{
		"title":"Hello",
		"content":"World",
		"module_id":"%s",
		"hall":"facts",
		"tags":["greeting","intro"]
	}`, leafID)
	w := memDoJSON(t, h, http.MethodPost, "/api/v2/memories", tok, body)
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	mid := memExtractID(t, w)
	// List with filter — team_id auto-derives from the project
	// owner when memory was created with module_id, so we filter
	// by module_id directly.
	w = memDoJSON(t, h, http.MethodGet, "/api/v2/memories?module_id="+leafID, tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), mid) {
		t.Fatalf("expected %s in list, got %s", mid, w.Body.String())
	}
	// Get one
	w = memDoJSON(t, h, http.MethodGet, "/api/v2/memories/"+mid, tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}
}

func TestMemoriesRejectsNonLeafModule(t *testing.T) {
	h := newMemTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	tok := h.MintToken("u_admin", RoleAdmin)

	// Create team + project.
	body := `{"name":"Team alpha","slug":"alpha","owner_id":"u_admin"}`
	w := memDoJSON(t, h, http.MethodPost, "/api/v2/teams", tok, body)
	if w.Code != http.StatusOK {
		t.Fatalf("create team: %d %s", w.Code, w.Body.String())
	}
	teamID := memExtractID(t, w)
	body = `{"name":"p1","slug":"p1"}`
	w = memDoJSON(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", tok, body)
	projectID := memExtractID(t, w)

	// Step 1: create a leaf as root ("internal-root-leaf").
	body = `{"name":"internal-root-leaf","is_leaf":true}`
	w = memDoJSON(t, h, http.MethodPost, "/api/v2/projects/"+projectID+"/modules", tok, body)
	if w.Code != http.StatusOK {
		t.Fatalf("create root leaf: %d %s", w.Code, w.Body.String())
	}
	internalID := memExtractID(t, w)

	// Step 2: create a child under it. After step 2 the parent
	// flips to is_leaf=0 (non-leaf / internal). We then attempt to
	// create a memory on the now-internal parent.
	body = `{"name":"child-of-internal","is_leaf":true,"parent_id":"` + internalID + `"}`
	w = memDoJSON(t, h, http.MethodPost, "/api/v2/projects/"+projectID+"/modules", tok, body)
	if w.Code != http.StatusOK {
		t.Fatalf("create child: %d %s", w.Code, w.Body.String())
	}

	// Now `internalID` is a non-leaf. Creating a memory under it
	// should be rejected with 400 (ErrModuleNotLeaf).
	body = fmt.Sprintf(`{"title":"X","content":"Y","module_id":"%s"}`, internalID)
	w = memDoJSON(t, h, http.MethodPost, "/api/v2/memories", tok, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-leaf module, got %d %s", w.Code, w.Body.String())
	}
}

func TestMemoriesDeveloperCannotCreate(t *testing.T) {
	h := newMemTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	h.seedUser(t, "u_dev", RoleDeveloper)
	_, _, leafID := h.seedStack(t, "alpha", "p1", "u_admin")
	// Add dev as team member but with maintainer? No — developer
	// per RBAC contract gets read-only on memories. The team is
	// owned by u_admin; we add u_dev as a regular member (role
	// "developer" in project_members lingo).
	if err := h.DB.AddTeamMember(context.Background(), h.teamIDFromLeaf(t, leafID), "u_dev", ProjectRoleDeveloper); err != nil {
		t.Fatalf("add member: %v", err)
	}
	body := fmt.Sprintf(`{"title":"X","content":"Y","module_id":"%s"}`, leafID)
	w := memDoJSON(t, h, http.MethodPost, "/api/v2/memories", h.MintToken("u_dev", RoleDeveloper), body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for developer on memory create, got %d %s", w.Code, w.Body.String())
	}
}

func TestMemoriesUpdateAndTagAndDelete(t *testing.T) {
	h := newMemTestHarness(t)
	h.seedUser(t, "u_admin", RoleAdmin)
	_, _, leafID := h.seedStack(t, "alpha", "p1", "u_admin")
	tok := h.MintToken("u_admin", RoleAdmin)

	create := fmt.Sprintf(`{"title":"orig","content":"orig","module_id":"%s","tags":["a"]}`, leafID)
	w := memDoJSON(t, h, http.MethodPost, "/api/v2/memories", tok, create)
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	mid := memExtractID(t, w)

	// Partial update — title and tags only.
	w = memDoJSON(t, h, http.MethodPut, "/api/v2/memories/"+mid, tok,
		`{"title":"new-title","tags":["b","c"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "new-title") {
		t.Fatalf("update response missing title: %s", w.Body.String())
	}
	// AddTag
	w = memDoJSON(t, h, http.MethodPost, "/api/v2/memories/"+mid+"/tags", tok, `{"tag":"a"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("add tag: %d %s", w.Code, w.Body.String())
	}
	// Soft delete
	w = memDoJSON(t, h, http.MethodDelete, "/api/v2/memories/"+mid, tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	// After delete, GET returns 404
	w = memDoJSON(t, h, http.MethodGet, "/api/v2/memories/"+mid, tok, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}
}

func TestMemoriesNoJWTDenied(t *testing.T) {
	h := newMemTestHarness(t)
	w := memDoJSON(t, h, http.MethodGet, "/api/v2/memories", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing JWT, got %d", w.Code)
	}
}

// teamIDFromLeaf walks back from the leaf module to its team.
// Used by tests that only know the leaf id.
func (h *memTestHarness) teamIDFromLeaf(t *testing.T, leafID string) string {
	t.Helper()
	var teamID string
	err := h.DB.QueryRowContext(context.Background(),
		`SELECT p.team_id FROM modules m JOIN projects p ON p.id = m.project_id WHERE m.id = ?`,
		leafID,
	).Scan(&teamID)
	if err != nil {
		t.Fatalf("resolve team: %v", err)
	}
	return teamID
}
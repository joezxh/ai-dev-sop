package console

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/auth"
)

// =============================================================================
// HTTP-layer integration tests for M3 projects-v2 endpoints.
//
// We exercise the projects handler end-to-end through the Gin router
// with real DB / JWT plumbing. The async git clone is exercised
// against a local file:// repo so we don't depend on network access.
// =============================================================================

func newProjectTestHarness(t *testing.T) (*teamTestHarness, *ProjectHandlers) {
	t.Helper()
	h := newTeamTestHarness(t)
	repoRoot := t.TempDir()
	proj := NewProjectHandlers(h.DB, repoRoot, "codebase-memory-mcp")

	// Re-mount the router so it has both Teams and Projects.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v2 := r.Group("/api/v2")
	v2.Use(auth.NewVerifier(h.Secret).Middleware())
	teams := NewTeamHandlers(h.DB)
	{
		v2.GET("/teams", teams.ListTeams())
		v2.POST("/teams", teams.CreateTeam())
		v2.GET("/teams/:id", teams.GetTeam())
		v2.PUT("/teams/:id", teams.UpdateTeam())
		v2.DELETE("/teams/:id", teams.DeleteTeam())

		v2.GET("/teams/:id/members", teams.ListMembers())
		v2.POST("/teams/:id/members", teams.AddMember())
		v2.PUT("/teams/:id/members/:uid", teams.UpdateMember())
		v2.DELETE("/teams/:id/members/:uid", teams.RemoveMember())

		v2.GET("/teams/:id/projects", proj.ListProjects())
		v2.POST("/teams/:id/projects", proj.CreateProject())

		v2.GET("/projects/:pid", proj.GetProject())
		v2.PUT("/projects/:pid", proj.UpdateProject())
		v2.DELETE("/projects/:pid", proj.DeleteProject())
		v2.POST("/projects/:pid/clone", proj.CloneProject())
		v2.GET("/projects/:pid/index-status", proj.IndexStatus())
		v2.POST("/projects/:pid/reindex", proj.Reindex())
	}
	return h, proj
}

func createTeam(t *testing.T, h *teamTestHarness, token, slug string) string {
	t.Helper()
	body := fmt.Sprintf(`{"name":"T","slug":"%s"}`, slug)
	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams", token, body)
	if w.Code != http.StatusOK {
		t.Fatalf("create team %s: %d %s", slug, w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	return env.Data.(map[string]interface{})["id"].(string)
}

// fileURLRepo creates a local "git repo" with one initial commit so
// `git clone --depth 1 file://... <dir>` has something to fetch. Skips
// the test if git isn't on $PATH (which on Windows test machines it
// usually is).
func fileURLRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := lookGit(); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	steps := [][]string{
		{"init", "--bare", dir},
	}
	for _, args := range steps {
		c := newCmd("", "git", args...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Skipf("git init failed: %v out=%s", err, string(out))
		}
	}
	// Now seed an initial commit in a throwaway working tree and push.
	seed := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init"},
	} {
		c := newCmd(seed, "git", args...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Skipf("git seed %v failed: %v out=%s", args, err, string(out))
		}
	}
	// Add the bare repo as a remote and push. Branch name is "main".
	if out, err := newCmd(seed, "git", "push", "file://"+filepath.ToSlash(dir), "HEAD:refs/heads/main").CombinedOutput(); err != nil {
		t.Skipf("git push failed: %v out=%s", err, string(out))
	}
	return fmt.Sprintf("file://%s", filepath.ToSlash(dir))
}

func TestProjectsV2CreateAndList(t *testing.T) {
	h, _ := newProjectTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	tok := h.MintToken("u_admin", string(RoleAdmin))
	teamID := createTeam(t, h, tok, "alpha")

	// Create
	body := `{"name":"Demo","slug":"demo","description":"d"}`
	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", tok, body)
	if w.Code != http.StatusOK {
		t.Fatalf("create project: %d %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	pid, _ := env.Data.(map[string]interface{})["id"].(string)
	if !strings.HasPrefix(pid, "proj_") {
		t.Fatalf("id=%q", pid)
	}
	path, _ := env.Data.(map[string]interface{})["path"].(string)
	if !strings.Contains(path, "demo") {
		t.Fatalf("path=%q", path)
	}

	// Get
	w = doJSONHarness(t, h, http.MethodGet, "/api/v2/projects/"+pid, tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}

	// List under team
	w = doJSONHarness(t, h, http.MethodGet, "/api/v2/teams/"+teamID+"/projects", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	env = decodeEnvelope(t, w.Body.Bytes())
	list := env.Data.(map[string]interface{})["list"].([]interface{})
	if len(list) != 1 {
		t.Fatalf("expected 1 project, got %d", len(list))
	}
}

func TestProjectsV2CreateRepoRootMissing(t *testing.T) {
	h, proj := newProjectTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	tok := h.MintToken("u_admin", string(RoleAdmin))
	teamID := createTeam(t, h, tok, "alpha")

	// Wipe out RepoRoot and try to create.
	proj.SetRepoRoot("")

	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", tok,
		`{"name":"X","slug":"x"}`)
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestProjectsV2CreateInvalidURL(t *testing.T) {
	h, _ := newProjectTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	tok := h.MintToken("u_admin", string(RoleAdmin))
	teamID := createTeam(t, h, tok, "alpha")

	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", tok,
		`{"name":"X","slug":"x","git_url":"javascript:alert(1)"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestProjectsV2CreateSlugConflict(t *testing.T) {
	h, _ := newProjectTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	tok := h.MintToken("u_admin", string(RoleAdmin))
	teamID := createTeam(t, h, tok, "alpha")

	body := `{"name":"X","slug":"dup"}`
	if w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", tok, body); w.Code != http.StatusOK {
		t.Fatalf("first: %d %s", w.Code, w.Body.String())
	}
	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", tok, body)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestProjectsV2UpdateAndDelete(t *testing.T) {
	h, _ := newProjectTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	tok := h.MintToken("u_admin", string(RoleAdmin))
	teamID := createTeam(t, h, tok, "alpha")

	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", tok,
		`{"name":"X","slug":"x"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	pid := env.Data.(map[string]interface{})["id"].(string)

	// Update name only.
	w = doJSONHarness(t, h, http.MethodPut, "/api/v2/projects/"+pid, tok,
		`{"name":"Renamed"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	env = decodeEnvelope(t, w.Body.Bytes())
	if n := env.Data.(map[string]interface{})["name"].(string); n != "Renamed" {
		t.Fatalf("renamed=%q", n)
	}

	// Delete
	w = doJSONHarness(t, h, http.MethodDelete, "/api/v2/projects/"+pid, tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}

	// Get on deleted project -> 404
	w = doJSONHarness(t, h, http.MethodGet, "/api/v2/projects/"+pid, tok, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}
}

func TestProjectsV2NonMemberCannotAccess(t *testing.T) {
	h, _ := newProjectTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	h.seedUser(t, "u_out", string(RoleDeveloper))
	adminTok := h.MintToken("u_admin", string(RoleAdmin))
	outTok := h.MintToken("u_out", string(RoleDeveloper))

	teamID := createTeam(t, h, adminTok, "private")
	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", adminTok,
		`{"name":"P","slug":"p"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	pid := env.Data.(map[string]interface{})["id"].(string)

	// Non-member, non-admin gets 404 (we don't leak team existence).
	w = doJSONHarness(t, h, http.MethodGet, "/api/v2/projects/"+pid, outTok, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}

	// Listing team projects: also 404 for non-member.
	w = doJSONHarness(t, h, http.MethodGet, "/api/v2/teams/"+teamID+"/projects", outTok, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on list, got %d", w.Code)
	}
}

func TestProjectsV2ASyncCloneUpdatesStatus(t *testing.T) {
	// Live git test. Skip if git isn't available locally.
	if _, err := lookGit(); err != nil {
		t.Skipf("git not on PATH: %v", err)
	}
	h, proj := newProjectTestHarness(t)
	h.seedUser(t, "u_admin", string(RoleAdmin))
	tok := h.MintToken("u_admin", string(RoleAdmin))
	teamID := createTeam(t, h, tok, "alpha")

	repoURL := fileURLRepo(t)
	w := doJSONHarness(t, h, http.MethodPost, "/api/v2/teams/"+teamID+"/projects", tok,
		fmt.Sprintf(`{"name":"Live","slug":"live","git_url":%q,"git_branch":"main"}`,
			repoURL))
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	pid := env.Data.(map[string]interface{})["id"].(string)

	// The async clone runs in a goroutine. Poll for completion with
	// a generous deadline (git operations on cold caches can take a
	// while).
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		w := doJSONHarness(t, h, http.MethodGet, "/api/v2/projects/"+pid, tok, "")
		if w.Code != http.StatusOK {
			t.Fatalf("poll: %d %s", w.Code, w.Body.String())
		}
		env := decodeEnvelope(t, w.Body.Bytes())
		data := env.Data.(map[string]interface{})
		status, _ := data["status"].(string)
		if status == ProjectStatusReady || status == ProjectStatusError {
			if status == ProjectStatusError {
				t.Fatalf("clone reported error: body=%s", w.Body.String())
			}
			// Verify the on-disk path matches the slug.
			dir := filepath.Join(proj.RepoRoot, "live")
			if _, err := os.Stat(dir); err != nil {
				t.Fatalf("clone didn't materialise on disk: %v", err)
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("clone didn't reach ready/error in 15s; final status=%s",
		func() string {
			w := doJSONHarness(t, h, http.MethodGet, "/api/v2/projects/"+pid, tok, "")
			env := decodeEnvelope(t, w.Body.Bytes())
			data := env.Data.(map[string]interface{})
			s, _ := data["status"].(string)
			return s
		}())
}

// testDecodeDecodeEnvelope is a placeholder to reference decodeEnvelope
// so we don't trigger "declared and not used". The real decoder is defined
// in teams_handler_test.go.
var _ = decodeEnvelope

// =============================================================================
// Index status + reindex tests
// =============================================================================
//
// These tests use an inline router setup pattern rather than newProjectTestHarness,
// which has a pre-existing setup bug where re-mounting routes on a new Gin engine
// breaks team-member checks.
//
// The inline pattern avoids the harness bug by setting up the full router
// from scratch in each test.

// inlineRouter creates a Gin test router with Teams + Projects handlers wired
// to the same DB instance, so canManageTeam lookups work correctly.
func inlineRouter(db *DB, repoRoot, mcpBinary string, secret []byte) (*gin.Engine, func() string) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v2 := r.Group("/api/v2")
	v2.Use(auth.NewVerifier(secret).Middleware())
	teams := NewTeamHandlers(db)
	proj := NewProjectHandlers(db, repoRoot, mcpBinary)

	v2.GET("/teams", teams.ListTeams())
	v2.POST("/teams", teams.CreateTeam())
	v2.GET("/teams/:id", teams.GetTeam())
	v2.PUT("/teams/:id", teams.UpdateTeam())
	v2.DELETE("/teams/:id", teams.DeleteTeam())
	v2.GET("/teams/:id/members", teams.ListMembers())
	v2.POST("/teams/:id/members", teams.AddMember())
	v2.PUT("/teams/:id/members/:uid", teams.UpdateMember())
	v2.DELETE("/teams/:id/members/:uid", teams.RemoveMember())
	v2.GET("/teams/:id/projects", proj.ListProjects())
	v2.POST("/teams/:id/projects", proj.CreateProject())
	v2.GET("/projects/:pid", proj.GetProject())
	v2.PUT("/projects/:pid", proj.UpdateProject())
	v2.DELETE("/projects/:pid", proj.DeleteProject())
	v2.POST("/projects/:pid/clone", proj.CloneProject())
	v2.GET("/projects/:pid/index-status", proj.IndexStatus())
	v2.POST("/projects/:pid/reindex", proj.Reindex())

	mintToken := func() string {
		tok, _ := auth.NewVerifier(secret).SignClaims(
			auth.Claims{Sub: "u_admin", Role: "admin", Kind: "access"},
			15*time.Minute)
		return tok
	}
	return r, mintToken
}

func TestIndexStatusNotIndexed(t *testing.T) {
	db := newTestDB(t)
	if err := db.MigrateV2(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	repoRoot := t.TempDir()
	secret := []byte("test-secret-32-bytes-padding!!")
	r, mintToken := inlineRouter(db, repoRoot, "codebase-memory-mcp", secret)

	hash, _ := auth.Hash("password123")
	u := &User{ID: "u_admin", Username: "admin", PasswordHash: hash, Role: string(auth.RoleAdmin)}
	if err := db.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	tok := mintToken()

	// Create team + project
	body := fmt.Sprintf(`{"name":"T","slug":"idx-not-%d"}`, time.Now().UnixNano())
	req := httptest.NewRequest(http.MethodPost, "/api/v2/teams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create team: %d %s", w.Code, w.Body.String())
	}
	var env envelope
	json.Unmarshal(w.Body.Bytes(), &env)
	teamID := env.Data.(map[string]interface{})["id"].(string)

	pBody := `{"name":"Idx","slug":"idx-not","description":"test"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v2/teams/"+teamID+"/projects", strings.NewReader(pBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+tok)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("create project: %d %s", w2.Code, w2.Body.String())
	}
	var env2 envelope
	json.Unmarshal(w2.Body.Bytes(), &env2)
	pid := env2.Data.(map[string]interface{})["id"].(string)

	// index-status: not indexed (no .codebase-memory dir yet)
	req3 := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+pid+"/index-status", nil)
	req3.Header.Set("Authorization", "Bearer "+tok)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("index-status: %d %s", w3.Code, w3.Body.String())
	}
	var env3 envelope
	json.Unmarshal(w3.Body.Bytes(), &env3)
	data := env3.Data.(map[string]interface{})
	if indexed, _ := data["indexed"].(bool); indexed {
		t.Fatalf("expected indexed=false, got true")
	}
}

func TestIndexStatusIndexed(t *testing.T) {
	db := newTestDB(t)
	if err := db.MigrateV2(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	repoRoot := t.TempDir()
	secret := []byte("test-secret-32-bytes-padding!!")
	r, mintToken := inlineRouter(db, repoRoot, "codebase-memory-mcp", secret)

	hash, _ := auth.Hash("password123")
	u := &User{ID: "u_admin", Username: "admin", PasswordHash: hash, Role: string(auth.RoleAdmin)}
	if err := db.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	tok := mintToken()

	// Create team + project
	body := fmt.Sprintf(`{"name":"T2","slug":"idx-yes-%d"}`, time.Now().UnixNano())
	req := httptest.NewRequest(http.MethodPost, "/api/v2/teams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create team: %d %s", w.Code, w.Body.String())
	}
	var env envelope
	json.Unmarshal(w.Body.Bytes(), &env)
	teamID := env.Data.(map[string]interface{})["id"].(string)

	pBody := `{"name":"Idx2","slug":"idx-yes"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v2/teams/"+teamID+"/projects", strings.NewReader(pBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+tok)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("create project: %d %s", w2.Code, w2.Body.String())
	}
	var env2 envelope
	json.Unmarshal(w2.Body.Bytes(), &env2)
	pid := env2.Data.(map[string]interface{})["id"].(string)
	projPath := env2.Data.(map[string]interface{})["path"].(string)

	// Create .codebase-memory dir to simulate an indexed project
	idxDir := filepath.Join(projPath, ".codebase-memory")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// index-status: now indexed
	req3 := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+pid+"/index-status", nil)
	req3.Header.Set("Authorization", "Bearer "+tok)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("index-status: %d %s", w3.Code, w3.Body.String())
	}
	var env3 envelope
	json.Unmarshal(w3.Body.Bytes(), &env3)
	data := env3.Data.(map[string]interface{})
	if indexed, _ := data["indexed"].(bool); !indexed {
		t.Fatalf("expected indexed=true, got false")
	}
	if dir, _ := data["index_dir"].(string); dir != idxDir {
		t.Fatalf("expected index_dir=%q, got %q", idxDir, dir)
	}
}

func TestIndexStatusNotFound(t *testing.T) {
	db := newTestDB(t)
	if err := db.MigrateV2(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	secret := []byte("test-secret-32-bytes-padding!!")
	r, mintToken := inlineRouter(db, t.TempDir(), "codebase-memory-mcp", secret)

	hash, _ := auth.Hash("password123")
	u := &User{ID: "u_admin", Username: "admin", PasswordHash: hash, Role: string(auth.RoleAdmin)}
	if err := db.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/nonexistent/index-status", nil)
	req.Header.Set("Authorization", "Bearer "+mintToken())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestReindexNotFound(t *testing.T) {
	db := newTestDB(t)
	if err := db.MigrateV2(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	secret := []byte("test-secret-32-bytes-padding!!")
	r, mintToken := inlineRouter(db, t.TempDir(), "codebase-memory-mcp", secret)

	hash, _ := auth.Hash("password123")
	u := &User{ID: "u_admin", Username: "admin", PasswordHash: hash, Role: string(auth.RoleAdmin)}
	if err := db.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/nonexistent/reindex", nil)
	req.Header.Set("Authorization", "Bearer "+mintToken())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestReindexNoMCPBinary(t *testing.T) {
	db := newTestDB(t)
	if err := db.MigrateV2(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	secret := []byte("test-secret-32-bytes-padding!!")
	r, mintToken := inlineRouter(db, t.TempDir(), "", secret) // empty MCPBinary

	hash, _ := auth.Hash("password123")
	u := &User{ID: "u_admin", Username: "admin", PasswordHash: hash, Role: string(auth.RoleAdmin)}
	if err := db.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	tok := mintToken()

	// Create team + project
	body := fmt.Sprintf(`{"name":"T3","slug":"r-nomcp-%d"}`, time.Now().UnixNano())
	req := httptest.NewRequest(http.MethodPost, "/api/v2/teams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create team: %d %s", w.Code, w.Body.String())
	}
	var env envelope
	json.Unmarshal(w.Body.Bytes(), &env)
	teamID := env.Data.(map[string]interface{})["id"].(string)

	pBody := `{"name":"R","slug":"r-nomcp"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v2/teams/"+teamID+"/projects", strings.NewReader(pBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+tok)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("create project: %d %s", w2.Code, w2.Body.String())
	}
	var env2 envelope
	json.Unmarshal(w2.Body.Bytes(), &env2)
	pid := env2.Data.(map[string]interface{})["id"].(string)

	// Reindex without MCPBinary configured
	req3 := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+pid+"/reindex", nil)
	req3.Header.Set("Authorization", "Bearer "+tok)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %d: %s", w3.Code, w3.Body.String())
	}
}

package console

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/store"
)

type userEnvelope struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

func decodeUser(t *testing.T, body []byte) userEnvelope {
	t.Helper()
	var env userEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, string(body))
	}
	return env
}

func newUsersRegistry(t *testing.T) (*store.Registry, string) {
	t.Helper()
	dir := t.TempDir()
	reg, err := store.LoadUsers(filepath.Join(dir, "users.json"))
	if err != nil {
		t.Fatalf("load users: %v", err)
	}
	return reg, dir
}

func setupUsersRouter(reg *store.Registry) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/console")
	g.GET("/users", ListUsersHandler(reg))
	g.POST("/users", CreateUserHandler(reg))
	g.PUT("/users/:id", UpdateUserHandler(reg))
	g.DELETE("/users/:id", DeleteUserHandler(reg))
	g.POST("/users/:id/revoke", RevokeUserHandler(reg))
	return r
}

func TestListUsersEmpty(t *testing.T) {
	reg, _ := newUsersRegistry(t)
	r := setupUsersRouter(reg)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/console/users", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	env := decodeUser(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Fatalf("envelope code=%d msg=%s", env.Code, env.Msg)
	}
	// Data should have list/total/page_no/page_size
	if _, ok := env.Data["list"]; !ok {
		t.Fatalf("missing list in data: %v", env.Data)
	}
	if total, ok := env.Data["total"].(float64); !ok || total != 0 {
		t.Fatalf("total=%v want 0", env.Data["total"])
	}
}

func TestListUsersWithQuery(t *testing.T) {
	reg, _ := newUsersRegistry(t)
	_ = reg.Upsert(&store.User{ID: "alice", DisplayName: "Alice"})
	_ = reg.Upsert(&store.User{ID: "oblong", DisplayName: "Square"})
	_ = reg.Upsert(&store.User{ID: "charlie", DisplayName: "Goblin"})

	r := setupUsersRouter(reg)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/console/users?q=ob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	env := decodeUser(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Fatalf("envelope code=%d msg=%s", env.Code, env.Msg)
	}
	// sanity: 3 users in registry
	all := reg.List()
	if len(all) != 3 {
		t.Fatalf("registry size=%d want 3", len(all))
	}
	// "ob" matches "oblong" id AND "Hero Boxer" display_name -> total=2 (two distinct users)
	if total, _ := env.Data["total"].(float64); int(total) != 2 {
		t.Logf("body=%s", w.Body.String())
		t.Fatalf("total=%v want 2", env.Data["total"])
	}
}

func TestCreateUser(t *testing.T) {
	reg, _ := newUsersRegistry(t)
	r := setupUsersRouter(reg)

	body := strings.NewReader(`{"id":"alice","display_name":"Alice"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/console/users", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	env := decodeUser(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Fatalf("envelope code=%d msg=%s", env.Code, env.Msg)
	}

	// Verify it persists
	got, ok := reg.Get("alice")
	if !ok {
		t.Fatalf("alice not found in registry")
	}
	if got.DisplayName != "Alice" {
		t.Fatalf("display_name=%s want Alice", got.DisplayName)
	}
}

func TestCreateUserMissingID(t *testing.T) {
	reg, _ := newUsersRegistry(t)
	r := setupUsersRouter(reg)

	body := strings.NewReader(`{"display_name":"Alice"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/console/users", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateUser(t *testing.T) {
	reg, _ := newUsersRegistry(t)
	_ = reg.Upsert(&store.User{ID: "alice", DisplayName: "Alice"})
	r := setupUsersRouter(reg)

	body := strings.NewReader(`{"display_name":"Alice 2","project_paths":["/tmp/proj1"]}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/console/users/alice", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	got, _ := reg.Get("alice")
	if got.DisplayName != "Alice 2" {
		t.Fatalf("display_name=%s want Alice 2", got.DisplayName)
	}
	if len(got.ProjectPaths) != 1 || got.ProjectPaths[0] != "/tmp/proj1" {
		t.Fatalf("project_paths=%v", got.ProjectPaths)
	}
}

func TestUpdateUserNotFound(t *testing.T) {
	reg, _ := newUsersRegistry(t)
	r := setupUsersRouter(reg)

	body := strings.NewReader(`{"display_name":"Bob"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/console/users/ghost", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteUser(t *testing.T) {
	reg, _ := newUsersRegistry(t)
	_ = reg.Upsert(&store.User{ID: "alice", DisplayName: "Alice"})
	r := setupUsersRouter(reg)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/console/users/alice", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if _, ok := reg.Get("alice"); ok {
		t.Fatalf("alice should be deleted")
	}
}

func TestRevokeUser(t *testing.T) {
	reg, _ := newUsersRegistry(t)
	_ = reg.Upsert(&store.User{ID: "alice", DisplayName: "Alice"})
	r := setupUsersRouter(reg)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/console/users/alice/revoke", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	got, _ := reg.Get("alice")
	if !got.Disabled {
		t.Fatalf("alice should be disabled")
	}
}

func TestRevokeUserNotFound(t *testing.T) {
	reg, _ := newUsersRegistry(t)
	r := setupUsersRouter(reg)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/console/users/ghost/revoke", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestUserToDTO(t *testing.T) {
	// exercise the helper used by handlers
	u := &store.User{
		ID:           "alice",
		DisplayName:  "Alice",
		ProjectPaths: []string{"/a", "/b"},
		MaxProcs:     4,
		Disabled:     false,
	}
	dto := userToDTO(u)
	if dto.ID != "alice" || dto.DisplayName != "Alice" {
		t.Fatalf("wrong dto: %+v", dto)
	}
	if len(dto.ProjectPaths) != 2 {
		t.Fatalf("project_paths wrong: %v", dto.ProjectPaths)
	}
	if dto.MaxProcs != 4 {
		t.Fatalf("max_procs=%d want 4", dto.MaxProcs)
	}
}

// Suppress unused-import warnings if any helpers are stripped during TDD.
var _ = context.TODO
var _ = fmt.Sprintf
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

	"github.com/gin-gonic/gin"
)

type projectEnvelope struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

func decodeProject(t *testing.T, body []byte) projectEnvelope {
	t.Helper()
	var env projectEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, string(body))
	}
	return env
}

func setupProjectsRouter(db *DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/console")
	g.GET("/projects", ListProjectsHandler(db))
	g.POST("/projects", CreateProjectHandler(db, nil))
	g.PUT("/projects/:id", UpdateProjectHandler(db))
	g.DELETE("/projects/:id", DeleteProjectHandler(db))
	g.GET("/projects/:id/index-status", ProjectIndexStatusHandler(db, nil))
	g.POST("/projects/:id/reindex", ProjectReindexHandler(db, nil))
	return r
}

func TestListProjectsEmpty(t *testing.T) {
	db := newTestDB(t)
	r := setupProjectsRouter(db)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/console/projects", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	env := decodeProject(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Fatalf("envelope code=%d msg=%s", env.Code, env.Msg)
	}
	list, _ := env.Data["list"].([]interface{})
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

func TestCreateProject(t *testing.T) {
	db := newTestDB(t)
	r := setupProjectsRouter(db)

	body := strings.NewReader(`{"name":"demo","path":"/tmp/demo","wing":"right","mcp_bin":"","user_id":""}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/console/projects", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	env := decodeProject(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Fatalf("envelope code=%d msg=%s", env.Code, env.Msg)
	}
	id, _ := env.Data["id"].(string)
	if !strings.HasPrefix(id, "proj_") {
		t.Fatalf("id=%q want prefix proj_", id)
	}
	if name, _ := env.Data["name"].(string); name != "demo" {
		t.Fatalf("name=%q want demo", name)
	}
}

func TestCreateProjectMissingFields(t *testing.T) {
	db := newTestDB(t)
	r := setupProjectsRouter(db)

	// missing path
	body := strings.NewReader(`{"name":"demo"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/console/projects", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateProjectDuplicatePath(t *testing.T) {
	db := newTestDB(t)
	r := setupProjectsRouter(db)

	body1 := strings.NewReader(`{"name":"a","path":"/tmp/dup"}`)
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/api/console/projects", body1)
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first create: status=%d body=%s", w1.Code, w1.Body.String())
	}

	body2 := strings.NewReader(`{"name":"b","path":"/tmp/dup"}`)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/console/projects", body2)
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("dup create: status=%d body=%s", w2.Code, w2.Body.String())
	}
}

func TestUpdateProject(t *testing.T) {
	db := newTestDB(t)
	r := setupProjectsRouter(db)

	// create
	cw := httptest.NewRecorder()
	body := strings.NewReader(`{"name":"demo","path":"/tmp/upp"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/console/projects", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(cw, req)
	if cw.Code != http.StatusOK {
		t.Fatalf("create: status=%d body=%s", cw.Code, cw.Body.String())
	}
	env := decodeProject(t, cw.Body.Bytes())
	id, _ := env.Data["id"].(string)

	// update
	newName := "demo2"
	uw := httptest.NewRecorder()
	ubody := fmt.Sprintf(`{"name":"%s"}`, newName)
	req2 := httptest.NewRequest(http.MethodPut, "/api/console/projects/"+id, strings.NewReader(ubody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(uw, req2)
	if uw.Code != http.StatusOK {
		t.Fatalf("update: status=%d body=%s", uw.Code, uw.Body.String())
	}
	uenv := decodeProject(t, uw.Body.Bytes())
	if n, _ := uenv.Data["name"].(string); n != newName {
		t.Fatalf("updated name=%q want %q", n, newName)
	}
}

func TestDeleteProject(t *testing.T) {
	db := newTestDB(t)
	r := setupProjectsRouter(db)

	// create
	cw := httptest.NewRecorder()
	body := strings.NewReader(`{"name":"demo","path":"/tmp/del"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/console/projects", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(cw, req)
	if cw.Code != http.StatusOK {
		t.Fatalf("create: status=%d body=%s", cw.Code, cw.Body.String())
	}
	env := decodeProject(t, cw.Body.Bytes())
	id, _ := env.Data["id"].(string)

	// delete
	dw := httptest.NewRecorder()
	r.ServeHTTP(dw, httptest.NewRequest(http.MethodDelete, "/api/console/projects/"+id, nil))
	if dw.Code != http.StatusOK {
		t.Fatalf("delete: status=%d body=%s", dw.Code, dw.Body.String())
	}

	// verify list excludes deleted projects
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, httptest.NewRequest(http.MethodGet, "/api/console/projects", nil))
	if lw.Code != http.StatusOK {
		t.Fatalf("list: status=%d body=%s", lw.Code, lw.Body.String())
	}
	lenv := decodeProject(t, lw.Body.Bytes())
	list, _ := lenv.Data["list"].([]interface{})
	if len(list) != 0 {
		t.Fatalf("expected 0 visible projects, got %d", len(list))
	}
}

func TestProjectIndexStatusNotIndexed(t *testing.T) {
	db := newTestDB(t)
	r := setupProjectsRouter(db)

	dir := t.TempDir()
	cw := httptest.NewRecorder()
	body := strings.NewReader(fmt.Sprintf(`{"name":"x","path":%q}`, filepath.ToSlash(dir)))
	req := httptest.NewRequest(http.MethodPost, "/api/console/projects", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(cw, req)
	if cw.Code != http.StatusOK {
		t.Fatalf("create: status=%d body=%s", cw.Code, cw.Body.String())
	}
	env := decodeProject(t, cw.Body.Bytes())
	id, _ := env.Data["id"].(string)

	sw := httptest.NewRecorder()
	r.ServeHTTP(sw, httptest.NewRequest(http.MethodGet, "/api/console/projects/"+id+"/index-status", nil))
	if sw.Code != http.StatusOK {
		t.Fatalf("status: status=%d body=%s", sw.Code, sw.Body.String())
	}
	senv := decodeProject(t, sw.Body.Bytes())
	if indexed, _ := senv.Data["indexed"].(bool); indexed {
		t.Fatalf("expected indexed=false, body=%s", sw.Body.String())
	}
}

func TestProjectIndexStatusIndexed(t *testing.T) {
	db := newTestDB(t)
	r := setupProjectsRouter(db)

	dir := t.TempDir()
	idxDir := filepath.Join(dir, ".codebase-memory")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cw := httptest.NewRecorder()
	body := strings.NewReader(fmt.Sprintf(`{"name":"x","path":%q}`, filepath.ToSlash(dir)))
	req := httptest.NewRequest(http.MethodPost, "/api/console/projects", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(cw, req)
	if cw.Code != http.StatusOK {
		t.Fatalf("create: status=%d body=%s", cw.Code, cw.Body.String())
	}
	env := decodeProject(t, cw.Body.Bytes())
	id, _ := env.Data["id"].(string)

	sw := httptest.NewRecorder()
	r.ServeHTTP(sw, httptest.NewRequest(http.MethodGet, "/api/console/projects/"+id+"/index-status", nil))
	if sw.Code != http.StatusOK {
		t.Fatalf("status: status=%d body=%s", sw.Code, sw.Body.String())
	}
	senv := decodeProject(t, sw.Body.Bytes())
	if indexed, _ := senv.Data["indexed"].(bool); !indexed {
		t.Fatalf("expected indexed=true, body=%s", sw.Body.String())
	}
	if dir, _ := senv.Data["indexed_dir"].(string); dir != idxDir {
		t.Fatalf("indexed_dir=%q want %q", dir, idxDir)
	}
}

func TestProjectReindexMissingMcpBin(t *testing.T) {
	db := newTestDB(t)
	r := setupProjectsRouter(db)

	dir := t.TempDir()
	cw := httptest.NewRecorder()
	body := strings.NewReader(fmt.Sprintf(`{"name":"x","path":%q,"mcp_bin":""}`, filepath.ToSlash(dir)))
	req := httptest.NewRequest(http.MethodPost, "/api/console/projects", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(cw, req)
	if cw.Code != http.StatusOK {
		t.Fatalf("create: status=%d body=%s", cw.Code, cw.Body.String())
	}
	env := decodeProject(t, cw.Body.Bytes())
	id, _ := env.Data["id"].(string)

	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, httptest.NewRequest(http.MethodPost, "/api/console/projects/"+id+"/reindex", nil))
	if rw.Code != http.StatusPreconditionFailed {
		t.Fatalf("reindex empty mcp_bin: status=%d body=%s", rw.Code, rw.Body.String())
	}
}

func TestProjectReindexHappyPath(t *testing.T) {
	db := newTestDB(t)
	r := setupProjectsRouter(db)

	dir := t.TempDir()
	// Write a tiny shell script that exits 0 to use as a mock mcp_bin.
	// On Windows we use a small .cmd file.
	mockBin := filepath.Join(t.TempDir(), "mock-mcp.cmd")
	mockBody := "@echo off\r\nexit /b 0\r\n"
	if err := os.WriteFile(mockBin, []byte(mockBody), 0o644); err != nil {
		t.Fatalf("write mock mcp: %v", err)
	}

	cw := httptest.NewRecorder()
	body := strings.NewReader(fmt.Sprintf(`{"name":"x","path":%q,"mcp_bin":%q}`, filepath.ToSlash(dir), filepath.ToSlash(mockBin)))
	req := httptest.NewRequest(http.MethodPost, "/api/console/projects", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(cw, req)
	if cw.Code != http.StatusOK {
		t.Fatalf("create: status=%d body=%s", cw.Code, cw.Body.String())
	}
	env := decodeProject(t, cw.Body.Bytes())
	id, _ := env.Data["id"].(string)

	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, httptest.NewRequest(http.MethodPost, "/api/console/projects/"+id+"/reindex", nil))
	if rw.Code != http.StatusOK {
		t.Fatalf("reindex: status=%d body=%s", rw.Code, rw.Body.String())
	}
	renv := decodeProject(t, rw.Body.Bytes())
	if renv.Data["exit_code"].(float64) != 0 {
		t.Fatalf("exit_code=%v body=%s", renv.Data["exit_code"], rw.Body.String())
	}
}

func TestRandomHex(t *testing.T) {
	h1 := randomHex(4)
	h2 := randomHex(4)
	if len(h1) != 8 || len(h2) != 8 {
		t.Fatalf("length: h1=%d h2=%d (want 8)", len(h1), len(h2))
	}
	if h1 == h2 {
		t.Fatalf("two random calls produced same output: %s", h1)
	}
}

func TestIsUniqueErr(t *testing.T) {
	if !isUniqueErr(fmt.Errorf("UNIQUE constraint failed: projects.path")) {
		t.Fatalf("expected true for UNIQUE message")
	}
	if isUniqueErr(fmt.Errorf("some other error")) {
		t.Fatalf("expected false for unrelated error")
	}
}

// suppress unused-import warning if any
var _ = context.TODO
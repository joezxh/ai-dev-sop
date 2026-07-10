package console

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/llm"
)

func TestDistillAndCommit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	sm := NewSessionManager(db, time.Hour)
	sID, err := sm.Create(context.TODO(), "admin", "127.0.0.1", "UA")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Seed s1 + one turn so loadTranscript finds it.
	if _, err := db.Exec(
		`INSERT INTO sessions (id, user_id, project_path, started_at) VALUES ('s1','admin','/p','2026-01-01')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO session_turns (session_id, turn_no, role, content, ts) VALUES ('s1',1,'user','hi','2026-01-01')`); err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	provider := llm.NewFake(`{
        "knowledge_fragments":[{"type":"fact","content":"use PGVector","tags":["db"],"value_score":0.92,"source_ref":"sess-1 turn-2"}],
        "decisions":[{"title":"use pg","decision":"yes","context":"c","source_ref":"sess-1"}],
        "tech_debt":[{"description":"timeout missing","severity":"high","source_ref":"sess-2"}]
    }`)
	mp := llm.NewMemPalace("http://example.invalid", "")

	r := gin.New()
	api := r.Group("/api/console")
	api.POST("/distill", RequireSession(sm), RequireCSRF(), DistillHandler(db, provider, mp))
	api.GET("/distill/:task_id", RequireSession(sm), GetDistillHandler(db))
	api.POST("/distill/:task_id/commit", RequireSession(sm), RequireCSRF(), CommitDistillHandler(db, mp))

	body := strings.NewReader(`{"source_ids":["s1"],"rules":{"min_value_score":0.7,"dimensions":["fact"]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/console/distill", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: CookieSession, Value: sID})
	req.AddCookie(&http.Cookie{Name: CookieCSRF, Value: "tok"})
	req.Header.Set(HeaderCSRF, "tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("distill=%d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if env.Code != 0 {
		t.Fatalf("envelope code=%d msg=%s", env.Code, env.Msg)
	}
	taskID, _ := env.Data["task_id"].(string)
	if taskID == "" {
		t.Fatalf("missing task_id, body=%s", w.Body.String())
	}
	if !strings.HasPrefix(taskID, "dis_") {
		t.Fatalf("task_id=%q want dis_ prefix", taskID)
	}

	// GET should report status=done.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/console/distill/"+taskID, nil)
	req2.AddCookie(&http.Cookie{Name: CookieSession, Value: sID})
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get distill=%d body=%s", w2.Code, w2.Body.String())
	}
	var env2 struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &env2)
	if status, _ := env2.Data["status"].(string); status != "done" {
		t.Fatalf("status=%q want done", status)
	}
}

func TestDistillCommitCapturesDrawers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	sm := NewSessionManager(db, time.Hour)
	sID, _ := sm.Create(context.TODO(), "admin", "127.0.0.1", "UA")

	if _, err := db.Exec(
		`INSERT INTO sessions (id, user_id, project_path, started_at) VALUES ('s1','admin','/p','2026-01-01')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	captured := make([][]byte, 0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"up"}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/drawers" {
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			captured = append(captured, buf)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	provider := llm.NewFake(`{
        "knowledge_fragments":[{"type":"fact","content":"use PGVector","tags":["db"],"value_score":0.92,"source_ref":"sess-1 turn-2"}],
        "decisions":[{"title":"use pg","decision":"yes","context":"c","source_ref":"sess-1"}],
        "tech_debt":[]
    }`)
	mp := llm.NewMemPalace(srv.URL, "")

	r := gin.New()
	api := r.Group("/api/console")
	api.POST("/distill", RequireSession(sm), RequireCSRF(), DistillHandler(db, provider, mp))
	api.POST("/distill/:task_id/commit", RequireSession(sm), RequireCSRF(), CommitDistillHandler(db, mp))

	// Distill
	body := strings.NewReader(`{"source_ids":["s1"],"rules":{"min_value_score":0.7,"dimensions":["fact"]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/console/distill", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: CookieSession, Value: sID})
	req.AddCookie(&http.Cookie{Name: CookieCSRF, Value: "tok"})
	req.Header.Set(HeaderCSRF, "tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("distill=%d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Data map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	taskID, _ := env.Data["task_id"].(string)
	if taskID == "" {
		t.Fatalf("missing task_id")
	}

	// Commit
	cBody := strings.NewReader(`{"target_wing":"wing-x"}`)
	req2 := httptest.NewRequest(http.MethodPost, "/api/console/distill/"+taskID+"/commit", cBody)
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(&http.Cookie{Name: CookieSession, Value: sID})
	req2.AddCookie(&http.Cookie{Name: CookieCSRF, Value: "tok"})
	req2.Header.Set(HeaderCSRF, "tok")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("commit=%d body=%s", w2.Code, w2.Body.String())
	}

	// Verify MemPalace saw 2 drawer POSTs (1 fragment, 1 decision).
	if len(captured) != 2 {
		t.Fatalf("expected 2 captured drawer POSTs, got %d (%s)", len(captured), captured)
	}
	wantFragment := `"hall":"facts"` + "\n"
	if !strings.Contains(strings.Join([]string{string(captured[0])}, ""), `"hall":"facts"`) {
		t.Fatalf("captured[0] does not contain hall=facts: %s", captured[0])
	}
	if !strings.Contains(string(captured[1]), `"hall":"advice"`) {
		t.Fatalf("captured[1] does not contain hall=advice: %s", captured[1])
	}
	_ = wantFragment
}

func TestDistillMissingSourceIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	sm := NewSessionManager(db, time.Hour)
	sID, _ := sm.Create(context.TODO(), "admin", "127.0.0.1", "UA")
	provider := llm.NewFake(`{}`)
	mp := llm.NewMemPalace("http://example.invalid", "")
	r := gin.New()
	api := r.Group("/api/console")
	api.POST("/distill", RequireSession(sm), RequireCSRF(), DistillHandler(db, provider, mp))

	req := httptest.NewRequest(http.MethodPost, "/api/console/distill",
		strings.NewReader(`{"source_ids":[],"rules":{"min_value_score":0.7,"dimensions":["fact"]}}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: CookieSession, Value: sID})
	req.AddCookie(&http.Cookie{Name: CookieCSRF, Value: "tok"})
	req.Header.Set(HeaderCSRF, "tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != 4000002 {
		t.Fatalf("envelope code=%d want 4000002", env.Code)
	}
}

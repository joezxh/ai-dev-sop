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

	"cbmem-team/internal/llm"
)

func TestSummarizeWithFake(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	sm := NewSessionManager(db, time.Hour)
	sID, err := sm.Create(context.TODO(), "admin", "127.0.0.1", "UA")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Seed session "s1" and one turn so loadTranscript finds it.
	if _, err := db.Exec(
		`INSERT INTO sessions (id, user_id, project_path, started_at) VALUES ('s1','admin','/p','2026-01-01')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO session_turns (session_id, turn_no, role, content, ts) VALUES ('s1',1,'user','hi','2026-01-01')`); err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	provider := llm.NewFake(`{"hall_facts":[{"title":"x","summary":"y","confidence":0.9}]}`)
	r := gin.New()
	api := r.Group("/api/console")
	api.POST("/summarize", RequireSession(sm), RequireCSRF(), SummarizeHandler(db, provider))
	api.GET("/summarize/:task_id", RequireSession(sm), GetSummarizeHandler(db))

	// POST
	body := strings.NewReader(`{"source_ids":["s1"],"depth":"deep","target_wing":"wing-x"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/console/summarize", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: CookieSession, Value: sID})
	req.AddCookie(&http.Cookie{Name: CookieCSRF, Value: "tok"})
	req.Header.Set(HeaderCSRF, "tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("post status=%d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal post: %v body=%s", err, w.Body.String())
	}
	if env.Code != 0 {
		t.Fatalf("post envelope code=%d msg=%s", env.Code, env.Msg)
	}
	taskID, _ := env.Data["task_id"].(string)
	if taskID == "" {
		t.Fatalf("missing task_id, body=%s", w.Body.String())
	}
	if !strings.HasPrefix(taskID, "sum_") {
		t.Fatalf("task_id prefix wrong: %q", taskID)
	}

	// GET
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/console/summarize/"+taskID, nil)
	req2.AddCookie(&http.Cookie{Name: CookieSession, Value: sID})
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", w2.Code, w2.Body.String())
	}
	var env2 struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &env2); err != nil {
		t.Fatalf("unmarshal get: %v body=%s", err, w2.Body.String())
	}
	if env2.Code != 0 {
		t.Fatalf("get envelope code=%d", env2.Code)
	}
	if status, _ := env2.Data["status"].(string); status != "done" {
		t.Fatalf("status=%q want done, body=%s", status, w2.Body.String())
	}
	res, _ := env2.Data["result"].(map[string]interface{})
	if res == nil {
		t.Fatalf("result missing or wrong type, body=%s", w2.Body.String())
	}
	if _, ok := res["hall_facts"]; !ok {
		t.Fatalf("result.hall_facts missing, body=%s", w2.Body.String())
	}
}

func TestSummarizeMissingSourceIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	sm := NewSessionManager(db, time.Hour)
	sID, _ := sm.Create(context.TODO(), "admin", "127.0.0.1", "UA")
	provider := llm.NewFake(`{}`)
	r := gin.New()
	api := r.Group("/api/console")
	api.POST("/summarize", RequireSession(sm), RequireCSRF(), SummarizeHandler(db, provider))

	body := bytes.NewBufferString(`{"source_ids":[],"depth":"deep"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/console/summarize", body)
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

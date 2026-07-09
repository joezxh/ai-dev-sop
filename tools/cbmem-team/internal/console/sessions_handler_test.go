package console

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type sessionEnvelope struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

func decodeSession(t *testing.T, body []byte) sessionEnvelope {
	t.Helper()
	var env sessionEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, string(body))
	}
	return env
}

// seedSessions inserts 2 sessions + 4 turns directly via SQL so the tests
// exercise the read paths without going through the capture middleware.
func seedSessions(t *testing.T, db *DB) {
	t.Helper()
	stmts := []string{
		`INSERT INTO sessions (id, user_id, project_path, started_at, turn_count, tool_count) VALUES ('s1','alice','/p1','2026-01-01 00:00:00',2,1)`,
		`INSERT INTO sessions (id, user_id, project_path, started_at, turn_count, tool_count) VALUES ('s2','alice','/p2','2026-01-02 00:00:00',2,0)`,
		`INSERT INTO session_turns (session_id, turn_no, role, content, tools_json, ts) VALUES ('s1',1,'user','hi',NULL,'2026-01-01 00:00:00')`,
		`INSERT INTO session_turns (session_id, turn_no, role, content, tools_json, ts) VALUES ('s1',2,'assistant','hello',NULL,'2026-01-01 00:00:01')`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(context.TODO(), s); err != nil {
			t.Fatalf("seed: %v (sql=%s)", err, s)
		}
	}
}

func setupSessionsRouter(db *DB, sm *SessionManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/console")
	g.POST("/login", func(c *gin.Context) {
		sid, err := sm.Create(c.Request.Context(), "admin", "127.0.0.1", "UA")
		if err != nil {
			t_FailInternal(c, err.Error())
			return
		}
		c.SetCookie(CookieSession, sid, int(8*time.Hour.Seconds()), "/", "", false, true)
		c.SetCookie(CookieCSRF, "tok", int(8*time.Hour.Seconds()), "/", "", false, false)
		OK(c, gin.H{"csrf": "tok", "session_id": sid, "admin_id": "admin"})
	})
	protected := g.Group("")
	protected.Use(RequireSession(sm), RequireCSRF())
	protected.GET("/sessions", ListSessionsHandler(db))
	protected.GET("/sessions/:id", SessionDetailHandler(db))
	protected.GET("/sessions-stats", SessionsStatsHandler(db))
	return r
}

// t_FailInternal is a tiny helper that mirrors Fail but is reachable from
// non-handler setup code without a *testing.T reference.
func t_FailInternal(c *gin.Context, msg string) {
	Fail(c, http.StatusInternalServerError, 5000001, msg)
}

func TestSessionsListAndDetail(t *testing.T) {
	db := newTestDB(t)
	seedSessions(t, db)
	sm := NewSessionManager(db, 8*time.Hour)
	r := setupSessionsRouter(db, sm)

	// log in to obtain a session cookie + csrf cookie
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, httptest.NewRequest(http.MethodPost, "/api/console/login", nil))
	if loginW.Code != http.StatusOK {
		t.Fatalf("login=%d body=%s", loginW.Code, loginW.Body.String())
	}
	var sessionCookie, csrfCookie *http.Cookie
	for _, ck := range loginW.Result().Cookies() {
		switch ck.Name {
		case CookieSession:
			sessionCookie = ck
		case CookieCSRF:
			csrfCookie = ck
		}
	}
	if sessionCookie == nil || csrfCookie == nil {
		t.Fatalf("missing cookies: %v", loginW.Result().Cookies())
	}

	call := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.AddCookie(&http.Cookie{Name: CookieSession, Value: sessionCookie.Value})
		req.AddCookie(&http.Cookie{Name: CookieCSRF, Value: csrfCookie.Value})
		req.Header.Set(HeaderCSRF, csrfCookie.Value)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// list — user_id=alice should return 2
	lw := call("GET", "/api/console/sessions?user_id=alice")
	if lw.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", lw.Code, lw.Body.String())
	}
	lenv := decodeSession(t, lw.Body.Bytes())
	if lenv.Code != 0 {
		t.Fatalf("list envelope code=%d msg=%s", lenv.Code, lenv.Msg)
	}
	totalF, ok := lenv.Data["total"].(float64)
	if !ok {
		t.Fatalf("list: total missing or not number: %T %v", lenv.Data["total"], lenv.Data["total"])
	}
	if int(totalF) != 2 {
		t.Fatalf("list total=%v want 2", totalF)
	}

	// detail
	dw := call("GET", "/api/console/sessions/s1")
	if dw.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", dw.Code, dw.Body.String())
	}
denv := decodeSession(t, dw.Body.Bytes())
	if denv.Code != 0 {
		t.Fatalf("detail envelope code=%d msg=%s", denv.Code, denv.Msg)
	}
	sessionObj, ok := denv.Data["session"].(map[string]interface{})
	if !ok {
		t.Fatalf("detail session missing: %T %v", denv.Data["session"], denv.Data["session"])
	}
	if sessionObj["id"] != "s1" {
		t.Fatalf("detail session.id=%v want s1", sessionObj["id"])
	}
	turns, ok := denv.Data["turns"].([]interface{})
	if !ok {
		t.Fatalf("detail turns missing: %T %v", denv.Data["turns"], denv.Data["turns"])
	}
	if len(turns) != 2 {
		t.Fatalf("detail turns=%d want 2 (body=%s)", len(turns), dw.Body.String())
	}

	// stats
	sw := call("GET", "/api/console/sessions-stats")
	if sw.Code != http.StatusOK {
		t.Fatalf("stats status=%d body=%s", sw.Code, sw.Body.String())
	}
	senv := decodeSession(t, sw.Body.Bytes())
	if senv.Code != 0 {
		t.Fatalf("stats envelope code=%d msg=%s", senv.Code, senv.Msg)
	}
	if v, _ := senv.Data["session_total"].(float64); int(v) != 2 {
		t.Fatalf("stats session_total=%v want 2 (data=%v)", v, senv.Data)
	}
	if v, _ := senv.Data["turn_total"].(float64); int(v) != 4 {
		t.Fatalf("stats turn_total=%v want 4 (data=%v)", v, senv.Data)
	}
	if v, _ := senv.Data["tool_call_total"].(float64); int(v) != 1 {
		t.Fatalf("stats tool_call_total=%v want 1 (data=%v)", v, senv.Data)
	}
}

func TestSessionsDetailNotFound(t *testing.T) {
	db := newTestDB(t)
	seedSessions(t, db)
	sm := NewSessionManager(db, 8*time.Hour)
	r := setupSessionsRouter(db, sm)

	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, httptest.NewRequest(http.MethodPost, "/api/console/login", nil))
	if loginW.Code != http.StatusOK {
		t.Fatalf("login=%d", loginW.Code)
	}
	var sessionCookie, csrfCookie *http.Cookie
	for _, ck := range loginW.Result().Cookies() {
		switch ck.Name {
		case CookieSession:
			sessionCookie = ck
		case CookieCSRF:
			csrfCookie = ck
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/console/sessions/ghost", nil)
	req.AddCookie(&http.Cookie{Name: CookieSession, Value: sessionCookie.Value})
	req.AddCookie(&http.Cookie{Name: CookieCSRF, Value: csrfCookie.Value})
	req.Header.Set(HeaderCSRF, csrfCookie.Value)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("detail-not-found status=%d body=%s", w.Code, w.Body.String())
	}
}

// silence unused-import linter when format helper is stripped.
var _ = fmt.Sprintf
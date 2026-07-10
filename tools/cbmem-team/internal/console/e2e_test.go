//go:build integration
// +build integration

package console

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/llm"
	"cbmem-team/internal/store"
)

func TestE2EConsoleFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	users, err := store.LoadUsers(filepath.Join(dataDir, "users.json"))
	if err != nil {
		t.Fatalf("load users: %v", err)
	}

	sm := NewSessionManager(db, 8*3600*1e9)
	adminTok := "e2e-admin-token"

	provider := llm.NewFake(`{"hall_facts":[],"hall_events":[],"hall_discoveries":[],"hall_preferences":[],"hall_advice":[]}`)

	r := gin.New()
	Mount(r, MountConfig{
		DB:         db,
		AdminToken: adminTok,
		Session:    sm,
		Users:      users,
		LLM:        provider,
		MemPalace:  nil,
	})

	srv := httptest.NewServer(r)
	defer srv.Close()

	jar := &cookieJar{}
	client := &http.Client{Jar: jar}

	// 1. login
	loginBody := `{"admin_id":"admin"}`
	req, _ := http.NewRequest("POST", srv.URL+"/api/console/login", strings.NewReader(loginBody))
	req.Header.Set("X-Admin-Token", adminTok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("login status=%d", resp.StatusCode)
	}
	var loginEnv struct {
		Code int `json:"code"`
		Data struct {
			CSRF    string `json:"csrf_token"`
			Session string `json:"session_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginEnv); err != nil {
		t.Fatalf("login decode: %v", err)
	}
	if loginEnv.Code != 0 || loginEnv.Data.CSRF == "" {
		t.Fatalf("login bad response: %+v", loginEnv)
	}
	t.Logf("login OK csrf=%s session=%s", loginEnv.Data.CSRF, loginEnv.Data.Session)

	// 2. me
	req, _ = http.NewRequest("GET", srv.URL+"/api/console/me", nil)
	req.Header.Set("X-CSRF-Token", loginEnv.Data.CSRF)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("me request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("me status=%d", resp.StatusCode)
	}
	var meEnv struct {
		Code int `json:"code"`
	}
	json.NewDecoder(resp.Body).Decode(&meEnv)
	if meEnv.Code != 0 {
		t.Fatalf("me code=%d", meEnv.Code)
	}
	t.Logf("me OK")

	// 3. users create
	body := bytes.NewBufferString(`{"id":"alice"}`)
	req, _ = http.NewRequest("POST", srv.URL+"/api/console/users", body)
	req.Header.Set("X-CSRF-Token", loginEnv.Data.CSRF)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("users create: %v", err)
	}
	if resp.StatusCode != 200 {
		buf, _ := io.ReadAll(resp.Body)
		t.Fatalf("users create status=%d body=%s", resp.StatusCode, string(buf))
	}
	resp.Body.Close()
	t.Logf("users create OK")

	// 4. users list
	req, _ = http.NewRequest("GET", srv.URL+"/api/console/users", nil)
	req.Header.Set("X-CSRF-Token", loginEnv.Data.CSRF)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("users list: %v", err)
	}
	defer resp.Body.Close()
	var listEnv struct {
		Code int `json:"code"`
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&listEnv)
	if listEnv.Code != 0 {
		t.Fatalf("users list code=%d", listEnv.Code)
	}
	if listEnv.Data.Total < 1 {
		t.Fatalf("users list total=%d", listEnv.Data.Total)
	}
	t.Logf("users list total=%d", listEnv.Data.Total)

	// 5. projects create
	body = bytes.NewBufferString(`{"name":"p","path":"/tmp/p-test"}`)
	req, _ = http.NewRequest("POST", srv.URL+"/api/console/projects", body)
	req.Header.Set("X-CSRF-Token", loginEnv.Data.CSRF)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("projects create: %v", err)
	}
	if resp.StatusCode != 200 {
		buf, _ := io.ReadAll(resp.Body)
		t.Fatalf("projects create status=%d body=%s", resp.StatusCode, string(buf))
	}
	resp.Body.Close()
	t.Logf("projects create OK")

	// 6. projects list
	req, _ = http.NewRequest("GET", srv.URL+"/api/console/projects", nil)
	req.Header.Set("X-CSRF-Token", loginEnv.Data.CSRF)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("projects list: %v", err)
	}
	defer resp.Body.Close()
	var plEnv struct {
		Code int `json:"code"`
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&plEnv)
	if plEnv.Code != 0 || plEnv.Data.Total < 1 {
		t.Fatalf("projects list code=%d total=%d", plEnv.Code, plEnv.Data.Total)
	}
	t.Logf("projects list total=%d", plEnv.Data.Total)

	// 7. sessions-stats
	req, _ = http.NewRequest("GET", srv.URL+"/api/console/sessions-stats", nil)
	req.Header.Set("X-CSRF-Token", loginEnv.Data.CSRF)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("sessions-stats: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("sessions-stats status=%d", resp.StatusCode)
	}
	t.Logf("sessions-stats OK")

	// 8. logout
	req, _ = http.NewRequest("POST", srv.URL+"/api/console/logout", nil)
	req.Header.Set("X-CSRF-Token", loginEnv.Data.CSRF)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("logout status=%d", resp.StatusCode)
	}
	t.Logf("logout OK")
	t.Logf("[e2e-console] OK")
}

type cookieJar struct {
	cookies []*http.Cookie
}

func (j *cookieJar) Cookies(u *url.URL) []*http.Cookie {
	return j.cookies
}

func (j *cookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, c := range cookies {
		j.cookies = append(j.cookies, c)
	}
}

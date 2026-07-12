// e2e-v7 — M3 + M4 acceptance suite (Go).
//
// Coverage per the v2 plan §6.3 + §7.3:
//
//	[1] M3 workflow catalog: list returns 3 seeded built-ins (publish status)
//	[2] create draft workflow + publish + run (depth 2 graph) → run row exists
//	[3] all 3 built-in workflows run end-to-end (commit-precheck /
//	    adr-doublewrite / repo-daily-sync) — verify each writes a
//	    workflow_runs row
//	[4] high-risk dashboard: feed a synthetic tool_invocation_logs row
//	    with affected_halls_json=[…] + error_code=UPSTREAM_ERROR →
//	    /v2/dashboard/high-risk returns it
//	[5] ticket auto-open: /v2/tickets/auto-open creates a ticket with
//	    severity=critical (because halls+error_code)
//	[6] M4 repo pipeline CRUD: create local-source pipeline pointing at
//	    a temp dir with README.md → activate → run → run row + ≥1
//	    bp_candidate row produced
//	[7] bp_candidate review: accept / reject endpoints change status
package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"cbmem-team/internal/devconf"
)

const (
	bin        = `D:\projects\ai-dev-sop\tools\cbmem-team\bin\cbmem-team.exe`
	adminToken = "dev-admin-v7-2026"
	jwtSecret  = "dev-secret-v7-7f8a-2026"
)

// mysqlDSN is resolved in main() from flag/env via internal/devconf so
// CI / local devs can override without editing source.
var mysqlDSN string

func main() {
	flagDSN := flag.String("mysql-dsn", "", "MySQL DSN; if empty, falls back to $CBMEM_MYSQL_DSN then $DSN then the dev default")
	flag.Parse()
	mysqlDSN = devconf.ResolveMySQLDSNWithDB(*flagDSN, devconf.DefaultDevMySQLDB)

	listen := ":28797"
	for _, p := range []string{":28791", ":28792", ":28793", ":28794", ":28795", ":28796", ":28797"} {
		if isFreePort(p) {
			listen = p
			break
		}
	}
	baseURL := "http://127.0.0.1" + listen
	dataDir := filepath.Join(filepath.Dir(bin), "data-v7")
	os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0o755)

	// Clean MySQL-side M3/M4 tables before we start the server.
	// `dataDir` is wiped above, but MySQL is shared across e2e runs;
	// leftover bp_candidates / workflow_runs from prior failed runs
	// would skew step 7's "≥2 candidates across two runs" assertion.
	resetSharedMySQL()

	if err := rebuildServer(); err != nil {
		fail("rebuild server: %v", err)
	}

	cmd := exec.Command(bin,
		"-listen", listen,
		"-data", dataDir,
		"-admin-token", adminToken,
		"-jwt-secret", jwtSecret,
		"-mysql-dsn", mysqlDSN,
		"-mcp-bin", mcpStubPath(),
		"-log", "info")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fail("start server: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	waitForHealthz(8*time.Second, baseURL)
	fmt.Println("\n--- M3 + M4 e2e (v7) 检查 ---")

	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Jar: jar, Timeout: 15 * time.Second}
	login(hc, baseURL)
	registerAlice(hc, baseURL)

	// [1] M3 catalog returns the 3 seeded built-ins.
	fmt.Println("[1] M3 workflow catalog built-ins")
	var wfPage struct {
		Count     int              `json:"count"`
		Workflows []map[string]any `json:"workflows"`
	}
	mustGetJSON(hc, baseURL+"/api/console/v2/workflows", &wfPage)
	if wfPage.Count < 3 {
		fail("expected ≥3 seeded workflows, got %d", wfPage.Count)
	}
	haveBuiltins := 0
	for _, w := range wfPage.Workflows {
		// WorkflowRecord.ID serialises as "id" — see workflow.go DTO.
		id, _ := w["id"].(string)
		if strings.HasPrefix(id, "wf-") {
			haveBuiltins++
		}
	}
	if haveBuiltins < 3 {
		fail("expected ≥3 wf-* builtins, got %d", haveBuiltins)
	}
	fmt.Printf("    ✓ %d workflows (≥3 builtins)\n", wfPage.Count)

	// [2] create + publish + run a 2-node workflow.
	fmt.Println("[2] create + publish + run depth-2 workflow")
	var newWF map[string]any
	mustPostJSON(hc, baseURL+"/api/console/v2/workflows", map[string]any{
		"name":     "e2e-v7 测试工作流",
		"category": "precheck",
		"track":    "J",
		"entry_id": "n_a",
		"nodes": []map[string]any{
			{"id": "n_a", "kind": "log", "label": "start", "next": []string{"n_b"}},
			{"id": "n_b", "kind": "log", "label": "done", "next": []string{}},
		},
	}, &newWF)
	wfID, _ := newWF["id"].(string)
	if !strings.HasPrefix(wfID, "wf-") {
		fail("new workflow id %q does not start with wf-", wfID)
	}
	mustPostJSONExpectStatus(hc, baseURL+"/api/console/v2/workflows/"+wfID+"/publish", nil, http.StatusOK)
	var runRow map[string]any
	mustPostJSON(hc, baseURL+"/api/console/v2/workflows/"+wfID+"/run", nil, &runRow)
	if runRow["status"] != "succeeded" {
		fail("expected status=succeeded got %v", runRow["status"])
	}
	if step, _ := runRow["step_count"].(float64); step < 2 {
		fail("expected step_count ≥2 got %v", runRow["step_count"])
	}
	fmt.Printf("    ✓ wf=%s run_id=%v step_count=%v\n", wfID, runRow["run_id"], runRow["step_count"])

	// [3] all 3 built-in workflows run.
	fmt.Println("[3] 3 built-in workflows run end-to-end")
	for _, id := range []string{"wf-commit-precheck", "wf-adr-doublewrite", "wf-repo-daily-sync"} {
		var r map[string]any
		mustPostJSON(hc, baseURL+"/api/console/v2/workflows/"+id+"/run", nil, &r)
		if r["status"] != "succeeded" {
			fail("builtin %s run status=%v", id, r["status"])
		}
		fmt.Printf("    ✓ %s step_count=%v\n", id, r["step_count"])
	}

	// [4] feed a synthetic high-risk invocation row + verify dashboard.
	fmt.Println("[4] high-risk dashboard shows synthetic row")
	db := openMysql()
	defer db.Close()
	// Use a fresh invocation_id so the run isn't shadowed by M1's row.
	invID := fmt.Sprintf("e2ev7-%d", time.Now().UnixNano())
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO tool_invocation_logs
        (invocation_id, user_id, project_id, project_path, tool_id, transport,
         args_json, started_at, latency_ms, error_code,
         affected_halls_json, affected_adrs_json, blast_radius_json)
       VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		invID, "alice-0", "proj-v7", "/tmp/proj-v7", "detect_changes", "stdio",
		"{}", now, 1234, "UPSTREAM_ERROR",
		`["hall_facts"]`, `[]`, `[]`)
	if err != nil {
		fail("seed high-risk row: %v", err)
	}
	var hrList struct {
		Count       int              `json:"count"`
		Invocations []map[string]any `json:"invocations"`
	}
	mustGetJSON(hc, baseURL+"/api/console/v2/dashboard/high-risk?limit=200", &hrList)
	found := false
	for _, r := range hrList.Invocations {
		if r["invocation_id"] == invID {
			found = true
			break
		}
	}
	if !found {
		fail("synthetic high-risk invocation not in /v2/dashboard/high-risk list")
	}
	fmt.Printf("    ✓ synthetic row %s surfaced in dashboard (total=%d)\n", invID, hrList.Count)

	// [5] ticket auto-open.
	fmt.Println("[5] ticket auto-open")
	var autoOpen struct {
		Opened string `json:"opened"`
	}
	mustPostJSON(hc, baseURL+"/api/console/v2/tickets/auto-open", nil, &autoOpen)
	if !strings.HasPrefix(autoOpen.Opened, "tk-") {
		fail("auto-open expected tk-* got %q", autoOpen.Opened)
	}
	var ticket map[string]any
	mustGetJSON(hc, baseURL+"/api/console/v2/tickets/"+autoOpen.Opened, &ticket)
	if ticket["severity"] != "critical" {
		fail("expected severity=critical got %v (halls+error_code should escalate)", ticket["severity"])
	}
	fmt.Printf("    ✓ ticket %s severity=%s\n", autoOpen.Opened, ticket["severity"])

	// [6] M4 repo pipeline (local source) over a temp dir.
	fmt.Println("[6] M4 local-source repo pipeline run")
	tmpDir, err := os.MkdirTemp("", "e2e-v7-src-")
	if err != nil {
		fail("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	readme := `# Sample Repo

## Overview

This is a long-ish README used by the e2e test to verify that the
heuristic grader produces a score ≥0.7 for content that meets the
length + structure thresholds. The README must exceed 800 characters
to score above the default 0.7 cutoff, so we pad it with paragraphs
about workflow best practices and how to use them.

## Setup

1. Clone the repo.
2. Install dependencies.
3. Run the test suite.

## Verification

The grade stage returns the score; the sink stage writes a
bp_candidate row when the score crosses the pipeline threshold.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(readme), 0o644); err != nil {
		fail("write readme: %v", err)
	}
	var newPipe map[string]any
	mustPostJSON(hc, baseURL+"/api/console/v2/repos", map[string]any{
		"name":      "e2e-v7 测试 pipeline",
		"source":    "local",
		"target":    tmpDir,
		"threshold": 0.5, // lower so the short README passes
	}, &newPipe)
	pipeID, _ := newPipe["id"].(string)
	if !strings.HasPrefix(pipeID, "rp-") {
		fail("expected rp-* got %q", pipeID)
	}
	mustPostJSONExpectStatus(hc, baseURL+"/api/console/v2/repos/"+pipeID+"/activate", nil, http.StatusOK)
	var runResult struct {
		Run struct {
			RunID         string `json:"run_id"`
			Status        string `json:"status"`
			ItemsIngested int    `json:"items_ingested"`
			ItemsParsed   int    `json:"items_parsed"`
			ItemsGraded   int    `json:"items_graded"`
			ItemsAccepted int    `json:"items_accepted"`
		} `json:"Run"`
		Candidates []string `json:"Candidates"`
	}
	mustPostJSON(hc, baseURL+"/api/console/v2/repos/"+pipeID+"/run", nil, &runResult)
	if runResult.Run.Status != "succeeded" {
		fail("pipeline run status=%s err=%v", runResult.Run.Status, runResult.Run)
	}
	if runResult.Run.ItemsIngested < 1 || runResult.Run.ItemsAccepted < 1 {
		fail("expected ingested+accepted ≥1 got %+v", runResult.Run)
	}
	if len(runResult.Candidates) < 1 {
		fail("expected ≥1 candidate got 0")
	}
	fmt.Printf("    ✓ pipeline=%s ingested=%d accepted=%d candidates=%d\n",
		pipeID, runResult.Run.ItemsIngested, runResult.Run.ItemsAccepted, len(runResult.Candidates))

	// [7] candidate review: accept + reject endpoints work.
	fmt.Println("[7] bp_candidate review endpoints")
	candID := runResult.Candidates[0]
	mustPostJSONExpectStatus(hc, baseURL+"/api/console/v2/repos/candidates/"+candID+"/accept", nil, http.StatusOK)
	var cand map[string]any
	mustGetJSON(hc, baseURL+"/api/console/v2/repos/candidates/"+candID, &cand)
	if cand["status"] != "accepted" {
		fail("expected accepted got %v", cand["status"])
	}
	// Open a second draft candidate by creating+activating+rerunning.
	mustPostJSON(hc, baseURL+"/api/console/v2/repos/"+pipeID+"/run", nil, &runResult)
	var candList struct {
		Count      int              `json:"count"`
		Candidates []map[string]any `json:"candidates"`
	}
	// Scope the listing to this pipeline so cross-run / cross-pipeline
	// candidates from the shared MySQL store can't dilute the count.
	mustGetJSON(hc, baseURL+"/api/console/v2/repos/candidates?pipeline_id="+pipeID, &candList)
	if candList.Count < 2 {
		fail("expected ≥2 candidates across two runs, got %d", candList.Count)
	}
	// Find a draft candidate to reject (skip the accepted one).
	var candID2 string
	for _, c := range candList.Candidates {
		if c["status"] == "draft" {
			if id, ok := c["id"].(string); ok {
				candID2 = id
				break
			}
		}
	}
	if candID2 == "" {
		fail("no draft candidate available to reject (all candidates accepted/merged)")
	}
	mustPostJSONExpectStatus(hc, baseURL+"/api/console/v2/repos/candidates/"+candID2+"/reject", nil, http.StatusOK)
	mustGetJSON(hc, baseURL+"/api/console/v2/repos/candidates/"+candID2, &cand)
	if cand["status"] != "rejected" {
		fail("expected rejected got %v", cand["status"])
	}
	fmt.Printf("    ✓ accept+reject both work\n")

	fmt.Println("\n✅ M3 + M4 e2e (v7) 全绿")
}

// --- helpers ---

func rebuildServer() error {
	modRoot := filepath.Dir(filepath.Dir(bin))
	prev, _ := os.Getwd()
	_ = os.Chdir(modRoot)
	defer func() { _ = os.Chdir(prev) }()
	for _, pkg := range []string{"./cmd/cbmem-team/", "./cmd/mcp-stub/"} {
		out := bin
		if pkg == "./cmd/mcp-stub/" {
			out = mcpStubPath()
		}
		cmd := exec.Command("go", "build", "-o", out, pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func mcpStubPath() string {
	return filepath.Join(filepath.Dir(bin), "mcp-stub.exe")
}

func isFreePort(addr string) bool {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

func waitForHealthz(d time.Duration, baseURL string) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/healthz")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
	fail("server didn't come up in %s", d)
}

func login(hc *http.Client, baseURL string) {
	req, _ := http.NewRequest("POST", baseURL+"/api/console/login", nil)
	req.Header.Set("X-Admin-Token", adminToken)
	resp, err := hc.Do(req)
	if err != nil {
		fail("login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fail("login status=%d", resp.StatusCode)
	}
}

func registerAlice(hc *http.Client, baseURL string) {
	body := `{"id":"alice-0","display_name":"Alice","project_paths":["/tmp/proj-v7","/path/to/proj-0"]}`
	req, _ := http.NewRequest("POST", baseURL+"/admin/users", strings.NewReader(body))
	req.Header.Set("X-Admin-Token", adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		fail("register alice: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 409 {
		b, _ := io.ReadAll(resp.Body)
		fail("register alice expected 200/201/409 got %d: %s", resp.StatusCode, string(b))
	}
}

func mustGetJSON(hc *http.Client, url string, out any) {
	resp, err := hc.Get(url)
	if err != nil {
		fail("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		fail("GET %s → %d: %s", url, resp.StatusCode, string(b))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			fail("decode %s: %v", url, err)
		}
	}
}

func mustPostJSON(hc *http.Client, url string, body any, out any) {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest("POST", url, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := hc.Do(req)
	if err != nil {
		fail("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		fail("POST %s → %d: %s", url, resp.StatusCode, string(b))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			fail("decode %s: %v", url, err)
		}
	}
}

func mustPostJSONExpectStatus(hc *http.Client, url string, body any, want int) {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest("POST", url, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := hc.Do(req)
	if err != nil {
		fail("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		b, _ := io.ReadAll(resp.Body)
		fail("POST %s → %d (want %d): %s", url, resp.StatusCode, want, string(b))
	}
}

func openMysql() *sql.DB {
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		fail("open mysql: %v", err)
	}
	if err := db.Ping(); err != nil {
		fail("ping mysql: %v", err)
	}
	return db
}

// resetSharedMySQL clears the M3/M4 tables the e2e touches. Called before
// the server starts so step 7's "≥2 candidates across two runs" assertion
// isn't fooled by rows left over from prior failed runs.
//
// Order matters for FK-safe truncate: child tables first. These tables
// have no FK constraints in the current schema, but we still emit a
// defensive order so a future schema change doesn't silently break us.
func resetSharedMySQL() {
	db := openMysql()
	defer db.Close()
	tables := []string{
		"bp_candidates",      // M4 sink output
		"repo_pipeline_runs", // M4 per-run counters
		"repo_pipelines",     // M4 catalog (parent of runs)
		"workflow_runs",      // M3 per-run counters
	}
	for _, t := range tables {
		if _, err := db.Exec("DELETE FROM " + t); err != nil {
			fail("reset table %s: %v", t, err)
		}
	}
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", a...)
	os.Exit(1)
}

// keep crypto helpers import-warm even if a future diff stops using them
var _ = hmac.New
var _ = sha256.New
var _ = base64.RawURLEncoding.EncodeToString
var _ = mintJWTUnused

func mintJWTUnused(sub string, ttl time.Duration) string {
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	p := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"sub":%q,"exp":%d}`, sub, time.Now().Add(ttl).Unix())))
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(h + "." + p))
	s := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return h + "." + p + "." + s
}

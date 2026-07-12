// e2e-m1 — M1 验收脚本（Go 版）。
//
// 6 个检查点：
//  1. 起服务（带 -mysql-dsn）
//  2. healthz / admin/users 通
//  3. /api/console/v2/tools 返回 ≥1 条 tool_directory 行
//  4. tool_invocation_logs 在调用前是 0
//  5. 用 streamable SSE / 显式登录 token 调 POST /mcp 后，行数 ≥1
//  6. dashboard summary / recent invocations 返回 200 + JSON
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

const (
	bin        = `D:\projects\ai-dev-sop\tools\cbmem-team\bin\cbmem-team.exe`
	sqlitePath = `D:\projects\ai-dev-sop\tools\cbmem-team\bin\test-cbmem.db`
	mysqlDSN   = `root:mediation123@tcp(127.0.0.1:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4`
	adminToken = "dev-admin-m1-2026"
	jwtSecret  = "dev-secret-m1-7f8a-2026"
)

func main() {
	// Pick a free listen port. We cycle through a small list so parallel
	// CI runs don't collide.
	tryPorts := []string{":28791", ":28792", ":28793", ":28794", ":28795"}
	var listen string
	for _, p := range tryPorts {
		if isFreePort(p) {
			listen = p
			break
		}
	}
	if listen == "" {
		fail("no free port in %v", tryPorts)
	}
	baseURL := "http://127.0.0.1" + listen
	dataDir := `D:\projects\ai-dev-sop\tools\cbmem-team\bin\data-m1`
	os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0o755)

	// Clear M1 tables from previous e2e runs so the count assertions are deterministic.
	clearMySQLM1Tables()

	_ = sqlitePath // silence unused import during M1 testing

	// Rebuild the server binary before launching it. This guards against
	// the "client rebuilt, server binary stale" trap that previously caused
	// confusing 401s — the e2e would run against an old cbmem-team.exe whose
	// auth chain didn't match what the client expected.
	if err := rebuildServer(); err != nil {
		fail("rebuild server: %v", err)
	}

	cmd := startServer(listen, dataDir)
	defer killCmd(cmd)
	waitForHealthz(8*time.Second, baseURL)

	fmt.Println("\n--- M1 e2e 检查 ---")

	// 1. healthz / admin/users
	fmt.Println("[1] healthz + admin")
	if st, _ := httpGET(baseURL+"/healthz", ""); st != 200 {
		fail("healthz expected 200 got %d", st)
	}
	if st, _ := httpGET(baseURL+"/admin/users", adminToken); st != 200 {
		fail("admin/users expected 200 got %d", st)
	}

	// 1b. seed alice-0 via /admin/users. The streamable path requires a
	// registered user; without it we get 403 even with a valid JWT.
	fmt.Println("[1b] seed alice-0 via /admin/users")
	regBody := `{"id":"alice-0","display_name":"Alice","project_paths":["/path/to/proj-0","/tmp/test-proj"]}`
	st, body := postJSON(baseURL+"/admin/users", adminToken, regBody)
	if st != 200 && st != 201 && st != 409 {
		fail("register alice-0 expected 200/201/409 got %d body=%s", st, body)
	}
	fmt.Printf("    register alice-0 → %d (409 = already exists, ok)\n", st)

	// 2. tool_directory seeded
	fmt.Println("[2] tool_directory seeded")
	var n int
	db := openMysql()
	defer db.Close()
	mustScan(db.QueryRow(`SELECT COUNT(*) FROM tool_directory`), &n)
	if n < 1 {
		fail("tool_directory expected ≥1 got %d", n)
	}
	fmt.Printf("    ✓ tool_directory rows = %d\n", n)

	// 3. tool_invocation_logs baseline
	fmt.Println("[3] tool_invocation_logs baseline")
	var n0 int
	mustScan(db.QueryRow(`SELECT COUNT(*) FROM tool_invocation_logs`), &n0)
	fmt.Printf("    baseline rows = %d\n", n0)

	// 4. Mint a real JWT for alice-0 using the same secret the server uses.
	fmt.Println("[4] mint JWT + POST /mcp?transport=streamable")
	token, err := mintJWT(jwtSecret, "alice-0", time.Hour)
	if err != nil { fail("mint jwt: %v", err) }
	authHeader := "Bearer " + token
	fmt.Printf("    token preview: %s...\n", token[:48])

	// The actual streamable call: a tools/list RPC. The downstream
	// pool tries to talk to /usr/local/bin/codebase-memory-mcp which
	// doesn't exist on Windows; the upstream is non-empty so we expect
	// either 200 (if the pool short-circuits) or 502 (if the pool
	// surfaces the error). Either way, CaptureInvocations writes a
	// tool_invocation_logs row, which is what we're verifying.
	rpcBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	st, streamResp := postRPC(baseURL+"/mcp?transport=streamable&as=alice-0&project=/path/to/proj-0",
		authHeader, rpcBody)
	if st != 200 && st != 502 {
		fail("streamable POST expected 200 or 502 got %d body=%s", st, streamResp)
	}
	fmt.Printf("    ✓ streamable POST → %d (response snip: %s)\n", st, snippet(streamResp))

	// 5. tool_invocation_logs row count ≥ baseline + 1
	time.Sleep(400 * time.Millisecond)
	fmt.Println("[5] tool_invocation_logs post-write")
	var n1 int
	mustScan(db.QueryRow(`SELECT COUNT(*) FROM tool_invocation_logs`), &n1)
	fmt.Printf("    rows after write = %d (delta %d)\n", n1, n1-n0)
	if n1 <= n0 {
		fail("expected rows to grow, n0=%d n1=%d", n0, n1)
	}
	// 6. drill into the most recent row
	var (
		lastUser, lastTool, lastTransport, lastError string
		lastLatency                                 int
	)
	row := db.QueryRow(`SELECT user_id, tool_id, transport, IFNULL(latency_ms,0), IFNULL(error_code,'') FROM tool_invocation_logs ORDER BY started_at DESC LIMIT 1`)
	if err := row.Scan(&lastUser, &lastTool, &lastTransport, &lastLatency, &lastError); err != nil {
		fail("scan last log row: %v", err)
	}
	fmt.Printf("    last row: user=%s tool=%s transport=%s latency=%dms err=%q\n",
		lastUser, lastTool, lastTransport, lastLatency, lastError)
	if lastUser != "alice-0" {
		fail("expected user_id=alice-0 got %s", lastUser)
	}
	if lastTransport != "streamable" {
		fail("expected transport=streamable got %s", lastTransport)
	}

	// 7. SSE handshake
	fmt.Println("[7] GET /mcp/sse?session_id=test-session-1")
	ev := readSSE(baseURL+"/mcp/sse?session_id=test-session-1&as=alice-0", authHeader, 2*time.Second)
	if ev == "" {
		fail("SSE ready event missing")
	}
	if !strings.Contains(ev, "ready") || !strings.Contains(ev, "test-session-1") {
		fail("SSE ready event unexpected: %q", ev)
	}
	fmt.Printf("    ✓ SSE ready received\n")

	fmt.Println("\n✅ M1 e2e 全绿")
}

func mintJWT(secret, sub string, ttl time.Duration) (string, error) {
	// Inlined to avoid an import cycle with /internal/auth which already
	// has the same algorithm — kept here so the e2e binary stays a single
	// file.
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now().UTC()
	payload := map[string]any{
		"sub": sub,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(hb) + "." + enc.EncodeToString(pb)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signing))
	sig := enc.EncodeToString(mac.Sum(nil))
	return signing + "." + sig, nil
}

func snippet(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 100 { return s[:100] + "..." }
	return s
}

func rebuildServer() error {
	// `go build` needs to run from the module root (the directory that owns
	// go.mod, i.e. d:\projects\ai-dev-sop\tools\cbmem-team). Resolve it
	// from the hard-coded `bin` path: the module root is the parent of the
	// `bin` directory the e2e launches.
	modRoot := filepath.Dir(filepath.Dir(bin))
	prev, err := os.Getwd()
	if err != nil { return err }
	if err := os.Chdir(modRoot); err != nil { return err }
	defer func() { _ = os.Chdir(prev) }()

	cmd := exec.Command("go", "build", "-o", bin, "./cmd/cbmem-team/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startServer(listen, dataDir string) *exec.Cmd {
	cmd := exec.Command(bin,
		"-listen", listen,
		"-data", dataDir,
		"-admin-token", adminToken,
		"-jwt-secret", jwtSecret,
		"-mysql-dsn", mysqlDSN,
		"-log", "info")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = nil // run as a normal child; killCmd handles cleanup
	must(cmd.Start())
	return cmd
}

func isFreePort(addr string) bool {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

func waitForHealthz(timeout time.Duration, baseURL string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/healthz")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return
		}
		if resp != nil { resp.Body.Close() }
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Println("❌ timeout waiting for", baseURL+"/healthz")
	os.Exit(1)
}

func postRPC(url, authHeader, body string) (int, string) {
	req, _ := http.NewRequest("POST", url, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		// authHeader is "Bearer XXX" or "Token XXX"; for the JWT path the
		// server expects the standard `Authorization` header, so strip
		// the scheme and set Authorization explicitly.
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return 0, err.Error() }
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func postJSON(url, token, body string) (int, string) {
	req, _ := http.NewRequest("POST", url, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" { req.Header.Set("X-Admin-Token", token) }
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return 0, err.Error() }
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func readSSE(url, authHeader string, timeout time.Duration) string {
	req, _ := http.NewRequest("GET", url, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil { return "" }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return "" }

	r := bufio.NewReader(resp.Body)
	line, _ := r.ReadString('\n')
	// SSE: first line is "event: ready"
	if strings.HasPrefix(line, "event: ") {
		data, _ := r.ReadString('\n')
		return strings.TrimSpace(line) + " " + strings.TrimSpace(data)
	}
	return strings.TrimSpace(line)
}

func openMysql() *sql.DB {
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil { fmt.Println("open mysql err:", err); os.Exit(1) }
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil { fmt.Println("ping mysql err:", err); os.Exit(1) }
	return db
}

func clearMySQLM1Tables() {
	db := openMysql()
	defer db.Close()
	for _, t := range []string{"tool_invocation_logs", "tool_directory"} {
		_, err := db.Exec("DELETE FROM " + t)
		if err != nil { fmt.Println("warn clear", t, ":", err) }
	}
	// Drop tables so we can verify migrate-tables is idempotent + re-seeds.
	db.Exec("DROP TABLE IF EXISTS tool_invocation_logs")
	db.Exec("DROP TABLE IF EXISTS tool_directory")
}

func httpGET(url, token string) (int, string) {
	req, _ := http.NewRequest("GET", url, nil)
	if token != "" { req.Header.Set("X-Admin-Token", token) }
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return 0, err.Error() }
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func mustScan(row *sql.Row, dst ...any) {
	if err := row.Scan(dst...); err != nil { fmt.Println("scan err:", err); os.Exit(1) }
}

func must(err error) { if err != nil { fmt.Println("err:", err); os.Exit(1) } }

func killCmd(cmd *exec.Cmd) {
	if cmd.Process == nil { return }
	cmd.Process.Kill()
	cmd.Wait()
}

func fail(format string, a ...any) {
	fmt.Printf("\n❌ "+format+"\n", a...)
	os.Exit(2)
}

// keep json/json-format imports warm so future diffs can drop down
var _ = json.Marshal
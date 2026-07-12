// e2e-v5 — M0.5 验收脚本（Go 版）。
//
// 5 阶段：
//  1. SQLite 路径 serve（无 -mysql-dsn），healthz + admin/users 200
//  2. MySQL 路径 serve（带 -mysql-dsn），同上
//  3. migrate-tables 幂等验证（跑 2 次）
//  4. migrate-sqlite-to-mysql 行数 1:1 验证
//  5. ETL 后 MySQL 行数与 SQLite 行数一致
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"

	"cbmem-team/internal/devconf"
)

const (
	bin        = `D:\projects\ai-dev-sop\tools\cbmem-team\bin\cbmem-team.exe`
	sqlitePath = `D:\projects\ai-dev-sop\tools\cbmem-team\bin\test-cbmem.db`
	adminToken = "dev-admin-token-e2e-v5-2026"
	jwtSecret  = "dev-secret-e2e-v5-2026-7f8a"
	listen     = ":18787"
	dataDir    = `D:\projects\ai-dev-sop\tools\cbmem-team\bin\data-e2e`
)

// mysqlDSN is resolved in main() from flag/env via internal/devconf so
// CI / local devs can override without editing source.
var mysqlDSN string

func main() {
	flagDSN := flag.String("mysql-dsn", "", "MySQL DSN; if empty, falls back to $CBMEM_MYSQL_DSN then $DSN then the dev default")
	flag.Parse()
	mysqlDSN = devconf.ResolveMySQLDSNWithDB(*flagDSN, devconf.DefaultDevMySQLDB)

	os.MkdirAll(dataDir, 0755)

	// Seed SQLite test DB if missing (idempotent across runs)
	if _, err := os.Stat(sqlitePath); err != nil {
		mustRun(exec.Command("go", "run", "./cmd/seed-sqlite/"))
	}

	// 阶段 1: SQLite path
	step1SQLite()

	// 阶段 2: MySQL path
	step2MySQL()

	// 阶段 3: migrate-tables idempotent
	step3MigrateTables()

	// 阶段 4+5: ETL + row count parity
	step4ETL()

	fmt.Println("\n✅ e2e-v5 全绿")
}

func step1SQLite() {
	fmt.Println("\n=== 阶段 1: SQLite 路径 serve ===")
	cmd := exec.Command(bin,
		"-listen", listen,
		"-data", dataDir,
		"-admin-token", adminToken,
		"-jwt-secret", jwtSecret,
		"-log", "info")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Start())
	defer killCmd(cmd)

	waitForHealthz(5*time.Second, "127.0.0.1:18787/healthz")

	// /healthz
	status, body := httpGET("http://127.0.0.1:18787/healthz", "")
	check(200, status, "GET /healthz", body)

	// /admin/users with admin token
	status, body = httpGET("http://127.0.0.1:18787/admin/users", adminToken)
	check(200, status, "GET /admin/users", body)
}

func step2MySQL() {
	fmt.Println("\n=== 阶段 2: MySQL 路径 serve ===")
	cmd := exec.Command(bin,
		"-listen", listen,
		"-data", dataDir,
		"-admin-token", adminToken,
		"-jwt-secret", jwtSecret,
		"-mysql-dsn", mysqlDSN,
		"-log", "info")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Start())
	defer killCmd(cmd)

	waitForHealthz(8*time.Second, "127.0.0.1:18787/healthz")

	status, body := httpGET("http://127.0.0.1:18787/healthz", "")
	check(200, status, "GET /healthz (mysql)", body)
	status, body = httpGET("http://127.0.0.1:18787/admin/users", adminToken)
	check(200, status, "GET /admin/users (mysql)", body)
}

func step3MigrateTables() {
	fmt.Println("\n=== 阶段 3: migrate-tables 幂等 ===")
	mustRun(exec.Command(bin, "migrate-tables", "-mysql-dsn", mysqlDSN))
	mustRun(exec.Command(bin, "migrate-tables", "-mysql-dsn", mysqlDSN))
	fmt.Println("✓ migrate-tables 跑两次都成功")
}

func step4ETL() {
	fmt.Println("\n=== 阶段 4+5: ETL + 行数 1:1 校对 ===")
	// Clear MySQL first to make parity check meaningful
	clearMySQLTables()
	mustRun(exec.Command(bin, "migrate-tables", "-mysql-dsn", mysqlDSN))
	mustRun(exec.Command(bin, "migrate-sqlite-to-mysql",
		"-sqlite", sqlitePath,
		"-mysql-dsn", mysqlDSN))

	tables := []string{"users", "projects", "sessions", "session_turns",
		"summarize_tasks", "distill_tasks", "console_sessions"}
	srcDB := mustOpen(sqlitePath, "sqlite")
	defer srcDB.Close()
	dstDB := mustOpen(mysqlDSN, "mysql")
	defer dstDB.Close()

	for _, t := range tables {
		var srcN, dstN int64
		must(srcDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", t)).Scan(&srcN))
		must(dstDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", t)).Scan(&dstN))
		ok := "✓"
		if srcN != dstN {
			ok = "❌"
		}
		fmt.Printf("  %-20s src=%-4d dst=%-4d %s\n", t, srcN, dstN, ok)
		if srcN != dstN {
			fmt.Println("FAIL: row count mismatch on", t)
			os.Exit(5)
		}
	}
}

func clearMySQLTables() {
	db := mustOpen(mysqlDSN, "mysql")
	defer db.Close()
	for _, t := range []string{
		"session_turns", "summarize_tasks", "distill_tasks",
		"console_sessions", "sessions", "projects", "users"} {
		_, err := db.Exec("DELETE FROM " + t)
		if err != nil {
			fmt.Println("warning: clear", t, err)
		}
	}
}

func httpGET(url, token string) (int, string) {
	req, _ := http.NewRequest("GET", url, nil)
	if token != "" {
		req.Header.Set("X-Admin-Token", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func check(want, got int, desc, body string) {
	if want != got {
		fmt.Printf("❌ %s: want %d got %d\n%s\n", desc, want, got, body)
		os.Exit(6)
	}
	var pretty string
	if len(body) < 200 && json.Valid([]byte(body)) {
		pretty = body
	} else if strings.HasPrefix(body, "{") {
		pretty = body[:min(len(body), 200)]
	} else {
		pretty = body
	}
	fmt.Printf("✓ %s → %d  %s\n", desc, got, pretty)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func waitForHealthz(timeout time.Duration, addr string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Println("❌ timeout waiting for", addr)
	os.Exit(7)
}

func mustOpen(dsn, driver string) *sql.DB {
	db, err := sql.Open(driver, wrapDSN(driver, dsn))
	if err != nil {
		fmt.Println("open err:", err)
		os.Exit(8)
	}
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		fmt.Println("ping err:", err)
		os.Exit(9)
	}
	return db
}

func wrapDSN(driver, dsn string) string {
	if driver == "sqlite" {
		return fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", dsn)
	}
	return dsn
}

func must(err error) {
	if err != nil {
		fmt.Println("err:", err)
		os.Exit(10)
	}
}

func mustRun(cmd *exec.Cmd) {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("run err:", err)
		os.Exit(11)
	}
}

func killCmd(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	cmd.Process.Kill()
	cmd.Wait()
}

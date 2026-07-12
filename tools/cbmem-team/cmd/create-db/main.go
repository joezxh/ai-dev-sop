// create-db — one-shot: create the `cbmem` schema with utf8mb4 if absent.
// Used by Gate M0.5 verification; not built into production binary.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"cbmem-team/internal/devconf"
)

func main() {
	flagDSN := flag.String("mysql-dsn", "", "MySQL DSN; if empty, falls back to $CBMEM_MYSQL_DSN then $DSN then "+devconf.DefaultDevMySQLDB+" default")
	flag.Parse()

	bootstrapDSN := devconf.ResolveMySQLDSN(*flagDSN)
	cbmemDSN := devconf.ResolveMySQLDSNWithDB(*flagDSN, "cbmem")

	db, err := sql.Open("mysql", bootstrapDSN)
	if err != nil {
		fmt.Println("open err:", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Idempotent CREATE. MySQL 9.0 still rejects `IF NOT EXISTS` on
	// CHARACTER SET / COLLATE clauses inside CREATE DATABASE, so we issue
	// two statements: one for the DB itself (with default utf8mb4_0900_ai_ci
	// per server collation), then ALTER DATABASE to enforce our target
	// collation.
	stmts := []string{
		`CREATE DATABASE IF NOT EXISTS cbmem CHARACTER SET utf8mb4`,
		`ALTER DATABASE cbmem CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			fmt.Printf("err on %s:\n  %v\n", s, err)
			os.Exit(2)
		}
		fmt.Println("ok:", s)
	}

	// Re-connect WITH the schema selected and verify
	db2, err := sql.Open("mysql", cbmemDSN)
	if err != nil {
		fmt.Println("reopen err:", err)
		os.Exit(3)
	}
	defer db2.Close()
	var n int
	if err := db2.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='cbmem'").Scan(&n); err != nil {
		fmt.Println("count tables err:", err)
		os.Exit(4)
	}
	fmt.Printf("DB cbmem ready, current table count = %d\n", n)
}

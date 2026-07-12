// Quick MySQL probe — verifies the connection works and reports server
// characteristics. Used by Gate M0.5 verification only; not built into
// the production binary.
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
	flagDSN := flag.String("mysql-dsn", "", "MySQL DSN; if empty, falls back to $CBMEM_MYSQL_DSN then $DSN then the dev default")
	flag.Parse()

	dsn := devconf.ResolveMySQLDSN(*flagDSN)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("open err:", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		ver, osName   string
		charset, coll string
		flushLog      int
	)
	if err := db.QueryRowContext(ctx, "SELECT VERSION(), @@version_compile_os, @@character_set_server, @@collation_server, @@innodb_flush_log_at_trx_commit").Scan(&ver, &osName, &charset, &coll, &flushLog); err != nil {
		fmt.Println("query err:", err)
		os.Exit(2)
	}
	fmt.Printf("MySQL version : %s (%s)\n", ver, osName)
	fmt.Printf("Charset/Coll  : %s / %s\n", charset, coll)
	fmt.Printf("Flush log trx : %d\n", flushLog)

	var dbExists int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = 'cbmem'").Scan(&dbExists); err != nil {
		fmt.Println("schema query err:", err)
		os.Exit(3)
	}
	fmt.Printf("DB cbmem      : %s\n", map[int]string{0: "missing", 1: "exists"}[dbExists])
}

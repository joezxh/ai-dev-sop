// show-tables — diagnostic: list tables + their engines/collations.
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

	dsn := devconf.ResolveMySQLDSNWithDB(*flagDSN, devconf.DefaultDevMySQLDB)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("open err:", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT table_name, engine, table_collation,
		       (SELECT COUNT(*) FROM information_schema.columns c WHERE c.table_schema='cbmem' AND c.table_name=t.table_name) cols,
		       ROUND((data_length+index_length)/1024, 1) AS kb
		FROM information_schema.tables t
		WHERE table_schema='cbmem'
		ORDER BY table_name`)
	if err != nil {
		fmt.Println("err:", err)
		os.Exit(1)
	}
	defer rows.Close()
	fmt.Printf("%-20s %-10s %-30s %4s %6s\n", "TABLE", "ENGINE", "COLLATION", "COLS", "KB")
	for rows.Next() {
		var name, engine, coll string
		var cols int
		var kb float64
		rows.Scan(&name, &engine, &coll, &cols, &kb)
		fmt.Printf("%-20s %-10s %-30s %4d %6.1f\n", name, engine, coll, cols, kb)
	}
}

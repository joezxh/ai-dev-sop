// show-indexes — diagnostic
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

	rows, _ := db.QueryContext(ctx, `
		SELECT table_name, index_name, GROUP_CONCAT(column_name ORDER BY seq_in_index)
		FROM information_schema.statistics
		WHERE table_schema='cbmem'
		GROUP BY table_name, index_name
		ORDER BY table_name, index_name`)
	defer rows.Close()
	fmt.Printf("%-20s %-32s %s\n", "TABLE", "INDEX", "COLUMNS")
	for rows.Next() {
		var t, idx, cols string
		rows.Scan(&t, &idx, &cols)
		fmt.Printf("%-20s %-32s %s\n", t, idx, cols)
	}
}

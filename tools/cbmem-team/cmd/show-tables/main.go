// show-tables — diagnostic: list tables + their engines/collations.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "root:mediation123@tcp(127.0.0.1:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4"
	db, _ := sql.Open("mysql", dsn)
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

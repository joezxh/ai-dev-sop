// show-indexes — diagnostic
package main

import (
	"context"
	"database/sql"
	"fmt"
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

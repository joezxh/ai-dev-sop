// verify_sqlite applies tools/cbmem-team/deploy/sql/{schema,seed}.sql
// to an in-memory SQLite database and reports the resulting tables +
// tool_directory row count + a spot-check on mempalace_add_drawer.
//
// Run from the repo root:
//
//     go run ./tools/cbmem-team/cmd/_dev/verify_sqlite tools/cbmem-team/deploy/sql
//
// Exits 0 on success, 1 on any error. Used by the deploy/sql README to
// confirm regenerated SQL files still parse and the seed round-trips.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: verify_sqlite <dir-with-schema.sql-and-seed.sql>")
		os.Exit(2)
	}
	dir := os.Args[1]
	dsn := "file::memory:?cache=shared&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()
	ctx := context.Background()

	for _, name := range []string{"schema.sql", "seed.sql"} {
		path := dir + "/" + name
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("%s: read: %v\n", name, err)
			os.Exit(1)
		}
		if _, err := db.ExecContext(ctx, string(b)); err != nil {
			fmt.Printf("%s: apply: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("%s: applied OK (%d bytes)\n", name, len(b))
	}

	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		fmt.Println("list tables:", err)
		os.Exit(1)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var n string
		rows.Scan(&n)
		fmt.Println("  table:", n)
		count++
	}
	fmt.Printf("total tables: %d\n", count)

	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tool_directory").Scan(&n); err != nil {
		fmt.Println("count tool_directory:", err)
		os.Exit(1)
	}
	fmt.Printf("tool_directory rows: %d\n", n)

	var toolID, track, cat string
	if err := db.QueryRowContext(ctx,
		"SELECT tool_id, track, category FROM tool_directory WHERE tool_id = ?",
		"mempalace_add_drawer",
	).Scan(&toolID, &track, &cat); err != nil {
		fmt.Println("spot-check:", err)
		os.Exit(1)
	}
	fmt.Printf("spot-check: %s [%s/%s]\n", toolID, track, cat)
}
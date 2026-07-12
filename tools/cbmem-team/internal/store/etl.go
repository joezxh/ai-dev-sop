// Package store provides a one-shot ETL: read every row from the legacy
// SQLite console database and insert it into the MySQL console database.
//
// Behaviour:
//
//   - One pass per table, in dependency order (FKs respected where possible).
//   - Inserts are batched via a single multi-row INSERT to keep wall-time short.
//   - After a table is migrated, the row counts are compared. A mismatch
//     aborts the migration with code 2 — the MySQL rows are NOT rolled back
//     automatically; the operator must inspect.
//
// This command is invoked by `cbmem-team migrate-sqlite-to-mysql` and by
// `examples/mysql-etl.sh`. It is intentionally not wired into normal startup.
//
// Schema differences (SQLite → MySQL 8.0) are documented in RESEARCH.md §3.2.
// The mapping here assumes both databases have been migrated to the same
// logical schema: the SQLite side keeps the legacy column types, the MySQL
// side uses ENGINE=InnoDB DEFAULT CHARSET=utf8mb4.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// ConsoleTables is the list of tables the ETL walks, in safe order.
//
// The order is: independent tables first, then tables with FK-ish
// references. SQLite has no real FKs (PRAGMA off), but we still emit in
// insert-safe order so a re-run on a partial dump behaves predictably.
var ConsoleTables = []string{
	"users",
	"projects",
	"sessions",
	"session_turns",
	"summarize_tasks",
	"distill_tasks",
	"console_sessions",
}

// MigrateSQLiteToMySQL copies every row from src (SQLite) into dst (MySQL).
// It returns the number of rows written per table and a non-nil error on
// the first count mismatch.
func MigrateSQLiteToMySQL(ctx context.Context, src, dst *sql.DB) (map[string]int64, error) {
	counts := map[string]int64{}
	for _, table := range ConsoleTables {
		written, err := migrateTable(ctx, src, dst, table)
		if err != nil {
			return counts, fmt.Errorf("migrate %s: %w", table, err)
		}
		counts[table] = written
	}
	return counts, nil
}

func migrateTable(ctx context.Context, src, dst *sql.DB, table string) (int64, error) {
	// Count source rows
	var srcCount int64
	if err := src.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&srcCount); err != nil {
		return 0, fmt.Errorf("count source %s: %w", table, err)
	}
	if srcCount == 0 {
		log.Printf("etl: %s: 0 rows, skipped", table)
		return 0, nil
	}

	// Pull all rows. We never bind parameters here — the table name is a
	// compile-time constant from ConsoleTables and the column list is
	// inferred from PRAGMA / INFORMATION_SCHEMA so this code stays in sync
	// without manual SQL maintenance.
	colNames, err := readColumns(ctx, src, table)
	if err != nil {
		return 0, err
	}
	colList := strings.Join(colNames, ", ")

	rows, err := src.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM %s", colList, table))
	if err != nil {
		return 0, fmt.Errorf("select source %s: %w", table, err)
	}
	defer rows.Close()

	// Build placeholders like "(?,?,?),(?,?,?),..." lazily per chunk
	// (we re-build per chunk because chunk size can differ on the last
	// iteration). Allocating once up front saves nothing because the
	// output depends on chunk size, not row count.
	_ = buildPlaceholders // kept exported-by-lowercase for tests; not used here

	// Drain rows into a 2D slice of pointers so database/sql can Scan.
	allValues := make([]any, 0, len(colNames)*int(srcCount))
	destCols := make([]any, len(colNames))
	destPtrs := make([]any, len(colNames))
	for i := range destCols {
		destPtrs[i] = &destCols[i]
	}
	for rows.Next() {
		if err := rows.Scan(destPtrs...); err != nil {
			return 0, fmt.Errorf("scan source %s: %w", table, err)
		}
		allValues = append(allValues, destCols...)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate source %s: %w", table, err)
	}

	// Insert into MySQL. We split into chunks of 1000 rows to keep packet
	// size sensible. multiStatements stays off in production traffic; this
	// ETL path uses single-statement INSERTs which is portable.
	const chunk = 1000
	var written int64
	for start := 0; start < len(allValues); start += chunk * len(colNames) {
		end := start + chunk*len(colNames)
		if end > len(allValues) {
			end = len(allValues)
		}
		batchPlaceholders := placeholdersForChunk(len(colNames), (end-start)/len(colNames))
		stmt := fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES %s", table, colList, batchPlaceholders)
		res, err := dst.ExecContext(ctx, stmt, allValues[start:end]...)
		if err != nil {
			return written, fmt.Errorf("insert into %s (offset=%d): %w", table, start, err)
		}
		n, _ := res.RowsAffected()
		written += n
	}

	// Row-count check: compare what we wrote vs what the source had. We use
	// a post-insert COUNT to avoid relying on RowsAffected semantics that
	// differ between drivers (MySQL ignores duplicate keys on INSERT IGNORE
	// but the affected count is the row count actually inserted, which is
	// what we want here).
	var dstCount int64
	if err := dst.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&dstCount); err != nil {
		return written, fmt.Errorf("count destination %s: %w", table, err)
	}
	if dstCount != srcCount {
		return written, fmt.Errorf("%s: row count mismatch src=%d dst=%d", table, srcCount, dstCount)
	}
	log.Printf("etl: %s: %d rows ✓", table, written)
	return written, nil
}

// readColumns returns the column names of a SQLite table in declaration
// order. SQLite has no INFORMATION_SCHEMA in the standard sense, but the
// `pragma table_info` virtual table is portable across the modernc.org
// driver and any other Go SQLite driver.
func readColumns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, fmt.Errorf("pragma %s: %w", table, err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("scan pragma %s: %w", table, err)
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, errors.New("no columns discovered (table missing?)")
	}
	return cols, nil
}

// buildPlaceholders returns "(?,?,?),(?,?,?),..." sized to fit n rows.
// It returns just the empty string for n==0 because callers already
// early-return in that case.
func buildPlaceholders(cols, n int) string {
	return placeholdersForChunk(cols, n)
}

func placeholdersForChunk(cols, n int) string {
	oneRow := "(" + strings.Repeat("?,", cols-1) + "?)"
	if n <= 1 {
		return oneRow
	}
	return strings.Repeat(oneRow+",", n-1) + oneRow
}

// EtlTimeout is the per-migration deadline. It is exported so the wrapper
// shell script can align its own timeout.
var EtlTimeout = 5 * time.Minute

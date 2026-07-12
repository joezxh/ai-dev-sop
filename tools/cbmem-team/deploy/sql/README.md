# cbmem-team console — SQL reference

This directory ships the schema DDL and initial data as plain SQL files so an
operator can bring up the console database **without compiling the Go binary**:

```bash
# SQLite (default)
sqlite3 /var/lib/cbmem-team/console.db < schema.sql
sqlite3 /var/lib/cbmem-team/console.db < seed.sql

# MySQL 8.0
mysql --user=root --password console < schema-mysql.sql
mysql --user=root --password console < seed-mysql.sql
```

> The Go binary's `cbmem-team migrate` subcommand remains the
> authoritative path for **upgrades** — it handles the probe-based
> idempotency for `CREATE INDEX` (MySQL 8.0 has no `IF NOT EXISTS`) and
> the DROP-then-CREATE pattern for views. The SQL files in this
> directory are written for **fresh databases only**.

## Files

| File | What it does |
|------|--------------|
| `schema.sql` | SQLite DDL: 7 legacy tables + M1–M4 (tool_directory, best_practices, workflows, repo_pipelines, bp_candidates, …). Idempotent. |
| `schema-mysql.sql` | MySQL 8.0 DDL, same logical tables. Uses `CREATE TABLE IF NOT EXISTS`; indexes are bare `CREATE INDEX` (must run on a fresh DB). |
| `seed.sql` | 49-row `tool_directory` catalog for SQLite. BP + workflow seeds require `cbmem-team migrate-seed` (see below). |
| `seed-mysql.sql` | Same 49 rows, MySQL syntax. |
| `dump_sql.py` | Generator that mirrors the in-Go schema strings + `ToolSeed49` into the files above. Re-run after every schema change. |

## What `seed.sql` does NOT cover

The best-practices (3 rows) and built-in workflows (3 rows) seeds live as Go
struct literals (`SeedBPs`, `SeedWorkflows`) because their JSON columns hold
structured payloads that are easier to maintain in Go than as raw SQL
literals. To materialise them after applying the schema/seed files:

```bash
cbmem-team migrate-seed
```

`migrate-seed` is idempotent — re-runs are no-ops once the tables have rows.

## How these files are kept in sync

The Go source of truth lives in:

- `tools/cbmem-team/internal/console/db.go` — `migrateConsole` (SQLite)
  and `mysqlConsoleDDL` / `mysqlConsoleIndexes` (MySQL).
- `tools/cbmem-team/internal/console/tools.go` — `M1ExtraSchema` + `ToolSeed49`.
- `tools/cbmem-team/internal/console/bp.go` — `M2ExtraSchema` + `SeedBPs`.
- `tools/cbmem-team/internal/console/workflow.go` — `M3ExtraSchema`.
- `tools/cbmem-team/internal/console/workflow_seeds.go` — `WorkflowSeedBuiltins`.
- `tools/cbmem-team/internal/console/repo_pipeline.go` — `M4ExtraSchema`.

When any of these change:

```bash
python tools/cbmem-team/deploy/sql/dump_sql.py
go run ./cmd/cbmem-team migrate        # ensure apply path still works
go run ./cmd/e2e-v7                    # e2e regression
```

`dump_sql.py` is a small parser that copies the backtick-quoted SQL strings
from the Go files (it does **not** re-parse the AST), so the generator and
the binary stay aligned by construction. The Python script has no runtime
dependencies (stdlib only).

## MySQL caveats (read before deploying)

1. **`CREATE INDEX` is not idempotent.** Re-running `schema-mysql.sql` on an
   existing database will fail with `ERROR 1061 (42000): duplicate key name`.
   Use `cbmem-team migrate` (which probes `information_schema.statistics`).

2. **`CREATE VIEW IF NOT EXISTS` is not supported.** The schema file uses
   `DROP VIEW IF EXISTS` + `CREATE VIEW` so a re-run works, but you lose the
   view's grant list. The Go binary uses the same pattern.

3. **`utf8mb4_unicode_ci` collation is required.** MySQL 8.0 defaults to
   `utf8mb4_0900_ai_ci`; the explicit `COLLATE=utf8mb4_unicode_ci` matches
   what the binary sets in its `CREATE DATABASE` step.

4. **JSON columns** (`best_practices.tools`, `best_practices.related_halls`,
   `workflows.nodes_json`, …) must be populated via `CAST('…' AS JSON)`.
   The Go binary does this in `MarshalJSON`-style helpers. Direct `INSERT`
   with a string literal will succeed (MySQL auto-converts) but is slower
   because no JSON validation runs.

## Verification

To verify a regenerated `schema.sql` still parses cleanly:

```bash
# from the repo root or anywhere, as long as the working directory is
# inside the cbmem-team module (so modernc.org/sqlite resolves)
cd tools/cbmem-team
go run ./cmd/_dev/verify_sqlite deploy/sql
```

This is the same verification we ran while writing this directory; it
applied both files to an in-memory DB and confirmed 18 business tables +
49 `tool_directory` rows + a spot-check on `mempalace_add_drawer`.
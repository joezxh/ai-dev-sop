#!/usr/bin/env python3
"""
dump_sql.py — emit deploy/sql/{schema,seed,schema-mysql,seed-mysql}.sql
from the in-Go schema strings + seed variables.

This script is the single source of truth for the SQL files. The Go code
owns the canonical schema/seed definitions; we *mirror* them here so an
operator can `sqlite3 console.db < schema.sql && sqlite3 console.db < seed.sql`
without compiling the binary.

Run from repo root:
    python tools/cbmem-team/deploy/sql/dump_sql.py
"""
from __future__ import annotations

import os
import re
import sys
from pathlib import Path

CONSOLE = Path("tools/cbmem-team/internal/console")
OUT = Path("tools/cbmem-team/deploy/sql")


# -----------------------------------------------------------------------------
# 1. Extract hardcoded SQLite DDL from db.go::migrateConsole() and the
#    ApplySQLite closures of M1..M4ExtraSchema().
# -----------------------------------------------------------------------------

def extract_stmts_from_apply_sqlite(go_src: str) -> list[str]:
    """Pull each backtick-quoted SQL string out of an ApplySQLite closure.

    The Go code uses raw strings like `CREATE TABLE ...`. We slice every
    `` ` `` … `` ` `` pair that sits between `ApplySQLite: func...` and
    the next top-level token of the struct.
    """
    # Walk through the file, find ApplySQLite blocks, capture strings inside.
    # Pattern: ApplySQLite: func(ctx context.Context, db *DB) error {
    #              stmts := []string{ `...`, `...`, ... }
    #          }
    # We use a simple state machine over braces to find the stmts list.
    out = []
    cursor = 0
    while True:
        idx = go_src.find("ApplySQLite", cursor)
        if idx == -1:
            break
        # find next `func(ctx context.Context, db *DB) error {`
        brace = go_src.find("{", idx)
        depth = 1
        end = brace + 1
        while depth > 0 and end < len(go_src):
            ch = go_src[end]
            if ch == "{":
                depth += 1
            elif ch == "}":
                depth -= 1
            end += 1
        body = go_src[brace + 1 : end - 1]
        # Now extract every backtick-quoted SQL string.
        for m in re.finditer(r"`([^`]+)`", body):
            sql = m.group(1).strip()
            if not sql:
                continue
            # Heuristic: real schema statements start with CREATE/INSERT/DROP.
            if re.match(r"^(CREATE|ALTER|DROP|INSERT)", sql, re.IGNORECASE):
                out.append(sql)
        cursor = end
    return out


def extract_migrate_console(go_src: str) -> list[str]:
    """Extract the 7-legacy SQLite DDL statements from migrateConsole()."""
    fn_idx = go_src.find("func (db *DB) migrateConsole(ctx context.Context) error")
    if fn_idx == -1:
        return []
    brace = go_src.find("{", fn_idx)
    depth = 1
    end = brace + 1
    while depth > 0 and end < len(go_src):
        ch = go_src[end]
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
        end += 1
    body = go_src[brace + 1 : end - 1]
    return [m.group(1).strip() for m in re.finditer(r"`([^`]+)`", body)]


def extract_mysql_ddl(db_go: str) -> tuple[list[str], list[tuple[str, str, str]]]:
    """Return (mysqlConsoleDDL strings, (table, name, ddl) index tuples)."""
    # mysqlConsoleDDL = []string{ ... }
    ddl_block_match = re.search(
        r"var mysqlConsoleDDL\s*=\s*\[\]string\{(.*?)\n\}",
        db_go,
        re.DOTALL,
    )
    if not ddl_block_match:
        raise RuntimeError("mysqlConsoleDDL block not found")
    ddl_block = ddl_block_match.group(1)
    ddls = [m.group(1).strip() for m in re.finditer(r"`([^`]+)`", ddl_block)]

    # mysqlConsoleIndexes = []indexSpec{ { ... , `...` }, ... }
    idx_block_match = re.search(
        r"var mysqlConsoleIndexes\s*=\s*\[\]indexSpec\{(.*?)\n\}",
        db_go,
        re.DOTALL,
    )
    if not idx_block_match:
        raise RuntimeError("mysqlConsoleIndexes block not found")
    idx_block = idx_block_match.group(1)
    # Each entry: {"table", "name", `ddl`}
    indexes = []
    for em in re.finditer(
        r"\{\s*\"([^\"]+)\"\s*,\s*\"([^\"]+)\"\s*,\s*`([^`]+)`\s*\}", idx_block
    ):
        indexes.append((em.group(1), em.group(2), em.group(3).strip()))
    return ddls, indexes


def extract_apply_mysql(go_src: str) -> list[str]:
    """Same idea as ApplySQLite but for ApplyMySQL — captures CREATE TABLE / INDEX / VIEW."""
    out = []
    cursor = 0
    while True:
        idx = go_src.find("ApplyMySQL", cursor)
        if idx == -1:
            break
        brace = go_src.find("{", idx)
        depth = 1
        end = brace + 1
        while depth > 0 and end < len(go_src):
            ch = go_src[end]
            if ch == "{":
                depth += 1
            elif ch == "}":
                depth -= 1
            end += 1
        body = go_src[brace + 1 : end - 1]
        for m in re.finditer(r"`([^`]+)`", body):
            sql = m.group(1).strip()
            if not sql:
                continue
            if re.match(r"^(CREATE|ALTER|DROP|INSERT)", sql, re.IGNORECASE):
                out.append(sql)
        cursor = end
    return out


# -----------------------------------------------------------------------------
# 2. Extract seed data — ToolSeed49, BPRecord seeds, WorkflowSeedBuiltins.
# -----------------------------------------------------------------------------

def extract_tool_seeds(tools_go: str) -> list[list[str]]:
    """ToolSeed49 is `var ToolSeed49 = []ToolRecord{ {"x","A","read",...}, ... }`.

    Each row is `{"<tool_id>","<track>","<category>","<name>","<signature>","<status>",<priority>,"<version>"}`
    """
    block = re.search(r"var ToolSeed49\s*=\s*\[\]ToolRecord\{(.*?)\n\}\n", tools_go, re.DOTALL)
    if not block:
        raise RuntimeError("ToolSeed49 block not found")
    body = block.group(1)
    rows = []
    # Split on `},` followed by `{` (the start of next row).
    row_pattern = re.compile(r'\{\s*"([^"]*)"\s*,\s*"([^"]*)"\s*,\s*"([^"]*)"\s*,\s*"((?:[^"\\]|\\.)*)"\s*,\s*"((?:[^"\\]|\\.)*)"\s*,\s*"([^"]*)"\s*,\s*(\d+)\s*,\s*"([^"]*)"\s*\}')
    for m in row_pattern.finditer(body):
        rows.append([
            m.group(1),  # tool_id
            m.group(2),  # track
            m.group(3),  # category
            m.group(4),  # name
            m.group(5),  # signature
            m.group(6),  # status
            m.group(7),  # priority
            m.group(8),  # version
        ])
    return rows


def extract_workflow_seeds(seeds_go: str) -> list[dict]:
    """WorkflowSeedBuiltins is more complex (nested WorkflowNode).

    Instead of fully parsing it, we hand-off to the Go binary via a small
    helper that dumps the canonical JSON. This script emits a SQL file
    using a single placeholder format that the operator must run through
    `cbmem-team migrate --emit-workflow-seeds` to materialise. See the
    doc-comment in the generated seed.sql for the exact command.
    """
    # We don't fully parse — see cmd/dump-workflow-seeds.go (separate).
    raise NotImplementedError("workflow seeds are emitted by `cbmem-team dump-workflow-seeds`")


# -----------------------------------------------------------------------------
# 3. Compose SQL files.
# -----------------------------------------------------------------------------

HEADER = """-- Auto-generated by tools/cbmem-team/deploy/sql/dump_sql.py — DO NOT EDIT BY HAND.
-- Canonical source: tools/cbmem-team/internal/console/{source_go}.
-- To regenerate after schema/seed changes, run:
--     python tools/cbmem-team/deploy/sql/dump_sql.py
-- and re-verify by running `cbmem-team migrate` against a fresh DB.

"""


def write_schema_sqlite(stmts: list[str]) -> None:
    target = OUT / "schema.sql"
    with target.open("w", encoding="utf-8") as fh:
        fh.write(HEADER.format(source_go="db.go + tools.go + bp.go + workflow.go + repo_pipeline.go"))
        fh.write("-- =====================================================================\n")
        fh.write("-- cbmem-team console — SQLite schema (v2)\n")
        fh.write("-- =====================================================================\n")
        fh.write("-- Idempotent. Run with:  sqlite3 console.db < schema.sql\n\n")
        fh.write("PRAGMA foreign_keys = ON;\n")
        fh.write("PRAGMA journal_mode = WAL;\n\n")
        # Split: legacy 7-table block, then M1, M2, M3, M4.
        legacy, m1, m2, m3, m4 = [], [], [], [], []
        for s in stmts:
            up = s.upper()
            if up.startswith("CREATE TABLE"):
                # Tables belonging to which milestone? Heuristic by table name.
                if any(t in s for t in ("users", "projects", "sessions", "session_turns",
                                          "summarize_tasks", "distill_tasks", "console_sessions")):
                    legacy.append(s)
                elif "tool_directory" in s or "tool_invocation_logs" in s:
                    m1.append(s)
                elif "best_practices" in s or "bp_versions" in s:
                    m2.append(s)
                elif "workflows" in s or "workflow_versions" in s or "workflow_runs" in s or "tickets" in s:
                    m3.append(s)
                elif "repo_pipelines" in s or "repo_pipeline_runs" in s or "bp_candidates" in s:
                    m4.append(s)
                else:
                    legacy.append(s)
            else:
                # CREATE INDEX → follow its parent table.
                if any(t in s for t in ("users", "projects", "sessions", "session_turns",
                                          "summarize_tasks", "distill_tasks", "console_sessions")):
                    legacy.append(s)
                elif "tool_directory" in s or "tool_invocation_logs" in s:
                    m1.append(s)
                elif "best_practices" in s or "bp_versions" in s:
                    m2.append(s)
                elif "workflows" in s or "workflow_runs" in s or "tickets" in s:
                    m3.append(s)
                elif "repo_pipelines" in s or "repo_pipeline_runs" in s or "bp_candidates" in s:
                    m4.append(s)
                else:
                    legacy.append(s)

        for label, block in [
            ("-- ----- legacy v1 (CON-01..CON-05) -----", legacy),
            ("-- ----- M1 tool_directory + tool_invocation_logs -----", m1),
            ("-- ----- M2 best_practices + bp_versions -----", m2),
            ("-- ----- M3 workflows + workflow_runs + tickets -----", m3),
            ("-- ----- M4 repo_pipelines + bp_candidates -----", m4),
        ]:
            if not block:
                continue
            fh.write(f"{label}\n\n")
            for s in block:
                fh.write(s)
                fh.write(";\n\n")


def write_schema_mysql(ddls: list[str], indexes: list[tuple[str, str, str]], extra_ddls: list[str]) -> None:
    target = OUT / "schema-mysql.sql"
    with target.open("w", encoding="utf-8") as fh:
        fh.write(HEADER.format(source_go="db.go + tools.go + bp.go + workflow.go + repo_pipeline.go"))
        fh.write("-- =====================================================================\n")
        fh.write("-- cbmem-team console — MySQL 8.0 schema (v2)\n")
        fh.write("-- =====================================================================\n")
        fh.write("-- Idempotent for CREATE TABLE (uses IF NOT EXISTS) and CREATE VIEW\n")
        fh.write("-- (uses DROP+CREATE since MySQL has no CREATE VIEW IF NOT EXISTS).\n")
        fh.write("-- CREATE INDEX has no IF NOT EXISTS in MySQL 8.0, so the migration\n")
        fh.write("-- entry point probes information_schema.statistics instead. The SQL\n")
        fh.write("-- files here are written for fresh databases; on existing databases,\n")
        fh.write("-- run `cbmem-team migrate` so the probe-based path applies.\n\n")
        fh.write("-- Legacy 7 tables -----------------------------------------------------\n\n")
        for s in ddls:
            fh.write(s)
            fh.write(";\n\n")
        fh.write("-- Legacy indexes ------------------------------------------------------\n\n")
        for table, name, ddl in indexes:
            fh.write(f"-- {table}.{name}\n{ddl};\n\n")
        fh.write("-- M1..M4 -------------------------------------------------------------\n\n")
        for s in extra_ddls:
            fh.write(s)
            fh.write(";\n\n")


def write_seed_sqlite(tool_rows: list[list[str]]) -> None:
    """seed.sql covers tools (49) + BPs + workflows.

    BP and workflow seeds are documented as "run `cbmem-team migrate-seed`
    instead" because they involve JSON columns whose representation in
    MySQL JSON differs from SQLite TEXT. The tool seed is portable.
    """
    target = OUT / "seed.sql"
    with target.open("w", encoding="utf-8") as fh:
        fh.write(HEADER.format(source_go="tools.go (ToolSeed49) + bp.go (BPRecord) + workflow_seeds.go (WorkflowSeedBuiltins)"))
        fh.write("-- =====================================================================\n")
        fh.write("-- cbmem-team console — initial data (SQLite)\n")
        fh.write("-- =====================================================================\n")
        fh.write("-- Run AFTER schema.sql. Use `sqlite3 console.db < seed.sql`.\n")
        fh.write("--\n")
        fh.write("-- This file ships the deterministic tool catalog only (49 rows).\n")
        fh.write("-- Best practices (3 rows) and built-in workflows (3 rows) are seeded\n")
        fh.write("-- by the Go binary because their JSON columns hold structured\n")
        fh.write("-- payloads that are easier to maintain in Go than as literal SQL.\n")
        fh.write("-- Run after applying this file:\n")
        fh.write("--     cbmem-team migrate-seed\n")
        fh.write("-- (idempotent — re-runs are no-ops once the tables have rows).\n\n")
        fh.write("BEGIN;\n\n")
        fh.write("-- ----- tool_directory (49 rows) -----\n\n")
        fh.write("INSERT INTO tool_directory\n")
        fh.write("    (tool_id, track, category, name, signature, status, priority, version, created_at, updated_at)\n")
        fh.write("VALUES\n")
        now = "2026-07-12 00:00:00"
        values = []
        for r in tool_rows:
            # Escape single quotes inside signature/name.
            def esc(s: str) -> str:
                return s.replace("'", "''")
            values.append(
                f"    ('{esc(r[0])}','{esc(r[1])}','{esc(r[2])}','{esc(r[3])}','{esc(r[4])}',"
                f"'{esc(r[5])}',{r[6]},'{esc(r[7])}','{now}','{now}')"
            )
        fh.write(",\n".join(values))
        fh.write(";\n\n")
        fh.write("COMMIT;\n")


def write_seed_mysql(tool_rows: list[list[str]]) -> None:
    target = OUT / "seed-mysql.sql"
    with target.open("w", encoding="utf-8") as fh:
        fh.write(HEADER.format(source_go="tools.go (ToolSeed49)"))
        fh.write("-- =====================================================================\n")
        fh.write("-- cbmem-team console — initial data (MySQL 8.0)\n")
        fh.write("-- =====================================================================\n")
        fh.write("-- Run AFTER schema-mysql.sql. Use:\n")
        fh.write("--     mysql console < schema-mysql.sql\n")
        fh.write("--     mysql console < seed-mysql.sql\n")
        fh.write("-- Then `cbmem-team migrate-seed` for BP/workflow seeds.\n\n")
        fh.write("SET autocommit=0;\n")
        fh.write("START TRANSACTION;\n\n")
        fh.write("-- ----- tool_directory (49 rows) -----\n\n")
        fh.write("INSERT INTO tool_directory\n")
        fh.write("    (tool_id, track, category, name, signature, status, priority, version, created_at, updated_at)\n")
        fh.write("VALUES\n")
        now = "2026-07-12 00:00:00"
        values = []
        for r in tool_rows:
            def esc(s: str) -> str:
                return s.replace("\\", "\\\\").replace("'", "\\'")
            values.append(
                f"    ('{esc(r[0])}','{esc(r[1])}','{esc(r[2])}','{esc(r[3])}','{esc(r[4])}',"
                f"'{esc(r[5])}',{r[6]},'{esc(r[7])}','{now}','{now}')"
            )
        fh.write(",\n".join(values))
        fh.write(";\n\n")
        fh.write("COMMIT;\n")


# -----------------------------------------------------------------------------
# 4. Main.
# -----------------------------------------------------------------------------

def main() -> int:
    if not CONSOLE.exists():
        print(f"error: {CONSOLE} not found; run from repo root", file=sys.stderr)
        return 1

    OUT.mkdir(parents=True, exist_ok=True)

    db_go = (CONSOLE / "db.go").read_text(encoding="utf-8")
    tools_go = (CONSOLE / "tools.go").read_text(encoding="utf-8")
    bp_go = (CONSOLE / "bp.go").read_text(encoding="utf-8")
    workflow_go = (CONSOLE / "workflow.go").read_text(encoding="utf-8")
    repo_go = (CONSOLE / "repo_pipeline.go").read_text(encoding="utf-8")

    # --- SQLite schema: legacy + M1..M4 ApplySQLite ---
    sqlite_stmts: list[str] = []
    sqlite_stmts.extend(extract_migrate_console(db_go))
    for src in (tools_go, bp_go, workflow_go, repo_go):
        sqlite_stmts.extend(extract_stmts_from_apply_sqlite(src))
    write_schema_sqlite(sqlite_stmts)

    # --- MySQL schema: mysqlConsoleDDL + indexes + M1..M4 ApplyMySQL ---
    mysql_ddls, mysql_indexes = extract_mysql_ddl(db_go)
    mysql_extra: list[str] = []
    for src in (tools_go, bp_go, workflow_go, repo_go):
        mysql_extra.extend(extract_apply_mysql(src))
    write_schema_mysql(mysql_ddls, mysql_indexes, mysql_extra)

    # --- Seed (portable part: tool_directory) ---
    tool_rows = extract_tool_seeds(tools_go)
    write_seed_sqlite(tool_rows)
    write_seed_mysql(tool_rows)

    print(f"OK: wrote 4 files under {OUT}/")
    print(f"   schema.sql      ({len(sqlite_stmts)} stmts)")
    print(f"   schema-mysql.sql ({len(mysql_ddls)} tables + {len(mysql_indexes)} legacy indexes + {len(mysql_extra)} M1..M4 stmts)")
    print(f"   seed.sql        ({len(tool_rows)} tool rows)")
    print(f"   seed-mysql.sql  ({len(tool_rows)} tool rows)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
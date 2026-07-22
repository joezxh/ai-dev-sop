# cmd/ Tools Overview

The `tools/cbmem-team/cmd/` directory contains standalone Go entry points, each with a specific responsibility. All MySQL diagnostic tools resolve DSN via `devconf.ResolveMySQLDSN` with precedence: CLI flag → `$CBMEM_MYSQL_DSN` → `$DSN` → dev default.

---

## 1. cbmem-team (Main Service)

**Purpose:** HTTP wrapper around `codebase-memory-mcp` that enables multiple developers to share a single remote instance while keeping each user's code-ast index strictly isolated.

**Entry (main):** Subcommand dispatcher, supports:

| Invocation | Behavior |
|---|---|
| `cbmem-team` (no args) | Default: start HTTP server (`runServe`) |
| `cbmem-team migrate-tables` | Create 7 console tables on MySQL |
| `cbmem-team migrate-sqlite-to-mysql` | One-shot ETL from SQLite to MySQL |
| `cbmem-team mysql-ping` | Check MySQL connectivity |

**`runServe` startup flow:**

1. `config.ParseFlags` parses all configuration
2. Load `users.json`, initialize repos manager and stdio process pool
3. Initialize JWT verifier & admin static token authenticator
4. Open Console DB (MySQL or SQLite, auto-detected)
5. Register routes:

| Route | Description |
|---|---|
| `GET /healthz` | Health check, returns `{"status":"ok","users":N}` |
| `POST /mcp` | MCP Streamable HTTP endpoint (JWT required) |
| `GET /mcp` | Streamable GET endpoint (JWT required) |
| `GET /mcp/sse` | MCP SSE endpoint (JWT required) |
| `POST /refresh` | User-driven JWT refresh (no admin required) |
| `POST /admin/*` | User CRUD, token minting, repos mgmt, stats (admin token required) |
| `/console/*` | SPA static files (optional, via `-console-dist`) |

6. Console module routes assembled by `console.Mount` (M1–M6)
7. Signal-aware graceful shutdown

**Key configuration flags:**

```
-listen              Listen address (default :18765)
-data                Data directory (users.json, sqlite, repos, etc.)
-admin-token         Admin static auth token
-jwt-secret          JWT signing secret
-mysql-dsn           MySQL DSN (auto-switches to MySQL backend when set)
-mcp-bin             codebase-memory-mcp binary path
-repo-root           Git repo root (M3 teams/projects v2)
-console-dist        SPA static files directory
-config-file         Hot-reload config file path
```

---

## 2. cbmem-mint-token (Offline JWT Minting)

**Purpose:** Mint a JWT Bearer Token for an existing user offline, without exposing `/admin/*` to the network.

**Entry (main):** Parse three flags, call `auth.NewVerifier.Sign` to issue an HS256 JWT, print to stdout.

```
cbmem-mint-token --secret $JWT_SECRET --user alice --ttl 720h
```

| Flag | Default | Description |
|---|---|---|
| `--secret` | required | JWT shared secret |
| `--user` | required | User ID (written to `sub` claim) |
| `--ttl` | `720h` (30 days) | Token lifetime |

---

## 3. create-db (MySQL Database Bootstrap)

**Purpose:** One-shot idempotent creation of the `cbmem` schema (utf8mb4 + unicode_ci), verifying table count is 0 afterward.

**Entry (main):**

1. Parse `-mysql-dsn` (with env fallback)
2. Execute on bootstrap DSN (no schema selected):
   ```sql
   CREATE DATABASE IF NOT EXISTS cbmem CHARACTER SET utf8mb4;
   ALTER DATABASE cbmem CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
   ```
3. Reconnect with schema, query `information_schema.tables` to count

> Note: MySQL 9.0 still rejects `IF NOT EXISTS` combined with `CHARACTER SET` in a single `CREATE DATABASE` statement, so two separate statements are used.

---

## 4. mcp-stub (MCP Stdio Test Stub)

**Purpose:** Minimal MCP stdio server for e2e test scenarios, eliminating the dependency on a real `codebase-memory-mcp` binary. Its `tools/call` has a built-in 200ms delay, making it useful for verifying rate limiter behavior.

**Entry (main):** Read stdin line-by-line as JSON-RPC, dispatch by method:

| Method | Behavior |
|---|---|
| `initialize` | Returns serverInfo + capabilities |
| `tools/list` | Returns `stub-echo` tool |
| `tools/call` | sleep 200ms → returns `{"content":[{"type":"text","text":"ok"}]}` |
| `ping` | Returns empty result |
| Others | Returns empty result, no error |

All responses terminated with `\n`, compliant with MCP stdio protocol.

---

## 5. mint-token-debug (Debug JWT Minting)

**Purpose:** Fixed-key JWT minting using `dev-secret-m1-7f8a-2026` to issue a 1-hour token for `sub=alice-0`. Used for manual 401 debugging. No flags required.

**Entry (main):** Pure stdlib (no external dependencies), prints JWT to stdout.

---

## Summary Table

| Program | Type | Purpose |
|---|---|---|
| `cbmem-team` | Main service | HTTP + MCP proxy, multi-user isolation |
| `cbmem-mint-token` | Tool | Offline JWT minting |
| `create-db` | Tool | MySQL database initialization |
| `mcp-stub` | Tool | MCP stdio test double |
| `mint-token-debug` | Tool | Debug-only fixed-key JWT |

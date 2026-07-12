# cbmem-team Operator Manual

> **Version**: v2 (M3 + M4 merged, dual-backend database)
> **Audience**: ops / devs / client integrators
> **Prerequisite reading**: [`README.md`](./README.md) (architecture), [`CONSOLE.md`](./CONSOLE.md) (console API reference), [`BUILD.md`](./BUILD.md) (build + deploy)
> **中文版本**: [`manual.md`](./manual.md)（canonical — this translation is best-effort）

This manual is **hands-on** — from a clean checkout to a fully running service. Five usage scenarios plus three appendices:

1. [Quick start](#1-quick-start)
2. [Foreground + background launch guide](#2-foreground--background-launch-guide)
3. [MCP service configuration & clients](#3-mcp-service-configuration--clients)
4. [Command-line tools catalogue](#4-command-line-tools-catalogue)
5. [Feature panorama](#5-feature-panorama)
6. [Troubleshooting FAQ](#6-troubleshooting-faq)

---

## 1. Quick Start

### 1.1 Prerequisites

| Dependency | Minimum | Purpose |
|---|---|---|
| **Go** | 1.24+ | Compile the binaries (no cgo) |
| **SQLite** | bundled | Default backend (zero install) |
| **MySQL** | 8.0 / 9.0 (optional) | Production backend; skip if `-mysql-dsn` is not set |
| **Node.js** | 20+ | Only needed to build `/console/` frontend (VitePress) |
| **codebase-memory-mcp** | latest | The backend MCP service, path via `-mcp-bin` |

### 1.2 First start (dev box + SQLite default)

```bash
cd tools/cbmem-team

# 1. Compile the main binary (Windows / Linux / macOS all use the same `go build`)
go build -o bin/cbmem-team ./cmd/cbmem-team

# 2. Prepare the data directory
mkdir -p bin/data

# 3. Start the service in foreground (logs go to stderr)
./bin/cbmem-team \
  -listen :8787 \
  -data ./bin/data \
  -mcp-bin /usr/local/bin/codebase-memory-mcp \
  -jwt-secret "dev-secret-please-rotate" \
  -admin-token "dev-admin-token-please-rotate" \
  -log info
```

Successful startup prints:

```
console db: driver=sqlite
console db: migrate console + M1..M4 schemas
seed tool_directory (49 rows)
seed best_practices (3 rows)
seed workflows (3 rows)
listening on :8787
```

### 1.3 Health check

```bash
# Is the service reachable?
curl http://127.0.0.1:8787/healthz
# {"status":"ok","users":0}

# Is the admin surface reachable? (replace with your -admin-token value)
curl -H "X-Admin-Token: dev-admin-token-please-rotate" http://127.0.0.1:8787/admin/stats
```

### 1.4 Create your first user + mint a token

```bash
ADMIN="dev-admin-token-please-rotate"

# Create user alice
curl -X POST http://127.0.0.1:8787/admin/users \
  -H "X-Admin-Token: $ADMIN" \
  -H "Content-Type: application/json" \
  -d '{"id":"alice","display_name":"Alice","project_paths":["/code/foo"]}'

# Mint a 30-day token
curl -X POST "http://127.0.0.1:8787/admin/users/alice/token?ttl=720h" \
  -H "X-Admin-Token: $ADMIN"
```

---

## 2. Foreground + Background Launch Guide

### 2.1 Launch modes at a glance

| Mode | Command | When to use |
|---|---|---|
| **Foreground (debug)** | `./cbmem-team <flags>` | Dev, watching logs, ad-hoc runs |
| **systemd background** | `systemctl start cbmem-team` | Production (recommended) |
| **Docker Compose** | `docker compose -f deploy/docker-compose.mysql.yml up -d` | MySQL only — the service binary still runs on the host |

### 2.2 Foreground mode (dev / debug)

Foreground is the simplest mode — all logs stream to stderr. All flags use the `-flag value` form:

#### 2.2.1 Required flags

| Flag | Description | Example |
|---|---|---|
| `-listen` | HTTP listen address | `:8787` |
| `-data` | Data directory (SQLite file lives here) | `/var/lib/cbmem-team` |
| `-mcp-bin` | Absolute path to `codebase-memory-mcp` binary | `/usr/local/bin/codebase-memory-mcp` |
| `-jwt-secret` | HS256 signing secret (**rotate in production**) | `openssl rand -hex 32` |
| `-admin-token` | Static admin token (**rotate in production**) | `openssl rand -hex 32` |

#### 2.2.2 Optional flags

| Flag | Default | Description |
|---|---|---|
| `-config` | `/etc/cbmem-team/config.yaml` | YAML config path (optional; CLI flags win) |
| `-log` | `info` | Log level: `debug` / `info` / `warn` / `error` |
| `-mysql-dsn` | empty (= SQLite) | MySQL DSN; non-empty switches backend to MySQL 8.0 |
| `-mysql-max-open` | `16` | MySQL pool ceiling |
| `-mysql-max-idle` | `4` | MySQL idle pool size |
| `-mysql-max-lifetime` | `30m` | MySQL connection max lifetime |
| `-llm-provider` | `fake` | `fake` / `openai` / `ollama` |
| `-llm-model` | empty | Model name (e.g. `gpt-4o-mini`) |
| `-llm-base-url` | empty | LLM API base URL |
| `-llm-api-key` | empty | LLM API key (Ollama can leave this empty) |
| `-mempalace-base` | empty | MemPalace HTTP base URL; empty disables it |
| `-console-dist` | empty | Path to VitePress console frontend build artifacts |

#### 2.2.3 Full dev-box startup example (SQLite)

```bash
cd tools/cbmem-team
go build -o bin/cbmem-team ./cmd/cbmem-team
mkdir -p bin/data

./bin/cbmem-team \
  -listen :8787 \
  -data ./bin/data \
  -mcp-bin $(which codebase-memory-mcp) \
  -jwt-secret "dev-jwt-$(date +%s)" \
  -admin-token "dev-admin-$(date +%s)" \
  -llm-provider fake \
  -log info
```

#### 2.2.4 Full production startup example (MySQL)

```bash
./bin/cbmem-team \
  -listen :8787 \
  -data /var/lib/cbmem-team \
  -mcp-bin /usr/local/bin/codebase-memory-mcp \
  -jwt-secret "$(cat /etc/cbmem-team/jwt-secret)" \
  -admin-token "$(cat /etc/cbmem-team/admin-token)" \
  -mysql-dsn "root:$(cat /etc/cbmem-team/mysql-pass)@tcp(127.0.0.1:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4" \
  -llm-provider openai \
  -llm-model gpt-4o-mini \
  -llm-base-url https://api.openai.com/v1 \
  -llm-api-key "$(cat /etc/cbmem-team/llm-api-key)" \
  -mempalace-base http://192.168.100.83:8089 \
  -console-dist /opt/cbmem-console/dist
```

### 2.3 systemd background deployment (recommended for production)

#### 2.3.1 Prepare the env file

`/etc/cbmem-team/cbmem-team.env` (loaded by systemd via `EnvironmentFile=`):

```ini
LISTEN=:8787
DATA_DIR=/var/lib/cbmem-team
MCP_BINARY=/usr/local/bin/codebase-memory-mcp
JWT_SECRET=$(openssl rand -hex 32)        # replace with strong random
ADMIN_TOKEN=$(openssl rand -hex 32)       # replace with strong random
LOG_LEVEL=info

IDLE_TTL=30m
MAX_PROCS_PER_USER=4
```

Lock down permissions (env file holds secrets):

```bash
sudo install -m 0600 deploy/cbmem-team.env /etc/cbmem-team/cbmem-team.env
```

#### 2.3.2 Install the systemd unit

The repo ships two units:

- `deploy/cbmem-team.service` — SQLite path, **rollback unit**
- `deploy/cbmem-team-mysql.service` — MySQL path, **production**

```bash
sudo install -m 0644 deploy/cbmem-team-mysql.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now cbmem-team-mysql
sudo systemctl status cbmem-team-mysql --no-pager
```

#### 2.3.3 Inspect logs

```bash
# Live tail
sudo journalctl -u cbmem-team-mysql -f

# Last 100 lines
sudo journalctl -u cbmem-team-mysql -n 100 --no-pager
```

#### 2.3.4 7-day rollback window (SQLite ↔ MySQL)

Both units share the `/var/lib/cbmem-team` data directory; only one can run at a time:

```bash
# Fall back to SQLite (emergency)
sudo systemctl stop cbmem-team-mysql
sudo systemctl start cbmem-team       # legacy unit has no -mysql-dsn flag

# Switch back to MySQL
sudo systemctl stop cbmem-team
sudo systemctl start cbmem-team-mysql
```

### 2.4 Subcommands (admin, not serve)

The `cbmem-team` binary embeds three admin subcommands on top of the default `serve`:

| Subcommand | Purpose | When to use |
|---|---|---|
| `cbmem-team migrate-tables -mysql-dsn <dsn>` | Create the 7 console tables on MySQL | First-time MySQL deploy |
| `cbmem-team migrate-sqlite-to-mysql -sqlite <path> -mysql-dsn <dsn>` | One-shot ETL from SQLite to MySQL | Backend switch |
| `cbmem-team mysql-ping -mysql-dsn <dsn>` | Probe DSN reachability | Quick connectivity check |

```bash
# 1. Create MySQL tables (first time)
cbmem-team migrate-tables -mysql-dsn "root:pw@tcp(host:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4"

# 2. SQLite → MySQL ETL (one-time)
cbmem-team migrate-sqlite-to-mysql \
  -sqlite /var/lib/cbmem-team/cbmem-team.db \
  -mysql-dsn "root:pw@tcp(host:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4"

# 3. Verify connection
cbmem-team mysql-ping -mysql-dsn "root:pw@tcp(host:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4"
```

### 2.5 Build artefacts

`go build` produces four classes of binaries:

| Binary | Purpose | Production? |
|---|---|---|
| `cbmem-team` | Main service (HTTP wrapper + console) | ✅ |
| `cbmem-mint-token` | Offline JWT minter (no need to expose `/admin`) | ✅ |
| `mcp-stub` | Deterministic MCP backend (e2e-only) | ❌ |
| `mint-token-debug` | Throwaway JWT minter for debugging 401s | ❌ |

---

## 3. MCP Service Configuration & Clients

`cbmem-team` exposes **three MCP endpoints**:

| Path | Method | Purpose |
|---|---|---|
| `/mcp` | POST | Streamable HTTP (v2, primary) |
| `/mcp/` | POST | Same as above, trailing-slash alias |
| `/mcp/sse` | GET | SSE (v1 fallback) |

Each endpoint requires `Authorization: Bearer <JWT>`. The JWT subject (`sub`) IS the user_id.

### 3.1 Cursor client config

`~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://your-server:8787/mcp?as=alice&project=${workspaceFolder}",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...."
      }
    }
  }
}
```

**Key points**:

- `${workspaceFolder}` must be replaced with the team's agreed literal path (e.g. `/code/foo`); that path determines which stdio subprocess workdir the server routes to
- Each developer only needs their **own JWT** — the URL `as=...` parameter is what isolates their indexes server-side
- Never expose `:8787` directly; terminate TLS with nginx/caddy upstream

### 3.2 Claude Desktop client config

`~/.config/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://your-server:8787/mcp?as=alice&project=/code/foo",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...."
      }
    }
  }
}
```

### 3.3 Offline JWT minting (no `/admin` exposure)

`cbmem-mint-token` lets you mint tokens without opening the admin port:

```bash
cbmem-mint-token \
  --secret "$JWT_SECRET" \
  --user alice \
  --ttl 720h
# prints: eyJhbGciOiJIUzI1NiIs...
```

Flags:

| Flag | Default | Description |
|---|---|---|
| `--secret` | (required) | Byte-for-byte equal to the server's `-jwt-secret` |
| `--user` | (required) | User id (`sub` claim) |
| `--ttl` | `720h` | Token lifetime |

### 3.4 End-to-end connectivity check

```bash
# 1. Server healthz
curl http://your-server:8787/healthz

# 2. Client initialize handshake
curl -X POST http://your-server:8787/mcp?as=alice\&project=/code/foo \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"curl","version":"0"},"capabilities":{}}}'

# Expected: returns serverInfo + capabilities
```

---

## 4. Command-line Tools Catalogue

`tools/cbmem-team/cmd/` contains one Go package per subdirectory, each compiling to its own binary. **14 tools total** (13 production + 1 dev):

### 4.1 Production tools (deployed to servers)

#### 4.1.1 `cbmem-team` — main service

- **Path**: `cmd/cbmem-team/`
- **Entry**: `main.go` + `handlers.go` + sibling files in the same package
- **Build**: `go build -o bin/cbmem-team ./cmd/cbmem-team`
- **Usage**: see chapter 2 and §2.4 subcommands
- **Flag reference**: see [2.2.1](#221-required-flags) / [2.2.2](#222-optional-flags)
- **Key endpoints**: `/healthz`, `/admin/*`, `/api/console/*`, `/api/console/v2/*`, `/mcp`, `/mcp/sse`

#### 4.1.2 `cbmem-mint-token` — offline JWT minter

- **Path**: `cmd/cbmem-mint-token/`
- **Build**: `go build -o bin/cbmem-mint-token ./cmd/cbmem-mint-token`
- **Usage**:

  ```bash
  cbmem-mint-token --secret "$JWT_SECRET" --user alice --ttl 720h
  ```

- **Flags**: `--secret`, `--user` (required), `--ttl` (default 720h)

### 4.2 Ops / migration tools (deploy- and switch-phase)

#### 4.2.1 `create-db` — create the `cbmem` database

- **Path**: `cmd/create-db/`
- **Build**: `go build -o bin/create-db ./cmd/create-db`
- **Behaviour**: connect MySQL → `CREATE DATABASE IF NOT EXISTS cbmem CHARACTER SET utf8mb4` + `ALTER DATABASE ... COLLATE utf8mb4_unicode_ci` → verify
- **Flags**:

  | Flag | Default | Description |
  |---|---|---|
  | `-mysql-dsn` | `$CBMEM_MYSQL_DSN` / `$DSN` / dev default | MySQL DSN (DB name `cbmem` not required) |

- **Example**:

  ```bash
  create-db -mysql-dsn "root:pw@tcp(127.0.0.1:3306)/?parseTime=true&loc=Local&charset=utf8mb4"
  ```

#### 4.2.2 `mysql-probe` — MySQL connection + server feature probe

- **Path**: `cmd/mysql-probe/`
- **Behaviour**: checks DSN reachability + reports VERSION / collation / flush_log config
- **Flag**: `-mysql-dsn` (same as above)
- **Example**:

  ```bash
  $ mysql-probe -mysql-dsn "$DSN"
  MySQL version : 8.0.36 (Linux)
  Charset/Coll  : utf8mb4 / utf8mb4_0900_ai_ci
  Flush log trx : 1
  DB cbmem      : exists
  ```

#### 4.2.3 `show-tables` — list all tables + engine + row counts

- **Path**: `cmd/show-tables/`
- **Behaviour**: scans `information_schema.tables`, prints table / engine / collation / column count / KB
- **Flag**: `-mysql-dsn` (same as above)

#### 4.2.4 `show-indexes` — list all indexes

- **Path**: `cmd/show-indexes/`
- **Behaviour**: scans `information_schema.statistics`, prints table / index / columns
- **Flags**: same as above

#### 4.2.5 `seed-sqlite` — write SQLite test data

- **Path**: `cmd/seed-sqlite/`
- **Behaviour**: writes 5 users / 3 projects / 4 sessions / 30 turns / 2 summarize / 2 distill / 1 console_session into `D:\projects\ai-dev-sop\tools\cbmem-team\bin\test-cbmem.db`, preparing fixtures for `e2e-v5` ETL validation
- **Flags**: none (hard-coded path)
- **When to use**: invoked automatically by `e2e-v5`; **not for production deploys**

#### 4.2.6 `migrate-tables` subcommand — see §2.4

#### 4.2.7 `migrate-sqlite-to-mysql` subcommand — see §2.4

#### 4.2.8 `mysql-ping` subcommand — see §2.4

### 4.3 End-to-end test tools (dev / CI only)

All `e2e-*` tools follow the same shape: compile → run → spawn `cbmem-team` subprocess → fire requests → assert → exit. **Not deployed to production.**

| Tool | Milestone | Checkpoints | DSN |
|---|---|---|---|
| **`e2e-m1`** | M1 tool directory + invocation log | 6: service boot, admin/users, tool_directory ≥1, log 0→≥1, streamable SSE, dashboard summary | `-mysql-dsn` |
| **`e2e-v5`** | M0.5 SQLite/MySQL dual-track + ETL | 5: SQLite serve, MySQL serve, migrate-tables idempotent, ETL row 1:1, parity after switch | same |
| **`e2e-v6`** | M2 tool rate-limit + BP CRUD + graph | 5: 49 tools, qps=1 → 429, POST bp, PUT bp+1 version, graph nodes + UI | same |
| **`e2e-v7`** | M3 workflow + M4 repo pipeline + bp_candidate review | 7: 3 seeded workflows, create+run, 3 builtins run, high-risk dashboard, ticket auto-open, repo CRUD, candidate review | same |

#### Common flags (all 4 e2e tools)

| Flag | Default | Description |
|---|---|---|
| `-mysql-dsn` | `$CBMEM_MYSQL_DSN` / `$DSN` / dev default | MySQL DSN; resolution rule lives in `internal/devconf` |

#### Common run examples

```bash
cd tools/cbmem-team
go run ./cmd/e2e-v7                  # uses dev default MySQL
go run ./cmd/e2e-v7 -mysql-dsn "..."  # custom DSN
CBMEM_MYSQL_DSN="..." go run ./cmd/e2e-v7   # override via env
```

### 4.4 Debug tools

#### 4.4.1 `mint-token-debug` — hand-rolled JWT minter (debug 401s)

- **Path**: `cmd/mint-token-debug/`
- **Behaviour**: prints a HS256 token hard-coded with secret `dev-secret-m1-7f8a-2026` and sub `alice-0`; useful to verify server-side signature validation when in doubt
- **When to use**: only when you suspect server-side JWT validation has a bug — never in production

#### 4.4.2 `mcp-stub` — deterministic MCP backend (e2e-only)

- **Path**: `cmd/mcp-stub/`
- **Behaviour**: implements a minimal MCP JSON-RPC over stdio. `initialize` / `tools/list` / `tools/call` / `ping` return fixed payloads; `tools/call` sleeps 200 ms on purpose to trigger the rate limiter
- **When to use**: as `-mcp-bin` replacement for e2e tests; **do not treat as a real MCP service**

### 4.5 Dev tools (local only)

| Tool | Path | Purpose |
|---|---|---|
| `_dev/verify_sqlite` | `cmd/_dev/verify_sqlite/` | Verify `deploy/sql/{schema,seed}.sql` runs on an in-memory SQLite (companion to `dump_sql.py`) |

### 4.6 Tool cheat-sheet

| Tool | Build command | Default port / path | Primary use |
|---|---|---|---|
| `cbmem-team` | `go build -o bin/cbmem-team ./cmd/cbmem-team` | `:8787` | Main service |
| `cbmem-mint-token` | `go build -o bin/cbmem-mint-token ./cmd/cbmem-mint-token` | n/a | Offline JWT minter |
| `create-db` | `go build -o bin/create-db ./cmd/create-db` | `:3306` | Create `cbmem` DB |
| `mysql-probe` | `go build -o bin/mysql-probe ./cmd/mysql-probe` | `:3306` | Probe MySQL |
| `show-tables` | `go build -o bin/show-tables ./cmd/show-tables` | `:3306` | List all tables |
| `show-indexes` | `go build -o bin/show-indexes ./cmd/show-indexes` | `:3306` | List all indexes |
| `seed-sqlite` | `go build -o bin/seed-sqlite ./cmd/seed-sqlite` | local .db | Seed SQLite test data |
| `e2e-m1` | `go run ./cmd/e2e-m1` | `:28791..28795` | M1 acceptance |
| `e2e-v5` | `go run ./cmd/e2e-v5` | `:18787` | M0.5 acceptance |
| `e2e-v6` | `go run ./cmd/e2e-v6` | `:28796` | M2 acceptance |
| `e2e-v7` | `go run ./cmd/e2e-v7` | `:28797` | M3+M4 acceptance |
| `mcp-stub` | `go run ./cmd/mcp-stub` | stdio | e2e MCP stub |
| `mint-token-debug` | `go run ./cmd/mint-token-debug` | n/a | Debug 401 |
| `_dev/verify_sqlite` | `go run ./cmd/_dev/verify_sqlite deploy/sql` | in-mem | Verify SQL files |

---

## 5. Feature Panorama

Each milestone bundles a set of related endpoints + tables + a seed dataset.

### 5.1 M0 — HTTP wrapper baseline

- **Core**: wrap the per-process `codebase-memory-mcp` as a multi-user HTTP service
- **Endpoints**:
  - `GET  /healthz` — health check
  - `POST /refresh` — JWT auto-renewal (a user with a valid token can extend it)
  - `POST /mcp` / `POST /mcp/` — Streamable HTTP (v2, primary)
  - `GET  /mcp/sse` — SSE (v1 fallback)
- **Multi-user isolation**:
  - one stdio subprocess per user
  - workdir shape: `/var/lib/cbmem-team/users/<user_id>/projects/<project>/`
  - SQLite file locks naturally isolate
- **Idle reaper**: child process killed after `-idle-ttl` of inactivity (default 30 min)

### 5.2 M0.5 — SQLite ↔ MySQL dual track

- **Core**: single-box (SQLite) + production high-throughput (MySQL)
- **Endpoints**: unchanged, all inherited from M0
- **Database**:
  - **SQLite**: default; file `<data>/cbmem-team.db`
  - **MySQL**: enabled via `-mysql-dsn`; tables built by `migrate-tables`
- **ETL**: `migrate-sqlite-to-mysql` moves every row from SQLite to MySQL in one shot
- **Properties**:
  - `migrate-tables` is idempotent (running it twice does not error)
  - Switching backends does not lose data (they share the `/var/lib/cbmem-team` SQLite snapshot)

### 5.3 M1 — Tool directory + invocation log (CON-01)

- **Tables**: `tool_directory` (49 built-in tools) + `tool_invocation_logs`
- **Endpoints**:
  - `GET /api/console/v2/tools` — tool list
  - `GET /api/console/v2/dashboard/summary` — invocation stats
  - `GET /api/console/v2/dashboard/recent-invocations` — recent calls
- **Capture**: a capture middleware sits outside the MCP handler and auto-writes `tool_invocation_logs`
- **Seed**: `SeedToolDirectory` (49 rows)

### 5.4 M2 — Tool rate limit + Best Practices CRUD (CON-02)

- **Tables**: `best_practices` + `bp_versions` (version snapshots)
- **Endpoints**:
  - `GET/POST/PUT/DELETE /api/console/v2/bps[/:id]`
  - `GET /api/console/v2/bps/:id/graph` — BP relationship graph
  - `PATCH /api/console/v2/rate-limit` — adjust rate-limit thresholds
- **Properties**:
  - Rate-limit buckets on `(user_id, tool_id)`; qps above threshold returns 429 + `retry_after`
  - BP versions: every PUT appends a snapshot to `bp_versions`
- **Seed**: `SeedBPs` (3 cross-category BPs: naming / performance / collaboration)

### 5.5 M3 — Workflow engine (CON-06)

- **Tables**: `workflows` + `workflow_versions` + `workflow_runs` + `tickets`
- **Node kinds**: `log` / `tool_call` / `bp_check` / `condition` / `wait_approval` / `parallel` / `loop_guard`
- **Endpoints**: see [`CONSOLE.md` §Workflows](./CONSOLE.md)
- **Properties**:
  - 3 built-in workflows auto-seeded on migration: `wf-commit-precheck` / `wf-adr-doublewrite` / `wf-repo-daily-sync`
  - `draft` → `published` freezes the graph; `published` → `archived`
  - `POST /run` runs synchronously; returns `run_id` + `step_count`
- **Seed**: `SeedWorkflows` (3 built-in)

### 5.6 M4 — Repo Pipeline + BP Candidates (CON-07)

- **Tables**: `repo_pipelines` + `repo_pipeline_runs` + `bp_candidates`
- **6-stage runtime**: `crawl` → `parse` → `grade` → `sink` → `rehearse`
- **Endpoints**: see [`CONSOLE.md` §Repo Pipeline](./CONSOLE.md)
- **Properties**:
  - `source` types: `local` / `github` / `rss` (interfaces stubbed for design phase)
  - BP candidate three-state: `draft` → `accepted` / `rejected`; `accepted` → `merged` writes to `best_practices`
  - Views: `v_repo_pipeline_success_rate_7d` (MySQL)
- **MySQL views**:
  - `v_high_risk_invocations_7d` (M3)
  - `v_repo_pipeline_success_rate_7d` (M4)

### 5.7 Console frontend (`/console/`)

- **Login**: open `http://server:8787/console/` and enter the admin token
- **5 modules**:
  - Users (`#/users`)
  - Projects (`#/projects`)
  - Sessions (`#/sessions`)
  - Summarize (`#/summarize`)
  - Distill (`#/distill`)
- **M3 UI**: workflow editor
- **M4 UI**: repo pipeline management + BP candidate review
- **Build**: `cd docs-site && pnpm install && pnpm build` — artefacts in `.vitepress/dist/`, pointed to via `-console-dist`

### 5.8 Cross-cutting

- **Unified envelope**: `{"code":0, "msg":"", "data":{}}`
- **Error code ranges**:
  - `0` — success
  - `40100xx` — auth failure (admin-token mismatch / session expired / CSRF)
  - `40300xx` — forbidden (allow-list breach / user disabled)
  - `40900xx` — conflict (e.g. duplicate user id)
  - `50000xx` — system error (LLM timeout / SQLite write failure)
- **Session capture (MCP tap)**: every `initialize` / `tools/call` carrying `messages/create` auto-creates or reuses `sessions` + `session_turns`; if a recent session exists within 30 min for the same `(user_id, project_path)` pair, it is reused

---

## 6. Troubleshooting FAQ

<a id="faq-q1"></a>
### Q1: Build fails with `undefined: requestLogger` / `refreshTokenHandler` ...

**Cause**: GoLand's Run/Debug Configuration is using a single-file path:

```bash
go build -o xxx main.go    # ❌ compiles only main.go
```

**Fix**: change Run Configuration "Run kind" to **Package**, path `./cmd/cbmem-team`. Equivalent CLI:

```bash
go build -o bin/cbmem-team ./cmd/cbmem-team    # ✅ whole-package build
```

<a id="faq-q2"></a>
### Q2: MySQL reports `ERROR 1061 (42000): duplicate key name`

**Cause**: `CREATE INDEX IF NOT EXISTS` is not supported on MySQL 8.0; re-running `schema-mysql.sql` collides on existing index names.

**Fix**: use `cbmem-team migrate-tables -mysql-dsn <dsn>` (it probes `information_schema.statistics` first); `schema-mysql.sql` is only meant for a **fresh database**.

<a id="faq-q3"></a>
### Q3: MySQL `utf8mb4_0900_ai_ci` vs `utf8mb4_unicode_ci` collation mismatch

**Cause**: MySQL 8.0 defaults to `utf8mb4_0900_ai_ci`; our schema enforces `utf8mb4_unicode_ci`.

**Fix**: after `CREATE DATABASE`, immediately `ALTER DATABASE cbmem CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci` (the `create-db` tool does this automatically).

<a id="faq-q4"></a>
### Q4: `/mcp` returns 401 Unauthorized

**Path to root cause**:

1. **Token expired**: reissue via `POST /admin/users/<id>/token?ttl=720h`
2. **JWT secret mismatch**: the server's `-jwt-secret` and the secret used to sign the token must be byte-for-byte identical
3. **Subject not a registered user**: the `sub` claim must exist in the `users` table
4. **`-admin-token` is not used here**: `/mcp` uses the user's JWT, NOT the admin token

<a id="faq-q5"></a>
### Q5: Cursor cannot connect

**Path to root cause**:

1. **URL/port correct**: default `:8787`; if behind nginx, use 443/80
2. **JWT expired**: reissue
3. **Service running**: `curl http://server:8787/healthz` to verify
4. **Query parameter order**: `?as=alice&project=/code/foo`, do not swap — server parses in strict order
5. **TLS**: Cursor will accept plain http, but production must be https

<a id="faq-q6"></a>
### Q6: SQLite database locked / write timeout

**Cause**: SQLite is single-writer; ≥ 8 concurrent writers trigger `database is locked`.

**Fix**:

- **Quick**: raise `busy_timeout` (default 5s; configurable via PRAGMA in the same directory as `-data`)
- **Permanent**: switch to the MySQL backend (`-mysql-dsn`)

<a id="faq-q7"></a>
### Q7: Switching from MySQL back to SQLite loses data

**Cause**: MySQL writes are not reverse-ETLed into SQLite; reverting to SQLite shows the pre-switch SQLite snapshot.

**Fix**: write a reverse ETL first (if you need it); otherwise stick to the MySQL track.

<a id="faq-q8"></a>
### Q8: `codebase-memory-mcp` under `$HOME` and systemd `ProtectHome=true` fails to start

**Cause**: systemd's default `ProtectHome=true` denies access to `/home/<user>`; the binary cannot be exec'd if it lives there.

**Fix** (pick one):

- Move the binary to `/usr/local/bin/codebase-memory-mcp`
- Add `ProtectHome=read-only` to the unit (the in-repo `deploy/cbmem-team.service` defaults to this)

<a id="faq-q9"></a>
### Q9: Bulk `curl -d` fails on PowerShell

**Cause**: PowerShell's `curl` is `Invoke-WebRequest`, which silently mangles JSON bodies.

**Fix**:

```powershell
# Option 1: body in a file
$body | Out-File -Encoding ascii body.json
curl -Method POST -InFile body.json -Uri http://...

# Option 2: use WSL / Git Bash
wsl curl -X POST http://...

# Option 3: use the in-repo examples/ scripts
bash examples/smoke.sh
```

<a id="faq-q10"></a>
### Q10: How do I tell whether an e2e really passed vs surface passed

**Answer**: each e2e tool has **ground-truth assertions**, for example:

- `e2e-v7` checkpoint 4: INSERT one row into `tool_invocation_logs` with `affected_halls_json=["..."]+error_code='UPSTREAM_ERROR'`, then GET `/v2/dashboard/high-risk` and confirm it **actually** appears in the response (not just HTTP 200)
- `e2e-v5` checkpoint 4: SQLite → MySQL ETL then `SELECT COUNT(*)` on both sides — diff must be 0

To see every step in detail, re-run `cbmem-team` with `-log debug`.

<a id="faq-q11"></a>
### Q11: Abnormal disk growth

**Cause**: `tool_invocation_logs` is the write-heaviest table (one row per MCP call).

**Fix**:

- Archive regularly: keep last 30 days online, move older rows to a cold table / S3
- Monitor: `du -sh /var/lib/cbmem-team/cbmem-team.db` or `du -sh /var/lib/cbmem-team/users/*`
- Adjust write strategy: batch + async

<a id="faq-q12"></a>
### Q12: How do I add a custom BP / workflow / repo pipeline?

- **BP**: `POST /api/console/v2/bps` (JSON: title/category/track/tools/...)
- **Workflow**: `POST /api/console/v2/workflows` (JSON: nodes array), then `POST /:id/publish`, finally `POST /:id/run`
- **Repo pipeline**: `POST /api/console/v2/repos` (JSON: name/source/target), `POST /:id/activate`, then `POST /:id/run`

<a id="faq-q13"></a>
### Q13: Large-file transfer during deploy fails

**Cause**: PuTTY's `pscp.exe` drops bytes on unstable networks.

**Fix**: use OpenSSH `scp`, and `sha256sum` both sides after transfer (hashes must match before you `install`).

<a id="faq-q14"></a>
### Q14: `docs-site/` VitePress build fails

**Prerequisites**: Node 20+, pnpm 9+.

```bash
cd docs-site
pnpm install
pnpm build      # artefacts in .vitepress/dist/
```

Then point `-console-dist` to the `dist/` directory; or rsync to `/opt/cbmem-console/dist`.

---

## Appendix A: Environment variables & config files

### A.1 systemd EnvironmentFile (`/etc/cbmem-team/cbmem-team.env`)

```ini
LISTEN=:8787
DATA_DIR=/var/lib/cbmem-team
MCP_BINARY=/usr/local/bin/codebase-memory-mcp
JWT_SECRET=<64-hex-chars>
ADMIN_TOKEN=<64-hex-chars>
LOG_LEVEL=info
IDLE_TTL=30m
MAX_PROCS_PER_USER=4
```

> The systemd unit loads this via `EnvironmentFile=`; CLI flags take precedence over env vars.

### A.2 dev-tool MySQL DSN resolution (`internal/devconf`)

All `cmd/*` tools resolve their MySQL DSN through `internal/devconf.ResolveMySQLDSN`, in this priority order:

1. `-mysql-dsn "..."` flag
2. `$CBMEM_MYSQL_DSN` environment variable
3. `$DSN` environment variable (legacy, kept for backward compat)
4. `dev default`: `root:adsop123@tcp(127.0.0.1:3306)/?...` (**dev only — not a production credential**)

Copy `deploy/dev.env.example` to `deploy/dev.env`, fill it in, then source it:

```bash
cp deploy/dev.env.example deploy/dev.env
# edit deploy/dev.env
set -a; source deploy/dev.env; set +a    # bash
# or PowerShell:
Get-Content deploy/dev.env | ForEach-Object { if ($_ -match '^([^#][^=]+)=(.*)$') { Set-Item -Path "Env:$($matches[1])" -Value $matches[2] } }
```

## Appendix B: Related docs

- [`README.md`](./README.md) — architecture diagram + client config + security checklist
- [`CONSOLE.md`](./CONSOLE.md) — 5 console modules + M3/M4 API reference
- [`BUILD.md`](./BUILD.md) — build + cross-compile + deploy to 192.168.100.83
- [`deploy/sql/README.md`](./deploy/sql/README.md) — how to use the SQL files
- [`deploy/dev.env.example`](./deploy/dev.env.example) — dev DSN env template
- [`cbmem-team-prd.md`](./cbmem-team-prd.md) — full product requirements (if present)

## Appendix C: Versions & history

- **v0.x**: single-user stdio (stock `codebase-memory-mcp`)
- **v1.0**: HTTP wrapper + JWT auth + per-user stdio pool
- **v1.5 (M0.5)**: SQLite ↔ MySQL dual track + 7-day rollback window
- **v2.0 (M1)**: tool directory + invocation log + console v1
- **v2.1 (M2)**: rate limit + BP CRUD + version snapshots
- **v2.2 (M3)**: workflow engine (draft/published/archive) + 3 built-in workflows
- **v2.3 (M4)**: repo pipeline + bp_candidate review + 2 MySQL views

---

*Last updated: 2026-07-12*

# cbmem-team User Manual

> **Version**: v3 (Role-oriented rewrite)
> **Target readers**: Developers / Operations
> **Prerequisites**: [`README.md`](./README.md) (architecture), [`CONSOLE.md`](./CONSOLE.md) (Console API reference)
> **Chinese version**: [`manual.md`](./manual.md)

---

## Preface

`cbmem-team` is an **HTTP multi-user wrapper** for `codebase-memory-mcp`, with an integrated **dual-track memory console**. It solves a core problem: `codebase-memory-mcp` is designed as a single-user local tool — `cbmem-team` wraps it into a team-shared remote service, keeping each user's code AST index strictly isolated, while adding session capture, knowledge distillation (summarize/distill), and a web-based management console.

### System Architecture

```
                         ┌──────────────────────────────────┐
                         │         Client Layer             │
                         │  Cursor / Qoder / Claude Desktop  │
                         └───────────────┬──────────────────┘
                                         │ HTTP + JWT Bearer
                         ┌───────────────▼──────────────────┐
                         │         cbmem-team               │
                         │  ┌────────────────────────────┐  │
                         │  │   Gin HTTP Server          │  │
                         │  │   JWT Auth / Capture       │  │
                         │  └────────────┬───────────────┘  │
                         │  ┌────────────┴───────────────┐  │
                         │  │ /mcp  │ /admin │ /console   │  │
                         │  └────────────────────────────┘  │
                         └──────────┬───────────┬───────────┘
                                    │           │
                    ┌───────────────▼──┐   ┌────▼──────────────┐
                    │  Process Pool    │   │   Console DB      │
                    │  (Per-user MCP)  │   │   (SQLite/MySQL)  │
                    └───────┬──────────┘   └────┬──────────────┘
                            │                    │
                    ┌───────▼──────────┐   ┌────▼──────────────┐
                    │ codebase-mem-mcp │   │   LLM Provider    │
                    │ (User A / B / …) │   │   (OpenAI/Ollama) │
                    └──────────────────┘   └────┬──────────────┘
                                                │
                                        ┌───────▼──────────┐
                                        │   MemPalace      │
                                        │   (optional)     │
                                        └──────────────────┘
```

### Prerequisites

| Dependency | Min Version | Purpose |
|---|---|---|
| **Go** | 1.24+ | Compile binary (no cgo) |
| **SQLite** | Built-in | Default backend (zero deps) |
| **MySQL** | 8.0 / 9.0 (optional) | Production backend; skip if `-mysql-dsn` not set |
| **Node.js** | 20+ | Only if building `/console/` frontend (VitePress) |
| **codebase-memory-mcp** | Latest | Backend MCP service, path via `-mcp-bin` |

---

# Part I: Developer Guide

---

## 1. MCP Client Configuration

This chapter covers how to configure MCP connections to cbmem-team in mainstream AI IDEs.

### 1.1 Obtaining a JWT Token

Connecting to cbmem-team requires a **JWT Bearer Token**. There are three ways to obtain one:

**Method 1: Admin issuance**

An admin issues a token via the Admin API:

```bash
# Admin executes (requires admin-token)
curl -X POST "http://server:8787/admin/users/<your-user-id>/token?ttl=720h" \
  -H "X-Admin-Token: <admin-token>"
```

Response example:

```json
{
  "user_id": "alice",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires": "2026-08-08T00:00:00Z"
}
```

**Method 2: Self-service renewal (no admin needed)**

If you hold a valid token, you can renew it yourself:

```bash
curl -sS -X POST "http://server:8787/refresh?ttl=4320h" \
  -H "Authorization: Bearer $OLD_JWT"
```

> `/refresh` does not require admin-token — it only verifies the caller's existing valid JWT.

**Method 3: Offline issuance**

Use the `cbmem-mint-token` tool to issue tokens without exposing the admin port:

```bash
cbmem-mint-token \
  --secret "$JWT_SECRET" \
  --user alice \
  --ttl 720h
# Output: eyJhbGciOiJIUzI1NiIs...
```

| Flag | Default | Description |
|---|---|---|
| `--secret` | (required) | Must match server's `-jwt-secret` exactly |
| `--user` | (required) | User ID (sub claim) |
| `--ttl` | `720h` | Token validity period |

### 1.2 Cursor Configuration

Edit `~/.cursor/mcp.json` (Windows: `%USERPROFILE%\.cursor\mcp.json`):

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://your-server:8787/mcp?as=alice&project=/path/to/project",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...."
      }
    }
  }
}
```

**URL Parameters**:

| Parameter | Description | Example |
|------|------|------|
| `as` | Your user ID, determines routing to per-user subprocess | `alice` |
| `project` | **Server-side** filesystem project path (not local path) | `/var/lib/cbmem-team/users/alice/repos/github.com__org_repo` |

**Notes**:
- `project` path must be a **server-side** path. After admin clones via `/admin/repos/clone`, the returned `local_path` is the correct value
- Don't expose `:8787` directly; put nginx/caddy in front for TLS termination in production
- Each developer only needs their own JWT

### 1.3 Qoder Configuration

Qoder's MCP configuration is similar to Cursor. Add MCP server in project root or global config:

**Method 1: Project-level config**

Create `.qoder/mcp.json` in project root:

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://your-server:8787/mcp?as=alice&project=/path/to/project",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...."
      }
    }
  }
}
```

**Method 2: Global config**

Edit `~/.qoder/mcp.json` with the same content.

### 1.4 Claude Desktop Configuration

Edit `~/.config/claude_desktop_config.json` (Windows: `%APPDATA%\Claude\claude_desktop_config.json`):

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

### 1.5 CodeBuddy Configuration

CodeBuddy supports MCP server configuration in Settings. Open CodeBuddy Settings → MCP Servers, add a new server:

- **Name**: `cbmem-team`
- **URL**: `http://your-server:8787/mcp?as=alice&project=/path/to/project`
- **Headers**:

```json
{
  "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...."
}
```

### 1.6 Token Renewal

JWT tokens have an expiration time (default issuance: 30 days). To renew:

```bash
# Self-service renewal (valid JWT required, no admin-token needed)
curl -sS -X POST "http://your-server:8787/refresh?ttl=4320h" \
  -H "Authorization: Bearer $OLD_JWT"
```

Replace the old token in your config file with the new JWT from the response.

### 1.7 Automated Configuration Scripts

The repository provides two scripts that automate the full flow: "mint JWT → write config → connectivity test":

| Script | Platform |
|------|------|
| `examples/cbmem-emit-mcp.ps1` | Windows / pwsh 7 |
| `examples/cbmem-emit-mcp.sh` | macOS / Linux |

**Linux / macOS**:

```bash
BASE=http://your-server:8787 \
ADMIN_TOKEN=$YOUR_ADMIN_TOKEN \
USER=alice \
PROJECT_SERVER_PATH=/var/lib/cbmem-team/users/alice/repos/github.com__org_repo \
bash examples/cbmem-emit-mcp.sh
```

**Windows**:

```powershell
pwsh examples/cbmem-emit-mcp.ps1 `
  -Base http://your-server:8787 `
  -AdminToken $YOUR_ADMIN_TOKEN `
  -User alice `
  -ProjectServerPath /var/lib/cbmem-team/users/alice/repos/github.com__org_repo `
  -Ttl 4320h
```

The script executes 5 steps automatically:
1. Call `POST /admin/users/<user>/token?ttl=<ttl>` to mint JWT
2. Load `~/.cursor/mcp.json` (create if missing)
3. Upsert the `cbmem-team` entry
4. Run `tools/list` smoke test
5. Print one-line summary

### 1.8 End-to-End Connectivity Verification

After configuration, run these verification steps:

**Step 1: Check server reachability**

```bash
curl http://your-server:8787/healthz
# Expected: {"status":"ok","users":N}
```

**Step 2: MCP initialize handshake**

```bash
curl -X POST "http://your-server:8787/mcp?as=alice&project=/code/foo" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"curl","version":"0"},"capabilities":{}}}'
```

Expected: returns `serverInfo` + `capabilities`. If you get 401, see [FAQ §401](#faq-401).

---

## 2. Console Operations Guide

cbmem-team provides a web console covering five modules: user management, project management, session records, session summarization, and session distillation.

### 2.1 Logging In

1. Open `http://server:8787/console/` in browser
2. Enter the **admin-token** (same as server's `-admin-token` parameter)
3. Click "Login"

**Authentication mechanism**:
- After login, server sets session cookie `cbmem_console` + CSRF token `cbmem_csrf`
- Non-GET requests automatically carry `X-CSRF-Token` header
- Session default TTL: 8 hours; re-login required after expiry
- Supports both `X-Admin-Token` and `Authorization: Bearer` for login

**Logout**: Click "Logout" button at top; session cookie is cleared.

### 2.2 User Management

Path: `/console/#/users`

**Create user**:

1. Click "New User"
2. Fill in fields:
   - **ID**: Unique user identifier (e.g., `alice`), immutable after creation
   - **Display Name**: Display name (e.g., `Alice Wang`)
   - **Project Paths Allowlist**: JSON array, e.g., `["/code/foo", "/code/bar"]`. Empty means unrestricted
3. Click "Submit"

**Management operations**:

| Operation | Description |
|------|------|
| Edit | Update display name, project paths allowlist, disabled status |
| Delete | Remove user from list |
| Issue Token | Generate JWT with TTL for this user (same as 1.1 Method 1) |
| Revoke Token | Disable user (`disabled=true`), all JWTs immediately invalidated |

**Search**: Enter keywords to search user ID or display name. List supports pagination, default 20 per page.

### 2.3 Project Management

Path: `/console/#/projects`

**Create project**:

1. Click "New Project"
2. Fill in fields:
   - **Name**: Project display name
   - **Path**: Server-side filesystem path (unique constraint)
   - **Wing**: MemPalace Wing binding (e.g., `engineering`), used for summarize/distill target
   - **MCP Binary Path**: Path to `codebase-memory-mcp` (optional, uses global default if empty)
3. Click "Submit"

**Management operations**:

| Operation | Description |
|------|------|
| View Index Status | Shows "Indexed / Not Indexed / Indexing" status tag |
| Trigger Index | Asynchronously runs `codebase-memory-mcp` to re-index project |
| Delete | Soft-delete project (associated session records preserved) |

### 2.4 Session Record Queries

Path: `/console/#/sessions`

**Session list**:

- Filter by user: Select target user
- Filter by project: Select target project
- Filter by time range: Set start/end dates
- Pagination supported

**Session details**:

Click "View" on a session; side drawer expands showing:
- Basic info: User ID, project path, start/end time, tool call count, turn count
- Turn list: Sorted by `turn_no`, each showing role (user/assistant) + content

**Session statistics**:

Call `GET /api/console/sessions-stats` or view in Console Dashboard:
- Total sessions, total turns, total tool calls
- Top users (by session count, descending)
- Top projects (by session count, descending)

**Session capture mechanism**:

cbmem-team wraps MCP handler with a capture middleware that automatically collects:
- Trigger conditions: JSON-RPC `initialize` / `tools/call` containing `messages/create`
- Reuse strategy: Same `(user_id, project_path)` reuses session if one exists within 30 minutes
- Async write: Does not block MCP requests

### 2.5 Session Summarization

Path: `/console/#/summarize`

Session summarization uses LLM to distill a set of sessions into 5 dimensions (5 Hall model).

**Steps**:

1. Select one or more sessions
2. Set summarization depth:
   - `shallow`: Quick overview
   - `deep`: Detailed analysis
   - `expert`: Most comprehensive
3. Set target Wing (e.g., `engineering`)
4. Click "Start Summarization"

**Results**:

After completion, results displayed in 5 Hall tabs:

| Hall | Meaning | Typical Content |
|------|------|---------|
| `hall_facts` | Facts, locked decisions | Tech decisions, architecture constraints |
| `hall_events` | Events, milestones | Meeting notes, debugging sessions |
| `hall_discoveries` | Breakthroughs, discoveries | Performance findings, bug root causes |
| `hall_preferences` | Habits, preferences | Code style, tool preferences |
| `hall_advice` | Advice, solutions | Best practices, pitfall avoidance |

**API calls**:

```bash
# Trigger summarization
curl -X POST http://server:8787/api/console/summarize \
  -H "X-CSRF-Token: <csrf>" -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"source_ids":["sess_xxx","sess_yyy"],"depth":"deep","target_wing":"engineering"}'

# Query task status
curl http://server:8787/api/console/summarize/<task_id> -b cookies.txt
```

### 2.6 Session Distillation

Path: `/console/#/distill`

Session distillation extracts reusable knowledge fragments, decision records, and tech debt markers from sessions.

**Steps**:

1. Select one or more sessions
2. Set distillation rules:
   - `min_value_score`: Minimum score threshold (default 0.7, higher = stricter)
   - `dimensions`: Extraction dimensions (default `["fact", "decision", "discovery"]`)
3. Set target Wing
4. Click "Start Distillation"

**Results**:

Distillation results include three types of extracts:
- **Knowledge fragments** (fact): Reusable technical knowledge
- **Decision records** (decision): Key decisions with context
- **Tech debt** (discovery): Identified tech debt and improvement suggestions

**Commit to MemPalace**:

After distillation, click "Write to MemPalace" to push results:
- Knowledge fragments → `room=room, hall=facts`
- Decisions → `room=room, hall=advice`

Commit is **idempotent** — repeated clicks won't cause duplicate writes.

**API calls**:

```bash
# Trigger distillation
curl -X POST http://server:8787/api/console/distill \
  -H "X-CSRF-Token: <csrf>" -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"source_ids":["sess_xxx"],"rules":{"min_value_score":0.7,"dimensions":["fact","decision","discovery"]}}'

# Query task status
curl http://server:8787/api/console/distill/<task_id> -b cookies.txt

# Write to MemPalace
curl -X POST http://server:8787/api/console/distill/<task_id>/commit \
  -H "X-CSRF-Token: <csrf>" -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"target_wing":"engineering"}'
```

---

## 3. MemPalace Integration

MemPalace is a local-first AI memory system using verbatim storage with semantic search retrieval. cbmem-team integrates with it via the `-mempalace-base` parameter, writing summarize/distill results to MemPalace.

### 3.1 Core Concepts

MemPalace uses a memory palace structured storage system:

```
Palace
  ├── Wing → People or projects
  │     └── Room → Topics
  │           └── Drawer → Verbatim content blocks
  │
  ├── Hall → Category labels
  │     ├── hall_facts       → Facts, decisions
  │     ├── hall_events      → Events, milestones
  │     ├── hall_discoveries → Discoveries, breakthroughs
  │     ├── hall_preferences → Preferences, habits
  │     └── hall_advice      → Advice, solutions
  │
  └── Tunnel → Cross-Wing connections
```

| Concept | Description | Example |
|------|------|------|
| **Wing** | Top-level org unit, represents people or projects | `wing_alice`, `wing_engineering` |
| **Room** | Topic classification within a Wing | `auth-migration`, `ci-pipeline` |
| **Hall** | Memory content type | `hall_facts`, `hall_advice`, etc. |
| **Drawer** | Stored verbatim text blocks | Full conversations, code snippets, decision records |

### 3.2 Writing Summarize/Distill Results to MemPalace

Configure MemPalace integration:

```bash
cbmem-team \
  -mempalace-base http://mempalace-server:8089 \
  -mempalace-timeout 10s \
  ...
```

**Write flow**:

1. Complete summarize or distill operation in console
2. After results generated, click "Write to MemPalace"
3. Specify `target_wing` (target Wing)
4. cbmem-team calls `POST {mempalace-base}/api/drawers` to write
5. Write mapping: knowledge fragments → `hall=facts`, decisions → `hall=advice`

> MemPalace has built-in auto-retry (3 attempts on failure).

### 3.3 Search and Retrieval

After writing to MemPalace, team members can search memories via MemPalace's MCP tools:

```bash
# Semantic search
mempalace search "why did we switch to GraphQL"

# Load context (new session startup)
mempalace wake-up
```

> For complete MemPalace installation and usage guide, see [MemPalace Official Docs](https://mempalaceofficial.com/).

---

## 4. codebase-memory-mcp Integration

### 4.1 What is codebase-memory-mcp

`codebase-memory-mcp` is a **single-user local** AI code memory tool. It builds code AST indexes for each project, providing semantic search via MCP protocol, enabling AI IDEs to understand your codebase.

Key features:
- SQLite-based local index storage
- MCP protocol standard interface, compatible with mainstream AI IDEs
- Automatic code structure and dependency analysis
- Incremental indexing support

### 4.2 Local Mode vs Team Mode

| Dimension | Local Mode (direct codebase-memory-mcp) | Team Mode (via cbmem-team) |
|------|------------------------------------------|---------------------------|
| Deployment | Each developer runs locally | Centralized server deployment |
| Protocol | stdio (standard I/O) | HTTP + JWT |
| Users | Single user | Multi-user |
| Index Isolation | Natural (local files) | Per-user process + SQLite file locks |
| Session Capture | None | Automatic |
| Knowledge Distillation | None | Summarize/Distill + MemPalace |
| Management UI | None | Web console |
| Use Case | Individual development | Team collaboration |

### 4.3 Index Directory Structure

cbmem-team maintains independent index directories per user:

```
${DataDir}/users/<userID>/projects/<sanitized_project>/
    └── codebase-memory-mcp (independent SQLite file)
```

Example:
```
/var/lib/cbmem-team/users/alice/projects/_code_foo/
/var/lib/cbmem-team/users/bob/projects/_var_lib_cbmem-team_users_bob_repos_github.com__org_repo/
```

`<sanitized_project>` is the project path with `/` replaced by `_` for safe directory naming.

### 4.4 MCP Protocol Interaction Flow

Clients interact with cbmem-team via HTTP; cbmem-team forwards to the corresponding per-user `codebase-memory-mcp` stdio subprocess:

```
1. initialize    → Handshake, get serverInfo + capabilities
2. tools/list    → Get available tool list
3. tools/call    → Call specific tools (e.g., semantic search, index build)
4. ping          → Heartbeat check
```

Request example:

```bash
# initialize handshake
POST /mcp?as=alice&project=/code/foo
Authorization: Bearer <jwt>
Content-Type: application/json

{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"cursor","version":"1.0"},"capabilities":{}}}
```

### 4.5 Automatic Session Capture

cbmem-team wraps the MCP handler with a **capture middleware** that automatically collects all conversation sessions:

**Trigger conditions**:
- JSON-RPC `initialize` request
- `tools/call` or `tools/invoke` calls
- Message calls containing `messages` array

**Captured content**:
- Each trigger → creates or reuses a `sessions` row
- Each turn → writes to `session_turns` (role, content, tools_json, ts)

**Reuse strategy**: Same `(user_id, project_path)` reuses session if one exists within 30 minutes, otherwise creates new session.

**Performance impact**: Capture runs asynchronously in a separate goroutine, never blocking MCP requests. Writes are serialized via per-user channels to avoid SQLite lock contention.

---

# Part II: Operations Guide

---

## 5. Deployment & Startup

### 5.1 Build

```bash
cd tools/cbmem-team

# Build main service
go build -o bin/cbmem-team ./cmd/cbmem-team

# Build offline JWT signing tool
go build -o bin/cbmem-mint-token ./cmd/cbmem-mint-token
```

Cross-platform build:

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o bin/cbmem-team-linux ./cmd/cbmem-team

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/cbmem-team.exe ./cmd/cbmem-team
```

### 5.2 Foreground Mode (Development / Debugging)

```bash
cd tools/cbmem-team
mkdir -p bin/data

./bin/cbmem-team \
  -listen :8787 \
  -data ./bin/data \
  -mcp-bin $(which codebase-memory-mcp) \
  -jwt-secret "dev-jwt-secret-please-rotate" \
  -admin-token "dev-admin-token-please-rotate" \
  -llm-provider fake \
  -log info
```

Success indicators:

```
console db: driver=sqlite
console db: migrate console + M1..M4 schemas
seed tool_directory (49 rows)
seed best_practices (3 rows)
seed workflows (3 rows)
listening on :8787
```

#### Required Flags

| Flag | Description | Example |
|---|---|---|
| `-listen` | HTTP listen address | `:8787` |
| `-data` | Data directory (SQLite location) | `/var/lib/cbmem-team` |
| `-mcp-bin` | `codebase-memory-mcp` binary absolute path | `/usr/local/bin/codebase-memory-mcp` |
| `-jwt-secret` | JWT HS256 signing key (**must change in production**) | `openssl rand -hex 32` |
| `-admin-token` | Admin static token (**must change in production**) | `openssl rand -hex 32` |

#### Optional Flags

| Flag | Default | Description |
|---|---|---|
| `-config` | `/etc/cbmem-team/config.yaml` | YAML config file path (CLI flags take priority) |
| `-log` | `info` | Log level: `debug` / `info` / `warn` / `error` |
| `-mysql-dsn` | Empty (= SQLite) | MySQL DSN; non-empty switches backend to MySQL 8.0 |
| `-mysql-max-open` | `16` | MySQL connection pool max |
| `-mysql-max-idle` | `4` | MySQL idle connections |
| `-mysql-max-lifetime` | `30m` | MySQL max connection lifetime |
| `-llm-provider` | `fake` | `fake` / `openai` / `ollama` |
| `-llm-model` | Empty | Model name (e.g., `gpt-4o-mini`) |
| `-llm-base-url` | Empty | LLM API base URL |
| `-llm-api-key` | Empty | LLM API key (optional for Ollama) |
| `-llm-timeout` | `30s` | LLM request timeout |
| `-mempalace-base` | Empty | MemPalace HTTP base URL, empty = disabled |
| `-mempalace-timeout` | `10s` | MemPalace request timeout |
| `-console-dist` | Empty | Console frontend VitePress build output directory |
| `-idle-ttl` | `30m` | Idle process recycle time |
| `-max-procs-per-user` | `4` | Max processes per user |

### 5.3 systemd Background Deployment (Recommended for Production)

#### Prepare Environment File

`/etc/cbmem-team/cbmem-team.env`:

```ini
LISTEN=:8787
DATA_DIR=/var/lib/cbmem-team
MCP_BINARY=/usr/local/bin/codebase-memory-mcp
JWT_SECRET=<64-hex-chars>        # openssl rand -hex 32
ADMIN_TOKEN=<64-hex-chars>       # openssl rand -hex 32
LOG_LEVEL=info
IDLE_TTL=30m
MAX_PROCS_PER_USER=4
```

Lock down permissions:

```bash
sudo install -m 0600 deploy/cbmem-team.env /etc/cbmem-team/cbmem-team.env
```

#### Install systemd Unit

The repository provides two units:

| Unit | Database | Purpose |
|------|--------|------|
| `deploy/cbmem-team.service` | SQLite | Fallback unit |
| `deploy/cbmem-team-mysql.service` | MySQL | **Production primary** |

```bash
sudo install -m 0644 deploy/cbmem-team-mysql.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now cbmem-team-mysql
sudo systemctl status cbmem-team-mysql --no-pager
```

#### View Logs

```bash
# Follow in real-time
sudo journalctl -u cbmem-team-mysql -f

# Last 100 lines
sudo journalctl -u cbmem-team-mysql -n 100 --no-pager
```

### 5.4 MySQL Deployment

#### Step 1: Start MySQL

```bash
cd tools/cbmem-team/deploy
docker compose -f docker-compose.mysql.yml up -d
```

#### Step 2: Configure DSN

```bash
sudo cp deploy/mysql.env.example /etc/cbmem-team/mysql.env
sudo chmod 0600 /etc/cbmem-team/mysql.env
# Edit /etc/cbmem-team/mysql.env, set strong random password
```

#### Step 3: Create Tables

```bash
cbmem-team migrate-tables -mysql-dsn "$(grep CBMEM_MYSQL_DSN /etc/cbmem-team/mysql.env | cut -d= -f2)"
```

> `migrate-tables` is idempotent, safe to run repeatedly.

#### Step 4: Initial ETL (Migrate from SQLite)

```bash
./examples/mysql-etl.sh --mysql-dsn "$(grep CBMEM_MYSQL_DSN /etc/cbmem-team/mysql.env | cut -d= -f2)"
```

#### Step 5: Switch to MySQL

```bash
sudo systemctl disable cbmem-team
sudo systemctl enable --now cbmem-team-mysql
```

#### Step 6: Install Backup Cron

```bash
sudo cp examples/mysql-backup.sh /etc/cron.daily/cbmem-mysql-backup
```

### 5.5 SQLite ↔ MySQL Switching

Both units share the `/var/lib/cbmem-team` data directory; only one can run at a time:

```bash
# Switch to SQLite (emergency fallback)
sudo systemctl stop cbmem-team-mysql
sudo systemctl start cbmem-team       # Old unit without -mysql-dsn flag

# Switch back to MySQL
sudo systemctl stop cbmem-team
sudo systemctl start cbmem-team-mysql
```

> **Note**: MySQL writes do not back-fill SQLite. Switching back to SQLite shows the snapshot from before the switch.

#### Subcommands

| Subcommand | Purpose |
|---|---|
| `cbmem-team migrate-tables -mysql-dsn <dsn>` | Create MySQL tables (idempotent) |
| `cbmem-team migrate-sqlite-to-mysql -sqlite <path> -mysql-dsn <dsn>` | SQLite → MySQL ETL |
| `cbmem-team mysql-ping -mysql-dsn <dsn>` | Check MySQL connectivity |

---

## 6. User & Project Management

### 6.1 Create User + Issue Token

```bash
ADMIN="your-admin-token"

# Create user alice
curl -X POST http://server:8787/admin/users \
  -H "X-Admin-Token: $ADMIN" \
  -H "Content-Type: application/json" \
  -d '{"id":"alice","display_name":"Alice","project_paths":["/code/foo"]}'

# Issue 30-day token
curl -X POST "http://server:8787/admin/users/alice/token?ttl=720h" \
  -H "X-Admin-Token: $ADMIN"
```

**User data structure**:

```json
{
  "id": "alice",
  "display_name": "Alice",
  "project_paths": ["/code/foo", "/code/bar"],
  "max_procs": 4,
  "disabled": false,
  "created_at": "2026-01-01T00:00:00Z"
}
```

**Admin endpoints**:

| Endpoint | Method | Description |
|------|------|------|
| `/admin/users` | GET | List all users |
| `/admin/users` | POST | Create user |
| `/admin/users/:id` | PATCH | Partial update (display_name / disabled / project_paths) |
| `/admin/users/:id` | DELETE | Delete user + clear process pool |
| `/admin/users/:id/token` | POST | Issue JWT (`?ttl=720h`) |
| `/admin/stats` | GET | Global statistics |

### 6.2 Manage Projects + Trigger Index

```bash
# Create project
curl -X POST http://server:8787/api/console/projects \
  -H "X-Admin-Token: $ADMIN" \
  -H "Content-Type: application/json" \
  -d '{"name":"My Project","path":"/code/foo","wing":"engineering"}'

# Trigger re-index
curl -X POST http://server:8787/api/console/projects/<project_id>/reindex \
  -H "X-Admin-Token: $ADMIN"
```

### 6.3 Offline JWT Issuance

Use `cbmem-mint-token` tool without exposing the admin port:

```bash
cbmem-mint-token \
  --secret "$JWT_SECRET" \
  --user alice \
  --ttl 720h
# Output: eyJhbGciOiJIUzI1NiIs...
```

### 6.4 Clone Remote Repository

Admins can clone Git repositories for users remotely:

```bash
curl -X POST http://server:8787/admin/repos/clone \
  -H "X-Admin-Token: $ADMIN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","url":"https://github.com/org/repo.git"}'
```

After cloning:
- Repository stored at `${DataDir}/users/<userID>/repos/<safeName>/`
- Repository path automatically added to user's `project_paths` allowlist

---

## 7. Advanced Features

### 7.1 Tool Directory + Invocation Logs (M1)

49 built-in tool directory entries, automatic logging of all MCP tool calls.

| Endpoint | Description |
|------|------|
| `GET /api/console/v2/tools` | Tool list |
| `GET /api/console/v2/dashboard/summary` | Call statistics |
| `GET /api/console/v2/dashboard/recent-invocations` | Recent calls |

### 7.2 Rate Limiting + Best Practices CRUD (M2)

**Rate limiting**:

- Dimension: `(user_id, tool_id)` bucket
- Over-threshold returns HTTP 429 + `retry_after`
- Adjust: `PATCH /api/console/v2/rate-limit`

**Best Practices**:

| Endpoint | Description |
|------|------|
| `GET /api/console/v2/bps` | BP list |
| `POST /api/console/v2/bps` | Create BP |
| `PUT /api/console/v2/bps/:id` | Update BP (auto version snapshot) |
| `DELETE /api/console/v2/bps/:id` | Delete BP |
| `GET /api/console/v2/bps/:id/graph` | BP relationship graph |

### 7.3 Workflow Engine (M3)

Workflows are publishable + executable graph node sequences in the console.

**Built-in workflows** (auto-seeded with migration):

| ID | Name | Nodes | Purpose |
|----|------|--------|------|
| `wf-commit-precheck` | Commit Pre-check | 7 | Auto compliance checks before high-risk commits |
| `wf-adr-doublewrite` | ADR Double-write | 6 | Write ADR to both MemPalace + file |
| `wf-repo-daily-sync` | Repo Daily Sync | 4 | Daily cron pull repo pipeline increments |

**State flow**: `draft` → `published` (read-only graph) → `archived`

| Endpoint | Description |
|------|------|
| `GET /api/console/v2/workflows` | List |
| `POST /api/console/v2/workflows` | Create |
| `POST /api/console/v2/workflows/:id/publish` | Publish |
| `POST /api/console/v2/workflows/:id/run` | Execute synchronously |

### 7.4 Repo Pipeline (M4)

Repo pipelines process external content through 6-stage runtime into `bp_candidates`:

```
crawl → parse → grade → sink → rehearse
```

**BP Candidate states**: `draft` → `accepted` / `rejected`; `accepted` → `merged`

| Endpoint | Description |
|------|------|
| `POST /api/console/v2/repos` | Create pipeline |
| `POST /api/console/v2/repos/:id/activate` | Activate |
| `POST /api/console/v2/repos/:id/run` | Execute once |
| `GET /api/console/v2/repos/candidates` | Candidate BP list |
| `POST /api/console/v2/repos/candidates/:id/accept` | Accept candidate |
| `POST /api/console/v2/repos/candidates/:id/merge` | Merge to best_practices |

---

## 8. CLI Tool Reference

### 8.1 Production Tools

| Tool | Build Command | Purpose |
|---|---|---|
| `cbmem-team` | `go build -o bin/cbmem-team ./cmd/cbmem-team` | Main service |
| `cbmem-mint-token` | `go build -o bin/cbmem-mint-token ./cmd/cbmem-mint-token` | Offline JWT signing |

### 8.2 Ops / Migration Tools

| Tool | Build Command | Purpose |
|---|---|---|
| `create-db` | `go build -o bin/create-db ./cmd/create-db` | Create `cbmem` MySQL database |
| `mysql-probe` | `go build -o bin/mysql-probe ./cmd/mysql-probe` | MySQL connectivity probe |
| `show-tables` | `go build -o bin/show-tables ./cmd/show-tables` | List all tables |
| `show-indexes` | `go build -o bin/show-indexes ./cmd/show-indexes` | List all indexes |

### 8.3 E2E Test Tools (dev / CI only)

| Tool | Coverage | Run Command |
|---|---|---|
| `e2e-m1` | M1 Tool directory + invocation logs | `go run ./cmd/e2e-m1` |
| `e2e-v5` | M0.5 SQLite/MySQL dual-track + ETL | `go run ./cmd/e2e-v5` |
| `e2e-v6` | M2 Rate limiting + BP CRUD | `go run ./cmd/e2e-v6` |
| `e2e-v7` | M3 Workflows + M4 Pipeline | `go run ./cmd/e2e-v7` |

Common flag: `-mysql-dsn "..."` or via environment variable `$CBMEM_MYSQL_DSN`.

### 8.4 Tool Quick Reference

| Tool | Primary Purpose | Production |
|---|---|---|
| `cbmem-team` | Main service (HTTP wrapper + console) | ✅ |
| `cbmem-mint-token` | Offline JWT signing | ✅ |
| `create-db` | Create MySQL database | ✅ |
| `mysql-probe` | Probe MySQL | ✅ |
| `show-tables` | List all tables | ✅ |
| `show-indexes` | List all indexes | ✅ |
| `seed-sqlite` | Write SQLite test data | ❌ |
| `e2e-m1` ~ `e2e-v7` | E2E acceptance | ❌ |
| `mcp-stub` | E2E MCP stub | ❌ |
| `mint-token-debug` | Debug 401 | ❌ |

---

# Part III: Reference

---

## 9. Configuration Parameters

### 9.1 Required Parameters

| Parameter | Description | Example |
|------|------|------|
| `-listen` | HTTP listen address | `:8787` |
| `-data` | Data directory (contains SQLite) | `/var/lib/cbmem-team` |
| `-mcp-bin` | `codebase-memory-mcp` binary path | `/usr/local/bin/codebase-memory-mcp` |
| `-jwt-secret` | JWT signing key | `openssl rand -hex 32` |
| `-admin-token` | Console admin static token | `openssl rand -hex 32` |
| `-console-dist` | VitePress build output directory | `/opt/cbmem-console/dist` |

### 9.2 LLM Parameters

| Parameter | Description | Default |
|------|------|------|
| `-llm-provider` | `openai` / `ollama` / `fake` | `fake` |
| `-llm-model` | Model name | `gpt-4o-mini` |
| `-llm-base-url` | API base URL | `https://api.openai.com/v1` |
| `-llm-api-key` | API key | `sk-...` |
| `-llm-timeout` | Timeout duration | `30s` |

### 9.3 MemPalace Parameters

| Parameter | Description | Default |
|------|------|------|
| `-mempalace-base` | MemPalace HTTP address | `http://localhost:8089` |
| `-mempalace-timeout` | Timeout duration | `10s` |

### 9.4 Optional Parameters

| Parameter | Description | Default |
|------|------|------|
| `-config` | Config file path | `/etc/cbmem-team/config.yaml` |
| `-log` | Log level | `info` |
| `-idle-ttl` | Idle process recycle time | `30m` |
| `-max-procs-per-user` | Max processes per user | `4` |
| `-mysql-dsn` | MySQL DSN (enables MySQL if non-empty) | Empty |
| `-mysql-max-open` | MySQL max open connections | `16` |
| `-mysql-max-idle` | MySQL max idle connections | `4` |
| `-mysql-max-lifetime` | MySQL max connection lifetime | `30m` |

---

## 10. Database Schema

### 10.1 SQLite and MySQL Dual-Track

cbmem-team supports two backends, switched via startup flag:

| Backend | Activation | When to Use |
|------|------|------|
| **SQLite** (default) | Don't set `-mysql-dsn` flag | Single machine / dev / small team |
| **MySQL 8.0** | `-mysql-dsn <dsn>` | Production / multi-instance / high concurrency |

### 10.2 Core Tables

| Table | Purpose |
|------|------|
| `users` | User info (including project_paths whitelist) |
| `projects` | Project config (including wing / mcp_bin) |
| `sessions` | Session records |
| `session_turns` | Session turns |

### 10.3 Task Tables

| Table | Purpose |
|------|------|
| `summarize_tasks` | Summarization tasks |
| `distill_tasks` | Distillation tasks |

### 10.4 Console Tables

| Table | Purpose |
|------|------|
| `console_sessions` | Console login sessions (with TTL) |

### 10.5 M1-M4 Extension Tables

| Table | Milestone | Purpose |
|------|------|------|
| `tool_directory` | M1 | Tool directory (49 built-in entries) |
| `tool_invocation_logs` | M1 | Tool invocation logs |
| `best_practices` + `bp_versions` | M2 | Best practices + version snapshots |
| `workflows` + `workflow_versions` + `workflow_runs` | M3 | Workflow engine |
| `tickets` | M3 | Tickets |
| `repo_pipelines` + `repo_pipeline_runs` + `bp_candidates` | M4 | Repo Pipeline |

---

## 11. API Quick Reference

### 11.1 Admin Endpoints (/admin/*)

| Path | Method | Description |
|------|--------|------|
| `/admin/users` | GET | List users |
| `/admin/users` | POST | Create user |
| `/admin/users/:id` | PATCH | Update user |
| `/admin/users/:id` | DELETE | Delete user |
| `/admin/users/:id/token` | POST | Issue token |
| `/admin/stats` | GET | Global stats |
| `/admin/repos/clone` | POST | Clone repository |
| `/admin/repos` | GET | List repositories |

### 11.2 Console Endpoints (/api/console/*)

| Path | Method | Description |
|------|--------|------|
| `/api/console/login` | POST | Admin-token login |
| `/api/console/logout` | POST | Logout |
| `/api/console/me` | GET | Current admin |
| `/api/console/stats` | GET | Global stats |
| `/api/console/users` | GET | User list |
| `/api/console/users` | POST | Create user |
| `/api/console/users/:id` | PUT | Update user |
| `/api/console/users/:id` | DELETE | Delete user |
| `/api/console/users/:id/token` | POST | Issue token |
| `/api/console/users/:id/revoke` | POST | Revoke all tokens |
| `/api/console/projects` | GET | Project list |
| `/api/console/projects` | POST | Create project |
| `/api/console/projects/:id` | DELETE | Delete project |
| `/api/console/projects/:id/reindex` | POST | Trigger indexing |
| `/api/console/sessions` | GET | Session list |
| `/api/console/sessions/:id` | GET | Session detail |
| `/api/console/sessions-stats` | GET | Stats aggregation |
| `/api/console/summarize` | POST | Trigger summarization |
| `/api/console/summarize/:task_id` | GET | Task status |
| `/api/console/distill` | POST | Trigger distillation |
| `/api/console/distill/:task_id` | GET | Task status |
| `/api/console/distill/:task_id/commit` | POST | Write to MemPalace |

### 11.3 MCP Endpoints

| Path | Method | Description |
|------|--------|------|
| `/mcp` | POST | Streamable HTTP (v2 primary) |
| `/mcp/` | POST | Same, trailing `/` compatibility |
| `/mcp/sse` | GET | SSE (v1 fallback) |
| `/healthz` | GET | Health check |
| `/refresh` | POST | JWT auto-renewal |

### 11.4 Unified Response Format + Error Codes

```json
{ "code": 0, "msg": "", "data": {} }
```

| Range | Meaning |
|------|------|
| `0` | Success |
| `40100xx` | Auth failure (wrong admin-token / session expired / CSRF) |
| `40300xx` | Forbidden (whitelist violation / user disabled) |
| `40900xx` | Resource conflict (e.g., duplicate user ID) |
| `50000xx` | System error (LLM timeout / SQLite write failure) |

---

## 12. FAQ

<a id="faq-401"></a>
### Q1: `/mcp` returns 401 Unauthorized

**Troubleshooting steps**:

1. **Token expired**: Re-issue via `POST /admin/users/<id>/token?ttl=720h`
2. **JWT secret mismatch**: Server's `-jwt-secret` and the secret used to sign the token must be **character-for-character identical**
3. **JWT subject is a registered user**: The sub claim must exist in the `users` table
4. **Admin token is not for here**: `/mcp` uses user JWT, not `-admin-token`

### Q2: Cursor client can't connect

**Troubleshooting steps**:

1. **Port correct**: Default `:8787`; if behind nginx reverse proxy, use 443/80
2. **JWT expired**: Re-issue
3. **Server running**: Verify with `curl http://server:8787/healthz`
4. **URL parameter order**: `?as=alice&project=/code/foo` — don't reverse
5. **TLS**: HTTP works for Cursor, but production must use HTTPS

### Q3: Compilation errors (undefined)

**Cause**: GoLand Run Configuration uses single-file path.

**Fix**: Change Run Configuration "Run kind" to **Package**, path: `./cmd/cbmem-team`.

### Q4: SQLite database locked

**Cause**: SQLite is single-writer; 8+ concurrent threads trigger `database is locked`.

**Fix**: Switch to MySQL backend (`-mysql-dsn`).

### Q5: Switching from MySQL back to SQLite loses data

**Cause**: MySQL writes don't back-fill SQLite.

**Fix**: Stay on MySQL single-track, or perform reverse ETL.

### Q6: systemd ProtectHome startup failure

**Cause**: `codebase-memory-mcp` is under `$HOME` and systemd denies access.

**Fix**: Move binary to `/usr/local/bin/`, or add `ProtectHome=read-only` to the unit file.

### Q7: PowerShell curl -d fails

**Cause**: PowerShell's `curl` is `Invoke-WebRequest`, which mangles JSON body.

**Fix**: Use WSL / Git Bash, or write body to file and use `-InFile`.

### Q8: Unexpected disk growth

**Cause**: `tool_invocation_logs` is the largest write table.

**Fix**: Periodically archive data older than 30 days; monitor `du -sh /var/lib/cbmem-team/`.

---

## Appendix

### A. Environment Variables & Config Files

#### systemd EnvironmentFile (`/etc/cbmem-team/cbmem-team.env`)

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

> systemd unit loads via `EnvironmentFile=`; CLI flags take precedence over env vars.

#### MySQL DSN Resolution for dev tools

All `cmd/*` tools resolve MySQL DSN uniformly via `internal/devconf.ResolveMySQLDSN`, priority:

1. `-mysql-dsn "..."` flag
2. `$CBMEM_MYSQL_DSN` environment variable
3. `$DSN` environment variable (legacy name, backward compatible)
4. `dev default` (dev only — **not production credentials**)

### B. Related Documentation

- [`README.md`](./README.md) — Architecture + client config + security checklist
- [`CONSOLE.md`](./CONSOLE.md) — Console 5 modules + M3/M4 API reference
- [`BUILD.md`](./BUILD.md) — Build + cross-compilation + deployment
- [`deploy/sql/README.md`](./deploy/sql/README.md) — SQL file usage
- [`deploy/dev.env.example`](./deploy/dev.env.example) — Dev DSN env template
- [`cbmem-team-prd.md`](./cbmem-team-prd.md) — Full product requirements

### C. Version History

- **v0.x**: Single-user stdio (`codebase-memory-mcp` native)
- **v1.0**: HTTP wrapper + JWT auth + per-user stdio pool
- **v1.5 (M0.5)**: SQLite ↔ MySQL dual-track + 7-day rollback window
- **v2.0 (M1)**: Tool directory + invocation logs + Console v1
- **v2.1 (M2)**: Rate limiting + BP CRUD + version snapshots
- **v2.2 (M3)**: Workflow engine + 3 built-in workflows
- **v2.3 (M4)**: Repo pipeline + bp_candidate review + 2 MySQL views
- **v3.0**: User manual rewrite (role-oriented)

---

*Last updated: 2026-07-08*

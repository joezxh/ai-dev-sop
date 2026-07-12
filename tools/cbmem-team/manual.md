# cbmem-team 用户手册

> **版本**：v2（M3+M4 已合并 / 双轨数据库）
> **适用读者**：运维 / 开发 / 客户端集成方
> **前置阅读**：[`README.md`](./README.md)（架构图）、[`CONSOLE.md`](./CONSOLE.md)（控制台 API 速查）、[`BUILD.md`](./BUILD.md)（构建+部署）

本文档面向**动手操作**——把项目从零启动到完整运行，按 5 个使用场景组织：

1. [快速上手](#1-快速上手) — 5 分钟跑起来
2. [前台 + 后台启动指南](#2-前台--后台启动指南) — 含 systemd / foreground / Docker
3. [MCP 服务配置与连接](#3-mcp-服务配置与连接) — Cursor / Claude / Qoder
4. [命令行工具总览](#4-命令行工具总览) — `cmd/` 下每个工具的功能、flag、用法
5. [功能特性全景](#5-功能特性全景) — 按 M0..M4 里程碑的模块清单
6. [常见问题排查 FAQ](#6-常见问题排查-faq)

---

## 1. 快速上手

### 1.1 前置条件

| 依赖 | 最低版本 | 用途 |
|---|---|---|
| **Go** | 1.24+ | 编译二进制（无 cgo 依赖） |
| **SQLite** | 内置 | 默认后端（零依赖） |
| **MySQL** | 8.0 / 9.0（可选） | 生产后端；未配置 `-mysql-dsn` 时跳过 |
| **Node.js** | 20+ | 仅当需要构建 `/console/` 前端（VitePress） |
| **codebase-memory-mcp** | 最新 | 后端 MCP 服务，路径通过 `-mcp-bin` 指定 |

### 1.2 第一次启动（开发机 + SQLite 默认）

```bash
cd tools/cbmem-team

# 1. 编译主二进制（Windows 下 go build 即可，Linux/Mac 同理）
go build -o bin/cbmem-team ./cmd/cbmem-team

# 2. 准备数据目录
mkdir -p bin/data

# 3. 启动（前台模式，便于看日志）
./bin/cbmem-team \
  -listen :8787 \
  -data ./bin/data \
  -mcp-bin /usr/local/bin/codebase-memory-mcp \
  -jwt-secret "dev-secret-please-rotate" \
  -admin-token "dev-admin-token-please-rotate" \
  -log info
```

启动成功的标志：

```
console db: driver=sqlite
console db: migrate console + M1..M4 schemas
seed tool_directory (49 rows)
seed best_practices (3 rows)
seed workflows (3 rows)
listening on :8787
```

### 1.3 健康检查

```bash
# 服务是否通
curl http://127.0.0.1:8787/healthz
# {"status":"ok","users":0}

# 管理接口是否通（替换为你的 -admin-token）
curl -H "X-Admin-Token: dev-admin-token-please-rotate" http://127.0.0.1:8787/admin/stats
```

### 1.4 创建第一个用户并签发 token

```bash
ADMIN="dev-admin-token-please-rotate"

# 创建用户 alice
curl -X POST http://127.0.0.1:8787/admin/users \
  -H "X-Admin-Token: $ADMIN" \
  -H "Content-Type: application/json" \
  -d '{"id":"alice","display_name":"Alice","project_paths":["/code/foo"]}'

# 签发 30 天 token
curl -X POST "http://127.0.0.1:8787/admin/users/alice/token?ttl=720h" \
  -H "X-Admin-Token: $ADMIN"
```

---

## 2. 前台 + 后台启动指南

### 2.1 启动模式总览

| 模式 | 命令 | 何时用 |
|---|---|---|
| **前台调试** | `./cbmem-team <flags>` | 开发、看日志、临时启动 |
| **systemd 后台** | `systemctl start cbmem-team` | 生产部署（推荐） |
| **Docker Compose** | `docker compose -f deploy/docker-compose.mysql.yml up -d` | 仅起 MySQL；服务二进制仍跑在主机 |

### 2.2 前台模式（开发 / 调试）

前台模式最直接，所有日志实时输出到 stderr。**所有 flag 通过 `-flag value` 形式传递**，参考下表：

#### 2.2.1 必需 flag

| Flag | 说明 | 示例 |
|---|---|---|
| `-listen` | HTTP 监听地址 | `:8787` |
| `-data` | 数据目录（SQLite 落点） | `/var/lib/cbmem-team` |
| `-mcp-bin` | `codebase-memory-mcp` 二进制绝对路径 | `/usr/local/bin/codebase-memory-mcp` |
| `-jwt-secret` | JWT HS256 签名密钥（**生产必换**） | `openssl rand -hex 32` |
| `-admin-token` | 管理端静态 token（**生产必换**） | `openssl rand -hex 32` |

#### 2.2.2 可选 flag

| Flag | 默认 | 说明 |
|---|---|---|
| `-config` | `/etc/cbmem-team/config.yaml` | YAML 配置文件路径（可选；CLI flag 优先） |
| `-log` | `info` | 日志级别：`debug` / `info` / `warn` / `error` |
| `-mysql-dsn` | 空（= SQLite） | MySQL DSN；非空时切换后端到 MySQL 8.0 |
| `-mysql-max-open` | `16` | MySQL 连接池上限 |
| `-mysql-max-idle` | `4` | MySQL 空闲连接数 |
| `-mysql-max-lifetime` | `30m` | MySQL 连接最大存活时间 |
| `-llm-provider` | `fake` | `fake` / `openai` / `ollama` |
| `-llm-model` | 空 | 模型名称（如 `gpt-4o-mini`） |
| `-llm-base-url` | 空 | LLM API base URL |
| `-llm-api-key` | 空 | LLM API 密钥（Ollama 可不填） |
| `-mempalace-base` | 空 | MemPalace HTTP base URL，空则禁用 |
| `-console-dist` | 空 | 控制台前端 VitePress 产物目录 |

#### 2.2.3 完整开发机启动示例（SQLite 路径）

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

#### 2.2.4 完整生产机启动示例（MySQL 路径）

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

### 2.3 后台 systemd 部署（推荐生产）

#### 2.3.1 准备配置文件

`/etc/cbmem-team/cbmem-team.env`（环境文件）：

```ini
LISTEN=:8787
DATA_DIR=/var/lib/cbmem-team
MCP_BINARY=/usr/local/bin/codebase-memory-mcp
JWT_SECRET=$(openssl rand -hex 32)        # 替换为强随机
ADMIN_TOKEN=$(openssl rand -hex 32)       # 替换为强随机
LOG_LEVEL=info

IDLE_TTL=30m
MAX_PROCS_PER_USER=4
```

权限必须锁紧（环境文件含敏感信息）：

```bash
sudo install -m 0600 deploy/cbmem-team.env /etc/cbmem-team/cbmem-team.env
```

#### 2.3.2 安装 systemd unit

仓库提供两套 unit：

- `deploy/cbmem-team.service` — SQLite 路径，**回退单元**
- `deploy/cbmem-team-mysql.service` — MySQL 路径，**生产主用**

```bash
sudo install -m 0644 deploy/cbmem-team-mysql.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now cbmem-team-mysql
sudo systemctl status cbmem-team-mysql --no-pager
```

#### 2.3.3 查看日志

```bash
# 实时跟踪
sudo journalctl -u cbmem-team-mysql -f

# 最近 100 行
sudo journalctl -u cbmem-team-mysql -n 100 --no-pager
```

#### 2.3.4 7 天回退窗口（SQLite ↔ MySQL）

两套 unit 共用 `/var/lib/cbmem-team` 数据目录；任一时刻只能跑一个：

```bash
# 切到 SQLite（紧急回退）
sudo systemctl stop cbmem-team-mysql
sudo systemctl start cbmem-team       # 旧单元无 -mysql-dsn flag

# 切回 MySQL
sudo systemctl stop cbmem-team
sudo systemctl start cbmem-team-mysql
```

### 2.4 子命令（管理类，非 serve）

`cbmem-team` 二进制除 `serve`（默认）外，还内置 3 个管理类子命令：

| 子命令 | 用途 | 何时用 |
|---|---|---|
| `cbmem-team migrate-tables -mysql-dsn <dsn>` | 在 MySQL 上建 7 张 console 表 | 首次部署 MySQL |
| `cbmem-team migrate-sqlite-to-mysql -sqlite <path> -mysql-dsn <dsn>` | 把 SQLite 数据一次性 ETL 到 MySQL | 切换后端 |
| `cbmem-team mysql-ping -mysql-dsn <dsn>` | 检查 DSN 是否可达 | 排错时快速验证 |

```bash
# 1. 建 MySQL 表（首次）
cbmem-team migrate-tables -mysql-dsn "root:pw@tcp(host:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4"

# 2. SQLite → MySQL ETL（一次性）
cbmem-team migrate-sqlite-to-mysql \
  -sqlite /var/lib/cbmem-team/cbmem-team.db \
  -mysql-dsn "root:pw@tcp(host:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4"

# 3. 验证连接
cbmem-team mysql-ping -mysql-dsn "root:pw@tcp(host:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4"
```

### 2.5 构建产物说明

`go build` 完成后产生 4 类二进制：

| 二进制 | 用途 | 是否生产 |
|---|---|---|
| `cbmem-team` | 主服务（HTTP wrapper + 控制台） | ✅ |
| `cbmem-mint-token` | 离线签发 JWT（无需暴露 `/admin`） | ✅ |
| `mcp-stub` | 确定性 MCP 后端（仅 e2e 用） | ❌ |
| `mint-token-debug` | 调试 401 用的临时 JWT minter | ❌ |

---

## 3. MCP 服务配置与连接

`cbmem-team` 对外暴露 **3 个 MCP 端点**：

| 路径 | 方法 | 用途 |
|---|---|---|
| `/mcp` | POST | Streamable HTTP（v2 主用） |
| `/mcp/` | POST | 同上，兼容尾部 `/` |
| `/mcp/sse` | GET | SSE（v1 回退） |

每个端点都要求 `Authorization: Bearer <JWT>`，subject (`sub`) 即 user_id。

### 3.1 Cursor 客户端配置

`~/.cursor/mcp.json`：

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

**关键点**：

- `${workspaceFolder}` 必须替换为团队约定的字面路径（如 `/code/foo`）；该路径决定 server 端路由到哪个 stdio 子进程的 `workdir`
- 每个开发人员只需要**自己的 JWT**——URL 的 `as=...` 是 server 端隔离 user 索引的依据
- 不要直接暴露 `:8787`；前面挂 nginx/caddy 做 TLS 终止

### 3.2 Claude Desktop 客户端配置

`~/.config/claude_desktop_config.json`：

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

### 3.3 离线签发 JWT（不暴露 /admin）

`cbmem-mint-token` 二进制让你在不开放管理端口的前提下签发 token：

```bash
cbmem-mint-token \
  --secret "$JWT_SECRET" \
  --user alice \
  --ttl 720h
# 输出: eyJhbGciOiJIUzI1NiIs...
```

flag：

| Flag | 默认 | 说明 |
|---|---|---|
| `--secret` | （必填） | 与服务端 `-jwt-secret` 完全一致 |
| `--user` | （必填） | 用户 ID（sub claim） |
| `--ttl` | `720h` | token 有效期 |

### 3.4 端到端连通性验证

```bash
# 1. 服务端 healthz 通
curl http://your-server:8787/healthz

# 2. 客户端 POST /mcp（initialize 握手）
curl -X POST http://your-server:8787/mcp?as=alice\&project=/code/foo \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"curl","version":"0"},"capabilities":{}}}'

# 期望: 返回 serverInfo + capabilities
```

---

## 4. 命令行工具总览

`tools/cbmem-team/cmd/` 下每个子目录是一个独立的 Go 包，编译后产生一个二进制。共 **14 个工具**（13 业务 + 1 dev）：

### 4.1 生产级工具（部署到服务器）

#### 4.1.1 `cbmem-team` — 主服务

- **路径**：`cmd/cbmem-team/`
- **入口**：`main.go` + `handlers.go` + 同包其他文件
- **编译**：`go build -o bin/cbmem-team ./cmd/cbmem-team`
- **用法**：见第 2 章 + 第 2.4 节子命令
- **flag 详表**：见 [2.2.1](#221-必需-flag) / [2.2.2](#222-可选-flag)
- **关键端点**：`/healthz`、`/admin/*`、`/api/console/*`、`/api/console/v2/*`、`/mcp`、`/mcp/sse`

#### 4.1.2 `cbmem-mint-token` — 离线 JWT minter

- **路径**：`cmd/cbmem-mint-token/`
- **编译**：`go build -o bin/cbmem-mint-token ./cmd/cbmem-mint-token`
- **用法**：

  ```bash
  cbmem-mint-token --secret "$JWT_SECRET" --user alice --ttl 720h
  ```

- **flag**：`--secret`、`--user`（必填）、`--ttl`（默认 720h）

### 4.2 运维 / 迁移工具（部署期 / 切换期使用）

#### 4.2.1 `create-db` — 创建 `cbmem` 数据库

- **路径**：`cmd/create-db/`
- **编译**：`go build -o bin/create-db ./cmd/create-db`
- **作用**：连 MySQL → `CREATE DATABASE IF NOT EXISTS cbmem CHARACTER SET utf8mb4` + `ALTER DATABASE ... COLLATE utf8mb4_unicode_ci` → 验证
- **flag**：

  | Flag | 默认 | 说明 |
  |---|---|---|
  | `-mysql-dsn` | `$CBMEM_MYSQL_DSN` / `$DSN` / `dev default` | MySQL DSN（库名 cbmem 不必含） |

- **示例**：

  ```bash
  create-db -mysql-dsn "root:pw@tcp(127.0.0.1:3306)/?parseTime=true&loc=Local&charset=utf8mb4"
  ```

#### 4.2.2 `mysql-probe` — MySQL 连接 + 服务器特性探测

- **路径**：`cmd/mysql-probe/`
- **作用**：检查 DSN 可达 + 报告 VERSION / collation / flush_log 配置
- **flag**：`-mysql-dsn`（同上）
- **示例**：

  ```bash
  $ mysql-probe -mysql-dsn "$DSN"
  MySQL version : 8.0.36 (Linux)
  Charset/Coll  : utf8mb4 / utf8mb4_0900_ai_ci
  Flush log trx : 1
  DB cbmem      : exists
  ```

#### 4.2.3 `show-tables` — 列出所有表 + engine + 行数

- **路径**：`cmd/show-tables/`
- **作用**：扫描 `information_schema.tables`，输出 table / engine / collation / 列数 / KB
- **flag**：`-mysql-dsn`（同上）

#### 4.2.4 `show-indexes` — 列出所有索引

- **路径**：`cmd/show-indexes/`
- **作用**：扫描 `information_schema.statistics`，输出 table / index / columns
- **flag**：同上

#### 4.2.5 `seed-sqlite` — 写入 SQLite 测试数据

- **路径**：`cmd/seed-sqlite/`
- **作用**：在 `D:\projects\ai-dev-sop\tools\cbmem-team\bin\test-cbmem.db` 写入 5 user / 3 project / 4 session / 30 turn / 2 summarize / 2 distill / 1 console_session——为 e2e-v5 的 ETL 验证准备数据
- **flag**：无（路径写死）
- **何时用**：仅 `e2e-v5` 自动调用；生产环境不部署

#### 4.2.6 `migrate-tables` 子命令 — 见 2.4

#### 4.2.7 `migrate-sqlite-to-mysql` 子命令 — 见 2.4

#### 4.2.8 `mysql-ping` 子命令 — 见 2.4

### 4.3 端到端测试工具（仅 dev / CI）

所有 e2e 工具都遵循相同模式：编译 → 运行 → 启动 `cbmem-team` 子进程 → 发请求 → 校验 → 退出。**生产环境不部署**。

| 工具 | 覆盖里程碑 | 检查点 | DSN |
|---|---|---|---|
| **`e2e-m1`** | M1 工具目录 + 调用日志 | 6 项：服务启动、admin/users、tool_directory ≥1、log 0→≥1、streamable SSE、dashboard summary | `-mysql-dsn` |
| **`e2e-v5`** | M0.5 SQLite/MySQL 双轨 + ETL | 5 项：SQLite serve、MySQL serve、migrate-tables 幂等、ETL 行数 1:1、切换后一致 | 同上 |
| **`e2e-v6`** | M2 工具限流 + BP CRUD + 关联图 | 5 项：49 tool、qps=1 → 429、POST bp、PUT bp+1 version、graph 节点 + UI | 同上 |
| **`e2e-v7`** | M3 工作流 + M4 repo pipeline + bp_candidate review | 7 项：3 seed workflow、create+run、3 内置 run、high-risk dashboard、ticket auto-open、repo CRUD、candidate review | 同上 |

#### 通用 flag（4 个 e2e 都支持）

| Flag | 默认 | 说明 |
|---|---|---|
| `-mysql-dsn` | `$CBMEM_MYSQL_DSN` / `$DSN` / `dev default` | MySQL DSN；解析规则见 `internal/devconf` |

#### 通用运行示例

```bash
cd tools/cbmem-team
go run ./cmd/e2e-v7                  # 用 dev default MySQL
go run ./cmd/e2e-v7 -mysql-dsn "..."  # 自定义 DSN
CBMEM_MYSQL_DSN="..." go run ./cmd/e2e-v7   # 通过 env 覆盖
```

### 4.4 调试工具

#### 4.4.1 `mint-token-debug` — 手写 JWT 调试 401

- **路径**：`cmd/mint-token-debug/`
- **作用**：直接输出一个 HS256 token，硬编码 secret `dev-secret-m1-7f8a-2026` 和 sub `alice-0`，仅用来快速验证 server 端签名校验逻辑
- **何时用**：仅当你怀疑 server 端 JWT 校验有问题（不要在生产使用）

#### 4.4.2 `mcp-stub` — 确定性 MCP 后端（仅 e2e 用）

- **路径**：`cmd/mcp-stub/`
- **作用**：实现最小 MCP JSON-RPC over stdio；`initialize` / `tools/list` / `tools/call` / `ping` 返回固定响应；`tools/call` 故意 sleep 200ms 以触发限流
- **何时用**：作为 `cbmem-team` 的 `-mcp-bin` 替换，e2e 测试用；**不要**当作真正的 MCP 服务

### 4.5 Dev 工具（仅本地）

| 工具 | 路径 | 作用 |
|---|---|---|
| `_dev/verify_sqlite` | `cmd/_dev/verify_sqlite/` | 验证 `deploy/sql/{schema,seed}.sql` 在 in-memory SQLite 上能跑（dump_sql.py 的姊妹工具） |

### 4.6 工具速查表

| 工具 | 编译命令 | 默认 port / 路径 | 主要用途 |
|---|---|---|---|
| `cbmem-team` | `go build -o bin/cbmem-team ./cmd/cbmem-team` | `:8787` | 主服务 |
| `cbmem-mint-token` | `go build -o bin/cbmem-mint-token ./cmd/cbmem-mint-token` | n/a | 离线签 JWT |
| `create-db` | `go build -o bin/create-db ./cmd/create-db` | `:3306` | 建 `cbmem` 库 |
| `mysql-probe` | `go build -o bin/mysql-probe ./cmd/mysql-probe` | `:3306` | 探活 MySQL |
| `show-tables` | `go build -o bin/show-tables ./cmd/show-tables` | `:3306` | 列所有表 |
| `show-indexes` | `go build -o bin/show-indexes ./cmd/show-indexes` | `:3306` | 列所有索引 |
| `seed-sqlite` | `go build -o bin/seed-sqlite ./cmd/seed-sqlite` | 本地 .db | 写 SQLite 测试数据 |
| `e2e-m1` | `go run ./cmd/e2e-m1` | `:28791..28795` | M1 验收 |
| `e2e-v5` | `go run ./cmd/e2e-v5` | `:18787` | M0.5 验收 |
| `e2e-v6` | `go run ./cmd/e2e-v6` | `:28796` | M2 验收 |
| `e2e-v7` | `go run ./cmd/e2e-v7` | `:28797` | M3+M4 验收 |
| `mcp-stub` | `go run ./cmd/mcp-stub` | stdio | e2e 用 MCP 桩 |
| `mint-token-debug` | `go run ./cmd/mint-token-debug` | n/a | 调试 401 |
| `_dev/verify_sqlite` | `go run ./cmd/_dev/verify_sqlite deploy/sql` | in-mem | 验证 SQL 文件 |

---

## 5. 功能特性全景

按里程碑划分，每个 M{n} 都包含一组相关 endpoint + 一组表格 + 一个 seed 数据集。

### 5.1 M0 — 基础 HTTP wrapper

- **核心**：把单进程 `codebase-memory-mcp` 包装成多用户 HTTP 服务
- **端点**：
  - `GET  /healthz` — 健康检查
  - `POST /refresh` — JWT 自动续期（持有有效 token 的用户自行续）
  - `POST /mcp` / `POST /mcp/` — Streamable HTTP（v2 主用）
  - `GET  /mcp/sse` — SSE（v1 回退）
- **多用户隔离**：
  - 一个 user 一个 stdio 子进程
  - workdir 形如 `/var/lib/cbmem-team/users/<user_id>/projects/<project>/`
  - SQLite 文件锁天然隔离
- **Idle 回收**：默认 30 分钟无请求即 kill 子进程（`-idle-ttl` 可调）

### 5.2 M0.5 — SQLite ↔ MySQL 双轨

- **核心**：单机能跑（SQLite）+ 生产高并发（MySQL）
- **端点**：不变，全部沿用 M0
- **数据库**：
  - **SQLite**：默认；文件 `<data>/cbmem-team.db`
  - **MySQL**：通过 `-mysql-dsn` flag 启用；通过 `migrate-tables` 建表
- **ETL**：`migrate-sqlite-to-mysql` 一次性把所有行从 SQLite 搬到 MySQL
- **特性**：
  - `migrate-tables` 幂等（连跑 2 次不报错）
  - 切换 backend 不丢数据（共用 `/var/lib/cbmem-team` 下的 SQLite 快照）

### 5.3 M1 — 工具目录 + 调用日志（CON-01）

- **数据库**：`tool_directory`（49 个内置工具）+ `tool_invocation_logs`
- **端点**：
  - `GET /api/console/v2/tools` — 工具列表
  - `GET /api/console/v2/dashboard/summary` — 调用统计
  - `GET /api/console/v2/dashboard/recent-invocations` — 最近调用
- **采集**：MCP handler 外包 capture middleware，自动写入 `tool_invocation_logs`
- **Seed**：`SeedToolDirectory`（49 行）

### 5.4 M2 — 工具限流 + Best Practices CRUD（CON-02）

- **数据库**：`best_practices` + `bp_versions`（版本快照）
- **端点**：
  - `GET/POST/PUT/DELETE /api/console/v2/bps[/:id]`
  - `GET /api/console/v2/bps/:id/graph` — BP 关联图
  - `PATCH /api/console/v2/rate-limit` — 限流阈值调整
- **特性**：
  - 限流维度：`(user_id, tool_id)` 桶；qps 超阈值返回 429 + `retry_after`
  - BP 版本：每次 PUT 都向 `bp_versions` 追加 snapshot
- **Seed**：`SeedBPs`（3 个跨类别 BP：naming / performance / collaboration）

### 5.5 M3 — 工作流引擎（CON-06）

- **数据库**：`workflows` + `workflow_versions` + `workflow_runs` + `tickets`
- **节点 kind**：`log` / `tool_call` / `bp_check` / `condition` / `wait_approval` / `parallel` / `loop_guard`
- **端点**：见 [`CONSOLE.md` §工作流](./CONSOLE.md)
- **特性**：
  - 3 个内置工作流随 migration 自动 seed：`wf-commit-precheck` / `wf-adr-doublewrite` / `wf-repo-daily-sync`
  - `draft` → `published` 后只读图；`published` → `archived`
  - `POST /run` 同步执行，返回 `run_id` + `step_count`
- **Seed**：`SeedWorkflows`（3 个内置）

### 5.6 M4 — Repo Pipeline + BP Candidates（CON-07）

- **数据库**：`repo_pipelines` + `repo_pipeline_runs` + `bp_candidates`
- **6 阶段 runtime**：`crawl` → `parse` → `grade` → `sink` → `rehearse`
- **端点**：见 [`CONSOLE.md` §Repo Pipeline](./CONSOLE.md)
- **特性**：
  - source 类型：`local` / `github` / `rss`（设计阶段已留接口）
  - BP candidate 三态：`draft` → `accepted` / `rejected`；`accepted` → `merged` 写入 `best_practices`
  - 视图 `v_repo_pipeline_success_rate_7d`（MySQL）
- **MySQL 视图**：
  - `v_high_risk_invocations_7d`（M3）
  - `v_repo_pipeline_success_rate_7d`（M4）

### 5.7 控制台前端（`/console/`）

- **登录**：浏览器打开 `http://server:8787/console/`，输入 admin token
- **5 大模块**：
  - 用户管理 (`#/users`)
  - 项目管理 (`#/projects`)
  - 会话记录 (`#/sessions`)
  - 会话归纳 (`#/summarize`)
  - 会话蒸馏 (`#/distill`)
- **M3 UI**：工作流编辑器
- **M4 UI**：repo pipeline 管理 + BP candidate review
- **构建**：`cd docs-site && pnpm install && pnpm build`，产物在 `.vitepress/dist/`，通过 `-console-dist` 指向

### 5.8 通用功能

- **统一响应**：`{"code":0, "msg":"", "data":{}}`
- **错误码区间**：
  - `0` — 成功
  - `40100xx` — 鉴权失败（admin-token 错 / session 过期 / CSRF）
  - `40300xx` — 无权（白名单违规 / 用户 disabled）
  - `40900xx` — 资源冲突（如 user id 重复）
  - `50000xx` — 系统错误（LLM 超时 / SQLite 写入失败）
- **会话采集（MCP Tap）**：每次 `initialize` / `tools/call` 含 `messages/create` 自动创建/复用 `sessions` + `session_turns`；30 分钟内有最近 session 则复用

---

## 6. 常见问题排查 FAQ

### Q1：编译报 `undefined: requestLogger` / `refreshTokenHandler` 等

**原因**：GoLand 的 Run/Debug Configuration 用了单文件路径：

```bash
go build -o xxx main.go    # ❌ 只编译 main.go
```

**修法**：把 Run Configuration 的 "Run kind" 改为 **Package**，路径填 `./cmd/cbmem-team`。等价命令行：

```bash
go build -o bin/cbmem-team ./cmd/cbmem-team    # ✅ 整包编译
```

### Q2：MySQL 报错 `ERROR 1061 (42000): duplicate key name`

**原因**：`CREATE INDEX IF NOT EXISTS` 在 MySQL 8.0 上不支持；重复跑 `schema-mysql.sql` 会撞同名索引。

**修法**：直接用 `cbmem-team migrate-tables -mysql-dsn <dsn>`（它会先 `information_schema.statistics` 探测）；`schema-mysql.sql` 仅用于**全新数据库**。

### Q3：MySQL `utf8mb4_0900_ai_ci` vs `utf8mb4_unicode_ci` 字符序不对齐

**原因**：MySQL 8.0 默认 `utf8mb4_0900_ai_ci`，我们的 schema 强制 `utf8mb4_unicode_ci`。

**修法**：在 `CREATE DATABASE` 后立即 `ALTER DATABASE cbmem CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`（`create-db` 工具已自动做这件事）。

### Q4：`/mcp` 报 401 Unauthorized

**排查路径**：

1. **token 是否过期**：`POST /admin/users/<id>/token?ttl=720h` 重发
2. **JWT secret 一致**：server 端 `-jwt-secret` 和签发 token 时用的 secret 必须**字符级一致**
3. **JWT subject 是否是注册用户**：sub claim 必须在 `users` 表存在
4. **admin token 不是这里用的**：`/mcp` 用的是用户 JWT，不是 `-admin-token`

### Q5：Cursor 客户端连不上

**排查路径**：

1. **URL 端口对吗**：默认 `:8787`，如果你用 nginx 反代要换成 443/80
2. **JWT 过期**：重新签发
3. **服务端是否在跑**：`curl http://server:8787/healthz` 验证
4. **URL 参数顺序**：`?as=alice&project=/code/foo`，**不能颠倒**——server 端严格按 query 解析
5. **TLS**：Cursor 走 http（明文）也可以，但生产必须 https

### Q6：SQLite 数据库锁死 / 写入超时

**原因**：SQLite 单写者；并发 8 线程以上会触发 `database is locked`。

**修法**：

- **临时**：拉高 `busy_timeout`（默认 5s，可在 `-data` 同级加 PRAGMA）
- **永久**：切到 MySQL 后端（`-mysql-dsn`）

### Q7：MySQL 后端切回 SQLite 时丢数据

**原因**：MySQL 写入不会回写 SQLite；切回 SQLite 后看到的是切换前的 SQLite 快照。

**修法**：先 `migrate-sqlite-to-mysql` 反向 ETL（如果写了反向）；或保持 MySQL 单轨。

### Q8：`codebase-memory-mcp` 在 `$HOME` 下时 systemd `ProtectHome=true` 启动失败

**原因**：systemd unit 默认拒绝访问 `/home/<user>`；`codebase-memory-mcp` 二进制在那里就 exec 不到。

**修法**（任选其一）：

- 把二进制移到 `/usr/local/bin/codebase-memory-mcp`
- unit 加 `ProtectHome=read-only`（仓库 `deploy/cbmem-team.service` 默认这么写）

### Q9：批量 `curl -d` 在 PowerShell 下失败

**原因**：PowerShell 的 `curl` 是 `Invoke-WebRequest`，会偷偷 mangle JSON body。

**修法**：

```powershell
# 写法 1：body 写文件
$body | Out-File -Encoding ascii body.json
curl -Method POST -InFile body.json -Uri http://...

# 写法 2：用 WSL / Git Bash
wsl curl -X POST http://...

# 写法 3：用仓库的 examples/ 脚本
bash examples/smoke.sh
```

### Q10：怎么看 e2e 是真的过了还是只表面过了？

**答**：每个 e2e 工具都有**真值断言**，例如：

- `e2e-v7` 第 4 项：在 `tool_invocation_logs` INSERT 一行 `affected_halls_json=["..."]+error_code='UPSTREAM_ERROR'`，然后 GET `/v2/dashboard/high-risk` 确认它**真的**出现在响应里（不是仅 HTTP 200）
- `e2e-v5` 第 4 项：SQLite → MySQL ETL 后，SELECT COUNT(*) 在两端对比，差 0 才算过

如果想看每一步的细节，加 `-log debug` 重跑 `cbmem-team` 主服务。

### Q11：磁盘增长异常

**原因**：`tool_invocation_logs` 是写入最大的表（每次 MCP 调用一行）。

**修法**：

- 定期归档：保留近 30 天在线，30 天前移到冷表 / S3
- 监控：`du -sh /var/lib/cbmem-team/cbmem-team.db` 或 `du -sh /var/lib/cbmem-team/users/*`
- 调参：写入策略可改成批量 + 异步

### Q12：怎么添加自定义 BP / 工作流 / repo pipeline？

- **BP**：`POST /api/console/v2/bps`（JSON：title/category/track/tools/...）
- **工作流**：`POST /api/console/v2/workflows`（JSON：nodes 数组），然后 `POST /:id/publish`，最后 `POST /:id/run`
- **Repo Pipeline**：`POST /api/console/v2/repos`（JSON：name/source/target），`POST /:id/activate` 后 `POST /:id/run`

### Q13：部署到服务器时传输大文件失败

**原因**：PuTTY 的 `pscp.exe` 在网络不稳定时丢字节。

**修法**：用 OpenSSH `scp`，传输后 `sha256sum` 双向校验（必须相等再 install）。

### Q14：`docs-site/` 里 VitePress 构建失败

**前置**：Node 20+、pnpm 9+。

```bash
cd docs-site
pnpm install
pnpm build      # 产物在 .vitepress/dist/
```

然后把 `dist/` 目录通过 `-console-dist` 指向；或者 rsync 到 `/opt/cbmem-console/dist`。

---

## 附录 A：环境变量与配置文件

### A.1 systemd EnvironmentFile（`/etc/cbmem-team/cbmem-team.env`）

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

> systemd unit 通过 `EnvironmentFile=` 加载；CLI flag 优先级高于 env。

### A.2 dev 工具的 MySQL DSN 解析（`internal/devconf`）

所有 `cmd/*` 工具统一通过 `internal/devconf.ResolveMySQLDSN` 解析，优先级：

1. `-mysql-dsn "..."` flag
2. `$CBMEM_MYSQL_DSN` 环境变量
3. `$DSN` 环境变量（旧名，向后兼容）
4. `dev default`：`root:adsop123@tcp(127.0.0.1:3306)/?...`（仅开发用，**不是生产凭证**）

复制 `deploy/dev.env.example` 到 `deploy/dev.env`，填好后 source 即可统一：

```bash
cp deploy/dev.env.example deploy/dev.env
# 编辑 deploy/dev.env
set -a; source deploy/dev.env; set +a    # bash
# 或 PowerShell:
Get-Content deploy/dev.env | ForEach-Object { if ($_ -match '^([^#][^=]+)=(.*)$') { Set-Item -Path "Env:$($matches[1])" -Value $matches[2] } }
```

## 附录 B：相关文档

- [`README.md`](./README.md) — 架构图 + 客户端配置 + 安全清单
- [`CONSOLE.md`](./CONSOLE.md) — 控制台 5 大模块 + M3/M4 API 速查
- [`BUILD.md`](./BUILD.md) — 构建 + 跨平台编译 + 部署到 192.168.100.83
- [`deploy/sql/README.md`](./deploy/sql/README.md) — SQL 文件使用方式
- [`deploy/dev.env.example`](./deploy/dev.env.example) — dev DSN env 模板
- [`cbmem-team-prd.md`](./cbmem-team-prd.md) — 完整产品需求（如果存在）

## 附录 C：版本与变更

- **v0.x**：单用户 stdio（`codebase-memory-mcp` 原生）
- **v1.0**：HTTP wrapper + JWT 鉴权 + per-user stdio pool
- **v1.5 (M0.5)**：SQLite ↔ MySQL 双轨 + 7 天回退窗口
- **v2.0 (M1)**：工具目录 + 调用日志 + 控制台 v1
- **v2.1 (M2)**：限流 + BP CRUD + 版本快照
- **v2.2 (M3)**：工作流引擎（draft/published/archive）+ 3 个内置 workflow
- **v2.3 (M4)**：repo pipeline + bp_candidate review + 2 个 MySQL 视图

---

*文档最后更新：2026-07-12*

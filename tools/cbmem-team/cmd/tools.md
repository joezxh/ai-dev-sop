# cmd/ 工具入口说明

`tools/cbmem-team/cmd/` 目录下包含多个独立的 Go 程序入口，各自承担特定职责。所有 MySQL 诊断工具均通过 `devconf.ResolveMySQLDSN` 解析 DSN，优先级为：命令行 flag → 环境变量 `$CBMEM_MYSQL_DSN` → 环境变量 `$DSN` → 开发默认值。

---

## 1. cbmem-team（主服务）

**功能：** HTTP 包装 `codebase-memory-mcp`，使多个开发者共享同一远端实例，同时通过 `user_id` 将每个用户的代码索引严格隔离。

**入口（main）：** 子命令分发器，支持以下运行方式：

| 调用方式 | 说明 |
|---|---|
| `cbmem-team`（无参数） | 默认启动 HTTP 服务（`runServe`） |
| `cbmem-team migrate-tables` | 在 MySQL 上创建 7 张 console 表 |
| `cbmem-team migrate-sqlite-to-mysql` | 一次性 ETL，从 SQLite 迁移到 MySQL |
| `cbmem-team mysql-ping` | 检测 MySQL 连接是否可达 |

**`runServe` 启动流程：**

1. `config.ParseFlags` 解析全部配置
2. 加载 `users.json`，初始化 repos 管理器、stdio 进程池
3. 初始化 JWT 验证器 & Admin 静态 Token 校验
4. 打开 Console DB（MySQL 或 SQLite）
5. 注册路由：

| 路由 | 说明 |
|---|---|
| `GET /healthz` | 健康检查，返回 `{"status":"ok","users":N}` |
| `POST /mcp` | MCP Streamable HTTP 端点（需 JWT） |
| `GET /mcp` | Streamable GET 端点（需 JWT） |
| `GET /mcp/sse` | MCP SSE 端点（需 JWT） |
| `POST /refresh` | 用户自主刷新 JWT（无需 admin） |
| `POST /admin/*` | 用户 CRUD、token 签发、repos 管理、统计（需 admin token） |
| `/console/*` | SPA 静态文件（可选，由 `-console-dist` 启用） |

6. Console 模块路由由 `console.Mount` 统一装配（M1–M6 各模块）
7. Signal-aware graceful shutdown

**关键配置参数：**

```
-listen              监听地址（默认 :18765）
-data                数据目录（users.json、sqlite、repos 等）
-admin-token         Admin 静态鉴权 Token
-jwt-secret          JWT 签名密钥
-mysql-dsn           MySQL DSN（设置后自动切 MySQL 后端）
-mcp-bin             codebase-memory-mcp 二进制路径
-repo-root           Git 仓库根目录（M3 teams/projects v2）
-console-dist        SPA 静态文件目录
-config-file         热更新配置文件路径
```

---

## 2. cbmem-mint-token（离线 JWT 签发）

**功能：** 离线签发 JWT Bearer Token，给已知用户颁发访问凭证，避免将 `/admin/*` 暴露在公网。

**入口（main）：** 解析三个 flag 后调用 `auth.NewVerifier` 签发 HS256 JWT，输出到 stdout。

```
cbmem-mint-token --secret $JWT_SECRET --user alice --ttl 720h
```

| 参数 | 默认值 | 说明 |
|---|---|---|
| `--secret` | 必填 | JWT 共享密钥 |
| `--user` | 必填 | 用户 ID（写入 `sub` claim） |
| `--ttl` | `720h`（30 天） | Token 有效期 |

---

## 3. create-db（MySQL 数据库初始化）

**功能：** 一次性 bootstrap 工具，幂等创建 `cbmem` 数据库（utf8mb4 + unicode_ci），并验证当前表数为 0。

**入口（main）：**

1. 解析 `-mysql-dsn`（支持 env 回退）
2. 用无 schema 的 bootstrap DSN 执行：
   ```sql
   CREATE DATABASE IF NOT EXISTS cbmem CHARACTER SET utf8mb4;
   ALTER DATABASE cbmem CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
   ```
3. 重新连接带 schema 的 DSN，查 `information_schema.tables` 计数

> 注：MySQL 9.0 仍拒绝 `IF NOT EXISTS` 与 `CHARACTER SET` 在同一条语句中组合，故拆分为两条执行。

---

## 4. mcp-stub（MCP stdio 替身）

**功能：** 极简 MCP stdio 服务端，用于 e2e 测试场景，避免依赖真实的 `codebase-memory-mcp` 二进制。其 `tools/call` 内置 200ms 延时，便于验证限流器行为。

**入口（main）：** 逐行读取 stdin JSON-RPC 请求，按 method 分发：

| Method | 行为 |
|---|---|
| `initialize` | 返回 serverInfo + capabilities |
| `tools/list` | 返回 `stub-echo` 工具 |
| `tools/call` | sleep 200ms → 返回 `{"content":[{"type":"text","text":"ok"}]}` |
| `ping` | 返回空 result |
| 其他 | 返回空 result，不报错 |

所有响应以 `\n` 结尾，符合 MCP stdio 协议。

---

## 5. mint-token-debug（调试用 JWT 签发）

**功能：** 固定使用开发密钥 `dev-secret-m1-7f8a-2026` 为 `sub=alice-0` 签发 1 小时有效期的 JWT。用于手动调试 401 响应，无需任何参数。

**入口（main）：** 纯标准库（无外部依赖），直接打印 JWT 到 stdout。

---

## 汇总表

| 程序 | 类型 | 用途 |
|---|---|---|
| `cbmem-team` | 主服务 | HTTP + MCP 中介，多用户隔离 |
| `cbmem-mint-token` | 工具 | 离线 JWT 签发 |
| `create-db` | 工具 | MySQL 数据库初始化 |
| `mcp-stub` | 工具 | MCP stdio 测试替身 |
| `mint-token-debug` | 工具 | 调试用固定 JWT |

# cbmem-team 用户手册

> **版本**：v3（角色导向重写版）
> **适用读者**：开发者 / 运维
> **前置阅读**：[`README.md`](./README.md)（架构图）、[`CONSOLE.md`](./CONSOLE.md)（控制台 API 速查）
> **英文版**：[`manual.en.md`](./manual.en.md)

---

## 前言

`cbmem-team` 是 `codebase-memory-mcp` 的 **HTTP 多用户包装器**，同时提供**双轨记忆控制台**。它解决的核心问题：`codebase-memory-mcp` 设计为单用户本地工具，`cbmem-team` 将其包装为团队共享的远程服务，保持每个用户的代码 AST 索引严格隔离，并在此基础上提供会话捕获、知识提炼（归纳/蒸馏）和控制台管理界面。

### 系统架构

```
                         ┌──────────────────────────────────┐
                         │          客户端层                 │
                         │  Cursor / Qoder / Claude Desktop  │
                         └───────────────┬──────────────────┘
                                         │ HTTP + JWT Bearer
                         ┌───────────────▼──────────────────┐
                         │         cbmem-team               │
                         │  ┌────────────────────────────┐  │
                         │  │   Gin HTTP Server          │  │
                         │  │   JWT 验证 / 会话捕获      │  │
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
                                        │   (可选)          │
                                        └──────────────────┘
```

### 前置条件

| 依赖 | 最低版本 | 用途 |
|---|---|---|
| **Go** | 1.24+ | 编译二进制（无 cgo 依赖） |
| **SQLite** | 内置 | 默认后端（零依赖） |
| **MySQL** | 8.0 / 9.0（可选） | 生产后端；未配置 `-mysql-dsn` 时跳过 |
| **Node.js** | 20+ | 仅当需要构建 `/console/` 前端（VitePress） |
| **codebase-memory-mcp** | 最新 | 后端 MCP 服务，路径通过 `-mcp-bin` 指定 |

---

# Part I: 开发者指南

---

## 1. MCP 客户端配置

本章介绍如何在各主流 AI IDE 中配置 MCP 连接到 cbmem-team 服务。

### 1.1 获取 JWT Token

连接 cbmem-team 需要 **JWT Bearer Token**。获取方式有三种：

**方式一：管理员签发**

管理员通过 Admin API 为你的账号签发 token：

```bash
# 管理员执行（需要 admin-token）
curl -X POST "http://server:8787/admin/users/<your-user-id>/token?ttl=720h" \
  -H "X-Admin-Token: <admin-token>"
```

响应示例：

```json
{
  "user_id": "alice",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires": "2026-08-08T00:00:00Z"
}
```

**方式二：自助续期（无需管理员）**

如果你已持有有效 token，可以自行续期：

```bash
curl -sS -X POST "http://server:8787/refresh?ttl=4320h" \
  -H "Authorization: Bearer $OLD_JWT"
```

> `/refresh` 不需要 admin-token，只验证调用者现有的有效 JWT。

**方式三：离线签发**

使用 `cbmem-mint-token` 工具在不开放管理端口的前提下签发 token：

```bash
cbmem-mint-token \
  --secret "$JWT_SECRET" \
  --user alice \
  --ttl 720h
# 输出: eyJhbGciOiJIUzI1NiIs...
```

| Flag | 默认 | 说明 |
|---|---|---|
| `--secret` | （必填） | 与服务端 `-jwt-secret` 完全一致 |
| `--user` | （必填） | 用户 ID（sub claim） |
| `--ttl` | `720h` | token 有效期 |

### 1.2 Cursor 配置

编辑 `~/.cursor/mcp.json`（Windows: `%USERPROFILE%\.cursor\mcp.json`）：

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

**URL 参数说明**：

| 参数 | 说明 | 示例 |
|------|------|------|
| `as` | 你的用户 ID，决定路由到哪个 per-user 子进程 | `alice` |
| `project` | **服务端**文件系统的项目路径（不是本地路径） | `/var/lib/cbmem-team/users/alice/repos/github.com__org_repo` |

**注意事项**：
- `project` 路径必须是**服务端**路径，不是本地路径。管理员通过 `/admin/repos/clone` 克隆仓库后，返回的 `local_path` 即为正确值
- 不要直接暴露 `:8787`；生产环境前面应挂 nginx/caddy 做 TLS 终止
- 每个开发人员只需要自己的 JWT

### 1.3 Qoder 配置

Qoder 的 MCP 配置方式与 Cursor 类似。在项目根目录或全局配置中添加 MCP 服务器：

**方式一：项目级配置**

在项目根目录创建 `.qoder/mcp.json`：

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

**方式二：全局配置**

编辑 `~/.qoder/mcp.json`，内容同上。

### 1.4 Claude Desktop 配置

编辑 `~/.config/claude_desktop_config.json`（Windows: `%APPDATA%\Claude\claude_desktop_config.json`）：

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

### 1.5 CodeBuddy 配置

CodeBuddy 支持在设置中配置 MCP 服务器。打开 CodeBuddy 设置 → MCP Servers，添加新服务器：

- **名称**：`cbmem-team`
- **URL**：`http://your-server:8787/mcp?as=alice&project=/path/to/project`
- **Headers**：

```json
{
  "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...."
}
```

### 1.6 Token 续期

JWT token 有过期时间（默认签发 30 天）。续期方式：

```bash
# 自助续期（持有有效 JWT 即可，不需要 admin-token）
curl -sS -X POST "http://your-server:8787/refresh?ttl=4320h" \
  -H "Authorization: Bearer $OLD_JWT"
```

响应中返回新的 JWT token，替换配置文件中的旧 token 即可。

### 1.7 自动配置脚本

仓库提供两个脚本，自动完成「签发 JWT → 写入配置文件 → 连通性验证」全流程：

| 脚本 | 平台 |
|------|------|
| `examples/cbmem-emit-mcp.ps1` | Windows / pwsh 7 |
| `examples/cbmem-emit-mcp.sh` | macOS / Linux |

**Linux / macOS**：

```bash
BASE=http://your-server:8787 \
ADMIN_TOKEN=$YOUR_ADMIN_TOKEN \
USER=alice \
PROJECT_SERVER_PATH=/var/lib/cbmem-team/users/alice/repos/github.com__org_repo \
bash examples/cbmem-emit-mcp.sh
```

**Windows**：

```powershell
pwsh examples/cbmem-emit-mcp.ps1 `
  -Base http://your-server:8787 `
  -AdminToken $YOUR_ADMIN_TOKEN `
  -User alice `
  -ProjectServerPath /var/lib/cbmem-team/users/alice/repos/github.com__org_repo `
  -Ttl 4320h
```

脚本自动执行 5 步：
1. 调用 `POST /admin/users/<user>/token?ttl=<ttl>` 签发 JWT
2. 加载 `~/.cursor/mcp.json`（不存在则创建）
3. 更新 `cbmem-team` 条目（upsert）
4. 执行 `tools/list` 冒烟测试
5. 输出一行摘要

### 1.8 端到端连通性验证

配置完成后，执行以下验证：

**Step 1：检查服务端是否可达**

```bash
curl http://your-server:8787/healthz
# 期望: {"status":"ok","users":N}
```

**Step 2：MCP initialize 握手**

```bash
curl -X POST "http://your-server:8787/mcp?as=alice&project=/code/foo" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"curl","version":"0"},"capabilities":{}}}'
```

期望返回 `serverInfo` + `capabilities`。如果返回 401，参考 [FAQ §401](#faq-401)。

---

## 2. 控制台操作指南

cbmem-team 提供 Web 控制台，覆盖用户管理、项目管理、会话记录、会话归纳、会话蒸馏五大模块。

### 2.1 登录控制台

1. 浏览器打开 `http://server:8787/console/`
2. 输入 **admin-token**（与服务端 `-admin-token` 参数一致）
3. 点击「登录」

**认证机制**：
- 登录成功后，服务端写入 session cookie `cbmem_console` + CSRF token `cbmem_csrf`
- 非 GET 请求自动携带 `X-CSRF-Token` header
- Session 默认 TTL 8 小时，过期后需重新登录
- 支持 `X-Admin-Token` 和 `Authorization: Bearer` 两种方式登录

**登出**：点击顶部「退出」按钮，session cookie 自动清除。

### 2.2 用户管理

路径：`/console/#/users`

**创建用户**：

1. 点击「新增用户」
2. 填写字段：
   - **ID**：用户唯一标识（如 `alice`），创建后不可修改
   - **显示名**：展示名称（如 `Alice Wang`）
   - **项目路径白名单**：JSON 数组，如 `["/code/foo", "/code/bar"]`。为空表示不限制
3. 点击「提交」

**管理操作**：

| 操作 | 说明 |
|------|------|
| 编辑 | 更新显示名、项目路径白名单、禁用状态 |
| 删除 | 从列表中移除用户 |
| 签发 Token | 为该用户生成带 TTL 的 JWT（同 1.1 方式一） |
| 撤销 Token | 禁用用户（`disabled=true`），所有 JWT 立即失效 |

**搜索**：输入关键词搜索用户 ID 或显示名。列表支持分页，默认每页 20 条。

### 2.3 项目管理

路径：`/console/#/projects`

**创建项目**：

1. 点击「新增项目」
2. 填写字段：
   - **名称**：项目显示名
   - **路径**：服务端文件系统路径（唯一约束）
   - **Wing**：MemPalace Wing 绑定（如 `engineering`），用于归纳/蒸馏写入
   - **MCP 二进制路径**：`codebase-memory-mcp` 的路径（可选，使用全局默认值可留空）
3. 点击「提交」

**管理操作**：

| 操作 | 说明 |
|------|------|
| 查看索引状态 | 显示「已索引 / 未索引 / 索引中」状态标签 |
| 触发索引 | 异步执行 `codebase-memory-mcp` 重新索引项目 |
| 删除 | 软删除项目（关联的 session 记录保留） |

### 2.4 会话记录查询

路径：`/console/#/sessions`

**会话列表**：

- 按用户筛选：选择目标用户
- 按项目筛选：选择目标项目
- 按时间范围筛选：设置起止日期
- 支持分页

**会话详情**：

点击某条会话「查看」，侧边抽屉展开显示：
- 基本信息：用户 ID、项目路径、开始/结束时间、工具调用次数、对话轮次
- 轮次列表：按 `turn_no` 排序，每条显示 role（user/assistant）+ content

**会话统计**：

调用 `GET /api/console/sessions-stats` 或在控制台 Dashboard 查看：
- 总会话数、总轮次数、工具调用总数
- Top 用户（按 session 数量降序）
- Top 项目（按 session 数量降序）

**会话采集机制**：

cbmem-team 在 MCP handler 外包一层 capture middleware，自动采集：
- 触发条件：JSON-RPC `initialize` / `tools/call` 含 `messages/create`
- 复用策略：同一 `(user_id, project_path)` 30 分钟内有最近 session 则复用
- 异步写入：不阻塞 MCP 请求

### 2.5 会话归纳（Summarize）

路径：`/console/#/summarize`

会话归纳使用 LLM 将一组会话提炼为 5 个维度（5 Hall 模型）。

**操作步骤**：

1. 选择一个或多个会话
2. 设置归纳深度：
   - `shallow`：浅层归纳，快速概览
   - `deep`：深度归纳，详细分析
   - `expert`：专家级归纳，最全面
3. 设置目标 Wing（如 `engineering`）
4. 点击「开始归纳」

**结果展示**：

归纳完成后，结果按 5 个 Hall 分 Tab 展示：

| Hall | 含义 | 典型内容 |
|------|------|---------|
| `hall_facts` | 事实、已锁定的选择 | 技术决策、架构约束 |
| `hall_events` | 事件、里程碑 | 会议记录、调试过程 |
| `hall_discoveries` | 突破、新发现 | 性能优化发现、Bug 根因 |
| `hall_preferences` | 习惯、偏好 | 代码风格、工具偏好 |
| `hall_advice` | 建议和解决方案 | 最佳实践、避坑指南 |

**API 调用**：

```bash
# 触发归纳
curl -X POST http://server:8787/api/console/summarize \
  -H "X-CSRF-Token: <csrf>" -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"source_ids":["sess_xxx","sess_yyy"],"depth":"deep","target_wing":"engineering"}'

# 查询任务状态
curl http://server:8787/api/console/summarize/<task_id> -b cookies.txt
```

### 2.6 会话蒸馏（Distill）

路径：`/console/#/distill`

会话蒸馏从会话中提取可重用的知识片段、决策记录和技术债务标记。

**操作步骤**：

1. 选择一个或多个会话
2. 设置蒸馏规则：
   - `min_value_score`：最小评分阈值（默认 0.7，越高越严格）
   - `dimensions`：提取维度（默认 `["fact", "decision", "discovery"]`）
3. 设置目标 Wing
4. 点击「开始蒸馏」

**结果展示**：

蒸馏结果包含三类提取物：
- **知识片段**（fact）：可重用的技术知识
- **决策记录**（decision）：关键决策及其上下文
- **技术债务**（discovery）：识别的技术债务和改进建议

**Commit 写入 MemPalace**：

蒸馏完成后，点击「写入 MemPalace」将结果推送到 MemPalace 服务：
- 知识片段 → `room=room, hall=facts`
- 决策 → `room=room, hall=advice`

Commit 是**幂等**的——重复点击不会重复写入。

**API 调用**：

```bash
# 触发蒸馏
curl -X POST http://server:8787/api/console/distill \
  -H "X-CSRF-Token: <csrf>" -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"source_ids":["sess_xxx"],"rules":{"min_value_score":0.7,"dimensions":["fact","decision","discovery"]}}'

# 查询任务状态
curl http://server:8787/api/console/distill/<task_id> -b cookies.txt

# 写入 MemPalace
curl -X POST http://server:8787/api/console/distill/<task_id>/commit \
  -H "X-CSRF-Token: <csrf>" -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"target_wing":"engineering"}'
```

---

## 3. MemPalace 集成

MemPalace 是一个本地优先的 AI 记忆系统，采用原文存储，通过语义搜索实现记忆检索。cbmem-team 通过 `-mempalace-base` 参数与其集成，将归纳/蒸馏结果写入 MemPalace。

### 3.1 核心概念

MemPalace 采用宫殿记忆法的结构化存储体系：

```
Palace（宫殿）
  ├── Wing（翅膀）→ 人员或项目
  │     └── Room（房间）→ 主题
  │           └── Drawer（抽屉）→ 原文内容块
  │
  ├── Hall（大厅）→ 分类标签
  │     ├── hall_facts       → 事实、决策
  │     ├── hall_events      → 事件、里程碑
  │     ├── hall_discoveries → 发现、突破
  │     ├── hall_preferences → 偏好、习惯
  │     └── hall_advice      → 建议、方案
  │
  └── Tunnel（隧道）→ 跨 Wing 关联
```

| 概念 | 说明 | 示例 |
|------|------|------|
| **Wing** | 顶层组织单元，代表人员或项目 | `wing_alice`、`wing_engineering` |
| **Room** | Wing 内的主题分类 | `auth-migration`、`ci-pipeline` |
| **Hall** | 记忆的内容类型 | `hall_facts`、`hall_advice` 等 |
| **Drawer** | 存储的原文文本块 | 完整对话、代码片段、决策记录 |

### 3.2 归纳/蒸馏结果写入 MemPalace

cbmem-team 通过以下配置启用 MemPalace 集成：

```bash
cbmem-team \
  -mempalace-base http://mempalace-server:8089 \
  -mempalace-timeout 10s \
  ...
```

**写入流程**：

1. 在控制台完成归纳或蒸馏操作
2. 结果生成后，点击「写入 MemPalace」
3. 指定 `target_wing`（目标 Wing）
4. cbmem-team 调用 `POST {mempalace-base}/api/drawers` 写入
5. 写入映射：知识片段 → `hall=facts`，决策 → `hall=advice`

> MemPalace 内置自动重试机制（失败重试 3 次）。

### 3.3 搜索与检索

写入 MemPalace 后，团队成员可通过 MemPalace 的 MCP 工具搜索记忆：

```bash
# 语义搜索
mempalace search "为什么我们切换到了 GraphQL"

# 加载上下文（新会话启动）
mempalace wake-up
```

> 关于 MemPalace 的完整安装和使用指南，请参考 [MemPalace 官方文档](https://mempalaceofficial.com/)。

---

## 4. codebase-memory-mcp 集成

### 4.1 什么是 codebase-memory-mcp

`codebase-memory-mcp` 是一个**单用户本地** AI 代码记忆工具。它为每个项目构建代码 AST 索引，通过 MCP 协议提供语义搜索能力，让 AI IDE 理解你的代码库。

核心特点：
- 基于 SQLite 的本地索引存储
- MCP 协议标准接口，兼容主流 AI IDE
- 自动分析代码结构、依赖关系
- 支持增量索引

### 4.2 本地模式 vs 团队模式

| 维度 | 本地模式（codebase-memory-mcp 直接使用） | 团队模式（通过 cbmem-team） |
|------|------------------------------------------|---------------------------|
| 部署 | 每个开发者本地运行 | 服务器统一部署 |
| 协议 | stdio（标准输入输出） | HTTP + JWT |
| 用户数 | 单用户 | 多用户 |
| 索引隔离 | 天然隔离（本地文件） | per-user 进程 + SQLite 文件锁 |
| 会话捕获 | 无 | 自动捕获 |
| 知识提炼 | 无 | 归纳/蒸馏 + MemPalace |
| 管理界面 | 无 | Web 控制台 |
| 适用场景 | 个人开发 | 团队协作 |

### 4.3 索引目录结构

cbmem-team 为每个用户维护独立的索引目录：

```
${DataDir}/users/<userID>/projects/<sanitized_project>/
    └── codebase-memory-mcp (独立 SQLite 文件)
```

例如：
```
/var/lib/cbmem-team/users/alice/projects/_code_foo/
/var/lib/cbmem-team/users/bob/projects/_var_lib_cbmem-team_users_bob_repos_github.com__org_repo/
```

`<sanitized_project>` 是将项目路径中的 `/` 替换为 `_` 后的安全目录名。

### 4.4 MCP 协议交互流程

客户端通过 HTTP 与 cbmem-team 交互，cbmem-team 转发到对应 per-user 的 `codebase-memory-mcp` stdio 子进程：

```
1. initialize    → 握手，获取 serverInfo + capabilities
2. tools/list    → 获取可用工具列表
3. tools/call    → 调用具体工具（如语义搜索、索引构建）
4. ping          → 心跳检测
```

请求示例：

```bash
# initialize 握手
POST /mcp?as=alice&project=/code/foo
Authorization: Bearer <jwt>
Content-Type: application/json

{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"cursor","version":"1.0"},"capabilities":{}}}
```

### 4.5 会话自动采集机制

cbmem-team 在 MCP handler 外包一层 **capture middleware**，自动采集所有对话会话：

**触发条件**：
- JSON-RPC `initialize` 请求
- `tools/call` 或 `tools/invoke` 调用
- 包含 `messages` 数组的消息调用

**采集内容**：
- 每次触发 → 创建或复用 `sessions` 行
- 每轮对话 → 写入 `session_turns`（role, content, tools_json, ts）

**复用策略**：同一 `(user_id, project_path)` 30 分钟内有最近 session 则复用，否则创建新 session。

**性能影响**：采集在独立 goroutine 中异步执行，不阻塞 MCP 请求。写入使用 per-user channel 串行化，避免 SQLite 锁竞争。

---

# Part II: 运维指南

---

## 5. 部署与启动

### 5.1 编译构建

```bash
cd tools/cbmem-team

# 编译主服务
go build -o bin/cbmem-team ./cmd/cbmem-team

# 编译离线 JWT 签发工具
go build -o bin/cbmem-mint-token ./cmd/cbmem-mint-token
```

跨平台编译：

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o bin/cbmem-team-linux ./cmd/cbmem-team

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/cbmem-team.exe ./cmd/cbmem-team
```

### 5.2 前台模式（开发 / 调试）

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

启动成功标志：

```
console db: driver=sqlite
console db: migrate console + M1..M4 schemas
seed tool_directory (49 rows)
seed best_practices (3 rows)
seed workflows (3 rows)
listening on :8787
```

#### 必需 Flag

| Flag | 说明 | 示例 |
|---|---|---|
| `-listen` | HTTP 监听地址 | `:8787` |
| `-data` | 数据目录（SQLite 落点） | `/var/lib/cbmem-team` |
| `-mcp-bin` | `codebase-memory-mcp` 二进制绝对路径 | `/usr/local/bin/codebase-memory-mcp` |
| `-jwt-secret` | JWT HS256 签名密钥（**生产必换**） | `openssl rand -hex 32` |
| `-admin-token` | 管理端静态 token（**生产必换**） | `openssl rand -hex 32` |

#### 可选 Flag

| Flag | 默认 | 说明 |
|---|---|---|
| `-config` | `/etc/cbmem-team/config.yaml` | YAML 配置文件路径（CLI flag 优先） |
| `-log` | `info` | 日志级别：`debug` / `info` / `warn` / `error` |
| `-mysql-dsn` | 空（= SQLite） | MySQL DSN；非空时切换后端到 MySQL 8.0 |
| `-mysql-max-open` | `16` | MySQL 连接池上限 |
| `-mysql-max-idle` | `4` | MySQL 空闲连接数 |
| `-mysql-max-lifetime` | `30m` | MySQL 连接最大存活时间 |
| `-llm-provider` | `fake` | `fake` / `openai` / `ollama` |
| `-llm-model` | 空 | 模型名称（如 `gpt-4o-mini`） |
| `-llm-base-url` | 空 | LLM API base URL |
| `-llm-api-key` | 空 | LLM API 密钥（Ollama 可不填） |
| `-llm-timeout` | `30s` | LLM 请求超时 |
| `-mempalace-base` | 空 | MemPalace HTTP base URL，空则禁用 |
| `-mempalace-timeout` | `10s` | MemPalace 请求超时 |
| `-console-dist` | 空 | 控制台前端 VitePress 产物目录 |
| `-idle-ttl` | `30m` | 空闲进程回收时间 |
| `-max-procs-per-user` | `4` | 每个用户的最大进程数 |

### 5.3 systemd 后台部署（推荐生产）

#### 准备环境文件

`/etc/cbmem-team/cbmem-team.env`：

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

权限锁紧：

```bash
sudo install -m 0600 deploy/cbmem-team.env /etc/cbmem-team/cbmem-team.env
```

#### 安装 systemd unit

仓库提供两套 unit：

| Unit | 数据库 | 用途 |
|------|--------|------|
| `deploy/cbmem-team.service` | SQLite | 回退单元 |
| `deploy/cbmem-team-mysql.service` | MySQL | **生产主用** |

```bash
sudo install -m 0644 deploy/cbmem-team-mysql.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now cbmem-team-mysql
sudo systemctl status cbmem-team-mysql --no-pager
```

#### 查看日志

```bash
# 实时跟踪
sudo journalctl -u cbmem-team-mysql -f

# 最近 100 行
sudo journalctl -u cbmem-team-mysql -n 100 --no-pager
```

### 5.4 MySQL 部署

#### 步骤 1：启动 MySQL

```bash
cd tools/cbmem-team/deploy
docker compose -f docker-compose.mysql.yml up -d
```

#### 步骤 2：配置 DSN

```bash
sudo cp deploy/mysql.env.example /etc/cbmem-team/mysql.env
sudo chmod 0600 /etc/cbmem-team/mysql.env
# 编辑 /etc/cbmem-team/mysql.env，设置强随机密码
```

#### 步骤 3：建表

```bash
cbmem-team migrate-tables -mysql-dsn "$(grep CBMEM_MYSQL_DSN /etc/cbmem-team/mysql.env | cut -d= -f2)"
```

> `migrate-tables` 是幂等的，可以安全重复执行。

#### 步骤 4：首次 ETL（从 SQLite 迁移）

```bash
./examples/mysql-etl.sh --mysql-dsn "$(grep CBMEM_MYSQL_DSN /etc/cbmem-team/mysql.env | cut -d= -f2)"
```

#### 步骤 5：切换到 MySQL

```bash
sudo systemctl disable cbmem-team
sudo systemctl enable --now cbmem-team-mysql
```

#### 步骤 6：安装备份 cron

```bash
sudo cp examples/mysql-backup.sh /etc/cron.daily/cbmem-mysql-backup
```

### 5.5 SQLite ↔ MySQL 切换

两套 unit 共用 `/var/lib/cbmem-team` 数据目录；任一时刻只能跑一个：

```bash
# 切到 SQLite（紧急回退）
sudo systemctl stop cbmem-team-mysql
sudo systemctl start cbmem-team       # 旧单元无 -mysql-dsn flag

# 切回 MySQL
sudo systemctl stop cbmem-team
sudo systemctl start cbmem-team-mysql
```

> **注意**：MySQL 写入不会回写 SQLite。切回 SQLite 后看到的是切换前的快照。

#### 子命令

| 子命令 | 用途 |
|---|---|
| `cbmem-team migrate-tables -mysql-dsn <dsn>` | MySQL 建表（幂等） |
| `cbmem-team migrate-sqlite-to-mysql -sqlite <path> -mysql-dsn <dsn>` | SQLite → MySQL ETL |
| `cbmem-team mysql-ping -mysql-dsn <dsn>` | 检查 MySQL 连接 |

---

## 6. 用户与项目管理

### 6.1 创建用户 + 签发 Token

```bash
ADMIN="your-admin-token"

# 创建用户 alice
curl -X POST http://server:8787/admin/users \
  -H "X-Admin-Token: $ADMIN" \
  -H "Content-Type: application/json" \
  -d '{"id":"alice","display_name":"Alice","project_paths":["/code/foo"]}'

# 签发 30 天 token
curl -X POST "http://server:8787/admin/users/alice/token?ttl=720h" \
  -H "X-Admin-Token: $ADMIN"
```

**用户数据结构**：

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

**管理端点**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/admin/users` | GET | 列出所有用户 |
| `/admin/users` | POST | 创建用户 |
| `/admin/users/:id` | PATCH | 部分更新（display_name / disabled / project_paths） |
| `/admin/users/:id` | DELETE | 删除用户 + 清除进程池 |
| `/admin/users/:id/token` | POST | 签发 JWT（`?ttl=720h`） |
| `/admin/stats` | GET | 全局统计 |

### 6.2 管理项目 + 触发索引

```bash
# 创建项目
curl -X POST http://server:8787/api/console/projects \
  -H "X-Admin-Token: $ADMIN" \
  -H "Content-Type: application/json" \
  -d '{"name":"My Project","path":"/code/foo","wing":"engineering"}'

# 触发重新索引
curl -X POST http://server:8787/api/console/projects/<project_id>/reindex \
  -H "X-Admin-Token: $ADMIN"
```

### 6.3 离线签发 JWT

使用 `cbmem-mint-token` 工具，无需暴露管理端口：

```bash
cbmem-mint-token \
  --secret "$JWT_SECRET" \
  --user alice \
  --ttl 720h
# 输出: eyJhbGciOiJIUzI1NiIs...
```

### 6.4 克隆远程仓库

管理员可远程为用户克隆 Git 仓库：

```bash
curl -X POST http://server:8787/admin/repos/clone \
  -H "X-Admin-Token: $ADMIN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","url":"https://github.com/org/repo.git"}'
```

克隆完成后：
- 仓库存储在 `${DataDir}/users/<userID>/repos/<safeName>/`
- 仓库路径自动添加到用户的 `project_paths` 白名单

---

## 7. 高级功能

### 7.1 工具目录 + 调用日志（M1）

内置 49 个工具目录条目，自动记录所有 MCP 工具调用。

| 端点 | 说明 |
|------|------|
| `GET /api/console/v2/tools` | 工具列表 |
| `GET /api/console/v2/dashboard/summary` | 调用统计 |
| `GET /api/console/v2/dashboard/recent-invocations` | 最近调用 |

### 7.2 工具限流 + Best Practices CRUD（M2）

**限流**：

- 限流维度：`(user_id, tool_id)` 桶
- 超阈值返回 HTTP 429 + `retry_after`
- 调整限流：`PATCH /api/console/v2/rate-limit`

**Best Practices**：

| 端点 | 说明 |
|------|------|
| `GET /api/console/v2/bps` | BP 列表 |
| `POST /api/console/v2/bps` | 创建 BP |
| `PUT /api/console/v2/bps/:id` | 更新 BP（自动保存版本快照） |
| `DELETE /api/console/v2/bps/:id` | 删除 BP |
| `GET /api/console/v2/bps/:id/graph` | BP 关联图 |

### 7.3 工作流引擎（M3）

工作流是 console 端的「可发布 + 可执行」图节点序列。

**内置工作流**（随 migration 自动 seed）：

| ID | 名称 | 节点数 | 用途 |
|----|------|--------|------|
| `wf-commit-precheck` | Commit 前置检查 | 7 | 高风险提交前自动合规校验 |
| `wf-adr-doublewrite` | ADR 双写 | 6 | ADR 同时写到 MemPalace + 文件 |
| `wf-repo-daily-sync` | 仓库每日同步 | 4 | 每日 cron 拉取 repo pipeline 增量 |

**状态流转**：`draft` → `published`（只读图）→ `archived`

| 端点 | 说明 |
|------|------|
| `GET /api/console/v2/workflows` | 列表 |
| `POST /api/console/v2/workflows` | 新建 |
| `POST /api/console/v2/workflows/:id/publish` | 发布 |
| `POST /api/console/v2/workflows/:id/run` | 同步执行 |

### 7.4 Repo Pipeline（M4）

Repo pipeline 把外部内容通过 6 阶段 runtime 落地为 `bp_candidates`：

```
crawl → parse → grade → sink → rehearse
```

**BP Candidate 三态**：`draft` → `accepted` / `rejected`；`accepted` → `merged`

| 端点 | 说明 |
|------|------|
| `POST /api/console/v2/repos` | 新建 pipeline |
| `POST /api/console/v2/repos/:id/activate` | 激活 |
| `POST /api/console/v2/repos/:id/run` | 执行一次 |
| `GET /api/console/v2/repos/candidates` | 候选 BP 列表 |
| `POST /api/console/v2/repos/candidates/:id/accept` | 接受候选 |
| `POST /api/console/v2/repos/candidates/:id/merge` | 合并到 best_practices |

---

## 8. 命令行工具参考

### 8.1 生产级工具

| 工具 | 编译命令 | 用途 |
|---|---|---|
| `cbmem-team` | `go build -o bin/cbmem-team ./cmd/cbmem-team` | 主服务 |
| `cbmem-mint-token` | `go build -o bin/cbmem-mint-token ./cmd/cbmem-mint-token` | 离线签 JWT |

### 8.2 运维 / 迁移工具

| 工具 | 编译命令 | 用途 |
|---|---|---|
| `create-db` | `go build -o bin/create-db ./cmd/create-db` | 建 `cbmem` MySQL 库 |
| `mysql-probe` | `go build -o bin/mysql-probe ./cmd/mysql-probe` | MySQL 连接探测 |
| `show-tables` | `go build -o bin/show-tables ./cmd/show-tables` | 列出所有表 |
| `show-indexes` | `go build -o bin/show-indexes ./cmd/show-indexes` | 列出所有索引 |

### 8.3 端到端测试工具（仅 dev / CI）

| 工具 | 覆盖里程碑 | 运行命令 |
|---|---|---|
| `e2e-m1` | M1 工具目录 + 调用日志 | `go run ./cmd/e2e-m1` |
| `e2e-v5` | M0.5 SQLite/MySQL 双轨 + ETL | `go run ./cmd/e2e-v5` |
| `e2e-v6` | M2 限流 + BP CRUD | `go run ./cmd/e2e-v6` |
| `e2e-v7` | M3 工作流 + M4 Pipeline | `go run ./cmd/e2e-v7` |

通用 flag：`-mysql-dsn "..."` 或通过环境变量 `$CBMEM_MYSQL_DSN`。

### 8.4 工具速查表

| 工具 | 主要用途 | 生产 |
|---|---|---|
| `cbmem-team` | 主服务（HTTP wrapper + 控制台） | ✅ |
| `cbmem-mint-token` | 离线签发 JWT | ✅ |
| `create-db` | 建 MySQL 库 | ✅ |
| `mysql-probe` | 探活 MySQL | ✅ |
| `show-tables` | 列所有表 | ✅ |
| `show-indexes` | 列所有索引 | ✅ |
| `seed-sqlite` | 写 SQLite 测试数据 | ❌ |
| `e2e-m1` ~ `e2e-v7` | 端到端验收 | ❌ |
| `mcp-stub` | e2e 用 MCP 桩 | ❌ |
| `mint-token-debug` | 调试 401 | ❌ |

---

# Part III: 参考

---

## 9. 配置参数全表

### 9.1 必需参数

| 参数 | 描述 | 示例 |
|------|------|------|
| `-listen` | HTTP 监听地址 | `:8787` |
| `-data` | 数据目录（含 SQLite） | `/var/lib/cbmem-team` |
| `-mcp-bin` | `codebase-memory-mcp` 二进制路径 | `/usr/local/bin/codebase-memory-mcp` |
| `-jwt-secret` | JWT 签名密钥 | `openssl rand -hex 32` |
| `-admin-token` | 控制台管理员静态 token | `openssl rand -hex 32` |
| `-console-dist` | VitePress 构建产物目录 | `/opt/cbmem-console/dist` |

### 9.2 LLM 参数

| 参数 | 描述 | 默认 |
|------|------|------|
| `-llm-provider` | `openai` / `ollama` / `fake` | `fake` |
| `-llm-model` | 模型名称 | `gpt-4o-mini` |
| `-llm-base-url` | API base URL | `https://api.openai.com/v1` |
| `-llm-api-key` | API 密钥 | `sk-...` |
| `-llm-timeout` | 超时时长 | `30s` |

### 9.3 MemPalace 参数

| 参数 | 描述 | 默认 |
|------|------|------|
| `-mempalace-base` | MemPalace HTTP 地址 | `http://localhost:8089` |
| `-mempalace-timeout` | 超时时长 | `10s` |

### 9.4 可选参数

| 参数 | 描述 | 默认 |
|------|------|------|
| `-config` | 配置文件路径 | `/etc/cbmem-team/config.yaml` |
| `-log` | 日志级别 | `info` |
| `-idle-ttl` | 空闲进程回收时间 | `30m` |
| `-max-procs-per-user` | 每用户最大进程数 | `4` |
| `-mysql-dsn` | MySQL DSN（非空则启用 MySQL） | 空 |
| `-mysql-max-open` | MySQL 连接池上限 | `16` |
| `-mysql-max-idle` | MySQL 空闲连接数 | `4` |
| `-mysql-max-lifetime` | MySQL 连接最大存活时间 | `30m` |

---

## 10. 数据库表结构

### 10.1 SQLite 与 MySQL 双轨

cbmem-team 支持两种后端，通过启动 flag 切换：

| 后端 | 启用方式 | 何时用 |
|------|---------|--------|
| **SQLite**（默认） | 不设 `-mysql-dsn` flag | 单机 / 开发 / 小团队 |
| **MySQL 8.0** | `-mysql-dsn <dsn>` | 生产 / 多实例 / 高并发 |

### 10.2 核心表

| 表名 | 用途 |
|------|------|
| `users` | 用户信息（含 project_paths 白名单） |
| `projects` | 项目配置（含 wing / mcp_bin） |
| `sessions` | 会话记录 |
| `session_turns` | 会话轮次 |

### 10.3 任务表

| 表名 | 用途 |
|------|------|
| `summarize_tasks` | 归纳任务 |
| `distill_tasks` | 蒸馏任务 |

### 10.4 控制台表

| 表名 | 用途 |
|------|------|
| `console_sessions` | 控制台登录 session（含 TTL） |

### 10.5 M1-M4 扩展表

| 表名 | 里程碑 | 用途 |
|------|--------|------|
| `tool_directory` | M1 | 工具目录（49 个内置条目） |
| `tool_invocation_logs` | M1 | 工具调用日志 |
| `best_practices` + `bp_versions` | M2 | 最佳实践 + 版本快照 |
| `workflows` + `workflow_versions` + `workflow_runs` | M3 | 工作流引擎 |
| `tickets` | M3 | 工单 |
| `repo_pipelines` + `repo_pipeline_runs` + `bp_candidates` | M4 | Repo Pipeline |

---

## 11. API 速查表

### 11.1 管理端点（/admin/*）

| Path | Method | 说明 |
|------|--------|------|
| `/admin/users` | GET | 列表 |
| `/admin/users` | POST | 新增 |
| `/admin/users/:id` | PATCH | 更新 |
| `/admin/users/:id` | DELETE | 删除 |
| `/admin/users/:id/token` | POST | 签发 token |
| `/admin/stats` | GET | 全局统计 |
| `/admin/repos/clone` | POST | 克隆仓库 |
| `/admin/repos` | GET | 列出仓库 |

### 11.2 控制台端点（/api/console/*）

| Path | Method | 说明 |
|------|--------|------|
| `/api/console/login` | POST | admin-token 登录 |
| `/api/console/logout` | POST | 登出 |
| `/api/console/me` | GET | 当前管理员 |
| `/api/console/stats` | GET | 全局统计 |
| `/api/console/users` | GET | 用户列表 |
| `/api/console/users` | POST | 新增用户 |
| `/api/console/users/:id` | PUT | 更新用户 |
| `/api/console/users/:id` | DELETE | 删除用户 |
| `/api/console/users/:id/token` | POST | 签发 token |
| `/api/console/users/:id/revoke` | POST | 撤销全部 token |
| `/api/console/projects` | GET | 项目列表 |
| `/api/console/projects` | POST | 新增项目 |
| `/api/console/projects/:id` | DELETE | 删除项目 |
| `/api/console/projects/:id/reindex` | POST | 触发索引 |
| `/api/console/sessions` | GET | 会话列表 |
| `/api/console/sessions/:id` | GET | 会话详情 |
| `/api/console/sessions-stats` | GET | 统计聚合 |
| `/api/console/summarize` | POST | 触发归纳 |
| `/api/console/summarize/:task_id` | GET | 任务状态 |
| `/api/console/distill` | POST | 触发蒸馏 |
| `/api/console/distill/:task_id` | GET | 任务状态 |
| `/api/console/distill/:task_id/commit` | POST | 写入 MemPalace |

### 11.3 MCP 端点

| Path | Method | 说明 |
|------|--------|------|
| `/mcp` | POST | Streamable HTTP（v2 主用） |
| `/mcp/` | POST | 同上，兼容尾部 `/` |
| `/mcp/sse` | GET | SSE（v1 回退） |
| `/healthz` | GET | 健康检查 |
| `/refresh` | POST | JWT 自动续期 |

### 11.4 统一响应格式 + 错误码

```json
{ "code": 0, "msg": "", "data": {} }
```

| 区间 | 含义 |
|------|------|
| `0` | 成功 |
| `40100xx` | 鉴权失败（admin-token 错 / session 过期 / CSRF） |
| `40300xx` | 无权（白名单违规 / 用户 disabled） |
| `40900xx` | 资源冲突（如 user id 重复） |
| `50000xx` | 系统错误（LLM 超时 / SQLite 写入失败） |

---

## 12. FAQ

<a id="faq-401"></a>
### Q1：`/mcp` 报 401 Unauthorized

**排查路径**：

1. **token 是否过期**：`POST /admin/users/<id>/token?ttl=720h` 重发
2. **JWT secret 一致**：server 端 `-jwt-secret` 和签发 token 时用的 secret 必须**字符级一致**
3. **JWT subject 是否是注册用户**：sub claim 必须在 `users` 表存在
4. **admin token 不是这里用的**：`/mcp` 用的是用户 JWT，不是 `-admin-token`

### Q2：Cursor 客户端连不上

**排查路径**：

1. **URL 端口对吗**：默认 `:8787`，用 nginx 反代要换 443/80
2. **JWT 过期**：重新签发
3. **服务端是否在跑**：`curl http://server:8787/healthz` 验证
4. **URL 参数顺序**：`?as=alice&project=/code/foo`，不能颠倒
5. **TLS**：Cursor 走 http 可以，但生产必须 https

### Q3：编译报 undefined

**原因**：GoLand 的 Run Configuration 用了单文件路径。

**修法**：把 Run Configuration 的 "Run kind" 改为 **Package**，路径填 `./cmd/cbmem-team`。

### Q4：SQLite 数据库锁死

**原因**：SQLite 单写者；并发 8 线程以上会触发 `database is locked`。

**修法**：切到 MySQL 后端（`-mysql-dsn`）。

### Q5：MySQL 后端切回 SQLite 丢数据

**原因**：MySQL 写入不会回写 SQLite。

**修法**：保持 MySQL 单轨，或做反向 ETL。

### Q6：systemd ProtectHome 启动失败

**原因**：`codebase-memory-mcp` 在 `$HOME` 下时 systemd 拒绝访问。

**修法**：把二进制移到 `/usr/local/bin/`，或 unit 加 `ProtectHome=read-only`。

### Q7：PowerShell curl -d 失败

**原因**：PowerShell 的 `curl` 是 `Invoke-WebRequest`，会 mangling JSON body。

**修法**：用 WSL / Git Bash，或写 body 到文件后用 `-InFile`。

### Q8：磁盘增长异常

**原因**：`tool_invocation_logs` 是写入最大的表。

**修法**：定期归档近 30 天在线数据，监控 `du -sh /var/lib/cbmem-team/`。

---

## 附录

### A. 环境变量与配置文件

#### systemd EnvironmentFile（`/etc/cbmem-team/cbmem-team.env`）

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

#### dev 工具的 MySQL DSN 解析

所有 `cmd/*` 工具统一通过 `internal/devconf.ResolveMySQLDSN` 解析，优先级：

1. `-mysql-dsn "..."` flag
2. `$CBMEM_MYSQL_DSN` 环境变量
3. `$DSN` 环境变量（旧名，向后兼容）
4. `dev default`（仅开发用，**不是生产凭证**）

### B. 相关文档

- [`README.md`](./README.md) — 架构图 + 客户端配置 + 安全清单
- [`CONSOLE.md`](./CONSOLE.md) — 控制台 5 大模块 + M3/M4 API 速查
- [`BUILD.md`](./BUILD.md) — 构建 + 跨平台编译 + 部署
- [`deploy/sql/README.md`](./deploy/sql/README.md) — SQL 文件使用方式
- [`deploy/dev.env.example`](./deploy/dev.env.example) — dev DSN env 模板
- [`cbmem-team-prd.md`](./cbmem-team-prd.md) — 完整产品需求

### C. 版本与变更

- **v0.x**：单用户 stdio（`codebase-memory-mcp` 原生）
- **v1.0**：HTTP wrapper + JWT 鉴权 + per-user stdio pool
- **v1.5 (M0.5)**：SQLite ↔ MySQL 双轨 + 7 天回退窗口
- **v2.0 (M1)**：工具目录 + 调用日志 + 控制台 v1
- **v2.1 (M2)**：限流 + BP CRUD + 版本快照
- **v2.2 (M3)**：工作流引擎 + 3 个内置 workflow
- **v2.3 (M4)**：repo pipeline + bp_candidate review + 2 个 MySQL 视图
- **v3.0**：用户手册重写（角色导向）

---

*文档最后更新：2026-07-08*

# cbmem-team 产品需求文档 (PRD)

> **文档版本**: v1.0  
> **创建日期**: 2026-07-10  
> **产品名称**: cbmem-team  
> **产品定位**: HTTP 多用户包装器，围绕 codebase-memory-mcp 实现团队共享实例，同时保持每个用户的代码 AST 索引严格隔离

---

## 目录

1. [产品概述](#1-产品概述)
2. [功能列表](#2-功能列表)
3. [技术架构](#3-技术架构)
4. [用户角色与权限](#4-用户角色与权限)
5. [API 接口说明](#5-api-接口说明)
6. [配置参数说明](#6-配置参数说明)
7. [数据库设计](#7-数据库设计)
8. [安全机制](#8-安全机制)
9. [部署指南](#9-部署指南)
10. [附录](#10-附录)

---

## 1. 产品概述

### 1.1 产品背景

`codebase-memory-mcp` 设计为单开发者、本地机器工具：每个开发者运行自己的二进制文件处理自己的代码副本。这种设计有利于隐私和零配置，但意味着团队无法共享中央实例。

### 1.2 产品目标

`cbmem-team` 是解决此问题的最小可能粘合代码：
- 让多个开发者共享单一远程实例
- 保持每个用户的代码 AST 索引严格隔离
- 提供集中式会话管理和知识提炼能力

### 1.3 核心价值

| 特性 | 描述 |
|------|------|
| **用户隔离** | 每个用户拥有独立的 stdio 子进程，SQLite 文件锁提供自然隔离 |
| **团队共享** | 团队成员共享中央 MCP 实例，降低资源消耗 |
| **会话捕获** | 自动采集开发者的所有对话会话 |
| **知识提炼** | LLM 驱动的会话归纳和蒸馏能力 |
| **集中管理** | Web 控制台提供用户、项目、会话的统一管理界面 |

---

## 2. 功能列表

### 2.1 核心功能总览

```
┌─────────────────────────────────────────────────────────────────┐
│                        cbmem-team                               │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │ MCP 包装器   │  │ 用户管理    │  │ 项目管理                 │ │
│  │ (HTTP Wrapper)│  │ (User Mgmt) │  │ (Project Management)    │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │ 会话捕获    │  │ 控制台 UI   │  │ LLM 集成                 │ │
│  │ (Capture)   │  │ (Console)   │  │ (Summarize/Distill)      │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │ MemPalace  │  │ 仓库管理    │  │ 进程池管理               │ │
│  │ 集成       │  │ (Repos)     │  │ (Process Pool)          │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 功能模块详细说明

#### 2.2.1 MCP 协议包装器 (MCP Wrapper)

| 功能 | 描述 | 实现方式 |
|------|------|----------|
| **HTTP 到 stdio 桥接** | 将 HTTP POST 请求转发到 per-user stdio 子进程 | `internal/mcp/mcp.go` |
| **用户路由** | 根据 JWT subject 路由到对应用户的子进程 | Handler 中的 `user_id` 上下文 |
| **白名单验证** | 验证项目路径是否在用户允许列表中 | `anyMatch()` 函数 |
| **JSON-RPC 透传** | 不解析 JSON，直接转发字节流 | 保持对上游 Schema 变化的兼容性 |

**请求流程**:
```
HTTP 客户端 (Cursor/Claude/Qoder)
    │
    ▼ POST /mcp?project=/code/foo
    │ Authorization: Bearer <jwt>
    ▼
cbmem-team (HTTP)
    │
    ▼ route by user_id
per-user stdio pool
    │
    ▼
codebase-memory-mcp <index dir>
```

#### 2.2.2 用户管理 (User Management)

| 功能 | 描述 | API 端点 |
|------|------|----------|
| **用户注册** | 创建新用户，设置 ID、显示名、项目路径白名单 | `POST /admin/users`, `POST /api/console/users` |
| **用户查询** | 列出所有用户，支持关键字搜索和分页 | `GET /admin/users`, `GET /api/console/users` |
| **用户更新** | 部分更新用户信息（显示名、项目路径、禁用状态） | `PATCH /admin/users/:id`, `PUT /api/console/users/:id` |
| **用户删除** | 从注册表中移除用户 | `DELETE /admin/users/:id`, `DELETE /api/console/users/:id` |
| **用户禁用** | 撤销用户访问权限（不删除数据） | `POST /api/console/users/:id/revoke` |
| **JWT 签发** | 为用户生成带 TTL 的 JWT token | `POST /admin/users/:id/token` |
| **JWT 刷新** | 用户自己刷新即将过期的 token | `POST /refresh` |

**用户数据结构**:
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

#### 2.2.3 项目管理 (Project Management)

| 功能 | 描述 | API 端点 |
|------|------|----------|
| **项目创建** | 注册新项目，设置名称、路径、Wing 绑定、MCP 二进制路径 | `POST /api/console/projects` |
| **项目查询** | 列出所有项目，支持搜索和分页 | `GET /api/console/projects` |
| **项目更新** | 更新项目配置（Wing、MCP 二进制等） | `PUT /api/console/projects/:id` |
| **项目删除** | 软删除项目（标记 deleted=1） | `DELETE /api/console/projects/:id` |
| **索引状态检查** | 检查项目是否有代码索引 | `GET /api/console/projects/:id/index-status` |
| **触发重新索引** | 执行 mcp_bin 重新索引项目 | `POST /api/console/projects/:id/reindex` |

**项目数据结构**:
```json
{
  "id": "proj_a1b2c3",
  "name": "My Project",
  "path": "/code/foo",
  "wing": "engineering",
  "mcp_bin": "/usr/local/bin/codebase-memory-mcp",
  "user_id": "alice",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "deleted": false
}
```

#### 2.2.4 会话捕获与管理 (Session Capture & Management)

| 功能 | 描述 | 实现方式 |
|------|------|----------|
| **自动捕获** | 拦截 MCP 请求，自动创建/复用会话记录 | `internal/console/capture.go` 中间件 |
| **会话复用策略** | 同一 (user_id, project_path) 30 分钟内有最近会话则复用 | 滚动会话模型 |
| **轮次记录** | 记录每轮对话的角色、内容、时间戳 | `session_turns` 表 |
| **会话列表** | 按用户、项目、时间范围筛选会话 | `GET /api/console/sessions` |
| **会话详情** | 查看会话及其所有轮次 | `GET /api/console/sessions/:id` |
| **会话统计** | 聚合统计（总数、轮次、工具调用、Top 用户/项目） | `GET /api/console/sessions-stats` |

**捕获触发条件**:
- JSON-RPC `initialize` 请求
- `tools/call` 或 `tools/invoke` 调用
- 包含 `messages` 数组的消息调用

**会话数据结构**:
```json
{
  "id": "sess_alice_abc123",
  "user_id": "alice",
  "project_id": "proj_xxx",
  "project_path": "/code/foo",
  "started_at": "2026-01-01T10:00:00Z",
  "ended_at": "2026-01-01T10:30:00Z",
  "tool_count": 15,
  "turn_count": 8
}
```

#### 2.2.5 控制台管理界面 (Console UI)

| 模块 | 路径 | 描述 |
|------|------|------|
| **用户管理** | `/console/#/users` | 用户 CRUD + Token 签发/撤销 |
| **项目管理** | `/console/#/projects` | 项目 CRUD + Wing 绑定 + 触发索引 |
| **会话记录** | `/console/#/sessions` | 会话列表 + 详情 + 统计 |
| **会话归纳** | `/console/#/summarize` | 选会话 → 5 Hall 结果 → 写入 MemPalace |
| **会话蒸馏** | `/console/#/distill` | 选会话 → 知识片段/决策/技术债务 → 写入 MemPalace |

**前端构建**:
- 使用 VitePress 构建
- 产物部署到 `-console-dist` 指定目录
- 静态文件服务：`GET /console/*`

#### 2.2.6 LLM 集成 (LLM Integration)

##### 2.2.6.1 会话归纳 (Summarization)

| 功能 | 描述 |
|------|------|
| **5 Hall 模型** | 将会话归纳为 5 个维度：Facts、Events、Discoveries、Preferences、Advice |
| **多深度支持** | 支持 shallow、deep、expert 三种归纳深度 |
| **异步任务** | 归纳任务异步执行，支持状态查询 |

**Halls 定义** (`halls.json`):
```json
{
  "hall_facts": "facts",
  "hall_events": "events",
  "hall_discoveries": "discoveries",
  "hall_preferences": "preferences",
  "hall_advice": "advice"
}
```

**API**:
- `POST /api/console/summarize` - 触发归纳任务
- `GET /api/console/summarize/:task_id` - 查询任务状态/结果

##### 2.2.6.2 会话蒸馏 (Distillation)

| 功能 | 描述 |
|------|------|
| **知识片段提取** | 从会话中提取可重用的知识片段 |
| **决策记录** | 记录关键决策及其上下文 |
| **技术债务标记** | 识别并标记技术债务 |
| **可配置维度** | 支持自定义提取维度（默认：fact、decision、discovery） |
| **评分阈值** | 可配置最小评分阈值（默认 0.7） |

**API**:
- `POST /api/console/distill` - 触发蒸馏任务
- `GET /api/console/distill/:task_id` - 查询任务状态
- `POST /api/console/distill/:task_id/commit` - 提交到 MemPalace

#### 2.2.7 MemPalace 集成

| 功能 | 描述 | 实现 |
|------|------|------|
| **Drawer 写入** | 将归纳/蒸馏结果写入 MemPalace | `POST /api/drawers` |
| **Wing 绑定** | 指定目标 Wing（组织单元） | 请求参数 `target_wing` |
| **自动重试** | 失败时自动重试 3 次 | `internal/llm/mempalace.go` |
| **健康检查** | 查询 MemPalace 服务状态 | `GET /healthz` |

**写入映射**:
- 知识片段 → `room=room, hall=facts`
- 决策 → `room=room, hall=advice`

#### 2.2.8 进程池管理 (Process Pool)

| 功能 | 描述 |
|------|------|
| **Per-user 子进程** | 每个用户一个独立的 codebase-memory-mcp 进程 |
| **惰性启动** | 首次请求时启动进程 |
| **空闲回收** | 超过 IdleTTL（默认 30 分钟）后自动终止 |
| **崩溃恢复** | 子进程崩溃后，下次请求时自动重启 |
| **进程统计** | 提供活跃进程数、用户、索引目录等统计信息 |

**进程隔离**:
```
${DataDir}/users/<userID>/projects/<sanitized_project>/
    └── codebase-memory-mcp (独立 SQLite 文件)
```

#### 2.2.9 仓库管理 (Repository Management)

| 功能 | 描述 |
|------|------|
| **Git Clone** | 管理员为用户克隆远程仓库到服务器 |
| **Git Fetch** | 已存在的仓库执行 fetch 保持更新 |
| **路径自动扩展** | 克隆的仓库自动添加到用户的项目路径白名单 |
| **多用户支持** | 不同用户克隆同一 URL 不冲突（基于 URL 生成安全目录名） |

**目录结构**:
```
${DataDir}/users/<userID>/repos/<safeName>/
    ├── .git/
    └── .cbmem-repo-url  (元数据文件)
```

#### 2.2.10 管理员功能 (Admin Endpoints)

| 端点 | 描述 |
|------|------|
| `POST /admin/users` | 创建用户 |
| `GET /admin/users` | 列出用户 |
| `PATCH /admin/users/:id` | 更新用户 |
| `DELETE /admin/users/:id` | 删除用户 |
| `POST /admin/users/:id/token` | 签发 token |
| `GET /admin/stats` | 全局统计 |
| `POST /admin/repos/clone` | 克隆仓库 |
| `GET /admin/repos` | 列出仓库 |

---

## 3. 技术架构

### 3.1 系统架构图

```
                           ┌─────────────────────────────────────┐
                           │            客户端层                 │
                           │  Cursor / Claude / Qoder            │
                           └──────────────┬──────────────────────┘
                                          │ HTTP + JWT Bearer
                           ┌──────────────▼──────────────────────┐
                           │          cbmem-team                 │
                           │  ┌───────────────────────────────┐  │
                           │  │     Gin HTTP Server           │  │
                           │  │  ┌─────────────────────────┐  │  │
                           │  │  │ JWT 验证中间件           │  │  │
                           │  │  │ 会话捕获中间件           │  │  │
                           │  │  │ 管理认证中间件           │  │  │
                           │  │  └─────────────────────────┘  │  │
                           │  └───────────────┬───────────────┘  │
                           │                  │                  │
                           │  ┌───────────────┴───────────────┐  │
                           │  │       路由层                  │  │
                           │  │  /mcp  │ /admin │ /api/console │
                           │  └───────────┬───────────────────┘  │
                           └──────────────┼──────────────────────┘
                                          │
              ┌───────────────────────────┼───────────────────────────┐
              │                           │                           │
              ▼                           ▼                           ▼
    ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
    │  Process Pool   │         │   Console DB    │         │  LLM Provider    │
    │ (Per-user MCP)  │         │   (SQLite)      │         │ (OpenAI/Ollama) │
    └────────┬────────┘         └────────┬────────┘         └────────┬────────┘
             │                           │                           │
             │                           │                           │
             ▼                           ▼                           ▼
    ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
    │ code-memory-mcp │         │  Sessions       │         │    MemPalace     │
    │ (User A)        │         │  Projects       │         │   (可选)         │
    └─────────────────┘         │  Users         │         └─────────────────┘
             │                   │  Tasks         │
             ▼                   └─────────────────┘
    ┌─────────────────┐
    │ code-memory-mcp │
    │ (User B)        │
    └─────────────────┘
```

### 3.2 核心技术栈

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| **HTTP 框架** | Gin | 高性能 Go HTTP 框架 |
| **数据库** | SQLite (modernc.org/sqlite) | WAL 模式，支持并发读 |
| **进程管理** | exec.Cmd | 标准库 os/exec |
| **JWT** | HS256 | 自实现，无外部依赖 |
| **LLM 集成** | OpenAI / Ollama | 可插拔 Provider 接口 |
| **前端** | VitePress | 静态站点生成器 |

### 3.3 目录结构

```
tools/cbmem-team/
├── cmd/
│   ├── cbmem-team/
│   │   ├── main.go        # 主程序入口
│   │   └── handlers.go    # Admin 端点处理器
│   └── cbmem-mint-token/  # Token 签发工具
├── internal/
│   ├── auth/             # JWT 和静态 token 认证
│   ├── console/           # 控制台 API 和 Web UI
│   │   ├── auth.go       # 会话管理 + CSRF
│   │   ├── capture.go    # 会话捕获中间件
│   │   ├── db.go         # 数据库 Schema
│   │   ├── distill_handler.go
│   │   ├── dto.go        # DTO 定义
│   │   ├── projects_handler.go
│   │   ├── router.go     # 控制台路由
│   │   ├── sessions_handler.go
│   │   ├── summarize_handler.go
│   │   └── users_handler.go
│   ├── llm/              # LLM 集成
│   │   ├── provider.go   # Provider 接口
│   │   ├── openai.go    # OpenAI 实现
│   │   ├── ollama.go    # Ollama 实现
│   │   ├── fake.go      # 测试用假实现
│   │   ├── mempalace.go # MemPalace 客户端
│   │   └── prompts/     # 提示词模板
│   ├── mcp/              # MCP 协议处理
│   │   └── mcp.go       # HTTP 到 stdio 桥接
│   ├── pool/             # 进程池管理
│   │   └── pool.go      # Per-user MCP 进程
│   ├── repos/            # Git 仓库管理
│   │   └── repos.go     # Clone/Fetch
│   └── store/            # 用户注册表
│       └── store.go     # JSON 文件存储
├── deploy/               # 部署文件
│   ├── cbmem-team.service
│   └── cbmem-team.env
├── examples/             # 示例脚本
└── dist/                 # 构建产物
```

---

## 4. 用户角色与权限

### 4.1 角色定义

| 角色 | 标识 | 描述 | 权限范围 |
|------|------|------|----------|
| **管理员 (Admin)** | `admin` | 系统运维人员 | 全部管理端点、控制台全部功能 |
| **开发者 (Developer)** | 用户 ID | 使用 MCP 服务的开发者 | 自己的 MCP 请求、JWT 刷新 |

### 4.2 认证机制

#### 4.2.1 JWT Bearer Token（用户认证）

```go
// JWT Claims 结构
type Claims struct {
    Sub string `json:"sub"` // user id
    Exp int64  `json:"exp"` // unix seconds
    Iat int64  `json:"iat"`
}
```

**特点**:
- 算法：HS256
- 签名密钥：共享密钥，支持配置
- Token 刷新：`POST /refresh` 端点

#### 4.2.2 静态 Admin Token（管理员认证）

**认证方式**:
- Header: `X-Admin-Token: <token>`
- 或 Header: `Authorization: Bearer <token>`

**使用场景**:
- `/admin/*` 端点
- `/api/console/login`

#### 4.2.3 控制台会话 Cookie

**Cookie**:
- `cbmem_console`: 会话 ID
- `cbmem_csrf`: CSRF Token

**CSRF 保护**:
- 非 GET 请求必须携带 `X-CSRF-Token` Header
- Token 值与 Cookie 中的 `cbmem_csrf` 匹配

### 4.3 权限控制点

| 检查点 | 描述 |
|--------|------|
| JWT 签名验证 | 所有 `/mcp` 请求 |
| 用户存在性 | 用户必须在注册表中 |
| 用户未禁用 | `disabled=false` |
| 项目路径白名单 | 在 `project_paths` 中或为仓库路径 |
| 控制台会话 | 有效且未过期 |
| CSRF Token | 非 GET 请求 |

---

## 5. API 接口说明

### 5.1 统一响应格式

```json
{
  "code": 0,
  "msg": "",
  "data": {}
}
```

### 5.2 错误码规范

| 区间 | 含义 | 说明 |
|------|------|------|
| `0` | 成功 | - |
| `40100xx` | 鉴权失败 | admin-token 错误 / session 过期 / CSRF 失败 |
| `40300xx` | 无权访问 | 白名单违规 / 用户已禁用 |
| `40400xx` | 资源不存在 | 用户/项目/会话未找到 |
| `40900xx` | 资源冲突 | 唯一约束冲突（如 user id 重复） |
| `50000xx` | 系统错误 | LLM 超时 / SQLite 写入失败 |

### 5.3 MCP 端点

#### POST /mcp

**功能**: 转发 MCP JSON-RPC 请求到用户子进程

**请求**:
```http
POST /mcp?project=/code/foo
Authorization: Bearer <jwt>
Content-Type: application/json

<JSON-RPC Request>
```

**响应**:
```json
<JSON-RPC Response>
```

### 5.4 管理端点

#### POST /admin/users

创建用户

**请求**:
```json
{
  "id": "alice",
  "display_name": "Alice",
  "project_paths": ["/code/foo"],
  "max_procs": 4
}
```

**响应**: `201 Created`

#### GET /admin/users

列出所有用户

**响应**:
```json
{
  "users": [
    {
      "id": "alice",
      "display_name": "Alice",
      "project_paths": ["/code/foo"],
      "max_procs": 4,
      "disabled": false,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

#### PATCH /admin/users/:id

部分更新用户

**请求**:
```json
{
  "display_name": "Alice Updated",
  "disabled": true
}
```

#### DELETE /admin/users/:id

删除用户（同时清除进程池）

#### POST /admin/users/:id/token

签发 JWT

**查询参数**: `ttl=720h`（默认 30 天）

**响应**:
```json
{
  "user_id": "alice",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6...",
  "expires": "2026-02-01T00:00:00Z"
}
```

#### GET /admin/stats

全局统计

**响应**:
```json
{
  "registered_users": 10,
  "pool": {
    "live_procs": 5,
    "users": {...}
  }
}
```

#### POST /admin/repos/clone

克隆仓库到用户目录

**请求**:
```json
{
  "user_id": "alice",
  "url": "https://github.com/example/repo.git"
}
```

### 5.5 控制台端点

#### 认证

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/console/login` | POST | admin-token 登录 |
| `/api/console/logout` | POST | 登出 |
| `/api/console/me` | GET | 当前管理员信息 |

#### 用户管理 (CON-01)

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/console/users` | GET | 列表，`?q=` 搜索 |
| `/api/console/users` | POST | 新增 |
| `/api/console/users/:id` | PUT | 更新 |
| `/api/console/users/:id` | DELETE | 删除 |
| `/api/console/users/:id/revoke` | POST | 撤销访问 |

#### 项目管理 (CON-02)

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/console/projects` | GET | 列表 |
| `/api/console/projects` | POST | 新增 |
| `/api/console/projects/:id` | PUT | 更新 |
| `/api/console/projects/:id` | DELETE | 删除（软删除） |
| `/api/console/projects/:id/index-status` | GET | 索引状态 |
| `/api/console/projects/:id/reindex` | POST | 触发重新索引 |

#### 会话管理 (CON-03)

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/console/sessions` | GET | 列表，`?user_id=&project=&from=&to=` |
| `/api/console/sessions/:id` | GET | 详情（含 turns） |
| `/api/console/sessions-stats` | GET | 统计聚合 |

#### 归纳 (CON-04)

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/console/summarize` | POST | 触发归纳 |
| `/api/console/summarize/:task_id` | GET | 任务状态/结果 |

**归纳请求**:
```json
{
  "source_ids": ["sess_xxx", "sess_yyy"],
  "depth": "deep",
  "target_wing": "engineering"
}
```

#### 蒸馏 (CON-05)

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/console/distill` | POST | 触发蒸馏 |
| `/api/console/distill/:task_id` | GET | 任务状态/结果 |
| `/api/console/distill/:task_id/commit` | POST | 写入 MemPalace |

**蒸馏请求**:
```json
{
  "source_ids": ["sess_xxx"],
  "rules": {
    "min_value_score": 0.7,
    "dimensions": ["fact", "decision", "discovery"]
  }
}
```

---

## 6. 配置参数说明

### 6.1 必需参数

| 参数 | 描述 | 示例 |
|------|------|------|
| `-listen` | HTTP 监听地址 | `:8787` |
| `-data` | 数据目录（含 SQLite） | `/var/lib/cbmem-team` |
| `-mcp-bin` | `codebase-memory-mcp` 二进制路径 | `/usr/local/bin/codebase-memory-mcp` |
| `-jwt-secret` | JWT 签名密钥 | `openssl rand -hex 32` |
| `-admin-token` | 控制台管理员静态 token | `openssl rand -hex 32` |
| `-console-dist` | VitePress 构建产物目录 | `/opt/cbmem-console/dist` |

### 6.2 LLM 参数

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `-llm-provider` | 提供商：`openai` / `ollama` / `fake` | `fake` |
| `-llm-model` | 模型名称 | `gpt-4o-mini` |
| `-llm-base-url` | API base URL | `https://api.openai.com/v1` |
| `-llm-api-key` | API 密钥 | - |
| `-llm-timeout` | 超时时长 | `30s` |

### 6.3 MemPalace 参数

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `-mempalace-base` | MemPalace HTTP 地址 | `http://localhost:8089` |
| `-mempalace-timeout` | 超时时长 | `10s` |

### 6.4 可选参数

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `-config` | 配置文件路径 | `/etc/cbmem-team/config.yaml` |
| `-log` | 日志级别：`debug\|info\|warn\|error` | `info` |
| `-idle-ttl` | 空闲进程回收时间 | `30m` |
| `-max-procs-per-user` | 每个用户的最大进程数 | `4` |

---

## 7. 数据库设计

### 7.1 表结构

#### users 表

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT | 主键，用户 ID |
| display_name | TEXT | 显示名称 |
| project_paths | TEXT | JSON 数组，项目路径白名单 |
| max_procs | INTEGER | 最大进程数限制 |
| disabled | INTEGER | 是否禁用（0/1） |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

#### projects 表

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT | 主键，项目 ID |
| name | TEXT | 项目名称 |
| path | TEXT | 项目路径（UNIQUE） |
| wing | TEXT | MemPalace Wing |
| mcp_bin | TEXT | MCP 二进制路径 |
| user_id | TEXT | 关联用户 ID |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |
| deleted | INTEGER | 软删除标记 |

#### sessions 表

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT | 主键，会话 ID |
| user_id | TEXT | 用户 ID |
| project_id | TEXT | 项目 ID（可选） |
| project_path | TEXT | 项目路径 |
| started_at | DATETIME | 开始时间 |
| ended_at | DATETIME | 结束时间 |
| tool_count | INTEGER | 工具调用次数 |
| turn_count | INTEGER | 对话轮次 |
| summary | TEXT | 摘要（预留） |

#### session_turns 表

| 字段 | 类型 | 描述 |
|------|------|------|
| id | INTEGER | 主键，自增 |
| session_id | TEXT | 会话 ID |
| turn_no | INTEGER | 轮次编号 |
| role | TEXT | 角色（user/assistant） |
| content | TEXT | 消息内容 |
| tools_json | TEXT | 工具调用 JSON |
| ts | DATETIME | 时间戳 |

#### summarize_tasks 表

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT | 主键，任务 ID |
| user_id | TEXT | 用户 ID |
| source_ids | TEXT | 源会话 ID 列表（JSON） |
| depth | TEXT | 归纳深度 |
| target_wing | TEXT | 目标 Wing |
| status | TEXT | 状态：running/done/error |
| result_json | TEXT | LLM 输出结果 |
| created_at | DATETIME | 创建时间 |
| finished_at | DATETIME | 完成时间 |

#### distill_tasks 表

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT | 主键，任务 ID |
| user_id | TEXT | 用户 ID |
| source_ids | TEXT | 源会话 ID 列表（JSON） |
| rules_json | TEXT | 蒸馏规则（JSON） |
| status | TEXT | 状态：running/done/error |
| result_json | TEXT | LLM 输出结果 |
| mempalace_synced | INTEGER | 是否已同步到 MemPalace |
| target_wing | TEXT | 目标 Wing |
| created_at | DATETIME | 创建时间 |
| finished_at | DATETIME | 完成时间 |

#### console_sessions 表

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT | 主键，会话 ID |
| user_id | TEXT | 用户 ID |
| created_at | DATETIME | 创建时间 |
| expires_at | DATETIME | 过期时间 |
| last_seen_at | DATETIME | 最后访问时间 |
| ip | TEXT | 客户端 IP |
| ua | TEXT | User-Agent |

### 7.2 索引

| 表 | 索引 | 用途 |
|----|------|------|
| projects | idx_projects_user | 按用户查询项目 |
| sessions | idx_sessions_user | 按用户查询会话 |
| sessions | idx_sessions_project | 按项目查询会话 |
| sessions | idx_sessions_started | 按时间排序 |
| session_turns | idx_session_turns_session | 查询会话轮次 |
| console_sessions | idx_console_sessions_expires | 过期会话清理 |

---

## 8. 安全机制

### 8.1 认证安全

| 机制 | 描述 | 实现 |
|------|------|------|
| JWT 签名验证 | HMAC-SHA256 签名 | `internal/auth/auth.go` |
| Token 过期 | Expiration 时间戳检查 | Claims 中的 `exp` |
| Admin Token | 常量时间比较 | `subtleEqual()` 防止时序攻击 |
| 会话 Cookie | TTL 限制 | 8 小时默认 |

### 8.2 访问控制

| 机制 | 描述 |
|------|------|
| 用户白名单 | 项目路径必须在 `project_paths` 中 |
| 用户禁用 | `disabled=true` 用户无法访问 MCP |
| CSRF 保护 | POST/PUT/DELETE 请求验证 CSRF Token |

### 8.3 数据隔离

| 机制 | 描述 |
|------|------|
| Per-user 进程 | 每个用户独立的 codebase-memory-mcp 进程 |
| SQLite 文件锁 | 每个用户的索引在独立文件中 |
| 进程池隔离 | 用户之间进程完全隔离 |

### 8.4 部署安全建议

```markdown
- [x] JWT 签名验证
- [x] Per-user 项目路径白名单
- [ ] TLS：部署在 nginx/caddy 后，不要直接暴露 :8787
- [ ] Admin token 轮换：每季度轮换一次，存储在 vault 中
- [ ] 磁盘配额：监控 /var/lib/cbmem-team/users/* 的磁盘使用
```

---

## 9. 部署指南

### 9.1 构建

```bash
cd tools/cbmem-team
go build -o cbmem-team ./cmd/cbmem-team
```

### 9.2 systemd 部署

```ini
[Unit]
Description=cbmem-team HTTP wrapper + Console

[Service]
ExecStart=/usr/local/bin/cbmem-team \
  -listen :8787 \
  -data /var/lib/cbmem-team \
  -mcp-bin /usr/local/bin/codebase-memory-mcp \
  -jwt-secret /run/secrets/cbmem-jwt-secret \
  -admin-token /run/secrets/cbmem-admin-token \
  -llm-provider openai \
  -llm-model gpt-4o-mini \
  -llm-base-url https://api.openai.com/v1 \
  -llm-api-key /run/secrets/cbmem-llm-api-key \
  -mempalace-base http://192.168.100.83:8089 \
  -console-dist /opt/cbmem-console/dist \
  -log info
Restart=always
User=cbmem
```

### 9.3 Cursor 客户端配置

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://mcp.internal.example.com:8787/mcp?as=alice&project=${workspaceFolder}",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...."
      }
    }
  }
}
```

### 9.4 目录权限要求

| 路径 | 权限 | 说明 |
|------|------|------|
| `/usr/local/bin/codebase-memory-mcp` | 可执行 | MCP 二进制文件 |
| `/var/lib/cbmem-team` | 755 | 数据目录 |
| `/etc/cbmem-team/` | 600 | 配置文件（包含密钥） |

---

## 10. 附录

### 10.1 提示词模板

#### 归纳提示词 (summarize_*.txt)

支持三种深度：
- `summarize_shallow.txt` - 浅层归纳
- `summarize_deep.txt` - 深度归纳
- `summarize_expert.txt` - 专家级归纳

#### 蒸馏提示词 (distill_default.txt)

模板变量：
- `{{.MinScore}}` - 最小评分阈值
- `{{.Dimensions}}` - 提取维度列表

### 10.2 限制与已知权衡

| 问题 | 当前状态 | 未来计划 |
|------|----------|----------|
| 传输协议 | 仅 HTTP | v0.3 支持 SSE |
| 认证模型 | 共享 HS256 密钥 | 切换到 per-user 密钥 + Vault transit |
| 索引共享 | 无（有意为之） | 添加 `team_share: true` 启用共享只读 SQLite 挂载 |
| 高可用 | 单实例 | 粘性会话负载均衡（无共享状态） |

### 10.3 常见问题

#### Q: 为什么需要 JWT secret？

A: JWT 用于验证用户身份，确保只有注册用户可以访问 MCP 服务。secret 是签名密钥，必须妥善保管。

#### Q: 项目路径白名单是可选的吗？

A: 是的。如果用户没有设置 `project_paths`，则允许任何项目路径。如果设置了，则只允许白名单中的路径。

#### Q: 会话捕获会影响 MCP 性能吗？

A: 不会。捕获在单独的 goroutine 中异步执行，不会阻塞 MCP 请求。写入使用 per-user channel 串行化，避免 SQLite 锁定问题。

#### Q: 如何处理大团队场景？

A: 当前设计支持水平扩展，每个实例独立运行。对于大团队，可以部署多个实例并使用负载均衡器。

---

**文档结束**

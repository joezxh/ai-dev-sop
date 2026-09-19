# cbmem-team 产品需求文档 (PRD)

> **文档版本**: v2.0  
> **创建日期**: 2026-07-10  
> **更新日期**: 2026-07-14  
> **产品名称**: cbmem-team  
> **产品定位**: HTTP 多用户包装器 + 控制台，围绕 codebase-memory-mcp 实现团队共享实例，集成团队/项目/模块/会话/内存管理与 AI 工具执行

---

## 目录

1. [产品概述](#1-产品概述)
2. [核心决策清单 (v2 变更)](#2-核心决策清单-v2-变更)
3. [功能列表](#3-功能列表)
4. [数据模型](#4-数据模型)
5. [用户角色与权限](#5-用户角色与权限)
6. [API 接口说明](#6-api-接口说明)
7. [配置参数说明](#7-配置参数说明)
8. [技术架构](#8-技术架构)
9. [安全机制](#9-安全机制)
10. [部署指南](#10-部署指南)
11. [MVP 范围界定](#11-mvp-范围界定)
12. [附录](#12-附录)

---

## 1. 产品概述

### 1.1 产品背景

`codebase-memory-mcp` 设计为单开发者、本地机器工具：每个开发者运行自己的二进制文件处理自己的代码副本。这种设计有利于隐私和零配置，但意味着团队无法共享中央实例。

### 1.2 产品目标

`cbmem-team` 是团队级协作平台：
- 让多个开发者共享单一远程实例
- 保持每个用户的代码 AST 索引严格隔离（per-user 进程池）
- 提供集中式的项目/团队/模块/会话/内存管理
- 支持 AI 工具的注册、授权与执行审计
- 提供团队级别的知识蒸馏与共享能力

### 1.3 核心价值

| 特性 | 描述 |
|------|------|
| **用户隔离** | 每个用户拥有独立的 stdio 子进程，SQLite 文件锁提供自然隔离 |
| **团队共享** | 团队成员共享中央 MCP 实例，降低资源消耗 |
| **团队组织** | 团队作为容器，团队↔成员多对多关系，支持跨项目协作 |
| **模块树管理** | 项目下支持自由树状模块结构，会话/内存严格归属叶子节点 |
| **会话捕获** | 自动采集开发者的所有对话会话 |
| **知识蒸馏** | LLM 驱动的会话归纳和蒸馏能力 |
| **AI 工具执行** | 注册第三方 AI 工具，支持在项目工作目录 + 角色权限控制下执行 |
| **集中管理** | Web 控制台提供用户、团队、项目、模块、会话、内存、AI 工具的统一管理界面 |

### 1.4 v2 主要变化

相比 v1.0，v2.0 在以下方面进行了重新设计：

1. **新增团队 (Team) 概念**：项目归属从单一用户改为团队，团队↔成员多对多
2. **新增模块树 (Module Tree)**：项目下支持多层级模块，会话/内存严格归属叶子节点
3. **新增 AI 工具 (AI Tool) 注册与执行**：支持第三方 AI 工具的注册、授权、执行审计
4. **用户认证模型变更**：从 JWT + admin-token 双轨，改为强制用户名/密码 + 首次登录强制改密
5. **项目路径管理简化**：统一单一全局路径（不再区分 admin-clone vs user-local）
6. **项目 git 仓库自动克隆**：创建项目时自动从 git URL 克隆
7. **内存生成方式**：手动 + 模板辅助
8. **角色权限边界**：严格分层（管理员仅管用户/团队，不管理业务数据）

---

## 2. 核心决策清单 (v2 变更)

本节是 v2 PRD 的"决策日志"，所有决策项均已在 2026-07-14 通过 AskQuestion 与用户确认。

| # | 决策项 | 选择 | 影响范围 |
|---|--------|------|----------|
| 1 | 用户登录认证 | **现有用户默认密码 + 首次登录强制修改** | 用户认证流程、登录 UI |
| 2 | 项目归属模型 | **团队作为容器，团队↔成员多对多** | teams 表、team_members 表、projects.team_id |
| 3 | 项目路径管理 | **单一全局路径** | projects.path 唯一约束、删除 project_paths 白名单 |
| 4 | 项目 git 仓库 | **创建项目时自动克隆** | 创建项目 API 增加 git_clone_url、自动触发 git clone |
| 5 | AI 工具定位 | **通过 cbmem-team 调用（双层架构）** | 新增 ai_tools、ai_tool_invocations 表、tool gateway |
| 6 | AI 工具执行权限 | **项目工作目录 + 角色权限** | ai_tool_invocations.working_dir、tool RBAC |
| 7 | 模块树设计 | **自由树状结构** | modules 表 parent_id 自引用、递归查询 |
| 8 | 会话/内存归属 | **严格归属叶子节点** | sessions.module_id NOT NULL、memories.module_id NOT NULL |
| 9 | 蒸馏触发机制 | **手动触发** | UI 增加"Summarize"按钮，无定时器 |
| 10 | 内存生成方式 | **手动 + 模板辅助** | UI 提供模板选择，LLM 仅做润色/分类 |
| 11 | 用户权限边界 | **严格分层（管理员不管数据）** | admin 角色仅管 users/teams，business 角色管业务数据 |
| 12 | 配置变更影响 | **即时生效** | 无需重启，watch 配置热更新 |
| 13 | MVP 范围 | **包含 AI 工具调用** | 一次性交付数据层 + UI + RBAC + AI 工具 |
| 14 | 实施节奏 | **一次性完整交付 MVP** | 完整设计 + 完整实施 + 完整验收 |

---

## 3. 功能列表

### 3.1 功能模块总览

```
┌─────────────────────────────────────────────────────────────────┐
│                        cbmem-team v2                            │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │ MCP 包装器   │  │ 用户/认证    │  │ 团队管理                 │ │
│  │ (HTTP Wrapper)│  │ (User+Auth) │  │ (Team Management)      │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │ 项目管理    │  │ 模块树      │  │ 会话捕获                  │ │
│  │ (Project)   │  │ (Module)    │  │ (Session Capture)        │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │ 内存管理    │  │ AI 工具     │  │ 控制台 UI                │ │
│  │ (Memory)    │  │ (AI Tools)  │  │ (Console UI)             │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │ 归纳/蒸馏    │  │ MemPalace   │  │ 进程池管理               │ │
│  │ (LLM Tasks) │  │ (Integration)│ │ (Process Pool)          │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心功能详细说明

#### 3.2.1 用户管理与认证 (User + Auth) — v2 重大变更

**v2 新增**：用户名/密码认证 + 首次登录强制改密。

| 功能 | 描述 | API 端点 |
|------|------|----------|
| **用户创建** | 管理员创建用户，设置 ID、显示名、初始密码、默认团队 | `POST /api/admin/users` |
| **用户查询** | 列出所有用户，支持关键字搜索和分页 | `GET /api/admin/users` |
| **用户更新** | 部分更新用户信息（显示名、禁用状态、团队） | `PUT /api/admin/users/:id` |
| **用户删除** | 从注册表中移除用户（级联清理 team_membership） | `DELETE /api/admin/users/:id` |
| **用户登录** | 用户名/密码登录，返回 access_token + refresh_token | `POST /api/auth/login` |
| **首次登录改密** | 强制首次登录用户修改密码 | `POST /api/auth/change-password` (must_change_password=true 时强制) |
| **Token 刷新** | 用 refresh_token 换取新 access_token | `POST /api/auth/refresh` |
| **当前用户** | 获取当前登录用户信息 | `GET /api/auth/me` |

**用户数据结构（v2）**：
```json
{
  "id": "alice",
  "username": "alice",
  "display_name": "Alice Wang",
  "email": "alice@example.com",
  "default_team_id": "team_eng",
  "role": "developer",  // developer / lead / admin
  "must_change_password": false,
  "disabled": false,
  "created_at": "2026-01-01T00:00:00Z",
  "last_login_at": "2026-07-14T08:30:00Z"
}
```

**角色定义**：
| 角色 | 标识 | 权限范围 |
|------|------|----------|
| **管理员** | `admin` | 用户/团队 CRUD、查看全局统计；**不管理业务数据（项目/模块/会话/内存）** |
| **团队负责人** | `lead` | 团队内所有项目/模块/会话/内存的完整 CRUD；可邀请成员 |
| **开发者** | `developer` | 自己创建的会话/内存 CRUD；可见团队内所有项目；可调用 AI 工具 |
| **只读** | `viewer` | 团队内只读访问 |

#### 3.2.2 团队管理 (Team Management) — v2 新增

**v2 新增**：团队作为顶层容器，承载项目和成员关系。

| 功能 | 描述 | API 端点 |
|------|------|----------|
| **创建团队** | 管理员创建团队 | `POST /api/admin/teams` |
| **列出团队** | 管理员列出所有团队 | `GET /api/admin/teams` |
| **更新团队** | 管理员更新团队信息（名称、描述） | `PUT /api/admin/teams/:id` |
| **删除团队** | 管理员删除团队（必须无项目关联） | `DELETE /api/admin/teams/:id` |
| **添加成员** | 管理员或团队负责人添加成员 | `POST /api/teams/:id/members` |
| **移除成员** | 管理员或团队负责人移除成员 | `DELETE /api/teams/:id/members/:user_id` |
| **列出成员** | 列出团队成员 | `GET /api/teams/:id/members` |
| **团队详情** | 查看团队信息（项目数、成员数） | `GET /api/teams/:id` |

**团队数据结构**：
```json
{
  "id": "team_eng",
  "name": "Engineering Team",
  "slug": "engineering",
  "description": "Backend engineering team",
  "owner_id": "alice",
  "member_count": 8,
  "project_count": 5,
  "created_at": "2026-01-01T00:00:00Z"
}
```

**团队↔成员多对多**：
```sql
team_members (
  team_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  role TEXT NOT NULL,  -- lead / developer / viewer
  joined_at DATETIME,
  PRIMARY KEY (team_id, user_id)
)
```

#### 3.2.3 项目管理 (Project Management) — v2 简化

**v2 变更**：单一全局路径 + git clone + 归属团队。

| 功能 | 描述 | API 端点 |
|------|------|----------|
| **创建项目** | 团队负责人创建项目，自动 git clone | `POST /api/teams/:team_id/projects` |
| **列出项目** | 按团队筛选项目 | `GET /api/projects?team_id=xxx` |
| **项目详情** | 查看项目（含 git 状态、最近会话） | `GET /api/projects/:id` |
| **更新项目** | 团队负责人更新项目 | `PUT /api/projects/:id` |
| **删除项目** | 软删除项目（级联清理模块、会话、内存） | `DELETE /api/projects/:id` |
| **索引状态** | 检查 AST 索引状态 | `GET /api/projects/:id/index-status` |
| **重新索引** | 触发增量或全量索引 | `POST /api/projects/:id/reindex` |
| **Git Pull** | 拉取最新代码 | `POST /api/projects/:id/git-pull` |
| **Git Status** | 查看 git 状态 | `GET /api/projects/:id/git-status` |

**创建项目请求**：
```json
{
  "name": "payment-service",
  "slug": "payment",
  "description": "Payment microservice",
  "git_url": "https://github.com/example/payment-service.git",
  "git_branch": "main",
  "default_branch": "main"
}
```

**自动 git clone 行为**：
1. 创建项目记录（status=cloning）
2. 后台 goroutine 异步 `git clone ${git_url} ${project.path}`
3. 完成后启动 codebase-memory-mcp 子进程并索引
4. 项目状态更新为 `ready`

**项目数据结构（v2）**：
```json
{
  "id": "proj_abc123",
  "team_id": "team_eng",
  "name": "payment-service",
  "slug": "payment",
  "description": "Payment microservice",
  "path": "/var/lib/cbmem-team/repos/team_eng/payment",
  "git_url": "https://github.com/example/payment-service.git",
  "git_branch": "main",
  "git_commit_sha": "a1b2c3d4",
  "status": "ready",  // cloning / indexing / ready / error
  "owner_id": "alice",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "deleted": false
}
```

#### 3.2.4 模块树 (Module Tree) — v2 新增

**v2 新增**：项目下的自由树状模块结构。

| 功能 | 描述 | API 端点 |
|------|------|----------|
| **创建模块** | 在项目下创建模块（可指定 parent） | `POST /api/projects/:project_id/modules` |
| **模块树** | 获取项目完整模块树 | `GET /api/projects/:project_id/modules/tree` |
| **模块详情** | 查看模块（含子模块计数、会话计数） | `GET /api/modules/:id` |
| **更新模块** | 重命名/移动模块 | `PUT /api/modules/:id` |
| **删除模块** | 删除模块（子树递归删除或阻止） | `DELETE /api/modules/:id` |
| **移动模块** | 修改 parent（不允许循环引用） | `POST /api/modules/:id/move` |

**模块数据结构**：
```json
{
  "id": "mod_backend",
  "project_id": "proj_abc123",
  "parent_id": null,  // 根模块
  "name": "backend",
  "path": "src/backend",  // 可选，代码内相对路径
  "description": "Backend service modules",
  "order": 0,
  "child_count": 5,
  "session_count": 23,
  "memory_count": 12,
  "created_at": "2026-01-01T00:00:00Z"
}
```

**层级示例**：
```
project: payment-service
├── mod: backend (parent: null)
│   ├── mod: api (parent: backend)
│   │   └── mod: controllers (parent: api) ← 叶子节点
│   ├── mod: service (parent: backend) ← 叶子节点
│   └── mod: repository (parent: backend) ← 叶子节点
└── mod: frontend (parent: null)
    └── mod: components (parent: frontend) ← 叶子节点
```

**叶子节点约束**：会话和内存**只能归属叶子节点**（即 `is_leaf = true` 的模块）。

#### 3.2.5 会话捕获与管理 (Session Capture & Management) — v2 强化归属

**v2 变更**：会话严格归属叶子模块。

| 功能 | 描述 | API 端点 |
|------|------|----------|
| **自动捕获** | 拦截 MCP 请求，自动创建/复用会话 | `internal/console/capture.go` |
| **会话复用策略** | 同一 (user, project, module) 30 分钟内有最近会话则复用 | 滚动会话模型 |
| **轮次记录** | 记录每轮对话的角色、内容、工具调用 | `session_turns` 表 |
| **模块归属** | 每个会话**必须**归属叶子模块 | sessions.module_id NOT NULL |
| **会话列表** | 按团队/项目/模块/用户/时间筛选 | `GET /api/sessions` |
| **会话详情** | 查看会话及其所有轮次 | `GET /api/sessions/:id` |
| **会话统计** | 聚合统计（模块维度） | `GET /api/sessions/stats` |

**会话数据结构（v2）**：
```json
{
  "id": "sess_xyz789",
  "user_id": "alice",
  "team_id": "team_eng",
  "project_id": "proj_abc123",
  "module_id": "mod_controllers",
  "project_path": "/var/lib/cbmem-team/repos/team_eng/payment",
  "started_at": "2026-01-01T10:00:00Z",
  "ended_at": "2026-01-01T10:30:00Z",
  "tool_count": 15,
  "turn_count": 8
}
```

#### 3.2.6 内存管理 (Memory Management) — v2 强化归属与模板

**v2 变更**：内存严格归属叶子模块 + 手动 + 模板辅助。

| 功能 | 描述 | API 端点 |
|------|------|----------|
| **手动创建内存** | 用户在叶子模块下手工添加内存 | `POST /api/modules/:id/memories` |
| **模板辅助** | 选择预置模板（如 ADR、技术决策、教训），填充字段后生成 | `GET /api/memory-templates` |
| **模板列表** | 列出可用模板 | `GET /api/memory-templates` |
| **模板渲染** | 将模板 + 用户输入渲染为完整 Markdown | `POST /api/memory-templates/:id/render` |
| **内存列表** | 按模块/项目/团队筛选 | `GET /api/memories` |
| **内存详情** | 查看单条内存 | `GET /api/memories/:id` |
| **更新内存** | 编辑内存内容 | `PUT /api/memories/:id` |
| **删除内存** | 删除内存 | `DELETE /api/memories/:id` |
| **标签** | 给内存打标签 | `POST /api/memories/:id/tags` |

**内存数据结构（v2）**：
```json
{
  "id": "mem_abc",
  "team_id": "team_eng",
  "project_id": "proj_abc123",
  "module_id": "mod_controllers",  // 必须为叶子模块
  "user_id": "alice",
  "title": "ADR-001: 选择 PostgreSQL 作为主数据库",
  "content": "...",
  "template_id": "tpl_adr",
  "tags": ["database", "architecture"],
  "hall": "facts",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

**预置模板**：
| 模板 ID | 名称 | 字段 | 适用场景 |
|---------|------|------|----------|
| `tpl_adr` | 架构决策记录 (ADR) | title / status / context / decision / consequences | 重要架构决策 |
| `tpl_lesson` | 经验教训 | problem / root_cause / solution / impact | 调试/事故复盘 |
| `tpl_snippet` | 代码片段 | language / description / code | 可复用代码 |
| `tpl_runbook` | 操作手册 | trigger / steps / rollback | 运维操作 |
| `tpl_decision` | 简单决策 | title / decision / context | 轻量决策 |

#### 3.2.7 AI 工具执行 (AI Tool Execution) — v2 新增

**v2 新增**：注册第三方 AI 工具，支持在项目工作目录 + 角色权限控制下执行。

**双层架构**：cbmem-team 作为网关，调用上游 AI 工具服务（如 codebase-memory-mcp、MemPalace、AI IDE 等）。

| 功能 | 描述 | API 端点 |
|------|------|----------|
| **注册工具** | 管理员/团队负责人注册 AI 工具 | `POST /api/teams/:id/ai-tools` |
| **列出工具** | 列出可用 AI 工具 | `GET /api/teams/:id/ai-tools` |
| **工具详情** | 查看工具配置与权限 | `GET /api/ai-tools/:id` |
| **更新工具** | 修改工具配置 | `PUT /api/ai-tools/:id` |
| **禁用工具** | 禁用工具（不删除） | `POST /api/ai-tools/:id/disable` |
| **调用工具** | 在项目工作目录中执行工具 | `POST /api/ai-tools/:id/invoke` |
| **调用记录** | 查看工具调用历史 | `GET /api/ai-tools/:id/invocations` |
| **调用状态** | 查询单次调用状态/结果 | `GET /api/ai-tool-invocations/:id` |

**AI 工具数据结构**：
```json
{
  "id": "tool_codegen",
  "team_id": "team_eng",
  "name": "Code Generator",
  "slug": "codegen",
  "description": "AI-powered code generator",
  "endpoint": "http://codegen.internal:9000",
  "protocol": "http",  // http / stdio
  "command": null,
  "args": null,
  "required_role": "developer",  // 调用所需的最低角色
  "allowed_projects": ["proj_abc", "proj_def"],  // null = 团队内所有
  "timeout_seconds": 300,
  "enabled": true,
  "created_at": "2026-01-01T00:00:00Z"
}
```

**调用请求**：
```json
{
  "project_id": "proj_abc123",
  "module_id": "mod_controllers",
  "input": {
    "prompt": "Generate a CRUD controller for User entity",
    "language": "go"
  },
  "working_dir": "/var/lib/cbmem-team/repos/team_eng/payment"
}
```

**执行流程**：
1. 鉴权（JWT + 角色检查）
2. 校验项目/模块归属 + 工具 allowed_projects
3. 创建 ai_tool_invocations 记录（status=pending）
4. 调用上游 AI 工具（HTTP/stdio）
5. 写回结果（status=success/error）+ 输出文件路径
6. 审计日志

**调用审计记录**：
```json
{
  "id": "inv_xyz",
  "tool_id": "tool_codegen",
  "user_id": "alice",
  "team_id": "team_eng",
  "project_id": "proj_abc123",
  "module_id": "mod_controllers",
  "working_dir": "/var/lib/cbmem-team/repos/team_eng/payment",
  "input": { ... },
  "output": { ... },
  "status": "success",
  "started_at": "2026-07-14T10:00:00Z",
  "finished_at": "2026-07-14T10:05:00Z",
  "error": null
}
```

#### 3.2.8 归纳与蒸馏 (Summarize & Distill) — v2 改为手动

**v2 变更**：手动触发，无定时器；LLM 仅做辅助润色。

| 功能 | 描述 | API 端点 |
|------|------|----------|
| **手动触发归纳** | 用户在 UI 选择会话点击"Summarize" | `POST /api/summarize-tasks` |
| **手动触发蒸馏** | 用户在 UI 选择会话点击"Distill" | `POST /api/distill-tasks` |
| **任务状态** | 查询归纳/蒸馏任务状态 | `GET /api/summarize-tasks/:id` |
| **结果预览** | 预览 LLM 输出（5 Hall / 知识片段） | 内嵌在前端 |

**v2 简化**：不再自动写入 MemPalace，由用户确认后手动 commit。

#### 3.2.9 控制台 UI (Console UI) — v2 重构

| 模块 | 路径 | 描述 | 权限 |
|------|------|------|------|
| **登录页** | `/login` | 用户名/密码登录 | 所有 |
| **首次改密** | `/change-password` | 强制首次登录改密 | 所有 |
| **项目列表** | `/projects` | 团队下所有项目 | 团队成员 |
| **项目详情** | `/projects/:id` | 项目概览 + 模块树 + 会话/内存统计 | 团队成员 |
| **模块详情** | `/modules/:id` | 模块信息 + 会话列表 + 内存列表 | 团队成员 |
| **会话详情** | `/sessions/:id` | 会话轮次 + 归纳/蒸馏入口 | 团队成员 |
| **内存详情** | `/memories/:id` | 内存查看/编辑 | 团队成员 |
| **AI 工具列表** | `/ai-tools` | 团队 AI 工具 | 团队成员 |
| **AI 工具详情** | `/ai-tools/:id` | 工具配置 + 调用历史 | 团队负责人 |
| **团队管理** | `/teams` | 团队 CRUD | 管理员 |
| **用户管理** | `/admin/users` | 用户 CRUD | 管理员 |
| **会话统计** | `/sessions/stats` | 团队/项目/模块维度统计 | 团队负责人 |

#### 3.2.10 MCP 协议包装器 (MCP Wrapper)

保留 v1 设计，per-user stdio 子进程路由。

```
HTTP 客户端 (Cursor/Claude/Qoder)
    │
    ▼ POST /mcp?project=/path
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

#### 3.2.11 MemPalace 集成（可选）

通过 AI 工具方式集成，而非直接调用。

#### 3.2.12 进程池管理 (Process Pool)

保留 v1 设计：per-user 独立进程、空闲回收、崩溃恢复。

#### 3.2.13 配置热更新 (Hot Reload) — v2 新增

| 功能 | 描述 |
|------|------|
| **watch 配置文件** | 启动时加载配置文件，运行时监听变更 |
| **即时生效** | 大部分配置变更无需重启进程 |
| **受限配置** | `-listen`、`-data` 等启动参数仍需重启 |

---

## 4. 数据模型

### 4.1 ER 关系图

```
                     ┌──────────┐
                     │  teams   │
                     └────┬─────┘
                          │ 1
                          │
              ┌───────────┼───────────┐
              │ N                   N │
              ▼                       ▼
      ┌──────────────────┐    ┌──────────────┐
      │ team_members     │    │  projects    │
      │ (team × user)    │    │              │
      └────────┬─────────┘    └──────┬───────┘
               │ N                   │ 1
               │                     │
               │ 1                   ▼ N
               │              ┌──────────────┐
               │              │   modules    │
               │              │  (自引用树)   │
               │              └──────┬───────┘
               │                     │ 1
               │                     │
               │                     ▼ N
               │              ┌──────────────┐
               │              │   sessions   │
               │              └──────┬───────┘
               │                     │ 1
               │                     │
               │                     ▼ N
               │              ┌──────────────┐
               │              │ session_turns│
               │              └──────────────┘
               │                     │
               │                     │ 1
               │                     ▼ N
               │              ┌──────────────┐
               ▼ N            │   memories   │
      ┌──────────────┐       └──────────────┘
      │    users     │
      └──────┬───────┘
             │ 1
             │
             ▼ N
      ┌──────────────────────┐
      │ ai_tool_invocations  │
      └──────────┬───────────┘
                 │ N
                 │
                 ▼ 1
        ┌──────────────┐
        │   ai_tools   │
        └──────────────┘
```

### 4.2 表结构（完整）

#### 4.2.1 users（v2 重构）

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT PK | 用户 ID |
| username | TEXT UNIQUE NOT NULL | 用户名（登录用） |
| display_name | TEXT | 显示名 |
| email | TEXT | 邮箱 |
| password_hash | TEXT NOT NULL | bcrypt 哈希 |
| default_team_id | TEXT FK | 默认团队 |
| role | TEXT | developer/lead/admin/viewer |
| must_change_password | INTEGER | 是否必须改密（0/1） |
| disabled | INTEGER | 是否禁用（0/1） |
| created_at | DATETIME | |
| updated_at | DATETIME | |
| last_login_at | DATETIME | |

#### 4.2.2 teams（v2 新增）

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT PK | 团队 ID |
| name | TEXT NOT NULL | 团队名称 |
| slug | TEXT UNIQUE NOT NULL | URL-friendly 标识 |
| description | TEXT | |
| owner_id | TEXT FK→users | 创建者 |
| created_at | DATETIME | |
| updated_at | DATETIME | |
| deleted | INTEGER | 软删除 |

#### 4.2.3 team_members（v2 新增）

| 字段 | 类型 | 描述 |
|------|------|------|
| team_id | TEXT FK→teams | |
| user_id | TEXT FK→users | |
| role | TEXT | lead/developer/viewer |
| joined_at | DATETIME | |
| PK | (team_id, user_id) | |

#### 4.2.4 projects（v2 重构）

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT PK | |
| team_id | TEXT FK→teams NOT NULL | 归属团队 |
| name | TEXT NOT NULL | |
| slug | TEXT NOT NULL | 在团队内唯一 |
| description | TEXT | |
| path | TEXT UNIQUE NOT NULL | 服务器端全局唯一路径 |
| git_url | TEXT | |
| git_branch | TEXT | |
| git_commit_sha | TEXT | 当前 commit |
| status | TEXT | cloning/indexing/ready/error |
| owner_id | TEXT FK→users | 创建者 |
| created_at | DATETIME | |
| updated_at | DATETIME | |
| deleted | INTEGER | |

**索引**：`UNIQUE(team_id, slug)`、`UNIQUE(path)`

#### 4.2.5 modules（v2 新增）

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT PK | |
| project_id | TEXT FK→projects | |
| parent_id | TEXT FK→modules NULL | 自引用 |
| name | TEXT NOT NULL | |
| path | TEXT | 代码内相对路径 |
| description | TEXT | |
| order | INTEGER | 排序 |
| is_leaf | INTEGER | 是否叶子（用于会话/内存归属校验） |
| created_at | DATETIME | |
| updated_at | DATETIME | |
| deleted | INTEGER | |

**索引**：`idx_modules_project`、`idx_modules_parent`、`UNIQUE(project_id, parent_id, name)`

#### 4.2.6 sessions（v2 强化归属）

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT PK | |
| user_id | TEXT FK→users NOT NULL | |
| team_id | TEXT FK→teams NOT NULL | |
| project_id | TEXT FK→projects NOT NULL | |
| module_id | TEXT FK→modules NOT NULL | **必须为叶子模块** |
| project_path | TEXT | 冗余 |
| started_at | DATETIME | |
| ended_at | DATETIME | |
| tool_count | INTEGER | |
| turn_count | INTEGER | |

**索引**：`idx_sessions_module`、`idx_sessions_user`、`idx_sessions_team`、`idx_sessions_started`

#### 4.2.7 session_turns

| 字段 | 类型 | 描述 |
|------|------|------|
| id | INTEGER PK AUTOINCREMENT | |
| session_id | TEXT FK→sessions | |
| turn_no | INTEGER | |
| role | TEXT | user/assistant/tool |
| content | TEXT | |
| tools_json | TEXT | |
| ts | DATETIME | |

#### 4.2.8 memory_templates（v2 新增）

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT PK | 模板 ID |
| name | TEXT NOT NULL | |
| description | TEXT | |
| fields_json | TEXT | 字段定义 JSON |
| body_template | TEXT | Markdown 模板 |
| is_builtin | INTEGER | 是否内置 |

#### 4.2.9 memories（v2 强化归属）

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT PK | |
| team_id | TEXT FK NOT NULL | |
| project_id | TEXT FK NOT NULL | |
| module_id | TEXT FK NOT NULL | **必须为叶子模块** |
| user_id | TEXT FK NOT NULL | 创建者 |
| title | TEXT NOT NULL | |
| content | TEXT NOT NULL | Markdown |
| template_id | TEXT FK | 使用的模板 |
| tags_json | TEXT | |
| hall | TEXT | facts/events/discoveries/preferences/advice |
| created_at | DATETIME | |
| updated_at | DATETIME | |
| deleted | INTEGER | |

#### 4.2.10 ai_tools（v2 新增）

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT PK | |
| team_id | TEXT FK→teams | |
| name | TEXT NOT NULL | |
| slug | TEXT NOT NULL | |
| description | TEXT | |
| endpoint | TEXT NOT NULL | HTTP endpoint |
| protocol | TEXT | http/stdio |
| command | TEXT | stdio 模式命令 |
| args_json | TEXT | |
| required_role | TEXT | developer/lead/admin |
| allowed_projects_json | TEXT | null=所有 |
| timeout_seconds | INTEGER | |
| enabled | INTEGER | |
| created_at | DATETIME | |
| updated_at | DATETIME | |

**索引**：`UNIQUE(team_id, slug)`

#### 4.2.11 ai_tool_invocations（v2 新增）

| 字段 | 类型 | 描述 |
|------|------|------|
| id | TEXT PK | |
| tool_id | TEXT FK→ai_tools | |
| user_id | TEXT FK→users | |
| team_id | TEXT FK→teams | |
| project_id | TEXT FK→projects | |
| module_id | TEXT FK→modules | |
| working_dir | TEXT NOT NULL | |
| input_json | TEXT | |
| output_json | TEXT | |
| status | TEXT | pending/running/success/error/timeout |
| started_at | DATETIME | |
| finished_at | DATETIME | |
| error | TEXT | |

#### 4.2.12 console_sessions

保留 v1 设计：控制台 web 会话。

#### 4.2.13 summarize_tasks / distill_tasks

保留 v1 设计。

### 4.3 RBAC 权限矩阵（v2 强化分层）

| 操作 | admin | lead (team owner) | developer | viewer |
|------|-------|-------------------|-----------|--------|
| 创建用户/团队 | ✅ | ❌ | ❌ | ❌ |
| 团队成员管理 | ✅ | ✅ (自己团队) | ❌ | ❌ |
| 项目 CRUD | ❌ | ✅ | ❌ | ❌ |
| 模块 CRUD | ❌ | ✅ | ❌ | ❌ |
| AI 工具注册 | ❌ | ✅ | ❌ | ❌ |
| AI 工具调用 | ❌ | ✅ | ✅ (受 allowed_projects) | ❌ |
| 会话查看 | ❌ | ✅ (全团队) | ✅ (自己) | ✅ (只读) |
| 内存 CRUD | ❌ | ✅ | ✅ (自己) | ❌ |
| 归纳/蒸馏触发 | ❌ | ✅ | ✅ (自己的会话) | ❌ |

**严格分层原则**：管理员只管"人"，不管理"业务数据"。

---

## 5. 用户角色与权限

### 5.1 角色定义（v2 重构）

| 角色 | 标识 | 描述 | 权限范围 |
|------|------|------|----------|
| **管理员 (Admin)** | `admin` | 系统运维 | 用户/团队 CRUD、全局统计；**不管理业务数据** |
| **团队负责人 (Lead)** | `lead` | 团队 owner | 团队内所有项目/模块/会话/内存的完整 CRUD |
| **开发者 (Developer)** | `developer` | 普通用户 | 自己创建的会话/内存 CRUD；可见团队内项目；可调用 AI 工具 |
| **只读 (Viewer)** | `viewer` | 旁观者 | 团队内只读访问 |

### 5.2 认证机制（v2 重构）

#### 5.2.1 用户名/密码登录

```http
POST /api/auth/login
Content-Type: application/json

{
  "username": "alice",
  "password": "initial-password-or-changed"
}
```

**响应**：
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "rt_abc123",
  "expires_in": 3600,
  "must_change_password": false,
  "user": {
    "id": "alice",
    "username": "alice",
    "display_name": "Alice",
    "role": "developer",
    "default_team_id": "team_eng"
  }
}
```

#### 5.2.2 首次登录强制改密

- 创建用户时 `must_change_password=true`
- 登录后前端检测到该标志，强制跳转 `/change-password`
- 修改成功后 `must_change_password=false`

#### 5.2.3 JWT Token 结构

```go
type Claims struct {
    Sub      string `json:"sub"`        // user id
    Username string `json:"username"`
    Role     string `json:"role"`
    TeamID   string `json:"team_id"`    // 当前团队
    Exp      int64  `json:"exp"`
    Iat      int64  `json:"iat"`
    Jti      string `json:"jti"`        // 用于撤销
}
```

#### 5.2.4 Token 刷新

```http
POST /api/auth/refresh
Content-Type: application/json

{
  "refresh_token": "rt_abc123"
}
```

### 5.3 权限控制点

| 检查点 | 描述 |
|--------|------|
| JWT 签名验证 | 所有 `/api/*` 请求 |
| 用户存在性 | users 表中存在 |
| 用户未禁用 | `disabled=false` |
| 角色检查 | 各端点 RBAC 中间件 |
| 团队归属 | 资源 team_id 必须等于用户 team_id |
| AI 工具 RBAC | tool.required_role + allowed_projects |
| 叶子模块约束 | sessions/memories 的 module_id 必须为叶子 |

---

## 6. API 接口说明

### 6.1 统一响应格式

```json
{
  "code": 0,
  "msg": "",
  "data": {}
}
```

### 6.2 错误码

| 区间 | 含义 |
|------|------|
| `0` | 成功 |
| `40000xx` | 请求参数错误 |
| `40100xx` | 鉴权失败（密码错误/token 无效/token 过期） |
| `40300xx` | 权限不足（角色不够/非团队成员） |
| `40400xx` | 资源不存在 |
| `40900xx` | 资源冲突（唯一约束冲突） |
| `42200xx` | 业务规则违反（如内存归属非叶子模块） |
| `50000xx` | 系统错误 |

### 6.3 认证端点（v2 重构）

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/auth/login` | POST | 用户名/密码登录 |
| `/api/auth/logout` | POST | 登出（撤销 refresh_token） |
| `/api/auth/refresh` | POST | 刷新 access_token |
| `/api/auth/me` | GET | 当前用户信息 |
| `/api/auth/change-password` | POST | 修改密码 |

### 6.4 管理端点（仅 admin）

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/admin/users` | GET | 列出用户 |
| `/api/admin/users` | POST | 创建用户 |
| `/api/admin/users/:id` | PUT | 更新用户 |
| `/api/admin/users/:id` | DELETE | 删除用户 |
| `/api/admin/users/:id/reset-password` | POST | 重置密码（admin 操作） |
| `/api/admin/teams` | GET | 列出团队 |
| `/api/admin/teams` | POST | 创建团队 |
| `/api/admin/teams/:id` | PUT | 更新团队 |
| `/api/admin/teams/:id` | DELETE | 删除团队 |

### 6.5 团队端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/teams/:id` | GET | 团队详情 |
| `/api/teams/:id/members` | GET | 列出成员 |
| `/api/teams/:id/members` | POST | 添加成员 |
| `/api/teams/:id/members/:user_id` | DELETE | 移除成员 |
| `/api/teams/:id/members/:user_id` | PUT | 修改成员角色 |

### 6.6 项目端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/projects` | GET | 列出项目（按 team 过滤） |
| `/api/teams/:team_id/projects` | POST | 创建项目（自动 git clone） |
| `/api/projects/:id` | GET | 项目详情 |
| `/api/projects/:id` | PUT | 更新项目 |
| `/api/projects/:id` | DELETE | 删除项目（软删除） |
| `/api/projects/:id/index-status` | GET | AST 索引状态 |
| `/api/projects/:id/reindex` | POST | 触发索引 |
| `/api/projects/:id/git-status` | GET | git 状态 |
| `/api/projects/:id/git-pull` | POST | git pull |

### 6.7 模块端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/projects/:project_id/modules` | POST | 创建模块 |
| `/api/projects/:project_id/modules/tree` | GET | 模块树 |
| `/api/modules/:id` | GET | 模块详情 |
| `/api/modules/:id` | PUT | 更新模块 |
| `/api/modules/:id` | DELETE | 删除模块 |
| `/api/modules/:id/move` | POST | 移动模块 |

### 6.8 会话端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/sessions` | GET | 列出（按 user/team/project/module 过滤） |
| `/api/sessions/:id` | GET | 会话详情 |
| `/api/sessions/stats` | GET | 统计聚合 |

### 6.9 内存端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/memories` | GET | 列表（按模块过滤） |
| `/api/modules/:id/memories` | POST | 创建内存 |
| `/api/memories/:id` | GET | 详情 |
| `/api/memories/:id` | PUT | 更新 |
| `/api/memories/:id` | DELETE | 删除 |
| `/api/memories/:id/tags` | POST | 加标签 |

### 6.10 内存模板端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/memory-templates` | GET | 列出模板 |
| `/api/memory-templates/:id` | GET | 模板详情 |
| `/api/memory-templates/:id/render` | POST | 渲染模板 |

### 6.11 AI 工具端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/teams/:team_id/ai-tools` | POST | 注册工具 |
| `/api/teams/:team_id/ai-tools` | GET | 列出工具 |
| `/api/ai-tools/:id` | GET | 工具详情 |
| `/api/ai-tools/:id` | PUT | 更新工具 |
| `/api/ai-tools/:id/disable` | POST | 禁用 |
| `/api/ai-tools/:id/enable` | POST | 启用 |
| `/api/ai-tools/:id/invoke` | POST | 调用工具 |
| `/api/ai-tools/:id/invocations` | GET | 调用历史 |
| `/api/ai-tool-invocations/:id` | GET | 单次调用状态 |

### 6.12 归纳/蒸馏端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/summarize-tasks` | POST | 触发归纳 |
| `/api/summarize-tasks/:id` | GET | 任务状态 |
| `/api/distill-tasks` | POST | 触发蒸馏 |
| `/api/distill-tasks/:id` | GET | 任务状态 |

### 6.13 MCP 端点（保留）

| 端点 | 方法 | 描述 |
|------|------|------|
| `/mcp` | POST | MCP JSON-RPC 转发 |
| `/healthz` | GET | 健康检查 |
| `/admin/stats` | GET | 全局统计 |

---

## 7. 配置参数说明

### 7.1 必需参数（v2 部分变更）

| 参数 | 描述 | 示例 |
|------|------|------|
| `-listen` | HTTP 监听地址 | `:8787` |
| `-data` | 数据目录 | `/var/lib/cbmem-team` |
| `-mcp-bin` | MCP 二进制路径 | `/usr/local/bin/codebase-memory-mcp` |
| `-jwt-secret` | JWT 签名密钥 | `openssl rand -hex 32` |
| `-console-dist` | VitePress 产物目录 | `/opt/cbmem-console/dist` |

### 7.2 v2 移除/变更

- **移除** `-admin-token`：v2 改为用户名/密码，不再使用 admin-token
- **移除** `-mempalace-*`：MemPalace 改为通过 AI 工具调用
- **新增** `-initial-admin-username` / `-initial-admin-password`：首次启动创建初始管理员

### 7.3 LLM 参数（保留）

| 参数 | 描述 | 默认 |
|------|------|------|
| `-llm-provider` | openai/ollama/fake | fake |
| `-llm-model` | 模型名称 | gpt-4o-mini |
| `-llm-base-url` | API base URL | - |
| `-llm-api-key` | API 密钥 | - |
| `-llm-timeout` | 超时 | 30s |

### 7.4 可选参数

| 参数 | 描述 | 默认 |
|------|------|------|
| `-config` | 配置文件 | /etc/cbmem-team/config.yaml |
| `-log` | 日志级别 | info |
| `-idle-ttl` | 进程空闲回收 | 30m |
| `-max-procs-per-user` | 每用户最大进程 | 4 |
| `-default-password` | 新用户初始密码 | `changeme` |
| `-config-watch` | 监听配置文件变更 | true |

### 7.5 配置热更新（v2 新增）

通过 fsnotify 监听 `-config` 文件变更：
- 大部分配置项（LLM、超时、日志级别）即时生效
- 启动参数（`-listen`、`-data`）需重启
- 配置变更审计日志写入数据库

---

## 8. 技术架构

### 8.1 系统架构图（v2）

```
                           ┌─────────────────────────────────────┐
                           │            客户端层                 │
                           │  Cursor / Claude / Qoder            │
                           │  控制台 Web (Browser)                │
                           └──────────────┬──────────────────────┘
                                          │ HTTP + JWT Bearer
                           ┌──────────────▼──────────────────────┐
                           │          cbmem-team v2              │
                           │  ┌───────────────────────────────┐  │
                           │  │  Gin HTTP Server              │  │
                           │  │  ┌─────────────────────────┐  │  │
                           │  │  │ JWT 验证中间件           │  │  │
                           │  │  │ 团队上下文中间件          │  │  │
                           │  │  │ RBAC 中间件             │  │  │
                           │  │  │ 会话捕获中间件           │  │  │
                           │  │  │ 配置热加载              │  │  │
                           │  │  └─────────────────────────┘  │  │
                           │  └───────────────┬───────────────┘  │
                           │                  │                  │
                           │  ┌───────────────┴───────────────┐  │
                           │  │       路由层                  │  │
                           │  │  /mcp  │ /api/auth  │ /api/admin│ │
                           │  │  /api/teams  │ /api/projects  │ │
                           │  │  /api/modules │ /api/sessions │ │
                           │  │  /api/memories │ /api/ai-tools │ │
                           │  └───────────┬───────────────────┘  │
                           └──────────────┼──────────────────────┘
                                          │
              ┌───────────────────────────┼───────────────────────────┐
              │                           │                           │
              ▼                           ▼                           ▼
    ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
    │  Process Pool   │         │  Console DB     │         │ AI Tool Gateway │
    │ (Per-user MCP)  │         │  (MySQL/SQLite) │         │ (HTTP/stdio)    │
    └────────┬────────┘         └────────┬────────┘         └────────┬────────┘
             │                           │                           │
             ▼                           ▼                           ▼
    ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
    │ code-memory-mcp │         │  Users/Teams    │         │ AI Tools        │
    │ (per-user)      │         │  Projects       │         │ (codegen etc.)  │
    └─────────────────┘         │  Modules        │         └─────────────────┘
                                │  Sessions       │
                                │  Memories       │
                                │  AI Tools       │
                                │  Audit Logs     │
                                └─────────────────┘
```

### 8.2 核心技术栈

| 组件 | 选型 | 说明 |
|------|------|------|
| HTTP 框架 | Gin | 高性能 Go HTTP 框架 |
| 数据库 | SQLite (modernc.org/sqlite) 或 MySQL | 复用 v1 抽象层 |
| 进程管理 | exec.Cmd | 标准库 os/exec |
| 密码哈希 | bcrypt | golang.org/x/crypto/bcrypt |
| JWT | HS256 | 自实现 |
| 配置热更新 | fsnotify | 监听配置文件 |
| LLM | OpenAI / Ollama | 可插拔 |
| 前端 | VitePress + Vue 3 | 同 v1 |

### 8.3 目录结构（v2）

```
tools/cbmem-team/
├── cmd/
│   ├── cbmem-team/
│   │   ├── main.go
│   │   └── handlers.go
│   └── cbmem-mint-token/
├── internal/
│   ├── auth/                 # JWT
│   ├── console/              # 控制台 API
│   │   ├── auth_handler.go   # v2: 登录/改密
│   │   ├── users_handler.go  # admin
│   │   ├── teams_handler.go  # admin
│   │   ├── team_members_handler.go
│   │   ├── projects_handler.go
│   │   ├── modules_handler.go # v2 新增
│   │   ├── sessions_handler.go
│   │   ├── memories_handler.go
│   │   ├── memory_templates_handler.go # v2 新增
│   │   ├── ai_tools_handler.go # v2 新增
│   │   ├── ai_tool_invocations_handler.go # v2 新增
│   │   ├── summarize_handler.go
│   │   ├── distill_handler.go
│   │   ├── rbac.go           # v2 强化
│   │   ├── capture.go
│   │   ├── db.go
│   │   └── router.go
│   ├── llm/                  # LLM 集成
│   ├── mcp/                  # MCP 协议
│   ├── pool/                 # 进程池
│   ├── repos/                # Git 仓库管理（v2 增强）
│   ├── config/               # v2 新增：配置热更新
│   │   └── config.go
│   └── store/
├── deploy/
├── examples/
└── dist/
```

---

## 9. 安全机制

### 9.1 认证安全

| 机制 | 描述 |
|------|------|
| bcrypt 密码 | cost=12 |
| JWT 签名验证 | HS256 |
| Token 过期 | exp 检查 |
| Refresh Token 撤销 | 数据库维护 jti 列表 |
| 首次登录强制改密 | must_change_password 标志 |

### 9.2 访问控制（v2 强化）

| 机制 | 描述 |
|------|------|
| RBAC 角色检查 | 中间件层强制 |
| 团队归属 | 资源 team_id 校验 |
| 叶子模块约束 | sessions/memories 写入前校验 |
| AI 工具 RBAC | tool.required_role + allowed_projects |

### 9.3 数据隔离

| 机制 | 描述 |
|------|------|
| Per-user 进程 | 每个用户独立 codebase-memory-mcp |
| SQLite 文件锁 | 每个用户独立索引文件 |
| 项目路径 | 服务器端全局唯一，git clone 隔离 |

### 9.4 审计

| 机制 | 描述 |
|------|------|
| AI 工具调用审计 | ai_tool_invocations 表 |
| 登录日志 | users.last_login_at + console_sessions |
| 配置变更 | audit_logs 表 |

---

## 10. 部署指南

### 10.1 构建

```bash
cd tools/cbmem-team
go build -o cbmem-team ./cmd/cbmem-team
```

### 10.2 systemd 部署（v2）

```ini
[Unit]
Description=cbmem-team v2

[Service]
ExecStart=/usr/local/bin/cbmem-team \
  -listen :8787 \
  -data /var/lib/cbmem-team \
  -mcp-bin /usr/local/bin/codebase-memory-mcp \
  -jwt-secret /run/secrets/cbmem-jwt-secret \
  -initial-admin-username admin \
  -initial-admin-password /run/secrets/cbmem-admin-password \
  -llm-provider openai \
  -llm-model gpt-4o-mini \
  -llm-api-key /run/secrets/cbmem-llm-api-key \
  -console-dist /opt/cbmem-console/dist \
  -config /etc/cbmem-team/config.yaml \
  -config-watch true \
  -log info
Restart=always
User=cbmem
```

### 10.3 首次启动流程

1. systemd 启动 cbmem-team
2. 检测到 `-initial-admin-*` 参数，自动创建初始 admin 账号
3. 管理员通过 `/login` 登录（强制改密）
4. 管理员创建团队、邀请成员
5. 团队负责人创建项目（git clone 自动执行）
6. 团队负责人在项目中创建模块树
7. 开发者开始使用 MCP 客户端开发

### 10.4 Cursor 客户端配置（v2）

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://mcp.internal.example.com:8787/mcp?project=${workspaceFolder}",
      "headers": {
        "Authorization": "Bearer <jwt-token>"
      }
    }
  }
}
```

### 10.5 数据迁移（v1 → v2）

提供迁移脚本：
- users 表增加 username/password_hash/must_change_password 字段
- 创建 teams、team_members 表
- projects 表增加 team_id/git_url/git_branch/status
- 创建 modules、memory_templates、ai_tools、ai_tool_invocations 表
- 删除 project_paths 字段（v2 改为单一全局路径）

---

## 11. MVP 范围界定

### 11.1 MVP 功能集（一次性交付）

✅ **包含**（MVP 必须完成）：

| 模块 | 功能 |
|------|------|
| 认证 | 用户名/密码登录、首次改密、JWT、Refresh Token |
| 用户管理 | admin 创建/列出/更新/删除用户、重置密码 |
| 团队管理 | admin 创建/列出/更新/删除团队 |
| 团队成员 | 添加/移除/列出/更新成员角色 |
| 项目管理 | 创建项目（自动 git clone）、CRUD、git pull/status、索引 |
| 模块树 | 模块 CRUD、树状结构、叶子校验 |
| 会话 | 自动捕获、列表、详情、统计 |
| 内存 | 手动 CRUD、模板辅助、标签 |
| 内存模板 | 5 个内置模板（ADR/lesson/snippet/runbook/decision） |
| AI 工具 | 注册/列出/调用/审计 |
| 归纳/蒸馏 | 手动触发任务 |
| 控制台 UI | 上述所有功能的 Web 界面 |
| MCP 包装 | per-user 进程池、路由 |
| 配置热更新 | fsnotify 监听配置 |

❌ **不包含**（v1.x 后续）：

- 跨服务器 MemPalace 直接集成（通过 AI 工具即可）
- 实时通知/WebSocket 推送
- 高级统计图表（仅基础统计）
- 多语言 i18n
- 移动端 UI 优化
- 公开 API 限流（仅 admin/lead 内置限流）

### 11.2 验收标准（MVP Done）

| 维度 | 标准 |
|------|------|
| 功能完整性 | 上述所有 ✅ 项目全部通过验收测试 |
| 测试覆盖率 | 单元测试 ≥ 80%，E2E 覆盖核心流程 |
| 文档完整 | PRD、实施计划、API 文档、用户手册齐全 |
| 部署验证 | 在至少 2 个不同 OS（Linux + macOS 或 Linux + WSL）上成功部署 |
| 性能基准 | 单实例支持 50 并发用户，会话捕获延迟 < 100ms |
| 安全验证 | 通过 OWASP Top 10 基础检查 |

---

## 12. 附录

### 12.1 v1 → v2 变更摘要

| 维度 | v1 | v2 |
|------|----|----|
| 认证 | JWT + admin-token 双轨 | 用户名/密码 + 首次改密 |
| 项目归属 | 单一用户 | 团队（多对多） |
| 项目路径 | 用户白名单 | 单一全局路径 |
| 项目 git | 手动 | 创建时自动 clone |
| 模块 | 无 | 自由树状结构 |
| 会话归属 | 项目级 | 叶子模块级 |
| 内存 | 自由 | 模板辅助 + 叶子模块归属 |
| AI 工具 | 直接 LLM 调用 | 注册/调用网关模式 |
| 管理员权限 | 全部 | 仅用户/团队 |
| 实施节奏 | 分阶段 | MVP 一次性交付 |

### 12.2 相关文档

| 文档 | 路径 |
|------|------|
| 设计 Spec | `docs/superpowers/specs/2026-07-14-dual-track-memory-tools-best-practices-design.md` |
| 实施计划 | `docs/superpowers/plans/2026-07-15-cbmem-team-mvp-implementation.md` |
| 数据模型 ER | `docs/superpowers/diagrams/cbmem-team-v2-erd.md` |
| Cursor 配置 | `tools/cbmem-team/cursor-mcp-config.md` |
| 工具速查 | `docs/quick-ref/tools-quick-ref.md` |

### 12.3 术语表

| 术语 | 说明 |
|------|------|
| Team | 团队，作为项目和成员的容器 |
| Project | 项目，归属团队，绑定 git 仓库 |
| Module | 模块，项目下的树状子结构 |
| Leaf Module | 叶子模块，会话/内存的归属点 |
| Memory Template | 内存模板，用于辅助生成结构化内存 |
| AI Tool | 第三方 AI 工具，通过 cbmem-team 网关调用 |
| Hall | 内存分类（facts/events/discoveries/preferences/advice） |
| RBAC | 基于角色的访问控制 |
| must_change_password | 首次登录强制改密标志 |

---

**文档版本**: v2.0  
**最后更新**: 2026-07-14  
**下一步**: 编写 MVP 实施计划
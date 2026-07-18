# cbmem-team Console API Reference

> 双轨记忆控制台 — REST API 参考文档 (v2)
>
> 版本：v2.0.0+
>
> 最后更新：2026-07-15

## 概述

cbmem-team Console API 提供以下功能：
- **认证**：JWT-based authentication with access/refresh tokens
- **团队管理**：多租户团队与成员管理
- **项目管理**：项目 CRUD + Git 克隆 + 索引管理
- **模块管理**：层级模块树结构
- **会话管理**：MCP 会话记录查询与统计
- **记忆管理**：记忆模板 + 记忆 CRUD
- **归纳/蒸馏**：基于 LLM 的会话分析与知识提取
- **AI 工具**：工具目录 + 调用 + 审计

## 认证

### 端点

#### POST /api/auth/login

用户名密码登录，获取 access token 和 refresh token。

**请求体**：
```json
{
  "username": "admin",
  "password": "your-password"
}
```

**响应** (200):
```json
{
  "code": 0,
  "msg": "",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900,
    "user": {
      "id": "user-xxx",
      "username": "admin",
      "role": "admin",
      "must_change_password": false
    }
  }
}
```

#### POST /api/auth/refresh

使用 refresh token 获取新的 access token。

**请求头**：
```
Authorization: Bearer <refresh_token>
```

**响应** (200):
```json
{
  "code": 0,
  "msg": "",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900
  }
}
```

#### POST /api/auth/change-password

修改密码（需持有有效 access token）。

**请求体**：
```json
{
  "old_password": "old-password",
  "new_password": "new-password"
}
```

**响应** (200):
```json
{
  "code": 0,
  "msg": "密码修改成功"
}
```

#### POST /api/auth/first-admin

首次启动时创建管理员账户。

**请求体**：
```json
{
  "username": "admin",
  "password": "your-admin-password"
}
```

### 统一响应格式

所有 API 响应使用统一格式：

```json
{
  "code": 0,
  "msg": "",
  "data": {}
}
```

**错误码**：

| 区间 | 含义 |
|------|------|
| 0 | 成功 |
| 40000xx | 请求参数错误 |
| 40100xx | 鉴权失败（token 过期/无效） |
| 40300xx | 无权访问 |
| 40400xx | 资源不存在 |
| 40900xx | 资源冲突 |
| 50000xx | 系统错误 |

## Teams API (M3)

基础路径：`/api/console/v2/teams`

### 认证

所有 team API 需要 JWT Bearer Token：

```
Authorization: Bearer <access_token>
```

### 端点

#### GET /api/console/v2/teams

列出当前用户可访问的团队。

**响应** (200):
```json
{
  "code": 0,
  "data": {
    "teams": [
      {
        "id": "team-xxx",
        "name": "研发团队",
        "slug": "dev-team",
        "description": "研发团队",
        "created_at": "2026-07-15T10:00:00Z",
        "member_count": 5
      }
    ]
  }
}
```

#### POST /api/console/v2/teams

创建新团队（需 admin 角色）。

**请求体**：
```json
{
  "name": "新团队",
  "slug": "new-team",
  "description": "团队描述"
}
```

#### GET /api/console/v2/teams/:id

获取团队详情。

#### PUT /api/console/v2/teams/:id

更新团队信息（需 admin 角色）。

#### DELETE /api/console/v2/teams/:id

删除团队（需 admin 角色）。

### 成员管理

#### GET /api/console/v2/teams/:id/members

列出团队成员。

**响应** (200):
```json
{
  "code": 0,
  "data": {
    "members": [
      {
        "user_id": "user-xxx",
        "username": "alice",
        "role": "member",
        "joined_at": "2026-07-15T10:00:00Z"
      }
    ]
  }
}
```

#### POST /api/console/v2/teams/:id/members

添加成员到团队（需 admin 角色）。

**请求体**：
```json
{
  "user_id": "user-xxx",
  "role": "member"
}
```

#### PUT /api/console/v2/teams/:id/members/:uid

更新成员角色（需 admin 角色）。

**请求体**：
```json
{
  "role": "admin"
}
```

#### DELETE /api/console/v2/teams/:id/members/:uid

从团队移除成员（需 admin 角色）。

### 团队项目

#### GET /api/console/v2/teams/:id/projects

列出团队下的所有项目。

**响应** (200):
```json
{
  "code": 0,
  "data": {
    "projects": [
      {
        "id": "proj-xxx",
        "name": "ai-dev-sop",
        "slug": "ai-dev-sop",
        "description": "AI 开发 SOP 项目",
        "git_url": "https://github.com/user/ai-dev-sop.git",
        "git_branch": "main",
        "status": "active",
        "created_at": "2026-07-15T10:00:00Z"
      }
    ]
  }
}
```

#### POST /api/console/v2/teams/:id/projects

在团队下创建新项目（需 admin 角色）。

**请求体**：
```json
{
  "name": "新项目",
  "slug": "new-project",
  "description": "项目描述",
  "git_url": "https://github.com/user/repo.git"
}
```

## Projects API (M3)

基础路径：`/api/console/v2/projects`

### 端点

#### GET /api/console/v2/projects/:pid

获取项目详情。

**响应** (200):
```json
{
  "code": 0,
  "data": {
    "id": "proj-xxx",
    "team_id": "team-xxx",
    "name": "ai-dev-sop",
    "slug": "ai-dev-sop",
    "description": "AI 开发 SOP 项目",
    "git_url": "https://github.com/user/ai-dev-sop.git",
    "git_branch": "main",
    "git_commit_sha": "abc1234",
    "status": "active",
    "index_status": {
      "status": "indexed",
      "indexed_at": "2026-07-15T12:00:00Z",
      "file_count": 1500
    },
    "created_at": "2026-07-15T10:00:00Z"
  }
}
```

#### PUT /api/console/v2/projects/:pid

更新项目信息。

#### DELETE /api/console/v2/projects/:pid

删除项目（软删除）。

#### POST /api/console/v2/projects/:pid/clone

克隆项目仓库。

#### GET /api/console/v2/projects/:pid/index-status

获取项目索引状态。

**响应** (200):
```json
{
  "code": 0,
  "data": {
    "status": "indexed",
    "indexed_at": "2026-07-15T12:00:00Z",
    "file_count": 1500,
    "last_error": ""
  }
}
```

#### POST /api/console/v2/projects/:pid/reindex

触发项目重新索引。

**响应** (202):
```json
{
  "code": 0,
  "msg": "reindex started"
}
```

## Modules API (M4)

基础路径：`/api/console/v2/modules`

### 端点

#### GET /api/console/v2/projects/:pid/modules

列出项目下的模块。

**响应** (200):
```json
{
  "code": 0,
  "data": {
    "modules": [
      {
        "id": "mod-xxx",
        "project_id": "proj-xxx",
        "parent_id": null,
        "name": "后端服务",
        "path": "/backend",
        "is_leaf": false,
        "created_at": "2026-07-15T10:00:00Z"
      }
    ]
  }
}
```

#### GET /api/console/v2/projects/:pid/modules/tree

获取模块树结构。

**响应** (200):
```json
{
  "code": 0,
  "data": {
    "tree": [
      {
        "id": "mod-1",
        "name": "后端服务",
        "path": "/backend",
        "is_leaf": false,
        "children": [
          {
            "id": "mod-2",
            "name": "API",
            "path": "/backend/api",
            "is_leaf": true,
            "children": []
          }
        ]
      }
    ]
  }
}
```

#### POST /api/console/v2/projects/:pid/modules

创建模块。

**请求体**：
```json
{
  "name": "新模块",
  "parent_id": "mod-xxx"
}
```

#### GET /api/console/v2/modules/:id

获取模块详情。

#### PUT /api/console/v2/modules/:id

更新模块。

#### DELETE /api/console/v2/modules/:id

删除模块。

#### POST /api/console/v2/modules/:id/move

移动模块到新的父模块。

**请求体**：
```json
{
  "new_parent_id": "mod-yyy"
}
```

## Sessions API (M4)

基础路径：`/api/console/v2/sessions`

### 端点

#### GET /api/console/v2/sessions

列出会话记录（支持分页和筛选）。

**查询参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| team_id | string | 按团队筛选 |
| project_id | string | 按项目筛选 |
| module_id | string | 按模块筛选 |
| user_id | string | 按用户筛选 |
| from | string | 开始时间 (ISO 8601) |
| to | string | 结束时间 (ISO 8601) |
| limit | int | 每页数量 (默认 50) |
| offset | int | 偏移量 |

**响应** (200):
```json
{
  "code": 0,
  "data": {
    "sessions": [
      {
        "id": "sess-xxx",
        "user_id": "user-xxx",
        "project_id": "proj-xxx",
        "module_id": "mod-xxx",
        "tool_count": 25,
        "turn_count": 10,
        "started_at": "2026-07-15T10:00:00Z",
        "ended_at": "2026-07-15T11:00:00Z"
      }
    ],
    "total": 100,
    "limit": 50,
    "offset": 0
  }
}
```

#### GET /api/console/v2/sessions/stats

获取会话统计。

**响应** (200):
```json
{
  "code": 0,
  "data": {
    "total_sessions": 1000,
    "total_turns": 5000,
    "total_tools": 15000,
    "by_project": [
      {"project_id": "proj-1", "count": 500}
    ],
    "by_user": [
      {"user_id": "user-1", "count": 200}
    ]
  }
}
```

#### GET /api/console/v2/sessions/:id

获取会话详情（含所有 turns）。

## Memory Templates API (M5)

基础路径：`/api/console/v2/memory-templates`

### 端点

#### GET /api/console/v2/memory-templates

列出记忆模板。

#### GET /api/console/v2/memory-templates/:id

获取模板详情。

#### POST /api/console/v2/memory-templates/:id/render

渲染模板。

**请求体**：
```json
{
  "session_ids": ["sess-1", "sess-2"],
  "variables": {
    "project_name": "AI-Dev-SOP"
  }
}
```

#### POST /api/console/v2/memory-templates

创建模板（需 admin 角色）。

## Memories API (M5)

基础路径：`/api/console/v2/memories`

### 端点

#### GET /api/console/v2/memories

列出记忆。

**查询参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| team_id | string | 按团队筛选 |
| module_id | string | 按模块筛选 |
| user_id | string | 按用户筛选 |
| tag | string | 按标签筛选 |
| q | string | 全文搜索 |
| limit | int | 每页数量 |
| offset | int | 偏移量 |

#### POST /api/console/v2/memories

创建记忆。

**请求体**：
```json
{
  "team_id": "team-xxx",
  "module_id": "mod-xxx",
  "content": "这是记忆内容",
  "tags": ["经验", "技术方案"]
}
```

#### GET /api/console/v2/memories/:id

获取记忆详情。

#### PUT /api/console/v2/memories/:id

更新记忆。

#### DELETE /api/console/v2/memories/:id

删除记忆。

#### POST /api/console/v2/memories/:id/tags

添加标签。

**请求体**：
```json
{
  "tag": "新标签"
}
```

## Summarize Tasks API (M5)

基础路径：`/api/console/v2/summarize-tasks`

### 端点

#### GET /api/console/v2/summarize-tasks

列出归纳任务。

#### POST /api/console/v2/summarize-tasks

创建归纳任务。

**请求体**：
```json
{
  "session_ids": ["sess-1", "sess-2"],
  "depth": "deep",
  "target_wing": "my-wing"
}
```

#### GET /api/console/v2/summarize-tasks/:id

获取任务状态和结果。

## Distill Tasks API (M5)

基础路径：`/api/console/v2/distill-tasks`

### 端点

#### GET /api/console/v2/distill-tasks

列出蒸馏任务。

#### POST /api/console/v2/distill-tasks

创建蒸馏任务。

**请求体**：
```json
{
  "session_ids": ["sess-1", "sess-2"],
  "rules": {
    "min_value_score": 0.7
  },
  "target_wing": "my-wing"
}
```

#### GET /api/console/v2/distill-tasks/:id

获取任务状态和结果。

## AI Tools API (M6)

基础路径：`/api/console/v2/ai-tools`

### 端点

#### GET /api/console/v2/ai-tools

列出 AI 工具。

#### POST /api/console/v2/ai-tools

注册新 AI 工具（需 admin 角色）。

**请求体**：
```json
{
  "name": "我的工具",
  "provider": "cursor",
  "endpoint": "https://api.example.com",
  "api_key": "sk-xxx",
  "description": "工具描述",
  "config_json": {}
}
```

#### GET /api/console/v2/ai-tools/:id

获取工具详情。

#### PUT /api/console/v2/ai-tools/:id

更新工具配置。

#### DELETE /api/console/v2/ai-tools/:id

删除工具。

#### POST /api/console/v2/ai-tools/:id/invoke

调用工具。

**请求体**：
```json
{
  "parameters": {
    "query": "搜索代码"
  }
}
```

#### GET /api/console/v2/ai-tools/:id/invocations

列出工具调用历史。

#### GET /api/console/v2/ai-tools/invocations/:inv_id

获取单次调用详情。

## Dashboard API

基础路径：`/api/console/v2/dashboard`

### 端点

#### GET /api/console/v2/dashboard/summary

获取仪表盘摘要。

**响应** (200):
```json
{
  "code": 0,
  "data": {
    "total_teams": 5,
    "total_projects": 20,
    "total_sessions": 1000,
    "total_tools": 50,
    "recent_activity": []
  }
}
```

#### GET /api/console/v2/dashboard/high-risk

获取高风险会话列表。

#### GET /api/console/v2/dashboard/high-risk/summary

获取高风险统计摘要。

## 速率限制

### 端点

#### GET /api/console/v2/rate-limits

列出所有工具的速率限制配置。

#### GET /api/console/v2/tools/:id/rate-limit

获取单个工具的速率限制。

#### PUT /api/console/v2/tools/:id/rate-limit

更新工具速率限制。

**请求体**：
```json
{
  "qps_per_user": 10,
  "qpm_per_user": 100,
  "concurrency_global": 20
}
```

#### DELETE /api/console/v2/tools/:id/rate-limit

重置速率限制为默认值。

#### POST /api/console/v2/tools/:id/rate-limit/reset

强制清除内存中的速率限制状态。

## 统一响应格式

所有 API 响应使用以下格式：

```json
{
  "code": 0,
  "msg": "",
  "data": {}
}
```

### HTTP 状态码

| 状态码 | 含义 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 202 | 接受/异步处理中 |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 无权访问 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 429 | 速率限制 |
| 500 | 服务器错误 |

### 错误码详解

| code | 说明 |
|------|------|
| 0 | 成功 |
| 4000001 | 参数缺失 |
| 4000002 | 参数格式错误 |
| 4010001 | Token 已过期 |
| 4010002 | Token 无效 |
| 4010003 | Refresh Token 无效 |
| 4030001 | 权限不足 |
| 4030002 | 用户已被禁用 |
| 4040001 | 团队不存在 |
| 4040002 | 项目不存在 |
| 4040003 | 模块不存在 |
| 4090001 | 团队 slug 重复 |
| 4090002 | 项目 slug 重复 |
| 5000001 | 数据库错误 |
| 5000002 | LLM 调用失败 |
| 5000003 | 服务未配置 |

# Mem0 多租户支持设计（userId × projectId 记忆隔离）

- 日期：2026-09-15
- 状态：已确认（用户批准）
- 范围：`mem0/server`（后端 + MCP）与 `mem0/server/dashboard`（Next.js 前端）
- 分支：`feat/mem0-llm-provider-i18n`（外层仓库存放本设计文档；`mem0` 为子模块）

## 1. 背景与目标

现有 `mem0/server` 是深度定制的 Mem0 OSS 服务端（FastAPI + SQLAlchemy/alembic + JWT/个人 API Key 认证 + Next.js dashboard）。目标是在其上实现多租户：

1. 记忆按 **userId + projectId** 双维度隔离；`projectId` 写入记忆 `metadata`。
2. 用户管理：每个用户独立 API Key（复用既有 api-keys 体系）。
3. 部门管理：一对多（一个部门 → 多个用户）。
4. 项目管理：项目 ID + 多个 Git 远程地址（`git remote -v` 输出）。
5. 开发工具（Qoder/CodeBuddy 等）经 MCP 传入用户 API Key 与本地工程 Git 地址。
6. 服务端按 Git 地址匹配项目 ID 并写入记忆 metadata。
7. Dashboard 提供登录与 Users / Departments / Projects / Memories 的增删改查与按项目/人员检索。

### 已确认的关键决策

| 决策点 | 结论 |
| --- | --- |
| 隔离强度 | **强隔离**：普通用户（API Key/JWT）强制 `user_id = 当前认证用户`；传他人 user_id → 403。管理员/ADMIN_API_KEY/AUTH_DISABLED 可跨用户。 |
| Git 未匹配 | **只匹配不创建**：返回 `project_id=null`，记忆照常入库（保留 `git_remote` 元数据）。 |
| MCP 身份传递 | **工具参数**：MCP 工具带 `api_key` / `git_remote` 入参，服务端以 `X-API-Key` 转发给 mem0-api。 |
| Dashboard 形态 | **功能完整、风格简洁**：与现有 Tailwind 风格一致的列表+表单 CRUD；不做 shadcn 弹窗重构。 |

## 2. 方案选型（project 维度如何进入记忆并可检索）

- **方案 A（采用）：metadata 注入 + 应用层后过滤**。`project_id` 写入 `metadata`；检索先按顶层可过滤的 `user_id` 圈定（Mem0 pgvector 仅支持 `payload->>顶层键` 过滤，已在容器内验证），再在应用层按 `metadata.project_id` 后过滤。零侵入 mem0 核心、升级安全；单用户记忆量级下后过滤开销可忽略。
- 方案 B（否决）：fork mem0 将 `project_id` 提升为顶层 payload 字段，SQL 级过滤最优但需长期维护 fork。
- 方案 C（否决）：独立租户网关（BFF），边界清晰但多一跳部署复杂度，YAGNI。

## 3. 数据模型（`models.py`）

新表经启动时 `Base.metadata.create_all(engine)` 补建（仅建缺失表，不影响 alembic 已管表）：

```
departments(id PK uuid, name unique, description, created_at)
users      (+ department_id FK → departments.id ON DELETE SET NULL, nullable, indexed;
              relationship: User.department / Department.users)
projects   (id PK uuid, project_id unique indexed, name, description,
              git_remotes JSON(list[str]), created_by FK → users.id, created_at, updated_at)
```

## 4. 认证与隔离规则

- 认证复用现状：dashboard 走 JWT（`/auth/login`）；程序化/MCP 走个人 API Key（`X-API-Key`）。
- 新增 `routers/users.py`（admin only）：`GET /users`、`POST /users`（name/email/password/role∈{admin,user}，密码 ≥8 位）、`DELETE /users/{id}`（禁止删自己）。
- 隔离助手 `_scope_identifiers(user, user_id, agent_id, run_id)`：
  - `user is None`（ADMIN_API_KEY / AUTH_DISABLED）或 `role == "admin"` → 透传客户端标识；
  - 普通用户 → 返回 `str(user.id)`，若客户端 user_id 与之不符 → **403**；agent_id/run_id 仅管理员可用。
- 应用于 `POST /memories`、`GET /memories`、`POST /search`；普通用户“列出全部”不允许（其 user_id 已被强制填入）。

## 5. Git 匹配（`gitmatch.py`）

- 归一化 `normalize_git_remote(url) → "host/org/repo"`：支持 scp 形式 `git@host:org/repo.git`、`scheme://[user:pass@]host[:port]/org/repo(.git)`；去协议、凭据、端口、`.git` 后缀、大小写不敏感；无法解析返回原串小写或空。
- `resolve_project_id(git_remote: str|list, db) → str|None`：对传入地址（支持多个 remote）归一化后与所有 `projects.git_remotes` 匹配，返回首个命中项目的 `project_id`。
- `POST /projects/match`（body `{git_remote: str|list}`）与 `GET /projects/match?git_remote=…` → `{project_id: str|null}`。**只匹配不创建**。

## 6. 记忆读写

- `POST /memories` 新增字段：`project_id?`、`git_remote?: str|list`。
  - 解析优先级：显式 `project_id` 优先；同时给出且与 git 解析结果冲突 → **400**。
  - 写入 metadata：`project_id`（若有）、`git_remote`（若有）、`department_id`（普通用户且其有部门时自动附带）。
- `GET /memories`、`POST /search` 新增 `project_id`：先按 user_id 检索（Mem0 顶层过滤）→ `_filter_by_project_id` 应用层后过滤（比对 `metadata.project_id`）。
- 返回体沿用现有 `_serialize_memory`，metadata 随结果透出。

## 7. MCP（`mcp_server.py`，同容器 supervisord 双进程）

- 工具：`add_memory(text, api_key, git_remote?, project_id?, metadata?, user_id?)`、`search_memories(query, api_key, project_id?, top_k)`、`get_memories(api_key, project_id?, limit)`、`get_memory / update_memory / delete_memory(memory_id, api_key?)`、`match_project(git_remote, api_key?)`。
- 鉴权：`api_key` → `X-API-Key` 转发；未传时回退 `MEM0_API_KEY`（管理员 Bearer）。user_id 隔离与项目解析全部由 mem0-api 完成。
- 依赖：官方 `mcp` Python SDK（fastmcp），pin `<2` 以兼容 starlette/fastapi；以 `streamable-http` 暴露，监听 `MEM0_MCP_PORT`（默认 **8080**，compose 已映射 `8080:8080`）。
- 开发工具配置示例（CodeBuddy/Qoder 的 MCP 配置）：工具入参 `api_key` 填用户个人 API Key，`git_remote` 填 `git remote -v` 输出的地址（支持多个）。

## 8. Dashboard（简洁完整版）

- 新增 TENANT 导航组（`main-nav.tsx`）：Users、Departments、Projects（`nav.users/departments/projects` i18n 键，zh/en 双语）。
- 页面（列表 + 内联新建/编辑表单 + 删除确认）：
  - `users/page.tsx`：name/email/password/role 创建；展示 department 归属；删除。
  - `departments/page.tsx`：name/description CRUD；展示成员数。
  - `projects/page.tsx`：project_id/name/description/git_remotes（多行文本，每行一个地址）CRUD；「测试 Git 地址匹配」面板调 `/projects/match`。
- `memories/page.tsx` 增强：项目下拉过滤（来自 projects 列表）+ Project 列；`Memory` 类型增 `project_id?`、`metadata?` 可选字段。
- 端点常量（`api-endpoints.ts`）：`USER_ENDPOINTS`、`DEPARTMENT_ENDPOINTS`、`PROJECT_ENDPOINTS{BASE,BY_ID,MATCH}`；类型（`types/api.ts`）：`Department/DepartmentMember/Project/DashboardUser`。

## 9. 错误处理

- 403：普通用户访问他人 user_id / 使用 agent_id·run_id / 非管理员访问 admin 资源。
- 400：`project_id` 与 git 解析冲突；参数缺失（无任何标识符）；密码过短；非法 user_ids。
- 409：部门重名、project_id 重复、邮箱已存在。
- 404：资源不存在；`/projects/match` 未命中不报错，返回 `null`。

## 10. 测试与验证

- 新增 pytest：gitmatch 归一化/匹配、departments & projects CRUD、`/projects/match`、`/memories` 强隔离（普通用户伪造 user_id → 403）、metadata 注入（project_id/department_id/git_remote）。
- 回归：既有 `tests/test_configure.py` 继续通过。
- 验证方式：`docker compose -f deploy/mem0/docker-compose.yaml up -d --build` 重建 `mem0-api`（含 MCP 进程），容器内 `python -m pytest` 全量通过；宿主机 smoke：登录 dashboard、建项目登记 git 地址、`/projects/match` 命中、MCP add/search 带 `api_key+git_remote`。

## 11. 非目标（本轮不做）

- 部门/项目级记忆共享池（普通用户只见自己的记忆）。
- shadcn 弹窗式交互重构、分页/搜索高级功能。
- fork mem0 实现顶层 project 过滤（方案 B）。
- git 未匹配自动建项目。

> [!WARNING]
> **已废弃（2026-09-18）**：本文档描述的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，记忆功能统一替换为自托管 mem0（见 docs/quick-ref/mem0-ai-tools-config-guide.md）。本文仅作历史归档保留，内容不再维护。

# cbmem-team MVP 一次性完整交付实施计划

> **基于**: [cbmem-team-prd.md v2.0](../../tools/cbmem-team/cbmem-team-prd.md)
> **版本**: v1.0
> **计划日期**: 2026-07-15
> **执行日期**: 2026-07-15 → 2026-08-10（约 4 周）
> **状态**: 📋 计划待执行
> **范围**: MVP 一次性完整交付（不包含分阶段里程碑）

---

## 1. 实施范围与目标

### 1.1 核心目标

基于 PRD v2 一次性交付完整的 MVP（Minimum Viable Product），覆盖：

1. **认证系统**：用户名/密码 + 首次改密 + JWT
2. **组织模型**：用户/团队/团队成员三层关系
3. **项目模型**：自动 git clone + 单一全局路径
4. **模块树**：自由树状结构 + 叶子校验
5. **会话/内存**：严格归属叶子模块 + 模板辅助
6. **AI 工具**：注册/调用/审计网关
7. **控制台 UI**：完整的 Web 管理界面
8. **配置热更新**：fsnotify 监听配置

### 1.2 实施范围

```
┌─────────────────────────────────────────────────────────────────┐
│                      MVP 实施范围 (一次性)                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  M1: 数据库迁移 + 认证重构                                       │
│      ├── users 表增加 username/password_hash/must_change_password│
│      ├── 新建 teams 表                                           │
│      ├── 新建 team_members 表                                    │
│      ├── 新建 modules 表                                         │
│      ├── 新建 memory_templates 表                                │
│      ├── 新建 ai_tools 表                                        │
│      ├── 新建 ai_tool_invocations 表                             │
│      └── projects 表重构（增加 team_id/git_url/status）          │
│                                                                  │
│  M2: 认证 + 用户/团队管理                                        │
│      ├── /api/auth/login (用户名/密码)                            │
│      ├── /api/auth/change-password (强制改密)                    │
│      ├── /api/auth/refresh                                       │
│      ├── /api/auth/me                                            │
│      ├── /api/admin/users CRUD + reset-password                  │
│      └── /api/admin/teams CRUD                                   │
│                                                                  │
│  M3: 团队成员 + 项目管理                                         │
│      ├── /api/teams/:id/members CRUD                              │
│      ├── /api/projects CRUD（含 git clone 自动触发）             │
│      ├── /api/projects/:id/git-status / git-pull                 │
│      ├── /api/projects/:id/index-status / reindex                │
│      └── projects.path 唯一约束                                  │
│                                                                  │
│  M4: 模块树 + 会话                                               │
│      ├── /api/modules CRUD + tree                                │
│      ├── /api/modules/:id/move                                   │
│      ├── 叶子模块校验中间件                                       │
│      ├── /api/sessions 列表/详情/统计                             │
│      └── capture.go 改造（sessions.module_id NOT NULL）          │
│                                                                  │
│  M5: 内存 + 模板 + 归纳/蒸馏                                     │
│      ├── /api/memories CRUD（含叶子校验）                         │
│      ├── /api/memory-templates (内置 5 个模板)                    │
│      ├── /api/summarize-tasks / distill-tasks                     │
│      └── 模板渲染辅助                                            │
│                                                                  │
│  M6: AI 工具网关                                                 │
│      ├── /api/ai-tools CRUD                                      │
│      ├── /api/ai-tools/:id/invoke (网关调用)                     │
│      ├── /api/ai-tool-invocations 审计                            │
│      ├── AI Tool Gateway (HTTP + stdio 协议)                     │
│      └── RBAC 检查中间件                                         │
│                                                                  │
│  M7: 控制台 UI 重构                                              │
│      ├── /login (用户名/密码)                                     │
│      ├── /change-password (强制改密)                             │
│      ├── /admin/users + /admin/teams                             │
│      ├── /projects (列表/详情)                                    │
│      ├── /modules (树状展示)                                      │
│      ├── /sessions + /memories                                   │
│      ├── /ai-tools (列表/调用)                                    │
│      └── /admin/stats                                            │
│                                                                  │
│  M8: 配置热更新 + MCP 包装 + 部署                                 │
│      ├── fsnotify 监听配置文件                                    │
│      ├── 配置变更审计                                            │
│      ├── MCP 包装器适配新用户模型                                 │
│      ├── 部署 systemd 单元 + Caddy 配置                          │
│      └── 首次启动流程（创建初始 admin）                            │
│                                                                  │
│  M9: 测试 + 文档 + 验收                                          │
│      ├── 单元测试（覆盖率 ≥ 80%）                                │
│      ├── E2E 测试（核心流程）                                     │
│      ├── API 文档自动生成                                         │
│      ├── 用户手册                                                 │
│      └── 部署验证（Linux + WSL/macOS）                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.3 成功标准

| 维度 | 指标 | 目标值 |
|------|------|--------|
| 功能完整 | PRD v2 全部 ✅ 项目交付 | 100% |
| 测试覆盖 | 单元测试覆盖率 | ≥ 80% |
| E2E 覆盖 | 核心用户流程 | ≥ 90% |
| 部署验证 | 跨 OS 部署 | ≥ 2 OS（Linux + macOS/WSL） |
| 性能 | 50 并发用户会话捕获 | < 100ms |
| 安全 | OWASP Top 10 基础 | 全通过 |
| 文档 | API 文档 + 用户手册 + 运维手册 | 齐全 |
| Lint | golangci-lint | 0 errors |

---

## 2. 总体时间规划

```
Week 1 (Day 1-5):   M1 + M2 (数据迁移 + 认证)
Week 2 (Day 6-10):  M3 + M4 (项目 + 模块 + 会话)
Week 3 (Day 11-15): M5 + M6 (内存 + AI 工具)
Week 4 (Day 16-20): M7 + M8 + M9 (UI + 配置 + 部署 + 测试)
```

**总工作量**：约 20 个工作日（4 周），单人全职或 2 人协作。

---

## 3. 详细任务分解

### 3.1 M1: 数据库迁移 (Day 1-2)

#### 任务清单

| Task ID | 任务名称 | 描述 | 验收 | 估时 |
|---------|----------|------|------|------|
| T1.1 | 编写 schema 迁移脚本 | SQL 迁移 v1 → v2 schema | 迁移可重复执行 | 4h |
| T1.2 | 编写 schema 初始化脚本 | 全新部署时创建 v2 schema | 新部署验证通过 | 2h |
| T1.3 | users 表改造 | 增加 username/password_hash/must_change_password | DDL + 测试 | 3h |
| T1.4 | teams 表新建 | teams + team_members DDL | DDL + 测试 | 2h |
| T1.5 | modules 表新建 | modules + 索引 | DDL + 测试 | 2h |
| T1.6 | memory_templates 表新建 | templates + 内置数据 | DDL + seed | 2h |
| T1.7 | ai_tools 表新建 | tools + invocations | DDL + 测试 | 2h |
| T1.8 | projects 表改造 | 增加 team_id/git_url/status，删除 project_paths | DDL + 数据迁移 | 4h |
| T1.9 | 迁移测试 | 准备测试 DB + 数据 + 验证迁移 | 测试通过 | 4h |

#### 产出文件

- `internal/console/db_migrate.go`：迁移逻辑
- `internal/console/db.go`：更新 schema
- `tools/cbmem-team/deploy/sql/v2_schema.sql`：新建 schema
- `tools/cbmem-team/deploy/sql/v1_to_v2_migrate.sql`：迁移 SQL

#### 验收标准

- [ ] 新部署一键创建 v2 schema
- [ ] v1 数据迁移到 v2 不丢失
- [ ] 迁移可重复执行（幂等）
- [ ] 所有外键约束正确

---

### 3.2 M2: 认证 + 用户/团队管理 (Day 3-5)

#### 任务清单

| Task ID | 任务名称 | 描述 | 验收 | 估时 |
|---------|----------|------|------|------|
| T2.1 | bcrypt 密码哈希工具 | 封装密码 hash/verify | 单元测试通过 | 2h |
| T2.2 | /api/auth/login | 用户名/密码登录 | 集成测试通过 | 4h |
| T2.3 | /api/auth/change-password | 修改密码（含首次强制） | 集成测试通过 | 3h |
| T2.4 | /api/auth/refresh | refresh token 轮换 | 集成测试通过 | 3h |
| T2.5 | /api/auth/logout | 撤销 refresh token | 集成测试通过 | 2h |
| T2.6 | /api/auth/me | 当前用户信息 | 集成测试通过 | 1h |
| T2.7 | JWT Claims 扩展 | 增加 username/role/team_id/jti | 单测通过 | 2h |
| T2.8 | refresh token 表 | 存储 jti + 过期 | DDL + 清理 goroutine | 3h |
| T2.9 | /api/admin/users CRUD | admin 端点 | 集成测试通过 | 4h |
| T2.10 | /api/admin/users/:id/reset-password | 重置密码 | 集成测试通过 | 2h |
| T2.11 | /api/admin/teams CRUD | admin 端点 | 集成测试通过 | 4h |
| T2.12 | RBAC 中间件 | admin 端点强制 admin 角色 | 单测通过 | 3h |
| T2.13 | 初始 admin 创建 | 首次启动创建初始账号 | E2E 通过 | 2h |

#### 产出文件

- `internal/auth/password.go`：bcrypt 封装
- `internal/auth/jwt.go`：JWT 扩展
- `internal/console/auth_handler.go`：登录/改密/刷新
- `internal/console/users_handler.go`：admin 用户 CRUD
- `internal/console/teams_handler.go`：admin 团队 CRUD
- `internal/console/rbac.go`：RBAC 中间件

#### 验收标准

- [ ] 登录返回 access_token + refresh_token
- [ ] 首次登录强制改密后能正常使用
- [ ] refresh token 轮换正常，旧 token 失效
- [ ] admin 端点拒绝非 admin 角色
- [ ] bcrypt 密码 cost=12

---

### 3.3 M3: 团队成员 + 项目管理 (Day 6-8)

#### 任务清单

| Task ID | 任务名称 | 描述 | 验收 | 估时 |
|---------|----------|------|------|------|
| T3.1 | /api/teams/:id/members GET | 列出成员 | 集成测试通过 | 2h |
| T3.2 | /api/teams/:id/members POST | 添加成员 | 集成测试通过 | 2h |
| T3.3 | /api/teams/:id/members/:uid DELETE | 移除成员 | 集成测试通过 | 2h |
| T3.4 | /api/teams/:id/members/:uid PUT | 更新角色 | 集成测试通过 | 2h |
| T3.5 | /api/projects GET | 列表（按 team 过滤） | 集成测试通过 | 3h |
| T3.6 | /api/teams/:team_id/projects POST | 创建项目 | 集成测试通过 | 4h |
| T3.7 | git clone 异步执行 | 后台 goroutine + 状态机 | E2E 通过 | 6h |
| T3.8 | /api/projects/:id GET | 项目详情（含 git 状态） | 集成测试通过 | 2h |
| T3.9 | /api/projects/:id PUT | 更新项目 | 集成测试通过 | 2h |
| T3.10 | /api/projects/:id DELETE | 软删除（级联） | 集成测试通过 | 3h |
| T3.11 | /api/projects/:id/git-status | git status | 集成测试通过 | 2h |
| T3.12 | /api/projects/:id/git-pull | git pull | 集成测试通过 | 2h |
| T3.13 | /api/projects/:id/index-status | AST 索引状态 | 集成测试通过 | 2h |
| T3.14 | /api/projects/:id/reindex | 重新索引 | 集成测试通过 | 3h |
| T3.15 | 团队上下文中间件 | 从 JWT 提取 team_id | 单测通过 | 2h |

#### 产出文件

- `internal/console/team_members_handler.go`
- `internal/console/projects_handler.go`（重构）
- `internal/repos/git_clone.go`：git clone 异步任务
- `internal/repos/git_ops.go`：git status/pull

#### 验收标准

- [ ] 创建项目后 5 秒内 git clone 完成（小项目）
- [ ] 大项目 git clone 不阻塞 API（异步）
- [ ] 项目 path 全局唯一
- [ ] 软删除项目级联清理模块、会话、内存
- [ ] 团队成员只有 lead 能管理

---

### 3.4 M4: 模块树 + 会话 (Day 9-10)

#### 任务清单

| Task ID | 任务名称 | 描述 | 验收 | 估时 |
|---------|----------|------|------|------|
| T4.1 | /api/projects/:id/modules POST | 创建模块 | 集成测试通过 | 3h |
| T4.2 | /api/projects/:id/modules/tree GET | 模块树 | 集成测试通过 | 4h |
| T4.3 | /api/modules/:id GET | 模块详情 | 集成测试通过 | 2h |
| T4.4 | /api/modules/:id PUT | 更新模块 | 集成测试通过 | 2h |
| T4.5 | /api/modules/:id DELETE | 删除模块（子树策略） | 集成测试通过 | 3h |
| T4.6 | /api/modules/:id/move POST | 移动模块（防环） | 集成测试通过 | 4h |
| T4.7 | 叶子模块自动维护 | is_leaf 字段在 CRUD 时自动更新 | 单测通过 | 4h |
| T4.8 | 叶子校验中间件 | sessions/memories 写入前校验 | 单测通过 | 3h |
| T4.9 | capture.go 改造 | sessions.module_id NOT NULL | E2E 通过 | 4h |
| T4.10 | 会话复用策略调整 | 按 (user, project, module) 复用 | 集成测试通过 | 3h |
| T4.11 | /api/sessions GET | 列表（按 user/team/project/module 过滤） | 集成测试通过 | 3h |
| T4.12 | /api/sessions/:id GET | 会话详情 | 集成测试通过 | 2h |
| T4.13 | /api/sessions/stats GET | 统计 | 集成测试通过 | 3h |

#### 产出文件

- `internal/console/modules_handler.go`
- `internal/console/modules_tree.go`：树查询
- `internal/console/leaf_validator.go`：叶子校验
- `internal/console/capture.go`（重构）

#### 验收标准

- [ ] 模块树正确处理嵌套层级（≥ 5 层）
- [ ] 移动模块时防循环引用
- [ ] 删除中间节点可选择级联或阻止
- [ ] 会话只能归属叶子模块
- [ ] is_leaf 字段在子节点变化时自动更新

---

### 3.5 M5: 内存 + 模板 + 归纳/蒸馏 (Day 11-12)

#### 任务清单

| Task ID | 任务名称 | 描述 | 验收 | 估时 |
|---------|----------|------|------|------|
| T5.1 | 内置 5 个模板 seed | ADR/lesson/snippet/runbook/decision | seed 脚本 | 3h |
| T5.2 | /api/memory-templates GET | 列出模板 | 集成测试通过 | 2h |
| T5.3 | /api/memory-templates/:id GET | 模板详情 | 集成测试通过 | 2h |
| T5.4 | /api/memory-templates/:id/render POST | 渲染模板 | 集成测试通过 | 3h |
| T5.5 | /api/modules/:id/memories POST | 创建内存（含叶子校验） | 集成测试通过 | 3h |
| T5.6 | /api/memories GET | 列表（按模块过滤） | 集成测试通过 | 2h |
| T5.7 | /api/memories/:id GET | 详情 | 集成测试通过 | 1h |
| T5.8 | /api/memories/:id PUT | 更新 | 集成测试通过 | 2h |
| T5.9 | /api/memories/:id DELETE | 删除 | 集成测试通过 | 1h |
| T5.10 | /api/memories/:id/tags POST | 加标签 | 集成测试通过 | 2h |
| T5.11 | /api/summarize-tasks POST | 触发归纳 | 集成测试通过 | 3h |
| T5.12 | /api/summarize-tasks/:id GET | 任务状态 | 集成测试通过 | 2h |
| T5.13 | /api/distill-tasks POST | 触发蒸馏 | 集成测试通过 | 3h |
| T5.14 | /api/distill-tasks/:id GET | 任务状态 | 集成测试通过 | 2h |
| T5.15 | LLM 任务 worker | 异步任务执行 | E2E 通过 | 4h |

#### 产出文件

- `internal/console/memory_templates_handler.go`
- `internal/console/memories_handler.go`
- `internal/console/summarize_handler.go`（适配）
- `internal/console/distill_handler.go`（适配）
- `internal/console/memory_templates_seed.go`：内置模板数据

#### 验收标准

- [ ] 5 个内置模板可列出/渲染
- [ ] 内存创建强制叶子校验
- [ ] 模板渲染支持变量替换
- [ ] 归纳/蒸馏任务异步执行

---

### 3.6 M6: AI 工具网关 (Day 13-14)

#### 任务清单

| Task ID | 任务名称 | 描述 | 验收 | 估时 |
|---------|----------|------|------|------|
| T6.1 | AI Tool 数据模型 | ai_tools + ai_tool_invocations | DDL | 2h |
| T6.2 | /api/teams/:id/ai-tools POST | 注册工具 | 集成测试通过 | 3h |
| T6.3 | /api/teams/:id/ai-tools GET | 列出工具 | 集成测试通过 | 2h |
| T6.4 | /api/ai-tools/:id GET | 详情 | 集成测试通过 | 2h |
| T6.5 | /api/ai-tools/:id PUT | 更新 | 集成测试通过 | 2h |
| T6.6 | /api/ai-tools/:id/disable | 禁用 | 集成测试通过 | 2h |
| T6.7 | /api/ai-tools/:id/invoke POST | 调用工具 | E2E 通过 | 6h |
| T6.8 | HTTP 协议适配 | 转发 HTTP 请求到上游 | 单测通过 | 4h |
| T6.9 | stdio 协议适配 | 通过 stdio 调用命令 | 单测通过 | 4h |
| T6.10 | 调用审计 | 写入 invocations 表 | 集成测试通过 | 3h |
| T6.11 | RBAC 检查 | role + allowed_projects | 单测通过 | 3h |
| T6.12 | /api/ai-tools/:id/invocations GET | 调用历史 | 集成测试通过 | 2h |
| T6.13 | /api/ai-tool-invocations/:id GET | 单次状态 | 集成测试通过 | 2h |
| T6.14 | 超时控制 | timeout_seconds + 取消 | 单测通过 | 3h |

#### 产出文件

- `internal/console/ai_tools_handler.go`
- `internal/console/ai_tool_invocations_handler.go`
- `internal/ai_gateway/http.go`：HTTP 协议
- `internal/ai_gateway/stdio.go`：stdio 协议
- `internal/ai_gateway/gateway.go`：统一接口

#### 验收标准

- [ ] HTTP 协议工具调用成功
- [ ] stdio 协议工具调用成功
- [ ] RBAC 拒绝低权限调用
- [ ] 调用审计完整
- [ ] 超时正确取消

---

### 3.7 M7: 控制台 UI 重构 (Day 15-17)

#### 任务清单

| Task ID | 任务名称 | 描述 | 验收 | 估时 |
|---------|----------|------|------|------|
| T7.1 | API 客户端封装 | TypeScript API client | 类型定义 | 3h |
| T7.2 | /login 页面 | 用户名/密码登录 | UI 测试通过 | 4h |
| T7.3 | /change-password 页面 | 强制改密 | UI 测试通过 | 3h |
| T7.4 | 路由守卫 | must_change_password 检测 | UI 测试通过 | 2h |
| T7.5 | /admin/users 页面 | 用户 CRUD | UI 测试通过 | 4h |
| T7.6 | /admin/teams 页面 | 团队 CRUD + 成员管理 | UI 测试通过 | 5h |
| T7.7 | /projects 页面 | 项目列表（团队筛选） | UI 测试通过 | 4h |
| T7.8 | /projects/:id 页面 | 项目详情 + 模块树 + 创建项目向导 | UI 测试通过 | 6h |
| T7.9 | /modules/:id 页面 | 模块详情 + 子模块 + 会话/内存 | UI 测试通过 | 5h |
| T7.10 | /sessions 列表 | 会话列表 | UI 测试通过 | 3h |
| T7.11 | /sessions/:id 详情 | 会话轮次 + 归纳按钮 | UI 测试通过 | 4h |
| T7.12 | /memories CRUD | 内存列表/编辑 | UI 测试通过 | 4h |
| T7.13 | 模板选择器 | 在创建内存时选择模板 | UI 测试通过 | 4h |
| T7.14 | /ai-tools 页面 | 工具列表 + 调用界面 | UI 测试通过 | 5h |
| T7.15 | /ai-tools/:id 详情 | 工具配置 + 调用历史 | UI 测试通过 | 4h |
| T7.16 | /admin/stats 页面 | 全局统计 | UI 测试通过 | 3h |

#### 产出文件

- `docs-site/.vitepress/theme/console/api/client.ts`：API 客户端
- `docs-site/.vitepress/theme/console/pages/Login.vue`（重构）
- `docs-site/.vitepress/theme/console/pages/ChangePassword.vue`（新增）
- `docs-site/.vitepress/theme/console/pages/Users.vue`（重构）
- `docs-site/.vitepress/theme/console/pages/Teams.vue`（新增）
- `docs-site/.vitepress/theme/console/pages/Projects.vue`（重构）
- `docs-site/.vitepress/theme/console/pages/Modules.vue`（新增）
- `docs-site/.vitepress/theme/console/pages/Sessions.vue`（重构）
- `docs-site/.vitepress/theme/console/pages/Memories.vue`（新增）
- `docs-site/.vitepress/theme/console/pages/AITools.vue`（新增）

#### 验收标准

- [ ] 所有页面通过 UI 测试
- [ ] 强制改密跳转逻辑正确
- [ ] 模板选择器可用
- [ ] 模块树可视化
- [ ] AI 工具调用界面可用

---

### 3.8 M8: 配置热更新 + MCP 包装 + 部署 (Day 18)

#### 任务清单

| Task ID | 任务名称 | 描述 | 验收 | 估时 |
|---------|----------|------|------|------|
| T8.1 | fsnotify 集成 | 监听配置文件 | 单测通过 | 3h |
| T8.2 | 热更新 handler | 各种配置的热应用 | 集成测试通过 | 4h |
| T8.3 | 配置变更审计 | 写入 audit_logs | 集成测试通过 | 2h |
| T8.4 | MCP 包装适配 | 用户路由适配新 auth 模型 | E2E 通过 | 4h |
| T8.5 | per-user 进程池保留 | 进程池逻辑 | 已有 | 0h |
| T8.6 | systemd 单元 | 更新为 v2 启动参数 | 部署测试通过 | 2h |
| T8.7 | Caddy 配置 | HTTPS 终止 + 反代 | 部署测试通过 | 2h |
| T8.8 | 首次启动流程 | 创建初始 admin | E2E 通过 | 2h |
| T8.9 | 升级文档 | v1 → v2 升级步骤 | 文档完成 | 2h |

#### 产出文件

- `internal/config/config.go`：配置结构
- `internal/config/watcher.go`：fsnotify 监听
- `internal/config/hotreload.go`：热应用
- `deploy/cbmem-team.service`：systemd 单元（v2）
- `deploy/Caddyfile`：Caddy 配置（v2）
- `deploy/MIGRATION-v1-to-v2.md`：升级文档

#### 验收标准

- [ ] 修改 config.yaml 中 LLM 参数立即生效
- [ ] 修改 `-listen` 仍需重启（启动参数）
- [ ] 首次启动自动创建初始 admin
- [ ] v1 数据可平滑升级到 v2

---

### 3.9 M9: 测试 + 文档 + 验收 (Day 19-20)

#### 任务清单

| Task ID | 任务名称 | 描述 | 验收 | 估时 |
|---------|----------|------|------|------|
| T9.1 | 单元测试补全 | 覆盖率 ≥ 80% | coverage report | 6h |
| T9.2 | E2E 测试套件 | 核心流程 | E2E 通过 | 6h |
| T9.3 | 性能压测 | 50 并发用户 | 性能报告 | 3h |
| T9.4 | 安全扫描 | OWASP Top 10 | 扫描报告 | 3h |
| T9.5 | API 文档自动生成 | swag/swagger | 文档生成 | 2h |
| T9.6 | 用户手册 | README + 操作指南 | 文档完成 | 3h |
| T9.7 | 运维手册 | 部署/升级/备份 | 文档完成 | 2h |
| T9.8 | 部署验证（Linux） | Ubuntu 22.04 部署 | 部署成功 | 2h |
| T9.9 | 部署验证（macOS） | macOS Sonoma 部署 | 部署成功 | 2h |
| T9.10 | MVP 验收报告 | VERIFICATION.md | 报告完成 | 2h |

#### 产出文件

- `internal/console/*_test.go`：补全单测
- `cmd/e2e-mvp/main.go`：E2E 测试
- `tools/cbmem-team/docs/api.md`：API 文档
- `tools/cbmem-team/docs/user-manual.md`：用户手册
- `tools/cbmem-team/docs/ops-manual.md`：运维手册
- `docs/superpowers/verifications/2026-08-10-cbmem-team-mvp.md`：验收报告

#### 验收标准

- [ ] 单元测试覆盖率 ≥ 80%
- [ ] E2E 测试核心流程全通过
- [ ] 50 并发用户压测通过
- [ ] 安全扫描无 High/Critical
- [ ] API 文档、用户手册、运维手册齐全
- [ ] 两个 OS 部署验证通过

---

## 4. 任务依赖关系图

```
M1 (DB 迁移)
    │
    ▼
M2 (认证 + 用户/团队)
    │
    ├────────────────────┐
    ▼                    ▼
M3 (团队成员 + 项目)    M4 (模块树 + 会话)
    │                    │
    ▼                    │
M5 (内存 + 模板 + 归纳/蒸馏)
    │
    ▼
M6 (AI 工具)
    │
    ▼
M7 (UI 重构)
    │
    ▼
M8 (配置 + MCP + 部署)
    │
    ▼
M9 (测试 + 文档 + 验收)
```

**关键路径**：M1 → M2 → M3 → M4 → M5 → M6 → M7 → M8 → M9

**并行可能性**：
- M3 和 M4 可并行（不同模块）
- M7 UI 可在 M6 完成后立即开始（与 M8 并行）
- M9 测试可从 M2 开始滚动进行

---

## 5. 里程碑

| 里程碑 | 日期 | 交付物 | 验收标准 |
|--------|------|--------|----------|
| **M1 Done** | Day 2 | DB 迁移 + v2 schema | 迁移可执行、新部署可创建 schema |
| **M2 Done** | Day 5 | 认证 + 用户/团队管理 | 登录、改密、admin CRUD 全通过 |
| **M3 Done** | Day 8 | 团队成员 + 项目管理 | 创建项目自动 git clone |
| **M4 Done** | Day 10 | 模块树 + 会话 | 叶子校验、树状 CRUD 全通过 |
| **M5 Done** | Day 12 | 内存 + 模板 + 归纳/蒸馏 | 模板可用、内存 CRUD 全通过 |
| **M6 Done** | Day 14 | AI 工具网关 | 调用 + 审计 + RBAC 全通过 |
| **M7 Done** | Day 17 | 控制台 UI 重构 | 所有页面 UI 测试通过 |
| **M8 Done** | Day 18 | 配置热更新 + 部署 | 热更新生效、首次启动流程通 |
| **MVP Done** | Day 20 | 测试 + 文档 + 验收 | 覆盖率 ≥ 80%，验收报告通过 |

---

## 6. 资源需求

### 6.1 人力

| 角色 | 工作量 | 职责 |
|------|--------|------|
| 后端开发 | 12 人天 | M1-M6、M8 后端、M9 测试 |
| 前端开发 | 4 人天 | M7 UI 重构 |
| 架构师 | 2 人天 | 方案评审、关键决策 |
| 测试 | 2 人天 | E2E、性能、安全扫描 |
| 文档 | 1 人天 | 用户手册、运维手册 |
| **总计** | **~21 人天** | 约 4 周 |

### 6.2 基础设施

| 资源 | 规格 | 数量 |
|------|------|------|
| 开发服务器 | 4CPU/8GB | 1 |
| CI runner | 2CPU/4GB | 1 |
| 测试数据库 | SQLite/MySQL | 1 |
| LLM API | OpenAI GPT-4o-mini | - |

---

## 7. 风险与缓解

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| v1 → v2 数据迁移丢失 | 高 | 中 | 多次迁移演练、备份原库 |
| git clone 大项目阻塞 | 中 | 中 | 强制异步 + 状态机 |
| 模块树深度过大 | 中 | 低 | 限制最大层级（如 10） |
| AI 工具调用超时 | 中 | 中 | 超时 + 取消 + 重试 |
| 配置热更新副作用 | 中 | 低 | 灰度发布 + 监控 |
| E2E 测试不稳定 | 中 | 中 | 隔离环境 + 重试机制 |
| 前端 UI 工作量超 | 中 | 高 | MVP 简化 UI、优先功能 |
| 安全漏洞 | 高 | 低 | 渗透测试 + 静态扫描 |

---

## 8. 任务分配建议

| 成员 | 负责模块 | 估时 |
|------|----------|------|
| **成员 A（后端主程）** | M1, M2, M3, M5, M8 | 12 天 |
| **成员 B（后端 + AI）** | M4, M6 | 6 天 |
| **成员 C（前端）** | M7 | 4 天 |
| **成员 D（测试 + 文档）** | M9 滚动进行 | 3 天 |

如单人执行：按 M1→M2→M3→M4→M5→M6→M7→M8→M9 顺序，约 20 天。

---

## 9. 验收清单（Definition of Done）

### 9.1 功能验收

- [ ] 用户名/密码登录、强制改密、JWT 全流程通过
- [ ] admin 可创建/管理用户和团队
- [ ] team lead 可添加/移除成员
- [ ] 创建项目自动 git clone 并进入 ready 状态
- [ ] 模块树 CRUD + 移动 + 删除（含子树策略）
- [ ] 叶子模块校验生效（sessions/memories）
- [ ] 会话自动捕获并归属叶子模块
- [ ] 内存模板 5 个内置 + 渲染
- [ ] 内存 CRUD + 标签
- [ ] AI 工具注册 + HTTP/stdio 调用 + 审计
- [ ] 归纳/蒸馏任务手动触发
- [ ] 控制台 UI 所有页面可用
- [ ] 配置热更新生效

### 9.2 质量验收

- [ ] 单元测试覆盖率 ≥ 80%
- [ ] E2E 测试核心流程通过
- [ ] golangci-lint 0 errors
- [ ] go vet 0 warnings
- [ ] 前端 TypeScript 类型检查通过
- [ ] 前端 ESLint 0 errors

### 9.3 性能验收

- [ ] 50 并发用户会话捕获 < 100ms
- [ ] 单实例支持 100 用户
- [ ] 数据库查询 < 50ms（P95）
- [ ] API 响应 < 200ms（P95）

### 9.4 安全验收

- [ ] OWASP Top 10 基础检查通过
- [ ] bcrypt cost ≥ 12
- [ ] JWT 签名验证
- [ ] SQL 注入防护（参数化查询）
- [ ] XSS 防护（前端转义）
- [ ] CSRF 保护（POST/PUT/DELETE）
- [ ] AI 工具 RBAC 强制

### 9.5 部署验收

- [ ] Linux（Ubuntu 22.04）部署成功
- [ ] macOS（Sonoma）或 WSL2 部署成功
- [ ] 首次启动流程正确创建初始 admin
- [ ] v1 → v2 升级脚本可执行
- [ ] systemd 单元正确

### 9.6 文档验收

- [ ] API 文档自动生成（OpenAPI/Swagger）
- [ ] 用户手册（README + 操作指南）
- [ ] 运维手册（部署/升级/备份）
- [ ] 升级文档（v1 → v2）
- [ ] 验收报告（VERIFICATION.md）

---

## 10. 后续规划（v1.x 路线图）

完成 MVP 后，按优先级迭代：

| 版本 | 时间 | 内容 |
|------|------|------|
| v1.1 | +1 周 | 跨服务器 MemPalace 直接集成 |
| v1.2 | +1 周 | 实时通知（WebSocket） |
| v1.3 | +1 周 | 高级统计图表 |
| v1.4 | +1 周 | 多语言 i18n |
| v2.0 | +4 周 | 多实例水平扩展、Vault 集成 |

---

## 11. 相关文档

| 文档 | 路径 |
|------|------|
| PRD v2 | `tools/cbmem-team/cbmem-team-prd.md` |
| 设计 Spec | `docs/superpowers/specs/2026-07-14-dual-track-memory-tools-best-practices-design.md` |
| 数据模型 ER | `docs/superpowers/diagrams/cbmem-team-v2-erd.md` |
| 工具速查 | `docs/quick-ref/tools-quick-ref.md` |
| 上一个计划 | `docs/superpowers/plans/2026-07-14-dual-track-memory-tools-implementation-plan.md` |

---

**计划状态**: 📋 待执行
**评审人**: @team
**评审日期**: 待定
**执行启动**: 2026-07-15
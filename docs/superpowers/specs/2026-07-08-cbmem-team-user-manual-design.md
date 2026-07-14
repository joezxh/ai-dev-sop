# cbmem-team 用户手册重写 — 设计规格

> **日期**：2026-07-08
> **目标**：替换现有 `tools/cbmem-team/manual.md`，重写为角色导向的用户手册
> **目标读者**：开发者 + 运维
> **参考材料**：`manual.md`（现有）、`CONSOLE.md`、`cbmem-team-prd.md`、`50 场景回归测试`、`develop-sop.md §5`、`EMIT-MCP.md`

---

## 1. 设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 结构 | 角色导向（Part I 开发者 / Part II 运维 / Part III 参考） | 读者只看自己需要的部分 |
| MemPalace | 仅集成部分，不重复安装指南 | 避免信息冗余 |
| codebase-memory-mcp | 独立章节，放在 MemPalace 集成之后 | 用户明确要求 |
| 文件位置 | 替换 `tools/cbmem-team/manual.md` + `tools/cbmem-team/manual.en.md` | 用户明确要求，中英文同步重写 |

---

## 2. 手册目录结构

```
cbmem-team 用户手册
│
├── 前言
│   ├── 产品定位（HTTP 多用户包装器 + 控制台）
│   ├── 架构图（文本 ASCII）
│   └── 前置条件
│
├── Part I: 开发者指南
│   ├── 1. MCP 客户端配置
│   │   ├── 1.1 获取 JWT Token（管理员签发 / 自助续期）
│   │   ├── 1.2 Cursor 配置（~/.cursor/mcp.json 完整示例）
│   │   ├── 1.3 Qoder 配置（MCP 配置文件示例）
│   │   ├── 1.4 Claude Desktop 配置（claude_desktop_config.json）
│   │   ├── 1.5 CodeBuddy 配置
│   │   ├── 1.6 Token 续期（POST /refresh，无需管理员）
│   │   ├── 1.7 自动配置脚本（cbmem-emit-mcp.ps1 / .sh）
│   │   └── 1.8 端到端连通性验证（curl 测试步骤）
│   │
│   ├── 2. 控制台操作指南
│   │   ├── 2.1 登录（admin-token → session cookie + CSRF）
│   │   ├── 2.2 用户管理（CRUD + Token 签发/撤销）
│   │   ├── 2.3 项目管理（CRUD + Wing 绑定 + 触发索引）
│   │   ├── 2.4 会话记录查询（列表筛选 + 详情 + 统计）
│   │   ├── 2.5 会话归纳 Summarize（选会话 → 5 Hall → 写入）
│   │   └── 2.6 会话蒸馏 Distill（选会话 → 知识片段/决策/技术债 → commit）
│   │
│   ├── 3. MemPalace 集成
│   │   ├── 3.1 核心概念（Palace/Wing/Room/Hall/Drawer/Tunnel）
│   │   ├── 3.2 归纳/蒸馏结果写入 MemPalace（target_wing 配置）
│   │   └── 3.3 搜索与检索（mempalace search / wake-up）
│   │
│   └── 4. codebase-memory-mcp 集成
│       ├── 4.1 什么是 codebase-memory-mcp（单用户本地 AST 索引工具）
│       ├── 4.2 本地模式 vs 团队模式对比
│       ├── 4.3 索引目录结构（${DataDir}/users/<userID>/projects/<project>/）
│       ├── 4.4 MCP 协议交互流程（initialize → tools/list → tools/call）
│       └── 4.5 会话自动采集机制（capture middleware）
│
├── Part II: 运维指南
│   ├── 5. 部署与启动
│   │   ├── 5.1 编译构建
│   │   ├── 5.2 前台模式（开发/调试）
│   │   ├── 5.3 systemd 后台部署（推荐生产）
│   │   ├── 5.4 MySQL 部署（Docker Compose + 建表 + ETL）
│   │   └── 5.5 SQLite ↔ MySQL 切换（7 天回退窗口）
│   │
│   ├── 6. 用户与项目管理
│   │   ├── 6.1 创建用户 + 签发 Token（Admin API）
│   │   ├── 6.2 管理项目 + 触发索引
│   │   ├── 6.3 离线签发 JWT（cbmem-mint-token）
│   │   └── 6.4 克隆远程仓库（/admin/repos/clone）
│   │
│   ├── 7. 高级功能
│   │   ├── 7.1 工具目录 + 调用日志（M1）
│   │   ├── 7.2 工具限流 + Best Practices CRUD（M2）
│   │   ├── 7.3 工作流引擎（M3：draft → published → run）
│   │   └── 7.4 Repo Pipeline（M4：crawl → parse → grade → sink → rehearse）
│   │
│   └── 8. 命令行工具参考
│       ├── 8.1 生产级工具（cbmem-team / cbmem-mint-token）
│       ├── 8.2 运维/迁移工具（create-db / mysql-probe / show-tables / show-indexes）
│       ├── 8.3 子命令（migrate-tables / migrate-sqlite-to-mysql / mysql-ping）
│       └── 8.4 工具速查表
│
├── Part III: 参考
│   ├── 9. 配置参数全表
│   │   ├── 9.1 必需参数
│   │   ├── 9.2 LLM 参数
│   │   ├── 9.3 MemPalace 参数
│   │   └── 9.4 可选参数
│   │
│   ├── 10. 数据库表结构
│   │   ├── 10.1 SQLite 与 MySQL 双轨说明
│   │   ├── 10.2 核心表（users / projects / sessions / session_turns）
│   │   ├── 10.3 任务表（summarize_tasks / distill_tasks）
│   │   ├── 10.4 控制台表（console_sessions）
│   │   └── 10.5 M1-M4 扩展表
│   │
│   ├── 11. API 速查表
│   │   ├── 11.1 管理端点（/admin/*）
│   │   ├── 11.2 控制台端点（/api/console/*）
│   │   ├── 11.3 MCP 端点（/mcp / /mcp/sse）
│   │   └── 11.4 统一响应格式 + 错误码
│   │
│   └── 12. FAQ
│       ├── 编译问题
│       ├── 连接问题（401 / Cursor 连不上）
│       ├── 数据库问题（SQLite 锁 / MySQL 字符序）
│       ├── 部署问题（systemd / 磁盘增长）
│       └── PowerShell 兼容
│
└── 附录
    ├── A. 环境变量与配置文件
    ├── B. 相关文档链接
    └── C. 版本与变更
```

---

## 3. 各章节内容规格

### 前言

- 一段话产品定位：cbmem-team 是 codebase-memory-mcp 的 HTTP 多用户包装器 + 双轨记忆控制台
- ASCII 架构图（复用 PRD 中的系统架构图，简化为 3 层：客户端 → cbmem-team → 后端）
- 前置条件表格（Go 1.24+、SQLite 内置、MySQL 8.0 可选、Node 20+ 可选）

### Part I §1 MCP 客户端配置

**核心内容**：

- **1.1 获取 JWT Token**：
  - 方式一：管理员通过 `POST /admin/users/:id/token?ttl=720h` 签发
  - 方式二：用户自助续期 `POST /refresh?ttl=4320h`（持有有效 JWT 即可）
  - 方式三：离线签发 `cbmem-mint-token --secret $JWT_SECRET --user alice --ttl 720h`

- **1.2-1.5 各 IDE 配置**：每个 IDE 提供
  - 配置文件路径
  - 完整 JSON 配置示例（含 url、headers、Authorization）
  - URL 参数说明（`as=<user_id>` 和 `project=<server_path>`）
  - 注意事项（路径必须是服务端路径、TLS 建议等）

- **1.6 Token 续期**：curl 示例 + 说明不需要 admin token

- **1.7 自动配置脚本**：
  - `cbmem-emit-mcp.ps1`（Windows）和 `cbmem-emit-mcp.sh`（Linux/macOS）
  - 完整使用示例（参考 EMIT-MCP.md）
  - 脚本自动执行 5 步：mint → load → upsert → smoke test → summary

- **1.8 端到端验证**：
  - Step 1: `curl /healthz`
  - Step 2: `curl POST /mcp` initialize 握手
  - 期望响应说明

### Part I §2 控制台操作指南

**核心内容**（参考 50 场景回归测试中的关键路径）：

- **2.1 登录**：
  - 浏览器打开 `http://server:8787/console/`
  - 输入 admin-token → 登录
  - 认证机制说明（session cookie + CSRF token）
  - 登出操作

- **2.2 用户管理**：
  - 创建用户（id / display_name / project_paths）
  - 用户列表（搜索、分页）
  - 编辑用户（更新 display_name / project_paths / disabled）
  - 删除用户
  - 撤销 Token / 禁用用户
  - 白名单限制说明

- **2.3 项目管理**：
  - 创建项目（name / path / wing / mcp_bin）
  - 项目列表（搜索、分页）
  - 删除项目（软删除）
  - 索引状态查看 + 触发重新索引
  - wing 字段与 MemPalace 的关系

- **2.4 会话记录查询**：
  - 会话列表（按用户/项目/时间范围筛选）
  - 会话详情（基本信息 + 所有 turns）
  - 会话统计（总数、轮次、工具调用、Top 用户/项目）
  - 会话采集机制简介（capture middleware 触发条件）

- **2.5 会话归纳**：
  - 操作步骤：选择会话 → 设置 depth（shallow/deep/expert）→ 设置 target_wing → 开始归纳
  - 结果展示：5 Hall Tab（facts / events / discoveries / preferences / advice）
  - 写入 MemPalace
  - API 调用示例

- **2.6 会话蒸馏**：
  - 操作步骤：选择会话 → 设置规则（min_value_score / dimensions）→ 开始蒸馏
  - 结果展示：知识片段 / 决策 / 技术债务
  - Commit 写入 MemPalace（target_wing 参数）
  - 幂等性说明
  - API 调用示例

### Part I §3 MemPalace 集成

- 核心概念表格（Palace/Wing/Room/Hall/Drawer/Tunnel）
- 5 Hall 说明（facts / events / discoveries / preferences / advice）
- cbmem-team 如何写入 MemPalace（通过 `-mempalace-base` 配置）
- 归纳/蒸馏结果写入流程
- 搜索与检索简要说明（指向 MemPalace 官方文档）

### Part I §4 codebase-memory-mcp 集成

- 什么是 codebase-memory-mcp：单用户本地 AST 索引工具
- 本地模式 vs 团队模式对比表：
  - 本地模式：直接 stdio，单用户，零配置
  - 团队模式：通过 cbmem-team HTTP，多用户，需 JWT
- 索引目录结构说明
- MCP 协议交互流程（initialize → tools/list → tools/call）
- 会话自动采集机制（capture middleware 如何工作）

### Part II §5 部署与启动

保留现有 manual.md §2 的内容，精简重复部分。

### Part II §6 用户与项目管理

保留现有 manual.md §1.4 + §3.3 的内容，补充 API 示例。

### Part II §7 高级功能

新增内容，参考 manual.md §5 + CONSOLE.md 的 M1-M4 部分。

### Part II §8 命令行工具参考

保留现有 manual.md §4 的内容，格式化为速查表。

### Part III §9-12

保留现有 manual.md 中的配置参数、数据库表结构、FAQ 等内容，重新组织。

---

## 4. 内容来源映射

| 手册章节 | 主要内容来源 | 新增内容 |
|----------|-------------|---------|
| 前言 | PRD §1 | 无 |
| §1 MCP 客户端配置 | manual.md §3 + EMIT-MCP.md | Qoder/CodeBuddy 配置、自动配置脚本 |
| §2 控制台操作 | CONSOLE.md + 50 场景 | **全新**：操作步骤、截图说明 |
| §3 MemPalace 集成 | develop-sop.md §5 | 精简为集成部分 |
| §4 cbmem-mcp 集成 | PRD §2.2.1 + manual.md §5.1 | **全新**：本地 vs 团队模式对比 |
| §5 部署与启动 | manual.md §2 | 精简 |
| §6 用户/项目管理 | manual.md §1.4 + PRD §5 | 补充 API 示例 |
| §7 高级功能 | manual.md §5 + CONSOLE.md | M3 工作流、M4 Pipeline |
| §8 命令行工具 | manual.md §4 | 格式化 |
| §9 配置参数 | manual.md 附录 A + PRD §6 | 无 |
| §10 数据库表 | manual.md 未覆盖 + PRD §7 | **新增** |
| §11 API 速查 | CONSOLE.md + PRD §5 | 合并整理 |
| §12 FAQ | manual.md §6 | 保留 + 补充 |

---

## 5. 预估规模

| 部分 | 预估行数 |
|------|---------|
| 前言 | ~50 |
| Part I 开发者指南 | ~500 |
| Part II 运维指南 | ~500 |
| Part III 参考 | ~300 |
| 附录 | ~50 |
| **总计** | **~1400（中文） + ~1200（英文）** |

---

## 6. 约束与假设

- 手册语言：中文
- 文件格式：Markdown
- 文件位置：`tools/cbmem-team/manual.md`（替换现有）
- 不创建新文件
- 英文版 `manual.en.md` 同步替换，结构与中文版一致
- 所有 API 路径和参数以当前代码为准，参考 CONSOLE.md 和 PRD

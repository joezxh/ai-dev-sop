> [!WARNING]
> **已废弃（2026-09-18）**：本文档描述的 cbmem-team 控制台/导航聚合方案基于已移除的旧记忆系统，仅作历史归档。

# cbmem-team 控制台前端集成 — 设计文档

> **Status**: DRAFT — awaiting user approval
> **Author**: brainstorming session, 2026-07-12 (rev 2 — integration over rebuild)
> **Approach**: 保留 docs-site VitePress 文档站，叠加控制台 SPA 模块；不新建独立 web 工程。
> **Out of scope**: cbmem-team 后端 API 变更（除非必要）、首次部署自动化

---

## 1. 核心方向反转（vs rev 1）

| 维度 | rev 1（已废） | rev 2（本稿） |
|---|---|---|
| 工程边界 | 新建 `tools/cbmem-team/web/` 独立 SPA | **在 `docs-site/` 内挂控制台模块** |
| 文档与控制台的关系 | 两套独立 UI、nginx 切换 | **同站共存，docs-site 当壳、控制台嵌内部** |
| 文档浏览 | 仅 docs-site 提供 | docs-site 保留全部现有 markdown 文档浏览 |
| 控制台接入 | SPA iframe / 反代 | docs-site 自家路由直接调用后端 API |
| 二进制部署 | 双产物（`cbmem-team` + `dist/`） | 单产物（`docs-site/.vitepress/dist/`），由 cbmem-team 用 `-console-dist` 挂载 |

**决定理由**：
1. docs-site 已用 Vue 3 + Element Plus + pinia + @vueuse/core，**复用现成依赖零成本**
2. VitePress 支持自定义主题与功能插件，可以"在文档站里加 SPA 子应用"
3. 单一构建链路、单一部署单元、文档与控制台侧边栏合并
4. 不破坏 cbmem-team 当前 `-console-dist` flag 的契约

---

## 2. 用户 8 项要求对照表

| # | 用户要求 | 本 spec 对应章节 | 状态 |
|---|---|---|---|
| 1 | 保留 docs-site 文档展示功能 | §3.1（路由结构）、§6.1（迁移边界） | ✅ |
| 2 | 增加 7 类控制台功能（服务状态/系统配置/用户权限/日志/统计/系统设置） | §4.2（控制台模块清单） | ✅ |
| 3 | 与文档浏览无缝集成 | §3.1（统一侧边栏 + 角色切换） | ✅ |
| 4 | 完整 UI/交互逻辑 | §5（每个模块的视图+交互清单） | ✅ |
| 5 | 遵循现有 UI/UX | §7（设计令牌 + Element Plus 主题覆盖） | ✅ |
| 6 | 权限控制 | §8（RBAC 模型 + 路由守卫 + UI 显隐） | ✅ |
| 7 | 完整功能测试 | §9（vitest + Playwright + 手动 smoke 矩阵） | ✅ |
| 8 | 更新相关文档 | §11（manual.md / BUILD.md / docs-site 自身 README 更新点） | ✅ |

---

## 3. 架构与集成

### 3.1 统一侧边栏与路由

```
docs-site/
├── guide/              # 现有文档（不动）
├── sop/                # 现有文档（不动）
├── reference/          # 现有文档（不动）
├── ja/, en/            # 现有 i18n 文档（不动）
├── public/             # 公共资源
└── console/            # ✨ 新增：控制台源码
    ├── modules/
    │   ├── service-status/        # 服务状态监控
    │   ├── config/               # 系统配置
    │   ├── users/                # 用户权限
    │   ├── logs/                 # 日志查看
    │   ├── stats/                # 数据统计
    │   ├── settings/             # 系统设置
    │   └── workflows-repos/      # 工作流+repo pipeline（M3/M4）
    ├── shared/
    │   ├── api-client.js         # axios + 信封解析
    │   ├── auth-guard.js         # 路由守卫
    │   ├── rbac.js               # 权限矩阵
    │   └── design-tokens.css     # 设计令牌
    └── views/                    # Vue SFC
```

**VitePress 自定义布局**（`.vitepress/theme/index.ts`）：在默认文档布局外，**新增 `/console/*` 路由**，挂载 Vue 子应用。**侧边栏合并规则**：

```
├─ 文档（原有，侧边栏渲染 markdown）
│   ├─ 介绍
│   ├─ SOP
│   ├─ Reference
│   └─ 多语言
└─ 控制台（新增，路由进入 SPA 视图）
    ├─ 服务状态
    ├─ 系统配置
    ├─ 用户权限
    ├─ 日志查看
    ├─ 数据统计
    ├─ 系统设置
    └─ 工作流 & Repo
```

切换由顶部 **NavBar** 一个 `📚 文档` / `🛠️ 控制台` tab 控制。

### 3.2 控制台部署在 docs-site dist 内的好处

- **单构建产物**：VitePress 已支持；新加模块走 vitepress-plugin-back-to-top 风格的"自定义插件"
- **零路由冲突**：VitePress 默认 history mode，但 SPA 子路由用 hash 分隔（`/console#/users`），不需要后端兜底
- **API 调用**：经由现有 `cbmem-team` 后端（沿用 `-console-dist` 旗标，直接把 dist 暴露在 8787 下）

### 3.3 数据流

```
User Browser
  ↓
  http://server:8787/  (or /docs/)
  ├─ 文档路径 → VitePress 静态 markdown
  └─ /console/* → SPA Vue 子应用
                ↓
                axios /api/console/* → cbmem-team Go backend → SQLite/MySQL
```

控制台模块**完全使用现有后端 API**（除 distill 待补 ~30 行 Go 外），不写 mock。

---

## 4. 控制台模块清单

### 4.1 7 大模块 vs 现有后端路由匹配表

| 控制台模块 | 后端 API | 现有后端完整度 | 新增后端工作量 |
|---|---|---|---|
| **服务状态监控** (`/console/status`) | `GET /healthz`、`GET /api/console/v2/dashboard/summary`、`/api/console/me`、MCP `initialize` 心跳 | 100% | 0 |
| **系统配置管理** (`/console/config`) | `-mysql-dsn` / `-listen` 改写（如果存在）→ 当前**无接口**（flag once-only）| 0% | **新增**：`GET/PUT /api/console/v2/config`（10 行 Go + 一个 kv 表） |
| **用户权限管理** (`/console/users`) | `/api/console/users[/:id][/revoke]` | 100% | 0 |
| **日志查看** (`/console/logs`) | **无接口**（当前只走 stderr / journal） | 0% | **新增**：`GET /api/console/v2/logs?level=&since=&limit=`（30 行 Go，加日志缓冲通道） |
| **数据统计面板** (`/console/stats`) | `/api/console/v2/dashboard/summary`、`/v2/invocations/recent`、`/v2/sessions-stats`、`/v2/dashboard/high-risk[/summary]` | 100% | 0 |
| **系统设置** (`/console/settings`) | `/api/console/me` + 用户偏好（如主题、语言）→ **无接口** | 0% | **新增**：`GET/PUT /api/console/v2/me/prefs`（20 行 Go + kv or user_pref 表） |
| **工作流 + Repo (M3/M4)** (`/console/workflows`) | `/api/console/v2/workflows/*`、`/v2/tickets/*`、`/v2/repos/*`、`/v2/repos/candidates/*` | 100%（router.go 完整） | 0 |
| **会话/summarize/distill (M0)** | `/api/console/sessions/*`、`/summarize`、`/distill`（**待补**） | 67% | **新增**：`/api/console/distill` 一组（30 行 Go） |

### 4.2 缺失后端模块的清单

为了让 6 大控制台模块全部接通，需补 4 个 Go 接口：

| Go 文件 | 行数 | 测试 |
|---|---|---|
| `internal/console/config_handler.go`（新） | ~60 | +60 |
| `internal/console/logs_handler.go`（新） | ~120 | +80 |
| `internal/console/prefs_handler.go`（新） | ~50 | +40 |
| `internal/console/distill_handler.go`（新） | ~80 | +60 |

合计新增后端 ~310 行 + 测试 ~240 行。

每个接口都做 TDD（红→绿→重构）；与 cbmem-team 后端现有的 `*_handler_test.go` 风格一致。

---

## 5. 每个控制台模块的视图与交互

### 5.1 服务状态监控 `/console/status`

**数据源**：`/healthz`、`/api/console/v2/dashboard/summary`、`/api/console/me`

**视图元素**：
- 顶部 4 张卡片：服务在线状态、活跃用户数、当前 QPS、最后一次请求时间戳
- 中间面板：进程信息（PID、uptime、Go 版本、监听端口）—— 后端需扩展一个 `/api/console/v2/status/runtime`
- 右下角：MCP 服务器心跳（每 5 秒 SSE 推一次）

**交互**：
- 点击"重启后端"按钮 → `POST /api/console/v2/status/restart` → 显示"已请求重启"
- 自动刷新：每 10 秒轮询一次（带 loading 状态）
- 异常状态变红 + 顶部 alert 通知

### 5.2 系统配置管理 `/console/config`

**数据源**：`/api/console/v2/config`（新增）

**视图元素**：
- 表格展示：flag 名 / 当前值 / 默认值 / 是否重启生效 / 说明
- 折叠分组：网络设置 / 数据库设置 / 限流默认值 / LLM 凭据（masked）/ MemPalace 端点

**交互**：
- 行内编辑 → `PUT /api/console/v2/config/<key>`
- "撤销 / 重做" 按钮（保留 10 步操作栈）
- 危险项（如 `-admin-token`）要求二次确认 + 输入原 token 验证
- LLM API Key 默认脱敏（点击眼睛图标切换显示）

### 5.3 用户权限管理 `/console/users`

**数据源**：`/api/console/users[/:id][/revoke]`

**视图元素**：
- 用户列表（el-table）：id / display_name / project_paths / role / enabled / created_at
- 顶部"创建用户"按钮 → 抽屉表单
- 行操作：编辑 / 启用停用 / 撤销 token / 删除（带二次确认）

**角色（RBAC）**：
- `admin`（默认）：所有权限
- `operator`：除"用户管理 / 配置管理"外全部
- `viewer`：只读
- 角色在用户创建/编辑时可选；UI 在 RBAC 后用 `v-show` 隐藏无权限按钮（防御性 + 视觉清爽）

### 5.4 日志查看 `/console/logs`

**数据源**：`GET /api/console/v2/logs?level=info&since=2026-07-12T00:00:00&limit=500`

**视图元素**：
- 顶部筛选条：level 多选 / since 时间选择 / limit 限制 / "导出 CSV"按钮
- 表格（虚拟滚动）：timestamp / level / logger / message / request_id
- 右侧详情：单条 message 全文（含 ANSI 高亮去除）

**交互**：
- "实时跟踪"开关：开启后 SSE 推流（`/api/console/v2/logs/stream`）
- 点击 level 单元格 → 显示堆栈折叠
- 关键字高亮：URL `?q=foo` 自动加红高亮

### 5.5 数据统计面板 `/console/stats`

**数据源**：`/api/console/v2/dashboard/summary`、`/invocations/recent`、`/sessions-stats`、`/dashboard/high-risk[/summary]`

**视图元素**：
- 6 张卡片：今日调用 / 7 日活跃用户 / 错误率 / 慢调用 P95 / 高危调用数 / 工单数
- 折线图：过去 7 日 QPS（element-plus 不带 chart → 用 `echarts` 或 `@antv/g2`）
- 表格：最近 50 次调用（点 → 调用详情抽屉）

**交互**：
- 时间范围切换：今日 / 7 日 / 30 日
- 图表下钻：点击数据点 → 跳到 SessionsView
- "导出 PNG" 按钮

### 5.6 系统设置 `/console/settings`

**数据源**：`/api/console/v2/me/prefs`（新增）

**视图元素**：
- 个人偏好：主题（浅/深）、语言（zh/en/ja）、时区、日期格式
- 通知设置：邮件 / 飞书 / 钉钉 webhook（脱敏）
- 危险区："撤销所有 token" 按钮

**交互**：
- 改动自动保存（500ms debounce）
- 撤销全部 token 跳二次确认 + 输入密码

### 5.7 工作流 + Repo Pipeline `/console/workflows` & `/console/repos`

> 合称 M3 + M4

**两个子模块**：

#### 5.7.1 工作流 `/console/workflows`

- 列表视图：id / 状态徽章（draft/published/archived）/ 节点数 / 最近 run 时间
- 列表项点击 → `WorkflowEditor.vue`（vue-flow 拖拽图）
- Editor 内：7 种节点 kind (`log/tool_call/bp_check/condition/wait_approval/parallel/loop_guard`) 可拖入
- Publish / Archive 按钮置右；run 后右抽屉显示 run logs

#### 5.7.2 Repo Pipeline + BP Candidates `/console/repos`

- 列表视图：name / source 类型 / 最近一次 run 状态 / 7 日成功率
- 详情视图：5 阶段进度条（crawl→parse→grade→sink→rehearse）+ runs 列表
- `/console/candidates`：BP 候选 review 工作台（accept / reject / merge 三按钮）

### 5.8 会话 + Summarize + Distill `/console/sessions`

> 这一项用户没明确列，但是 manual.md §5.7 承诺的一部分，作为隐藏的"必做项"补完 M0：

- `/console/sessions`：会话列表 + 详情（含 turn 列表 + tool invocation 时间线）
- `/console/summarize`：选 session → 调 LLM → 异步轮询 → 展示归纳结果
- `/console/distill`：选归纳 → 喂 MemPalace → 显示蒸馏候选 BP（**后端 distill API 阶段一并补**）

---

## 6. 与现有功能的兼容性

### 6.1 docs-site 不动的部分

- **所有 markdown 文件**（86 个 .md）原样保留
- **现有 VitePress 主题**：左侧文档导航 / 顶部 navbar / 搜索 全部继续工作
- **i18n**：zh / en / ja 三套文档翻译链接保留

### 6.2 docs-site 改动点（最小化）

| 改动点 | 必要性 |
|---|---|
| `.vitepress/config.ts`：侧边栏新增"控制台"入口 | 必要 |
| `.vitepress/theme/index.ts`：注册 SPA 子应用 | 必要 |
| `package.json` 新增 dev 依赖：`axios`、`echarts`、`vue-flow`、`pinia`（已装） | 必要 |
| `package.json` 新增 dev 依赖：`@vue/test-utils`、`vitest` | 测试需要 |

总计 docs-site 自身改动 < 150 行（config + theme + new deps）。

### 6.3 cbmem-team 后端改动点

| 改动点 | 行数 |
|---|---|
| `internal/console/router.go`：新增 4 个路由组（config / logs / prefs / distill） | ~20 |
| `internal/console/config_handler.go`（新） | ~60 |
| `internal/console/logs_handler.go`（新） | ~120 |
| `internal/console/prefs_handler.go`（新） | ~50 |
| `internal/console/distill_handler.go`（新） | ~80 |
| 对应 `*_test.go`（新） | ~240 |

合计 ~570 行 Go 代码（含测试）。

### 6.4 不动的部分

- `tools/cbmem-team/internal/console/static/m2-console.html`（保留作为 `/ui/m2/` fallback，6 个月后移除）
- `cbmem-team` 二进制协议（`-console-dist` flag 继续生效）
- 已有 12 个 `*_handler_test.go`

---

## 7. UI/UX 规范

### 7.1 设计令牌（CSS 变量，写入 `console/shared/design-tokens.css`）

```css
:root {
  --color-primary: #2563eb;
  --color-success: #16a34a;
  --color-warning: #f59e0b;
  --color-danger: #dc2626;
  --color-text: #1f2937;
  --color-text-muted: #6b7280;
  --color-border: #e5e7eb;
  --color-bg: #fafbfc;
  --color-bg-card: #ffffff;
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 10px;
  --shadow-card: 0 1px 2px rgba(0,0,0,.03), 0 1px 4px rgba(0,0,0,.05);
  --space-xs: 4px;
  --space-sm: 8px;
  --space-md: 12px;
  --space-lg: 16px;
  --font-mono: ui-monospace, 'Cascadia Mono', 'Consolas', monospace;
}
```

### 7.2 Element Plus 主题覆盖

通过 SCSS 变量覆盖 Element Plus 默认色板，确保按钮/表格/Tag 与 docs-site 现有 navbar 视觉对齐：

```scss
// console/shared/ep-overrides.scss
:root {
  --el-color-primary: var(--color-primary);
  --el-border-radius-base: var(--radius-md);
  --el-font-family: -apple-system, system-ui, 'PingFang SC', sans-serif;
}
```

### 7.3 统一交互规范

- **加载状态**：所有 axios 调用有 loading 状态（按钮 spinner / 表格骨架）
- **错误处理**：401 自动跳登录；403/409/500 toast；网络断 → "重试"按钮
- **空状态**：每个列表/卡片都有 empty 文案 + 一键创建入口
- **确认弹窗**：删除/撤销全部 token 等不可逆操作必须有 ConfirmDialog（防误触）
- **键盘可达**：Tab 顺序合理，Esc 关闭抽屉

---

## 8. 权限控制（RBAC）

### 8.1 三种角色

| 角色 | 可见模块 | 可执行操作 |
|---|---|---|
| `admin` | 全部 | 全部（含配置 / 删除用户 / 重启服务） |
| `operator` | 除"用户管理 / 系统配置 / 系统设置"外 | 查看 + 业务操作（创建 BP / 跑工作流 / 接受 BP 候选） |
| `viewer` | 全部视图 | 只读（所有 PUT/POST/DELETE/PATCH 显隐） |

### 8.2 实现位置

- **后端**：每个 handler 用现有 `RequireRole(role string)` 中间件（**已有 `RequireSession` / `RequireCSRF`，但没有 RBAC**——需新增 `internal/console/rbac.go`）
- **前端**：`console/shared/rbac.js` 暴露 `can(action, resource)`；UI 用 `<button v-if="can('edit', 'users')">` 控制按钮显隐

### 8.3 关键操作权限矩阵

| 操作 | admin | operator | viewer |
|---|---|---|---|
| 查看所有模块 | ✅ | ✅ | ✅ |
| 创建/编辑用户 | ✅ | ❌ | ❌ |
| 编辑系统配置 | ✅ | ❌ | ❌ |
| 修改限流默认值 | ✅ | ✅ | ❌ |
| 创建 BP | ✅ | ✅ | ❌ |
| 跑工作流 | ✅ | ✅ | ❌ |
| 接受 BP 候选 | ✅ | ✅ | ❌ |
| 查看日志 | ✅ | ✅ | ✅ |
| 查看 dashboard | ✅ | ✅ | ✅ |
| 修改个人偏好 | ✅ | ✅ | ✅ |

---

## 9. 测试策略

### 9.1 单元测试（vitest，60% 覆盖率目标）

- `shared/api-client.test.js`：信封解析、拦截器、错误码分流
- `shared/rbac.test.js`：can() 矩阵
- `users/users.test.js`：mocked 后端
- `config/config.test.js`
- `logs/logs.test.js`
- `stats/echarts-wrapper.test.js`

### 9.2 组件测试（@vue/test-utils）

- UsersView 完整 CRUD 流程
- LogsView 筛选 + 高亮
- ServiceStatusView 轮询与重启按钮

### 9.3 端到端（Playwright，配置 docs-site dev server）

- `e2e/status.spec.js`：健康检查 + 重启
- `e2e/users.spec.js`：登录 → 创建 → 改权限 → 删除
- `e2e/rbac.spec.js`：用 viewer 身份登录，确认编辑按钮全隐藏
- `e2e/logs.spec.js`：筛选 + 实时跟踪

### 9.4 与现有后端测试协同

后端新增的 4 个 `*_handler_test.go` 与现有 12 个串行运行；前端 vitest 在 CI 中独立 stage。

### 9.5 手动 smoke 矩阵

CI 通过后，每次发版前人工跑：
- 启动 cbmem-team → 打开 `http://server:8787/`（配 -console-dist docs-site dist）
- 依次访问 7 个模块
- 测试 admin / operator / viewer 三种角色
- 测试错误码 toast（人为触发 409 用户重复）

---

## 10. 实施计划

### 10.1 阶段划分（30 人天 / 6 周单人 或 3 周 2 人并行）

#### 阶段 A — docs-site 基础设施（3 天）

- A1：装依赖（axios / echarts / vue-flow / vitest） + 配置
- A2：自定义 VitePress 主题（侧边栏新增"控制台"+ 路由切换）
- A3：共享模块（`api-client.js`、`auth-guard.js`、`rbac.js`、`design-tokens.css`）
- A4：验证：原文档站可访问 + `/console` 空壳可访问

#### 阶段 B — 后端补全（4 天）

- B1：新增 `rbac.go` 中间件 + 现有 handler 加 role 检查
- B2：新增 config handler + 测试
- B3：新增 logs handler + 测试（含 SSE）
- B4：新增 prefs handler + 测试
- B5：新增 distill handler + 测试
- B6：`router.go` 挂 4 个新路由组
- B7：跑 e2e-v7 验不回归

#### 阶段 C — 控制台模块实施（15 天）

- C1：服务状态（2 天）
- C2：用户权限（3 天）
- C3：数据统计（2 天）
- C4：日志查看（3 天）
- C5：系统配置（2 天）
- C6：系统设置（1 天）
- C7：会话/归纳/蒸馏（2 天）
- C8：工作流编辑器 vue-flow（3 天）
- C9：Repo Pipeline + Candidates（3 天）

#### 阶段 D — 测试 + 文档（4 天）

- D1：vitest 单元测试覆盖率 ≥ 60%
- D2：Playwright e2e（4 个 spec）
- D3：更新 manual.md §5.7（修正"前端产物路径"为 docs-site dist）
- D4：更新 BUILD.md（加 pnpm build + 引入 docs-site / 移除 cbmem-team console flag 描述差异）
- D5：docs-site 自身 README.md 加"控制台"章节
- D6：在 `console/shared/` 写 README，列出每个模块的 RBAC 与 API 对应

#### 阶段 E — 灰度与 fallback（4 天）

- E1：保留 `m2-console.html` 6 个月（`-console-dist` 切换为 docs-site dist）
- E2：在 docs-site README 加 "legacy M2 console at /ui/m2/" 链接
- E3：监控面板收集错误 → CI 报警

### 10.2 人员配置

- **单人**：6 周
- **2 人**（一人后端一人前端）：3 周
- **3 人**（+1 测试）：2 周

### 10.3 风险与缓解

| 风险 | 影响 | 缓解 |
|---|---|---|
| VitePress 子路由与文档路由冲突 | /console 404 | 用 hash 模式 `/console/#/users` |
| CSP 拦截 SSE 流（如果部署 nginx 严格 CSP） | 日志实时跟踪失效 | 在 nginx 配置 `proxy_buffering off` |
| vue-flow 与 Vue 3.4 兼容问题 | M3 编辑器失败 | spike A3 验证；不行 fallback mermaid 只读 |
| 后端 4 个新接口与现有 e2e-v7 冲突 | 回归测试失败 | B 阶段结束跑全量 e2e |
| 用户角色字段历史数据缺 | 部分用户默认 viewer 而非 admin | 启动时一次性迁移脚本：admin_token 拥有者置 admin |
| docs-site 主题修改影响 markdown 渲染 | 文档站视觉跑偏 | A2 主题修改限制在 `.vitepress/theme/console/` 命名空间 |

---

## 11. 文档更新清单

| 文件 | 改动 |
|---|---|
| `tools/cbmem-team/manual.md` §5.7 | 把"前端构建产物在 docs-site/.vitepress/dist/"保留并强化（事实现在对了）；列出 7 大控制台模块 + RBAC 矩阵 |
| `tools/cbmem-team/manual.en.md` | 同步更新 |
| `tools/cbmem-team/BUILD.md` | 加 `pnpm --dir docs-site build` → 写到 `cbmem-team -console-dist docs-site/.vitepress/dist` |
| `tools/cbmem-team/CONSOLE.md` | 已有；新模块详细 API 列表作为附录 |
| `docs-site/README.md` | 加 "控制台接入指南" 段落 |
| `docs-site/console/README.md`（新） | 控制台模块 README，列出每个模块的 URL + 角色权限 |
| `.vitepress/config.ts` | 侧边栏新增"控制台"组，链接到 7 个模块 |

---

## 12. 交付物总览

### 12.1 docs-site 侧（Vue 3 + VitePress）

- `docs-site/console/`：7 个模块源码 + 共享代码
- `docs-site/.vitepress/config.ts`：侧边栏扩展
- `docs-site/.vitepress/theme/index.ts`：自定义主题入口
- `docs-site/.vitepress/theme/console/`：SPA 子应用入口
- `docs-site/tests/console/`：vitest 单元 + Playwright e2e
- `docs-site/console/README.md`

### 12.2 cbmem-team 后端侧（Go）

- `internal/console/rbac.go`（新）
- `internal/console/config_handler.go`（新）
- `internal/console/logs_handler.go`（新）
- `internal/console/prefs_handler.go`（新）
- `internal/console/distill_handler.go`（新）
- 上述 5 个对应 `*_test.go`
- `internal/console/router.go`：挂 5 个新路由组
- `deploy/sql/schema*.sql`：新增 `user_prefs` / `kv_config` / `log_buffer` / `distill_records` 4 张表

### 12.3 文档

- `tools/cbmem-team/manual.md` 更新
- `tools/cbmem-team/manual.en.md` 同步
- `tools/cbmem-team/BUILD.md` 更新
- `tools/cbmem-team/CONSOLE.md` 更新
- `docs-site/README.md` 新段落
- `docs-site/console/README.md` 新文件

---

## 13. 决策记录

- **2026-07-12 rev 1**：选定独立 `web/` 工程。已被本稿（rev 2）取代。
- **2026-07-12 rev 2（当前）**：选定"在 docs-site 上叠加控制台"，理由：复用现有 Vue/EP 生态、单一构建、单一部署、文档与控制台同 UI。
- **2026-07-12**：RBAC 三角色（admin/operator/viewer）选定；前端 v-show + 后端中间件双层防御。
- **2026-07-12**：vue-flow 替代 mermaid 用于 M3 工作流编辑器（M4 candidates 不需要图）。

---

## 14. 下一步

> Spec written (rev 2). Awaiting user review and approval.
> 用户已指示"通过后执行"——所以批准后跳过 writing-plans，直接进入实施。
> 实施顺序：阶段 A → B → C → D → E（每阶段独立 commit）。

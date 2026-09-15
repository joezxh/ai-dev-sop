# Mem0 本地版与云端功能对齐设计

- 日期：2026-09-15
- 状态：已确认（用户批准）
- 依据文档：`docs/awesome-design-md/mem0-cloud-dashboard-design.md`（Mem0 Cloud 控制台实测分析）
- 范围：`mem0/server`（FastAPI 后端）与 `mem0/server/dashboard`（Next.js 前端）

## 1. 背景与已确认决策

本地版是在深度定制 Mem0 OSS 之上的自部署记忆服务（JWT/个人 API Key、用户/部门/项目多租户、git remote→projectId 匹配、MCP、供应商配置自愈）。目标是对照云端控制台的 22 项功能逐项判定必要性并对齐。

| 决策点 | 结论 |
| --- | --- |
| 产品定位 | **个人/小团队开发工具**：核心场景为 CodeBuddy/Qoder 等开发工具提供记忆服务；不做 SaaS 运营能力（计费/席位/促销）。 |
| 资源模型映射 | **部门≈组织、项目≈项目**：不加 Organization 层，云端组织级设置落到部门/全局设置。 |
| 对齐策略 | **方案 A：场景裁剪对齐**——每项云端功能显式判定（必要/可选/无需），不做全量复刻。 |

## 2. 功能对比与必要性判定清单

### 2.1 全局框架

| 云端功能 | 本地现状 | 判定 | 依据 |
| --- | --- | --- | --- |
| 组织切换器 | 无 | 可选(P1) | 部门≈组织；侧栏加部门/项目上下文切换即可，无多组织需求 |
| 侧边导航 SETUP/ACTIVITY/ACCOUNT 分组 | 分组混乱，CLOUD FEATURES 全为占位 | 必要(小改) | 重组导航、移除 PRO 占位，对齐云端 IA |
| 项目切换器（全站上下文） | Memories 页有项目下拉 | 可选(P1) | 各页已有过滤参数，非阻断 |
| 用量速览（配额进度条） | 无 | 无需实现 | 无配额/计费概念；统计并入总览页 |
| 帮助/Status/促销位 | 无 | 无需实现 | 本地部署无运营诉求 |

### 2.2 SETUP

| 云端功能 | 本地现状 | 判定 | 依据 |
| --- | --- | --- | --- |
| Install Mem0 向导 | 无 | 必要(P0) | 核心场景是开发工具接入；改造为「MCP 配置 + REST 示例」三语言（MCP/Python/cURL），含 API Key 创建与 git remote 占位 |
| Playground 调试沙盒 | 无 | 必要(P0) | 无代码验证记忆读写/抽取；`user_id=sandbox:*` 隔离实现"不碰生产数据" |
| API Keys | 已有且更强（用户级 Key、前缀脱敏、last_used_at） | 无需改动 | 已适配多租户，超出云端能力 |
| Copilot AI 配置助手 | 无 | 无需实现 | 需完整 LLM 编排链路；配置低频场景性价比低。远期 P2 |

### 2.3 ACTIVITY

| 云端功能 | 本地现状 | 判定 | 依据 |
| --- | --- | --- | --- |
| Dashboard 总览指标 | 无总览页 | 必要(P0) | `request_logs` 已有数据，补聚合端点即可；云端 recall score/Global median 无上游支持，不做 |
| Requests 请求日志 | 已有（auth_type/latency_ms） | 可选(小改) | 补操作类型过滤 + 时间范围 |
| Entities 实体 | 已有（Users/Agents/Runs） | 无需改动 | 云端 Apps/Sessions 对开发工具场景无用 |
| Memories 明细 | 已有且增强（项目/用户过滤、metadata、分页） | 可选(小改) | 云端 Changelog/Input 视图：history API 端点常量已存在，列为 P2 |
| Dream 记忆治理 | 无 UI | 无需实现 | Synthesis 为云端 Pro 引擎能力，上游无 API；Supersede/Merge 等价行为已内置于 mem0 add（LLM 自动去重/更新/删除） |
| Graph 图谱 | mem0ai[graph] 已装、Neo4j 已备、**未启用** | 可选(P2) | 需改 config + 图链路 + 可视化 UI；非刚需，本轮不启用 |
| Webhooks | 占位页 | 可选(P1) | 轻量实现（ADD/UPDATE/DELETE 事件 POST）；Categorize 事件上游不存在，不做 |
| Memory Exports | 占位页 | 可选(P1, 简化版) | 合规/迁移需求真实；Pydantic Schema 编辑器过重，做 JSONL 过滤导出即可 |

### 2.4 ACCOUNT

| 云端功能 | 本地现状 | 判定 | 依据 |
| --- | --- | --- | --- |
| Project General + Danger Zone | Projects 页已有 General；清空记忆无 UI | 必要(小改) | 补 Delete All Memories（二次确认） |
| Memory Extraction（Custom Instructions） | 无 | 必要(P1) | 记忆质量核心杠杆，上游原生支持 `prompt` 注入；落 Project 字段 |
| Categories 自定义分类 | 占位页 | 无需实现 | 云端为平台引擎能力，上游 OSS 无此机制，做了无法生效；价值并入 Custom Instructions |
| Retention：Expiration | 后端已支持 `expiration_date` 透传 | 必要(小改) | 只差项目级默认值与 UI |
| Retention：Memory Decay | 无 | 无需实现 | 上游引擎不支持访问加权衰减 |
| Org Members 三级角色 | 部门页已有成员；本地 admin/user 两级 | 无需大改 | 小团队两级角色够用（三级角色列为可选远期） |
| Profile / 外观 / 登出 | 已有 | 无需改动 | Delete Account 本地无意义 |
| Usage & Billing | 无 | 无需实现 | 无计费场景；用量统计并入总览 |

### 2.5 本地独有（保留并强化，不参与裁剪）

多租户（Users/Departments/Projects + 强隔离 + 项目共享池）、git remote→projectId 匹配、MCP 服务、供应商配置+测试维度自愈、中英 i18n。

## 3. 信息架构（导航重组）

```
SETUP 接入
  ├─ Get Started        # 新增（P0）
  ├─ Playground         # 新增（P0）
  └─ API Keys           # 已有
ACTIVITY 数据与监控
  ├─ Dashboard          # 新增（P0）
  ├─ Requests           # 已有 + 过滤增强（P1）
  ├─ Entities           # 已有
  ├─ Memories           # 已有
  ├─ Webhooks           # 占位 → 真实现（P1）
  └─ Memory Exports     # 占位 → 真实现（P1）
TENANT 租户管理
  └─ Users / Departments / Projects   # 本地独有
ACCOUNT 账号
  └─ Configuration / Settings
```

移除：`categories`、`analytics` 占位页、全部 PRO 徽标与云端导流 UpgradeBanner。LEARN 组不做（docs 外链留页脚）。

## 4. 后端设计

### 4.1 总览统计 `GET /stats?range=24h|7d|30d|90d|all`（admin only）

- 从 `request_logs` 聚合：请求数按操作类型（路径映射：`POST /memories`→ADD、`POST /search`→SEARCH、`GET /memories`→GET_ALL）与时间分桶；按 range 过滤时间窗。
- 记忆总量：`vector_store.list` 计数；实体数：按 user/agent/run 聚合（复用 entities 路由逻辑）。
- 响应：`{ requests: {add, search, get_all, total}, timeseries: [{t, count}], memories_total, entities: {users, agents, runs} }`。

### 4.2 项目级配置（Project 表扩展）

- 新字段：`custom_instructions TEXT NULL`、`memory_expiration_date DATE NULL`。
- `POST /memories`：解析出项目范围（显式 project_id 或 git_remote 命中）后：
  - `custom_instructions` 非空 → 作为 `prompt` 传给 mem0 抽取（显式请求里的 prompt 优先）；
  - `memory_expiration_date` 非空且请求未显式传 `expiration_date` → 采用项目默认。
- `PUT /projects/{id}` 接受两个新字段；Projects 编辑弹窗加输入。

### 4.3 Danger Zone：清空项目记忆

- `DELETE /memories?project_id=X`（admin only）：列出全部记忆 → 按 `metadata.project_id` 过滤 → 逐条删除；返回删除数。未提供 `project_id` → 400（禁止无范围全删）；项目不存在 → 404。

### 4.4 Webhooks（P1）

- 表：`webhooks(id, name, url, events JSON(list[str]), secret, created_at)`。
- 路由 CRUD（admin only）：`GET/POST/DELETE /webhooks`。
- 触发：`add_memory` 成功（按结果 event 类型映射 `memory_added`/`memory_updated`/`memory_deleted`）后 fire-and-forget `httpx.post(url, json=payload, headers={X-Webhook-Secret})`，5s 超时，失败仅记日志。事件名：`memory_added / memory_updated / memory_deleted`。

### 4.5 导出（P1）

- `GET /exports/memories?project_id&user_id&start&end&format=jsonl`（admin only）：复用项目池读路径 + 用户过滤 + 日期过滤（按 created_at），`StreamingResponse` 返回 JSONL 附件。

### 4.6 Playground 与向导

- 不新增后端：Playground 直调现有 `/memories`（`infer` 开关、`user_id=sandbox:<随机后缀>`）、`/search`；沙盒用户遵循现有隔离规则。代码片段全部为前端模板渲染。

## 5. 前端设计

| 页面 | 内容 |
| --- | --- |
| Get Started（`/dashboard/get-started`） | 三语言 Tab：MCP（生成 CodeBuddy/Qoder JSON 配置：`http://<host>:8080/mcp` + `api_key`/`git_remote` 占位）、Python SDK、cURL；四步引导（创建 API Key→配置工具→添加记忆→检索记忆）；空态页统一 CTA |
| Playground（`/dashboard/playground`） | Add/Search 双模式；会话消息编辑器（USER/ASSISTANT 翻转）；Extraction 开关（infer）；Custom instructions 试运行输入（随请求注入 prompt）；Metadata JSON（仅注入示例代码）；Output/Code 双视图；顶部「沙盒数据不进入项目记忆」横幅 |
| Dashboard 总览（`/dashboard/overview`） | 时间范围选择（24h/7d/30d/90d/All）+ 卡片（Total Memories / Requests by type / Entities）+ 快捷跳转 |
| Projects 页增强 | 编辑弹窗加 Custom Instructions（多行）与 Memory Expiration Date（日期）；列表行 Danger Zone：Delete All Memories（DeleteConfirmationModal 二次确认） |
| Requests / Memories | 操作类型过滤 + 时间范围（复用 range 语义） |
| Webhooks / Exports | 占位页替换为真实 CRUD / 导出表单（P1） |
| main-nav | 分组重组为 SETUP/ACTIVITY/TENANT/ACCOUNT；移除 categories、analytics、旧 export 占位与 PRO 徽标 |

## 6. 错误处理

- `/stats`、`/exports`、`/webhooks`、`DELETE /memories?project_id`：非管理员 → 403；资源不存在 → 404。
- Webhook 投递失败仅记录日志，不影响记忆写入主流程。
- Playground 沙盒遵循强隔离：admin 可指定 sandbox user_id；普通用户无该页入口（前端隐藏 + 后端权限既有规则）。
- 项目级 instructions/expiration 仅在项目范围解析成功时生效，静默降级不影响无项目记忆写入。

## 7. 测试与验证

- 新增 pytest：stats 聚合（含 range 边界与操作类型映射）；custom_instructions/expiration 注入（有/无项目范围、显式值优先）；项目清空只删匹配项；webhooks CRUD + 事件触发（mock httpx，事件订阅过滤）；导出过滤参数。
- 回归：既有 27 个测试全部通过。
- 容器内验证：重建 `mem0-api`/`mem0-dashboard`，pytest 全量 + smoke（stats 端点、Playground 请求、Projects 弹窗保存）。

## 8. 分期实施

- **P0（必要）**：导航重组与占位清理；Get Started；Playground；Dashboard 总览（`/stats`）；Project custom_instructions + expiration + Danger Zone。
- **P1（可选，建议做）**：Requests 过滤增强；Webhooks 真实现；JSONL 导出；部门/项目上下文切换器。
- **P2（远期可选）**：Graph 启用与可视化；记忆 Changelog（history API）；Copilot。
- **不做**：Billing/配额、三级席位角色、Categories 引擎级、Memory Decay、Dream Synthesis、促销位、多组织、Delete Account。

## 9. 非目标

- 不 fork mem0 上游以实现引擎级能力（Categories/Decay/Synthesis）。
- 不引入 Organization 新层。
- 本轮不启用 graph store（P2 再评估）。

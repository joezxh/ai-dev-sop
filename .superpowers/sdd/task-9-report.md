# Task 9 报告 — Playground 调试沙盒页 `/dashboard/playground`

**状态：完成**
**提交：`8edc8d50`**（mem0 子模块，分支 `feat/llm-provider-i18n`）
**提交信息：** `feat(dashboard): playground sandbox (add/search, infer switch, instructions)`
**改动：** 新增 1 个文件，229 行
- `mem0/server/dashboard/src/app/(root)/dashboard/playground/page.tsx`（新增）

## 实现内容

按 brief 的 Step 1 创建页面，复用现有组件与工具，未引入新依赖：

| 项 | 实现 |
| --- | --- |
| 模式切换 | Add Memories / Search Memories 双模式（Button 分段切换，切换时清空 Output） |
| 消息行 | 动态增删；点击角色按钮在 `USER` / `ASSISTANT` 间翻转；空内容行不提交 |
| infer 开关 | checkbox，默认 `true`，标签说明「开 = 抽取事实，关 = 逐字存储」 |
| Custom Instructions | Textarea，非空时以 `prompt` 字段随请求注入 |
| Output | `pre` 展示响应 JSON（缩进 2、最大高度 80、横向滚动） |
| 等效 cURL | 随模式生成，含 `messages`/`infer`/`user_id`/`prompt`，或 `query`/`user_id` |
| 沙盒隔离 | `user_id` 固定为 `sandbox-<8 位随机>`，顶部琥珀色声明「不会进入你的项目记忆」 |

后端契约已核对 `mem0/server/main.py`：
- `POST /memories`：`infer`（L246）、`prompt`（L248）均在 MemoryCreate 上，`user_id` 透传（L686-690）。
- `POST /search`：`SearchRequest.query` / `user_id`（L262-264，user_id 标记 deprecated 但仍可用）。
- `MEMORY_ENDPOINTS.BASE === "/memories"`（`src/utils/api-endpoints.ts:12`）；`api` 实例 baseURL 为 `NEXT_PUBLIC_API_URL`（compose 中 `http://localhost:8888`），Search 走相对路径 `"/search"`，最终命中 `http://localhost:8888/search`。
- 路由此前确实 404；`src/app/(root)/dashboard/components/main-nav.tsx:83` 与 `get-started/page.tsx:181` 已存在指向 `/dashboard/playground` 的链接，本任务补齐目标页。

## 对 brief 代码的两处修正（不影响契约）

1. **沙盒 user_id 改为稳定状态**（`useState<string>(newId)`）
   brief 中 `const sandboxUserId = newId();` 写在组件体内，每次渲染都会生成新 id：顶部展示的 id 会随输入抖动，且 Add 与 Search 会用不同 id，导致「先 add 再 search 搜不到刚写入的数据」。改为组件级 state 后，整个会话共用同一沙盒 id，add→search 可命中。附带一个「换一个」按钮用于手动重置沙盒隔离范围。
2. **Run 按钮加输入校验**：add 模式要求至少一条非空消息，search 模式要求非空 query，否则禁用（避免必然 400 的空请求）。

其余保持 brief 原样：未引入 i18n key（沿用 brief 的中英混排文案）、未使用 `import React`（与仓库其他页面一致，避免未使用变量）。

## 验证

| 项 | 结果 |
| --- | --- |
| `npx tsc --noEmit` | 通过（无输出） |
| `npx next lint --dir src/app/(root)/dashboard/playground` | `✔ No ESLint warnings or errors`（含 `react/no-unescaped-entities`，无未转义字符告警） |
| `npx prettier --write`（仓库 lint 实际是 `prettier --check .`） | 已格式化 |
| `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard` | `mem0-dashboard Built` + `Started`，构建日志无 error |
| 容器日志 | `✓ Ready in 371ms`，无 error |
| 未登录 `GET http://localhost:3001/dashboard/playground` | 307 → `/login?next=%2Fdashboard%2Fplayground`（预期，middleware 要求 `mem0_refresh_token`） |
| 带 cookie `mem0_refresh_token=probe` | **200**，响应体含 `Playground` |

## Concerns

1. **未在 UI 中实际点 Run 验证端到端**（需浏览器登录态）；本次仅验证到路由 200 + 页面渲染。Add 的 `results` 返回依赖后端 LLM/embedding 可用性，与前端无关，但建议人工在页面点一次确认。
2. **`user_id` 在 `/search` 已标记 deprecated**（服务端建议改用 `filters`）。当前仍可用，若后续后端移除该字段，Playground 的 Search 需改为 `filters: { user_id }`。
3. **沙盒 id 为前端随机生成、仅存在于当前会话**：刷新页面即换新 id，旧沙盒数据无法通过 Playground 再检索（仍存储于后端，需到 Memories 页按 user_id 查）。如需可复现调试，可后续改为可手动输入 user_id。
4. **Custom Instructions 只作用于本次请求**（以 `prompt` 传入），不会写回全局配置；全局配置在 `/dashboard/configuration`。文案已标注「试运行」。
5. CRLF/LF：提交时 Git 提示该文件 LF 将在下次被替换为 CRLF（Windows `core.autocrlf`），与其他仓库文件行为一致，非本次引入。

---

# Task 9 复审修复 — 2 Important + 4 Minor

**状态：完成**
**提交：`4f644186`**（mem0 子模块，分支 `feat/llm-provider-i18n`）
**提交信息：** `fix(dashboard): gate playground for non-admins and escape cURL snippet`
**改动：** 2 个文件，+64 / −18
- `mem0/server/dashboard/src/app/(root)/dashboard/playground/page.tsx`（修改）
- `mem0/server/dashboard/src/utils/api-endpoints.ts`（新增 `SEARCH_ENDPOINTS`）

**验证：** `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard` → `mem0-dashboard Built` + `Started`；构建日志 `✓ Compiled successfully in 33.4s` + `Linting and checking validity of types ...`（无 error/warning），21/21 静态页生成成功，无 error。

## Important 1 — 非 admin 必然 403

**问题：** 后端对非 admin 传入他人 `user_id` 返回 403，而沙盒 `user_id`（`sandbox-xxxx`）对任何登录用户都是「他人」，页面却未做角色收敛，非 admin 点 Run 必然失败且无解释。

**修复：**
- 引入 `const { isAdmin } = useAuth();`（先例 `configuration/page.tsx:35`；`isAdmin = user?.role === "admin"`，见 `src/lib/auth.tsx:125`）。
- `canRun = hasInput && isAdmin`，Run 按钮 `disabled={running || !canRun}`，`run()` 入口同样以 `canRun` 守卫（双保险）。
- Run 按钮右侧在非 admin 时追加说明文案「沙盒需管理员权限或 AUTH_DISABLED 部署。」。
- 顶部横幅改为条件渲染：admin 显示原隔离声明；非 admin 显示「当前账号不是管理员，Playground 已停用：沙盒所用的独立 user_id 对非管理员会被后端拒绝（403）。沙盒需管理员权限或 AUTH_DISABLED 部署才能运行」，不再声称可以写入沙盒（原文案「沙盒数据使用隔离的 user_id…不会进入你的项目记忆」在非 admin 下不准确）。

## Important 2 — cURL 单引号未转义

**问题：** 生成的 `-d '{...}'` 中，用户输入的消息/query/instructions 若含 `'` 会提前闭合单引号，截断命令（shell 注入面）。

**修复：** 新增 helper 并用它包裹整个 JSON body：

```ts
const shq = (s: string) => `'${s.replace(/'/g, "'\\''")}'`;
```

`codeSnippet` 改为 `-d ${shq(JSON.stringify(addBody))}` / `-d ${shq(JSON.stringify(searchBody))}`。同时把请求体提取为 `addBody` / `searchBody` 常量，供实际请求与 cURL 片段共用，消除两处 body 构造不一致的风险。

## Minor 1 — 硬编码 `/search`

`utils/api-endpoints.ts` 新增 `export const SEARCH_ENDPOINTS = { BASE: "/search" } as const;`；页面的 `api.post("/search", …)` 改为 `api.post(SEARCH_ENDPOINTS.BASE, …)`，cURL 片段同样引用该常量。

## Minor 2 — 失败时残留旧结果

`catch` 分支首行加 `setOutput("")`，避免上一次成功输出与本次错误 toast 同屏。

## Minor 3 — 「换一个」未清空结果

抽出 `resetSandbox()`：`setSandboxUserId(newId())` + `setOutput("")` + `setQuery("")`（换沙盒后旧 query 与旧沙盒 id 绑定，一并清空）。

## Minor 4 — `$API` 占位符

`const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "";`（与 `memories/page.tsx:38` 先例一致），cURL 中 `$API` → `${API_BASE_URL}`；鉴权头 `$KEY` → `X-API-Key: <你的 API Key>` 占位，并在 Output 卡片下补一行说明「页面实际走登录态 Bearer 令牌，`X-API-Key` 仅为命令行调用占位」。

## Minor 5 — a11y

- Custom Instructions：`Label` 加 `htmlFor="custom-instructions"`，`Textarea` 加对应 `id="custom-instructions"`。
- 消息行删除按钮（✕）加 `aria-label="删除该消息"`，消除纯图标按钮无可读名称的问题。

## 约束遵守

仅改动 `playground/page.tsx` 与 `utils/api-endpoints.ts`（后者仅新增常量）；未引入新依赖；未改动后端。

---

# Task 9 文案修正 — 移除 AUTH_DISABLED 误导表述

**状态：完成**
**提交：`322ba5a6`**（mem0 子模块，分支 `feat/llm-provider-i18n`）
**提交信息：** `fix(dashboard): correct playground admin-only wording`
**改动：** 1 个文件，+3 / −3
- `mem0/server/dashboard/src/app/(root)/dashboard/playground/page.tsx`（仅文案）

## 问题

页面两处文案（顶部非 admin 横幅约 108 行、Run 按钮旁约 245 行）写「沙盒需管理员权限**或 AUTH_DISABLED 部署**才能运行」，但门控为 `canRun = hasInput && isAdmin`，前端对 `AUTH_DISABLED` 无任何感知。该表述会让运维误以为在 `AUTH_DISABLED` 部署下可用，实际仍被前端禁用。

## 修改

两处统一改为「沙盒需要管理员权限才能运行（普通用户请改用自己项目的记忆写入）。」：

| 位置 | 原文案 | 新文案 |
| --- | --- | --- |
| 非 admin 横幅第二行 | 沙盒需管理员权限或 AUTH_DISABLED 部署才能运行；当前沙盒 user_id：`<code>` | 沙盒需要管理员权限才能运行（普通用户请改用自己项目的记忆写入）。当前沙盒 user_id：`<code>` |
| Run 按钮右侧提示 | 沙盒需管理员权限或 AUTH_DISABLED 部署。 | 沙盒需要管理员权限才能运行（普通用户请改用自己项目的记忆写入）。 |

门控逻辑未变：`canRun = hasInput && isAdmin`，Run 按钮 `disabled={running || !canRun}`，`run()` 内 `if (!canRun) return;` 保持原样；未改动其它文件。

## 验证

| 项 | 结果 |
| --- | --- |
| `git diff` 范围 | 1 文件，3 增 3 删，均为文案行（无逻辑/门控改动） |
| Lint（IDE 诊断） | 0 error / 0 warning |
| `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard` | `mem0-dashboard Built` → `Container mem0-dashboard Recreated` → `Started`，构建日志无 error |

## Concerns

1. 文案层面已消除误导，但**未验证 `AUTH_DISABLED` 部署下前端 `isAdmin` 的实际取值**（取决于 `useAuth` / `/api/v1/auth/status` 或 user payload）。若后端在该模式下返回的角色不是 `admin`，即使后端允许写入，Playground 仍会被前端禁用——即「文案说必须 admin」与「后端可能放行」仍存在潜在不一致。本次按约束仅改文案，未做 `AUTH_DISABLED` 感知；如需真正对齐，应另开任务在 `useAuth` 侧暴露该标志并调整门控。
2. 未做浏览器人工点击验证（仅构建 + lint），文案渲染位置与换行由 JSX 文本节点决定，风险极低。

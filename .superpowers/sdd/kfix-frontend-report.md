# mem0 前端修复报告（最终审查 Known issues 可修项）

- 分支：`feat/llm-provider-i18n`（mem0 子模块）
- 提交 1：`f67709a6` `fix(dashboard): useApiQuery deps option; single-source context key; write-back filter`
- 提交 2：`d66922fd` `fix(dashboard): pagination clamp, i18n selector, copy buttons, metadata JSON in playground`
- 构建：`docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard` → `mem0-dashboard Built` / `Started`，容器 `Ready in 303ms`，无 error（镜像构建含 Next.js 类型检查与 lint，全部通过；工作区 lint 检查 0 诊断）。
- 约束遵守：未触碰 `server/` 下除 `server/dashboard` 之外的任何后端文件。

## 各项结论

| 项 | 结论 | 说明 |
|----|------|------|
| F1 | ✅ 完成 | `use-api-query.ts` 增量新增可选 `deps?: unknown[]`，effect 依赖改为 `[enabled, run, depsKey]`（deps 经 `JSON.stringify` 得到稳定 `depsKey`，避免每次渲染重跑；未传 deps 时行为与现状完全一致，注释说明了用途与旧闭包坑）。`memories/page.tsx` 改用 `deps: [userId, projectId, projectContextLoaded]` 自动重拉，删除 Enter/onChange 里的手动 `refetch()`（输入状态保留）；`requests/page.tsx` 的 `appliedFilters` ref + effect 方案无损迁移为 `deps: [action, range]` 并删除对应 effect（useRef/useEffect 导入一并清理）；`overview/page.tsx` **保持现状**，理由见下。memories/requests 顺带补了页码 clamp（过滤变化后 `page` 可能越界，原手动方案里有 `setPage(0)`，迁移后由 clamp 承担） |
| F2 | ✅ 完成 | `PROJECT_CONTEXT_KEY = "mem0-project-context"` 定义于 `src/utils/api-endpoints.ts`，`main-nav.tsx` 与 `memories/page.tsx` 均删除本地常量改为导入 |
| F3 | ✅ 完成 | `en.ts`/`zh.ts` 新增 `tenant.allProjects`（"All projects"/"全部项目"），main-nav 下拉默认项改用 `t("tenant.allProjects")` |
| F4 | ✅ 完成 | memories 页项目下拉 onChange 同步 `localStorage.setItem(PROJECT_CONTEXT_KEY, value)`（try/catch 包裹，与侧栏切换器对称；userId 不写） |
| F5 | ✅ 完成 | `departments/projects/users/webhooks` 四页均在 `load()` 内 `setItems` 后执行 `setPage((p) => Math.min(p, Math.max(0, Math.ceil(next.length / PAGE_SIZE) - 1)))`。搜索/保存/删除共用同一 load，非删除路径不受影响（搜索 Enter 本就先 `setPage(0)`，clamp 只在越界时收敛） |
| F6 | ✅ 完成 | webhooks 提交前 `!name.trim()` → `setFormError("Enter a webhook name.")` 并 return，不发请求（HTML `required` 放不住纯空白输入，故做显式前端校验） |
| F7 | ✅ 完成 | get-started 代码块外加 `relative` 容器，右上角复制按钮，使用已有依赖 `react-copy-to-clipboard`（api-keys/setup/login 同款先例），lucide Copy/Check 图标 + 2 秒"已复制"按钮态切换。三段代码（MCP JSON/Python/cURL）共用该按钮，随 Tab 切换复制当前块 |
| F8 | ✅ 完成 | playground add 模式新增「Metadata JSON（可选）」`Textarea`：输入非空即解析，合法 JSON 对象并入请求 `metadata`，非法/非对象则 `metadataError` 提示解析错误并禁用 Run（`canRun = hasInput && isAdmin && !metadataError`），非法输入永不发送。cURL 片段由 `addBody` 生成，自动包含 metadata，且整个 body 经 `shq` 转义 |
| F9 | ✅ 完成 | main-nav 的 projects `useApiQuery` 改为 `enabled: isAdmin && !isSidebarCollapsed`（折叠展开切换会触发重拉，符合预期可接受） |

## 放弃/缩减项及理由

- **overview/page.tsx 未迁移到 deps（F1 可选子项）**：该页的 `prevRange` ref 守卫是刻意设计——注释明确说明用它规避 React StrictMode 双挂载时的重复请求。迁到 `deps` 后挂载时 effect 会按 depsKey 触发，dev StrictMode 下将产生一次额外请求，属行为变化，按任务约定保持现状。生产环境无影响，且该页现方案功能正确（range 变化即以最新 fetcher 重拉）。
- **memories 页 userId 输入为"边输边查"**：deps 方案下每次击键触发一次请求（原方案 Enter 才查）。这是任务指定的 `deps: [userId, ...]` 的直接结果，与 requests 页既有筛选下拉的即时行为一致；未加防抖以避免引入超时参数等计划外行为。页码越界由 clamp 兜底。

## 验证

- `git status` 干净；两笔提交均不含文件删除（`--diff-filter=D` 无意外项）。
- 容器日志：`✓ Ready in 303ms`，无 error 输出。

---

# mem0 前端复审修复报告（2 Important + 2 Minor）

- 分支：`feat/llm-provider-i18n`（mem0 子模块）
- 提交：`4260fc1a` `fix(dashboard): request-sequence guard, mode-scoped metadata error, webhook type alignment`（3 files changed, +36 −6）
- 验证：`docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard` → `✓ Compiled successfully`（Next.js 15.5.21 生产构建，含类型检查与 lint），22/22 页面生成成功，`mem0-dashboard Built` / `Recreated` / `Started`，退出码 0，无 error。工作区 read_lints 对四个改动文件 0 诊断。
- 约束遵守：仅改动 `playground/page.tsx`、`use-api-query.ts`、`types/api.ts` 三个文件（webhooks 页面经核对无需改动，见 Minor ① 说明）。

## 逐项说明

| 项 | 结论 | 说明 |
|----|------|------|
| I-01 | ✅ 完成 | `playground/page.tsx` 的 `canRun` 改为 `hasInput && isAdmin && (mode === "add" ? !metadataError : true)`，并附注释说明 metadata 只参与 add 请求、仅在 add 模式约束 Run。选择"仅 add 模式受约束"方案：metadata 错误提示本就渲染在 add 卡片内（Textarea 下方），add 模式下错误与 Run 禁用同屏可见，切换到 Search 后 Run 不再被残留的非法 metadata 永久禁用；切回 add 时错误提示与禁用状态一并恢复——行为自洽 |
| I-02 | ✅ 完成 | `use-api-query.ts` 的 `run` 内新增模块无关的组件级 `reqIdRef = useRef(0)`，每次调用 `const requestId = ++reqIdRef.current`；成功路径 `setData(result)`、错误路径 `setError`/toast 均在 `requestId !== reqIdRef.current` 时直接 return（过期响应不覆盖新数据也不触发旧错误的 toast）；发起时以本请求 id 为最新才 `setIsLoading(true)`/`setError("")`；finally 中仅 `requestId === reqIdRef.current` 才 `setIsLoading(false)`（过期请求迟到完成不会隐藏新请求的 loading）。未传 `deps` 的页面单请求语义不变（发起必为最新，正常走完整状态流转） |
| Minor ① | ✅ 完成 | 先读后端 `mem0/server/routers/webhooks.py` 确认契约：`WebhookResponse`（GET 列表）返回 `secret_masked: str | None`，`WebhookCreatedResponse`（POST 创建）额外含一次性 `secret: str | None`。`types/api.ts` 的 `Webhook` 相应改为 `secret_masked: string | null` 并加注释；同时保留可选 `secret?: string`（仅 create 响应携带，注释说明）。`webhooks/page.tsx` 经全文检索确认从未读取列表项的 `secret` 字段（页面里的 `secret` state 是表单输入，创建响应被丢弃、无一次性明文展示逻辑），故该页面无需改动，类型上的可选 `secret` 已与后端 create 响应对齐 |
| Minor ③ | ✅ 完成 | `playground/page.tsx` 的 `resetSandbox` 追加 `setMetadataJson("")`，换沙盒时同步清空 metadata 输入，避免残留非法 JSON 挡住下一次 add 运行（与清空 `query`/`output` 行为对称） |

## 验证

- 构建输出无 error；`git show --stat 4260fc1a` 确认仅 3 个目标文件、无删除。

---
phase: kfix-frontend-fixup
reviewed: 2026-09-16T12:30:00Z
depth: deep
files_reviewed: 5
files_reviewed_list:
  - mem0/server/dashboard/src/app/(root)/dashboard/playground/page.tsx
  - mem0/server/dashboard/src/hooks/use-api-query.ts
  - mem0/server/dashboard/src/types/api.ts
  - mem0/server/dashboard/src/app/(root)/dashboard/webhooks/page.tsx
  - mem0/server/routers/webhooks.py
findings:
  critical: 0
  warning: 0
  info: 2
  total: 2
status: clean
---

# 复审报告：mem0 dashboard 修复提交 4260fc1a

**Reviewed:** 2026-09-16T12:30:00Z
**Depth:** deep（含跨文件调用链核对）
**Files Reviewed:** 3 个改动文件 + 2 个交叉核对文件（webhooks 页面 / 后端 webhooks 路由）
**Status:** clean（0 Critical / 0 Warning，2 Info）

## Summary

复审对象为提交 `4260fc1a`（`fix(dashboard): request-sequence guard, mode-scoped metadata error, webhook type alignment`，3 files changed, +36 −6）。已用 `git show --stat 4260fc1a` 核对实际提交与差异包 `kfix-diff-v2.txt` 完全一致（blob hash 相同），无夹带改动。构建通过采信报告（Next.js 15.5.21 生产构建 ✓ Compiled successfully，22/22 页面，退出码 0）。

## 逐项验证结论

### 1. I-01：canRun 按模式收敛 — ✅ 成立

- `playground/page.tsx:78-79`：`hasInput && isAdmin && (mode === "add" ? !metadataError : true)` —— add 受 `!metadataError` 约束，search 明确不受约束。
- 无「Search 静默禁用」残留路径：`metadataError` 仅由 `metadataJson` 派生（:44-56）；`metadataJson` 不进入 `searchBody`（:65）；错误提示只渲染在 add 卡片内（:255）；`run()` 的 `!canRun` 早退（:94）在 search 模式下不再被非法 metadata 触发。
- 与 `resetSandbox` 清空 metadataJson（见第 4 项）配合，模式切换后无残留约束。

### 2. I-02：reqIdRef 请求序号守卫 — ✅ 成立

逐条核实 `use-api-query.ts:48-75`：

- **setData / setError 过期早退**：成功路径 `:56`、失败路径 `:60` 均以 `requestId !== reqIdRef.current` 早退；错误 toast 在早退之后才执行，过期请求不会弹窗。
- **isLoading**：每次 `run` 开始置 true（:51）；`finally` 仅最新请求清除（:73），过期请求晚到不会熄灭新请求的 spinner。
- **无死锁**：任意时刻「最新」请求的 `finally` 必然执行清除；被超越的请求跳过清除不会阻塞后来者。超时挂起场景 isLoading 保持 true 属正确语义。enabled 翻转 / StrictMode 双触发 / overview 页 range 切换 refetch 均推演无卡死。
- **未传 deps 的单请求语义不变**：`depsKey` 无 deps 时为常量 `null`（:79）；`run` 的 useCallback 依赖仅 `[errorToast]`（:75），核对全部 6 个消费方（api-keys / entities / configuration / main-nav / requests / memories / overview）传入的 errorToast 均为字面量字符串，`run` 身份稳定 → effect 仅挂载时触发一次。
- 跨页核对：memories 页 `deps: [userId, projectId, projectContextLoaded]`、requests 页 `deps: [action, range]` 配合守卫，慢响应覆盖新结果的原缺陷已被修复。

### 3. Webhook 类型对齐 — ✅ 成立（1 条 Info）

- 后端 `routers/webhooks.py:39-52`：`WebhookResponse` 字段为 `id/name/url/events/secret_masked/created_at`，`WebhookCreatedResponse` 仅在 POST 创建响应附加 `secret: str | None = None`。
- 前端 `types/api.ts:67-80`：`secret_masked: string | null`、`secret?: string`（带一次性明文语义注释）、`created_at: string` —— 与后端一致。
- 全 dashboard 源码检索确认：**无任何对已删 `secret` 字段的引用**（webhooks 页面表格仅渲染 name/url/events/created_at，表单 `secret` 为本地输入 state，与响应类型无关）。

### 4. resetSandbox 清 metadataJson — ✅ 成立

`playground/page.tsx:86-91`：`setMetadataJson("")` 已加入，与 sandboxUserId/query/output 一起复位，符合「换一个沙盒后不残留非法 metadata 约束」的意图。

### 5. 无新增缺陷 / 构建通过 — ✅ 成立

构建采信报告（退出码 0，类型检查与 lint 通过）；本次复审发现的 2 条 Info 均为无害冗余/精度问题，不构成缺陷。

## Info（非阻断）

### IN-01: run() 入口处守卫条件恒为真（冗余代码）

**File:** `mem0/server/dashboard/src/hooks/use-api-query.ts:50`
**Issue:** `const requestId = ++reqIdRef.current; if (requestId === reqIdRef.current)` —— 自增与比较同步执行，条件恒为 true，该 if 分支属死代码。行为正确（isLoading/error 正常重置），仅是冗余。
**Fix:** 可简化为无条件 `setIsLoading(true); setError("");`，或删除该 if 保留直写；不影响当前正确性，可留待后续清理。

### IN-02: Webhook.secret 类型与后端序列化精度略差

**File:** `mem0/server/dashboard/src/types/api.ts:78`
**Issue:** 后端 `WebhookCreatedResponse.secret: str | None = None`，未传 secret 时 JSON 序列化为 `null`；前端声明 `secret?: string` 不含 `null`。当前无任何页面读取该字段，无运行时影响，仅为类型精度问题。
**Fix:** 改为 `secret?: string | null;` 以精确匹配创建响应。

---

_Reviewed: 2026-09-16T12:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_

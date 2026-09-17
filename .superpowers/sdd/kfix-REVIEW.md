# mem0 Known Issues 修复 · 对抗性复审

**基准:** d2f5e426 → HEAD（4 提交：da573fda / 13f872a8 / f67709a6 / d66922fd）
**深度:** deep（跨文件 + 前后端契约）
**结论: REQUEST_CHANGES**

## 逐项核验结果

### 后端 K1–K6：全部通过

- **K1 ✅** `routers/webhooks.py:39-64` 列表仅返回 `secret_masked`（前4+***+后4，≤8 全掩码），`WebhookCreatedResponse` 仅创建时返回一次性明文。`webhooks_client.py:33,44` 先序列化 body 字符串、`content=body` 发送、`sign_body(secret, body)` 对同一字节串计算 HMAC-SHA256 —— 签名与传输字节一致，实现正确。
- **K2 ✅** `routers/webhooks.py:81-117`：IP 字面量免 DNS、域名走 `getaddrinfo`、**任一**解析结果命中 private/loopback/link-local 即拒（严格语义，报告已声明偏离）、解析失败 OSError → 400、`WEBHOOK_ALLOW_PRIVATE_NET` 调用时读取（可测）。既有功能无回归（公网域名创建用例在）。
- **K3 ✅** 非 2xx（`>=300`）记 warning；模块级 `ThreadPoolExecutor(4)`；update/delete payload 带 `id`（`main.py:971,993`）。
- **K4 ✅** `ALLOWED_EVENTS = tuple(_EVENT_MAP.values())` 单源于 webhooks_client.py:12；空订阅 400（webhooks.py:122-123）。
- **K5 ✅** `ACTION_METHOD_PATH`/`API_KEY_AUTH_TYPES` 单源于 stats.py:364,369，requests.py 反向导入并经 `_METHOD_PATH_TO_ACTION` 派生 `request_action`，语义与 SQL 谓词一致；`is` 同一性测试锁定。
- **K6 ✅** `SCAN_LIMIT=10_000`（server_state.py:10）；export/delete 统一用它；**admin 列表路径未误改**（`main.py:718` `ALL_MEMORIES_LIMIT=1000`，top_k ≤ 1000 与 `get_all_memories` 两处调用点原样）；`x-total-scanned`/`truncated` 语义未变（仍为"实际扫描行数 / 扫描量达到上限"），测试经 `monkeypatch.setattr(m, "SCAN_LIMIT", N)` 收窄——因 main.py 函数在运行时查模块全局，补丁生效路径正确。stats SQL 聚合 3×count，fake db 对谓词真实求值且对未知谓词显式 fail（非恒真）。

### 前端 F1–F9

- **F1 ⚠️ 见 I-02**：`deps` 为增量可选——未传时 `depsKey=null`，effect 依赖 `[enabled, run, null]` 与旧行为等价，**所有未迁移页面无回归面**（overview 页经 `git diff d2f5e426..HEAD -- overview/` 确认零改动）。memories/requests 迁移干净：旧的 `setPage(0)+refetch()` 手动方案及 `appliedFilters` ref+effect 均无残留（requests 页 `refetch` 仍被刷新按钮使用，无死代码）。但迁移本身引入乱序风险（I-02）。
- **F2 ✅ / F3 ✅ / F4 ✅** 单源键、i18n 词条（en/zh 各 1 条）、onChange 写回 localStorage（try/catch）均落地。
- **F5 ✅** departments/projects/users/webhooks 四页 clamp 覆盖，公式对空列表收敛到 0，正常翻页不受影响（仅越界收敛）。
- **F6 ✅**（`!name.trim()` 前端拦截）；**F7 ✅**（已有依赖 react-copy-to-clipboard，无新依赖）；**F9 ✅**。
- **F8 ⚠️ 见 I-01**：非法 JSON 永不发送、Run 禁用、cURL 含 metadata——主路径正确，但存在跨模式缺陷。

## 发现

### Critical

无。

### Important

**I-01 Playground：Search 模式下 Run 被无提示禁用**
`playground/page.tsx:76` `canRun = hasInput && isAdmin && !metadataError` 对两种模式生效，但错误提示只渲染在 add 模式卡片内（:251），且 `switchMode`（:78-81）不清空 `metadataJson`。在 add 模式输入非法 JSON → 切到 Search → 输入查询 → Run 灰置，界面无任何错误说明，用户无从排查。
**Fix:** `canRun = hasInput && isAdmin && (mode === "search" || !metadataError)`；或 `switchMode` 清空 `metadataJson`；或把错误提示移到 Run 按钮上方全局渲染。

**I-02 useApiQuery 无乱序防护，memories 页逐键 deps 放大成 stale data**
`use-api-query.ts:41-67` 的 `run` 无请求序号/AbortController：先发后至的旧请求会在 `setData` 时覆盖新响应（isLoading 亦被先完成者置 false）。原方案（Enter 触发、下拉触发）请求频率低，风险可忽略；本次迁移使 memories 页 `deps: [userId, ...]` **每次击键发一次请求**，快速输入时旧 filter 的慢响应覆盖新 filter 数据的概率显著上升，直接违背本次迁移"始终展示最新 filter 数据"的目标。
**Fix:** `run` 内加序号守卫：`const seq = ++runSeqRef.current; ... const result = await fetcherRef.current(); if (seq !== runSeqRef.current) return; setData(result);`

### Minor

**M-01 dashboard `Webhook` 类型与后端契约漂移** — `types/api.ts:68` 仍声明 `secret: string | null`，后端列表已改为 `secret_masked`；当前页面未渲染该字段故无运行时错误，但运行时读到 `undefined` 而类型声称 `string | null`，属埋雷。Fix: 改为 `secret_masked: string | null`。

**M-02 memories 页写 localStorage 后侧栏不同步** — `memories/page.tsx` onChange 持久化 `PROJECT_CONTEXT_KEY`，但 `main-nav.tsx` 仅在 mount 读取一次，切换后侧栏选择器显示旧值直到整页刷新。与报告"与侧栏切换器对称"的说法不完全成立（持久化对称、显示不对称）。Fix: 侧栏监听 `storage` 事件或改为受控同步。

**M-03 resetSandbox 不清空 metadataJson** — playground 重置沙盒后旧 metadata 仍并入新 add 请求，与"沙盒重置"语义不符。

**M-04 `func.rtrim(path, "/")` 跨库可移植性** — `stats.py:406` 双参 rtrim 在 SQLite/PostgreSQL 语义一致（去尾字符集），MySQL 不支持双参 RTRIM。当前栈 SQLite 可接受，跨库移植时需替换为 `TRIM(TRAILING '/')`。

**M-05 `_webhook_executor` 退出注释与实际相反** — `main.py:93-97` 注释称"进程退出时在途任务丢失"；Python 3.9+ 的 ThreadPoolExecutor 为非 daemon 线程且经 atexit join，实际退出时会**等待**队列排空（受 5s 超时 × 队列长度影响）。仅为文档性问题，不阻塞发布。

## 残留 known issues（本轮后仍不修）

1. `X-Webhook-Secret` 明文头仍随请求发送（兼容保留，报告已声明；建议下版本默认关闭/移除，签名头已就位）。
2. SSRF 仅在创建时解析校验，投递时 httpx 重新解析 → DNS rebinding（TTL=0 公私网翻转）可绕过；重定向向量已被 httpx 默认不跟随缓解。彻底修复需投递侧 IP 校验（自定义 connect/transport）。
3. overview 页未迁移 deps（StrictMode 双请求的刻意取舍，声明成立）。
4. memories 页 userId 逐键请求无防抖（声明的取舍，与 I-02 叠加）。
5. 前端 `WEBHOOK_EVENTS` 与后端 `ALLOWED_EVENTS` 跨语言双源（仅注释同步）。

## 测试证据采信说明

92 passed、变异自查（search 谓词改 GET → 5 failed）、dashboard Built/Started 均采信报告，未重跑。fake db 谓词求值实现经人工复核：IN 为展开 BindParameter（tuple value）、rtrim/method/created_at 谓词逐一求值、未知谓词显式 AssertionError——变异敏感声明成立。

---

_Reviewed: 2026-09-17 · Reviewer: adversarial code review (gsd-code-reviewer) · Depth: deep_

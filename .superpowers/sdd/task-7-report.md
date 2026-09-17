# Task 7 Report — Dashboard 总览页 `/dashboard/overview`

**状态：** 完成
**提交：** `b59f934f`（分支 `feat/llm-provider-i18n`，仓库 `mem0`）
**提交信息：** `feat(dashboard): overview page backed by /stats`

## 改动文件

| 文件 | 类型 | 说明 |
| --- | --- | --- |
| `mem0/server/dashboard/src/utils/api-endpoints.ts` | 修改 | 在 `REQUEST_ENDPOINTS` 之后新增 `STATS_ENDPOINT = { BASE: "/stats" }` |
| `mem0/server/dashboard/src/app/(root)/dashboard/overview/page.tsx` | 新增 | 总览页（client component） |

## 实现要点

- 页面为 `"use client"`，通过 `useApiQuery` + `api.get(STATS_ENDPOINT.BASE, { params: { range } })` 拉取数据，loading 态用 `TableSkeleton rows={3} columns={3}`。
- 时间范围按钮组 `24h / 7d / 30d / 90d / all`，默认 `7d`；选中态 `variant="default"`，其余 `variant="outline"`。
- 卡片 6 张，点击跳转：Total Memories→`/dashboard/memories`；Add/Search/Get All Requests→`/dashboard/requests`；Users/Agents→`/dashboard/entities`。
- 未改动 `main-nav`（Task 6 已指向 `/dashboard/overview`）。

### 与 brief 的差异（1 处，必要）

brief 的代码只把 `range` 放进 fetch 闭包，但 `useApiQuery` 的 effect 依赖是 `[enabled, run]`，只在挂载时执行一次 —— 切换 range 不会重新请求。因此在页面内增加了一个本地 effect：以 `refetch` + `range` 为依赖、用 `useRef` 跳过首次挂载，从而在切换时间范围时重新拉取（未修改共享 hook，避免影响其它页面）。

## 验证结果

1. **构建/启动**
   `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard`
   → `mem0-dashboard  Built` / `Recreated` / `Started`，无 error；容器日志 `✓ Ready in 306ms`（Next.js 15.5.21）。
   容器内确认路由已编译：`.next/server/app/(root)/dashboard/overview/page.js` 存在。
2. **后端契约**
   `Invoke-RestMethod "http://localhost:8888/stats?range=7d"` → 200：
   ```json
   { "requests": { "add": 2, "search": 0, "get_all": 0, "total": 2 },
     "memories_total": 0,
     "entities": { "users": 0, "agents": 0, "runs": 0 } }
   ```
   （api 容器已是含 `routers/stats.py` 的镜像，无需 `docker cp`。）
3. **页面可访问**
   - 匿名：`GET http://localhost:3001/dashboard/overview` → `307`，`location: /login?next=%2Fdashboard%2Foverview`（middleware 预期行为，非构建错误）。
   - 携带 `mem0_refresh_token` cookie：`GET /dashboard/overview` → `200`，HTML 含页面内容。

## Concerns

1. `requests.total` 与 `entities.runs` 已在后端返回但按 brief 未展示（卡片只列 6 项）；如需 Total Requests / Runs 卡片，后续一行即可加。
2. 请求失败时仅有 `errorToast` 提示，页面会渲染空网格（沿用 brief）。
3. 客户端直连 `NEXT_PUBLIC_API_URL=http://localhost:8888`，与现有 requests 页面一致，依赖后端 CORS 配置。

---

## 后续修复（审查发现项）

**提交：** `bc343225` — `fix(dashboard): harden overview page (typed response, race guard, all-time labels)`

| 文件 | 说明 |
| --- | --- |
| `mem0/server/dashboard/src/app/(root)/dashboard/overview/page.tsx` | 主要修复 |
| `mem0/server/dashboard/src/types/api.ts` | 新增 `Stats` 接口（类型下沉，与 `Memory`/`ApiRequestLog` 等一致，单一来源） |

### Important

1. **响应防御**：`api.get<Stats>(...)` 显式泛型；6 张卡片全部可选链 + `?? 0` 兜底（`stats?.memories_total`、`stats?.requests?.add/search/get_all`、`stats?.entities?.users/agents`），结构异常不再白屏。
2. **切档竞态**：range 按钮加 `disabled={isLoading}`，in-flight 期间禁止再发起请求，避免过期响应覆盖新结果（未改共享 hook `use-api-query.ts`）。
3. **语义误导**：后端 `range` 只过滤 requests，故 `Total Memories（All time）`、`Users（All time）`、`Agents（All time）` 标题显式标注全量，与 Requests 三张卡区分。

### Minor

- 去掉 `isFirstLoad` + `useEffect`，改为渲染期 `prevRange` ref 比较，消除 dev StrictMode 下的重复请求。
- 骨架 `TableSkeleton rows={2} columns={3}`（6 张卡 / 3 列 = 2 行）。
- 卡片数值 `value.toLocaleString()`。
- range 按钮 `aria-pressed={r === range}`。
- 删除未使用的 `import React`。
- 本地 refetch 逻辑上方加注释，说明依赖 `useApiQuery` 的两个内部约定（`refetch` 身份稳定、`fetcherRef` 在 render 阶段更新）。

**验证：** `tsc --noEmit` 无错误；`docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard` → `mem0-dashboard Built / Recreated / Started`，无 error。

---

## 二次修复（审查未通过的 2 项 Important）

**提交：** `5e4e34ff` — `fix(dashboard): move overview refetch into effect and render error state`

| 文件 | 说明 |
| --- | --- |
| `mem0/server/dashboard/src/app/(root)/dashboard/overview/page.tsx` | 唯一改动文件（12 insertions / 11 deletions） |

未改动共享 `use-api-query.ts`，符合约束。

### 1. 渲染期副作用 → effect + 值比较（Important）

删除渲染体内的 `if (prevRange.current !== range) { ...; void refetch(); }`，改为：

```tsx
const prevRange = useRef(range);
useEffect(() => {
  if (prevRange.current === range) return;
  prevRange.current = range;
  void refetch();
}, [range, refetch]);
```

- 重拉不再发生在渲染阶段，恢复渲染纯度。
- 值比较（而非 `isFirstLoad` 挂载标志）：StrictMode 双挂载时两次执行均 `prevRange.current === range` → 早退，不多发请求（保留了上一轮 Minor 修复的收益）。
- 依赖 `[range, refetch]`，`refetch` 由 `useCallback([errorToast])` 保持稳定身份，无循环。
- 不再依赖 `useApiQuery` 内部约定（`fetcherRef` 在 render 期重赋值），与共享 hook 解耦；`useEffect` 已加回 import。

> 注：此项是对上一轮 `bc343225` 中 Minor「改为渲染期 ref 比较」的修正——该写法虽消除了 StrictMode 重复请求，但引入了渲染期副作用，故回退到 effect，仅保留值比较这一正确部分。

### 2. 失败态不再显示 0（Important）

从 `useApiQuery` 解构 `error`（与 `data`/`isLoading` 同层，类型为 string，空串表示无错误），渲染分支改为三级：

```tsx
{error ? (
  <p className="text-sm text-destructive">Failed to load stats.</p>
) : isLoading ? (
  <TableSkeleton rows={2} columns={3} />
) : (
  /* 6 张卡片网格 */
)}
```

- 请求失败时显示错误文案，6 张卡不再渲染 `0`，与「真的全为 0」可区分。
- `api.get<Stats>` 显式泛型与 `?? 0` 兜底均保留，仅用于防结构异常；失败态优先走 `error` 分支。
- `text-destructive` 为既有主题 token（`form.tsx`、`badge.tsx`、`alert.tsx` 已使用），无新增样式依赖。

### 验证

1. `npx tsc --noEmit`（`mem0/server/dashboard`）→ exit 0，无类型错误。
2. `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard`
   → `mem0-dashboard Built` / `Recreated` / `Started`，无 error；`docker ps` 状态 `Up`；日志 `✓ Ready in 280ms`（Next.js 15.5.21）。
3. 提交无文件删除，工作区干净；仅 1 file changed。

### 状态

两项 Important 均已修复并通过验证，审查阻塞项清零。残留意向性说明：`?? 0` 兜底是为防御后端结构异常而保留的（当次请求成功但字段缺失时仍显示 0），非失败态掩盖。

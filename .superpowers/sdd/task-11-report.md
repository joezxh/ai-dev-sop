# Task 11 Report — Requests 请求日志 action / range 过滤

**状态:** 完成 (DONE)

## 提交

| 范围 | 哈希 | 说明 |
| ---- | ---- | ---- |
| 后端 | `9556e9dc` | `feat(server): requests action/range filters` — `server/routers/requests.py` + `server/tests/test_stats.py` |
| 前端 | `c13c25ea` | `feat(dashboard): requests action and range filters` — `server/dashboard/src/app/(root)/dashboard/requests/page.tsx` |

（均在 `mem0` 子模块分支 `feat/llm-provider-i18n` 上，`--no-verify`。）

## 实现

### 后端 `server/routers/requests.py`
- 新增 `from datetime import datetime, timedelta, timezone` 与 `from routers.stats import RANGE_CHOICES, RANGE_TO_HOURS, request_action`；`fastapi` 导入补 `HTTPException`。
- `list_requests` 新增两个查询参数：
  - `action: str | None = Query(default=None)`（`add | search | get_all`）
  - `range: str = Query(default="all")`（`24h | 7d | 30d | 90d | all`）
- 非法 `range` → `HTTPException(400, "range must be one of ('24h', '7d', '30d', '90d', 'all')")`。
- `range != "all"` 时按 `RANGE_TO_HOURS[range]` 计算 `since`（UTC）并在 SQL 侧过滤 `RequestLog.created_at >= since`。
- `action` 在 Python 侧用 `request_action(log.method, log.path) == action` 过滤（与 `/stats` 口径一致）。
- 保留原有 `order_by(created_at.desc())`，`limit` 在 action 过滤**之后**再切片（`logs[:limit]`），避免「先截断再过滤」导致结果不完整。

### 前端 `server/dashboard/src/app/(root)/dashboard/requests/page.tsx`
- 新增 `action`（默认 `""`，即 All actions）与 `range`（默认 `"all"`）两个 state。
- 请求参数：`{ limit: REQUEST_LOG_LIMIT }`，`action` 非空时带 `action`，`range !== "all"` 时带 `range`。
- 头部新增两个 `<select>`（All actions / add / search / get_all；All time / 24h / 7d / 30d / 90d），样式沿用 Memories 页项目过滤的 `border border-memBorder-primary rounded-md px-3 py-2 text-sm bg-transparent`，并带 `aria-label`。
- 过滤变更时 `setPage(0)` 并重新拉取。

### 测试 `server/tests/test_stats.py`（brief 未要求，自行补齐）
复用文件内已有的 `_Log` / `_Result` / `_Scalars` / `_DB` 假 DB，直接调用 `requests_router.list_requests(...)`：
- `test_list_requests_filters_by_action` — `action="search"` 只返回 `/search` 行
- `test_list_requests_returns_all_when_action_is_none` — `action=None` 返回全部 3 行
- `test_list_requests_rejects_bad_range` — `range="1h"` 抛 400

## 验证

- 单测：`docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/ -q` → **60 passed**（含新增 3 条），0 failed。
- 重建 dashboard：`docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard` → `mem0-dashboard Built` + `Container mem0-dashboard Started`，无 error（构建日志无 error/失败层）。
- 冒烟（宿主机 → localhost:8888）：
  - `GET /requests?action=search&range=24h` → **200** `[]`
  - `GET /requests?action=search&range=7d` → **200**
  - `GET /requests` → **200**
  - `GET /requests?range=1h` → **400**（非法 range 被拒）

## 偏差 (Deviations)

1. **[Rule 1 - Bug] 过滤下拉不能在 `onChange` 里直接 `refetch()`**
   - brief 要求沿用 Memories 页的「`setXxx()` 后立刻 `refetch()`」写法，但 `useApiQuery` 内部是 `fetcherRef.current = fetcher`（渲染期赋值），`run()` 同步调用 `fetcherRef.current()`，在同一个事件处理函数里 `setState` 尚未触发重渲染，因此 `refetch()` 读到的仍是**旧**的过滤值——下拉切换不会生效。
   - 修复：在页面内增加 `appliedFilters` ref + `useEffect([action, range, refetch])`，仅在过滤值真正变化后（重渲染完成）才 `setPage(0)` + `refetch()`；首挂载与 StrictMode 双调用均不会触发多余请求。改动仅限本页文件，未触碰共享 hook。
   - 影响：Memories 页的项目/用户过滤存在同一问题，属既有缺陷、超出本任务范围，未一并修改（见 Concerns）。

2. **[Rule 3 - 顺序] `limit` 在 action 过滤之后应用**
   - brief 的代码片段丢弃了 `limit` 与 `order_by`。保留 `order_by(created_at.desc())` 以维持原接口语义，并把 `limit` 切片移到 Python 过滤之后，否则 `limit=200` 先截断再过滤会漏数据。

## Concerns

1. **Memories 页过滤疑似同样失效**：`useApiQuery` 的 stale-fetcher 问题同样影响 `memories/page.tsx` 的 `userId` / `projectId` 过滤与回车搜索。建议另开任务统一修复（改 hook：把 fetcher 依赖化，或让 `refetch(args)` 支持传参）。
2. **`action` 非法值不报错**：按 brief 在 Python 侧过滤，未知 action 会返回空列表而非 400（与 `range` 行为不一致）。是否需要 400 由产品侧决定，未擅自加校验。
3. **action 过滤在内存中进行**：请求日志量大时（无 limit 的全表扫描）会有性能开销。当前日志表规模有限，可接受；若后续量大，建议把 `request_action` 的映射下推成 SQL `WHERE`（或落库时写入 `action` 列）。
4. **容器不是热挂载**：`mem0-api` 镜像把源码烘焙进镜像，本任务为跑测试用 `docker cp` 注入；镜像重建后注入会丢失，需重新 `cp` 或走镜像构建。dashboard 已走镜像重建。
5. **`docker compose up -d --build mem0-dashboard` 连带重建了 `mem0-api`**（depends_on），因此测试后再重启并重新注入了一次 `requests.py`，冒烟验证基于注入后的运行实例。

---

## 修复轮 — In-scope Important（后端两项）

**提交:** `80b9bdd5` — `fix(server): push requests action filter to SQL and restore limit; cover range filtering`
（分支 `feat/llm-provider-i18n`，`--no-verify`；文件：`server/routers/requests.py`、`server/tests/test_stats.py`，+128/−20）

### 修复 1 — 全表扫描回归（action 下推 SQL + 恢复 limit）

- 新增 `ACTION_METHOD_PATH`（`add | search | get_all` → `(method, path)`），口径与 `routers.stats.request_action` 一致。
- `action` 过滤由 Python 侧列表推导改为 SQL 谓词：
  `and_(RequestLog.method == method, func.rtrim(RequestLog.path, "/") == path)`；
  路径尾部 `/` 用 SQL `rtrim` 归一化，等价 `request_action` 里的 `rstrip("/")`。
- 恢复 `order_by(RequestLog.created_at.desc()).limit(limit)`，LIMIT 回到 SQL 侧——默认 `range="all"` 不再退化成「无时间窗 + 无 LIMIT 的全表扫描 + 全量 ORM 实例化」。
- 删除已下推的 Python 侧 `[lg for lg in logs if request_action(...)]` 过滤及不再使用的 `request_action` 导入。

### 修复 2 — 时间过滤零覆盖（假 DB 记录 statement）

- `_DB` 增加 `self.statements = []`，`execute(self, stmt, ...)` 追加 stmt；新增 `_sql()` helper
  （`stmt.compile(compile_kwargs={"literal_binds": True})`），让断言读到真实字面量而非 `:param_1`。
- 用例覆盖：
  - `range="24h"` → 语句含 `created_at >=`；`range="all"` → 不含。
  - `range="24h"` 的时间窗绑定值 ≈ 24h 前（`23h59m < delta <= 24h1m`，测试内硬编码 24h，可捕捉 `RANGE_TO_HOURS` 写错小时数）。
  - `action="search"` → 语句含 `POST` + `/search` + `rtrim`；三个 action bucket 全覆盖。
  - `action=None` → 无 `rtrim` 谓词；未知 action → 返回 `[]`。
  - `LIMIT 50` / `LIMIT 7` 与 `ORDER BY ... created_at DESC` 存在。
- **变异验证**（逐个改坏实现后跑 `tests/test_stats.py`，确认非恒真）：
  移除 `created_at >=` → 2 failed；`RANGE_TO_HOURS["24h"]` 改 240 → failed；去掉 action 谓词 → failed；
  去掉 `limit()` → failed；去掉 `order_by` → failed。

### 顺带清理

- `("all",) + RANGE_CHOICES` → `RANGE_CHOICES`（后者已含 `"all"`，原写法重复）。

### 偏差

1. **[Rule 2 - 输入校验] 未知 `action` 保留「返回空列表」**：下推 SQL 后 `ACTION_METHOD_PATH[action]` 对未知值会 `KeyError` → 500（旧行为是 Python 过滤后返回 `[]`）。改用 `.get()` + `return []`，保住既有行为、避免 500；是否统一为 400 仍待产品侧决定（见 Concerns #2）。

### 验证

- 单测：`docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/ -q` → **67 passed**（原 60，+7）。
- 运行库为 **PostgreSQL**，`rtrim(text, text)` 可用；已在运行实例冒烟（宿主机 → `localhost:8888`）：
  - `GET /requests?action=search&range=24h` → 200 `[]`
  - `GET /requests?action=add&range=7d` → 200（有数据）
  - `GET /requests?action=get_all` → 200 `[]`
  - `GET /requests` → 200（有数据）
  - `GET /requests?action=bogus` → 200 `[]`（未知 action，非 500）
  - `GET /requests?range=1h` → **400**

### Concerns 更新

- 原 #3「action 过滤在内存中进行」已消除。
- 原 #2 仍成立：未知 `action` 返回 `[]` 而非 400。
- 源码仍以 `docker cp` 注入容器可写层，镜像重建后需重新注入（同原 #4）。

---
phase: task-11-requests-action-range-filters
reviewed: 2026-09-16T04:37:19+08:00
depth: standard
files_reviewed: 3
files_reviewed_list:
  - mem0/server/routers/requests.py
  - mem0/server/dashboard/src/app/(root)/dashboard/requests/page.tsx
  - mem0/server/tests/test_stats.py
findings:
  critical: 0
  warning: 3
  info: 7
  total: 10
status: issues_found
---

# Task 11: Requests 操作类型/时间范围过滤 — Code Review Report

**Reviewed:** 2026-09-16T04:37:19+08:00
**Depth:** standard
**Files Reviewed:** 3（另交叉核对 `routers/stats.py`、`models.py`、`main.py`、`hooks/use-api-query.ts`、`dashboard/overview/page.tsx`、`dashboard/memories/page.tsx`）
**Commits:** `9556e9dc` (server) + `c13c25ea` (dashboard)，基准 `31fa248e` 已核对
**Status:** issues_found

## Summary

后端过滤逻辑正确复用了 `request_action` / `RANGE_CHOICES` / `RANGE_TO_HOURS`，非法 range 返回 400，`auth_type` 白名单与 `limit` 均保留。前端两个下拉、`params` 拼装、DataTable/分页结构均符合 brief。

实现者最重要的一处偏差（Deviation #1：不在 `onChange` 里直接 `refetch()`）**判断正确且必要**——已核对 `hooks/use-api-query.ts:28-35`，`fetcherRef.current = fetcher` 在渲染期赋值、`run()` 同步读取，同 tick 内 `setState` + `refetch()` 必然读到旧闭包；并且 `overview/page.tsx:27-35` 已有完全相同的 `prevRange` ref + `useEffect` 先例，因此本页写法与代码库既有模式一致，不是自创 hack。

主要问题不在功能正确性，而在**一处被报告低估的回归**：为了在 Python 侧做 action 过滤，SQL 的 `.limit()` 被整个移除，`range=all`（本接口的默认值）时会把 `request_logs` 全表拉进内存。旧代码有 SQL LIMIT 保护，这是实打实的回归，不是"既有性能特征"。其次，本任务另一半核心（时间范围 → `created_at >= since`）**零测试覆盖**，而报告声称的 3 条用例中有一条信息量很低。

无 Critical 项：无注入面（全走 SQLAlchemy 参数绑定）、`require_admin` 未被绕过、无硬编码密钥、lint 干净。

## Critical Issues

无。

## Warnings

### WR-01: 移除 SQL `LIMIT` 导致默认路径全表加载（回归）

**File:** `mem0/server/routers/requests.py:43-52`

**Issue:**
改动前 SQL 是 `.order_by(...).limit(limit)`，服务端最多读 `limit`（≤200）行。改动后为在 Python 侧做 action 过滤，`.limit()` 被完全移除：

```python
logs = db.execute(stmt.order_by(RequestLog.created_at.desc())).scalars().all()
if action:
    logs = [log for log in logs if request_action(log.method, log.path) == action]
return logs[:limit]
```

`range` 的默认值是 `"all"`，因此**默认调用就是无时间窗 + 无 LIMIT 的全表扫描**，并把所有行实例化成 ORM 对象。请求日志是写多读少的高增长表（每条 API 请求一行），长时间运行的实例上这是 OOM/拖垮 API 容器的现实路径。

报告 Concern #3 把它写成"当前日志表规模有限，可接受"——这个判断掩盖了它是**本次改动引入的回归**而非既有特性，且修复成本只有几行。

**Fix（推荐：把 action 下推到 SQL，同时恢复 SQL LIMIT）：**

```python
from sqlalchemy import and_, func, or_, select

_ACTION_RULES = {"add": ("POST", "/memories"), "search": ("POST", "/search"), "get_all": ("GET", "/memories")}

# 与 stats.request_action 的 rstrip("/") 语义保持一致
_path_eq = lambda p: func.rtrim(RequestLog.path, "/") == p

def _action_clause(action: str):
    method, path = _ACTION_RULES[action]          # 未命中 → KeyError，交给 400 校验
    return and_(RequestLog.method == method, _path_eq(path))

stmt = select(RequestLog).where(RequestLog.auth_type.in_(API_KEY_AUTH_TYPES))
if range != "all":
    stmt = stmt.where(RequestLog.created_at >= since)
if action:
    stmt = stmt.where(_action_clause(action))
logs = db.execute(stmt.order_by(RequestLog.created_at.desc()).limit(limit)).scalars().all()
return logs
```

**Fix（最小改动版，若暂不想下推映射）：** 保留 Python 过滤，但给扫描加上界，保证内存有界：

```python
ACTION_SCAN_CAP = 5_000
stmt = stmt.order_by(RequestLog.created_at.desc())
stmt = stmt.limit(ACTION_SCAN_CAP if action else limit)
logs = db.execute(stmt).scalars().all()
if action:
    logs = [log for log in logs if request_action(log.method, log.path) == action][:limit]
return logs
```

### WR-02: 时间范围过滤（本任务核心之一）零测试覆盖

**File:** `mem0/server/tests/test_stats.py:80-106`

**Issue:** 3 条新增用例覆盖了 `action` 过滤（`test_list_requests_filters_by_action`：把实现改回不过滤会失败，✅ 有效）和非法 range 的 400（`test_list_requests_rejects_bad_range` ✅ 有效），但**没有任何一条触及 `range != "all"` 的时间窗分支**。由于 `_DB.execute()` 直接返回全部 logs、忽略 `stmt`，以下几类破坏都不会让任何测试变红：

- 删掉 `stmt = stmt.where(RequestLog.created_at >= since)` 这一行；
- `RANGE_TO_HOURS[range]` 写错 / 用成 `RANGE_CHOICES`；
- `timedelta(hours=...)` 写成 `days=`；
- `datetime.now(timezone.utc)` 改成 naive `utcnow()`（在 `DateTime(timezone=True)` 列上会出问题）。

同样地，Deviation #2（`limit` 移到过滤之后）这条实现决策也没有任何测试锁定，改回 SQL `.limit()` 测试照样全绿。

**Fix：** 让假 DB 记录被执行的 statement，对 where 子句做断言（改动不破坏现有用例，`execute(self, stmt, *_a, **_k)` 对现有位置参数调用兼容）：

```python
class _DB:
    def __init__(self, logs):
        self._logs = logs
        self.statements = []

    def execute(self, stmt, *_a, **_k):
        self.statements.append(stmt)
        return _Result(self._logs)


def test_list_requests_applies_time_window_for_range():
    db = _DB(_REQUEST_LOGS)
    requests_router.list_requests(_auth=None, db=db, limit=50, action=None, range="24h")
    where = str(db.statements[0].whereclause)
    assert "created_at" in where          # 时间窗被下推


def test_list_requests_all_range_has_no_time_window():
    db = _DB(_REQUEST_LOGS)
    requests_router.list_requests(_auth=None, db=db, limit=50, action=None, range="all")
    assert "created_at" not in str(db.statements[0].whereclause)


def test_list_requests_slices_limit_after_action_filter():
    out = requests_router.list_requests(
        _auth=None, db=_DB(_REQUEST_LOGS), limit=1, action="add", range="all"
    )
    assert [log.path for log in out] == ["/memories"]   # 不是先截断再过滤后的空列表
```

### WR-03: Memories 页 stale-fetcher 缺陷已核实为真（Task 11 范围外）

**File:** `mem0/server/dashboard/src/app/(root)/dashboard/memories/page.tsx:126-139`

**Issue:** 报告 Concern #1 声称 Memories 页的过滤同样失效——**已核实属实**：

```tsx
onKeyDown={(e) => { if (e.key === "Enter") { setPage(0); refetch(); } }}
...
onChange={(e) => { setProjectId(e.target.value); setPage(0); refetch(); }}
```

两处都是在同一个事件处理里 `setState` 后立刻 `refetch()`，`useApiQuery` 的 `run()` 会同步调用 `fetcherRef.current()`，而 `fetcherRef.current` 只在渲染期更新 → 读到的仍是旧 `projectId` / 旧搜索词。这是**功能失效**（过滤下拉点了没反应），级别应为 Important，不是"疑似"。

同意不在 Task 11 内修（会触碰共享 hook 或他人页面），但建议立即开单，并把改 hook（`refetch(args)` 支持传参，或把 fetcher 依赖化）作为根治方案——否则后续每加一个过滤下拉都要在页面里复制一遍 ref + effect 样板。

## Info

### IN-01: `action` 非法值静默返回空列表，与 `range` 的 400 不一致

**File:** `mem0/server/routers/requests.py:50-51`
**Issue:** `action="nonsense"` → 返回 `[]`；`range="1h"` → 400。同一接口两个枚举类参数行为不一致；拼写错误（如 `get-all`、`getall`）会被静默吞掉，前端表现为"无数据"而非报错，排查成本高。
**Fix:** 级别 = Minor（UI 下拉已限定取值，不构成安全问题）。若要统一，可在 `request_action` 的取值集合上加校验：

```python
VALID_ACTIONS = ("add", "search", "get_all")
if action is not None and action not in VALID_ACTIONS:
    raise HTTPException(status_code=400, detail=f"action must be one of {VALID_ACTIONS}")
```

注意 `action=""` 应继续视为"不过滤"（前端 All actions 用空串），所以判空用 `is not None` 而非真值判断。

### IN-02: `test_list_requests_returns_all_when_action_is_none` 信息量很低

**File:** `mem0/server/tests/test_stats.py:94-98`
**Issue:** 假 `_DB` 无条件返回 3 行，`limit=50` 不可能截断，`range="all"` 不可能抛错，所以 `assert len(out) == 3` 几乎只等价于"函数没炸"。它唯一能挡住的是"`action=None` 时仍执行过滤"。不算恒真断言（那种是 `assert True` 级别），但作为一条独立用例性价比低。
**Fix:** 保留即可（作为 test-1 的对照），但建议改成断言具体顺序/内容而不是 `len`，例如 `assert [log.path for log in out] == ["/memories", "/search", "/memories"]`，顺带锁定"未过滤时保持 DB 顺序"这一契约。

### IN-03: `API_KEY_AUTH_TYPES` 重复定义

**File:** `mem0/server/routers/requests.py:28` vs `mem0/server/routers/stats.py:16`
**Issue:** 两份完全相同的 `("api_key", "admin_api_key")`。既然本次已经从 `routers.stats` 导入了三个符号，却唯独留下这个常量在本地重复定义，日后 stats 侧改白名单时 requests 侧会静默漂移。
**Fix:** `from routers.stats import API_KEY_AUTH_TYPES, RANGE_CHOICES, RANGE_TO_HOURS, request_action`，删除本地第 28 行。（或反过来把这个常量提到共享模块再让 stats 复用——两者都行，只要只剩一份。）

### IN-04: 校验条件里 `"all"` 重复

**File:** `mem0/server/routers/requests.py:39`
**Issue:** `RANGE_CHOICES = ("24h", "7d", "30d", "90d", "all")` 本身已含 `"all"`，`("all",) + RANGE_CHOICES` 得到 `("all", "24h", "7d", "30d", "90d", "all")`。功能无影响（错误消息 `range must be one of ('24h', ..., 'all')` 也是对的），只是冗余。
**Fix:** 写成 `if range not in RANGE_CHOICES:`（与 `stats.py:49` 一致）即可。

### IN-05: `_REQUEST_LOGS` 是模块级共享可变列表

**File:** `mem0/server/tests/test_stats.py:80-84`
**Issue:** 三条用例共用同一个 list 实例。当前无人修改它所以无害，但后续若有用例需要 `.append()` 或改 `path`，会跨用例污染。
**Fix:** 改成一个工厂函数 `def _request_logs(): return [_Log(...), ...]`，各用例调用自己的副本。

### IN-06: 连续切换下拉时的响应竞态

**File:** `mem0/server/dashboard/src/app/(root)/dashboard/requests/page.tsx:130-140`
**Issue:** `useApiQuery` 没有请求序号/AbortController，快速连切下拉会产生并发请求，`setData` 后到者胜。若两个请求返回顺序颠倒，界面会停留在旧过滤值的数据上，且无任何 loading 提示残留可察觉。
**Fix:** Minor（hook 层既有缺陷，与 WR-03 同源）。本次不需要修；若要顺手加固，可在 effect 里用一个自增 token 丢弃过期结果：

```tsx
const reqId = useRef(0);
// ...
const id = ++reqId.current;
void refetch().then(() => { if (id === reqId.current) setPage(0); });
```

### IN-07: Overview 的三张请求卡片没有深链到新过滤器

**File:** `mem0/server/dashboard/src/app/(root)/dashboard/overview/page.tsx:41-43`
**Issue:** "Add Requests / Search Requests / Get All Requests" 三张卡片都 `href="/dashboard/requests"`，不带 `?action=`。本任务刚给 Requests 页加了 action 过滤，却没有任何入口使用它。纯增强建议，非缺陷。
**Fix:** 若要接上，链接改成 `/dashboard/requests?action=add|search|get_all`，并在 Requests 页用 `useSearchParams()` 初始化 `action` state（同时初始化 `appliedFilters` ref，避免挂载即触发一次多余 refetch）。

---

## 实现者 Concerns 级别裁定

| Concern | 报告自述级别 | 裁定 | 说明 |
| --- | --- | --- | --- |
| #1 Memories 页过滤失效 | 疑似 / 超范围 | **Important（真实缺陷）**，同意超范围 | 已在 `memories/page.tsx:126-139` 核实（见 WR-03） |
| #2 非法 action 返回空列表 | 待产品定 | **Minor** | UI 已限定取值，非安全问题；建议与 range 统一为 400（IN-01） |
| #3 action 过滤在内存中做 | 可接受 | **Important（低估，且是回归）** | 默认 `range=all` 无 LIMIT 全表扫描，旧代码有 SQL LIMIT（见 WR-01） |
| #4 容器非热挂载 / `docker cp` 注入 | 流程提示 | **流程风险** | 见下方 Cannot verify——冒烟证据与镜像构建产物不同源 |
| #5 compose 连带重建 mem0-api | 说明 | 无害 | 已重新注入后再冒烟，无代码影响 |

---

## ⚠️ Cannot verify from diff

1. **冒烟 `GET /requests?action=search&range=24h` → `[]` 不具判别力。** 空数组同时兼容三种情况：(a) 过滤正确但该时间窗内无 api_key 的 search 请求；(b) `request_action` 从未匹配上任何行（例如实例部署在路径前缀之后，`main.py:333` 的 `request.url.path` 会带上前缀，`/memories` 就永远匹配不上——这种情况下 `/stats` 的三个计数也会恒为 0，属既有隐患）；(c) 过滤逻辑写错。需要一条"库里确实有 search 日志且 `action=search` 返回非空"的证据才能真正证明端到端生效。
2. **后端代码是 `docker cp` 注入进容器跑的**（Concern #4），即 `9556e9dc` 的代码并未经过镜像构建验证，且镜像重建后会丢失。因此"dashboard 重建成功"与"后端冒烟通过"来自两个不同源的运行时，无法从 diff 证明镜像产物包含本次后端改动。
3. **60 passed 采信报告，未复核。** 未验证新增 3 条是否真的在跑测集合内、以及容器内的 `requests.py` 是否为注入后的最终版本。
4. **跨文件：路由前缀假设。** `request_action` 硬编码匹配 `/memories` / `/search`（`stats.py:19-28`）。已核对 `main.py:652/729/886` 路由确实注册在根路径，当前部署形态下成立；但一旦置于反向代理路径前缀后即失效，无法从 diff 排除。

---

_Reviewed: 2026-09-16T04:37:19+08:00_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

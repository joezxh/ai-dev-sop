---
phase: task-11-requests-action-range-filters
reviewed: 2026-09-16T05:30:00+08:00
depth: standard
files_reviewed: 3
files_reviewed_list:
  - mem0/server/routers/requests.py
  - mem0/server/tests/test_stats.py
  - mem0/server/routers/stats.py
findings:
  critical: 0
  warning: 0
  info: 0
  total: 0
status: clean
---

# Task 11 复审报告（修复轮 v2）

**Reviewed:** 2026-09-16T05:30:00+08:00
**Depth:** standard
**Files Reviewed:** 3（另交叉核对 `models.py`、`main.py`、`db.py`、第一轮 `task-11-REVIEW.md`）
**Diff:** `task-11-diff-v2.txt`（基准 `c13c25ea`，修复提交 `80b9bdd5`）
**Status:** clean — **Spec: ✅ / Quality: Approved**

## Summary

仅验证第一轮两项 Important（WR-01、WR-02）是否闭环。结论：**两项均已闭环，且未发现本轮修复引入的新缺陷**。验证方式：读取仓库当前实现与测试源码（非仅采信 diff），并交叉核对落库口径（`main.py` 日志中间件）、数据库方言（`db.py` → PostgreSQL）、列类型（`models.py`）与 `RANGE_CHOICES` 取值。

## Important #1（WR-01 全表扫描回归）— ✅ 已闭环

`mem0/server/routers/requests.py`，逐项核实：

| 要求 | 核实结果 |
| --- | --- |
| action 过滤下推 SQL | ✅ `:62-64` `and_(RequestLog.method == method, func.rtrim(RequestLog.path, "/") == path)` |
| 恢复 `order_by(...).limit(limit)` | ✅ `:66` SQL 侧 LIMIT 回归，`range="all"` 默认路径不再全表扫描 |
| Python 侧过滤移除 | ✅ 列表推导与 `request_action` 导入均已删除（`:9-10` 现仅导入 `RANGE_CHOICES, RANGE_TO_HOURS`） |
| `range` 非法 → 400 | ✅ `:47-50`；顺带清理 `("all",) + RANGE_CHOICES` → `RANGE_CHOICES` **合法**——已核对 `stats.py:14`，`RANGE_CHOICES = ("24h", "7d", "30d", "90d", "all")` 含 `"all"`，默认值不受影响 |
| `auth_type` 白名单保留 | ✅ `:51` `RequestLog.auth_type.in_(API_KEY_AUTH_TYPES)` |

**口径等价性核查（新增验证，diff 之外）：**
- `rtrim(path, "/")` 在 PostgreSQL（`db.py:13` 确认方言）为 `rtrim(text, characters)` 双参形式，语义与 Python `rstrip("/")` 一致（移除尾部所有 `/`）。
- 落库 `path` 来自 `request.url.path`（`main.py:383`），不含 query string，`rtrim` 匹配无失配面。
- `created_at` 为 `DateTime(timezone=True)`（`models.py:57`），与 `datetime.now(timezone.utc)` 的 aware 比较正确。
- 未知 `action` 用 `.get()` + `return []`（`:56-60`），保持旧行为，未引入 `KeyError`→500。

## Important #2（WR-02 时间窗零覆盖）— ✅ 已闭环

`mem0/server/tests/test_stats.py`，逐项核实：

| 要求 | 核实结果 |
| --- | --- |
| 假 `_DB` 记录 statement | ✅ `:35-39` `self.statements.append(stmt)` |
| 断言时间窗 where | ✅ `:108-114`（`24h` 含 `created_at >=`）、`:117-122`（`all` 不含） |
| 断言 action 条件在 SQL 中 | ✅ `:142-150`（`POST` + `/search` + `rtrim`）、`:153-162`（三个 bucket 全覆盖，期望值硬编码于测试） |
| LIMIT / ORDER BY 锁定 | ✅ `:182-196`（`ORDER BY` + `created_at DESC` + `LIMIT 50` / `LIMIT 7`） |

**非恒真独立复核（不采信报告的变异验证，自行推演）：**
- 若实现回退为 Python 侧过滤 → SQL 中无 `rtrim`/`POST`/`/search` 谓词，`:148-150` 必红；
- 若删掉时间窗 → `:114` 必红，且 `:131-135` 的 `next()` 在无 datetime 绑定参数时抛 `StopIteration` → 用例报错；
- 若 `RANGE_TO_HOURS["24h"]` 写错 → `:137-139` 硬编码 24h 窗口断言必红（测试不依赖实现常量）；
- 若去掉 `.limit()` / `.order_by()` → `:188-190` 必红。
以上断言均作用于假 DB 记录的**真实 SQLAlchemy statement**（`literal_binds` 编译），与实现行为耦合，非恒真。✅

**67 passed：** 按指示采信报告（60 → 67，+7 与新增用例数一致）。

## 新增缺陷扫描

本轮 diff（实现 + 测试）未发现新的 Critical / Warning 级缺陷。备注两点（均非缺陷，不计 finding）：

1. `ACTION_METHOD_PATH`（requests.py:32-36）与 `stats.request_action` 的映射属重复定义，但漂移风险已被测试缓解——测试侧硬编码期望值同时锁定两侧；且这正是第一轮 WR-01 修复建议采纳的方案。
2. 第一轮 Info 项（IN-01 未知 action 非 400、IN-03 `API_KEY_AUTH_TYPES` 重复、IN-05/06/07）**均不在本轮闭环范围，状态不变**，继续以第一轮裁定为准。

## 结论

- **Important #1（WR-01）：Closed** — action 下推 SQL、LIMIT/ORDER BY 恢复、Python 过滤移除、400 校验保留，且语义等价性经方言/落库口径/列类型三面核实。
- **Important #2（WR-02）：Closed** — statement 记录 + SQL 文本断言 + 时间窗数值校验，非恒真成立。
- **新增缺陷：无**。

**Spec: ✅ / Quality: Approved**

---

_Reviewed: 2026-09-16T05:30:00+08:00_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard（复审：仅验证两项 Important 闭环 + 新增缺陷扫描）_

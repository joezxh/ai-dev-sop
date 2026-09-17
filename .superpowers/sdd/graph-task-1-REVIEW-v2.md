---
phase: graph-task-1-fix-verification
reviewed: 2026-06-27T00:00:00Z
depth: deep
files_reviewed: 2
files_reviewed_list:
  - mem0/server/graph_memory.py
  - mem0/server/tests/test_graph_memory.py
findings:
  critical: 1
  warning: 1
  info: 3
  total: 5
status: issues_found
---

# Graph Task 1（修复提交 683151e9）复审报告

**Reviewed:** 2026-06-27
**Depth:** deep（差异包 graph-task-1-diff-v2.txt，基准 5a9cf86b；全文件重读 + 跨函数调用链核查 + Neo4j 官方文档交叉验证）
**Files Reviewed:** 2
**Status:** issues_found

## 结论

**Spec: ✅ / Quality: NOT Approved（1 BLOCKER + 1 WARNING + 3 INFO）**

修复清单逐项核对结果：

| # | 修复项 | 落盘核对 | 判定 |
| --- | --- | --- | --- |
| CR-01 | 节点身份 (name, user_id, project_id) | MERGE 键已含 project_id（graph_memory.py:217/221）、无条件 SET 已移除；消歧仅限 `_scope_match` 同身份；`_scope_match/_scope_params` scope 驱动 + `coalesce` 谓词（:115-133） | **字面成立，但存在 BLOCKER（见 BLK-01）** |
| CR-02 | hard_delete 无 scope → warning + no-op | :339-345 防护完整；`test_hard_delete_without_scope_is_noop` 回退防护即红；project 谓词仅 project_id（`test_project_hard_delete_uses_project_predicate` 断言 user_id 不在 params） | ✅ 闭环 |
| IMP-01 | threshold float 在 try 内 | :93-99 ✅（日志问题见 IN-01） | ✅ 闭环 |
| IMP-02 | add 缺 user_id → warning + 空返回 | :200-204 ✅ | ✅ 闭环 |
| IMP-03 | 关系端点 "i"→user_id | :163-167 ✅，`test_extract_relations_maps_self_reference` 回退映射即红 | ✅ 闭环 |
| IMP-04 | 软删三处谓词由 `_scope_match` 驱动 | :316-328 ✅，含无 user_id 且无 project_id 防护 :310-312 | ✅ 闭环 |
| IMP-05 | `_norm` 折叠空白 | :33 ✅ | ✅ 闭环 |
| Minor ①②③ | BM25 except Exception 同构兜底 :283-286；数字开头关系跳过 :170-174；移除 datetime 导入 | ✅ | ✅ 闭环 |
| 无新增缺陷 | — | **不成立**：发现 BLK-01 | ❌ |

## BLOCKER（必须修复后方可入库）

### BLK-01: 个人作用域 `add` 的 MERGE 键含 null project_id，Neo4j 运行时必然报错 —— 每次个人写入都是无效操作

**File:** `mem0/server/graph_memory.py:217,221,237`
**Issue:**
`add` 的节点身份键写为 `MERGE (a:__Entity__ {name: $source, user_id: $user_id, project_id: $project_id})`，而 params 中 `"project_id": filters.get("project_id")` 在个人作用域（无 project_id，或为 None）下为 **null**。

Neo4j 官方 Cypher Manual（MERGE 章节 "null properties" 一节）明确：

> **"MERGE cannot be used with a graph element property value that is null."**

模式属性绑定为 null 时查询**直接报错**（错误码 22N31，旧版本文案即 `Cannot merge node using null property value for 'project_id'`），**并非**修复报告所声称的"Cypher MERGE 按 null 匹配"。该异常被 `add` 的降级 `except Exception` 吞掉，仅记 `add failed (degraded)` 日志后返回空。

**实际后果：所有个人作用域（未携带 project_id）的 `add` 在真实 Neo4j 上 100% 落库失败**——实体、关系、embedding 全部不写入，且表现为静默降级，无任何功能告警。修复报告中的 105 passed 无法暴露此缺陷，因为 FakeNeo4j 只记录 Cypher 字符串、不执行 Cypher；4 个新回归用例断言的均是 params 字典内容，与 MERGE 的真实执行语义无关。这也说明本项目缺少至少一条走真实 Cypher 执行（或语义仿真）的用例。

**Fix:**
个人维度需将 MERGE 键中的 project_id 物化为可匹配的非 null 值（与 `_scope_match` 的 `coalesce(n.project_id,'')` 语义一致）：

```python
params = {
    ...
    "user_id": filters.get("user_id"),
    # MERGE 键不可为 null（Cypher: MERGE cannot be used with a null property
    # value）。'' 即"个人维度"，_scope_match 的 coalesce 谓词已兼容。
    "project_id": filters.get("project_id") or "",
}
```

配套两点：
1. 同步更新 `test_personal_node_not_reused_by_project_add`（断言 `("alice","u1","")` 取代 `("alice","u1",None)`）；
2. 遗留数据迁移：基线 5a9cf86b 期间写入的旧个人节点 `project_id` 为 null，与 `''` 节点构成同身份重复，需一次性 `MATCH (n:__Entity__) WHERE n.project_id IS NULL SET n.project_id = ''`（或显式文档化接受）。

## WARNING

### WR-01: `test_personal_node_not_reused_by_project_add` 未"真实锁定" CR-01 核心改动

**File:** `mem0/server/tests/test_graph_memory.py:180-182`
**Issue:**
该用例只断言 MERGE 查询的 **params**（`p.get("project_id")`），不断言 MERGE **模式键**。若仅回退 CR-01 的核心改动——恢复 `MERGE (a {name, user_id})` + 无条件 `SET a.project_id`——而保留新的 `_scope_params`/add params（恒双带 project_id），params 仍含 None 与 "p1" 两个值，**测试依然全绿**。任务书要求"真实锁定（实现回退会红）"只在对 `_scope_params` 一并回退时才成立，对最易回归的 MERGE 键本身是弱锁定。

**Fix:**
补 cypher 字符串断言，使最小回退即红：

```python
merge_cyphers = [c for c, _ in neo4j.queries if "MERGE (a" in c]
assert all("project_id: $project_id" in c for c in merge_cyphers)
all_cyphers = " || ".join(c for c, _ in neo4j.queries)
assert "SET a.project_id" not in all_cyphers and "SET b.project_id" not in all_cyphers
```

## INFO

### IN-01: 阈值告警日志打印的是构造参数而非非法 env 值

**File:** `mem0/server/graph_memory.py:98`
**Issue:** `logger.warning("Invalid GRAPH_THRESHOLD value %r", threshold, ...)` —— 当调用方未传 `threshold`（None）且 env `GRAPH_THRESHOLD` 非法时，日志显示 `value None`，丢失了真正非法的 env 值，误导排障。
**Fix:** 先取 `raw = threshold if threshold is not None else os.environ.get("GRAPH_THRESHOLD", DEFAULT_THRESHOLD)` 再 try float，日志打印 `raw`。

### IN-02: 多个修复分支缺少直接回归用例

**File:** `mem0/server/tests/test_graph_memory.py`
**Issue:** 以下已实现的防护/行为无用例覆盖：hard_delete project 分支缺 project_id（:340-342）、add 缺 user_id 防护（:200-204）、`_norm` 连续空白折叠（:33，现有用例仅单空格）。
**Fix:** 各补一条最小用例（模式与现有 fake 用例一致）。

### IN-03: project 作用域消歧范围与 MERGE 身份键不一致（既有设计后果，建议文档化）

**File:** `mem0/server/graph_memory.py:210-211 vs 217`
**Issue:** scope=project 时 `_find_similar_node` 在整个项目池（不限 user_id）消歧，命中他用户同名节点后，MERGE 键仍含本用户 user_id，将以本用户身份再建同名列节点——项目池内同名重复节点随时间累积。这是 (name, user_id, project_id) 身份设计的固有结果，非本次修复引入，但应在模块 docstring 中明确该语义，避免后续被当缺陷反复"修复"。
**Fix:** 在 docstring 补充说明，或评估消歧结果是否应与 MERGE 键对齐（限同 user）。

---

_Reviewed: 2026-06-27_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_

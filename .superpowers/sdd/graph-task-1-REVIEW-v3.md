---
phase: graph-task-1
reviewed: 2025-10-28T00:00:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - mem0/server/graph_memory.py
  - mem0/server/tests/test_graph_memory.py
findings:
  critical: 0
  warning: 0
  info: 0
  total: 0
status: clean
---

# Graph Task 1：第三轮复审（差异包 diff-v3，基准 683151e9）

**Reviewed:** 2025-10-28
**Depth:** standard（对照实际落盘源文件逐项核验，非仅审 diff）
**Files Reviewed:** 2
**Status:** clean —— Spec: ✅ / Quality: Approved

## Summary

本轮只验证上轮 BLOCKER（BLK-01）与 WARNING（WR-01）是否闭环，顺带核验 IN-01。所有验证均基于工作区实际文件（`mem0/server/graph_memory.py`、`mem0/server/tests/test_graph_memory.py`），与差异包 `graph-task-1-diff-v3.txt` 一致。105 passed 按要求采信报告。

## 逐项闭环验证

### BLK-01（BLOCKER）✅ 已闭环 —— 个人作用域 project_id 全部物化为 ""

三处写路径 + 一处 docstring，逐一核验：

1. **`add` 的 MERGE/SET params**（graph_memory.py:259）：
   `"project_id": filters.get("project_id") or ""`
   —— 两个节点 MERGE 键（:237、:241）与关系 `r.project_id`（:247）共用同一 param，不再有 null 传给 MERGE。注释明确引用 Neo4j 22N31 错误。
2. **`_scope_params` 个人分支**（graph_memory.py:140）：
   `return {"user_id": filters.get("user_id"), "project_id": filters.get("project_id") or ""}`
   —— 覆盖全部下游调用方：`_find_similar_node` 消歧谓词（:197）、`search`（:289）、`get_all`（:324）、`soft_delete_for_text`（:350）、`hard_delete`（:370）。project 分支（:134）在 `hard_delete` 入口有 project_id 非空守卫（:361-364），不会传 null。
3. **与 `_scope_match` 读写一致**（graph_memory.py:127-130）：
   读谓词 `coalesce(n.project_id, '') = coalesce($project_id, '')` —— 写侧个人节点 project_id 固化为 `''`，读侧参数物化为 `''`，`coalesce('','') = ''` 命中；即使存在遗留 null 节点，`coalesce(null,'') = ''` 同样命中。软删的关系属性谓词（`scope_r`，:340/:345）同源，关系创建时 `r.project_id` 亦物化为 `''`（:247），读写一致。✅
4. **docstring 迁移事项**（graph_memory.py:205-215）：
   `add` docstring 明确注明：遗留 null project_id 节点不会被个人/项目检索命中，需一次性 backfill（`SET n.project_id = '' WHERE n.project_id IS NULL`），当前无真实存量数据，仅文档化。✅

### WR-01（WARNING）✅ 已闭环 —— 测试锁定 MERGE 模式键，防回退

`test_personal_node_not_reused_by_project_add`（test_graph_memory.py:167-190）：

- :185 `assert ("alice", "u1", "") in vals and ("alice", "u1", "p1") in vals` —— params 物化后 None→"" 的正确期望值；
- :189 `assert "project_id: $project_id" in cypher` —— MERGE 模式键必须含 project_id；
- :190 `assert "SET a.project_id" not in cypher` —— 禁止回退「MERGE(name,user_id) + SET project_id」旧行为。

两次 add 构造的 cypher 字符串相同（仅 params 不同），取首个 MERGE 调用断言即可覆盖两个作用域，无遗漏。

### IN-01 ✅ 顺带闭环

graph_memory.py:97-101：非法 threshold 告警改为打印实际读到的原始值（`raw = threshold if threshold is not None else os.environ.get("GRAPH_THRESHOLD")`）。当非法值来自 env 时不再打印无诊断价值的 None；env 未设置时走默认值分支不会进入 except，无边界漏洞。

### 回归测试

105 passed（报告口径，按要求采信）；无新增用例、无既有断言删除，仅 WR-01 增强。

## 新增缺陷排查

对 v3 差异包整体扫描（含未在 diff 上下文中但受影响的调用方）：

- 写侧物化（`or ""`）与读侧 coalesce 谓词在全仓调用点一致，无遗漏调用方仍传 null；
- `soft_delete_for_text` / `hard_delete` 的作用域参数均经 `_scope_params` 物化，无 null 参数路径；
- `search`/`get_all` 的 project 分支若 scope=="project" 但缺 project_id 会降级为个人作用域——属既有读侧行为，非本轮差异引入，不构成本轮缺陷；
- 无死代码、无未使用导入、无调试残留。

**结论：上轮全部 BLOCKER/WARNING 闭环，无新增缺陷。Spec: ✅ / Quality: Approved**

---

_Reviewer: Claude (gsd-code-reviewer) — re-review round 3_

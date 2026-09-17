---
task: Graph Task 1 (GraphMemory 图记忆核心)
commit: 5a9cf86b (mem0 submodule, feat/graph-memory)
base: 4260fc1a
reviewed: 2026-09-16
depth: deep
files_reviewed_list:
  - mem0/server/graph_memory.py
  - mem0/server/tests/test_graph_memory.py
findings:
  critical: 2
  important: 5
  minor: 4
  total: 11
status: issues_found
---

# Graph Task 1: Code Review Report

**结论：Spec 主链路 ✅（含两处已核验偏差，均批准）；Quality: Needs fixes（2 Critical）**

## Critical Issues

### CR-1: MERGE 键省略 project_id + 无条件改写 project_id → 项目硬删连带删除个人图节点（跨作用域数据丢失）

**File:** `mem0/server/graph_memory.py:190`（MERGE 键）、`:193/:198`（无条件 `SET a.project_id`）、`:300-307`（hard_delete）、`:112-116`（_scope_match）

**Issue:** spec §4 规定节点键 = `(name, user_id, project_id)`，实现 MERGE 键只有 `(name, user_id)`，且 `add` 无条件执行 `SET a.project_id = $project_id`。后果链：
1. 用户 u1 个人 add → 节点 `(alice, u1, project_id=null)`；
2. 同用户以 `scope=project, project_id=p1` add 同名实体 → MERGE 按 `(alice, u1)` 命中**同一个节点** → `project_id` 被改写为 p1；
3. Danger Zone `hard_delete({scope:project, project_id:p1})` 的谓词 `n.project_id = $p1` 命中该节点 → `DETACH DELETE` 连带删除 u1 的**个人图**数据。

**Fix:** 节点键补上 project_id（或按 spec 用 `(name, user_id, project_id)` 三元 MERGE 键），并仅在项目作用域下 SET project_id，个人 add 显式不清除既有 project_id：
```cypher
MERGE (a:__Entity__ {name: $source, user_id: $user_id, project_id: $scope_project_id})
```

### CR-2: `hard_delete({"project_id"})`（无 scope 键）是静默 no-op，且对应测试为空转通过

**File:** `mem0/server/graph_memory.py:304, 112-121`；`mem0/server/tests/test_graph_memory.py:479-484`

**Issue:** brief 自带测试 `g.hard_delete({"project_id": "p1"})`（spec §6 Danger Zone 的入参形态）。实现中 `_scope_match` 要求 `scope == "project"` 才走 project 谓词，否则生成 `WHERE n.user_id = $user_id` 且 `$user_id = None`——Cypher 中 `= null` 永不匹配，**删除 0 个节点**。测试只断言 cypher 含 `"DETACH DELETE"` 字符串，因此空转通过（vacuous pass），掩盖了语义缺陷。若后续 router 不显式传 `scope="project"`，Danger Zone 硬删将静默失效（数据残留，违反 spec §6）。

**Fix:** 二选一：(a) `_scope_match/_scope_params` 在 `project_id` 存在且 `user_id` 缺失时自动落入 project 谓词；(b) 修正测试断言为「无 scope 时行为明确定义」（如要求必传 user_id 或 scope），并让 router 合同显式化。至少要给 hard_delete 增加参数校验：`user_id` 与 `project_id` 均缺失时记日志并拒绝执行，杜绝全图误删。

## Important Issues

### IM-1: 构造函数可抛异常——`GRAPH_THRESHOLD` 非法时 `float()` 在 try 块之外

**File:** `mem0/server/graph_memory.py:94-96`

**Issue:** `self.threshold = float(... env GRAPH_THRESHOLD ...)` 位于 try/except 之后，env 值非法（如 `"0.7x"`）时 ValueError 直接冒泡，违背 spec §1「构造任何失败 → 记日志 + enabled=False，绝不抛异常」。

**Fix:** 将 threshold 解析移入 try 块，或 `try: self.threshold = float(...) except ValueError: logger.warning(...); self.threshold = DEFAULT_THRESHOLD`。

### IM-2: 项目作用域无 user_id 时 MERGE 键含 null → 每次调用新建重复节点

**File:** `mem0/server/graph_memory.py:190` + `:118-121`

**Issue:** `scope=project` 且 filters 无 `user_id` 时，add 的 params 中 `user_id=None`；Cypher `MERGE (a {name:$s, user_id:$user_id})` 的 null 属性永不匹配既有节点 → 同名实体每次 add 都新建节点/关系，项目池图无限膨胀（embedding 消歧找到了节点也无济于事，MERGE 键匹配失败）。

**Fix:** 项目作用域下 MERGE 键使用固定哨兵（如 `user_id: "__project_pool__"`）或采用 CR-1 的三元键方案，确保键值永不为 null。

### IM-3: 关系抽取缺少自我指涉映射 → "I" 实体分裂

**File:** `mem0/server/graph_memory.py:148-153`（对比 `:135-136`）

**Issue:** `{"I" → user_id}` 映射只在 `_extract_entities` 做；`_extract_relations` 对 LLM 返回的 `source/destination == "I"` 仅 `_norm` 成 `"i"`，未映射为 user_id 实体 → 图谱分裂出孤立的 `i` 节点，add/search/soft_delete 语义不一致。1.x 对齐口径下该映射应在两侧一致。

**Fix:** 在 `_extract_relations` 循环内复用同一映射：`if s == "i" or s == _norm(user_id): s = user_id`（destination 同理）。

### IM-4: `soft_delete_for_text` 忽略 project 作用域谓词 → 跨作用域软删 / 项目池场景失配

**File:** `mem0/server/graph_memory.py:287-296`

**Issue:** 软删 cypher 仅按 `(name, user_id)` 精确匹配节点 + `type(r) = $rel`，完全不看 project_id/valid。由于 CR-1 的键共享，同节点对上个人关系与项目关系共存时，按文本软删会把**两个作用域的同型关系一起置 valid=false**（过度删除）；反之项目池无 user_id 场景下（user_id=None）软删永远匹配不到任何关系（静默 no-op）。

**Fix:** 复用 `_scope_match/_scope_params` 构造软删谓词，与 add 的键语义对齐。

### IM-5: `_norm` 不折叠连续空白——与偏差①的修复不一致，只修了一半

**File:** `mem0/server/graph_memory.py:33-34`

**Issue:** 实现者声明偏差①时把 `sanitize_relationship` 改为 `re.sub(r"\s+", "_", ...)`，但 `_norm` 仍是单次 `replace(" ", "_")`：`"Alice  Smith"` → `"alice__smith"`。同一实体不同空白写法会 MERGE 出多个节点，且 `_norm(user_id)` 与 `"i"` 的比较同理受影响。brief 版 `_norm` 与 brief 版 sanitize 是同一个 bug 的两处实例，只修一处属修复不完整。

**Fix:** `re.sub(r"\s+", "_", (value or "").strip().lower())`，与 sanitize_relationship 保持同一折叠规则。

## Minor Issues

### MN-1: BM25 段只捕获 ImportError，兜底路径 payload 形状不一致

**File:** `mem0/server/graph_memory.py:247-259`

**Issue:** `except ImportError` 之外的 BM25 运行时异常（如退化 corpus）会落到外层 except 令整个 search 返回 `[]`，而非设计意图的「缺 rank_bm25 时顺序截断」；且 ImportError 兜底返回 `all_triples[:5]` 携带 `similarity`/`sim` 等额外键，与 BM25 路径的严格三键 payload 不一致，下游前端需容忍两种形状。

**Fix:** `except Exception:`（记 debug 日志）+ 兜底返回前投影为三键字典。

### MN-2: sanitize 后的关系类型可能以数字开头 → 非法 Cypher 标识符

**File:** `mem0/server/graph_memory.py:37-39`

**Issue:** `"1 2"` → `"1_2"`，`:1_2` 不是合法的未加反引号 Cypher 标识符，add 将抛异常降级为空。防注入目标已达成（[A-Z0-9_] 白名单），仅是可用性边角。

**Fix:** 清洗后若首字符为数字则前缀 `_` 或直接丢弃该三元组。

### MN-3: 未使用导入

**File:** `mem0/server/graph_memory.py:8`（`datetime, timezone` 全未使用）；`mem0/server/tests/test_graph_memory.py`（`SimpleNamespace` 未使用）

**Fix:** 删除。

### MN-4: 复合索引与 spec §4「单键索引」措辞矛盾

**File:** `mem0/server/graph_memory.py:105`

**Issue:** spec §4 写「`(__Entity__.name, user_id)`（单键索引，规避企业版组合索引限制）」——spec 文字自相矛盾（name+user_id 本身就是组合索引）；实现照字面建了复合索引。Neo4j Community 支持复合 range 索引，功能无碍，但 spec 措辞应修正以免后续审计误判。

## 偏差核验（实现者声明的两处）

1. **① `sanitize_relationship` 空白折叠为单下划线 — 批准，判定成立。**
   brief 版实现 `replace(" ", "_")` 对 brief 自带测试输入 `"works  on!"` 产出 `WORKS__ON`，与测试断言 `"WORKS_ON"` 直接冲突——这是 brief 源码内部的实现 bug，不是实现者改语义。修复方向正确（安全白名单不变，仅折叠规则更合理），测试未改动，符合「测试语义优先」。注意：同型 bug 在 `_norm` 中仍存在（见 IM-5）。

2. **② `test_project_scope_uses_project_id_predicate` 给一个实体 — 批准，判定成立。**
   原测试 LLM 返回 `{"entities":[]}` 时，`search` 在 `if not entities: return []`（graph_memory.py:226-227）即返回，`neo4j.queries` 为空 → `neo4j.queries[-1]` 抛 IndexError，project_id 断言永远不可达。实现者微调为返回一个实体以进入查询阶段，绑定断言（`project_id` 在末条 cypher 与 params 中）原样保留，属必要且最小的修复。

## Cannot verify from diff

- **router/调用方的 filters 合同**（`scope` 是否总与 `project_id` 同传）——决定 CR-2 的实际影响面与 IM-2 的可达性；建议 Task 2（graph_router）落地时显式约束并在该层加参数校验。
- `rank_bm25` / `langchain_neo4j` 在容器内的实际安装与版本兼容（报告称测试经容器运行，采信）。
- 测试结果 9 passed / 回归 101 passed（按任务书约定采信报告，未重跑）。
- `NEO4J_PASSWORD` 默认空串的生产 env 注入方式（部署侧配置，不在本 diff）。
- 提交 `5a9cf86b` 仅含两文件的声明（报告自述 `git show --stat` 2 files changed, 468 insertions，与差异包一致）。

# Task 1 报告：GraphMemory 核心实现

**状态：DONE**
**提交：`5a9cf86b`**（mem0 子模块，分支 `feat/graph-memory`）
**测试：`tests/test_graph_memory.py` 9 passed；全量回归 `pytest tests/ -q` 101 passed（基线 92 + 新增 9），3 warnings 均为既有依赖弃用告警，与本任务无关。**

## TDD 流程

1. **RED**：先写入测试文件，容器内运行 → `ModuleNotFoundError: No module named 'graph_memory'` ✅（按预期红灯）。
2. **GREEN**：按 brief 实现代码逐字落盘 `server/graph_memory.py`，首轮 7/9 passed。
3. **修复 2 处失败**（见「偏差」），复跑 9/9 passed。
4. **回归**：全量 101 passed（92 基线 + 9 新增）。

## 交付物

| 文件 | 说明 |
| --- | --- |
| `mem0/server/graph_memory.py` | 图记忆核心：`GraphMemory` 类 + `sanitize_relationship` 模块函数。对齐 mem0 v1.0.11 `MemoryGraph` 语义，带项目维度扩展 |
| `mem0/server/tests/test_graph_memory.py` | 9 个单元测试，全部使用 fake（LLM/embedder/Neo4j driver），不连真实服务 |

## 语义核对清单（与 brief 约定逐项一致）

- ✅ `GRAPH_ENABLED` env 开关，默认 true；false 时构造短路，`add/search/...` 直接返回空、零副作用（`test_disabled_when_env_off`）。
- ✅ 构造函数任何失败（导入失败 / Neo4j 不可达 / driver 传 None 走 `langchain_neo4j.Neo4jGraph` 构造）→ `logger.exception` + `enabled=False`，绝不抛异常（`test_degrades_when_driver_init_fails`）。
- ✅ 实体名规范化 `_norm`（小写、空格→下划线）；"I" 自我指涉映射为 `{user_id}` 实体（`test_extract_entities_normalizes`）。
- ✅ 关系类型清洗 `sanitize_relationship`：大写化 + 空白折叠为单下划线 + 保留 `[A-Z0-9_]`（`test_relationship_type_is_sanitized`）。
- ✅ LLM 纯 JSON prompt（`EXTRACT_ENTITIES_SYSTEM` / `EXTRACT_RELATIONS_SYSTEM`），`_parse_json` 容忍 markdown 围栏 + 兜底正则抽取。
- ✅ 节点 MERGE 键 `(name, user_id)` + `project_id` 属性 + mentions 计数 + `db.create.setNodeVectorProperty` 写 embedding。
- ✅ embedding 消歧：`_find_similar_node` 用 `vector.similarity.cosine ≥ threshold` 作用域内查找（含个人作用域下对 project 节点的额外谓词拼接）。
- ✅ 作用域谓词 `_scope_match` / `_scope_params`：`scope == "project"` 且有 project_id → 按 project_id；否则按 user_id（`test_project_scope_uses_project_id_predicate`）。
- ✅ search：逐实体 embedding 匹配 → 1 跳关系（`r.valid IS NULL OR r.valid = true` + 对端作用域）→ BM25 重排 top5（`SEARCH_TOP_K=5`；`rank_bm25` ImportError 时顺序截断兜底）（`test_search_returns_bm25_top_triples`）。
- ✅ `soft_delete_for_text` 置 `r.valid = false` + `invalidated_at`；`hard_delete` 用 `DETACH DELETE`。
- ✅ `get_all`、公开 API 全部 try/except 降级返回空，主流程不受影响。
- ✅ 索引创建 best-effort（失败仅 debug 日志，不阻断构造）。

## 偏差记录（测试语义优先 / Rule 1）

1. **[Rule 1 - Bug] `sanitize_relationship` 双空格 bug**
   - 首轮实现 `replace(" ", "_")` 对 `"works  on!"`（双空格）产出 `WORKS__ON`，与测试语义「空白序列→单下划线」冲突。
   - 修复：先 `re.sub(r"\s+", "_", ...)` 折叠空白，再剥除非法字符。测试未改动。
2. **[测试微调] `test_project_scope_uses_project_id_predicate`**
   - 原 brief 测试让 LLM 返回 `{"entities":[]}`，此时 `search` 在抽取阶段即返回（v1.x 语义：无实体不查询），永远触达不了查询断言。
   - 按「测试表达语义、实现满足语义」原则微调：LLM 返回一个实体以进入查询阶段，绑定断言（`project_id` 出现在末条 cypher 与 params 中）不变。
3. **[测试整理] brief 中 `test_extract_entities_normalizes` 的重复 `def` 行**：brief 源码里该函数头重复了一次（Step 4 提示的三处 malformed sketch 之一），落盘时去重，其余测试代码逐字保留。

## 环境备注

- `mem0-api` 容器原先未运行，已通过 `docker compose -f deploy/mem0/docker-compose.yaml up -d mem0-api` 启动后执行全部测试。
- `test_degrades_when_driver_init_fails` 按原样用 `bolt://invalid.invalid:7687` 走真实构造降级路径，未出现 DNS 超时（约 3.9s 完成整组测试），未启用 monkeypatch 替代方案。

## 自审结论

- 红灯→实现→绿灯→回归流程完整，提交哈希 `5a9cf86b` 已验证存在。
- 提交仅含本任务两个文件（`git show --stat` 确认 2 files changed, 468 insertions），子模块工作区其余改动（`server/main.py` 等）为既有未提交内容，未混入。
- 无 stub、无认证门、无威胁面新增（纯内部模块，未新增网络端点；所有外部访问均可注入 fake 并整体降级）。

---

# Task 1 修复报告：Code Review 缺陷修复（2 Critical + 5 Important + Minor）

**状态：DONE**
**提交：`683151e9`**（mem0 子模块，分支 `feat/graph-memory`，2 files changed, 118 insertions, 24 deletions）
**测试：容器内 `pytest tests/ -q` → 105 passed（基线 92 + 图记忆 13 = 9 既有适配 + 4 新增回归），3 warnings 均为既有依赖弃用告警。**

## 逐项修复说明

| # | 级别 | 修复内容 |
| --- | --- | --- |
| CR-01 | Critical | 节点身份改为 `(name, user_id, project_id)`：`add` 两个 MERGE 键均含 `project_id`（可为 null，Cypher MERGE 按 null 匹配），删除无条件 `SET a.project_id`/`SET b.project_id`；`_find_similar_node` 移除旧的 project 拼接 `extra`，消歧仅在 `_scope_match` 的同身份作用域内；`_scope_match`/`_scope_params` 重构为 scope 驱动且参数恒双带——project 作用域谓词 `n.project_id = $project_id`（params 仅 project_id），个人作用域谓词 `n.user_id = $user_id AND coalesce(n.project_id,'') = coalesce($project_id,'')`（params 恒含 user_id + project_id），谓词签名不变 |
| CR-02 | Critical | `hard_delete` 入口校验：`scope=="project"` 需 project_id、默认 user 作用域需 user_id，缺失 → `logger.warning` + 直接 return，绝不生成无界 DETACH DELETE |
| IMP-01 | Important | `GRAPH_THRESHOLD` 的 `float()` 包入独立 try 块，非法值 → warning 日志 + 回落默认 0.7，构造函数仍绝不抛异常 |
| IMP-02 | Important | `add` 入口校验 `filters.get("user_id")`，缺失 → warning + 返回空（检索可共享项目池，写入必须来自真实用户） |
| IMP-03 | Important | `_extract_relations` 对 source/destination 归一化后应用与实体相同的自我指涉映射：`== "i"` → `user_id` |
| IMP-04 | Important | `soft_delete_for_text`：a、b、r 三处谓词全部改由 `_scope_match(filters, alias)` 驱动（project 池软删按 `r.project_id`；关系属性谓词与作用域一致），params 用 `_scope_params`；无 user_id 且无 project_id → warning + return |
| IMP-05 | Important | `_norm` 改为 `re.sub(r"\s+", "_", value.strip().lower())`，连续空白折叠为单下划线（与 sanitize 修复一致） |
| Minor ① | Minor | BM25 段 `except ImportError` → `except Exception`，兜底路径与 BM25 路径同构（仅 source/relationship/destination 三键，顺序截断 top5） |
| Minor ② | Minor | 采用**跳过**方案：sanitize 后为空或以数字开头的关系 → 该三元组跳过（debug 日志记录），不加 R 前缀 |
| Minor ③ | Minor | 删除未使用的 `from datetime import datetime, timezone` 导入 |

## 测试

- 既有 9 个用例适配 1 处：`test_hard_delete_uses_detach_delete` 的 filters 从 `{"project_id": "p1"}` 改为 `{"user_id": "u1"}`（hard_delete 新增 CR-02 scope 校验后，无 user_id 的默认作用域调用会被防护拦截；测试意图「发出 DETACH DELETE」不变）。
- 新增 4 个回归用例（追加到同文件）：`test_personal_node_not_reused_by_project_add`、`test_hard_delete_without_scope_is_noop`、`test_project_hard_delete_uses_project_predicate`、`test_extract_relations_maps_self_reference`。
- 运行方式：文件复制进 `mem0-api` 容器后 `python -m pytest tests/ -q`，105 passed 全绿。

## 偏差记录（Rule 3：回归用例与 Fake 基础设施不匹配的适配）

1. **[Rule 3] `test_personal_node_not_reused_by_project_add` LLM 响应条数**：brief 给的 FakeLLM 只有 2 条响应，但每次 `add` 需 2 次 LLM 调用（实体抽取 + 关系抽取），两次 `add` 共 4 次——第 2 次 add 触发 `IndexError` 被 add 的降级 except 吞掉，断言必然失败。补充重复的 2 条响应（语义不变：两个作用域各执行一次相同抽取）。
2. **[Rule 3] `test_hard_delete_without_scope_is_noop` / `test_project_hard_delete_uses_project_predicate` 索引查询污染**：构造函数 `_ensure_indexes` 会向 FakeNeo4j 发出 3 条 CREATE INDEX 查询，brief 断言 `queries == []` / `queries[0]` 与之冲突。适配为：前者记录构造后基线长度断言无新增查询，后者改取 `queries[-1]`（即 DETACH DELETE 查询）。绑定语义（无界删除不发生 / project 谓词正确）不变。

以上均为测试脚手架适配，产品代码行为与 brief 修复要求逐字一致。

---

# Task 1 修复报告：复审 BLOCKER + WARNING 修复（BLK-01 + WR-01 + IN-01）

**状态：DONE**
**提交：`2f3eff57`**（mem0 子模块，分支 `feat/graph-memory`，2 files changed, 37 insertions, 7 deletions）
**测试：copy 后容器内 `python -m pytest tests/ -q` → 105 passed 全绿，3 warnings 均为既有依赖弃用告警，与本任务无关。**

## 逐项修复说明

| # | 级别 | 修复内容 |
| --- | --- | --- |
| BLK-01 | BLOCKER | **个人作用域 project_id 参数物化为空串（"")**。真实 Neo4j 对 MERGE 的 null 属性键直接报错（`Cannot merge node using null property value`，22N31），且被 `add` 的降级 except 吞掉 → 个人图写入静默全失效（fake 不执行 cypher，105 全绿为假象）。三处同源修复：① `add` 的 MERGE/SET params：`"project_id": filters.get("project_id") or ""`（两个节点参数同源，与 `_scope_match` 个人谓词 `coalesce(n.project_id,'')` 语义对齐）；② `_scope_params` 个人分支返回物化值（同时覆盖 `_find_similar_node` 的消歧谓词 params 与 `soft_delete_for_text` 的 params，保证 `coalesce(n.project_id,'') = coalesce($project_id,'')` 读写一致）；③ `add` docstring 注明：遗留 null project_id 节点（若存在）不会被个人/项目检索命中，属一次性迁移事项（`SET n.project_id = '' WHERE n.project_id IS NULL`），本轮无真实存量数据，仅文档化。 |
| WR-01 | Warning | `test_personal_node_not_reused_by_project_add` 追加两条字符串断言锁定 MERGE 模式键：`assert "project_id: $project_id" in cypher`（MERGE 模式键必须含 project_id）与 `assert "SET a.project_id" not in cypher`（身份键之外不得再改写，防止回退成「MERGE (name,user_id) + SET project_id」旧行为）。同步回归断言 None→""：`("alice", "u1", "") in vals`（params 物化后的正确期望值）。 |
| IN-01 | 顺带 | threshold 非法告警日志改为打印实际读到的原始值：新增 `raw = threshold if threshold is not None else os.environ.get("GRAPH_THRESHOLD")`，warning 打印 `raw`（原先打印 `threshold` 参数，当非法值来自 env 时该参数为 None，日志无诊断价值）。 |

## 测试

- 既有 13 个图记忆用例语义不变，仅 `test_personal_node_not_reused_by_project_add` 按上述 WR-01 增强（无新增用例，无既有断言删除）。
- 运行方式：`docker cp` 两个文件进 `mem0-api` 容器 → `docker exec -w /app mem0-api python -m pytest tests/ -q` → **105 passed, 3 warnings in 12.62s**。

## 备注

- 提交按任务指定命令执行：`git add server/graph_memory.py server/tests/test_graph_memory.py && git commit --no-verify -m "fix(server): materialize empty project_id for personal graph merge (null MERGE is invalid)"`。
- `--no-verify` 为任务 brief 明确要求；提交内容仅含本任务两个文件。

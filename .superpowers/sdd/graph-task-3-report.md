# Task 3 报告：main.py 图记忆层集成（接线）

**状态：完成** | **提交：`e8e2cc34`** (`feat(server): graph build scheduling, /search relations fusion, delete links`，分支 `feat/graph-memory`) | **测试：116 passed**（基线 110 + 新增 6，容器内全绿，25.7s）

## 变更内容

### server/main.py（+91）
1. `from server_state import get_graph_memory` 引入单例（单例本身在 server_state，DCL 实现，本任务不重复实现）。
2. `SearchRequest.include_graph: Optional[bool] = True`。
3. 新增接线辅助：
   - `_graph_executor = ThreadPoolExecutor(max_workers=2, thread_name_prefix="graph")`（fire-and-forget，语义与 webhook executor 一致）；
   - `_schedule_graph_build(text, filters)`：整体 try/except 包裹，禁用时跳过，调度失败仅记日志不破坏写路径；
   - `_fuse_graph_relations(data, query, filters, include_graph)`：include_graph=False 直接透传；禁用/异常返回 `relations: []`，绝不破坏搜索。
4. 五个接线点：
   - **add_memory**：mem0 调用前捕获 `graph_text`（消息内容拼接）与 `graph_filters = {"user_id": owner_user_id, "project_id": project_id or None}`；成功后 `if graph_text.strip(): _schedule_graph_build(...)`（在 webhook 之前）。
   - **search_memories**：项目池分支 `graph_filters = {"project_id": project_scope, "scope": "project"}`，个人分支经 `_scope_identifiers` 钉死后的 `{"user_id": ..., "scope": "user"}`；两个分支返回值均改经 `_fuse_graph_relations(_filter_by_project_id(...), ...)`（先项目后过滤再融合）。admin 无 user_id 的无作用域搜索由 GraphMemory.search 的 unscoped-read guard 兜底返回 `[]`。
   - **delete_memory**：删除前 best-effort `get(memory_id)`（异常吞掉）；成功后若取到 `memory` 与 `user_id`（dict 或属性访问双兼容，mem0 `Memory.get` 返回 model_dump dict），executor 提交 `soft_delete_for_text`（独立 try/except）。
   - **delete_memories_by_project**：`deleted > 0` 时 executor 提交 `hard_delete({"project_id": ..., "scope": "project"})`（try/except）。
   - **delete_entity**：见下（在 routers/entities.py）。

### server/routers/entities.py（+16）
`delete_entity` 成功后提交 `hard_delete({"user_id": entity_id, "scope": "user"})`。该端点不在 main.py 而在本路由；main 在模块级导入本路由，故采用函数内懒导入 `main._graph_executor` 避免循环导入。**仅对 `entity_type == "user"` 触发**：图节点只按 user_id/project_id 定域，agent/run 实体无图表示，按 brief 原文对任意类型以 user_id=entity_id 硬删会误删同名 user 节点。

### server/tests/test_graph_integration.py（新增，6 条）
- `test_get_graph_memory_is_lazy_singleton`：**适配偏差**——单例在 `server_state`，改为 patch `graph_memory.GraphMemory` + 重置 `server_state._graph_memory`，断言两次调用同对象且仅构造一次。
- `test_search_fusion_adds_relations` / `_disabled_returns_empty_relations` / `_include_graph_false`：逐字采用 brief，patch `m.get_graph_memory`（main 经 `from server_state import get_graph_memory` 引入模块全局，patch 可行，符合偏差说明）。
- `test_schedule_graph_build_skips_when_disabled` / `_submits_when_enabled`：逐字采用 brief，patch `m._graph_executor`。

## 验证
- 容器内：`pytest tests/ -q` → **116 passed**（基线 110 + 6），3 个预存 DeprecationWarning，无新告警。
- 提交核对：`git show --stat` 确认仅含 3 个任务文件；工作区遗留改动（docker-compose.yaml、requirements.txt、favicon 删除、server/data/、seed.sql）未夹带。

## 偏差与说明
1. **[偏差-单例位置]** brief 测试 1 针对旧 main.py 单例；按指示改测 `server_state.get_graph_memory`（见上）。
2. **[偏差-提交范围]** brief/指令的 `git add` 只列两个文件，但接线点 5（delete_entity）代码位于 `routers/entities.py`，该文件属于任务必需修改，已一并提交；非无关遗留改动。
3. **[偏差-类型守卫]** delete_entity 仅对 user 类型实体触发图硬删（理由见上）。
4. **[加固]** `_schedule_graph_build` 与 delete 路径的提交均带 try/except（Rule 2：图清理绝不反噬主流程）；`_fuse_graph_relations` 按 brief 保留异常兜底。
5. **[已知限制]** admin 在 /search 用 agent_id/run_id（无 user_id）时 graph_filters 无 user_id，图搜索返回空——与 GraphMemory unscoped-read guard 语义一致，属预期行为。

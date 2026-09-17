# Graph Task 2 报告：`/graph/search` 与 `/graph/get_all` 路由（作用域解析 + 隔离）

**状态：** 完成 ✅
**分支：** `feat/graph-memory`
**提交：**
- `1661945a` feat(server): /graph/search and /graph/get_all endpoints with scoping（routers/graph.py、tests/test_graph_router.py、main.py）
- `d0455262` feat(server): graph memory singleton in server_state + unscoped-read guards in search/get_all（server_state.py、graph_memory.py —— 原提交命令清单遗漏的功能必需件）

**测试：** RED 确认（`ModuleNotFoundError: No module named 'routers.graph'`）→ GREEN 5 passed（brief 4 条逐字 + 1 条 GraphMemory 守卫断言）→ 全量回归 `pytest tests/ -q` **110 passed**（容器 mem0-api）。

## 实现要点

### routers/graph.py（新建）
- `POST /graph/search`（body: query/project_id/git_remote/user_id/limit=25）、`GET /graph/get_all`（query params，limit=100）。
- `_resolve_scope` 作用域契约（Task 1 审查要求全部满足）：
  - 返回 filters **恒含 `scope` 键**："project" / "user" / "none"。
  - project 模式必带非空 project_id（git_remote 先经 `resolve_project_id` 解析）。
  - 非 admin：user_id 钉死为调用者；传他人 user_id → **403**；无 project → 个人作用域（不 400）。
  - admin：显式 user_id → scope=user；project → scope=project；**两者皆无 → `{"scope": "none"}`**（按任务修正，路由直接返回 `{"relations": []}` 不触达图，brief 测试 `test_get_all_requires_project_or_user` 语义）。
- `get_graph_memory` 从 `server_state` 导入（推荐方案，无循环导入；测试 monkeypatch `routers.graph.get_graph_memory` 模块属性有效）。
- limit 校验：pydantic `Field(ge=1, le=1000)`（search body）；GET 的 limit 用 `max(1, min(limit, 1000))` clamp（Rule 2 输入校验）。

### server_state.py（修改）
- 新增 `get_graph_memory()` 懒加载单例（RLock 保护），复用 `get_memory_instance().embedding_model` 与 `.llm`（已验证容器内 mem0 Memory 确有这两个实例属性）。
- `update_config()` 重建 memory 时将 `_graph_memory` 置 None，避免图单例持有过期 embedder/LLM 引用。

### graph_memory.py（修改）
- 新增 `_guard_unscoped_read()`，`search` / `get_all` 接入（与 `hard_delete` 同款守卫）：scope=project 无 project_id → 拒绝；无 user_id 且无 project_id → 拒绝（警告日志 + 返回空，不发出任何 Cypher）。守卫测试断言 init 后零新增查询。

### main.py（修改）
- `from routers import graph as graph_router`（import 按字母序）+ `app.include_router(graph_router.router)`（stats_router 之后）。

## 与 brief 的偏差（均为必要修正）

1. **`resolve_project_id` 模块级导入**：brief 代码在函数内 `from gitmatch import ...`，会导致其自己的测试 `monkeypatch.setattr(graph_router, "resolve_project_id", ...)` 抛 AttributeError（模块属性不存在）。改为顶层导入后 monkeypatch 生效。
2. **admin 空参 → `scope:"none"` 空结果**（brief 原为 400）：按任务指令修正，brief 测试 `test_get_all_requires_project_or_user` 断言 200 + `{"relations": []}`。
3. **GraphMemory 守卫**：brief 无此内容，按任务指令补齐（search/get_all 与 hard_delete 对齐）。
4. `GraphGetAllRequest` 按 brief 保留但目前无端点使用（为未来 POST 变体预留）。

## Concerns

1. ⚠️ **`1661945a` 连带提交了 main.py 中先于本任务存在的未提交调试改动**：`logging.basicConfig(level=DEBUG)`、启动耗时 `_mark()` 埋点（5 处）、CORS origins 扩展（`CORS_ALLOW_ORIGINS` env + localhost:3000/3001）。这些在工作区本来就被修改，而任务指定的提交命令整体暂存 main.py。若非有意入库，建议后续单独 revert 这三处（不影响本功能）。
2. 追加了第二个提交 `d0455262`（server_state.py + graph_memory.py），超出原提交命令的三文件清单——不提交则仓库内功能不完整（路由引用的 `server_state.get_graph_memory` 与图守卫缺失）。
3. 非 admin 传 project_id 即可读整个项目图（含其他用户节点）——brief 测试 1 明确此语义（共享项目池），符合设计，但值得在文档中显式声明。
4. 工作区仍有未提交项：`server/docker-compose.yaml`、`server/requirements.txt`、删除的 favicon.ico、`server/data/`、`server/scripts/seed.sql` —— 非本任务范围，未触碰。

---

## 审查修复追加（Graph Task 2 Review Fixes）

**状态：** 完成 ✅
**提交：** `1be73f4d` fix(server): revert debug instrumentation riding in graph commits; align unscoped-read guard（server/main.py、server/graph_memory.py、server/server_state.py）

### 三处还原（Important，全部完成）

1. **日志级别还原**：`server/main.py` 的 `logging.basicConfig` 由 DEBUG 恢复为基准（2f3eff57）写法 **INFO**，避免生产日志被 mem0/sqlalchemy/httpx DEBUG 洪流污染。已与 `git show 2f3eff57:server/main.py` 基准逐行核对。
2. **_mark 启动埋点还原**：删除 _tmod/_TMARK/_mark 定义（原 :1-7）及全部 5 处调用（logging setup、telemetry、initialize_state、create_all、routers 注册处）。
3. **CORS origins 还原**：移除 CORS_ALLOW_ORIGINS 环境变量驱动的扩展逻辑，恢复基准写法 `allow_origins=[DASHBOARD_URL]`。
   - **说明**：指令描述的基准为"localhost:3000/3001 + DASHBOARD_URL 显式列表"，但 `2f3eff57:server/main.py` 的实际基准是单元素显式列表 `[DASHBOARD_URL]`（默认值即 localhost:3000，基准中无 3001）。按真实基准还原，避免以"还原"名义夹带新增 3001 的范围外改动；如确需 3001 可另行提出。

### 保留确认（未触碰）

graph 路由集成（`from routers import graph`、`app.include_router(graph_router.router)`）及 Task 2 的 include router 等功能改动全部保留。还原后 `git diff 2f3eff57 -- server/main.py` 仅剩上述两行 graph 集成差异（2 insertions）。

### Minor 修复（2 处，均完成）

1. **graph_memory._guard_unscoped_read 对齐 hard_delete 守卫**：原逻辑"无 user_id **且**无 project_id"才拒绝，改为 user 作用域（显式或默认）**无 user_id 即拒绝**——scope=user 且仅 project_id（无 user_id）时返回空，避免白跑 LLM 实体抽取（search 场景抽取发生在查询之前）。routers/graph.py 的 _resolve_scope 恒将 project_id 映射为 scope=project，故路由层行为不受影响，纯防御性收紧。
2. **server_state.get_graph_memory 改 double-checked locking**：先无锁判空（命中即返回），未命中才加 _state_lock 构造并二次判空——构造期间的 Neo4j 连接不再阻塞 get_memory_instance() 调用方。

### 验证

- 三文件 docker cp 进容器 mem0-api（/app 下扁平布局，修正了首次 cp 的 /app/server 路径）→ 容器内确认 INFO 级别、_mark 计数 0。
- `pytest tests/ -q`：**110 passed**（10.91s，3 warnings 均为既有 deprecation）。
- 启动冒烟（`docker restart` 后）：启动日志无 DEBUG 洪流（仅 INFO/WARN + 既有 crypt DeprecationWarning），`GET /graph/get_all` → **HTTP 200**。

---

## 复审（1be73f4d，对抗式复核）

**结论：Spec: ✅ / Quality: Approved**（四项验证全部通过，未发现新增缺陷）

1. **三处夹带还原到位**（以仓库实际状态核实，不采信差异包自述）：
   - 日志级别：`git diff 2f3eff57 1be73f4d -- server/main.py` 无 basicConfig 差异，实际代码 `logging.basicConfig(level=logging.INFO, ...)`（main.py:61），与基准一致；
   - `_mark` 埋点：`git grep` 全仓 0 命中（auth.py 的 `_mark_auth_type` 与测试名 `test_soft_delete_marks_invalid` 为子串误中，非埋点）；`_TMARK/_tmod` 0 命中；
   - CORS：`CORS_ALLOW_ORIGINS` 0 命中，`allow_origins=[DASHBOARD_URL]`（main.py:221），与真实基准一致（单元素列表，报告第 3 条说明成立——基准中本无 3000/3001）。
2. **graph 功能改动全保留**：`git diff 2f3eff57 1be73f4d -- server/main.py` 逐行核实，仅剩 2 处插入（`from routers import graph as graph_router` + `app.include_router(graph_router.router)`），与报告声明完全一致。
3. **守卫与锁**：
   - `_guard_unscoped_read`（graph_memory.py:268-286）与 `hard_delete` 守卫（:385-391）语义逐条等价：project 作用域缺 project_id 拒绝；非 project（user 作用域，含缺省）缺 user_id 拒绝。`search`（:291）/`get_all`（:338）均已接入；
   - `get_graph_memory` 双重检查锁（server_state.py:108-129）正确：无锁快速路径读模块级全局在 GIL 下原子且对象仅在构造完成后绑定（无半构造发布）；`_state_lock` 为 RLock，锁内调用 `get_memory_instance()` 可重入不死锁；锁内二次判空防并发重复构造；`update_config` 持锁置 None 与构造路径互斥。
4. **提交范围干净**：`git show --stat 1be73f4d` 仅三文件（graph_memory.py / main.py / server_state.py），与声明一致；工作区遗留未提交项（docker-compose.yaml、requirements.txt、favicon 删除、data/、seed.sql）为报告 Concern 4 所列既有项，未混入本提交。
5. **测试**：110 passed 采信报告（含守卫收紧与双检锁改动后的全量回归）。
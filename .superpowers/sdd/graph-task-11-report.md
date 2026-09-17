# Graph Task 11 报告 — 图数据重试队列 + 导出

**状态：✅ 完成** — commit `3e1c62e6`（feat(graph): ingest retry queue + graph data export endpoint）；容器内全量 pytest **122 passed**（116 baseline+graph + 6 new）。

## 变更
### graph_memory.py（重试队列）
- `__init__` 启动守护线程 `_retry_loop`（daemon），维护 `queue.Queue` + `threading.Event` 停止信号；`max_retries=MAX_RETRIES=5`，`_backoff=lambda a: min(2**a, 30)`。
- `add()` 拆为：公开 `add`（try→`_ingest`；异常则 `_enqueue_retry` 并优雅返回空，绝不阻塞主流程）与 `_ingest`（真实抽取+写库，可抛异常）。
- `_retry_loop`：取出任务调 `_ingest`；失败且未超 `max_retries` 则退避后重新入队，超出则记日志丢弃。
- 新增 `retry_pending()`（队列长度，便于观测）、`stop()`。
- 性质：尽力而为的内存队列，**不跨进程重启持久化**（已在代码注释与 progress.md 标注）。

### graph_memory.py（导出）
- 新增 `export(filters, limit=1000)`：按 scope 查节点（name/type/mentions/user_id/project_id）与边（source/relationship/destination/valid），返回 `{nodes, relations, exported_at}`；禁用/未授权/异常均降级为空。

### routers/graph.py
- 新增 `GET /graph/export`（scope 契约同 get_all；admin 无 scope 返回空）。

### dashboard
- `GRAPH_ENDPOINTS.EXPORT = "/graph/export"`（api-endpoints.ts）。
- `graph/page.tsx` 工具栏新增「导出数据」按钮（`exportData()` 下载 `memgraph-data-<ts>.json`）。

### 测试
- `tests/test_graph_retry_queue.py`：FakeLLM/FakeEmbedder/RaisingNeo4j；验证 add 失败优雅入队、重试线程最终耗尽排空；driver 正常时不出队。
- `tests/test_graph_export.py`：FakeNeo4j 返回节点/边；验证结构、边界、异常降级。

## 验证
- `py_compile` graph_memory.py / routers/graph.py → OK。
- 容器 `pytest tests/test_graph_*.py tests/test_mcp_graph.py` → 30 passed；全量 → 122 passed，无回归。

## 偏差 / 注意
- 分支上存在预提交 `c3be02aa`（mem0-llm-provider-i18n WIP 打包，含一段更早的图谱导出/重试尝试）。其 graph_memory.py 改动已被本次干净覆盖——`graph_memory.py` 每个方法仅一份定义，无重复、无死代码。其余 WIP 文件（embeddings/openai.py、pgvector.py、main.py 等）不在图任务提交中。

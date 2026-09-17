# Graph Task 10 报告 — MCP 新增 graph 工具

**状态：✅ 完成** — commit `6295b8e9`（feat(mcp): expose graph tools (search_graph, get_all_graph)）

## 变更
- `server/mcp_server.py` 新增两个 MCP 工具（复用既有 `_client` 代理 mem0-api）：
  - `search_graph(query, api_key, git_remote?, project_id?, user_id?, limit=25)` → `POST /graph/search`。
  - `get_all_graph(api_key, git_remote?, project_id?, user_id?, limit=100)` → `GET /graph/get_all`。
  - 参数契约与 dashboard /graph 端点一致（project_id / user_id / git_remote 任一确定 scope）。
- `tests/test_mcp_graph.py`：用假 `_client` 断言动词/路径/参数（POST /graph/search、GET /graph/get_all 带 project_id/limit）。

## 验证
- `python -m py_compile mcp_server.py` → OK。
- 容器内 `pytest tests/test_mcp_graph.py` → 2 passed。

## 偏差
- 无；工具仅做代理转发，scope 校验在服务端 /graph 端点完成。

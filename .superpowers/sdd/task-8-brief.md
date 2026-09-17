# Task 8 (extracted from implementation plan)

- [ ] **Step 1: 全量重建** — `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-api mem0-dashboard` → 两 Built/Started 无 error。
- [ ] **Step 2: 全量 pytest** — 容器内 `pytest tests/ -q` → 全绿（92 + graph 新增 ≈ 105+）。
- [ ] **Step 3: Smoke** — `/graph/search`（带假数据路径：GRAPH_ENABLED=true + Neo4j 可达 + 空 LLM → relations 空数组 200）；`/search?include_graph=false` → 无 relations 字段；`/dashboard/graph` 307/200；Memories → 查看关联图跳转。
- [ ] **Step 4: 手动 E2E（有真实 key 的环境）** — add 含实体的记忆 → `:7474` 浏览器验证节点/边 → search 出 relations → delete 后软删。无 key 环境如实记录为运行时限制。
- [ ] **Step 5: 提交任何遗留修复**（mem0 子模块）。

---

## Self-Review Notes

- **Spec coverage:** §3 配置/开关 → Task 1/4；§4 数据模型/索引 → Task 1；§5 写入链路 → Task 3；§6 检索/删除 → Task 2/3；§8 可视化 → Task 5/6/7；§7 测试 → 各任务 + Task 8。
- **已知偏离（计划内）**：LLM 用纯 JSON prompt（2.x `generate_response` 无工具调用，1.x 是 tool-call）；`base_label` 固定启用；实体消歧限定在同 scope；图 LLM/embedder 绑定首次实例化（provider 变更后需重启，代码注释注明）。
- **Type consistency:** `GraphMemory.add/search/get_all/soft_delete_for_text/hard_delete` 签名在 Task 1 定义、Task 2/3 消费；前端 `GraphRelation` 与后端三元组键一致（`destination`，非 1.x 文档中的 `target`）。

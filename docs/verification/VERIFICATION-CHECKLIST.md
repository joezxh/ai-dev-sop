# mem0 记忆系统验证检查清单

> **版本**: v2.0
> **适用阶段**: Phase 5 - 验证与优化
> **目标读者**: 测试人员、新成员
> **变更说明**: 旧双轨记忆系统（MemPalace/cbmem-team/codebase-memory-mcp）已移除。

---

## 1. 环境验证

### 1.1 服务可达性

| 检查项 | 验证命令 | 期望结果 |
|--------|----------|----------|
| mem0 API | `curl -s http://localhost:8888/docs` | 返回 Swagger 页 |
| MCP 端点 | `curl -s http://127.0.0.1:8080/mcp` | 返回 serverInfo |
| Dashboard | 浏览器打开 `http://localhost:3001` | 登录页正常 |
| 容器状态 | `docker compose -f deploy/mem0/docker-compose.yaml ps` | 全部 healthy |

### 1.2 MCP 连接

| 检查项 | 验证方式 | 期望结果 |
|--------|----------|----------|
| IDE MCP 面板 | 重启 IDE 查看 | mem0 server 绿色/在线 |
| 工具列表 | 调用工具列表 | 出现 `add_memory` / `search_memories` 等 |
| 认证 | 携带 `api_key` 调用 | 无 401 |

---

## 2. 功能验证

### 2.1 记忆读写

| 功能 | 验证步骤 | 期望结果 |
|------|----------|----------|
| 写入记忆 | `add_memory(text="测试记忆", api_key=..., git_remote=...)` | 返回 memory_id |
| 语义搜索 | `search_memories(query="测试记忆")` | 召回刚写入的条目 |
| 全量拉取 | `get_memories(project_id="ai-dev-sop")` | 返回项目共享池 |
| 更新记忆 | `update_memory(memory_id, text="更新")` | 更新成功 |
| 删除记忆 | `delete_memory(memory_id)` | 删除成功 |

### 2.2 项目共享池

| 功能 | 验证步骤 | 期望结果 |
|------|----------|----------|
| 跨用户读取 | 成员 A 写入 → 成员 B `get_memories(project_id=...)` | B 能读到 A 的记忆 |
| git_remote 解析 | 携带 `git_remote` 写入后查看 metadata | `project_id` 正确写入 |
| 个人隔离 | 不带 project_id 写入 → 其他用户检索 | 不可见 |

### 2.3 会话留痕

| 功能 | 验证步骤 | 期望结果 |
|------|----------|----------|
| 自动入库 | 对话一轮后 `search_memories(query="Q: <本轮问题>")` | 该轮 Q/A 已入库 |
| session 分组 | Dashboard → Memories 按 `metadata.session_id` 过滤 | 同会话各轮完整有序 |
| 格式正确 | 查看记忆正文 | `Q:`/`A:` 分段，A 为 Markdown 全量回复 |

### 2.4 图记忆（GRAPH_ENABLED=true 时）

| 功能 | 验证步骤 | 期望结果 |
|------|----------|----------|
| Neo4j 连通 | 容器日志无图存储报错 | 正常 |
| 实体关系 | `search_graph`（如有启用图检索）或 Dashboard Graph | 实体关系可见 |

---

## 3. 集成验证

| 检查项 | 验证方式 | 期望结果 |
|--------|----------|----------|
| 自动拉取 | 新会话首轮，Agent 是否汇报召回摘要 | CODEBUDDY.md §1 行为触发 |
| 自动提交 | 陈述一个持久事实，观察是否自动 `add_memory` | CODEBUDDY.md §2 行为触发 |
| 每轮留痕 | 任意一轮对话后查询 | type=conversation 记录存在 |
| 失败降级 | 停止 mem0-api 后会话 | Agent 告知服务不可达且不编造记忆 |

---

## 4. 验证脚本

```bash
#!/bin/bash
# smoke-mem0.sh
set -e
curl -sf http://localhost:8888/docs > /dev/null && echo "API OK"
curl -sf http://localhost:3001 > /dev/null && echo "Dashboard OK"
docker compose -f deploy/mem0/docker-compose.yaml ps | grep -q healthy && echo "Containers OK"
echo "Manual MCP checks: add_memory → search_memories in your IDE."
```

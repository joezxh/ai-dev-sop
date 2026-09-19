# 记忆系统工具速查表（mem0）

> **版本**: v2.1
> **最后更新**: 2026-09-19
> **变更说明**: 移除旧工具映射表；收录《mem0 配置手册》（见 [mem0-manual.md](./mem0-manual.md)）。

---

## 1. 连接与凭证

| 配置项 | 位置 | 内容 |
|--------|------|------|
| MCP 连接 | `~/.codebuddy/mcp.json` → `mem0-local` | `http://127.0.0.1:8080/mcp`（streamable-http） |
| 凭证与作用域 | `.codebuddy/mem0.config.json`（git-ignored） | `api_key` / `admin_user_id` / `project_id` / `git_remote` / `roster` |
| 服务部署 | `deploy/mem0/docker-compose.yaml` | mem0-api（8888/8080）+ mem0-dashboard（3001） |
| 项目 ID | 服务端解析 `git_remote` | `ai-dev-sop` |
| 规则文件 | 根 `CODEBUDDY.md` | 自动拉取/提交的行为约定 |

完整接入手册见 [mem0-manual.md](./mem0-manual.md)。

---

## 2. MCP 工具速查（mem0-local）

### 2.1 读取工具

| 工具 | 关键参数 | 说明 | 示例 |
|------|----------|------|------|
| `get_memories` | `api_key`, `git_remote`/`project_id`, `limit` | 拉取项目共享池（跨用户） | `get_memories(api_key, project_id="ai-dev-sop")` |
| `search_memories` | `query`, `api_key`, `top_k`, `project_id` | 语义搜索 | `search_memories("支付网关决策", top_k=5)` |
| `get_memory` | `memory_id` | 单条读取 | `get_memory("abc123")` |
| `list_entities` | - | 枚举 user/agent/run 实体 | - |
| `list_events` | 过滤+分页 | 记忆操作流水（审计） | - |

### 2.2 写入工具

| 工具 | 关键参数 | 说明 |
|------|----------|------|
| `add_memory` | `text`, `api_key`, `git_remote`, `user_id`, `infer`, `metadata` | 存储记忆（`infer=False` 原文留痕；`True` LLM 抽取） |
| `update_memory` | `memory_id`, `text` | 覆写单条记忆 |

### 2.3 删除工具（谨慎）

| 工具 | 说明 |
|------|------|
| `delete_memory` | 按 `memory_id` 删除单条（须用户明确要求） |
| `delete_all_memories` | ⚠️ 范围清空，不可逆 |
| `delete_entities` | 删除实体及其全部记忆 |

### 2.4 metadata 约定

```json
{
  "type": "fact | decision | preference | note | conversation",
  "people": ["<相关人>"],
  "by": "<提交者>",
  "session_id": "cb-<YYYYMMDD>-<6hex>",
  "agent": "CodeBuddy",
  "created_at": "<ISO8601>"
}
```

---

## 3. 验证

```text
1. docker compose -f deploy/mem0/docker-compose.yaml ps   # mem0-api healthy
2. MCP 面板 mem0-local 状态绿色
3. search_memories("冒烟测试") → 有返回
4. Dashboard → Memories → 可见 type=conversation 会话流
```

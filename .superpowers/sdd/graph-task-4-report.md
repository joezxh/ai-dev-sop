# Task 4 报告：GRAPH_* 配置接线 + Dockerfile 清理无效安装

**日期：** 2026-02-27（执行日）
**状态：** ✅ 完成

## 改动内容

### 1. `deploy/mem0/.env`（外层仓库，untracked）
末尾追加：

```
# ---- Graph memory (self-built, 1.x aligned) ----
GRAPH_ENABLED=true
GRAPH_THRESHOLD=0.7
GRAPH_CUSTOM_PROMPT=
```

### 2. `deploy/mem0/docker-compose.yaml`
`mem0-api.environment` 追加三条：

```yaml
- GRAPH_ENABLED=${GRAPH_ENABLED:-true}
- GRAPH_THRESHOLD=${GRAPH_THRESHOLD:-0.7}
- GRAPH_CUSTOM_PROMPT=${GRAPH_CUSTOM_PROMPT:-}
```

### 3. `deploy/mem0/api/Dockerfile`
删除无效的 `"mem0ai[graph]"` extra 安装（mem0ai 2.x 无该 extra），仅安装
`rank-bm25 langchain-neo4j neo4j`，并加注释说明：图引擎 2.x 不存在，图层自建于
`server/graph_memory.py`（基于 langchain-neo4j）。

## 验证结果

| 项目 | 结果 |
| --- | --- |
| `docker compose up -d --build mem0-api` | ✅ Built / Started（镜像 mem0-local-mem0-api，sha256:873cdcef…） |
| 容器内 `env \| Select-String GRAPH` | ✅ 三条均在：`GRAPH_ENABLED=true`、`GRAPH_THRESHOLD=0.7`、`GRAPH_CUSTOM_PROMPT=`（空） |
| 容器内 `pytest tests/ -q` | ✅ **116 passed**, 3 warnings（仅既有 DeprecationWarning，9.49s） |
| 启动日志 graph 检查 | ✅ 无 "GraphMemory degraded" 或任何 graph 相关报错（Neo4j 可达，图层正常启用） |

## 提交

| 仓库 | 哈希 | 说明 |
| --- | --- | --- |
| 外层 ai-dev-sop | **852b791** | `chore(deploy): wire GRAPH_* env and drop no-op mem0ai[graph] install`（2 files changed, 116 insertions；`deploy/mem0/api/Dockerfile` 与 `deploy/mem0/docker-compose.yaml` 首次入库 create mode） |
| mem0 子模块 | 无新提交（HEAD 仍为 e8e2cc34） | 本任务改动全部在外层 deploy/ 目录，子模块 Dockerfile 不存在，无需提交 |

## .env 入库说明（concerns）

- `deploy/mem0/.env` 在外层仓库中为 **untracked（??）**，本次**未提交**——保持现状。
  该文件含本机密码（`POSTGRES_PASSWORD`、`NEO4J_PASSWORD`、`JWT_SECRET`），
  属本地部署机密，入库会泄露敏感值；仅 `docker-compose.yaml`（通过 `${VAR}` 引用）
  入库即可。如需可追加到 `.gitignore`（当前靠 untracked 状态维持）。
- 外层 `deploy/mem0/docker-compose.yaml` 与 `deploy/mem0/api/` 此前也是 untracked，
  本次提交使其首次入库（符合任务指定的 add 范围）。
- 子模块遗留工作区改动（`server/docker-compose.yaml`、`server/requirements.txt` 修改、
  `favicon.ico` 删除、`server/data/`、`server/scripts/seed.sql` 未跟踪）与本任务无关，
  **未提交、未改动**，保持原样待早前工作自行收尾。

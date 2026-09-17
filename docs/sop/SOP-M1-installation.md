# SOP-M1: mem0 记忆系统安装配置指南

> **版本**: v2.0
> **最后更新**: 2026-09-18
> **变更说明**: 旧双轨记忆系统（MemPalace :8089 / cbmem-team :8787 / codebase-memory-mcp）
> 已于 2026-09 移除，本指南更新为自托管 mem0 的安装配置。

---

## 1. 概述

### 1.1 目标

完成 mem0 记忆系统的完整安装配置，使团队成员能够在 30 分钟内完成环境搭建。

### 1.2 系统组件

```
┌─────────────────────────────────────────────────────────────┐
│                    mem0 记忆系统                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   AI IDE (CodeBuddy / Qoder / Cursor / Codex ...)            │
│            │  MCP (streamable-http)                          │
│            ▼                                                 │
│   ┌──────────────────────┐    ┌──────────────────────┐      │
│   │  mem0-api :8888      │    │ mem0-dashboard :3001 │      │
│   │  MCP 端点 :8080/mcp  │    │ (管理控制台)          │      │
│   └──────────┬───────────┘    └──────────────────────┘      │
│              ▼                                               │
│   ┌──────────────────────┐    ┌──────────────────────┐      │
│   │ PostgreSQL (pgvector)│    │      Neo4j           │      │
│   │  向量记忆存储          │    │   图记忆存储          │      │
│   └──────────────────────┘    └──────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 前置条件

- Docker & Docker Compose
- 复用 `mwb-infra-network` 网络中的 `mwb-postgres-pgvector` 与 `mwb-neo4j` 容器

---

## 2. 服务端安装

### 2.1 一键安装（推荐）

```bash
# Linux/macOS
./scripts/install-all.sh

# Windows
./scripts/install-all.ps1
```

### 2.2 手动部署

```bash
# 1. 进入部署目录
cd deploy/mem0

# 2. 准备环境变量（首次）
#    参考 docker-compose.yaml 头部注释：
#    POSTGRES_PASSWORD / NEO4J_PASSWORD / JWT_SECRET / GRAPH_ENABLED 等
cp .env.example .env 2>/dev/null || vi .env

# 3. 启动服务
docker compose up -d

# 4. 验证
curl -s http://localhost:8888/docs > /dev/null && echo "API OK"
curl -s http://localhost:3001 > /dev/null && echo "Dashboard OK"
```

### 2.3 关键配置项（`deploy/mem0/.env`）

| 变量 | 说明 | 默认 |
|------|------|------|
| `POSTGRES_PASSWORD` | pgvector 数据库密码 | - |
| `NEO4J_PASSWORD` | 图数据库密码 | - |
| `JWT_SECRET` | API/Dashboard 会话签名密钥 | - |
| `GRAPH_ENABLED` | 图记忆开关 | `true` |
| `GRAPH_THRESHOLD` | 图边相似度阈值 | `0.7` |
| `AUTH_DISABLED` | 认证开关（**对外暴露必须 false**） | `false` |

### 2.4 存储

| 存储 | 用途 | 说明 |
|------|------|------|
| PostgreSQL + pgvector | 向量记忆 | 数据库 `mem0`，集合 `memories` |
| Neo4j | 图记忆/实体关系 | `GRAPH_ENABLED=true` 时使用 |
| 应用库 | 用户/Key/Dashboard 数据 | `mem0_app`（首次启动自动创建） |

---

## 3. API Key 与凭证

```text
1. 打开 Dashboard: http://localhost:3001 → 登录
2. Settings → API Keys → Create API Key
3. 密钥仅创建时完整可见（格式 m0-...，之后哈希脱敏），立即妥善保存
4. 项目级凭证写入 .codebuddy/mem0.config.json（git-ignored）：
   api_key / admin_user_id / project_id / git_remote / roster
```

---

## 4. 客户端（IDE）接入

配置模板全集见 [docs/ide-config/ide-mcp-templates.md](../ide-config/ide-mcp-templates.md)，速览：

| IDE | 配置位置 | 方式 |
|-----|----------|------|
| CodeBuddy | Settings → MCP → Add MCP | HTTP `http://127.0.0.1:8080/mcp` |
| Cursor | `~/.cursor/mcp.json` | HTTP + Bearer Key |
| Qoder | 个人设置 → MCP 服务 | HTTP/SSE（插件 ≥ v2.5.0） |
| Codex | `~/.codex/config.toml` | `bearer_token_env_var = "MEM0_API_KEY"` |

示例（Cursor）：

```json
{
  "mcpServers": {
    "mem0": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "headers": { "Authorization": "Bearer m0-你的密钥" }
    }
  }
}
```

---

## 5. 验证清单

### 5.1 服务端

| 检查项 | 命令 | 期望结果 |
|--------|------|----------|
| API 健康 | `curl -s http://localhost:8888/docs` | 返回 Swagger 页 |
| MCP 端点 | `curl -s http://127.0.0.1:8080/mcp` | 返回 MCP serverInfo |
| Dashboard | 浏览器打开 `http://localhost:3001` | 登录页正常 |
| 容器状态 | `docker compose -f deploy/mem0/docker-compose.yaml ps` | 全部 healthy |

### 5.2 IDE 端

| 检查项 | 操作 | 期望结果 |
|--------|------|----------|
| MCP 连接 | IDE 重启后查看 MCP 面板 | mem0 显示在线（绿色） |
| 工具列表 | 调用工具列表 | 出现 `add_memory` / `search_memories` 等 |
| 语义搜索 | `search_memories("冒烟测试")` | 返回结果或空列表（非报错） |
| 写入回读 | `add_memory` 后再 `search_memories` | 能召回刚写入的记录 |

### 5.3 团队共享池验证

```text
带 git_remote 写入 → 其他成员 get_memories(git_remote=<同一远程地址>) 可读到
```

---

## 6. 故障排查

| 现象 | 原因 | 解决 |
|------|------|------|
| 连接拒绝 | mem0-api 未启动 | `docker compose up -d`；确认 `mwb-postgres-pgvector` / `mwb-neo4j` healthy |
| 401 | Key 无效/未传 | 重新创建 Key；检查 Bearer 头格式 |
| MCP 404 | URL 少 `/mcp` | 使用 `http://127.0.0.1:8080/mcp` |
| 图记忆报错 | Neo4j 不可达 | 检查 `.env` 中 `NEO4J_URI/USERNAME/PASSWORD` |
| 检索为空 | Key/Project 不匹配 | 确认 Key 属于同一 Project；user_id 一致 |
| Dashboard 无法登录 | JWT_SECRET 变更 | 清浏览器缓存重新登录 |

更多排障见 [quick-ref/mem0-ai-tools-config-guide.md §10](../quick-ref/mem0-ai-tools-config-guide.md)。

---

## 7. 历史版本说明

旧双轨记忆系统的安装流程（MemPalace Docker 部署、cbmem-team 编译/systemd/MySQL 后端、
codebase-memory-mcp 二进制安装、JWT Token 分发）已随 `tools/` 目录移除。
历史文档见 git 历史（`git log -- tools/`）与 `docs/superpowers/specs/` 归档设计文档。

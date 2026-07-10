# cbmem-team Console

> 双轨记忆控制台 — 管理界面，覆盖用户管理、项目管理、会话记录、会话归纳、会话蒸馏五大模块。

## 启用

```bash
cbmem-team \
  -listen :8787 \
  -data /var/lib/cbmem-team \
  -mcp-bin /usr/local/bin/codebase-memory-mcp \
  -jwt-secret "$(cat /etc/cbmem-team/jwt-secret)" \
  -admin-token "$(cat /etc/cbmem-team/admin-token)" \
  -llm-provider openai \
  -llm-model gpt-4o-mini \
  -llm-base-url https://api.openai.com/v1 \
  -llm-api-key "$(cat /etc/cbmem-team/llm-api-key)" \
  -mempalace-base http://192.168.100.83:8089 \
  -console-dist /opt/cbmem-console/dist
```

### 必需 Flag

| Flag | 说明 | 示例 |
|------|------|------|
| `-listen` | HTTP 监听地址 | `:8787` |
| `-data` | 数据目录（含 SQLite） | `/var/lib/cbmem-team` |
| `-mcp-bin` | `codebase-memory-mcp` 二进制路径 | `/usr/local/bin/codebase-memory-mcp` |
| `-jwt-secret` | JWT 签名密钥 | `openssl rand -hex 32` |
| `-admin-token` | 控制台管理员静态 token | `openssl rand -hex 32` |
| `-console-dist` | VitePress 构建产物目录 | `/opt/cbmem-console/dist` |

### LLM Flag（归纳/蒸馏引擎）

| Flag | 说明 | 默认 |
|------|------|------|
| `-llm-provider` | `openai` / `ollama` / `fake` | `fake` |
| `-llm-model` | 模型名称 | `gpt-4o-mini` |
| `-llm-base-url` | API base URL | `https://api.openai.com/v1` |
| `-llm-api-key` | API 密钥 | `sk-...` |
| `-llm-timeout` | 超时时长 | `30s` |

### MemPalace Flag

| Flag | 说明 | 默认 |
|------|------|------|
| `-mempalace-base` | MemPalace HTTP 地址 | `http://localhost:8089` |
| `-mempalace-timeout` | 超时时长 | `10s` |

## 登录

浏览器打开 `http://server:8787/console/`，输入 admin token 登录。

## 控制台模块

| 模块 | 路径 | 说明 |
|------|------|------|
| 用户管理 | `/console/#/users` | 用户 CRUD + Token 签发/撤销 |
| 项目管理 | `/console/#/projects` | 项目 CRUD + Wing 绑定 + 触发索引 |
| 会话记录 | `/console/#/sessions` | 会话列表 + 详情 + 统计 |
| 会话归纳 | `/console/#/summarize` | 选会话 → 5 Hall 结果 → 写入 MemPalace |
| 会话蒸馏 | `/console/#/distill` | 选会话 → 知识片段/决策/技术债务 → 写入 MemPalace |

## API 速查

| Path | Method | 说明 |
|------|--------|------|
| `/api/console/login` | POST | admin-token 登录 |
| `/api/console/logout` | POST | 登出 |
| `/api/console/me` | GET | 当前管理员 |
| `/api/console/stats` | GET | 全局统计 |

### 用户（CON-01）

| Path | Method | 说明 |
|------|--------|------|
| `/api/console/users` | GET | 列表，`?q=` 搜索 |
| `/api/console/users` | POST | 新增 `{id, display_name, project_paths[]}` |
| `/api/console/users/:id` | PUT | 更新 `{display_name, project_paths[], disabled}` |
| `/api/console/users/:id` | DELETE | 删除 |
| `/api/console/users/:id/token` | POST | 签发 token，`?ttl=720h` |
| `/api/console/users/:id/revoke` | POST | 撤销全部 token |

### 项目（CON-02）

| Path | Method | 说明 |
|------|--------|------|
| `/api/console/projects` | GET | 列表 |
| `/api/console/projects` | POST | 新增 `{name, path, wing, mcp_bin}` |
| `/api/console/projects/:id` | DELETE | 删除 |
| `/api/console/projects/:id/reindex` | POST | 触发索引 |

### 会话（CON-03）

| Path | Method | 说明 |
|------|--------|------|
| `/api/console/sessions` | GET | 列表，`?user_id=&project_id=&from=&to=` |
| `/api/console/sessions/:id` | GET | 详情（含 turns） |
| `/api/console/sessions-stats` | GET | 统计聚合 |

### 归纳（CON-04）

| Path | Method | 说明 |
|------|--------|------|
| `/api/console/summarize` | POST | 触发 `{source_ids[], depth, target_wing}` |
| `/api/console/summarize/:task_id` | GET | 任务状态/结果 |

### 蒸馏（CON-05）

| Path | Method | 说明 |
|------|--------|------|
| `/api/console/distill` | POST | 触发 `{source_ids[], rules}` |
| `/api/console/distill/:task_id` | GET | 任务状态/结果 |
| `/api/console/distill/:task_id/commit` | POST | 写入 MemPalace `{target_wing}` |

## 统一响应格式

```json
{ "code": 0, "msg": "", "data": {} }
```

错误码：

| 区间 | 含义 |
|------|------|
| 0 | 成功 |
| 40100xx | 鉴权失败（admin-token 错 / session 过期 / CSRF） |
| 40300xx | 无权（白名单违规 / 用户 disabled） |
| 40900xx | 资源冲突（如 user id 重复） |
| 50000xx | 系统错误（LLM 超时 / SQLite 写入失败） |

## 会话采集（MCP Tap）

cbmem-team 在 MCP handler 外包一层 capture middleware，自动采集：

- 用户每次发送消息 → 创建/复用 `sessions` 行
- 每轮对话写入 `session_turns`（role, content, tools_json）
- 采集触发：JSON-RPC `initialize` / `tools/call` 含 `messages/create`
- session 复用策略：`(user_id, project_path)` 30 分钟内有最近 session 则复用

## 数据库表

| 表名 | 用途 |
|------|------|
| `users` | 用户信息（含 project_paths 白名单） |
| `projects` | 项目配置（含 wing / mcp_bin） |
| `sessions` | 会话记录 |
| `session_turns` | 会话轮次 |
| `summarize_tasks` | 归纳任务 |
| `distill_tasks` | 蒸馏任务 |
| `console_sessions` | 控制台登录 session（含 TTL） |

## systemd 部署

```ini
[Unit]
Description=cbmem-team HTTP wrapper + Console

[Service]
ExecStart=/usr/local/bin/cbmem-team \
  -listen :8787 \
  -data /var/lib/cbmem-team \
  -mcp-bin /usr/local/bin/codebase-memory-mcp \
  -jwt-secret /run/secrets/cbmem-jwt-secret \
  -admin-token /run/secrets/cbmem-admin-token \
  -llm-provider openai \
  -llm-model gpt-4o-mini \
  -llm-base-url https://api.openai.com/v1 \
  -llm-api-key /run/secrets/cbmem-llm-api-key \
  -mempalace-base http://192.168.100.83:8089 \
  -console-dist /opt/cbmem-console/dist \
  -log info
Restart=always
User=cbmem
```

## 前端构建

```bash
cd docs-site
npm install
npm run build          # VitePress 构建产物在 .vitepress/dist/
# 将产物 symlink 到 -console-dist 指定路径
```

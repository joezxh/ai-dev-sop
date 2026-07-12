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

### 工作流（CON-06 · M3）

工作流是 console 端的「可发布 + 可执行」图节点序列，每个节点是
一种 `kind`（`log` / `repo-pipeline` / `bp-candidate-review` 等）。
内置 3 个工作流随迁移自动 seed；用户可创建 + publish 后通过
`/run` 端到端触发。

| Path | Method | 说明 |
|------|--------|------|
| `/api/console/v2/workflows` | GET | 列表（含内置 + 用户），`?category=&track=&status=` |
| `/api/console/v2/workflows` | POST | 新建 `{name, category, track, entry_id, nodes[]}` |
| `/api/console/v2/workflows/:id` | GET | 详情 |
| `/api/console/v2/workflows/:id` | PUT | 更新节点图（仅 draft） |
| `/api/console/v2/workflows/:id` | DELETE | 删除 |
| `/api/console/v2/workflows/:id/publish` | POST | draft → published（只读图） |
| `/api/console/v2/workflows/:id/archive` | POST | 归档 |
| `/api/console/v2/workflows/:id/run` | POST | 同步执行，返回 `run_id` + `step_count` |

内置工作流（`wf-*` 前缀，随 migration 自动 seed）：

| ID | 名称 | 节点数 | 用途 |
|----|------|--------|------|
| `wf-commit-precheck` | Commit 前置检查 | 7 | 高风险提交前自动跑 7 步合规校验 |
| `wf-adr-doublewrite` | ADR 双写 | 6 | 把 ADR 同时写到 MemPalace + 文件 |
| `wf-repo-daily-sync` | 仓库每日同步 | 4 | 每日 cron 拉取 repo pipeline 增量 |

### Repo Pipeline（CON-07 · M4）

repo pipeline 把外部内容（本地 markdown / GitHub commit / RSS feed）
通过 6 阶段 runtime（crawl → parse → grade → sink → rehearse）落地
为 `bp_candidates`，由人工 review 后接受 / 拒绝 / 合并到
`best_practices`。

| Path | Method | 说明 |
|------|--------|------|
| `/api/console/v2/repos` | GET | 列表，?status=draft\|active\|archived |
| `/api/console/v2/repos` | POST | 新建 `{name, source, target, cron_expr?, threshold?}` |
| `/api/console/v2/repos/:id` | GET | 详情 |
| `/api/console/v2/repos/:id` | PUT | 更新（仅 draft / archived） |
| `/api/console/v2/repos/:id/activate` | POST | draft → active |
| `/api/console/v2/repos/:id/archive` | POST | active → archived |
| `/api/console/v2/repos/:id/run` | POST | 同步执行一次 pipeline |
| `/api/console/v2/repos/:id/runs` | GET | 该 pipeline 的执行历史 |
| `/api/console/v2/repos/runs/:run_id` | GET | 单次 run 的 stage 状态 + 计数 |
| `/api/console/v2/repos/candidates` | GET | 候选 BP 列表（见下文） |
| `/api/console/v2/repos/candidates/:id` | GET | 单条候选详情 |
| `/api/console/v2/repos/candidates/:id/accept` | POST | 接受，状态变为 `accepted` |
| `/api/console/v2/repos/candidates/:id/reject` | POST | 拒绝，状态变为 `rejected` |
| `/api/console/v2/repos/candidates/:id/merge` | POST | 合并到 `best_practices` |

`GET /api/console/v2/repos/candidates` 查询参数：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `status` | string | — | 精确匹配 `draft` / `accepted` / `rejected` / `merged` |
| `pipeline_id` | string | — | 限制为某条 pipeline，例如 `?pipeline_id=rp-xxxxxxxx` |
| `limit` | int | 50 | 分页大小，超出 [1,500] 强制回到 50 |

返回结构：`{"candidates": [...], "count": N}`，按 `created_at DESC` 排序。

示例：

```bash
# 列出某条 pipeline 当前所有的 draft 候选
curl -b cookies.txt \
  'http://127.0.0.1:8787/api/console/v2/repos/candidates?pipeline_id=rp-abc123&status=draft'

# 全局最近 200 条候选（接受 / 拒绝 / 已合并）
curl -b cookies.txt \
  'http://127.0.0.1:8787/api/console/v2/repos/candidates?limit=200'
```

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

## 数据库（SQLite · MySQL 双轨）

cbmem-team 支持两种后端，通过启动 flag 切换：

| 后端 | 启用方式 | 何时用 |
|------|---------|--------|
| **SQLite**（默认） | 不设 `-mysql-dsn` flag | 单机 / 开发 / 小团队 |
| **MySQL 8.0** | `-mysql-dsn <dsn>` | 生产 / 多实例 / 高并发 |

**两套 systemd 单元**：
- `deploy/cbmem-team.service`（旧 v2）—— SQLite 路径，仍保留作为**回退**（7 天窗口）
- `deploy/cbmem-team-mysql.service`（v3）—— MySQL 路径，**生产主用**

### MySQL 部署清单

1. 起 MySQL：`cd tools/cbmem-team/deploy && docker compose -f docker-compose.mysql.yml up -d`
2. 写 DSN：`sudo cp deploy/mysql.env.example /etc/cbmem-team/mysql.env && sudo chmod 0600 /etc/cbmem-team/mysql.env`
3. 改 `/etc/cbmem-team/mysql.env` 中 `CBMEM_MYSQL_PASSWORD` 为强随机串
4. 建表：`cbmem-team migrate-tables -mysql-dsn "$(grep CBMEM_MYSQL_DSN /etc/cbmem-team/mysql.env | cut -d= -f2)"`
5. ETL（首次迁移）：`./examples/mysql-etl.sh --mysql-dsn "$(grep CBMEM_MYSQL_DSN /etc/cbmem-team/mysql.env | cut -d= -f2)"`
6. 切换：`sudo systemctl disable cbmem-team && sudo systemctl enable --now cbmem-team-mysql`
7. 装 cron：`sudo cp examples/mysql-backup.sh /etc/cron.daily/cbmem-mysql-backup`

### 7 天回退窗口

`cbmem-team` 二进制启动时**只读** `-data` 中 `cbmem-team.db`。任何时候：
```bash
sudo systemctl stop cbmem-team-mysql
sudo systemctl start cbmem-team    # SQLite 旧单元，无 -mysql-dsn flag
```
即可瞬时回退到 SQLite，验证后再切回 MySQL。

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

# Mem0 × AI 开发工具配置手册

> **版本**: v1.1（新增 §9 自动提交与拉取的规则文件编写说明）
> **最后更新**: 2026-09-18
> **适用对象**: 开发者（覆盖初次接入 → 进阶配置）
> **覆盖工具**: CodeBuddy / Qoder / Cursor / Codex / Claude Code / VS Code / Windsurf

---

## 1. Mem0 接入形态总览

Mem0 向 AI 开发工具提供记忆能力有三种形态，接入前先做选型：

| 形态 | 端点 | 适用场景 | 维护方 |
|------|------|----------|--------|
| **A. 官方云 MCP（推荐）** | `https://mcp.mem0.ai/mcp` | 直接使用 Mem0 Cloud，零部署 | Mem0 官方 |
| **B. 自托管 OSS Server** | `http://localhost:8888`（API）/ `http://127.0.0.1:8080/mcp`（MCP） | 数据不出本地的私有化部署（本仓库 `deploy/mem0/`） | 自维护 |
| **C. OpenMemory MCP** | 本地 stdio 进程 | 多客户端共享同一本地记忆库，带本地管理 UI | Mem0 官方开源 |

> ⚠️ 社区版 `mem0-mcp-server`（`mem0ai/mem0-mcp` 仓库）已于 **2026-03-24 归档停更**，stdio 方式（`uvx mem0-mcp-server`）仍可使用，但新接入建议优先形态 A 或 B。

```
                    ┌─────────────────────────────┐
   CodeBuddy ──────┐│                             │┌────── Mem0 Cloud
   Qoder ──────────┼│   MCP 协议 (stdio / HTTP)   ││       (mcp.mem0.ai)
   Cursor ─────────┤│                             │├────── 自托管 (localhost:8080/mcp)
   Codex ──────────┤│  add / search / get / update││       (deploy/mem0)
   Claude Code ────┤│  delete / entities / events │├────── OpenMemory (本地)
   VS Code ────────┘│                             ││└────── 共享记忆池
                    └─────────────────────────────┘
```

---

## 2. 支持的 AI 开发工具及版本要求

| 工具 | 最低版本要求 | MCP 传输 | 快速配置方式 | 认证方式 |
|------|--------------|----------|--------------|----------|
| **CodeBuddy IDE** | 当前稳定版 | stdio / HTTP | Settings → MCP → Add MCP（JSON） | API Key |
| **Qoder** | JetBrains/VS Code 插件 **v2.5.0+**；智能体模式 + qwen3 | stdio / SSE | 个人设置 → MCP 服务（配置文件） | API Key |
| **Cursor** | 2024.10+ | HTTP / stdio | `mcp-add` 一键 / 手动 `mcp.json` | OAuth 登录 / API Key |
| **Codex CLI** | 支持 `config.toml` MCP 的版本 | HTTP（手写 TOML）/ stdio | 手动编辑 `~/.codex/config.toml` | API Key（环境变量） |
| **Claude Code** | Node.js 18+ | HTTP / stdio | `mcp-add` | OAuth 登录 / API Key |
| **VS Code (Copilot)** | 支持 MCP 的版本 | HTTP / stdio | `mcp-add` | OAuth 登录 / API Key |
| **Windsurf** | 支持 MCP 的版本 | HTTP / stdio | `mcp-add` | OAuth 登录 / API Key |

**通用运行时依赖**（stdio 形态需要）：

- Node.js **≥ v18**（npm ≥ v8）→ 提供 `npx`
- Python **≥ 3.8** + [uv](https://docs.astral.sh/uv/) → 提供 `uvx`（Python 版 MCP server）
- Codex 一键配置 `npx mcp-add` 同样依赖 Node.js 18+

---

## 3. 通用准备：API 密钥

### 3.1 密钥获取

| 形态 | 获取位置 |
|------|----------|
| Mem0 Cloud | https://app.mem0.ai/dashboard/api-keys → **Create API Key** |
| 自托管 | 本地 Dashboard `http://localhost:3001` 登录后创建 |

### 3.2 密钥格式与安全

- 格式：`m0-<32位随机串>`，如 `m0-iMmVU*****************************0q2m`
- ⚠️ **密钥仅创建时完整可见一次**，云端立即哈希化存储；丢失只能重建并更新集成
- 密钥按 组织/项目 隔离：一把 Key 只能访问其所属 Project 的记忆

### 3.3 环境变量约定（推荐统一）

```powershell
# PowerShell（用户级永久）
[Environment]::SetEnvironmentVariable("MEM0_API_KEY", "m0-你的密钥", "User")
# bash / zsh
export MEM0_API_KEY=m0-你的密钥
```

各工具配置中一律通过环境变量引用密钥，**避免把明文 Key 写进提交到仓库的配置文件**。

---

## 4. 各工具接入配置

### 4.1 CodeBuddy IDE

**入口**：侧栏对话面板右上角 **CodeBuddy Settings** → **MCP** 标签页 → **Add MCP**

**方式一：HTTP（官方云 / 自托管）**

```json
{
  "mcpServers": {
    "mem0": {
      "type": "http",
      "url": "https://mcp.mem0.ai/mcp",
      "headers": {
        "Authorization": "Bearer m0-你的密钥"
      },
      "description": "Mem0 long-term memory"
    }
  }
}
```

> 自托管场景把 `url` 换为 `http://127.0.0.1:8080/mcp`。若 MCP server 自身用 api_key 鉴权（见 §6.2），也可以用 `api_key` 调用参数方式传递（mem0-local server 的每个工具都接受 `api_key` 参数）。

**方式二：stdio（Python mem0-mcp-server）**

```json
{
  "mcpServers": {
    "mem0": {
      "type": "stdio",
      "command": "uvx",
      "args": ["mem0-mcp-server"],
      "env": {
        "MEM0_API_KEY": "m0-你的密钥",
        "MEM0_DEFAULT_USER_ID": "joezxh"
      },
      "description": "Mem0 memory (stdio)"
    }
  }
}
```

**验证**：MCP 列表项显示绿色状态，或点击 **Try to Run**；随后在 Craft Agent 中说"记住我偏好 TypeScript"并复述验证。

### 4.2 Qoder

**前提**：插件 ≥ v2.5.0；**智能体模式**（未打开工程目录时是智能问答模式，无法调用 MCP）；最多同时 10 个 MCP 服务。

**入口**：右上角头像 → **个人设置** → **MCP 服务** → 右上角 **"+"** → 配置文件添加

**stdio 示例**：

```json
{
  "mcpServers": {
    "mem0": {
      "command": "uvx",
      "args": ["mem0-mcp-server"],
      "env": {
        "MEM0_API_KEY": "m0-你的密钥",
        "MEM0_DEFAULT_USER_ID": "joezxh"
      }
    }
  }
}
```

**SSE / 远程示例**（表单方式）：类型选 `SSE`，服务地址填：

```
https://mcp.mem0.ai/mcp        # 官方云（Streamable HTTP）
http://127.0.0.1:8080/mcp      # 自托管
```

> 注意：Qoder 的配置为**用户级全局**，添加后跨工程、跨 IDE 生效；Qoder CN 暂不支持白屏化添加 HTTP 类型时自定义 Header，API Key 建议走 stdio `env` 或服务端参数方式。

### 4.3 Cursor

**方式一：一键配置（需 Node.js 18+）**

```bash
npx mcp-add --name mem0-mcp --type http \
  --url "https://mcp.mem0.ai/mcp" --clients "cursor"
```

**方式二：手动编辑** `~/.cursor/mcp.json`：

```json
{
  "mcpServers": {
    "mem0-mcp": {
      "type": "http",
      "url": "https://mcp.mem0.ai/mcp"
    }
  }
}
```

保存后 Cursor 会弹出浏览器 OAuth 授权；CI/headless 场景改用 API Key（§6.2）。Mem0 另提供 Cursor **插件版**（含自动记忆 hooks，无需手动 add/search），从 `docs.mem0.ai/integrations/cursor` 安装，首次使用时填入 API Key 后重启 Cursor。

### 4.4 Codex CLI

⚠️ `codex mcp add` 命令**仅支持 stdio**；HTTP 类型的 server 必须**直接手写 `~/.codex/config.toml`**：

```toml
[mcp_servers.mem0]
url = "https://mcp.mem0.ai/mcp"
bearer_token_env_var = "MEM0_API_KEY"
```

然后确保启动 Codex 的 shell 中已 `export MEM0_API_KEY=m0-...`。

- 服务名建议用 `mem0`（TOML 键名即服务名，无需与其他客户端保持一致）
- Codex 也可安装 `mem0ai/mem0` 仓库提供的插件（memory protocol skill + lifecycle hooks 自动读写），但**不可与 Direct MCP 方式混用**

### 4.5 Claude Code / VS Code / Windsurf（批量一键）

```bash
npx mcp-add \
  --name mem0-mcp \
  --type http \
  --url "https://mcp.mem0.ai/mcp" \
  --clients "claude code,cursor,windsurf,vscode,opencode"
```

Claude Desktop 无 `mcp-add` 支持：Settings → Connectors → Add custom connector → 名称 `mem0-mcp`、URL `https://mcp.mem0.ai/mcp`，保存重启。

### 4.6 自托管（本仓库 deploy/mem0）

```bash
cd d:/projects/ai-dev-sop/deploy/mem0
docker compose up -d          # mem0-api(8888/8080) + mem0-dashboard(3001)
```

| 端点 | 地址 | 用途 |
|------|------|------|
| REST API | `http://localhost:8888`（Docs: `/docs`） | SDK / 脚本批量读写 |
| MCP | `http://127.0.0.1:8080/mcp`（Streamable HTTP） | 各 AI 工具接入 |
| Dashboard | `http://localhost:3001` | 管理 / 验证记忆 |

依赖外部网络 `mwb-infra-network` 中的 `mwb-postgres-pgvector`（pgvector）与 `mwb-neo4j`（图记忆）。所有客户端配置把 URL 指向 `http://127.0.0.1:8080/mcp` 即可。

---

## 5. MCP 工具清单与关键参数

### 5.1 工具列表（官方云 11 个）

| 工具 | 功能 | 关键参数 |
|------|------|----------|
| `add_memory` | 存储文本/对话记忆 | `text`、`api_key`、`user_id`、`infer`、`metadata`、`git_remote`/`project_id` |
| `search_memories` | 语义检索 | `query`、`top_k`（默认 5）、`user_id`、过滤器 |
| `get_memories` | 列表查询 | `limit`（默认 100）、`project_id`（拉共享池） |
| `get_memory` | 单条读取 | `memory_id` |
| `update_memory` | 覆写文本/元数据 | `memory_id`、`text` |
| `delete_memory` | 删除单条 | `memory_id` |
| `delete_all_memories` | ⚠️ 范围清空 | `user_id` 等范围参数 |
| `delete_entities` | 删除实体及其记忆 | user/agent/app/run 标识 |
| `list_entities` | 枚举 user/agent/app/run | - |
| `list_events` | 操作事件流水 | 过滤 + 分页 |
| `get_event_status` | 异步任务状态 | `event_id` |

### 5.2 `add_memory` 关键参数详解（以 mem0-local 为例）

| 参数 | 默认 | 说明 |
|------|------|------|
| `text` | 必填 | 原始文本（支持整段对话记录） |
| `api_key` | 必填 | 用户标识 + 鉴权；服务端据此路由到 user/project |
| `user_id` | - | 仅 admin key 生效，显式指定归属用户 |
| `infer` | `False` | **`False` = 原文留痕存档（跳过 LLM 抽取，保留中文原文）；`True` = 让 Mem0 抽取事实要点**。会话归档用默认，提炼偏好用 `True` |
| `metadata` | - | 自定义键值对（如 `{"source":"codebuddy","session":"2026-09-18"}`），检索时可过滤 |
| `git_remote` | - | 传 `git remote -v` 的 URL，服务端解析为 `project_id`，写入项目共享池 |
| `project_id` | - | 显式指定项目（与 `git_remote` 二选一） |

---

## 6. 记忆存储、检索与更新配置选项

### 6.1 存储策略

| 选项 | 配置点 | 建议 |
|------|--------|------|
| 原文留痕 vs 事实抽取 | `infer=False/True` | 会话日志/审计 → `False`；用户偏好/项目知识 → `True` |
| 自定义提取指令 | Mem0 Cloud Dashboard → Settings → Memory Extraction | "永远不存 X" 类隐私规则在此配置 |
| 分类体系 | Dashboard → Settings → Categories | 默认 15 类；自定义分类便于过滤检索 |
| 保留策略 | Dashboard → Settings → Retention | Memory Decay 开关 + 过期日期 |
| 自动治理 | Dream（Supersede/Merge 常开；Synthesis 需 Pro） | 存量记忆去重/保鲜 |

### 6.2 检索策略

| 选项 | 配置点 | 说明 |
|------|--------|------|
| `top_k` | `search_memories` 参数 | 每次返回条数，默认 5 |
| 作用域过滤 | `user_id` / `project_id` / `metadata` | 搜索 API 至少需要一个标识（user_id/agent_id/run_id） |
| Graph 记忆 | `MEM0_ENABLE_GRAPH_DEFAULT=true`（stdio env）或服务端 `GRAPH_ENABLED=true` | 实体关系检索，需图存储（Neo4j） |
| 检索阈值 | 自托管 `GRAPH_THRESHOLD=0.7` | 图边建立相似度阈值 |

### 6.3 更新与生命周期

- 单条更新用 `update_memory`（需先 `search`/`get_memories` 拿到 `memory_id`）
- Agent 侧约定"先搜后写"：写入前先 `search_memories` 判断是否已有同类记忆，避免重复
- 批量迁移用 REST API `client.get_all(user_id=...)` 分页拉取（不要用 MCP 逐条翻页）

---

## 7. 常见参数速查

| 参数 | 位置 | 取值 | 说明 |
|------|------|------|------|
| 作用域 `user_id` | 工具参数 | 任意字符串 | 记忆归属；跨工具共享时各工具必须统一（§8.1） |
| `MEM0_DEFAULT_USER_ID` | stdio `env` | 如 `joezxh` | 未显式传 user_id 时注入的默认值 |
| `MEM0_API_KEY` | 环境变量 | `m0-...` | 所有 stdio server 的必备凭证 |
| `MEM0_ENABLE_GRAPH_DEFAULT` | stdio `env` | true/false | 默认是否启用图记忆 |
| `MEM0_MCP_AGENT_MODEL` | stdio `env` | 如 `openai:gpt-4o-mini` | server 内示例 agent 的 LLM |
| 超时 | 客户端侧 | - | MCP 初始化超时（Qoder 报 `context deadline exceeded`）；服务慢时优先检查网络/镜像源 |
| 日志级别 | 服务端 | - | 自托管 FastAPI 通过容器日志查看：`docker logs -f mem0-api`；`MEM0_TELEMETRY=false` 关闭遥测 |
| `AUTH_DISABLED` | 自托管 `.env` | true/false | 本地调试可关认证；对外暴露必须 `false` |
| `JWT_SECRET` | 自托管 `.env` | 随机串 | Dashboard/API 会话签名 |

---

## 8. 跨工具共享记忆

### 8.1 方案一：统一 user_id（最简单）

所有工具的 MCP 配置指向**同一个** Mem0 端点 + **同一个** `user_id`（或同一把 API Key），即天然共享：

```
CodeBuddy ──┐
Qoder ──────┼── mcp.mem0.ai 或 localhost:8080/mcp ── 同一 Project / user_id="joezxh"
Cursor ─────┤
Codex ──────┘
```

要点：密钥按 Project 隔离——**共享 = 使用同一 Project 的 Key + 同一 user_id**。

### 8.2 方案二：项目共享池（git_remote 聚合）

mem0-local server 支持把 `git_remote` 解析为 `project_id`：团队内各成员以各自 `api_key` 写入，带相同 `git_remote` 的记忆进入**项目共享池**；`search_memories` / `get_memories` 带 `git_remote` 时跨用户检索。适合"项目级团队记忆"。

### 8.3 方案三：OpenMemory MCP（本地共享层）

多个客户端连接同一个本地 OpenMemory 进程，记忆统一落本地库并带管理 UI，适合"数据不出机器 + 多工具互通"。各客户端按 stdio 配置即可，无需改用云端。

### 8.4 差异与兼容性对照

| 维度 | CodeBuddy | Qoder | Cursor | Codex |
|------|-----------|-------|--------|-------|
| 配置文件 | Settings→MCP JSON | 个人设置→MCP JSON | `~/.cursor/mcp.json` | `~/.codex/config.toml`（TOML！） |
| HTTP 原生支持 | ✅ | ✅（SSE 表单） | ✅ | ✅ 但只能手写 TOML |
| Header 传 Key | ✅ `headers` | ⚠️ 表单不支持自定义 Header | ✅ OAuth 或 Key | ✅ `bearer_token_env_var` |
| 配置生效范围 | 用户级 | 用户级全局 | 用户级 | 用户级 |
| 服务命名建议 | `mem0` | `mem0` | `mem0-mcp` | `mem0` |

---

## 9. 自动提交与拉取：Agent 规则文件编写说明（CODEBUDDY.md 模式）

MCP 工具本身是**被动**的——配置好 server 只意味着 Agent"能"调用记忆工具，不等于"会"自动调用。要让记忆在每次会话中**自动拉取（pull on session start）+ 自动提交（push on durable facts / 每轮留痕）**，需要一层 Agent 规则文件。本工程的范本是根目录 [`CODEBUDDY.md`](../../CODEBUDDY.md)（"Project Memory via mem0 MCP"），本章说明如何为各工具编写同类规则文档。

### 9.1 各工具的规则文件位置对照

| 工具 | 规则文件 | 作用范围 | 说明 |
|------|----------|----------|------|
| **CodeBuddy IDE** | 仓库根 `CODEBUDDY.md` | 项目级，每会话自动注入 | 本工程现用范本 |
| **Codex CLI** | 仓库根 `AGENTS.md` | 项目级 | Codex 每会话自动读取 |
| **Claude Code** | 仓库根 `CLAUDE.md`（可 `@import` 复用 `CODEBUDDY.md`） | 项目级 | 内容可与其他工具共用一份 |
| **Cursor** | `.cursor/rules/*.mdc`（新版）或根 `AGENTS.md` | 项目级 | 旧版 `.cursorrules` 已过渡 |
| **Qoder IDE** | `.qoder/rules/*.md` | 项目级，仅当前项目生效 | 官方"规则"机制 |
| **Qoder CLI** | 仓库根 `AGENTS.md` | 项目级 | 与 Codex 同标准 |
| **VS Code (Copilot)** | `.github/copilot-instructions.md` | 项目级 | |
| **Windsurf** | `.windsurfrules` 或 `AGENTS.md` | 项目级 | |

> **共用策略**：把核心规则写成一份 `CODEBUDDY.md`，其他工具用一行引用/软链接复用（如 `CLAUDE.md` 写 `@CODEBUDDY.md`），避免多份漂移。

### 9.2 规则文件必写的六个部分（以 CODEBUDDY.md 为范本）

**① 声明连接与配置来源（不许硬编码）**

开头明确：使用哪个 MCP server（如 `mem0-local`，配置于 `~/.codebuddy/mcp.json`）、端点由配置文件驱动而非写死；凭证从本地 git-ignored 文件读取（本工程为 `.codebuddy/mem0.config.json`，含 `api_key` / `git_remote` / `project_id` / admin `user_id` / `roster`），并声明"永远从配置读，不写进代码"。

**② 会话启动自动拉取（Loading）**

固定三步，且写明 **"Do this proactively; do not wait for the user to ask"**：

```
1. get_memories(api_key=..., git_remote="<配置>")   # 一次调用拉全项目跨用户共享池
   （等价：get_memories(api_key=..., project_id="ai-dev-sop")）
2. 任务具体时追加：search_memories(api_key=..., query=<关键词>, project_id=..., top_k=5)
3. 向用户汇报 2–3 行召回摘要（"已加载本工程记忆 12 条：…"）
```

**③ 持久事实自动提交（Saving durable facts）**

定义**什么该存**（durable：人物/项目/团队的事实、决策、偏好、约束）与**什么不该存**（ephemeral 任务状态）。给出标准调用模板：

```python
add_memory(
    text="<事实的自然语言陈述>",
    api_key="<from config>",
    git_remote="<from config>",          # 必传：服务端解析为 project_id 并写入 metadata
    user_id="<记忆主体；默认 admin>",
    project_id="ai-dev-sop",
    metadata={"type":"fact|decision|preference|note",
              "people":["<name>"], "by":"<admin>", "created_at":"<ISO8601>"}
)
```

配套三条纪律：`git_remote` 必传（绑定项目池）；相关事实合并成一条（避免碎句刷屏）；mem0 按内容哈希去重，重复提交/补录是安全的。

**④ 每轮会话留痕（Conversation transcript，进阶可选）**

要求每轮对话原文强制入库、按 `session_id` 在 dashboard 分组。编写要点：

- **session_id 规则**：格式 `cb-<YYYYMMDD>-<6位hex>`；会话开始生成并持久化到 `.codebuddy/.session_id`（git-ignored），同会话复用，新会话才重新生成
- **text 固定格式**：`Q: <用户原话>\n\nA: <Agent 完整回复>`（Q/A 间空一行，dashboard 按 `Q:`/`A:` 切分两栏；A 部分用 Markdown 组织、含推理过程，不得只写摘要）
- **metadata 规范**：`type=conversation`、`session_id`、`agent`（固定工具名）、`role=turn`、`turn_seq`（会话内递增）、`created_at`

```python
add_memory(
    text="Q: <用户本轮原始提问>\n\nA: <完整回复，Markdown>",
    api_key="<from config>", git_remote="<from config>",
    user_id="admin@mem0.dev", project_id="ai-dev-sop",
    metadata={"type":"conversation", "session_id":"cb-20260917-a1b2c3",
              "agent":"CodeBuddy", "role":"turn", "people":["joezxh"],
              "turn_seq":"<递增序号>", "created_at":"<ISO8601>"}
)
```

- 与 ③ 分流：事实用 `type=fact/decision/preference`，会话流用 `type=conversation`，互不替代；每轮固定触发（确认/寒暄也入库）
- dashboard 按 `metadata.session_id` 过滤分组、按 `created_at`/`turn_seq` 排序

**⑤ 权限与同步策略（Permissions & sync）**

写明同步模型：**会话启动时拉取 + 遇持久事实时提交**；记忆为追加式 + 哈希去重，重拉重存不产生重复，无需冲突解决；`user_id=X` 的个人记忆仅 X 可见（admin key 可越权读项目池）；**仅在用户明确要求时删除**（`delete_memory`）。个人隔离与项目共享池的读写矩阵可引用 §8。

**⑥ 配置面与失败降级（Configuration surface & fallback）**

用表格列出三处配置及其声明内容（MCP 连接 / 本地凭证文件 / 长期记忆镜像），并写明降级行为：

> 若 `mem0.config.json` 缺失或 MCP server 不可达，**告知用户并停止——不得编造记忆**。

### 9.3 编写检查清单

- [ ] 开头声明 MCP server 名与配置文件来源，无任何硬编码端点/密钥
- [ ] 拉取规则标注 "proactive"（不等用户开口）
- [ ] 存储规则区分 durable 事实与 ephemeral 状态，metadata schema 固定
- [ ] 每轮留痕有 session_id 生成/持久化规则与 Q:/A: 格式约束
- [ ] 明确 git_remote 必传与哈希去重语义
- [ ] 有失败降级条款（配置缺失/服务不可达 → 停止并告知）
- [ ] 其他工具通过 AGENTS.md/CLAUDE.md 引用同一份规则，避免重复维护

---

## 10. 配置验证与故障排查

### 10.1 标准验证流程

1. MCP 面板中 server 状态为**绿色/已连接**
2. 让 Agent 执行写入："记住我偏好 TypeScript，不要用 tabs"
3. 让 Agent 执行检索："我的代码风格偏好是什么？" —— 能正确读回即通过
4. 云端/本地 Dashboard 的 Memories 列表确认记录已落库

### 10.2 常见故障

| 现象 | 原因 | 解决 |
|------|------|------|
| `401 Authentication required` | 未登录 / Key 无效 | OAuth 重新授权；或重新生成 Key 更新配置 |
| `Invalid API key` | Key 复制不完整/已重建 | 从 Dashboard 重新复制 |
| 工具列表为空 | 客户端未重启 | 重启客户端 / 点击重启服务 |
| `exec: "npx": not found` | Node 缺失 | 安装 Node ≥ 18；Windows 用 nvm-windows |
| `exec: "uvx": not found` | uv 缺失 | 安装 uv（需 Python ≥ 3.8） |
| `context deadline exceeded` | 网络/镜像源/安全拦截 | 复制启动命令到终端跑看详细报错；配镜像源：`npm config set registry https://registry.npmmirror.com`，`UV_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/`；公司安全软件加白 |
| 模型不调用 MCP 工具 | Qoder 处于问答模式 / 服务未连接 / 服务重名歧义 | 切智能体模式；重启服务；服务命名去重 |
| MCP 面板空白（Android Studio） | JCEF 不支持 | 启用 `ide.browser.jcef.enabled` 并换新 Runtime |
| 自托管连接拒绝 | 容器未启动 | `docker compose ps`；`docker compose up -d`；确认 `mwb-postgres-pgvector`/`mwb-neo4j` healthy |
| 自托管 404 | URL 少 `/mcp` 路径 | 严格使用 `http://127.0.0.1:8080/mcp` |
| 记忆检索不到 | user_id 不一致 / Key 跨 Project | 检查各工具 `user_id` 是否统一；确认 Key 属于同一 Project |

---

## 11. 安全与权限注意事项

1. **密钥管理**
   - Key 一律走环境变量或客户端密钥存储，禁止硬编码进仓库文件（`.gitignore` 覆盖 `.env`）
   - 云端 Key 创建后即哈希化，列表不可回看；泄露立即在 Dashboard 重建
   - 不同工具可分配**不同 Key**，便于按工具撤销与审计
2. **作用域最小化**：为每个工具/项目分配对应 Project 的 Key，避免一把大 Key 全局使用；共享池按 `git_remote` 聚合，团队记忆与个人记忆分开
3. **危险操作防护**：`delete_all_memories` / `delete_entities` 不可逆；生产环境通过 Role 权限收敛（Can Read < Can Edit < Admin，见 Mem0 Cloud Settings → Members）
4. **数据驻留**：合规敏感场景用自托管（形态 B）或 OpenMemory（形态 C），数据不离开本地；Mem0 Cloud 数据按其平台条款存储
5. **传输安全**：云端 MCP 全程 HTTPS；自托管不要将 8080 端口暴露到公网，`AUTH_DISABLED` 保持 `false`
6. **`infer` 与隐私**：`infer=True` 会把内容送 LLM 抽取（云端形态即出域）；本地脱敏存档用 `infer=False`，并用 Extraction 自定义指令配置"永不存储 X"
7. **审计**：云端 Enterprise 支持审计日志；自托管可通过 `list_events` 追溯记忆操作流水

---

## 附录 A：各工具最小可用配置速拷

**CodeBuddy / Cursor / Qoder（HTTP 云端）**
```json
{ "mcpServers": { "mem0": { "type": "http", "url": "https://mcp.mem0.ai/mcp" } } }
```

**Codex（`~/.codex/config.toml`）**
```toml
[mcp_servers.mem0]
url = "https://mcp.mem0.ai/mcp"
bearer_token_env_var = "MEM0_API_KEY"
```

**stdio 自托管（任意支持 stdio 的客户端）**
```json
{
  "mcpServers": {
    "mem0": {
      "command": "uvx",
      "args": ["mem0-mcp-server"],
      "env": { "MEM0_API_KEY": "m0-...", "MEM0_DEFAULT_USER_ID": "joezxh" }
    }
  }
}
```

## 附录 B：参考链接

- 官方 MCP 文档：https://docs.mem0.ai/platform/mem0-mcp
- Cursor 集成：https://docs.mem0.ai/integrations/cursor
- mem0-mcp（已归档）：https://github.com/mem0ai/mem0-mcp
- OpenMemory：https://github.com/mem0ai/openmemory
- Qoder MCP 指南：https://docs.qoder.cn/user-guide/guide-for-using-mcp
- CodeBuddy MCP 指南：https://www.codebuddy.ai/docs/zh/ide/User-guide/MCP
- 本仓库自托管部署：`deploy/mem0/docker-compose.yaml`

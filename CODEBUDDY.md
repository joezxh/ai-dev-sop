# 通过 mem0 MCP 的项目记忆（mem0 MCP 服务 `mem0`）

本项目使用自托管的 **mem0** 服务作为长期记忆，通过 `~/.codebuddy/mcp.json` 中配置的
mem0 MCP 服务访问——**服务名遵循约定常量 `mem0`（由 `scripts/mem0-setup` 写入；环境差异只用 URL 区分，不改名）**。连接配置（`url` → `http://127.0.0.1:8080/mcp`，
streamable-http）实际端点由该配置文件决定——此处切勿硬编码。连接已经建立，
下列规则让 Agent **在每轮会话中自动**加载与保存记忆——无需脚本，无需手动命令。

> 保密说明：mem0 的 `api_key` 与管理员 `user_id` 存储在长期记忆中
> （knowledge："本地 mem0 MCP 服务配置与凭据"），并镜像保存在本地、
> 已被 git 忽略的 `.mem0/mem0.config.json`。请始终从那里读取，切勿在代码中硬编码。

## 1. 加载记忆（自动，会话开始时）

在本仓库的每轮会话首次开始时，通过 `~/.codebuddy/mcp.json` 中实际配置的 mem0 MCP
服务（约定名 `mem0`）加载项目记忆池。
从 `.mem0/mem0.config.json` 读取 `api_key`、`git_remote`、`project_id`——切勿硬编码。（兼容：若 `.mem0/mem0.config.json` 不存在，回退读取 `.codebuddy/mem0.config.json`。）

1. 用一次 MCP 调用拉取**整个项目共享池**（包含所有用户）：
   `get_memories(api_key=..., git_remote="<git_remote>")`——服务端会将 git remote
   解析为 `project_id` 并返回跨用户池。等价写法：
   `get_memories(api_key=..., project_id="ai-dev-sop")`。
2. 如果任务较为具体，再额外调用
   `search_memories(api_key=..., query=<关键词>, project_id="ai-dev-sop", top_k=5)`。
3. 给用户 2–3 行的召回摘要（例如："已加载本工程记忆
   12 条：涉及人员 A/B，关键决策 X"）。

请主动执行，不要等待用户询问。

## 2. 保存记忆（自动——通过 MCP，无需脚本，无需手动命令）

当用户透露关于某人、项目或团队的**持久**事实、决策、偏好或约束时，通过
`~/.codebuddy/mcp.json` 中实际配置的 mem0 MCP 服务保存。不要保存临时性的任务状态。每次都从
`.mem0/mem0.config.json` 读取 `api_key` / `git_remote` / `user_id`。

```python
add_memory(
    text="<对事实的自然语言描述>",
    api_key="<来自配置>",
    git_remote="<来自配置>",          # 服务端解析 → project_id，写入 metadata
    user_id="<主体；默认 admin>",  # 仅 admin 密钥有效；记忆是关于谁的
    project_id="ai-dev-sop",
    metadata={"type":"fact|decision|preference|note","people":["<姓名>"],"by":"<admin>","created_at":"<ISO8601>"}
)
```

规则：
- **始终传入 `git_remote`**——它将记忆绑定到本仓库的共享池（服务端会将其转换为
  `project_id` 并记录在 metadata 中）。
- `user_id` = 记忆的主体。当事实是关于项目/团队整体时，默认使用 admin 用户。
  （仅在 admin 密钥下才有意义。）
- 相关的多个事实应合理合并成一条记忆，避免逐句刷屏式存储。
- mem0 按内容哈希去重，因此重新保存/回填是安全的。

### 2.1 会话留痕（每轮对话原文强制入库）

本工程要求**每一轮对话的原始文本都强制提交到 mem0**，使服务端（dashboard）能按
`session_id` 把同一工程的不同会话分别显示。纯 MCP 方案无 Hook，故由 Agent 在每轮
回复**完成时固定触发** `add_memory`，不得跳过。

**权威存档目标（本节最高准则）**：mem0 服务端是本会话的**权威完整存档**——
在其他任何端（另一台机器的 CodeBuddy、dashboard、REST/MCP 客户端）拉取本会话记忆
并按 Markdown 渲染后，所看到的内容必须与**本端 CodeBuddy 界面当前画面完全一致**。
因此提交的标准不是"摘要够用"，而是"跨端无损还原"：画面上出现过的每一个元素
（用户消息、思考过程、每次工具调用及其结果、最终回复）都必须在存档中有对应记录。
**全部内容以 Markdown 格式为准**——`text` 是唯一权威载体，服务端不保存任何
非 Markdown 的等价副本；Markdown 结构（标题/列表/代码块/表格）必须完整保留，
使任何支持 Markdown 渲染的端都能还原同样的画面。

会话 ID（session_id）：
- 每个 Agent 会话使用唯一 `session_id`，格式 `cb-<YYYYMMDD>-<6位hex>`（如
  `cb-20260917-a1b2c3`）。
- 会话开始时生成，持久化到本地 `.mem0/.session_id-cb`（已 git-ignore）；同会话
  所有轮次复用同一 id；新会话重新生成。
- 若 `.mem0/.session_id-cb` 已存在则直接复用，避免同会话产生多个 id。
  **但存在以下任一情形时必须重新生成**：
  1. 文件中 id 的日期部分与**当天日期不一致**（跨天继续对话，旧 id 属于昨天的
     会话分组，直接复用会把今天所有轮次混入昨天的 dashboard 会话流）；
  2. 用户明确表示开启了新会话/新话题且旧 id 来自历史会话。
  重新生成后用新 id 覆写 `.mem0/.session_id-cb`，本会话内恒定。

提交内容（原文，非摘要，且为 Markdown）：
- **提问与回答都要入库**：每条 memory 的 `text` 写入该轮原始对话，固定格式为
  `Q: <用户原话>\n\nA: <本轮完整执行转录>`（Q 与 A 之间空一行，便于 dashboard
  详情面板按 `Q:`/`A:` 切分两栏显示）。**Q 部分必须逐字保留用户本轮的完整消息**
  （含用户在消息中引用的文件路径、附件说明、原文引用块）。
- **A 部分 = 界面内容的按序完整转录**：严格按**界面实际输出顺序**组织（Deep Thinking / 过程叙述 / Tool 调用在回合中是交替出现的，最终回复出现在最后）。用一条时间线呈现，每段前以原文叙述行、`> Think:` 或 `> Tool:` 标明类型：
  1. **叙述行 + Tool 调用 + Think 片段（交替，按发生顺序）**：界面上每次工具调用前后的过渡叙述行逐字保留；紧随其后记录该次调用 `> Tool: 工具名(关键参数) → 执行结果`（保留**完整输出**；超长输出用 Markdown 代码块原样粘贴，确实极端超长时可截取首尾并注明 `... (N 行省略)`，但关键数据、错误信息、结论行不得省略；文件编辑类需注明目标文件与 old→new 要点；失败/被取消的调用也记录，含错误信息）。两次调用之间的思考片段以 `> Think: …` 插在**对应位置**，不得集中到文末重写。
  2. **最终回复原文（界面可见，位于时间线末尾）**：本轮呈现在对话中的最终回答文本，逐字保留完整 Markdown——标题层级（`##`/`###`）、表格（含对齐行）、代码块（含语言标注）、加粗/行内代码/链接，一个字符都不许改。
  3. **完成结果**：任务最终状态（成功/部分完成/失败）、交付物清单（文件路径）、验证结论（测试/检查输出）。
- **顺序红线**：时间线必须与界面实际顺序一致——叙述了一条就要有对应的 Tool 记录；不得把 Thinking 压成文末大段，不得把多次调用合并成"×N"概括。
- **增量式记录（硬性）**：回合进行中**每完成一次工具调用即当场追加**该条记录；回合末尾只做拼接与自检。**禁止在回合末尾凭记忆一次性重建整条记录**（事后重建必然压缩并丢失叙述行）。
  推荐的 A 部分结构（dashboard 按 Markdown 渲染）：

  ```markdown
  A: <叙述行 1，逐字>
  > Think: <该步骤前的真实思考片段>
  > Tool: 工具名(关键参数) → 执行结果（完整输出）

  <叙述行 2，逐字>
  > Tool: …（含失败/被取消的调用，注明错误）

  ### 最终回复原文
  <逐字完整 Markdown>

  ### 完成结果
  <状态 + 交付物清单（文件路径） + 验证结论>
  ```
- **一致性红线**：以 CodeBuddy 界面本轮实际显示的内容为准——把界面会话区全选
  复制为 Markdown，所得文本必须能在 A 部分中**原样找到**（叙述行 + 最终回复，
  顺序、换行、格式完全一致）；禁止只提交精简摘要、或只提交最终回复而丢失
  过程叙述与执行过程。
- **提交前自检（跨端还原测试）**：提交前逐项核对——
  ① 界面复制的 Markdown 文本（叙述行 + 最终回复）与 A 部分对应内容逐字节一致？
  ② 每条叙述行背后对应的 Tool 调用都有记录（名称/参数/完整输出）？
  ③ 思考片段（`> Think:`）保留在真实发生位置、未集中到文末重写？
  ④ 完成结果含状态/交付物/验证？
  ⑤ **时间线顺序与界面实际输出顺序一致**（叙述/Tool/Think 交替，最终回复在末尾）？
  ⑥ **`created_at` 是提交时刻的真实当前时间**（非 `00:00:00Z` 之类占位）？
  任何一项不满足必须补齐后再提交。
- `agent` 固定为 `"CodeBuddy"`。
- **`created_at` 必须取提交时刻的真实当前时间**（禁止 `00:00:00Z` 之类占位值）：
  Windows 用 `(Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")` 或带偏移的本地 ISO
  `(Get-Date).ToString("yyyy-MM-ddTHH:mm:sszzz")`；macOS/Linux 用 `date -u +%Y-%m-%dT%H:%M:%SZ`。
  提交前先执行取时间命令，把真实值填入 metadata，不得凭印象填写。

```python
add_memory(
    text="Q: <用户本轮原始提问>\n\nA: <叙述行/Think/Tool 按实际发生顺序的时间线>\n\n### 最终回复原文（逐字完整 Markdown）\n\n### 完成结果\n<状态/交付物/验证>",
    api_key="<来自配置>",
    git_remote="<来自配置>",
    user_id="admin@mem0.dev",
    project_id="ai-dev-sop",
    metadata={
        "type":"conversation",
        "session_id":"cb-20260917-a1b2c3",   # 同会话恒定
        "agent":"CodeBuddy",
        "role":"turn",                         # 一轮含提问+回答
        "people":["joezxh"],
        "turn_seq":"<同会话内递增序号，从1>",
        "created_at":"<ISO8601>"
    }
)
```

约束：
- **每轮固定触发**，不挑轮次（确认/寒暄也入库，保证会话流完整）。
- **内容一致性（跨端）**：mem0 服务端内容必须与 CodeBuddy 本轮界面显示完全一致；
  把界面会话区复制为 Markdown，所得文本（过程叙述行 + 最终回复）必须能在 A 部分
  原样找到，再叠加 Deep Thinking 全文与每次工具调用的参数与完整结果；在其他端
  拉取并渲染后应能还原同样画面。有工具调用的轮次不得只提交最终回复。
- **Markdown 为准**：`text` 是唯一权威载体，全部内容以 Markdown 组织；标题层级、
  列表、代码块、表格必须完整保留，不得退化为无格式纯文本。
- 与 §2 的"持久事实"分开：事实用 `type=fact/decision/preference`，会话流用
  `type=conversation`，互不替代。
- dashboard 分组：mem0 dashboard 按 `metadata.session_id` 过滤/分组即可分别显示同一
  工程的各会话；按 `created_at` 或 `turn_seq` 排序即得会话内顺序。
- 记忆量会显著增长，属预期（用户明确要求原文完整入库）。

## 3. 跨用户识别与抽取

- **按用户隔离**：mem0 以 `user_id` 划定范围。每个人的记忆仅对该 id 私有，
  除非通过项目共享池共享。
- **项目共享池**：所有标记了 `project_id="ai-dev-sop"` 的记忆，任何项目成员都可通过
  `get_memories(project_id=...)` 或 `get_memories(git_remote=...)` 读取。
  这正是打开工程时"抽取所有相关人员记忆"的方式——无需枚举 user_id。
- **管理员密钥**：配置的密钥为管理员密钥，因此 `get_memories(project_id=...)`
  / `search_memories(project_id=...)` 已返回**完整**的跨用户池——
  无需逐用户枚举。个人（非池）记忆在以其自身 `user_id` 保存时，仍仅在该用户下可见。
- **花名册（Roster）**：相关人员的集合位于 `.mem0/mem0.config.json`
  （`roster`）。相关人首次出现时即添加进去，以便后续加载时对其划定范围。

## 4. 权限与同步

- 写入隔离：以 `user_id=X` 保存的记忆归 X 所有，仅出现在 X 的个列表中，
  除非同时也在项目共享池中。
- 读取权限：项目池 = 跨用户可读；个列表 = 仅该用户可见（admin 密钥可覆盖）。
- 同步策略：**会话开始时拉取**（步骤 1），**持久事实出现时推送**（步骤 2）。
  记忆是追加式且按哈希去重，因此重新加载与重新保存不会产生重复。无需冲突解决。
- 仅当用户明确要求时才删除（MCP `delete_memory`）。

## 5. 配置面

| 位置 | 声明内容 |
|---|---|
| `~/.codebuddy/mcp.json` → mem0 MCP 条目（约定名 `mem0`） | MCP 连接（`url`、transport） |
| `.mem0/mem0.config.json` | `git_remote`、`project_id`、admin `user_id`、`api_key`、`roster`（人员 + user_id）、metadata 默认值 |
| 长期记忆 | 为没有配置文件的会话镜像保存的 `api_key` / `user_id` |

如果 `mem0.config.json` 缺失或 MCP 服务不可达，请告知用户并停止——不要臆造记忆。

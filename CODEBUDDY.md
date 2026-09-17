# Project Memory via mem0 MCP (mem0-local)

This project uses a self-hosted **mem0** server as long-term memory, accessed through
the `mem0-local` MCP server (configured in `~/.codebuddy/mcp.json` →
`http://127.0.0.1:8080/mcp`, streamable-http). The actual endpoint is driven by that
config file — never hard-code it here. The connection is already established; the rules
below make the agent load and save memories **automatically**
on every session — no scripts, no manual commands.

> Secret note: the mem0 `api_key` and admin `user_id` are stored in long-term memory
> (knowledge: "本地 mem0 MCP 服务配置与凭据") and mirrored in the local, git-ignored
> `.codebuddy/mem0.config.json`. Always read them from there; never hard-code in code.

## 1. Loading memories (automatic, on session start)

On the first turn of every session in this repo, load the project memory pool via the
`mem0-local` MCP server. Read `api_key`, `git_remote`, `project_id` from
`.codebuddy/mem0.config.json` — never hard-code them.

1. Pull the **whole project shared pool** (all users) with one MCP call:
   `get_memories(api_key=..., git_remote="<git_remote>")` — the server resolves the
   git remote to a `project_id` and returns the cross-user pool. Equivalent:
   `get_memories(api_key=..., project_id="ai-dev-sop")`.
2. If the task is specific, also run
   `search_memories(api_key=..., query=<keywords>, project_id="ai-dev-sop", top_k=5)`.
3. Give the user a 2–3 line summary of what was recalled (e.g. "已加载本工程记忆
   12 条：涉及人员 A/B，关键决策 X").

Do this proactively; do not wait for the user to ask.

## 2. Saving memories (automatic — via MCP, no scripts, no manual commands)

When the user reveals a **durable** fact, decision, preference, or constraint about a
person, the project, or the team, save it through the `mem0-local` MCP server. Do NOT
save ephemeral task state. Read `api_key` / `git_remote` / `user_id` from
`.codebuddy/mem0.config.json` each time.

```python
add_memory(
    text="<natural-language statement of the fact>",
    api_key="<from config>",
    git_remote="<from config>",          # server resolves → project_id, written to metadata
    user_id="<subject; default admin>",  # admin key only; who the memory is ABOUT
    project_id="ai-dev-sop",
    metadata={"type":"fact|decision|preference|note","people":["<name>"],"by":"<admin>","created_at":"<ISO8601>"}
)
```

Rules:
- **Always pass `git_remote`** — it ties the memory to this repo's shared pool (the
  server converts it to `project_id` and records it in metadata).
- `user_id` = the subject of the memory. Default to the admin user when the fact is
  about the project/team generally. (Only meaningful under the admin key.)
- Batch related facts into one memory where sensible; avoid 1-sentence spam.
- mem0 dedupes by content hash, so re-saving / backfilling is safe.

### 2.1 会话留痕（每轮对话原文强制入库）

本工程要求**每一轮对话的原始文本都强制提交到 mem0**，使服务端（dashboard）能按
`session_id` 把同一工程的不同会话分别显示。纯 MCP 方案无 Hook，故由 Agent 在每轮
回复**完成时固定触发** `add_memory`，不得跳过。

会话 ID（session_id）：
- 每个 Agent 会话使用唯一 `session_id`，格式 `cb-<YYYYMMDD>-<6位hex>`（如
  `cb-20260917-a1b2c3`）。
- 会话开始时生成，持久化到本地 `.codebuddy/.session_id`（已 git-ignore）；同会话
  所有轮次复用同一 id；新会话重新生成。
- 若 `.codebuddy/.session_id` 已存在则直接复用，避免同会话产生多个 id。

提交内容（原文，非摘要，且为 Markdown）：
- **提问与回答都要入库**：每条 memory 的 `text` 写入该轮原始对话，固定格式为
  `Q: <用户原话>\n\nA: <Agent 本轮完整回复>`（Q 与 A 之间空一行，便于 dashboard
  详情面板按 `Q:`/`A:` 切分两栏显示）。
- **A（回答）部分必须包含完整信息**：覆盖整个执行过程，包括内部 `thinking` 推理内容
  （需求分析、决策依据、关键步骤、踩坑与结论等）一并提交，**不得只写精简摘要**；文本
  以 **Markdown 格式**组织（标题、列表、代码块、加粗等），dashboard 详情面板会以
  Markdown 渲染 A 部分。
- `agent` 固定为 `"CodeBuddy"`，`created_at` 为该轮完成时的 ISO8601 时间戳。

```python
add_memory(
    text="Q: <用户本轮原始提问>\n\nA: <Agent 本轮完整回复，含 thinking 推理，Markdown 格式>",
    api_key="<from config>",
    git_remote="<from config>",
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
- 与 §2 的"持久事实"分开：事实用 `type=fact/decision/preference`，会话流用
  `type=conversation`，互不替代。
- dashboard 分组：mem0 dashboard 按 `metadata.session_id` 过滤/分组即可分别显示同一
  工程的各会话；按 `created_at` 或 `turn_seq` 排序即得会话内顺序。
- 记忆量会显著增长，属预期（用户明确要求原文入库）。

## 3. Cross-user identification & extraction

- **Per-user isolation**: mem0 scopes by `user_id`. Each person's memories are
  private to that id unless shared via a project pool.
- **Project shared pool**: every memory tagged with `project_id="ai-dev-sop"` is
  readable by anyone in the project through `get_memories(project_id=...)` or
  `get_memories(git_remote=...)`. This is how "all relevant people's memories" are
  extracted on open — no need to enumerate user_ids.
- **Admin key**: the configured key is an admin key, so `get_memories(project_id=...)`
  / `search_memories(project_id=...)` already return the **entire** cross-user pool —
  no per-user enumeration needed. Personal (non-pool) memories are still only visible
  under their own `user_id` when saved that way.
- **Roster**: the set of relevant people lives in `.codebuddy/mem0.config.json`
  (`roster`). Add a person there the first time they appear, so future loads can
  scope to them.

## 4. Permissions & sync

- Write isolation: a memory saved with `user_id=X` is owned by X and only surfaces
  in X's personal list unless also in a project pool.
- Read access: project pool = cross-user readable; personal lists = restricted to
  that user (admin key can override).
- Sync strategy: **pull on session start** (step 1), **push on durable facts**
  (step 2). Memories are append/dedupe-by-hash, so re-loading and re-saving do not
  create duplicates. No conflict resolution needed.
- Delete only when the user explicitly asks (MCP `delete_memory`).

## 5. Configuration surface

| Where | Declares |
|---|---|
| `~/.codebuddy/mcp.json` → `mem0-local` | MCP connection (`url`, transport) |
| `.codebuddy/mem0.config.json` | `git_remote`, `project_id`, admin `user_id`, `api_key`, `roster` (people + user_ids), metadata defaults |
| long-term memory | mirrored `api_key` / `user_id` for sessions without the config file |

If `mem0.config.json` is missing or the MCP server is unreachable, tell the user
and stop — do not invent memories.

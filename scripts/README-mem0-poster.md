# mem0 转录直投管道（零 LLM 逐字留痕）接入说明

> 目标：让会话留痕 **零 LLM 参与、100% 逐字**——数据源为本地会话转录，纯脚本组装后
> 直投 mem0 REST（`infer=false`）。

> **⚠️ 落盘事实（2026-09-19 修订）**：
> - **CodeBuddy CLI** 会话转录落盘 `~/.codebuddy/projects/<dir>/<uuid>.jsonl`
>   （message/reasoning/function_call/function_call_result 全部原文）→ 由 `mem0-transcript-poster.mjs` 零 LLM 直投；
> - **CodeBuddy IDE** 会话转录**实际落盘本地**于
>   `%LOCALAPPDATA%/CodeBuddyExtension/Data/<userId>/CodeBuddyIDE/<userId>/history/<workspaceHash>/<sessionId>/messages/<msgId>.json`
>   （消息 / 思考 / 工具调用 / 工具结果 全部原文，`index.json` 给出消息顺序）→ 由 `mem0-ide-session-poster.mjs`
>   **零 LLM** 直投 mem0（无需任何 LLM 调用）；
> - IDE 工具层另经 `PostToolUse` hook（`mem0-tool-capture-hook.mjs`）原样落盘
>   `<repo>/.mem0/tool-events.jsonl`（零 LLM），与 IDE 转录通道互补。

> **⚡ 一键安装**：`scripts/mem0-setup.ps1` / `mem0-setup.sh` 现已自动完成
> 两个 poster 复制到 `~/.codebuddy/hooks/`、幂等合并 `SessionStart` + `SessionEnd` hook（CLI + IDE 转录），
> 以及 `mem0-tool-capture-hook.mjs` 的 `PostToolUse` 工具捕获 hook（仅 codebuddy）。

## 架构

```
CodeBuddy CLI 会话
  └─ ~/.codebuddy/projects/<munged-cwd>/<session-uuid>.jsonl
       └─ SessionStart hook → node scripts/mem0-transcript-poster.mjs --workspace <repo>
            ├─ 读取该工作区全部 JSONL（断点续传：.mem0/.poster-state.json 按字节偏移）
            ├─ 按 user 消息切轮次 → 组装 Q:/### Thinking N/### Tool N/### 最终回复原文
            └─ POST {rest}/memories (X-API-Key + infer=false + metadata.session_id)

CodeBuddy IDE 会话（零 LLM 逐字留痕，主通道）
  └─ %LOCALAPPDATA%/CodeBuddyExtension/Data/<userId>/CodeBuddyIDE/<userId>/history/<workspaceHash>/<sessionId>/messages/*.json
       └─ SessionStart hook → node scripts/mem0-ide-session-poster.mjs --workspace <repo>
            ├─ 读取 index.json 得消息顺序，逐条解析（reasoning / tool-call / tool-result）
            ├─ 按 user 消息切轮次 → 组装 Q:/### Thinking N/### Tool N/### 最终回复原文
            ├─ 断点续传：.mem0/.ide-poster-state.json 按已提交轮次
            └─ POST {rest}/memories (X-API-Key + infer=false + metadata.session_id)
```

## 1. 脚本（Node ≥ 18，零依赖）

- `scripts/mem0-transcript-poster.mjs`：CLI 会话 JSONL → mem0
- `scripts/mem0-ide-session-poster.mjs`：IDE 本地落盘转录（`CodeBuddyExtension\Data\...\history\...\messages\*.json`）→ mem0，零 LLM

```bash
# CLI 会话（提交到 mem0）
node scripts/mem0-transcript-poster.mjs --workspace D:\projects\ai-dev-sop

# IDE 会话（零 LLM 直投）
node scripts/mem0-ide-session-poster.mjs --workspace D:\projects\ai-dev-sop
#   --session <id>    指定会话目录名（缺省自动 flush 最近产生的会话）
#   --limit N         仅处理前 N 轮（测试用）

# 预演（不落盘、不提交）
node scripts/mem0-ide-session-poster.mjs --workspace . --dry-run

# 端到端原样校验：取一个真实轮次 POST→回读→字节级比对→清理（退出码 0=PASS）
node scripts/mem0-ide-session-poster.mjs --workspace . --verify --rest-url http://localhost:8002

# 可选参数
--rest-url http://localhost:8002      # REST 地址（默认 localhost:8002）
# 端口说明：默认 8002 对应本地 mem0/server 原生 uvicorn（非 Docker）。若用 Docker 部署
#   （REST 在 8888），请显式传 --rest-url http://localhost:8888（setup 脚本对应 -RestUrl / -s）。
```

凭证读 `<workspace>/.mem0/mem0.config.json`（api_key / git_remote / project_id /
admin_user_id）；IDE 通道的 `session_id` 取 IDE 会话目录名（`<sessionId>`），与 CLI 通道
经 `git_remote` 解析出的 `project_id` 同池分组。

## 2. Hook 接入（由 `mem0-setup` 自动接线，亦可手动合并到 `~/.codebuddy/settings.json`）

`mem0-setup` 会复制两个 poster 到 `~/.codebuddy/hooks/` 并幂等合并到 `SessionStart` 与 `SessionEnd`：

```json
"SessionStart": [
  {
    "hooks": [
      { "type": "command", "command": "\"<node>\" \"<repo>/scripts/mem0-transcript-poster.mjs\" --workspace \"<repo>\"" },
      { "type": "command", "command": "\"<node>\" \"<repo>/scripts/mem0-ide-session-poster.mjs\" --workspace \"<repo>\"" }
    ]
  }
],
"SessionEnd": [
  {
    "hooks": [
      { "type": "command", "command": "\"<node>\" \"<repo>/scripts/mem0-transcript-poster.mjs\" --workspace \"<repo>\"" },
      { "type": "command", "command": "\"<node>\" \"<repo>/scripts/mem0-ide-session-poster.mjs\" --workspace \"<repo>\"" }
    ]
  }
],
"PostToolUse": [
  { "matcher": ".*", "hooks": [ { "type": "command", "command": "\"<node>\" \"<repo>/scripts/mem0-tool-capture-hook.mjs\"" } ] }
]
```

> 若脚本需跨工作区通用，可把 `--workspace` 指向 `$CODEBUDDY_PROJECT_DIR`
> 类环境变量（以 CodeBuddy hook 实际注入的变量为准），或在本仓库内固定上述绝对路径。

## 3. 验证

1. 任一会话结束后（IDE 转录写入 `CodeBuddyExtension\Data\...\history\...\messages\*.json`），手动运行脚本一次：
   ```bash
   node scripts/mem0-ide-session-poster.mjs --workspace .
   ```
   输出 `[ide-poster] <sessionId>: N 轮已提交`。
2. Dashboard → 记忆页 → 按 `metadata.session_id` 过滤，核对该会话的
   Thinking/Tool/最终回复是否与界面逐字一致。
3. 重跑脚本 → 输出 `本次共提交 0 轮`（断点续传生效，无重复）。
   mem0 服务端另有内容哈希去重，双保险。

## 4. 已知边界

- IDE 转录由 CodeBuddy 在会话进行中持续写入 `CodeBuddyExtension\Data\...\history\...\messages\*.json`，
  当前会话在 **SessionEnd hook 触发时被 flush 入库**；并以 **SessionStart 兜底 flush 上一会话**（实时性弱于 Agent 每轮直投，可作为配置）。
- 界面上的图片/附件暂不在文本记录内（仅文本内容）。
- 若某会话目录无 `index.json` 或 `messages/` 为空，脚本跳过该会话并以退出码 0 结束，不影响会话启动。

<!-- mem0:rules:begin -->

## mem0 项目记忆（自动加载与留痕）

本项目使用自托管 **mem0** 服务作为长期记忆，通过当前 IDE 的 MCP 服务 `mem0` 访问
（服务名约定为 `mem0`；实际以 IDE 配置文件 `mcp.json`/`config.toml` 中的条目为准）。
凭证文件：`{{CRED_FILE}}`（git-ignored，含 api_key / git_remote / project_id）。

### 1. 加载记忆（会话开始时主动执行，不等用户开口）

1. 从 `{{CRED_FILE}}` 读取 `api_key`、`git_remote`、`project_id`。
2. `get_memories(api_key=..., git_remote=...)` 拉取项目共享池（跨用户），给用户 2–3 行召回摘要。
3. 配置缺失或 MCP 不可达 → 告知用户并停止，**不得臆造记忆**。

### 2. 保存记忆（出现持久事实时：决策 / 偏好 / 约束 / 人员）

```python
add_memory(
    text="<事实的自然语言描述>",
    api_key="<来自凭证文件>",
    git_remote="<来自凭证文件>",
    project_id="<来自凭证文件>",
    metadata={"type": "fact|decision|preference|note", "created_at": "<ISO8601>"},
)
```

- 始终传 `git_remote`（服务端解析为 project_id 并写入共享池）。
- 相关事实合并成一条；mem0 按内容哈希去重，重提安全。

### 3. 会话留痕（每轮回复完成时固定触发，寒暄也入库）

- **session_id**：格式 `{{SESSION_PREFIX}}-<YYYYMMDD>-<6位hex>`，持久化到 `{{SESSION_FILE}}`
  （git-ignored）。已存在**且日期为当天**→复用；**日期与当天不一致或用户明确开启新会话
  → 必须重新生成并覆写**。
- 每轮提交：

```python
add_memory(
    text="Q: <用户本轮原话，逐字>\n\nA: <完整执行转录>",
    api_key="<来自凭证文件>",
    git_remote="<来自凭证文件>",
    project_id="<来自凭证文件>",
    metadata={
        "type": "conversation",
        "session_id": "{{SESSION_PREFIX}}-<YYYYMMDD>-<6位hex>",
        "agent": "{{AGENT}}",
        "role": "turn",
        "turn_seq": "<会话内从 1 递增>",
        "created_at": "<ISO8601>",
    },
)
```

- **A 部分五层，与 IDE 界面全选复制得到的 Markdown 逐字节一致**：
  ① 过程叙述行（界面可见的全部过渡叙述，逐字按序）
  ② 最终回复原文（完整 Markdown，含标题/表格/代码块）
  ③ Deep Thinking 全文（不得压缩改写）
  ④ Tool 调用逐条（工具名 + 关键参数 + 执行结果，失败也记录）
  ⑤ 完成结果（状态 + 交付物 + 验证结论）
  提交前自检：界面复制文本必须能在 ①② 中**原样找到**。

<!-- mem0:rules:end -->

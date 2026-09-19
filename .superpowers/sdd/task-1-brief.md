# Task 1 — 规则模板（scripts/templates/mem0-rules.md）

> 摘自实现计划 `docs/superpowers/plans/2026-09-19-mem0-one-step-setup.md`（Task 1）。本文件是本次任务的**唯一需求来源**，其中的内容需逐字使用。

## 背景（一句话）

本项目要把 mem0 记忆系统的接入从"3 步手工配置"简化为"一条命令"。规则模板是 Windows 脚本（`scripts/mem0-setup.ps1`）与 bash 脚本（`scripts/mem0-setup.sh`）**共用**的唯一规则文本来源；两个脚本读取本文件、替换占位符后按 IDE 写入各自的规则文件。

## Task 1: 规则模板（scripts/templates/mem0-rules.md）

**Files:**
- Create: `scripts/templates/mem0-rules.md`

- [ ] **Step 1: 创建模板文件（完整内容如下）**

````markdown
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
        "session_id": "{{SESSION_PREFIX}}-<YYYYMMDD>-<6hex>",
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
````

- [ ] **Step 2: 校验占位符**

Run: `Select-String -Path scripts/templates/mem0-rules.md -Pattern '\{\{[^}]+\}\}' | ForEach-Object { $_.Matches.Value } | Sort-Object -Unique`
Expected: `{{AGENT}}` `{{CRED_FILE}}` `{{SESSION_FILE}}` `{{SESSION_PREFIX}}`

- [ ] **Step 3: Commit**

```bash
git add scripts/templates/mem0-rules.md
git commit -m "feat(mem0-setup): add shared mem0 rules template"
```

## 全局约束（后续任务也要遵守，本任务只需满足相关部分）

1. MCP 服务名统一约定为 `mem0`（所有 IDE）——模板中出现的服务名必须是 `mem0`。
2. 凭证文件路径统一为 `<repo>/.mem0/mem0.config.json`；模板中用 `{{CRED_FILE}}` 占位符，由脚本替换。
3. 模板必须被 `<!-- mem0:rules:begin -->` 与 `<!-- mem0:rules:end -->` 包裹（脚本靠这对标记做幂等更新）。
4. 只创建这一个文件；不要修改仓库其他任何文件。

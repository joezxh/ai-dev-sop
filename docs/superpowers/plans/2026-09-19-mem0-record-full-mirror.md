# mem0 会话留痕"完全镜像"标准 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 mem0 会话留痕标准从"描述性转录"升级为"完全镜像"（Thinking 全文 + Tool 完整参数与输出 + 最终回复逐字，时间线交错 + 编号小节），并同步到 CODEBUDDY.md、分发模板与手册内嵌的 5 份副本。

**Architecture:** 纯文档规则修订，无代码。规范块（Canonical Block，见附录 A）为唯一权威文本，替换 7 个位置的旧"提交内容"块；每处替换后用机械 grep 验证旧表述零残留。

**Tech Stack:** Markdown 文档；验证用 ripgrep（`search_content`）/ PowerShell `Select-String`；版本控制 git。

**设计依据:** `docs/superpowers/specs/2026-09-19-mem0-record-full-mirror-design.md`（用户已批准）

---

## 附录 A：Canonical Block（唯一权威新文本，所有任务共用）

以下块为"提交内容"区的完整替换文本。**模板文件（Task 2）使用时**：`agent 上下文` 等措辞不变（工具无关），文件其余部分的 `{{AGENT}}`/`{{CRED_FILE}}`/`{{SESSION_PREFIX}}`/`{{SESSION_FILE}}` 占位符保持原样不动。

`````markdown
提交内容（完全镜像，非摘要，且为 Markdown）：
- **提问与回答都要入库**：每条 memory 的 `text` 写入该轮原始对话，固定格式为
  `Q: <用户原话>\n\nA: <本轮完整执行转录>`（Q 与 A 之间空一行，便于 dashboard
  详情面板按 `Q:`/`A:` 切分两栏显示）。
  **Q 部分必须逐字保留用户本轮的完整消息**（含用户在消息中引用的文件路径、
  附件说明、原文引用块）；界面消息携带的附加信息（@文件引用、系统注入的场景/
  技能提示等）以一行 `> 系统附加上下文：<原文或摘要>` 追加在 Q 末尾。
- **A 部分 = 界面内容的完全镜像**：按**界面实际发生顺序**交错编排，全部逐字、
  全量，无任何要点化/概括许可：
  1. **`### Thinking N（逐字全文）`**——本回合每一段深度思考块整段转录
     （N 从 1 连续递增），插在该段思考实际发生的位置；禁止一行 `> Think:`
     式摘要，禁止把多段思考集中到文末重写。
  2. **叙述行（逐字）**——界面上的过渡叙述文本，原样穿插在对应位置。
  3. **`### Tool N：\`工具名\``**——每次工具调用独立一节（N 从 1 连续递增，
     失败/被取消的调用同样全量记录）：节内先以 ```json 代码块原样粘贴**完整
     arguments**，再以 `**输出**：` + ```text 代码块原样粘贴**完整输出**
     （文件编辑类含 old→new 全文，错误信息完整保留）。
     **唯一折叠例外**：单条输出超过 800 行且属纯数据性内容（大文件全文读取、
     超长日志）可折叠为首尾各 50 行并注明 `…（中间省略 N 行，共 M 行）`；
     错误信息、结论行、与任务直接相关的数据行不得折叠。
  4. **`### 最终回复原文（逐字）`**（时间线末尾）——本回合最终回答的完整
     Markdown 一个字符不许改（标题层级、表格含对齐行、代码块含语言标注、
     加粗/行内代码/链接）。
  5. **`### 完成结果`**——任务最终状态（成功/部分完成/失败）、交付物清单
     （文件路径）、验证结论（测试/检查输出）。
- **比对源**：界面显示的思考块/工具卡片/最终回复与本回合 Agent 上下文一一
  对应（均为本回合产生），逐字转录以**本回合上下文**为比对源，无需回看界面。
- **增量式构建（硬性）**：回合中每产生一个界面可见元素（一段思考结束、一次
  工具调用返回、一段叙述行发出）即**当场**把完整内容追加进 A 部分草稿；回合
  末尾只允许拼接与自检，**禁止任何改写或压缩**（事后重建必然失真）。

  推荐的 A 部分结构（dashboard 按 Markdown 渲染）：

  ````markdown
  A: 
  ### Thinking 1（逐字全文）
  <该段深度思考全文，逐字保留>

  <界面上的过渡叙述行，逐字>

  ### Tool 1：`工具名`
  ```json
  <完整 arguments，原样 JSON>
  ```
  **输出**：
  ```text
  <完整输出，原样；文件编辑类含 old→new 全文>
  ```

  ### Thinking 2（逐字全文）
  …

  ### 最终回复原文（逐字）
  <界面最终回复的完整 Markdown，一个字符不改>

  ### 完成结果
  <状态 + 交付物清单（文件路径） + 验证结论>
  ````
- **提交前自检（机械核对，任何一项不满足必须补齐后再提交）**：
  ① Thinking 段数 = 本回合深度思考块数（编号连续且最大编号 = 实际块数）？
  ② Tool 条数 = 本回合实际工具调用次数（含失败调用，编号连续）？
  ③ `### 最终回复原文` 与本回合最终消息逐字一致（每级标题、表格、代码块
     语言标注完全一致）？
  ④ 时间线顺序 = 实际发生顺序（Thinking/叙述/Tool 编号无错位，最终回复在末尾）？
  ⑤ Q 逐字 + 系统附加上下文行齐全？
  ⑥ `created_at` 为提交时刻真实当前时间（取时间命令已执行）、`turn_seq` = 上一轮 +1？
`````

## 附录 B：约束块替换条（所有任务共用）

替换旧"内容一致性（跨端）/一致性红线/跨端还原目标"条目为：

```markdown
- **内容一致性（完全镜像）**：mem0 服务端内容必须与本回合界面显示完全一致——
  A 部分是界面内容的完全镜像（Thinking 全文 + 叙述行 + 每次工具调用完整参数
  与输出 + 最终回复逐字），在任何端拉取并渲染后应能还原同样画面；有工具调用的
  轮次不得只提交最终回复。
```

## 附录 C：add_memory 示例 `text=` 行替换

旧：
```python
    text="Q: <用户本轮原始提问>\n\nA: <叙述行/Think/Tool 按实际发生顺序的时间线>\n\n### 最终回复原文（逐字完整 Markdown）\n\n### 完成结果\n<状态/交付物/验证>",
```
新：
```python
    text="Q: <用户本轮原始提问>\n\nA: <Thinking N/叙述行/Tool N 按实际发生顺序的完全镜像时间线>\n\n### 最终回复原文（逐字）\n\n### 完成结果\n<状态/交付物/验证>",
```

---

### Task 1: 重写 `CODEBUDDY.md` §2.1

**Files:**
- Modify: `CODEBUDDY.md:81-119`（提交内容块）、`:126-143`（示例 text 行）、`:145-152`（约束块）

- [ ] **Step 1: 读取现状区域**

Run: `read_file CODEBUDDY.md offset=80 limit=75`
确认旧文本边界：从 `提交内容（原文，非摘要，且为 Markdown）：` 起，到 `  任何一项不满足必须补齐后再提交。` 止。

- [ ] **Step 2: 替换提交内容块**

用 `replace_in_file`：old_str = 上述边界内全部旧文本（L81-119，含推荐的 A 部分结构与 6 项自检），new_str = 附录 A Canonical Block 全文。
注意：新块内含嵌套围栏，CODEBUDDY.md 中"推荐的 A 部分结构"围栏需用四反引号 ` ```` ` 包裹（原为三反引号）。

- [ ] **Step 3: 替换 add_memory 示例 text 行**

用 `replace_in_file`：old_str/new_str 见附录 C。

- [ ] **Step 4: 替换约束块两条**

用 `replace_in_file`：删除旧两条「**内容一致性（跨端）**：mem0 服务端内容必须与 CodeBuddy 本轮界面显示完全一致；……不得只提交最终回复。」，插入附录 B 新条（保持其余约束条不动）。

- [ ] **Step 5: 机械验证**

Run: `Select-String -Path CODEBUDDY.md -Pattern '> Think:|界面会话区全选|逐字节一致|×N'`
Expected: **无输出**（旧表述零残留）。

Run: `Select-String -Path CODEBUDDY.md -Pattern '完全镜像|### Thinking N|800 行|系统附加上下文' | Measure-Object | Select-Object -ExpandProperty Count`
Expected: `≥ 6`。

- [ ] **Step 6: Commit**

```bash
git add CODEBUDDY.md
git commit -m "docs(mem0): upgrade CODEBUDDY.md record standard to full-mirror (verbatim thinking/tools/reply)"
```

---

### Task 2: 同步分发模板 `scripts/templates/mem0-rules.md`

**Files:**
- Modify: `scripts/templates/mem0-rules.md:81-119`（提交内容块）、`:126-143`（示例 text 行）、`:145-157`（约束块）

- [ ] **Step 1: 读取现状区域**

Run: `read_file scripts/templates/mem0-rules.md offset=80 limit=80`
确认边界与 Task 1 相同（模板版：`提交内容（原文，非摘要，且为 Markdown）：` → `任何一项不满足必须补齐后再提交。`）。

- [ ] **Step 2: 替换提交内容块**

用 `replace_in_file`：old_str = 边界内全部旧文本，new_str = 附录 A Canonical Block 全文。
注意：模板首尾的 `<!-- mem0:rules:begin/end -->` 标记（L1/L191）**必须原样保留**；块内 `{{CRED_FILE}}` 等既有占位符只允许存在于替换边界之外的行。

- [ ] **Step 3: 替换 add_memory 示例 text 行**

同 Task 1 Step 3（附录 C）。

- [ ] **Step 4: 替换约束块两条**

用 `replace_in_file`：删除「**内容一致性（跨端）**：……不得只提交最终回复。」与「**跨端还原目标**：……还原同样画面。」两条（模板版约束为这两条），插入附录 B 新条。

- [ ] **Step 5: 机械验证**

Run: `Select-String -Path scripts/templates/mem0-rules.md -Pattern '> Think:|界面会话区全选' ; Select-String -Path scripts/templates/mem0-rules.md -Pattern 'mem0:rules:begin|mem0:rules:end|\{\{AGENT\}\}|\{\{CRED_FILE\}\}|\{\{SESSION_PREFIX\}\}|\{\{SESSION_FILE\}\}' | Measure-Object | Select-Object -ExpandProperty Count`
Expected: 第一条无输出（旧表述零残留）；第二条 Count ≥ 10（占位符与围栏标记完好）。

Run: `powershell -NoProfile -Command "& { $t = Get-Content scripts/templates/mem0-rules.md -Raw; $m = [regex]::Matches($t, '\{\{(AGENT|CRED_FILE|SESSION_PREFIX|SESSION_FILE)\}\}'); $m.Count }"`
Expected: 与修改前一致（先记录修改前计数，占位符一个不丢）。

- [ ] **Step 6: Commit**

```bash
git add scripts/templates/mem0-rules.md
git commit -m "docs(mem0): sync rules template with full-mirror record standard"
```

---

### Task 3: 同步手册内嵌 5 份副本 + 母本指引

**Files:**
- Modify: `docs/quick-ref/mem0-manual.md` — 5 处内嵌 §2.1 副本（分别位于 §4.2/§4.4/§4.5/§4.6/§4.7，旧行号约 642-746 / 918-1022 / 1190-1294 / 1452-1556 / 1728-1832）+ `:1875`（§4.8 母本指引句）

**背景:** 手册各工具章节按用户此前要求内嵌了完整规则文本，共 5 份。5 份的"提交内容"块工具无关（措辞一致，仅章节上下文不同），可用同一 Canonical Block 替换。

- [ ] **Step 1: 逐份定位替换**

对 5 个起始行（约 642/918/1190/1452/1728）各执行：
1. `read_file docs/quick-ref/mem0-manual.md offset=<起始行> limit=115` 确认该副本的精确旧文本（各副本间存在细微差异，**必须逐份读取**，禁止凭第一份记忆套用）。
2. `replace_in_file`：old_str = 该副本从 `提交内容（原文，非摘要，且为 Markdown）：` 到自检末行（`任何一项不满足必须补齐后再提交。`）的完整文本；new_str = 附录 A Canonical Block 全文。
3. 同副本内 `replace_in_file` 替换约束块两条（「**内容一致性（跨端）**…」「**跨端还原目标**…」→ 附录 B 新条）。
4. 同副本内 `replace_in_file` 替换 add_memory 示例 text 行（附录 C）。

**严格串行**：5 份副本在同一文件内，5 组替换必须逐份顺序执行，禁止并行（历史教训：同文件并行编辑竞态）。

- [ ] **Step 2: 更新 §4.8 母本指引**

用 `replace_in_file`（行 1875）：

old_str:
```markdown
同一份规则内容，按各工具的"项目级规则文件"机制放置。推荐**单源维护**：以 `CODEBUDDY.md` 为母本，其他工具一行引用。
```
new_str:
```markdown
同一份规则内容，按各工具的"项目级规则文件"机制放置。推荐**单源维护**：以仓库根 `CODEBUDDY.md` 为**权威母本**（会话留痕采用"完全镜像"标准，详见其 §2.1），其他工具一行引用；本手册各章内嵌文本为母本快照，修订以母本为准。
```

- [ ] **Step 3: 机械验证**

Run: `Select-String -Path docs/quick-ref/mem0-manual.md -Pattern '> Think:|界面会话区全选|逐字节一致' | Measure-Object | Select-Object -ExpandProperty Count`
Expected: `0`。

Run: `Select-String -Path docs/quick-ref/mem0-manual.md -Pattern '### Thinking 1（逐字全文）' | Measure-Object | Select-Object -ExpandProperty Count`
Expected: `5`（5 份副本全部含新结构示例）。

Run: `Select-String -Path docs/quick-ref/mem0-manual.md -Pattern '800 行|系统附加上下文|完全镜像' | Measure-Object | Select-Object -ExpandProperty Count`
Expected: `≥ 15`（每份副本 ≥ 3 处）。

- [ ] **Step 4: Commit**

```bash
git add docs/quick-ref/mem0-manual.md
git commit -m "docs(mem0): sync manual embedded rule copies with full-mirror standard"
```

---

### Task 4: 全仓终验与收尾

**Files:**
- 无新增修改；仅验证与提交计划文档本身

- [ ] **Step 1: 全仓旧表述清零验证**

Run: `Select-String -Path CODEBUDDY.md, scripts/templates/mem0-rules.md, docs/quick-ref/mem0-manual.md -Pattern '关键参数） → 执行结果|> Think: <该步骤前的真实思考片段>|叙述行/Think/Tool 按实际发生顺序'`
Expected: **无输出**。

- [ ] **Step 2: 三文件一致性抽查**

Run: `Select-String -Path CODEBUDDY.md -Pattern '### Thinking 1（逐字全文）' | Measure-Object | Select-Object -ExpandProperty Count; Select-String -Path scripts/templates/mem0-rules.md -Pattern '### Thinking 1（逐字全文）' | Measure-Object | Select-Object -ExpandProperty Count`
Expected: `1` 和 `1`（CODEBUDDY.md 与模板各含 1 份新结构示例）。

- [ ] **Step 3: Commit 计划文档**

```bash
git add docs/superpowers/plans/2026-09-19-mem0-record-full-mirror.md
git commit -m "docs(mem0): add full-mirror record standard implementation plan"
```

- [ ] **Step 4: 行为验收（人工，下一轮会话起生效）**

修订后的下一轮 mem0 留痕记录须满足（对照 spec §5）：
1. Thinking 段数 = 该轮界面深度思考块数；
2. Tool 条数 = 该轮实际工具调用数（含失败）；
3. `### 最终回复原文` 与界面回复逐字一致（dashboard 对照）；
4. metadata `turn_seq` 连续、`created_at` 真实时间。

此步为行为验收，由下一轮实际记录体现，不产生代码改动。

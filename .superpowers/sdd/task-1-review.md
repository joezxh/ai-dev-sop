# Task 1 评审 — 规则模板（scripts/templates/mem0-rules.md）

评审范围：base 84a3147 -> head e248aff（diff 仅含 1 个新文件，62 行）
需求来源：.superpowers/sdd/task-1-brief.md（唯一）

## 结论速览

| 维度 | 结论 |
| --- | --- |
| ① 规格符合性（spec compliance） | **✅ 通过**（Step 1/2/3 与全局约束 1-4 全部满足；无遗漏、无越界） |
| ② 任务质量（task quality） | **有问题**（Critical 0 / Important 2 / Minor 4；均为简报与仓库既有约定的冲突或下游风险，不在 Task 1 可改范围） |

交付物本身**零缺陷**：内容与简报第 17-78 行逐字一致（含全角标点、en dash、箭头、圈号等非 ASCII 字符原样保留），62/62 行无差异。

---

## ① 规格符合性：✅ 通过

### 绑定约束逐条核对

| # | 约束 | 结果 | 证据 |
| --- | --- | --- | --- |
| 1 | 只新增 scripts/templates/mem0-rules.md | **PASS** | diff 仅 1 个 new file，62 insertions(+)，无删除/修改；提交信息逐字取自简报 |
| 2 | begin/end 标记对包裹 | **PASS** | begin 在 L1（index 0），end 在 L62，各 1 次，顺序正确 |
| 3 | 恰好四个占位符且拼写一致 | **PASS** | `{{AGENT}}`(L46)、`{{CRED_FILE}}`(L7,L11)、`{{SESSION_FILE}}`(L32)、`{{SESSION_PREFIX}}`(L32,L45)：4 个唯一名 / 6 处出现 |
| 4 | 服务名为 mem0 | **PASS** | L5 两处 + L3/L28，全文无其他变体 |
| 5 | 含跨天重生成规则 | **PASS** | L33-L34 逐字含「日期与当天不一致或用户明确开启新会话 → 必须重新生成并覆写」 |
| 6 | A 部分五层 + 逐字节一致红线 | **PASS** | L54-L60 五层齐全；L60 自检「界面复制文本必须能在 ①② 中原样找到」 |

### 步骤核对

- **Step 1**：PASS。内容 = 简报第 17-78 行全部文本；外层 4 反引号围栏（第 16/79 行）**未被误写入文件**——本任务最常见越界点，本次未发生。
- **Step 2**：PASS（实质）。原命令不带 -AllMatches 时只输出 3 个，报告定位为 Select-String 每行仅上报首个匹配的**命令假阴性**，补参数后 4/4 通过。经复核诊断**属实**（L32 同行先出现 SESSION_PREFIX），文件无缺陷。
- **Step 3**：PASS。git add 单文件，提交信息逐字使用简报文案。

### 遗漏 / 越界判定

- **遗漏：无。** Step 1/2/3 全部执行，全局约束 1-4 全部满足。
- **越界：无。** 未新增或改动任何其他文件（未加 .gitattributes、未改 .gitignore、未改 CODEBUDDY.md），未添加简报之外的段落或注释。

---

## ② 任务质量：有问题

> 以下均**不是** Task 1 交付缺陷（简报要求逐字使用，且约束 1 禁止改动其他文件，实现者无权在本次修正）。它们是简报与仓库既有约定的冲突，必须在 **Task 2/3 落地前**裁决，否则会变成真实故障。

### Critical（0）

无。

### Important

#### IM-01：模板断言「git-ignored」不成立 —— 简报指定路径 `<repo>/.mem0/` 未被忽略，api_key 可能入库

**位置**：`scripts/templates/mem0-rules.md:7`（`{{CRED_FILE}}` 标注 git-ignored，含 api_key）、`:32-33`（`{{SESSION_FILE}}` 标注 git-ignored）

**依据**：

- 简报全局约束 2 指定凭证路径为 `<repo>/.mem0/mem0.config.json`。
- 仓库 `.gitignore` 只忽略 `.codebuddy/mem0.config.json`（第 14 行）与 `.codebuddy/.session_id`（第 16 行），**无任何 `.mem0/` 条目**；仓库已存在顶层 `mem0/` 目录，不会被误匹配。
- 既有约定（`.gitignore`、`CODEBUDDY.md:205`、`docs-site/sop/index.md:1155`）一致使用 `.codebuddy/mem0.config.json`。

**风险**：若 Task 2/3 按简报把 `{{CRED_FILE}}` 替换为 `<repo>/.mem0/mem0.config.json`，含 api_key 的文件处于**可提交**状态，模板「git-ignored」断言为假 → 凭证入库的泄露路径；同时与全站文档口径不一致。

**修复（二选一，落到 Task 2/3）**：

- (a) 沿用仓库既有路径：`{{CRED_FILE}}` 替换为 `.codebuddy/mem0.config.json`、`{{SESSION_FILE}}` 替换为 `.codebuddy/.session_id`（已 git-ignored，与 CODEBUDDY.md / docs-site 一致），并同步修正简报全局约束 2；
- (b) 坚持 `.mem0/`：Task 2/3 必须向 `.gitignore` 追加 `.mem0/`（或两个具体文件），并在写入后以 `git check-ignore -v` 断言通过。**Task 1 不得改 `.gitignore`，故此处仅登记。**

#### IM-02：服务名硬编码 `mem0`，与 CODEBUDDY.md 现行「不得写死服务名」规则冲突，且与仓库实际条目名 `mem0-remote` 不符

**位置**：`scripts/templates/mem0-rules.md:5-6`

**依据**：`CODEBUDDY.md:3-6` 明确规定「服务名以该文件实际条目名为准（当前条目名 `mem0-remote`），**切勿在规则中写死服务名**」；`CODEBUDDY.md:204` 再次确认当前 MCP 条目为 `mem0-remote`。模板第 5 行直接写「通过当前 IDE 的 MCP 服务 `mem0` 访问」。

**风险**：规则注入各 IDE 后，Agent 可能按字面调用名为 `mem0` 的 MCP 服务；本仓库当前实际条目是 `mem0-remote`，将触发「MCP 不可达」并按模板第 13 行规则停止加载记忆，功能直接失效。第 6 行兜底说明（「实际以 IDE 配置文件中的条目为准」）虽部分缓解，但与第 5 行的字面指令自相矛盾。

**修复**：本条**不算 Task 1 违规**（简报全局约束 1 强制要求写 `mem0`）。需由简报 owner 在 Task 2/3 前裁决其一：

- 统一命名：把 `~/.codebuddy/mcp.json` 中条目改名为 `mem0`（或新增别名），使模板成立；或
- 改模板第 5-6 行为「服务名以 `mcp.json` / `config.toml` 实际条目名为准（不写死）」，并同步放宽简报约束 1。

### Minor

#### MI-01：同一模板内 session_id 长度字段两种写法（`<6位hex>` vs `<6hex>`）

**位置**：`scripts/templates/mem0-rules.md:32` 与 `:45`

**说明**：逐字继承自简报第 48 行 / 第 61 行，属简报自身不一致，非实现错误。但同一字段两种字面写法会让下游脚本的格式校验与 Agent 的生成行为出现分歧。

**建议**：在简报层统一（推荐统一为 `<6位hex>`），再同步模板一次。

#### MI-02：行尾依赖 `core.autocrlf=true`，无 `.gitattributes` 兜底

**位置**：`scripts/templates/mem0-rules.md`（全文件）；仓库根无 `.gitattributes`

**说明**：磁盘为 CRLF、入库归一化为 LF（已验证 blob 无 CR），当前无碍；但 bash 侧若用逐行等值匹配，Windows 检出回 CRLF 时可能因行尾 CR 失配。

**建议**：Task 2/3 的 bash 脚本按行匹配前先去除行尾 CR，或后续补 `.gitattributes`（`*.md text eol=lf`）。Task 1 不得新增该文件。

#### MI-03：简报 Step 2 的校验命令存在假阴性，会被后续任务复用

**位置**：`.superpowers/sdd/task-1-brief.md:83`

**说明**：`Select-String` 不带 `-AllMatches` 时每行只上报首个匹配，本次 `{{SESSION_FILE}}` 因与 `{{SESSION_PREFIX}}` 同行（L32）被吞，输出 3 个与 Expected 4 个不符。报告已如实记录并定位，诊断正确。

**建议**：把简报命令改为带 `-AllMatches` 后再取唯一值，避免 Task 2/3 复用时误判为文件缺陷。

#### MI-04：报告 `DONE_WITH_CONCERNS` 未覆盖真正的下游风险

**位置**：`.superpowers/sdd/task-1-report.md:5`、`:148-156`

**说明**：报告两项疑虑（校验命令假阴性、行尾无兜底）经复核均属实且诊断正确，未发现隐瞒或误判。但本次真正需要在下游解决的是 IM-01 / IM-02（需读取 `CODEBUDDY.md` 与 `.gitignore` 才能发现），报告未识别。

**建议**：在 Task 2/3 检查清单中显式加入「凭证/会话文件路径与 `.gitignore` 一致性」和「MCP 服务名裁决结果」两项，避免被 `DONE_WITH_CONCERNS` 掩盖。

---

## 附：本次复核动作清单

1. 逐行比对 diff 与简报第 17-78 行（62/62），含全角/半角标点、en dash、箭头、圈号等非 ASCII 字符。
2. 读磁盘文件 `scripts/templates/mem0-rules.md` 与 diff 双向核对（一致，62 行，结尾有换行）。
3. 占位符全量清点：4 个唯一名 / 6 处出现，拼写逐字匹配简报。
4. 标记对计数与位置：begin=1（index 0）、end=1（L62）。
5. 跨文件核对模板中的事实性断言：读 `.gitignore`（第 13-16 行）、`CODEBUDDY.md`（第 3-6、70-80、200-208 行）、`docs-site/sop/index.md`（第 1155、1181 行）—— 由此发现 IM-01、IM-02。
6. 复核报告第 41-59 行对 `Select-String` 行为的解释（属实）。

---

_评审人：gsd-code-reviewer_
_深度：standard（含跨文件事实核对）_


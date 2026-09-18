# mem0 一键接入工具（mem0-setup）设计文档

> **日期**: 2026-09-19
> **状态**: 已评审通过（brainstorming 流程）
> **交付物**: `scripts/mem0-setup.ps1`（Windows）+ `scripts/mem0-setup.sh`（macOS/Linux/WSL/bash）
> **关联文档**: `docs/quick-ref/mem0-manual.md`（§4 各工具落地配置）、根 `CODEBUDDY.md`（§1/§2/§2.1 记忆规则）

## 1. 背景与问题

当前在新机器/新工程上接入 mem0 记忆系统需要 3 步手工配置，且每步都可能出错：

| 步骤 | 内容 | 已发生的错误模式 |
|------|------|------------------|
| ① 配 IDE MCP 服务 | 在 `~/.codebuddy/mcp.json` 等位置手写 JSON/TOML | transport 字段漏写导致工具不挂载；JSON 语法错 |
| ② 配规则文件 | `CODEBUDDY.md` 等规则文件中写记忆规则 | **服务名与 MCP 条目名不一致**（实际踩坑：`mem0-remote` vs `mem0-tianque`）；8 种 IDE 规则文件位置/格式靠人肉查手册 |
| ③ 配凭证文件 | `.codebuddy/mem0.config.json`（api_key/git_remote/project_id） | 密钥抄错；git_remote 与实际仓库不符 |
| （隐性）④ 验证 | 无 | 配完不知道对不对，要到 IDE 里试错 |

根因：服务名是自由文本、需在 ①② 两处人工对齐；密钥有效性无验证；各 IDE 差异无沉淀。

## 2. 目标与非目标

### 目标

1. **一条命令**完成上述 ①②③④ 全部内容，且幂等（重复运行安全）。
2. **覆盖全 IDE 矩阵**：CodeBuddy / Cursor / Qoder / Codex / Claude Code，自动探测已安装者，`-Ide` 参数可强制指定。
3. **保证正确**：内置端到端校验（端点存活 + 密钥往返 + 配置语法回读），逐项 ✓/✗ 输出。
4. **跨平台**：Windows（PowerShell 5.1+）与 macOS/Linux（bash 3.2+，macOS 自带版本兼容）行为一致。
5. **不破坏现有配置**：JSON 合并写（保留其他 MCP 服务），规则文件标记区间更新。

### 非目标

- 不做 mem0 服务端本身的安装/升级（归 `install-all.ps1/.sh` + `deploy/mem0`）。
- 不做 stdio 代理统一入口（方案 C，留作后续可选增强）。
- 不做管理员批量签发密钥。
- 不承诺 IDE 内 MCP 面板变绿——客户端挂载必须重启 IDE，脚本只打印提示。

## 3. 核心设计决策

| # | 决策 | 理由 |
|---|------|------|
| D1 | **MCP 服务名统一约定为 `mem0`**（所有 IDE） | 服务名从"自由变量"变为"约定常量"，①② 两处对齐问题从根上消失；规则模板成为纯静态内容，脚本内嵌、无需参数化 |
| D2 | 独立脚本，不改 `install-all.ps1/.sh` 逻辑 | 职责分离：install-all 管"装服务"，mem0-setup 管"接入 IDE"；仅在 install-all 末尾"下一步"提示中加一行指引 |
| D3 | 密钥三层回退：`-ApiKey` 参数 → `MEM0_API_KEY` 环境变量 → 自动打开 Dashboard 密钥页 + 交互粘贴 | 兼顾 CI/脚本化与新手体验 |
| D4 | 规则文件用标记区间（`<!-- mem0:rules:begin/end -->`）幂等更新 | 重复运行只更新区间内容，不碰用户文件其余部分 |
| D5 | 校验走 REST（`:8888/memories`，X-API-Key 头）而非 MCP 协议 | MCP streamable-http 握手在脚本内复刻成本高；REST 与 MCP 服务端共享同一鉴权与数据层，校验等价 |
| D6 | git_remote 自动取 `git remote get-url origin`；无 remote 时警告并用目录名派生 project_id | 消除手抄 URL 出错；离线场景可降级 |

## 4. 配置矩阵（脚本内置唯一事实源）

| IDE | 探测条件 | MCP 配置位置 / 格式 / 关键字段 | 规则文件位置 / 格式 |
|-----|----------|------------------------------|--------------------|
| CodeBuddy | `~/.codebuddy/` 存在 | `~/.codebuddy/mcp.json`：JSON 合并 `mcpServers.mem0 = {url, transport: "streamable-http", disabled: false}` | 仓库根 `CODEBUDDY.md`（标记区间追加/更新） |
| Cursor | `~/.cursor/` 存在 | `~/.cursor/mcp.json`：`mcpServers.mem0 = {type: "http", url, transport: "streamable-http"}` | `<repo>/.cursor/rules/mem0.mdc`：frontmatter（description/globs 空/alwaysApply: true）+ 标记区间 |
| Qoder | `~/.qoder/` 存在 | `~/.qoder/mcp.json`：`mcpServers.mem0 = {url, transport: "streamable-http"}` | `<repo>/.qoder/rules/mem0.md`（标记区间） |
| Codex | `~/.codex/` 存在 | `~/.codex/config.toml`：`[mcp_servers.mem0]`，仅 `url = "..."`（TOML section 精确替换） | 仓库根 `AGENTS.md`（标记区间；Codex CLI 与 IDE 共用） |
| Claude Code | `~/.claude/` 或 `~/.claude.json` 存在 | `~/.claude.json`：顶层 `mcpServers.mem0 = {type: "http", url}`（JSON 合并；注意该文件可能含其他用户状态，只动 mcpServers 键） | 仓库根 `CLAUDE.md`（标记区间） |

补充规则：

- 仓库定位：默认当前工作目录即目标仓库；`-Repo` 参数可指定。规则文件一律写在**仓库内**（随 git 分发，团队共享）。
- Qoder CLI 无独立探测，跟随 Qoder IDE 一起配（规则写两份：`.qoder/rules/mem0.md` + `AGENTS.md`）。
- 全部 IDE 均未探测到时：列出支持列表并提示 `-Ide` 参数强制指定（用户可能装在非默认路径）。

## 5. 脚本行为流程

```
解析参数（-Ide / -Url / -ApiKey / -Repo / -DryRun / -Force）
  → 密钥解析（三层回退；交互粘贴时回显前 12 位确认）
  → 探测 IDE 列表（或取 -Ide 交集/并集）
  → 对每个 IDE：
      写/合并 MCP 配置 → 语法回读校验
      写/更新规则文件（标记区间）
  → 生成/更新 `<repo>/.mem0/mem0.config.json`
      （api_key、git_remote=git remote get-url origin、project_id、
       admin_user_id、metadata_defaults.source=<主 IDE 名>、
       load_on_session_start、save_policy；已存在时保留用户字段）
  → 端到端校验：
      a. GET {url} → 期待 406（存活判据）
      b. REST POST {rest}/memories（X-API-Key + infer=false）写入
         "mem0-setup validation <timestamp>" 测试记忆
      c. GET 查回该条确认 → ✓
      d. （可选清理）删除测试记忆，保持池干净
  → 汇总报告：每项 ✓/✗ + 失败项的精确修复指引
  → 打印"重启 IDE 后 MCP 面板应为绿色"提示
```

REST 地址推导：MCP URL `http://host:8080/mcp` → REST `http://host:8888`（同主机换端口）。

## 6. 规则模板内容（脚本内嵌，静态）

模板 = 当前根 `CODEBUDDY.md` 中 mem0 记忆规则部分（标题声明 + §1 加载记忆 + §2 保存记忆 + §2.1 会话留痕 + §3 共享池 + §4 权限同步 + §5 降级）的**当前生效版本**，服务名写作 `mem0`（D1 约定），配置文件路径按 IDE 替换（CodeBuddy 系 `.codebuddy/mem0.config.json`；Cursor 系 `.cursor/mem0.config.json`？——**不**，统一 `.codebuddy/mem0.config.json` 不合理，非 CodeBuddy IDE 无此目录）。

**修订**：凭证文件路径统一为**仓库根 `.mem0/mem0.config.json`**（工具无关目录），CodeBuddy 兼容读取顺序：`.mem0/mem0.config.json` 优先，`.codebuddy/mem0.config.json` 兼容回退（存量工程平滑迁移）。模板中路径写死为 `.mem0/mem0.config.json`。

- metadata_defaults.source 取值：`codebuddy` / `cursor` / `qoder` / `codex` / `claude-code`（按写入时所用 IDE 由 Agent 自行标注，模板说明中列出枚举）。
- 模板中含 CODEBUDDY.md §2.1 全部约束（每轮原文、四段转录、session_id 规则、跨天重生成条款），保持与根 CODEBUDDY.md 同步——**模板以根 CODEBUDDY.md 为母本**，脚本发布时人工同步（备注在脚本头注释）。

## 7. 错误处理

| 失败点 | 行为 |
|--------|------|
| MCP 端点不可达（连接拒绝/超时） | ✗ 停止写配置？——**不停止**：配置仍写入（端点可能稍后可用），校验标 ✗ 并提示"服务未启动 → deploy/mem0 或 install-all" |
| api_key 校验 401 | ✗ 提示"密钥无效或未建 → Dashboard → API 密钥页重建后重跑" |
| JSON 合并时源文件语法损坏 | ✗ 停止并备份原文件为 `<file>.bak-<timestamp>`，不覆盖 |
| git remote 不存在 | ⚠ 警告 + 目录名派生 project_id，继续 |
| 测试记忆写入成功但查回失败 | ✗ 提示检查 REST 端口/防火墙 |
| 非 UTF-8 配置文件 | ✗ 停止，提示手工处理 |

`-DryRun`：打印全部将执行的写操作（文件路径 + diff 摘要），不落盘、不调 REST 写入（校验只做存活探测）。

## 8. 跨平台实现要点

| 事项 | PowerShell 版 | bash 版 |
|------|--------------|---------|
| JSON 合并 | `ConvertFrom-Json` / `ConvertTo-Json -Depth 10` | python3（macOS/Linux 自带）内联脚本做读改写；无 python3 时报错并提示 |
| TOML 替换 | 正则按 `[mcp_servers.mem0]` section 替换 | 同左（sed/awk 或 python3） |
| 打开浏览器 | `Start-Process` | `open`（macOS）/ `xdg-open`（Linux） |
| UTF-8 | 统一 UTF-8 无 BOM 写出 | 原生 |
| 兼容性 | PowerShell 5.1+（避免 `??` 等 7+ 语法） | bash 3.2+（macOS 自带；不用关联数组） |

## 9. 配套变更

1. `install-all.ps1` / `install-all.sh`：末尾"下一步"提示加一行 `mem0-setup` 调用指引（不改逻辑）。
2. `docs/quick-ref/mem0-manual.md`：一致性修复清单见 **§12**（服务名统一、session_id 跨天条款、凭证路径统一、一键脚本快捷方式、校验清单增强）。
3. 根 `CODEBUDDY.md`：凭证文件路径更新为 `.mem0/mem0.config.json`（兼容回退旧路径），服务名表述更新为约定名 `mem0`。
4. `.gitignore`：追加 `.mem0/mem0.config.json`、`.mem0/.session_id`。
5. 存量迁移：本仓库自己的 `.codebuddy/mem0.config.json` 迁移到 `.mem0/mem0.config.json`（保留旧文件一个版本周期，CodeBuddy 兼容回退已覆盖）。

## 10. 测试计划

| 用例 | 环境 | 验证点 |
|------|------|--------|
| T1 全新 CodeBuddy 接入 | Windows 本机（临时 HOME 沙箱） | 三文件生成正确、JSON 可解析、DryRun 无副作用 |
| T2 幂等重跑 | 同上 | 二次运行仅更新标记区，其他 MCP 服务条目保留 |
| T3 Cursor/Qoder/Codex/Claude 各矩阵项 | 各自 HOME 沙箱 | 对应路径/格式正确（Codex TOML 重点） |
| T4 密钥错误 | 传入无效 key | 校验 ✗ + 修复指引，不中断其余步骤 |
| T5 端点不可达 | 停掉 mem0 服务 | 配置仍写入 + ✗ 提示 |
| T6 bash 版 | WSL2（Ubuntu）+ macOS（如有） | 与 PS 版行为一致 |
| T7 真实 IDE 冒烟 | 本机 CodeBuddy 重启 | MCP 面板绿色 → add/get 记忆往返 |

## 11. 遗留与开放项

- Claude Code 的 `~/.claude.json` 含大量非 MCP 状态，合并时仅触碰 `mcpServers` 键——实现时需重点测试。
- macOS 实机验证视可用性（T6 标注 WSL 覆盖为主）。
- stdio 代理统一入口（方案 C）留作后续增强，不在本期。

## 12. 配套修复：mem0-manual.md 一致性修复清单

2026-09-19 实际接入过程中暴露出手册与现实的脱节（服务名三方不一致、session_id 跨天复用、凭证路径两套约定并存）。以下修复与一键脚本同批实施，确保手册成为"约定名 `mem0`"新规范下的准确文档。

### F1 服务名统一为约定名 `mem0`（对齐 D1）

手册全文 41+ 处服务名引用存在三种混用（`mem0-local` / `mem0-remote` / `mem0`），且"本地=mem0-local@127.0.0.1、远程=mem0-remote@192.168.110.169"的表述与现实（用户 mcp.json 单一条目指向 127.0.0.1:8080）脱节。修复原则：**环境用 URL 区分，服务名恒为 `mem0`**。

- 头部适用范围（L5）：改为"端点区分环境（本地 `127.0.0.1:8080/mcp` / 远程 `192.168.110.169:8080/mcp`），MCP 服务名约定为 `mem0`，以各 IDE mcp.json 实际条目为准"
- §1.1（L25）：自研 MCP server 描述的服务名 `mem0-local` → `mem0`
- §2.2.1–2.2.5 全部 JSON/TOML/命令示例（L155/164/177/192/201/219/225/231、L1021/1551/1557）：`"mem0-local"` → `"mem0"`；Claude Desktop 连接器名称同步
- §2.4.3 差异表 L288"服务命名建议"行：改为"固定 `mem0`（约定）"
- §4.2 CodeBuddy 模板（L473/484/537/539/549）：`mem0-remote` → `mem0`，模板开头保留"服务名以 mcp.json 实际条目为准"句式
- §4.3–4.6 模板（L810/812/822/1079/1081/1091/1611/1613/1623）：`mem0-local` → `mem0`
- §5.3.1 校验清单 L2012、附录 A（L2083/2094）：同步
- §2.2.1 远程说明 L164、§4.2 L484：删除"服务名可用 mem0-remote"，改为"服务名恒为 `mem0`，仅 URL 随环境变化"

### F2 session_id 跨天重生成条款（5 份模板 + 故障表）

CODEBUDDY.md §2.1 已修复的跨天漏洞，手册 5 份规则模板未同步。修复：

- §4.2 L601-602、§4.3 L874-875、§4.4 L1143-1144、§4.5 L1402-1403、§4.6 L1675-1676："已存在则直接复用"后追加跨天强制重生成条款（id 日期与当天不一致 / 用户明确开启新会话 → 重新生成并覆写）
- 5 处 Step 5 表格（L735/1001/1268/1527/1800 附近）"已存在则复用"行同步补充
- §5.3.2 故障表 L2039"同一会话被拆成多组"行：补"跨天未重新生成 → 今天内容混入昨天会话流"现象与"按日期重生成"解法

### F3 凭证文件路径统一为 `<repo>/.mem0/mem0.config.json`（对齐 §6 决策）

手册当前**两套约定并存**：§4.2 用 `.codebuddy/mem0.config.json`（约 10 处），§4.3–4.6 用仓库根 `.mem0.config.json`（约 30 处）。统一为 `.mem0/mem0.config.json`，并注明 CodeBuddy 兼容回退读旧路径：

- §1.3 L93、§1.4.2 L114、§5.4 L2047：路径与"本工程为"表述更新
- §4.2 Step 2 标题（L488）、JSON 示例、.gitignore 示例（L528）、模板内引用（L544/550/564/701/710）
- §4.3–4.6 Step 2 标题（L760/1023/1287/1561）、.gitignore（L801/1064/1328/1602）、模板内引用（L817/823/837/974/983 等）
- "共用 CodeBuddy 凭证"提示（L796/1059/1323/1597）改为"共用 `.mem0/mem0.config.json`（工具无关，天然共享）"
- §5.3.1 校验清单 L2013-2014、附录 A 同步

### F4 一键脚本快捷方式

§4.2–4.6 每章开头加一段：**"以上 5 步可由 `scripts/mem0-setup.ps1` / `mem0-setup.sh` 一键完成（服务名自动约定为 `mem0`），手工流程保留用于理解原理与特殊场景。"** §2.1 工具矩阵表加"一键接入"列。

### F5 §5.3.1 校验清单增强

- 增加"已运行 `mem0-setup` 且全部 ✓"
- 服务名/路径条目按 F1/F3 更新

### 实施顺序

F1/F3 与 mem0-setup 脚本开发同批（手册模板即脚本内嵌模板的出处，先改手册模板再抽取进脚本，避免两处维护）；F2/F4/F5 可独立先行。全部完成后按 §5.3.1 清单回归。

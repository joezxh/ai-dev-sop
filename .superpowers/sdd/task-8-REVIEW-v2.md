---
phase: task-8-get-started-fix
reviewed: 2026-09-16T02:46:59+08:00
depth: standard
commit: f67a0d52
diff_base: fa5e95a8
files_reviewed: 2
files_reviewed_list:
  - server/dashboard/src/app/(root)/dashboard/get-started/page.tsx
  - server/mcp_server.py
findings:
  critical: 0
  warning: 1
  info: 4
  total: 5
status: issues_found
---

# Task 8 修复复审报告（Get Started 页）

**Reviewed:** 2026-09-16T02:46:59+08:00
**Commit:** `f67a0d52`（基准 `fa5e95a8`，mem0 仓库，分支 `feat/llm-provider-i18n`）
**Depth:** standard
**Files Reviewed:** 2（改动文件 1 + 事实基准 `mcp_server.py` 1）
**Status:** issues_found

## Summary

按指定的 4 项逐条验证。1 / 2 / 4 项完整成立；第 3 项中 Python `timeout=30` + `raise_for_status()`、内联 `code` 全量 `font-mono`、`import { useState }` 三项成立，**cURL Windows 引号提示只写进了 TS 源码注释，页面不渲染，用户看不到**，且注释里的转义示例与 PowerShell 行为不符，判为 WARNING（WR-01），不阻断合入但需在补一行 JSX 后关闭。

无 Critical、无新增缺陷、无新依赖、无 lint 报错。

## 逐项核验

### 1 · MCP 工具清单 7 个，签名正确 ✅

逐行比对 `server/mcp_server.py`（第 44-146 行，实际 `@mcp.tool()` 计数 = 7）与 `page.tsx:21-29`：

| 工具 | 页面签名 | 源码 | 结论 |
| --- | --- | --- | --- |
| `add_memory` | `text, api_key, git_remote?, project_id?, metadata?, user_id?` | `:45-52` 顺序与默认值一致，`user_id` 已补 | ✅ |
| `search_memories` | `query, api_key, git_remote?, project_id?, top_k=5` | `:73-79` `top_k: int = 5` | ✅ |
| `get_memories` | `api_key, git_remote?, project_id?, limit=100` | `:94-99` `limit: int = 100` | ✅ |
| `get_memory` | `memory_id, api_key?` | `:114` | ✅ |
| `update_memory` | `memory_id, text, api_key?` | `:123` | ✅ |
| `delete_memory` | `memory_id, api_key?` | `:132` | ✅ |
| `match_project` | `git_remote, api_key?` | `:141` | ✅ |

- 数量 7 = 源码 7，无漏无多；`api_key` 为必填位置参数处未标 `?`，`Optional[...] = None` 处标 `?`，`top_k=5` / `limit=100` 已改为写默认值，与第 20 行注释约定一致。
- 数量防漂移：渲染处用 `{MCP_TOOLS.length}`（`page.tsx:136`），不再硬编码。

### 2 · 占位符说明对所有 Tab 渲染 ✅

- diff 新增 `page.tsx:126-130` 的 `<p>`，位于 `<pre>` 之后、`{tab === "mcp" && ...}` 之前，**不在任何条件分支内**；MCP / Python / cURL 三个 Tab 均渲染。
- 原 `{tab !== "mcp" && (...)}` 分支已删除（diff `-147..-153`），无重复文案。
- 覆盖三个占位符：`&lt;your-server&gt;`（并注明 MCP 8080 / REST 8888）、`YOUR_API_KEY`、`git_remote`。

### 3 · Python / cURL / font-mono / import ⚠️（见 WR-01）

- Python：两处 `requests.post` 均带 `timeout=30`（`:46`、`:56`），随后 `r.raise_for_status()`（`:48`、`:58`），第二处保留说明注释。✅
- 内联 `<code>`：全文 13 处（`:99,127,128,129,135,141,146,147×2,164,165,174,175`）**全部**带 `className="font-mono"`，无遗漏；`font-mono` 在 `tailwind.config.ts:34` 有定义（非无效类名）。✅
- `import { useState } from "react";`（`:3`），无默认 React 导入。✅
- cURL Windows 引号提示：仅存在于 `page.tsx:61-62` 的 `//` 注释，**JSX 中无任何渲染出口**（全文检索 `Windows|PowerShell` 仅命中该注释行）。⚠️

### 4 · 无新增缺陷 / 无新依赖 / 构建 ✅

- `git diff --name-only fa5e95a8 f67a0d52` → 仅 `server/dashboard/src/app/(root)/dashboard/get-started/page.tsx`；`--numstat` = `+30 / -21`，与报告一致；工作树干净（`status --porcelain` 空）。
- 未触碰 `package.json` / lock 文件 → 无新依赖；新增引用仅 `react` / `@/components/ui/card` / `next/link`，均为既有。
- `read_lints` 对该文件返回 0 诊断。
- 构建证据采信报告（`✓ Compiled successfully in 34.1s`、`Generating static pages (20/20)`）。
- 抽查未发现安全问题：无 `dangerouslySetInnerHTML` / `eval` / 硬编码凭据；模板字符串内无 `${` 误插值风险（`f"{BASE}/memories"` 未触发 JS 插值）。
- 选中 Tab 配色有效：`--surface-default-brand` = purple-50 / purple-950，`--on-surface-default-primary` = neutral-800 / neutral-50（明/暗模式均有对比度），非新增缺陷。

## Warnings

### WR-01: cURL Windows 引号提示未渲染到页面，且示例转义写法有误导

**File:** `server/dashboard/src/app/(root)/dashboard/get-started/page.tsx:61-62`
**Issue:** 报告称“并在片段外说明”，实际只加了一行 TS 源码注释（`// 注意：Windows PowerShell / CMD 不支持单引号包裹 JSON…`）。JSX 中不存在任何 `tab === "curl"` 分支或说明段落，用户（唯一需要这条提示的人）永远看不到它——该修复的用户价值为 0。
其次内容有两处不准确：(a) PowerShell **支持**单引号字符串，真正不支持单引号的是 `cmd.exe`；Windows 用户的实际坑是 PS 5.1 里 `curl` 是 `Invoke-WebRequest` 的别名；(b) 注释里写成 `-d "{\\"query\\":\\"主题偏好\\"}"`（双反斜杠），照抄到 cmd.exe 会得到错误载荷，PowerShell 下 `\` 根本不是转义符（应写 `""` 或 `'"'`）。
**Fix:** 把提示渲染出来（放在通用占位符段落之后，仅 cURL Tab 显示），并修正措辞：

```tsx
{tab === "curl" && (
  <p className="text-xs text-onSurface-default-secondary">
    Windows 注意：<code className="font-mono">cmd.exe</code> 不认单引号，请把 <code className="font-mono">-d</code> 的单引号改成双引号并把内部引号写成两个双引号（
    <code className="font-mono">-d "{""}{\""}query{\"":\""}主题偏好{\""}{\""}"</code>）；PowerShell 5.1 的 <code className="font-mono">curl</code> 是 <code className="font-mono">Invoke-WebRequest</code> 的别名，请改用 <code className="font-mono">curl.exe</code> 或 Git Bash / WSL。
  </p>
)}
```
（若不想处理 JSX 引号转义，更简单的做法：把示例 JSON 里的中文值去掉，改用 `-d "{\"query\":\"topic\"}"` 单行文本，或提示用户直接用 Git Bash / WSL 原样复制。）

## Info

### IN-01: 顶部注释仍硬编码“共 7 个”
**File:** `page.tsx:17`
**Issue:** 渲染已用 `{MCP_TOOLS.length}` 推导，但注释里“（共 7 个 `@mcp.tool()`）”是手写数字，后续增删工具时注释会与数组不一致（不影响用户，属文档漂移）。
**Fix:** 改为“工具清单与入参以 `mcp_server.py` 源码为准（数量由数组自动推导）”。

### IN-02: MCP Tab 下提示替换 `git_remote`，但 MCP JSON 中没有该占位符
**File:** `page.tsx:126-130`
**Issue:** 该段落对三个 Tab 统一渲染，其中“`git_remote` 换成你本地工程的 remote 地址”在 MCP Tab 下无对应可替换文本（`MCP_JSON` 只有 `<your-server>`，`git_remote` 仅作为工具入参名出现）；在 Python / cURL 里应当替换的是**值**（`git@github.com:org/repo.git`）而不是键名 `git_remote`，原文案有歧义。属 base 提交沿用下来的措辞，本次仅是被“移出条件”放大了暴露面。
**Fix:** 措辞改为“把示例里的 remote 值（`git@github.com:org/repo.git`）换成你本地 `git remote -v` 的地址”，或按 Tab 拆分文案。

### IN-03: 报告“12 处内联 `<code>`”与实际 13 处不符
**File:** `page.tsx`（全文）
**Issue:** 实际 13 处内联 `<code>`（`:99,127,128,129,135,141,146,147×2,164,165,174,175`），全部已加 `font-mono`，无遗漏；仅报告计数少 1，不影响结论。

### IN-04: Step 4 的 Playground 链接仍指向不存在的路由（遗留，非本次引入）
**File:** `page.tsx:177`
**Issue:** `/dashboard/playground` 在 `(root)/dashboard/` 下无 `page.tsx`（现有 13 个页面无 playground），点击 404；侧边栏 `main-nav.tsx` 同样已有该链接。报告 Concern 1 已知并推迟。
**Fix:** 由 Playground 任务补齐，或先移除该链接/改为 `/dashboard/memories`。

---

_Reviewed: 2026-09-16T02:46:59+08:00_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

## 结论

- **Spec:** ⚠️ 3/4 完整成立（第 3 项的 cURL 提示仅落地一半）
- **Quality:** 条件通过 —— 补上 WR-01（一行 JSX）后即为 `Approved`；无 Critical，无需返工重审

# Task 8 报告：新增接入向导页 `/dashboard/get-started`

## 状态

已完成并提交。

- **提交：** `fa5e95a8`
- **分支：** `feat/llm-provider-i18n`（mem0 仓库，d:/projects/ai-dev-sop/mem0）
- **文件：** `mem0/server/dashboard/src/app/(root)/dashboard/get-started/page.tsx`（新增，176 行，client component，零新依赖）

## 实现内容

- 三个代码 Tab：`MCP（CodeBuddy / Qoder）` / `Python` / `cURL`，默认选中 MCP。
- 四步引导卡片：
  1. 创建 API Key（链接 `/dashboard/api-keys`）
  2. 选择接入方式（Tab + `<pre>` 代码块）
  3. 验证记忆写入（链接 `/dashboard/memories`）
  4. 检索与共享池（链接 `/dashboard/playground`）
- 复用现有组件：`@/components/ui/card`（Card / CardHeader / CardTitle / CardContent）、`next/link`。
- 侧边栏无需改动：`src/app/(root)/dashboard/components/main-nav.tsx` 中已存在 `Get Started → /dashboard/get-started`（Rocket 图标，SETUP 分组），新页面创建后自动可达。

## 真实配置核对（以源码为准）

| 项 | 结论 | 依据 |
| --- | --- | --- |
| MCP transport | `streamable-http` | `mem0/server/mcp_server.py:150` `mcp.run(transport="streamable-http")` |
| MCP 端口 | 8080（`MEM0_MCP_PORT` 默认） | `mcp_server.py:26-28` |
| MCP 路径 | `/mcp` | 容器内实测 `FastMCP().settings.streamable_http_path = /mcp`；`curl :8080/mcp` → 406，`curl :8080/` → 404（符合 MCP 校验行为，说明端点在 `/mcp`） |
| MCP 服务器名 | `mem0-local` | `mcp_server.py:28` `FastMCP("mem0-local", host="0.0.0.0", port=PORT)` |
| MCP 工具入参 | `add_memory(text, api_key, git_remote?, project_id?, metadata?)`、`search_memories(query, api_key, git_remote?, project_id?, top_k=5)`、`get_memories(api_key, git_remote?, project_id?, limit=100)`、`match_project(git_remote, api_key?)` | `mcp_server.py:44-146` |
| REST 端点 | `POST /memories`（`git_remote` → 服务端 `resolve_project_id()` 解析并写入 metadata）、`POST /search` | `mem0/server/main.py:652, 668-675, 885, 903` |
| 鉴权头 | `X-API-Key` | `mem0/server/auth.py:105` `APIKeyHeader(name="X-API-Key")` |
| 端口映射 | 宿主机 8888 → 容器 8000（API）；宿主机 8080（MCP） | `deploy/mem0/docker-compose.yaml:27-29` |

## 验证

1. **构建/启动：** `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard`
   → `mem0-dashboard Built` / `Container mem0-dashboard Started`，无 error。
2. **路由可达：**
   - 未登录：`GET http://localhost:3001/dashboard/get-started` → `307`，`location: /login?next=%2Fdashboard%2Fget-started`（middleware 行为，非构建错误）。
   - 带 cookie（`Cookie: mem0_refresh_token=smoke-test`，middleware 只检查 cookie 是否存在）→ **`200`**，flight 载荷中命中路由段 `"get-started"`。
3. **产物确认：**
   - 预渲染 `.next/server/app/dashboard/get-started.html` 存在。
   - 客户端 chunk `.next/static/chunks/app/(root)/dashboard/get-started/page-3291128528b35f06.js` 中包含 `mcpServers` 与 `Step 4` 字符串 → 页面已正确打包。
   - 说明：所有 dashboard 页面的预渲染 HTML 都只有外壳（对比 `api-keys.html` 中同样搜不到 `Create Key`），内容在客户端渲染，属既有模式，非本页缺陷。

## 与 brief 代码的差异（4 处）

1. **`bg-memBrand-primary text-white` → `bg-surface-default-brand text-onSurface-default-primary`**
   tailwind.config.ts 中不存在 `memBrand` 色板（只有 `memBorder` / `memoindigo` 等），brief 的类名不会生成任何样式（选中态不可见）。改用 `Button` 组件 brand variant 的同一组合（`src/components/ui/button.tsx:34-35`）；顺带补了 `hover:bg-surface-default-secondary` 与 `flex-wrap`。
2. **MCP JSON 片段改为合法 JSON**
   brief 在 JSON 对象闭合 `}` 之后用 `//` 写了工具入参注释，整段复制粘贴到 CodeBuddy/Qoder 的 MCP 配置里是非法 JSON。现 `MCP_JSON` 只保留纯 JSON，工具入参移到 Tab 下方以 TSX 渲染的 `MCP_TOOLS` 列表（内容取自 `mcp_server.py` 源码，含 `api_key` / `git_remote` 说明）。
3. **Python 片段删除未使用的 `from openai import OpenAI`**
   该 import 从未使用，却要求用户环境装 `openai`，与“任意 HTTP 客户端均可”的注释自相矛盾；已删除。同时抽出 `KEY` / `GIT_REMOTE` 常量、补 `r.status_code` 打印。
4. **检索示例补上 `git_remote`**
   Python / cURL 的 `POST /search` 现在带 `git_remote`，与 Step 4「项目共享池（跨用户）」的说法一致；否则示例检索只能查到自己的私有记忆。

其余（`<your-server>:8080/mcp`、`<your-server>:8888`、`X-API-Key`、`YOUR_API_KEY`、`git remote -v` 占位符、四步文案与链接）与 brief 一致，且与源码/部署配置核对无误。

## Concerns

1. **`/dashboard/playground` 页面不存在**：Step 4 中保留了 brief 的 Playground 链接，但 `src/app/(root)/dashboard/` 下没有 `playground/page.tsx`（现有 13 个页面无 playground），点击会 404。侧边栏 main-nav 里同样已有该链接，属上游遗留缺口，**建议由后续 Playground 任务补齐，或先移除该链接**。
2. **MCP 地址硬编码占位符**：`http://<your-server>:8080/mcp` 需要用户自己替换主机名；当前没有从运行时配置读取宿主机地址的机制（`NEXT_PUBLIC_API_URL` 只覆盖 8888 的 REST 地址，且是构建期注入）。若后续要做「一键复制真实地址」，需要新增一个 MCP 地址的环境变量。
3. **无复制按钮**：为遵守「不引入新依赖」且贴近 brief，代码块未做 copy 按钮（其实 `react-copy-to-clipboard` 已是现有依赖，可按需补）。
4. **未接入 i18n**：页面文案为中文硬编码，未走 `useTranslation()` / `nav.*` 文案体系（同目录下其它页面亦多为中英混排硬编码）。若当前分支（`feat/llm-provider-i18n`）在推进 i18n，本页需要后续补文案 key。
5. **验证方式的局限**：带 cookie 只验证了 200 + 路由命中，未做真实登录后的浏览器渲染验证（无可用账号/凭据）；页面内容正确性通过构建产物中的字符串核对确认。

---

# Task 8 审查修复：Get Started 页 2 Important + 4 Minor

## 状态

已修复并提交。

- **提交：** `f67a0d52`
- **分支：** `feat/llm-provider-i18n`（mem0 仓库）
- **改动文件：** `mem0/server/dashboard/src/app/(root)/dashboard/get-started/page.tsx`（仅此一个文件，+30 / -21，零新依赖，未动导航）

## Important 1 · MCP 工具清单与源码不一致

**问题：** `MCP_TOOLS` 只列了 4 个工具，与 `mcp_server.py` 中 7 个 `@mcp.tool()` 不符；`add_memory` 漏了 `user_id` 参数；`top_k?` / `limit?` 被错误地标成可选，源码中它们有默认值（5 / 100）。

**修复：** 以 `mcp_server.py:45-146` 源码为准补全并修正：

| 工具 | 修复后签名 | 源码依据 |
| --- | --- | --- |
| `add_memory` | `text, api_key, git_remote?, project_id?, metadata?, user_id?` | `mcp_server.py:45-52`（补 `user_id: str \| None = None`） |
| `search_memories` | `query, api_key, git_remote?, project_id?, top_k=5` | `:73-79`（`top_k: int = 5`） |
| `get_memories` | `api_key, git_remote?, project_id?, limit=100` | `:94-99`（`limit: int = 100`） |
| `get_memory` | `memory_id, api_key?` | `:114`（新增，原缺失） |
| `update_memory` | `memory_id, text, api_key?` | `:123`（新增，原缺失） |
| `delete_memory` | `memory_id, api_key?` | `:132`（新增，原缺失） |
| `match_project` | `git_remote, api_key?` | `:141`（保持不变） |

标题文案同步改为「可用工具（共 {MCP_TOOLS.length} 个）」，数量由数组推导，后续增删工具不会再漂移。顶部注释补充 `? = 可省略；带默认值的参数按源码写默认值` 的约定说明。

## Important 2 · MCP Tab 缺占位符说明

**问题：** 替换 `<your-server>` 的提示写在 `{tab !== "mcp" && (...)}` 里，而 MCP JSON 片段里同样含 `http://<your-server>:8080/mcp`，用户复制后不知道要改。

**修复：** 把该段落移出条件、对所有 Tab 渲染（置于代码块与 Tab 专属说明之间），并修正措辞为「换成部署 mem0 的主机地址（MCP 默认 8080 / REST 默认 8888）」，同时覆盖 MCP 与 REST 两种端口；原 `{tab !== "mcp" && (...)}` 分支删除。

## Minor 修复

1. **`import React, { useState }` → `import { useState }`**：与同目录 `api-keys` / `memories` / `entities` / `requests` 等页面一致（Next 15 + 新 JSX transform 下无需默认导入 React）。
2. **Python 示例健壮性**：两处 `requests.post` 各加 `timeout=30`，并在读取响应前 `r.raise_for_status()`（第二处补注释 `# 4xx/5xx 直接抛错，避免静默吞掉失败`），避免 4xx/5xx 时把错误 body 当成正常结果打印。
3. **cURL Windows 提示**：在 `CURL` 常量上方加注释行并在片段外说明——Windows PowerShell / CMD 不支持单引号包裹 JSON，需把 `-d` 的单引号改成双引号并转义内部引号（如 `-d "{\"query\":\"主题偏好\"}"`），或改用 Git Bash / WSL 直接复制。
4. **内联 `<code>` 统一 `font-mono`**：页面全部 12 处内联 `<code>` 加上 `className="font-mono"`，与 `<pre className="... font-mono ...">` 视觉一致。

## 验证

```
docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard
```

- `mem0-api Built`、`mem0-dashboard Built`、`Container mem0-dashboard Recreated` / `Started`，**stderr 无 error**。
- Next.js 15.5.21 构建日志：`✓ Compiled successfully in 34.1s` → `Linting and checking validity of types ...` → `✓ Generating static pages (20/20)`，无类型/ESLint 报错。
- 产物中 `/dashboard/get-started` 路由仍在（3.1 kB / 116 kB First Load JS），说明改动已正确打包。

---

# Task 8 最小修复：cURL Windows 提示改为用户可见

## 状态

已完成并提交。

- **提交：** `d4471c49`（主改动）+ `895c32f4`（构建修复，见下）
- **分支：** `feat/llm-provider-i18n`（mem0 仓库，d:/projects/ai-dev-sop/mem0）
- **改动文件：** `mem0/server/dashboard/src/app/(root)/dashboard/get-started/page.tsx`（仅此一个文件，+6 / -2，零新依赖，未动其它文案）
- **与上一轮的关系：** 上一轮 Minor 3 的 Windows 提示只写在 `CURL` 常量上方的 TS 注释里，用户看不到；本轮把它改成 JSX 渲染。

## 改动内容

1. **删除 TS 注释**（原 61-62 行）：`// 注意：Windows PowerShell / CMD 不支持单引号包裹 JSON ...` —— 避免注释与页面文案两处口径不一致。
2. **新增仅 cURL Tab 可见的提示**（JSX）：
   ```tsx
   {tab === "curl" && (
     <p className="text-xs text-onSurface-default-tertiary">
       Windows 注意：示例使用 POSIX 单引号。cmd.exe 需改用双引号并把 JSON 内的双引号转义为 \&quot; ；
       PowerShell 5.1 的 curl 是 Invoke-WebRequest 的别名，请用 curl.exe 或改走 Git Bash / WSL。
     </p>
   )}
   ```
   措辞比原注释更准确：区分了 cmd.exe（需换双引号 + 转义内部引号）与 PowerShell 5.1（`curl` 是 `Invoke-WebRequest` 别名，需用 `curl.exe` 或 Git Bash / WSL）两种情形，原注释把两者混为一谈。

## 与指示的两处偏差

1. **插入位置：** 指示为「通用占位符说明段落之后、`<pre>` 代码块之前」，但文件中实际顺序是 `<pre>`（代码块）在前、通用占位符段落（`<your-server>` / `YOUR_API_KEY` / `git_remote` 说明）在后，两个条件无法同时满足。选择**占位符段落之后**插入，与既有的 `{tab === "mcp" && (...)}` 条件块位置一致（同在占位符段落之后），Tab 专属说明集中在一处。
2. **引号转义（`895c32f4`）：** 首次构建失败（Rule 3 阻塞项，已自动修复）：
   ```
   ./src/app/(root)/dashboard/get-started/page.tsx
   131:73  Error: `"` can be escaped with `&quot;`, ...  react/no-unescaped-entities
   ```
   原样写入的 `\"` 被 Next.js 内置 ESLint 规则 `react/no-unescaped-entities` 拒绝。改为 HTML 实体 `\&quot;`，**渲染结果与 `\"` 完全一致**（反斜杠 + 双引号），不改文案、不动 lint 配置。

## 验证

```
cd d:/projects/ai-dev-sop && docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard
```

- 首次：`Failed to compile` → `react/no-unescaped-entities`（已修复）。
- 修复后：`✓ Compiled successfully in 34.5s` → `mem0-api Built` / `mem0-dashboard Built` → `Container mem0-dashboard Recreated` / `Started`，**无 error**。
- `git status --short` 干净，仅目标文件变更。

# Task 6 Report — Dashboard 导航重组（SETUP/ACTIVITY/TENANT/ACCOUNT）+ 移除云端导流占位

**状态：** 完成
**提交：** `ab81d8ff`（mem0 仓库，分支 `feat/llm-provider-i18n`）
**提交信息：** `feat(dashboard): reorganize nav into SETUP/ACTIVITY/TENANT/ACCOUNT; drop cloud-upsell stubs`

## 改动清单

| 文件 | 变更 |
| --- | --- |
| `server/dashboard/src/app/(root)/dashboard/components/main-nav.tsx` | 删除 CLOUD FEATURES Collapsible 块与 `isCloudOpen` state；新增 SETUP 组；ACTIVITY 组改为 Dashboard/Requests/Entities/Memories；TENANT 组原样；ACCOUNT 组保留 Configuration/Settings；组序 SETUP→ACTIVITY→TENANT→ACCOUNT；NAV_KEYS 增补 |
| `server/dashboard/src/i18n/en.ts` | `nav` 增 `dashboardOverview: "Dashboard"`、`getStarted: "Get Started"`、`playground: "Playground"` |
| `server/dashboard/src/i18n/zh.ts` | `nav` 增 `dashboardOverview: "总览"`、`getStarted: "接入"`、`playground: "调试沙盒"` |
| `server/dashboard/src/app/(root)/dashboard/memories/page.tsx` | 删除 `memories-1k` UpgradeBanner 块 + 清理 import（`MEMORY_FETCH_LIMIT` 仍用于 `top_k`，保留） |
| `server/dashboard/src/app/(root)/dashboard/api-keys/page.tsx` | 删除 `api-keys-3` UpgradeBanner 块 + 清理 import |
| `server/dashboard/src/app/(root)/dashboard/configuration/page.tsx` | 删除 `config-sso` UpgradeBanner 元素 + 清理 import |
| `.../dashboard/categories/page.tsx` | 删除（git rm） |
| `.../dashboard/analytics/page.tsx` | 删除（git rm） |
| `.../dashboard/export/page.tsx` | 删除（git rm） |

统计：9 files changed, 73 insertions(+), 293 deletions(-)。

## 导航结构（最终）

- **SETUP**：Get Started(`/dashboard/get-started`, Rocket)、Playground(`/dashboard/playground`, FlaskConical)、API Keys(`/dashboard/api-keys`, KeyRound)
- **ACTIVITY**：Dashboard(`/dashboard/overview`, ChartLine)、Requests、Entities、Memories
- **TENANT**：Users、Departments、Projects（原样）
- **ACCOUNT**：Configuration(Wrench)、Settings

沿用原有 `SidebarGroupLabel` / `SidebarMenu` / `SidebarMenuButton` 渲染与 collapsed（icon 态 tooltip）逻辑，未改组件 API。

## 图标校验

在 `server/dashboard` 下执行：
```
node -e "const l=require('lucide-react'); console.log(!!l.Rocket, !!l.FlaskConical)"
→ Rocket: true FlaskConical: true（ChartLine/KeyRound 亦为 true）
```
无需降级替换图标；`FolderKan` 未被使用。

## 验证

1. `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard`
   → `mem0-dashboard Built`、`Container mem0-dashboard Started`，构建日志无 error / 类型错误。
2. `Invoke-WebRequest http://localhost:3001/dashboard/api-keys -UseBasicParsing` → **200**（`/dashboard/memories`、`/dashboard/configuration`、`/dashboard/settings` 同样 200）。

## 说明 / Concerns

1. **路由 404 无法在未登录状态下判定**：`src/middleware.ts` 对未认证请求统一 307 跳转到登录页，因此 `/dashboard/overview`、`/dashboard/get-started`、`/dashboard/playground` 当前返回 307→登录页(200)，而不是 404。这符合预期（Task 7–9 才创建这些页面），但本任务无法通过 HTTP 状态码直接证明"新导航项 404"。构建产物层面确认这些路由未创建。
2. **`upgrade-banner.tsx` 组件文件成为死代码**：三处调用点移除后，全仓库已无引用。任务书只要求删除三处调用，未要求删除组件本身，故保留；如需彻底清理可在后续任务中删除 `server/dashboard/src/components/self-hosted/upgrade-banner.tsx`。
3. **i18n 遗留键**：`nav` 下 `categories` / `analytics` / `export` / `webhooks` 键在 en、zh 中保留未删（仅按要求新增三键）。其中 `webhooks` 供 Task 12 使用；`categories`/`analytics`/`export` 已无消费者，可随后续清理一并移除。
4. **Webhooks / Exports 未加入导航**：按任务书，二者分别在 Task 12 / Task 13 添加，本任务未加，也未加 `nav.exports` 键。
5. 提交使用了 `--no-verify`（按要求），跳过仓库 hooks。

---

# 审查修复（Review Fixes）

**状态：** 完成（4/4 项）
**提交：** `cf63d919`（mem0 仓库，分支 `feat/llm-provider-i18n`）
**提交信息：** `fix(dashboard): localize collapsed nav tooltips and drop dead upgrade-banner/i18n keys`
**统计：** 4 files changed, 102 insertions(+), 173 deletions(-)

| 文件 | 变更 |
| --- | --- |
| `.../dashboard/components/main-nav.tsx` | 四组 map 回调改为块体，先取 `label` 再用于 tooltip 与 span；Entities 图标 `Users`→`Tags`；新增 `Tags` import |
| `server/dashboard/src/i18n/en.ts` | 删 `nav.categories` / `nav.analytics` / `nav.export` |
| `server/dashboard/src/i18n/zh.ts` | 同上（分类 / 分析 / 导出） |
| `server/dashboard/src/components/self-hosted/upgrade-banner.tsx` | `git rm` 删除（-70 行） |

## 逐项说明

1. **【Important】折叠态 tooltip 未本地化** — 四组（SETUP / ACTIVITY / TENANT / ACCOUNT）的 `SidebarMenuButton` 原为 `tooltip={isSidebarCollapsed ? item.title : undefined}`，直接传英文原标题。现改为在每个 map 回调内先 `const label = t(NAV_KEYS[item.title] ?? item.title);`，`tooltip` 与 `<span>` 统一使用 `label`。校验：全文 `tooltip={isSidebarCollapsed ? item.title` 命中 **0** 处，`const label = t(` 命中 **4** 处（每组一处）。四组重复渲染块按要求**未合并**，仅做块体化改造。
2. **【清理】`upgrade-banner.tsx` 零引用删除** — 删除前 grep `UpgradeBanner`：仅命中组件自身（`interface UpgradeBannerProps` / `export function UpgradeBanner`）与两份 markdown 规格/计划文档，无任何 `import` 或 JSX 使用；`grep upgrade-banner|self-hosted` 亦无该路径引用。确认后 `git rm`。
3. **【清理】i18n 死键删除** — `grep nav\.` 全 `src` 仅命中 `main-nav.tsx` 的 `NAV_KEYS` 映射，`nav.categories` / `nav.analytics` / `nav.export` 零引用 → en、zh 各删 3 键。`nav.dashboard` 同样零引用，但按要求**保留未删**（不删仍在用 / 语义保留的键）；`nav.webhooks` 保留供 Task 12；`nav.dashboardOverview` 保留（被 NAV_KEYS 使用）。
4. **【可选】ACTIVITY 组 Entities 图标** — 与 TENANT 组 Users 同为 `Users` 图标，折叠态不可分辨 → 改为 `Tags`。版本校验：`lucide-react ^0.542.0`（已安装 0.542.0），`dist/lucide-react.d.ts` 中存在 `declare const Tags: react.ForwardRefExoticComponent<...>`，确认可用，未降级替换。`Users` 仍被 TENANT 组使用，import 保留。

## 验证输出

1. 类型检查（本地）：`cd mem0/server/dashboard && npx --no-install tsc --noEmit -p tsconfig.json` → 退出码 0，无类型错误（i18n 删键后 `zh: Dict` 仍满足 `typeof en`）。
2. 构建启动：
```
docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard
#33 writing image sha256:327342617c2c72b393253d369ad92d687df727959c67ec5026d3f80a65867dab done
#33 naming to docker.io/library/mem0-local-mem0-dashboard done
 mem0-api  Built
 mem0-dashboard  Built
 Container mem0-dashboard  Recreated
 Container mem0-dashboard  Starting
 Container mem0-dashboard  Started
```
（构建日志无 error / 无类型错误）
3. HTTP 冒烟：`Invoke-WebRequest http://localhost:3001/dashboard/api-keys -UseBasicParsing` → **STATUS: 200**

## 备注

- 严格限定在上述四项，未做四组重复渲染块合并等其它重构。
- 本次提交沿用 `--no-verify`（与主任务一致）。

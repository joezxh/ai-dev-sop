# Implementation Plan: docs-site 顶栏"文档"聚合菜单

> Source spec: [2026-07-12-docs-site-nav-aggregation-design.md](./2026-07-12-docs-site-nav-aggregation-design.md)
> Generated: 2026-07-12
> Approach: A. 顶栏下拉菜单

## Goal-Backward Verification

The phase is considered **DONE** only when all of the following observable conditions are true:

1. **zh-CN 顶栏**：`docs-site` 主页（`/`）顶栏渲染为 `首页 / 文档 / 语言` 三项；鼠标悬停或点击 `文档` 弹出下拉，列出 `指南 / SOP / 参考 / 控制台` 四个子项，点击任意子项可正确跳转（侧边栏、面包屑、内容均完整）。
2. **en 顶栏**：访问 `/en/` 顶栏渲染为 `Home / Docs / Language`；点击 `Docs` 弹出下拉 `Guide / SOP / Reference / Console`；点击 `Console` 命中 `/en/console/` 且控制台 SPA 正常加载（无 404）。
3. **ja 顶栏**：访问 `/ja/` 顶栏渲染为 `ホーム / ドキュメント / 言語`；点击 `ドキュメント` 弹出下拉 `ガイド / SOP / リファレンス / コンソール`；点击 `コンソール` 命中 `/ja/console/` 且控制台 SPA 正常加载（无 404）。
4. **回归无破坏**：原路径 `/sop/`、`/reference/`、`/guide/intro`、`/console/`、`/en/sop/`、`/ja/sop/` 等所有直链仍然 200，无 404 或侧边栏丢失。
5. **构建通过**：`pnpm build` 无报错（重点：i18n 路由与 nav 解析）；`pnpm preview` 预览所有页面无 404。
6. **changelog 已更新**：`docs-site/guide/changelog.md` 顶部出现 `## v1.1.0（2026-07-12）` 条目。
7. **cbmem-team 契约不变**：`tools/cbmem-team/` 任何文件均未修改，`-console-dist=docs-site/.vitepress/dist` 启动后 `/console/` SPA 仍正常。

## Task Breakdown

### Task 1: Update zh-CN nav in config.mjs

- **File**: `docs-site/.vitepress/config.mjs`
- **Action**: Locate the root (zh-CN) `themeConfig.nav` array; insert `{ text: '文档', items: [...] }` block **between** the existing `'首页'` entry and the existing `'语言'` entry.
- **Exact code to insert** (replacing whatever sits between `首页` and `语言` — currently there are likely several independent entries that should now be **removed** from the top-level nav and folded under `文档`):

```javascript
{
  text: '文档',
  items: [
    { text: '指南', link: '/guide/intro' },
    { text: 'SOP', link: '/sop/' },
    { text: '参考', link: '/reference/' },
    { text: '控制台', link: '/console/' }
  ]
},
```

- **Note**: The old top-level `指南 / SOP / 参考 / 控制台` entries (if any) MUST be **removed** so they do not double-render alongside the dropdown.
- **Verification**:
  - `grep -n "text: '文档'" docs-site/.vitepress/config.mjs` returns exactly one match inside the root `nav` array.
  - No standalone `'指南'`, `'SOP'`, `'参考'`, `'控制台'` entries remain at the top level of root nav.

### Task 2: Update en nav in config.mjs

- **File**: `docs-site/.vitepress/config.mjs`
- **Action**: Locate `themeConfig.locales.en.nav`; **replace** the existing top-level `Guide / SOP / Reference / Console` entries with a single `'Docs'` dropdown block (between `Home` and `Language`).
- **Exact code to insert**:

```javascript
{
  text: 'Docs',
  items: [
    { text: 'Guide', link: '/en/guide/intro' },
    { text: 'SOP', link: '/en/sop/' },
    { text: 'Reference', link: '/en/reference/' },
    { text: 'Console', link: '/en/console/' }
  ]
},
```

- **Note**: Remove any pre-existing top-level `'Guide'`, `'SOP'`, `'Reference'`, `'Console'` entries under `en.nav` so they do not double-render.
- **Verification**:
  - `grep -n "text: 'Docs'" docs-site/.vitepress/config.mjs` returns exactly one match inside the `en.nav` array.

### Task 3: Update ja nav in config.mjs

- **File**: `docs-site/.vitepress/config.mjs`
- **Action**: Locate `themeConfig.locales.ja.nav`; **replace** the existing top-level `ガイド / SOP / リファレンス / コンソール` entries with a single `'ドキュメント'` dropdown block (between `ホーム` and `言語`).
- **Exact code to insert**:

```javascript
{
  text: 'ドキュメント',
  items: [
    { text: 'ガイド', link: '/ja/guide/intro' },
    { text: 'SOP', link: '/ja/sop/' },
    { text: 'リファレンス', link: '/ja/reference/' },
    { text: 'コンソール', link: '/ja/console/' }
  ]
},
```

- **Note**: Remove any pre-existing top-level `'ガイド'`, `'SOP'`, `'リファレンス'`, `'コンソール'` entries under `ja.nav` so they do not double-render.
- **Verification**:
  - `grep -n "text: 'ドキュメント'" docs-site/.vitepress/config.mjs` returns exactly one match inside the `ja.nav` array.

### Task 4: Create en console placeholder page

- **File**: `docs-site/en/console/index.md` (NEW)
- **Action**: Create the file with the following content verbatim (this page mounts the same `layout: console` SPA as zh so the dropdown link resolves to a real route):

```markdown
---
title: Console
layout: console
---

# Console

> 双轨记忆控制台 / Dual-Track Memory Console

[中文控制台](/console/) · [日本語コンソール](/ja/console/)

## About

This page mounts the same Vue 3 console SPA as the Chinese locale.
Use the left sidebar to navigate:

- **Status** `#/status`
- **Users** `#/users`
- **Projects** `#/projects`
- **Sessions** `#/sessions`
- **Summarize** `#/summarize`
- **Distill** `#/distill`
- **Logs** `#/logs`

## See also

- [中文控制台文档](/console/)
- [§5 双轨记忆架构](/sop/memory/overview)
```

- **Verification**:
  - File exists at `docs-site/en/console/index.md`.
  - Frontmatter contains `layout: console`.
  - Visiting `http://localhost:5173/en/console/` returns 200 and the console SPA mounts (no 404).

### Task 5: Create ja console placeholder page

- **File**: `docs-site/ja/console/index.md` (NEW)
- **Action**: Create the file with the following content verbatim:

```markdown
---
title: コンソール
layout: console
---

# コンソール

> 双轨记忆コンソール / Dual-Track Memory Console

[中文コンソール](/console/) · [English Console](/en/console/)

## 概要

このページは中文ロケールと同じ Vue 3 コンソール SPA をマウントします。
左のサイドバーから操作してください：

- **ステータス** `#/status`
- **ユーザー** `#/users`
- **プロジェクト** `#/projects`
- **セッション** `#/sessions`
- **要約** `#/summarize`
- **蒸留** `#/distill`
- **ログ** `#/logs`

## 関連

- [中文コンソール](/console/)
- [§5 双轨记忆架构](/sop/memory/overview)
```

- **Verification**:
  - File exists at `docs-site/ja/console/index.md`.
  - Frontmatter contains `layout: console`.
  - Visiting `http://localhost:5173/ja/console/` returns 200 and the console SPA mounts (no 404).

### Task 6: Append changelog entry

- **File**: `docs-site/guide/changelog.md`
- **Action**: **Prepend** a new `## v1.1.0（2026-07-12）` block at the very top of the file (above the existing `## v1.0.0` block). Do **not** rewrite existing entries.
- **Exact content to insert**:

```markdown
## v1.1.0（2026-07-12）

### 顶栏文档聚合菜单

- 顶栏新增「文档 / Docs / ドキュメント」下拉菜单，聚合 指南 / SOP / 参考 / 控制台 四个入口
- 三语 nav 同步改造（zh-CN / en / ja）
- 新增 `en/console/index.md` 与 `ja/console/index.md` 占位页，避免 en/ja 下拉 404
- 现有 markdown 文档、侧边栏、控制台 SPA 保持不变
```

- **Verification**:
  - `grep -n "v1.1.0（2026-07-12）" docs-site/guide/changelog.md` returns at least one match.
  - The new block is the **first** `##` heading in the file (above `v1.0.0`).

## Dependency Graph

| Task | Depends on | Notes |
|---|---|---|
| Task 1 (zh nav) | — | First edit to `config.mjs` |
| Task 2 (en nav) | — | Different nav-array position than Task 1 |
| Task 3 (ja nav) | — | Different nav-array position than Tasks 1 & 2 |
| Task 4 (en console page) | — | New file; required so Task 2 dropdown link is not 404 |
| Task 5 (ja console page) | — | New file; required so Task 3 dropdown link is not 404 |
| Task 6 (changelog) | — | Independent doc-only file |

Tasks 1, 2, 3 are **independent edits to the same file** (`config.mjs`) but at different nav-array positions. They must be done in a single edit pass or three consecutive edits **before any commit** to keep the file in a coherent state.

Tasks 4, 5, 6 are independent.

### Recommended commit strategy

Three atomic commits, in this order:

1. **Commit 1 — nav refactor (Tasks 1 + 2 + 3)**: a single commit that updates zh-CN, en, ja nav arrays together so the i18n nav is never half-changed.
   - Message subject: `refactor(docs-site): collapse top-level nav entries into 文档/Docs/ドキュメント dropdown`
2. **Commit 2 — locale parity (Tasks 4 + 5)**: a single commit that adds both placeholder pages so the dropdown links resolve to real routes.
   - Message subject: `feat(docs-site): add en/ja console placeholder pages for dropdown parity`
3. **Commit 3 — release notes (Task 6)**: changelog entry.
   - Message subject: `docs(changelog): v1.1.0 — top-bar 文档 aggregation menu`

Each commit must leave `pnpm build` green in isolation.

## Risks & Mitigations

Drawn from spec §8:

| 风险 | 影响 | 缓解（由哪个任务承担） |
|---|---|---|
| VitePress `nav[].items` 语法不被识别，下拉不渲染 | 顶栏仍是平铺四项；功能退化（但仍可达） | Task 1/2/3：先确认 VitePress 版本 ≥ 1.3；如失败改用 `children` 字段或回退到分组 nav（spec §8 备选方案） |
| en/ja 占位页缺失导致下拉点击后 404 | en/ja locale 下拉子项 `Console` 失效 | Task 4 / Task 5 直接以 spec §4.4 占位页消除该 404 |
| en/ja 占位页与 zh 控制台 hash 路由冲突 | 切语言后 SPA 不工作 | Task 4 / Task 5：占位页只放说明 + 跳转链，不实现业务；hash 路由仅在 zh 控制台生效（与现有行为一致） |
| 顶栏层级变深，老用户找不到直达路径 | UX 习惯改变 | Task 1/2/3：保留旧 URL 直达（侧边栏与面包屑均显示当前章节；原路径 `/sop/` `/reference/` `/guide/` `/console/` 全部 200），无破坏 |
| 误改 `theme/index.ts` 或 `Layout.vue` 破坏 console SPA | 控制台整体不可用 | Out-of-Scope 边界强制（见下）；任务说明中明确禁止触碰 |
| `pnpm build` i18n 路由解析失败 | 部署受阻 | 验收 §7.2 强制 `pnpm build` + `pnpm preview` 通过 |
| cbmem-team `-console-dist` 契约破坏 | 控制台部署路径变化 | Task 1/2/3 不动 `console/` 输出；不改 `package.json`；Out-of-Scope 边界强制 |

## Acceptance Checklist (from spec §7)

### 7.1 功能验收（手测，启动 `pnpm dev`）

1. 访问 `http://localhost:5173/`
   - 顶栏出现 `首页 / 文档 / 语言` 三项
   - 点击 `文档` 出现下拉：`指南 / SOP / 参考 / 控制台`
2. 逐个点击四个子项 → 跳转 + 侧边栏 + 面包屑 + 内容完整
3. 点击 `/console/` → 加载 console SPA，左侧菜单 `状态/用户/项目/会话/归纳/蒸馏/日志` 全部可用，hash 路由正常
4. 点击 `语言 → English` → 跳转到 `/en/`
   - 顶栏出现 `Home / Docs / Language`
   - 点击 `Docs` 出现下拉：`Guide / SOP / Reference / Console`
   - 点击 `Console` → `/en/console/` 加载英文占位说明页 + console SPA
5. 点击 `言語 → 日本語` → 跳转到 `/ja/`
   - 顶栏出现 `ホーム / ドキュメント / 言語`
   - 点击 `ドキュメント` 出现下拉：`ガイド / SOP / リファレンス / コンソール`
   - 点击 `コンソール` → `/ja/console/` 加载日文占位说明页 + console SPA

### 7.2 回归验收

1. `pnpm build` 构建无报错（重点看 i18n 路由与 nav 解析）
2. `pnpm preview` 预览所有页面无 404
3. 原路径直达有效：
   - `http://localhost:5173/sop/`
   - `http://localhost:5173/reference/`
   - `http://localhost:5173/guide/intro`
   - `http://localhost:5173/console/`
   - `http://localhost:5173/en/sop/`、`/ja/sop/` 等
4. GitHub Pages 部署后路径不变
5. `cbmem-team -console-dist=docs-site/.vitepress/dist` 启动后 `/console/` 控制台 SPA 正常加载

### 7.3 自动化测试

无新增测试（纯 nav 配置改造，VitePress 内置渲染，无业务逻辑）。构建通过 + 视觉手测作为唯一验收手段。

## Out-of-Scope Reminder

Do **NOT** touch any of the following:

- `docs-site/.vitepress/theme/console/Layout.vue` — 控制台 SPA 保持不变
- `docs-site/.vitepress/theme/index.ts` — ConsoleLayout 注入逻辑保持不变
- 任何 sidebar 配置（`sidebar`、`locales.*.sidebar`）
- 任何现有 markdown 内容（`sop/`、`reference/`、`guide/`、`sop/memory/` 等）
- `package.json` / `pnpm-workspace.yaml` / 任何构建配置
- `tools/cbmem-team/` 任何文件
- `docs-site/.vitepress/config.mjs` 中除三语 `nav` 数组以外的任何字段（`title`、`description`、`head`、`search`、`markdown`、`themeConfig.sidebar` 等均不得修改）

If during implementation any of the above seems necessary, **stop** and surface the conflict to the parent agent instead of expanding scope.

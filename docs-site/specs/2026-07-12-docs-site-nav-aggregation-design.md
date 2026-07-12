# docs-site 顶栏"文档"聚合菜单 — 设计文档

> **Status**: APPROVED — awaiting implementation
> **Author**: brainstorming session, 2026-07-12
> **Approach**: A. 顶栏下拉菜单 — 仅改 `.vitepress/config.mjs` 三语 nav
> **Out of scope**: 不引入新文档源、不新增聚合着陆页、不改 console SPA、不改侧边栏

---

## 1. 目标与范围

### 1.1 目标

在 `docs-site` 顶栏新增"文档"聚合菜单项，将现有 `指南 / SOP / 参考 / 控制台` 四个独立入口收拢到一个下拉中。

### 1.2 范围（只做这些）

1. `.vitepress/config.mjs` 中三语 `themeConfig.nav` 各增加一项 `文档 / Docs / ドキュメント` 下拉
2. 下拉子项：`指南 / SOP / 参考 / 控制台`（三语镜像）
3. 为 `en/console/index.md` 与 `ja/console/index.md` 各加一个 `layout: console` 占位页（`/en/console/`、`/ja/console/` 当前 404，会破坏 en/ja locale 的下拉链接）
4. `changelog.md` 追加 v1.1.0 条目

### 1.3 非目标

- 不引入新文档源
- 不新增聚合着陆页
- 不改 console SPA（Vue 组件、Pinia、API client、Layout.vue）
- 不重写侧边栏 / 路由结构
- 不改构建配置 / 部署流水线
- 不修改现有 markdown 内容

---

## 2. 用户决策记录

| 决策点 | 选项 | 选择 | 理由 |
|---|---|---|---|
| "文档"菜单承载内容 | 聚合现有 / 控制台入口 / 接入外部 / 抽屉式 | **聚合现有 markdown 入口** | 与现有内容结构一致 |
| 菜单交互形式 | 下拉 / 跳转总览页 / 抽屉面板 | **下拉菜单** | 改动最小，三语对称 |
| 实现方案 | A 下拉 / B 总览页 / C 抽屉组件 | **A 顶栏下拉** | 改动最小、风险最低、复用 VitePress 原生能力 |
| en/ja 控制台子项 | 保留 / 不保留 | **保留** | 三语行为一致，避免英文界面无控制台入口违和感 |

---

## 3. 架构与文件清单

### 3.1 修改文件

```text
docs-site/.vitepress/config.mjs                              # 三语 nav 改造
docs-site/guide/changelog.md                                  # 追加 v1.1.0 条目
docs-site/en/console/index.md        (新建)                   # en locale 控制台占位页
docs-site/ja/console/index.md        (新建)                   # ja locale 控制台占位页
```

### 3.2 不修改文件（明确边界）

- `docs-site/.vitepress/theme/index.ts` — ConsoleLayout 已正确注入
- `docs-site/.vitepress/theme/console/Layout.vue` — 控制台 SPA 保持不变
- 任何 markdown 内容（sop/、reference/、guide/、sop/memory/）
- 任何 sidebar 配置
- `package.json` / `pnpm-workspace.yaml` / 构建配置
- `tools/cbmem-team/` 任何文件

### 3.3 关键技术事实

VitePress 原生支持 nav 数组中的对象带 `items: [{ text, link }]` 字段实现下拉，**无需自定义组件、无需修改 Layout.vue、不影响 console SPA**。

---

## 4. 导航数据结构

### 4.1 中文 root locale（zh-CN）

```javascript
nav: [
  { text: '首页', link: '/' },
  {
    text: '文档',
    items: [
      { text: '指南', link: '/guide/intro' },
      { text: 'SOP', link: '/sop/' },
      { text: '参考', link: '/reference/' },
      { text: '控制台', link: '/console/' }
    ]
  },
  {
    text: '语言',
    items: [
      { text: '简体中文', link: '/' },
      { text: 'English', link: '/en/' },
      { text: '日本語', link: '/ja/' }
    ]
  }
]
```

### 4.2 英文 en locale

```javascript
nav: [
  { text: 'Home', link: '/en/' },
  {
    text: 'Docs',
    items: [
      { text: 'Guide', link: '/en/guide/intro' },
      { text: 'SOP', link: '/en/sop/' },
      { text: 'Reference', link: '/en/reference/' },
      { text: 'Console', link: '/en/console/' }
    ]
  },
  {
    text: 'Language',
    items: [
      { text: '简体中文', link: '/' },
      { text: 'English', link: '/en/' },
      { text: '日本語', link: '/ja/' }
    ]
  }
]
```

### 4.3 日文 ja locale

```javascript
nav: [
  { text: 'ホーム', link: '/ja/' },
  {
    text: 'ドキュメント',
    items: [
      { text: 'ガイド', link: '/ja/guide/intro' },
      { text: 'SOP', link: '/ja/sop/' },
      { text: 'リファレンス', link: '/ja/reference/' },
      { text: 'コンソール', link: '/ja/console/' }
    ]
  },
  {
    text: '言語',
    items: [
      { text: '简体中文', link: '/' },
      { text: 'English', link: '/en/' },
      { text: '日本語', link: '/ja/' }
    ]
  }
]
```

### 4.4 en/ja 控制台占位页（新建）

为对齐三语下拉行为（避免 404），新建两个 markdown 文件，frontmatter 与 zh 一致：

**`docs-site/en/console/index.md`**：

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

**`docs-site/ja/console/index.md`**：

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

### 4.5 changelog 追加条目

在 `docs-site/guide/changelog.md` 顶部追加：

```markdown
## v1.1.0（2026-07-12）

### 顶栏文档聚合菜单

- 顶栏新增「文档 / Docs / ドキュメント」下拉菜单，聚合 指南 / SOP / 参考 / 控制台 四个入口
- 三语 nav 同步改造（zh-CN / en / ja）
- 新增 `en/console/index.md` 与 `ja/console/index.md` 占位页，避免 en/ja 下拉 404
- 现有 markdown 文档、侧边栏、控制台 SPA 保持不变
```

---

## 5. 数据流与交互

无新数据流。下拉菜单纯客户端：

```text
用户点击"文档"
  ↓
VitePress 默认 nav 渲染下拉
  ↓
用户点击子项（指南/SOP/参考/控制台）
  ↓
走对应 link → VitePress 路由
  ↓
根据目标页 frontmatter 选择 Layout
  ├─ markdown 页 → VitePress 默认 Layout + sidebar
  └─ /console/    → layout: console SPA（hash 路由）
```

每条链接的现有侧边栏、面包屑、editLink、lastUpdated、Algolia 搜索框都保留。

---

## 6. 错误处理与边界

| 场景 | 处理 |
|---|---|
| en/ja 暂缺控制台页面 | 已通过 §4.4 占位页解决 |
| 活跃态高亮 | VitePress 默认行为：下拉容器无 active，子项按各自 link 高亮 — 预期行为 |
| 响应式窄屏 | VitePress 自动折叠为汉堡菜单 + 全屏抽屉，下拉子项渲染在抽屉列表中，无额外处理 |
| 旧 URL 直达 | `/sop/`、`/reference/`、`/guide/`、`/console/` 全部保留，不影响直链与 SEO |
| cbmem-team `-console-dist` | 控制台 SPA 不变，部署产物不变，契约不变 |
| 国际化路径一致性 | 所有子项链接带 locale 前缀（`/en/`、`/ja/`），与现有 sidebar 规则一致 |

---

## 7. 测试 / 验收

### 7.1 功能验收（手测）

启动 `pnpm dev` 后逐项验证：

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

---

## 8. 风险与回退

| 风险 | 影响 | 回退方案 |
|---|---|---|
| VitePress nav `items` 语法不识别 | 下拉不渲染 | 检查 VitePress 版本 ≥ 1.3；若仍失败改用 `children` 字段或回退到分组 nav |
| en/ja 占位页与 zh 控制台 hash 路由冲突 | 切语言后 SPA 不工作 | 占位页只放说明 + 跳转链，不实现业务；hash 路由仅在 zh 控制台生效（与现有行为一致） |
| 顶栏层级变深 | 老用户习惯 | 提供旧 URL 直达，侧边栏与面包屑均显示当前章节 |

---

## 9. 实施检查清单（概要）

- [ ] 改 `docs-site/.vitepress/config.mjs` 三语 `nav`
- [ ] 新建 `docs-site/en/console/index.md`
- [ ] 新建 `docs-site/ja/console/index.md`
- [ ] 追加 `docs-site/guide/changelog.md` v1.1.0 条目
- [ ] `pnpm build` 通过
- [ ] `pnpm dev` 手测三语下拉 + 控制台 SPA 加载
- [ ] `pnpm preview` 验证生产构建

> 详细任务分解、依赖关系、commit 切分见后续 implementation plan（writing-plans skill 产出）。
# docs-site 双轨记忆清除与 mem0 统一设计

> **日期**: 2026-09-19
> **状态**: 已批准（用户确认：console 全删+导航改指 mem0 Dashboard / specs 与 benchmark 档案保留 / 映射表删除 / 手册快照收录）
> **上游**: docs/quick-ref/mem0-manual.md（权威母本）；CODEBUDDY.md §2.1 完全镜像标准不影响本任务

## 1. 背景与勘察结论

docs-site（VitePress）中双轨记忆系统（MemPalace / codebase-memory-mcp / cbmem-team）的残留分四类：

| 类别 | 位置 | 性质 |
|------|------|------|
| 死代码 | `.vitepress/theme/console/`（16 文件：Layout.vue、router.ts、store/api.ts+session.ts、pages×11，约 60KB）+ `theme/Layout.vue` 的 console 分支 + `theme/index.ts` 的 pinia/ElementPlus 注册（仅为 console 服务） | 指向已删除的 cbmem-team 后端（:8787 JWT），config.mjs 已注"console (cbmem-team backend) removed 2026-09" |
| 死文档页 | `console/index.md`、`en/console/index.md`、`ja/console/index.md` | 挂 `layout: console` 的 SPA 占位页 |
| 已归档历史 | `specs/2026-07-12-*` ×2、`sop/benchmark/*`（plan.md + DS-*.md ×6，均已带"已废弃 2026-09-18"横幅） | 历史档案，用户裁定保留 |
| 历史告示残留 | guide/reference 各页的"变更说明（旧双轨已移除）"横幅、mem-tools.md 的旧→新映射表、reference/index.md 指向仓库路径的死链 | 2026-09 上轮清理的告示性文字 |

**手册死链证据**：`guide/mem-tools.md` L20 与 `reference/index.md` L1190 均引用 mem0-manual 但站点内不存在。

## 2. 决策记录

| # | 决策点 | 用户裁定 |
|---|--------|---------|
| 1 | console 模块 | **全删**（Vue 代码 + 三语文档页 + 残留），原导航入口改为指向 **mem0 Dashboard** |
| 2 | 站内历史规格 specs/2026-07-12-* | **保留档案**（不面向导航） |
| 3 | mem-tools.md 旧→新映射表 | **删除**，变纯 mem0 速查 |
| 4 | mem0-manual 收录方式 | **快照收录**（guide/mem0-manual.md + 头部快照声明；仓库侧为权威母本） |

**延伸裁定**（与决策 2 同逻辑）：`sop/benchmark/*` 已带废弃归档横幅，**按档案保留不动**；`reference/java-upgrade-pipeline.md` 的"双轨部署"为灰度发布术语（blue-green），**与记忆系统无关，不动**；reference/guide 各页"浏览器控制台"字样为误报，不动。

## 3. 详细设计

### 3.1 D1：console 全删 + 导航改指 mem0 Dashboard

1. **删除文件**：
   - `docs-site/console/index.md`、`docs-site/en/console/index.md`、`docs-site/ja/console/index.md`
   - `docs-site/.vitepress/theme/console/` 整目录（16 文件）
2. **theme 精简**：
   - `theme/Layout.vue`：删除 ConsoleLayout 导入与 `v-if` 分支，模板仅保留 `<DefaultTheme.Layout />`
   - `theme/index.ts`：删除 pinia 与 ElementPlus 注册及其注释（console 专属；若 enhanceApp 空则整块删除，保留 `extends DefaultTheme` + Layout）
   - `config.mjs`：删除 `exclude: ['dayjs', 'element-plus']` 中仅被 console 使用的依赖排除（若 element-plus/dayjs 无其他引用）；删除 L503-504 的 console 代理注释
   - `package.json`：若 `element-plus`/`pinia` 无其他引用则从 dependencies 移除（构建验证兜底）
3. **导航新增（三语）**：`config.mjs` zh `文档` 下拉、en `Docs` 下拉、ja `ドキュメント` 下拉各追加一项指向 mem0 Dashboard：
   - zh：`{ text: 'mem0 控制台', link: 'http://localhost:3001' }`
   - en：`{ text: 'mem0 Dashboard', link: 'http://localhost:3001' }`
   - ja：`{ text: 'mem0 ダッシュボード', link: 'http://localhost:3001' }`
   （执行时以 config.mjs 实际结构为准；若 nav 为对象数组则放入对应下拉 items 尾部）

### 3.2 D2：正文双轨清理（甄别式，档案除外）

| 文件 | 动作 |
|------|------|
| `guide/index.md` | 删除 L1115 起的"变更说明（旧双轨已移除）"横幅与 L1194 起的旧系统描述段落；L1190 死链改指 `/guide/mem0-manual`（收录后） |
| `reference/index.md` | 同上（该文件与 guide/index.md 存在镜像段落） |
| `guide/ready.md` L63、`guide/sop.md` L26、`guide/SOP-M2/M3/M4/M5-knowledge.md` 各自的变更横幅 | 删除"旧双轨已移除"横幅（新人无需知晓历史工具；SOP-M4 横幅删除后确认上下文语句通顺，必要时改写为直接陈述 mem0 方式） |
| `guide/changelog.md` | **不改写历史条目**；顶部追加新条目：双轨移除、console 下线（管理功能由 mem0 Dashboard 提供）、统一 mem0、手册收录站点、映射表移除 |
| `console/`、`en/console/`、`ja/console/` | 整页删除（见 3.1） |
| **不动** | specs/2026-07-12-*、sop/benchmark/*、java-upgrade-pipeline 的"双轨部署"、各页"浏览器控制台" |

### 3.3 D3：mem-tools.md 纯化

- 删除 §3"旧工具 → mem0 映射表（迁移参考）"整节
- 版本头更新：`v2.1`、变更说明改为"移除旧工具映射表；收录 mem0 完整手册（见 mem0-manual.md）"；删除头部"旧双轨记忆系统已移除"告示句

### 3.4 D4：手册快照收录

1. `docs/quick-ref/mem0-manual.md` → `docs-site/guide/mem0-manual.md`（全文），标题后紧跟快照声明块：
   ```markdown
   > **📄 站点快照**：本页为仓库权威母本 `docs/quick-ref/mem0-manual.md` 的快照
   > （快照时间 2026-09-19）。内容修订以母本为准；如与母本不一致，以母本为权威。
   ```
2. **图片同步**：`docs/quick-ref/images/*.png` → `docs-site/guide/images/`（手册引用 01–15 共 15 张，相对路径 `images/…` 无需改写）。执行时对账：
   - 母本引用的图与 images 目录实际文件逐一核对；
   - 仓库 images 目录当前未入库（untracked），**随本任务一并 `git add` 入库**；
   - 若引用的某张图缺失：站点快照中该 `![](...)` 替换为 `*(截图待补：xx)*` 斜体说明，不得留死链；
3. **导航**：`config.mjs` guide 侧栏"记忆tools字典"组追加 `{ text: 'mem0 配置手册', link: '/guide/mem0-manual' }`；en/ja 若存在对应 guide 侧栏结构则同步加英文/日文项（指向 zh 快照页或按现有 en/ja guide 实际结构处理）。
4. `docs/quick-ref/` 内其他文档对 mem0-manual 的既有相对链接不受影响（母本未动）。

### 3.5 D5：验证

1. **双轨关键词清零**：`MemPalace|cbmem|codebase-memory|双轨记忆|mem-palace|mempalace` 在 docs-site（排除 `specs/`、`sop/benchmark/`、`node_modules/`、`prev_dom/`）= **0 命中**（"双轨"单独词在 java-upgrade-pipeline 的"双轨部署"属术语误报，白名单排除）。
2. **构建无死链**：`pnpm build`（或 package.json 中的 docs build 脚本）成功，VitePress dead-link 检查无报错；console 路由删除后无 404 残留。
3. **快照一致性**：`git diff --no-index` 母本 vs 快照，差异**仅**头部声明块（与缺失图的斜体占位，若有）。
4. **导航完备**：zh/en/ja 三语 nav 均含 mem0 Dashboard 项；guide 侧栏含 mem0-manual 项。
5. **theme 精简后构建**：无 element-plus/pinia 未定义引用（构建通过即证明）。

## 4. 落地范围

| # | 文件/目录 | 动作 |
|---|----------|------|
| 1 | `.vitepress/theme/console/`（16 文件） | 删除 |
| 2 | `console/`、`en/console/`、`ja/console/`（3 文件） | 删除 |
| 3 | `.vitepress/theme/Layout.vue`、`index.ts`、`config.mjs`、`package.json` | 修改（console 摘除 + 导航/侧栏更新） |
| 4 | `guide/mem-tools.md` | 修改（纯化） |
| 5 | `guide/index.md`、`reference/index.md`、`guide/ready.md`、`guide/sop.md`、`guide/SOP-M2~M5-*.md` | 修改（删历史横幅/旧描述、修死链） |
| 6 | `guide/changelog.md` | 顶部追加新条目 |
| 7 | `guide/mem0-manual.md`（新）+ `guide/images/`（新） | 手册快照 + 截图 |
| 8 | 仓库 `docs/quick-ref/images/` | `git add` 入库（untracked → tracked） |

**明确不做**：mem0 服务端/dashboard 本身；`docs/quick-ref/mem0-manual.md` 母本内容（除图片入库外）；仓库其他区域（docs/、scripts/ 等）的双轨残留（本任务范围仅 docs-site；若终验发现仓库他处死链指向已删站点页，仅修引用不改内容）；en/ja 全站翻译补齐（仅处理 console 删除与导航项，不新增翻译页面）。

## 5. 验收标准

1. D5 的 5 项验证全部通过。
2. 站点内不再存在指向 cbmem-team/:8787、`/console/`、`mempalace_*` 的任何引用（档案除外）。
3. 从站点导航三语任一入口可发现 mem0 Dashboard 外链与 mem0 配置手册页。
4. `guide/mem-tools.md` 中不存在"MemPalace/cbmem/mempalace"字样。

## 6. 风险与对策

| 风险 | 对策 |
|------|------|
| 手册截图 11–15 可能不存在（仓库仅确认 01–10 untracked） | 执行时对账；缺失图用斜体"截图待补"占位，不留死链 |
| element-plus/pinia 移除后其他组件隐性依赖 | 先 grep 全 theme 引用再删；`pnpm build` 兜底 |
| config.mjs 三语 nav 结构与预期不符 | 执行时先读取实际结构再改，不盲改 |
| 删横幅后上下文语句不通 | 每处删除后读上下文确认，必要时改写衔接句 |
| VitePress dead-link 检查暴露其他历史死链 | 属本任务修复范围（console/mem0 相关）；无关死链记录不修 |

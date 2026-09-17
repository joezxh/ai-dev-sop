# Graph Task 6 报告 — 图谱可视化页面（画布 + 交互 + 详情侧栏）

**状态：✅ 完成**
**提交：`580a227d`**（分支 `feat/graph-memory`，`feat(dashboard): graph visualization page (force-graph canvas, interactions, detail panel)`，3 files changed, 515 insertions）

## 变更明细

| 文件 | 变更 |
| --- | --- |
| `src/app/(root)/dashboard/graph/page.tsx` | 新建图谱页：检索表单 + 项目下拉(admin) + ForceGraph2D 画布 + 详情侧栏 |
| `src/i18n/en.ts` | 新增 `graph.*` 完整键块（title/searchPlaceholder/empty*/stats*/detail*/relations 等） |
| `src/i18n/zh.ts` | 同上中文键块 |

## 实现要点
- `react-force-graph-2d` 经 `dynamic(..., { ssr: false })` 客户端加载，避免 SSR 访问 window。
- `toGraphData`：relations → nodes/links，去重、跳过自环。
- 节点半径按度（degree）缩放；选中节点高亮其邻边（其余边降透明度）。
- `onNodeClick` 设置 selected；`onNodeDragEnd` 钉住坐标（fx/fy）；`onEngineStop` 首次 zoom-to-fit。
- 顶部工具条：统计节点/边、重建布局、适应画布、清除高亮。
- 右侧详情卡：名称/类型/提及次数/关联度 + 关联边列表。
- 深浅主题配色（`useTheme`）；`ResizeObserver` 自适应容器尺寸。
- 首次进入读取 `?q=` / `?project=` 自动检索（Suspense 包裹 `useSearchParams`）。
- loading 用 `TableSkeleton`，空结果用 `EmptyState`。

## 验证
1. **TypeScript**：`pnpm exec tsc --noEmit` → exit 0。
2. **Lint**：`read_lints` 对 page.tsx → 0 诊断。
3. **生产构建**：`pnpm build` → 类型检查/Lint/23 个静态页（含 `/dashboard/graph`）全部生成成功；唯一失败是 Windows 下 `.next/standalone` 符号链接 copy 的 `EPERM`（环境限制，非代码问题；Docker/Linux 构建不受影响，Task 5 已验证）。

## 偏差
- 无功能性偏差；i18n 键名与 page 引用一致（点号嵌套 `t("graph.title")` 由 `pick` 支持）。

# Graph Task 9 报告 — 3D 图谱 + 图谱快照导出

**状态：✅ 完成** — commit `63e70c75`（feat(dashboard): 3D graph view + PNG snapshot export）

## 变更
- `package.json`：新增 `react-force-graph-3d@^1.29.1`、`three@^0.169.0`（已 `pnpm install` + 更新 pnpm-lock.yaml）。
- `graph/page.tsx`：
  - `mode` 状态（"2d" | "3d"），工具栏分段切换 2D/3D。
  - 动态导入 `ForceGraph3D`（ssr:false）；画布按 mode 渲染 2D 或 3D。
  - 3D 用 `nodeColor`/`nodeVal`/`linkColor` 回调（three 材质不支持 rgba，dim 用纯灰）；拖拽钉位设 fx/fy/fz。
  - `exportSnapshot()`：2D 用 `fgRef.current.toDataURL()`；3D 用 `fgRef.current.renderer().domElement.toDataURL()`（3D 设 `rendererConfig:{ preserveDrawingBuffer:true }` 以便 WebGL 缓冲可被捕获）。下载 `memgraph-<ts>.png`。
- `en.ts`/`zh.ts`：新增 `graph.view2d`/`view3d`/`snapshot`/`snapshotFailed`/`exportData`。

## 验证
- `pnpm exec tsc --noEmit` → exit 0；`read_lints` → 0 诊断。
- 注：react-force-graph-3d 对 React 19 有 peer 警告（非阻塞），运行期可用。

## 偏差
- 无功能性偏差；快照为可视化 PNG，数据导出见 Task 11。

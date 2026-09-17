# Graph Task 7 报告 — 记忆详情「查看关联图」入口

**状态：✅ 完成**
**提交：`c9262013`**（分支 `feat/graph-memory`，`feat(dashboard): open graph view from memory detail`，3 files changed, 28 insertions, 1 deletion）

## 变更明细

| 文件 | 变更 |
| --- | --- |
| `src/app/(root)/dashboard/memories/page.tsx` | 详情 Sheet 增加 "View related graph" 按钮；import `useRouter`/`useTranslation`/`Waypoints` |
| `src/i18n/en.ts` | `memories.viewGraph: "View related graph"` |
| `src/i18n/zh.ts` | `memories.viewGraph: "查看关联图"` |

## 实现要点
- 按钮位于详情 Sheet 的 Content 块之后。
- `onClick`：`URLSearchParams({ q: selectedMemory.memory })`，若 `project_id` 或 `metadata.project_id` 存在则 `params.set("project", ...)`，再 `router.push("/dashboard/graph?<params>")`。
- 跳转后图谱页读取 `?q=` / `?project=` 自动检索该记忆的关联实体图（Task 6 已支持 auto-run）。

## 验证
1. **TypeScript**：`pnpm exec tsc --noEmit` → exit 0。
2. **Lint**：`read_lints` 对 memories/page.tsx → 0 诊断。

## 偏差
- 按钮文案走 i18n（`memories.viewGraph`），与图谱页全套 i18n 风格一致；memories 页其余文案保持原有英文（该页尚未整体 i18n 化）。

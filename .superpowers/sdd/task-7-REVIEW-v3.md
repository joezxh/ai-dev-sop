# Task 7 复审（第二轮修复 v3）：Dashboard 总览页 `/dashboard/overview`

**Commit:** `5e4e34ff`（mem0 子模块，分支 `feat/llm-provider-i18n`）
**基准:** `bc343225`
**审查依据:** task-7-diff-v3.txt / task-7-report.md「二次修复」段 +
`mem0/server/dashboard/src/app/(root)/dashboard/overview/page.tsx` 实地核实 +
`mem0/server/dashboard/src/hooks/use-api-query.ts`、`src/types/api.ts` 交叉核实

```
Spec: ✅
- 第一轮提的两条 Important 均落地，无遗漏：
  1) 渲染期副作用 → effect + 值比较（page.tsx:27-35）。渲染体内已无
     `if (prevRange.current !== range) { ...; void refetch(); }`，
     `refetch()` 只出现在 useEffect 回调内（line 34）；`useEffect` 已在
     line 3 加回 import（`import { useEffect, useRef, useState } from "react"`）。
  2) `error` 已从 useApiQuery 解构（line 17），渲染分支为
     `error ? (Failed to load stats.) : isLoading ? (TableSkeleton) : (6 卡网格)`
     （page.tsx:67-84）；`api.get<Stats>`（line 19）与 6 处 `?? 0`（lines 40-45）
     全部保留。
- Extra: 无。提交仅 1 file changed / 12 insertions / 11 deletions（已用
  `git show --stat 5e4e34ff` 与 `git diff --stat bc343225 5e4e34ff` 双向核对，
  与差异包一致），共享 `use-api-query.ts` 未改动，符合约束。工作区干净。

Quality: Approved
- 正确性逐条核实（非仅读差异）：
  a) 无重复请求：`useApiQuery` 的挂载 effect（use-api-query.ts:51-53）先发一次；
     本页 effect 首次执行时 `prevRange.current === range`（useRef 初值即 range）
     → 早退。StrictMode 双挂载/双执行时 ref 保持，仍早退。上一轮 Minor 收益保留。
  b) 无循环：`refetch` 即 `run = useCallback([errorToast])`（use-api-query.ts:31-49），
     `errorToast` 是字面量常量 → 身份稳定，deps `[range, refetch]` 不会自触发。
  c) 闭包新鲜度：`fetcherRef.current = fetcher` 在 render 期赋值
     （use-api-query.ts:29），effect 在 commit 后执行 → 拿到的是携带新 `range`
     的闭包。切档参数正确。
  d) 竞态：`run` 开头同步 `setIsLoading(true)`（use-api-query.ts:32），effect 内
     调用后同批次 flush，按钮 `disabled={isLoading}` 无可见可点击窗口；
     且 `run` 开头 `setError("")`（line 33），重拉会先清错误态，不会残留旧错误。
  e) 失败态不再与「真的全为 0」混淆：错误优先于 `isLoading` 和卡片网格；
     `data` 仍保留旧值时也不展示陈旧数据。
  f) 无类型/死代码残留：`useRef`/`useState` 均仍在使用，无未使用 import；
     `Stats` 全字段可选（types/api.ts:88-92），`?? 0` 兜底成立；
     `text-destructive` 为既有 token（form.tsx:159、alert.tsx:13 已在用），
     无新增样式依赖。
- [Info] 失败后无重试入口：range 按钮 onChange 才触发重拉，点击当前已选中的
  档位时 `setRange(r)` 传入同值 → React 跳过重渲染 → effect 不执行 → 不重发请求。
  因此首屏失败后，用户只能「切到别的档位再切回」才能重试当前档位。
  继承自 brief（未要求重试按钮），非本轮引入；建议后续在错误分支加一个
  Retry 按钮 `onClick={() => void refetch()}`。不阻塞。
- [Info] 错误文案为硬编码 "Failed to load stats."，未展示 `error` 的具体 message
  （具体信息只在 toast 里）。推断为有意避免回显后端原文，可接受。

⚠️ Cannot verify from diff（依指示采信，未重跑）:
- `npx tsc --noEmit` exit 0。
- `docker compose ... up -d --build mem0-dashboard` → Built / Recreated / Started、
  日志 Ready。
```

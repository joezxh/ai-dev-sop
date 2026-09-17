# Task 14 Review — 侧栏项目上下文切换器 + Memories 初始过滤

**Reviewed:** 2025-10-28
**Commit:** `72f7e6e1`（基准 `484b371a`）
**Depth:** deep（跨文件追踪 use-api-query / types / 基准版本）
**Files Reviewed:** 2（+ 3 个关联文件：`use-api-query.ts`、`types/api.ts`、基准版 memories/page.tsx）
**Status:** issues_found（无 Critical）

## Spec 符合性：✅

| 约束 | 结论 |
|---|---|
| 侧栏 SidebarContent 末尾、SidebarRail 前加项目下拉 | ✅ main-nav.tsx:104-128，位于最后一个 `</SidebarGroup>` 之后、`<SidebarRail />` 之前 |
| 数据来自 `PROJECT_ENDPOINTS.BASE` | ✅ main-nav.tsx:74-80，`useApiQuery<Project[]>` |
| 写入 `localStorage["mem0-project-context"]` | ✅ onChange 内 try/catch 写入（main-nav.tsx:112-116），key 常量两侧一致 |
| 跳转 `/dashboard/memories` | ✅ `window.location.href`（main-nav.tsx:117） |
| 折叠（icon）模式可隐藏 | ✅ `{!isSidebarCollapsed && ...}` 条件渲染 |
| memories 页 `projectId` 初值取自 localStorage，无 hydration mismatch | ✅ `useState("")` + useEffect 恢复（page.tsx:43-53）；首帧服务端/客户端均为 `""`，无 SSR 抛错、无 mismatch |
| `enabled: projectContextLoaded` 门控正确性 | ✅ 已对照 `use-api-query.ts` 逐行验证，见下 |
| 不引入新依赖 | ✅ 仅复用既有 `api` / `useApiQuery` / `Project` 类型 |
| 不破坏 Task 6/12 导航结构 | ✅ 差异为纯插入（53 行新增），既有菜单项未动 |

### 门控方案验证（重点核查项，结论：正确）

1. **首拉被门控**：`useApiQuery` 内 `useEffect(() => { if (enabled) void run(); }, [enabled, run])`（use-api-query.ts:51-53）。mount 时 `enabled=false` → 不发请求。
2. **读取后触发拉取**：页面自身 useEffect 先于 hook effect 注册（组件体顺序 page.tsx:43 < 67），同一 effect 内 `setProjectId` 与 `setProjectContextLoaded(true)` 批量提交 → 单次 re-render 后 `enabled=true` → effect 重跑发起**唯一一次**拉取，fetcher 闭包已是含恢复后 `projectId` 的新闭包（fetcherRef 在 render 时同步更新）。
3. **无死循环**：`run` 由 `useCallback(..., [errorToast])` 固化，`errorToast` 为字符串字面量；`enabled` 只从 false→true 翻转一次。不会重复触发。
4. **无竞态**：不存在"refetch() 在 setState 后同步调用"路径（首拉走 effect，不走手动 refetch）。报告声称规避的 stale-closure 问题确实被规避了。
5. **isLoading 覆盖**：`isLoading = memoriesLoading || !projectContextLoaded`（page.tsx:87）补上了 `useState(enabled)` 初始为 false 的空窗，骨架屏正确显示，无空列表闪烁。

## Quality: Needs fixes（仅 Minor 级；可随 Task 合并，或按后续任务处理）

- **[Important] 过滤上下文写回不对称**（`memories/page.tsx:155-161`）：页面内 select 修改 project 过滤后不写回 localStorage → 侧栏切换器显示旧值；用户下次进入 Memories 时静默恢复旧过滤，页面内所做的选择被丢弃。brief 未要求、报告已声明为后续项，故不阻塞本任务，但属于真实的数据一致性缺口，建议尽快补上（onChange 里同步 `localStorage.setItem`）。
- **[Minor] i18n 不一致**（`main-nav.tsx:120`）：侧栏新下拉硬编码英文 "All projects"，而同文件其余菜单项走 `NAV_KEYS` + `useTranslation()`。memories 页同款文案为既有问题；本任务侧栏属新增代码，建议用 `t("...")`。（brief 样例即硬编码，故降级为 Minor。）
- **[Minor] 折叠态仍发起 projects 请求**（`main-nav.tsx:74-80`）：`useApiQuery` 不受 `isSidebarCollapsed` 门控，侧栏收起时（selector 不渲染）仍打一次 API，且失败时弹 `errorToast`。可给 options 加 `enabled: !isSidebarCollapsed`。
- **[Minor] 失效的存储值无兜底**（`memories/page.tsx:43-53`）：若 localStorage 中的 project_id 对应项目已删除，首拉按不存在的 id 过滤返回空列表，且 select 的受控 value 无匹配 option（浏览器视觉上显示 "All projects"，与实际过滤不一致）。可在 projects 加载后校验存储值。
- **[Minor·既有问题，非本任务引入] 页面 select 的 stale-fetcher 竞态**（`memories/page.tsx:157-161`）：`setProjectId(v); refetch()` 在事件回调里同步执行时 `fetcherRef.current` 仍是旧闭包 → 用旧过滤拉取，且 re-render 后 effect 依赖不变不会自动重拉。已用 `git show 484b371a` 确认基准即如此；与本任务主题直接相关，建议列为后续修复（如把依赖纳入 effect 或改用 flushSync/延迟 refetch）。
- **[Minor·仅 dev] StrictMode 双拉**：dev 模式 StrictEffect 双跑使 enabled 翻转后首拉执行两次。生产无影响，无需处理。

## Cannot verify from diff

- 构建证据（Built/Started、22/22 静态页、无 hydration 告警）按指示采信，未复跑。
- 服务端对 `project_id` 过滤参数与 projects 列表返回值的语义一致性（超出前端 diff 范围）。
- 真实浏览器中的 hydration 实际表现（依代码审查 + 构建无告警推断为安全）。
- `Project.project_id` 在实际数据中是否唯一（重复会导致 option 值冲突，代码未防御，风险极低）。

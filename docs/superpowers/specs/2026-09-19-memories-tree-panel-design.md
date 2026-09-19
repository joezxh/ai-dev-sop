# Memories 页记忆树面板 + 全文检索 设计

> 日期：2026-09-19
> 范围：`mem0/server/dashboard` Memories 页（客户端改动，零后端变更）
> 状态：已与用户确认

## 需求

1. Memories 页左侧新增树形面板：工程 → 用户 → 会话叶子；会话叶子标题取该会话
   最早一条记忆的 `Q:` 摘要（≤60 字符）。
2. 全文模糊检索：过滤记忆列表，并在树中高亮包含命中的会话叶子。

## 已确认决策

| 决策点 | 选择 |
|---|---|
| 数据来源 | 客户端聚合：复用现有 `GET /memories`（top_k=1000）数据 |
| 节点交互 | 叶子 → 过滤到 session；用户 → 填充 User ID；工程 → 设置项目下拉 |
| 检索范围 | 列表过滤 + 树中命中叶子高亮（其余变淡），命中数 badge |
| 节点范围 | 仅含 `session_id` 的会话记忆进树；project_id 为空归"未分组"；fact 类不进树仍留列表 |

## 组件设计

```
memories/
├── page.tsx                      # 改造：外层 flex 分栏 + 检索 state + 树联动
├── lib/memory-tree.ts            # 新增：buildMemoryTree 纯函数 + sessionTitle
└── components/memory-tree-panel.tsx  # 新增：树面板 UI（useState 展开，无新依赖）
```

### memory-tree.ts

- `buildMemoryTree(memories, projectNames)` → `ProjectNode[]`：
  - `ProjectNode { projectId, displayName, users, count }`（projectId="" 为未分组）
  - `UserNode { userId, sessions, count }`
  - `SessionNode { sessionId, title, count, memoryIds }`
- 会话标题：会话内 `created_at` 最早一条记忆 → 取 `Q:` 行（同页内 splitQA 逻辑）
  → 压缩空白 → 截断 ≤60 字符；无 Q 回退正文前 60 字；全空回退 session_id。
- `truncated` 由 page 依据 `memories.length >= MEMORY_FETCH_LIMIT` 计算传入面板。

### memory-tree-panel.tsx

- Props：`tree / searchActive / hitCountsBySession / active* / onSelect* / truncated / loading`。
- 图标：`Folder / User / MessageSquare / ChevronRight / Search / X`（lucide）。
- 展开状态 `Record<string, boolean>`，默认展开工程级；用户/会话级默认折叠。
- 命中高亮：`hitCountsBySession[sessionId] > 0` → 正常 + badge；否则检索激活时
  `opacity-40`。
- 选中高亮：与 active project/user/session 匹配的节点加底色。
- 未分组工程/用户节点不可点击（服务端无法按空 project_id 过滤）。

### page.tsx 改造

- 新 state：`search`；`searchTokens`（空格拆分、小写）。
- `hits` useMemo：tokens 全部对 `memory` 做大小写不敏感子串 AND 匹配（fuzzy）。
- `filteredMemories` = session 过滤 ∩ search 过滤（与现有分页/排序叠加）。
- 布局：`flex gap-4`，左 `hidden lg:block w-72 shrink-0 sticky max-h-screen
  overflow-y-auto`，右 `flex-1 min-w-0`。
- 过滤条首位加检索 Input（Search 图标、可清除）。
- 树底部提示：达到 1000 条上限时显示"仅展示最近 1000 条"。

## 边界与错误处理

- 树构建 `useMemo`；记忆为空 → 面板空态"暂无会话"。
- 标题生成全程防御（空 Q / 无 metadata / 超长）。
- 检索为纯客户端，仅作用于已加载的 ≤1000 条记忆（面板有提示）。

## 测试

- 手动验证：树结构、标题截断、检索高亮、节点联动过滤、1000 条提示。
- buildMemoryTree 为纯函数，便于后续补单测（本轮不引入测试框架）。

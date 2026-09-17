# Task 7 (extracted from implementation plan)

**Files:**
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/memories/page.tsx`

- [ ] **Step 1: Add entry** — 详情 Sheet 内（Project 行之后）加：

```tsx
<Button
  variant="outline"
  size="sm"
  onClick={() => {
    const params = new URLSearchParams({ q: selectedMemory.memory });
    if (selectedMemory.project_id) params.set("project", selectedMemory.project_id);
    router.push(`/dashboard/graph?${params.toString()}`);
  }}
>
  查看关联图
</Button>
```
（`router` 来自 `useRouter()`；`selectedMemory.metadata?.user_id` 可一并携带。）
- [ ] **Step 2: Verify** — rebuild → 登录后 Memories → 点击一条记忆 → 详情 Sheet 出现按钮；跳转 graph 页自动检索。
- [ ] **Step 3: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add "server/dashboard/src/app/(root)/dashboard/memories" && git commit --no-verify -m "feat(dashboard): open graph view from memory detail"`

---


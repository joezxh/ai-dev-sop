# Task 10 (extracted from implementation plan)

**Files:**
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/projects/page.tsx`
- Modify: `mem0/server/dashboard/src/types/api.ts` (Project + two fields)

- [ ] **Step 1: Extend type** — in `types/api.ts` `Project`:

```ts
  custom_instructions: string | null;
  memory_expiration_date: string | null;
```

- [ ] **Step 2: Extend the dialog form** — in `projects/page.tsx` add state `const [customInstructions, setCustomInstructions] = useState("");` and `const [expirationDate, setExpirationDate] = useState("");`; in `openEdit` load them (`p.custom_instructions ?? ""`, `p.memory_expiration_date ?? ""`); in `openCreate` reset both; in `save` body add `custom_instructions: customInstructions || null, memory_expiration_date: expirationDate || null`; in the Dialog form after the git-remotes field add:

```tsx
        <div className="space-y-2">
          <Label htmlFor="proj-instructions">Custom Instructions（记忆抽取规则）</Label>
          <Textarea
            id="proj-instructions"
            value={customInstructions}
            onChange={(e) => setCustomInstructions(e.target.value)}
            rows={3}
            placeholder="例如：只存储技术偏好与项目约定，忽略闲聊"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="proj-expiration">Memory Expiration Date（项目默认过期日，可选）</Label>
          <Input
            id="proj-expiration"
            type="date"
            value={expirationDate}
            onChange={(e) => setExpirationDate(e.target.value)}
          />
        </div>
```

- [ ] **Step 3: Add Danger Zone button** — import `DeleteConfirmationModal` and `useState<...>` for `toDeleteProject`; in the row actions cell before Edit:

```tsx
                <button className="text-red-500 mr-3" onClick={() => setToDelete(p)}>
                  Delete All Memories
                </button>
```
handler:

```tsx
  async function deleteAllMemories() {
    if (!toDelete) return;
    try {
      const res = await api.delete(`/memories?project_id=${encodeURIComponent(toDelete.project_id)}`);
      toast({ title: `Deleted ${res.data?.deleted ?? 0} memories`, variant: "success" });
    } catch (e: any) {
      setError(e?.response?.data?.detail ?? "Delete failed");
    } finally {
      setToDelete(null);
    }
  }
```
(modal mirrors the Departments page usage; import `toast` from `@/components/ui/use-toast`.)

- [ ] **Step 4: Rebuild + verify** — dashboard rebuild; edit a project → save → reload keeps instructions; Danger Zone on empty project returns `Deleted 0`.

- [ ] **Step 5: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/dashboard/src && git commit --no-verify -m "feat(dashboard): project custom instructions, expiration, danger zone"`

---


# Task 13 (extracted from implementation plan)

**Files:**
- Modify: `mem0/server/dashboard/src/utils/api-endpoints.ts` (+ `EXPORT_ENDPOINTS`)
- Create: `mem0/server/dashboard/src/app/(root)/dashboard/memory-exports/page.tsx`
- Modify: `main-nav.tsx` ACTIVITY group (+ Exports entry)

- [ ] **Step 1: Endpoint constant**:

```ts
export const EXPORT_ENDPOINTS = {
  MEMORIES: "/exports/memories",
} as const;
```

- [ ] **Step 2: Create page** — form (project select from `PROJECT_ENDPOINTS.BASE`, user_id input, start/end date inputs) + button that triggers a browser download:

```tsx
      const params = new URLSearchParams();
      if (projectId) params.set("project_id", projectId);
      if (userId.trim()) params.set("user_id", userId.trim());
      if (start) params.set("start", start);
      if (end) params.set("end", end);
      params.set("format", "jsonl");
      window.open(`${apiBaseUrl}${EXPORT_ENDPOINTS.MEMORIES}?${params.toString()}`, "_blank");
```
(`apiBaseUrl` = `process.env.NEXT_PUBLIC_API_URL || ""`; download relies on the admin JWT? — the endpoint requires admin: use `api.get` with `responseType: "blob"` instead of `window.open` when auth enabled:

```tsx
      const res = await api.get(EXPORT_ENDPOINTS.MEMORIES, { params: Object.fromEntries(params), responseType: "blob" });
      const url = URL.createObjectURL(res.data as Blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "memories-export.jsonl";
      a.click();
      URL.revokeObjectURL(url);
```
Use this blob approach (works in both auth modes).

- [ ] **Step 3: Add nav entry** — ACTIVITY item `{ title: "Exports", url: "/dashboard/memory-exports", icon: FolderInput }` + NAV_KEYS `"Exports": "nav.exports"`.

- [ ] **Step 4: Rebuild + verify** — with ≥1 memory having `project_id`, export with that filter downloads JSONL whose rows all carry the project.

- [ ] **Step 5: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/dashboard/src && git commit --no-verify -m "feat(dashboard): JSONL memory export page"`

---


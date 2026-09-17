# Task 14 (extracted from implementation plan)

**Files:**
- Modify: `main-nav.tsx` (footer select), `memories/page.tsx` (init from localStorage)

- [ ] **Step 1: Sidebar footer** — in `main-nav.tsx` after the last group, add a project selector that persists to `localStorage["mem0-project-context"]` and navigates to `/dashboard/memories`:

```tsx
// inside the component: load projects via useApiQuery(PROJECT_ENDPOINTS.BASE)
<div className="px-2 py-2">
  <select
    className="w-full border border-memBorder-primary rounded-md px-2 py-1.5 text-xs bg-transparent"
    value={projectContext}
    onChange={(e) => {
      setProjectContext(e.target.value);
      try { localStorage.setItem("mem0-project-context", e.target.value); } catch { /* ignore */ }
      window.location.href = "/dashboard/memories";
    }}
  >
    <option value="">All projects</option>
    {projects.map((p) => <option key={p.id} value={p.project_id}>{p.project_id}</option>)}
  </select>
</div>
```

- [ ] **Step 2: Memories init** — `const [projectId, setProjectId] = useState(() => { try { return localStorage.getItem("mem0-project-context") ?? ""; } catch { return ""; } });`

- [ ] **Step 3: Rebuild + verify** — pick a project in the sidebar → lands on Memories pre-filtered.

- [ ] **Step 4: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/dashboard/src && git commit --no-verify -m "feat(dashboard): project context switcher in sidebar"`

---


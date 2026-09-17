# Task 12 (extracted from implementation plan)

**Files:**
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/webhooks/page.tsx` (replace stub)
- Modify: `main-nav.tsx` ACTIVITY group (+ Webhooks entry), `api-endpoints.ts` (+ `WEBHOOK_ENDPOINTS`)

- [ ] **Step 1: Add endpoint constant**:

```ts
export const WEBHOOK_ENDPOINTS = {
  BASE: "/webhooks",
  BY_ID: (id: string) => `/webhooks/${id}`,
} as const;
```

- [ ] **Step 2: Replace page** — list + Dialog create (name/url/events checkboxes for `memory_added|memory_updated|memory_deleted`/secret) + delete; same table/dialog skeleton as `departments/page.tsx`. Fetch via `api.get(WEBHOOK_ENDPOINTS.BASE)`; create posts `{name,url,events,secret}`; delete `api.delete(WEBHOOK_ENDPOINTS.BY_ID(id))`.

- [ ] **Step 3: Add nav entry** — ACTIVITY group item `{ title: "Webhooks", url: "/dashboard/webhooks", icon: WebhookIcon }` + NAV_KEYS `"Webhooks": "nav.webhooks"` (i18n keys already exist).

- [ ] **Step 4: Rebuild + verify** — create a webhook pointing at `http://host.docker.internal:9` (dead) → add a memory → container log shows `Webhook ... delivery failed` warning (proves trigger fired); list/delete work.

- [ ] **Step 5: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/dashboard/src && git commit --no-verify -m "feat(dashboard): real webhooks management page"`

---


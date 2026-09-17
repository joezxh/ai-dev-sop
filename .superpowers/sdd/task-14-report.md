# Task 14 Report — Sidebar Project Context Switcher

**Status:** ✅ Complete
**Commit:** `72f7e6e1` — `feat(dashboard): project context switcher in sidebar`
**Branch:** `feat/llm-provider-i18n`

## Changes

### 1. `src/app/(root)/dashboard/components/main-nav.tsx`
- Added project selector at the bottom of `SidebarContent` (after the last nav group, before `SidebarRail`), hidden when the sidebar is collapsed.
- Loads the project list via `useApiQuery<Project[]>(PROJECT_ENDPOINTS.BASE)` (same pattern as the Memories page).
- On change: updates local state, persists to `localStorage["mem0-project-context"]` (try/catch), then `window.location.href = "/dashboard/memories"` for a full navigation so Memories picks up the restored context.
- **SSR-safe:** `useState("")` + `useEffect` to read localStorage after mount (NOT lazy `useState(() => localStorage...)`, which would throw / mismatch during server render).

### 2. `src/app/(root)/dashboard/memories/page.tsx`
- `projectId` starts as `useState("")`; a `useEffect` restores it from `localStorage["mem0-project-context"]`.
- Memories query gated with `enabled: projectContextLoaded` so the **first** fetch already carries the restored `project_id` (no wasted default-value request, no double fetch). `isLoading` additionally covers the pre-init window.
- Page-local project `<select>` behavior unchanged.

## Verification

- `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard` → `Built` / `Started`, exit 0.
- `next build` inside the image: ✓ Compiled successfully, type/lint check passed, 22/22 static pages generated, no hydration warnings.
- Container logs clean (`✓ Ready in 294ms`, no SSR errors after page requests).
- `GET :3001/dashboard/memories` → 307 (auth middleware redirect, expected) → follows to 200.

## Deviations from Brief

- Brief Step 2 suggested `useState(() => localStorage...)` lazy init; replaced with `useState("")` + `useEffect` per task constraints (SSR/hydration safety).
- Memories page uses `enabled` gating instead of a manual post-init `refetch()` — same outcome (first fetch is pre-filtered), avoids the stale-closure race where `refetch()` called synchronously after `setProjectId` would still see the old fetcher via `fetcherRef`.

## Concerns

- Memories page's own project `<select>` does NOT persist its change back to localStorage (out of scope per brief) — so changing the filter on the Memories page can leave the sidebar switcher (and next-visit context) stale. Candidate follow-up.
- Sidebar switcher uses `window.location.href` (full reload) per brief; an SPA navigation would be smoother but adds coupling.
- Switcher hidden when sidebar is collapsed (icon mode) — no touch target there; acceptable for minimal version.
- The initial render of the sidebar select may show "All projects" for one tick before the useEffect restores the stored value; purely cosmetic, no hydration mismatch.

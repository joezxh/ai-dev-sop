# Task 5 (extracted from implementation plan)

**Files:**
- Modify: `mem0/server/dashboard/package.json` (+ `react-force-graph-2d`)
- Modify: `mem0/server/dashboard/src/utils/api-endpoints.ts`, `src/types/api.ts`, `src/app/(root)/dashboard/components/main-nav.tsx`, `src/i18n/en.ts`, `src/i18n/zh.ts`

- [ ] **Step 1: Install dependency** — in `mem0/server/dashboard`: `npm install react-force-graph-2d` (canvas-only, no three.js peer needed for 2D).
- [ ] **Step 2: Endpoints** — `api-endpoints.ts`:

```ts
export const GRAPH_ENDPOINTS = {
  SEARCH: "/graph/search",
  GET_ALL: "/graph/get_all",
} as const;
```

- [ ] **Step 3: Types** — `types/api.ts`:

```ts
export interface GraphRelation {
  source: string;
  relationship: string;
  destination: string;
}

export interface GraphSearchResponse {
  relations: GraphRelation[];
}
```

- [ ] **Step 4: Nav + i18n** — ACTIVITY 组加 `{ title: "Graph", url: "/dashboard/graph", icon: <验证存在的图标，如 Share2/Waypoints> }`（先 `node -e "const l=require('lucide-react');console.log(!!l.Share2,!!l.Waypoints)"` 验证）；`NAV_KEYS` 加 `"Graph": "nav.graph"`；i18n `nav.graph`（en "Graph" / zh "图谱"）。
- [ ] **Step 5: Verify** — dashboard rebuild → Built/Started 无 error（新页面在 Task 6 创建前导航可 404，属预期）。
- [ ] **Step 6: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/dashboard && git commit --no-verify -m "feat(dashboard): graph page scaffolding (deps/endpoints/types/nav)"`

---


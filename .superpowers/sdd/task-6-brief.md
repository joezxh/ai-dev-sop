# Task 6 (extracted from implementation plan)

**Files:**
- Create: `mem0/server/dashboard/src/app/(root)/dashboard/graph/page.tsx`

- [ ] **Step 1: Create the page** — 结构（完整实现按 spec §8，核心骨架如下）：

```tsx
"use client";

import React, { useEffect, useMemo, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import ForceGraph2D from "react-force-graph-2d";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EmptyState } from "@/components/self-hosted/empty-state";
import { TableSkeleton } from "@/components/shared/table-skeleton";
import { api } from "@/utils/api";
import { GRAPH_ENDPOINTS, PROJECT_ENDPOINTS } from "@/utils/api-endpoints";
import { GraphRelation, Project } from "@/types/api";
import { useAuth } from "@/hooks/use-auth";

type GraphNode = { id: string; name: string; type?: string; mentions?: number };
type GraphLink = { source: string; target: string; relationship: string };
type GraphData = { nodes: GraphNode[]; links: GraphLink[] };

function toGraphData(relations: GraphRelation[]): GraphData {
  const nodes = new Map<string, GraphNode>();
  const links: GraphLink[] = [];
  for (const r of relations) {
    if (!r.source || !r.destination || r.source === r.destination) continue;
    if (!nodes.has(r.source)) nodes.set(r.source, { id: r.source, name: r.source });
    if (!nodes.has(r.destination)) nodes.set(r.destination, { id: r.destination, name: r.destination });
    links.push({ source: r.source, target: r.destination, relationship: r.relationship });
  }
  return { nodes: [...nodes.values()], links };
}

export default function GraphPage() {
  const { isAdmin } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [query, setQuery] = useState(searchParams.get("q") ?? "");
  const [projectId, setProjectId] = useState(searchParams.get("project") ?? "");
  const [relations, setRelations] = useState<GraphRelation[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<GraphNode | null>(null);
  const fgRef = useRef<any>(null);

  const { data: projects = [] } = useApiQueryProject(isAdmin);

  async function run(q: string) {
    setLoading(true);
    setError(null);
    try {
      const params: Record<string, unknown> = q.trim()
        ? { query: q.trim() }
        : { limit: 100 };
      if (projectId) params.project_id = projectId;
      const res = q.trim()
        ? await api.post(GRAPH_ENDPOINTS.SEARCH, params)
        : await api.get(GRAPH_ENDPOINTS.GET_ALL, { params });
      setRelations(res.data?.relations ?? []);
    } catch (e: any) {
      setError(e?.response?.data?.detail ?? "Graph query failed");
      setRelations([]);
    } finally {
      setLoading(false);
    }
  }
  // 首次进入：读取 ?q= 自动检索（useEffect + useSearchParams）

  const graphData = useMemo(
    () => toGraphData(relations ?? []),
    [relations],
  );

  const highlightSet = useMemo(() => {
    if (!selected) return null;
    const ids = new Set<string>([selected.id]);
    for (const l of graphData.links) {
      const s = typeof l.source === "object" ? (l.source as any).id : l.source;
      const t = typeof l.target === "object" ? (l.target as any).id : l.target;
      if (s === selected.id) ids.add(t);
      if (t === selected.id) ids.add(s);
    }
    return ids;
  }, [selected, graphData]);

  // 渲染：顶部检索表单（输入 + 项目下拉(admin) + 查询按钮）
  //       ForceGraph2D：nodeCanvasObject 画圆+名称；linkColor 按 highlightSet 降透明；
  //       onNodeClick setSelected；onEngineStop zoom-to-fit（fgRef.current.zoomToFit(400, 40)）
  //       右侧 Sheet/卡片：selected 的属性 + 关联边列表
  //       空态：relations 空数组 → EmptyState（提示添加记忆或调整检索）
}
```
（完整实现需补齐：检索表单 JSX、`ForceGraph2D` 的 `nodeCanvasObject`/`linkColor`/`onNodeClick`/`onNodeDragEnd`、右侧详情卡片、`useApiQueryProject` 的 admin 门控 `enabled: isAdmin`、深浅主题配色 `useTheme`、空态与 loading 骨架。）

- [ ] **Step 2: Verify** — dashboard rebuild → Built/Started；`/dashboard/graph` 登录后 200；无后端 relations 时页面 EmptyState 正常。
- [ ] **Step 3: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/dashboard && git commit --no-verify -m "feat(dashboard): graph visualization page (force-graph canvas, interactions, detail panel)"`

---


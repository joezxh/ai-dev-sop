# Task 11 (extracted from implementation plan)

**Files:**
- Modify: `mem0/server/routers/requests.py` (+ `action`, `range` params, reuse `request_action`)
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/requests/page.tsx` (+ two filter selects)

- [ ] **Step 1: Backend** — in `routers/requests.py`:

```python
from datetime import datetime, timedelta, timezone

from routers.stats import RANGE_CHOICES, RANGE_TO_HOURS, request_action
```

```python
@router.get("", response_model=list[RequestLogItem])
def list_requests(
    _auth=Depends(require_admin),
    db: Session = Depends(get_db),
    limit: int = Query(default=50, ge=1, le=200),
    action: str | None = Query(None),
    range: str = Query(default="all"),
):
    if range not in ("all",) + RANGE_CHOICES:
        raise HTTPException(status_code=400, detail="invalid range")
    stmt = select(RequestLog).where(RequestLog.auth_type.in_(API_KEY_AUTH_TYPES))
    if range != "all":
        since = datetime.now(timezone.utc) - timedelta(hours=RANGE_TO_HOURS[range])
        stmt = stmt.where(RequestLog.created_at >= since)
    logs = db.execute(stmt).scalars().all()
    if action:
        logs = [lg for lg in logs if request_action(lg.method, lg.path) == action]
    return logs
```
(add `HTTPException` to the fastapi import.)

- [ ] **Step 2: Frontend** — in `requests/page.tsx` where the query is built, pass params and add two `<select>` filters (same pattern as the Memories project filter):

```tsx
      const params: Record<string, unknown> = { limit: REQUEST_LOG_LIMIT };
      if (action) params.action = action;
      if (range !== "all") params.range = range;
      const res = await api.get(REQUEST_ENDPOINTS.BASE, { params });
```
State: `const [action, setAction] = useState(""); const [range, setRange] = useState("all");` with selects (`All actions / add / search / get_all`; `all / 24h / 7d / 30d / 90d`) calling `refetch()` on change (plus `setPage(0)` if pagination exists).

- [ ] **Step 3: Rebuild both images + verify** — `GET /requests?action=search&range=24h` → 200.

- [ ] **Step 4: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/routers/requests.py "server/dashboard/src/app/(root)/dashboard/requests" && git commit --no-verify -m "feat: requests action/range filters (server+dashboard)"`

---


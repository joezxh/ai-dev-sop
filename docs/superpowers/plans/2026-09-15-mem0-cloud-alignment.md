# Mem0 Cloud Alignment (P0+P1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align the self-hosted Mem0 with the cloud dashboard per the approved spec (`docs/superpowers/specs/2026-09-15-mem0-cloud-alignment-design.md`): P0 (nav reorg, /stats overview, project-level instructions/expiration + Danger Zone, Get Started, Playground) and P1 (Requests filters, Webhooks, JSONL exports, context switcher).

**Architecture:** FastAPI backend (`mem0/server`) — new thin routers + project fields + one self-healing write path; Next.js frontend (`mem0/server/dashboard`) — new pages reusing `useApiQuery`/shadcn patterns. Tests are unit tests run inside the api container (established workflow: `docker compose cp` + `pytest`), commits happen inside the `mem0` git submodule.

**Tech Stack:** FastAPI/SQLAlchemy/pydantic; Next.js/React/Tailwind/shadcn; pytest; httpx (webhooks).

**Test commands** (from `d:/projects/ai-dev-sop`):
- Copy a test file into the running container and run it:
  `docker compose -f deploy/mem0/docker-compose.yaml cp mem0/server/tests/<file>.py mem0-api:/app/tests/ ; docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/<file>.py -q`
- Full suite: `docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/ -q` (expect **27 passed** before this plan's changes).

**Commit convention:** inside the submodule — `cd d:/projects/ai-dev-sop/mem0 && git add <files> && git commit --no-verify -m "<msg>"`.

---

## File Structure Map

| File | Responsibility |
| --- | --- |
| `mem0/server/routers/stats.py` | NEW — `GET /stats` aggregation (requests/memories/entities) |
| `mem0/server/routers/webhooks.py` | NEW — webhook CRUD router (P1) |
| `mem0/server/webhooks_client.py` | NEW — fire-and-forget delivery + event mapping (P1) |
| `mem0/server/models.py` | + `Project.custom_instructions`, `Project.memory_expiration_date`, `Webhook` |
| `mem0/server/main.py` | + include stats router; `add_memory` project defaults & lost-params fix; `DELETE /memories?project_id`; `GET /exports/memories`; webhook triggers |
| `mem0/server/routers/projects.py` | accept/return the two new project fields |
| `mem0/server/routers/requests.py` | + `action`/`range` filters (P1) |
| `mem0/server/tests/test_stats.py`, `test_project_defaults.py`, `test_delete_project_memories.py`, `test_webhooks.py`, `test_exports.py` | NEW tests |
| `mem0/server/dashboard/src/app/(root)/dashboard/components/main-nav.tsx` | nav reorg (SETUP/ACTIVITY/TENANT/ACCOUNT), new entries |
| `mem0/server/dashboard/src/app/(root)/dashboard/overview/page.tsx` | NEW — Dashboard 总览 |
| `mem0/server/dashboard/src/app/(root)/dashboard/get-started/page.tsx` | NEW — 接入向导 |
| `mem0/server/dashboard/src/app/(root)/dashboard/playground/page.tsx` | NEW — 调试沙盒 |
| `mem0/server/dashboard/src/app/(root)/dashboard/projects/page.tsx` | + custom instructions / expiration / Danger Zone |
| `mem0/server/dashboard/src/app/(root)/dashboard/webhooks/page.tsx` | replace stub with real CRUD (P1) |
| `mem0/server/dashboard/src/app/(root)/dashboard/memory-exports/page.tsx` | NEW (P1), replaces old `export` stub |
| DELETE: `.../categories/page.tsx`, `.../analytics/page.tsx`, `.../export/page.tsx` | remove cloud-upsell stubs |
| `mem0/server/dashboard/src/utils/api-endpoints.ts` | + STATS / WEBHOOK / EXPORT endpoints |

---

### Task 1: `GET /stats` aggregation endpoint (TDD)

**Files:**
- Create: `mem0/server/routers/stats.py`
- Create: `mem0/server/tests/test_stats.py`
- Modify: `mem0/server/main.py` (import + include router, next to the other routers)

- [ ] **Step 1: Write the failing test** — `mem0/server/tests/test_stats.py`:

```python
from fastapi import HTTPException
import pytest

from routers import stats as stats_router


class _Log:
    def __init__(self, method, path):
        self.method = method
        self.path = path


class _Scalars:
    def __init__(self, items):
        self._items = items

    def all(self):
        return self._items


class _Result:
    def __init__(self, items):
        self._items = items

    def scalars(self):
        return _Scalars(self._items)


class _DB:
    def __init__(self, logs):
        self._logs = logs

    def execute(self, *_a, **_k):
        return _Result(self._logs)


class _Row:
    def __init__(self, payload):
        self.payload = payload


class _VS:
    def list(self, top_k):
        return [[_Row({"user_id": "u1", "agent_id": "a1"}), _Row({"user_id": "u1"})]]


class _Mem:
    vector_store = _VS()


def test_request_action_mapping():
    assert stats_router.request_action("POST", "/memories") == "add"
    assert stats_router.request_action("POST", "/search") == "search"
    assert stats_router.request_action("GET", "/memories") == "get_all"
    assert stats_router.request_action("GET", "/memories/123") is None
    assert stats_router.request_action("POST", "/configure") is None


def test_get_stats_counts_and_entities(monkeypatch):
    logs = [
        _Log("POST", "/memories"),
        _Log("POST", "/search"),
        _Log("GET", "/memories"),
        _Log("GET", "/requests"),  # not counted
    ]
    monkeypatch.setattr(stats_router, "get_memory_instance", lambda: _Mem())
    out = stats_router.get_stats(range="24h", _admin=None, db=_DB(logs))
    assert out.requests == {"add": 1, "search": 1, "get_all": 1, "total": 3}
    assert out.memories_total == 2
    assert out.entities == {"users": 1, "agents": 1, "runs": 0}


def test_get_stats_rejects_bad_range():
    with pytest.raises(HTTPException) as exc:
        stats_router.get_stats(range="1h", _admin=None, db=_DB([]))
    assert exc.value.status_code == 400
```

- [ ] **Step 2: Run to verify it fails**

```
docker compose -f deploy/mem0/docker-compose.yaml cp mem0/server/tests/test_stats.py mem0-api:/app/tests/ ; docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/test_stats.py -q
```
Expected: FAIL (`ModuleNotFoundError: routers.stats` — copy of routers/stats.py not yet made).

- [ ] **Step 3: Implement** — `mem0/server/routers/stats.py`:

```python
from datetime import datetime, timedelta, timezone

from auth import require_admin
from db import get_db
from fastapi import APIRouter, Depends, HTTPException, Query
from models import RequestLog
from pydantic import BaseModel
from sqlalchemy import select
from sqlalchemy.orm import Session
from server_state import get_memory_instance

router = APIRouter(prefix="/stats", tags=["stats"])

RANGE_CHOICES = ("24h", "7d", "30d", "90d", "all")
RANGE_TO_HOURS = {"24h": 24, "7d": 168, "30d": 720, "90d": 2160}
API_KEY_AUTH_TYPES = ("api_key", "admin_api_key")


def request_action(method: str, path: str) -> str | None:
    """Map a request-log row to a dashboard action bucket."""
    p = (path or "").rstrip("/")
    if method == "POST" and p == "/memories":
        return "add"
    if method == "POST" and p == "/search":
        return "search"
    if method == "GET" and p == "/memories":
        return "get_all"
    return None


def _iter_memory_payloads(limit: int = 10_000) -> list[dict]:
    results = get_memory_instance().vector_store.list(top_k=limit)
    rows = results[0] if results and isinstance(results, list) and isinstance(results[0], list) else results or []
    return [getattr(row, "payload", None) or {} for row in rows]


class StatsResponse(BaseModel):
    requests: dict
    memories_total: int
    entities: dict


@router.get("", response_model=StatsResponse)
def get_stats(
    range: str = Query(default="7d"),
    _admin=Depends(require_admin),
    db: Session = Depends(get_db),
):
    if range not in RANGE_CHOICES:
        raise HTTPException(status_code=400, detail=f"range must be one of {RANGE_CHOICES}")
    stmt = select(RequestLog).where(RequestLog.auth_type.in_(API_KEY_AUTH_TYPES))
    if range != "all":
        since = datetime.now(timezone.utc) - timedelta(hours=RANGE_TO_HOURS[range])
        stmt = stmt.where(RequestLog.created_at >= since)
    logs = db.execute(stmt).scalars().all()
    counts = {"add": 0, "search": 0, "get_all": 0}
    for log in logs:
        action = request_action(log.method, log.path)
        if action:
            counts[action] += 1

    payloads = _iter_memory_payloads()
    entities: dict[str, set] = {"users": set(), "agents": set(), "runs": set()}
    for payload in payloads:
        for field, key in (("user_id", "users"), ("agent_id", "agents"), ("run_id", "runs")):
            if payload.get(field):
                entities[key].add(str(payload[field]))

    return StatsResponse(
        requests={**counts, "total": sum(counts.values())},
        memories_total=len(payloads),
        entities={k: len(v) for k, v in entities.items()},
    )
```

- [ ] **Step 4: Register in `main.py`** — add import + include:

```python
from routers import stats as stats_router
```
(after the other `from routers import ...` lines) and inside the include block after `app.include_router(users_router.router)`:

```python
app.include_router(stats_router.router)
```

- [ ] **Step 5: Run to verify pass** — copy both files, run `pytest tests/test_stats.py -q`. Expected: `3 passed`.

- [ ] **Step 6: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/routers/stats.py server/tests/test_stats.py server/main.py && git commit --no-verify -m "feat(server): add GET /stats aggregation endpoint"`

---

### Task 2: Project-level instructions/expiration + lost write-params fix (TDD)

**Files:**
- Modify: `mem0/server/models.py` (Project: two new columns)
- Modify: `mem0/server/main.py` (`_apply_project_defaults`, `_load_project`, `add_memory` params fix)
- Modify: `mem0/server/routers/projects.py` (accept/return new fields)
- Create: `mem0/server/tests/test_project_defaults.py`

Note: the current `add_memory` builds `params` manually and **lost** `infer`/`prompt`/`expiration_date`/`memory_type` passthrough — this task fixes that regression together with the new project defaults.

- [ ] **Step 1: Write the failing test** — `mem0/server/tests/test_project_defaults.py`:

```python
from types import SimpleNamespace

from main import _apply_project_defaults


def _mc(prompt=None, expiration_date=None):
    return SimpleNamespace(prompt=prompt, expiration_date=expiration_date)


def _proj(instructions=None, expiration=None):
    return SimpleNamespace(custom_instructions=instructions, memory_expiration_date=expiration)


def test_no_project_no_overrides():
    assert _apply_project_defaults(_mc(), None) == {}


def test_project_defaults_applied():
    p = _proj(instructions="Only store tech preferences", expiration=__import__("datetime").date(2027, 1, 1))
    out = _apply_project_defaults(_mc(), p)
    assert out["prompt"] == "Only store tech preferences"
    assert out["expiration_date"] == "2027-01-01"


def test_explicit_request_values_win():
    p = _proj(instructions="proj", expiration=__import__("datetime").date(2027, 1, 1))
    out = _apply_project_defaults(_mc(prompt="explicit", expiration_date="2030-05-05"), p)
    assert out == {}
```

- [ ] **Step 2: Run to verify it fails** (copy + pytest). Expected: FAIL (`ImportError: cannot import name '_apply_project_defaults'`).

- [ ] **Step 3: Add columns** — in `mem0/server/models.py`, extend imports (`from datetime import date` at top; add `Date` to the sqlalchemy import) and add to `Project`:

```python
    custom_instructions: Mapped[str | None] = mapped_column(Text, nullable=True)
    memory_expiration_date: Mapped[date | None] = mapped_column(Date, nullable=True)
```
(The tables are (re)created by the startup `create_all` only when missing — for the existing `projects` table add an idempotent ALTER to `db.ensure_tenant_schema`:)

```python
        conn.execute(text("ALTER TABLE projects ADD COLUMN IF NOT EXISTS custom_instructions TEXT NULL"))
        conn.execute(text("ALTER TABLE projects ADD COLUMN IF NOT EXISTS memory_expiration_date DATE NULL"))
```

- [ ] **Step 4: Implement in `main.py`** — add helpers before `add_memory`:

```python
def _load_project(project_id: str):
    from sqlalchemy import select as _select
    from models import Project

    with SessionLocal() as session:
        return session.scalar(_select(Project).where(Project.project_id == project_id))


def _apply_project_defaults(memory_create, project) -> Dict[str, Any]:
    """Project-scoped defaults: custom extraction prompt and expiration date.
    Explicit per-request values always win."""
    overrides: Dict[str, Any] = {}
    if project is None:
        return overrides
    if not memory_create.prompt and project.custom_instructions:
        overrides["prompt"] = project.custom_instructions
    if memory_create.expiration_date is None and project.memory_expiration_date is not None:
        overrides["expiration_date"] = project.memory_expiration_date.isoformat()
    return overrides
```

Then in `add_memory`, after the existing `if project_id: metadata["project_id"] = project_id` line, add:

```python
    overrides = _apply_project_defaults(memory_create, _load_project(project_id) if project_id else None)
```

and replace the `params = {...}` block with:

```python
    params = {k: v for k, v in {
        "user_id": owner_user_id,
        "agent_id": owner_agent_id,
        "run_id": owner_run_id,
        "metadata": metadata,
        "infer": True if memory_create.infer is None else memory_create.infer,
        "memory_type": memory_create.memory_type,
        "prompt": memory_create.prompt or overrides.get("prompt"),
        "expiration_date": memory_create.expiration_date or overrides.get("expiration_date"),
    }.items() if v is not None}
```

- [ ] **Step 5: Expose fields in `routers/projects.py`** — add to `ProjectCreate` and `ProjectUpdate`:

```python
    custom_instructions: str | None = None
    memory_expiration_date: str | None = None  # YYYY-MM-DD
```
In `_to_response` add:

```python
        custom_instructions=project.custom_instructions,
        memory_expiration_date=project.memory_expiration_date.isoformat() if project.memory_expiration_date else None,
```
In `create_project` / `update_project` apply them (parse date with `date.fromisoformat(v)` guarded by try/except ValueError → 400). Add `from datetime import date` import.

- [ ] **Step 6: Run tests** — copy test file + rebuilt files, run `pytest tests/test_project_defaults.py tests/test_multitenant.py tests/test_configure.py -q`. Expected: all pass (multitenant regression guards the rewrite).

- [ ] **Step 7: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/models.py server/db.py server/main.py server/routers/projects.py server/tests/test_project_defaults.py && git commit --no-verify -m "feat(server): project-level custom instructions + expiration defaults; restore write params"`

---

### Task 3: Danger Zone — `DELETE /memories?project_id` (TDD)

**Files:**
- Modify: `mem0/server/main.py` (new endpoint after `get_all_memories`)
- Create: `mem0/server/tests/test_delete_project_memories.py`

- [ ] **Step 1: Write the failing test** — `mem0/server/tests/test_delete_project_memories.py`:

```python
from types import SimpleNamespace

import pytest
from fastapi import HTTPException

import main as m


def test_requires_project_id():
    with pytest.raises(HTTPException) as exc:
        m.delete_memories_by_project(project_id=None, _auth=None)
    assert exc.value.status_code == 400


def test_rejects_non_admin():
    user = SimpleNamespace(role="user")
    with pytest.raises(HTTPException) as exc:
        m.delete_memories_by_project(project_id="p", _auth=user)
    assert exc.value.status_code == 403


def test_404_when_project_missing(monkeypatch):
    monkeypatch.setattr(m, "_load_project", lambda pid: None)
    with pytest.raises(HTTPException) as exc:
        m.delete_memories_by_project(project_id="ghost", _auth=None)
    assert exc.value.status_code == 404


def test_deletes_only_matching(monkeypatch):
    monkeypatch.setattr(m, "_load_project", lambda pid: SimpleNamespace(project_id=pid))
    monkeypatch.setattr(
        m, "_list_all_memories",
        lambda limit: {"results": [
            {"id": "1", "metadata": {"project_id": "p"}},
            {"id": "2", "metadata": {"project_id": "other"}},
            {"id": "3", "metadata": {}},
        ]},
    )
    deleted_ids = []
    inst = SimpleNamespace(delete=lambda mid: deleted_ids.append(mid))
    monkeypatch.setattr(m, "get_memory_instance", lambda: inst)
    out = m.delete_memories_by_project(project_id="p", _auth=None)
    assert out == {"deleted": 1}
    assert deleted_ids == ["1"]
```

- [ ] **Step 2: Run to verify it fails** (copy + pytest). Expected: FAIL (`AttributeError: module 'main' has no attribute 'delete_memories_by_project'`).

- [ ] **Step 3: Implement** — add to `main.py` after `get_all_memories`:

```python
@app.delete("/memories", summary="Delete all memories of a project (admin)")
def delete_memories_by_project(
    project_id: Optional[str] = Query(None),
    _auth=Depends(verify_auth),
):
    """Danger-Zone: delete every memory whose metadata.project_id matches."""
    if _auth is not None and _auth.role != "admin":
        raise HTTPException(status_code=403, detail="Admin role required.")
    if not project_id:
        raise HTTPException(status_code=400, detail="project_id is required.")
    if _load_project(project_id) is None:
        raise HTTPException(status_code=404, detail="Project not found.")
    data = _list_all_memories(limit=ALL_MEMORIES_LIMIT)
    ids = [
        r["id"]
        for r in data.get("results", [])
        if (r.get("metadata") or {}).get("project_id") == project_id
    ]
    deleted = 0
    for memory_id in ids:
        try:
            get_memory_instance().delete(memory_id)
            deleted += 1
        except Exception:
            logging.exception("Failed to delete memory %s", memory_id)
    return {"deleted": deleted}
```

- [ ] **Step 4: Run to verify pass.** Expected: `4 passed`.

- [ ] **Step 5: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/main.py server/tests/test_delete_project_memories.py && git commit --no-verify -m "feat(server): project-scoped delete-all memories endpoint"`

---

### Task 4: Webhooks backend (TDD) — P1

**Files:**
- Modify: `mem0/server/models.py` (+ `Webhook`)
- Create: `mem0/server/webhooks_client.py`
- Create: `mem0/server/routers/webhooks.py`
- Modify: `mem0/server/main.py` (include router; trigger after add/update/delete)
- Create: `mem0/server/tests/test_webhooks.py`

- [ ] **Step 1: Write the failing test** — `mem0/server/tests/test_webhooks.py`:

```python
from types import SimpleNamespace

from webhooks_client import events_from_add_response, trigger_webhooks


def test_events_from_add_response():
    results = [{"event": "ADD"}, {"event": "UPDATE"}, {"event": "NOOP"}]
    assert events_from_add_response(results) == {"memory_added", "memory_updated"}
    assert events_from_add_response([]) == set()


def test_trigger_filters_by_subscription(monkeypatch):
    calls = []

    def fake_post(url, json=None, headers=None, timeout=None):
        calls.append((url, json))

    monkeypatch.setattr("webhooks_client.httpx.post", fake_post)
    hooks = [
        SimpleNamespace(name="all", url="http://x/1", events=["memory_added"], secret="s1"),
        SimpleNamespace(name="other", url="http://x/2", events=["memory_deleted"], secret=None),
    ]
    trigger_webhooks(hooks, "memory_added", {"id": "1"})
    assert [(u, j) for u, j in calls] == [("http://x/1", {"event": "memory_added", "data": {"id": "1"}})]


def test_trigger_never_raises(monkeypatch):
    def boom(*a, **k):
        raise RuntimeError("down")

    monkeypatch.setattr("webhooks_client.httpx.post", boom)
    hook = SimpleNamespace(name="h", url="http://x", events=["memory_added"], secret=None)
    trigger_webhooks([hook], "memory_added", {})  # must not raise
```

- [ ] **Step 2: Run to verify it fails** (`ModuleNotFoundError: webhooks_client`).

- [ ] **Step 3: Implement `webhooks_client.py`**:

```python
import logging

import httpx

_EVENT_MAP = {"ADD": "memory_added", "UPDATE": "memory_updated", "DELETE": "memory_deleted"}


def events_from_add_response(results: list) -> set[str]:
    events = set()
    for item in results or []:
        mapped = _EVENT_MAP.get((item or {}).get("event"))
        if mapped:
            events.add(mapped)
    return events


def trigger_webhooks(hooks, event: str, payload: dict) -> None:
    """Fire-and-forget delivery; never raises, never blocks the caller long."""
    for hook in hooks or []:
        if event not in (hook.events or []):
            continue
        headers = {"Content-Type": "application/json"}
        if hook.secret:
            headers["X-Webhook-Secret"] = hook.secret
        try:
            httpx.post(hook.url, json={"event": event, "data": payload}, headers=headers, timeout=5.0)
        except Exception:  # noqa: BLE001
            logging.warning("Webhook %s delivery failed for %s", getattr(hook, "name", "?"), event)
```

- [ ] **Step 4: Add model** — in `models.py`:

```python
class Webhook(Base):
    __tablename__ = "webhooks"

    id: Mapped[uuid.UUID] = mapped_column(primary_key=True, default=_new_uuid)
    name: Mapped[str] = mapped_column(String(255))
    url: Mapped[str] = mapped_column(Text)
    events: Mapped[list] = mapped_column(JSON, default=list)
    secret: Mapped[str | None] = mapped_column(String(255), nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=_utcnow)
```

- [ ] **Step 5: Implement router** — `routers/webhooks.py`:

```python
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from sqlalchemy import select
from sqlalchemy.orm import Session

from auth import require_admin
from db import get_db
from models import Webhook

router = APIRouter(prefix="/webhooks", tags=["webhooks"])

ALLOWED_EVENTS = ("memory_added", "memory_updated", "memory_deleted")


class WebhookCreate(BaseModel):
    name: str
    url: str
    events: list[str]
    secret: str | None = None


class WebhookResponse(BaseModel):
    id: str
    name: str
    url: str
    events: list[str]
    secret: str | None
    created_at: str


def _to_response(hook: Webhook) -> WebhookResponse:
    return WebhookResponse(
        id=str(hook.id), name=hook.name, url=hook.url,
        events=list(hook.events or []), secret=hook.secret,
        created_at=hook.created_at.isoformat() if hook.created_at else "",
    )


@router.get("", response_model=list[WebhookResponse])
def list_webhooks(_admin=Depends(require_admin), db: Session = Depends(get_db)):
    hooks = db.execute(select(Webhook).order_by(Webhook.created_at)).scalars().all()
    return [_to_response(h) for h in hooks]


@router.post("", response_model=WebhookResponse, status_code=201)
def create_webhook(body: WebhookCreate, _admin=Depends(require_admin), db: Session = Depends(get_db)):
    bad = [e for e in body.events if e not in ALLOWED_EVENTS]
    if bad:
        raise HTTPException(status_code=400, detail=f"Unknown events: {bad}. Allowed: {ALLOWED_EVENTS}")
    hook = Webhook(name=body.name, url=body.url, events=body.events, secret=body.secret)
    db.add(hook)
    db.commit()
    db.refresh(hook)
    return _to_response(hook)


@router.delete("/{webhook_id}")
def delete_webhook(webhook_id: str, _admin=Depends(require_admin), db: Session = Depends(get_db)):
    import uuid as _uuid
    try:
        hook = db.get(Webhook, _uuid.UUID(webhook_id))
    except (TypeError, ValueError):
        hook = None
    if hook is None:
        raise HTTPException(status_code=404, detail="Webhook not found.")
    db.delete(hook)
    db.commit()
    return {"message": "Webhook deleted."}
```

- [ ] **Step 6: Wire triggers in `main.py`** — import + include router; add helper and call sites:

```python
from routers import webhooks as webhooks_router
from webhooks_client import events_from_add_response, trigger_webhooks
```

```python
def _fire_webhooks(results: list) -> None:
    """Deliver memory events to subscribed webhooks off-thread."""
    events = events_from_add_response(results)
    if not events:
        return
    import threading

    from db import SessionLocal as _SL
    from models import Webhook as _Webhook
    from sqlalchemy import select as _select

    def _run() -> None:
        try:
            with _SL() as session:
                hooks = session.execute(_select(_Webhook)).scalars().all()
                for event in events:
                    trigger_webhooks(hooks, event, {"results": results})
        except Exception:
            logging.exception("Webhook dispatch failed")

    threading.Thread(target=_run, daemon=True).start()
```

Call `_fire_webhooks(response.get("results") or [])` in `add_memory` right after the successful `response = ...` line. In `update_memory` call `_fire_webhooks([{"event": "UPDATE"}])`, in `delete_memory` `_fire_webhooks([{"event": "DELETE"}])` after their successful mem0 calls.

- [ ] **Step 7: Run tests + regression.** `pytest tests/test_webhooks.py tests/ -q` — expect all pass.

- [ ] **Step 8: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/models.py server/webhooks_client.py server/routers/webhooks.py server/main.py server/tests/test_webhooks.py && git commit --no-verify -m "feat(server): webhooks CRUD + fire-and-forget memory event delivery"`

---

### Task 5: JSONL export endpoint (TDD) — P1

**Files:**
- Modify: `mem0/server/main.py` (+ endpoint & pure filter helper)
- Create: `mem0/server/tests/test_exports.py`

- [ ] **Step 1: Write the failing test** — `mem0/server/tests/test_exports.py`:

```python
from main import _filter_memories_for_export


def _row(id, project_id=None, user_id=None, created_at=None):
    return {
        "id": id,
        "user_id": user_id,
        "metadata": {"project_id": project_id} if project_id else {},
        "created_at": created_at,
    }


ROWS = [
    _row("1", project_id="p", user_id="u1", created_at="2026-09-01T00:00:00"),
    _row("2", project_id="p", user_id="u2", created_at="2026-09-10T00:00:00"),
    _row("3", user_id="u1", created_at="2026-09-05T00:00:00"),
]


def test_filter_by_project():
    assert [r["id"] for r in _filter_memories_for_export(ROWS, project_id="p")] == ["1", "2"]


def test_filter_by_user_and_date():
    out = _filter_memories_for_export(ROWS, user_id="u1", start="2026-09-03", end="2026-09-06")
    assert [r["id"] for r in out] == ["3"]


def test_no_filters_returns_all():
    assert len(_filter_memories_for_export(ROWS)) == 3
```

- [ ] **Step 2: Run to verify it fails.**

- [ ] **Step 3: Implement in `main.py`** (helper + endpoint):

```python
def _filter_memories_for_export(
    rows: list,
    project_id: Optional[str] = None,
    user_id: Optional[str] = None,
    start: Optional[str] = None,
    end: Optional[str] = None,
) -> list:
    def _ts(row: dict):
        try:
            return datetime.fromisoformat(str(row.get("created_at")).replace("Z", "+00:00"))
        except (TypeError, ValueError):
            return None

    start_dt = datetime.fromisoformat(start).replace(tzinfo=timezone.utc) if start else None
    end_dt = datetime.fromisoformat(end).replace(tzinfo=timezone.utc) if end else None
    out = []
    for row in rows:
        if project_id and (row.get("metadata") or {}).get("project_id") != project_id:
            continue
        if user_id and row.get("user_id") != user_id:
            continue
        ts = _ts(row)
        if start_dt and (ts is None or ts < start_dt):
            continue
        if end_dt and (ts is None or ts > end_dt):
            continue
        out.append(row)
    return out


@app.get("/exports/memories", summary="Export memories as JSONL (admin)")
def export_memories(
    project_id: Optional[str] = None,
    user_id: Optional[str] = None,
    start: Optional[str] = None,
    end: Optional[str] = None,
    format: str = "jsonl",
    _auth=Depends(verify_auth),
):
    if _auth is not None and _auth.role != "admin":
        raise HTTPException(status_code=403, detail="Admin role required.")
    if format != "jsonl":
        raise HTTPException(status_code=400, detail="Only format=jsonl is supported.")
    data = _list_all_memories(limit=ALL_MEMORIES_LIMIT)
    rows = _filter_memories_for_export(data.get("results", []), project_id, user_id, start, end)

    def _gen():
        for row in rows:
            yield (json.dumps(row, ensure_ascii=False, default=str) + "\n")

    filename = f"memories-export-{int(time.time())}.jsonl"
    return StreamingResponse(
        _gen(),
        media_type="application/x-ndjson",
        headers={"Content-Disposition": f'attachment; filename="{filename}"'},
    )
```
Add missing imports in `main.py` if absent: `import json`, `import time`, `from datetime import datetime, timezone`, `from fastapi.responses import StreamingResponse`.

- [ ] **Step 4: Run to verify pass.** Expected: `3 passed`.

- [ ] **Step 5: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/main.py server/tests/test_exports.py && git commit --no-verify -m "feat(server): JSONL memory export with project/user/date filters"`

---

### Task 6: Nav reorg + remove cloud-upsell stubs (P0)

**Files:**
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/components/main-nav.tsx`
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/api-keys/page.tsx`, `memories/page.tsx`, `configuration/page.tsx` (remove UpgradeBanner blocks)
- Delete: `.../dashboard/categories/page.tsx`, `.../dashboard/analytics/page.tsx`, `.../dashboard/export/page.tsx`

- [ ] **Step 1: Delete stub pages**:
```
cd d:/projects/ai-dev-sop/mem0/server/dashboard/src/app/(root)/dashboard
git rm -r categories analytics export
```

- [ ] **Step 2: Rewrite `main-nav.tsx` groups.** Replace the entire `SidebarContent` children with four groups (keep the existing `SidebarMenuItem`/`SidebarMenuButton` rendering pattern used in the file). Group data:

```tsx
// SETUP
{[
  { title: "Get Started", url: "/dashboard/get-started", icon: Rocket },
  { title: "Playground", url: "/dashboard/playground", icon: FlaskConical },
  { title: "API Keys", url: "/dashboard/api-keys", icon: KeyRound },
]}
// ACTIVITY
{[
  { title: "Dashboard", url: "/dashboard/overview", icon: ChartLine },
  { title: "Requests", url: "/dashboard/requests", icon: Activity },
  { title: "Entities", url: "/dashboard/entities", icon: Users },
  { title: "Memories", url: "/dashboard/memories", icon: GalleryVerticalEnd },
]}
// TENANT (unchanged)
{[ Users / Departments / Projects items as-is ]}
// ACCOUNT
{[
  { title: "Webhooks", url: "/dashboard/webhooks", icon: WebhookIcon },      // added in Task 12 — add entry there
  { title: "Exports", url: "/dashboard/memory-exports", icon: FolderInput }, // added in Task 13 — add entry there
  { title: "Configuration", url: "/dashboard/configuration", icon: Wrench },
  { title: "Settings", url: "/dashboard/settings", icon: Settings },
]}
```
Remove: the `CLOUD FEATURES` Collapsible block and its PRO badges, the old `Requests` entry duplication, `Tags`/`FolderInput` imports if unused, add `Rocket`, `FlaskConical` to the lucide import. NAV_KEYS additions:

```tsx
  "Get Started": "nav.getStarted",
  Playground: "nav.playground",
  Dashboard: "nav.dashboardOverview",
  Exports: "nav.exports",
```
and to `i18n/en.ts` + `i18n/zh.ts` `nav` sections:
```ts
// en
getStarted: "Get Started", playground: "Playground", dashboardOverview: "Dashboard", exports: "Exports",
// zh
getStarted: "接入", playground: "调试沙盒", dashboardOverview: "总览", exports: "导出",
```

- [ ] **Step 3: Remove UpgradeBanner blocks** — delete the `{memories.length >= MEMORY_FETCH_LIMIT && (<UpgradeBanner .../>)}` block in `memories/page.tsx`, the `{keys.length >= 3 && (<UpgradeBanner .../>)}` block in `api-keys/page.tsx`, and the `<UpgradeBanner id="config-sso" ... />` element in `configuration/page.tsx` (also drop now-unused imports).

- [ ] **Step 4: Rebuild dashboard** — `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard`. Expected: `Built` + `Started`, no type errors (nav references `/dashboard/webhooks` & `/dashboard/memory-exports` only after Tasks 12/13 — **do not add those two entries yet**).

- [ ] **Step 5: Verify** — `GET http://localhost:3001/dashboard/overview` etc. will 404 until Tasks 7–9; check `http://localhost:3001/dashboard/api-keys` → 200.

- [ ] **Step 6: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add -A server/dashboard/src server/dashboard && git commit --no-verify -m "feat(dashboard): reorganize nav into SETUP/ACTIVITY/TENANT/ACCOUNT; drop cloud-upsell stubs"`

---

### Task 7: Dashboard overview page (P0)

**Files:**
- Create: `mem0/server/dashboard/src/app/(root)/dashboard/overview/page.tsx`
- Modify: `mem0/server/dashboard/src/utils/api-endpoints.ts` (+ `STATS_ENDPOINT`)

- [ ] **Step 1: Add endpoint** — in `api-endpoints.ts` next to `REQUEST_ENDPOINTS`:

```ts
export const STATS_ENDPOINT = {
  BASE: "/stats",
} as const;
```

- [ ] **Step 2: Create the page**:

```tsx
"use client";

import React, { useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { TableSkeleton } from "@/components/shared/table-skeleton";
import { api } from "@/utils/api";
import { STATS_ENDPOINT } from "@/utils/api-endpoints";
import { useApiQuery } from "@/hooks/use-api-query";

const RANGES = ["24h", "7d", "30d", "90d", "all"] as const;

type Stats = {
  requests: { add: number; search: number; get_all: number; total: number };
  memories_total: number;
  entities: { users: number; agents: number; runs: number };
};

export default function OverviewPage() {
  const [range, setRange] = useState<(typeof RANGES)[number]>("7d");
  const { data: stats, isLoading } = useApiQuery<Stats | null>(
    async () => {
      const res = await api.get(STATS_ENDPOINT.BASE, { params: { range } });
      return res.data ?? null;
    },
    { errorToast: "Failed to load stats", initialData: null },
  );

  const cards = stats
    ? [
        { title: "Total Memories", value: stats.memories_total, href: "/dashboard/memories" },
        { title: "Add Requests", value: stats.requests.add, href: "/dashboard/requests" },
        { title: "Search Requests", value: stats.requests.search, href: "/dashboard/requests" },
        { title: "Get All Requests", value: stats.requests.get_all, href: "/dashboard/requests" },
        { title: "Users", value: stats.entities.users, href: "/dashboard/entities" },
        { title: "Agents", value: stats.entities.agents, href: "/dashboard/entities" },
      ]
    : [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold font-fustat">Dashboard</h1>
        <div className="flex gap-1">
          {RANGES.map((r) => (
            <Button
              key={r}
              size="sm"
              variant={r === range ? "default" : "outline"}
              onClick={() => setRange(r)}
            >
              {r}
            </Button>
          ))}
        </div>
      </div>
      {isLoading ? (
        <TableSkeleton rows={3} columns={3} />
      ) : (
        <div className="grid grid-cols-2 gap-4 md:grid-cols-3">
          {cards.map((c) => (
            <Link key={c.title} href={c.href}>
              <Card className="border-memBorder-primary hover:bg-surface-default-secondary">
                <CardHeader>
                  <CardTitle className="text-xs text-onSurface-default-tertiary">{c.title}</CardTitle>
                </CardHeader>
                <CardContent className="text-2xl font-semibold">{c.value}</CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 3: Rebuild + verify** — dashboard rebuild; `GET /dashboard/overview` → 200; `GET http://localhost:8888/stats?range=7d` → 200 JSON.

- [ ] **Step 4: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/dashboard/src && git commit --no-verify -m "feat(dashboard): overview page backed by /stats"`

---

### Task 8: Get Started page (P0)

**Files:**
- Create: `mem0/server/dashboard/src/app/(root)/dashboard/get-started/page.tsx`

- [ ] **Step 1: Create the page** — three code tabs (MCP JSON for CodeBuddy/Qoder, Python, cURL) with `YOUR_API_KEY` / `git remote -v` placeholders and a 4-step layout:

```tsx
"use client";

import React, { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import Link from "next/link";

const MCP_JSON = `{
  "mcpServers": {
    "mem0-local": {
      "url": "http://<your-server>:8080/mcp"
    }
  }
}
// 工具调用入参：
//   api_key: 你的个人 API Key（/dashboard/api-keys 创建）
//   git_remote: 本地工程 git remote -v 输出的地址（自动匹配项目）`;

const PYTHON = `from openai import OpenAI  # 任意 HTTP 客户端均可
import requests

BASE = "http://<your-server>:8888"
KEY = "YOUR_API_KEY"

# 1) 添加记忆（git_remote 自动解析 projectId 写入 metadata）
requests.post(f"{BASE}/memories", headers={"X-API-Key": KEY}, json={
    "messages": [{"role": "user", "content": "本项目使用 FastAPI + pgvector"}],
    "git_remote": "git@github.com:org/repo.git",
})

# 2) 检索记忆
r = requests.post(f"{BASE}/search", headers={"X-API-Key": KEY},
                  json={"query": "本项目用什么数据库？"})
print(r.json())`;

const CURL = `# 1) 添加记忆
curl -X POST http://<your-server>:8888/memories \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"messages":[{"role":"user","content":"我喜欢深色主题"}],"git_remote":"git@github.com:org/repo.git"}'

# 2) 检索记忆
curl -X POST http://<your-server>:8888/search \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"query":"主题偏好"}'`;

const TABS = [
  { id: "mcp", label: "MCP（CodeBuddy / Qoder）", code: MCP_JSON },
  { id: "python", label: "Python", code: PYTHON },
  { id: "curl", label: "cURL", code: CURL },
] as const;

export default function GetStartedPage() {
  const [tab, setTab] = useState<(typeof TABS)[number]["id"]>("mcp");
  const active = TABS.find((t) => t.id === tab)!;

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold font-fustat">Get Started</h1>

      <Card className="border-memBorder-primary">
        <CardHeader><CardTitle className="text-sm">Step 1 · 创建 API Key</CardTitle></CardHeader>
        <CardContent className="text-sm">
          在 <Link className="underline" href="/dashboard/api-keys">API Keys</Link> 页创建个人密钥（仅创建时可见）。
        </CardContent>
      </Card>

      <Card className="border-memBorder-primary">
        <CardHeader><CardTitle className="text-sm">Step 2 · 选择接入方式</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <div className="flex gap-2">
            {TABS.map((t) => (
              <button
                key={t.id}
                onClick={() => setTab(t.id)}
                className={`px-3 py-1.5 text-sm rounded border ${t.id === tab ? "bg-memBrand-primary text-white" : "border-memBorder-primary"}`}
              >
                {t.label}
              </button>
            ))}
          </div>
          <pre className="text-xs bg-surface-default-secondary p-3 rounded font-mono overflow-x-auto max-h-96">
            {active.code}
          </pre>
        </CardContent>
      </Card>

      <Card className="border-memBorder-primary">
        <CardHeader><CardTitle className="text-sm">Step 3 · 验证记忆写入</CardTitle></CardHeader>
        <CardContent className="text-sm">
          运行上方“添加记忆”代码，然后到 <Link className="underline" href="/dashboard/memories">Memories</Link> 查看
          （metadata 中应包含 <code>git_remote</code> 与解析出的 <code>project_id</code>）。
        </CardContent>
      </Card>

      <Card className="border-memBorder-primary">
        <CardHeader><CardTitle className="text-sm">Step 4 · 检索与共享池</CardTitle></CardHeader>
        <CardContent className="text-sm">
          检索默认只返回你自己的记忆；带 <code>git_remote</code>/<code>project_id</code> 的检索/拉取会返回该项目的共享池（跨用户）。
          可先在 <Link className="underline" href="/dashboard/playground">Playground</Link> 无代码试玩。
        </CardContent>
      </Card>
    </div>
  );
}
```

- [ ] **Step 2: Rebuild + verify** — `GET /dashboard/get-started` → 200.

- [ ] **Step 3: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/dashboard/src/app/(root)/dashboard/get-started && git commit --no-verify -m "feat(dashboard): get-started wizard (MCP/Python/cURL)"`

---

### Task 9: Playground page (P0)

**Files:**
- Create: `mem0/server/dashboard/src/app/(root)/dashboard/playground/page.tsx`

- [ ] **Step 1: Create the page** — Add/Search modes, role-toggling message rows, `infer` switch, instructions input (injected as `prompt`), Output/Code views, sandbox banner:

```tsx
"use client";

import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/use-toast";
import { getErrorMessage } from "@/lib/error-message";
import { api } from "@/utils/api";
import { MEMORY_ENDPOINTS } from "@/utils/api-endpoints";

type Msg = { role: "user" | "assistant"; content: string };

const newId = () => `sandbox-${Math.random().toString(36).slice(2, 10)}`;

export default function PlaygroundPage() {
  const [mode, setMode] = useState<"add" | "search">("add");
  const [messages, setMessages] = useState<Msg[]>([{ role: "user", content: "" }]);
  const [query, setQuery] = useState("");
  const [infer, setInfer] = useState(true);
  const [instructions, setInstructions] = useState("");
  const [output, setOutput] = useState("");
  const [running, setRunning] = useState(false);

  const sandboxUserId = newId();

  const codeSnippet =
    mode === "add"
      ? `curl -X POST $API/memories -H "X-API-Key: $KEY" -d '{"messages": ${JSON.stringify(messages)}, "infer": ${infer}, "user_id": "${sandboxUserId}"${instructions ? `, "prompt": ${JSON.stringify(instructions)}` : ""}}'`
      : `curl -X POST $API/search -H "X-API-Key: $KEY" -d '{"query": ${JSON.stringify(query)}, "user_id": "${sandboxUserId}"}'`;

  async function run() {
    setRunning(true);
    try {
      if (mode === "add") {
        const res = await api.post(MEMORY_ENDPOINTS.BASE, {
          messages: messages.filter((m) => m.content.trim()),
          infer,
          user_id: sandboxUserId,
          ...(instructions.trim() ? { prompt: instructions.trim() } : {}),
        });
        setOutput(JSON.stringify(res.data, null, 2));
      } else {
        const res = await api.post("/search", { query, user_id: sandboxUserId });
        setOutput(JSON.stringify(res.data, null, 2));
      }
    } catch (error) {
      toast({ title: "Playground call failed", description: getErrorMessage(error), variant: "destructive" });
    } finally {
      setRunning(false);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold font-fustat">Playground</h1>
      <div className="rounded border border-amber-300 bg-amber-50 dark:bg-amber-950/30 px-3 py-2 text-xs">
        沙盒数据使用隔离的 user_id（<code>{sandboxUserId}</code> 前缀 sandbox-），不会进入你的项目记忆。
      </div>

      <div className="flex gap-2">
        {(["add", "search"] as const).map((m) => (
          <Button key={m} size="sm" variant={m === mode ? "default" : "outline"} onClick={() => setMode(m)}>
            {m === "add" ? "Add Memories" : "Search Memories"}
          </Button>
        ))}
      </div>

      {mode === "add" ? (
        <Card className="border-memBorder-primary">
          <CardHeader><CardTitle className="text-sm">会话消息</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {messages.map((m, i) => (
              <div key={i} className="flex gap-2 items-center">
                <button
                  className="px-2 py-1 text-xs border rounded border-memBorder-primary w-20"
                  onClick={() =>
                    setMessages((prev) => prev.map((x, j) => (j === i ? { ...x, role: x.role === "user" ? "assistant" : "user" } : x)))
                  }
                >
                  {m.role}
                </button>
                <Input
                  value={m.content}
                  onChange={(e) => setMessages((prev) => prev.map((x, j) => (j === i ? { ...x, content: e.target.value } : x)))}
                  placeholder={m.role === "user" ? "用户输入…" : "助手回复…"}
                />
                <Button variant="ghost" size="sm" onClick={() => setMessages((prev) => prev.filter((_, j) => j !== i))}>
                  ✕
                </Button>
              </div>
            ))}
            <Button variant="outline" size="sm" onClick={() => setMessages((prev) => [...prev, { role: "user", content: "" }])}>
              + 添加消息
            </Button>
            <div className="flex items-center gap-2 pt-2">
              <input id="infer" type="checkbox" checked={infer} onChange={(e) => setInfer(e.target.checked)} />
              <Label htmlFor="infer" className="text-xs">Extraction（开=抽取事实，关=逐字存储）</Label>
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Custom Instructions（试运行）</Label>
              <Textarea value={instructions} onChange={(e) => setInstructions(e.target.value)} rows={2}
                placeholder="例如：只存储技术偏好，忽略闲聊" />
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card className="border-memBorder-primary">
          <CardHeader><CardTitle className="text-sm">检索</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="搜索查询…" />
          </CardContent>
        </Card>
      )}

      <div className="flex items-center gap-3">
        <Button onClick={run} disabled={running}>{running ? "Running…" : "Run"}</Button>
      </div>

      {output && (
        <Card className="border-memBorder-primary">
          <CardHeader><CardTitle className="text-sm">Output</CardTitle></CardHeader>
          <CardContent>
            <pre className="text-xs bg-surface-default-secondary p-3 rounded font-mono overflow-x-auto max-h-80">{output}</pre>
            <p className="text-xs text-onSurface-default-tertiary mt-2">Code: <code>{codeSnippet}</code></p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Rebuild + verify** — `GET /dashboard/playground` → 200；在页面 Run 一条 add（AUTH_DISABLED 环境 admin 权限）→ Output 返回 `results`。

- [ ] **Step 3: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add "server/dashboard/src/app/(root)/dashboard/playground" && git commit --no-verify -m "feat(dashboard): playground sandbox (add/search, infer switch, instructions)"`

---

### Task 10: Projects dialog fields + Danger Zone (P0)

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

### Task 11: Requests filters (P1)

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

### Task 12: Webhooks page + nav (P1)

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

### Task 13: Exports page + nav (P1)

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

### Task 14: Context switcher (P1, minimal)

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

### Task 15: Final verification (P0+P1)

- [ ] **Step 1: Full rebuild** — `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-api mem0-dashboard`. Expected: both `Built` + `Started`, dashboard type-check passes.
- [ ] **Step 2: Full pytest** — `docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/ -q`. Expected: `27 + 16 new ≈ 43 passed`（以实际用例数为准，0 failed）.
- [ ] **Step 3: Smoke** — `GET /stats?range=7d` → 200；`GET /dashboard/{overview,get-started,playground,webhooks,memory-exports}` → 200；`DELETE /memories`（无 project_id）→ 400。
- [ ] **Step 4: Commit any stragglers** in the submodule and report.

---

## Self-Review Notes

- **Spec coverage:** P0 items → Tasks 1,2,3,6,7,8,9,10；P1 → Tasks 4,5,11,12,13,14。「不做」清单无对应任务（符合预期）。
- **Placeholder scan:** 前端任务（12/13 页面）给出组件骨架与关键代码，复用 Task 10/6 已建立的 modal/nav 锚点；无 TBD/TODO。
- **Type consistency:** `_load_project`（Task 2 定义，Task 3 复用）、`request_action`/`RANGE_*`（Task 1 定义，Task 11 复用）、`trigger_webhooks`/`events_from_add_response`（Task 4 定义并使用）、`_filter_memories_for_export`（Task 5）。

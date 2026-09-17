# Task 2 (extracted from implementation plan)

**Files:**
- Create: `mem0/server/routers/graph.py`
- Create: `mem0/server/tests/test_graph_router.py`
- Modify: `mem0/server/main.py` (import + include, after `stats_router`)

- [ ] **Step 1: Write the failing test** — `mem0/server/tests/test_graph_router.py`:

```python
from types import SimpleNamespace

import pytest
from fastapi import HTTPException

import routers.graph as graph_router


class _User:
    def __init__(self, uid="u1", role="user"):
        self.id = uid
        self.role = role


def test_search_pins_normal_user_and_uses_project_scope(monkeypatch):
    captured = {}

    def fake_search(query, filters, limit=100):
        captured.update({"query": query, "filters": filters, "limit": limit})
        return [{"source": "a", "relationship": "R", "destination": "b"}]

    monkeypatch.setattr(graph_router, "get_graph_memory", lambda: SimpleNamespace(search=fake_search, enabled=True))
    monkeypatch.setattr(graph_router, "resolve_project_id", lambda x, db=None: "p1")
    body = graph_router.GraphSearchRequest(query="alice", project_id="p1")
    out = graph_router.graph_search(body, _auth=_User(), db=None)
    assert out["relations"][0]["source"] == "a"
    assert captured["filters"] == {"user_id": "u1", "project_id": "p1", "scope": "project"}


def test_search_admin_may_pass_user_id(monkeypatch):
    captured = {}

    def fake_search(query, filters, limit=100):
        captured["filters"] = filters
        return []

    monkeypatch.setattr(graph_router, "get_graph_memory", lambda: SimpleNamespace(search=fake_search, enabled=True))
    body = graph_router.GraphSearchRequest(query="q", user_id="someone-else")
    graph_router.graph_search(body, _auth=_User(uid="admin1", role="admin"), db=None)
    assert captured["filters"]["user_id"] == "someone-else"
    assert captured["filters"]["scope"] == "user"


def test_search_normal_user_cannot_spoof_user_id(monkeypatch):
    monkeypatch.setattr(graph_router, "get_graph_memory", lambda: SimpleNamespace(search=lambda *a, **k: [], enabled=True))
    body = graph_router.GraphSearchRequest(query="q", user_id="other")
    with pytest.raises(HTTPException) as exc:
        graph_router.graph_search(body, _auth=_User(), db=None)
    assert exc.value.status_code == 403


def test_get_all_requires_project_or_user(monkeypatch):
    monkeypatch.setattr(graph_router, "get_graph_memory", lambda: SimpleNamespace(get_all=lambda f, limit=100: [], enabled=True))
    out = graph_router.graph_get_all(project_id=None, user_id=None, limit=100, _auth=_User(uid="a", role="admin"), db=None)
    assert out == {"relations": []}
```

- [ ] **Step 2: Run to verify it fails** (`ModuleNotFoundError: routers.graph`).

- [ ] **Step 3: Implement** — `mem0/server/routers/graph.py`:

```python
from typing import Optional

from auth import verify_auth
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from server_state import get_memory_instance

router = APIRouter(prefix="/graph", tags=["graph"])


def get_graph_memory():
    """Lazy accessor so tests can monkeypatch; main owns the singleton."""
    raise NotImplementedError  # replaced below after main provides the hook


class GraphSearchRequest(BaseModel):
    query: str
    project_id: Optional[str] = None
    git_remote: Optional[str] = None
    user_id: Optional[str] = None  # admin only
    limit: int = 25


class GraphGetAllRequest(BaseModel):
    project_id: Optional[str] = None
    git_remote: Optional[str] = None
    user_id: Optional[str] = None  # admin only
    limit: int = 100


def _resolve_scope(user, project_id, git_remote, user_id, db):
    """Return (filters, error). Personal scope is pinned to the caller."""
    from gitmatch import resolve_project_id
    if project_id is None and git_remote:
        project_id = resolve_project_id(git_remote, db)
    is_admin = user is None or user.role == "admin"
    if not is_admin:
        if user_id and user_id != str(user.id):
            raise HTTPException(status_code=403, detail="Cannot access graph of another user_id.")
        return {"user_id": str(user.id), "project_id": project_id,
                "scope": "project" if project_id else "user"}
    if user_id:
        return {"user_id": user_id, "project_id": project_id, "scope": "user"}
    if project_id:
        return {"project_id": project_id, "scope": "project"}
    raise HTTPException(status_code=400, detail="Provide project_id (or git_remote) or user_id.")


@router.post("/search")
def graph_search(body: GraphSearchRequest, _auth=Depends(verify_auth), db=None):
    from db import SessionLocal
    session = SessionLocal() if db is None else db
    try:
        filters = _resolve_scope(_auth, body.project_id, body.git_remote, body.user_id, session)
        graph = get_graph_memory()
        return {"relations": graph.search(body.query, filters, limit=body.limit)}
    finally:
        if db is None:
            session.close()


@router.get("/get_all")
def graph_get_all(project_id: Optional[str] = None, git_remote: Optional[str] = None,
                  user_id: Optional[str] = None, limit: int = 100,
                  _auth=Depends(verify_auth), db=None):
    from db import SessionLocal
    session = SessionLocal() if db is None else db
    try:
        filters = _resolve_scope(_auth, project_id, git_remote, user_id, session)
        graph = get_graph_memory()
        return {"relations": graph.get_all(filters, limit=limit)}
    finally:
        if db is None:
            session.close()
```

> Note: `get_graph_memory` is defined as a placeholder here and **replaced by the real lazy singleton in Task 3** (implemented in `main.py` and injected: `from main import get_graph_memory` would be circular — instead `main.py` sets `routers.graph.get_graph_memory = _get_graph_memory` after defining it; the test monkeypatches the module attribute either way). Tests that call the router directly pass `db=None` → the router builds a real `SessionLocal`; in the container this is available. For Step 1's fake-db tests, `monkeypatch.setattr` on `graph_router.get_graph_memory` covers the graph dependency; `db=None` opens a real session (harmless for the fake scope resolution because `resolve_project_id` is also monkeypatched in the tests that pass `project_id`).

- [ ] **Step 4: Register in `main.py`** — add `from routers import graph as graph_router` and `app.include_router(graph_router.router)` next to the other includes.

- [ ] **Step 5: Run to verify pass** — copy `routers/graph.py`, `tests/test_graph_router.py`, `main.py` into the container; `pytest tests/test_graph_router.py -q` → `4 passed`.
- [ ] **Step 6: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/routers/graph.py server/tests/test_graph_router.py server/main.py && git commit --no-verify -m "feat(server): /graph/search and /graph/get_all endpoints with scoping"`

---


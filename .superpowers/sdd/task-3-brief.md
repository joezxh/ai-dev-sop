# Task 3 (extracted from implementation plan)

**Files:**
- Modify: `mem0/server/main.py`
- Create: `mem0/server/tests/test_graph_integration.py`

- [ ] **Step 1: Write the failing test** — `mem0/server/tests/test_graph_integration.py`:

```python
from types import SimpleNamespace

import main as m


def test_get_graph_memory_is_lazy_singleton(monkeypatch):
    calls = {"n": 0}

    class FakeGM:
        enabled = True

        def __init__(self):
            calls["n"] += 1

    monkeypatch.setattr(m, "GraphMemory", FakeGM)
    m._graph_instance = None  # reset singleton
    a = m.get_graph_memory()
    b = m.get_graph_memory()
    assert a is b and calls["n"] == 1


def test_search_fusion_adds_relations(monkeypatch):
    inst = SimpleNamespace(search=lambda query, filters, **kw: {"results": [{"id": "1", "memory": "m"}]})
    monkeypatch.setattr(m, "get_memory_instance", lambda: inst)
    graph = SimpleNamespace(enabled=True, search=lambda q, f, limit=5: [
        {"source": "a", "relationship": "R", "destination": "b"}])
    monkeypatch.setattr(m, "get_graph_memory", lambda: graph)
    data = {"results": [{"id": "1", "memory": "m"}]}
    out = m._fuse_graph_relations(data, query="alice", filters={"user_id": "u1", "scope": "user"}, include_graph=True)
    assert out["relations"][0]["source"] == "a"


def test_search_fusion_disabled_returns_empty_relations(monkeypatch):
    graph = SimpleNamespace(enabled=False)
    monkeypatch.setattr(m, "get_graph_memory", lambda: graph)
    data = {"results": []}
    out = m._fuse_graph_relations(data, query="q", filters={"user_id": "u1", "scope": "user"}, include_graph=True)
    assert out["relations"] == []


def test_search_fusion_include_graph_false(monkeypatch):
    data = {"results": []}
    out = m._fuse_graph_relations(data, query="q", filters={"user_id": "u1"}, include_graph=False)
    assert "relations" not in out or out.get("relations") is None


def test_schedule_graph_build_skips_when_disabled(monkeypatch):
    graph = SimpleNamespace(enabled=False)
    monkeypatch.setattr(m, "get_graph_memory", lambda: graph)
    submitted = []
    monkeypatch.setattr(m, "_graph_executor", SimpleNamespace(submit=lambda fn, *a: submitted.append((fn, a))))
    m._schedule_graph_build("text", {"user_id": "u1"})
    assert submitted == []


def test_schedule_graph_build_submits_when_enabled(monkeypatch):
    graph = SimpleNamespace(enabled=True, add=lambda *a, **k: None)
    monkeypatch.setattr(m, "get_graph_memory", lambda: graph)
    submitted = []
    monkeypatch.setattr(m, "_graph_executor", SimpleNamespace(submit=lambda fn, *a: submitted.append((fn, a))))
    m._schedule_graph_build("alice works on payments", {"user_id": "u1", "project_id": None})
    assert len(submitted) == 1
```

- [ ] **Step 2: Run to verify it fails** (AttributeError on `m.get_graph_memory` etc.).

- [ ] **Step 3: Implement in `main.py`**:

```python
from graph_memory import GraphMemory
from concurrent.futures import ThreadPoolExecutor

_graph_instance = None
_graph_executor = ThreadPoolExecutor(max_workers=2, thread_name_prefix="graph")


def get_graph_memory():
    """Lazy singleton; auto-degrades when deps/Neo4j are unavailable."""
    global _graph_instance
    if _graph_instance is None:
        _graph_instance = GraphMemory(
            embedder=get_memory_instance().embedding_model,
            llm=get_memory_instance().llm,
        )
    return _graph_instance


def _schedule_graph_build(text: str, filters: dict) -> None:
    graph = get_graph_memory()
    if not graph.enabled:
        return
    _graph_executor.submit(graph.add, text, filters)


def _fuse_graph_relations(data: dict, query: str, filters: dict, include_graph: bool) -> dict:
    if not include_graph:
        return data
    try:
        graph = get_graph_memory()
        if graph.enabled:
            data["relations"] = graph.search(query, filters, limit=5)
        else:
            data["relations"] = []
    except Exception:  # noqa: BLE001 — fusion must never break search
        logging.exception("graph fusion failed")
        data["relations"] = []
    return data
```

Wire-in points:
1. **`add_memory`**: capture `graph_text = " ".join(m.content for m in memory_create.messages)` and `graph_filters = {"user_id": owner_user_id, "project_id": (project_id or None)}` before the mem0 call; after the successful `response = ...` line (before webhook firing) add `if graph_text.strip(): _schedule_graph_build(graph_text, graph_filters)`.
2. **`search_memories`**: build `graph_filters` — if project scope resolved (`project_scope = search_req.project_id or resolve_project_id(search_req.git_remote)`) → `{"project_id": project_scope, "scope": "project"}` else personal: resolve via `_scope_identifiers` (admin + explicit `user_id` in filters → that id, else pinned own id) → `{"user_id": ..., "scope": "user"}`; after `data = get_memory_instance().search(...)` wrap return: `return _fuse_graph_relations(data, query=search_req.query, filters=graph_filters, include_graph=search_req.include_graph is not False)`. Add `include_graph: Optional[bool] = True` to `SearchRequest`.
3. **`delete_memory`**: before `get_memory_instance().delete(memory_id)` capture `memory_text` via `get_memory_instance().get(memory_id)` (best-effort `.memory` field, may be None); after successful delete: `if memory_text: _graph_executor.submit(get_graph_memory().soft_delete_for_text, memory_text, {"user_id": <owner user_id from payload>})` — extract owner from the memory payload's `user_id` (fallback: skip). Guard the whole block with `try/except`.
4. **`delete_memories_by_project`**: after successful purge, `_graph_executor.submit(get_graph_memory().hard_delete, {"project_id": project_id, "scope": "project"})`.
5. **`delete_entity`**: after success, `hard_delete` with `{"user_id": entity_id, "scope": "user"}`.
- Reset `_graph_instance = None` in `update_config` path is unnecessary (embedder/llm instances are bound once — if the provider changes, the graph LLM/embedder become stale; acceptable for this round, note in code comment).

- [ ] **Step 4: Run to verify pass** — copy `main.py` + test file; `pytest tests/test_graph_integration.py tests/ -q` → all green (baseline 92 + new).
- [ ] **Step 5: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/main.py server/tests/test_graph_integration.py && git commit --no-verify -m "feat(server): graph build scheduling, /search relations fusion, delete links"`

---


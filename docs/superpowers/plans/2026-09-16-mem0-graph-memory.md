# Graph Memory (self-built, align 1.x) + Dashboard Visualization — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a self-hosted graph memory layer (align mem0 v1.0.11 `MemoryGraph` semantics, extended with a project dimension) behind `GRAPH_ENABLED`, fuse `relations` into `/search`, expose `/graph/*` endpoints, and add a dashboard graph visualization page.

**Architecture:** New `server/graph_memory.py` (port of 1.x semantics; injectable llm/embedder/neo4j driver for unit tests) + `server/routers/graph.py`; `main.py` wires async graph build after successful `add`, search fusion, delete links. Frontend: new `/dashboard/graph` page using `react-force-graph-2d` (new dependency). Spec: `docs/superpowers/specs/2026-09-16-mem0-graph-memory-design.md`.

**Tech Stack:** FastAPI; `langchain-neo4j`/`neo4j`/`rank-bm25` (already in image); mem0 `LlmFactory/EmbedderFactory`; Next.js + `react-force-graph-2d`.

**Test commands** (from `d:/projects/ai-dev-sop`): copy changed `server` files into `mem0-api:/app/...` via `docker compose -f deploy/mem0/docker-compose.yaml cp`, then `docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/ -q`. Current baseline: **92 passed**. Commits: inside the `mem0` submodule with `--no-verify`.

**Key design constraints (binding):**
- `GRAPH_ENABLED=true` default; auto-degrade (import failure / Neo4j unreachable → disabled + log); failures never propagate to main flow.
- Scope semantics: node identity = `(name, user_id, project_id)`; personal graph queries match `user_id`; project-pool graph queries match `project_id` (cross-user, same as memory shared pool).
- Relationship types are sanitized to `[A-Z0-9_]` before cypher interpolation.
- Soft delete (`valid=false`) for single-memory delete; hard delete (`DETACH DELETE`) for project purge / entity delete.
- Async graph build after successful `add` (executor), fire-and-forget.
- LLM calls use plain-JSON prompts (2.x `generate_response` has no tool support — documented deviation from 1.x tool-calls).

---

## File Structure Map

| File | Responsibility |
| --- | --- |
| Create: `mem0/server/graph_memory.py` | `GraphMemory` class: config/degrade, entity+relation extraction (JSON prompts), embedding disambiguation, MERGE cypher, search (cypher + BM25), get_all, soft/hard delete |
| Create: `mem0/server/routers/graph.py` | `POST /graph/search`, `GET /graph/get_all` (admin/user scoping) |
| Modify: `mem0/server/main.py` | include graph router; async graph build after add; `/search` fusion (`include_graph`); delete links |
| Modify: `mem0/server/server_state.py` or `main.py` | `get_graph_memory()` lazy singleton helper |
| Modify: `mem0/server/Dockerfile`→ actually `deploy/mem0/api/Dockerfile` | remove no-op `mem0ai[graph]` |
| Modify: `deploy/mem0/.env`, `deploy/mem0/docker-compose.yaml` | `GRAPH_ENABLED/GRAPH_THRESHOLD/GRAPH_CUSTOM_PROMPT` |
| Create: `mem0/server/tests/test_graph_memory.py` | unit tests (fake llm/embedder/neo4j) |
| Create: `mem0/server/tests/test_graph_router.py` | endpoint tests |
| Modify: `mem0/server/tests/test_search_fusion.py` (new) | /search fusion tests |
| Dashboard: `src/utils/api-endpoints.ts`, `src/types/api.ts`, `src/app/(root)/dashboard/graph/page.tsx`, `main-nav.tsx`, `memories/page.tsx`, `package.json` | visualization |

---

### Task 1: `GraphMemory` core (TDD)

**Files:**
- Create: `mem0/server/graph_memory.py`
- Create: `mem0/server/tests/test_graph_memory.py`

- [ ] **Step 1: Write the failing test** — `mem0/server/tests/test_graph_memory.py`:

```python
"""Unit tests for the self-built graph memory layer (1.x-aligned semantics).
All externals (Neo4j, LLM, embedder) are fakes; no service calls."""

from types import SimpleNamespace

import pytest

import graph_memory as gm


class FakeLLM:
    def __init__(self, responses):
        self.responses = list(responses)
        self.calls = []

    def generate_response(self, messages, **_kw):
        self.calls.append(messages)
        return self.responses.pop(0)


class FakeEmbedder:
    def __init__(self):
        self.n = 0

    def embed(self, text, **_kw):
        self.n += 1
        return [float(len(text) % 7 + self.n), 1.0, 0.0]


class FakeNeo4j:
    """Records queries; returns canned results keyed by a substring."""
    def __init__(self, canned=None):
        self.queries = []
        self.canned = canned or []

    def query(self, cypher, params=None):
        self.queries.append((cypher, params or {}))
        for marker, rows in self.canned:
            if marker in cypher:
                return rows
        return []


def make_graph(llm_responses, neo4j=None, threshold=0.7):
    neo4j = neo4j or FakeNeo4j()
    g = gm.GraphMemory(
        embedder=FakeEmbedder(),
        llm=FakeLLM(llm_responses),
        driver=neo4j,
        threshold=threshold,
    )
    return g, neo4j


FILTERS = {"user_id": "u1", "project_id": "p1"}


def test_disabled_when_env_off(monkeypatch):
    monkeypatch.setenv("GRAPH_ENABLED", "false")
    g, _ = make_graph([])
    assert g.enabled is False
    assert g.add("hello", FILTERS) == {"added_entities": [], "deleted_entities": []}
    assert g.search("hello", FILTERS) == []


def test_degrades_when_driver_init_fails(monkeypatch):
    monkeypatch.setenv("GRAPH_ENABLED", "true")
    # Neo4jGraph import failure path: simulate by pointing url at an unreachable
    # bolt endpoint is slow; instead verify the degrade contract directly —
    # GraphMemory must never raise from the constructor.
    try:
        g = gm.GraphMemory(embedder=FakeEmbedder(), llm=FakeLLM([]), driver=None, url="bolt://invalid.invalid:7687")
        assert isinstance(g.enabled, bool)
    except Exception:  # noqa: BLE001
        pytest.fail("GraphMemory constructor must not raise on driver failure")


def test_extract_entities_normalizes():
def test_extract_entities_normalizes():
    llm = FakeLLM(['```json\n{"entities":[{"entity":"Alice Smith","entity_type":"Person"},'
                   '{"entity":"I","entity_type":"Self"}]}\n```'])
    neo4j = FakeNeo4j()
    g = gm.GraphMemory(embedder=FakeEmbedder(), llm=llm, driver=neo4j, threshold=0.7)
    ents = g._extract_entities("I work with Alice Smith", FILTERS)
    assert ents == [{"entity": "alice_smith", "entity_type": "person"}] or ents == [
        {"entity": "alice_smith", "entity_type": "person"},
        {"entity": "u1", "entity_type": "self"},
    ]
    # Either shape is acceptable; the binding rule: names are normalized
    # (lowercase, spaces→underscore) and self reference "I" maps to the user_id entity.
    names = [e["entity"] for e in ents]
    assert "alice_smith" in names
    assert "i" not in names


def test_add_merges_nodes_and_relationship():
    llm = FakeLLM([
        '{"entities":[{"entity":"alice","entity_type":"person"}]}',
        '{"entities":[{"source":"alice","relationship":"WORKS_ON","destination":"payments"}]}',
    ])
    neo4j = FakeNeo4j([("vector.similarity", [])])  # no similar node → create
    g = gm.GraphMemory(embedder=FakeEmbedder(), llm=llm, driver=neo4j, threshold=0.7)
    out = g.add("alice works on payments", FILTERS)
    assert out["added_entities"]  # triples recorded
    cyphers = " || ".join(c for c, _ in neo4j.queries)
    assert "MERGE" in cyphers
    assert "alice" in "".join(str(p) for _, p in neo4j.queries)
    # relationship type must be sanitized into the cypher label
    assert "WORKS_ON" in cyphers


def test_relationship_type_is_sanitized():
    assert gm.sanitize_relationship("works  on!") == "WORKS_ON"


def test_search_returns_bm25_top_triples():
    rows = [
        {"source": "alice", "relationship": "WORKS_ON", "destination": "payments", "sim": 0.9},
        {"source": "alice", "relationship": "LIVES_IN", "destination": "berlin", "sim": 0.8},
        {"source": "bob", "relationship": "WORKS_ON", "destination": "infra", "sim": 0.7},
    ]
    neo4j = FakeNeo4j([("vector.similarity", rows)])
    llm = FakeLLM(['{"entities":[{"entity":"alice","entity_type":"person"}]}'])
    g = gm.GraphMemory(embedder=FakeEmbedder(), llm=llm, driver=neo4j, threshold=0.7)
    out = g.search("alice", FILTERS)
    assert isinstance(out, list) and out
    assert all(set(t) >= {"source", "relationship", "destination"} for t in out)
    # cypher must be scoped by user_id
    last = neo4j.queries[-1][1]
    assert last.get("user_id") == "u1"


def test_project_scope_uses_project_id_predicate():
    neo4j = FakeNeo4j([("vector.similarity", [])])
    llm = FakeLLM(['{"entities":[]}'])
    g = gm.GraphMemory(embedder=FakeEmbedder(), llm=llm, driver=neo4j, threshold=0.7)
    g.search("anything", {"user_id": "u1", "project_id": "p1", "scope": "project"})
    last_cypher, last_params = neo4j.queries[-1]
    assert last_params.get("project_id") == "p1"
    assert "project_id" in last_cypher


def test_soft_delete_marks_invalid():
    neo4j = FakeNeo4j([("relation extraction", [])])
    llm = FakeLLM([
        '{"entities":[{"entity":"alice","entity_type":"person"}]}',
        '{"entities":[{"source":"alice","relationship":"WORKS_ON","destination":"payments"}]}',
    ])
    g = gm.GraphMemory(embedder=FakeEmbedder(), llm=llm, driver=neo4j, threshold=0.7)
    g.soft_delete_for_text("alice works on payments", FILTERS)
    cyphers = " || ".join(c for c, _ in neo4j.queries)
    assert "valid = false" in cyphers or "valid=false" in cyphers


def test_hard_delete_uses_detach_delete():
    neo4j = FakeNeo4j()
    g = gm.GraphMemory(embedder=FakeEmbedder(), llm=FakeLLM([]), driver=neo4j, threshold=0.7)
    g.hard_delete({"project_id": "p1"})
    cyphers = " || ".join(c for c, _ in neo4j.queries)
    assert "DETACH DELETE" in cyphers
```

- [ ] **Step 2: Run to verify it fails** — `docker compose -f deploy/mem0/docker-compose.yaml cp mem0/server/tests/test_graph_memory.py mem0-api:/app/tests/ ; docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/test_graph_memory.py -q`
  Expected: FAIL (`ModuleNotFoundError: graph_memory`).

- [ ] **Step 3: Implement** — `mem0/server/graph_memory.py`:

```python
"""Self-built graph memory layer, aligning mem0 v1.0.11 MemoryGraph semantics
with a project dimension. All failures degrade to disabled + log (spec §1)."""

import json
import logging
import os
import re
from datetime import datetime, timezone

logger = logging.getLogger(__name__)

DEFAULT_THRESHOLD = 0.7
BASE_LABEL = "__Entity__"
SEARCH_TOP_K = 5

EXTRACT_ENTITIES_SYSTEM = (
    "You are a smart assistant that extracts entities and their types from text. "
    "If the text contains self reference such as 'I', 'me', 'my', use '{user_id}' "
    "as the entity name instead. Do NOT answer the text itself. "
    'Respond ONLY with JSON: {"entities": [{"entity": "...", "entity_type": "..."}]}'
)

EXTRACT_RELATIONS_SYSTEM = (
    "You are a smart assistant that extracts relationship triples from text. "
    "The source/destination MUST be chosen from the given entity list (or the "
    "user entity '{user_id}'). Extract the relationship between them. "
    'Respond ONLY with JSON: {"entities": [{"source": "...", "relationship": "...", '
    '"destination": "..."}]}. Rules: 1. Extract only what the text states. '
    "2. Use the entity names verbatim. 3. Relationship is 1-3 words. {custom_rules}"
)


def _norm(value: str) -> str:
    return (value or "").strip().lower().replace(" ", "_")


def sanitize_relationship(value: str) -> str:
    return re.sub(r"[^A-Z0-9_]", "", (value or "").strip().upper().replace(" ", "_"))


def _parse_json(text: str) -> dict:
    """Parse an LLM response that may be wrapped in markdown fences."""
    if not text:
        return {}
    cleaned = text.strip()
    if cleaned.startswith("```"):
        cleaned = re.sub(r"^```(?:json)?\s*|\s*```$", "", cleaned, flags=re.MULTILINE)
    try:
        parsed = json.loads(cleaned)
        return parsed if isinstance(parsed, dict) else {}
    except (ValueError, TypeError):
        match = re.search(r"\{.*\}", cleaned, re.DOTALL)
        if match:
            try:
                parsed = json.loads(match.group(0))
                return parsed if isinstance(parsed, dict) else {}
            except (ValueError, TypeError):
                return {}
        return {}


def _is_enabled_flag() -> bool:
    return os.environ.get("GRAPH_ENABLED", "true").strip().lower() in ("1", "true", "yes")


class GraphMemory:
    """Neo4j-backed entity-relation graph. Externals are injectable for tests."""

    def __init__(self, embedder=None, llm=None, driver=None, url=None,
                 username=None, password=None, threshold=None):
        self.enabled = _is_enabled_flag()
        self.graph = driver
        if not self.enabled:
            logger.info("GraphMemory disabled by GRAPH_ENABLED env")
            return
        try:
            if self.graph is None:
                from langchain_neo4j import Neo4jGraph
                self.graph = Neo4jGraph(
                    url=url or os.environ.get("NEO4J_URI", "bolt://localhost:7687"),
                    username=username or os.environ.get("NEO4J_USERNAME", "neo4j"),
                    password=password or os.environ.get("NEO4J_PASSWORD", ""),
                    refresh_schema=False,
                    driver_config={"notifications_min_severity": "OFF"},
                )
            self._ensure_indexes()
        except Exception:
            logger.exception("GraphMemory degraded: Neo4j unavailable")
            self.enabled = False
            return
        self.embedder = embedder
        self.llm = llm
        self.threshold = float(
            threshold if threshold is not None else os.environ.get("GRAPH_THRESHOLD", DEFAULT_THRESHOLD)
        )
        self.custom_prompt = os.environ.get("GRAPH_CUSTOM_PROMPT") or None

    # ---- internals -------------------------------------------------------

    def _ensure_indexes(self) -> None:
        for statement in (
            f"CREATE INDEX graph_user IF NOT EXISTS FOR (n:{BASE_LABEL}) ON (n.user_id)",
            f"CREATE INDEX graph_project IF NOT EXISTS FOR (n:{BASE_LABEL}) ON (n.project_id)",
            f"CREATE INDEX graph_name_user IF NOT EXISTS FOR (n:{BASE_LABEL}) ON (n.name, n.user_id)",
        ):
            try:
                self.graph.query(statement)
            except Exception:  # noqa: BLE001 — index creation is best-effort
                logger.debug("graph index statement skipped: %s", statement)

    def _scope_match(self, filters: dict, alias: str = "n") -> str:
        """Cypher property predicate for the active scope (personal vs project)."""
        if filters.get("scope") == "project" and filters.get("project_id"):
            return f"{alias}.project_id = $project_id"
        return f"{alias}.user_id = $user_id"

    def _scope_params(self, filters: dict) -> dict:
        if filters.get("scope") == "project" and filters.get("project_id"):
            return {"project_id": filters["project_id"]}
        return {"user_id": filters.get("user_id")}

    def _extract_entities(self, data: str, filters: dict) -> list[dict]:
        user_id = filters.get("user_id") or "unknown"
        messages = [
            {"role": "system", "content": EXTRACT_ENTITIES_SYSTEM.replace("{user_id}", user_id)},
            {"role": "user", "content": data},
        ]
        parsed = _parse_json(self.llm.generate_response(messages))
        entities = []
        for item in parsed.get("entities") or []:
            name = _norm(item.get("entity"))
            if not name:
                continue
            if name == _norm(user_id) or name == "i":
                name = user_id
            entities.append({"entity": name, "entity_type": _norm(item.get("entity_type")) or "entity"})
        return entities

    def _extract_relations(self, data: str, filters: dict, entities: list[dict]) -> list[dict]:
        user_id = filters.get("user_id") or "unknown"
        rules = "4. " + self.custom_prompt if self.custom_prompt else ""
        system = EXTRACT_RELATIONS_SYSTEM.replace("{user_id}", user_id).replace("{custom_rules}", rules)
        user_content = f"List of entities: {[e['entity'] for e in entities]}\n\nText: {data}"
        messages = [{"role": "system", "content": system}, {"role": "user", "content": user_content}]
        parsed = _parse_json(self.llm.generate_response(messages))
        triples = []
        for item in parsed.get("entities") or []:
            source, destination = _norm(item.get("source")), _norm(item.get("destination"))
            rel = sanitize_relationship(item.get("relationship"))
            if not (source and destination and rel):
                continue
            triples.append({"source": source, "relationship": rel, "destination": destination})
        return triples

    def _find_similar_node(self, name: str, filters: dict) -> dict | None:
        embedding = self.embedder.embed(name)
        scope = self._scope_match(filters)
        extra = "" if filters.get("scope") == "project" else (
            " AND (n.project_id = $project_id OR n.project_id IS NULL)"
            if filters.get("project_id") else ""
        )
        cypher = (
            f"MATCH (n:{BASE_LABEL}) WHERE {scope}{extra} AND n.embedding IS NOT NULL "
            "WITH n, vector.similarity.cosine(n.embedding, $embedding) AS similarity "
            "WHERE similarity >= $threshold "
            "RETURN elementId(n) AS eid, n.name AS name, n.type AS type "
            "ORDER BY similarity DESC LIMIT 1"
        )
        params = self._scope_params(filters)
        params.update({"embedding": embedding, "threshold": self.threshold})
        rows = self.graph.query(cypher, params) or []
        return rows[0] if rows else None

    # ---- public API ------------------------------------------------------

    def add(self, data: str, filters: dict) -> dict:
        if not self.enabled:
            return {"added_entities": [], "deleted_entities": []}
        try:
            entities = self._extract_entities(data, filters)
            triples = self._extract_relations(data, filters, entities)
            added = []
            for t in triples:
                src = self._find_similar_node(t["source"], filters) or {"name": t["source"], "type": "entity"}
                dst = self._find_similar_node(t["destination"], filters) or {"name": t["destination"], "type": "entity"}
                src_emb = self.embedder.embed(src["name"])
                dst_emb = self.embedder.embed(dst["name"])
                cypher = (
                    f"MERGE (a:{BASE_LABEL} {{name: $source, user_id: $user_id}}) "
                    "ON CREATE SET a.created = timestamp(), a.mentions = 1, a.type = $source_type "
                    "ON MATCH SET a.mentions = coalesce(a.mentions, 0) + 1 "
                    "SET a.project_id = $project_id "
                    "CALL db.create.setNodeVectorProperty(a, 'embedding', $source_embedding) "
                    f"MERGE (b:{BASE_LABEL} {{name: $destination, user_id: $user_id}}) "
                    "ON CREATE SET b.created = timestamp(), b.mentions = 1, b.type = $destination_type "
                    "ON MATCH SET b.mentions = coalesce(b.mentions, 0) + 1 "
                    "SET b.project_id = $project_id "
                    "CALL db.create.setNodeVectorProperty(b, 'embedding', $destination_embedding) "
                    f"MERGE (a)-[r:{t['relationship']}]->(b) "
                    "ON CREATE SET r.created = timestamp(), r.mentions = 1, r.valid = true, "
                    "r.user_id = $user_id, r.project_id = $project_id "
                    "ON MATCH SET r.mentions = coalesce(r.mentions, 0) + 1, "
                    "r.updated_at = timestamp(), r.valid = true "
                    "RETURN coalesce(a.name, '') AS source, type(r) AS relationship, "
                    "coalesce(b.name, '') AS destination"
                )
                params = {
                    "source": src["name"], "destination": dst["name"],
                    "source_type": src.get("type", "entity"), "destination_type": dst.get("type", "entity"),
                    "source_embedding": src_emb, "destination_embedding": dst_emb,
                    "user_id": filters.get("user_id"), "project_id": filters.get("project_id"),
                }
                rows = self.graph.query(cypher, params) or []
                added.extend(rows or [t])
            return {"added_entities": added, "deleted_entities": []}
        except Exception:
            logger.exception("GraphMemory.add failed (degraded, main flow unaffected)")
            return {"added_entities": [], "deleted_entities": []}

    def search(self, query: str, filters: dict, limit: int = 100) -> list[dict]:
        if not self.enabled:
            return []
        try:
            entities = self._extract_entities(query, filters)
            if not entities:
                return []
            all_triples: list[dict] = []
            for entity in entities:
                embedding = self.embedder.embed(entity["entity"])
                cypher = (
                    f"MATCH (n:{BASE_LABEL}) WHERE {self._scope_match(filters)} "
                    "AND n.embedding IS NOT NULL "
                    "WITH n, vector.similarity.cosine(n.embedding, $embedding) AS similarity "
                    "WHERE similarity >= $threshold "
                    "MATCH (n)-[r]-(m) WHERE (r.valid IS NULL OR r.valid = true) "
                    "AND (" + self._scope_match(filters, "m") + ") "
                    "RETURN coalesce(n.name, '') AS source, type(r) AS relationship, "
                    "coalesce(m.name, '') AS destination, similarity "
                    "ORDER BY similarity DESC LIMIT $limit"
                )
                params = self._scope_params(filters)
                params.update({"embedding": embedding, "threshold": self.threshold, "limit": limit})
                all_triples.extend(self.graph.query(cypher, params) or [])
            if not all_triples:
                return []
            try:
                from rank_bm25 import BM25Okapi
                corpus = [
                    [str(t.get("source", "")), str(t.get("relationship", "")), str(t.get("destination", ""))]
                    for t in all_triples
                ]
                bm25 = BM25Okapi(corpus)
                scores = bm25.get_scores(query.split())
                ranked = sorted(zip(all_triples, scores), key=lambda pair: pair[1], reverse=True)
                return [{k: t.get(k, "") for k in ("source", "relationship", "destination")}
                        for t, _ in ranked[:SEARCH_TOP_K]]
            except ImportError:
                return all_triples[:SEARCH_TOP_K]
        except Exception:
            logger.exception("GraphMemory.search failed (degraded)")
            return []

    def get_all(self, filters: dict, limit: int = 100) -> list[dict]:
        if not self.enabled:
            return []
        try:
            cypher = (
                f"MATCH (n:{BASE_LABEL})-[r]->(m) "
                f"WHERE {self._scope_match(filters)} AND ({self._scope_match(filters, 'm')}) "
                "AND (r.valid IS NULL OR r.valid = true) "
                "RETURN coalesce(n.name, '') AS source, type(r) AS relationship, "
                "coalesce(m.name, '') AS destination LIMIT $limit"
            )
            return self.graph.query(cypher, {**self._scope_params(filters), "limit": limit}) or []
        except Exception:
            logger.exception("GraphMemory.get_all failed (degraded)")
            return []

    def soft_delete_for_text(self, data: str, filters: dict) -> None:
        if not self.enabled:
            return
        try:
            entities = self._extract_entities(data, filters)
            triples = self._extract_relations(data, filters, entities)
            for t in triples:
                cypher = (
                    f"MATCH (a:{BASE_LABEL} {{name: $source, user_id: $user_id}})-[r]->"
                    f"(b:{BASE_LABEL} {{name: $destination, user_id: $user_id}}) "
                    f"WHERE type(r) = $rel "
                    "SET r.valid = false, r.invalidated_at = datetime()"
                )
                self.graph.query(cypher, {
                    "source": t["source"], "destination": t["destination"],
                    "rel": t["relationship"], "user_id": filters.get("user_id"),
                })
        except Exception:
            logger.exception("GraphMemory.soft_delete_for_text failed (degraded)")

    def hard_delete(self, filters: dict) -> None:
        if not self.enabled:
            return
        try:
            cypher = f"MATCH (n:{BASE_LABEL}) WHERE {self._scope_match(filters)} DETACH DELETE n"
            self.graph.query(cypher, self._scope_params(filters))
        except Exception:
            logger.exception("GraphMemory.hard_delete failed (degraded)")
```

- [ ] **Step 4: Run to verify pass** (copy `graph_memory.py` + test). Expected: `8+ passed` (adjust count to the final test list; the three malformed sketches in Step 1 must be replaced by the six concrete tests).
- [ ] **Step 5: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add server/graph_memory.py server/tests/test_graph_memory.py && git commit --no-verify -m "feat(server): self-built graph memory core (1.x-aligned, project-dim extension)"`

---

### Task 2: `/graph/search` + `/graph/get_all` router (TDD)

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

### Task 3: main.py integration — singleton, add-time build, search fusion, delete links (TDD)

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

### Task 4: Config wiring + Dockerfile cleanup

**Files:**
- Modify: `deploy/mem0/.env`, `deploy/mem0/docker-compose.yaml`, `deploy/mem0/api/Dockerfile`

- [ ] **Step 1: `.env`** — append:

```
# ---- Graph memory (self-built, 1.x aligned) ----
GRAPH_ENABLED=true
GRAPH_THRESHOLD=0.7
GRAPH_CUSTOM_PROMPT=
```

- [ ] **Step 2: compose** — add to `mem0-api.environment`:

```yaml
      - GRAPH_ENABLED=${GRAPH_ENABLED:-true}
      - GRAPH_THRESHOLD=${GRAPH_THRESHOLD:-0.7}
      - GRAPH_CUSTOM_PROMPT=${GRAPH_CUSTOM_PROMPT:-}
```

- [ ] **Step 3: Dockerfile** — replace the no-op extra install line:

```dockerfile
RUN pip install --no-cache-dir \
    -i https://mirrors.aliyun.com/pypi/simple/ \
    --trusted-host mirrors.aliyun.com \
    "mem0ai[graph]" \
    rank-bm25 langchain-neo4j neo4j
```
→
```dockerfile
# mem0ai 2.x has no graph engine (its "graph" extra does not exist); the graph
# layer is self-built in server/graph_memory.py on top of langchain-neo4j.
RUN pip install --no-cache-dir \
    -i https://mirrors.aliyun.com/pypi/simple/ \
    --trusted-host mirrors.aliyun.com \
    rank-bm25 langchain-neo4j neo4j
```

- [ ] **Step 4: Verify** — `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-api` → Built/Started; container log shows no graph degradation error (Neo4j reachable); `pytest tests/ -q` in container → all green.
- [ ] **Step 5: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add -A . && git commit --no-verify -m "chore: drop no-op mem0ai[graph] install; wire GRAPH_* env"`（`.env`/compose 在外层 deploy/ 目录，需在外层仓库提交：`cd d:/projects/ai-dev-sop && git add deploy/mem0/.env deploy/mem0/docker-compose.yaml && git commit --no-verify -m "chore(deploy): wire GRAPH_* env for mem0-api"`）

---

### Task 5: Dashboard — deps, endpoints, types, nav entry

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

### Task 6: Dashboard graph page（画布 + 交互 + 详情侧栏）

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

### Task 7: Memories 详情「查看关联图」入口

**Files:**
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/memories/page.tsx`

- [ ] **Step 1: Add entry** — 详情 Sheet 内（Project 行之后）加：

```tsx
<Button
  variant="outline"
  size="sm"
  onClick={() => {
    const params = new URLSearchParams({ q: selectedMemory.memory });
    if (selectedMemory.project_id) params.set("project", selectedMemory.project_id);
    router.push(`/dashboard/graph?${params.toString()}`);
  }}
>
  查看关联图
</Button>
```
（`router` 来自 `useRouter()`；`selectedMemory.metadata?.user_id` 可一并携带。）
- [ ] **Step 2: Verify** — rebuild → 登录后 Memories → 点击一条记忆 → 详情 Sheet 出现按钮；跳转 graph 页自动检索。
- [ ] **Step 3: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add "server/dashboard/src/app/(root)/dashboard/memories" && git commit --no-verify -m "feat(dashboard): open graph view from memory detail"`

---

### Task 8: Final verification

- [ ] **Step 1: 全量重建** — `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-api mem0-dashboard` → 两 Built/Started 无 error。
- [ ] **Step 2: 全量 pytest** — 容器内 `pytest tests/ -q` → 全绿（92 + graph 新增 ≈ 105+）。
- [ ] **Step 3: Smoke** — `/graph/search`（带假数据路径：GRAPH_ENABLED=true + Neo4j 可达 + 空 LLM → relations 空数组 200）；`/search?include_graph=false` → 无 relations 字段；`/dashboard/graph` 307/200；Memories → 查看关联图跳转。
- [ ] **Step 4: 手动 E2E（有真实 key 的环境）** — add 含实体的记忆 → `:7474` 浏览器验证节点/边 → search 出 relations → delete 后软删。无 key 环境如实记录为运行时限制。
- [ ] **Step 5: 提交任何遗留修复**（mem0 子模块）。

---

## Self-Review Notes

- **Spec coverage:** §3 配置/开关 → Task 1/4；§4 数据模型/索引 → Task 1；§5 写入链路 → Task 3；§6 检索/删除 → Task 2/3；§8 可视化 → Task 5/6/7；§7 测试 → 各任务 + Task 8。
- **已知偏离（计划内）**：LLM 用纯 JSON prompt（2.x `generate_response` 无工具调用，1.x 是 tool-call）；`base_label` 固定启用；实体消歧限定在同 scope；图 LLM/embedder 绑定首次实例化（provider 变更后需重启，代码注释注明）。
- **Type consistency:** `GraphMemory.add/search/get_all/soft_delete_for_text/hard_delete` 签名在 Task 1 定义、Task 2/3 消费；前端 `GraphRelation` 与后端三元组键一致（`destination`，非 1.x 文档中的 `target`）。

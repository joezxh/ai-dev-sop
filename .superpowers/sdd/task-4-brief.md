# Task 4 (extracted from implementation plan)

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


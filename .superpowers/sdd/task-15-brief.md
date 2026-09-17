# Task 15 (extracted from implementation plan)

- [ ] **Step 1: Full rebuild** — `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-api mem0-dashboard`. Expected: both `Built` + `Started`, dashboard type-check passes.
- [ ] **Step 2: Full pytest** — `docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/ -q`. Expected: `27 + 16 new ≈ 43 passed`（以实际用例数为准，0 failed）.
- [ ] **Step 3: Smoke** — `GET /stats?range=7d` → 200；`GET /dashboard/{overview,get-started,playground,webhooks,memory-exports}` → 200；`DELETE /memories`（无 project_id）→ 400。
- [ ] **Step 4: Commit any stragglers** in the submodule and report.

---

## Self-Review Notes

- **Spec coverage:** P0 items → Tasks 1,2,3,6,7,8,9,10；P1 → Tasks 4,5,11,12,13,14。「不做」清单无对应任务（符合预期）。
- **Placeholder scan:** 前端任务（12/13 页面）给出组件骨架与关键代码，复用 Task 10/6 已建立的 modal/nav 锚点；无 TBD/TODO。
- **Type consistency:** `_load_project`（Task 2 定义，Task 3 复用）、`request_action`/`RANGE_*`（Task 1 定义，Task 11 复用）、`trigger_webhooks`/`events_from_add_response`（Task 4 定义并使用）、`_filter_memories_for_export`（Task 5）。

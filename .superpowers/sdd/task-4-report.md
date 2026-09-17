# Task 4 报告：Webhooks 后端（模型 + CRUD 路由 + fire-and-forget 事件投递）

**状态：** DONE
**提交：** `0b3c23ad`（mem0 子模块，分支 `feat/llm-provider-i18n`）— `feat(server): webhooks CRUD + fire-and-forget memory event delivery`

## 测试结果

| 阶段 | 命令 | 结果 |
|------|------|------|
| RED | `pytest tests/test_webhooks.py -q`（容器内，实现前） | `ModuleNotFoundError: No module named 'webhooks_client'`（预期失败 ✅） |
| GREEN | `pytest tests/test_webhooks.py -q` | **3 passed** |
| 回归 | `pytest tests/ -q` | **40 passed**（37 原有 + 3 新增），仅 2 个预先存在的 deprecation warning |

## 交付文件

| 文件 | 动作 | 内容 |
|------|------|------|
| `server/tests/test_webhooks.py` | 新建 | 3 个测试（逐字取自 brief）：事件映射、按订阅过滤、投递永不抛异常 |
| `server/webhooks_client.py` | 新建 | `events_from_add_response`（ADD/UPDATE/DELETE → memory_added/updated/deleted）、`trigger_webhooks`（fire-and-forget，httpx.post timeout=5.0，异常仅 warning） |
| `server/models.py` | 修改 | 在 `Project` 之后新增 `Webhook` 模型（`webhooks` 表：id/name/url/events(JSON)/secret/created_at），复用已有 `_new_uuid`/`_utcnow` |
| `server/routers/webhooks.py` | 新建 | `/webhooks` CRUD：GET 列表、POST 创建（事件白名单校验，非法事件 400）、DELETE /{id}（非法 UUID / 不存在 404），全部 `require_admin` |
| `server/main.py` | 修改 | ① import + `app.include_router(webhooks_router.router)`（stats_router 之后）；② `_fire_webhooks` helper（daemon 线程 + 独立 SessionLocal，异常 `logging.exception`）；③ `add_memory` 成功后 `_fire_webhooks(response.get("results") or [])`；④ `update_memory` 成功后 `_fire_webhooks([{"event": "UPDATE"}])`（保留原返回值）；⑤ `delete_memory` 成功后 `_fire_webhooks([{"event": "DELETE"}])` |

## TDD 门禁合规

- ✅ RED：实现前测试因模块不存在而失败（ModuleNotFoundError）
- ✅ GREEN：最小实现后 3/3 通过
- 本次无 REFACTOR 需求（实现即 brief 逐字代码）

## 验证细节

- 所有文件均已 cp 到容器对应路径（`/app/models.py`、`/app/webhooks_client.py`、`/app/routers/webhooks.py`、`/app/main.py`、`/app/tests/test_webhooks.py`），测试不依赖建表（不连 DB），`webhooks` 表由启动时 `Base.metadata.create_all` 自动创建。
- Lint：修改/新建的 5 个文件均无诊断错误。
- main.py diff 自审确认：helper 定义在 `add_memory` 之前；import 块新增两行；include_router 位于 `stats_router` 之后；`update_memory` 通过 `result = ...` 保留 mem0 原返回值。

## 偏差 / 已知关注点

- 无计划偏差，全部代码逐字采用 brief 提供的实现。
- Concern（非阻塞，设计取舍）：`delete_all_memories`（DELETE /memories 批量删除）与 `delete_memories_by_project` 未触发 DELETE 事件——brief 仅要求三个单条端点，按 brief 执行未扩展；如后续需要批量删除事件，可在下个任务补充。
- Concern（非阻塞）：`_fire_webhooks` 中 UPDATE/DELETE 的 payload 为 `{"results": [...]}` 包装（`{"results": [{"event": "UPDATE"}]}`），与 add 一致，符合 brief 设计。

---

## 审查修复记录（2 个 Important 问题）

**状态：** DONE
**提交：** `6b9d4099`（mem0 子模块，分支 `feat/llm-provider-i18n`）— `fix(server): harden webhook dispatch (no-throw guard) and validate webhook URL scheme`

### 修复点

1. **异常保护（`server/main.py` `_fire_webhooks`）**：函数体整体包入 `try/except Exception`，异常仅 `logging.exception("Webhook dispatch failed")`，任何输入都不会向调用方抛出——避免 `events_from_add_response(results)` 或 `threading.Thread(...).start()` 的异常沿调用链被 `add_memory` 外层 `except → upstream_error()` 捕获，把已成功的记忆写入翻转为 500。
2. **SSRF 校验（`server/routers/webhooks.py` `create_webhook`）**：新增 `_validate_url`，用 `urllib.parse.urlparse` 校验 `body.url`——scheme 必须为 `http`/`https` 且 `netloc` 非空，否则 `HTTPException(status_code=400, detail="url must be a valid http(s) URL")`。按 brief 不阻止私有网段（admin-gated，保持简单）。

### 覆盖用例（`server/tests/test_webhooks.py` 末尾追加 3 个）

- `test_create_webhook_rejects_non_http_url`：`ftp://` URL → 400
- `test_create_webhook_rejects_missing_host`：`http://`（无 host）→ 400
- `test_fire_webhooks_never_propagates`：monkeypatch `events_from_add_response` 抛 `RuntimeError`，`_fire_webhooks` 不外抛

### 测试结果

| 阶段 | 命令 | 结果 |
|------|------|------|
| 覆盖测试 | `docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/test_webhooks.py -q` | **6 passed**（3 原有 + 3 新增） |
| 全量回归 | `docker compose -f deploy/mem0/docker-compose.yaml exec -T mem0-api python -m pytest tests/ -q` | **43 passed**（40 + 3 新增），仅 2 个预先存在的 deprecation warning |

修复文件均已 `docker compose cp` 至 mem0-api 容器 `/app/` 对应路径；Lint 无诊断错误。

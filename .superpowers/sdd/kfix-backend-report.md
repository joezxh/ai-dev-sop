# mem0 后端 Known Issues 修复报告（K1–K6）

日期：2026-09-17　|　范围：`mem0/server`（含 dashboard 一处注释）

## 测试结果

- 全量：`docker exec mem0-api pytest tests/ -q` → **92 passed, 0 failed**（修复前 69 collected，本次净增 23 个用例，满足 ≥75 passed 要求）
- 变异自查：临时把 `ACTION_METHOD_PATH["search"]` 的 method 从 POST 改为 GET → `test_stats.py` **5 failed**；恢复原文件后 **18 passed**。证明 SQL 计数用例对谓词真实敏感（fake db 对语句谓词做真实求值，非硬编码返回）。
- 运行方式：改动文件逐个 `docker compose cp` 进 mem0-api 容器对应路径后在容器内跑 pytest。

## 提交

| 哈希 | 说明 |
| --- | --- |
| `da573fda` | fix(server): webhook secret masking, HMAC signature, SSRF guard, delivery observability |
| `13f872a8` | refactor(server): single-source action map/auth types, SQL count aggregation, unified scan limit |

## 各项结论

### K1（webhook secret 不回显 + HMAC 签名）— ✅ 完成
- `routers/webhooks.py`：`WebhookResponse.secret` → `secret_masked`（前 4 + `***` + 后 4；长度 ≤8 全 `***`；None 透传）。新增 `WebhookCreatedResponse(WebhookResponse)`，创建响应额外返回一次性明文 `secret`（与 API Key 安全设计一致，创建后不可再取），列表接口仅返回掩码。
- `webhooks_client.py`：先把 payload `json.dumps(..., ensure_ascii=False, default=str)` 成 body 字符串，`httpx.post(content=body)`（保证签名与传输字节一致）；带 secret 时增加 `X-Webhook-Signature: hmac_sha256_hex(secret, body)`，保留 `X-Webhook-Secret` 兼容头。
- 测试：HMAC 捕获 headers 后本地重算比对；掩码列表/创建一次性明文/短 secret 全掩码等 6 个新用例。

### K2（webhook SSRF：默认拒绝私网）— ✅ 完成
- `_validate_url`：scheme/netloc 校验后，读 `WEBHOOK_ALLOW_PRIVATE_NET`（`1/true/yes` 放行，调用时读取便于测试）；host 为 IP 字面量直接判断，域名走 `socket.getaddrinfo` 解析，解析失败/OSError → 400 invalid host。
- 实现说明（偏离点）：判定采用 **任一** 解析结果 IP 属于 private/loopback/link-local 即拒绝（比"全部 IP 均私有才拒绝"更严格，可防 DNS 混合解析绕过）；任务给出的三组用例（127.0.0.1、169.254.169.254、公网域名）全部满足。
- 测试：loopback/metadata IP/私网域名（monkeypatch getaddrinfo）/不可解析域名 → 400；公网域名正常创建；环境变量放行私网 → 创建成功。

### K3（投递可观测性 + 线程池 + payload）— ✅ 完成
- 非 2xx 记 `logging.warning("Webhook %s delivered %s -> %s", name, url, status_code)`。
- 模块级 `ThreadPoolExecutor(max_workers=4, thread_name_prefix="webhook")` 替代每次新建 daemon 线程；`_fire_webhooks` 计算事件后 `executor.submit(_dispatch, events, payload)`，`_dispatch` 内完成 hooks 读取与投递；注释说明进程退出时在途任务丢失可接受（fire-and-forget）。
- payload：update/delete 触发改为 `{"event": "memory_updated|memory_deleted", "id": memory_id}`（`_fire_webhooks` 增加关键字参数 `events`/`payload`，add 路径保持 `{"results": [...]}`）。
- 测试：`test_fire_webhooks_never_propagates` 无需适配仍通过（events 计算仍在 try 内）；新增 executor.submit 捕获断言 payload 含 id、update/delete 路由函数级验证、非 2xx 日志断言。

### K4（事件白名单单源 + 空订阅拒绝）— ✅ 完成
- `ALLOWED_EVENTS = tuple(_EVENT_MAP.values())` 移至 `webhooks_client.py`，`routers/webhooks.py` 导入；`create_webhook` 校验 `not body.events → 400 "at least one event is required"`。
- 前端 `dashboard/.../webhooks/page.tsx` 在 `WEBHOOK_EVENTS` 上方加同步注释（跨语言无法共享单源）。
- 测试：空订阅 400、白名单派生正确性。

### K5（action 映射单源）— ✅ 完成
- `ACTION_METHOD_PATH`、`API_KEY_AUTH_TYPES` 移至 `routers/stats.py`；`requests.py` 从 stats 导入。`request_action` 改为由映射反向派生（`_METHOD_PATH_TO_ACTION`），语义与 SQL 谓词一致。
- 测试：新增 `test_action_map_is_single_source`（断言 `requests_router.ACTION_METHOD_PATH is stats_router.ACTION_METHOD_PATH`）；既有镜像覆盖用例全部通过，无断言导入路径的用例需要改。

### K6（上限统一 + stats SQL 计数）— ✅ 完成
- `SCAN_LIMIT = 10_000` 定义于 `server_state.py`；stats `_iter_memory_payloads` 默认值、export/delete 的 `_list_all_memories(limit=SCAN_LIMIT)` 与截断阈值统一用它；`ALL_MEMORIES_LIMIT` 保留为 API `top_k` 上界语义（admin 列表路径不变）。
- `test_exports.py` / `test_delete_project_memories.py` 的截断用例受影响，按计划同步：改为 monkeypatch `m.SCAN_LIMIT` 缩小阈值（保持用例快速且语义真实），`x-total-scanned` 用例语义保持。
- stats 计数改 SQL 聚合：按三个谓词（`auth_type.in_(...)` + `and_(method==, rtrim(path,"/")==)`）分别 `select(func.count())` 取计数，`total` 为三者之和；range=all 不再把全部行载入内存。
- `test_stats.py` fake db 改造：对语句 whereclause 做**真实谓词求值**（rtrim/method/path/auth_type in_/created_at >=，未识别谓词直接 fail），并支持 `.scalar()`（count）与 `.scalars().all()`（行查询）两种消费方式——变异自查证实改坏谓词用例会红。
- `request_action` 保留（已注明用途：ACTION_METHOD_PATH 的 Python 镜像，用于行级分类语义；生产计数路径已全部走 SQL）。

## 放弃项

无——六项全部完成，无缩减、无伪造通过。

## 备注

- 提交均在 mem0 子模块 `feat/llm-provider-i18n` 分支，`--no-verify` 按指示执行；工作区提交后干净。
- 容器内运行的 mem0-api 为热更新后的代码（cp 进容器），镜像重建后代码一致。

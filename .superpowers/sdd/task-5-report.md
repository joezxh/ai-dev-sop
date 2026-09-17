# Task 5 自审报告：GET /exports/memories JSONL 导出端点

## 状态：完成

## 交付内容

- `mem0/server/main.py`：新增纯过滤 helper `_filter_memories_for_export(rows, project_id, user_id, start, end)` 与端点 `GET /exports/memories`（StreamingResponse，media_type `application/x-ndjson`，`Content-Disposition: attachment; filename="memories-export-<ts>.jsonl"`）。端点位于 Task 3 的 `delete_memories_by_project` 之后。
- `mem0/server/tests/test_exports.py`：brief 的 3 个用例（逐字）+ 2 个补强用例（invalid start / invalid end → 400）。
- imports 补齐：`import json`、`from datetime import datetime, timezone`、`StreamingResponse`（追加进既有 `fastapi.responses` 行）；`time` 原本已有，未重复导入。

## TDD 过程

1. RED：仅复制测试进容器运行 → `ImportError: cannot import name '_filter_memories_for_export'`，符合预期。
2. GREEN：实现后 `pytest tests/test_exports.py -q` → **5 passed**。
3. 回归：`pytest tests/ -q` → **48 passed**（46 个既有/计划内用例全绿 + 本次新增的 2 个非法日期用例，无回归）。

## 语义核对

- 权限：`_auth is None`（管理员模式/AUTH_DISABLED/ADMIN_API_KEY）或 `_auth.role == "admin"` 放行，否则 403 —— 与 Task 3 `delete_memories_by_project` 完全一致。
- 依赖：复用 `_list_all_memories(limit=ALL_MEMORIES_LIMIT)`（返回 `{"results": [...]}`），`ALL_MEMORIES_LIMIT = 1000`。
- 日期：start/end 为 YYYY-MM-DD 字符串，`fromisoformat` 解析后 `.replace(tzinfo=timezone.utc)`；行内 `created_at` 兼容 `Z` 后缀（`.replace("Z", "+00:00")`）。

## 偏差（对 brief 的最小必要补强）

1. **[Rule 1] naive 行内时间补 UTC**：brief 的 `_ts` 直接返回 `fromisoformat` 结果，当行内 `created_at` 为 naive（如测试数据 `"2026-09-05T00:00:00"`）而 start/end 已被置为 aware 时，`ts < start_dt` 会抛 `TypeError: can't compare offset-naive and offset-aware datetimes` —— brief 自带的 `test_filter_by_user_and_date` 用例无法通过。修复：`_ts` 中对 `tzinfo is None` 的时间补 `timezone.utc`。
2. **[Rule 2] 非法 start/end → 400**：brief 代码中 `datetime.fromisoformat(start)` 抛 ValueError 会导致端点 500。在端点中用 `try/except ValueError → HTTPException(400, "invalid start/end date")` 包裹 helper 调用（仅包裹过滤调用，不影响其他异常路径），并补充 2 个测试用例。

## Concerns

- 无阻塞问题。备注：`_list_all_memories` 上限 1000 条，导出为该上限内的全量流式输出；`format` 参数仅支持 `jsonl`，其余 400。
- commit 在子模块分支 `feat/llm-provider-i18n` 上，父仓库 submodule 指针需随计划收尾一并推进。

## 证据

- Commit：`556cdb4d`（mem0 子模块，分支 `feat/llm-provider-i18n`）
- 提交信息：`feat(server): JSONL memory export with project/user/date filters`
- 变更：2 files changed, 120 insertions(+), 1 deletion(-)

# Task 2 Report: Project 表 custom_instructions / memory_expiration_date + add_memory 透传修复

- **状态:** DONE
- **提交:** `2a989800`（mem0 子模块，分支 `feat/llm-provider-i18n`）
  `feat(server): project-level custom instructions + expiration defaults; restore write params`
- **测试:** `tests/test_project_defaults.py tests/test_multitenant.py tests/test_configure.py tests/test_configure_dims.py -q`
  → **30 passed**（27 回归 + 3 新增），0 失败。

## 实现内容

### Step 1-2: TDD 红灯（先失败）
- 新建 `mem0/server/tests/test_project_defaults.py`（用例逐字取自 brief）。
- 复制进容器运行，确认预期失败：`ImportError: cannot import name '_apply_project_defaults' from 'main'`。

### Step 3: models.py + db.py 迁移
- `server/models.py`:
  - 顶部 `from datetime import date`；sqlalchemy import 行加入 `Date`。
  - `Project` 新增 `custom_instructions: Mapped[str | None]`（Text, nullable）与
    `memory_expiration_date: Mapped[date | None]`（Date, nullable）。
- `server/db.py`: `ensure_tenant_schema` 在原有 `users.department_id` 语句（含 DO $$ FK 块）之后
  追加两条幂等 ALTER（原语句未改动）：
  - `ALTER TABLE projects ADD COLUMN IF NOT EXISTS custom_instructions TEXT NULL`
  - `ALTER TABLE projects ADD COLUMN IF NOT EXISTS memory_expiration_date DATE NULL`

### Step 4: main.py
- 新增 `_load_project(project_id)`（按 brief 逐字）与 `_apply_project_defaults(memory_create, project)`（逐字）。
- `add_memory`: 在 `metadata["project_id"] = project_id` 之后加入
  `overrides = _apply_project_defaults(memory_create, _load_project(project_id) if project_id else None)`。
- `params` 块替换为带 `infer/memory_type/prompt/expiration_date` 的完整块（逐字），修复透传回归。
- `_scope_identifiers`、`_filter_by_project_id`、git_remote 解析逻辑未改动。

### Step 5: routers/projects.py 字段暴露
- `from datetime import date` 导入。
- `ProjectCreate` / `ProjectUpdate` 新增 `custom_instructions`、`memory_expiration_date`（str | None，YYYY-MM-DD）。
- `ProjectResponse` 新增两字段；`_to_response` 输出 `custom_instructions` 与 `memory_expiration_date.isoformat() or None`。
- 新增 `_parse_expiration_date` 助手：`date.fromisoformat(v)`，`ValueError` → HTTP 400。
- `create_project` / `update_project` 均写入新字段（update 沿用既有的 `is not None` 惯例，与 `git_remotes` 处理一致）。

## 验证

| 项 | 结果 |
| --- | --- |
| 红灯（ImportError） | 确认 |
| 新增测试 3 条 | 通过 |
| 回归 27 条（multitenant / configure / configure_dims） | 通过 |
| 容器内文件同步 | models.py / main.py / db.py / routers/projects.py / 新测试均已 cp 进 mem0-api |

## 自审 / Deviations

- 无偏离：所有代码块按 brief 逐字落地。
- 备注 1: `update_project` 采用 `is not None` 判定（Pydantic 无法区分"未传"与显式 null），与文件内既有字段惯例一致，不视为偏离。
- 备注 2: `_apply_project_defaults` 中 expiration 用 `is None` 判定、prompt 用 falsy 判定，均按 brief 语义实现。
- 备注 3: `update_project` 无法通过传 `"memory_expiration_date": null` 清空日期（brief 未要求，与 git_remotes 行为一致）。若需清空能力可后续在 ProjectUpdate 上加 sentinel 处理。
- 无 stub、无未跟踪残留文件（子模块工作区干净）。

## 文件清单

- Modify: `mem0/server/models.py`
- Modify: `mem0/server/db.py`
- Modify: `mem0/server/main.py`
- Modify: `mem0/server/routers/projects.py`
- Create: `mem0/server/tests/test_project_defaults.py`

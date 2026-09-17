---
phase: task-10
reviewed: 2026-02-19T00:00:00Z
depth: deep
files_reviewed: 6
files_reviewed_list:
  - mem0/server/routers/projects.py
  - mem0/server/tests/test_project_defaults.py
  - mem0/server/models.py
  - mem0/server/main.py
  - mem0/server/dashboard/src/app/(root)/dashboard/projects/page.tsx
  - mem0/server/dashboard/src/components/ui/delete-confirmation-modal.tsx
findings:
  critical: 1
  warning: 1
  info: 3
  total: 5
status: issues_found
---

# Task 10 第三轮修复 复审报告（CR-01 / WR-01 / WR-02）

**基准：** `95c0942b`　**提交：** `4531d1e5`（server）、`4a4a7196`（dashboard）
**结论：`Spec: ❌ / Quality: Changes Requested`** —— CR-01 的 `name` 分支确已闭环，但同一缺陷类在 `git_remotes` 上存活，仍会产生未捕获 `IntegrityError` → HTTP 500。

## 验收结论

| # | 验收项 | 结论 | 证据 |
| - | ------ | ---- | ---- |
| 1 | `update_project` 对 `name` 为 `null` / 全空白返回 422，不触发 500 | **✅ 通过** | `projects.py:151-155` 校验在 `db.commit()`（:160）前；测试 `test_project_defaults.py:78-101` 断言 `status_code == 422`、`db.committed is False`、`db.project.name == "n"` —— 断言真实，非伪造通过（若实现回退，断言必失败） |
| 2 | `custom_instructions` / `description` 保存前 `.trim()`，空白串 → `null` | **✅ 通过** | `page.tsx:86,91`；`name` 亦 trim 并前置拦截（:78-81） |
| 3 | Danger Zone truncated 提示与成功分支同为英文 | **✅ 通过** | `page.tsx:148-155`，两条 toast 全英文；`use-toast.ts:8,33-34` 新增 `warning` 变体，sonner `^1.4.41` 支持 `.warning` |
| 4 | 上一轮 3 项未破坏 | **✅ 通过** | ① `exclude_unset` 显式 null 可清空：`projects.py:150`（未回退）；② `truncated` 返回：`main.py:807` + 路由 `response_model=None`（:976）未被裁剪；③ purging 禁用态：`page.tsx:141-142,160` + `delete-confirmation-modal.tsx:40,74-76` |

**未推翻报告中的 55 passed。**

## Critical Issues

### CR-01a：`NON_NULLABLE_FIELDS` 漏掉 `git_remotes`，`{"git_remotes": null}` 仍 → IntegrityError 500

**File:** `mem0/server/routers/projects.py:20`（常量）、`:35`（`ProjectUpdate.git_remotes`）、`:150-159`（循环）
**关联：** `mem0/server/models.py:109`

常量注释自称覆盖「Fields mapped to NOT NULL columns」，但实际只列了 `name`。`Project` 里还有第二个非 Optional 且可更新的列：

```python
# models.py:109 —— Mapped[list] 非 Optional ⇒ nullable=False
git_remotes: Mapped[list] = mapped_column(JSON, default=list)
```

`ProjectUpdate.git_remotes: list[str] | None = None` 允许 `null`，而 `git_remotes` 不在守卫集合内，于是：

```
PUT /projects/{id}  {"git_remotes": null}
  → setattr(project, "git_remotes", None)   # projects.py:159
  → db.commit()                             # projects.py:160
  → sqlalchemy.exc.IntegrityError (NOT NULL)
  → 未捕获，且 projects.py 无 try/except IntegrityError，main.py 仅注册
    RateLimitExceeded / UpstreamError handler（main.py:212-213）
  → HTTP 500
```

这是与 CR-01 **完全同源**的缺陷（同一次 500、同一条 setattr→commit 路径）。表由 `Base.metadata.create_all` 建立（alembic 目录中无 `projects` 表迁移，`git_remotes` 零命中），故 DDL 直接取自模型 ⇒ `NOT NULL` 成立。

**Fix：**

```python
# projects.py:20
NON_NULLABLE_FIELDS = {"name", "git_remotes"}

# projects.py:150-159 —— 先全量校验，再写入（同时消除"中途抛错已 setattr"的隐患）
updates = body.model_dump(exclude_unset=True)
for field, value in updates.items():
    if field in NON_NULLABLE_FIELDS and value is None:
        raise HTTPException(status_code=422, detail=f"{field} cannot be null")
    if field in NON_NULLABLE_FIELDS and field != "git_remotes" and not str(value).strip():
        raise HTTPException(status_code=422, detail=f"{field} cannot be blank")
for field, value in updates.items():
    if field == "memory_expiration_date":
        setattr(project, field, _parse_expiration_date(value))
    else:
        setattr(project, field, value)
```

并补一条测试：`ProjectUpdate(git_remotes=None)` 期望 422。

## Warnings

### WR-03：`test_update_project_allows_nullable_fields` 是空断言，无法锁住上一轮的「显式 null 可清空」

**File:** `mem0/server/tests/test_project_defaults.py:104-116`

`_fake_project()`（:45-58）的 `description` 与 `custom_instructions` 初始值**本来就是 `None`**，测试随后断言 `is None`。因此即使把实现回退成 `model_dump(exclude_unset=True, exclude_none=True)` 或加回 `if value is not None: continue`（即上一轮修复 #1 被完全撤销），该测试**仍然通过**。它名义上是上一轮修复的回归网，实际是空网。

（同一个测试对「未显式传 `name` 时不误报 422」这一方向是有效的，问题只在「清空」方向。）

**Fix：**

```python
def test_update_project_allows_nullable_fields():
    import routers.projects as rp
    proj = _fake_project()
    proj.description = "old desc"
    proj.custom_instructions = "old prompt"
    db = _FakeDB(project=proj)
    rp.update_project(
        "p",
        rp.ProjectUpdate(description=None, custom_instructions=None),
        _admin=None, db=db,
    )
    assert db.committed is True
    assert proj.description is None           # 真实清空，而非"本来就是 None"
    assert proj.custom_instructions is None
```

## Info

### IN-01：`create_project` 只校验不归一化，`name` 未 trim 且与 `project_id` 状态码口径不一致

**File:** `mem0/server/routers/projects.py:100-112`

`project_id` 走 `.strip()` 且空白返回 **400**，`name` 走 `.strip()` 判空返回 **422** 但写入的是未 trim 的 `body.name`（:108）。同函数内两套状态码、两套归一化策略；`" Foo "` 会原样落库。UPDATE 路径同样不 trim（`setattr(project, field, value)`，:159），规范化目前完全依赖前端。

**Fix：** `create_project` 写 `name=body.name.strip()`，并把 `project_id` 空白也统一为 422（或两者统一为 400），使创建与更新语义一致。

### IN-02：守卫测试未覆盖 HTTP 层

**File:** `mem0/server/tests/test_project_defaults.py:78-101`

4 条新测试均直接调用 `rp.update_project(...)`（绕过路由）。若将来给 `PUT /projects/{id}` 加 `response_model` 转换或全局异常处理器把 422 改写，测试仍绿而线上行为已变。`test_update_project_rejects_blank_name` 也缺少 `db.project.name == "n"` 断言（null 用例有，blank 用例没有）。

**Fix：** 用 `fastapi.testclient.TestClient` 补一条 `PUT` → `assert r.status_code == 422` 的端到端用例。

### IN-03：`projects/page.tsx` 表单标签仍为中英混排（既有，非本次引入）

**File:** `mem0/server/dashboard/src/app/(root)/dashboard/projects/page.tsx:361,367,371-372`

`Custom Instructions（记忆抽取规则）`、`Memory Expiration Date（项目默认过期日，可选）`、占位符`例如：只存储技术偏好与项目约定，忽略闲聊` 为硬编码中英混排，而同页其它标签走 `t("tenant.*")` i18n。

Danger Zone（:395-409）与两条 toast（:148-155）**已全英文**，WR-01 范围内存量中文已清零，此项仅作提示，不阻断验收。

---

_Reviewed: 2026-02-19_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_

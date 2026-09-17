---
phase: task-10-projects-enhancement-fix
reviewed: 2026-09-16T04:03:39+08:00
depth: standard
commit: 95c0942b / 8f413045
diff_base: 44ffe074
files_reviewed: 6
files_reviewed_list:
  - server/routers/projects.py
  - server/main.py
  - server/dashboard/src/app/(root)/dashboard/projects/page.tsx
  - server/dashboard/src/components/ui/delete-confirmation-modal.tsx
  - server/dashboard/src/components/ui/use-toast.ts
  - server/tests/test_delete_project_memories.py
  - server/tests/test_project_defaults.py
findings:
  critical: 1
  warning: 2
  info: 3
  total: 6
status: issues_found
---

# Task 10 修复复审报告（Projects 页增强 — Important ×3）

**Reviewed:** 2026-09-16T04:03:39+08:00
**Commit:** `95c0942b` + `8f413045`（基准 `44ffe074`，mem0 仓库，分支 `feat/llm-provider-i18n`）
**Depth:** standard
**Files Reviewed:** 7（6 个改动文件 + `models.py` 作为约束事实基准）
**Status:** issues_found

## Summary

3 项 Important **功能目标全部落地**（逐字段赋值 / 显式 null 清空 / `truncated` 上报 / 前端 `|| null` / `purging` 禁用态 / 两个新测试用例），无一处虚报。

但 `update_project` 的通用 `setattr` 循环**把一个原本被 `is not None` 守卫住的 NOT NULL 字段暴露成了崩溃入口**：`{"name": null}` 现在会写 NULL 进 `projects.name`（`Mapped[str]` → NOT NULL），`db.commit()` 抛未捕获的 `IntegrityError` → **HTTP 500**。已用独立探针实测复现（SQLAlchemy 2.0.49）。这是本次改动新引入的，旧代码不可能触发。判 BLOCKER。

另有 2 个 WARNING：`.trim()` 被顺手删掉导致纯空白的 custom instructions 会被落库并当作抽取 prompt；以及新增的 toast 文案硬编码中文、绕过页面已有的 `t()` i18n 体系。

**结论：Spec: ❌（存在 1 个 BLOCKER 级新增缺陷） / Quality: Request changes**

---

## 逐项核验

### 1 · `update_project` 逐字段赋值 + 显式 null 清空 ✅（但见 CR-01）

`server/routers/projects.py:143-147`：

```python
for field, value in body.model_dump(exclude_unset=True).items():
    if field == "memory_expiration_date":
        setattr(project, field, _parse_expiration_date(value))
    else:
        setattr(project, field, value)
```

- `exclude_unset=True` 语义正确：FastAPI 用 `ProjectUpdate(**body)` 构造，显式提供的字段才进 `model_fields_set`，未提供的字段不会被写回默认值。✅
- `_parse_expiration_date(None) is None` ✅（`projects.py:58-59`，早于 `date.fromisoformat` 返回，清空路径成立）
- 非法日期 400 ✅（`projects.py:62-65`）
- 合法日期正常 ✅
- 显式 `null` 清空 `description` / `custom_instructions` / `memory_expiration_date` ✅ — 三列均为 `Mapped[... | None]`（`models.py:108,111,112`），可空。
- 显式 `null` 清空 `git_remotes` ✅ — 列虽为 `Mapped[list]`（NOT NULL），但 SQLAlchemy `JSON` 默认 `none_as_null=False`，落库为 JSON 标量 `null`，读回为 Python `None`，`_to_response` 的 `list(project.git_remotes or [])`（`projects.py:74`）与 `gitmatch.py:64,102,105` 的 `or []` 均能兜住。探针已验证 SQLite 下写入成功且回读 `null`。
- **既有调用方不会意外清空** ✅ — 全仓检索 `PUT /projects/{id}` 调用方：`mem0/mem0/client/project.py` 走的是 hosted 平台 `PATCH /api/v1/orgs/.../projects/{id}/`（另一套 API），`mcp_server.py` 只调 `/projects/match`。dashboard 是唯一调用方，且 `page.tsx:79-88` 恒定发送全部 6 个字段。

⚠️ 但 `name` 是第 5 个字段，且**不可空** —— 见 CR-01。

### 2 · `delete_memories_by_project` 返回 `truncated` ✅

`server/main.py:797-807`：`_list_all_memories(limit=ALL_MEMORIES_LIMIT)`（`main.py:723-726`，`vector_store.list(top_k=1000)`）最多返回 1000 行，`truncated = len(rows) >= ALL_MEMORIES_LIMIT` 在扫描条数达到上限时为 `true`。`scanned = len(rows)` 如实上报。✅

配套测试 `test_reports_truncation`（`test_delete_project_memories.py:47-54`）与更新后的 `test_deletes_only_matching`（断言 `{"deleted": 1, "scanned": 3, "truncated": False}`）✅。

### 3 · 前端保存载荷 / Danger Zone 提示 / `purging` 禁用态 ✅（但见 WR-01 / WR-02）

- 载荷统一 `|| null`：`page.tsx:82`（description）、`86`（custom_instructions）、`87`（memory_expiration_date）；`git_remotes` 恒为数组（`83`）。✅
- 权宜注释已移除 ✅ —— 旧注释（解释「为何发空字符串而不是 null」）已被替换；`page.tsx:84-85` 是解释新语义的说明性注释，不是权宜注释。
- `truncated` 时警告文案 ✅（`page.tsx:143-147`），且用 `res.data?.scanned` 而非硬编码 1000，与用户要求一致。
- `purging` 禁用态 ✅ —— `delete-confirmation-modal.tsx:74` `disabled={!isDeleteEnabled || loading}`、`76` 文案切换 `Deleting…`、`40` `if (loading) return` 双保险、`page.tsx:136` `if (!toPurge || purging) return`、`155` `finally { setPurging(false) }` 无泄漏。
- `use-toast.ts` 增加 `warning` 变体并映射到 `sonnerToast.warning` ✅（sonner `^1.4.41` 自 1.4 起提供 `warning`，无新依赖）。

### 4 · 测试用例与构建 ✅（采信报告）

- `test_parse_expiration_date_none_is_none`（`test_project_defaults.py:31-33`）与 `test_parse_expiration_date_bad_value_raises_400`（`:36-42`）均存在且断言正确。✅
- 全量 pytest 51 passed、dashboard 重建通过：采信报告，未复跑。

---

## Critical Issues

### CR-01: 显式 `{"name": null}` 写入 NOT NULL 列 → 未捕获 IntegrityError → HTTP 500

**File:** `server/routers/projects.py:143-147`（配合 `:28` `name: str | None = None`）
**Severity:** BLOCKER

**Issue:**
`ProjectUpdate.name` 的类型是 `str | None = None`（`projects.py:28`），即 **API 契约公开允许 `null`**。旧代码 `if body.name is not None: project.name = body.name` 让 `null` 退化成 no-op；改成通用 `setattr` 后，`null` 会被直接赋给 ORM 对象并在 `db.commit()` 时落库。

`Project.name` 是 `Mapped[str] = mapped_column(String(255))`（`models.py:107`），非 Optional → SQLAlchemy 2.0 推导 `nullable=False`，DDL 为 `name VARCHAR(255) NOT NULL`。

实测复现（SQLAlchemy 2.0.49，SQLite 探针，模拟 `setattr` + `commit`）：

```
[('name', False), ('git_remotes', False), ('desc', True)]
name=None -> REJECTED: IntegrityError: (sqlite3.IntegrityError) NOT NULL constraint failed: p.name
```

`IntegrityError` 无异常处理器，`get_db`（`db.py:25-31`）只在 `finally` 里 `close()`，异常直接冒泡 → FastAPI 返回 **500**，且 `updated_at` 之前的写入全部回滚。

**触发条件：** 任何非 dashboard 调用方（curl / 自研脚本 / MCP / 未来的第二个前端）在 partial update 时按 pydantic 契约发 `{"name": null}`（或 `{"name": null, "description": "x"}` 这类标准 PATCH 语义载荷）。报告自身就是用 curl 直接打 API 做验证的，说明直连 API 是真实使用模式，不是假想场景。

**Fix:**
在循环内对不可空字段保留守卫，或把校验提到循环之前：

```python
NON_NULLABLE = {"name"}  # projects.name / git_remotes 不接受 NULL

for field, value in body.model_dump(exclude_unset=True).items():
    if field == "memory_expiration_date":
        setattr(project, field, _parse_expiration_date(value))
    elif field in NON_NULLABLE and value is None:
        raise HTTPException(status_code=422, detail=f"{field} cannot be null.")
    else:
        setattr(project, field, value)
```

若希望保持「null = 忽略」的兼容语义（对老客户端更安全），则改为 `if field in NON_NULLABLE and value is None: continue`。二选一，但必须二选一 —— 当前实现会让合法载荷产生 500。

> 备注：如果团队明确接受「`PUT /projects/{id}` 的唯一契约消费者就是 dashboard，不做外部兼容」，此项可降级为 WARNING；但即便如此，两行守卫的成本远低于一次线上 500 的排查成本，建议直接修掉。

---

## Warnings

### WR-01: Danger Zone 警告文案硬编码中文，绕过页面已有的 i18n

**File:** `server/dashboard/src/app/(root)/dashboard/projects/page.tsx:145`
**Severity:** WARNING

**Issue:**
```tsx
title: `已删除 ${deleted} 条，但仅扫描前 ${res.data?.scanned ?? 0} 条，可能未删完，请重试`,
```
同一文件里其他所有用户可见文案都走 `t()`（`:179,180,220-222,296,305,309,310,322,332,336,345,353,378,380`），`src/i18n/en.ts` / `zh.ts` 也都有 `tenant.*` 命名空间。这条新文案是本次修复新加的，且是**唯一**一条硬编码中文 toast —— 在英文界面下会突然弹出中文；同一函数里的成功分支（`:149`）仍是英文 `"Deleted N memories"`，同一个操作两种语言。

**Fix:**
加到 `en.ts` / `zh.ts`（如 `tenant.purgeTruncated`），页面用插值：

```tsx
toast({
  title: t("tenant.purgeTruncated", { deleted, scanned: res.data?.scanned ?? 0 }),
  variant: "warning",
});
```

> 附注（非本次引入，仅记录）：`:356` `Custom Instructions（记忆抽取规则）`、`:362` placeholder、`:367` `Memory Expiration Date（项目默认过期日，可选）` 同样是中英混排硬编码，属 Task 10 基线圈层，建议与 WR-01 一起收口。

### WR-02: 删除 `.trim()` 导致纯空白 custom instructions 被落库并当作抽取 prompt

**File:** `server/dashboard/src/app/(root)/dashboard/projects/page.tsx:86`
**Severity:** WARNING

**Issue:**
diff 把 `customInstructions.trim() || null` 改成了 `customInstructions || null`。纯空白字符串（`"   "`）在 JS 里是 truthy，因此会被发送到后端；后端 `_apply_project_defaults` 用的正是**真值判断**（`main.py:617` `if not memory_create.prompt and project.custom_instructions:`），于是 `"   "` 会被当作有效 prompt 覆盖进记忆抽取请求。

也就是说：用户在 Custom Instructions 里只敲几个空格再保存，UI 上看起来是空的，但后端从此给该项目所有记忆抽取拼上一个空白 prompt。这是本次修复顺手引入的回退，旧代码（`.trim() || null`）不会。

**Fix:**
```tsx
custom_instructions: customInstructions.trim() || null,
```
（`description: description || null` 同理建议加 `.trim()`；`expirationDate` 来自 `<input type="date">`，不会含空白，可保持不变。）

---

## Info

### IN-01: 400 错误信息被静默改写，且与校验实际强度不符

**File:** `server/routers/projects.py:65`
**Issue:** `detail` 从 `"memory_expiration_date must be in YYYY-MM-DD format."` 改成 `"memory_expiration_date must be YYYY-MM-DD"`。这是一处未在报告中声明的对外契约变更（报告只提到「解析失败 → 400」，未提消息改写）。另外 Python 3.11+ 的 `date.fromisoformat` 会接受 `20270101`、`2027-W01-1` 等更多 ISO 形式，所以「must be YYYY-MM-DD」这句断言并不严格成立。
**Fix:** 若需要精确格式校验，改用 `datetime.strptime(value, "%Y-%m-%d")`；否则至少在报告里显式记录消息变更，避免依赖旧文案的客户端/文档失配。

### IN-02: 请求进行中 Cancel / Esc / 遮罩关闭仍未禁用

**File:** `server/dashboard/src/components/ui/delete-confirmation-modal.tsx:41-43, 48, 68-70`
**Issue:** 只禁用了确认按钮（`:74`），但 `handleClose`（Cancel 按钮、X、Esc、点遮罩 → `:48` 的 `onOpenChange`）在 `loading` 期间仍然可用。用户可在删除请求飞行中关掉弹窗，从而看不到任何进行中反馈（toast 仍会弹，但上下文已消失）。另外 `handleConfirm`（`:39-43`）在 `onConfirm()` 后立即 `setConfirmationText("")`，若请求失败需要重试，用户必须重新输入一遍 project_id。
**Fix:** `handleClose` 增加 `if (loading) return;`；`<Button ... disabled={loading}>` 给 Cancel；`onOpenChange={(o) => { if (!o && !loading) handleClose(); }}`。

### IN-03: 核心行为（显式 null 清空）没有端到端测试，只有解析函数单测

**File:** `server/tests/test_project_defaults.py:31-42`
**Issue:** 新增的两个用例只覆盖 `_parse_expiration_date` 这个纯函数，`update_project` 的 `exclude_unset` 语义（未提供的字段不被写回、显式 null 被写回）**零覆盖**。这正是本次修复的核心行为，一旦有人把 `exclude_unset=True` 改回 `exclude_unset=False` 或改回 `is not None` 守卫，测试仍全绿。CR-01 之所以能溜过 51 passed，也是同一个盲区。
**Fix:** 用内存 SQLite + `TestClient` 加一个用例：先 POST 建项目 → `PUT {"description": null, "custom_instructions": null, "memory_expiration_date": null}` → 断言响应中三个字段均为 `null` 且 `name` 未变；再加一个 `PUT {"custom_instructions": "x"}` → 断言 `description` 仍为原值（验证 `exclude_unset`）。

---

_Reviewed: 2026-09-16T04:03:39+08:00_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

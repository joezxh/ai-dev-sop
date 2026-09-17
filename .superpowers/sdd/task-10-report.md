# Task 10 Report — Projects 页增强（Custom Instructions / Expiration / Danger Zone）

## 状态

**完成（Done）**，含 1 处偏离（见下）。

> 后续追加：审查发现的 3 个 Important 问题已全部修复，见文末「[补充：审查问题修复（Important ×3）](#补充审查问题修复important-3)」，其中 Concern A / B 已关闭。

## 提交

| Hash | Message |
| ---- | ------- |
| `44ffe074` | `feat(dashboard): project custom instructions, expiration, danger zone` |
| `effaedd1` | `fix(dashboard): send empty string to clear project custom instructions` |

分支：`feat/llm-provider-i18n`（worktree）
改动文件（2 个，+70/-1 → +73/-1）：

- `mem0/server/dashboard/src/types/api.ts`
- `mem0/server/dashboard/src/app/(root)/dashboard/projects/page.tsx`

## 实现内容

### 1. 类型扩展 — `types/api.ts`

`Project` 增加两个字段（带注释说明语义）：

```ts
/** Prompt appended when extracting memories for this project. Null = unset. */
custom_instructions: string | null;
/** Default expiration for memories in this project, "YYYY-MM-DD". Null = never. */
memory_expiration_date: string | null;
```

### 2. 弹窗表单 — `projects/page.tsx`

- 新增 state：`customInstructions`、`expirationDate`
- `openEdit`：`p.custom_instructions ?? ""` / `p.memory_expiration_date ?? ""`
- `openCreate`：两字段重置为 `""`
- `save` body 增加两字段
- 在 git-remotes 字段之后追加 Custom Instructions（`Textarea`，rows=3）与 Memory Expiration Date（`Input type="date"`）
- 沿用既有 shadcn `Label` / `Textarea` / `Input`，未改动页面结构

### 3. Danger Zone — Delete All Memories

- 新增 state `toPurge`，与既有项目删除的 `toDelete` 相互独立
- 表格 Actions 单元格、Edit 之前新增危险按钮（复用 `text-onSurface-danger-primary`）
- 复用 `DeleteConfirmationModal`（输入 project_id 二次确认）
- handler `deleteAllMemories()`：

```ts
await api.delete(`${MEMORY_ENDPOINTS.BASE}?project_id=${encodeURIComponent(toPurge.project_id)}`);
toast({ title: `Deleted ${res.data?.deleted ?? 0} memories`, variant: "success" });
```

- 失败走页面既有 `setError(e?.response?.data?.detail ?? ...)` 通道
- 用 `MEMORY_ENDPOINTS.BASE` 而非硬编码字符串，与其他页面约定一致

## 验证结果

| # | 检查项 | 结果 |
| - | ------ | ---- |
| 1 | `docker compose up -d --build mem0-dashboard` | `Built` + `Recreated` + `Started`，无 error |
| 2 | TS 类型检查 | 通过（`next.config.mjs` 中 `typescript.ignoreBuildErrors: false`，构建即类型检查） |
| 3 | `/dashboard/projects` 路由 | 307（未登录重定向，非 500），路由正常解析 |
| 4 | 两字段往返持久化 | POST → GET 均正确返回 `custom_instructions` / `memory_expiration_date` |
| 5 | `DELETE /memories?project_id=<empty project>` | `{"deleted": 0}` ✅ 与 Step 4 预期一致 |
| 6 | `DELETE /memories?project_id=nonexistent` | `404 {"detail":"Project not found."}` ✅ |
| 7 | 容器日志 | `✓ Ready in 286ms`，无运行时报错 |
| 8 | lint | 0 diagnostics |

验证用的临时项目 `t10-verify` 已清理，未残留数据。

## 偏离与 Concerns

### 偏离 1（Rule 1 — Bug）：清空 Custom Instructions 发送 `""` 而非 `null`

**发现过程**：按 brief 原样发送 `custom_instructions: null` 后，后端 `PUT /projects/{id}` 返回体中该字段**仍是旧值**——即「清空保存」静默失效。

**根因**（`mem0/server/routers/projects.py:141-150`）：后端 `update_project` 是 partial-update 语义，对每个字段都是 `if body.X is not None:` 才赋值。因此 `null` 表示「未提供」，是 no-op，无法清空。

**实测矩阵**：

| 发送值 | custom_instructions | memory_expiration_date |
| ------ | ------------------- | ---------------------- |
| `null` | no-op（保留旧值）❌ | no-op（保留旧值）❌ |
| `""`   | **清空成功** ✅     | `400 memory_expiration_date must be in YYYY-MM-DD format.` ❌ |

**修复**：`custom_instructions: customInstructions.trim()`（空则发 `""`）。`""` 在下游 `_apply_project_defaults` 中与 `null` 等价（同为 falsy，不触发 prompt 覆盖），无副作用。已在代码中加注释说明原因。

### Concern A（需后端跟进）：过期日无法清空

`memory_expiration_date` 目前**没有任何可清空的方式**：`""` 报 400，`null` 是 no-op。用户一旦设置过期日，前端无法取消。
建议后端改为 `model_dump(exclude_unset=True)` 或引入显式清空标记（如 `""` → None）。**此限制不影响本次新增之外的任何功能，且非本任务文件范围，故未改后端。**

### Concern B（`description` 同样无法清空 — 既有问题，未修）

同一个 partial-update 根因导致：清空 Description 后保存也会静默保留旧值。这是**既有行为**（非本次引入），按 scope boundary 未一并修改，但建议与 Concern A 一起由后端统一处理。

### Concern C：权限提示

`DELETE /memories?project_id=` 为 **admin only**（后端校验 `role != "admin"` → 403）。前端未做角色判断，非 admin 点击后会通过 `setError` 显示后端返回的 `Admin role required.`。功能正确，但按钮对非 admin 未隐藏/禁用——如需更严谨，可后续按 `useAuth` 的 role 做隐藏。

### Concern D：分支命名

本 worktree 位于 `feat/llm-provider-i18n`（非 `worktree-agent-*` 命名空间）。经确认该分支已有本计划前置任务提交（如 `322ba5a6 fix(dashboard): correct playground admin-only wording`），且非 main/master 等受保护分支，判定为本计划的既定工作分支，故正常提交。

## 备注

- 未重写页面结构，完全沿用既有 shadcn Dialog / Card / 表格 / 分页 / DeleteConfirmationModal 模式。
- 两个 `DeleteConfirmationModal` 实例并列存在（项目删除 + 清空记忆），状态独立，互不干扰。

---

## 补充：审查问题修复（Important ×3）

**状态：完成（Done）**，2 条提交，测试全绿。

### 提交

| Hash | Message | 范围 |
| ---- | ------- | ---- |
| `95c0942b` | `fix(server): honor explicit nulls in project update and report delete truncation` | `server/routers/projects.py`、`server/main.py`、`server/tests/test_project_defaults.py`、`server/tests/test_delete_project_memories.py` |
| `8f413045` | `fix(dashboard): clear project fields via null and warn on truncated purge` | `server/dashboard/src/app/(root)/dashboard/projects/page.tsx`、`.../ui/delete-confirmation-modal.tsx`、`.../ui/use-toast.ts` |

### 修复 1 & 2：字段无法清空（`routers/projects.py`）

- `_parse_expiration_date`：`None` → `None`（清空）；接受 `date` 实例；其余走 `date.fromisoformat(str(value))`，解析失败 → `400 memory_expiration_date must be YYYY-MM-DD`。
- `update_project` 改为 `for field, value in body.model_dump(exclude_unset=True).items()` 显式赋值，`memory_expiration_date` 走 `_parse_expiration_date`，其余直接 `setattr`。保留 404 与项目查找逻辑，`ProjectUpdate` 字段不变。

效果：`null` 现在能真正清空 `custom_instructions` / `memory_expiration_date` / `description`，`""` 对日期字段仍返回 400（前端不再发送）。**关闭 Concern A 与 Concern B**。

### 修复 3：Danger Zone 漏删却谎报成功（`main.py`）

`delete_memories_by_project` 返回体增加扫描窗口信息：

```python
return {"deleted": deleted, "scanned": len(rows), "truncated": len(rows) >= ALL_MEMORIES_LIMIT}
```

前端按 `truncated` 分流：`true` → 警告文案「已删除 N 条，但仅扫描前 M 条，可能未删完，请重试」；否则维持 `Deleted N memories`。

### 前端配套改动

- `save` 载荷统一为 `custom_instructions: customInstructions || null`、`memory_expiration_date: expirationDate || null`，移除原「无法清空」的权宜注释。
- 新增 `purging` state：请求期间禁用确认按钮（`DeleteConfirmationModal` 新增可选 `loading` 属性，按钮 `disabled={!isDeleteEnabled || loading}`、文案变 `Deleting…`，`handleConfirm` 增加 `if (loading) return` 双保险）。
- `toast` 支持 `warning` 变体并映射到 `sonnerToast.warning`（原 `ToastVariant` 仅有 default/destructive/success，直接用 `"warning"` 会类型报错、构建失败）。

### 偏离（2 处，均为 Rule 1/3 自动修复）

1. **既有测试断言失效**：`tests/test_delete_project_memories.py::test_deletes_only_matching` 原断言 `out == {"deleted": 1}`，新增返回字段后会失败 → 更新为 `{"deleted": 1, "scanned": 3, "truncated": False}`，并新增 `test_reports_truncation` 覆盖 `truncated=True` 场景。故该文件一并纳入后端提交。
2. **`warning` toast 变体缺失**：`use-toast.ts` 的 `ToastVariant` 不含 `"warning"`，按计划使用会 TS 类型错误导致 `next build` 失败 → 扩展类型并映射到 sonner `warning`。

### 验证结果

| # | 检查项 | 结果 |
| - | ------ | ---- |
| 1 | `python -m pytest tests/ -q`（容器内，镜像重建后复跑） | **51 passed, 0 failed** |
| 2 | `docker compose up -d --build mem0-dashboard` | `mem0-api Built` + `mem0-dashboard Built` / `Recreated` / `Started`，无 error |
| 3 | 容器状态 | `mem0-api Up`、`mem0-dashboard Up`；日志 `✓ Ready in 254ms`，无运行时报错 |
| 4 | lint（6 个改动文件） | 0 diagnostics |

新增用例：`test_parse_expiration_date_none_is_none`、`test_parse_expiration_date_bad_value_raises_400`、`test_reports_truncation`。

---

## 复审修复（1 BLOCKER + 2 WARNING）

| 提交 | 哈希 | 说明 |
| ---- | ---- | ---- |
| `fix(server): reject null name in project update (NOT NULL guard)` | `4531d1e5` | `server/routers/projects.py`、`server/tests/test_project_defaults.py` |
| `fix(dashboard): trim project fields and unify purge warning wording` | `4a4a7196` | `server/dashboard/src/app/(root)/dashboard/projects/page.tsx` |

### CR-01（BLOCKER，后端）：`{"name": null}` → IntegrityError 500

模块级新增 `NON_NULLABLE_FIELDS = {"name"}`，`update_project` 的逐字段 `setattr` 循环内前置校验：`None` → `422 name cannot be null`；全空白字符串 → `422 name cannot be blank`。校验发生在 `db.commit()` 之前，不会留下半写状态。

同时在 `create_project` 补 `name` 空白校验（422），避免创建路径写入空名——与 UPDATE 守卫保持同一语义。

### WR-02（前端）：全空白 Custom Instructions 落库

`save()` 载荷改为 `custom_instructions: customInstructions.trim() ? customInstructions.trim() : null`，`description` 同样 `.trim()` 后判空取 `null`。配套：`name` 改发 `name.trim()`，并在提交前拦截 `!name.trim()` → `setError("name is required")`（后端现在会 422，前端先给出明确提示）。

### WR-01（前端文案混排）：Danger Zone 警告统一为英文

`已删除 N 条，但仅扫描前 M 条，可能未删完，请重试` → `Deleted N memories (only the first M were scanned — run again to finish)`，与成功分支 `Deleted N memories` 一致（保留 `warning` 变体）。全仓已无该中文串残留。

### 追加测试（`server/tests/test_project_defaults.py`）

构造最小 `_FakeDB`（`scalar`/`commit`/`refresh`）+ `_fake_project()`（补齐 `id`/`created_by`/`created_at`/`updated_at` 以走通 `_to_response`）：

- `test_update_project_rejects_null_name` — 422，且 `db.committed is False`、原名未被改写（证明在写库前拦截）
- `test_update_project_rejects_blank_name` — `"   "` → 422，未提交
- `test_update_project_allows_nullable_fields` — `description`/`custom_instructions` 为 `null` 仍可正常清空（守卫未误伤可空字段）
- `test_update_project_still_accepts_valid_name` — `name="renamed"` 正常提交并写入

### 验证结果

| # | 检查项 | 结果 |
| - | ------ | ---- |
| 1 | `python -m pytest tests/ -q`（容器内，镜像重建后在新容器复跑） | **55 passed, 0 failed** |
| 2 | `docker compose up -d --build mem0-dashboard` | `mem0-api Built` + `mem0-dashboard Built` / `Recreated` / `Started`，无 error |
| 3 | 容器状态 | `mem0-api Up`、`mem0-dashboard Up`；dashboard 日志 `✓ Ready in 279ms`，无运行时报错 |
| 4 | lint（3 个改动文件） | 0 diagnostics |

### 偏离（1 处，Rule 2 自动修复）

`create_project` 未校验空白 `name`，与新的 UPDATE 守卫语义不一致（同一 NOT NULL 列）→ 补 422 校验；前端同步加 `name is required` 拦截。

---

## 复审第三轮修复（1 BLOCKER + 1 空断言测试）

| 提交 | 哈希 | 说明 |
| ---- | ---- | ---- |
| `fix(server): clear git_remotes on explicit null and strengthen update tests` | `31fa248e` | `server/routers/projects.py`、`server/tests/test_project_defaults.py` |

### CR-02（BLOCKER）：`{"git_remotes": null}` → IntegrityError 500

`models.py` 中 `git_remotes` 为 `Mapped[list]`（NOT NULL，非 Optional），而 `ProjectUpdate.git_remotes` 允许 `null` 且不在 `NON_NULLABLE_FIELDS = {"name"}` 守卫内 → `setattr(project, "git_remotes", None)` → `db.commit()` 抛 IntegrityError → 500。

修复（`update_project` 循环内，`setattr` 之前）：

```python
if field == "git_remotes" and value is None:
    # Explicit ``null`` means "clear the remotes", not "write NULL" —
    # the column is NOT NULL, so store an empty list instead.
    value = []
```

语义：`git_remotes: null` = 清空远程地址（写 `[]`）；`name` 仍走 422 拒绝路径，循环后未新增无关逻辑。

### 空断言测试：`test_update_project_allows_nullable_fields`

原用例的假对象 `description` / `custom_instructions` 初始值就是 `None`，`assert ... is None` 恒真 —— 即使撤销 `exclude_unset` 修复也会通过。

修复：`_fake_project()` 预置非空值（`description="original description"`、`custom_instructions="original instructions"`、`memory_expiration_date=date(2027,1,1)`、`git_remotes=["https://x/y.git"]`），用例内先断言初始值非空，再传显式 `null`，断言清空（`git_remotes` → `[]`）。

新增用例：

- `test_update_project_null_git_remotes_clears_to_empty_list` — 覆盖 CR-02（`git_remotes: null` → `[]`）
- `test_update_project_leaves_omitted_fields_untouched` — 未出现在载荷中的字段保持原值，锁住 `exclude_unset` 语义

### 验证结果

| # | 检查项 | 结果 |
| - | ------ | ---- |
| 1 | `python -m pytest tests/ -q`（容器内） | **57 passed, 0 failed** |
| 2 | 变异验证（临时把修复条件置为 `and False`） | `test_update_project_null_git_remotes_clears_to_empty_list` 失败（`assert None == []`），恢复后复跑 57 passed —— 新用例真正锁住修复 |
| 3 | 改动文件 | 仅 `server/routers/projects.py`、`server/tests/test_project_defaults.py`（+46/-5） |

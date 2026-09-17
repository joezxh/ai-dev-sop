# Task 3 报告：DELETE /memories?project_id=X（按项目清空记忆）

## 状态

**COMPLETE** — commit `27f11b4e`（mem0 子模块，分支 `feat/llm-provider-i18n`）

## 测试摘要

- 红灯：`tests/test_delete_project_memories.py` 4 failed（全部 `AttributeError: module 'main' has no attribute 'delete_memories_by_project'`，与预期一致）
- 绿灯：4 passed
- 回归：`pytest tests/ -q` → **37 passed**（无失败）

## 交付物

| 文件 | 变更 |
|------|------|
| `mem0/server/main.py` | 新增 `delete_memories_by_project`（get_all_memories 之后）；既有 `DELETE /memories` 路由合并 `project_id` 分发 |
| `mem0/server/tests/test_delete_project_memories.py` | 新建（4 个用例，逐字来自 brief） |

核心行为：admin-only（403）→ 缺 project_id（400）→ 项目不存在（404）→ 按 `metadata.project_id` 后过滤取 id 逐条删除；单条失败 `logging.exception` 并继续，返回 `{"deleted": N}`。

## 偏差（相对 brief 代码，Rule 1 自动修复）

brief 的 Step 3 代码带 `@app.delete("/memories", ...)` 装饰器并放在 `get_all_memories` 之后。仓库中 **line 839 已存在同路径同方法的 `delete_all_memories`**（user_id/run_id/agent_id 删除，OSS 兼容接口）。FastAPI 按注册顺序匹配，先注册者胜出——照搬会导致旧端点被遮蔽、不可达（对既有客户端是破坏性回归）。修复：

1. `delete_memories_by_project` 保留 brief 函数体逐字，**去掉装饰器**（测试直接调用函数，不受影响）。
2. 路由合并进既有 `delete_all_memories`：新增 `project_id: Optional[str] = Query(None)`，有 project_id 时委托给新函数；否则走原有「按标识符删除」分支（403 → 400 顺序保持原语义）。
3. 去掉该路由的 `response_model=MessageResponse`（项目分支返回 `{"deleted": N}`，保留 response_model 会触发 Pydantic 响应校验 500）。`MessageResponse` 导入仍被 `delete_memory` 使用，无残留。
4. 依赖注入从 `Depends(require_admin)` 改为 `Depends(verify_auth)`（合并路由必须二选一），随后在函数体内手动校验 admin。与 `require_admin` 的唯一语义差异：`AUTH_DISABLED`/`ADMIN_API_KEY`（`_auth=None`）直接视为 admin，不再解析 users 表默认用户并校验其角色——与 `get_all_memories` 等新端点的既有约定一致。task-15 的 smoke（`DELETE /memories` 无参数 → 400）在合并设计下依然成立。

## 测试计数说明

回归预期「33 passed」与实际 37 passed 不符：容器与仓库测试文件一致，**既有基线本就是 33 个用例**（configure_dims 6 + configure 5 + multitenant 16 + project_defaults 3 + stats 3），加本任务 4 个后为 37。规划时基线被误算为 29，非回归问题。

## Self-Check

- [x] 测试文件与 brief 逐字一致
- [x] 函数体与 brief 逐字一致（仅去装饰器，偏差已记录）
- [x] 红灯（AttributeError）→ 绿灯（4 passed）→ 回归（37 passed）
- [x] 单条删除失败 catch + `logging.exception` + 继续删除其余
- [x] 依赖函数 `_load_project` / `_list_all_memories` / `get_memory_instance` / `ALL_MEMORIES_LIMIT` 语义未改动
- [x] 子模块内提交 `27f11b4e`，工作区干净，无意外删除文件

## Concerns

1. **无。** 唯一值得后续复核的点：合并路由的 admin 判定语义（见偏差第 4 条），若下游测试依赖 `require_admin` 的默认用户解析行为（当前测试未覆盖此边角），可在后续任务补一个用例。

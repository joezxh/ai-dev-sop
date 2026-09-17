# Task 3 审查：DELETE /memories?project_id（项目清空）

**Commit:** 27f11b4e（mem0 子模块）
**审查依据:** task-3-brief.md / task-3-report.md / task-3-diff.txt + main.py、auth.py 实地核实

```
Spec: ✅
- Missing: 无。403（非 admin）→ 400（缺 project_id）→ 404（项目不存在）→
  metadata.project_id 过滤逐条删除、单条失败 logging.exception 继续、
  返回 {"deleted": N} —— 全部落地（main.py:743-772, 872-898）。
  测试文件与 brief 逐字一致（4 用例）。
- Extra: ①未按 brief 新增带装饰器的端点，而是合并进既有 delete_all_memories
  做分发 —— 偏差合理且必要：brief 原样照搬会在 line 839 既有 OSS 路由之后注册
  同路径同方法端点，先注册者胜出导致新端点不可达（或反之遮蔽 OSS 兼容接口，
  构成对既有客户端的破坏性回归）。②去掉 response_model=MessageResponse ——
  必要：项目分支返回 {"deleted": N} 与 MessageResponse 不兼容，保留会触发
  Pydantic 响应校验 500。③鉴权由 Depends(require_admin) 改为 verify_auth +
  函数内 admin 校验 —— 合并路由只能选一个依赖，偏差已评估（见下）。

Quality: Approved
- [Minor] 鉴权语义漂移（auth.py:198-220 vs main.py:877,892）：对持有有效
  JWT/API key 的用户，函数内 `role != "admin"` 判定与 require_admin 等价；
  唯一差异是 _auth=None（ADMIN_API_KEY / AUTH_DISABLED）路径 —— 旧
  require_admin 会解析 users 表首个用户、非 admin 则 403，新逻辑无条件视为
  admin。评估：ADMIN_API_KEY 是服务端环境级凭据，授予 admin 正确；
  AUTH_DISABLED 是部署方显式关闭鉴权，本就全员可达。无实际越权引入，
  且与 add_memory（main.py:622 `_auth is None or role=="admin"`）既有约定
  一致。建议在文档/OpenAPI 描述中注明该差异（报告 Concerns 已自我披露）。
- [Minor] main.py:890 `if project_id:` 用真值判断：`?project_id=`（空串）或
  纯空白会落入 legacy 分支，得到 400 "At least one identifier is required"
  而非语义更准确的 "project_id is required"。仍是 400，符合 spec 底线，
  仅错误信息有误导。可改为 `if project_id is not None:` + 空串显式 400。
- [Minor] main.py:762 `data = _list_all_memories(limit=ALL_MEMORIES_LIMIT)`：
  匹配记忆数超过 ALL_MEMORIES_LIMIT 时尾部记忆会被静默漏删，{"deleted": N}
  低报。继承自 brief 代码原样，非实现者偏差；建议后续任务在响应中携带
  total/truncated 提示或分页清空。
- [Minor] main.py:765 `r["id"]` 直接下标：_serialize_memory 用
  `getattr(row, "id", None)`（main.py:676），理论上可产出 id=None 的行，
  触发 delete(None)。继承自 brief；实际向量库行均有 id，风险极低。

偏差逐项判定（重点审查项）:
a) 路由匹配歧义：不存在。DELETE /memories 与 DELETE /memories/{memory_id}
   （main.py:859）段数不同，查询参数不参与 FastAPI 路径匹配；合并后同路径
   同方法仅剩一个路由。
b) 越权评估：无新增越权面（见上方 Minor 第 1 条详析）。
c) 无 project_id 时的 OSS 行为：保持。403→400 顺序、delete_all 调用、
   MessageResponse 返回体、upstream_error 包装均未变（main.py:892-898）；
   response_model=None 仅影响 OpenAPI schema 文档化，不影响响应体。

⚠️ Cannot verify from diff:
- 回归 "37 passed" 依指示采信报告，未重跑。
- 端到端 HTTP 层行为（依赖注入解析、真实向量库 metadata 写入 project_id）
  仅有单元级证据；task-15 smoke（DELETE /memories 无参 → 400）未在本次
  差异包中看到对应测试文件，依赖报告声明。
```

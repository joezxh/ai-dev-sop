> [!WARNING]
> **已废弃（2026-09-18）**：本评测数据集配套的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，统一替换为 mem0（见仓库 docs/quick-ref/mem0-manual.md）。本文仅作历史归档。

# DS-CallPath · 代码结构调用链检索数据集

> **状态**：📘 数据集规范 v0.1｜2026-07-06
> **主战场**：B · codebase-mem-mcp（SQLite 知识图谱：Function / Class / Route / HTTP_CALLS）
> **样本数**：100（每条 = 1 个结构化 query + 多个 ground truth qualified_name）
> **预期主模式**：B-only / A+B
> **数据文件**：`datasets/DS-CallPath.jsonl`

---

## 1. 任务定义

**目标**：给定一个代码入口（函数 / 类 / 路由），找出它的完整调用链上下游节点。

**示例 query**：
- "谁调用了 UserService.login？"
- "VectorStoreService.save 的下游是什么？"
- "POST /api/v1/auth/login 这条路由经过哪些 handler？"
- "OrderController.createOrder 间接调用了哪些 service？"

**Ground truth**：通过 `trace_path` BFS（depth 1-5）得到的、应当出现在结果中的 qualified_name 列表（按深度分组）。

---

## 2. 样本格式（JSONL）

每行一条 JSON：

```json
{
  "id": "DS-CP-0001",
  "query_type": "downstream",
  "entry_point": {
    "qualified_name": "adsop.module.system.service.UserService.login",
    "label": "Function",
    "file": "adsop-module-system/src/main/java/.../UserService.java",
    "line": 42
  },
  "max_depth": 3,
  "ground_truth": [
    {
      "qualified_name": "adsop.module.system.controller.AuthController.handleLogin",
      "label": "Function",
      "depth": 1,
      "edge_type": "CALLS",
      "relevance": 3,
      "rationale": "上游直接调用"
    },
    {
      "qualified_name": "adsop.module.system.service.TokenService.generate",
      "label": "Function",
      "depth": 2,
      "edge_type": "CALLS",
      "relevance": 3,
      "rationale": "login 内部调用"
    }
  ],
  "metadata": {
    "language": "java",
    "category": "auth",
    "difficulty": "easy",
    "created_at": "2026-07-02"
  }
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | ✅ | DS-CP-NNNN |
| `query_type` | enum | ✅ | upstream（调用了谁） / downstream（被谁调用） / bidirectional |
| `entry_point.qualified_name` | string | ✅ | 入口符号的限定名 |
| `entry_point.label` | enum | ✅ | Function / Class / Route |
| `entry_point.file` | string | ✅ | 源文件相对路径 |
| `entry_point.line` | int | ✅ | 起始行号 |
| `max_depth` | int | ✅ | BFS 最大深度（1-5） |
| `ground_truth[].qualified_name` | string | ✅ | 期望出现在结果中的符号 |
| `ground_truth[].depth` | int | ✅ | BFS 距离入口的深度 |
| `ground_truth[].edge_type` | enum | ✅ | CALLS / HTTP_CALLS / EMITS / IMPORTS |
| `ground_truth[].relevance` | int (1-3) | ✅ | 1=边缘，2=相关，3=核心 |
| `metadata.language` | enum | ✅ | java / ts / py / go |
| `metadata.category` | enum | ✅ | auth / cache / db / api / search / 异步事件 |
| `metadata.difficulty` | enum | ✅ | easy / medium / hard |

---

## 3. 类别与难度分布

| 类别 | 样本数 | 说明 |
|------|--------|------|
| auth | 20 | 登录 / 鉴权 / 权限 |
| cache | 15 | Redis / Caffeine / 多级缓存 |
| db | 20 | ORM / 事务 / 连接池 |
| api | 20 | REST / HTTP 路由 |
| search | 10 | 全文 / 向量检索 |
| 异步事件 | 15 | Kafka / EventBus / 队列 |

| 难度 | 样本数 | 含义 |
|------|--------|------|
| easy | 30 | 直接调用（depth=1） |
| medium | 50 | depth=2-3，需多跳 |
| hard | 20 | depth=4-5 或跨服务 HTTP_CALLS |

---

## 4. Ground Truth 标注流程

### 4.1 自动预标注

1. 用 codebase-mem-mcp `trace_path` 跑 max_depth 范围
2. 把所有 depth ≤ max_depth 的节点作为候选集
3. 按 BFS 距离填入 `depth` 与 `edge_type`

### 4.2 人工核验

- 由 2 名熟悉工程的开发者独立核验：
  - 删除已不存在的节点（dead code）
  - 删除运行时不会触发的条件分支节点（if 内部不命中）
  - 标注 `relevance` 1/2/3

### 4.3 双人 IoU ≥ 0.8 后入库

---

## 5. 评测运行

### 5.1 B-only 模式

```python
for sample in load_jsonl("DS-CallPath.jsonl"):
    response = mcp_call(
        server="codebase-memory-mcp",
        tool="trace_path",
        args={
            "start": sample["entry_point"]["qualified_name"],
            "direction": sample["query_type"],
            "max_depth": sample["max_depth"]
        }
    )
    metrics.update(compute_ranking_metrics(response, sample["ground_truth"]))
```

### 5.2 A+B 模式（检测双轨退化）

```python
# B 为主，A 仅做语义补充（双轨应不显著退化）
response_b = mcp_call(server="codebase-memory-mcp", tool="trace_path", args={...})
response_a = mcp_call(server="mempalace", tool="mempalace_recall",
                     args={"query": f"调用链：{sample['entry_point']['qualified_name']}"})
merged = priority_merge(response_b, response_a, weight_b=0.8)
```

---

## 6. 通过标准

| 指标 | 阈值 | 含义 |
|------|------|------|
| B-only R@5 | ≥ 0.85 | 结构检索足够强 |
| B-only R@10 | ≥ 0.95 | 长尾覆盖 |
| B-only MRR | ≥ 0.75 | 排序质量 |
| A+B R@5 退化 | ≤ 5% | 双轨未引入干扰 |
| B 亚毫秒延迟 P50 | ≤ 2ms | 性能基线 |

---

## 7. 样例样本（5 条）

```json
{"id": "DS-CP-0001", "query_type": "upstream", "entry_point": {"qualified_name": "adsop.module.system.service.UserService.login", "label": "Function", "file": "adsop-module-system/src/main/java/.../UserService.java", "line": 42}, "max_depth": 3, "ground_truth": [{"qualified_name": "adsop.module.system.controller.AuthController.handleLogin", "label": "Function", "depth": 1, "edge_type": "CALLS", "relevance": 3, "rationale": "直接调用 login"}, {"qualified_name": "adsop.module.system.service.TokenService.generate", "label": "Function", "depth": 2, "edge_type": "CALLS", "relevance": 3, "rationale": "login 内部生成 token"}], "metadata": {"language": "java", "category": "auth", "difficulty": "easy"}}
{"id": "DS-CP-0002", "query_type": "downstream", "entry_point": {"qualified_name": "adsop.module.kms.service.VectorStoreService.save", "label": "Function", "file": "adsop-module-kms/src/main/java/.../VectorStoreService.java", "line": 88}, "max_depth": 2, "ground_truth": [{"qualified_name": "adsop.module.kms.dao.VectorStoreMapper.insert", "label": "Function", "depth": 1, "edge_type": "CALLS", "relevance": 3, "rationale": "MyBatis 持久化"}], "metadata": {"language": "java", "category": "db", "difficulty": "easy"}}
{"id": "DS-CP-0003", "query_type": "downstream", "entry_point": {"qualified_name": "POST /api/v1/auth/login", "label": "Route", "file": "adsop-web/src/api/auth.ts", "line": 12}, "max_depth": 3, "ground_truth": [{"qualified_name": "adsop.module.system.controller.AuthController.handleLogin", "label": "Function", "depth": 1, "edge_type": "HTTP_CALLS", "relevance": 3, "rationale": "前端到后端"}, {"qualified_name": "adsop.module.system.service.UserService.login", "label": "Function", "depth": 2, "edge_type": "CALLS", "relevance": 3, "rationale": "后端 service"}], "metadata": {"language": "ts", "category": "auth", "difficulty": "medium"}}
{"id": "DS-CP-0004", "query_type": "upstream", "entry_point": {"qualified_name": "adsop.module.order.event.OrderPaidEvent", "label": "Class", "file": "adsop-module-order/src/main/java/.../OrderPaidEvent.java", "line": 5}, "max_depth": 2, "ground_truth": [{"qualified_name": "adsop.module.order.service.OrderService.markPaid", "label": "Function", "depth": 1, "edge_type": "EMITS", "relevance": 3, "rationale": "发出事件"}, {"qualified_name": "adsop.module.notify.listener.OrderPaidListener", "label": "Function", "depth": 1, "edge_type": "LISTENS_ON", "relevance": 3, "rationale": "下游监听"}], "metadata": {"language": "java", "category": "async", "difficulty": "medium"}}
{"id": "DS-CP-0005", "query_type": "upstream", "entry_point": {"qualified_name": "adsop.module.case.service.CaseSearchService.hybridSearch", "label": "Function", "file": "adsop-module-case/src/main/java/.../CaseSearchService.java", "line": 156}, "max_depth": 5, "ground_truth": [{"qualified_name": "adsop.module.case.dao.CaseMapper.fullTextSearch", "label": "Function", "depth": 1, "edge_type": "CALLS", "relevance": 3, "rationale": "PG 全文检索"}, {"qualified_name": "adsop.module.kms.service.VectorStoreService.query", "label": "Function", "depth": 1, "edge_type": "CALLS", "relevance": 3, "rationale": "向量检索"}, {"qualified_name": "adsop.module.case.service.RerankService.rerank", "label": "Function", "depth": 1, "edge_type": "CALLS", "relevance": 2, "rationale": "重排序"}], "metadata": {"language": "java", "category": "search", "difficulty": "hard"}}
```

---

## 8. 变更记录

| 版本 | 日期 | 作者 | 变更 |
|------|------|------|------|
| v0.1 | 2026-07-06 | AI 架构助手 | 初稿：100 条样本规范 + 自动预标注 + 阈值 |
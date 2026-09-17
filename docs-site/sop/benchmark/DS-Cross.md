> [!WARNING]
> **已废弃（2026-09-18）**：本评测数据集配套的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，统一替换为 mem0（见仓库 docs/quick-ref/mem0-ai-tools-config-guide.md）。本文仅作历史归档。

# DS-Cross · 双轨复合检索数据集（核心 KPI）

> **状态**：📘 数据集规范 v0.1｜2026-07-06
> **主战场**：A + B 联合（A · MemPalace 自然语言 + B · codebase-mem-mcp 代码结构）
> **样本数**：100（每条 query 需要两侧证据互补）
> **预期主模式**：**A+B**（核心 KPI 评测数据集）
> **数据文件**：`datasets/DS-Cross.jsonl`
> **目标**：通过此数据集验证 §5.10.3 P1-1/1-2 的 Symbol ↔ Drawer 桥接与 §5.7.7 双轨联合工具的真实价值。

---

## 1. 任务定义

**目标**：给定一个跨记忆类型的查询，**必须**同时涉及"自然语言决策"与"代码结构符号"两类证据。期望 A+B 联合检索能比任何单侧召回更高。

**示例 query**：

| 类型 | query 样例 |
|------|----------|
| 决策 + 实现 | "为什么用 PGVector 而不是 Qdrant？代码里哪里体现了？" |
| Bug + 修复 | "2024-08 那个 OOM 是哪个 service 引发的？现在的修复点在哪儿？" |
| ADR + 影响 | "ADR-2026-0007 决定用 PGVector，它影响了哪些 Class？" |
| 客户需求 + 落地 | "客户要求支持中文语义检索，代码哪个模块落地了？" |
| 评审结论 + 代码 | "上次 Code Review 说 VectorStoreService 的 save 方法要加索引，做了吗？" |

**Ground truth**：同时包含 drawer_id 与 qualified_name 的混合列表。

---

## 2. 样本格式（JSONL）

每行一条 JSON：

```json
{
  "id": "DS-CROSS-0001",
  "query": "为什么用 PGVector 而不是 Qdrant？代码里哪里体现了？",
  "query_type": "decision_plus_implementation",
  "expected_strategy": "A_then_B_or_parallel",
  "ground_truth": [
    {
      "side": "A",
      "type": "drawer",
      "ref": "drawer_8f2c1a",
      "wing": "wing-org-decisions",
      "room": "database",
      "hall": "facts",
      "relevance": 3,
      "rationale": "记录 PGVector 选型决策"
    },
    {
      "side": "A",
      "type": "drawer",
      "ref": "drawer_c5d3ee",
      "wing": "wing-project-adsop",
      "room": "architecture",
      "hall": "facts",
      "relevance": 2,
      "rationale": "对比 Qdrant 与 PGVector 性能"
    },
    {
      "side": "B",
      "type": "symbol",
      "ref": "adsop.module.kms.config.VectorStoreConfig",
      "label": "Class",
      "file": "adsop-module-kms/src/main/java/.../VectorStoreConfig.java",
      "line": 18,
      "relevance": 3,
      "rationale": "PGVector Bean 配置"
    },
    {
      "side": "B",
      "type": "symbol",
      "ref": "adsop.module.kms.service.VectorStoreService",
      "label": "Class",
      "file": "adsop-module-kms/src/main/java/.../VectorStoreService.java",
      "line": 1,
      "relevance": 3,
      "rationale": "实现 PGVector 客户端"
    }
  ],
  "linked_decision": {
    "adr_id": "ADR-2026-0007",
    "drawer_id": "drawer_8f2c1a"
  },
  "metadata": {
    "category": "database",
    "difficulty": "medium",
    "language": "zh",
    "both_sides_required": true,
    "created_at": "2026-07-03"
  }
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | ✅ | DS-CROSS-NNNN |
| `query` | string | ✅ | 跨类型自然语言 query |
| `query_type` | enum | ✅ | decision_plus_implementation / bug_plus_fix / adr_plus_impact / customer_plus_code / review_plus_code |
| `expected_strategy` | enum | ✅ | A_then_B（先后）/ parallel（并行）/ B_then_A |
| `ground_truth[].side` | enum | ✅ | A（MemPalace） / B（codebase-mem） |
| `ground_truth[].type` | enum | ✅ | drawer / symbol |
| `ground_truth[].ref` | string | ✅ | drawer_id 或 qualified_name |
| `ground_truth[].relevance` | int (1-3) | ✅ | 1=边缘，2=相关，3=核心 |
| `linked_decision` | object | ❌ | 若 query 与 ADR 关联，填入 ADR↔drawer 双向引用 |
| `metadata.both_sides_required` | bool | ✅ | true：双侧都至少 1 条 ground truth；false：可只命中一侧 |

---

## 3. 类别分布

| query_type | 样本数 | 含义 |
|-----------|--------|------|
| decision_plus_implementation | 30 | 「为什么 X？代码哪里体现？」 |
| bug_plus_fix | 20 | 「XX 故障现在的修复点在哪儿？」 |
| adr_plus_impact | 20 | 「ADR-N 决定了什么？影响哪些 Class？」 |
| customer_plus_code | 15 | 「客户需求 X 由哪个模块落地？」 |
| review_plus_code | 15 | 「Review 说 X 要改，改了吗？」 |

| 难度 | 样本数 | 含义 |
|------|--------|------|
| easy | 30 | 双侧 ground truth 字面含关键词 |
| medium | 50 | 需语义 + 结构组合 |
| hard | 20 | 含历史变更 / supersede 关系 / 跨服务 |

---

## 4. Ground Truth 标注流程（最严格）

### 4.1 三步法

```
Step 1：单独运行 A 与 B 两个检索，记录各自 top-30
Step 2：人工阅读所有候选，标注每条的 relevance
Step 3：对 query 涉及的 ADR / 决策 / Bug 做事实核验（git blame / PR / 邮件）
```

### 4.2 双向独立

- A 侧标注员（熟悉自然语言检索）
- B 侧标注员（熟悉代码）
- 双方互不知道对方的标注结果
- 最终 ground truth 取并集 + relevance 协商

### 4.3 IoU & 仲裁

- A、B 两组 ground truth 的并集 IoU ≥ 0.8 方可入库
- 否则由第 3 人仲裁（同时熟悉 A + B）

### 4.4 桥接标注

对每条同时命中 A + B 的 ground truth，**必须**确认存在 `has_drawer_id` 或 `MENTIONS` 边（P1-1/1-2 已落地时），否则在 `linked_decision` 字段标注「待补桥接」。

---

## 5. 评测运行

### 5.1 A-only 模式

```python
# 仅用 mempalace_search + mempalace_recall
response = mcp_call(server="mempalace", tool="mempalace_search",
                   args={"query": sample["query"]})
metrics.update(compute_ranking_metrics(
    response,
    [gt for gt in sample["ground_truth"] if gt["side"] == "A"]
))
```

### 5.2 B-only 模式

```python
# 仅用 codebase-mem 的语义查询
response = mcp_call(server="codebase-memory-mcp", tool="semantic_query",
                   args={"query": sample["query"], "limit": 10})
metrics.update(compute_ranking_metrics(
    response,
    [gt for gt in sample["ground_truth"] if gt["side"] == "B"]
))
```

### 5.3 A+B 模式（核心）

```python
# 双轨联合，按 expected_strategy 编排
if sample["expected_strategy"] == "A_then_B":
    response_a = mcp_call(server="mempalace", tool="mempalace_search", args={...})
    # 用 A 拿到的 drawer 提取符号名，再喂给 B
    symbols = extract_symbols(response_a)
    response_b = mcp_call(server="codebase-memory-mcp", tool="search_graph",
                         args={"name_pattern": symbols})
    merged = merge_a_then_b(response_a, response_b)
elif sample["expected_strategy"] == "parallel":
    # 并行调用，§5.7.7 control room 负责合并
    response_a, response_b = parallel_call([...])
    merged = round_robin_merge(response_a, response_b, weight_a=0.5, weight_b=0.5)
```

---

## 6. 通过标准（**DS-Cross 是核心 KPI**）

| 指标 | 阈值 | 含义 |
|------|------|------|
| A+B R@5 | ≥ 0.70 | 联合检索能给出至少 1 个相关结果在前 5 |
| A+B R@10 | ≥ 0.85 | 长尾覆盖 |
| A+B MRR | ≥ 0.55 | 排序质量 |
| **Complementarity-Gain** | **≥ 0.30** | **R@5(A+B) - max(R@5(A), R@5(B)) ≥ 30 个百分点** |
| 双侧命中率 | ≥ 50% | top-10 内同时出现 A 侧 + B 侧结果 |

### 6.1 互补增益计算示例

```
假设 R@5(A-only) = 0.40
      R@5(B-only) = 0.35
      R@5(A+B)    = 0.78

Complementarity-Gain = 0.78 - max(0.40, 0.35) = 0.38 ≥ 0.30 ✅ 双轨联合显著优于单侧
```

### 6.2 不达预期的处理

如果 Gain < 0.30：

| 可能原因 | 验证手段 |
|---------|---------|
| P1-1/1-2 桥接未实现 | 检查 `memplace_symbol_to_drawers` 是否返回空 |
| A 侧 drawer 缺失相关决策 | 检查 wing 是否被正确索引 |
| B 侧语义查询质量差 | 改用 `search_graph` + 关键词过滤 |
| query 实际只需单侧 | 重新评估 ground truth 的 both_sides_required |

---

## 7. 样例样本（5 条）

```json
{"id": "DS-CROSS-0001", "query": "为什么用 PGVector 而不是 Qdrant？代码里哪里体现了？", "query_type": "decision_plus_implementation", "expected_strategy": "A_then_B", "ground_truth": [{"side": "A", "type": "drawer", "ref": "drawer_8f2c1a", "wing": "wing-org-decisions", "relevance": 3, "rationale": "PGVector 选型决策"}, {"side": "B", "type": "symbol", "ref": "adsop.module.kms.config.VectorStoreConfig", "label": "Class", "file": ".../VectorStoreConfig.java", "line": 18, "relevance": 3, "rationale": "PGVector Bean 配置"}], "linked_decision": {"adr_id": "ADR-2026-0007", "drawer_id": "drawer_8f2c1a"}, "metadata": {"category": "database", "difficulty": "medium", "language": "zh", "both_sides_required": true}}
{"id": "DS-CROSS-0002", "query": "2024-08 那个 OOM 是哪个 service 引发的？现在的修复点在哪儿？", "query_type": "bug_plus_fix", "expected_strategy": "A_then_B", "ground_truth": [{"side": "A", "type": "drawer", "ref": "drawer_b92e88", "wing": "wing-team-backend", "room": "incidents", "hall": "discoveries", "relevance": 3, "rationale": "2024-08 OOM 复盘记录"}, {"side": "B", "type": "symbol", "ref": "adsop.module.case.service.CaseSearchService.fullTextSearch", "label": "Function", "file": ".../CaseSearchService.java", "line": 178, "relevance": 3, "rationale": "OOM 根因函数，已加 LIMIT"}], "metadata": {"category": "performance", "difficulty": "hard", "language": "zh", "both_sides_required": true}}
{"id": "DS-CROSS-0003", "query": "ADR-2026-0007 决定用 PGVector，它影响了哪些 Class？", "query_type": "adr_plus_impact", "expected_strategy": "B_then_A", "ground_truth": [{"side": "B", "type": "symbol", "ref": "adsop.module.kms.service.VectorStoreService", "label": "Class", "relevance": 3, "rationale": "PGVector 客户端实现"}, {"side": "B", "type": "symbol", "ref": "adsop.module.kms.dao.VectorStoreMapper", "label": "Class", "relevance": 2, "rationale": "通过 VectorStoreService 间接依赖"}, {"side": "A", "type": "drawer", "ref": "drawer_8f2c1a", "wing": "wing-org-decisions", "relevance": 3, "rationale": "ADR 原文"}], "metadata": {"category": "database", "difficulty": "medium", "language": "zh", "both_sides_required": true}}
{"id": "DS-CROSS-0004", "query": "客户要求支持中文语义检索，代码哪个模块落地了？", "query_type": "customer_plus_code", "expected_strategy": "A_then_B", "ground_truth": [{"side": "A", "type": "drawer", "ref": "drawer_4e2c81", "wing": "wing-person-fde", "room": "customer-requests", "hall": "events", "relevance": 3, "rationale": "客户原话：'我们需要中文语义检索'"}, {"side": "B", "type": "symbol", "ref": "adsop.module.kms.service.EmbeddingService.encode", "label": "Function", "relevance": 3, "rationale": "中文 embedding 模型加载"}], "metadata": {"category": "feature", "difficulty": "medium", "language": "zh", "both_sides_required": true}}
{"id": "DS-CROSS-0005", "query": "上次 Code Review 说 VectorStoreService 的 save 方法要加索引，做了吗？", "query_type": "review_plus_code", "expected_strategy": "A_then_B", "ground_truth": [{"side": "A", "type": "drawer", "ref": "drawer_a31b9d", "wing": "wing-team-backend", "room": "reviews", "hall": "advice", "relevance": 3, "rationale": "2026-06-15 Review 评论"}, {"side": "B", "type": "symbol", "ref": "adsop.module.kms.dao.VectorStoreMapper.insert", "label": "Function", "relevance": 2, "rationale": "加了 idx_vector_created_at 索引"}], "metadata": {"category": "quality", "difficulty": "medium", "language": "zh", "both_sides_required": true}}
```

---

## 8. 变更记录

| 版本 | 日期 | 作者 | 变更 |
|------|------|------|------|
| v0.1 | 2026-07-06 | AI 架构助手 | 初稿：100 条样本规范 + 双向独立标注 + Gain ≥ 30% 阈值 |
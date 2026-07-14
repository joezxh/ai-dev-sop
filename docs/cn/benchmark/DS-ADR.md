# DS-ADR · 架构决策记录（ADR）版本治理数据集

> **状态**：📘 数据集规范 v0.1｜2026-07-06
> **主战场**：B · codebase-mem-mcp `manage_adr`（含 supersede 关系）
> **样本数**：50（每条 = 1 个 ADR 查询 + 多个 ground truth ADR 节点）
> **预期主模式**：B-only
> **数据文件**：`datasets/DS-ADR.jsonl`

---

## 1. 任务定义

**目标**：测试 ADR 检索 + 版本治理能力，重点验证 **supersede（被替代）** 关系的正确返回。

**示例 query**：

| 类型 | query |
|------|------|
| 当前生效 | "向量库后端当前用的是什么？" |
| 历史 | "我们以前用过 ChromaDB 吗？什么时候停用的？" |
| 演进链 | "向量库后端经历了哪些变更？" |
| 影响范围 | "ADR-2026-0007 被哪些 ADR 替代了？" |
| 同主题 | "所有关于鉴权框架的 ADR" |

---

## 2. 样本格式（JSONL）

```json
{
  "id": "DS-ADR-0001",
  "query": "向量库后端当前用的是什么？",
  "query_type": "current_state",
  "ground_truth": [
    {
      "adr_id": "ADR-2026-0007",
      "title": "向量库后端选型：PGVector",
      "status": "accepted",
      "supersedes": ["ADR-2024-0003", "ADR-2025-0001"],
      "superseded_by": null,
      "relevance": 3,
      "rationale": "当前生效的 ADR"
    },
    {
      "adr_id": "ADR-2024-0003",
      "title": "向量库后端选型：ChromaDB 试点",
      "status": "superseded",
      "supersedes": [],
      "superseded_by": "ADR-2025-0001",
      "relevance": 1,
      "rationale": "已被替代，不应推荐"
    }
  ],
  "metadata": {
    "topic": "vector-db",
    "depth_chain": 3,
    "difficulty": "medium",
    "created_at": "2026-07-04"
  }
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | ✅ | DS-ADR-NNNN |
| `query` | string | ✅ | 自然语言 query |
| `query_type` | enum | ✅ | current_state / historical / evolution_chain / impact / topic_search |
| `ground_truth[].adr_id` | string | ✅ | ADR 唯一 ID（ADR-YYYY-NNNN） |
| `ground_truth[].status` | enum | ✅ | proposed / accepted / superseded / deprecated |
| `ground_truth[].supersedes` | array | ✅ | 直接被本 ADR 替代的前置 ADR 列表 |
| `ground_truth[].superseded_by` | string / null | ✅ | 替代本 ADR 的后继 ADR（如有） |
| `ground_truth[].relevance` | int (1-3) | ✅ | 1=应排除（已 superseded），2=历史相关，3=当前生效 |
| `metadata.depth_chain` | int | ✅ | 演进链深度（1 = 单 ADR，>1 = 含多次替代） |
| `metadata.difficulty` | enum | ✅ | easy / medium / hard |

---

## 3. 类别分布

| query_type | 样本数 | 含义 |
|-----------|--------|------|
| current_state | 15 | 当前生效 ADR |
| historical | 10 | 已被替代的历史 ADR |
| evolution_chain | 10 | 完整演进链（≥ 3 个 ADR） |
| impact | 8 | 反向：谁替代了 ADR-X |
| topic_search | 7 | 同主题多个 ADR |

| 难度 | 样本数 | 含义 |
|------|--------|------|
| easy | 15 | depth_chain=1，状态单一 |
| medium | 25 | depth_chain=2，需跨越一次 supersede |
| hard | 10 | depth_chain≥3，需追溯整条演进链 |

---

## 4. Ground Truth 标注流程

### 4.1 数据源

- codebase-mem-mcp `manage_adr list --all`
- MemPalace `mempalace search --wing wing-org-decisions --hall facts` 交叉验证
- 必要时查 git log 与 PR 历史

### 4.2 标注规则

- `accepted` / `proposed` 视为可推荐（relevance ≥ 2）
- `superseded` 视为应排除（除非 query 显式问历史）
- `deprecated` 视为相关但不推荐
- 演进链必须完整列出所有节点（depth_chain = 节点数 - 1）

### 4.3 双向独立

- 由 1 名架构师 + 1 名资深后端独立标注
- IoU ≥ 0.8 后入库

---

## 5. 评测运行

```python
for sample in load_jsonl("DS-ADR.jsonl"):
    if sample["query_type"] == "current_state":
        # 用 status=accepted 过滤
        response = mcp_call(server="codebase-memory-mcp", tool="manage_adr",
                           args={"action": "list", "filter": "status=accepted",
                                 "topic": sample["metadata"]["topic"]})
    elif sample["query_type"] == "evolution_chain":
        # 拿 topic 下所有 ADR（含 superseded），按时间排序
        response = mcp_call(server="codebase-memory-mcp", tool="manage_adr",
                           args={"action": "list", "include_superseded": True,
                                 "topic": sample["metadata"]["topic"]})

    metrics.update(compute_adr_metrics(response, sample["ground_truth"]))
```

---

## 6. 通过标准

| 指标 | 阈值 | 含义 |
|------|------|------|
| R@5 | ≥ 0.70 | 当前 ADR 排在前 5 |
| MRR | ≥ 0.60 | 排序质量 |
| **supersede 准确率** | **≥ 0.95** | **已 superseded 的 ADR 不被推荐为「当前」** |
| 演进链完整率 | ≥ 0.80 | 跨多次 supersede 的 query 召回完整链 |

### 6.1 关键失败模式

| 失败 | 含义 |
|------|------|
| 把 superseded ADR 当成「当前」 | 治理严重错误，必须修 |
| 演进链截断 | codebase-mem 中间 ADR 缺失 |
| 跨主题串扰 | topic 过滤失败，召回无关 ADR |

---

## 7. 样例样本（5 条）

```json
{"id": "DS-ADR-0001", "query": "向量库后端当前用的是什么？", "query_type": "current_state", "ground_truth": [{"adr_id": "ADR-2026-0007", "title": "向量库后端选型：PGVector", "status": "accepted", "supersedes": ["ADR-2024-0003", "ADR-2025-0001"], "superseded_by": null, "relevance": 3, "rationale": "当前生效"}], "metadata": {"topic": "vector-db", "depth_chain": 3, "difficulty": "medium"}}
{"id": "DS-ADR-0002", "query": "向量库后端经历了哪些变更？", "query_type": "evolution_chain", "ground_truth": [{"adr_id": "ADR-2024-0003", "status": "superseded", "supersedes": [], "superseded_by": "ADR-2025-0001", "relevance": 2, "rationale": "ChromaDB 试点"}, {"adr_id": "ADR-2025-0001", "status": "superseded", "supersedes": ["ADR-2024-0003"], "superseded_by": "ADR-2026-0007", "relevance": 2, "rationale": "Qdrant 试点"}, {"adr_id": "ADR-2026-0007", "status": "accepted", "supersedes": ["ADR-2025-0001"], "superseded_by": null, "relevance": 3, "rationale": "PGVector 当前生效"}], "metadata": {"topic": "vector-db", "depth_chain": 3, "difficulty": "hard"}}
{"id": "DS-ADR-0003", "query": "我们以前用过 ChromaDB 吗？", "query_type": "historical", "ground_truth": [{"adr_id": "ADR-2024-0003", "status": "superseded", "supersedes": [], "superseded_by": "ADR-2025-0001", "relevance": 3, "rationale": "ChromaDB 试点 ADR"}], "metadata": {"topic": "vector-db", "depth_chain": 3, "difficulty": "easy"}}
{"id": "DS-ADR-0004", "query": "ADR-2026-0007 被谁替代了？", "query_type": "impact", "ground_truth": [], "metadata": {"topic": "vector-db", "depth_chain": 3, "difficulty": "easy", "note": "当前未被替代，应返回空"}}
{"id": "DS-ADR-0005", "query": "所有鉴权框架相关的 ADR", "query_type": "topic_search", "ground_truth": [{"adr_id": "ADR-2024-0012", "title": "鉴权框架：自研 JWT", "status": "accepted", "relevance": 3, "rationale": "当前鉴权"}, {"adr_id": "ADR-2025-0008", "title": "鉴权扩展：OAuth2.1 接入", "status": "proposed", "relevance": 2, "rationale": "扩展提案"}], "metadata": {"topic": "auth", "depth_chain": 1, "difficulty": "medium"}}
```

---

## 8. 变更记录

| 版本 | 日期 | 作者 | 变更 |
|------|------|------|------|
| v0.1 | 2026-07-06 | AI 架构助手 | 初稿：50 条样本规范 + supersede 准确率 ≥ 95% 阈值 |
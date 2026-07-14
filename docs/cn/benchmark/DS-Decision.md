# DS-Decision · 自然语言技术决策语义检索数据集

> **状态**：📘 数据集规范 v0.1｜2026-07-06
> **主战场**：A · MemPalace（Wing → Room → Hall → Drawer）
> **样本数**：100（每条 = 1 个 query + 多个 ground truth drawer_id）
> **预期主模式**：A-only / A+B
> **数据文件**：`datasets/DS-Decision.jsonl`

---

## 1. 任务定义

**目标**：给定一句自然语言 query（描述某个技术决策、约定或踩坑），从 MemPalace 中检索出最相关的 drawer 列表。

**示例 query**：
- "我们为什么选择 PostgreSQL 而不是 MySQL？"
- "调解平台的前端框架是什么？"
- "Redis 缓存过期的策略是怎么定的？"

**Ground truth**：人工标注的、与该 query 强相关的 drawer_id 列表（1-N 条）。

---

## 2. 样本格式（JSONL）

每行一条 JSON：

```json
{
  "id": "DS-DEC-0001",
  "query": "我们为什么选择 PostgreSQL 而不是 MySQL？",
  "expected_tools": ["mempalace_search", "mempalace_recall"],
  "expected_wing_scope": ["wing-org-decisions", "wing-project-adsop"],
  "ground_truth": [
    {
      "drawer_id": "drawer_8f2c1a",
      "wing": "wing-org-decisions",
      "room": "database",
      "hall": "facts",
      "relevance": 3,
      "annotator": "alice",
      "rationale": "明确指出 PG 在 JSONB / GIS 上的优势决定选 PG"
    },
    {
      "drawer_id": "drawer_3e9b04",
      "wing": "wing-project-adsop",
      "room": "architecture",
      "hall": "facts",
      "relevance": 2,
      "annotator": "alice",
      "rationale": "记录了 MySQL 性能测试被 PG 反超的过程"
    }
  ],
  "metadata": {
    "category": "database",
    "difficulty": "easy",
    "language": "zh",
    "created_at": "2026-07-01"
  }
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | ✅ | DS-DEC-NNNN |
| `query` | string | ✅ | 自然语言问题 |
| `expected_tools` | array | ✅ | 期望调用的 MemPalace 工具 |
| `expected_wing_scope` | array | ✅ | 标注员认为应被检索的 Wing 范围（用于调试） |
| `ground_truth` | array | ✅ | ground truth 列表 |
| `ground_truth[].drawer_id` | string | ✅ | MemPalace drawer ID |
| `ground_truth[].relevance` | int (1-3) | ✅ | 1=边缘相关，2=相关，3=高度相关 |
| `ground_truth[].annotator` | string | ✅ | 标注人 |
| `ground_truth[].rationale` | string | ✅ | 标注理由 |
| `metadata.category` | enum | ✅ | 类别：database / framework / security / performance / devops / ui |
| `metadata.difficulty` | enum | ✅ | easy / medium / hard |
| `metadata.language` | enum | ✅ | zh / en |

---

## 3. 类别与难度分布（建议）

| 类别 | 样本数 | 说明 |
|------|--------|------|
| database | 20 | 库选型 / 表设计 / 索引 / 事务 |
| framework | 20 | 前后端框架 / 中间件 |
| security | 15 | 鉴权 / 脱敏 / 审计 |
| performance | 15 | 性能压测 / 优化决策 |
| devops | 15 | CI/CD / 部署 / 监控 |
| ui | 15 | 设计规范 / 组件库 |

| 难度 | 样本数 | 含义 |
|------|--------|------|
| easy | 30 | drawer 字面包含 query 关键词 |
| medium | 50 | 需语义理解但无歧义 |
| hard | 20 | 需结合多 drawer / 含反义 / 含历史变更 |

---

## 4. Ground Truth 标注流程

### 4.1 单人标注

1. 标注人阅读 query，在 MemPalace 中人工 search
2. 阅读 top-30 结果，挑选相关性 ≥ 1 的所有 drawer
3. 对每条标注 `relevance`（1/2/3）和 `rationale`（一句话）
4. 记录 `expected_wing_scope`（实际被命中的 Wing 列表）

### 4.2 双人交叉

- 由 2 名标注人独立完成
- 计算 IoU = |A ∩ B| / |A ∪ B| ≥ 0.8 才视为一致
- IoU < 0.8 的样本由第 3 名资深成员仲裁

### 4.3 仲裁规则

- 若仅 1 个标注员标注的 drawer：默认 relevance=1，除非仲裁员确认 ≥ 2
- 仲裁员在 `rationale` 字段后追加 `[ARB: <id>]` 后缀

---

## 5. 评测运行

### 5.1 A-only 模式

```python
# scripts/run_bench.py
for sample in load_jsonl("DS-Decision.jsonl"):
    response = mcp_call(
        server="mempalace",
        tool="mempalace_search",
        args={"query": sample["query"], "wing_scope": sample["expected_wing_scope"]}
    )
    metrics.update(compute_ranking_metrics(response, sample["ground_truth"]))
```

### 5.2 A+B 模式

```python
# A-only 跑完后追加 B 侧补充检索（用于退化检测）
response_a = mcp_call(server="mempalace", tool="mempalace_search", args={...})
response_b = mcp_call(server="codebase-memory-mcp", tool="search_graph", args={"semantic_query": sample["query"]})
merged = interleave([response_a, response_b])  # 控制台默认 round-robin
```

---

## 6. 通过标准

| 指标 | 阈值 | 含义 |
|------|------|------|
| A-only R@5 | ≥ 0.80 | MemPalace 在自身主战场足够 |
| A-only R@10 | ≥ 0.92 | 长尾覆盖 |
| A-only MRR | ≥ 0.65 | 排序质量 |
| A+B R@5 退化 | ≤ 5% | 双轨未引入干扰 |
| verbatim 一致率 | 100% | 回显原文不修改 |

---

## 7. 样例样本（5 条）

```json
{"id": "DS-DEC-0001", "query": "我们为什么选择 PostgreSQL 而不是 MySQL？", "expected_tools": ["mempalace_search"], "expected_wing_scope": ["wing-org-decisions"], "ground_truth": [{"drawer_id": "drawer_8f2c1a", "wing": "wing-org-decisions", "room": "database", "hall": "facts", "relevance": 3, "annotator": "alice", "rationale": "记录 PG 选型对比"}], "metadata": {"category": "database", "difficulty": "easy", "language": "zh"}}
{"id": "DS-DEC-0002", "query": "Redis 缓存过期的策略是怎么定的？", "expected_tools": ["mempalace_search"], "expected_wing_scope": ["wing-project-adsop"], "ground_truth": [{"drawer_id": "drawer_4a7e2d", "wing": "wing-project-adsop", "room": "cache", "hall": "facts", "relevance": 3, "annotator": "alice", "rationale": "TTL=30min + 主动刷新策略"}], "metadata": {"category": "performance", "difficulty": "medium", "language": "zh"}}
{"id": "DS-DEC-0003", "query": "为什么不用 Spring Security 而是自己写鉴权？", "expected_tools": ["mempalace_search", "mempalace_recall"], "expected_wing_scope": ["wing-team-backend"], "ground_truth": [{"drawer_id": "drawer_2c91be", "wing": "wing-team-backend", "room": "auth", "hall": "discoveries", "relevance": 3, "annotator": "bob", "rationale": "记录 Spring Security 学习成本 + 灵活度不够"}], "metadata": {"category": "framework", "difficulty": "medium", "language": "zh"}}
{"id": "DS-DEC-0004", "query": "前端为什么从 Vue 2 升级到 Vue 3？", "expected_tools": ["mempalace_search"], "expected_wing_scope": ["wing-team-frontend", "wing-project-adsop"], "ground_truth": [{"drawer_id": "drawer_71d40c", "wing": "wing-team-frontend", "room": "frontend", "hall": "events", "relevance": 3, "annotator": "alice", "rationale": "2025-Q3 评审会决议"}], "metadata": {"category": "framework", "difficulty": "easy", "language": "zh"}}
{"id": "DS-DEC-0005", "query": "我们曾因为什么方案吃过亏？", "expected_tools": ["mempalace_recall"], "expected_wing_scope": ["wing-org-decisions", "wing-team-backend"], "ground_truth": [{"drawer_id": "drawer_b92e88", "wing": "wing-team-backend", "room": "incidents", "hall": "discoveries", "relevance": 2, "annotator": "bob", "rationale": "2024-08 OOM 事故复盘"}, {"drawer_id": "drawer_5c4f1a", "wing": "wing-org-decisions", "room": "ops", "hall": "events", "relevance": 2, "annotator": "bob", "rationale": "Kafka 集群选型踩坑"}], "metadata": {"category": "devops", "difficulty": "hard", "language": "zh"}}
```

---

## 8. 变更记录

| 版本 | 日期 | 作者 | 变更 |
|------|------|------|------|
| v0.1 | 2026-07-06 | AI 架构助手 | 初稿：100 条样本规范 + 标注流程 + 阈值 |
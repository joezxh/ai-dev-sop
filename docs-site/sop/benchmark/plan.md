> [!WARNING]
> **已废弃（2026-09-18）**：本评测数据集配套的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，统一替换为 mem0（见仓库 docs/quick-ref/mem0-ai-tools-config-guide.md）。本文仅作历史归档。

# 双轨记忆框架对比评测方案 v0.1

> **状态**：📘 评测方案 v0.1｜2026-07-06
> **上游依据**：`docs/cn/develop-sop.md` §5.10.5 对比评测方案 + §5.10 双轨记忆框架
> **目标**：量化 **MemPalace-only / codebase-mem-only / 双轨联合** 三种模式在 6 类典型查询任务上的召回率、延迟、Token 消耗、保真度，并据此回答「是否值得双写」、「双写是否产生冗余」等关键治理问题。
> **运行平台**：所有数据集与脚本均可在任意开发机运行（MemPalace 服务端 + codebase-mem-mcp 本地二进制）。无需 GPU。

---

## 0. 文档目的

§5.10.5 在 SOP 中以索引式摘要列出了评测方案，本文档是 **可执行的工程级方案**，包含：

1. 评测目标与假设
2. 6 个数据集的 schema、样本格式、ground truth 标注规范
3. 评测运行协议（A-only / B-only / A+B 三种模式）
4. 评测指标体系（R@K / P@K / MRR / 延迟 / Token / 保真度 / 覆盖率）
5. 互补增益公式与判定规则
6. 评测运行脚本骨架
7. 报告模板与回流路径

每一份数据集的详细规范见同目录下：

| 数据集 | 文档 |
|--------|------|
| DS-Decision | `DS-Decision.md` |
| DS-CallPath | `DS-CallPath.md` |
| DS-Cross | `DS-Cross.md` |
| DS-ADR | `DS-ADR.md` |
| DS-Customer | `DS-Customer.md` |
| DS-DeadCode | `DS-DeadCode.md` |

---

## 1. 评测目标与假设

### 1.1 三类目标

| # | 目标 | 度量 |
|---|------|------|
| G1 | 单工具在自身主战场是否足够强 | A 在 DS-Decision/DS-Customer 上的 R@5；B 在 DS-CallPath/DS-DeadCode 上的 R@5 |
| G2 | 双轨联合能否产生「互补增益」 | DS-Cross 上的 Complementarity-Gain ≥ 30%（阈值） |
| G3 | 双轨联合是否产生「冗余退化」 | DS-Decision / DS-CallPath 上 A+B 模式 vs 单工具的 R@5 差值（应 ≤ ±5%） |

### 1.2 假设

| 假设 ID | 内容 |
|---------|------|
| H1 | codebase-mem-mcp 索引已构建完毕，DS-CallPath / DS-DeadCode 的 ground truth 节点全部存在于图中 |
| H2 | MemPalace 服务已挖掘对应数据，DS-Decision / DS-Customer 的 ground truth drawer 全部存在 |
| H3 | 评测查询使用的 IDE 客户端已正确加载 §5.3 双 MCP 配置 |
| H4 | ground truth 标注由至少 2 名团队成员独立完成，IoU ≥ 0.8 才视为一致 |

### 1.3 反向假设（应被证伪的猜测）

| 猜测 | 证伪条件 |
|------|---------|
| 双轨联合比单工具慢 ≥ 50% | DS-Decision P95 延迟：双轨联合 < 1.5 × 单工具 |
| 双轨联合会引入大量重复 drawer/ADR | 同一 ground truth 在 A / B 两端重合率 ≤ 30% |

---

## 2. 评测数据集总览

| ID | 任务类型 | 主战场 | 样本数 | 双轨期望 |
|----|---------|--------|--------|---------|
| DS-Decision | 自然语言语义检索 | A · MemPalace | 100 | A 单独 ≥ 0.8 R@5；A+B 应不显著退化 |
| DS-CallPath | 代码结构调用链检索 | B · codebase-mem | 100 | B 单独 ≥ 0.8 R@5；A+B 应不显著退化 |
| DS-Cross | 「决策 + 代码」复合检索 | A + B 联合 | 100 | **A+B R@5 ≥ max(A, B) + 30%** |
| DS-ADR | ADR 版本治理检索 | B · codebase-mem | 50 | B 在 supersede 关系上 R@5 ≥ 0.7 |
| DS-Customer | 客户原话 verbatim 保真 | A · MemPalace | 30 | 文本 diff 一致率 ≥ 99% |
| DS-DeadCode | 死代码发现 | B · codebase-mem | 50 | B 检出率 ≥ 0.9 |

合计 **430 条样本**，按需可分批运行。

---

## 3. 评测运行协议

### 3.1 三种模式

| 模式 | 工具集 | 适用任务 |
|------|--------|---------|
| **A-only** | 仅 MemPalace MCP | DS-Decision / DS-Customer |
| **B-only** | 仅 codebase-mem-mcp | DS-CallPath / DS-ADR / DS-DeadCode |
| **A+B** | 两个 MCP 同时使用，控制台按 §5.7.7 联合工具调用 | DS-Cross |

### 3.2 一次评测的步骤

```
Step 1. 加载数据集 DS-* → 逐条 sample 读取 (query, ground_truth, expected_tools)
Step 2. 启动对应 IDE 会话，按 expected_tools 加载 MCP
Step 3. 对每条 query 调用一次工具，等待响应
Step 4. 把响应结果 (top-K 列表) 与 ground_truth 对比
Step 5. 记录指标（recall, precision, latency, tokens）
Step 6. 写入评测报告模板 benchmark/report-vX.Y.md
```

### 3.3 控制变量

- **模型**：固定为 Claude-Sonnet-4（同 baseline）
- **温度**：0.0（确定性）
- **K 值**：默认 top-10，记录 R@5 / R@10
- **重试**：失败任务重试 ≤ 2 次，否则标记 `FAIL`
- **隔离**：每条 query 之间清空对话历史

---

## 4. 评测指标体系

### 4.1 召回率 / 精度（核心）

| 指标 | 定义 | 计算 |
|------|------|------|
| **R@K** | ground truth 中前 K 个结果至少 1 个命中的比例 | `\|{q: gt ∩ top-K(q) ≠ ∅}\| / \|Q\|` |
| **P@K** | top-K 中 ground truth 占比 | `mean(\|gt ∩ top-K(q)\| / K)` |
| **MRR** | 第一个 ground truth 在排序中的倒数排名均值 | `mean(1/rank_first_gt(q))` |

### 4.2 性能（次要）

| 指标 | 定义 | 采集方式 |
|------|------|---------|
| **Latency P50** | 50% 分位响应时长（ms） | MCP 客户端日志 |
| **Latency P95** | 95% 分位响应时长（ms） | MCP 客户端日志 |
| **Latency P99** | 99% 分位响应时长（ms） | MCP 客户端日志 |
| **Token 消耗** | query + response 累计 token 数 | LLM 调用日志 |

### 4.3 治理（合规）

| 指标 | 定义 | 适用数据集 |
|------|------|-----------|
| **verbatim 一致率** | 客户原话检索返回 drawer 与原文 100% 字符匹配 | DS-Customer |
| **索引覆盖率** | 被索引代码行 / 工程总代码行 | DS-DeadCode 前置 |
| **桥接成功率** | Symbol ↔ Drawer 双向查询命中率 | DS-Cross |

### 4.4 互补增益（核心 KPI）

```
Complementarity-Gain (DS-Cross) = R@5(A+B) - max(R@5(A), R@5(B))
Complementarity-Gain (DS-Decision) = R@5(A+B) - R@5(A)   # 应接近 0（不显著退化）
Complementarity-Gain (DS-CallPath) = R@5(A+B) - R@5(B)   # 应接近 0（不显著退化）
```

**判定规则：**

| 数据集 | Gain 期望 | 含义 |
|--------|----------|------|
| DS-Cross | ≥ 0.30 | ✅ 双轨联合优于任一单工具 |
| DS-Decision | -0.05 ≤ x ≤ 0.05 | ✅ 双轨未引入干扰 |
| DS-CallPath | -0.05 ≤ x ≤ 0.05 | ✅ 双轨未引入干扰 |

---

## 5. 报告模板（benchmark/report-template.md）

每次评测生成一份：

```markdown
# 评测报告 vX.Y · YYYY-MM-DD

## 0. 元数据
- 评测人：
- 模型 / 客户端：
- 数据集版本：
- 总耗时：

## 1. 摘要表
| 数据集 | 模式 | R@5 | R@10 | MRR | P@5 | P50 延迟 | P95 延迟 | Token |
|--------|------|-----|------|-----|-----|---------|---------|-------|

## 2. 互补增益分析
- DS-Cross Complementarity-Gain = ...
- DS-Decision Complementarity-Gain = ...
- DS-CallPath Complementarity-Gain = ...

## 3. 反向假设验证
- [ ] H-反向-1: 双轨延迟 ... ✅ / ❌
- [ ] H-反向-2: 双轨冗余 ... ✅ / ❌

## 4. 异常与失效案例
- top-5 失败案例 + 根因假设

## 5. 建议
- 调整 §5.10 优化点优先级
- 调整 MemPlace 控制台路由策略
- 调整 P0-3 双写触发频率
```

---

## 6. 评测产物清单

每次完整运行产出：

```
docs/cn/benchmark/
├── plan.md                                   # 本文档
├── datasets/
│   ├── DS-Decision.jsonl                     # 100 条语义检索样本
│   ├── DS-Decision.md                        # 标注规范
│   ├── DS-CallPath.jsonl
│   ├── DS-CallPath.md
│   ├── DS-Cross.jsonl
│   ├── DS-Cross.md
│   ├── DS-ADR.jsonl
│   ├── DS-ADR.md
│   ├── DS-Customer.jsonl
│   ├── DS-Customer.md
│   ├── DS-DeadCode.jsonl
│   └── DS-DeadCode.md
├── scripts/
│   ├── run_bench.py                          # 评测运行脚本
│   ├── compute_metrics.py                    # 指标计算
│   └── render_report.py                      # 报告渲染
└── reports/
    └── report-v0.1-YYYY-MM-DD.md             # 一次评测报告
```

---

## 7. 回流机制

评测结论回写到：

| 结论类型 | 回流目标 |
|---------|---------|
| 双轨联合在 X 场景显著增益 | §5.10.3 P0-3 双写触发频率提升 |
| 双轨联合在 X 场景无增益 | §5.10.4 路线图延后该优化点 |
| A-only 场景在 B 侧无对应能力 | §3 场景管线补充 B 侧接入 |
| B-only 场景在 A 侧有冗余 | §5.10.3 优化功能点清单减项 |

---

## 8. 变更记录

| 版本 | 日期 | 作者 | 变更 |
|------|------|------|------|
| v0.1 | 2026-07-06 | AI 架构助手 | 初稿：6 数据集 + 3 模式 + 指标体系 |
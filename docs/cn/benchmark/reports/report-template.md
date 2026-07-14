# 评测报告 v0.1 · 2026-07-06

> **状态**：📊 模板｜双轨记忆框架对比评测
> **生成方式**：`python scripts/run_bench.py --all --report reports/report-vX.Y-YYYY-MM-DD.md`
> **对应方案**：`benchmark/plan.md`
> **对应数据集**：`benchmark/datasets/DS-*.jsonl`

---

## 0. 元数据

| 字段 | 值 |
|------|----|
| 评测人 |  |
| 评测日期 | 2026-07-06 |
| 模型 / 客户端 | Claude-Sonnet-4 / Cursor IDE 0.42 |
| 双 MCP 版本 | mempalace v0.8.3 / codebase-memory-mcp v0.5.1 |
| 数据集版本 | v0.1（6 数据集 / 共 430 样本） |
| 总耗时 |  |
| 评测模式 | A-only / B-only / A+B |

---

## 1. 摘要表

| 数据集 | 模式 | R@5 | R@10 | MRR | P@5 | P@10 | P50 延迟 | P95 延迟 | Token |
|--------|------|-----|------|-----|-----|------|---------|---------|-------|
| DS-Decision | A-only |  |  |  |  |  |  |  |  |
| DS-Decision | A+B |  |  |  |  |  |  |  |  |
| DS-CallPath | B-only |  |  |  |  |  |  |  |  |
| DS-CallPath | A+B |  |  |  |  |  |  |  |  |
| DS-Cross | A-only |  |  |  |  |  |  |  |  |
| DS-Cross | B-only |  |  |  |  |  |  |  |  |
| **DS-Cross** | **A+B** |  |  |  |  |  |  |  |  |
| DS-ADR | B-only |  |  |  |  |  |  |  |  |
| DS-Customer | A-only |  |  |  |  |  |  |  |  |
| DS-DeadCode | B-only |  |  |  |  |  |  |  |  |

---

## 2. 互补增益分析（核心 KPI）

### 2.1 DS-Cross 互补增益

```
R@5(A-only) = 
R@5(B-only) = 
R@5(A+B)    = 

Complementarity-Gain = R@5(A+B) - max(R@5(A), R@5(B)) = 
阈值要求：≥ 0.30
判定：□ ✅ 达标   □ ❌ 未达标
```

### 2.2 DS-Decision 互补增益（退化检测）

```
R@5(A-only) = 
R@5(A+B)    = 

Complementarity-Gain = R@5(A+B) - R@5(A-only) = 
阈值要求：-0.05 ≤ x ≤ 0.05
判定：□ ✅ 双轨未引入干扰   □ ❌ 引入退化
```

### 2.3 DS-CallPath 互补增益（退化检测）

```
R@5(B-only) = 
R@5(A+B)    = 

Complementarity-Gain = R@5(A+B) - R@5(B-only) = 
阈值要求：-0.05 ≤ x ≤ 0.05
判定：□ ✅ 双轨未引入干扰   □ ❌ 引入退化
```

---

## 3. 反向假设验证

| # | 假设 | 实测 | 判定 |
|---|------|------|------|
| H-反向-1 | 双轨联合 P95 延迟 < 1.5 × 单工具 |  | □ ✅ / □ ❌ |
| H-反向-2 | 同一 ground truth 在 A / B 两端重合率 ≤ 30% |  | □ ✅ / □ ❌ |
| H-正向-1 | A 在 DS-Decision R@5 ≥ 0.80 |  | □ ✅ / □ ❌ |
| H-正向-2 | B 在 DS-CallPath R@5 ≥ 0.85 |  | □ ✅ / □ ❌ |
| H-正向-3 | B 在 DS-DeadCode F1 ≥ 0.87 |  | □ ✅ / □ ❌ |

---

## 4. 治理指标

### 4.1 verbatim 保真度（DS-Customer）

| 指标 | 实测 | 阈值 |
|------|------|------|
| 字符匹配率 |  | ≥ 0.99 |
| 编辑距离 |  | = 0 |
| 总结干预率 |  | = 0% |
| 切片边界正确率 |  | 100% |
| 敏感信息脱敏率 |  | 100% |

### 4.2 ADR 版本治理（DS-ADR）

| 指标 | 实测 | 阈值 |
|------|------|------|
| 当前 ADR R@5 |  | ≥ 0.70 |
| supersede 准确率 |  | ≥ 0.95 |
| 演进链完整率 |  | ≥ 0.80 |

---

## 5. 异常与失效案例（Top 5）

| # | sample_id | query | 实际 top-1 | 期望 | 根因假设 |
|---|-----------|-------|----------|------|---------|
| 1 |  |  |  |  |  |
| 2 |  |  |  |  |  |
| 3 |  |  |  |  |  |
| 4 |  |  |  |  |  |
| 5 |  |  |  |  |  |

---

## 6. 结论与建议

### 6.1 双轨是否值得落地？

□ **强烈推荐**：DS-Cross Gain ≥ 0.30，且 DS-Decision / DS-CallPath 无显著退化
□ **有条件推荐**：DS-Cross Gain ≥ 0.20，建议先优化 P1-1 桥接后复测
□ **不推荐**：DS-Cross Gain < 0.20，回到单轨

### 6.2 优化建议（回流到 §5.10.3 / §5.10.4）

| 建议 | 回流目标 |
|------|---------|
|  |  |
|  |  |
|  |  |

### 6.3 下一轮评测重点

- [ ] 补充 ___ 场景样本
- [ ] 增加 ___ 数据源
- [ ] 优化 ___ 合并策略

---

## 7. 附录

### 7.1 运行命令

```bash
# 完整运行
python scripts/run_bench.py --all --report reports/report-v0.1-2026-07-06.md

# 单独跑某个数据集
python scripts/run_bench.py --dataset DS-Cross --mode A+B --limit 20

# 仅核心 KPI
python scripts/run_bench.py --dataset DS-Cross --mode A+B
python scripts/run_bench.py --dataset DS-Decision --mode A-only
python scripts/run_bench.py --dataset DS-CallPath --mode B-only
```

### 7.2 环境变量

```
MEMPALACE_URL=http://192.168.110.60:8080/mcp
CODEBASE_MEM_BIN=codebase-memory-mcp
LLM_MODEL=claude-sonnet-4
```

---

## 8. 变更记录

| 版本 | 日期 | 作者 | 变更 |
|------|------|------|------|
| v0.1 | 2026-07-06 | AI 架构助手 | 初稿模板 |
|  |  |  |  |
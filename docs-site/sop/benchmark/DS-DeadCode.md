> [!WARNING]
> **已废弃（2026-09-18）**：本评测数据集配套的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，统一替换为 mem0（见仓库 docs/quick-ref/mem0-ai-tools-config-guide.md）。本文仅作历史归档。

# DS-DeadCode · 死代码发现数据集

> **状态**：📘 数据集规范 v0.1｜2026-07-06
> **主战场**：B · codebase-mem-mcp `search_graph`（degree filter）+ `query_graph` Cypher
> **样本数**：50（每条 = 1 个候选函数/类 + 是否 dead + ground truth 证据链）
> **预期主模式**：B-only
> **数据文件**：`datasets/DS-DeadCode.jsonl`
> **核心目标**：验证 codebase-mem-mcp 静态图谱对**未引用代码**的识别精度。

---

## 1. 任务定义

**目标**：给定一个 Function / Class 节点，判断它是否为死代码（即无任何 inbound CALLS / HTTP_CALLS / LISTENS_ON 等被调用关系）。

**示例 query**：
- "adsop.module.legacy.OldReportService 是死代码吗？"
- "列出所有 degree=0 的 Function 节点"

**Ground truth**：通过 git blame + 运行时 APM 校验得到「该节点确实无任何引用」。

---

## 2. 样本格式（JSONL）

```json
{
  "id": "DS-DC-0001",
  "candidate": {
    "qualified_name": "adsop.module.legacy.OldReportService.exportToExcel2007",
    "label": "Function",
    "file": "adsop-module-legacy/src/main/java/.../OldReportService.java",
    "line": 87,
    "language": "java"
  },
  "is_dead": true,
  "graph_evidence": {
    "inbound_count": 0,
    "outbound_count": 3,
    "last_modified_commit": "abc123def",
    "last_modified_at": "2023-11-08",
    "last_call_from_commit": null
  },
  "runtime_evidence": {
    "apm_calls_30d": 0,
    "apm_calls_90d": 0,
    "apm_calls_365d": 0,
    "apm_last_call_at": null
  },
  "expected_tools": [
    {
      "tool": "search_graph",
      "args": {"label": "Function", "name_pattern": ".*OldReportService.*"},
      "expected_degree": 0
    },
    {
      "tool": "query_graph",
      "args": {"query": "MATCH (n:Function {name: 'OldReportService.exportToExcel2007'})<-[:CALLS|HTTP_CALLS|LISTENS_ON]-() RETURN count(*) AS inbound"},
      "expected_value": 0
    }
  ],
  "metadata": {
    "category": "legacy",
    "language": "java",
    "lines_since_last_modify": 700,
    "difficulty": "easy"
  }
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | ✅ | DS-DC-NNNN |
| `candidate.qualified_name` | string | ✅ | 候选节点限定名 |
| `candidate.label` | enum | ✅ | Function / Class |
| `is_dead` | bool | ✅ | 是否真正死代码 |
| `graph_evidence.inbound_count` | int | ✅ | 入边数（CALLS / HTTP_CALLS / LISTENS_ON 等） |
| `graph_evidence.outbound_count` | int | ✅ | 出边数（本节点调用了谁） |
| `runtime_evidence.apm_calls_30d` | int | ✅ | 近 30 天 APM 调用次数 |
| `runtime_evidence.apm_last_call_at` | string / null | ✅ | 最近一次运行时调用时间 |
| `expected_tools` | array | ✅ | 验证 dead code 应调用的工具 + 期望值 |
| `metadata.language` | enum | ✅ | java / ts / py / go |
| `metadata.lines_since_last_modify` | int | ✅ | 距上次修改的提交数 |

---

## 3. 类别分布

| 类别 | 样本数 | 含义 |
|------|--------|------|
| 真死代码 | 30 | runtime + graph 双证据均显示 0 引用 |
| 真活代码 | 10 | 有真实运行时调用 |
| 边界情况 | 10 | 见下方 |

### 3.1 边界情况子分类（10 条）

| 边界 | 样本数 | 含义 |
|------|--------|------|
| 仅测试引用 | 3 | 仅被 *Test.java / *.spec.ts 引用 |
| 仅反射调用 | 2 | 通过 Class.forName 等反射调用 |
| 仅定时任务调用 | 2 | 被 @Scheduled / cron 触发 |
| 仅事件订阅 | 2 | 通过 @EventListener / on('event') 触发 |
| 注释 @Deprecated | 1 | 已被标注 deprecated 但仍有调用 |

---

## 4. Ground Truth 标注流程

### 4.1 三源交叉

```
图谱证据  +  Git 历史  +  APM 运行时   ⇒  是否 dead
```

1. **图谱证据**：`codebase-memory-mcp query_graph` 数 inbound 边
2. **Git 历史**：`git log -p <file>` 找最近一次调用方修改时间
3. **运行时 APM**：查近 30 / 90 / 365 天调用次数

三源中至少 2 个为 0 视为 dead code。

### 4.2 双向独立

- 由 1 名代码考古师（熟悉 git）+ 1 名 SRE（熟悉 APM）独立标注
- 不一致时由第 3 名架构师仲裁

### 4.3 边界情况处理

- 测试代码引用 → 标记为 **「测试死代码」**，不算 dead（仍有清理价值但非紧急）
- 反射 / 定时 / 事件 → 标记为 **「隐性活跃」**，图谱易误报
- @Deprecated → 单独统计，不混入 dead

---

## 5. 评测运行

```python
for sample in load_jsonl("DS-DeadCode.jsonl"):
    # B-only：用图谱判断
    response_inbound = mcp_call(server="codebase-memory-mcp", tool="query_graph",
                               args={"query": f"MATCH (n {{qualified_name: '{sample['candidate']['qualified_name']}'}})<-[r:CALLS|HTTP_CALLS|LISTENS_ON]-() RETURN count(r)"})
    detected_dead = (response_inbound["count"] == 0)

    # 对照 ground truth
    metrics.update({
        "true_positive":  detected_dead and sample["is_dead"],
        "false_positive": detected_dead and not sample["is_dead"],
        "false_negative": not detected_dead and sample["is_dead"],
        "true_negative":  not detected_dead and not sample["is_dead"]
    })
```

---

## 6. 通过标准

| 指标 | 阈值 | 含义 |
|------|------|------|
| **检出率（recall）** | **≥ 0.90** | 真死代码中至少 90% 被识别 |
| **精确率（precision）** | **≥ 0.85** | 识别为死代码的样本中至少 85% 真是 |
| F1 | ≥ 0.87 | 综合指标 |
| **边界 case 误报率** | **≤ 20%** | 反射 / 定时 / 事件不应被误判为 dead |
| 索引覆盖率 | ≥ 0.95 | 前置条件：图谱已索引 ≥ 95% 代码 |

### 6.1 关键失败模式

| 失败 | 含义 | 处理 |
|------|------|------|
| F1 < 0.87 | 图谱 / 查询路径有问题 | 检查 Tree-Sitter 解析 + Hybrid LSP 类型解析 |
| 边界误报 > 20% | 没识别反射 / 事件订阅 | 引入运行时 trace 反哺（§5.10.3 P3-2） |

---

## 7. 样例样本（5 条）

```json
{"id": "DS-DC-0001", "candidate": {"qualified_name": "adsop.module.legacy.OldReportService.exportToExcel2007", "label": "Function", "file": ".../OldReportService.java", "line": 87, "language": "java"}, "is_dead": true, "graph_evidence": {"inbound_count": 0, "outbound_count": 3, "last_modified_commit": "abc123def", "last_modified_at": "2023-11-08", "last_call_from_commit": null}, "runtime_evidence": {"apm_calls_30d": 0, "apm_calls_90d": 0, "apm_calls_365d": 0, "apm_last_call_at": null}, "expected_tools": [{"tool": "search_graph", "args": {"label": "Function", "name_pattern": ".*OldReportService.*"}, "expected_degree": 0}], "metadata": {"category": "legacy", "language": "java", "lines_since_last_modify": 700, "difficulty": "easy"}}
{"id": "DS-DC-0002", "candidate": {"qualified_name": "adsop.module.system.service.UserService.login", "label": "Function", "file": ".../UserService.java", "line": 42, "language": "java"}, "is_dead": false, "graph_evidence": {"inbound_count": 4, "outbound_count": 2}, "runtime_evidence": {"apm_calls_30d": 12483, "apm_calls_90d": 38124, "apm_calls_365d": 152000, "apm_last_call_at": "2026-07-06"}, "expected_tools": [{"tool": "search_graph", "args": {"name_pattern": "UserService.login"}, "expected_degree": 4}], "metadata": {"category": "core", "language": "java", "lines_since_last_modify": 5, "difficulty": "easy"}}
{"id": "DS-DC-0003", "candidate": {"qualified_name": "adsop.module.cron.MonthlyReportJob.run", "label": "Function", "file": ".../MonthlyReportJob.java", "line": 23, "language": "java"}, "is_dead": false, "graph_evidence": {"inbound_count": 0, "outbound_count": 5, "note": "@Scheduled 注解触发，图谱无 inbound"}, "runtime_evidence": {"apm_calls_30d": 1, "apm_calls_90d": 3, "apm_calls_365d": 12, "apm_last_call_at": "2026-07-01T03:00:00Z"}, "metadata": {"category": "scheduled_task", "language": "java", "lines_since_last_modify": 30, "difficulty": "medium", "note": "定时任务，边界 case"}}
{"id": "DS-DC-0004", "candidate": {"qualified_name": "adsop.module.notify.listener.OrderPaidListener.onMessage", "label": "Function", "file": ".../OrderPaidListener.java", "line": 18, "language": "java"}, "is_dead": false, "graph_evidence": {"inbound_count": 0, "outbound_count": 4, "note": "@EventListener 触发"}, "runtime_evidence": {"apm_calls_30d": 5421, "apm_calls_90d": 16000, "apm_calls_365d": 64000, "apm_last_call_at": "2026-07-06"}, "metadata": {"category": "event_listener", "language": "java", "lines_since_last_modify": 12, "difficulty": "medium", "note": "事件订阅，边界 case"}}
{"id": "DS-DC-0005", "candidate": {"qualified_name": "adsop.module.kms.util.LegacyTokenizer.split", "label": "Function", "file": ".../LegacyTokenizer.java", "line": 5, "language": "java"}, "is_dead": false, "graph_evidence": {"inbound_count": 1, "outbound_count": 0, "note": "仅被 *Test.java 引用"}, "runtime_evidence": {"apm_calls_30d": 0, "apm_calls_90d": 0, "apm_calls_365d": 0, "apm_last_call_at": null}, "metadata": {"category": "test_only", "language": "java", "lines_since_last_modify": 200, "difficulty": "medium", "note": "仅测试引用，边界 case"}}
```

---

## 8. 变更记录

| 版本 | 日期 | 作者 | 变更 |
|------|------|------|------|
| v0.1 | 2026-07-06 | AI 架构助手 | 初稿：50 条样本规范 + 三源交叉标注 + 边界 case 处理 |
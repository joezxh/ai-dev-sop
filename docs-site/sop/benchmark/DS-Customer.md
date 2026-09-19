> [!WARNING]
> **已废弃（2026-09-18）**：本评测数据集配套的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，统一替换为 mem0（见仓库 docs/quick-ref/mem0-manual.md）。本文仅作历史归档。

# DS-Customer · 客户原话 verbatim 保真数据集

> **状态**：📘 数据集规范 v0.1｜2026-07-06
> **主战场**：A · MemPalace（verbatim 存储）
> **样本数**：30（每条 = 1 段客户原话 + 期望检索响应）
> **预期主模式**：A-only
> **数据文件**：`datasets/DS-Customer.jsonl`
> **核心目标**：验证 MemPalace **不总结 / 不改写 / 不提取** 的核心承诺。

---

## 1. 任务定义

**目标**：给定一段客户原话（来自客户邮件 / 会议纪要 / IM 截图），通过 MemPalace 检索，应返回**逐字匹配**的 drawer 内容。

**关键不变量**：
- ✅ 检索返回的 drawer `content` 字段必须与原话 **字符级 100% 一致**
- ❌ 任何总结、改写、提取、润色都视为保真失败

**示例 query**：
- 给定客户原话片段 "我们希望支持中文语义检索，但是担心 embedding 模型的费用"
- 检索应返回包含完整原话的 drawer

---

## 2. 样本格式（JSONL）

```json
{
  "id": "DS-CUST-0001",
  "source": {
    "channel": "email",
    "from": "customer_abc@partner.com",
    "subject": "Re: 业务平台需求沟通",
    "received_at": "2026-05-12",
    "raw_quote": "我们希望支持中文语义检索，但是担心 embedding 模型的费用。如果 PGVector 能跑通，我们愿意做 POC。"
  },
  "expected_drawer": {
    "drawer_id": "drawer_4e2c81",
    "wing": "wing-person-fde",
    "room": "customer-requests",
    "hall": "events",
    "stored_content": "我们希望支持中文语义检索，但是担心 embedding 模型的费用。如果 PGVector 能跑通，我们愿意做 POC。",
    "stored_at": "2026-05-12",
    "added_by": "joe"
  },
  "fidelity_check": {
    "expected_match_rate": 1.00,
    "expected_edit_distance": 0,
    "expected_summary_present": false
  },
  "metadata": {
    "language": "zh",
    "length_chars": 48,
    "contains_sensitive": false,
    "source_category": "customer_requirement",
    "difficulty": "easy"
  }
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | ✅ | DS-CUST-NNNN |
| `source.channel` | enum | ✅ | email / im / meeting / phone |
| `source.raw_quote` | string | ✅ | 客户原话（已脱敏） |
| `expected_drawer.drawer_id` | string | ✅ | 期望命中的 drawer |
| `expected_drawer.stored_content` | string | ✅ | 抽屉存储内容（应 = raw_quote） |
| `fidelity_check.expected_match_rate` | float | ✅ | 期望字符匹配率 |
| `fidelity_check.expected_edit_distance` | int | ✅ | 期望编辑距离（0 = 完全一致） |
| `fidelity_check.expected_summary_present` | bool | ✅ | 期望响应中是否含总结（必须 false） |
| `metadata.length_chars` | int | ✅ | 原话字符数（验证 800 char 切片边界） |
| `metadata.contains_sensitive` | bool | ✅ | 是否含敏感信息（邮箱 / 电话 / 合同金额等） |

---

## 3. 类别分布

| 长度 | 样本数 | 切片边界测试 |
|------|--------|------------|
| < 200 字 | 8 | 单切片内 |
| 200–500 字 | 10 | 单切片 |
| 500–800 字 | 7 | 单切片边界 |
| **800 字切片边界** | **5** | 切片重叠区（验证 100 字 overlap 内的检索） |
| 跨切片（> 800） | 5 | 多切片拼接（验证分块逻辑） |
| 含敏感信息 | 3 | 含邮箱 / 手机号 / 合同金额 |
| 跨语言（混中英） | 2 | 含 code-switching |

---

## 4. 保真度校验流程

### 4.1 字符级 diff

```python
from difflib import SequenceMatcher

matcher = SequenceMatcher(None, raw_quote, response_content)
match_rate = matcher.ratio()
edit_distance = len(raw_quote) + len(response_content) - 2 * sum(block.size for block in matcher.get_matching_blocks())
```

### 4.2 总结检测

```python
# 用 LLM 检测响应是否被改写
llm_judge_prompt = f"""
原文：{raw_quote}
响应：{response_content}

请判断响应是否：
1. 完全照搬原文（✅ 通过）
2. 进行了总结或提取关键点（❌ 失败）
3. 进行了润色或同义改写（❌ 失败）

仅回答 PASS 或 FAIL + 一句话理由。
"""
```

### 4.3 双侧独立

- A 侧由 1 名熟悉 FDE 业务的标注员核验
- B 侧由 1 名算法工程师独立校验字符 diff
- IoU ≥ 1.0（即字符级完全匹配）

---

## 5. 评测运行

```python
for sample in load_jsonl("DS-Customer.jsonl"):
    # A-only 检索
    response = mcp_call(server="mempalace", tool="mempalace_search",
                       args={"query": sample["source"]["raw_quote"],
                             "wing_scope": [sample["expected_drawer"]["wing"]]})

    # 字符级 diff
    fidelity = compute_fidelity(sample["source"]["raw_quote"], response)

    # 总结检测
    summary_judgement = llm_judge(sample["source"]["raw_quote"], response)

    metrics.update({
        "match_rate": fidelity["match_rate"],
        "edit_distance": fidelity["edit_distance"],
        "summary_present": summary_judgement["has_summary"]
    })
```

---

## 6. 通过标准

| 指标 | 阈值 | 含义 |
|------|------|------|
| **match_rate** | **≥ 0.99** | 字符匹配率（含切片边界容忍 ±1 char） |
| **edit_distance** | **= 0** | 期望编辑距离为 0 |
| **summary_present** | **= false** | 任何总结都视为失败 |
| 切片边界正确率 | 100% | 跨 800 char 的样本应正确拼接 |
| 敏感信息脱敏率 | 100% | 邮箱 / 电话应被 PII 钩子脱敏（不能直接入库） |

### 6.1 关键失败模式

| 失败 | 含义 | 处理 |
|------|------|------|
| match_rate < 0.99 | MemPalace 切片时损坏了原文 | 检查 800 char 切片 + 100 char overlap 实现 |
| summary_present = true | 有人为总结干预 | 检查是否有 LLM 在 add-drawer 路径上 |
| 跨切片样本出现丢字 | 切片边界丢失 | 验证 overlap 实现 |

---

## 7. 样例样本（5 条）

```json
{"id": "DS-CUST-0001", "source": {"channel": "email", "raw_quote": "我们希望支持中文语义检索，但是担心 embedding 模型的费用。如果 PGVector 能跑通，我们愿意做 POC。"}, "expected_drawer": {"drawer_id": "drawer_4e2c81", "wing": "wing-person-fde", "room": "customer-requests", "hall": "events", "stored_content": "我们希望支持中文语义检索，但是担心 embedding 模型的费用。如果 PGVector 能跑通，我们愿意做 POC。", "stored_at": "2026-05-12", "added_by": "joe"}, "fidelity_check": {"expected_match_rate": 1.00, "expected_edit_distance": 0, "expected_summary_present": false}, "metadata": {"language": "zh", "length_chars": 48, "contains_sensitive": false, "source_category": "customer_requirement", "difficulty": "easy"}}
{"id": "DS-CUST-0002", "source": {"channel": "meeting", "raw_quote": "客户张总在会上原话：我不管你们用什么技术，业务成功率必须提升 20%，这是死命令。"}, "expected_drawer": {"drawer_id": "drawer_8a91f3", "wing": "wing-person-fde", "room": "customer-requests", "hall": "events", "stored_content": "客户张总在会上原话：我不管你们用什么技术，业务成功率必须提升 20%，这是死命令。"}, "fidelity_check": {"expected_match_rate": 1.00, "expected_edit_distance": 0, "expected_summary_present": false}, "metadata": {"language": "zh", "length_chars": 41, "contains_sensitive": false, "source_category": "kpi_demand", "difficulty": "easy"}}
{"id": "DS-CUST-0003", "source": {"channel": "im", "raw_quote": "催办：合同尾款 [email protected] 还没打，催一下。"}, "expected_drawer": {"drawer_id": "drawer_pii_redacted_xxxx", "wing": "wing-person-fde", "room": "finance", "hall": "events", "stored_content": "催办：合同尾款 [email protected] 还没打，催一下。", "note": "邮箱已脱敏"}, "fidelity_check": {"expected_match_rate": 1.00, "expected_edit_distance": 0, "expected_summary_present": false, "expected_pii_redacted": true}, "metadata": {"language": "zh", "length_chars": 32, "contains_sensitive": true, "source_category": "finance", "difficulty": "medium"}}
{"id": "DS-CUST-0004", "source": {"channel": "email", "raw_quote": "Hi team, we're evaluating your adsop platform for our EU clients. Could you confirm GDPR compliance? Best, Anna from Berlin."}, "expected_drawer": {"drawer_id": "drawer_intl_eu_xxxx", "wing": "wing-person-fde", "room": "compliance", "hall": "events", "stored_content": "Hi team, we're evaluating your adsop platform for our EU clients. Could you confirm GDPR compliance? Best, Anna from Berlin."}, "fidelity_check": {"expected_match_rate": 1.00, "expected_edit_distance": 0, "expected_summary_present": false}, "metadata": {"language": "en", "length_chars": 134, "contains_sensitive": false, "source_category": "compliance", "difficulty": "medium"}}
{"id": "DS-CUST-0005", "source": {"channel": "meeting", "raw_quote": "关于涉外案件的数据出境合规问题，我们目前确认三件事：1. 数据不出境 2. 客户清单匿名化 3. 接口调用审计可追溯。请按此三条执行。"}, "expected_drawer": {"drawer_id": "drawer_3c9e44", "wing": "wing-person-fde", "room": "compliance", "hall": "events", "stored_content": "关于涉外案件的数据出境合规问题，我们目前确认三件事：1. 数据不出境 2. 客户清单匿名化 3. 接口调用审计可追溯。请按此三条执行。"}, "fidelity_check": {"expected_match_rate": 1.00, "expected_edit_distance": 0, "expected_summary_present": false}, "metadata": {"language": "zh", "length_chars": 75, "contains_sensitive": false, "source_category": "compliance", "difficulty": "easy"}}
```

---

## 8. 变更记录

| 版本 | 日期 | 作者 | 变更 |
|------|------|------|------|
| v0.1 | 2026-07-06 | AI 架构助手 | 初稿：30 条样本规范 + 字符级 diff + LLM judge |
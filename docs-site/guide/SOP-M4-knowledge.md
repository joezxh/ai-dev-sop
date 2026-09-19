# SOP-M4: 知识沉淀指南

> **版本**: v2.0
> **适用阶段**: 开发流程 M4 - 知识沉淀
> **目标读者**: 开发者
> **沉淀方式**: **mem0** 沉淀——Agent 自动提交（MCP `add_memory`）+ Dashboard 人工浏览，
> AI 归纳/蒸馏由 Agent 在会话内完成（结合 `infer=True` 事实抽取）。

---

## 1. 概述

### 1.1 目标

将开发过程中的知识结构化沉淀，写入 mem0 项目共享池供团队复用。

### 1.2 mem0 沉淀模式

```
┌─────────────────────────────────────────────────────────────────┐
│                     mem0 知识沉淀                                  │
├───────────────────────────┬─────────────────────────────────────┤
│   Agent 自动提交            │     Dashboard 人工管理               │
│   (MCP add_memory)        │     (localhost:3001)                │
├───────────────────────────┼─────────────────────────────────────┤
│  ✓ 会话留痕（每轮 Q/A）     │  ✓ 浏览 / 检索 / 分组查看            │
│  ✓ 决策与事实记录          │  ✓ Copilot 辅助归纳                  │
│  ✓ 最佳实践 / 经验教训      │  ✓ 按 session_id / type 过滤         │
│  ✓ infer=True 事实抽取      │  ✓ Webhooks 事件通知                 │
└───────────────────────────┴─────────────────────────────────────┘
```

---

## 2. 分类体系（metadata.type）

### 2.1 类型说明

| type | 说明 | 内容类型 | 示例 |
|------|------|----------|------|
| `fact` | 事实 | 技术决策、架构约束 | "我们使用 PostgreSQL 作为主数据库" |
| `decision` | 决策 | ADR、方案取舍 | "选择 EntityGraph 而非 DTO 投影" |
| `preference` | 偏好 | 代码风格、工具选择 | "团队偏好使用 Lombok 简化 POJO" |
| `note` | 经验/发现 | 性能问题、Bug 根因、建议 | "发现 N+1 查询问题，使用 EntityGraph 解决" |
| `conversation` | 会话留痕 | 每轮 Q/A 原文 | 见 `CODEBUDDY.md` §2.1 |

### 2.2 作用域约定

| 维度 | 取值 | 示例 |
|------|------|------|
| 个人归属 | `user_id` | `admin@mem0.dev`（见 `.codebuddy/mem0.config.json` roster） |
| 项目归属 | `project_id`（由 `git_remote` 解析） | `ai-dev-sop` |
| 关联人物 | `metadata.people[]` | `["alice", "bob"]` |

---

## 3. 知识归纳（AI 辅助）

### 3.1 适用场景

- 架构评审会议后
- 技术方案讨论后
- Bug 分析会议后
- Sprint 回顾后

### 3.2 归纳流程

```
┌─────────────────────────────────────────────────────────────┐
│                  会话归纳流程 (Agent + mem0)                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 对 Agent 提供会议纪要 / 会话上下文                        │
│     ↓                                                        │
│  2. 要求 Agent 归纳关键信息（事实/决策/发现/偏好/建议）        │
│     ↓                                                        │
│  3. Agent 逐条组织为自然语言陈述                              │
│     ↓                                                        │
│  4. 逐条 add_memory(text=..., metadata.type=...)             │
│     （或合并为一条 add_memory(infer=True) 让 mem0 抽取）       │
│     ↓                                                        │
│  5. search_memories 验证写入结果                              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 3.3 归纳结果示例

```
fact:      决定使用 DDD 架构模式；用户模块采用 CQRS 读写分离
decision:  订单模块使用 Saga 模式处理分布式事务
note:      当前单体架构难以支持高并发，需要提前规划微服务拆分
preference: 团队倾向于使用注解式事务；偏好 Lombok 简化代码
note(建议): 新模块建议直接从 DDD Template 创建；跨模块调用优先事件驱动
```

---

## 4. 知识蒸馏（深度提炼）

### 4.1 适用场景

- 性能问题解决后
- 复杂 Bug 修复后
- 技术难点突破后

### 4.2 蒸馏流程

```
1. Agent 回顾解决过程（问题 → 原因 → 方案 → 验证）
   ↓
2. 提炼为 3 类知识：
   - knowledge_fragments → type=note
   - decisions           → type=decision
   - tech_debt           → type=note（正文标注"技术债务"）
   ↓
3. 逐条 add_memory（含上下文与建议）
   ↓
4. search_memories 验证
```

### 4.3 蒸馏结果示例

**知识碎片（type=note）**：
```
使用 @EntityGraph 注解可以解决 N+1 查询问题，配合 @QueryHints 实现加载超时控制。
来源: OrderService 优化过程
```

**决策记录（type=decision）**：
```
选择 EntityGraph 而非 DTO 投影：EntityGraph 更灵活，可动态控制加载深度，
适合复杂聚合根场景。上下文: 用户模块订单查询优化。
```

**技术债务（type=note）**：
```
[技术债务] 历史代码未使用批量操作：早期实现存在 for 循环内单条查询，影响中等。
建议重构为 MyBatis-Plus batch 操作。
```

---

## 5. 手动添加记忆

### 5.1 添加架构决策

```bash
add_memory(
  text="ADR-001: 选择 PostgreSQL 作为主数据库。原因: 1. 支持 JSONB 类型，适合灵活 schema;
        2. pgvector 支持向量存储; 3. 团队熟悉度高。决策日期: 2024-01-15，决策者: 技术委员会",
  api_key="<from config>", git_remote="<from config>",
  user_id="admin@mem0.dev", project_id="ai-dev-sop",
  metadata={"type":"decision", "people":["技术委员会"], "created_at":"<ISO8601>"}
)
```

### 5.2 添加最佳实践

```bash
add_memory(
  text="API 错误处理规范: 1. 使用统一的 ErrorResponse 结构; 2. HTTP 状态码遵循语义
        (4xx 客户端错误 / 5xx 服务端错误); 3. 错误消息对用户友好，内部日志记录详细堆栈;
        4. 敏感信息脱敏处理",
  metadata={"type":"note", "people":[], "created_at":"<ISO8601>"}
)
```

### 5.3 添加经验教训

```bash
add_memory(
  text="支付模块优化经验: 使用 Redis 分布式锁解决并发重复支付; 幂等键设计 orderId+timestamp;
        补偿机制为定时任务扫描超时订单; 幂等性通过唯一索引保证",
  metadata={"type":"note", "people":["支付模块负责人"], "created_at":"<ISO8601>"}
)
```

### 5.4 标签

mem0 无独立标签操作，主题标签写入 `metadata`（如 `"topic": ["支付","Redis"]`），
或在正文中自然提及，由语义检索覆盖。

---

## 6. 沉淀最佳实践

### 6.1 选择合适的内容

| 内容类型 | 是否沉淀 | 说明 |
|----------|----------|------|
| 架构评审结论 | ✅ type=decision/fact | 归纳决策和共识 |
| Bug 根因分析 | ✅ type=note | 根因 + 修复方案 |
| 性能优化经验 | ✅ type=note | 方案 + 数据 |
| 日常开发琐事 | ❌ | 不够 durable，不要入库 |

### 6.2 提交流程

```
1. 确认内容质量（具体、可操作、有上下文）
   ↓
2. 组织为自然语言陈述
   ↓
3. add_memory 提交（mem0 按内容哈希去重，重复提交安全）
   ↓
4. search_memories 验证写入结果
```

---

## 7. 知识质量标准

### 7.1 高质量知识特征

| 特征 | 说明 | 示例 |
|------|------|------|
| 具体 | 有明确的上下文 | "使用 @EntityGraph 解决 N+1" |
| 可操作 | 他人能据此行动 | "在 OrderRepository 方法上加注解" |
| 有价值 | 避免显而易见的内容 | "不要使用 SELECT *" |
| 结构化 | 有清晰的组织 | "问题→原因→解决方案" |

### 7.2 应该避免的内容

- 过于显而易见的常识
- 无法验证的猜测
- 与已有记忆重复的内容（先 `search_memories` 查重）
- 缺乏上下文的孤立信息

### 7.3 知识更新

```bash
# 1. 找到旧记忆的 memory_id
search_memories(query="<旧记忆关键词>")

# 2. 更新内容
update_memory(memory_id="xxx", text="更新后的内容...")

# 3. 新事实替代旧事实也可直接写入新记忆：
#    mem0 的 Dream 机制（Supersede/Merge）会自动淘汰过时记忆、合并重复项
```

---

## 8. 验证清单

### 8.1 内容验证

| 检查项 | 验证方式 |
|--------|----------|
| 类型正确 | `metadata.type` 与内容匹配 |
| 内容完整 | 包含必要的上下文 |
| 可操作性 | 他人看后知道如何行动 |

### 8.2 流程验证

```bash
# 查看写入结果
get_memories(api_key=..., project_id="ai-dev-sop")

# 搜索验证
search_memories(query="API 错误处理", top_k=5)

# Dashboard 人工复核
# http://localhost:3001 → Memories → 按 type 过滤
```

---

## 9. 常见问题

### 9.1 归纳结果质量差

```
原因: 会话内容不够丰富
解决: 提供更完整的上下文（会议纪要/决策记录），让 Agent 分主题归纳
```

### 9.2 检索不到刚写入的记忆

```
原因: 作用域不一致（user_id / project_id / api_key 不匹配）
解决: 统一使用 .codebuddy/mem0.config.json 中的凭证与 project_id
```

### 9.3 重复提交

```
解决: mem0 按内容哈希去重，重复提交不会产生重复内容；
     Dream(Merge) 会在后台合并语义重复的记忆
```

---

*文档更新: 2026-09-18*
*下一步: [SOP-M5: 团队协作](./SOP-M5-collaboration.md)*

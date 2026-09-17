# mem0 记忆系统 AI IDE 自动化配置

> **版本**: v2.0
> **适用阶段**: Phase 4 - AI IDE 自动化
> **变更说明**: 旧双轨工具（MemPalace / codebase-mem-mcp）已移除。自动化不再依赖
> Cursor rules JSON 等工具路由配置，统一改为 **Agent 规则文件**（CODEBUDDY.md 模式）
> + mem0 MCP 工具，行为见根 `CODEBUDDY.md` 与
> [quick-ref/mem0-ai-tools-config-guide.md §9](../quick-ref/mem0-ai-tools-config-guide.md)。

---

## 1. 概述

### 1.1 目标

配置 AI IDE 自动调用 mem0 记忆工具，实现智能化的开发辅助体验。

### 1.2 自动化架构

```
┌──────────────────────────────────────────────────────┐
│              Agent 规则文件（自动注入）                  │
│   CODEBUDDY.md / AGENTS.md / CLAUDE.md / .qoder/rules │
├──────────────────────────────────────────────────────┤
│              触发行为                                   │
│   ① 会话启动 → get_memories 拉取项目共享池              │
│   ② 任务相关 → search_memories 语义检索                │
│   ③ durable 事实 → add_memory 自动提交                 │
│   ④ 每轮完成 → 会话留痕 add_memory(type=conversation)  │
├──────────────────────────────────────────────────────┤
│              mem0-local MCP (:8080/mcp)                │
└──────────────────────────────────────────────────────┘
```

---

## 2. 规则设计原则

1. **上下文感知**: 根据用户问题上下文决定是否检索记忆
2. **先记忆后代码**: 先 `search_memories` 获取团队约定，再分析代码实现
3. **渐进式**: 先全量拉取共享池，再按关键词精准检索（`top_k` 控制）
4. **显式降级**: 配置缺失/服务不可达时告知用户并停止，不编造记忆

---

## 3. 意图 → 工具映射

| 用户意图（关键词） | 调用工具 | 参数示例 |
|--------------------|----------|----------|
| 搜索代码、找代码 | IDE 原生（`search_code`/LSP） | `--query "{text}" --scope "**/*"` |
| 搜索记忆、团队知识 | `search_memories` | `query="{text}", top_k=5` |
| 查文档、查规范 | `search_memories` | `query="{text} 规范", top_k=5` |
| 影响分析、变更分析 | git diff + `findReferences` | - |
| 性能、慢查询 / Bug、问题 | LSP 调用链 + `search_memories` | 组合调用 |
| 记录、记住 | `add_memory` | `text=..., metadata.type=...` |
| 加载上下文 / 项目记忆 | `get_memories` | `project_id="ai-dev-sop"` |

---

## 4. 规则文件模板

各工具规则文件位置：

| 工具 | 文件 |
|------|------|
| CodeBuddy | 根 `CODEBUDDY.md`（本工程范本） |
| Codex / Qoder CLI | 根 `AGENTS.md` |
| Claude Code | `CLAUDE.md`（`@CODEBUDDY.md` 引用） |
| Cursor | `.cursor/rules/*.mdc` |
| Qoder IDE | `.qoder/rules/*.md` |

**最小可用规则模板**（新增工具时复制此骨架）：

```markdown
# Project Memory via mem0 MCP (mem0-local)

本工程使用自托管 mem0 作为长期记忆（MCP: mem0-local @ http://127.0.0.1:8080/mcp）。
凭证读 .codebuddy/mem0.config.json（api_key / git_remote / project_id / roster），
不得硬编码。

## 自动拉取（每会话首轮，主动执行）
1. get_memories(api_key=..., git_remote=...) 拉取项目共享池
2. 任务具体时 search_memories(query=..., project_id=..., top_k=5)
3. 向用户汇报 2-3 行召回摘要

## 自动提交（durable 事实/决策/偏好）
add_memory(text=..., api_key=..., git_remote=...,
           metadata={"type":"fact|decision|preference|note", ...})
- 会话留痕：每轮 add_memory("Q: ...\n\nA: ...", type=conversation,
  session_id=cb-<YYYYMMDD>-<6hex>, agent=<工具名>)

## 降级
配置缺失或服务不可达 → 告知用户并停止，不编造记忆。
```

---

## 5. 快捷指令（可选）

IDE 自定义指令与触发词示例（按各 IDE 的指令/快捷方式机制配置）：

| 指令 | 触发场景 | 自动调用 |
|------|----------|----------|
| `/understand` | 项目理解 | `search_memories` + IDE 架构分析 |
| `/remember` | 知识沉淀 | `add_memory`（交互确认 type/topic） |
| `/wake` | 加载上下文 | `get_memories(project_id=...)` |
| `/forget` | 删除记忆 | `search_memories` → 用户确认 → `delete_memory` |

---

## 6. 验证

| 场景 | 输入 | 预期调用 |
|------|------|----------|
| 项目理解 | "这个项目的架构是什么？" | `search_memories` + IDE 分析 |
| 代码定位 | "UserService 在哪里？" | IDE 语义导航（LSP） |
| 历史借鉴 | "这个问题之前遇到过吗？" | `search_memories` |
| 知识沉淀 | "帮我记住这个架构决策" | `add_memory`（type=decision） |
| 会话留痕 | 任意一轮对话后查询 | type=conversation 记录存在 |

自动化行为回归清单见 [verification/VERIFICATION-CHECKLIST.md](../verification/VERIFICATION-CHECKLIST.md)。

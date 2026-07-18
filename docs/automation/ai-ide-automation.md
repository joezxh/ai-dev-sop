# 双轨记忆系统 AI IDE 自动化配置

> **版本**: v1.0
> **适用阶段**: Phase 4 - AI IDE 自动化
> **目标读者**: 开发者、架构师

---

## 1. 概述

### 1.1 目标

配置 AI IDE 自动调用双轨工具，实现智能化的开发辅助体验。

### 1.2 自动化架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI IDE 自动化架构                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  用户问题 ──▶ AI 分析 ──▶ 自动调用 ──▶ 返回结果                   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────┐      │
│  │                  AI 引擎                                 │      │
│  │  ┌───────────────┐  ┌───────────────┐                 │      │
│  │  │  自然语言理解  │  │  规则匹配    │                 │      │
│  │  └───────┬───────┘  └───────┬───────┘                 │      │
│  │          │                  │                          │      │
│  │          └────────┬─────────┘                          │      │
│  │                   ▼                                    │      │
│  │          ┌─────────────────┐                           │      │
│  │          │  工具选择规则    │                           │      │
│  │          └────────┬────────┘                           │      │
│  └───────────────────┼─────────────────────────────────────┘      │
│                      ▼                                            │
│  ┌──────────────────────────────────────────────────────┐      │
│  │                    工具调用                             │      │
│  │  ┌──────────────┐  ┌──────────────┐                   │      │
│  │  │ MemPalace   │  │ codebase-    │                   │      │
│  │  │ (轨道 A)    │  │ mem-mcp     │                   │      │
│  │  │             │  │ (轨道 B)     │                   │      │
│  │  └──────────────┘  └──────────────┘                   │      │
│  └──────────────────────────────────────────────────────┘      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 工具自动调用规则

### 2.1 规则设计原则

1. **上下文感知**: 根据用户问题上下文选择合适的工具
2. **双轨协同**: 优先使用 MemPalace 获取团队记忆，再使用 Codebase 获取代码
3. **渐进式**: 先简单搜索，再深度分析

### 2.2 规则分类

#### 2.2.1 理解类规则

| 关键词 | 调用工具 | 参数示例 |
|--------|----------|----------|
| 架构、项目结构 | `get_architecture` | `--scope full` |
| 模块、组件 | `search_graph` | `--label class --name "{keyword}"` |
| 函数、方法 | `search_code` | `--query "{keyword}" --scope "**/*.java"` |
| 调用关系 | `trace_path` | `--target "{method}" --depth 5` |

#### 2.2.2 搜索类规则

| 关键词 | 调用工具 | 参数示例 |
|--------|----------|----------|
| 搜索代码、找代码 | `search_code` | `--query "{text}" --scope "**/*"` |
| 搜索记忆、团队知识 | `mempalace_search` | `query="{text}"` |
| 查文档、查规范 | `mempalace_search` | `query="{text}" --wing "team-backend"` |

#### 2.2.3 分析类规则

| 关键词 | 调用工具 | 参数示例 |
|--------|----------|----------|
| 影响分析、变更分析 | `detect_changes` | `--git_diff "{diff}"` |
| 性能、慢查询 | `trace_path` + `mempalace_search` | 组合调用 |
| Bug、问题、错误 | `trace_path` + `mempalace_search` | 组合调用 |

#### 2.2.4 知识沉淀类规则

| 关键词 | 调用工具 | 参数示例 |
|--------|----------|----------|
| 记录、记住 | `mempalace_add_drawer` | 多参数 |
| 保存、检查点 | `mempalace_checkpoint` | `--session_id "{id}"` |

---

## 3. 自动调用配置模板

### 3.1 Cursor Rules 配置

**配置文件位置**: `~/.cursor/rules/` 或项目 `.cursor/rules/`

#### 3.1.1 双轨理解规则

```json
// .cursor/rules/dual-track-understanding.json
{
  "name": "dual-track-understanding",
  "description": "双轨理解模式：先查 MemPalace 团队记忆，再查 Codebase 代码",
  "triggers": [
    "架构是什么",
    "怎么实现的",
    "项目结构",
    "模块关系",
    "使用了什么技术"
  ],
  "actions": [
    {
      "step": 1,
      "tool": "mempalace_search",
      "params": {
        "query": "{user_query}",
        "wing": "team-shared"
      },
      "description": "先搜索团队记忆"
    },
    {
      "step": 2,
      "tool": "get_architecture",
      "params": {
        "scope": "full"
      },
      "description": "获取架构概览"
    },
    {
      "step": 3,
      "tool": "search_code",
      "params": {
        "query": "{extracted_keyword}",
        "scope": "**/*.{lang}"
      },
      "description": "搜索相关代码"
    }
  ],
  "output_format": "综合团队记忆与代码分析的回答"
}
```

#### 3.1.2 调试追踪规则

```json
// .cursor/rules/debug-tracing.json
{
  "name": "debug-tracing",
  "description": "调试追踪模式：追踪调用链 + 搜索历史经验",
  "triggers": [
    "调用链",
    "被谁调用",
    "调用了谁",
    "问题在哪",
    "为什么会错"
  ],
  "actions": [
    {
      "step": 1,
      "tool": "trace_path",
      "params": {
        "target": "{extracted_method}",
        "depth": 10
      },
      "description": "追踪调用链"
    },
    {
      "step": 2,
      "tool": "mempalace_search",
      "params": {
        "query": "{method_name} {error_type} 历史"
      },
      "description": "搜索历史经验"
    }
  ],
  "output_format": "调用链 + 历史解决方案"
}
```

#### 3.1.3 变更影响规则

```json
// .cursor/rules/change-impact.json
{
  "name": "change-impact",
  "description": "变更影响分析：检测代码变更的影响范围",
  "triggers": [
    "影响",
    "改了这个",
    "会有什么后果",
    "回归测试"
  ],
  "actions": [
    {
      "step": 1,
      "tool": "detect_changes",
      "params": {
        "git_diff": "{current_diff}"
      },
      "description": "分析变更影响"
    },
    {
      "step": 2,
      "tool": "mempalace_search",
      "params": {
        "query": "回归测试 {module_name}"
      },
      "description": "搜索相关测试建议"
    }
  ],
  "output_format": "影响组件列表 + 风险级别 + 测试建议"
}
```

---

## 4. 提示词模板

### 4.1 项目理解提示词

```
你是一个代码架构分析师。请分析项目并回答用户的问题。

工作流程:
1. 先搜索 MemPalace 团队记忆 (mempalace_search)
2. 再分析代码结构 (get_architecture, search_code)
3. 综合给出回答

回答格式:
## 架构概览
[项目整体架构说明]

## 团队约定
[来自 MemPalace 的团队记忆]

## 代码实现
[来自 Codebase 的代码分析]

## 建议
[如果有相关建议]
```

### 4.2 代码搜索提示词

```
你是一个代码搜索引擎。请帮用户找到相关代码。

工作流程:
1. 搜索团队约定 (mempalace_search)
2. 搜索代码实现 (search_code, search_graph)
3. 获取代码片段 (get_code_snippet)

搜索关键词: {user_keyword}
文件类型: {file_type}
```

### 4.3 知识沉淀提示词

```
你是团队知识管理员。请帮助用户沉淀知识。

工作流程:
1. 询问用户要记录什么
2. 分析内容类型 (Hall 分类)
3. 建议合适的 Wing/Room
4. 调用 mempalace_add_drawer 保存

Hall 分类参考:
- hall_facts: 技术决策、架构约束
- hall_events: 会议、里程碑
- hall_discoveries: 性能问题、Bug 根因
- hall_preferences: 代码风格、工具选择
- hall_advice: 最佳实践、避坑指南
```

---

## 5. 快捷命令

### 5.1 Cursor 快捷指令

| 指令 | 触发场景 | 自动调用 |
|------|----------|----------|
| `/understand` | 项目理解 | `mempalace_search` + `get_architecture` |
| `/search` | 代码搜索 | `search_code` + `search_graph` |
| `/trace` | 调用追踪 | `trace_path` |
| `/impact` | 变更影响 | `detect_changes` |
| `/remember` | 知识沉淀 | `mempalace_add_drawer` |
| `/wake` | 加载上下文 | `mempalace_get_context` |

### 5.2 自定义快捷指令配置

```json
// .cursor/keybindings.json
{
  "keybindings": [
    {
      "key": "ctrl+shift+u",
      "command": "mcp:dual-track-understand",
      "description": "双轨理解"
    },
    {
      "key": "ctrl+shift+s",
      "command": "mcp:code-search",
      "description": "代码搜索"
    },
    {
      "key": "ctrl+shift+t",
      "command": "mcp:trace-path",
      "description": "调用追踪"
    },
    {
      "key": "ctrl+shift+r",
      "command": "mcp:remember",
      "description": "知识沉淀"
    }
  ]
}
```

---

## 6. 自动触发配置

### 6.1 AI 自动决策流程

```
┌─────────────────────────────────────────────────────────────┐
│                   AI 工具选择决策树                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  用户问题                                                     │
│     │                                                        │
│     ▼                                                        │
│  ┌──────────────────┐                                       │
│  │ 包含 "团队记忆"?  │                                       │
│  └────────┬─────────┘                                       │
│         YES │ NO                                             │
│           ▼ ▼                                                │
│  ┌─────────────┐  ┌──────────────────┐                       │
│  │mempalace    │  │ 包含 "代码"?      │                       │
│  │_search     │  └────────┬─────────┘                       │
│  └─────────────┘         YES │ NO                             │
│                             ▼ ▼                               │
│                    ┌─────────────┐  ┌─────────────┐          │
│                    │search_code │  │ 其他关键词   │          │
│                    │search_graph│  │ 使用规则表  │          │
│                    └─────────────┘  └─────────────┘          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 规则优先级

| 优先级 | 规则类型 | 说明 |
|--------|----------|------|
| 1 | 显式指令 | 用户明确说 "搜索代码" |
| 2 | 上下文关键词 | 包含特定关键词 |
| 3 | 语义匹配 | AI 判断用户意图 |
| 4 | 默认行为 | 无匹配时使用默认 |

---

## 7. 配置验证

### 7.1 测试场景

| 场景 | 输入 | 预期调用 |
|------|------|----------|
| 项目理解 | "这个项目的架构是什么？" | `mempalace_search` + `get_architecture` |
| 代码定位 | "UserService 在哪里？" | `search_code` |
| 调用追踪 | "getUser 的调用链是什么？" | `trace_path` |
| 变更影响 | "改 UserService 会影响哪些模块？" | `detect_changes` |
| 知识沉淀 | "帮我记住这个架构决策" | `mempalace_add_drawer` |

### 7.2 验证脚本

```bash
#!/bin/bash
# verify-auto-call.sh

echo "=== 验证自动调用配置 ==="

# 测试项目理解
echo "1. 测试项目理解..."
# TODO: 集成测试

# 测试代码搜索
echo "2. 测试代码搜索..."
# TODO: 集成测试

# 测试调用追踪
echo "3. 测试调用追踪..."
# TODO: 集成测试

echo "=== 验证完成 ==="
```

---

## 8. 最佳实践

### 8.1 规则编写原则

1. **简洁明确**: 规则描述清晰，避免歧义
2. **渐进式**: 从简单到复杂，逐步深入
3. **可测试**: 每个规则都有对应的测试场景
4. **可维护**: 规则易于修改和扩展

### 8.2 提示词优化

1. **具体示例**: 提供具体的输入输出示例
2. **角色设定**: 明确 AI 的角色和职责
3. **输出格式**: 指定期望的输出格式
4. **限制条件**: 明确不做的事情

### 8.3 持续优化

1. **收集反馈**: 记录用户对自动调用的满意度
2. **分析日志**: 分析工具调用频率和使用效果
3. **迭代规则**: 根据反馈持续优化规则

---

*文档更新: 2026-07-14*

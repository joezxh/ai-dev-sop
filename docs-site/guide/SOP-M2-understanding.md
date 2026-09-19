# SOP-M2: 项目理解指南

> **版本**: v2.0
> **适用阶段**: 开发流程 M2 - 项目理解
> **目标读者**: 开发者
> **记忆检索**: 统一使用 **mem0 MCP**（`search_memories` / `get_memories`），代码分析
> 使用 IDE 原生能力（语义导航 / 文本搜索）。

---

## 1. 概述

### 1.1 目标

理解项目结构和代码组织，加载相关上下文记忆，识别关键架构决策。

### 1.2 记忆 + 代码 双源理解模式

```
┌─────────────────────────────────────────────────────────────────┐
│                 mem0 记忆 + IDE 代码分析                            │
├───────────────────────────┬─────────────────────────────────────┤
│   mem0 (团队记忆)          │     IDE 原生代码分析                  │
├───────────────────────────┼─────────────────────────────────────┤
│  ✓ 架构决策               │  ✓ 项目结构                          │
│  ✓ 技术选型               │  ✓ 类图/调用关系（语义导航）          │
│  ✓ 团队约定               │  ✓ 代码片段                          │
│  ✓ 历史经验               │  ✓ 依赖关系                          │
└───────────────────────────┴─────────────────────────────────────┘

推荐流程:
1. 先查 mem0 记忆 → 理解团队约定（search_memories）
2. 再用 IDE 分析   → 理解代码实现（LSP / 文本搜索）
```

---

## 2. 首次项目理解

### 2.1 加载团队记忆

**目标**: 了解项目的背景、架构决策和团队约定。会话开始时 mem0 记忆由 Agent
按 `CODEBUDDY.md` 约定自动拉取（`get_memories(project_id=...)`），也可主动检索：

```bash
# 1. 搜索架构相关记忆
search_memories(query="项目架构 技术栈", top_k=5)

# 2. 搜索技术决策
search_memories(query="ADR 技术选型", top_k=5)

# 3. 全量加载项目共享池
get_memories(api_key=..., git_remote="<配置中的 git_remote>")
```

**示例问题**:
- "这个项目整体架构是什么样的？"
- "使用了哪些核心技术栈？"
- "有哪些重要的架构决策？"

### 2.2 分析代码结构

**目标**: 理解代码层面的架构和模块组织（使用 IDE 原生语义导航与文本搜索）。

```bash
# 1. 获取文件/符号结构（LSP documentSymbol）
documentSymbol("src/main/java/.../UserService.java")

# 2. 定位符号定义
workspaceSymbol(query="UserService")

# 3. 搜索关键模块
search_code --query "class UserService" --scope "**/*.java"
```

### 2.3 综合理解

**推荐问题模板**:

```
问: "这个项目的整体架构是什么样的？使用了哪些核心技术栈？"

AI 自动调用:
1. search_memories "项目架构 技术栈"
   → 查找团队记忆中的架构文档
2. IDE 语义导航（documentSymbol / workspaceSymbol）
   → 获取代码层面的架构分析
3. 文本搜索（search_code）
   → 定位关键实现

输出: 综合 mem0 团队记忆 + IDE 代码分析的结构化回答
```

---

## 3. 模块级理解

### 3.1 理解特定模块

**目标**: 深入理解某个功能模块的实现。

```bash
# 1. 搜索团队记忆
search_memories(query="用户认证 登录 OAuth", top_k=5)

# 2. 搜索代码符号（IDE 语义导航）
workspaceSymbol(query="User")

# 3. 追踪调用链路（LSP 调用层级）
incomingCalls(filePath="UserService.java", scope="login")
```

**推荐问题模板**:

```
问: "用户认证模块是怎么实现的？有哪些关键组件？"

AI 自动调用:
1. search_memories "用户认证 登录 OAuth"
   → 查找团队记忆中的认证决策
2. IDE 语义导航搜索 User 相关类
   → 定位认证相关实体
3. LSP 调用层级追踪 login 调用链

输出:
- mem0: "2024-01 决定使用 JWT + Redis 实现认证"
- IDE: 类结构 + 调用关系
```

### 3.2 理解数据模型

```bash
# 搜索实体类（IDE 语义导航）
workspaceSymbol(query="Order", kinds=["class"])

# 读取符号定义（含代码片段）
readSymbol(filePath="models/Order.java", scope="Order")
```

### 3.3 理解 API 设计

```bash
# 搜索 API 相关代码
search_code --query "RestController" --scope "**/*.java"

# 搜索端点定义
search_code --query "@GetMapping|@PostMapping" --scope "**/*.java"
```

---

## 4. 搜索与定位

### 4.1 搜索代码实现

**基础搜索**:

```bash
# 搜索代码片段
search_code --query "pagination" --scope "**/*.java"

# 定位符号定义（IDE 语义导航）
workspaceSymbol(query="User", kinds=["class"])

# 正则搜索
search_code --query "validate.*password" --scope "**/*.java"
```

**高级搜索**:

```bash
# 搜索包含特定注解的类
search_code --query "@Service|@Repository" --scope "**/*.java"

# 搜索异常处理
search_code --query "throw new.*Exception" --scope "**/*.java"
```

### 4.2 查找相似实现

```bash
# 1. 搜索类似功能
search_code --query "分页" --scope "**/*.java"

# 2. 搜索团队约定
search_memories(query="分页 最佳实践", top_k=5)

# 3. 综合结果
输出: 代码位置 + 团队约定的分页模式
```

### 4.3 追踪调用链路

```bash
# 追踪方法调用（LSP incoming/outgoing calls）
incomingCalls(filePath="OrderService.java", scope="checkStatus")

# 追踪入口到实现
outgoingCalls(filePath="UserController.java", scope="createOrder")
```

---

## 5. 架构分析

### 5.1 整体架构

```bash
# 目录结构总览
tree /F /A   # 或 IDE 文件树 / list_files

# 分层扫描（按包名聚合符号）
documentSymbol("src/main/java/com/example/order/...")

# 模块依赖（分析 import）
search_code --query "^import com\.example\." --scope "**/*.java"
```

### 5.2 模块架构

```bash
# 分析特定模块（IDE 语义导航 + 文件树）
documentSymbol("order-service/")

# 获取模块类图
workspaceSymbol(query="Order", kinds=["class"])
```

### 5.3 数据流分析

```bash
# 追踪数据流（LSP 调用层级）
outgoingCalls(filePath="OrderService.java", scope="createOrder")

# 分析入口到数据库
incomingCalls(filePath="OrderRepository.java", scope="save")
```

---

## 6. 最佳实践

### 6.1 理解流程

```
┌─────────────────────────────────────────────────────────────┐
│                    项目理解流程                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. mem0 团队记忆（get_memories / search_memories）           │
│     ↓                                                        │
│  2. IDE 语义导航获取架构概览                                  │
│     ↓                                                        │
│  3. search_code 搜索关键代码                                  │
│     ↓                                                        │
│  4. LSP 调用层级追踪调用链路                                  │
│     ↓                                                        │
│  5. add_memory 补充记忆（mem0）                              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 推荐问题集

| 场景 | 推荐问题 |
|------|----------|
| 首次接手 | "这个项目是做什么的？核心模块有哪些？" |
| 架构理解 | "模块间依赖关系是怎样的？" |
| 功能定位 | "订单处理流程涉及哪些类？" |
| 技术决策 | "为什么要用这个技术方案？" |
| 代码复用 | "类似的 XX 功能在哪里实现过？" |

### 6.3 注意事项

- **先团队后代码**: 先查 mem0 记忆理解团队约定，再用 IDE 分析代码实现
- **上下文加载**: 会话开始时 `get_memories(project_id=...)` 一次性拉取项目共享池
- **搜索范围**: 使用 `--scope` 参数或 `top_k` 限定范围，避免无关结果
- **记录理解**: 发现新信息时及时 `add_memory` 写入 mem0（规则见 `CODEBUDDY.md`）

---

## 7. 验证清单

### 7.1 理解验证

| 检查项 | 验证方式 |
|--------|----------|
| 理解项目目标 | 能用自己的话描述项目是做什么的 |
| 理解架构 | 能画出主要模块及其关系 |
| 理解数据模型 | 能说出核心实体及其关系 |
| 理解技术选型 | 能解释为什么选择这些技术 |

### 7.2 操作验证

```bash
# 架构分析（IDE 语义导航）
documentSymbol("src/main/java/com/example/Application.java")

# 搜索验证
search_code --query "main" --scope "**/*.java"

# 调用链验证（LSP）
incomingCalls(filePath="UserService.java", scope="getUser")
```

---

## 8. 常见问题

### 8.1 搜索结果太多

```bash
# 使用更精确的查询
search_code --query "UserService" --scope "**/service/*.java"

# 限制文件类型
search_code --query "pagination" --scope "**/*.java"
```

### 8.2 调用链太深

```bash
# 限制深度
trace_path --target "UserService" --depth 5

# 分段追踪
trace_path --target "UserController" --depth 3
trace_path --target "UserService" --depth 3
```

### 8.3 找不到相关信息

```bash
# 换关键词模糊检索（语义搜索对表述不敏感）
search_memories(query="<模糊关键词>", top_k=10)

# 枚举实体，确认记忆归属
list_entities()

# 检查项目共享池是否拉全
get_memories(api_key=..., project_id="ai-dev-sop", limit=100)
```

---

*文档更新: 2026-09-18*
*下一步: [SOP-M3: 开发调试](./SOP-M3-development.md)*

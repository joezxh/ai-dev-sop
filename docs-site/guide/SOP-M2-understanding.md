# SOP-M2: 项目理解指南


## 1. 概述

### 1.1 目标

理解项目结构和代码组织，加载相关上下文记忆，识别关键架构决策。

### 1.2 双轨理解模式

```
┌─────────────────────────────────────────────────────────────────┐
│                     双轨理解模式                                    │
├───────────────────────────┬─────────────────────────────────────┤
│   轨道 A: MemPalace       │     轨道 B: codebase-mem-mcp       │
│   (团队记忆)              │     (代码结构)                        │
├───────────────────────────┼─────────────────────────────────────┤
│  ✓ 架构决策               │  ✓ 项目结构                          │
│  ✓ 技术选型               │  ✓ 类图/调用关系                     │
│  ✓ 团队约定               │  ✓ 代码片段                          │
│  ✓ 历史经验               │  ✓ 依赖关系                          │
└───────────────────────────┴─────────────────────────────────────┘

推荐流程:
1. 先查 MemPalace → 理解团队约定
2. 再查 Codebase   → 理解代码实现
```

---

## 2. 首次项目理解

### 2.1 加载团队记忆

**目标**: 了解项目的背景、架构决策和团队约定。

```bash
# 1. 搜索架构相关记忆
mempalace search "项目架构 技术栈"

# 2. 搜索技术决策
mempalace search "ADR 技术选型"

# 3. 加载项目上下文
mempalace get-context --wing "project-myapp"
```

**示例问题**:
- "这个项目整体架构是什么样的？"
- "使用了哪些核心技术栈？"
- "有哪些重要的架构决策？"

### 2.2 分析代码结构

**目标**: 理解代码层面的架构和模块组织。

```bash
# 1. 获取架构概览
get_architecture --scope full

# 2. 获取图谱 Schema
get_graph_schema

# 3. 搜索关键模块
search_graph --label class --name "User"
search_graph --label class --name "Service"
```

### 2.3 综合理解

**推荐问题模板**:

```
问: "这个项目的整体架构是什么样的？使用了哪些核心技术栈？"

AI 自动调用:
1. mempalace_search "项目架构 技术栈"
   → 查找团队记忆中的架构文档
2. get_architecture --scope full
   → 获取代码层面的架构分析
3. get_graph_schema
   → 获取图谱 Schema

输出: 综合 MemPalace 团队记忆 + Codebase 代码分析的结构化回答
```

---

## 3. 模块级理解

### 3.1 理解特定模块

**目标**: 深入理解某个功能模块的实现。

```bash
# 1. 搜索团队记忆
mempalace search "用户认证 登录 OAuth"

# 2. 搜索代码图谱
search_graph --label class --name "User"

# 3. 追踪调用链路
trace_path --target "login" --depth 5
```

**推荐问题模板**:

```
问: "用户认证模块是怎么实现的？有哪些关键组件？"

AI 自动调用:
1. mempalace_search "用户认证 登录 OAuth"
   → 查找团队记忆中的认证决策
2. search_graph --label class --name "User"
   → 在代码图谱中搜索 User 相关实体
3. trace_path --target "login" --depth 5
   → 追踪登录相关的调用链路

输出:
- MemPalace: "2024-01 决定使用 JWT + Redis 实现认证"
- Codebase: 类图结构 + 调用关系
```

### 3.2 理解数据模型

```bash
# 搜索实体类
search_graph --label class --name "Order"

# 获取代码片段
get_code_snippet --file "models/Order.java" --line_range "1-100"
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

# 搜索图谱实体
search_graph --label class --name "User"

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
mempalace search "分页 最佳实践"

# 3. 综合结果
输出: 代码位置 + 团队约定的分页模式
```

### 4.3 追踪调用链路

```bash
# 追踪方法调用
trace_path --target "OrderService.checkStatus" --depth 10

# 追踪入口到实现
trace_path --target "UserController" --depth 5
```

---

## 5. 架构分析

### 5.1 整体架构

```bash
# 获取完整架构
get_architecture --scope full

# 获取分层架构
get_architecture --scope layered

# 获取模块依赖
get_architecture --scope dependencies
```

### 5.2 模块架构

```bash
# 分析特定模块
get_architecture --scope module --module "order-service"

# 获取模块类图
search_graph --label class --name "Order*"
```

### 5.3 数据流分析

```bash
# 追踪数据流
trace_path --target "createOrder" --depth 8

# 分析入口到数据库
trace_path --target "save" --depth 10
```

---

## 6. 最佳实践

### 6.1 理解流程

```
┌─────────────────────────────────────────────────────────────┐
│                    项目理解流程                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. MemPalace 团队记忆                                      │
│     ↓                                                        │
│  2. get_architecture 架构概览                                 │
│     ↓                                                        │
│  3. search_code 搜索关键代码                                  │
│     ↓                                                        │
│  4. trace_path 追踪调用链路                                  │
│     ↓                                                        │
│  5. mempalace_add_drawer 补充记忆                           │
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

- **先团队后代码**: 先查 MemPalace 理解团队约定，再查 Codebase 理解实现
- **上下文加载**: 使用 `get_context` 一次性加载多个 Wing 的上下文
- **搜索范围**: 使用 `--scope` 参数限定搜索范围，避免无关结果
- **记录理解**: 发现新信息时及时写入 MemPalace

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
# 架构分析
get_architecture --scope full | head -50

# 搜索验证
search_code --query "main" --scope "**/*.java"

# 调用链验证
trace_path --target "UserService" --depth 3
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
# 扩展搜索范围
mempalace recall "模糊关键词"

# 查看所有 Room
mempalace_list_rooms --wing "project-myapp"
```

---
---
*上一步: [SOP-M1: 工作安装](./ready.md#1-记忆体安装)*  
*下一步: [SOP-M3: 开发调试](./SOP-M3-development.md)*

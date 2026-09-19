# SOP-M3: 开发调试指南

> **版本**: v2.0
> **适用阶段**: 开发流程 M3 - 开发调试
> **目标读者**: 开发者
> **工具约定**: 代码追踪使用 IDE 原生能力
> （LSP 语义导航 / 文本搜索 / git），调试记忆使用 mem0 MCP。

---

## 1. 概述

### 1.1 目标

高效搜索和定位代码，追踪问题和调用链路，检测代码变更影响。

### 1.2 调试模式

```
┌─────────────────────────────────────────────────────────────────┐
│              mem0 记忆 + IDE 原生代码追踪                          │
├───────────────────────────┬─────────────────────────────────────┤
│   mem0 (调试记忆)          │     IDE 原生 (代码追踪)              │
├───────────────────────────┼─────────────────────────────────────┤
│  ✓ 调试发现记录           │  ✓ 代码定位（语义导航/文本搜索）      │
│  ✓ 历史问题经验           │  ✓ 调用链追踪（LSP calls）           │
│  ✓ 解决方案备忘           │  ✓ 变更影响分析（git diff + 引用查找）│
└───────────────────────────┴─────────────────────────────────────┘
```

---

## 2. 日常开发工具使用

### 2.1 快速代码定位

**场景**: 快速找到需要修改的代码。

```bash
# 1. 文本搜索相关代码
search_code --query "payment.*callback" --scope "**/*.java"

# 2. 定位符号并读取片段（LSP）
workspaceSymbol(query="PaymentCallback", kinds=["class"])
readSymbol(filePath="PaymentCallback.java", scope="PaymentCallback")
```

**推荐问题**:

```
问: "支付模块的回调处理在哪里？"

AI 自动调用:
1. search_code --query "payment.*callback" --scope "**/*.java"
   → 搜索支付回调相关代码
2. incomingCalls(filePath="PaymentCallback.java", scope="onCallback")
   → 追踪回调处理链路

结果: 精确定位到 PaymentCallback.java + 调用顺序
```

### 2.2 调用链追踪

**场景**: 理解代码执行路径（LSP 调用层级）。

```bash
# 追踪方法被谁调用（向上）
incomingCalls(filePath="OrderService.java", scope="checkStatus")

# 追踪方法调用了谁（向下）
outgoingCalls(filePath="UserController.java", scope="getUser")
```

**示例**:

```
问: "用户获取的完整调用链是什么？"

AI 自动调用:
outgoingCalls(filePath="UserController.java", scope="getUser")

输出:
UserController.getUser()
  → UserService.getById()
    → UserRepository.findById()
      → UserDao.selectById()
```

### 2.3 代码片段获取

```bash
# 读取符号定义（推荐，按符号粒度）
readSymbol(filePath="src/main/java/UserService.java", scope="UserService.getUser")

# 或直接读取文件行区间
read_file(filePath="src/main/java/UserService.java", offset=50, limit=50)
```

---

## 3. 变更影响分析

### 3.1 提交前分析

**场景**: 提交代码前评估影响范围（git diff + LSP 引用查找组合）。

```bash
# 1. 查看变更
git diff HEAD~1 --stat

# 2. 对每个变更符号做引用查找（findReferences）
findReferences(filePath="UserService.java", scope="getUser")
```

**输出示例**（Agent 汇总）:

```json
{
  "affected_components": [
    "UserController",
    "UserRepository",
    "AuthService"
  ],
  "risk_level": "medium",
  "recommendations": [
    "建议回归测试 AuthService 相关功能",
    "检查 UserRepository 的事务边界"
  ]
}
```

### 3.2 文件影响分析

```bash
# 查找某类的所有使用方
findReferences(filePath="src/main/java/UserService.java", scope="UserService")

# 查找某接口的所有实现
goToImplementation(filePath="UserService.java", scope="UserService")
```

### 3.3 依赖分析

```bash
# 分析 import 依赖
search_code --query "^import com\.example\.service\." --scope "**/*.java"

# 分析包内聚（文件树 + 符号聚合）
documentSymbol("src/main/java/com/example/user/")
```

---

## 4. Bug 追踪与调试

### 4.1 调用链追踪定位

```bash
# 追踪 NPE 发生位置的调用链
incomingCalls(filePath="OrderService.java", scope="checkStatus")

# 追踪异常传播
outgoingCalls(filePath="ExceptionHandler.java", scope="handleException")
```

### 4.2 搜索历史经验

```bash
# 搜索类似问题的解决方案（mem0）
search_memories(query="OrderService NPE 历史", top_k=5)

# 搜索性能问题
search_memories(query="N+1 查询 解决", top_k=5)
```

**示例**:

```
问: "这个 NPE 发生在 OrderService.checkStatus()，调用链是什么？"

AI 自动调用:
1. incomingCalls(filePath="OrderService.java", scope="checkStatus")
   → 获取完整调用链

2. search_memories "OrderService NPE 历史"
   → 查找团队记忆中的类似问题

输出: 调用链 + 历史解决方案
```

### 4.3 调试发现保存

```bash
# 保存调试发现（mem0，type=note）
add_memory(
  text="OrderService.checkStatus() N+1 查询问题: 每次调用执行 5 次额外查询。
        解决方案: 添加 @BatchSize 注解",
  api_key="<from config>", git_remote="<from config>",
  project_id="ai-dev-sop",
  metadata={"type":"note", "people":[], "created_at":"<ISO8601>"}
)

# 会话留痕由 Agent 按 CODEBUDDY.md §2.1 每轮自动提交，无需手动 checkpoint
```

---

## 5. 性能分析

### 5.1 调用耗时分析

```bash
# 定位慢查询调用链（LSP）
incomingCalls(filePath="UserRepository.java", scope="findAll")

# 分析循环调用
outgoingCalls(filePath="OrderService.java", scope="getOrders")
```

### 5.2 数据库查询分析

```bash
# 搜索 SQL 相关代码
search_code --query "@Query|JdbcTemplate" --scope "**/*.java"

# 分析 ORM 使用
search_code --query "findBy|select.*from" --scope "**/*.java"
```

### 5.3 性能问题记录

```bash
add_memory(
  text="性能问题: 位置 OrderService.listOrders()；问题 N+1 查询；
        影响 100 个订单 = 101 次查询；解决 使用 @EntityGraph 或 JOIN FETCH",
  metadata={"type":"note", "people":[], "created_at":"<ISO8601>"}
)
```

---

## 6. 代码重构辅助

### 6.1 重构影响评估

```bash
# 分析方法重命名影响（引用查找）
findReferences(filePath="UserService.java", scope="getUser")

# 分析接口变更影响
search_code --query "implements.*UserService" --scope "**/*.java"
```

### 6.2 依赖关系分析

```bash
# 分析类的使用方
findReferences(filePath="UserService.java", scope="UserService")

# 分析包结构
documentSymbol("src/main/java/com/example/user/")
```

### 6.3 重构验证

```bash
# 追踪重构后的调用
incomingCalls(filePath="UserService.java", scope="newMethodName")

# 验证原有功能（回归测试）
search_code --query "UserService" --scope "**/test/**/*.java"
```

---

## 7. 调试最佳实践

### 7.1 调试流程

```
┌─────────────────────────────────────────────────────────────┐
│                    调试流程                                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. LSP 调用层级追踪调用链                                    │
│     ↓                                                        │
│  2. search_code 定位相关代码                                 │
│     ↓                                                        │
│  3. search_memories 查历史经验（mem0）                        │
│     ↓                                                        │
│  4. 分析问题根因                                            │
│     ↓                                                        │
│  5. add_memory 记录发现（mem0）                              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 推荐问题集

| 场景 | 推荐问题 |
|------|----------|
| 快速定位 | "XX 功能在哪个文件实现的？" |
| 理解流程 | "这个方法被哪些地方调用？" |
| Bug 追踪 | "为什么会报这个错？调用链是什么？" |
| 变更评估 | "改这个类会影响哪些模块？" |
| 性能分析 | "这个查询为什么会慢？" |

### 7.3 注意事项

- **先追踪后搜索**: 先用 LSP 调用层级理解调用链，再用 `search_code` 定位
- **记录发现**: 调试发现及时 `add_memory` 写入 mem0，避免重复踩坑
- **深度限制**: LSP 调用层级默认返回有限页，过深时分段追踪
- **历史借鉴**: 先 `search_memories` 看是否有人遇到过类似问题

---

## 8. 验证清单

### 8.1 功能验证

| 检查项 | 验证方式 |
|--------|----------|
| 代码定位 | 能快速找到指定功能的代码位置 |
| 调用链理解 | 能画出关键方法的调用关系图 |
| 变更影响 | 能列出修改某处代码的影响范围 |

### 8.2 调试验证

```bash
# 调用链验证（LSP）
incomingCalls(filePath="UserService.java", scope="getUser")

# 代码搜索验证
search_code --query "class User" --scope "**/*.java"

# 变更检测验证
git diff HEAD~1 --stat
```

---

## 9. 常见问题

### 9.1 调用链截断

```bash
# LSP 引用有分页，翻页获取
findReferences(..., offset=<nextOffset>)

# 分段追踪
incomingCalls(..., scope="Controller.createOrder")
incomingCalls(..., scope="Service.getById")
```

### 9.2 搜索结果不准确

```bash
# 使用更精确的查询
search_code --query "class UserService" --scope "**/service/*.java"

# 限定文件类型
search_code --query "getUser" --scope "**/*.java"
```

### 9.3 调试记录丢失

```
解决: 会话留痕由 Agent 按 CODEBUDDY.md §2.1 自动提交（每轮 Q/A 入库），
     无需手动 checkpoint；重要发现用 add_memory(type=note) 显式持久化
```

---

*文档更新: 2026-09-18*
*下一步: [SOP-M4: 知识沉淀](./SOP-M4-knowledge.md)*

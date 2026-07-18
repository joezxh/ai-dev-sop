# SOP-M3: 开发调试指南

> **版本**: v1.0
> **适用阶段**: 开发流程 M3 - 开发调试
> **目标读者**: 开发者

---

## 1. 概述

### 1.1 目标

高效搜索和定位代码，追踪问题和调用链路，检测代码变更影响。

### 1.2 双轨调试模式

```
┌─────────────────────────────────────────────────────────────────┐
│                     双轨调试模式                                    │
├───────────────────────────┬─────────────────────────────────────┤
│   轨道 A: MemPalace       │     轨道 B: codebase-mem-mcp       │
│   (调试记忆)              │     (代码追踪)                       │
├───────────────────────────┼─────────────────────────────────────┤
│  ✓ 调试发现记录           │  ✓ 代码定位                          │
│  ✓ 历史问题经验           │  ✓ 调用链追踪                        │
│  ✓ 解决方案备忘           │  ✓ 变更影响分析                       │
└───────────────────────────┴─────────────────────────────────────┘
```

---

## 2. 日常开发工具使用

### 2.1 快速代码定位

**场景**: 快速找到需要修改的代码。

```bash
# 1. 搜索相关代码
search_code --query "payment.*callback" --scope "**/*.java"

# 2. 获取代码片段
get_code_snippet --file "src/main/java/PaymentCallback.java" --line_range "1-50"
```

**推荐问题**:

```
问: "支付模块的回调处理在哪里？"

AI 自动调用:
1. search_code --query "payment.*callback" --scope "**/*.java"
   → 搜索支付回调相关代码
2. trace_path --target "PaymentCallback" --depth 3
   → 追踪回调处理链路

结果: 精确定位到 PaymentCallback.java + 调用顺序
```

### 2.2 调用链追踪

**场景**: 理解代码执行路径。

```bash
# 追踪方法调用
trace_path --target "OrderService.checkStatus" --depth 10

# 追踪入口到实现
trace_path --target "UserController.getUser" --depth 5
```

**示例**:

```
问: "用户获取的完整调用链是什么？"

AI 自动调用:
trace_path --target "UserController.getUser" --depth 8

输出:
UserController.getUser()
  → UserService.getById()
    → UserRepository.findById()
      → UserDao.selectById()
```

### 2.3 代码片段获取

```bash
# 获取指定文件的代码片段
get_code_snippet --file "src/main/java/UserService.java" --line_range "50-100"

# 获取完整文件
get_code_snippet --file "src/main/java/UserService.java"
```

---

## 3. 变更影响分析

### 3.1 提交前分析

**场景**: 提交代码前评估影响范围。

```bash
# 分析 git diff
detect_changes --git_diff "$(git diff HEAD~1)"

# 完整示例
git diff HEAD~1 | detect_changes --stdin
```

**输出示例**:

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
# 分析特定文件的变更影响
detect_changes --file "src/main/java/UserService.java"

# 分析包的变更影响
detect_changes --package "com.example.service"
```

### 3.3 依赖分析

```bash
# 分析类依赖
search_graph --label depends --name "UserService"

# 分析包依赖
get_architecture --scope dependencies --module "user-service"
```

---

## 4. Bug 追踪与调试

### 4.1 调用链追踪定位

```bash
# 追踪 NPE 发生位置
trace_path --target "OrderService.checkStatus" --depth 10

# 追踪异常传播
trace_path --target "handleException" --depth 8
```

### 4.2 搜索历史经验

```bash
# 搜索类似问题的解决方案
mempalace search "OrderService NPE 历史"

# 搜索性能问题
mempalace search "N+1 查询 解决"
```

**示例**:

```
问: "这个 NPE 发生在 OrderService.checkStatus()，调用链是什么？"

AI 自动调用:
1. trace_path --target "OrderService.checkStatus" --depth 10
   → 获取完整调用链

2. mempalace_search "OrderService NPE 历史"
   → 查找团队记忆中的类似问题

输出: 调用链 + 历史解决方案
```

### 4.3 调试会话保存

```bash
# 保存调试发现
mempalace_add_drawer \
  --wing "project-myapp" \
  --room "debug" \
  --hall "discoveries" \
  --content "OrderService.checkStatus() N+1 查询问题:
- 每次调用执行 5 次额外查询
- 解决方案: 添加 @BatchSize 注解"

# 保存检查点
mempalace_checkpoint --session_id "debug-session-001"
```

---

## 5. 性能分析

### 5.1 调用耗时分析

```bash
# 追踪慢查询
trace_path --target "UserRepository.findAll" --depth 5

# 分析循环调用
trace_path --target "getOrders" --depth 10
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
# 记录性能发现
mempalace_add_drawer \
  --wing "project-myapp" \
  --room "performance" \
  --hall "discoveries" \
  --content "性能问题:
- 位置: OrderService.listOrders()
- 问题: N+1 查询
- 影响: 100个订单 = 101次查询
- 解决: 使用 @EntityGraph 或 JOIN FETCH"
```

---

## 6. 代码重构辅助

### 6.1 重构影响评估

```bash
# 分析方法重命名影响
detect_changes --git_diff "M src/UserService.java"

# 分析接口变更影响
search_code --query "implements.*UserService" --scope "**/*.java"
```

### 6.2 依赖关系分析

```bash
# 分析类的使用方
search_graph --label uses --name "UserService"

# 分析包内聚度
get_architecture --scope module --module "user-service"
```

### 6.3 重构验证

```bash
# 追踪重构后的调用
trace_path --target "newMethodName" --depth 5

# 验证原有功能
detect_changes --verify "src/test/java/*Test.java"
```

---

## 7. 调试最佳实践

### 7.1 调试流程

```
┌─────────────────────────────────────────────────────────────┐
│                    调试流程                                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. trace_path 追踪调用链                                   │
│     ↓                                                        │
│  2. search_code 定位相关代码                                 │
│     ↓                                                        │
│  3. mempalace_search 历史经验                                │
│     ↓                                                        │
│  4. 分析问题根因                                            │
│     ↓                                                        │
│  5. mempalace_add_drawer 记录发现                          │
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

- **先追踪后搜索**: 使用 `trace_path` 理解调用链，再用 `search_code` 定位
- **记录发现**: 调试发现及时写入 MemPalace，避免重复踩坑
- **深度限制**: `trace_path` 有深度限制，过深会截断
- **历史借鉴**: 搜索 MemPalace 看是否有人遇到过类似问题

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
# 调用链验证
trace_path --target "UserService.getUser" --depth 5

# 代码搜索验证
search_code --query "class User" --scope "**/*.java"

# 变更检测验证
detect_changes --file "src/main/java/User.java"
```

---

## 9. 常见问题

### 9.1 调用链截断

```bash
# 限制深度
trace_path --target "UserService" --depth 5

# 分段追踪
trace_path --target "Controller" --depth 3
trace_path --target "Service" --depth 3
```

### 9.2 搜索结果不准确

```bash
# 使用更精确的查询
search_code --query "class UserService" --scope "**/service/*.java"

# 限定文件类型
search_code --query "getUser" --scope "**/*.java"
```

### 9.3 调试记录丢失

```bash
# 定期保存
mempalace_checkpoint --session_id "debug-$(date +%Y%m%d)"

# 记录重要发现
mempalace_add_drawer --wing "debug-sessions" --room "session-001" \
  --hall "discoveries" --content "..."
```

---

*文档更新: 2026-07-14*
*下一步: [SOP-M4: 知识沉淀](./SOP-M4-knowledge.md)*

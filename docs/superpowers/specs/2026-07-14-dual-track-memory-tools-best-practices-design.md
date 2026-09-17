> [!WARNING]
> **已废弃（2026-09-18）**：本文档描述的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，记忆功能统一替换为自托管 mem0（见 docs/quick-ref/mem0-ai-tools-config-guide.md）。本文仅作历史归档保留，内容不再维护。

# 双轨记忆系统工具使用最佳实践设计文档

> **版本**: v1.0
> **创建日期**: 2026-07-14
> **设计目标**: 基于 develop-sop.md 开发流程，按阶段整理 codebase-mem-mcp 与 MemPalace 的工具使用最佳实践
> **核心理念**: MemPalace 解决"人/组织/决策记忆"，codebase-mem-mcp 解决"机器/代码/架构记忆"

---

## 1. 设计概述

### 1.1 双轨记忆系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      双轨记忆系统                                  │
├───────────────────────────┬─────────────────────────────────────┤
│    轨道 A: MemPalace      │     轨道 B: codebase-mem-mcp        │
│    "人的记忆"             │     "机器的记忆"                    │
├───────────────────────────┼─────────────────────────────────────┤
│  ✓ 对话记录              │  ✓ 代码 AST 索引                   │
│  ✓ 技术决策              │  ✓ 函数调用链路                    │
│  ✓ 团队知识              │  ✓ 项目结构理解                    │
│  ✓ 偏好习惯              │  ✓ 架构分析                       │
│  ✓ 会议纪要              │  ✓ 变更检测                        │
│  ✓ 建议方案              │  ✓ ADR 管理                        │
└───────────────────────────┴─────────────────────────────────────┘
```

### 1.2 工具分类总览

| 分类 | 工具数量 | 轨道 | 优先级 |
|------|---------|------|--------|
| MemPalace 读取工具 | 10 | A | P0-P1 |
| MemPalace 写入工具 | 6 | A | P2-P3 |
| MemPalace 管理工具 | 19 | A | P4-P6 |
| Codebase 索引工具 | 4 | B | P0-P1 |
| Codebase 查询工具 | 6 | B | P0-P1 |
| Codebase 分析工具 | 4 | B | P0-P1 |
| 双轨联合工具 | 6 | J | P0-P1 |

---

## 2. 开发流程阶段与工具映射

```
┌─────────────────────────────────────────────────────────────────┐
│                     开发流程七大阶段                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. 安装配置 ───────────────────────────────────────────→ M1    │
│      │                                                        │
│      ▼                                                        │
│  2. 项目理解 ───────────────────────────────────────────→ M2    │
│      │                                                        │
│      ▼                                                        │
│  3. 开发调试 ───────────────────────────────────────────→ M3    │
│      │                                                        │
│      ▼                                                        │
│  4. 知识沉淀 ───────────────────────────────────────────→ M4    │
│      │                                                        │
│      ▼                                                        │
│  5. 团队协作 ───────────────────────────────────────────→ M5    │
│      │                                                        │
│      ▼                                                        │
│  6. 文档自动化                                               │
│  7. 运营回流                                                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 各阶段最佳实践设计

### 3.1 M1: 安装配置阶段

#### 3.1.1 阶段目标
- 完成 MemPalace 和 codebase-mem-mcp 的安装配置
- 配置 AI IDE MCP 连接
- 验证连接和基础功能

#### 3.1.2 工具使用清单

**MemPalace 安装配置:**

| 工具 | 优先级 | 说明 |
|------|--------|------|
| `mempalace init` | P0 | 初始化 Palace |
| `mempalace_status` | P0 | 验证安装状态 |
| `mempalace_create_wing` | P1 | 创建项目 Wing |

**Codebase 安装配置:**

| 工具 | 优先级 | 说明 |
|------|--------|------|
| `codebase-memory-mcp` | P0 | 安装 MCP 二进制 |
| `index_repository` | P1 | 创建初始索引 |
| `index_status` | P1 | 验证索引状态 |

#### 3.1.3 最佳实践示例

**实践 1: MemPalace 初始化**

```bash
# 1. 安装 MemPalace (使用 uv)
uv tool install mempalace

# 2. 初始化项目 Palace
mempalace init ~/projects/myapp

# 3. 验证安装
mempalace_status

# 预期输出:
# {
#   "palace_id": "...",
#   "wings": [...],
#   "drawers": 0
# }

# 4. 创建项目 Wing
mempalace_create_wing --wing "project-myapp" --description "My App 项目"

# 5. 创建主题 Room
mempalace_create_room --wing "project-myapp" --room "architecture" --description "架构决策"
```

**实践 2: Codebase 索引创建**

```bash
# 1. 索引项目代码
codebase-memory-mcp index_repository --repo-path /path/to/project --mode incremental

# 2. 验证索引状态
codebase-memory-mcp index_status --project-id <id>

# 预期输出:
# {
#   "indexed": true,
#   "file_count": 1234,
#   "last_update": "2026-07-14T..."
# }
```

**实践 3: AI IDE MCP 配置 (Cursor)**

```json
// ~/.cursor/mcp.json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://server:8787/mcp?as=alice&project=${workspaceFolder}",
      "headers": {
        "Authorization": "Bearer <jwt-token>"
      }
    },
    "mempalace": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "mempalace-data:/data", "mempalace"]
    }
  }
}
```

#### 3.1.4 注意事项
- JWT Token 需要从管理员处获取
- `project` 参数是服务器端路径，非本地路径
- MemPalace 数据存储在 Docker volume 中，注意备份

---

### 3.2 M2: 项目理解阶段

#### 3.2.1 阶段目标
- 理解项目结构和代码组织
- 加载相关上下文记忆
- 识别关键架构决策

#### 3.2.2 工具使用清单

**MemPalace 读取:**  

| 工具 | 优先级 | 场景 |
|------|--------|------|
| `mempalace_search` | P0 | 搜索架构决策 |
| `mempalace_list_rooms` | P1 | 查看项目主题 |
| `mempalace_get_context` | P1 | 加载上下文束 |
| `mempalace_recall` | P2 | 模糊召回 |

**Codebase 查询:**

| 工具 | 优先级 | 场景 |
|------|--------|------|
| `search_code` | P0 | 搜索代码片段 |
| `search_graph` | P0 | 搜索图谱实体 |
| `get_graph_schema` | P1 | 获取图谱结构 |
| `get_code_snippet` | P1 | 获取代码片段 |

**Codebase 分析:**

| 工具 | 优先级 | 场景 |
|------|--------|------|
| `get_architecture` | P0 | 获取架构概览 |
| `trace_path` | P1 | 追踪调用链路 |

#### 3.2.3 最佳实践示例

**实践 1: 理解项目架构**

```
# AI IDE 中使用自然语言查询

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

**实践 2: 理解特定模块**

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

**实践 3: 查找相关实现**

```
问: "类似的分页功能在哪里有实现？"

AI 自动调用:
1. search_code --query "pagination" --scope "**/*.java"
   → 在代码中搜索分页实现
2. mempalace_search "分页 最佳实践"
   → 查找团队记忆中的分页规范

输出: 代码位置 + 团队约定的分页模式
```

#### 3.2.4 注意事项
- 先查 MemPalace 获取团队约定，再查 Codebase 获取实现
- `get_context` 可一次性加载多个 Wing 的上下文
- `search_code` 支持正则表达式和文件范围限定

---

### 3.3 M3: 开发调试阶段

#### 3.3.1 阶段目标
- 高效搜索和定位代码
- 追踪问题和调用链路
- 检测代码变更影响

#### 3.3.2 工具使用清单

**Codebase 查询:**

| 工具 | 优先级 | 场景 |
|------|--------|------|
| `search_code` | P0 | 搜索代码 |
| `trace_path` | P0 | 追踪调用 |
| `query_graph` | P1 | Cypher 查询 |
| `get_code_snippet` | P1 | 获取代码 |

**Codebase 分析:**

| 工具 | 优先级 | 场景 |
|------|--------|------|
| `detect_changes` | P0 | 变更检测 |
| `ingest_traces` | P2 | 导入追踪数据 |

**MemPalace 写入:**

| 工具 | 优先级 | 场景 |
|------|--------|------|
| `mempalace_add_drawer` | P1 | 记录调试发现 |
| `mempalace_checkpoint` | P2 | 保存调试会话 |

#### 3.3.3 最佳实践示例

**实践 1: 快速定位代码**

```
问: "支付模块的回调处理在哪里？"

AI 自动调用:
1. search_code --query "payment.*callback" --scope "**/*.java"
   → 搜索支付回调相关代码
2. trace_path --target "PaymentCallback" --depth 3
   → 追踪回调处理链路

结果: 精确定位到 PaymentCallback.java + 调用顺序
```

**实践 2: 分析变更影响**

```
# 在提交前询问
问: "我修改了 UserService.java，这会影响哪些模块？"

AI 自动调用:
1. detect_changes --git_diff <diff内容>
   → 分析代码变更

输出:
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

**实践 3: 追踪 bug 根因**

```
问: "这个 NPE 发生在 OrderService.checkStatus()，调用链是什么？"

AI 自动调用:
1. trace_path --target "OrderService.checkStatus" --depth 10
   → 获取完整调用链

2. mempalace_search "OrderService NPE 历史"
   → 查找团队记忆中的类似问题

输出: 调用链 + 历史解决方案
```

**实践 4: 调试过程中保存发现**

```
# AI 在调试过程中自动保存
AI: 发现了一个性能问题，让我记录下来...

自动调用:
mempalace_add_drawer --wing "project-myapp" --room "performance" --hall "discoveries"
  --content "OrderService.checkStatus() 中存在 N+1 查询问题，
           每次调用都执行 5 次额外数据库查询，
           建议添加 @BatchSize 注解优化"
```

#### 3.3.4 注意事项
- `detect_changes` 需要 git diff 作为输入
- 调试发现应及时写入 MemPalace，避免遗忘
- `trace_path` 有深度限制，过深会截断

---

### 3.4 M4: 知识沉淀阶段

#### 3.4.1 阶段目标
- 将开发过程中的知识结构化沉淀
- 使用归纳/蒸馏提取关键信息
- 写入 MemPalace 供团队复用

#### 3.4.2 工具使用清单

**会话归纳 (cbmem-team Console):**
| 工具 | 优先级 | 场景 |
|------|--------|------|
| `/api/console/summarize` | P0 | 5 Hall 归纳 |
| `/api/console/distill` | P0 | 知识蒸馏 |

**MemPalace 写入:**

| 工具 | 优先级 | 场景 |
|------|--------|------|
| `mempalace_add_drawer` | P0 | 添加记忆抽屉 |
| `mempalace_checkpoint` | P1 | 批量保存 |
| `mempalace_tag_drawer` | P2 | 标签分类 |

**Hall 分类:**

| Hall | 说明 | 内容类型 |
|------|------|---------|
| hall_facts | 事实 | 技术决策、架构约束 |
| hall_events | 事件 | 会议、里程碑、调试 |
| hall_discoveries | 发现 | 性能问题、Bug 根因 |
| hall_preferences | 偏好 | 代码风格、工具选择 |
| hall_advice | 建议 | 最佳实践、避坑指南 |

#### 3.4.3 最佳实践示例

**实践 1: 会话归纳 (Console)**

```
场景: 完成了模块设计评审，需要将讨论结果沉淀到团队记忆

步骤:
1. 在 cbmem-team Console 中选择会话
   → 路径: /console/#/summarize

2. 选择归纳深度: deep
   → shallow: 快速概览
   → deep: 详细分析 (推荐)
   → expert: 专家级

3. 设置目标 Wing: "project-myapp"

4. 点击 "Start Summarization"

5. 归纳结果 (5 Hall):

hall_facts:
- 决定使用 DDD 架构模式
- 用户模块采用 CQRS 读写分离
- 订单模块使用 Saga 模式处理分布式事务

hall_events:
- 2024-07-10 架构评审会议纪要
- 确认技术选型方向

hall_discoveries:
- 当前单体架构难以支持高并发
- 需要提前规划微服务拆分

hall_preferences:
- 团队倾向于使用注解式事务
- 偏好 Lombok 简化代码

hall_advice:
- 新模块建议直接从 DDD Template 创建
- 跨模块调用优先使用事件驱动

6. 点击 "Write to MemPalace" 写入团队记忆
```

**实践 2: 知识蒸馏 (Console)**

```
场景: 发现了性能问题解决方案，需要提炼为可复用知识

步骤:
1. 在 cbmem-team Console 中选择会话
   → 路径: /console/#/distill

2. 设置蒸馏规则:
   - min_value_score: 0.7 (评分阈值)
   - dimensions: ["fact", "decision", "discovery"]

3. 点击 "Start Distillation"

4. 蒸馏结果:

knowledge_fragments:
{
  "content": "使用 @EntityGraph 注解可以解决 N+1 查询问题，
              配合 @QueryHints 实现加载超时控制",
  "source": "OrderService 优化过程",
  "score": 0.95
}

decisions:
{
  "title": "选择 EntityGraph 而非 DTO 投影",
  "decision": "EntityGraph 更灵活，可以动态控制加载深度，
             适合复杂聚合根场景",
  "context": "用户模块订单查询优化"
}

tech_debt:
{
  "title": "历史代码未使用批量操作",
  "description": "早期实现中存在 for 循环内单条查询",
  "impact": "中等",
  "suggestion": "建议重构为 MyBatis-Plus batch 操作"
}

5. 点击 "Write to MemPalace" 提交
```

**实践 3: 手动添加关键记忆**

```bash
# 添加架构决策
mempalace_add_drawer \
  --wing "project-myapp" \
  --room "architecture" \
  --hall "facts" \
  --content "ADR-001: 选择 PostgreSQL 作为主数据库
原因:
1. 支持 JSONB 类型，适合灵活 schema
2. pgvector 支持向量存储
3. 团队熟悉度高

决策日期: 2024-01-15
决策者: 技术委员会"

# 添加最佳实践
mempalace_add_drawer \
  --wing "project-myapp" \
  --room "best-practices" \
  --hall "advice" \
  --content "API 错误处理规范:
1. 使用统一的 ErrorResponse 结构
2. HTTP 状态码遵循语义 (4xx 客户端错误, 5xx 服务端错误)
3. 错误消息对用户友好，内部日志记录详细堆栈
4. 敏感信息脱敏处理"
```

#### 3.4.4 注意事项
- 归纳/蒸馏依赖 LLM，建议使用 deep 深度
- 蒸馏结果的评分阈值可调整 (0.7 为默认值)
- 提交到 MemPalace 是幂等的，重复提交不会产生重复

---

### 3.5 M5: 团队协作阶段

#### 3.5.1 阶段目标
- 跨项目共享知识和经验
- 团队成员间高效协作
- 知识资产的持续积累

#### 3.5.2 工具使用清单

**团队共享:**

| 工具 | 优先级 | 场景 |
|------|--------|------|
| `mempalace_create_wing` | P1 | 创建团队 Wing |
| `mempalace_grant_access` | P2 | 授权访问 |
| `mempalace_export` | P3 | 导出知识 |

**跨 Wing 检索:**

| 工具 | 优先级 | 场景 |
|------|--------|------|
| `mempalace_search` | P0 | 跨团队搜索 |
| `mempalace_get_context` | P1 | 加载多 Wing 上下文 |

**Codebase 团队功能:**

| 工具 | 优先级 | 场景 |
|------|--------|------|
| `manage_adr` | P1 | ADR 管理 |
| `list_projects` | P2 | 团队项目列表 |

#### 3.5.3 最佳实践示例

**实践 1: 团队 Wing 组织**

```bash
# 创建团队共享 Wing
mempalace_create_wing \
  --wing "team-backend" \
  --description "后端团队共享知识库"

# 创建子 Room
mempalace_create_room --wing "team-backend" --room "api-standards" \
  --description "API 设计规范与标准"
mempalace_create_room --wing "team-backend" --room "database-guidelines" \
  --description "数据库设计规范"
mempalace_create_room --wing "team-backend" --room "incident-reports" \
  --description "事故报告与复盘"

# 授权团队成员
mempalace_grant_access --wing "team-backend" --user "alice"
mempalace_grant_access --wing "team-backend" --user "bob"
```

**实践 2: 跨项目知识复用**

```
问: "其他项目有没有做过类似的文件上传功能？用了什么方案？"

AI 自动调用:
1. mempalace_search --wing "team-backend" "文件上传 方案"
   → 在团队共享 Wing 中搜索
2. mempalace_get_context --wing-ids ["project-payment", "project-oss"]
   → 加载多个相关项目的上下文

输出:
{
  "from_wing": "project-payment",
  "hall": "decisions",
  "content": "使用 MinIO 作为对象存储，封装 FileService 统一管理..."
}
```

**实践 3: ADR 跨项目共享**

```bash
# 记录架构决策
manage_adr --op create --adr '{
  "id": "ADR-042",
  "title": "统一使用阿里云 OSS",
  "status": "accepted",
  "context": "需要统一文件存储方案，多项目分散使用带来管理成本",
  "decision": "采用阿里云 OSS，封装 FileService 统一接口",
  "consequences": [
    "增加云服务成本",
    "需要统一配置管理"
  ]
}'

# 查询团队 ADR
manage_adr --op search --query "文件存储"
```

**实践 4: 团队知识贡献流程**

```bash
# 1. 完成开发后，贡献有价值的技术发现
mempalace_add_drawer \
  --wing "team-backend" \
  --room "lessons-learned" \
  --hall "discoveries" \
  --content "支付模块优化经验:
- 使用 Redis 分布式锁解决并发重复支付
- 幂等键设计: orderId + timestamp
- 补偿机制: 定时任务扫描超时订单"
  --tags ["支付", "性能优化", "Redis"]

# 2. 创建跨项目隧道连接
mempalace_create_tunnel \
  --source-wing "project-payment" \
  --source-room "implementation" \
  --target-wing "team-backend" \
  --target-room "lessons-learned" \
  --label "支付模块经验共享"

# 3. 导出知识给新成员
mempalace_export --wing "team-backend" --format markdown --output onboarding.md
```

#### 3.5.4 注意事项
- 团队 Wing 命名建议: `team-{dept}` 或 `team-{platform}`
- Tunnel 连接跨 Wing 关联，便于发现相关知识
- 定期导出知识作为新人 onboarding 材料

---

## 4. 工具速查表

### 4.1 MemPalace 工具 (轨道 A)

#### 读取工具 (P0-P2)

| 工具 | 参数 | 说明 | 优先级 |
|------|------|------|--------|
| `mempalace_status` | - | 系统状态 | P0 |
| `mempalace_list_wings` | - | 列出 Wing | P0 |
| `mempalace_search` | query | 语义搜索 | P0 |
| `mempalace_list_rooms` | wing_id | 列出 Room | P1 |
| `mempalace_recall` | query | 模糊召回 | P2 |
| `mempalace_wing_info` | wing_id | Wing 详情 | P2 |
| `mempalace_list_halls` | - | 列出 Hall | P2 |
| `mempalace_list_drawers` | hall_id | 列出抽屉 | P2 |
| `mempalace_view_drawer` | drawer_id | 查看抽屉 | P0 |
| `mempalace_get_context` | wing_ids[] | 加载上下文 | P1 |

#### 写入工具 (P2-P3)

| 工具 | 参数 | 说明 | 优先级 |
|------|------|------|--------|
| `mempalace_add_drawer` | drawer{original,context} | 添加抽屉 | P2 |
| `mempalace_checkpoint` | session_id | 批量保存 | P3 |
| `mempalace_mine` | path,wing_id | 挖掘项目 | P3 |
| `mempalace_update_drawer` | drawer_id,body | 更新抽屉 | P3 |
| `mempalace_delete_drawer` | drawer_id | 删除抽屉 | P4 |
| `mempalace_tag_drawer` | drawer_id,tags[] | 标签抽屉 | P3 |

#### 管理工具 (P4-P6)

| 工具 | 参数 | 说明 | 优先级 |
|------|------|------|--------|
| `mempalace_create_wing` | wing_name | 创建 Wing | P3 |
| `mempalace_create_room` | wing_id,room_name | 创建 Room | P4 |
| `mempalace_request_access` | wing_id | 请求访问 | P5 |
| `mempalace_grant_access` | wing_id,user_id | 授权访问 | P4 |
| `mempalace_revoke_access` | wing_id,user_id | 撤销访问 | P5 |
| `mempalace_config` | key,value | 配置管理 | P5 |
| `mempalace_stats` | scope | 统计信息 | P5 |
| `mempalace_export` | wing_id,format | 导出数据 | P5 |
| `mempalace_merge_drawer` | src_id,dst_id | 合并抽屉 | P6 |
| `mempalace_archive_wing` | wing_id | 归档 Wing | P6 |
| `mempalace_restore_wing` | wing_id | 恢复 Wing | P6 |
| `mempalace_reindex_search` | wing_id | 重建索引 | P6 |
| `mempalace_quota_set` | wing_id,bytes | 设置配额 | P6 |
| `mempalace_audit_log` | filter | 审计日志 | P6 |
| `mempalace_backup` | wing_id | 备份 | P7 |
| `mempalace_restore` | backup_id | 恢复 | P7 |

### 4.2 Codebase 工具 (轨道 B)

#### 索引工具 (P0-P1)

| 工具 | 参数 | 说明 | 优先级 |
|------|------|------|--------|
| `index_repository` | repo_path,mode | 索引仓库 | P0 |
| `list_projects` | - | 列出项目 | P2 |
| `index_status` | project_id | 索引状态 | P1 |
| `delete_project` | project_id | 删除项目 | P3 |

#### 查询工具 (P0-P2)

| 工具 | 参数 | 说明 | 优先级 |
|------|------|------|--------|
| `search_graph` | label,name | 搜索图谱 | P0 |
| `trace_path` | target,depth | 追踪路径 | P0 |
| `search_code` | query,scope | 代码搜索 | P0 |
| `query_graph` | cypher,params | Cypher 查询 | P1 |
| `get_code_snippet` | file,line_range | 获取片段 | P1 |
| `get_graph_schema` | - | 获取 Schema | P2 |

#### 分析工具 (P0-P2)

| 工具 | 参数 | 说明 | 优先级 |
|------|------|------|--------|
| `detect_changes` | git_diff | 变更检测 | P0 |
| `get_architecture` | scope | 架构分析 | P0 |
| `manage_adr` | op,adr | ADR 管理 | P1 |
| `ingest_traces` | trace_bundle | 导入追踪 | P2 |

---

## 5. 双轨联合使用模式

### 5.1 模式一: 理解 → 开发

```
1. mempalace_search "相关技术决策"
   ↓
2. get_architecture
   ↓
3. search_code / trace_path
   ↓
4. 开发实现
   ↓
5. detect_changes 分析影响
   ↓
6. mempalace_add_drawer 沉淀发现
```

### 5.2 模式二: 调试 → 知识

```
1. trace_path 定位问题
   ↓
2. 调试分析
   ↓
3. mempalace_checkpoint 保存会话
   ↓
4. distill 提炼关键发现
   ↓
5. mempalace_add_drawer 写入团队知识库
```

### 5.3 模式三: 评审 → 归档

```
1. 架构评审会议
   ↓
2. summarize 归纳会议要点
   ↓
3. manage_adr 记录架构决策
   ↓
4. mempalace_add_drawer 写入 hall_events
   ↓
5. mempalace_create_tunnel 关联相关项目
```

---

## 6. 附录

### 6.1 Hall 分类指南

| Hall | 使用场景 | 示例内容 |
|------|---------|---------|
| hall_facts | 事实性知识 | "我们使用 PostgreSQL 存储用户数据" |
| hall_events | 时间线事件 | "2024-07-10 完成了支付模块重构" |
| hall_discoveries | 技术发现 | "发现 N+1 查询问题，使用 EntityGraph 解决" |
| hall_preferences | 团队偏好 | "团队偏好使用 Lombok 简化 POJO" |
| hall_advice | 建议指导 | "新模块建议使用 DDD 架构" |

### 6.2 Wing/Room 命名规范

| 层级 | 命名规范 | 示例 |
|------|---------|------|
| 个人 Wing | `wing_{name}` | `wing_alice` |
| 项目 Wing | `project_{name}` | `project-payment` |
| 团队 Wing | `team_{dept}` | `team-backend` |
| Room | `{主题}` | `architecture`, `api-standards` |

### 6.3 优先级定义

| 优先级 | 定义 | 使用建议 |
|--------|------|---------|
| P0 | 核心必用 | 日常开发每天使用 |
| P1 | 高频使用 | 每周多次使用 |
| P2 | 常规使用 | 每周使用 |
| P3 | 偶尔使用 | 每月使用 |
| P4-P7 | 管理功能 | 按需使用 |

---

**文档状态**: 设计完成，待评审
**下一步**: 实施执行计划编写

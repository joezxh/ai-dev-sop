# SOP-M5: 团队协作指南

> **版本**: v2.0
> **适用阶段**: 开发流程 M5 - 团队协作
> **目标读者**: 开发者、团队负责人
> **协作方式**: 团队协作统一基于 mem0：项目共享池（`project_id`/`git_remote`）、
> `metadata` 分类、Dashboard 管理、组织成员权限。

---

## 1. 概述

### 1.1 目标

跨项目共享知识和经验，团队成员间高效协作，知识资产的持续积累。

### 1.2 团队协作架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    mem0 团队协作架构                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  成员 A (CodeBuddy) ──┐                                          │
│  成员 B (Cursor)   ───┼──► mem0 项目共享池                        │
│  成员 C (Qoder)    ───┘    (project_id = f(git_remote))          │
│                              │                                   │
│              metadata.type 分类归档（fact/decision/...）          │
│                              │                                   │
│                    ┌─────────▼──────────┐                        │
│                    │ mem0 Dashboard      │                       │
│                    │ 管理/检索/事件通知   │                        │
│                    └────────────────────┘                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 团队记忆组织

### 2.1 作用域模型

| 作用域 | 实现 | 用途 | 可见性 |
|--------|------|------|--------|
| 个人记忆 | `user_id`（个人 Key 写入，不带 project_id） | 个人记忆 | 仅本人 |
| 项目共享池 | `project_id`（`git_remote` 解析，见 CODEBUDDY.md） | 项目共享 | 项目成员（admin key 跨用户读取） |
| 团队/组织 | 同一组织下多个 Project | 团队共享 | 组织成员 |

### 2.2 分类体系（metadata.type + topic）

| type | 用途 | 示例 |
|------|------|------|
| `fact` | 架构决策、技术约束 | "主数据库为 PostgreSQL" |
| `decision` | ADR、方案取舍 | "ADR-042: 统一使用 OSS" |
| `preference` | 代码风格、工具选择 | "偏好构造器注入" |
| `note` | 经验教训、最佳实践、事故复盘 | "N+1 问题与解决方案" |
| `conversation` | 会话留痕 | 每轮 Q/A 原文 |

主题维度写入 `metadata.topic`（如 `["支付","Redis"]`），检索靠语义搜索天然覆盖。

### 2.3 建议的团队知识结构

```
项目共享池 (project_id="ai-dev-sop")
├── type=decision          # 架构决策 / ADR
├── type=fact              # 技术事实
├── type=note (topic=...)  # 经验教训 / 最佳实践 / 事故复盘
├── type=preference        # 团队约定
└── type=conversation      # 会话留痕（按 session_id 分组）
```

---

## 3. 权限管理

### 3.1 成员与角色（mem0 Settings → Members）

```text
1. Dashboard → Settings → ORGANIZATION → Members
2. Invite Member: 填写邮箱、选择作用域（Organization / Project）、选择角色
```

### 3.2 权限级别

| 角色 | 权限 | 说明 |
|------|------|------|
| Can Read | 只读 | 标准 API 请求 + 读基础数据 |
| Can Edit | 读写 | 增删改记忆、管理实体 |
| Admin | 全部 | 含计费、成员、项目设置 |

### 3.3 凭证分发

```text
1. 每位成员在 Dashboard 创建自己的 API Key（绑定对应 Project）
2. 写入本地 git-ignored 配置（CodeBuddy 为 .codebuddy/mem0.config.json）
3. 成员加入 roster：在 mem0.config.json 的 roster[] 登记姓名与 user_id
4. 项目共享池由 admin key 统一拉取（get_memories(project_id=...) 跨用户）
```

---

## 4. 跨项目知识共享

### 4.1 多项目隔离与检索

mem0 按 Project 隔离记忆；跨项目检索 = 分别查询各 Project（各持对应 Key）：

```bash
# 项目 A 的共享池
get_memories(api_key="<keyA>", project_id="project-a")

# 项目 B 的共享池
get_memories(api_key="<keyB>", project_id="project-b")
```

### 4.2 知识关联

旧 Tunnel（跨 Wing 连接）由 mem0 的语义检索替代：相关主题的记忆
（即使写入自不同会话/成员）在语义空间中天然相邻，`search_memories` 一次召回。

### 4.3 团队事件通知（Webhooks）

```text
Dashboard → Webhooks → Add New Webhook
  事件: Add Memory / Update Memory / Delete Memory / Categorize Memory
  用途: 新知识入库时通知团队 IM（Slack/钉钉等）
```

---

## 5. ADR 管理

### 5.1 创建 ADR（写入共享池）

```bash
add_memory(
  text="ADR-042: 统一使用阿里云 OSS。状态: accepted。
        背景: 需要统一文件存储方案。决策: 采用阿里云 OSS，封装 FileService 统一接口，
        使用 STS 令牌实现临时凭证。影响: 增加云服务成本，需要统一配置管理。",
  api_key="<from config>", git_remote="<from config>",
  project_id="ai-dev-sop",
  metadata={"type":"decision", "topic":["ADR","存储"], "created_at":"<ISO8601>"}
)
```

### 5.2 查询 ADR

```bash
# 语义搜索
search_memories(query="文件存储 ADR", top_k=5)

# 全量浏览（按 created_at 排序）
get_memories(api_key=..., project_id="ai-dev-sop")
```

### 5.3 更新/废弃 ADR

```bash
# 1. 找到旧 ADR
search_memories(query="ADR-023")

# 2a. 更新内容
update_memory(memory_id="xxx", text="ADR-023（已废弃）: ... 新决策见 ADR-042")

# 2b. 或写入新 ADR 记忆，mem0 的 Dream(Supersede) 会自动淘汰过时事实
```

> 文件形态的 ADR（`docs/**/ADR-*.md`）仍以文件为源；mem0 中的 ADR 记忆是其
> 可检索摘要，两者通过 ADR 编号对应。

---

## 6. 团队知识贡献

### 6.1 贡献流程

```
┌─────────────────────────────────────────────────────────────┐
│                    知识贡献流程                                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 完成开发或调试                                          │
│     ↓                                                        │
│  2. 提取有价值的知识（durable 事实/决策/经验）                 │
│     ↓                                                        │
│  3. 选择 metadata.type 与 topic                              │
│     ↓                                                        │
│  4. add_memory 写入项目共享池                                 │
│     ↓                                                        │
│  5. search_memories 验证（可选）                              │
│     ↓                                                        │
│  6. Webhook 自动通知团队（如已配置）                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 贡献类型

| 类型 | metadata.type | 示例 |
|------|---------------|------|
| 技术决策 | `decision` | ADR 记录 |
| 最佳实践 | `note` (topic=best-practices) | 代码规范 |
| 经验教训 | `note` | Bug 分析 |
| 工具配置 | `preference` | IDE 配置 |
| 事故报告 | `note` (topic=incident) | 复盘记录 |

### 6.3 贡献示例

```bash
# 贡献架构决策
add_memory(
  text="ADR-042: 统一使用阿里云 OSS。状态 accepted，日期 2024-07-10。
        决策: 采用 OSS 统一文件存储；封装 FileService 统一接口；STS 临时凭证。
        影响: 需要统一配置管理；考虑多账号隔离。",
  metadata={"type":"decision", "topic":["ADR","存储"], "created_at":"<ISO8601>"}
)

# 贡献最佳实践
add_memory(
  text="Spring Boot 最佳实践: 1. 构造器注入代替 @Autowired; 2. @ConfigurationProperties
        管理配置; 3. @Validated 参数校验; 4. @ControllerAdvice 统一异常;
        5. 日志用占位符而非字符串拼接。",
  metadata={"type":"note", "topic":["best-practices","spring"], "created_at":"<ISO8601>"}
)
```

---

## 7. 新人 Onboarding

### 7.1 Onboarding 知识结构

```
项目共享池 (metadata.topic="onboarding")
├── getting-started      # 入门指南（环境搭建/开发流程）
├── tools-setup          # 工具配置（IDE/Git/CI/CD）
└── coding-standards     # 代码规范（Java/SQL/API）
```

### 7.2 Onboarding 知识导出

```bash
# Agent 批量检索后整理为 Markdown 手册
get_memories(api_key=..., project_id="ai-dev-sop", limit=100)
# → Agent 过滤 topic=onboarding 的条目 → 生成 onboarding-guide.md
```

### 7.3 新人接收流程

```
1. 按 SOP-M1 完成 mem0 环境接入
   ↓
2. 首次会话自动拉取项目共享池（CODEBUDDY.md 规则）
   ↓
3. 向 Agent 提问 onboarding 相关问题（search_memories 自动召回）
   ↓
4. 完成第一个任务
   ↓
5. 开始贡献团队知识（add_memory）
```

---

## 8. 团队协作最佳实践

### 8.1 记忆维护

- **定期清理**: Dashboard → Memories 审查过时记忆，`delete_memory` 或由 Dream 自动淘汰
- **类型规范**: 确保 `metadata.type` 准确，便于过滤统计
- **监控使用**: Dashboard → Dashboard 页查看存储/召回/利用率指标

### 8.2 知识质量

- **具体性**: 提供足够的上下文和示例
- **可操作性**: 他人看后能据此行动
- **定期更新**: 过时知识用 `update_memory` 更新或写入新记忆（Supersede 自动替代）

### 8.3 协作文化

- **鼓励贡献**: 认可知识贡献者（metadata.by 记录提交者）
- **分享文化**: 有价值的信息及时入库
- **反馈机制**: 发现错误记忆及时修正

---

## 9. 验证清单

### 9.1 团队设置验证

| 检查项 | 验证方式 |
|--------|----------|
| 成员已加入 | Dashboard → Members 列表可见 |
| 权限配置正确 | Can Read 成员尝试写入应被拒 |
| 各成员 Key 可用 | 各自 `get_memories` 非报错 |

### 9.2 协作验证

| 检查项 | 验证方式 |
|--------|----------|
| 知识可搜索 | 成员 A 写入 → 成员 B `search_memories` 可召回 |
| 共享池完整 | `get_memories(project_id=...)` 返回跨用户记忆 |
| 事件通知 | 配置 Webhook 后写入记忆，观察回调 |

---

## 10. 常见问题

### 10.1 无法访问共享池

```
解决:
1. 确认 api_key 属于同一 Project
2. 确认使用了 project_id / git_remote 参数（个人 user_id 池不可跨用户）
3. admin key 可跨用户读取；普通成员 Key 需成员身份
```

### 10.2 知识冲突

```
解决:
1. 先 search_memories 确认是否重复
2. update_memory 更新已有记忆而非创建新的
3. 语义重复项由 Dream(Merge) 后台自动合并
```

### 10.3 成员离职

```
解决:
1. Dashboard → Members 移除成员 / 降级为 Can Read
2. 其记忆仍在共享池中（metadata.by 可追溯）
3. 必要时 delete_entities 清理其个人实体记忆
```

---

*文档更新: 2026-09-18*
*下一步: [Phase 4: AI IDE 自动化配置](./SOP-P4-automation.md)*

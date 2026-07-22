# SOP-M5: 团队协作指南


---

## 1. 概述

### 1.1 目标

跨项目共享知识和经验，团队成员间高效协作，知识资产的持续积累。

### 1.2 团队协作架构

```
┌─────────────────────────────────────────────────────────────────┐
│                     团队协作架构                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐       │
│  │  Wing:      │    │  Wing:      │    │  Wing:      │       │
│  │  team-      │◄──►│  project-   │◄──►│  wing_      │       │
│  │  backend     │    │  payment    │    │  alice      │       │
│  └─────────────┘    └─────────────┘    └─────────────┘       │
│        │                  │                  │                  │
│        └──────────────────┼──────────────────┘                  │
│                           │                                     │
│                    ┌──────▼──────┐                              │
│                    │   Tunnel     │                              │
│                    │  (跨Wing连接)│                              │
│                    └─────────────┘                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 团队 Wing 组织

### 2.1 Wing 类型

| 类型 | 命名规范 | 用途 | 可见性 |
|------|----------|------|----------|
| 个人 Wing | `wing_{name}` | 个人记忆 | 仅本人 |
| 项目 Wing | `project_{name}` | 项目共享 | 项目成员 |
| 团队 Wing | `team_{dept}` | 团队共享 | 部门成员 |

### 2.2 创建团队 Wing

```bash
# 创建后端团队 Wing
mempalace_create_wing \
  --name "team-backend" \
  --description "后端团队知识库"
```

### 2.3 创建团队 Room

```bash
# API 设计规范
mempalace_create_room \
  --wing "team-backend" \
  --name "api-standards" \
  --description "API 设计规范与标准"

# 数据库规范
mempalace_create_room \
  --wing "team-backend" \
  --name "database-guidelines" \
  --description "数据库设计规范"

# 事故报告
mempalace_create_room \
  --wing "team-backend" \
  --name "incident-reports" \
  --description "生产事故报告"
```

### 2.4 建议的团队 Wing 结构

```
team-backend/
├── api-standards        # API 设计规范
├── database-guidelines  # 数据库规范
├── architecture        # 架构决策
├── incident-reports    # 事故报告
├── lessons-learned     # 经验教训
└── best-practices      # 最佳实践

team-frontend/
├── component-library   # 组件库规范
├── coding-style        # 编码风格
├── design-system      # 设计系统
└── accessibility      # 无障碍规范

team-shared/
├── onboarding         # 新人入门
├── tools-setup        # 工具配置
└── process           # 流程规范
```

---

## 3. 权限管理

### 3.1 授权访问

```bash
# 授权团队成员
mempalace_grant_access \
  --wing "team-backend" \
  --user "alice"

mempalace_grant_access \
  --wing "team-backend" \
  --user "bob"
```

### 3.2 撤销访问

```bash
# 撤销访问权限
mempalace_revoke_access \
  --wing "team-backend" \
  --user "former_member"
```

### 3.3 请求访问

```bash
# 请求访问团队 Wing
mempalace_request_access \
  --wing "team-backend"
```

### 3.4 权限级别

| 级别 | 权限 | 说明 |
|------|------|------|
| owner | 全部 | Wing 创建者 |
| admin | 全部 | Wing 管理员 |
| member | 读写 | 团队成员 |
| viewer | 只读 | 访客 |

---

## 4. 跨项目知识共享

### 4.1 Tunnel 连接

Tunnel 用于跨 Wing 关联相关知识。

```bash
# 创建 Tunnel
mempalace_create_tunnel \
  --source-wing "project-payment" \
  --source-room "implementation" \
  --target-wing "team-backend" \
  --target-room "lessons-learned" \
  --label "支付模块经验共享"
```

### 4.2 跨 Wing 搜索

```bash
# 搜索多个 Wing
mempalace_search \
  --wing-ids ["project-payment", "team-backend"] \
  --query "文件上传 方案"
```

### 4.3 加载多 Wing 上下文

```bash
# 加载多个相关项目的上下文
mempalace_get_context \
  --wing-ids ["project-payment", "project-oss", "team-backend"]
```

---

## 5. ADR 管理

### 5.1 创建 ADR

```bash
manage_adr --op create --adr '{
  "id": "ADR-042",
  "title": "统一使用阿里云 OSS",
  "status": "accepted",
  "context": "需要统一文件存储方案",
  "decision": "采用阿里云 OSS",
  "consequences": [
    "增加云服务成本",
    "需要统一配置管理"
  ]
}'
```

### 5.2 查询 ADR

```bash
# 搜索 ADR
manage_adr --op search --query "文件存储"

# 查看特定 ADR
manage_adr --op get --id "ADR-042"
```

### 5.3 更新 ADR 状态

```bash
# 废弃 ADR
manage_adr --op update --id "ADR-023" --status "deprecated"

# 修改 ADR
manage_adr --op update --id "ADR-023" --decision "新决策内容..."
```

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
│  2. 提取有价值的知识                                        │
│     ↓                                                        │
│  3. 选择合适的 Wing/Room                                    │
│     ↓                                                        │
│  4. mempalace_add_drawer 添加                              │
│     ↓                                                        │
│  5. 创建 Tunnel 关联 (可选)                                 │
│     ↓                                                        │
│  6. 通知团队成员 (可选)                                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 贡献类型

| 类型 | Room | Hall | 示例 |
|------|------|------|------|
| 技术决策 | `architecture` | `hall_facts` | ADR 记录 |
| 最佳实践 | `best-practices` | `hall_advice` | 代码规范 |
| 经验教训 | `lessons-learned` | `hall_discoveries` | Bug 分析 |
| 工具配置 | `tools-setup` | `hall_preferences` | IDE 配置 |
| 事故报告 | `incident-reports` | `hall_events` | 复盘记录 |

### 6.3 贡献示例

```bash
# 贡献架构决策
mempalace_add_drawer \
  --wing "team-backend" \
  --room "architecture" \
  --hall "hall_facts" \
  --content "ADR-042: 统一使用阿里云 OSS

状态: accepted
日期: 2024-07-10

决策:
- 采用阿里云 OSS 作为统一文件存储
- 封装 FileService 统一接口
- 使用 STS 令牌实现临时凭证

影响:
- 需要统一配置管理
- 考虑多账号隔离"

# 贡献最佳实践
mempalace_add_drawer \
  --wing "team-backend" \
  --room "best-practices" \
  --hall "hall_advice" \
  --content "Spring Boot 最佳实践:

1. 使用构造器注入代替 @Autowired
2. 配置使用 @ConfigurationProperties
3. 使用 @Validated 进行参数校验
4. 异常统一使用 @ControllerAdvice 处理
5. 日志使用占位符而非字符串拼接"
```

---

## 7. 新人 Onboarding

### 7.1 Onboarding Wing 结构

```
onboarding/
├── getting-started     # 入门指南
│   ├── 环境搭建
│   ├── 代码规范
│   └── 开发流程
├── tools-setup       # 工具配置
│   ├── IDE 配置
│   ├── Git 配置
│   └── CI/CD
└── coding-standards # 代码规范
    ├── Java 规范
    ├── SQL 规范
    └── API 规范
```

### 7.2 Onboarding 知识导出

```bash
# 导出 onboarding 知识
mempalace_export \
  --wing "onboarding" \
  --format markdown \
  --output onboarding-guide.md
```

### 7.3 新人接收流程

```
1. 克隆团队 Palace
   ↓
2. 获取 onboarding guide
   ↓
3. 按指南配置开发环境
   ↓
4. 完成第一个任务
   ↓
5. 开始贡献团队知识
```

---

## 8. 团队协作最佳实践

### 8.1 Wing 维护

- **定期清理**: 归档不再活跃的项目 Wing
- **更新命名**: 确保 Wing 名称清晰易懂
- **监控使用**: 定期检查各 Wing 的活跃度

### 8.2 知识质量

- **具体性**: 提供足够的上下文和示例
- **可操作性**: 他人看后能据此行动
- **定期更新**: 过时知识及时更新或归档

### 8.3 协作文化

- **鼓励贡献**: 认可知识贡献者
- **分享文化**: 有价值的信息及时共享
- **反馈机制**: 发现问题及时纠正

### 8.4 命名规范

| 资源 | 命名规范 | 示例 |
|------|----------|------|
| 个人 Wing | `wing_{name}` | `wing_alice` |
| 项目 Wing | `project_{name}` | `project-payment` |
| 团队 Wing | `team_{dept}` | `team-backend` |
| Room | `{topic}` | `api-standards` |
| Drawer | 描述性标题 | `ADR-042: 统一使用 OSS` |

---

## 9. 验证清单

### 9.1 团队设置验证

| 检查项 | 验证方式 |
|--------|----------|
| Team Wing 存在 | `mempalace_list_wings` 查看 |
| Room 结构正确 | `mempalace_list_rooms --wing "team-backend"` |
| 权限配置正确 | 尝试访问团队 Wing |

### 9.2 协作验证

| 检查项 | 验证方式 |
|--------|----------|
| 知识可搜索 | `mempalace_search` 测试 |
| Tunnel 连接正常 | 跨 Wing 搜索验证 |
| ADR 管理可用 | `manage_adr --op search` 测试 |

---

## 10. 常见问题

### 10.1 无法访问团队 Wing

```
解决:
1. 检查是否被授权: mempalace_list_wings
2. 联系 Wing 所有者请求授权
3. 确认 Wing 未被归档
```

### 10.2 知识冲突

```
解决:
1. 搜索现有知识确认是否重复
2. 使用 mempalace_merge_drawer 合并
3. 更新已有记忆而非创建新的
```

### 10.3 Wing 归档恢复

```bash
# 恢复归档的 Wing
mempalace_restore_wing --wing_id "archived-project"
```

---

*下一步: [Phase 4: AI IDE 自动化配置](./SOP-P4-automation.md)*

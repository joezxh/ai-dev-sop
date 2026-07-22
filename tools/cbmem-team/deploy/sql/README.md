# cbmem-team SQL 文件说明

## 文件列表

| 文件 | 说明 |
|------|------|
| `schema-mysql.sql` | MySQL 8.0 完整建表语句（27 张表 + 外键 + Views） |
| `schema-sqlite.sql` | SQLite 完整建表语句（27 张表） |
| `seed-mysql.sql` | MySQL 初始化数据（admin 用户 + 49 工具 + 3 BP + 3 工作流 + 5 模板 + 3 AI 工具） |
| `seed-sqlite.sql` | SQLite 初始化数据（同上） |

## 主键设计原则

### BIGINT AUTO_INCREMENT（主键）
以下表的主键使用 `BIGINT AUTO_INCREMENT`，由数据库自动生成：
- `sys_users.id`
- `pm_teams.id`
- `pm_projects.id`
- `pm_modules.id`
- `ai_sessions.id`
- `ai_memories_templates.id`
- `ai_memories.id`
- 其他业务表的主键（自增 BIGINT）

### VARCHAR（业务标识符）
以下表的主键使用业务生成的字符串标识符：
- `ai_tool_directory.tool_id` - 工具唯一标识（如 `mempalace_status`）
- `pm_best_practices.id` - BP 标识（如 `bp-naming-mempalace-drawer`）
- `pm_workflows.workflow_id` - 工作流标识（如 `wf-commit-precheck`）
- `ai_tools.id` - AI 工具标识（如 `aitool_mempalace`）
- `pm_tickets.ticket_id` - 工单标识
- `pm_repo_pipelines.pipeline_id` - 流水线标识

## 表分类

### 核心表（v1 legacy）
| 表名 | 说明 |
|------|------|
| `sys_users` | 用户表（含 password_hash, role, must_change_password） |
| `pm_projects` | 项目表（v1 兼容） |
| `ai_sessions` | AI 会话记录（含 MemPalace 同步水印） |
| `ai_session_turns` | 会话轮次详情 |
| `ai_summarize_tasks` | 总结任务 |
| `ai_distill_tasks` | 提炼任务 |
| `sys_console_sessions` | 控制台会话（cookie 认证） |

### M1 扩展
| 表名 | 说明 |
|------|------|
| `ai_tool_directory` | 工具目录（49 内置条目） |
| `ai_tool_invocation_logs` | 工具调用日志 |

### M2 扩展
| 表名 | 说明 |
|------|------|
| `pm_teams` | 团队表 |
| `pm_team_members` | 团队成员关联 |
| `pm_modules` | 模块树 |
| `pm_best_practices` | 最佳实践 |
| `pm_bp_versions` | 最佳实践版本历史 |

### M3 扩展
| 表名 | 说明 |
|------|------|
| `pm_workflows` | 工作流定义 |
| `pm_workflow_versions` | 工作流版本历史 |
| `pm_workflow_runs` | 工作流运行记录 |
| `pm_tickets` | 问题工单 |

### M4 扩展
| 表名 | 说明 |
|------|------|
| `pm_repo_pipelines` | 仓库流水线 |
| `pm_repo_pipeline_runs` | 流水线运行记录 |
| `pm_bp_candidates` | BP 候选 |

### M5 扩展
| 表名 | 说明 |
|------|------|
| `ai_memories_templates` | 记忆模板 |
| `ai_memories` | 记忆条目 |

### M6 扩展
| 表名 | 说明 |
|------|------|
| `ai_tools` | AI 工具目录 |
| `ai_tool_invocations` | AI 工具调用记录 |

### 系统表
| 表名 | 说明 |
|------|------|
| `sys_refresh_tokens` | JWT 刷新令牌 |
| `sys_audit_logs` | 审计日志 |

## 部署顺序

### MySQL
```bash
# 1. 创建数据库
mysql -u root -p -e "CREATE DATABASE cbmem CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 2. 执行建表
mysql -u root -p cbmem < schema-mysql.sql

# 3. 初始化数据
mysql -u root -p cbmem < seed-mysql.sql
```

### SQLite
```bash
# 1. 创建数据库
sqlite3 cbmem.db < schema-sqlite.sql

# 2. 初始化数据
sqlite3 cbmem.db < seed-sqlite.sql
```

## 默认账号

| 用户名 | 密码 | 角色 |
|--------|------|------|
| admin | admin123 | admin |

## 重要说明

### 主键类型
- **BIGINT AUTO_INCREMENT**: 适用于需要数据库自动生成 ID 的表（如用户、团队、项目等）
- **VARCHAR**: 适用于有业务含义的标识符（如工具 ID、BP ID、工作流 ID 等）

### MemPalace 同步
- `ai_sessions` 表包含 `mempalace_synced_turns` 水印字段
- 用于追踪已同步到 MemPalace 的轮次
- 支持幂等自动同步

## 版本历史

| 版本 | 更新内容 |
|------|----------|
| v4.0 | M5/M6 表结构 + MemPalace 自动同步水印 |
| v3.0 | 完整 M1-M4 表结构 |
| v2.0 | 初始双轨支持 |

# 双轨记忆系统工具速查表

 - [安装](./ready.md#1-记忆体安装)  
 - [项目理解](./SOP-M2-understanding.md)
 - [开发调试](./SOP-M3-development.md)
 - [知识沉淀](./SOP-M4-knowledge.md) [蒸馏部分开发中]
 - [团队协作](./SOP-M5-collaboration.md)
 
---

## 1. MemPalace 工具 (轨道 A)

### 1.1 读取工具 (P0-P2)

| 工具 | 参数 | 说明 | 示例 |
|------|------|------|------|
| `mempalace_status` | - | 系统状态 | `mempalace status` |
| `mempalace_list_wings` | - | 列出 Wing | `mempalace list-wings` |
| `mempalace_search` | query | 语义搜索 | `mempalace search "架构决策"` |
| `mempalace_list_rooms` | wing_id | 列出 Room | `mempalace list-rooms --wing "team-backend"` |
| `mempalace_recall` | query | 模糊召回 | `mempalace recall "忘了存哪了"` |
| `mempalace_wing_info` | wing_id | Wing 详情 | `mempalace wing-info --wing "team-backend"` |
| `mempalace_view_drawer` | drawer_id | 查看抽屉 | `mempalace view-drawer --id "xxx"` |
| `mempalace_get_context` | wing_ids[] | 加载上下文 | `mempalace get-context --wing-ids "proj1,proj2"` |

### 1.2 写入工具 (P2-P3)

| 工具 | 参数 | 说明 | 示例 |
|------|------|------|------|
| `mempalace_add_drawer` | wing,room,hall,content | 添加抽屉 | `mempalace add-drawer --wing "proj" --room "arch" --hall "facts" --content "..."` |
| `mempalace_checkpoint` | session_id | 批量保存 | `mempalace checkpoint --session "s1"` |
| `mempalace_update_drawer` | drawer_id,body | 更新抽屉 | `mempalace update-drawer --id "xxx" --body "新内容"` |
| `mempalace_delete_drawer` | drawer_id | 删除抽屉 | `mempalace delete-drawer --id "xxx"` |
| `mempalace_tag_drawer` | drawer_id,tags[] | 标签抽屉 | `mempalace tag-drawer --id "xxx" --tags "支付,Redis"` |

### 1.3 管理工具 (P4-P6)

| 工具 | 参数 | 说明 | 示例 |
|------|------|------|------|
| `mempalace_create_wing` | wing_name | 创建 Wing | `mempalace create-wing --name "team-xxx"` |
| `mempalace_create_room` | wing_id,room_name | 创建 Room | `mempalace create-room --wing "t" --name "api"` |
| `mempalace_grant_access` | wing_id,user_id | 授权访问 | `mempalace grant --wing "t" --user "alice"` |
| `mempalace_revoke_access` | wing_id,user_id | 撤销访问 | `mempalace revoke --wing "t" --user "bob"` |
| `mempalace_export` | wing_id,format | 导出数据 | `mempalace export --wing "t" --format md` |
| `mempalace_create_tunnel` | src,tgt | 创建隧道 | `mempalace create-tunnel --src "p1/r1" --tgt "p2/r2"` |

---

## 2. Codebase 工具 (轨道 B)

### 2.1 索引工具 (P0-P1)

| 工具 | 参数 | 说明 | 示例 |
|------|------|------|------|
| `index_repository` | repo_path,mode | 索引仓库 | `index_repository --repo-path ./src --mode incremental` |
| `index_status` | project_id | 索引状态 | `index_status --project-id "xxx"` |
| `list_projects` | - | 列出项目 | `list_projects` |
| `delete_project` | project_id | 删除项目 | `delete_project --project-id "xxx"` |

### 2.2 查询工具 (P0-P2)

| 工具 | 参数 | 说明 | 示例 |
|------|------|------|------|
| `search_code` | query,scope | 代码搜索 | `search_code --query "pagination" --scope "**/*.java"` |
| `search_graph` | label,name | 搜索图谱 | `search_graph --label class --name "User"` |
| `trace_path` | target,depth | 追踪路径 | `trace_path --target "UserService" --depth 5` |
| `query_graph` | cypher,params | Cypher 查询 | `query_graph --cypher "MATCH (n)"` |
| `get_code_snippet` | file,line_range | 获取片段 | `get_code_snippet --file "a.java" --line-range "1-50"` |
| `get_graph_schema` | - | 获取 Schema | `get_graph_schema` |

### 2.3 分析工具 (P0-P2)

| 工具 | 参数 | 说明 | 示例 |
|------|------|------|------|
| `get_architecture` | scope | 架构分析 | `get_architecture --scope full` |
| `detect_changes` | git_diff | 变更检测 | `detect_changes --git-diff "$(git diff)"` |
| `manage_adr` | op,adr | ADR 管理 | `manage_adr --op create --adr '{...}'` |
| `ingest_traces` | trace_bundle | 导入追踪 | `ingest_traces --bundle ./traces.json` |

---

## 3. Hall 分类速查

| Hall | 用途 | 内容示例 |
|------|------|----------|
| `hall_facts` | 事实性知识 | "我们使用 PostgreSQL 存储数据" |
| `hall_events` | 时间线事件 | "2024-07-10 完成支付模块重构" |
| `hall_discoveries` | 技术发现 | "发现 N+1 查询，使用 EntityGraph 解决" |
| `hall_preferences` | 团队偏好 | "团队偏好 Lombok 简化 POJO" |
| `hall_advice` | 建议指导 | "新模块建议使用 DDD 架构" |

---

## 4. Wing 命名规范

| 类型 | 格式 | 示例 |
|------|------|------|
| 个人 | `wing_{name}` | `wing_alice` |
| 项目 | `project_{name}` | `project-payment` |
| 团队 | `team_{dept}` | `team-backend` |
| Onboarding | `onboarding` | `onboarding` |

---

## 5. 常用命令速查

### 5.1 项目理解

```bash
# 架构概览
get_architecture --scope full

# 搜索代码
search_code --query "关键词" --scope "**/*.java"

# 追踪调用
trace_path --target "方法名" --depth 5
```

### 5.2 团队记忆

```bash
# 搜索记忆
mempalace search "架构决策"

# 加载上下文
mempalace get-context --wing-ids "project-myapp"

# 添加记忆
mempalace add-drawer --wing "proj" --room "arch" --hall "facts" --content "..."
```

### 5.3 调试追踪

```bash
# 追踪问题
trace_path --target "Service.method" --depth 10

# 变更影响
detect_changes --git-diff "$(git diff)"
```

### 5.4 知识沉淀

```bash
# 会话归纳 (Console)
# 路径: /console/#/summarize

# 知识蒸馏 (Console)
# 路径: /console/#/distill

# 手动添加
mempalace add-drawer --wing "team" --room "lessons" --hall "discoveries" --content "..."
```

---

## 6. 问题与解决

| 问题 | 解决 |
|------|------|
| 搜索无结果 | 先 `mempalace mine` 挖掘数据 |
| 调用链太深 | 限制 `--depth` 参数 |
| JWT 401 | 续期: `POST /refresh` |
| 服务离线 | 检查 Docker/systemd 状态 |

---

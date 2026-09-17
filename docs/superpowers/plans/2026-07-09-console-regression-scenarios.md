> [!WARNING]
> **已废弃（2026-09-18）**：本文档描述的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，记忆功能统一替换为自托管 mem0（见 docs/quick-ref/mem0-ai-tools-config-guide.md）。本文仅作历史归档保留，内容不再维护。

# 双轨记忆控制台 — 50 场景回归测试

> **状态**：本文档为 Task 18 交付物，覆盖 Session/Auth、Users CRUD、Projects CRUD、Sessions、Summarize、Distill 六大模块共 50 个浏览器/手动测试场景。
> **测试环境**：cbmem-team 运行于 `http://127.0.0.1:8787`，控制台 `http://127.0.0.1:8787/console/`

---

## 会话与鉴权（10 场景）

### 1. 正常登录
| 字段 | 内容 |
|------|------|
| 编号 | AUTH-01 |
| 功能 | 登录 |
| 步骤 | 1. 打开 `/console/` → 2. 输入正确的 admin-token → 3. 点击「登录」 |
| 预期结果 | 登录成功，跳转到 `/console/#/users`，侧边栏显示 5 个菜单项 |
| 检查点 | - session cookie `cbmem_console` 已写入 - CSRF token 已获取 - 顶部显示「退出」按钮 |

### 2. 错误 token 登录
| 字段 | 内容 |
|------|------|
| 编号 | AUTH-02 |
| 功能 | 登录 |
| 步骤 | 1. 打开 `/console/` → 2. 输入错误的 token → 3. 点击「登录」 |
| 预期结果 | 页面显示红色错误提示「...」，不跳转 |
| 检查点 | - HTTP 401 响应 - 错误消息友好，不泄露服务端细节 |

### 3. 登出
| 字段 | 内容 |
|------|------|
| 编号 | AUTH-03 |
| 功能 | 登出 |
| 步骤 | 1. 已登录状态 → 2. 点击顶部「退出」按钮 |
| 预期结果 | 跳转回登录页，session cookie 已清除 |
| 检查点 | - `sessionId` / `csrfToken` 已清空 - 侧边栏不可见 |

### 4. 无 session 访问受保护端点
| 字段 | 内容 |
|------|------|
| 编号 | AUTH-04 |
| 功能 | 鉴权 |
| 步骤 | 1. 无登录状态 → 2. 直接访问 `/console/#/users` |
| 预期结果 | 自动重定向到登录页 |
| 检查点 | - 无 session 时 `current` computed 返回 Login 组件 |

### 5. CSRF token 缺失（POST）
| 字段 | 内容 |
|------|------|
| 编号 | AUTH-05 |
| 功能 | CSRF 保护 |
| 步骤 | 1. 登录后 → 2. 手动 curl POST `/api/console/users` 不带 `X-CSRF-Token` header |
| 预期结果 | HTTP 403，响应体 `{code: 4010001, msg: "CSRF token required"}` |
| 检查点 | - CSRF 中间件拦截 - 请求体未执行 |

### 6. CSRF token 错误
| 字段 | 内容 |
|------|------|
| 编号 | AUTH-06 |
| 功能 | CSRF 保护 |
| 步骤 | 1. 登录后 → 2. curl POST `/api/console/users` 带错误的 `X-CSRF-Token` header |
| 预期结果 | HTTP 403，响应体 `{code: 4010002, msg: "CSRF token mismatch"}` |
| 检查点 | - token 比对失败 - 请求体未执行 |

### 7. Session 过期
| 字段 | 内容 |
|------|------|
| 编号 | AUTH-07 |
| 功能 | Session TTL |
| 步骤 | 1. 登录后等待 session TTL 过期（或手动 DELETE session） → 2. 刷新页面 |
| 预期结果 | 重定向到登录页 |
| 检查点 | - cookie 过期 - `RequireSession` 中间件返回 401 |

### 8. 重复登录（旧 session 有效）
| 字段 | 内容 |
|------|------|
| 编号 | AUTH-08 |
| 功能 | 登录 |
| 步骤 | 1. 已登录状态 → 2. 再登录一次（同一 token） |
| 预期结果 | 两个 session cookie 均有效，后登录不使旧 session 失效 |
| 检查点 | - 旧 session 仍在 DB 中 - 新 session 独立创建 |

### 9. Session 数据库持久化
| 字段 | 内容 |
|------|------|
| 编号 | AUTH-09 |
| 功能 | Session 持久化 |
| 步骤 | 1. 登录 → 2. 重启 cbmem-team 进程 → 3. 同一浏览器继续访问 |
| 预期结果 | Session 仍然有效（DB 中 `console_sessions` 行未过期） |
| 检查点 | - `console_sessions.expires_at > now()` - 无需重新登录 |

### 10. Admin-token 作为 Bearer 登录
| 字段 | 内容 |
|------|------|
| 编号 | AUTH-10 |
| 功能 | 登录 |
| 步骤 | 1. `curl -X POST /api/console/login -H "Authorization: Bearer <admin-token>"` |
| 预期结果 | 登录成功，响应体含 `session_id` 和 `csrf_token` |
| 检查点 | - `X-Admin-Token` 和 `Authorization: Bearer` 均支持 |

---

## 用户管理 CRUD（10 场景）

### 11. 创建用户（完整字段）
| 字段 | 内容 |
|------|------|
| 编号 | USER-01 |
| 功能 | 用户创建 |
| 步骤 | 1. 登录 → 2. 进入「用户」→ 3. 点击「新增用户」→ 4. 填写 id/display_name/project_paths → 5. 提交 |
| 预期结果 | 用户出现在列表顶部，HTTP 200 |
| 检查点 | - `users` 表新增一行 - 列表自动刷新 - 成功提示 |

### 12. 创建用户（重复 ID）
| 字段 | 内容 |
|------|------|
| 编号 | USER-02 |
| 功能 | 用户创建 |
| 步骤 | 1. 登录 → 2. 新增用户 id=`alice`（已存在）→ 3. 提交 |
| 预期结果 | 弹窗不关闭，显示错误提示「id 已存在」或 HTTP 409 |
| 检查点 | - `users.id` 唯一约束生效 - 列表无变化 |

### 13. 用户列表分页
| 字段 | 内容 |
|------|------|
| 编号 | USER-03 |
| 功能 | 用户列表 |
| 步骤 | 1. 创建 25 个用户 → 2. 访问用户列表 |
| 预期结果 | 默认每页 20 条，底部有分页控件 |
| 检查点 | - 响应含 `list[]` 和 `total` - 分页切换正确 |

### 14. 更新用户 display_name
| 字段 | 内容 |
|------|------|
| 编号 | USER-04 |
| 功能 | 用户更新 |
| 步骤 | 1. 登录 → 2. 点击某用户「编辑」→ 3. 修改 display_name → 4. 提交 |
| 预期结果 | 列表中该用户的显示名已更新，`updated_at` 时间戳更新 |
| 检查点 | - `users.display_name` 已变更 - API PUT `/users/:id` 正确 |

### 15. 删除用户
| 字段 | 内容 |
|------|------|
| 编号 | USER-05 |
| 功能 | 用户删除 |
| 步骤 | 1. 登录 → 2. 点击某用户「删除」→ 3. 确认弹窗 → 4. 确认 |
| 预期结果 | 用户从列表消失 |
| 检查点 | - `users` 表该行保留（软删除）或移除 - 相关 session 仍存在 |

### 16. 撤销用户 Token
| 字段 | 内容 |
|------|------|
| 编号 | USER-06 |
| 功能 | Token 撤销 |
| 步骤 | 1. 登录 → 2. 用户列表找到某用户 → 3. 调用 `POST /users/:id/revoke` |
| 预期结果 | 用户 `disabled` 标志置 1，该用户所有 JWT 失效 |
| 检查点 | - 用户状态变为「停用」Tag - 旧 token 无法使用 |

### 17. Disabled 用户拒绝访问
| 字段 | 内容 |
|------|------|
| 编号 | USER-07 |
| 功能 | 权限控制 |
| 步骤 | 1. 将某用户 disabled=true → 2. 该用户用 JWT 尝试访问 MCP |
| 预期结果 | MCP 请求被拒绝，返回 403 |
| 检查点 | - 中间件检查 `users.disabled` - 不创建新的 process pool |

### 18. 用户 project_paths 白名单限制
| 字段 | 内容 |
|------|------|
| 编号 | USER-08 |
| 功能 | 权限控制 |
| 步骤 | 1. 用户 `alice` 的 `project_paths=["/code/foo"]` → 2. alice 尝试 MCP 请求 `project=/code/bar` |
| 预期结果 | 请求被拒绝，返回 403 Forbidden |
| 检查点 | - 白名单校验在 MCP handler 中执行 - `projects` 表路径匹配 |

### 19. 无白名单用户全路径访问
| 字段 | 内容 |
|------|------|
| 编号 | USER-09 |
| 功能 | 权限控制 |
| 步骤 | 1. 用户 `bob` 的 `project_paths=[]`（空）→ 2. bob 尝试 MCP 请求任意 project |
| 预期结果 | 请求通过（全路径允许） |
| 检查点 | - 空数组等同于无限制 - 不返回 403 |

### 20. 用户列表搜索过滤
| 字段 | 内容 |
|------|------|
| 编号 | USER-10 |
| 功能 | 用户列表 |
| 步骤 | 1. 用户列表有多个用户 → 2. 输入关键词搜索 |
| 预期结果 | 列表过滤为匹配 `id` 或 `display_name` 的结果 |
| 检查点 | - `?q=` 参数传递给后端 - 搜索词为空时返回全部 |

---

## 项目管理 CRUD（10 场景）

### 21. 创建项目
| 字段 | 内容 |
|------|------|
| 编号 | PROJ-01 |
| 功能 | 项目创建 |
| 步骤 | 1. 登录 → 2. 进入「项目」→ 3. 点击「新增项目」→ 4. 填写 name/path/wing/mcp_bin → 5. 提交 |
| 预期结果 | 项目出现在列表，HTTP 200 |
| 检查点 | - `projects` 表新增一行 - ID 自动生成 |

### 22. 项目列表分页
| 字段 | 内容 |
|------|------|
| 编号 | PROJ-02 |
| 功能 | 项目列表 |
| 步骤 | 1. 创建 30 个项目 → 2. 访问项目列表 |
| 预期结果 | 分页展示，每页 20 条 |
| 检查点 | - 响应含 `list[]` 和 `total` |

### 23. 删除项目
| 字段 | 内容 |
|------|------|
| 编号 | PROJ-03 |
| 功能 | 项目删除 |
| 步骤 | 1. 登录 → 2. 点击某项目「删除」→ 3. 确认 → 4. 确认 |
| 预期结果 | 项目从列表消失 |
| 检查点 | - `projects.deleted=1`（软删）- 关联 session 保留 |

### 24. 项目索引状态查询
| 字段 | 内容 |
|------|------|
| 编号 | PROJ-04 |
| 功能 | 项目索引 |
| 步骤 | 1. 项目列表 → 2. 查看某项目的索引状态列 |
| 预期结果 | 显示索引状态（已索引/未索引/索引中） |
| 检查点 | - 状态从 `codebase-memory-mcp` 实时获取 - Tag 颜色正确 |

### 25. 触发项目增量索引
| 字段 | 内容 |
|------|------|
| 编号 | PROJ-05 |
| 功能 | 项目索引 |
| 步骤 | 1. 点击某项目「触发索引」→ 2. 等待异步完成 |
| 预期结果 | `POST /projects/:id/reindex` 返回成功，状态变为「索引中」后转为「已索引」 |
| 检查点 | - 异步任务不阻塞响应 - 状态轮询或 webhook 更新 |

### 26. 项目列表搜索过滤
| 字段 | 内容 |
|------|------|
| 编号 | PROJ-06 |
| 功能 | 项目列表 |
| 步骤 | 1. 项目列表有多个 → 2. 输入路径关键词搜索 |
| 预期结果 | 列表过滤为匹配 `path` 或 `name` 的结果 |
| 检查点 | - `?q=` 参数正确传递 - 清空后恢复全部 |

### 27. 项目 wing 字段正确保存
| 字段 | 内容 |
|------|------|
| 编号 | PROJ-07 |
| 功能 | 项目配置 |
| 步骤 | 1. 新增项目，填写 wing=`wing-person-alice` → 2. 保存后查看详情 |
| 预期结果 | wing 字段正确显示 |
| 检查点 | - `projects.wing` 非空时写入 - 归纳/蒸馏时可用 |

### 28. 项目 mcp_bin 自定义路径保留
| 字段 | 内容 |
|------|------|
| 编号 | PROJ-08 |
| 功能 | 项目配置 |
| 步骤 | 1. 新增项目，填写 `mcp_bin=/custom/path/to/mcp` → 2. 保存后查看 |
| 预期结果 | `mcp_bin` 字段正确保留 |
| 检查点 | - 字段传入后端存储 - 不被默认值覆盖 |

### 29. 项目路径重复创建
| 字段 | 内容 |
|------|------|
| 编号 | PROJ-09 |
| 功能 | 项目创建 |
| 步骤 | 1. 已存在 path=`/code/foo` 的项目 → 2. 再创建相同 path 的项目 |
| 预期结果 | HTTP 409 或错误提示，项目不重复创建 |
| 检查点 | - `projects.path` UNIQUE 约束 |

### 30. 删除项目不影响已有 session
| 字段 | 内容 |
|------|------|
| 编号 | PROJ-10 |
| 功能 | 数据隔离 |
| 步骤 | 1. 创建项目 P，创建 session S（含 project_path=P.path）→ 2. 删除项目 P → 3. 查询 session S |
| 预期结果 | session S 仍可查询，project_path 字段保留原值 |
| 检查点 | - `sessions` 表无外键级联删除 - 项目软删除不清理 session |

---

## 会话记录（10 场景）

### 31. MCP 请求触发 session 采集
| 字段 | 内容 |
|------|------|
| 编号 | SESS-01 |
| 功能 | 会话采集 |
| 步骤 | 1. 用户在 IDE 使用 MCP 工具（如 `messages/create`）→ 2. 查询 session 列表 |
| 预期结果 | 新增 session 行，`user_id`/`project_path`/`started_at` 正确 |
| 检查点 | - capture middleware 触发 - `sessions` 表新增一行 - `turn_count >= 1` |

### 32. 多轮对话多条 turns 记录
| 字段 | 内容 |
|------|------|
| 编号 | SESS-02 |
| 功能 | 会话采集 |
| 步骤 | 1. 同一 session 发 5 条消息 → 2. 查询 session 详情 |
| 预期结果 | 5 条 `session_turns` 行，role 交替（user/assistant） |
| 检查点 | - `turn_no` 递增 - `role` 字段正确 |

### 33. 会话列表按用户筛选
| 字段 | 内容 |
|------|------|
| 编号 | SESS-03 |
| 功能 | 会话列表 |
| 步骤 | 1. 有多个用户的 session → 2. 调用 `GET /sessions?user_id=alice` |
| 预期结果 | 仅返回 alice 的 session |
| 检查点 | - SQL WHERE user_id 正确 - 列表不含其他用户 |

### 34. 会话详情完整展示
| 字段 | 内容 |
|------|------|
| 编号 | SESS-04 |
| 功能 | 会话详情 |
| 步骤 | 1. 点击某 session「查看」→ 2. 侧边抽屉展开 |
| 预期结果 | 显示 session 基本信息 + 所有 turns |
| 检查点 | - `session.user_id` / `session.project_path` 显示 - turns 按 turn_no 排序 |

### 35. Session turns 内容展示
| 字段 | 内容 |
|------|------|
| 编号 | SESS-05 |
| 功能 | 会话详情 |
| 步骤 | 1. 打开 session 详情 → 2. 查看每个 turn |
| 预期结果 | 每个 turn 显示 role + content，`pre` 标签正确渲染换行 |
| 检查点 | - `white-space: pre-wrap` 样式 - content 非空 |

### 36. 会话统计聚合
| 字段 | 内容 |
|------|------|
| 编号 | SESS-06 |
| 功能 | 统计 |
| 步骤 | 1. 调用 `GET /sessions-stats` |
| 预期结果 | 返回总会话数、总 turns 数、工具调用数 |
| 检查点 | - 聚合查询正确 - 数值非负整数 |

### 37. Top 用户按 session 数量
| 字段 | 内容 |
|------|------|
| 编号 | SESS-07 |
| 功能 | 统计 |
| 步骤 | 1. 调用 `GET /sessions-stats` |
| 预期结果 | 响应含 `top_users[]`，按 session 数量降序 |
| 检查点 | - 数组排序正确 - 用户名匹配 |

### 38. Top 项目按 session 数量
| 字段 | 内容 |
|------|------|
| 编号 | SESS-08 |
| 功能 | 统计 |
| 步骤 | 1. 调用 `GET /sessions-stats` |
| 预期结果 | 响应含 `top_projects[]`，按 session 数量降序 |
| 检查点 | - 数组排序正确 - 项目路径匹配 |

### 39. 无 user_id 的请求不采集
| 字段 | 内容 |
|------|------|
| 编号 | SESS-09 |
| 功能 | 隐私保护 |
| 步骤 | 1. 匿名请求（无 JWT）调用 MCP → 2. 查询 session 列表 |
| 预期结果 | 不创建 session 行 |
| 检查点 | - capture 中间件检查 user_id - 无记录写入 |

### 40. tool call 无 messages 不采集
| 字段 | 内容 |
|------|------|
| 编号 | SESS-10 |
| 功能 | 数据质量 |
| 步骤 | 1. MCP 请求 method=`tools/call` 但无 `messages/create` → 2. 检查 session |
| 预期结果 | 不创建 session 和 turn |
| 检查点 | - 方法名白名单过滤 - 非消息类工具不触发采集 |

---

## 会话归纳（5 场景）

### 41. 触发归纳返回 task_id
| 字段 | 内容 |
|------|------|
| 编号 | SUM-01 |
| 功能 | 归纳触发 |
| 步骤 | 1. 登录 → 2. 进入「归纳」→ 3. 填写 source_ids/depth/target_wing → 4. 点击「开始归纳」 |
| 预期结果 | HTTP 200，响应含 `task_id` |
| 检查点 | - `summarize_tasks` 表新增一行 - status=`running` |

### 42. 轮询任务状态
| 字段 | 内容 |
|------|------|
| 编号 | SUM-02 |
| 功能 | 归纳状态 |
| 步骤 | 1. 触发归纳后 → 2. 调用 `GET /summarize/:task_id` 多次 |
| 预期结果 | 状态从 `running` 变为 `done`，含 `result` 字段 |
| 检查点 | - 同步返回（不等待异步）- `result.hall_*` 字段存在 |

### 43. 归纳结果含 5 Hall
| 字段 | 内容 |
|------|------|
| 编号 | SUM-03 |
| 功能 | 归纳结果 |
| 步骤 | 1. 归纳任务完成 → 2. 页面 Tab 展示 |
| 预期结果 | 5 个 Tab：facts / events / discoveries / preferences / advice，各含数组 |
| 检查点 | - `hall_facts` 非空（若 session 有相关内容）- JSON 格式化显示 |

### 44. fake provider 归纳成功
| 字段 | 内容 |
|------|------|
| 编号 | SUM-04 |
| 功能 | LLM 集成 |
| 步骤 | 1. 启动时 `-llm-provider=fake` → 2. 触发归纳 |
| 预期结果 | 归纳成功返回固定结构（fake 回包） |
| 检查点 | - 不依赖外部 LLM - 结果写入 DB |

### 45. 空 source_ids 返回错误
| 字段 | 内容 |
|------|------|
| 编号 | SUM-05 |
| 功能 | 参数校验 |
| 步骤 | 1. 触发归纳，source_ids=[] → 2. 提交 |
| 预期结果 | HTTP 400，错误提示「source_ids required」 |
| 检查点 | - 前端或后端校验生效 - 不创建空任务 |

---

## 会话蒸馏（5 场景）

### 46. 触发蒸馏返回 task_id
| 字段 | 内容 |
|------|------|
| 编号 | DIST-01 |
| 功能 | 蒸馏触发 |
| 步骤 | 1. 登录 → 2. 进入「蒸馏」→ 3. 填写 source_ids/min_value_score/target_wing → 4. 点击「开始蒸馏」 |
| 预期结果 | HTTP 200，响应含 `task_id` |
| 检查点 | - `distill_tasks` 表新增一行 - status=`running` |

### 47. Commit 写入 MemPalace
| 字段 | 内容 |
|------|------|
| 编号 | DIST-02 |
| 功能 | 蒸馏提交 |
| 步骤 | 1. 蒸馏任务完成后 → 2. 点击「写入 MemPalace」 |
| 预期结果 | HTTP 200，`POST {mempalace-base}/api/drawers` 被调用 |
| 检查点 | - `distill_tasks.mempalace_synced=1` - MemPalace 有对应 drawer |

### 48. Commit 带 target_wing
| 字段 | 内容 |
|------|------|
| 编号 | DIST-03 |
| 功能 | 蒸馏提交 |
| 步骤 | 1. 蒸馏完成后填写 target_wing=`wing-person-alice` → 2. 点击「写入 MemPalace」 |
| 预期结果 | MemPalace drawer 写入到指定 wing |
| 检查点 | - wing 参数正确传递 - MemPalace API 调用的 room/hall 正确 |

### 49. Commit 幂等（重复提交无副作用）
| 字段 | 内容 |
|------|------|
| 编号 | DIST-04 |
| 功能 | 幂等性 |
| 步骤 | 1. 已 commit 的 task → 2. 再次点击「写入 MemPalace」 |
| 预期结果 | 第二次提交返回成功（idempotent），不重复写入 |
| 检查点 | - `mempalace_synced=1` 时直接返回 - 不重复调用 MemPalace API |

### 50. min_value_score 过滤
| 字段 | 内容 |
|------|------|
| 编号 | DIST-05 |
| 功能 | 蒸馏规则 |
| 步骤 | 1. 触发蒸馏，`min_value_score=0.9`（高阈值）→ 2. 查看结果 |
| 预期结果 | 仅 value_score >= 0.9 的片段出现在结果中 |
| 检查点 | - 低分片段被过滤 - 结果数量减少 |

---

## 验收标准

| 编号 | 条件 |
|------|------|
| 1 | 全部 50 场景通过 HTTP API 或浏览器 UI 验证 |
| 2 | `pnpm build` 无构建错误（SSR 错误可忽略，为运行时行为） |
| 3 | `examples/e2e-console.sh` headless 测试全绿 |
| 4 | 浏览器打开 `/console/` 出现登录页 → 登录 → 5 模块全部可用 |

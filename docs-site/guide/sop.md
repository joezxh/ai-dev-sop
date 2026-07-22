
---


## 2.开发测试提示词生成

### 2.1 已有工程提示词
从已有工程生成。

系统模块
```
/brainstorm
参照模板 以@mediation-web/docs/scene/scene-template.md 的结构与风格，生成一份完整的服务端到端浏览器自动化测试开发文档。模板的所有章节顺序、命名、子节、表格列、代码块结构、占位符语义都必须 1:1 对齐:
-   后端 @mediation-basic/mediation-module-system服务名：system 端口 8082
-   前端 views目录：@mediation-web/src/api/systemapi模块: @mediation-web/src/views/system

输出文件路径：`mediation-web/docs/scene/system-scene.md`
文档更新要求：对已有文件中的各个测试场景进行维护，只能添加新场景或细化现有场景，严禁删除任何已有功能的测试场景。必须确保遍历每个页面的所有功能，页面上的每个按钮、每个交互元素的功能都必须有对应的测试场景覆盖，保证测试的完整性和全面性。
特别强调：System 服务测试模块 必须完成所有模块的测试提示词的编写
```

AI模块
```
参照模板 以@mediation-web/docs/scene/scene-template.md 的结构与风格，生成一份完整的服务端到端浏览器自动化测试开发文档。模板的所有章节顺序、命名、子节、表格列、代码块结构、占位符语义都必须 1:1 对齐:
-   后端  mediation-harness-module 服务名：harness 端口 8087
-   前端 views目录：@mediation-web/src/api/harness  api模块: @mediation-web/src/views/harness

文档更新要求：对已有文件中的各个测试场景进行维护，只能添加新场景或细化现有场景，严禁删除任何已有功能的测试场景。必须确保遍历每个页面的所有功能，页面上的每个按钮、每个交互元素的功能都必须有对应的测试场景覆盖，保证测试的完整性和全面性。
输出文件路径：`mediation-web/docs/scene/harness-scene.md`
特别强调：harness 服务测试模块 必须完成所有模块的测试提示词的编写
```

单个模块测试提示词
```
/brainstorm
根据页面 @mediation-web/src/views/kms/legal/area 以及常规行政区域管理功能的需求，对 KMS-08 的测试场景进行详细细化和完善。请分析该行政区域管理页面的实际功能特性，包括但不限于列表展示、搜索筛选、新增、编辑、删除等操作，并基于真实的页面交互流程和业务逻辑，制定更加全面和贴近实际使用的测试场景。
```

测试提示词转换为开发提示词
```
使用 /qa skill 执行测试时，针对模板 ​ @mediation-web/docs/scene/scene-template.md ​ 中   @scene-template.md (246-365)  模块的测试场景，如果发现这些功能尚未开发实现，请将测试流程转换为模块开发流程。

请提供一个完整的提示词转换方案，将原本的测试提示词转化为开发提示词，并生成相应的 Skill 配置。

具体要求：
1. 将测试场景（CRUD 操作验证）转换为对应的开发任务（前端页面 + 后端接口实现）
2. 保持原有的模块结构和功能边界不变
3. 生成完整的开发提示词，完整开发流程必须包含以下几点：
   - 前端页面组件 (Vue 文件)
   - 前端 API 接口封装 (TypeScript 文件) 
   - 后端 Controller、Service、DTO、DO、Mapper 等 Java 文件
   - 数据库表结构（如需要）
   - 菜单权限配置
   - 字典数据初始化（如需要）

4. 生成对应的 Skill 配置文件，用于自动化执行开发任务
5. 确保开发完成后能够满足原测试场景的所有验证要求

---

**/qa-dev Skill (二合一-增强版)**

`/qa-dev` = `/qa` + 浏览器测试 + 报告生成 + 自动修复 + 无人值守批量执行。

Skill 文件: `@.claude/skills/qa-dev/SKILL.md`

```
┌────────────────────────────────────────────────────────────────────────────┐
│                              /qa-dev                                       │
├────────────────────────────────────────────────────────────────────────────┤
│  Phase 1: 浏览器自动化测试 → Phase 2: 测试报告 → Phase 3: 自动开发       │
└────────────────────────────────────────────────────────────────────────────┘
```

**使用方式:**

```bash
# 完整流程: 测试 → 报告 → 修复 → 开发 → 验证
/qa-dev --file docs/scene/system-scene.md --module SYS-01

# 仅测试不开发
/qa-dev --test-only --file docs/scene/system-scene.md --module SYS-01

# 仅开发不测试
/qa-dev --dev-only --module SYS-01

# ========== 无人值守批量执行 ==========

# 无人值守批量执行所有模块
/qa-dev --batch --unattended --file docs/scene/system-scene.md

# 无人值守批量执行指定范围
/qa-dev --batch --unattended --file docs/scene/system-scene.md --from SYS-01 --to SYS-10

# 仅测试无人值守
/qa-dev --batch --test-only --unattended --file docs/scene/system-scene.md

# 仅开发失败模块无人值守
/qa-dev --batch --dev-only --failed --unattended --file docs/scene/system-scene.md
```

**核心能力对比:**

| 能力 | /qa | /qa-dev |
|------|-----|---------|
| 浏览器自动化测试 | ✅ | ✅ |
| 端到端交互测试 | ✅ | ✅ |
| 测试报告生成 | ✅ | ✅ |
| 自动修复问题 | ✅ | ✅ |
| **无人值守批量执行** | ❌ | ✅ |
| **开发代码生成** | ❌ | ✅ |
| **回归验证** | ❌ | ✅ |

**无人值守模式特性:**
- 无任何提示、选择或确认请求
- 所有决策自动执行
- 自动处理失败和错误
- 完成后生成完整报告

**Skill 安装清单:**
- [x] `.claude/skills/qa-dev/SKILL.md` (增强版)
- [x] `.claude/skills/qa-dev/QUICKSTART.md`
- [x] `.claude/skills/qa-dev/scripts/run.sh`

---

**/qa-to-dev Skill (二阶段)**

`/qa-to-dev` = `/qa` + 转换为开发提示词，需要手动执行开发。

Skill 文件: `@.claude/skills/qa-to-dev/SKILL.md`

**Skill 安装清单:**
- [x] `.claude/skills/qa-to-dev/SKILL.md`
- [x] `docs/scene/qa-to-dev-complete-solution.md`

---

**Skill 对比:**

| Skill | 功能 | 使用场景 |
|-------|------|---------|
| `/qa` | 仅测试 + 报告 | 已知功能已开发 |
| `/qa-to-dev` | 测试→开发提示词 | 手动执行开发 |
| `/qa-dev` | 测试→报告→开发→验证 | **一体化自动化+无人值守** |

---

┌─────────────────────────────────────────────────────────────┐
│  1. 执行 /qa-dev 测试                                       │
│     ↓                                                       │
│  2. 浏览器自动化测试 + 生成报告                             │
│     ↓                                                       │
│  3. 发现功能未开发 → 自动分析缺失项                        │
│     ↓                                                       │
│  4. 自动生成代码 + 执行开发                                 │
│     ↓                                                       │
│  5. 回归验证                                               │
│     ↓                                                       │
│  6. 全部通过 ✓ / 失败模块记录 ✓                           │
└─────────────────────────────────────────────────────────────┘

**Cursor 模型推荐 Claude-opus-4.8-thinging-high 模型**  
**Qoder 模型推荐 QWen3.7-MAX 模型**  

提示词生成，其中的测试提示词各个场景等要认证审核Review，与产品需求一致

### 2.2 需求转为提示词

需求转为提示词，生成一份完整的AI开发场景技能字典文档。
```
根据提供的标准需求PRD文档，以 scene-template.md 文件作为模板，生成一个完整的AI开发场景技能字典文档。

具体要求如下：

1. 文档结构：
   - 保持与 scene-template.md 相同的整体结构和章节组织
   - 包含测试前置条件、全局测试策略、Skill调用方式、问题发现与修复流程等章节
   - 为XX服务的优先级模块（XXX-1至XXX-24）创建完整的测试模块定义

2. 模块整理：
   - 整理出以下N个XX模块的具体实现方案：[需修改，模块通过全局整理提示给出]
     * SYS-18 API 访问日志 - 记录所有 API 请求日志
     * SYS-19 API 错误日志 - 记录 API 错误日志  

3. 测试提示词开发：
   - 为每个模块编写详细的测试场景和测试提示词
   - 包含前置操作、测试步骤、预期结果、API端点对照等
   - 遵循现有的测试模式（页面加载、数据加载、交互响应、表单验证等）

4. 代码开发提示词：
   - 为每个模块提供前端页面组件开发提示词（位于xxx目录下）
   - 提供后端API接口和业务逻辑开发提示词
   - 包含文件路径、权限标识、API端点等详细信息

5. AI开发工具集成：
   - 充分利用现有的Skill工具，如：
     * /qa, /qa-only, /open-gstack-browser, MCP Playwright
     * brainstorming, writing-plans, gsd-*系列命令
     * agent-skills的/spec, /plan, /build等功能
   - 提供具体的Skill调用方式和参数示例
   - 确保提示词能够指导AI工具完成完整的开发、测试、验证流程

6. 一致性要求：
   - 保持与现有xxx模块的一致性
   - 遵循现有的代码规范和架构模式
   - 确保前后端集成测试通过
```

### 2.3 一句话PRD文档生成

```
# 角色
你是 mediation-platform 资深产品经理 + 架构师,负责将一句话需求转换为 PRD。

# 输入
- 一句话需求:`{{USER_INPUT}}`
- 服务短名:`{{NAME}}`
- 中文标题:`{{NAME_TITLE}}`
- 后端端口:`{{PORT}}`
- 输出路径:`{{PRD_FILE}}`

# 任务
生成完整 PRD 文档,章节结构 **1:1 对齐** `mediation-web/docs/scene/scene-template.md` 的 §1 ~ §8 骨架。
本 PRD 是后续 N2~N5 的唯一数据源,请最大化信息密度。

# 输出文档骨架(必须严格遵循)

```markdown
# {{NAME_TITLE}} 服务 — PRD 文档

> 一句话需求原文:{{USER_INPUT}}
> 服务名:{{NAME}}(:{{PORT}})
> 文档版本:1.0.0 (YYYY-MM-DD)

---

## 目录
- [1.项目背景与目标](#1-项目背景与目标)
- [2.用户角色与权限模型](#2-用户角色与权限模型)
- [3.功能需求](#3-功能需求)
- [4.非功能需求](#4-非功能需求)
- [5.接口契约(API 草案)](#5-接口契约api-草案)
- [6.数据模型(初稿)](#6-数据模型初稿)
- [7.UI/UX 规范](#7-uiux-规范)
- [8.验收标准](#8-验收标准)
- [9.风险与依赖](#9-风险与依赖)
- [10.子模块拆分(由 N2 填充)](#10-子模块拆分由-n2-填充)

---

## 1.项目背景与目标
- 1.1 业务背景
- 1.2 业务目标
- 1.3 范围(包含 / 不包含)
- 1.4 成功指标(KPI)

## 2.用户角色与权限模型
| 角色 | 权限码前缀 | 关键权限 | 备注 |

## 3.功能需求
按 Epic → Story → Acceptance Criteria 三级展开
### 3.1 Epic-1 <名称>
#### Story 1.1 <用户故事>
**AC**:
- [ ] AC1
- [ ] AC2

## 4.非功能需求
- 4.1 性能(QPS / RT / 并发)
- 4.2 安全(鉴权 / 脱敏 / 审计)
- 4.3 可用性(SLA / 容灾)
- 4.4 可观测性(日志 / 指标 / 链路)

## 5.接口契约(API 草案)
| 模块 | 操作 | 方法 | 路径 | 入参 | 出参 | 权限码 |

## 6.数据模型(初稿)
| 表名 | 字段 | 类型 | 索引 | 备注 |

## 7.UI/UX 规范
- UI 库:ant-design-vue
- 业务组件复用:DictSelect / DictTag / DictSwitch / FormModal
- 列表规范:列定义 / 分页 / 搜索 / 操作列
- 表单规范:必填 / 校验 / 回填 / 联动
- 反馈规范:message / confirm / notification

## 8.验收标准
- 8.1 功能验收
- 8.2 性能验收
- 8.3 安全验收
- 8.4 文档交付

## 9.风险与依赖
| 风险 | 等级 | 缓解措施 | 负责人 |


# 规则
1. 章节顺序、命名、子节编号、表格列 **不得改动**。
2. 信息不全时,使用 `<待定>` 占位,**不要省略章节**。
3. 字段命名、字典类型、权限码命名遵循 mediation-platform 现有规范(case:manage:* / system:config:* / ai:model:*)。
4. 完成后输出文件路径与字节数。
```

### 2.4 一句话需求完成开发

参考[one-sentence-pipeline.md](one-sentence-pipeline.md)


## 3.测试与开发

### 3.1 执行全新开发

拷贝1生成的提示词，执行全新开发。

```
请实现【向量存储】CRUD + 导出功能。

【模块信息】
- 服务: AI(:8086)
- 页面: vectorStore/index.vue
- 权限: ai:vector-store:create/update/delete/export

【功能需求】
1. 存储列表(分页):存储名称、类型、地址、维度、状态、创建时间、操作
2. 搜索:存储名称、存储类型(Select)
3. 新增/编辑:名称(必填)、类型(必填)、地址(必填)、端口(必填)、维度(必填,默认1536)、API Key、状态
4. 存储类型 Tag 颜色映射
5. 维度输入校验(正整数)
6. 导出 Excel

【交付物清单】
- [ ] 前端 index.vue + VectorStoreFormModal.vue
- [ ] API vectorStore.ts
- [ ] 后端 Controller + Service + DO + Mapper
- [ ] 路由 + 菜单 + 权限码注册
- [ ] 通过 5.4.2 所有测试场景
```




### 3.2 /qa-dev 批量执行功能

**🚀 批量执行命令** 

 执行所有24个模块

```bash
/qa-dev --batch --file docs/scene/system-scene.md
```

**分批执行**

```bash
# SYS-01 ~ SYS-05
/qa-dev --batch --file docs/scene/system-scene.md --from SYS-01 --to SYS-05

# 仅邮件/短信模块
/qa-dev --batch --file docs/scene/system-scene.md --from SYS-07 --to SYS-17
```

**智能流程**

```bash
# 1. 先批量测试所有模块
/qa-dev --batch --test-only --file docs/scene/system-scene.md

# 2. 仅开发失败的模块
/qa-dev --batch --dev-only --failed --file docs/scene/system-scene.md

# 3. 重新验证
/qa-dev --batch --file docs/scene/system-scene.md
```


**批量执行流程**

```
┌────────────────────────────────────────────────────────────────────┐
│  1. 解析文件 → 提取24个模块                                        │
│  2. 过滤模块 (--from/--to/--failed)                              │
│  3. 逐个执行: 测试 → 开发 → 验证                                   │
│  4. 生成汇总报告                                                   │
└────────────────────────────────────────────────────────────────────┘
```

| AI IDE | 推荐模型 |
| ---- | ---- |
| cursor | composer-2.5 |
| qoder | Lite |
| CodeBuddy | Hy3 |

### 3.3 执行测试/开发

拷贝1生成的提示词，执行测试. 

```
/qa
打开 OntoGraph Console,执行完整的登录、注销与首页信息展示测试。

【前置操作】
1. 确认前端(:3000)和后端(:9090)均已启动
2. 确认浏览器无已登录 Cookie(建议用无痕/隐私模式)

---

【测试场景 1:登录页渲染】
1. 打开 http://localhost:3000/login
2. 验证页面正常渲染,无白屏、无 JS 错误
3. 验证包含以下元素:
  - Logo 区域(OntoGraph Console 标题)
  - 用户名输入框(username)
  - 密码输入框(password)
  - 登录按钮(Login / 登录)
  - 语言切换组件(如存在)
4. 验证表单布局美观、输入框对齐

预期结果:
✅ 页面 8 列完整
✅ Logo 标题显示
✅ 用户名/密码输入框存在
✅ 登录按钮可点击

---

【测试场景 2:正确账号登录】
1. 在用户名输入框输入:admin
2. 在密码输入框输入:admin123
3. 点击「登录」按钮
4. 验证:
  - API POST /auth/login 成功返回 token
  - 页面跳转到首页 /dashboard
  - Header 右上角显示用户昵称(admin)
  - 左侧 Sidebar 正确加载四大分组菜单
  - 控制台无 Error 级别报错

预期结果:
✅ 登录 API 返回成功
✅ 跳转到 /dashboard
✅ Header 显示用户名
✅ Sidebar 菜单完整渲染

---

【测试场景 3:错误密码登录】
1. 在用户名输入框输入:admin
2. 在密码输入框输入:wrongpassword
3. 点击「登录」按钮
4. 验证:
  - API POST /auth/login 返回错误(code≠0 或 HTTP 401)
  - 页面不跳转,停留在登录页
  - 显示错误提示信息(「用户名或密码错误」)

预期结果:
✅ 错误提示显示
✅ 页面不跳转
✅ 可重新输入

---

【测试场景 4:空账号登录(前端校验)】
1. 不填写任何内容,直接点击「登录」
2. 验证浏览器前端表单校验生效
3. 验证必填提示(如:请输入用户名)

预期结果:
✅ 前端表单校验生效
✅ 提示「请输入用户名」
✅ 不触发后端请求

---

【测试场景 5:注销功能】
1. 在首页右上角找到用户下拉菜单(点击头像或用户名)
2. 点击「注销」或「Logout」
3. 验证:
  - API POST /auth/logout 成功
  - 页面跳转到登录页 /login
  - localStorage/SessionStorage 中的 Token 被清除
  - 再次访问 /dashboard 被重定向到 /login

预期结果:
✅ 注销成功跳转
✅ Token 已清除
✅ 受限页面重定向正常

---

【测试场景 6:首页信息展示(Header + Sidebar)】
1. 登录成功后停留在首页 /dashboard
2. 验证 Header 组件:
  - Logo 区域(OntoGraph Console 文字/图标)可点击
  - 点击 Logo 返回首页
  - 右上角:语言切换器
  - 右上角:通知铃铛(带 Badge)
  - 右上角:用户名/头像下拉菜单(个人中心/注销)
3. 验证 Sidebar 组件:
  - 四大分组菜单:
    a. 「图谱管理」(含图谱列表/Graph IDE/时序历史/社区发现)
    b. 「数据管理」(含类管理/属性管理/约束管理/实体管理/边管理/导入/导出/法律知识图谱)
    c. 「工具」(含混合搜索/自定义指令/Prompt 管理)
    d. 「系统管理」(含用户管理/角色管理/菜单管理/参数配置/操作日志/系统监控)
  - 每个分组可折叠/展开
  - 当前激活菜单项高亮显示

预期结果:
✅ Header Logo 可点击
✅ 语言切换器存在
✅ 通知铃铛存在
✅ 用户下拉菜单正常
✅ Sidebar 四大分组完整
✅ 菜单可折叠/展开
✅ 当前页菜单高亮

---

【测试场景 7:首页内容展示(Dashboard)】
1. 停留在 /dashboard 页面
2. 验证 Dashboard 页面内容加载:
  - 页面标题/欢迎语
  - 统计数据卡片(如图谱数量、实体数量等)
  - 内容区域正常渲染
  - 无 Loading 卡死
  - 无 JS 错误

预期结果:
✅ Dashboard 页面完整渲染
✅ 无 JS 错误

---

【测试场景 8:未登录访问受限页(重定向)】
1. 清除浏览器 Cookie 和 Token
2. 直接访问 http://localhost:5173/dashboard
3. 验证页面自动重定向到 /login
4. 登录后验证正确跳转回 /dashboard

预期结果:
✅ 未登录时重定向到 /login
✅ 登录后正确跳转

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 用户登录 | POST | /auth/login |
| 用户注销 | POST | /auth/logout |
| 获取用户信息 | GET | /auth/info |
| 获取菜单树 | GET | /auth/menus |

---

【问题诊断】
- 登录页白屏 → 检查 Vue 路由配置 login 组件是否正确注册
- 登录后不跳转 → 检查 authStore.login() 是否正确处理 LoginResult.token
- Header 不显示用户名 → 检查 authApi.getInfo() 返回的 nickname/username 字段
- Sidebar 菜单不加载 → 检查 authApi.getMenus() 返回的 MenuItem[] 树结构
- 注销后仍能访问 → 检查路由守卫 beforeEach 是否正确拦截
- Token 不存储 → 检查 auth.ts login() 返回后是否写入 localStorage/sessionStorage
- 登录 API 404 → 检查后端 AuthController 是否映射 /auth/login
```

自动测试完毕，然后突出3个待修复问题：

 已知未修复问题

| 问题 | 说明 | 影响 |
|------|------|------|
| 动态菜单子项为空 | 后端 `/auth/menus` 返回扁平列表，前端按 `parentId` 分组后 children 为空 | 侧边栏展开后无子菜单（静态备用菜单在动态菜单加载后不显示） |
| 仪表盘菜单重复 | 静态 "Dashboard" + 动态 "nav.dashboard" 两项并存 | 侧边栏有两个 Dashboard 入口 |
| 登录页无语言切换组件 | 语言切换仅在 Header 中，登录页未集成 | 非中文用户无法在登录前切换语言 |

导出菜单sql数据到文件，给qoder,然后修正问题完成开发

**推荐模型**：  

| AI IDE | 推荐模型 |
| ---- | ---- |
| cursor | composer-2.5 |
| qoder | Lite |
| CodeBuddy | Hy3 |


## 4.场景开发SOP

### 4.1 基线脚手架开发
```
实现目标：
基于已知技术栈与要求，开发一个基线脚手架（baseline scaffold）并建立完整的自动化Pipeline,将实现过程写入 @docs/scene 下 jixian-pipeline.md。

具体要求：
1. 基线脚手架应包含完整的项目结构、配置文件、基础依赖、代码规范、测试框架、CI/CD模板等基础设施
2. 技术栈要求：现代全栈架构（如React/Vue前端 + Node.js/Java后端 + 数据库 + Docker + CI/CD）
3. 需要建立一个可复用的通用Pipeline，涵盖从需求分析到脚手架发布的完整流程

请整理一个通用的Pipeline实现以上目标，包括：
- Pipeline的各个节点（N1-N6）及其职责定义
- 每个节点的具体操作步骤
- 每个节点对应的提示词模板
- 节点间的输入输出关系
- Pipeline的执行顺序和依赖关系

可使用的Skills工具：
1. Superpowers - 用于头脑风暴、计划制定、TDD开发等创造性工作
2. gstack - 用于产品咨询、设计评审、工程审查、QA测试等
3. Get Shit Done (GSD) - 用于项目初始化、阶段规划、执行验证、发布管理等
4. OpenSpec - 用于需求规格化、变更提案、规范驱动开发等
5. awesome-design-md - 用于设计系统规范、UI组件标准、视觉一致性等

Pipeline需要能够：
- 自动化生成项目基础结构
- 集成代码质量检查工具
- 配置开发环境和依赖管理
- 设置测试框架和覆盖率检查
- 配置CI/CD流水线
- 生成项目文档模板
- 确保代码规范和最佳实践
```
[framework-pipeline.md](framework-pipeline.md)

### 4.2 一句话需求到前端原型

```
使用 这些skills：

1. superpower;
2.gstack
3.get-shit-done
4.OpenSpec
5.awesome-design-md 

目标：实现从一句话需求生成完整PRD文档，并根据文档按照最小级别子模块，生成 在ai coding 工具cursor 可以使用的提示词，包括完整测试提示词语开发提示词，结构参考@mediation-web/docs/scene/scene-template.md。
输出： pipeline实现的流程与各个节点的操作，包括提示词


范例提示词：
参照模板 以@mediation-web/docs/scene/scene-template.md 的结构与风格，生成一份完整的服务端到端浏览器自动化测试开发文档。模板的所有章节顺序、命名、子节、表格列、代码块结构、占位符语义都必须 1:1 对齐:
-   后端 @mediation-basic/mediation-module-system服务名：system 端口 8082
-   前端 views目录：@mediation-web/src/api/systemapi模块: @mediation-web/src/views/system

输出文件路径：`mediation-web/docs/scene/system-scene.md`
特别强调：System 服务测试模块 必须完成所有模块的测试提示词、开发提示词的编写
```

[one-sentence-pipeline.md](one-sentence-pipeline.md)

### 4.3 项目文档自动化
实现类似Qoder Repo-wkik的文档  
```
目标：从当前工程生成位于 @.qoder/repowiki/zh/content 的完整文档，参考其结构与目录，逆向整理出实现该目标的通用pipeline，包括各节点的具体操作与提示词。
输出： @docs/scene 下 docs-pipeline.md
具体要求：
1. 分析当前工程中 @.qoder/repowiki/zh/content 目录下的文档结构和内容组织方式
2. 识别文档生成的关键节点和流程步骤
3. 逆向设计一个通用的文档生成pipeline，包含：
   - 每个节点的具体操作内容
   - 每个节点对应的提示词模板
   - 节点间的依赖关系和数据流转
   - 文档质量控制标准

可使用的skills组合如下：
- superpowers: 用于头脑风暴、计划制定、TDD开发、系统化调试、代码审查等
- gstack: 用于产品咨询、设计评审、工程审查、QA测试、安全审计等
- get-shit-done: 用于项目管理、阶段执行、工作流编排
- OpenSpec: 用于规范驱动开发、提案制定、变更管理
- awesome-design-md: 用于设计系统规范、UI/UX指导

最终输出应包括：
- 详细的pipeline流程图或步骤说明
- 每个节点的输入/输出定义
- 操作指南和提示词模板
- 可复用的文档生成框架
```

[docs-pipeline.md](docs-pipeline.md)

### 4.4 线上已有Web项目复制

```
实现目标：
已知线上系统，包括登录账号密码，复制其功能实现一样系统；
整理一个通用的pipeline实现以上目标,包括各节点的操作与提示词，整理内容写入文档：@mediation-web/docs/scene  copy-web-pipeline.md。

具体要求：
1. 基于现有线上系统进行逆向工程分析，包括但不限于：
   - 分析系统功能模块和业务流程
   - 识别技术架构和数据流向
   - 提取API接口和数据结构
   - 分析用户界面和交互逻辑

2. 设计一个完整的开发pipeline，包含以下阶段：
   - 需求分析与规格制定
   - 系统架构设计
   - 数据库设计
   - 前后端开发
   - 测试验证
   - 部署上线

3. 每个pipeline节点需要包含：
   - 具体操作步骤
   - AI提示词模板
   - 预期输出产物
   - 质量验收标准

4. 可使用的skills：
   - superpowers: 用于头脑风暴、计划制定、TDD开发、系统化调试、代码审查等
   - gstack: 用于产品咨询、设计评审、工程审查、QA测试、安全审计等
   - get-shit-done: 用于项目管理、阶段执行、工作流编排
   - OpenSpec: 用于规范驱动开发、提案制定、变更管理
   - awesome-design-md: 用于设计系统规范、UI/UX指导

5. pipeline需要覆盖从需求分析到系统上线的完整生命周期，确保可复用性和可扩展性。

6. 最终输出应包含完整的pipeline文档，详细说明每个节点的操作方法和对应的AI提示词。
```

[copy-web-pipeline.md](copy-web-pipeline.md)
### 4.5 复制移动App

```
实现目标：
1. 分析现有的移动APP（包含登录、账号、密码等功能模块），全面理解其功能架构和实现逻辑
2. 使用uniapp框架开发一个功能相同的跨平台应用（支持iOS、Android、H5、小程序等多端）
3. 设计并实现一个完整的自动化Pipeline，涵盖从需求分析到部署发布的全流程

具体要求：
- 逆向分析目标APP的功能点、UI界面、交互逻辑和数据流程
- 基于uniapp框架实现相同功能，确保各端体验一致
- 设计Pipeline包含以下节点：需求分析(N1) → 架构设计(N2) → 代码生成(N3) → 功能实现(N4) → 测试验证(N5) → 部署发布(N6)
- 每个节点需定义具体操作步骤和AI提示词模板
- 将完整的Pipeline文档写入文件：mediation-web/docs/scene/copy-app-pipeline.md

可使用的Skills工具集：
1. Superpowers - 用于头脑风暴、TDD开发、系统化调试
2. gstack - 用于多角色评审、QA测试、工程管理
3. Get Shit Done (GSD) - 用于项目管理和阶段性执行
4. OpenSpec - 用于需求规范化和变更管理
5. awesome-design-md - 用于UI设计规范和视觉一致性

输出产物：
- copy-app-pipeline.md文档，包含完整的6节点Pipeline流程
- 每个节点的详细操作说明
- 每个节点对应的AI提示词模板
- Pipeline的执行验证和质量保障机制
```

[copy-app-pipeline.md](copy-app-pipeline.md)

### 4.6 升级已有源码工程【技术栈语言变更】

#### 4.6.1 开发语言变更 Python-->Java


#### 4.6.2 原技术栈升级
```
实现目标：
将已有工程进行现代化升级：
1. 后端 Java 8 工程升级到 Java 21，包括：
   - Java 版本从 8 升级到 21
   - Spring Framework 相关依赖包升级到兼容 Java 21 的最新版本
   - 相关第三方库和插件升级到兼容 Java 21 的版本
   - 代码语法和 API 调用适配 Java 21 特性

2. 前端 Vue 2 工程升级到 Vue 3 + Ant Design + Vite，包括：
   - Vue 从 2.x 升级到 3.x
   - 集成 Ant Design Vue 组件库
   - 构建工具从 Webpack 迁移到 Vite
   - 代码语法从 Options API 迁移到 Composition API
   - 相关依赖库升级到兼容 Vue 3 的版本

在 Cursor/QoDer 开发工具中，整理一个通用的自动化 pipeline，通过零编码方式实现以上升级目标。该 pipeline 需要包含：
- 各个升级节点的具体操作步骤
- 每个节点对应的 AI 提示词（prompts）
- 错误处理和回滚机制
- 升级前后对比验证步骤

将整理好的 pipeline 内容详细写入文档：@mediation-web/docs/scene/java-upgrade-pipeline.md
文档应包含完整的升级流程、技术栈变更说明、注意事项和最佳实践。
```
[java-upgrade-pipeline.md](java-upgrade-pipeline.md)


## 5. MemPalace 记忆共享

MemPalace 是一个本地优先（Local-first）的开源 AI 记忆系统，采用原文存储方式（verbatim storage），通过语义搜索实现记忆检索。它不进行总结、提取或改写，确保记忆的完整性和可追溯性。

**核心特性：**
- 原文存储：对话历史和代码内容以原始文本形式保存
- 可插拔后端：支持 ChromaDB、SQLite exact、Qdrant、pgvector 等多种向量存储
- MCP 协议集成：35 个 MCP 工具，支持 Claude Code、Codex、Cursor、Gemini 等主流 AI IDE
- Auto-save Hooks：自动保存机制，无需手动操作
- 性能优异：LongMemEval 基准测试 R@5 达到 96.6%（无需 LLM 调用）
- 完全本地化：数据默认存储在本地，无需 API Key

> **官方资源**：[GitHub 仓库](https://github.com/MemPalace/mempalace) | [官方文档](https://mempalaceofficial.com/) | [PyPI](https://pypi.org/project/mempalace/)
>
> **注意**：请认准官方域名 mempalaceofficial.com，其他域名均为仿冒站点。

### 5.1 MemPalace 安装与部署

#### 5.1.1 环境要求

| 组件 | 最低要求 | 推荐配置 |
|------|---------|---------|
| Python | 3.9+ | 3.11+ |
| 向量存储 | ChromaDB (默认) | ChromaDB / Qdrant / pgvector |
| 磁盘空间 | ~300 MB | ~500 MB (含缓存) |
| 内存 | 2GB RAM | 4GB RAM |

#### 5.1.2 安装方式

**方式一：使用 uv（推荐）**

`uv` 会在隔离环境中安装 CLI，避免依赖冲突：

```bash
# 安装
uv tool install mempalace

# 初始化项目
mempalace init ~/projects/myapp
```

**方式二：使用 pip**

```bash
# 仅在激活的虚拟环境中使用
python -m venv .venv && source .venv/bin/activate  # Linux/macOS
# 或
python -m venv .venv && .venv\Scripts\activate    # Windows

pip install mempalace
```

**方式三：使用 pipx**

```bash
pipx install mempalace
```

**方式四：从源码安装**

```bash
git clone https://github.com/MemPalace/mempalace.git
cd mempalace
uv sync --extra dev   # 或 pip install -e ".[dev]"
```

#### 5.1.3 Docker 部署

```bash
# 构建镜像（CPU 版本）
docker build -t mempalace .

# 运行 MCP 服务器（stdio 模式）
docker run -i --rm -v mempalace-data:/data mempalace

# 运行 CLI 命令
docker run --rm -v mempalace-data:/data -v /path/to/project:/work mempalace mine /work
docker run --rm -v mempalace-data:/data mempalace search "查询内容"

# GPU 加速版本
docker build -f Dockerfile.gpu -t mempalace:gpu .
docker run --gpus all -i --rm -v mempalace-data:/data mempalace:gpu
```

#### 5.1.4 存储后端配置

| 后端 | 说明 | 配置方式 |
|------|------|---------|
| ChromaDB | 默认后端，开箱即用 | 无需配置 |
| sqlite_exact | 本地精确向量，用于正确性验证 | `--backend sqlite_exact` |
| Qdrant | REST API 外部向量服务 | `MEMPALACE_QDRANT_URL=http://localhost:6333` |
| pgvector | PostgreSQL + pgvector 扩展 | `MEMPALACE_PGVECTOR_DSN=postgresql://localhost:5432/mempalace` |

**配置示例：**

```bash
# 使用 Qdrant 后端
export MEMPALACE_QDRANT_URL=http://localhost:6333
export MEMPALACE_QDRANT_API_KEY=your_api_key
mempalace mine ~/projects/myapp --backend qdrant

# 使用 pgvector 后端
export MEMPALACE_PGVECTOR_DSN=postgresql://localhost:5432/mempalace
pip install mempalace[pgvector]
mempalace mine ~/projects/myapp --backend pgvector
```

### 5.2 MemPalace 核心概念

MemPalace 采用宫殿记忆法的结构化存储体系：

```
组织层级：Palace（宫殿）
  ├── Wing（翅膀）→ 人员或项目
  │     └── Room（房间）→ 主题
  │           └── Drawer（抽屉）→ 原文内容块
  │
  ├── Hall（大厅）→ 分类标签
  │     ├── hall_facts      → 决策、已锁定的选择
  │     ├── hall_events     → 会议、里程碑、调试
  │     ├── hall_discoveries → 突破、新发现
  │     ├── hall_preferences → 习惯、偏好、观点
  │     └── hall_advice     → 建议和解决方案
  │
  └── Tunnel（隧道）→ 跨 Wing 连接
```

| 概念 | 说明 | 示例 |
|------|------|------|
| **Wing** | 顶层组织单元，代表人员或项目 | `wing_kai`、`wing_driftwood`、`project-api` |
| **Room** | Wing 内的主题分类 | `auth-migration`、`graphql-switch`、`ci-pipeline` |
| **Hall** | 记忆的内容类型 | `hall_facts`、`hall_events` 等 |
| **Drawer** | 存储的原文文本块 | 完整的对话、代码片段、决策记录 |
| **Tunnel** | 跨 Wing 的关联连接 | API 设计与数据库 Schema 的关联 |

### 5.3 MemPalace 快速上手

#### 5.3.1 初始化 Palace

```bash
# 初始化项目
mempalace init ~/projects/myapp
# 或在当前目录
mempalace init .
```

初始化过程会：
- 扫描项目目录结构
- 检测人员和项目信息
- 创建对应的 Wing 和 Room
- 确保 `~/.mempalace/` 配置目录存在

#### 5.3.2 挖掘数据（Mining）

```bash
# 挖掘项目文件（代码、文档、笔记）
mempalace mine ~/projects/myapp

# 挖掘对话导出（Claude、ChatGPT、Slack 等）
mempalace mine ~/chats/ --mode convos

# 带自动分类的对话挖掘
mempalace mine ~/chats/ --mode convos --extract general
```

**挖掘模式说明：**

| 模式 | 说明 | 输出 |
|------|------|------|
| `projects` | 代码和文档，自动检测 Room | Wing → Room 结构化内容 |
| `convos` | 对话导出，按交互对分块 | 原文对话记录 |
| `extract general` | 额外分类为决策、偏好、里程碑、问题、情感 | 带标签的分类内容 |

#### 5.3.3 搜索

```bash
# 语义搜索
mempalace search "为什么我们切换到了 GraphQL"

# 加载上下文（新会话启动）
mempalace wake-up
```

#### 5.3.4 后续使用

完成初始配置后，无需手动运行命令。AI 工具会通过 MCP 协议自动调用 MemPalace：

- 提问："上个月我们关于认证做了什么决定？"
- AI 自动调用 `mempalace_search`
- 获取原文结果并回答

### 5.4 Claude Code 中的 MCP 集成

Claude Code 支持通过 Auto-save Hooks 实现自动保存功能。

#### 5.4.1 安装 Auto-save Hooks

**步骤 1：编辑配置文件**

在 `.claude/settings.local.json` 中添加：

```json
{
  "hooks": {
    "Stop": [{
      "matcher": "*",
      "hooks": [{
        "type": "command",
        "command": "/absolute/path/to/hooks/mempal_save_hook.sh",
        "timeout": 30
      }]
    }],
    "PreCompact": [{
      "hooks": [{
        "type": "command",
        "command": "/absolute/path/to/hooks/mempal_precompact_hook.sh",
        "timeout": 30
      }]
    }]
  }
}
```

**步骤 2：赋予执行权限**

```bash
chmod +x hooks/mempal_save_hook.sh hooks/mempal_precompact_hook.sh
```

#### 5.4.2 Hook 工作机制

| Hook | 触发时机 | 行为 |
|------|---------|------|
| **Save Hook** | 每 15 次人类消息 | 阻止 AI，指示其保存关键主题/决策/引用到 Palace |
| **PreCompact Hook** | 上下文压缩前 | 紧急保存——强制 AI 在丢失上下文前保存一切 |

#### 5.4.3 配置选项

编辑 `mempal_save_hook.sh` 修改以下配置：

```bash
SAVE_INTERVAL=15      # 每次保存间隔的消息数
STATE_DIR=~/.mempalace/hook_state/  # Hook 状态存储目录
MEMPAL_DIR=           # 可选：设置为对话目录以自动运行 mempalace mine
```

### 5.5 Cursor IDE 中的 MCP 集成

Cursor IDE 通过插件和 Auto-save Hooks 实现 MemPalace 集成。

#### 5.5.1 安装 Cursor 插件

1. 克隆 MemPalace 仓库
2. 将 `.cursor-plugin/` 文件夹复制到 `~/.cursor/plugins/local/mempalace`

#### 5.5.2 安装 Auto-save Hooks

使用安装脚本（推荐）：

```bash
# 预览变更（不写入）
hooks/cursor/install.sh --scope user --dry-run

# 用户级安装（全局生效）
hooks/cursor/install.sh --scope user

# 项目级安装（仅当前项目）
hooks/cursor/install.sh --scope project --target /path/to/your/repo
```

**手动安装（用户级）：**

在 `~/.cursor/hooks.json` 中添加：

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [
      { "command": "/absolute/path/to/hooks/cursor/mempal_wake_hook_cursor.sh" }
    ],
    "stop": [
      {
        "command": "/absolute/path/to/hooks/cursor/mempal_save_hook_cursor.sh",
        "loop_limit": 1
      }
    ],
    "preCompact": [
      { "command": "/absolute/path/to/hooks/cursor/mempal_precompact_hook_cursor.sh" }
    ]
  }
}
```

赋予执行权限：

```bash
chmod +x hooks/cursor/mempal_save_hook_cursor.sh \
         hooks/cursor/mempal_precompact_hook_cursor.sh \
         hooks/cursor/mempal_wake_hook_cursor.sh
```

#### 5.5.3 Cursor Hooks 三层召回机制

| 层级 | 触发时机 | 作用域 | 获取方式 |
|------|---------|--------|---------|
| `sessionStart` | 新对话打开时 | 注入 wing 范围的召回上下文 | Hook 页面 |
| `mempalace-recall` skill | 请求匹配描述或手动附加 | 完整搜索-回答协议 | Cursor 插件 skills/ |
| Recall rule | Cursor 匹配器判断为召回相关 | 简短提示先搜索 | 插件 rules/ |

#### 5.5.4 Cursor Hook 配置选项

| 环境变量 | 默认值 | 说明 |
|---------|-------|------|
| `MEMPAL_SAVE_INTERVAL` | 15 | Stop 事件间的保存间隔 |
| `MEMPAL_CURSOR_SILENT` | 0 | 设为 1 可完全禁止 followup_message |
| `MEMPAL_STATE_DIR` | `~/.mempalace/hook_state/` | Hook 计数器、日志目录 |
| `MEMPAL_DISABLE_HOOK` | 0 | 紧急禁用开关，设为 1 禁用所有 hooks |

### 5.6 Codex CLI 中的 MCP 集成

Codex CLI 支持与 Claude Code 相同的 Auto-save Hooks 机制。

#### 5.6.1 安装 Auto-save Hooks

在 `.codex/hooks.json` 中添加：

```json
{
  "Stop": [{
    "type": "command",
    "command": "/absolute/path/to/hooks/mempal_save_hook.sh",
    "timeout": 30
  }],
  "PreCompact": [{
    "type": "command",
    "command": "/absolute/path/to/hooks/mempal_precompact_hook.sh",
    "timeout": 30
  }]
}
```

赋予执行权限：

```bash
chmod +x hooks/mempal_save_hook.sh hooks/mempal_precompact_hook.sh
```

### 5.7 Qoder 中的 MCP 集成

Qoder 是华为推出的 AI 代码开发工具，通过 MCP 协议集成 MemPalace 可实现跨项目记忆共享。

#### 5.7.1 MCP 客户端配置

Qoder 支持通过 MCP 协议连接外部服务。首先需要启用 Qoder 的 MCP 支持：

1. 打开 Qoder 设置（`Ctrl + ,` 或 `Cmd + ,`）
2. 导航至「扩展」或「Plugins」选项卡
3. 搜索并安装「MCP Client」插件（如有）
4. 重启 Qoder 使插件生效

#### 5.7.2 添加 MemPalace MCP 服务器

在 Qoder 的 MCP 配置文件中添加 MemPalace 配置：

**Windows 配置文件路径：** `C:\Users\<用户名>\.qoder\mcp.json`

**macOS/Linux 配置文件路径：** `~/.qoder/mcp.json`

```json
{
  "mcpServers": {
    "mempalace": {
      "type": "stdio",
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "mempalace-data:/data", "mempalace"],
      "env": {}
    }
  }
}
```

**使用本地 Python 版本：**

```json
{
  "mcpServers": {
    "mempalace": {
      "type": "stdio",
      "command": "mempalace",
      "args": ["mcp", "run"],
      "env": {}
    }
  }
}
```

#### 5.7.3 Qoder 中使用 MemPalace

**通过命令面板操作：**

1. 按 `Ctrl + Shift + P` 打开命令面板
2. 输入 `MCP: Search MemPalace` 进行语义搜索
3. 输入 `MCP: Add Drawer` 添加记忆

**常用 MCP 工具：**

| 工具 | 功能 | 使用方式 |
|------|------|---------|
| `mempalace_search` | 语义搜索记忆 | 在命令面板输入搜索关键词 |
| `mempalace_add_drawer` | 添加原文内容 | 选中代码后调用 |
| `mempalace_mine` | 挖掘项目文件 | 在项目目录执行 |
| `mempalace_status` | 查看 Palace 状态 | 查看记忆统计 |

#### 5.7.4 Qoder 调试与日志

```bash
# 查看 MemPalace Docker 日志
docker logs mempalace

# 重新启动 MCP 服务器
# 在 Qoder 中按 Ctrl + Shift + P，输入 "MCP: Restart Server"

# 检查 MCP 连接状态
# 在 Qoder 中按 Ctrl + Shift + P，输入 "MCP: Show Status"
```

### 5.8 CodeBuddy 中的 MCP 集成

CodeBuddy 是腾讯推出的 AI 代码助手，支持通过 MCP 协议扩展功能。

#### 5.8.1 启用 MCP 支持

1. 打开 CodeBuddy 设置
2. 导航至「高级设置」→「MCP 配置」
3. 启用「MCP 客户端」选项
4. 重启 CodeBuddy

#### 5.8.2 配置 MemPalace 连接

**配置文件路径：**

**Windows：** `C:\Users\<用户名>\.codebuddy\mcp_config.json`

**macOS/Linux：** `~/.codebuddy/mcp_config.json`

```json
{
  "mcp": {
    "enabled": true,
    "servers": {
      "mempalace": {
        "type": "stdio",
        "command": "docker",
        "args": ["run", "-i", "--rm", "-v", "mempalace-data:/data", "mempalace"],
        "capabilities": [
          "memory.store",
          "memory.search",
          "memory.mine",
          "context.inject"
        ]
      }
    }
  }
}
```

**使用本地 Python 版本：**

```json
{
  "mcp": {
    "enabled": true,
    "servers": {
      "mempalace": {
        "type": "stdio",
        "command": "mempalace",
        "args": ["mcp", "run"],
        "capabilities": [
          "memory.store",
          "memory.search",
          "memory.mine",
          "context.inject"
        ]
      }
    }
  }
}
```

#### 5.8.3 CodeBuddy 中的 MemPalace 操作

CodeBuddy 通过侧边栏面板提供 MemPalace 交互功能：

| 功能 | 说明 |
|------|------|
| **记忆面板** | 显示当前项目的记忆列表 |
| **快速搜索** | 在侧边栏顶部提供搜索框 |
| **智能补全** | 基于记忆库提供代码补全建议 |
| **上下文注入** | 自动将相关记忆注入到 AI 对话上下文 |

**通过聊天命令使用：**

```
@mem search "API 设计规范"
@mem add "选中的代码片段"
@mem mine ./src
@mem status
```

#### 5.8.4 CodeBuddy 高级配置

**配置记忆可见性：**

```json
{
  "mcp": {
    "enabled": true,
    "servers": {
      "mempalace": {
        "type": "stdio",
        "command": "mempalace",
        "args": ["mcp", "run"],
        "settings": {
          "default_wing": "codebuddy-project",
          "auto_recall": true,
          "recall_threshold": 0.7
        }
      }
    }
  }
}
```

| 设置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `default_wing` | string | - | 默认使用的 Wing 名称 |
| `auto_recall` | boolean | true | 自动注入相关记忆到上下文 |
| `recall_threshold` | number | 0.7 | 记忆召回相似度阈值 |

#### 5.8.5 故障排查

| 问题 | 可能原因 | 解决方案 |
|------|---------|---------|
| MCP 连接失败 | Docker 未运行 | 启动 Docker Desktop |
| 命令无响应 | MCP 服务器未加载 | 重启 CodeBuddy，检查配置文件 |
| 搜索无结果 | 尚未挖掘数据 | 运行 `mempalace mine <path>` |
| 权限错误 | 配置文件路径错误 | 确认配置文件存在且格式正确 |

**验证 MCP 连接：**

在 CodeBuddy 中打开终端，运行：

```bash
# 测试 MCP 服务器
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | docker run -i --rm -v mempalace-data:/data mempalace
```

### 5.9 MCP 服务器配置

#### 5.9.1 MCP JSON 配置

将以下配置添加到 MCP 客户端配置文件：

**通用配置（Docker）：**

```json
{
  "mcpServers": {
    "mempalace": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "mempalace-data:/data", "mempalace"]
    }
  }
}
```

**本地 Python 版本：**

```json
{
  "mcpServers": {
    "mempalace": {
      "command": "mempalace",
      "args": ["mcp", "run"]
    }
  }
}
```

#### 5.9.2 MCP 工具列表

MemPalace 提供 35 个 MCP 工具，完整列表请参考 [官方文档](https://mempalaceofficial.com/reference/mcp-tools)。

**Palace 读取工具：**

| 工具 | 功能 |
|------|------|
| `mempalace_status` | Palace 概览 |
| `mempalace_list_wings` | 列出所有 Wings |
| `mempalace_list_rooms` | 列出 Wing 内的 Rooms |
| `mempalace_search` | 语义搜索 |
| `mempalace_check_duplicate` | 检查重复内容 |

**Palace 写入工具：**

| 工具 | 功能 |
|------|------|
| `mempalace_add_drawer` | 添加原文内容 |
| `mempalace_checkpoint` | 批量保存会话 |
| `mempalace_mine` | 挖掘目录 |
| `mempalace_delete_drawer` | 删除抽屉 |
| `mempalace_sync` | 同步清理 |

**知识图谱工具：**

| 工具 | 功能 |
|------|------|
| `mempalace_kg_query` | 查询实体关系 |
| `mempalace_kg_add` | 添加事实 |
| `mempalace_kg_invalidate` | 失效事实 |
| `mempalace_kg_timeline` | 时间线视图 |

### 5.10 团队开发配置

MemPalace 原生支持本地优先，但在团队协作场景下需要额外配置。

#### 5.10.1 团队数据存储架构

```
个人 Palace (本地)          团队 Palace (共享)
~/.mempalace/              共享存储 (NFS/S3/Git)
    ├── wing_personal      ├── wing_team-shared
    └── wing_project       └── wing_project
```

#### 5.10.2 共享 Palace 配置

**方案一：使用 Git 仓库共享**

```bash
# 创建共享 palace 仓库
git init --bare ~/.mempalace/team-shared.git

# 添加共享 remote
git -C ~/.mempalace remote add team ssh://team@git-server/mempalace.git

# 共享记忆
git -C ~/.mempalace push team main

# 获取团队记忆
git -C ~/.mempalace pull team main
```

**方案二：使用 NAS/SMB 共享**

```bash
# 挂载共享存储
mount -t cifs //nas-server/mempalace /mnt/nas-mempalace -o username=team

# 链接到本地
ln -s /mnt/nas-mempalace ~/.mempalace/shared
```

**方案三：使用 S3 兼容存储**

```bash
# 配置 S3 后端
export MEMPALACE_S3_BUCKET=team-mempalace
export MEMPALACE_S3_ENDPOINT=https://s3.example.com

# 同步到 S3
mempalace sync --remote s3://team-mempalace
```

#### 5.10.3 团队协作命令

```bash
# 标记为团队记忆
mempalace add-drawer \
  --wing "team-shared" \
  --room "api-design" \
  --content "统一 RESTful API 规范 v2.0"

# 跨项目检索
mempalace search --wing "team-shared" --query "数据库设计规范"

# 创建跨 Wing 隧道
mempalace create-tunnel \
  --source-wing "project-frontend" \
  --source-room "component-lib" \
  --target-wing "project-backend" \
  --target-room "api-design" \
  --label "前后端组件规范同步"
```

#### 5.10.4 团队命名规范建议

| Wing 命名 | 说明 | 示例 |
|-----------|------|------|
| `wing-{project}` | 项目专用 | `wing-payment`、`wing-auth` |
| `wing-{person}` | 人员专用 | `wing-kai`、`wing-priya` |
| `wing-team-{name}` | 团队共享 | `wing-team-backend`、`wing-team-infra` |

#### 5.10.5 团队使用工作流

**新成员加入流程：**

```bash
# 1. 安装 MemPalace
uv tool install mempalace

# 2. 克隆团队 palace（使用 Git）
git clone ssh://git-server/team/mempalace.git ~/.mempalace/team

# 3. 合并到本地
git -C ~/.mempalace remote add personal ~/.mempalace
git -C ~/.mempalace pull personal main

# 4. 初始化项目
mempalace init ~/projects/current-project

# 5. 挖掘项目数据
mempalace mine ~/projects/current-project --mode projects

# 6. 搜索团队知识
mempalace search --wing "wing-team-backend" --query "API 规范"
```

**团队记忆贡献流程：**

```bash
# 1. 贡献有价值的技术决策
mempalace add-drawer \
  --wing "wing-team-backend" \
  --room "decisions" \
  --content "采用 PostgreSQL 作为主数据库，原因：1)成熟稳定 2)pgvector 支持 3)团队熟悉"

# 2. 创建跨项目隧道
mempalace create-tunnel \
  --source-wing "wing-payment" \
  --source-room "database-design" \
  --target-wing "wing-team-backend" \
  --target-room "decisions"

# 3. 推送到团队仓库
git -C ~/.mempalace add .
git -C ~/.mempalace commit -m "Add PostgreSQL decision record"
git -C ~/.mempalace push team main
```

### 5.11 性能基准与调优

#### 5.11.1 基准测试结果

| 基准 | 指标 | 得分 | 备注 |
|------|------|------|------|
| LongMemEval (raw) | R@5 | **96.6%** | 无 LLM，无需 API Key |
| LongMemEval (Hybrid v4) | R@5 | **98.4%** | 50 题调优后泛化 |
| LongMemEval (Hybrid + LLM) | R@5 | ≥99% | 任何可用 LLM |
| LoCoMo (session) | R@10 | 60.3% | 1986 题 |
| LoCoMo (hybrid v5) | R@10 | 88.9% | 同集合 |
| ConvoMem | Avg recall | 92.9% | 250 项 |
| MemBench | R@5 | 80.3% | 8500 项 |

#### 5.11.2 调优建议

```bash
# 使用更精确的后端
mempalace mine ~/projects --backend sqlite_exact

# 启用混合搜索
mempalace search "query" --mode hybrid --rerank

# 使用更大 embedding 模型
python -m mempalace.onboarding
```

### 5.12 故障排查

| 问题 | 可能原因 | 解决方案 |
|------|---------|---------|
| 安装失败 | PEP 668 错误 | 使用 `uv tool install` 或 `pipx install` |
| MCP 连接失败 | 服务未启动 | 运行 `mempalace mcp run` 或启动 Docker 容器 |
| 搜索无结果 | 尚未挖掘数据 | 先运行 `mempalace mine <path>` |
| Hook 不触发 | 权限问题 | 确保 hook 脚本有执行权限 `chmod +x` |
| 索引过期 | 外部脚本修改了 palace | 运行 `mempalace reconnect` 刷新索引 |
| Docker 挂载失败 | 路径问题 | 使用绝对路径，确保 volume 存在 |

**调试日志：**

```bash
# Claude/Codex Hook 日志
cat ~/.mempalace/hook_state/hook.log

# Cursor Hook 日志
cat ~/.mempalace/hook_state/cursor_hook.log

# 详细调试
export MEMPAL_VERBOSE=1
mempalace mine ~/projects/myapp
```

### 5.13 相关资源

| 资源 | 链接 |
|------|------|
| GitHub 仓库 | https://github.com/MemPalace/mempalace |
| 官方文档 | https://mempalaceofficial.com/ |
| PyPI 包 | https://pypi.org/project/mempalace/ |
| Getting Started | https://mempalaceofficial.com/guide/getting-started.html |
| MCP 工具参考 | https://mempalaceofficial.com/reference/mcp-tools |
| The Palace 概念 | https://mempalaceofficial.com/concepts/the-palace |
| Cursor Hooks | https://mempalaceofficial.com/guide/cursor-hooks |
| Auto-Save Hooks | https://mempalaceofficial.com/guide/hooks |


## 6.文档编写

### 6.1 产品研究与设计  

#### 6.1.1 新产品设计：  
设计提示词：  
```
/brainstorming  基于AI技术开发面向金融行业【证券与加密货币】的投资与分析解决方案，目标用户包括投资者、基金、证券机构等。请参考类似OpenClaw的龙虾技术方案，结合以下GitHub项目和当前工程：
- https://github.com/agentscope-ai/CoPaw.git
- https://github.com/higress-group/hiclaw.git
- https://github.com/brokermr810/QuantDinger 
- https://github.com/Fincept-Corporation/FinceptTerminal 
- https://github.com/TauricResearch/TradingAgents 
- https://github.com/wbh604/UZI-Skill 
- https://github.com/muxuuu/serenity-skill 
- https://github.com/agi-now/buffett-skills 

为我制定详细的技术实施方案，并进行市场前景分析。

具体要求：
1. 技术方案应包含：系统架构设计、核心技术组件、AI模型选择、数据处理流程、用户交互界面等
2. 方案需考虑金融行业的特殊性：合规、投资模拟、投资预测、企业/投资账号数据图关系，明星企业供应链关系等专业需求
3. 应考虑实现的功能不会被基础大模型的发展所替代：包括skill等新技术，基础模型新功能等
4. 投资分析功能应支持：数据分析、模拟仿真、趋势预测、情绪预测、企业、账号关系分析等
5. 市场前景分析需涵盖：目标市场规模、竞争格局、商业模式、盈利预测、风险评估等
6. 结合当前AI技术发展趋势，提出可行的实施路线图和关键里程碑
写一份方案在  @docs/bp 

```

**Cursor 模型推荐 Claude-opus-4.8-thinging-high/GPT5.5 模型[高端模型]**  
**Qoder 模型推荐 QWen3.7-MAX 模型[高端模型]**  

#### 6.1.2 产品完整文档：  
紧接编写： 产品完整文档

```
A. 编写商业计划书 BP 50-100 页完整 BP
B. 编写投资人简报 10-15 页 PPT
C. 编写产品白皮书 技术 + 业务 30-50 页
D. 编写技术架构白皮书 30-50 页技术架构
E. 编写 MVP 产品 PRD 教学版 1.0 详细 PRD
F. 编写融资材料 商业模型 + 财务模型 + 股权设计
G. 编写组织设计 团队 + 薪酬 + 期权 + 招聘
H. 编写合规手册 全法规 + 措施 + 流程
深入编写以上文档，写入  @docs/product  目录下。
```
### 6.2 市场调研

#### 6.2.1  市场规模和总可寻址市场分析

```
你是麦肯锡级别的市场分析师。我需要针对【你的行业/产品】的总可寻址市场（TAM）分析。

请提供：
• 自上而下方法：从全球市场到细分市场
• 自下而上方法：从单位经济学×潜在客户计算
• TAM、SAM、SOM详细分解
• 未来5年增长率预测（CAGR）
• 每项估计背后的关键假设

格式为投资者就绪的市场规模幻灯片。
```
#### 6.2.2  竞争格局深度分析


```
你是贝恩公司的高级战略顾问。我需要【你的行业】的完整竞争格局分析。

请提供：
• 直接竞争者：按市场份额、收入、融资排名前10
• 间接竞争者：5家可能进入市场的相邻公司
• 各竞争者分析：定价模式、核心功能、目标受众、优劣势、最新战略举措
• 市场定位图及竞争护城河分析

格式为结构化竞争情报报告。
```

#### 6.2.3 客户画像与细分

```
你是世界级消费者研究专家。我需要为【你的产品/服务】创建深度客户画像。

请构建4个详细画像，每个包含：
• 人口统计学：年龄、收入、教育、地点、职务
• 心理特征：价值观、信念、生活方式、性格特征
• 痛点：日常经历的5大挫折
• 购买行为：如何发现、评估、购买产品
• 媒体消费：在线和离线时间分配
• 价格敏感性：各细分市场的支付意愿

还需提供：细分市场规模和优先级矩阵。
```

#### 6.2.4 行业趋势分析

```

你是高盛研究部的高级分析师。我需要【你的行业】板块的全面趋势报告。

请提供：
• 宏观趋势：塑造该行业的5大全球力量
• 微观趋势：过去12个月的7个新兴模式
• 技术颠覆：什么新技术改变游戏规则及何时主流化
• 监管变化：即将出台的法规或政策变化
• 消费者行为变化：购买偏好如何演变
• 投资信号：聪明钱流向何处（VC交易、并购、IPO）

格式为趋势情报简报，各趋势附影响评分。
```

#### 6.2.5  SWOT + 波特五力分析

```
你是哈佛商学院战略教授。我需要【你的公司/产品】的SWOT和波特五力组合分析。

SWOT分析：
• 优势：7个内部优势及证据
• 劣势：7个内部限制及诚实评估
• 机会：7个可利用的外部因素
• 威胁：7个可能伤害的外部因素

波特五力分析：
• 供应商势力、买方势力、竞争强度
• 替代品威胁、新进入者威胁

各要素评分（1-10）及行业整体吸引力评分。
```

#### 6.2.6 定价策略分析

```
你是与财富500强公司合作过的定价策略顾问。我需要【你的产品/服务】的全面定价分析。

请提供：
• 竞争对手定价审计：市场地图与定价比较
• 基于价值的定价模型：根据客户价值计算价格
• 成本加成分析：从成本结构确定底线价格
• 价格需求弹性估算
• 心理定价策略：锚定、魅力定价、诱饵策略
• 分层推荐：3个定价档次与功能分配
• 折扣策略：何时折扣、折扣幅度、针对对象

格式为定价策略方案附具体美元建议。
```

#### 6.2.7 上市战略

```
你是启动过20+产品的首席战略官。我需要【你的产品】的完整上市计划。

请提供：
• 上市分阶段：预上市（60天）、上市（第1周）、上市后（90天）
• 渠道策略：按预期ROI排名前7个获客渠道
• 信息框架：核心价值主张、3个支撑信息、证明点
• 内容策略：漏斗各阶段所需内容
• 合作伙伴机会：5个加速增长的战略伙伴
• 预算分配：【预算】跨渠道的分配
• KPI框架：10个关键指标及目标基准

格式为可操作的上市手册。
```

#### 6.2.8 客户旅程地图

```
你是顶级咨询公司的客户体验战略师。我需要【你的产品/服务】的完整客户旅程地图。

请绘制客户生命周期各阶段：
• 认知：首次发现？什么触发搜索？
• 考虑：他们比较什么？需要什么信息？
• 决策：什么让他们转化？什么阻止他们？
• 入职：前7天体验如何影响留存？
• 参与：什么让他们回访？激活时刻？
• 忠诚：什么将用户变成倡导者？
• 流失：为什么离开？早期警告信号？

格式为详细旅程地图附情感曲线描述。
```

#### 6.2.9 财务建模与单位经济学

```
你是高增长初创公司的财务副总裁。我需要【你的业务】的完整单位经济学和财务模型。

请提供：
• 单位经济学细分：CAC、LTV、LTV:CAC比率、回收期
• 毛利率和贡献利润分析
• 3年财务预测：第1年每月、第2-3年每季度
• 成本结构细分：固定vs变动成本
• 盈亏平衡分析、现金流预测、敏感性分析
• 关键假设表及基准对比

格式为财务模型摘要附清晰表格和公式。
```

#### 6.2.10 风险评估与情景规划

```
你是德勤的风险管理合伙人。我需要【你的业务/项目】的全面风险分析和情景规划。

请列出15项风险：
• 市场风险、运营风险、财务风险、监管风险、声誉风险

各风险提供：
• 概率评分（1-5）、影响严重程度（1-5）、风险得分
• 早期预警指标、缓解策略、应急计划

情景规划：
• 最好情况、基础情况、最坏情况、黑天鹅事件
• 各情景的收入影响、时间表和战略应对

格式为风险矩阵优先级排序的执行报告。
```

#### 6.2.11 市场进入与扩张战略

```
你是协助公司进入30+市场的全球扩张战略师。我需要【你的业务】进入【目标市场/地理/细分】的市场进入分析。

请提供：
• 市场吸引力评分：市场规模、增长率、竞争强度、监管环境评分
• 进入方式分析：直接进入、合资、并购、许可、数字优先方式的优劣
• 本地化需求：产品适配、定价调整、文化营销、法规合规
• 12个月进入路线图：月度行动计划与里程碑
• 投资需求：预算估算及资源分配

格式为可执行的扩张蓝图附成功指标。
```

#### 6.2.12 执行战略综合（终极提示词）

```

你是麦肯锡的资深合伙人向CEO汇报。我需要将【你的业务】所有方面综合成一项战略建议。

请提供：
• 执行摘要：CEO可2分钟读完的3段战略概览
• 当前状况：业务现状（诚实评估）
• 战略选项：3条不同路径，各含预期成果、投资、时间表、风险
• 推荐战略及理由
• 优先举措：未来90天排名前5个最高影响力行动
• 资源需求：人员、资金、工具

格式为麦肯锡风格战略甲板附清晰建议和后续步骤。
```









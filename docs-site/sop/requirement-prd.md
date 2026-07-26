# PRD → 子模块 → 测试/开发提示词 — 一体化 Pipeline

> 本文档定义**从一句话需求生成完整 PRD,再到最小子模块级测试/开发提示词**的可复用 Pipeline。
> 输出文档结构 **1:1 对齐** `mediation-web/docs/scene/scene-template.md`。
> 适用于 Cursor / Claude Code / gstack `/qa` / Superpowers brainstroming / OpenSpec / awesome-design-md 等 AI 编码工作流。

---

## 0.Pipeline 总览

```
┌──────────────────────────────────────────────────────────────────────┐
│  输入:一句话需求 (1~3 句中文,例:"做一个案件登记页面")                     │
└──────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
        ┌─────────────────────────────────────────────┐
        │ N1 — 一句话需求 → 完整 PRD 文档              │  prompt:  N1.md
        │ 产物: mediation-web/docs/scene/{name}-prd.md│
        └─────────────────────────────────────────────┘
                                  │
                                  ▼
        ┌─────────────────────────────────────────────┐
        │ N2 — PRD → 最小子模块清单 (M1..Mn)           │  prompt:  N2.md
        │ 产物: {name}-prd.md 内嵌「子模块拆分表」      │
        └─────────────────────────────────────────────┘
                                  │
                                  ▼
        ┌─────────────────────────────────────────────┐
        │ N3 — 每个子模块 → 测试提示词 (Browser E2E)  │  prompt:  N3.md
        │ 产物: {name}-scene.md 第 5 章所有子模块       │
        └─────────────────────────────────────────────┘
                                  │
                                  ▼
        ┌─────────────────────────────────────────────┐
        │ N4 — 每个子模块 → 开发提示词 (Cursor IDE)    │  prompt:  N4.md
        │ 产物: {name}-scene.md 第 5 章每个 5.x.3       │
        └─────────────────────────────────────────────┘
                                  │
                                  ▼
        ┌─────────────────────────────────────────────┐
        │ N5 — 文档合并 + 第 6/7/8 章生成              │  prompt:  N5.md
        │ 产物: 完整 {name}-scene.md                   │
        └─────────────────────────────────────────────┘
                                  │
                                  ▼
        ┌─────────────────────────────────────────────┐
        │ N6 — Cursor 中串行调用 N1→N2→N3→N4→N5       │  prompt:  N6.md
        │ 产物: mediation-web/docs/scene/{name}-scene.md│
        └─────────────────────────────────────────────┘
```

### 数据流约定

| 节点 | 输入文件 | 输出文件 | 关键依赖 |
|------|----------|----------|----------|
| N1 | 用户一句话 | `{name}-prd.md` | `scene-template.md` 第 1~8 章骨架 |
| N2 | `{name}-prd.md` | `{name}-prd.md` (追加子模块表) | 最小可测试单元原则 |
| N3 | `{name}-prd.md` + 子模块表 | `{name}-scene.md` 第 5.x.2 | gstack `/qa`、MCP Playwright |
| N4 | `{name}-prd.md` + 子模块表 | `{name}-scene.md` 第 5.x.3 | Cursor 编码工作流 |
| N5 | 全部 5.x 子模块 | `{name}-scene.md` 全文 | 第 6/7/8 章固定模板 |
| N6 | 用户一句话 | 调度 N1~N5 | Cursor 多步执行 |

### 通用占位符(下游所有提示词都使用)

```
{{NAME}}           服务短名,例: system / case / mediation
{{NAME_TITLE}}     中文标题,例: 系统管理 / 案件管理 / 人民调解
{{PORT}}           后端端口,例: 8082
{{FRONT_VIEW_DIR}} 前端 views 目录,例: mediation-web/src/views/{{NAME}}
{{API_DIR}}        前端 api 目录,例: mediation-web/src/api/{{NAME}}
{{BACKEND_DIR}}    后端基础路径,例: mediation-basic/mediation-module-{{NAME}}
{{SCENE_FILE}}     最终输出文件,例: mediation-web/docs/scene/{{NAME}}-scene.md
{{PRD_FILE}}       中间 PRD 文件,例: mediation-web/docs/scene/{{NAME}}-prd.md
```

---

## 1.N1 — 一句话需求 → 完整 PRD

### 1.1 节点职责

把任意一句业务需求扩展为结构化 PRD,包含 8 大章节骨架(对齐 `scene-template.md` 第 1~8 章)。

### 1.2 入口参数

- `{{USER_INPUT}}`:用户原始一句话
- `{{NAME}} {{NAME_TITLE}} {{PORT}}`:服务标识三元组

### 1.3 N1 提示词

````markdown
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

## 10.子模块拆分(由 N2 填充)
> 此章节由 N2 节点填充,本节点留空。
```

# 规则
1. 章节顺序、命名、子节编号、表格列 **不得改动**。
2. 不要生成第 10 章内容,留给 N2。
3. 信息不全时,使用 `<待定>` 占位,**不要省略章节**。
4. 字段命名、字典类型、权限码命名遵循 mediation-platform 现有规范(case:manage:* / system:config:* / ai:model:*)。
5. 完成后输出文件路径与字节数。
````

### 1.4 验收

- 文件存在,行数 ≥ 200
- 包含 10 个二级标题
- 至少 5 个 Epic、10 个 Story、30 个 AC

---

## 2.N2 — PRD → 最小子模块拆解

### 2.1 节点职责

把 PRD §3 的 Epic/Story 拆解为 **最小可测试子模块**(每个子模块对应 scene-template §5.x 的一节,内含 5.x.1 测试场景 / 5.x.2 测试提示词 / 5.x.3 开发提示词)。

### 2.2 N2 提示词

````markdown
# 角色
你是 mediation-platform 架构师,负责将 PRD 拆解为最小可测试子模块。

# 输入
- PRD 文件:`{{PRD_FILE}}`(N1 已生成)
- 服务短名:`{{NAME}}`

# 任务
读取 PRD §3 的所有 Epic/Story,**穷尽**拆解为子模块,输出 **子模块拆分表** 并 **追加到 PRD 第 10 章**。

# 拆解原则(最小级别子模块)
1. **单一职责**:一个子模块对应一个菜单页面或一个弹窗/抽屉。
2. **独立可测**:每个子模块至少包含 3 个测试场景,能被 §5 模板完整覆盖。
3. **CRUD 完备**:数据类子模块必须包含 增/改/查/删 四类场景。
4. **复合拆解**:Tab 页、抽屉、详情弹窗均为独立子模块(例:SYS-07 邮件管理 → 账号/模板/日志 三个子模块)。
5. **不能跨服务**:子模块边界必须落在 `{{NAME}}` 服务内。
6. **ID 编码**:`{{NAME|upper}}-NN`,NN 从 01 起,顺序与左侧菜单顺序一致。

# 输出(追加到 PRD §10)

```markdown
## 10.子模块拆分(由 N2 填充)

### 10.1 子模块清单
| ID | 子模块名称 | 菜单路径 | 前端目录 | API 文件 | 后端 Controller | 测试场景数 | 优先级 |
|----|-----------|----------|----------|----------|----------------|-----------|--------|
| {{NAME|upper}}-01 | <名称> | <一级>/<二级> | views/{{NAME}}/<dir>/index.vue | api/{{NAME}}/<file>.ts | <ControllerName> | N | P0/P1/P2 |

### 10.2 依赖关系图
{{NAME|upper}}-01 → {{NAME|upper}}-02 (字典数据依赖)
{{NAME|upper}}-03 → {{NAME|upper}}-01 (外键关联)

### 10.3 排序与分组
- 实施批次 1(P0 核心):01, 02, 05
- 实施批次 2(P1 增强):03, 04, 06
- 实施批次 3(P2 辅助):07, 08
```

# 规则
1. 每个子模块必须能在 PRD §3 中找到对应 Story。
2. 表格行数 = 子模块总数,通常 8 ~ 30。
3. 完成后输出:子模块总数、P0/P1/P2 分布、是否存在 Story 缺失(若有,列出)。
````

### 2.3 验收

- PRD §10 已填充
- 子模块 ID 唯一且连续
- 每个子模块有明确的菜单路径、前端目录、API 文件、Controller

---

## 3.N3 — 每个子模块 → 测试提示词(对齐 §5.x.2)

### 3.1 节点职责

为每个子模块生成 **端到端浏览器自动化测试提示词**,可直接喂给 gstack `/qa`、`/open-gstack-browser`、MCP Playwright。

### 3.2 N3 提示词

````markdown
# 角色
你是 mediation-platform QA Lead,负责编写浏览器端到端测试提示词。

# 输入
- PRD 文件:`{{PRD_FILE}}`(含 §10 子模块表)
- 模板骨架:`mediation-web/docs/scene/scene-template.md` 的 §5.x.2 结构
- 当前处理的子模块:`{{MODULE_ID}} = {{NAME|upper}}-NN`

# 任务
为子模块 `{{MODULE_ID}}` 生成完整的 **测试提示词**,严格遵循 scene-template §5.x.2 的代码块结构。

# 模板骨架(必须 1:1 对齐)

```markdown
### {{MODULE_ID}} <子模块中文名>

**页面路径**: 左侧菜单「<一级>」→「<二级>」  
**源码文件**: `{{FRONT_VIEW_DIR}}/<dir>/index.vue`, 必要时附 FormModal.vue / DetailModal.vue  
**API 文件**: `{{API_DIR}}/<file>.ts`  
**权限标识**: `<perm>:create`, `<perm>:update`, `<perm>:delete`, `<perm>:query`

#### {{MODULE_ID}}.1 测试场景

#### {{MODULE_ID}}.2 测试提示词

\`\`\`
/browser 或 /open-gstack-browser
打开<页面名>页面,执行完整的 <子模块名> 测试。

【前置操作】
1. 使用 admin/admin123 登录系统
2. 在左侧菜单点击「<一级>」→「<二级>」
3. 等待页面加载完成

---

【测试场景 1:<场景名>】
1. 步骤 1
2. 步骤 2
3. 验证点

预期结果:
✅ 验证 1
✅ 验证 2

---

【测试场景 2:<场景名>】
... (按需扩展至 N 个,通常 3~12 个)

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表分页 | GET | /admin-api/{{NAME}}/<resource>/page |
| 详情 | GET | /admin-api/{{NAME}}/<resource>/get?id=X |
| 创建 | POST | /admin-api/{{NAME}}/<resource>/create |
| 更新 | PUT | /admin-api/{{NAME}}/<resource>/update |
| 删除 | DELETE | /admin-api/{{NAME}}/<resource>/delete?id=X |

---

【问题诊断】
- <现象 1> → <检查点>
- <现象 2> → <检查点>
\`\`\`

#### {{MODULE_ID}}.3 开发提示词

(此节由 N4 填充)
```

# 规则
1. 测试场景数 = PRD §10 表中"测试场景数"列。
2. 每个场景必须包含 【测试场景 N】/ 步骤 / 预期结果 三段式。
3. API 端点表至少 4 行,完整覆盖 CRUD。
4. 问题诊断至少 3 条,覆盖常见白屏/无数据/表单不回填/权限丢失/Token 过期。
5. 代码块使用 ` ``` ` 包裹,不要混入语言标记。
6. 输出时在文档中**预留** {{MODULE_ID}}.3 标题,内容由 N4 填充。
````

### 3.3 验收

- 每个子模块 §5.x.2 包含完整测试提示词代码块
- 测试场景数 ≥ 3
- API 端点表完整

---

## 4.N4 — 每个子模块 → 开发提示词(对齐 §5.x.3)

### 4.1 节点职责

为每个子模块生成 **Cursor IDE 可直接使用**的开发提示词,驱动 AI 完成前后端实现。

### 4.2 N4 提示词

````markdown
# 角色
你是 mediation-platform 资深全栈架构师,负责编写 Cursor 可用的开发提示词。

# 输入
- PRD 文件:`{{PRD_FILE}}`
- 当前处理的子模块:`{{MODULE_ID}}`

# 任务
为子模块 `{{MODULE_ID}}` 生成完整的 **开发提示词**,严格遵循 scene-template §5.x.3 的代码块结构。

# 模板骨架(必须 1:1 对齐)

```markdown
#### {{MODULE_ID}}.3 开发提示词

\`\`\`
请基于以下信息,在 mediation-platform 仓库中实现【<子模块中文名>】模块。

【模块信息】
- 服务: <Name>(:<Port>)
- 页面路径: 左侧菜单「<一级>」→「<二级>」
- 路由: /<route>
- 权限前缀: <perm>:
- 涉及权限码:
  - <perm>:create (新增)
  - <perm>:update (编辑)
  - <perm>:delete (删除)
  - <perm>:query (查询)
  - <perm>:export (导出,如适用)

【前端文件清单】
- 主页面: {{FRONT_VIEW_DIR}}/<dir>/index.vue
- 弹窗: {{FRONT_VIEW_DIR}}/<dir>/<ModalName>.vue (如适用)
- API 封装: {{API_DIR}}/<file>.ts (含 TypeScript 类型)

【后端文件清单】
- Controller: {{BACKEND_DIR}}/controller/<Name>Controller.java
- Service 接口: {{BACKEND_DIR}}/service/<Name>Service.java
- Service 实现: {{BACKEND_DIR}}/service/impl/<Name>ServiceImpl.java
- DTO/Request: {{BACKEND_DIR}}/controller/vo/<req|resp>/<Name>XxxReqVO.java
- DO: {{BACKEND_DIR}}/dal/dataobject/<Name>DO.java
- Mapper: {{BACKEND_DIR}}/dal/mapper/<Name>Mapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | /admin-api/{{NAME}}/<resource>/page |
| 详情 | GET | /admin-api/{{NAME}}/<resource>/get?id=X |
| 创建 | POST | /admin-api/{{NAME}}/<resource>/create |
| 更新 | PUT | /admin-api/{{NAME}}/<resource>/update |
| 删除 | DELETE | /admin-api/{{NAME}}/<resource>/delete?id=X |
| 导出 Excel (如适用) | GET | /admin-api/{{NAME}}/<resource>/export-excel |

【功能需求】
1. <核心功能 1>
2. <核心功能 2>
3. <表单字段清单>
4. <字典类型 / 业务组件复用>
5. <异常处理>

【UI 规范】
- UI 库: ant-design-vue
- 表单组件: <DictSelect / DictTag / DictSwitch 等>
- 操作按钮: <新增 / 编辑 / 删除 / 导出 / 导入 等>
- 分页: pageNo + pageSize (默认 10/20/50/100)

【参考实现】
请参考以下已实现模块:
- <模块 1>(<相似点>)
- <模块 2>(<相似点>)
- 复用 {{FRONT_VIEW_DIR}}/components 中的 <组件名> 组件

【测试验证】
实现完成后,使用 {{MODULE_ID}}.2 测试提示词中的所有场景验证,重点验证:
1. <验证点 1>
2. <验证点 2>
3. <验证点 3>

【交付物清单】
- [ ] 前端主页面 .vue
- [ ] 前端 API 封装 .ts (含 TypeScript 类型)
- [ ] 后端 Controller (含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册
- [ ] 菜单注册
- [ ] 权限码注册
- [ ] 字典数据初始化 SQL (如适用)
- [ ] 通过 {{MODULE_ID}}.2 所有测试场景
\`\`\`
```

# 规则
1. 所有占位符必须填充为真实值,不能保留 `{{}}`。
2. 前端文件清单 + 后端文件清单 = 9~12 个文件,对应阿里 COLA / mediation-framework 4 层结构。
3. 【功能需求】必须 ≥ 5 条,从 PRD §3 对应 Story 提炼。
4. 【交付物清单】至少 10 项,使用 Markdown checkbox。
5. 参考实现必须指向 mediation-platform 现有模块,不要假设不存在的代码。
````

### 4.3 验收

- 每个子模块 §5.x.3 包含完整开发提示词代码块
- 所有占位符已替换
- 至少 5 条功能需求 + 10 项交付物

---

## 5.N5 — 文档合并 + 第 6/7/8 章生成

### 5.1 节点职责

把所有子模块的 §5.x 内容合并到 `{NAME}-scene.md`,并补充第 6/7/8 章(测试报告模板 / 模块补全清单 / 文档版本)。

### 5.2 N5 提示词

````markdown
# 角色
你是 mediation-platform 技术文档工程师,负责文档归档。

# 输入
- 全部子模块的 §5.x 内容(N3 + N4 已生成)
- 输出文件:`{{SCENE_FILE}}`

# 任务
合并并生成 `{{SCENE_FILE}}`,**严格 1:1 对齐** `mediation-web/docs/scene/scene-template.md` 的完整结构。

# 输出文档完整结构

```markdown
# {{NAME_TITLE}} 服务 — 端到端浏览器自动化测试 Skill

> 本文档为 AI 浏览器测试 Agent 提供完整的 {{NAME_TITLE}}(<NAME>)服务测试提示词。
> 使用 `/browser` 启动浏览器自动化测试,或使用 MCP Playwright 执行测试。
> 适用于 gstack `/qa` 和 `/qa-only` Skill,通过 `/open-gstack-browser` 导入认证 Cookie 后执行端到端测试。
> 文档同时包含问题自动定位与修复建议机制,支持端到端回归测试。

---

## 目录
- [1.测试前置条件](#1测试前置条件)
- [2.全局测试策略](#2全局测试策略)
- [3.Skill 调用方式](#3skill-调用方式)
- [4.问题发现与自动修复流程](#4问题发现与自动修复流程)
- [5.{{NAME_TITLE}} 服务测试模块](#5{{NAME|lower}}-服务测试模块)
  - [{{NAME|upper}}-01 <子模块名>](#{{NAME|lower}}-01-<子模块名>)
  - [{{NAME|upper}}-02 <子模块名>](#{{NAME|lower}}-02-<子模块名>)
  - ... (按 PRD §10 顺序)
- [6.测试结果报告模板](#6测试结果报告模板)
- [7.模块开发对照与补全清单](#7模块开发对照与补全清单)
- [8.文档版本](#8文档版本)

---

## 1.测试前置条件
(直接复用 template §1.1~§1.3 全部表格与列表,仅替换端口与账号字段)

### 1.1 环境要求
| 项目 | 值 |
|------|------|
| 前端地址 | http://localhost:5173 |
| 网关地址 | http://localhost:8080 |
| {{NAME_TITLE}} 服务 | http://localhost:{{PORT}} |
| 超级管理员账号 | admin |
| 超级管理员密码 | admin123 |
| 默认租户 ID | 1 |

### 1.2 服务依赖检查
(列出 {{NAME}} 服务依赖的所有中间件)

### 1.3 浏览器环境要求
(直接复用 template)

---

## 2.全局测试策略
### 2.1 模块列表
(从 PRD §10 子模块表聚合,生成 Module Summary Table)

| 模块 ID | 模块名称 | 测试场景数 | 说明 |
|---------|---------|-----------|------|
| {{NAME|upper}}-01 | <名称> | N | <说明> |
| ... | ... | ... | ... |
| **合计** | **N 个模块** | **Σ** | **~Xh 测试时间** |

### 2.2 每个测试用例的验证清单
(直接复用 template §2.2 全部 9 项)

### 2.3 通用测试模式
(直接复用 template §2.3)

---

## 3.Skill 调用方式
### 3.1 方式一:使用 gstack /qa Skill(推荐)
### 3.2 方式二:使用 gstack /open-gstack-browser
### 3.3 方式三:直接使用 MCP Playwright
### 3.4 方式四:使用 /qa-only(仅发现问题不修复)
(全部直接复用 template,无需改写)

---

## 4.问题发现与自动修复流程
### 4.1 三阶段闭环
### 4.2 常见错误自动匹配表
(直接复用 template §4.1、§4.2,匹配表按需追加 {{NAME}} 特有错误)

---

## 5.{{NAME_TITLE}} 服务测试模块
(由 N3 + N4 生成的 §5.x 内容,按 PRD §10 顺序合并)

---

## 6.测试结果报告模板
(直接复用 template §6,仅替换顶部服务名与端口)

---

## 7.模块开发对照与补全清单
### {{NAME_TITLE}} 模块补全对照表
(从 PRD §10 + 当前仓库扫描结果生成)

| 模块 | tianque-ui 路径 | mediation-platform 路径 | 前端状态 | 后端状态 | 补全建议 |

---

## 8.文档版本
| 版本 | 日期 | 修改内容 | 作者 |
|------|------|---------|------|
| 1.0.0 | YYYY-MM-DD | 初始版本,生成 {{NAME_TITLE}} 服务测试模块,覆盖 N 个业务模块 M 个测试场景 | AI Agent |
```

# 规则
1. 章节顺序、命名、子节编号、表格列、占位符语义 **1:1 对齐** scene-template.md。
2. §1、§2、§3、§4 大部分章节直接复用 template 内容,只替换端口/服务名。
3. §5 是 N3+N4 合并产物,每个子模块保留 5.x.1 / 5.x.2 / 5.x.3 三个子节。
4. §7 表格必须先扫描 mediation-platform 仓库确认模块状态,不要凭空填写 ✅。
5. 输出最终文件路径与总行数。
````

### 5.3 验收

- 文件结构与 scene-template.md 完全对齐
- 目录链接全部可点击
- 8 个一级章节齐全

---

## 6.N6 — Cursor 中串行调用总编排提示词

### 6.1 节点职责

让用户在 Cursor 对话框中**一句触发**,自动按 N1→N2→N3→N4→N5 顺序生成完整文档。

### 6.2 N6 主提示词(用户直接粘贴到 Cursor)

````markdown
# 一句话触发 — PRD → 子模块 → 测试/开发提示词 一体化生成

## 你的任务
执行 Pipeline,按 N1→N2→N3→N4→N5 顺序生成完整文档。

## 输入参数
- 服务短名(NAME):`{{NAME}}`
- 中文标题(NAME_TITLE):`{{NAME_TITLE}}`
- 后端端口(PORT):`{{PORT}}`
- 前端 views 目录:`{{FRONT_VIEW_DIR}}`
- 前端 api 目录:`{{API_DIR}}`
- 后端基础路径:`{{BACKEND_DIR}}`
- 一句话需求:`{{USER_INPUT}}`

## 输出路径
- 中间 PRD:`{{PRD_FILE}}`
- 最终文档:`{{SCENE_FILE}}`

## 执行步骤(严格按序)

### Step 1 — 加载技能
读取并遵循以下 skill:
- `superpowers:using-superpowers`(纪律)
- `superpowers:brainstorming`(需求澄清,如信息不足)
- `gstack:gsd-help` 或 `get-shit-done:gsd-new-project`(任务规划)
- `awesome-design-md`(Markdown 文档美学)
- `OpenSpec`(规范驱动开发)

### Step 2 — 执行 N1
读取 `mediation-web/docs/scene/scene-template.md` 作为骨架模板。
按 N1 提示词生成 `{{PRD_FILE}}`,覆盖 §1~§9。

### Step 3 — 执行 N2
读取 `{{PRD_FILE}}`,按 N2 提示词拆解子模块,填充 §10。

### Step 4 — 执行 N3 (循环每个子模块)
对 §10 中的每个子模块 {{NAME|upper}}-NN,按 N3 提示词生成 §5.x.2 测试提示词。

### Step 5 — 执行 N4 (循环每个子模块)
对 §10 中的每个子模块 {{NAME|upper}}-NN,按 N4 提示词生成 §5.x.3 开发提示词。

### Step 6 — 执行 N5
合并所有 §5.x 内容,生成完整 `{{SCENE_FILE}}`,包含 §1~§8 全部章节。

### Step 7 — 校验
- [ ] 文档结构与 scene-template.md 1:1 对齐
- [ ] 所有占位符已替换
- [ ] §10 子模块数 = §5 子模块数
- [ ] 每个子模块都有 5.x.1 / 5.x.2 / 5.x.3 三个子节
- [ ] §7 模块状态已扫描仓库确认

## 完成后输出
1. 最终文档绝对路径
2. 总章节数、子模块数、测试场景总数
3. 任何偏离模板的差异说明

## 模板锚点
- 骨架模板:`@mediation-web/docs/scene/scene-template.md`
- 范例文档:`@mediation-web/docs/scene/ai-scene.md`
- 本 Pipeline:`@mediation-web/docs/scene/pipeline.md`(本文件)

## 注意事项
- 任何步骤失败,必须停下并报告,不得跳过。
- 信息不足时使用 `<待定>` 而非省略。
- 不要生成未在 PRD 中声明的子模块。
- §7 表格的状态必须扫描 `{{FRONT_VIEW_DIR}}` 与 `{{BACKEND_DIR}}` 实际文件后填写。
````

### 6.3 使用流程

```
1. 用户在 Cursor 对话框粘贴 N6 提示词
2. 替换所有 {{}} 占位符为实际值
3. 发送,Cursor 自动串行执行 N1~N5
4. 等待输出 {NAME}-scene.md
5. 可选:使用 gstack /qa 执行 §5 测试提示词进行回归
```

---

## 7.工作流集成(技能矩阵)

| Pipeline 节点 | Superpowers | gstack / GSD | OpenSpec | awesome-design-md |
|---------------|-------------|--------------|----------|-------------------|
| N1 | brainstorming, writing-plans | gsd-discuss-phase, gsd-spec-phase | openspec-proposal | markdown 美学 |
| N2 | subagent-driven-development | gsd-plan-phase | openspec-tasks | — |
| N3 | test-driven-development | gsd-execute-phase (qa) | — | — |
| N4 | test-driven-development | gsd-execute-phase | openspec-apply | — |
| N5 | verification-before-completion | gsd-verify-work | — | markdown 美学 |
| N6 | using-superpowers | gsd-manager | openspec-orchestrator | — |

---

## 8.快速复用 — 最小化输入

如果用户只提供 1 句话,**最小可执行版本**示例:

```yaml
USER_INPUT: "做一个案件登记页面"
NAME: case
NAME_TITLE: 案件管理
PORT: 8084
FRONT_VIEW_DIR: mediation-web/src/views/case
API_DIR: mediation-web/src/api/case
BACKEND_DIR: mediation-basic/mediation-module-case
PRD_FILE: mediation-web/docs/scene/case-prd.md
SCENE_FILE: mediation-web/docs/scene/case-scene.md
```

直接粘贴上述 YAML 到 N6 提示词顶部,即可全自动生成。

---

## 9.验收 Checklist(发布前)

- [ ] `{NAME}-scene.md` 文件已生成,行数 ≥ 1500
- [ ] 8 个一级章节齐全(1~8)
- [ ] §5 包含 N 个子模块,每个子模块都有 5.x.1 / 5.x.2 / 5.x.3
- [ ] §6 测试报告模板与 scene-template 一致
- [ ] §7 表格行数 = 子模块数
- [ ] §8 文档版本号 = 1.0.0,日期 = 今日
- [ ] §10 子模块数 = §5 子模块数
- [ ] 至少 1 个子模块使用本 Pipeline 实测过(以 SYS-01 为 sanity check)

---

## 10.文档版本

| 版本 | 日期 | 修改内容 | 作者 |
|------|------|---------|------|
| 1.0.0 | 2026-06-17 | 初始版本,定义 N1~N6 六节点 Pipeline | AI Agent |

# {SERVICE_NAME} 服务 — 端到端浏览器自动化测试 Skill

> 本文档为 AI 浏览器测试 Agent 提供完整的 {SERVICE_NAME}({SERVICE_NAME_CN})服务测试提示词。
> 使用 `/browser` 启动浏览器自动化测试,或使用 MCP Playwright 执行测试。
> 适用于 gstack `/qa` 和 `/qa-only` Skill,通过 `/open-gstack-browser` 导入认证 Cookie 后执行端到端测试。
> 文档同时包含问题自动定位与修复建议机制,支持端到端回归测试。

---

## 模板使用说明

> **本模板为通用开发文档模板,适用于任意业务模块的端到端测试与开发指导。**
>
> 使用方式:
> 1. 复制本模板文件,重命名为 `{module}-dev-template.md`
> 2. 全局替换所有 `{PLACEHOLDER}` 占位符为实际值
> 3. 根据实际模块数量扩展测试模块章节(第 5 章)
> 4. 删除本「模板使用说明」章节
>
> **占位符清单:**
>
> | 占位符 | 说明 | 示例 |
> |--------|------|------|
> | `{SERVICE_NAME}` | 服务英文名 | `System`、`Case`、`Order` |
> | `{SERVICE_NAME_CN}` | 服务中文名 | `系统管理`、`案件管理`、`订单管理` |
> | `{SERVICE_PORT}` | 服务端口号 | `8082`、`8084` |
> | `{FRONTEND_URL}` | 前端访问地址 | `http://localhost:5173` |
> | `{GATEWAY_URL}` | 网关地址 | `http://localhost:8080` |
> | `{ADMIN_USER}` | 管理员账号 | `admin` |
> | `{ADMIN_PASS}` | 管理员密码 | `admin123` |
> | `{TENANT_ID}` | 默认租户 ID | `1` |
> | `{PROJECT_NAME}` | 项目名称/仓库名 | `business-platform` |
> | `{MODULE_ID}` | 模块编号前缀 | `SYS`、`CASE`、`ORD` |
> | `{MODULE_NAME}` | 模块名称 | `参数配置`、`案件登记` |
> | `{MENU_PATH}` | 菜单导航路径 | `系统管理 → 参数配置` |
> | `{ROUTE_PATH}` | 前端路由路径 | `/system/config` |
> | `{PERMISSION_PREFIX}` | 权限标识前缀 | `system:config:` |
> | `{API_BASE_PATH}` | API 路径前缀 | `/admin-api/system/config` |
> | `{UI_FRAMEWORK}` | UI 组件库 | `ant-design-vue`、`element-plus` |
> | `{REF_PROJECT}` | 参考项目名 | `example-ui` |
>
> **章节扩展说明:**
> - 第 5 章按模块逐一展开,每个模块包含:测试场景、测试提示词、开发提示词
> - 模块数量不限,按 `{MODULE_ID}-01`、`{MODULE_ID}-02`... 顺序编号
> - 第 7 章对照表根据实际模块数量填写

---

## 目录

- [1.测试前置条件](#测试前置条件)
- [2.全局测试策略](#全局测试策略)
- [3.Skill 调用方式](#skill-调用方式)
- [4.问题发现与自动修复流程](#问题发现与自动修复流程)
- [5.{SERVICE_NAME} 服务测试模块](#服务测试模块)
  - [{MODULE_ID}-01 {MODULE_NAME}](#module-01)
- [6.测试结果报告模板](#测试结果报告模板)
- [7.模块开发对照与补全清单](#模块开发对照与补全清单)
- [8.文档版本](#文档版本)

---

## 1.测试前置条件

### 1.1 环境要求

| 项目 | 值 |
|------|------|
| 前端地址 | `{FRONTEND_URL}` |
| 网关地址 | `{GATEWAY_URL}` |
| {SERVICE_NAME} 服务 | `{GATEWAY_URL}:{SERVICE_PORT}` |
| 超级管理员账号 | `{ADMIN_USER}` |
| 超级管理员密码 | `{ADMIN_PASS}` |
| 默认租户 ID | `{TENANT_ID}` |

### 1.2 服务依赖检查

测试前需确认以下服务已启动:

1. **Gateway**(:{GATEWAY_PORT}) — API 网关
2. **{SERVICE_NAME}**(:{SERVICE_PORT}) — {SERVICE_NAME_CN}服务
3. **{DB_TYPE}** — 数据库
4. **{CACHE_TYPE}** — 缓存
5. **{REGISTRY_TYPE}** — 服务注册中心

### 1.3 浏览器环境要求

- 浏览器:Chromium / Chrome(headless 模式或带 UI 模式均可)
- Cookie 导入:通过 `/open-gstack-browser` 导入已登录 Cookie,避免重复登录
- 如需手动登录:账号 `{ADMIN_USER}`,密码 `{ADMIN_PASS}`,租户 ID `{TENANT_ID}`

---

## 2.全局测试策略

### 2.1 模块列表:

| 模块 ID | 模块名称 | 测试场景数 | 说明 |
|---------|---------|-----------|------|
| {MODULE_ID}-01 | {MODULE_NAME_1} | {N1} | {说明_1} |
| {MODULE_ID}-02 | {MODULE_NAME_2} | {N2} | {说明_2} |
| ... | ... | ... | ... |
| **合计** | **{TOTAL_MODULES} 个模块** | **{TOTAL_SCENARIOS} 个场景** | **~{EST_HOURS}h 测试时间** |

### 2.2 每个测试用例的验证清单

1. **页面加载**:页面是否正常渲染,无白屏、无 JS 错误
2. **数据加载**:表格数据是否成功加载,loading 状态是否正确消失
3. **交互响应**:按钮点击是否有响应,弹窗是否正常弹出
4. **表单验证**:必填字段校验是否生效,错误提示是否显示
5. **数据回填**:编辑时已有数据是否正确回填到表单
6. **操作反馈**:成功/失败是否有 message 提示
7. **状态更新**:操作后列表数据是否自动刷新
8. **权限控制**:`v-has-permi` 指令是否正确控制按钮显隐
9. **控制台检查**:浏览器控制台是否有报错或警告

### 2.3 通用测试模式

每个模块遵循 **CRUD 测试循环**:

```
1. 打开页面 → 验证列表加载
2. 搜索/筛选 → 验证条件过滤
3. 点击「新增」→ 验证弹窗/表单初始化 → 填写并提交 → 验证列表刷新
4. 点击「编辑」→ 验证数据回填 → 修改并提交 → 验证更新生效
5. 点击「删除」→ 验证确认弹窗 → 确认删除 → 验证列表更新
6. 边界测试 → 空数据、超长文本、特殊字符、并发操作
```

---

## 3.Skill 调用方式

### 3.1 方式一:使用 gstack /qa Skill(推荐)

```bash
/qa --tier standard --url {FRONTEND_URL}
```

### 3.2 方式二:使用 gstack /open-gstack-browser

```
/open-gstack-browser
启动 GStack 浏览器后,执行以下测试流程...

# 或导入 Cookie 跳过登录
/setup-browser-cookies
选择已登录的 {PROJECT_NAME} 会话
执行测试...
```

### 3.3 方式三:直接使用 MCP Playwright

```bash
# 启动浏览器
$B goto {FRONTEND_URL}/login
$B screenshot "login-page.png"

# 登录
$B fill "#username" "{ADMIN_USER}"
$B fill "#password" "{ADMIN_PASS}"
$B click "button[type=submit]"

# 验证跳转
$B wait-for-url "**/dashboard"
$B screenshot "after-login.png"

# 继续测试各模块...
```

### 3.4 方式四:使用 /qa-only(仅发现问题不修复)

```
/qa-only --tier exhaustive --scope "{SERVICE_NAME}服务-{MODULE_NAME}"
```

---

## 4.问题发现与自动修复流程

### 4.1 三阶段闭环

```
阶段 1:问题发现(Detect)
  → 浏览器测试发现异常
  → 截图记录当前状态
  → 记录控制台错误日志
  → 标记失败测试用例

阶段 2:问题定位(Diagnose)
  → 根据错误信息判断问题层级:
    - 前端渲染问题 → 检查 Vue 组件
    - API 请求失败 → 检查 Network 面板(状态码、响应体)
    - 后端逻辑错误 → 检查后端日志
    - 数据问题 → 检查数据库状态
  → 定位到具体文件和行号

阶段 3:自动修复建议(Fix)
  → 输出修复方案(代码 diff)
  → 常见修复模式:
    - 组件未导入 → 补充 import
    - API 路径错误 → 对齐后端路由
    - v-model 绑定问题 → 检查响应式
    - 权限码不匹配 → 对齐前后端权限标识
    - 表单初始值缺失 → 补充默认值
  → 修复后自动重新执行失败的测试用例
```

### 4.2 常见错误自动匹配表

| 浏览器现象 | 可能原因 | 检查文件 |
|-----------|---------|---------|
| 白屏 | 路由组件未注册 / import 路径错误 | `route-helper.ts`, `views/` |
| 表格无数据 | API 返回 code≠0 / 后端未启动 | `request.ts`, Network 面板 |
| 弹窗打不开 | `v-model:open` / `v-model:visible` 绑定问题 | 对应 FormModal.vue |
| 表单不回填 | watch 未监听 / props 延迟 | FormModal.vue 的 watch |
| 按钮不显示 | 权限码不匹配 | `access.ts`, 后端权限配置 |
| 401 错误 | Token 过期 / 刷新 Token 逻辑失败 | `request.ts`, `auth.ts` |
| 删除失败 | 外键约束 / 关联数据未清理 | 后端 Service |
| 分页异常 | pageNo/pageSize 参数不对 | API 请求参数 |
| 树形不展开 | `handleTree` 函数异常 | `menuTreeUtils.ts` |
| 表格列错位 | columns 定义顺序与数据不匹配 | index.vue 的 columns 定义 |
| DictTag 不显示 | 字典类型未注册 / 值为空 | `DictTag.vue`, 后端字典表 |

---

## 5.{SERVICE_NAME} 服务测试模块

---

### 5.1 {MODULE_ID}-01 {MODULE_NAME}

**页面路径**: 左侧菜单「{MENU_PATH}」  
**源码文件**: `src/views/{MODULE_DIR}/index.vue`, `src/views/{MODULE_DIR}/{FormModal}.vue`  
**API 文件**: `src/api/{MODULE_DIR}/index.ts`  
**权限标识**: `{PERMISSION_PREFIX}create`, `{PERMISSION_PREFIX}update`, `{PERMISSION_PREFIX}delete`

#### 5.1.1 测试场景

<!-- 按实际模块填写测试场景列表 -->

| 场景编号 | 场景名称 | 验证要点 |
|---------|---------|---------|
| 1 | 列表加载与基础显示 | 表格列完整、分页正常、标签渲染正确 |
| 2 | 搜索功能 | 条件过滤正确、重置清空 |
| 3 | 新增 | 弹窗表单完整、提交成功、列表刷新 |
| 4 | 编辑 | 数据回填正确、修改提交后更新 |
| 5 | 唯一性/业务校验 | 后端校验生效、错误提示正确 |
| 6 | 删除 | 确认弹窗、删除成功、列表刷新 |

#### 5.1.2 测试提示词

```
/qa 

前端：{FRONTEND_URL}  后端： {GATEWAY_URL}

打开{MODULE_NAME}页面,执行完整的{MODULE_NAME} CRUD 测试。

【前置操作】
1. 使用 {ADMIN_USER}/{ADMIN_PASS} 登录系统
2. 在左侧菜单点击「{MENU_PATH}」
3. 等待页面加载完成

---

【测试场景 1:列表加载与基础显示】
1. 验证{MODULE_NAME}列表正常加载
2. 检查列:{COLUMN_LIST}
3. 验证分页功能正常
4. 验证特殊列渲染(如 DictTag、状态标签等)

预期结果:
✅ 表格列完整
✅ 特殊列渲染正确
✅ 分页正常

---

【测试场景 2:搜索功能】
1. 在「{SEARCH_FIELD_1}」输入框输入关键词,点击「搜索」
2. 验证列表过滤出匹配的结果
3. 在「{SEARCH_FIELD_2}」输入框输入关键词,点击「搜索」
4. 验证列表过滤出匹配的结果
5. 点击「重置」,验证条件清空,列表恢复

预期结果:
✅ {SEARCH_FIELD_1}搜索过滤正确
✅ {SEARCH_FIELD_2}搜索过滤正确
✅ 重置清空条件

---

【测试场景 3:新增{MODULE_NAME}】
1. 点击顶部「新增」按钮
2. 验证弹窗弹出,标题为「新增{MODULE_NAME}」
3. 验证表单字段:
   - {FORM_FIELD_1}(必填)
   - {FORM_FIELD_2}(必填)
   - {FORM_FIELD_3}(可选)
   - ...
4. 填写测试数据并提交
5. 验证:
   - API POST {API_BASE_PATH}/create
   - 显示「新增成功」
   - 弹窗关闭,列表刷新

预期结果:
✅ 新增弹窗表单字段完整
✅ 提交成功,列表刷新

---

【测试场景 4:编辑{MODULE_NAME}】
1. 点击刚创建记录的「编辑」按钮
2. 验证弹窗弹出,数据正确回填
3. 修改某字段值
4. 提交并验证更新

预期结果:
✅ 编辑弹窗正确回填
✅ 修改提交后数据更新

---

【测试场景 5:业务校验】
1. 新增记录,触发唯一性/业务规则校验
2. 提交验证:后端返回校验错误
3. 弹窗不关闭,显示错误提示

预期结果:
✅ 后端校验生效
✅ 前端正确展示错误信息

---

【测试场景 6:删除{MODULE_NAME}】
1. 点击记录的「删除」按钮
2. 验证确认弹窗
3. 点击确认,验证删除成功
4. 验证列表刷新,记录消失

预期结果:
✅ 删除确认正常
✅ 删除成功后列表刷新

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表分页 | GET | {API_BASE_PATH}/page |
| 详情 | GET | {API_BASE_PATH}/get?id=X |
| 创建 | POST | {API_BASE_PATH}/create |
| 更新 | PUT | {API_BASE_PATH}/update |
| 删除 | DELETE | {API_BASE_PATH}/delete?id=X |

---

【问题诊断】
- 列表不加载 → 检查 API 请求是否成功
- 编辑时数据不回填 → 检查 FormModal watch 逻辑
- 删除失败 → 检查是否存在关联数据保护(后端约束)
```

#### 5.1.3 开发提示词

```
请基于以下信息,在 {PROJECT_NAME} 仓库中实现【{MODULE_NAME}】模块。

【模块信息】
- 服务: {SERVICE_NAME}(:{SERVICE_PORT})
- 页面路径: 左侧菜单「{MENU_PATH}」
- 路由: {ROUTE_PATH}
- 权限前缀: {PERMISSION_PREFIX}
- 涉及权限码:
  - {PERMISSION_PREFIX}create(新增)
  - {PERMISSION_PREFIX}update(编辑)
  - {PERMISSION_PREFIX}delete(删除)
  - {PERMISSION_PREFIX}query(查询)

【前端文件清单】
- 主页面: {FRONTEND_PROJECT}/src/views/{MODULE_DIR}/index.vue
- 表单弹窗: {FRONTEND_PROJECT}/src/views/{MODULE_DIR}/{FormModal}.vue
- API 封装: {FRONTEND_PROJECT}/src/api/{MODULE_DIR}/index.ts

【后端文件清单】
- Controller: {BACKEND_MODULE_PATH}/controller/{Entity}Controller.java
- Service 接口: {BACKEND_MODULE_PATH}/service/{Entity}Service.java
- DTO/Request: {BACKEND_MODULE_PATH}/dto/{Entity}Request.java
- DO: {BACKEND_MODULE_PATH}/dal/dataobject/{Entity}DO.java
- Mapper: {BACKEND_MODULE_PATH}/dal/mapper/{Entity}Mapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | {API_BASE_PATH}/page |
| 获取详情 | GET | {API_BASE_PATH}/get |
| 创建 | POST | {API_BASE_PATH}/create |
| 更新 | PUT | {API_BASE_PATH}/update |
| 删除 | DELETE | {API_BASE_PATH}/delete |

【功能需求】
1. {需求描述_1}
2. {需求描述_2}
3. {需求描述_3}
4. 表单验证:{验证规则描述}
5. 提交失败时显示错误提示

【UI 规范】
- UI 库: {UI_FRAMEWORK}
- 表单字段:{FORM_FIELD_LIST}
- 字典类型: {DICT_TYPE}(如有)
- 业务组件复用: {BUSINESS_COMPONENTS}
- 操作按钮:{BUTTON_LIST}

【参考实现】
请参考以下已实现模块:
- {REF_MODULE_PATH_1}({参考说明_1})
- {REF_MODULE_PATH_2}({参考说明_2})
- 复用 {SHARED_COMPONENTS_PATH} 中的公共组件

【测试验证】
实现完成后,使用 5.1.2 测试提示词中的测试场景验证,重点验证:
1. {验证点_1}
2. {验证点_2}
3. {验证点_3}
4. {验证点_4}
5. {验证点_5}

【交付物清单】
- [ ] 前端主页面 .vue
- [ ] 前端表单弹窗组件 .vue
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册({ROUTE_PATH})
- [ ] 菜单注册(左侧菜单「{MENU_PATH}」)
- [ ] 权限码注册({PERMISSION_PREFIX}create 等)
- [ ] 字典数据初始化 SQL(如有)
- [ ] 通过 5.1.2 所有测试场景
```

---

*(注:如需更多模块,按 `{MODULE_ID}-02`、`{MODULE_ID}-03`... 格式继续扩展,每个模块包含 5.x.1 测试场景、5.x.2 测试提示词、5.x.3 开发提示词 三个子章节)*

---

## 6.测试结果报告模板

```markdown
# {SERVICE_NAME} 服务测试报告

**测试日期**: YYYY-MM-DD  
**测试人员**: AI Agent (gstack /qa)  
**测试环境**: {FRONTEND_URL}  
**{SERVICE_NAME} 服务**: {GATEWAY_URL}:{SERVICE_PORT}

## 测试摘要

| 指标 | 值 |
|------|------|
| 总用例数 | N |
| 通过 | X |
| 失败 | Y |
| 跳过 | Z |
| 通过率 | X/N% |
| 执行时长 | M 分钟 |

## 结果汇总

| 模块 | 用例数 | 通过 | 失败 | 跳过 | 通过率 |
|------|--------|------|------|------|--------|
| {MODULE_ID}-01 {MODULE_NAME_1} | {N1} | {P1} | {F1} | {S1} | {R1}% |
| {MODULE_ID}-02 {MODULE_NAME_2} | {N2} | {P2} | {F2} | {S2} | {R2}% |
| ... | ... | ... | ... | ... | ... |
| **合计** | **N** | **X** | **Y** | **Z** | **X/N%** |

## 失败用例详情

### {MODULE_ID}-{XX} {MODULE_NAME} — 测试场景 {N}:{场景名}

- **现象**: {异常现象描述}
- **控制台错误**: `{错误信息}`
- **截图**: [.gstack/qa-reports/screenshots/issue-{NNN}.png](.gstack/qa-reports/screenshots/issue-{NNN}.png)
- **定位**: {问题定位描述}
- **修复建议**: {修复方案}
- **修复文件**: `{文件路径}`
- **状态**: 已修复 / 待修复

## 建议

1. **高优先级**:{高优先级建议}
2. **中优先级**:{中优先级建议}
3. **低优先级**:{低优先级建议}
4. **测试补充**:{补充测试建议}
```

---

## 7.模块开发对照与补全清单

本章节对照 {REF_PROJECT}(参考项目)与 {PROJECT_NAME}(当前项目),梳理 {SERVICE_NAME} 模块的实现状态与补全建议。

### {SERVICE_NAME} 模块补全对照表

| 模块 | {REF_PROJECT} 路径 | {PROJECT_NAME} 路径 | 前端状态 | 后端状态 | 补全建议 |
|------|-----------------|----------------------|---------|---------|---------|
| {MODULE_NAME_1} | {ref_path_1} | {cur_path_1} | {状态} | {状态} | {建议} |
| {MODULE_NAME_2} | {ref_path_2} | {cur_path_2} | {状态} | {状态} | {建议} |
| ... | ... | ... | ... | ... | ... |

<!-- 
状态标记说明:
- ✅ 完整 — 功能已实现且与参考项目对齐
- ⚠️ 部分 — 核心功能已有,部分细节缺失
- ❌ 缺失 — 尚未实现
- 🔄 进行中 — 正在开发

补全建议示例:
- 对齐:无明显差距
- 已完成:{具体实现说明}
- 待补全:{缺失功能描述}
-->

---

## 8.文档版本

| 版本 | 日期 | 修改内容 | 作者 |
|------|------|---------|------|
| 1.0.0 | {DATE} | 初始版本,基于模板创建 | {AUTHOR} |

| 项目 | 值 |
|------|------|
| 适用服务 | {SERVICE_NAME}(:{SERVICE_PORT}) |
| 前端入口 | {FRONTEND_URL} |
| 默认账号 | {ADMIN_USER} / {ADMIN_PASS} |
| 默认租户 ID | {TENANT_ID} |
| 测试 Skill | `/qa`, `/qa-only`, `/open-gstack-browser`, MCP Playwright |

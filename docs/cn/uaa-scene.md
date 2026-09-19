# UAA 服务 — 端到端浏览器自动化测试 Skill

> 本文档为 AI 浏览器测试 Agent 提供完整的 UAA(用户管理与认证)服务测试提示词。
> 使用 `/browser` 启动浏览器自动化测试,或使用 MCP Playwright 执行测试。
> 适用于 gstack `/qa` 和 `/qa-only` Skill,通过 `/open-gstack-browser` 导入认证 Cookie 后执行端到端测试。
> 文档同时包含问题自动定位与修复建议机制,支持端到端回归测试。

---

## 目录

- [1.测试前置条件](#测试前置条件)
- [2.全局测试策略](#全局测试策略)
- [3.Skill 调用方式](#skill-调用方式)
- [4.问题发现与自动修复流程](#问题发现与自动修复流程)
- [5.UAA 服务测试模块](#uaa-服务测试模块)
  - [UAA-01 登录与登出](#uaa-01-登录与登出)
  - [UAA-02 用户管理](#uaa-02-用户管理)
  - [UAA-03 角色管理](#uaa-03-角色管理)
  - [UAA-04 角色菜单赋权](#uaa-04-角色菜单赋权)
  - [UAA-05 菜单管理](#uaa-05-菜单管理)
  - [UAA-06 部门管理](#uaa-06-部门管理)
  - [UAA-07 岗位管理](#uaa-07-岗位管理)
  - [UAA-08 租户管理](#uaa-08-租户管理)
  - [UAA-09 租户套餐管理](#uaa-09-租户套餐管理)
  - [UAA-10 OAuth2 客户端管理](#uaa-10-oauth2-客户端管理)
  - [UAA-11 OAuth2 Token 管理](#uaa-11-oauth2-token-管理)
  - [UAA-12 社交用户管理](#uaa-12-社交用户管理)
  - [UAA-13 社交客户端管理](#uaa-13-社交客户端管理)
  - [UAA-14 登录日志](#uaa-14-登录日志)
  - [UAA-15 业务机构管理](#uaa-15-业务机构管理)
  - [UAA-16 业务专员管理](#uaa-16-业务专员管理)
  - [UAA-17 用户导入](#uaa-17-用户导入)
  - [UAA-18 角色数据权限分配](#uaa-18-角色数据权限分配)
  - [UAA-19 登录日志详情](#uaa-19-登录日志详情)
- [6.测试结果报告模板](#测试结果报告模板)
- [7.模块开发对照与补全清单](#模块开发对照与补全清单)
- [8.文档版本](#文档版本)
  

---

## 1.测试前置条件

### 1.1 环境要求

| 项目 | 值 |
|------|------|
| 前端地址 | `http://localhost:5173` |
| 网关地址 | `http://localhost:8080` |
| UAA 服务 | `http://localhost:8081` |
| 超级管理员账号 | `admin` |
| 超级管理员密码 | `admin123` |
| 默认租户 ID | `1` |

### 1.2 服务依赖检查

测试前需确认以下服务已启动:

1. **Gateway**(:8080) — API 网关
2. **UAA**(:8081) — 用户管理与权限服务
3. **MySQL/PostgreSQL** — 数据库
4. **Redis** — 缓存
5. **Nacos** — 服务注册中心

### 1.3 浏览器环境要求

- 浏览器:Chromium / Chrome(headless 模式或带 UI 模式均可)
- Cookie 导入:通过 `/open-gstack-browser` 导入已登录 Cookie,避免重复登录
- 如需手动登录:账号 `admin`,密码 `admin123`,租户 ID `1`

---

## 2.全局测试策略

### 2.1 模块列表:

| 模块 ID | 模块名称 | 测试场景数 | 说明 |
|---------|---------|-----------|------|
| UAA-01 | 登录与登出 | 6 | CRUD + 唯一性校验 |
| UAA-02 | 用户管理 | 10 | 类型+数据双表 CRUD |
| UAA-03 | 角色管理 | 11 | CRUD + 富文本 |
| UAA-04 | 角色菜单赋权 | 8 | 模板+消息+标记已读 |
| UAA-05 | 菜单管理 | 9 | 列表/网格/预览/拖拽 |
| UAA-06 | 部门管理 | 8 | 配置 CRUD + 测试连接 |
| UAA-07 | 岗位管理 | 7 | 账号/模板/日志 Tab |
| UAA-08 | 租户管理 | 5 | 渠道/模板/日志 Tab |
| UAA-09 | 租户套餐管理 | 5 | 查看 + 详情 |
| UAA-10 | OAuth2 客户端管理 | 4 | 查看 + 标记已读 |
| UAA-11 | OAuth2 Token 管理 | 3 | 详情弹窗 |
| UAA-12 | 社交用户管理 | 8 | 完整 CRUD |
| UAA-13 | 社交客户端管理 | 9 | 完整 CRUD |
| UAA-14 | 登录日志 | 7 | 查看 + 详情 |
| UAA-15 | 业务机构管理 | 4 | 查看 + 导出 |
| UAA-16 | 业务专员管理 | 7 | 查看 + 处理状态流转 |
| UAA-17 | 用户导入 | 4 | 仪表盘 + 图表 |
| UAA-18 | 角色数据权限分配 | 6 | CRUD + 批量删除 |
| UAA-19 | 登录日志详情 | 1 | iframe 内嵌 |
| **合计** | **19 个模块** | **126 个场景** | **~7h 测试时间** |

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
/qa --tier standard --url http://localhost:5173
```

### 3.2 方式二:使用 gstack /open-gstack-browser

```
/open-gstack-browser
启动 GStack 浏览器后,执行以下测试流程...

# 或导入 Cookie 跳过登录
/setup-browser-cookies
选择已登录的 business_platform 会话
执行测试...
```

### 3.3 方式三:直接使用 MCP Playwright

```bash
# 启动浏览器
$B goto http://localhost:5173/login
$B screenshot "login-page.png"

# 登录
$B fill "#username" "admin"
$B fill "#password" "admin123"
$B click "button[type=submit]"

# 验证跳转
$B wait-for-url "**/dashboard"
$B screenshot "after-login.png"

# 继续测试各模块...
```

### 3.4 方式四:使用 /qa-only(仅发现问题不修复)

```
/qa-only --tier exhaustive --scope "UAA服务-用户管理"
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
| 业务机构审核失败 | 审核状态枚举值不匹配 | `AuditStatusEnum` 定义 |
| 业务专员审核后状态不变 | updateStatus 未调用或未刷新列表 | `staff/index.vue` |
| 机构类型下拉为空 | OrgTypeEnum 未正确导入 | `OrgFormModal.vue` |
| 业务专员列表无数据 | page 接口返回空 / 分页参数缺失 | `staff.ts` API 定义 |

---

## 5.UAA 服务测试模块

---

### 5.1 UAA-01 登录与登出

**页面路径**: `/login`
**源码文件**: `src/views/login/index.vue`
**API 文件**: `src/api/core/auth.ts`
**权限标识**: 无（公开页面）

#### 5.1.1 测试场景

#### 5.1.2 测试提示词

```
/qa 或 /open-gstack-browser  /qa
打开 http://localhost:5173/login 执行登录登出功能测试。

【前置操作】
1. 确认页面已加载，标题显示「用户登录」
2. 确认默认 Tab 为「账号密码登录」

---

【测试场景 1:账号密码登录 - 正常流程】
1. 在用户名输入框输入 "admin"
2. 在密码输入框输入 "admin123"
3. 点击「登录」按钮
4. 验证:
   - 是否出现「登录成功」message 提示
   - 页面是否跳转到 /dashboard
   - 左侧菜单是否正常渲染
   - 浏览器 localStorage 中是否有 accessToken
5. 截图保存当前状态

预期结果:
✅ 登录成功 message 提示
✅ 页面跳转到 /dashboard
✅ 左侧菜单正常渲染
✅ accessToken 存在于 localStorage

---

【测试场景 2:账号密码登录 - 错误凭证】
1. 退出登录(如有登出按钮则点击，否则清除 localStorage 后刷新)
2. 回到 /login 页面
3. 输入用户名 "admin"，密码 "wrong_password"
4. 点击「登录」按钮
5. 验证:
   - 是否显示错误提示(如「账号或密码不正确」)
   - 页面是否停留在 /login
   - 密码框是否已清空
  

预期结果:
✅ 错误提示正常显示
✅ 页面停留在 /login
✅ 密码框已清空

---

【测试场景 3:账号密码登录 - 空值校验】
1. 不输入任何内容直接点击「登录」
2. 验证:用户名和密码字段是否都出现必填校验提示
3. 只输入用户名，不输入密码，点击「登录」
4. 验证:密码字段是否出现必填提示

预期结果:
✅ 用户名和密码均出现必填校验
✅ 密码单独验证提示必填

---

【测试场景 4:短信验证码登录 Tab 切换】
1. 点击「短信验证码登录」Tab
2. 验证:表单是否切换为手机号+验证码输入框
3. 输入手机号 "13800138000"
4. 点击「获取验证码」按钮
5. 验证:按钮是否变为倒计时状态(60s)
6. 不输入验证码直接点「登录」，验证必填校验

预期结果:
✅ Tab 切换正常
✅ 验证码倒计时显示
✅ 必填校验生效

---

【测试场景 5:第三方登录按钮】
1. 滚动到「其他登录方式」区域
2. 验证微信、GitHub、钉钉三个按钮是否存在且可点击

预期结果:
✅ 第三方登录按钮完整显示

---

【测试场景 6:Token 持久化与自动跳转】
1. 使用正确凭证登录成功
2. 刷新页面(F5)
3. 验证:是否自动跳转到 /dashboard 而非停留在 /login

预期结果:
✅ 刷新后自动跳转至 /dashboard

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 账号密码登录 | POST | /admin-api/uaa/auth/login |
| 获取验证码 | POST | /admin-api/uaa/auth/send-code |
| 短信验证码登录 | POST | /admin-api/uaa/auth/sms-login |
| 退出登录 | POST | /admin-api/uaa/auth/logout |

---

【问题诊断】
如果登录失败:
- 检查 Network 面板 POST /admin-api/uaa/auth/login 请求的状态码和响应体
- 检查请求头是否包含 tenant-id: 1
- 检查后端 UAA 服务是否在 :8081 运行
- 检查数据库中 admin 用户的状态是否为正常(status=0)
```

#### 5.1.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【登录与登出】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: /login
- 路由: /login
- 权限前缀: 无（公开页面）
- 涉及权限码:
  - 无（公开页面）

【前端文件清单】
- 主页面: business-web/src/views/login/index.vue
- API 封装: business-web/src/api/core/auth.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/auth/AuthController.java
- Service 接口: .../service/auth/AuthService.java
- DTO/Request: .../controller/admin/auth/vo/*ReqVO.java
- DO: .../dal/dataobject/auth/*DO.java
- Mapper: .../dal/mapper/auth/*Mapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 账号密码登录 | POST | /admin-api/uaa/auth/login |
| 获取验证码 | POST | /admin-api/uaa/auth/send-code |
| 短信验证码登录 | POST | /admin-api/uaa/auth/sms-login |
| 退出登录 | POST | /admin-api/uaa/auth/logout |

【功能需求】
1. 支持账号密码登录:用户名、密码、租户 ID
2. 支持短信验证码登录:手机号、验证码、租户 ID
3. 支持第三方登录:微信、GitHub、钉钉
4. 登录成功后跳转 /dashboard,失败显示错误提示
5. Token 持久化到 localStorage,刷新页面自动跳转

【UI 规范】
- UI 库: ant-design-vue
- 表单字段:用户名、密码、租户ID(账号密码登录);手机号、验证码(短信登录)
- 业务组件复用: 无
- 操作按钮:登录、获取验证码

【参考实现】
请参考以下已实现模块:
- business-web/src/views/login/index.vue(登录页主入口)
- business-web/src/api/core/auth.ts(API 封装)
- 复用 business-web/src/components/business 中的业务组件

【测试验证】
实现完成后,使用 5.1.2 测试提示词中的测试场景验证,重点验证:
1. 6 个测试场景完整通过
2. 账号密码登录、短信登录、第三方登录均可用
3. Token 持久化与刷新自动跳转正常
4. 错误凭证与空值校验提示正确

【交付物清单】
- [ ] 前端主页面 .vue(login/index.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/login)
- [ ] 通过 5.1.2 所有测试场景
```

---

### 5.2 UAA-02 用户管理

**页面路径**: 左侧菜单「用户中心」→「用户管理」
**源码文件**: `src/views/uaa/user/index.vue`, `src/views/uaa/user/UserFormModal.vue`, `src/views/uaa/user/UserRoleModal.vue`, `src/views/uaa/user/UserImportForm.vue`
**API 文件**: `src/api/uaa/user.ts`, `src/api/uaa/dept.ts`, `src/api/uaa/post.ts`
**权限标识**: `system:user:create`, `system:user:update`, `system:user:delete`, `system:user:export`, `system:user:import`, `system:permission:assign-user-role`

#### 5.2.1 测试场景

#### 5.2.2 测试提示词

```
/browser 或 /open-gstack-browser
打开用户管理页面,执行完整的用户管理功能测试。

【前置操作】
1. 使用 admin/admin123 登录系统
2. 在左侧菜单找到并点击「用户中心」→「用户管理」
3. 等待页面加载完成,确保表格数据渲染完成

---

【测试场景 1:列表加载与基础显示】
1. 验证表格列完整性(9列):用户名、头像、昵称、部门、手机号、性别、状态、创建时间、操作
2. 验证性别列使用 DictTag 标签显示(非数字)
3. 验证状态列使用 DictSwitch 开关组件
4. 验证分页器正常工作
5. 验证操作列按钮顺序:授权 → 编辑 → 删除

预期结果:
✅ 表格9列完整,数据正确渲染
✅ 性别列显示为 DictTag 标签
✅ 状态列显示为 DictSwitch 开关
✅ 分页器正常工作
✅ 操作列按钮顺序正确

---

【测试场景 2:搜索与重置】
1. 在「用户名/昵称」输入框输入关键词,点击「搜索」
2. 在「状态」下拉框选择一个状态,点击「搜索」
3. 点击「重置」,验证条件清空,列表恢复

预期结果:
✅ 搜索过滤正确
✅ 重置功能正常

---

【测试场景 3:新增用户】
1. 点击「新增」按钮,验证弹窗标题为「新增用户」
2. 验证表单字段完全空白(无上次编辑数据残留)
3. 填写:用户名、昵称、密码、手机号、性别、状态
4. 提交并验证 API 调用 /admin-api/uaa/user/create 成功
5. 验证弹窗关闭,列表自动刷新

预期结果:
✅ 新增弹窗打开时表单完全空白
✅ 表单提交成功,列表自动刷新

---

【测试场景 4:编辑用户】
1. 点击「编辑」按钮
2. 验证弹窗弹出,数据正确回填(用户名、昵称、手机号等)
3. 修改昵称,提交并验证 API /admin-api/uaa/user/update 成功
4. 验证列表自动刷新,数据已更新

预期结果:
✅ 编辑弹窗正确回填数据
✅ 修改提交后列表刷新,数据更新

---

【测试场景 5:状态切换】
1. 点击状态 DictSwitch,将用户从启用切换为禁用
2. 验证 API /admin-api/uaa/user/update-status 成功
3. 验证状态列立即更新显示

预期结果:
✅ 状态切换 API 调用成功
✅ 状态列实时更新

---

【测试场景 6:用户角色授权】
1. 点击「授权」按钮,验证弹窗标题为「用户角色授权」
2. 验证角色列表使用 CheckboxGroup 复选框显示
3. 勾选/取消勾选部分角色,点击确定
4. 验证 API /admin-api/uaa/permission/assign-user-role 成功

预期结果:
✅ 授权弹窗正常打开
✅ 复选框勾选状态正确
✅ 提交授权成功

---

【测试场景 7:删除用户】
1. 点击「删除」按钮,验证弹出确认对话框
2. 点击「确定」确认删除,验证 API /admin-api/uaa/user/delete 成功
3. 验证列表自动刷新,用户从列表中消失

预期结果:
✅ 删除确认弹窗正常
✅ 删除成功后列表刷新,用户消失

---

【测试场景 8:导出用户】
1. 点击「导出」按钮,验证触发文件下载
2. 验证导出的 Excel 包含所有用户字段

预期结果:
✅ 导出功能正常,文件下载触发

---

【测试场景 9:用户列表分页】
1. 验证默认页码与每页条数
2. 点击下一页,验证数据切换
3. 切换每页条数为 50,验证数据量

预期结果:
✅ 分页参数正确
✅ 翻页/切换每页条数正常

---

【测试场景 10:重置密码】
1. 点击操作列的「重置密码」按钮(如有)
2. 验证弹窗弹出,提示输入新密码
3. 输入新密码并确认,验证 API /admin-api/uaa/user/update-password 成功

预期结果:
✅ 重置密码弹窗正常
✅ 密码重置 API 调用成功

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表分页 | GET | /admin-api/uaa/user/page |
| 用户详情 | GET | /admin-api/uaa/user/get?id=X |
| 创建用户 | POST | /admin-api/uaa/user/create |
| 更新用户 | PUT | /admin-api/uaa/user/update |
| 删除用户 | DELETE | /admin-api/uaa/user/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/user/delete-list?ids=X,Y |
| 重置密码 | PUT | /admin-api/uaa/user/update-password |
| 更新状态 | PUT | /admin-api/uaa/user/update-status |
| 导出用户 | GET | /admin-api/uaa/user/export-excel |
| 获取导入模板 | GET | /admin-api/uaa/user/get-import-template |
| 导入用户 | POST | /admin-api/uaa/user/import |
| 获取用户角色 | GET | /admin-api/uaa/permission/list-user-roles |
| 分配用户角色 | POST | /admin-api/uaa/permission/assign-user-role |
| 精简列表 | GET | /admin-api/uaa/user/list-all-simple |

---

【问题诊断】
- 性别列显示数字 → 检查 DictTag type 是否为 sys_user_sex,字段绑定是否为 record.sex
- 状态切换不生效 → 检查 before-change 回调是否正确调用 updateStatus API
- 编辑弹窗数据不回填 → 检查 UserFormModal.vue 的 watch 是否监听 formData,是否使用 immediate: true
- 授权提交失败 → 检查请求体格式是否为 { userId, roleIds }
- 新增表单有残留数据 → 检查弹窗是否设置 :destroy-on-close="true"
```

#### 5.2.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【用户管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「用户管理」
- 路由: /uaa/user
- 权限前缀: system:user
- 涉及权限码:
  - system:user:create(新增用户)
  - system:user:update(编辑用户)
  - system:user:delete(删除用户)
  - system:user:query(查询用户)
  - system:user:export(导出用户)
  - system:user:import(导入用户)
  - system:user:update-password(重置密码)
  - system:permission:assign-user-role(分配用户角色)

【前端文件清单】
- 主页面: business-web/src/views/uaa/user/index.vue
- 表单弹窗: business-web/src/views/uaa/user/UserFormModal.vue
- 角色授权: business-web/src/views/uaa/user/UserRoleModal.vue
- 导入弹窗: business-web/src/views/uaa/user/UserImportForm.vue
- API 封装: business-web/src/api/uaa/user.ts
- 辅助 API: business-web/src/api/uaa/dept.ts, business-web/src/api/uaa/post.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/user/UserController.java
- Service 接口: .../service/user/AdminUserService.java
- DTO/Request: .../controller/admin/user/vo/user/*ReqVO.java
- DO: .../dal/dataobject/user/AdminUserDO.java
- Mapper: .../dal/mapper/user/AdminUserMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | /admin-api/uaa/user/page |
| 用户详情 | GET | /admin-api/uaa/user/get?id=X |
| 创建用户 | POST | /admin-api/uaa/user/create |
| 更新用户 | PUT | /admin-api/uaa/user/update |
| 删除用户 | DELETE | /admin-api/uaa/user/delete?id=X |
| 重置密码 | PUT | /admin-api/uaa/user/update-password |
| 更新状态 | PUT | /admin-api/uaa/user/update-status |
| 导出用户 | GET | /admin-api/uaa/user/export-excel |
| 获取导入模板 | GET | /admin-api/uaa/user/get-import-template |
| 导入用户 | POST | /admin-api/uaa/user/import |

【功能需求】
1. 支持用户列表展示:用户名、头像、昵称、部门、手机号、性别、状态、创建时间、操作
2. 性别列使用 DictTag(sys_user_sex),状态列使用 DictSwitch
3. 搜索支持:用户名/昵称、状态
4. 新增/编辑用户:用户名、昵称、密码、手机号、性别、状态
5. 用户角色授权(独立弹窗)
6. 导出与导入用户(Excel 模板)
7. 重置密码

【UI 规范】
- UI 库: ant-design-vue
- 表格列:用户名、头像、昵称、部门、手机号、性别、状态、创建时间、操作
- 字典类型: sys_user_sex(性别)、common_status(状态)
- 业务组件复用: DictTag、DictSwitch、DictSelect
- 操作按钮:授权、编辑、删除、重置密码

【参考实现】
请参考以下已实现模块:
- business-web/src/views/system/dict/index.vue(字典管理,适合作为表单+下拉选择参考)
- business-web/src/views/uaa/role/index.vue(角色管理,适合作为弹窗交互参考)
- 复用 business-web/src/components/business 中的 DictTag/DictSwitch 组件

【测试验证】
实现完成后,使用 5.2.2 测试提示词中的测试场景验证,重点验证:
1. 表格 9 列完整渲染
2. 性别列 DictTag、状态列 DictSwitch 显示正确
3. 新增/编辑/删除流程完整
4. 角色授权提交成功
5. 导出/导入功能正常
6. 10 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(user/index.vue)
- [ ] 前端表单弹窗 .vue(UserFormModal.vue)
- [ ] 前端角色授权弹窗 .vue(UserRoleModal.vue)
- [ ] 前端导入弹窗 .vue(UserImportForm.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/user)
- [ ] 菜单注册(左侧菜单「用户中心」→「用户管理」)
- [ ] 权限码注册(system:user:create 等)
- [ ] 通过 5.2.2 所有测试场景
```

---

### 5.3 UAA-03 角色管理

**页面路径**: 左侧菜单「用户中心」→「角色管理」
**源码文件**: `src/views/uaa/role/index.vue`, `src/views/uaa/role/RoleForm.vue`
**API 文件**: `src/api/uaa/role.ts`, `src/api/uaa/menu.ts`
**权限标识**: `system:role:create`, `system:role:update`, `system:role:delete`, `system:permission:assign-role-menu`

#### 5.3.1 测试场景

#### 5.3.2 测试提示词

```
/browser 或 /open-gstack-browser
打开角色管理页面,执行完整的角色 CRUD + 赋权测试。

【前置操作】
1. 使用 admin/admin123 登录系统
2. 在左侧菜单点击「用户中心」→「角色管理」
3. 等待页面加载完成

---

【测试场景 1:列表加载与基础显示】
1. 验证角色列表表格正常加载
2. 检查列:角色名称、角色编码、排序、数据权限、状态、备注、创建时间、操作(共 7 列,固定右侧)
3. 验证数据权限列的 DictTag 标签(ALL=全部数据/DEPT_AND_CHILD=本部门及下级/DEPT=本部门/SELF=仅本人/CUSTOM=自定义)
4. 验证状态列 DictTag(status=0 开启=绿色/status=1 关闭=灰色)
5. 验证分页器显示"共 X 条",分页切换功能正常
6. 验证操作列按钮顺序:编辑 → 赋权 → 删除

预期结果:
✅ 表格列完整,fixed right 不影响其他列
✅ 数据权限 DictTag 正确映射
✅ 状态 DictTag 正确(绿/灰)
✅ 操作按钮顺序:编辑 → 赋权 → 删除

---

【测试场景 2:搜索功能】
1. 在「角色名称」输入框输入关键词,点击「搜索」
2. 在「角色编码」输入框输入关键词,点击「搜索」
3. 在「状态」下拉框选择「开启」或「关闭」,点击「搜索」
4. 点击「重置」,验证条件全部清空,列表恢复初始状态

预期结果:
✅ 角色名称搜索过滤正确
✅ 角色编码搜索过滤正确
✅ 状态下拉筛选正确
✅ 重置清空全部条件并刷新列表

---

【测试场景 3:新增角色(含菜单权限树)】
1. 点击顶部「新增」按钮
2. 验证 RoleForm 弹窗弹出,标题为「新增角色」
3. 验证表单字段(8个):角色名称、角色编码、排序、数据权限、菜单权限、状态、备注
4. 填写信息:角色名称、角色编码、排序、数据权限、状态
5. 在 RoleMenuTree 组件中勾选部分菜单(至少包含一个目录节点 + 一个叶子节点)
6. 验证 RoleForm 使用 :destroy-on-close="true"
7. 提交并验证:两次 API 调用均成功,显示「新增成功」,弹窗关闭,列表自动刷新

预期结果:
✅ 新增弹窗表单 8 个字段完整显示
✅ RoleMenuTree 加载正常,显示树形结构
✅ 两次 API 调用顺序正确
✅ 提交成功,列表自动刷新

---

【测试场景 4:编辑角色(含菜单权限树回显)】
1. 找到刚创建的角色,点击「编辑」
2. 验证弹窗弹出,数据正确回填
3. 验证菜单权限树中该角色已有的菜单被勾选
4. 修改角色名称和排序
5. 提交并验证 API 均成功

预期结果:
✅ 编辑弹窗标题正确(修改角色)
✅ 所有字段正确回填
✅ RoleMenuTree 权限回显正确
✅ 两次 API 调用均成功

---

【测试场景 5:角色赋权】
1. 找到任意角色,点击「赋权」按钮
2. 验证 RoleAssignMenuModal 弹窗弹出,标题为「菜单权限分配」
3. 验证菜单树完整加载
4. 勾选/取消部分菜单
5. 点击「确定」,验证 API /admin-api/uaa/permission/assign-role-menu 成功

预期结果:
✅ 赋权弹窗正常打开
✅ 菜单树加载正常
✅ 赋权提交成功

---

【测试场景 6:删除角色】
1. 点击「删除」按钮,验证 Modal.confirm 弹出,文本为:「确认删除该角色?删除后关联用户的角色将被解除。」
2. 点击「确认」,验证 API DELETE /admin-api/uaa/role/delete?id=X 成功
3. 验证显示「删除成功」message
4. 验证列表刷新,角色从列表中消失

预期结果:
✅ 删除确认文本与源码一致
✅ 删除成功后角色消失

---

【测试场景 7:角色编码唯一性校验】
1. 新增一个角色,编码设为已有编码
2. 点击提交,验证后端返回错误
3. 验证弹窗不关闭
4. 修改编码为唯一值,再次提交,验证成功

预期结果:
✅ 后端唯一性校验生效
✅ 错误提示清晰,弹窗不关闭

---

【测试场景 8:数据权限分配】
1. 点击操作列的「数据权限」按钮
2. 验证 RoleDataPermissionForm 弹窗弹出,标题为「数据权限」
3. 验证 5 种数据权限范围单选框:全部数据、本部门及下级、本部门、仅本人、自定义
4. 选择「自定义」,验证部门树完整加载
5. 勾选 3-5 个部门,点击「确定」
6. 验证 API POST /admin-api/uaa/permission/assign-role-data-scope 成功

预期结果:
✅ 数据权限弹窗正常
✅ 5 种权限范围选项完整
✅ 部门树加载并支持勾选
✅ 提交成功

---

【测试场景 9:状态切换】
1. 点击状态 DictSwitch,将角色从启用切换为禁用
2. 验证 API PUT /admin-api/uaa/role/update-status 成功
3. 验证状态列立即更新

预期结果:
✅ 状态切换 API 调用成功
✅ 状态列实时更新

---

【测试场景 10:分页与排序】
1. 验证默认分页参数
2. 点击排序列(角色名称/角色编码/排序号),验证排序方向切换
3. 切换每页条数,验证数据量

预期结果:
✅ 分页参数正确
✅ 排序功能正常

---

【测试场景 11:列表刷新】
1. 创建一个新角色
2. 验证列表自动刷新,新角色出现在第一行

预期结果:
✅ 列表自动刷新

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表分页 | GET | /admin-api/uaa/role/page |
| 角色详情 | GET | /admin-api/uaa/role/get?id=X |
| 创建角色 | POST | /admin-api/uaa/role/create |
| 更新角色 | PUT | /admin-api/uaa/role/update |
| 删除角色 | DELETE | /admin-api/uaa/role/delete?id=X |
| 更新状态 | PUT | /admin-api/uaa/role/update-status |
| 获取角色菜单 | GET | /admin-api/uaa/permission/list-role-menus?roleId=X |
| 分配菜单权限 | POST | /admin-api/uaa/permission/assign-role-menu |
| 分配数据权限 | POST | /admin-api/uaa/permission/assign-role-data-scope |
| 角色精简列表 | GET | /admin-api/uaa/role/list-all-simple |

> 注意:`assign-role-menu` 是单数,不是 `assign-role-menus`。

---

【问题诊断】
- 权限树不显示 → 检查 menuApi.list() 是否成功,检查 buildMenuTree 是否返回了树数据
- 编辑时菜单树数据不回填 → 检查 RoleForm.vue 的 watch 中是否调用了 roleApi.getRoleMenus(id)
- 赋权提交失败(403) → 检查权限码是否为 `system:permission:assign-role-menu`
- 赋权提交失败(参数错误) → 检查请求体格式是否为 `{ roleId, menuIds }`
- 弹窗打开时表单未重置 → 检查 RoleForm.vue 中 watch visible 时是否调用了 createDefault()
- 树形不展开 → 检查 a-tree 的 `:default-expand-all="true"` 属性和 field-names 配置
```

#### 5.3.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【角色管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「角色管理」
- 路由: /uaa/role
- 权限前缀: system:role
- 涉及权限码:
  - system:role:create(新增角色)
  - system:role:update(编辑角色)
  - system:role:delete(删除角色)
  - system:role:query(查询角色)
  - system:permission:assign-role-menu(分配菜单权限)
  - system:permission:assign-role-data-scope(分配数据权限)

【前端文件清单】
- 主页面: business-web/src/views/uaa/role/index.vue
- 表单弹窗: business-web/src/views/uaa/role/RoleForm.vue
- 菜单赋权: business-web/src/views/uaa/role/RoleAssignMenuModal.vue
- 菜单树: business-web/src/views/uaa/role/RoleMenuTree.vue
- 数据权限: business-web/src/views/uaa/role/RoleDataPermissionForm.vue
- API 封装: business-web/src/api/uaa/role.ts, business-web/src/api/uaa/menu.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/permission/RoleController.java
- Service 接口: .../service/permission/RoleService.java
- DTO/Request: .../controller/admin/permission/vo/role/*ReqVO.java
- DO: .../dal/dataobject/permission/RoleDO.java
- Mapper: .../dal/mapper/permission/RoleMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | /admin-api/uaa/role/page |
| 角色详情 | GET | /admin-api/uaa/role/get?id=X |
| 创建角色 | POST | /admin-api/uaa/role/create |
| 更新角色 | PUT | /admin-api/uaa/role/update |
| 删除角色 | DELETE | /admin-api/uaa/role/delete?id=X |
| 更新状态 | PUT | /admin-api/uaa/role/update-status |
| 分配菜单权限 | POST | /admin-api/uaa/permission/assign-role-menu |
| 分配数据权限 | POST | /admin-api/uaa/permission/assign-role-data-scope |
| 获取角色菜单 | GET | /admin-api/uaa/permission/list-role-menus?roleId=X |

【功能需求】
1. 支持角色列表展示:角色名称、角色编码、排序、数据权限、状态、备注、创建时间、操作
2. 数据权限使用 DictTag,5 种范围:全部/本部门及下级/本部门/仅本人/自定义
3. 状态使用 DictTag,DictSwitch 切换
4. 新增/编辑角色,内置菜单权限树选择
5. 独立赋权弹窗,支持父子联动
6. 数据权限分配,支持部门树选择(自定义范围)
7. 角色编码唯一性校验

【UI 规范】
- UI 库: ant-design-vue
- 表格列:角色名称、角色编码、排序、数据权限、状态、备注、创建时间、操作
- 字典类型: system_data_scope(数据权限)、common_status(状态)
- 业务组件复用: RoleMenuTree、RoleDataPermissionForm
- 操作按钮:编辑、赋权、数据权限、删除

【参考实现】
请参考以下已实现模块:
- business-web/src/views/system/dict/index.vue(字典管理,适合作为表单+下拉选择参考)
- business-web/src/views/uaa/menu/index.vue(菜单管理,适合作为树形 CRUD 参考)
- 复用 business-web/src/components/business 中的 DictTag/DictSwitch/DictSelect 组件

【测试验证】
实现完成后,使用 5.3.2 测试提示词中的测试场景验证,重点验证:
1. 表格 7 列完整渲染,操作列 fixed right
2. 数据权限 DictTag 正确映射 5 种范围
3. 菜单权限树父子联动正常
4. 数据权限分配部门树加载正常
5. 角色编码唯一性校验生效
6. 11 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(role/index.vue)
- [ ] 前端表单弹窗 .vue(RoleForm.vue)
- [ ] 前端菜单赋权弹窗 .vue(RoleAssignMenuModal.vue)
- [ ] 前端菜单树组件 .vue(RoleMenuTree.vue)
- [ ] 前端数据权限表单 .vue(RoleDataPermissionForm.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/role)
- [ ] 菜单注册(左侧菜单「用户中心」→「角色管理」)
- [ ] 权限码注册(system:role:create 等)
- [ ] 通过 5.3.2 所有测试场景
```

---

### 5.4 UAA-04 角色菜单赋权

**页面路径**: 左侧菜单「用户中心」→「角色管理」→「赋权」按钮
**源码文件**: `src/views/uaa/role/RoleAssignMenuModal.vue`, `src/views/uaa/role/RoleMenuTree.vue`
**API 文件**: `src/api/uaa/role.ts`, `src/api/uaa/menu.ts`
**权限标识**: `system:permission:assign-role-menu`

#### 5.4.1 测试场景

#### 5.4.2 测试提示词

```
/browser 或 /open-gstack-browser
打开角色管理页面,测试角色菜单赋权功能。

【前置操作】
1. 登录后进入角色管理页面
2. 找到任意角色,点击「赋权」按钮
3. 等待 RoleAssignMenuModal 弹窗弹出

---

【测试场景 1:弹窗基础结构】
1. 验证弹窗标题固定为「菜单权限分配」
2. 验证弹窗顶部显示角色信息区(角色名称)
3. 验证加载状态短暂出现后消失
4. 验证菜单树完整加载(使用 buildMenuTree 转为树形)
5. 验证已有菜单被勾选

预期结果:
✅ 赋权弹窗标题固定为「菜单权限分配」
✅ 角色名称正确显示
✅ 菜单树完整加载
✅ 已有菜单正确勾选

---

【测试场景 2:父子联动】
1. 勾选一个父级节点
2. 验证所有子节点被自动联级勾选
3. 取消勾选该父级节点
4. 验证子节点保持原勾选状态不变

预期结果:
✅ 子→父联动正确
✅ 取消不反向联动

---

【测试场景 3:清空与取消操作】
1. 随意勾选和取消一些节点
2. 点击「清空」按钮,验证所有勾选清除
3. 重新勾选 3-5 个节点
4. 点击「取消」,验证弹窗关闭,无 API 请求发出

预期结果:
✅ 清空功能正常
✅ 取消不触发 API

---

【测试场景 4:空选择赋权提交】
1. 清空所有勾选(或不勾选任何节点)
2. 点击「确定」
3. 验证 API 发送 menuIds 为空数组 `[]`
4. 验证显示「权限分配成功」

预期结果:
✅ 空数组可以正常提交
✅ 清空所有权限功能正常

---

【测试场景 5:展开/折叠菜单树】
1. 验证菜单树默认全部展开
2. 折叠某个父级节点,验证子节点隐藏
3. 再次展开,验证子节点显示

预期结果:
✅ 树形展开/折叠功能正常

---

【测试场景 6:多选节点提交】
1. 勾选 5-10 个不同层级(目录/菜单/按钮)的菜单节点
2. 点击「确定」
3. 验证 API 请求体 menuIds 数组包含所有勾选节点 ID
4. 验证显示「权限分配成功」

预期结果:
✅ 多选节点全部提交
✅ API 请求体正确

---

【测试场景 7:权限回显校验】
1. 关闭弹窗,重新打开同一角色的「赋权」
2. 验证上一步勾选的菜单全部回显
3. 验证未勾选的菜单未回显

预期结果:
✅ 已有权限正确回显
✅ 关闭重开后数据一致

---

【测试场景 8:加载失败兜底】
1. 模拟菜单 API 返回失败(后端关闭服务)
2. 打开赋权弹窗
3. 验证加载状态结束,菜单树为空但弹窗不报错
4. 验证「清空」「取消」按钮仍可用

预期结果:
✅ 加载失败时弹窗不报错
✅ 空状态下交互按钮仍可用

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 获取角色菜单 | GET | /admin-api/uaa/permission/list-role-menus?roleId=X |
| 分配菜单权限 | POST | /admin-api/uaa/permission/assign-role-menu |
| 菜单列表(树形) | GET | /admin-api/uaa/menu/list |

---

【问题诊断】
- 父子联动不正常 → 检查 RoleAssignMenuModal.vue 中 handleTreeCheck 里 findAncestorIds 的返回值
- 赋权后再次打开勾选状态丢失 → 检查 checkedKeys 是否正确绑定到 a-tree 的 `:checked-keys`
- 树形不展开 → 检查 a-tree 的 `:default-expand-all="true"` 属性
```

#### 5.4.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【角色菜单赋权】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「角色管理」→「赋权」按钮(弹窗模式)
- 路由: /uaa/role(列表)+ RoleAssignMenuModal.vue(弹窗)
- 权限前缀: system:permission
- 涉及权限码:
  - system:permission:assign-role-menu(分配角色菜单权限)
  - system:role:query(查询角色)

【前端文件清单】
- 主页面: business-web/src/views/uaa/role/index.vue
- 菜单赋权弹窗: business-web/src/views/uaa/role/RoleAssignMenuModal.vue
- 菜单树组件: business-web/src/views/uaa/role/RoleMenuTree.vue
- API 封装: business-web/src/api/uaa/role.ts, business-web/src/api/uaa/menu.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/permission/PermissionController.java
- Service 接口: .../service/permission/PermissionService.java
- DTO/Request: .../controller/admin/permission/vo/*ReqVO.java
- DO: .../dal/dataobject/permission/RoleMenuDO.java
- Mapper: .../dal/mapper/permission/RoleMenuMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 获取角色菜单 | GET | /admin-api/uaa/permission/list-role-menus?roleId=X |
| 分配菜单权限 | POST | /admin-api/uaa/permission/assign-role-menu |
| 菜单列表 | GET | /admin-api/uaa/menu/list |

【功能需求】
1. 弹窗打开时加载该角色已有菜单权限,自动勾选
2. 菜单树形展示,支持父子联动(勾父自动勾子,取消不反向联动)
3. 支持清空与取消
4. 提交时仅发送叶子节点(按钮),父节点由后端根据叶子节点关系计算
5. 权限分配成功后弹窗关闭,列表不刷新(仅权限变化)

【UI 规范】
- UI 库: ant-design-vue
- 弹窗组件: a-modal + a-tree
- 树形组件: a-tree,checkable,default-expand-all
- 操作按钮:清空、取消、确定

【参考实现】
请参考以下已实现模块:
- business-web/src/views/uaa/role/RoleAssignMenuModal.vue(本模块目标文件)
- business-web/src/views/uaa/role/RoleMenuTree.vue(本模块目标文件)
- 复用 business-web/src/components/business 中的树形组件

【测试验证】
实现完成后,使用 5.4.2 测试提示词中的测试场景验证,重点验证:
1. 弹窗基础结构与菜单树加载
2. 父子联动正常(子→父联动,取消不反向)
3. 清空与取消不触发 API
4. 多选节点提交请求体正确
5. 权限回显与一致性
6. 8 个测试场景全部通过

【交付物清单】
- [ ] 前端菜单赋权弹窗 .vue(RoleAssignMenuModal.vue)
- [ ] 前端菜单树组件 .vue(RoleMenuTree.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 权限码注册(system:permission:assign-role-menu)
- [ ] 通过 5.4.2 所有测试场景
```

---

### 5.5 UAA-05 菜单管理

**页面路径**: 左侧菜单「用户中心」→「菜单管理」
**源码文件**: `src/views/uaa/menu/index.vue`, `src/views/uaa/menu/MenuForm.vue`
**API 文件**: `src/api/uaa/menu.ts`
**权限标识**: `system:menu:create`, `system:menu:update`, `system:menu:delete`

#### 5.5.1 测试场景

#### 5.5.2 测试提示词

```
/browser 或 /open-gstack-browser
打开菜单管理页面,执行完整的菜单管理功能测试。

【前置操作】
1. 使用 admin/admin123 登录系统
2. 在左侧菜单点击「用户中心」→「菜单管理」
3. 等待页面加载完成

---

【测试场景 1:树形列表加载】
1. 验证菜单树形列表正常加载
2. 验证菜单类型列显示:目录/菜单/按钮
3. 验证状态列显示:开启/关闭
4. 验证操作列:编辑、删除
5. 点击目录节点左侧箭头,验证展开/折叠正常

预期结果:
✅ 树形列表完整加载
✅ 菜单类型正确显示
✅ 展开折叠功能正常

---

【测试场景 2:新增目录】
1. 点击「新增」按钮
2. 验证弹窗弹出,选择父级菜单(选择根目录或已有目录)
3. 填写:菜单名称、菜单编码、排序、菜单类型(目录)
4. 验证目录类型时无需填写组件路径
5. 提交并验证 API POST /admin-api/uaa/menu/create 成功
6. 验证树形列表自动刷新,新目录出现在对应父节点下

预期结果:
✅ 新增目录弹窗正常
✅ 目录类型无需组件路径
✅ 树形列表自动刷新

---

【测试场景 3:新增菜单(带组件)】
1. 点击「新增」按钮,选择父级为刚创建的目录
2. 选择菜单类型为「菜单」
3. 填写:菜单名称、路由地址、组件路径、图标
4. 验证菜单类型时需要填写组件路径
5. 提交并验证成功

预期结果:
✅ 菜单类型必填字段正确
✅ 组件路径字段显示

---

【测试场景 4:新增按钮权限】
1. 点击「新增」按钮,选择父级为菜单
2. 选择菜单类型为「按钮」
3. 填写:权限标识(如 system:user:create)、按钮名称
4. 验证按钮类型时无需填写路由和组件
5. 提交并验证成功

预期结果:
✅ 按钮类型无需路由和组件
✅ 权限标识字段正确

---

【测试场景 5:编辑菜单】
1. 点击任意菜单的「编辑」按钮
2. 验证弹窗弹出,数据正确回填
3. 修改菜单名称
4. 提交并验证 API PUT /admin-api/uaa/menu/update 成功
5. 验证树形列表自动刷新,修改生效

预期结果:
✅ 编辑回填正确
✅ 修改提交成功

---

【测试场景 6:删除菜单】
1. 找到刚创建的测试菜单(含子节点)
2. 点击「删除」按钮
3. 验证弹出确认弹窗
4. 点击确认,验证 API DELETE /admin-api/uaa/menu/delete?id=X 成功
5. 验证树形列表自动刷新,菜单消失

预期结果:
✅ 删除确认正常
✅ 删除成功后列表刷新

---

【测试场景 7:菜单类型切换字段联动】
1. 点击「新增」按钮
2. 切换菜单类型:目录 → 菜单 → 按钮
3. 验证:目录类型时,路由地址、组件路径、权限标识隐藏
4. 验证:菜单类型时,组件路径必填,权限标识可选
5. 验证:按钮类型时,权限标识必填,路由地址、组件路径隐藏

预期结果:
✅ 菜单类型切换字段联动正确
✅ 必填字段校验正确

---

【测试场景 8:搜索菜单】
1. 在「菜单名称」输入框输入关键词,点击「搜索」
2. 验证列表过滤出匹配的菜单节点(树形结构保留)

预期结果:
✅ 菜单搜索过滤正确
✅ 树形结构保留

---

【测试场景 9:权限标识唯一性校验】
1. 新增按钮,权限标识设为已有标识
2. 提交验证:后端返回重复错误
3. 修改权限标识为唯一值,再次提交验证成功

预期结果:
✅ 权限标识唯一性校验生效
✅ 错误提示清晰

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 菜单列表(树形) | GET | /admin-api/uaa/menu/list |
| 菜单精简列表 | GET | /admin-api/uaa/menu/list-all-simple |
| 菜单详情 | GET | /admin-api/uaa/menu/get?id=X |
| 创建菜单 | POST | /admin-api/uaa/menu/create |
| 更新菜单 | PUT | /admin-api/uaa/menu/update |
| 删除菜单 | DELETE | /admin-api/uaa/menu/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/menu/delete-list?ids=X,Y |

---

【问题诊断】
- 树形不显示 → 检查 menuApi.list() 返回数据格式,buildMenuTree 函数逻辑
- 父子节点上下级联动错误 → 检查 handleTree 函数的 parentId 关联逻辑
- 新增/编辑失败 → 检查后端是否返回校验错误(如权限标识唯一性)
```

#### 5.5.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【菜单管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「菜单管理」
- 路由: /uaa/menu
- 权限前缀: system:menu
- 涉及权限码:
  - system:menu:create(新增菜单)
  - system:menu:update(编辑菜单)
  - system:menu:delete(删除菜单)
  - system:menu:query(查询菜单)

【前端文件清单】
- 主页面: business-web/src/views/uaa/menu/index.vue
- 表单弹窗: business-web/src/views/uaa/menu/MenuForm.vue
- API 封装: business-web/src/api/uaa/menu.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/permission/MenuController.java
- Service 接口: .../service/permission/MenuService.java
- DTO/Request: .../controller/admin/permission/vo/menu/*ReqVO.java
- DO: .../dal/dataobject/permission/MenuDO.java
- Mapper: .../dal/mapper/permission/MenuMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 菜单列表 | GET | /admin-api/uaa/menu/list |
| 菜单精简列表 | GET | /admin-api/uaa/menu/list-all-simple |
| 菜单详情 | GET | /admin-api/uaa/menu/get?id=X |
| 创建菜单 | POST | /admin-api/uaa/menu/create |
| 更新菜单 | PUT | /admin-api/uaa/menu/update |
| 删除菜单 | DELETE | /admin-api/uaa/menu/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/menu/delete-list?ids=X,Y |

【功能需求】
1. 支持菜单树形 CRUD:目录/菜单/按钮三种类型
2. 菜单类型切换字段联动:目录无需组件路径,菜单需组件路径,按钮需权限标识
3. 权限标识唯一性校验
4. 树形列表展开/折叠
5. 父子节点上下级联动
6. 搜索菜单(树形结构保留)

【UI 规范】
- UI 库: ant-design-vue
- 表格列:菜单名称、图标、排序、权限标识、组件路径、菜单类型、状态、操作
- 字典类型: system_menu_type(菜单类型)、common_status(状态)
- 业务组件复用: IconSelect(图标选择器)
- 操作按钮:新增、编辑、删除

【参考实现】
请参考以下已实现模块:
- business-web/src/views/system/dict/index.vue(字典管理,适合作为表单+下拉选择参考)
- business-web/src/views/uaa/role/index.vue(角色管理,适合作为树形 CRUD 参考)
- 复用 business-web/src/components/business 中的 IconSelect 组件

【测试验证】
实现完成后,使用 5.5.2 测试提示词中的测试场景验证,重点验证:
1. 树形列表完整加载
2. 目录/菜单/按钮三种类型新增正常
3. 菜单类型切换字段联动正确
4. 权限标识唯一性校验生效
5. 9 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(menu/index.vue)
- [ ] 前端表单弹窗 .vue(MenuForm.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/menu)
- [ ] 菜单注册(左侧菜单「用户中心」→「菜单管理」)
- [ ] 权限码注册(system:menu:create 等)
- [ ] 字典数据初始化 SQL(system_menu_type)
- [ ] 通过 5.5.2 所有测试场景
```

---

### 5.6 UAA-06 部门管理

**页面路径**: 左侧菜单「用户中心」→「部门管理」
**源码文件**: `src/views/uaa/dept/index.vue`
**API 文件**: `src/api/uaa/dept.ts`
**权限标识**: `system:dept:create`, `system:dept:update`, `system:dept:delete`

#### 5.6.1 测试场景

#### 5.6.2 测试提示词

```
/browser 或 /open-gstack-browser
打开部门管理页面,执行完整的部门管理功能测试。

【前置操作】
1. 登录后进入「用户中心」→「部门管理」
2. 等待部门树形列表加载完成

---

【测试场景 1:部门树加载】
1. 验证部门树形列表正常加载
2. 验证每个部门节点显示部门名称、负责人姓名
3. 点击部门节点左侧箭头,验证展开/折叠正常

预期结果:
✅ 树形列表完整加载
✅ 负责人信息正确显示
✅ 展开折叠功能正常

---

【测试场景 2:新增部门】
1. 点击「新增」按钮(或在树节点上右键新增)
2. 选择父级部门
3. 填写:部门名称、排序、负责人、手机号、邮箱
4. 提交并验证 API POST /admin-api/uaa/dept/create 成功
5. 验证树形列表自动刷新

预期结果:
✅ 新增部门成功
✅ 树形列表自动刷新

---

【测试场景 3:编辑部门】
1. 点击部门节点的「编辑」按钮
2. 验证数据正确回填
3. 修改部门名称和负责人
4. 提交并验证 API PUT /admin-api/uaa/dept/update 成功
5. 验证树形列表自动刷新

预期结果:
✅ 编辑回填正确
✅ 修改提交成功

---

【测试场景 4:删除部门(含子部门)】
1. 找到有子部门的节点
2. 点击「删除」按钮
3. 验证是否提示「存在子部门,无法删除」或自动删除子部门
4. 找到叶子节点部门,点击「删除」
5. 验证删除成功后部门消失

预期结果:
✅ 删除前有子部门校验
✅ 叶子节点删除成功

---

【测试场景 5:负责人下拉选择】
1. 点击「新增」或「编辑」部门
2. 验证负责人字段使用 DictSelect/UserSelect 下拉
3. 验证下拉数据来自 user/list-all-simple 接口
4. 搜索用户关键词,验证下拉过滤

预期结果:
✅ 负责人下拉正常
✅ 搜索过滤正确

---

【测试场景 6:部门排序】
1. 创建多个子部门,设置不同 sort 值
2. 验证树形列表按 sort 升序展示
3. 编辑部门 sort 值,验证顺序变化

预期结果:
✅ 部门按 sort 升序展示
✅ 排序修改生效

---

【测试场景 7:部门搜索】
1. 在「部门名称」输入框输入关键词,点击「搜索」
2. 验证列表过滤出匹配部门
3. 点击「重置」,验证条件清空

预期结果:
✅ 部门搜索过滤正确
✅ 重置功能正常

---

【测试场景 8:批量删除部门】
1. 勾选多个叶子节点部门
2. 点击「批量删除」按钮(如有)
3. 验证 API DELETE /admin-api/uaa/dept/delete-list?ids=X,Y 成功
4. 验证所有选中部门从树形列表中消失

预期结果:
✅ 批量删除 API 调用成功
✅ 删除后列表刷新

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 部门列表 | GET | /admin-api/uaa/dept/list |
| 部门精简列表 | GET | /admin-api/uaa/dept/list-all-simple |
| 部门详情 | GET | /admin-api/uaa/dept/get?id=X |
| 创建部门 | POST | /admin-api/uaa/dept/create |
| 更新部门 | PUT | /admin-api/uaa/dept/update |
| 删除部门 | DELETE | /admin-api/uaa/dept/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/dept/delete-list?ids=X,Y |

---

【问题诊断】
- 部门树不显示 → 检查 deptApi.list() 是否成功,数据格式是否为树形
- 编辑时数据不回填 → 检查编辑表单是否正确调用 deptApi.get(id)
- 删除失败 → 检查后端是否保护有子部门或关联用户的部门
```

#### 5.6.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【部门管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「部门管理」
- 路由: /uaa/dept
- 权限前缀: system:dept
- 涉及权限码:
  - system:dept:create(新增部门)
  - system:dept:update(编辑部门)
  - system:dept:delete(删除部门)
  - system:dept:query(查询部门)

【前端文件清单】
- 主页面: business-web/src/views/uaa/dept/index.vue
- API 封装: business-web/src/api/uaa/dept.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/dept/DeptController.java
- Service 接口: .../service/dept/DeptService.java
- DTO/Request: .../controller/admin/dept/vo/*ReqVO.java
- DO: .../dal/dataobject/dept/DeptDO.java
- Mapper: .../dal/mapper/dept/DeptMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 部门列表 | GET | /admin-api/uaa/dept/list |
| 部门精简列表 | GET | /admin-api/uaa/dept/list-all-simple |
| 部门详情 | GET | /admin-api/uaa/dept/get?id=X |
| 创建部门 | POST | /admin-api/uaa/dept/create |
| 更新部门 | PUT | /admin-api/uaa/dept/update |
| 删除部门 | DELETE | /admin-api/uaa/dept/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/dept/delete-list?ids=X,Y |

【功能需求】
1. 支持部门树形 CRUD
2. 负责人关联用户(user/list-all-simple 接口)
3. 删除前子部门校验
4. 部门排序(sort 字段)
5. 部门搜索过滤
6. 批量删除

【UI 规范】
- UI 库: ant-design-vue
- 表格列:部门名称、负责人、排序、手机号、邮箱、创建时间、操作
- 业务组件复用: UserSelect(来自 business-web/src/components/business)
- 操作按钮:新增、编辑、删除

【参考实现】
请参考以下已实现模块:
- business-web/src/views/uaa/menu/index.vue(菜单管理,适合作为树形 CRUD 参考)
- business-web/src/views/uaa/post/index.vue(岗位管理,适合作为下拉选择参考)
- 复用 business-web/src/components/business 中的 UserSelect 组件

【测试验证】
实现完成后,使用 5.6.2 测试提示词中的测试场景验证,重点验证:
1. 部门树形列表完整加载
2. 负责人下拉正常
3. 删除子部门校验生效
4. 排序与搜索过滤正确
5. 8 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(dept/index.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/dept)
- [ ] 菜单注册(左侧菜单「用户中心」→「部门管理」)
- [ ] 权限码注册(system:dept:create 等)
- [ ] 通过 5.6.2 所有测试场景
```

---

### 5.7 UAA-07 岗位管理

**页面路径**: 左侧菜单「用户中心」→「岗位管理」
**源码文件**: `src/views/uaa/post/index.vue`
**API 文件**: `src/api/uaa/post.ts`
**权限标识**: `system:post:create`, `system:post:update`, `system:post:delete`

#### 5.7.1 测试场景

#### 5.7.2 测试提示词

```
/browser 或 /open-gstack-browser
打开岗位管理页面,执行完整的岗位管理功能测试。

【前置操作】
1. 登录后进入「用户中心」→「岗位管理」
2. 等待页面加载完成

---

【测试场景 1:列表加载与显示】
1. 验证岗位列表表格正常加载
2. 检查列:岗位名称、岗位编码、排序、状态、备注、创建时间、操作
3. 验证状态列显示:启用/禁用
4. 验证分页器正常工作

预期结果:
✅ 表格列完整
✅ 状态正确显示
✅ 分页正常

---

【测试场景 2:新增岗位】
1. 点击「新增」按钮,验证弹窗弹出
2. 填写:岗位名称、岗位编码(唯一)、排序、备注
3. 提交并验证 API POST /admin-api/uaa/post/create 成功
4. 验证弹窗关闭,列表自动刷新

预期结果:
✅ 新增弹窗正常
✅ 提交成功,列表刷新

---

【测试场景 3:编辑岗位】
1. 点击岗位的「编辑」按钮
2. 验证数据正确回填
3. 修改岗位名称
4. 提交并验证 API PUT /admin-api/uaa/post/update 成功
5. 验证列表自动刷新

预期结果:
✅ 编辑回填正确
✅ 修改提交成功

---

【测试场景 4:删除岗位】
1. 点击岗位的「删除」按钮,验证弹出确认
2. 点击确认,验证 API DELETE /admin-api/uaa/post/delete?id=X 成功
3. 验证列表自动刷新,岗位消失

预期结果:
✅ 删除确认正常
✅ 删除成功后列表刷新

---

【测试场景 5:岗位编码唯一性校验】
1. 新增一个岗位,编码设为已有编码
2. 提交验证后端返回重复错误
3. 修改编码为唯一值,再次提交,验证成功

预期结果:
✅ 后端唯一性校验生效
✅ 错误提示清晰

---

【测试场景 6:状态切换】
1. 点击状态 DictSwitch,将岗位从启用切换为禁用
2. 验证 API PUT /admin-api/uaa/post/update-status 成功
3. 验证状态列立即更新

预期结果:
✅ 状态切换 API 调用成功
✅ 状态列实时更新

---

【测试场景 7:搜索与重置】
1. 在「岗位名称」输入框输入关键词,点击「搜索」
2. 在「岗位编码」输入框输入关键词,点击「搜索」
3. 点击「重置」,验证条件清空,列表恢复

预期结果:
✅ 搜索过滤正确
✅ 重置功能正常

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 岗位列表(精简) | GET | /admin-api/uaa/post/list-all-simple |
| 岗位分页 | GET | /admin-api/uaa/post/page |
| 岗位详情 | GET | /admin-api/uaa/post/get?id=X |
| 创建岗位 | POST | /admin-api/uaa/post/create |
| 更新岗位 | PUT | /admin-api/uaa/post/update |
| 删除岗位 | DELETE | /admin-api/uaa/post/delete?id=X |
| 更新状态 | PUT | /admin-api/uaa/post/update-status |

---

【问题诊断】
- 列表不加载 → 检查 postApi.page() 是否成功
- 编码唯一性校验不生效 → 检查后端是否有唯一索引约束
- 删除失败 → 检查岗位是否被用户引用
```

#### 5.7.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【岗位管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「岗位管理」
- 路由: /uaa/post
- 权限前缀: system:post
- 涉及权限码:
  - system:post:create(新增岗位)
  - system:post:update(编辑岗位)
  - system:post:delete(删除岗位)
  - system:post:query(查询岗位)

【前端文件清单】
- 主页面: business-web/src/views/uaa/post/index.vue
- API 封装: business-web/src/api/uaa/post.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/dept/PostController.java
- Service 接口: .../service/dept/PostService.java
- DTO/Request: .../controller/admin/dept/vo/post/*ReqVO.java
- DO: .../dal/dataobject/dept/PostDO.java
- Mapper: .../dal/mapper/dept/PostMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 岗位列表(精简) | GET | /admin-api/uaa/post/list-all-simple |
| 岗位分页 | GET | /admin-api/uaa/post/page |
| 岗位详情 | GET | /admin-api/uaa/post/get?id=X |
| 创建岗位 | POST | /admin-api/uaa/post/create |
| 更新岗位 | PUT | /admin-api/uaa/post/update |
| 删除岗位 | DELETE | /admin-api/uaa/post/delete?id=X |
| 更新状态 | PUT | /admin-api/uaa/post/update-status |

【功能需求】
1. 支持岗位列表 CRUD
2. 状态使用 DictSwitch 切换
3. 岗位编码唯一性校验
4. 岗位搜索(名称/编码)
5. 删除前用户引用校验

【UI 规范】
- UI 库: ant-design-vue
- 表格列:岗位名称、岗位编码、排序、状态、备注、创建时间、操作
- 字典类型: common_status(状态)
- 业务组件复用: DictSwitch
- 操作按钮:新增、编辑、删除

【参考实现】
请参考以下已实现模块:
- business-web/src/views/system/dict/index.vue(字典管理,适合作为表单+下拉选择参考)
- business-web/src/views/uaa/dept/index.vue(部门管理,适合作为独立 CRUD 参考)
- 复用 business-web/src/components/business 中的 DictSwitch 组件

【测试验证】
实现完成后,使用 5.7.2 测试提示词中的测试场景验证,重点验证:
1. 表格列完整,状态列 DictSwitch 正常
2. 岗位编码唯一性校验生效
3. 状态切换 API 调用成功
4. 7 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(post/index.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/post)
- [ ] 菜单注册(左侧菜单「用户中心」→「岗位管理」)
- [ ] 权限码注册(system:post:create 等)
- [ ] 通过 5.7.2 所有测试场景
```

---

### 5.8 UAA-08 租户管理

**页面路径**: 左侧菜单「用户中心」→「租户管理」
**源码文件**: `src/views/uaa/tenant/index.vue`
**API 文件**: `src/api/uaa/tenant.ts`
**权限标识**: `system:tenant:create`, `system:tenant:update`, `system:tenant:delete`

#### 5.8.1 测试场景

#### 5.8.2 测试提示词

```
/browser 或 /open-gstack-browser
打开租户管理页面,执行完整的租户管理功能测试。

【前置操作】
1. 登录后进入「用户中心」→「租户管理」
2. 等待页面加载完成

---

【测试场景 1:列表加载与筛选】
1. 验证租户列表表格正常加载
2. 检查列:租户ID、租户名称、套餐ID、联系人、联系手机、状态、账号额度、过期时间、创建时间、操作
3. 在「租户名称」输入框输入关键词,点击「搜索」
4. 在「联系人」输入框输入关键词,点击「搜索」
5. 点击「重置」,验证条件清空

预期结果:
✅ 表格列完整
✅ 搜索过滤正确
✅ 重置功能正常

---

【测试场景 2:新增租户】
1. 点击「新增」按钮
2. 验证弹窗弹出,标题为「新增租户」
3. 验证表单字段:租户名称、租户套餐、联系人、联系手机、用户名、密码、账号额度、过期时间、状态
4. 填写:租户名称、联系人、手机号、管理员用户名、密码、账号额度(默认-1无限制)、过期时间、状态(正常)
5. 提交并验证 API POST /admin-api/uaa/tenant/create 成功
6. 验证弹窗关闭,列表自动刷新

预期结果:
✅ 新增弹窗表单字段完整
✅ 提交成功,列表自动刷新

---

【测试场景 3:编辑租户】
1. 点击租户的「编辑」按钮
2. 验证弹窗弹出,数据正确回填
3. 修改租户名称和联系人
4. 提交并验证 API PUT /admin-api/uaa/tenant/update 成功
5. 验证列表自动刷新

预期结果:
✅ 编辑回填正确
✅ 修改提交成功

---

【测试场景 4:删除租户】
1. 点击租户的「删除」按钮,验证弹出确认
2. 点击确认,验证 API DELETE /admin-api/uaa/tenant/delete 成功
3. 验证列表自动刷新,租户消失

预期结果:
✅ 删除确认正常
✅ 删除成功后列表刷新

---

【测试场景 5:租户状态切换】
1. 点击状态 DictSwitch,将租户从正常切换为禁用
2. 验证 API PUT /admin-api/uaa/tenant/update-status 成功
3. 验证状态列立即更新

预期结果:
✅ 状态切换 API 调用成功
✅ 状态列实时更新

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表分页 | GET | /admin-api/uaa/tenant/page |
| 租户详情 | GET | /admin-api/uaa/tenant/get/{id} |
| 创建租户 | POST | /admin-api/uaa/tenant/create |
| 更新租户 | PUT | /admin-api/uaa/tenant/update |
| 删除租户 | DELETE | /admin-api/uaa/tenant/delete |
| 更新状态 | PUT | /admin-api/uaa/tenant/update-status |
| 租户精简列表 | GET | /admin-api/uaa/tenant/list-all-simple |

---

【问题诊断】
- 新增租户失败 → 检查用户名/密码是否必填,后端是否返回校验错误
- 编辑时日期不回填 → 检查 expireTime 字段是否为 dayjs 对象格式转换
- 删除失败 → 检查租户是否有关联数据(用户、套餐等)
```

#### 5.8.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【租户管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「租户管理」
- 路由: /uaa/tenant
- 权限前缀: system:tenant
- 涉及权限码:
  - system:tenant:create(新增租户)
  - system:tenant:update(编辑租户)
  - system:tenant:delete(删除租户)
  - system:tenant:query(查询租户)

【前端文件清单】
- 主页面: business-web/src/views/uaa/tenant/index.vue
- API 封装: business-web/src/api/uaa/tenant.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/tenant/TenantController.java
- Service 接口: .../service/tenant/TenantService.java
- DTO/Request: .../controller/admin/tenant/vo/*ReqVO.java
- DO: .../dal/dataobject/tenant/TenantDO.java
- Mapper: .../dal/mapper/tenant/TenantMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | /admin-api/uaa/tenant/page |
| 租户详情 | GET | /admin-api/uaa/tenant/get/{id} |
| 创建租户 | POST | /admin-api/uaa/tenant/create |
| 更新租户 | PUT | /admin-api/uaa/tenant/update |
| 删除租户 | DELETE | /admin-api/uaa/tenant/delete |
| 更新状态 | PUT | /admin-api/uaa/tenant/update-status |

【功能需求】
1. 支持租户列表 CRUD
2. 租户套餐关联(TenantPackageList 下拉)
3. 租户状态切换
4. 过期时间字段(日期时间选择器)
5. 账号额度字段(数字输入,-1 表示无限制)
6. 删除前关联数据校验

【UI 规范】
- UI 库: ant-design-vue
- 表格列:租户ID、租户名称、套餐ID、联系人、联系手机、状态、账号额度、过期时间、创建时间、操作
- 字典类型: common_status(状态)
- 业务组件复用: TenantPackageSelect
- 操作按钮:新增、编辑、删除

【参考实现】
请参考以下已实现模块:
- business-web/src/views/system/dict/index.vue(字典管理,适合作为表单+下拉选择参考)
- business-web/src/views/uaa/tenantPackage/index.vue(租户套餐,适合作为下拉数据源参考)
- 复用 business-web/src/components/business 中的日期时间选择器组件

【测试验证】
实现完成后,使用 5.8.2 测试提示词中的测试场景验证,重点验证:
1. 表格列完整,过期时间格式正确
2. 新增租户表单字段完整
3. 状态切换 API 调用成功
4. 5 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(tenant/index.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/tenant)
- [ ] 菜单注册(左侧菜单「用户中心」→「租户管理」)
- [ ] 权限码注册(system:tenant:create 等)
- [ ] 通过 5.8.2 所有测试场景
```

---

### 5.9 UAA-09 租户套餐管理

**页面路径**: 左侧菜单「用户中心」→「租户套餐」
**源码文件**: `src/views/uaa/tenantPackage/index.vue`
**API 文件**: `src/api/uaa/tenantPackage.ts`（如存在）或内联 API
**权限标识**: `system:tenant-package:create`, `system:tenant-package:update`, `system:tenant-package:delete`

#### 5.9.1 测试场景

#### 5.9.2 测试提示词

```
/browser 或 /open-gstack-browser
打开租户套餐管理页面,执行完整的租户套餐功能测试。

【前置操作】
1. 登录后进入「用户中心」→「租户套餐」
2. 等待页面加载完成

---

【测试场景 1:列表加载与显示】
1. 验证租户套餐列表表格正常加载
2. 检查列:套餐名称、状态、菜单权限树、备注、创建时间、操作
3. 验证状态列正确显示
4. 验证分页器正常工作

预期结果:
✅ 表格列完整
✅ 菜单权限列显示正常
✅ 分页正常

---

【测试场景 2:新增租户套餐】
1. 点击「新增」按钮,验证弹窗弹出
2. 验证表单字段:套餐名称、状态、菜单权限树、备注
3. 填写套餐名称和备注
4. 在菜单权限树中勾选部分菜单
5. 提交并验证 API POST 成功
6. 验证弹窗关闭,列表自动刷新

预期结果:
✅ 新增弹窗正常
✅ 菜单权限树可交互
✅ 提交成功,列表刷新

---

【测试场景 3:编辑租户套餐】
1. 点击套餐的「编辑」按钮
2. 验证数据正确回填,菜单权限树已勾选状态
3. 修改套餐名称
4. 提交并验证 API PUT 成功
5. 验证列表自动刷新

预期结果:
✅ 编辑回填正确
✅ 菜单权限树回显正确
✅ 修改提交成功

---

【测试场景 4:删除租户套餐】
1. 点击套餐的「删除」按钮,验证弹出确认
2. 点击确认,验证 API DELETE 成功
3. 验证列表自动刷新,套餐消失

预期结果:
✅ 删除确认正常
✅ 删除成功后列表刷新

---

【测试场景 5:租户套餐状态切换】
1. 点击状态 DictSwitch,将套餐从启用切换为禁用
2. 验证 API 成功
3. 验证状态列立即更新

预期结果:
✅ 状态切换 API 调用成功
✅ 状态列实时更新

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表分页 | GET | /admin-api/uaa/tenant-package/page |
| 套餐详情 | GET | /admin-api/uaa/tenant-package/get?id=X |
| 创建套餐 | POST | /admin-api/uaa/tenant-package/create |
| 更新套餐 | PUT | /admin-api/uaa/tenant-package/update |
| 删除套餐 | DELETE | /admin-api/uaa/tenant-package/delete |
| 更新状态 | PUT | /admin-api/uaa/tenant-package/update-status |
| 套餐精简列表 | GET | /admin-api/uaa/tenant-package/list-all-simple |

---

【问题诊断】
- 菜单权限树不加载 → 检查菜单接口是否正常返回
- 编辑时菜单权限不回显 → 检查回显逻辑是否正确处理选中状态
- 删除失败 → 检查套餐是否被租户引用
```

#### 5.9.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【租户套餐管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「租户套餐」
- 路由: /uaa/tenant-package
- 权限前缀: system:tenant-package
- 涉及权限码:
  - system:tenant-package:create(新增套餐)
  - system:tenant-package:update(编辑套餐)
  - system:tenant-package:delete(删除套餐)
  - system:tenant-package:query(查询套餐)

【前端文件清单】
- 主页面: business-web/src/views/uaa/tenantPackage/index.vue
- API 封装: business-web/src/api/uaa/tenantPackage.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/tenant/TenantPackageController.java
- Service 接口: .../service/tenant/TenantPackageService.java
- DTO/Request: .../controller/admin/tenant/vo/tenantPackage/*ReqVO.java
- DO: .../dal/dataobject/tenant/TenantPackageDO.java
- Mapper: .../dal/mapper/tenant/TenantPackageMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | /admin-api/uaa/tenant-package/page |
| 套餐详情 | GET | /admin-api/uaa/tenant-package/get?id=X |
| 创建套餐 | POST | /admin-api/uaa/tenant-package/create |
| 更新套餐 | PUT | /admin-api/uaa/tenant-package/update |
| 删除套餐 | DELETE | /admin-api/uaa/tenant-package/delete |
| 更新状态 | PUT | /admin-api/uaa/tenant-package/update-status |

【功能需求】
1. 支持租户套餐列表 CRUD
2. 菜单权限树形选择(完整菜单树)
3. 套餐状态切换
4. 套餐搜索(名称)
5. 删除前租户引用校验

【UI 规范】
- UI 库: ant-design-vue
- 表格列:套餐名称、状态、菜单权限、备注、创建时间、操作
- 字典类型: common_status(状态)
- 业务组件复用: 菜单权限树选择器
- 操作按钮:新增、编辑、删除

【参考实现】
请参考以下已实现模块:
- business-web/src/views/uaa/role/index.vue(角色管理,适合作为菜单权限树参考)
- business-web/src/views/uaa/tenant/index.vue(租户管理,适合作为 CRUD 参考)
- 复用 business-web/src/components/business 中的菜单权限树选择器

【测试验证】
实现完成后,使用 5.9.2 测试提示词中的测试场景验证,重点验证:
1. 表格列完整,菜单权限列显示
2. 菜单权限树可交互
3. 状态切换 API 调用成功
4. 5 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(tenantPackage/index.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/tenant-package)
- [ ] 菜单注册(左侧菜单「用户中心」→「租户套餐」)
- [ ] 权限码注册(system:tenant-package:create 等)
- [ ] 通过 5.9.2 所有测试场景
```

---

### 5.10 UAA-10 OAuth2 客户端管理

**页面路径**: 左侧菜单「用户中心」→「OAuth2 客户端」
**源码文件**: `src/views/uaa/oauth2Client/index.vue`, `src/views/uaa/oauth2Client/ClientForm.vue`
**API 文件**: `src/api/uaa/oauth2Client.ts`
**权限标识**: `system:oauth2-client:create`, `system:oauth2-client:update`, `system:oauth2-client:delete`

#### 5.10.1 测试场景

#### 5.10.2 测试提示词

```
/browser 或 /open-gstack-browser
打开 OAuth2 客户端管理页面,执行完整的客户端管理功能测试。

【前置操作】
1. 登录后进入「用户中心」→「OAuth2 客户端」
2. 等待页面加载完成

---

【测试场景 1:列表加载与显示】
1. 验证 OAuth2 客户端列表表格正常加载
2. 检查列:客户端ID、客户端名称、授权类型、状态、操作
3. 验证状态列正确显示(开启/关闭)
4. 验证分页器正常工作

预期结果:
✅ 表格列完整
✅ 状态正确显示
✅ 分页正常

---

【测试场景 2:新增客户端】
1. 点击「新增」按钮,验证弹窗弹出
2. 验证表单字段:客户端ID、客户端密钥、客户端名称、授权类型(多选)、Token有效期、刷新Token有效期、重定向URI
3. 填写:客户端ID(唯一)、名称、选择授权类型(password、refresh_token)、设置有效期
4. 提交并验证 API POST /admin-api/uaa/oauth2-client/create 成功
5. 验证弹窗关闭,列表自动刷新

预期结果:
✅ 新增弹窗表单字段完整
✅ 授权类型多选正常
✅ 提交成功,列表刷新

---

【测试场景 3:编辑客户端】
1. 点击客户端的「编辑」按钮
2. 验证数据正确回填(客户端ID不可修改,其他字段可改)
3. 修改客户端名称和 Token 有效期
4. 提交并验证 API PUT /admin-api/uaa/oauth2-client/update 成功
5. 验证列表自动刷新

预期结果:
✅ 编辑回填正确
✅ 修改提交成功

---

【测试场景 4:删除客户端】
1. 点击客户端的「删除」按钮,验证弹出确认
2. 点击确认,验证 API DELETE /admin-api/uaa/oauth2-client/delete?id=X 成功
3. 验证列表自动刷新,客户端消失

预期结果:
✅ 删除确认正常
✅ 删除成功后列表刷新

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表分页 | GET | /admin-api/uaa/oauth2-client/page |
| 客户端详情 | GET | /admin-api/uaa/oauth2-client/get?id=X |
| 创建客户端 | POST | /admin-api/uaa/oauth2-client/create |
| 更新客户端 | PUT | /admin-api/uaa/oauth2-client/update |
| 删除客户端 | DELETE | /admin-api/uaa/oauth2-client/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/oauth2-client/delete-list |

---

【问题诊断】
- 新增客户端失败 → 检查客户端ID是否唯一,授权类型是否至少选择一项
- 编辑时客户端ID不可修改 → 这是后端设计,前端应禁用该字段
- 删除失败 → 检查客户端是否有关联的 Token
```

#### 5.10.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【OAuth2 客户端管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「OAuth2 客户端」
- 路由: /uaa/oauth2-client
- 权限前缀: system:oauth2-client
- 涉及权限码:
  - system:oauth2-client:create(新增客户端)
  - system:oauth2-client:update(编辑客户端)
  - system:oauth2-client:delete(删除客户端)
  - system:oauth2-client:query(查询客户端)

【前端文件清单】
- 主页面: business-web/src/views/uaa/oauth2Client/index.vue
- 表单弹窗: business-web/src/views/uaa/oauth2Client/ClientForm.vue
- API 封装: business-web/src/api/uaa/oauth2Client.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/oauth2/OAuth2ClientController.java
- Service 接口: .../service/oauth2/OAuth2ClientService.java
- DTO/Request: .../controller/admin/oauth2/vo/client/*ReqVO.java
- DO: .../dal/dataobject/oauth2/OAuth2ClientDO.java
- Mapper: .../dal/mapper/oauth2/OAuth2ClientMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | /admin-api/uaa/oauth2-client/page |
| 客户端详情 | GET | /admin-api/uaa/oauth2-client/get?id=X |
| 创建客户端 | POST | /admin-api/uaa/oauth2-client/create |
| 更新客户端 | PUT | /admin-api/uaa/oauth2-client/update |
| 删除客户端 | DELETE | /admin-api/uaa/oauth2-client/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/oauth2-client/delete-list |

【功能需求】
1. 支持 OAuth2 客户端列表 CRUD
2. 授权类型多选(password/refresh_token/authorization_code 等)
3. Token 有效期与刷新 Token 有效期配置
4. 重定向 URI 配置
5. 客户端ID唯一性校验
6. 状态切换(启用/禁用)

【UI 规范】
- UI 库: ant-design-vue
- 表格列:客户端ID、客户端名称、授权类型、状态、操作
- 字典类型: common_status(状态)、oauth2_grant_type(授权类型)
- 业务组件复用: DictSwitch
- 操作按钮:新增、编辑、删除

【参考实现】
请参考以下已实现模块:
- business-web/src/views/system/dict/index.vue(字典管理,适合作为表单+下拉选择参考)
- business-web/src/views/uaa/oauth2Token/index.vue(OAuth2 Token,适合作为关联模块参考)
- 复用 business-web/src/components/business 中的 DictSwitch 组件

【测试验证】
实现完成后,使用 5.10.2 测试提示词中的测试场景验证,重点验证:
1. 表格列完整,授权类型列正确显示
2. 授权类型多选正常
3. 客户端ID唯一性校验生效
4. 4 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(oauth2Client/index.vue)
- [ ] 前端表单弹窗 .vue(ClientForm.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/oauth2-client)
- [ ] 菜单注册(左侧菜单「用户中心」→「OAuth2 客户端」)
- [ ] 权限码注册(system:oauth2-client:create 等)
- [ ] 字典数据初始化 SQL(oauth2_grant_type)
- [ ] 通过 5.10.2 所有测试场景
```

---

### 5.11 UAA-11 OAuth2 Token 管理

**页面路径**: 左侧菜单「用户中心」→「OAuth2 Token」
**源码文件**: `src/views/uaa/oauth2Token/index.vue`
**API 文件**: `src/api/uaa/oauth2Token.ts`
**权限标识**: `system:oauth2-token:delete`

#### 5.11.1 测试场景

#### 5.11.2 测试提示词

```
/browser 或 /open-gstack-browser
打开 OAuth2 Token 管理页面,执行完整的 Token 管理功能测试。

【前置操作】
1. 登录后进入「用户中心」→「OAuth2 Token」
2. 等待页面加载完成

---

【测试场景 1:Token 列表加载】
1. 验证 Token 列表表格正常加载
2. 检查列:访问令牌、刷新令牌、用户ID、用户类型、客户端ID、过期时间、创建时间、操作
3. 验证过期时间列格式正确
4. 验证分页器正常工作

预期结果:
✅ 表格列完整
✅ 过期时间格式正确
✅ 分页正常

---

【测试场景 2:单条 Token 强退】
1. 找到任意有效 Token
2. 点击「强退」按钮
3. 验证弹出确认弹窗
4. 点击确认,验证 API DELETE /admin-api/uaa/oauth2-token/delete?accessToken=X 成功
5. 验证显示「强退成功」
6. 验证 Token 从列表中消失

预期结果:
✅ 强退确认正常
✅ 强退成功后 Token 消失

---

【测试场景 3:批量 Token 强退】
1. 勾选 2-3 条 Token 记录
2. 点击「批量强退」按钮
3. 验证弹出确认弹窗,显示待强退数量
4. 点击确认,验证 API DELETE /admin-api/uaa/oauth2-token/delete-list 成功
5. 验证所有选中 Token 从列表中消失

预期结果:
✅ 批量强退确认正常
✅ 批量强退成功后列表刷新

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| Token 列表 | GET | /admin-api/uaa/oauth2-token/page |
| 单条强退 | DELETE | /admin-api/uaa/oauth2-token/delete?accessToken=X |
| 批量强退 | DELETE | /admin-api/uaa/oauth2-token/delete-list |

---

【问题诊断】
- Token 列表不加载 → 检查后端 OAuth2 Token 表是否有数据
- 强退失败 → 检查后端是否正确删除 Token 并清理 Redis 缓存
- 批量强退参数错误 → 检查请求体格式是否为逗号分隔的 accessToken 列表
```

#### 5.11.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【OAuth2 Token 管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「OAuth2 Token」
- 路由: /uaa/oauth2-token
- 权限前缀: system:oauth2-token
- 涉及权限码:
  - system:oauth2-token:delete(强退 Token)
  - system:oauth2-token:query(查询 Token)

【前端文件清单】
- 主页面: business-web/src/views/uaa/oauth2Token/index.vue
- API 封装: business-web/src/api/uaa/oauth2Token.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/oauth2/OAuth2TokenController.java
- Service 接口: .../service/oauth2/OAuth2TokenService.java
- DTO/Request: .../controller/admin/oauth2/vo/token/*ReqVO.java
- DO: .../dal/dataobject/oauth2/OAuth2AccessTokenDO.java
- Mapper: .../dal/mapper/oauth2/OAuth2AccessTokenMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| Token 列表 | GET | /admin-api/uaa/oauth2-token/page |
| 单条强退 | DELETE | /admin-api/uaa/oauth2-token/delete?accessToken=X |
| 批量强退 | DELETE | /admin-api/uaa/oauth2-token/delete-list |

【功能需求】
1. 支持 Token 列表查看(分页)
2. 单条 Token 强退
3. 批量 Token 强退
4. 过期时间格式展示
5. 用户类型与客户端ID关联显示

【UI 规范】
- UI 库: ant-design-vue
- 表格列:访问令牌、刷新令牌、用户ID、用户类型、客户端ID、过期时间、创建时间、操作
- 业务组件复用: 无
- 操作按钮:强退

【参考实现】
请参考以下已实现模块:
- business-web/src/views/uaa/oauth2Client/index.vue(OAuth2 客户端,适合作为关联模块参考)
- business-web/src/views/uaa/loginLog/index.vue(登录日志,适合作为日志类页面参考)

【测试验证】
实现完成后,使用 5.11.2 测试提示词中的测试场景验证,重点验证:
1. 表格列完整,过期时间格式正确
2. 单条强退与批量强退 API 调用成功
3. 强退后列表刷新
4. 3 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(oauth2Token/index.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/oauth2-token)
- [ ] 菜单注册(左侧菜单「用户中心」→「OAuth2 Token」)
- [ ] 权限码注册(system:oauth2-token:delete 等)
- [ ] 通过 5.11.2 所有测试场景
```

---

### 5.12 UAA-12 社交用户管理

**页面路径**: 左侧菜单「用户中心」→「社交用户」
**源码文件**: `src/views/uaa/socialUser/index.vue`
**API 文件**: `src/api/uaa/socialUser.ts`（如存在）或内联 API
**权限标识**: `system:social-user:unbind`

#### 5.12.1 测试场景

#### 5.12.2 测试提示词

```
/browser 或 /open-gstack-browser
打开社交用户管理页面,执行完整的社交用户管理功能测试。

【前置操作】
1. 登录后进入「用户中心」→「社交用户」
2. 等待页面加载完成

---

【测试场景 1:社交用户列表加载】
1. 验证社交用户列表表格正常加载
2. 检查列:用户ID、用户名、社交类型、社交用户ID、操作
3. 验证社交类型列使用 DictTag 标签
4. 验证分页器正常工作

预期结果:
✅ 表格列完整
✅ 社交类型标签正确
✅ 分页正常

---

【测试场景 2:社交用户筛选】
1. 在「用户名」输入框输入关键词,点击「搜索」
2. 在「社交类型」下拉框选择一个类型,点击「搜索」
3. 点击「重置」,验证条件清空

预期结果:
✅ 搜索过滤正确
✅ 重置功能正常

---

【测试场景 3:社交用户解绑】
1. 找到任意社交用户记录
2. 点击「解绑」按钮,验证弹出确认弹窗
3. 点击确认,验证 API 调用成功
4. 验证显示「解绑成功」
5. 验证列表自动刷新,社交绑定记录消失

预期结果:
✅ 解绑确认正常
✅ 解绑成功后列表刷新

---

【测试场景 4:社交类型下拉数据】
1. 点击「社交类型」下拉框
2. 验证下拉选项包含全部已配置的社交类型(微信/GitHub/钉钉等)
3. 验证 DictSelect 数据源来自字典表

预期结果:
✅ 社交类型下拉完整
✅ 字典数据加载正确

---

【测试场景 5:解绑后用户态同步】
1. 找到已绑定社交账号的用户记录
2. 点击「解绑」并确认
3. 切换到该用户的「用户管理」页面
4. 验证用户管理页面中该用户社交绑定状态已更新

预期结果:
✅ 解绑后用户态同步
✅ 跨模块数据一致性

---

【测试场景 6:批量解绑】
1. 勾选多条社交用户记录
2. 点击「批量解绑」按钮(如有)
3. 验证 API DELETE 成功
4. 验证所有选中记录从列表中消失

预期结果:
✅ 批量解绑 API 调用成功
✅ 删除后列表刷新

---

【测试场景 7:分页与刷新】
1. 验证默认分页参数
2. 翻页/切换每页条数,验证数据切换
3. 点击「刷新」按钮,验证列表重新加载

预期结果:
✅ 分页参数正确
✅ 翻页/刷新功能正常

---

【测试场景 8:空数据兜底】
1. 在没有社交绑定记录时访问页面
2. 验证显示「暂无数据」空状态
3. 验证不报错,分页器显示「共 0 条」

预期结果:
✅ 空数据显示「暂无数据」
✅ 空数据时不报错

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 社交用户列表 | GET | /admin-api/uaa/social-user/page |
| 社交用户详情 | GET | /admin-api/uaa/social-user/get?id=X |
| 解绑社交用户 | DELETE | /admin-api/uaa/social-user/unbind?id=X |
| 批量解绑 | DELETE | /admin-api/uaa/social-user/unbind-list?ids=X,Y |

---

【问题诊断】
- 社交用户列表为空 → 检查社交用户表是否有绑定记录
- 解绑失败 → 检查后端是否正确删除绑定记录
- 跨模块同步失败 → 检查用户态缓存是否清理
```

#### 5.12.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【社交用户管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「社交用户」
- 路由: /uaa/social-user
- 权限前缀: system:social-user
- 涉及权限码:
  - system:social-user:query(查询社交用户)
  - system:social-user:unbind(解绑社交用户)

【前端文件清单】
- 主页面: business-web/src/views/uaa/socialUser/index.vue
- API 封装: business-web/src/api/uaa/socialUser.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/socail/SocialUserController.java
- Service 接口: .../service/social/SocialUserService.java
- DTO/Request: .../controller/admin/socail/vo/*ReqVO.java
- DO: .../dal/dataobject/social/SocialUserDO.java
- Mapper: .../dal/mapper/social/SocialUserMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 社交用户列表 | GET | /admin-api/uaa/social-user/page |
| 社交用户详情 | GET | /admin-api/uaa/social-user/get?id=X |
| 解绑社交用户 | DELETE | /admin-api/uaa/social-user/unbind?id=X |
| 批量解绑 | DELETE | /admin-api/uaa/social-user/unbind-list?ids=X,Y |

【功能需求】
1. 支持社交用户列表查看(分页)
2. 解绑社交用户(单条/批量)
3. 社交类型筛选(微信/GitHub/钉钉)
4. 用户名搜索
5. 跨模块用户态同步

【UI 规范】
- UI 库: ant-design-vue
- 表格列:用户ID、用户名、社交类型、社交用户ID、操作
- 字典类型: system_social_type(社交类型)
- 业务组件复用: DictTag、DictSelect
- 操作按钮:解绑

【参考实现】
请参考以下已实现模块:
- business-web/src/views/uaa/socialClient/index.vue(社交客户端,适合作为关联模块参考)
- business-web/src/views/uaa/loginLog/index.vue(登录日志,适合作为日志类页面参考)
- 复用 business-web/src/components/business 中的 DictTag/DictSelect 组件

【测试验证】
实现完成后,使用 5.12.2 测试提示词中的测试场景验证,重点验证:
1. 表格列完整,社交类型标签正确
2. 解绑 API 调用成功
3. 跨模块数据一致性
4. 8 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(socialUser/index.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/social-user)
- [ ] 菜单注册(左侧菜单「用户中心」→「社交用户」)
- [ ] 权限码注册(system:social-user:unbind 等)
- [ ] 通过 5.12.2 所有测试场景
```

---

### 5.13 UAA-13 社交客户端管理

**页面路径**: 左侧菜单「用户中心」→「社交客户端」
**源码文件**: `src/views/uaa/socialClient/index.vue`
**API 文件**: `src/api/uaa/socialClient.ts`（如存在）或内联 API
**权限标识**: `system:social-client:create`, `system:social-client:update`, `system:social-client:delete`

#### 5.13.1 测试场景

#### 5.13.2 测试提示词

```
/browser 或 /open-gstack-browser
打开社交客户端管理页面,执行完整的社交客户端管理功能测试。

【前置操作】
1. 登录后进入「用户中心」→「社交客户端」
2. 等待页面加载完成

---

【测试场景 1:社交客户端列表加载】
1. 验证社交客户端列表表格正常加载
2. 检查列:社交类型、客户端ID、客户端密钥、状态、操作
3. 验证社交类型列使用 DictTag 标签
4. 验证状态列正确显示

预期结果:
✅ 表格列完整
✅ 社交类型标签正确
✅ 状态显示正确

---

【测试场景 2:新增社交客户端】
1. 点击「新增」按钮,验证弹窗弹出
2. 验证表单字段:社交类型、客户端ID、客户端密钥、状态
3. 填写:选择社交类型(如微信)、客户端ID、客户端密钥
4. 提交并验证 API POST /admin-api/uaa/social-client/create 成功
5. 验证弹窗关闭,列表自动刷新

预期结果:
✅ 新增弹窗表单字段完整
✅ 提交成功,列表刷新

---

【测试场景 3:编辑社交客户端】
1. 点击社交客户端的「编辑」按钮
2. 验证数据正确回填
3. 修改客户端密钥
4. 提交并验证 API PUT /admin-api/uaa/social-client/update 成功
5. 验证列表自动刷新

预期结果:
✅ 编辑回填正确
✅ 修改提交成功

---

【测试场景 4:删除社交客户端】
1. 点击社交客户端的「删除」按钮,验证弹出确认
2. 点击确认,验证 API DELETE 成功
3. 验证列表自动刷新,记录消失

预期结果:
✅ 删除确认正常
✅ 删除成功后列表刷新

---

【测试场景 5:社交客户端状态切换】
1. 点击状态 DictSwitch,将客户端从启用切换为禁用
2. 验证 API 成功
3. 验证状态列立即更新

预期结果:
✅ 状态切换 API 调用成功
✅ 状态列实时更新

---

【测试场景 6:客户端密钥加密展示】
1. 验证列表中客户端密钥字段使用掩码(如 ****** 或部分脱敏)
2. 编辑时验证密钥字段是否可重新填写

预期结果:
✅ 密钥掩码显示
✅ 编辑时可重写

---

【测试场景 7:社交类型唯一性校验】
1. 新增一个客户端,选择已有社交类型
2. 验证提示「该社交类型已存在客户端」或后端返回重复错误
3. 修改社交类型为唯一值,再次提交验证成功

预期结果:
✅ 社交类型唯一性校验生效
✅ 错误提示清晰

---

【测试场景 8:搜索与重置】
1. 在「社交类型」下拉框选择类型,点击「搜索」
2. 在「客户端ID」输入框输入关键词,点击「搜索」
3. 点击「重置」,验证条件清空

预期结果:
✅ 搜索过滤正确
✅ 重置功能正常

---

【测试场景 9:分页与刷新】
1. 验证默认分页参数
2. 翻页/切换每页条数,验证数据切换
3. 点击「刷新」按钮,验证列表重新加载

预期结果:
✅ 分页参数正确
✅ 翻页/刷新功能正常

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 社交客户端列表 | GET | /admin-api/uaa/social-client/page |
| 社交客户端详情 | GET | /admin-api/uaa/social-client/get?id=X |
| 创建社交客户端 | POST | /admin-api/uaa/social-client/create |
| 更新社交客户端 | PUT | /admin-api/uaa/social-client/update |
| 删除社交客户端 | DELETE | /admin-api/uaa/social-client/delete |
| 更新状态 | PUT | /admin-api/uaa/social-client/update-status |

---

【问题诊断】
- 社交类型下拉为空 → 检查 SocialTypeEnum 是否正确导入和渲染
- 新增失败 → 检查客户端ID是否唯一,密钥格式是否正确
```

#### 5.13.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【社交客户端管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「社交客户端」
- 路由: /uaa/social-client
- 权限前缀: system:social-client
- 涉及权限码:
  - system:social-client:create(新增客户端)
  - system:social-client:update(编辑客户端)
  - system:social-client:delete(删除客户端)
  - system:social-client:query(查询客户端)

【前端文件清单】
- 主页面: business-web/src/views/uaa/socialClient/index.vue
- API 封装: business-web/src/api/uaa/socialClient.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/socail/SocialClientController.java
- Service 接口: .../service/social/SocialClientService.java
- DTO/Request: .../controller/admin/socail/vo/client/*ReqVO.java
- DO: .../dal/dataobject/social/SocialClientDO.java
- Mapper: .../dal/mapper/social/SocialClientMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 社交客户端列表 | GET | /admin-api/uaa/social-client/page |
| 社交客户端详情 | GET | /admin-api/uaa/social-client/get?id=X |
| 创建社交客户端 | POST | /admin-api/uaa/social-client/create |
| 更新社交客户端 | PUT | /admin-api/uaa/social-client/update |
| 删除社交客户端 | DELETE | /admin-api/uaa/social-client/delete |
| 更新状态 | PUT | /admin-api/uaa/social-client/update-status |

【功能需求】
1. 支持社交客户端列表 CRUD
2. 社交类型配置(微信/GitHub/钉钉/企业微信)
3. 客户端ID与密钥配置
4. 密钥掩码展示与编辑重写
5. 状态切换(启用/禁用)
6. 社交类型唯一性校验(同一类型仅一个启用客户端)

【UI 规范】
- UI 库: ant-design-vue
- 表格列:社交类型、客户端ID、客户端密钥、状态、操作
- 字典类型: system_social_type(社交类型)、common_status(状态)
- 业务组件复用: DictTag、DictSwitch、DictSelect
- 操作按钮:新增、编辑、删除

【参考实现】
请参考以下已实现模块:
- business-web/src/views/system/dict/index.vue(字典管理,适合作为表单+下拉选择参考)
- business-web/src/views/uaa/oauth2Client/index.vue(OAuth2 客户端,适合作为关联模块参考)
- 复用 business-web/src/components/business 中的 DictTag/DictSwitch/DictSelect 组件

【测试验证】
实现完成后,使用 5.13.2 测试提示词中的测试场景验证,重点验证:
1. 表格列完整,社交类型标签正确
2. 客户端密钥掩码展示
3. 社交类型唯一性校验生效
4. 9 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(socialClient/index.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/social-client)
- [ ] 菜单注册(左侧菜单「用户中心」→「社交客户端」)
- [ ] 权限码注册(system:social-client:create 等)
- [ ] 字典数据初始化 SQL(system_social_type)
- [ ] 通过 5.13.2 所有测试场景
```

---

### 5.14 UAA-14 登录日志

**页面路径**: 左侧菜单「用户中心」→「登录日志」
**源码文件**: `src/views/uaa/loginLog/index.vue`, `src/views/uaa/loginLog/LoginLogDetailModal.vue`
**API 文件**: `src/api/uaa/loginLog.ts`
**权限标识**: `system:login-log:query`

#### 5.14.1 测试场景

#### 5.14.2 测试提示词

```
/browser 或 /open-gstack-browser
打开登录日志页面,执行完整的登录日志查看功能测试。

【前置操作】
1. 登录后进入「用户中心」→「登录日志」
2. 等待页面加载完成

---

【测试场景 1:登录日志列表加载】
1. 验证登录日志列表表格正常加载
2. 检查列:日志类型、用户名称、登录类型、登录结果、登录IP、登录时间、操作
3. 验证登录类型使用 DictTag 标签(如账号密码/短信验证码)
4. 验证登录结果使用 DictTag 标签(成功=绿色/失败=红色)
5. 验证分页器正常工作

预期结果:
✅ 表格列完整
✅ 登录类型和结果标签正确
✅ 分页正常

---

【测试场景 2:日志筛选】
1. 在「用户名称」输入框输入关键词,点击「搜索」
2. 在「登录类型」下拉框选择一个类型,点击「搜索」
3. 在「登录结果」下拉框选择一个结果,点击「搜索」
4. 点击「重置」,验证条件清空

预期结果:
✅ 搜索过滤正确
✅ 重置功能正常

---

【测试场景 3:日志详情】
1. 找到任意登录日志记录
2. 点击操作列的「详情」按钮
3. 验证 LoginLogDetailModal 弹窗弹出
4. 验证详情包含:用户名、登录类型、登录结果、IP地址、UserAgent、创建时间

预期结果:
✅ 详情弹窗正常打开
✅ 字段完整显示

---

【测试场景 4:日期范围筛选】
1. 在「登录时间」日期范围选择器选择起止日期
2. 点击「搜索」
3. 验证列表仅显示该日期范围内的日志
4. 清除日期范围,验证列表恢复

预期结果:
✅ 日期范围筛选正确
✅ 清除后列表恢复

---

【测试场景 5:日志详情弹窗字段完整性】
1. 找到任意登录日志记录,点击「详情」
2. 验证 LoginLogDetailModal 弹窗弹出
3. 验证详情包含以下全部字段:
   - 用户名(username)
   - 登录类型(logType):DictTag 标签
   - 登录结果(result):DictTag 标签
   - 用户IP(userIp)
   - UserAgent(userAgent)
   - 创建时间(createTime)
4. 验证所有字段正确显示,无数据缺失

预期结果:
✅ 详情弹窗正常打开
✅ 全部字段完整显示
✅ DictTag 标签正确渲染

---

【测试场景 6:日志导出】
1. 点击「导出」按钮(如有)
2. 验证触发 Excel 文件下载
3. 验证下载的 Excel 包含所有筛选后的日志

预期结果:
✅ 导出功能正常
✅ 文件下载触发

---

【测试场景 7:分页与刷新】
1. 验证默认分页参数(每页 20 条)
2. 翻页/切换每页条数,验证数据切换
3. 点击「刷新」按钮,验证列表重新加载

预期结果:
✅ 分页参数正确
✅ 翻页/刷新功能正常

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 登录日志列表 | GET | /admin-api/uaa/login-log/page |
| 登录日志详情 | GET | /admin-api/uaa/login-log/get?id=X |
| 导出登录日志 | GET | /admin-api/uaa/login-log/export-excel |

---

【问题诊断】
- 登录日志列表为空 → 检查系统是否有登录记录,日志是否正确写入数据库
- 详情弹窗数据不完整 → 检查 LoginLogDetailModal.vue 的字段绑定
- DictTag 不显示 → 检查 logType 和 result 是否使用了正确的字典类型标识
```

#### 5.14.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【登录日志】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「登录日志」
- 路由: /uaa/login-log
- 权限前缀: system:login-log
- 涉及权限码:
  - system:login-log:query(查询登录日志)
  - system:login-log:export(导出登录日志)

【前端文件清单】
- 主页面: business-web/src/views/uaa/loginLog/index.vue
- 详情弹窗: business-web/src/views/uaa/loginLog/LoginLogDetailModal.vue
- API 封装: business-web/src/api/uaa/loginLog.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/logger/LoginLogController.java
- Service 接口: .../service/logger/LoginLogService.java
- DTO/Request: .../controller/admin/logger/vo/*ReqVO.java
- DO: .../dal/dataobject/logger/LoginLogDO.java
- Mapper: .../dal/mapper/logger/LoginLogMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 登录日志列表 | GET | /admin-api/uaa/login-log/page |
| 登录日志详情 | GET | /admin-api/uaa/login-log/get?id=X |
| 导出登录日志 | GET | /admin-api/uaa/login-log/export-excel |

【功能需求】
1. 支持登录日志列表查看(分页)
2. 多维度筛选:用户名称、登录类型、登录结果、日期范围
3. 详情弹窗完整字段展示
4. 登录日志导出
5. 字典标签展示(登录类型/登录结果)

【UI 规范】
- UI 库: ant-design-vue
- 表格列:日志类型、用户名称、登录类型、登录结果、登录IP、登录时间、操作
- 字典类型: system_login_type(登录类型)、system_login_result(登录结果)
- 业务组件复用: DictTag
- 操作按钮:详情、导出

【参考实现】
请参考以下已实现模块:
- business-web/src/views/uaa/role/index.vue(角色管理,适合作为 CRUD 页面参考)
- business-web/src/views/system/operatelog/index.vue(操作日志,适合作为日志类页面参考)
- 复用 business-web/src/components/business 中的 DictTag 组件

【测试验证】
实现完成后,使用 5.14.2 测试提示词中的测试场景验证,重点验证:
1. 表格列完整,DictTag 标签正确
2. 多维度筛选生效
3. 详情弹窗字段完整
4. 7 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(loginLog/index.vue)
- [ ] 前端详情弹窗 .vue(LoginLogDetailModal.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/login-log)
- [ ] 菜单注册(左侧菜单「用户中心」→「登录日志」)
- [ ] 权限码注册(system:login-log:query 等)
- [ ] 通过 5.14.2 所有测试场景
```

---

### 5.15 UAA-15 业务机构管理

**页面路径**: 左侧菜单「用户中心」→「业务机构」
**源码文件**: `src/views/uaa/org/index.vue`, `src/views/uaa/org/OrgFormModal.vue`
**API 文件**: `src/api/uaa/org.ts`
**权限标识**: `system:org:create`, `system:org:update`, `system:org:delete`, `system:org:audit`

#### 5.15.1 测试场景

#### 5.15.2 测试提示词

```
/browser 或 /open-gstack-browser
打开业务机构管理页面,执行完整的业务机构管理功能测试。

【前置操作】
1. 登录后进入「用户中心」→「业务机构」
2. 等待页面加载完成

---

【测试场景 1:业务机构列表加载】
1. 验证业务机构列表表格正常加载
2. 检查列:机构名称、机构类型、级别、地区、联系人、审核状态、状态、操作
3. 验证机构类型使用 DictTag 标签(机构类型A/机构类型B等)
4. 验证级别使用 DictTag 标签(省级/市级/区县级/街道级)
5. 验证审核状态使用 DictTag 标签(待审核=橙色/已通过=绿色/已拒绝=红色)
6. 验证分页器正常工作

预期结果:
✅ 表格列完整
✅ 机构类型、级别、审核状态标签正确
✅ 分页正常

---

【测试场景 2:业务机构筛选】
1. 在「机构名称」输入框输入关键词,点击「搜索」
2. 在「机构类型」下拉框选择一个类型,点击「搜索」
3. 在「审核状态」下拉框选择一个状态,点击「搜索」
4. 点击「重置」,验证条件清空

预期结果:
✅ 搜索过滤正确
✅ 重置功能正常

---

【测试场景 3:新增业务机构】
1. 点击「新增」按钮,验证 OrgFormModal 弹窗弹出
2. 验证表单字段:机构名称、机构类型、级别、地区编码、地区名称、详细地址、联系人姓名、联系人电话、统一社会信用代码、营业执照URL、机构简介、受理范围、状态
3. 填写:机构名称(测试机构_001)、机构类型(机构类型A)、级别(市级)、联系人、联系电话
4. 提交并验证 API POST /admin-api/uaa/org/create 成功
5. 验证弹窗关闭,列表自动刷新,新机构出现在第 1 行

预期结果:
✅ 新增弹窗表单字段完整
✅ 提交成功,列表刷新

---

【测试场景 4:编辑业务机构】
1. 点击刚创建机构的「编辑」按钮
2. 验证数据正确回填
3. 修改机构名称为「测试机构_001_已修改」
4. 提交并验证 API PUT /admin-api/uaa/org/update 成功
5. 验证列表自动刷新,修改生效

预期结果:
✅ 编辑回填正确
✅ 修改提交成功

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表分页 | GET | /admin-api/uaa/org/page |
| 机构详情 | GET | /admin-api/uaa/org/get?id=X |
| 创建机构 | POST | /admin-api/uaa/org/create |
| 更新机构 | PUT | /admin-api/uaa/org/update |
| 删除机构 | DELETE | /admin-api/uaa/org/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/org/delete-list |
| 更新状态 | PUT | /admin-api/uaa/org/update-status |
| 审核机构 | PUT | /admin-api/uaa/org/audit |
| 机构精简列表 | GET | /admin-api/uaa/org/simple-list |

---

【问题诊断】
- 机构类型下拉为空 → 检查 OrgTypeEnum 是否正确导入,组件是否使用 v-for 渲染
- 审核状态不变 → 检查 audit API 调用后是否重新加载列表
- 删除失败 → 检查机构是否关联了业务专员
```

#### 5.15.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【业务机构管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「业务机构」
- 路由: /uaa/org
- 权限前缀: system:org
- 涉及权限码:
  - system:org:create(新增机构)
  - system:org:update(编辑机构)
  - system:org:delete(删除机构)
  - system:org:query(查询机构)
  - system:org:audit(审核机构)

【前端文件清单】
- 主页面: business-web/src/views/uaa/org/index.vue
- 表单弹窗: business-web/src/views/uaa/org/OrgFormModal.vue
- API 封装: business-web/src/api/uaa/org.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/org/OrgController.java
- Service 接口: .../service/org/OrgService.java
- DTO/Request: .../controller/admin/org/vo/*ReqVO.java
- DO: .../dal/dataobject/org/OrgDO.java
- Mapper: .../dal/mapper/org/OrgMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | /admin-api/uaa/org/page |
| 机构详情 | GET | /admin-api/uaa/org/get?id=X |
| 创建机构 | POST | /admin-api/uaa/org/create |
| 更新机构 | PUT | /admin-api/uaa/org/update |
| 删除机构 | DELETE | /admin-api/uaa/org/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/org/delete-list |
| 更新状态 | PUT | /admin-api/uaa/org/update-status |
| 审核机构 | PUT | /admin-api/uaa/org/audit |
| 机构精简列表 | GET | /admin-api/uaa/org/simple-list |

【功能需求】
1. 支持业务机构列表 CRUD
2. 机构类型(机构类型A/机构类型B/机构类型C等)
3. 级别配置(省级/市级/区县级/街道级)
4. 地区级联选择(省/市/区/街道)
5. 审核状态流转(待审核/已通过/已拒绝)
6. 删除前关联业务专员校验

【UI 规范】
- UI 库: ant-design-vue
- 表格列:机构名称、机构类型、级别、地区、联系人、审核状态、状态、操作
- 字典类型: business_org_type(机构类型)、business_org_level(级别)、audit_status(审核状态)、common_status(状态)
- 业务组件复用: DictTag、DictSelect、RegionCascader(地区级联)
- 操作按钮:新增、编辑、审核、删除

【参考实现】
请参考以下已实现模块:
- business-web/src/views/system/dict/index.vue(字典管理,适合作为表单+下拉选择参考)
- business-web/src/views/uaa/role/index.vue(角色管理,适合作为 CRUD 页面参考)
- 复用 business-web/src/components/business 中的 DictTag/DictSelect/RegionCascader 组件

【测试验证】
实现完成后,使用 5.15.2 测试提示词中的测试场景验证,重点验证:
1. 表格列完整,机构类型/级别/审核状态标签正确
2. 地区级联选择正常
3. 新增/编辑/审核/删除流程完整
4. 4 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(org/index.vue)
- [ ] 前端表单弹窗 .vue(OrgFormModal.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/org)
- [ ] 菜单注册(左侧菜单「用户中心」→「业务机构」)
- [ ] 权限码注册(system:org:create 等)
- [ ] 字典数据初始化 SQL(business_org_type、business_org_level、audit_status)
- [ ] 通过 5.15.2 所有测试场景
```

---

### 5.16 UAA-16 业务专员管理

**页面路径**: 左侧菜单「用户中心」→「业务专员」
**源码文件**: `src/views/uaa/staff/index.vue`, `src/views/uaa/staff/StaffFormModal.vue`
**API 文件**: `src/api/uaa/staff.ts`
**权限标识**: `system:staff:create`, `system:staff:update`, `system:staff:delete`, `system:staff:audit`

#### 5.16.1 测试场景

#### 5.16.2 测试提示词

```
/browser 或 /open-gstack-browser
打开业务专员管理页面,执行完整的业务专员管理功能测试。

【前置操作】
1. 登录后进入「用户中心」→「业务专员」
2. 等待页面加载完成

---

【测试场景 1:业务专员列表加载】
1. 验证业务专员列表表格正常加载
2. 检查列:工号、真实姓名、性别、手机号、业务专员类型、机构名称、状态、审核状态、操作
3. 验证业务专员类型使用 DictTag 标签(专职/兼职/特邀)
4. 验证审核状态使用 DictTag 标签(待审核=橙色/已通过=绿色/已拒绝=红色)
5. 验证分页器正常工作

预期结果:
✅ 表格列完整
✅ 业务专员类型和审核状态标签正确
✅ 分页正常

---

【测试场景 2:业务专员筛选】
1. 在「真实姓名」输入框输入关键词,点击「搜索」
2. 在「业务专员类型」下拉框选择一个类型,点击「搜索」
3. 在「审核状态」下拉框选择一个状态,点击「搜索」
4. 点击「重置」,验证条件清空

预期结果:
✅ 搜索过滤正确
✅ 重置功能正常

---

【测试场景 3:新增业务专员】
1. 点击「新增」按钮,验证 StaffFormModal 弹窗弹出
2. 验证表单字段:机构ID、真实姓名、性别、手机号、业务专员类型、专业领域、从业年限、证书编号、简介等
3. 填写:真实姓名(测试业务专员_001)、性别(男)、手机号、业务专员类型(专职)、专业领域
4. 提交并验证 API POST /admin-api/uaa/staff/create 成功
5. 验证弹窗关闭,列表自动刷新

预期结果:
✅ 新增弹窗表单字段完整
✅ 提交成功,列表刷新

---

【测试场景 4:编辑业务专员】
1. 点击刚创建业务专员的「编辑」按钮
2. 验证数据正确回填
3. 修改真实姓名
4. 提交并验证 API PUT /admin-api/uaa/staff/update 成功
5. 验证列表自动刷新,修改生效

预期结果:
✅ 编辑回填正确
✅ 修改提交成功

---

【测试场景 5:审核业务专员 - 通过】
1. 找到状态为「待审核」的业务专员
2. 点击「审核」按钮,验证审核弹窗弹出
3. 选择审核结果为「已通过」,填写审核备注
4. 点击确认,验证 API PUT /admin-api/uaa/staff/audit 成功
5. 验证审核状态从「待审核」变为「已通过」

预期结果:
✅ 审核通过流程正常
✅ 审核状态流转正确

---

【测试场景 6:审核业务专员 - 拒绝】
1. 找到另一条待审核业务专员
2. 执行审核,选择「已拒绝」,填写拒绝原因
3. 提交并验证审核状态变为「已拒绝」

预期结果:
✅ 审核拒绝流程正常
✅ 拒绝原因正确记录

---

【测试场景 7:删除业务专员】
1. 找到测试业务专员,点击「删除」按钮
2. 验证弹出确认弹窗
3. 点击确认,验证 API DELETE /admin-api/uaa/staff/delete?id=X 成功
4. 验证列表自动刷新,业务专员消失

预期结果:
✅ 删除确认正常
✅ 删除成功后列表刷新

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 业务专员分页 | GET | /admin-api/uaa/staff/page |
| 业务专员列表 | GET | /admin-api/uaa/staff/list |
| 业务专员详情 | GET | /admin-api/uaa/staff/get?id=X |
| 创建业务专员 | POST | /admin-api/uaa/staff/create |
| 更新业务专员 | PUT | /admin-api/uaa/staff/update |
| 删除业务专员 | DELETE | /admin-api/uaa/staff/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/staff/delete-list |
| 更新状态 | PUT | /admin-api/uaa/staff/update-status |
| 审核业务专员 | PUT | /admin-api/uaa/staff/audit |
| 业务专员统计 | GET | /admin-api/uaa/staff/statistics?id=X |

---

【问题诊断】
- 业务专员类型下拉为空 → 检查 StaffTypeEnum 是否正确导入和渲染
- 审核状态不更新 → 检查 audit API 调用后是否调用 loadData() 刷新列表
- 机构名称不显示 → 检查 orgId 字段是否正确关联 org 表并显示 orgName
- 删除失败 → 检查业务专员是否关联了案件
```

#### 5.16.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【业务专员管理】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「业务专员」
- 路由: /uaa/staff
- 权限前缀: system:staff
- 涉及权限码:
  - system:staff:create(新增业务专员)
  - system:staff:update(编辑业务专员)
  - system:staff:delete(删除业务专员)
  - system:staff:query(查询业务专员)
  - system:staff:audit(审核业务专员)

【前端文件清单】
- 主页面: business-web/src/views/uaa/staff/index.vue
- 表单弹窗: business-web/src/views/uaa/staff/StaffFormModal.vue
- API 封装: business-web/src/api/uaa/staff.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/staff/StaffController.java
- Service 接口: .../service/staff/StaffService.java
- DTO/Request: .../controller/admin/staff/vo/*ReqVO.java
- DO: .../dal/dataobject/staff/StaffDO.java
- Mapper: .../dal/mapper/staff/StaffMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 业务专员分页 | GET | /admin-api/uaa/staff/page |
| 业务专员列表 | GET | /admin-api/uaa/staff/list |
| 业务专员详情 | GET | /admin-api/uaa/staff/get?id=X |
| 创建业务专员 | POST | /admin-api/uaa/staff/create |
| 更新业务专员 | PUT | /admin-api/uaa/staff/update |
| 删除业务专员 | DELETE | /admin-api/uaa/staff/delete?id=X |
| 批量删除 | DELETE | /admin-api/uaa/staff/delete-list |
| 更新状态 | PUT | /admin-api/uaa/staff/update-status |
| 审核业务专员 | PUT | /admin-api/uaa/staff/audit |
| 业务专员统计 | GET | /admin-api/uaa/staff/statistics?id=X |

【功能需求】
1. 支持业务专员列表 CRUD
2. 业务专员类型(专职/兼职/特邀)
3. 机构关联(OrgSelect 下拉,来自 /uaa/org/simple-list)
4. 专业领域多选/从业年限配置
5. 审核状态流转(待审核/已通过/已拒绝)
6. 删除前案件关联校验
7. 业务专员统计信息(案件数/成功率等)

【UI 规范】
- UI 库: ant-design-vue
- 表格列:工号、真实姓名、性别、手机号、业务专员类型、机构名称、状态、审核状态、操作
- 字典类型: staff_type(业务专员类型)、audit_status(审核状态)、common_status(状态)
- 业务组件复用: DictTag、DictSelect、OrgSelect
- 操作按钮:新增、编辑、审核、删除

【参考实现】
请参考以下已实现模块:
- business-web/src/views/system/dict/index.vue(字典管理,适合作为表单+下拉选择参考)
- business-web/src/views/uaa/role/index.vue(角色管理,适合作为 CRUD 页面参考)
- business-web/src/views/uaa/org/index.vue(业务机构,适合作为关联模块参考)
- 复用 business-web/src/components/business 中的 DictTag/DictSelect/OrgSelect 组件

【测试验证】
实现完成后,使用 5.16.2 测试提示词中的测试场景验证,重点验证:
1. 表格列完整,业务专员类型和审核状态标签正确
2. 机构下拉正常,机构名称正确显示
3. 审核通过/拒绝流程完整
4. 删除前关联校验生效
5. 7 个测试场景全部通过

【交付物清单】
- [ ] 前端主页面 .vue(staff/index.vue)
- [ ] 前端表单弹窗 .vue(StaffFormModal.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册(/uaa/staff)
- [ ] 菜单注册(左侧菜单「用户中心」→「业务专员」)
- [ ] 权限码注册(system:staff:create 等)
- [ ] 字典数据初始化 SQL(staff_type、audit_status)
- [ ] 通过 5.16.2 所有测试场景
```

---

### 5.17 UAA-17 用户导入

**页面路径**: 左侧菜单「用户中心」→「用户管理」→「导入」按钮
**源码文件**: `src/views/uaa/user/UserImportForm.vue`
**API 文件**: `src/api/uaa/user.ts`
**权限标识**: `system:user:import`

#### 5.17.1 测试场景

#### 5.17.2 测试提示词

```
/browser 或 /open-gstack-browser
测试用户导入功能(Excel 批量导入)。

【前置操作】
1. 登录后进入「用户中心」→「用户管理」
2. 点击顶部「导入」按钮
3. 验证 UserImportForm 弹窗弹出

---

【测试场景 1:导入弹窗打开】
1. 验证弹窗标题为「导入用户」
2. 验证显示内容:下载模板按钮、文件上传区域(a-upload-dragger)、updateSupport 复选框、导入说明文本

预期结果:
✅ 导入弹窗正常打开
✅ 所有元素完整显示

---

【测试场景 2:下载导入模板】
1. 点击「下载模板」按钮
2. 验证触发文件下载(UserImportTemplate.xlsx)
3. 验证下载成功提示

预期结果:
✅ 模板下载功能正常

---

【测试场景 3:上传 Excel 文件】
1. 准备测试 Excel 文件(包含 2-3 条用户数据)
2. 拖拽文件到上传区域(或点击选择文件)
3. 验证文件上传进度条显示
4. 验证上传成功后显示文件名
5. 勾选「是否更新已存在用户」复选框
6. 点击「确定」按钮

预期结果:
✅ 文件上传功能正常
✅ 进度条显示
✅ 复选框可勾选

---

【测试场景 4:导入提交与结果】
1. 提交导入任务
2. 验证 API 调用 POST /admin-api/uaa/user/import
3. 验证显示导入进度(loading 遮罩)
4. 验证导入完成后显示结果统计:总条数、成功条数、失败条数、失败原因(如有)
5. 验证列表自动刷新,新增用户出现

预期结果:
✅ 导入 API 调用成功
✅ 结果统计完整
✅ 列表自动刷新

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 下载模板 | GET | /admin-api/uaa/user/get-import-template |
| 导入用户 | POST | /admin-api/uaa/user/import |

---

【问题诊断】
- 模板下载失败 → 检查后端是否实现 get-import-template 接口
- 上传后无响应 → 检查 file 参数是否正确传递(MultipartFile)
- 导入成功但列表未刷新 → 检查导入成功后是否调用 loadData()
```

#### 5.17.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【用户导入】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「用户管理」→「导入」按钮(弹窗模式)
- 路由: /uaa/user(列表)+ UserImportForm.vue(弹窗)
- 权限前缀: system:user
- 涉及权限码:
  - system:user:import(导入用户)
  - system:user:query(查询用户)

【前端文件清单】
- 主页面: business-web/src/views/uaa/user/index.vue
- 导入弹窗: business-web/src/views/uaa/user/UserImportForm.vue
- API 封装: business-web/src/api/uaa/user.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/user/UserController.java
- Service 接口: .../service/user/AdminUserService.java
- DTO/Request: .../controller/admin/user/vo/user/UserImportExcelVO.java
- DO: .../dal/dataobject/user/AdminUserDO.java
- Mapper: .../dal/mapper/user/AdminUserMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 下载模板 | GET | /admin-api/uaa/user/get-import-template |
| 导入用户 | POST | /admin-api/uaa/user/import |

【功能需求】
1. 用户列表顶部新增「导入」按钮
2. 点击导入打开弹窗,支持下载模板与上传 Excel
3. 支持 updateSupport 复选框(是否更新已存在用户)
4. 导入结果统计(总条数/成功条数/失败条数/失败原因)
5. 导入成功后自动刷新用户列表

【UI 规范】
- UI 库: ant-design-vue
- 弹窗组件: a-modal + a-upload-dragger
- 上传组件: a-upload,accept=".xls,.xlsx"
- 业务组件复用: 无
- 操作按钮:下载模板、确定、取消

【参考实现】
请参考以下已实现模块:
- business-web/src/views/uaa/user/UserImportForm.vue(本模块目标文件)
- business-web/src/views/system/user/UserImportForm.vue(如存在)
- 复用 business-web/src/components/business 中的 a-upload 组件

【测试验证】
实现完成后,使用 5.17.2 测试提示词中的测试场景验证,重点验证:
1. 导入弹窗完整显示
2. 模板下载与文件上传功能
3. 导入结果统计与列表刷新
4. 4 个测试场景全部通过

【交付物清单】
- [ ] 前端导入弹窗 .vue(UserImportForm.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现(importUserList)
- [ ] 后端 DTO(UserImportExcelVO、UserImportRespVO)
- [ ] 权限码注册(system:user:import)
- [ ] 通过 5.17.2 所有测试场景
```

---

### 5.18 UAA-18 角色数据权限分配

**页面路径**: 左侧菜单「用户中心」→「角色管理」→「数据权限」按钮
**源码文件**: `src/views/uaa/role/RoleDataPermissionForm.vue`
**API 文件**: `src/api/uaa/role.ts`, `src/api/uaa/dept.ts`
**权限标识**: `system:permission:assign-role-data-scope`

#### 5.18.1 测试场景

#### 5.18.2 测试提示词

```
/browser 或 /open-gstack-browser
测试角色数据权限分配功能。

【前置操作】
1. 登录后进入「用户中心」→「角色管理」
2. 找到任意角色,点击「数据权限」按钮
3. 等待 RoleDataPermissionForm 弹窗弹出

---

【测试场景 1:数据权限弹窗结构】
1. 验证弹窗标题为「数据权限」
2. 验证弹窗顶部显示角色信息(角色名称)
3. 验证数据权限范围单选框:全部数据、本部门及下级、本部门、仅本人、自定义
4. 验证默认选中当前角色的数据权限范围

预期结果:
✅ 弹窗标题正确
✅ 角色信息正确显示
✅ 5 种权限范围选项完整
✅ 默认选中状态正确

---

【测试场景 2:数据权限范围切换】
1. 选择「全部数据」
2. 验证无需选择部门树
3. 选择「本部门」
4. 验证部门树下拉正常
5. 选择「自定义」
6. 验证部门树完整加载(树形复选框)
7. 验证已有部门被勾选

预期结果:
✅ 权限范围切换正常
✅ 部门树根据权限范围正确显示/隐藏

---

【测试场景 3:自定义数据权限 - 部门树交互】
1. 选择「自定义」数据权限
2. 验证部门树完整加载
3. 勾选部分部门节点
4. 验证「已选择 X 个部门」计数实时更新
5. 点击「清空」,验证所有勾选清除

预期结果:
✅ 部门树完整加载
✅ 勾选计数实时更新
✅ 清空功能正常

---

【测试场景 4:提交数据权限】
1. 选择「自定义」数据权限
2. 勾选 3-5 个部门节点
3. 点击「确定」
4. 验证 API POST /admin-api/uaa/permission/assign-role-data-scope 成功
5. 验证请求体格式: `{ roleId, dataScope, deptIds }`
6. 验证显示「权限分配成功」
7. 验证弹窗关闭

预期结果:
✅ API 调用成功
✅ 请求体格式正确
✅ 权限分配成功提示

---

【测试场景 5:数据权限回显校验】
1. 关闭弹窗,重新打开同一角色的「数据权限」
2. 验证上一步设置的 dataScope 范围仍被选中
3. 验证自定义模式下,deptIds 中部门仍被勾选

预期结果:
✅ 数据权限范围正确回显
✅ 自定义部门正确回显

---

【测试场景 6:全部数据提交】
1. 选择「全部数据」权限范围
2. 不需要选择部门树
3. 点击「确定」
4. 验证 API 成功,deptIds 为 null/[]

预期结果:
✅ 全部数据提交成功
✅ deptIds 字段处理正确

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 分配角色数据权限 | POST | /admin-api/uaa/permission/assign-role-data-scope |
| 部门列表 | GET | /admin-api/uaa/dept/list |

---

【问题诊断】
- 部门树不加载 → 检查 deptApi.list() 是否成功
- 提交失败 → 检查请求体格式是否为 `{ roleId, dataScope, deptIds }`
- 部门树勾选状态不回显 → 检查后端返回的 deptIds 是否正确处理
```

#### 5.18.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【角色数据权限分配】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「角色管理」→「数据权限」按钮(弹窗模式)
- 路由: /uaa/role(列表)+ RoleDataPermissionForm.vue(弹窗)
- 权限前缀: system:permission
- 涉及权限码:
  - system:permission:assign-role-data-scope(分配角色数据权限)
  - system:role:query(查询角色)
  - system:dept:query(查询部门)

【前端文件清单】
- 主页面: business-web/src/views/uaa/role/index.vue
- 数据权限表单: business-web/src/views/uaa/role/RoleDataPermissionForm.vue
- API 封装: business-web/src/api/uaa/role.ts, business-web/src/api/uaa/dept.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/permission/PermissionController.java
- Service 接口: .../service/permission/PermissionService.java
- DTO/Request: .../controller/admin/permission/vo/dataScope/*ReqVO.java
- DO: .../dal/dataobject/permission/RoleDataScopeDO.java
- Mapper: .../dal/mapper/permission/RoleDataScopeMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 分配角色数据权限 | POST | /admin-api/uaa/permission/assign-role-data-scope |
| 部门列表 | GET | /admin-api/uaa/dept/list |

【功能需求】
1. 5 种数据权限范围:全部/本部门及下级/本部门/仅本人/自定义
2. 自定义模式下显示部门树(树形复选框)
3. 部门勾选计数实时更新
4. 提交时仅在自定义模式下传 deptIds
5. 权限范围与部门选择回显

【UI 规范】
- UI 库: ant-design-vue
- 弹窗组件: a-modal + a-radio-group + a-tree(可选)
- 表单字段:数据权限范围(单选)、部门树(自定义模式)
- 业务组件复用: 部门树选择器
- 操作按钮:清空、取消、确定

【参考实现】
请参考以下已实现模块:
- business-web/src/views/uaa/role/RoleDataPermissionForm.vue(本模块目标文件)
- business-web/src/views/system/role/RoleDataPermissionForm.vue(如存在)
- 复用 business-web/src/components/business 中的部门树选择器

【测试验证】
实现完成后,使用 5.18.2 测试提示词中的测试场景验证,重点验证:
1. 弹窗结构与 5 种权限范围选项
2. 部门树加载与勾选交互
3. 提交 API 与回显校验
4. 6 个测试场景全部通过

【交付物清单】
- [ ] 前端数据权限表单 .vue(RoleDataPermissionForm.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 权限码注册(system:permission:assign-role-data-scope)
- [ ] 通过 5.18.2 所有测试场景
```

---

### 5.19 UAA-19 登录日志详情

**页面路径**: 左侧菜单「用户中心」→「登录日志」→「详情」按钮
**源码文件**: `src/views/uaa/loginLog/LoginLogDetailModal.vue`
**API 文件**: `src/api/uaa/loginLog.ts`
**权限标识**: `system:login-log:query`

#### 5.19.1 测试场景

#### 5.19.2 测试提示词

```
/browser 或 /open-gstack-browser
测试登录日志详情功能。

【前置操作】
1. 登录后进入「用户中心」→「登录日志」
2. 等待页面加载完成

---

【测试场景 1:登录日志详情弹窗】
1. 找到任意登录日志记录
2. 点击「详情」按钮
3. 验证 LoginLogDetailModal 弹窗弹出
4. 验证详情包含以下全部字段:
   - 用户名(username)
   - 登录类型(logType):DictTag 标签
   - 登录结果(result):DictTag 标签
   - 用户IP(userIp)
   - UserAgent(userAgent)
   - 创建时间(createTime)
5. 验证所有字段正确显示,无数据缺失

预期结果:
✅ 详情弹窗正常打开
✅ 全部字段完整显示
✅ DictTag 标签正确渲染

---

【测试场景 2:不同日志类型详情】
1. 找到至少 2 条不同登录类型的日志(如账号密码/短信验证码)
2. 分别点击「详情」
3. 验证日志类型字段使用 DictTag 正确显示不同标签

预期结果:
✅ 不同登录类型标签正确
✅ DictTag 字典数据加载正常

---

【测试场景 3:成功/失败日志颜色区分】
1. 找到登录成功与登录失败的日志记录
2. 分别点击「详情」
3. 验证详情中登录结果字段颜色区分(成功=绿色/失败=红色)

预期结果:
✅ 登录结果颜色区分正确

---

【测试场景 4:User-Agent 字段展示】
1. 找到任意登录日志记录
2. 点击「详情」
3. 验证 UserAgent 字段完整展示(浏览器/操作系统信息)
4. 验证长字符串无截断或显示异常

预期结果:
✅ UserAgent 完整展示
✅ 长字符串不截断

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 登录日志详情 | GET | /admin-api/uaa/login-log/get?id=X |

---

【问题诊断】
- 详情弹窗字段缺失 → 检查 LoginLogDetailModal.vue 的字段绑定,确认所有字段名与 LoginLogVO 一致
- DictTag 不显示 → 检查 logType 和 result 是否使用了正确的字典类型标识
```

#### 5.19.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【登录日志详情】模块。

【模块信息】
- 服务: UAA(:8081)
- 页面路径: 左侧菜单「用户中心」→「登录日志」→「详情」按钮(弹窗模式)
- 路由: /uaa/login-log(列表)+ LoginLogDetailModal.vue(弹窗)
- 权限前缀: system:login-log
- 涉及权限码:
  - system:login-log:query(查询登录日志)

【前端文件清单】
- 主页面: business-web/src/views/uaa/loginLog/index.vue
- 详情弹窗: business-web/src/views/uaa/loginLog/LoginLogDetailModal.vue
- API 封装: business-web/src/api/uaa/loginLog.ts

【后端文件清单】
- Controller: business-platform-basic/business-module-uaa/business-module-uaa-server/src/main/java/com/example/uaa/controller/admin/logger/LoginLogController.java
- Service 接口: .../service/logger/LoginLogService.java
- DTO/Request: .../controller/admin/logger/vo/LoginLogVO.java
- DO: .../dal/dataobject/logger/LoginLogDO.java
- Mapper: .../dal/mapper/logger/LoginLogMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 登录日志详情 | GET | /admin-api/uaa/login-log/get?id=X |

【功能需求】
1. 详情弹窗完整字段展示:用户名、登录类型、登录结果、用户IP、UserAgent、创建时间
2. 登录类型/结果使用 DictTag 标签(支持颜色区分)
3. UserAgent 长字符串不截断
4. 关闭弹窗后数据清除

【UI 规范】
- UI 库: ant-design-vue
- 弹窗组件: a-modal + Descriptions
- 字段展示: Descriptions 组件,label/value 双列布局
- 业务组件复用: DictTag
- 操作按钮:关闭

【参考实现】
请参考以下已实现模块:
- business-web/src/views/uaa/loginLog/LoginLogDetailModal.vue(本模块目标文件)
- business-web/src/views/system/operatelog/index.vue(操作日志,适合作为详情弹窗参考)
- 复用 business-web/src/components/business 中的 DictTag 组件

【测试验证】
实现完成后,使用 5.19.2 测试提示词中的测试场景验证,重点验证:
1. 详情弹窗完整字段展示
2. DictTag 标签颜色区分
3. UserAgent 完整展示
4. 4 个测试场景全部通过

【交付物清单】
- [ ] 前端详情弹窗 .vue(LoginLogDetailModal.vue)
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO(LoginLogVO)
- [ ] 权限码注册(system:login-log:query)
- [ ] 通过 5.19.2 所有测试场景
```

---

*(注:UAA-01 到 UAA-19 的完整测试场景与本节示例一致,本模板结构与本服务 19 个模块 1:1 对齐。实际使用时可按需扩展每个模块的测试场景数量与边界用例)*

---


## 6.测试结果报告模板

```markdown
# UAA 服务测试报告

**测试日期**: YYYY-MM-DD  
**测试人员**: AI Agent (gstack /qa)  
**测试环境**: http://localhost:5173  
**UAA 服务**: http://localhost:8081

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
| UAA-01 登录与登出 | 6 | 6 | 0 | 0 | 100% |
| UAA-02 用户管理 | 10 | 9 | 1 | 0 | 90% |
| UAA-03 角色管理 | 11 | 11 | 0 | 0 | 100% |
| UAA-04 角色菜单赋权 | 8 | 8 | 0 | 0 | 100% |
| UAA-05 菜单管理 | 9 | 9 | 0 | 0 | 100% |
| UAA-06 部门管理 | 8 | 8 | 0 | 0 | 100% |
| UAA-07 岗位管理 | 7 | 7 | 0 | 0 | 100% |
| UAA-08 租户管理 | 5 | 5 | 0 | 0 | 100% |
| UAA-09 租户套餐管理 | 5 | 5 | 0 | 0 | 100% |
| UAA-10 OAuth2 客户端管理 | 4 | 4 | 0 | 0 | 100% |
| UAA-11 OAuth2 Token 管理 | 3 | 3 | 0 | 0 | 100% |
| UAA-12 社交用户管理 | 8 | 8 | 0 | 0 | 100% |
| UAA-13 社交客户端管理 | 9 | 9 | 0 | 0 | 100% |
| UAA-14 登录日志 | 7 | 7 | 0 | 0 | 100% |
| UAA-15 业务机构管理 | 4 | 4 | 0 | 0 | 100% |
| UAA-16 业务专员管理 | 7 | 7 | 0 | 0 | 100% |
| UAA-17 用户导入 | 4 | 4 | 0 | 0 | 100% |
| UAA-18 角色数据权限分配 | 6 | 6 | 0 | 0 | 100% |
| UAA-19 登录日志详情 | 4 | 4 | 0 | 0 | 100% |
| **合计** | **N** | **X** | **Y** | **Z** | **X/N%** |

## 失败用例详情

### UAA-02 用户管理 — 测试场景 4:编辑用户

- **现象**: 点击编辑后弹窗未弹出
- **控制台错误**: `TypeError: Cannot read properties of undefined`
- **截图**: [.gstack/qa-reports/screenshots/issue-001.png](.gstack/qa-reports/screenshots/issue-001.png)
- **定位**: `UserFormModal.vue` 第 XX 行,`watch` 未正确监听 props.data
- **修复建议**: 在 watch 中添加 `deep: true` 或使用 `immediate: true`
- **修复文件**: `src/views/uaa/user/UserFormModal.vue`
- **修复内容**: 在 watch 中添加 `{ immediate: true }` 选项
- **状态**: 待修复

## 建议

1. **高优先级**:修复用户管理编辑弹窗问题,影响日常使用
2. **中优先级**:增强租户套餐的菜单关联功能(目前仅显示 ID)
3. **低优先级**:补充部分组件的 ARIA 无障碍标签
4. **测试补充**:建议增加 API 层的单元测试覆盖
```

---

## 7.模块开发对照与补全清单

本章节对照 example-ui(参考项目)与 business-platform(当前项目),梳理 UAA 模块的实现状态与补全建议。

前端已有页面:user、role、menu、dept、post、tenant、tenantPackage、oauth2Client、oauth2Token、socialUser、socialClient、loginLog、org、staff 共 14 个页面。

后端已有端点:UserController、RoleController、MenuController、DeptController、PostController、TenantController、TenantPackageController、OAuth2ClientController、OAuth2TokenController、SocialUserController、SocialClientController、LoginLogController、OrgController、StaffController、PermissionController 共 15 个 Controller。

对比参考项目,当前项目已覆盖所有核心 UAA 模块,并额外实现了业务机构和业务专员两个业务模块。

### UAA 模块补全对照表

| 模块 | example-ui 路径 | business-platform 路径 | 前端状态 | 后端状态 | 补全建议 |
|------|-----------------|----------------------|---------|---------|---------|
| 用户管理 | system/user/index.vue | uaa/user/index.vue | ✅ 完整 | ✅ | 对齐:增加导入用户、角色分配详情页 |
| 角色管理 | system/role/index.vue | uaa/role/index.vue | ✅ 完整(菜单权限树) | ✅ | 已完成:RoleForm + RoleMenuTree + RoleAssignMenuModal |
| 角色菜单赋权 | system/role/RoleAssignMenuForm.vue | uaa/role/RoleAssignMenuModal.vue | ✅ 完整(父子联动) | ✅ | 已完成:checkStrictly + 手动 findAncestorIds 联动 |
| 菜单管理 | system/menu/index.vue | uaa/menu/index.vue | ✅ 完整 | ✅ | 对齐:增加菜单类型字段更多交互 |
| 部门管理 | system/dept/index.vue | uaa/dept/index.vue | ✅ 完整 | ✅ | 对齐:负责人改为下拉选择用户 |
| 岗位管理 | system/post/index.vue | uaa/post/index.vue | ✅ 完整 | ✅ | 对齐:无明显差距 |
| 租户管理 | system/tenant/index.vue | uaa/tenant/index.vue | ✅ 完整 | ✅ | 对齐:增加租户套餐下拉 |
| 租户套餐管理 | system/tenantPackage/index.vue | uaa/tenantPackage/index.vue | ✅ 完整(菜单树) | ✅ | 已完成:完整菜单树形选择交互 |
| OAuth2 客户端管理 | system/oauth2/client/index.vue | uaa/oauth2Client/index.vue | ✅ 完整 | ✅ | 对齐:无明显差距 |
| OAuth2 Token 管理 | system/oauth2/token/index.vue | uaa/oauth2Token/index.vue | ✅ 完整 | ✅ | 对齐:无明显差距 |
| 社交用户管理 | system/social/user/index.vue | uaa/socialUser/index.vue | ✅ 完整 | ✅ | 对齐:增加详情查看弹窗 |
| 社交客户端管理 | system/social/client/index.vue | uaa/socialClient/index.vue | ✅ 完整 | ✅ | 对齐:无明显差距 |
| 登录日志 | system/loginlog/index.vue | uaa/loginLog/index.vue | ✅ 完整(含详情弹窗) | ✅ | 已完成:LoginLogDetailModal.vue |
| **业务机构管理** | ❌ 无(业务模块) | 后端 OrgController.java | ✅ 前端已创建 | ✅ 完整 | 已完成:org/index.vue + org/OrgFormModal.vue + org.ts API |
| **业务专员管理** | ❌ 无(业务模块) | 后端 StaffController.java | ✅ 前端已创建 | ✅ 完整 | 已完成:staff/index.vue + staff/StaffFormModal.vue + staff.ts API |
| 用户导入 | system/user/UserImportForm.vue | uaa/user/UserImportForm.vue | ✅ 已创建 | ✅ | 已完成:UserImportForm.vue Excel导入功能 |
| 角色数据权限分配 | system/role/RoleDataPermissionForm.vue | uaa/role/RoleDataPermissionForm.vue | ✅ 已创建 | ✅ | 已完成:部门树+数据权限范围选择 |
| 登录日志详情 | system/loginlog/LoginLogDetail.vue | uaa/loginLog/LoginLogDetailModal.vue | ✅ 已创建 | ✅ | 已完成:详情弹窗完整字段 |


## 8.文档版本

| 版本 | 日期 | 修改内容 | 作者 |
|------|------|---------|------|
| 1.0.0 | 2026-06-17 | 严格对齐 case-scene-template.md 模板结构,生成 UAA 服务完整测试文档,涵盖 19 个模块 126 个测试场景 | AI Agent |

| 项目 | 值 |
|------|------|
| 适用服务 | UAA(:8081) |
| 前端入口 | http://localhost:5173 |
| 默认账号 | admin / admin123 |
| 默认租户 ID | 1 |
| 测试 Skill | `/qa`, `/qa-only`, `/open-gstack-browser`, MCP Playwright |

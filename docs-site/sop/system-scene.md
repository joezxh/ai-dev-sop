# System 服务 — 端到端浏览器自动化测试 Skill

> 本文档为 AI 浏览器测试 Agent 提供完整的 System(系统管理)服务测试提示词。
> 使用 `/browser` 启动浏览器自动化测试,或使用 MCP Playwright 执行测试。
> 适用于 gstack `/qa` 和 `/qa-only` Skill,通过 `/open-gstack-browser` 导入认证 Cookie 后执行端到端测试。
> 文档同时包含问题自动定位与修复建议机制,支持端到端回归测试。

---

## 目录

- [1.测试前置条件](#测试前置条件)
- [2.全局测试策略](#全局测试策略)
- [3.Skill 调用方式](#skill-调用方式)
- [4.问题发现与自动修复流程](#问题发现与自动修复流程)
- [5.System 服务测试模块](#system-服务测试模块)
  - [SYS-01 参数配置](#sys-01-参数配置)
  - [SYS-02 字典管理](#sys-02-字典管理)
  - [SYS-03 通知公告](#sys-03-通知公告)
  - [SYS-04 站内信管理](#sys-04-站内信管理)
  - [SYS-05 文件管理](#sys-05-文件管理)
  - [SYS-06 文件配置管理](#sys-06-文件配置管理)
  - [SYS-07 邮件管理](#sys-07-邮件管理)
  - [SYS-08 短信管理](#sys-08-短信管理)
  - [SYS-09 操作日志](#sys-09-操作日志)
  - [SYS-10 我的站内信](#sys-10-我的站内信)
  - [SYS-11 站内信详情](#sys-11-站内信详情消息模板发送后详情)
  - [SYS-12 短信渠道管理(增强)](#sys-12-短信渠道管理增强)
  - [SYS-13 短信模板管理](#sys-13-短信模板管理)
  - [SYS-14 短信日志管理](#sys-14-短信日志管理)
  - [SYS-15 邮件账号管理](#sys-15-邮件账号管理)
  - [SYS-16 邮件模板管理](#sys-16-邮件模板管理)
  - [SYS-17 邮件日志管理](#sys-17-邮件日志管理)
  - [SYS-18 API 访问日志](#sys-18-api-访问日志)
  - [SYS-19 API 错误日志](#sys-19-api-错误日志)
  - [SYS-20 Redis 缓存监控](#sys-20-redis-缓存监控)
  - [SYS-21 数据源配置](#sys-21-数据源配置)
  - [SYS-22 Druid SQL 监控](#sys-22-druid-sql-监控)
  - [SYS-23 服务节点监控](#sys-23-服务节点监控)
  - [SYS-24 Swagger/API 文档](#sys-24-swaggerapi-文档)
- [6.测试结果报告模板](#测试结果报告模板)
- [7.模块开发对照与补全清单](#模块开发对照与补全清单)
- [8.文档版本]
  

---

## 1.测试前置条件

### 1.1 环境要求

| 项目 | 值 |
|------|------|
| 前端地址 | `http://localhost:5173` |
| 网关地址 | `http://localhost:8080` |
| System 服务 | `http://localhost:8082` |
| 超级管理员账号 | `admin` |
| 超级管理员密码 | `admin123` |
| 默认租户 ID | `1` |

### 1.2 服务依赖检查

测试前需确认以下服务已启动:

1. **Gateway**(:8080) — API 网关
2. **System**(:8082) — 系统公共服务
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
| SYS-01 | 参数配置 | 6 | CRUD + 分类筛选 + 可见性 |
| SYS-02 | 字典管理 | 10 | 类型+数据双表 CRUD |
| SYS-03 | 通知公告 | 6 | CRUD + 富文本 |
| SYS-04 | 站内信管理 | 7 | 模板+消息+标记已读 |
| SYS-05 | 文件管理 | 11 | 列表/网格/预览/拖拽/批量 |
| SYS-06 | 文件配置管理 | 8 | 配置 CRUD + 测试连接 |
| SYS-07 | 邮件管理 | 6 | 账号/模板/日志 Tab |
| SYS-08 | 短信管理 | 5 | 渠道/模板/日志 Tab |
| SYS-09 | 操作日志 | 5 | 查看 + 详情 |
| SYS-10 | 我的站内信 | 4 | 查看 + 标记已读 |
| SYS-11 | 站内信详情 | 3 | 详情弹窗 |
| SYS-12 | 短信渠道管理(增强) | 8 | 完整 CRUD |
| SYS-13 | 短信模板管理 | 9 | 完整 CRUD |
| SYS-14 | 短信日志管理 | 7 | 查看 + 详情 |
| SYS-15 | 邮件账号管理 | 8 | 完整 CRUD |
| SYS-16 | 邮件模板管理 | 9 | 完整 CRUD |
| SYS-17 | 邮件日志管理 | 7 | 查看 + 详情 |
| SYS-18 | API 访问日志 | 4 | 查看 + 导出 |
| SYS-19 | API 错误日志 | 7 | 查看 + 处理状态流转 |
| SYS-20 | Redis 缓存监控 | 4 | 仪表盘 + 图表 |
| SYS-21 | 数据源配置 | 6 | CRUD + 批量删除 |
| SYS-22 | Druid SQL 监控 | 1 | iframe 内嵌 |
| SYS-23 | 服务节点监控 | 1 | iframe 内嵌 |
| SYS-24 | Swagger/API 文档 | 1 | iframe 内嵌 |
| **合计** | **24 个模块** | **135 个场景** | **~8h 测试时间** |

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
/qa-only --tier exhaustive --scope "System服务-字典管理"
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

## 5.System 服务测试模块

---

### 5.1 SYS-01 参数配置

**页面路径**: 左侧菜单「系统管理」→「参数配置」  
**源码文件**: `src/views/system/config/index.vue`, `src/views/system/config/ConfigFormModal.vue`  
**API 文件**: `src/api/system/config.ts`  
**权限标识**: `system:config:create`, `system:config:update`, `system:config:delete`

#### 5.1.1 测试场景

#### 5.1.2 测试提示词

```
/browser 或 /open-gstack-browser
打开参数配置页面,执行完整的参数配置 CRUD 测试。

【前置操作】
1. 使用 admin/admin123 登录系统
2. 在左侧菜单点击「系统管理」→「参数配置」
3. 等待页面加载完成

---

【测试场景 1:列表加载与基础显示】
1. 验证参数列表正常加载
2. 检查列:配置分类、配置名称、配置键名、配置值、可见、备注、创建时间、操作
3. 验证分页功能正常(showSizeChanger、showTotal)
4. 验证可见列使用 Tag 标签(可见=绿色/隐藏=红色)
5. 验证配置值列超长时 ellipsis 截断

预期结果:
✅ 表格8列完整
✅ 可见 Tag 颜色正确
✅ 分页正常

---

【测试场景 2:搜索功能】
1. 在「配置名称」输入框输入关键词,点击「搜索」
2. 验证列表过滤出匹配配置名称的结果
3. 在「配置键名」输入框输入关键词,点击「搜索」
4. 验证列表过滤出匹配配置键名的结果
5. 在「配置分类」下拉框选择(系统/业务/用户),点击「搜索」
6. 验证列表过滤出匹配分类的结果
7. 点击「重置」,验证条件清空,列表恢复

预期结果:
✅ 配置名称搜索过滤正确
✅ 配置键名搜索过滤正确
✅ 配置分类下拉过滤正确
✅ 重置清空条件

---

【测试场景 3:新增配置】
1. 点击顶部「新增配置」按钮
2. 验证 ConfigFormModal 弹窗弹出,标题为「新增配置」
3. 验证表单字段:
   - 配置分类(必填,下拉:系统/业务/用户)
   - 配置名称(必填)
   - 配置键名(必填,唯一)
   - 配置值(必填)
   - 可见(开关)
   - 备注(可选)
4. 填写:
   - 配置分类:系统
   - 配置名称:测试参数
   - 配置键名:test.param.key
   - 配置值:test_value_001
   - 可见:开启
   - 备注:自动化测试创建
5. 提交并验证:
   - API POST /admin-api/system/config/create
   - 显示「新增成功」
   - 弹窗关闭,列表刷新

预期结果:
✅ 新增弹窗表单字段完整
✅ 提交成功,列表刷新

---

【测试场景 4:编辑配置】
1. 点击刚创建配置的「编辑」按钮
2. 验证弹窗弹出,数据正确回填
3. 修改配置值为 test_value_001_modified
4. 提交并验证更新

预期结果:
✅ 编辑弹窗正确回填
✅ 修改提交后数据更新

---

【测试场景 5:配置键名唯一性校验】
1. 新增配置,键名设为已存在的键名
2. 提交验证:后端返回重复错误
3. 弹窗不关闭

预期结果:
✅ 后端唯一性校验生效

---

【测试场景 6:删除配置】
1. 点击配置的「删除」按钮
2. 验证 Popconfirm 确认弹窗,文案为「确定删除该配置？」
3. 点击确认,验证删除成功
4. 验证列表刷新,配置消失

预期结果:
✅ 删除确认正常
✅ 删除成功后列表刷新

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表分页 | GET | /admin-api/system/config/page |
| 配置详情 | GET | /admin-api/system/config/get?id=X |
| 按键名获取值 | GET | /admin-api/system/config/get-value-by-key?configKey=X |
| 创建配置 | POST | /admin-api/system/config/create |
| 更新配置 | PUT | /admin-api/system/config/update |
| 删除配置 | DELETE | /admin-api/system/config/delete?id=X |

---

【问题诊断】
- 列表不加载 → 检查 configApi.page() 是否成功
- 编辑时数据不回填 → 检查 ConfigFormModal watch 逻辑
- 删除失败 → 检查配置是否被系统引用(后端保护)
- 分类筛选无效 → 检查前端 category 映射到后端 type 参数
```

#### 5.1.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【参数配置】模块。

【模块信息】
- 服务: System(:8082)
- 页面路径: 左侧菜单「系统管理」→「参数配置」
- 路由: /system/config
- 权限前缀: system:config:
- 涉及权限码:
  - system:config:create(新增配置)
  - system:config:update(编辑配置)
  - system:config:delete(删除配置)
  - system:config:query(查询配置)

【前端文件清单】
- 主页面: business-web/src/views/system/config/index.vue
- 表单弹窗: business-web/src/views/system/config/ConfigFormModal.vue
- API 封装: business-web/src/api/system/config.ts

【后端文件清单】
- Controller: business-basic/business-module-system/business-module-system-server/src/main/java/.../controller/ConfigController.java
- Service 接口: .../service/ConfigService.java
- DO: .../dal/dataobject/ConfigDO.java
- Mapper: .../dal/mapper/ConfigMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | /admin-api/system/config/page |
| 配置详情 | GET | /admin-api/system/config/get?id=X |
| 按键名获取值 | GET | /admin-api/system/config/get-value-by-key?configKey=X |
| 创建配置 | POST | /admin-api/system/config/create |
| 更新配置 | PUT | /admin-api/system/config/update |
| 删除配置 | DELETE | /admin-api/system/config/delete?id=X |

【功能需求】
1. 配置列表,字段:配置分类、配置名称、配置键名、配置值、可见、备注、创建时间、操作
2. 搜索:配置名称(Input)、配置键名(Input)、配置分类(Select:系统/业务/用户)
3. 新增/编辑表单弹窗(ConfigFormModal),字段:配置分类、配置名称、配置键名、配置值、可见(Switch)、备注
4. 配置键名唯一性校验(后端)
5. 可见列使用 Tag 展示(可见=绿色/隐藏=红色)
6. 删除使用 Popconfirm 确认

【UI 规范】
- UI 库: ant-design-vue
- 搜索栏: Row > Col > Input/Select + Button(搜索/重置/新增配置)
- 表格: Table + pagination(showSizeChanger, showTotal)
- 弹窗: ConfigFormModal(v-model:open)
- 标签: Tag(可见/隐藏)

【参考实现】
请参考以下已实现模块:
- business-web/src/views/system/dict/index.vue(字典管理,适合作为 CRUD 参考)
- business-web/src/views/system/notice/index.vue(通知公告,适合作为弹窗表单参考)

【测试验证】
实现完成后,使用 5.1.2 测试提示词中的测试场景验证,重点验证:
1. 表格 8 列完整渲染
2. 配置名称/键名/分类三个搜索条件正常
3. 新增/编辑弹窗表单字段完整
4. 可见 Tag 颜色正确(绿/红)
5. 删除 Popconfirm 确认正常

【交付物清单】
- [ ] 前端主页面 index.vue
- [ ] 前端表单弹窗 ConfigFormModal.vue
- [ ] 前端 API 封装 config.ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DO / Mapper
- [ ] 路由注册(/system/config)
- [ ] 菜单注册(左侧菜单「系统管理」→「参数配置」)
- [ ] 权限码注册(system:config:create/update/delete)
- [ ] 通过 5.1.2 所有测试场景
```

---

### 5.2 SYS-02 字典管理

**页面路径**: 左侧菜单「系统管理」→「字典管理」  
**源码文件**: `src/views/system/dict/index.vue`, `src/views/system/dict/DictTypeFormModal.vue`, `src/views/system/dict/DictDataFormModal.vue`  
**API 文件**: `src/api/system/dict/type.ts`, `src/api/system/dict/data.ts`  
**权限标识**: `system:dict-type:create`, `system:dict-type:update`, `system:dict-type:delete`, `system:dict-data:create`, `system:dict-data:update`, `system:dict-data:delete`

#### 5.2.1 测试场景

#### 5.2.2 测试提示词

```
/browser 或 /open-gstack-browser
打开字典管理页面,执行完整的字典类型 + 字典数据 CRUD 测试。

【前置操作】
1. 使用 admin/admin123 登录系统
2. 在左侧菜单点击「系统管理」→「字典管理」
3. 等待页面加载完成


---

【测试场景 1:字典类型列表加载】
1. 验证字典类型列表正常加载
2. 检查列:字典名称、字典类型、状态、备注、创建时间、操作
3. 验证分页功能正常(默认每页5条)
4. 验证状态列使用 DictTag(common_status)标签

预期结果:
✅ 表格6列完整
✅ DictTag 状态标签正确
✅ 分页正常(默认5条/页)

---

【测试场景 2:字典类型搜索】
1. 在「字典名称」输入框输入关键词,点击「搜索」
2. 验证列表过滤出匹配结果
3. 点击「重置」,验证条件清空并重置分页

预期结果:
✅ 名称搜索过滤正确
✅ 重置清空条件

---

【测试场景 3:新增字典类型】
1. 点击「新增类型」按钮
2. 验证 DictTypeFormModal 弹窗弹出
3. 验证表单字段:字典名称(必填)、字典类型(必填,唯一)、状态(单选)、备注(可选)
4. 填写:名称=测试字典,类型=test_dict,状态=正常
5. 提交并验证列表刷新

预期结果:
✅ 新增表单字段完整
✅ 提交成功,列表刷新

---

【测试场景 4:编辑字典类型】
1. 点击字典类型的「编辑」按钮
2. 验证数据正确回填
3. 修改备注为「已修改」
4. 提交并验证更新

预期结果:
✅ 编辑弹窗正确回填
✅ 修改后数据更新

---

【测试场景 5:查看字典数据】
1. 点击字典类型的「数据」按钮
2. 验证页面下方展示字典数据列表(Divider 分隔)
3. 验证数据列:字典标签、字典值、排序、状态、备注、操作
4. 验证标题显示:「字典数据：XXX（ xxx ）」

预期结果:
✅ 字典数据列表正确展示
✅ Divider 标题正确

---

【测试场景 6:新增字典数据】
1. 在字典数据区域点击「新增数据」按钮
2. 验证 DictDataFormModal 弹窗弹出
3. 验证字段:字典标签(必填)、字典值(必填)、排序(数字)、状态、备注
4. 填写:标签=选项A,值=1,排序=1
5. 提交并验证数据列表刷新

预期结果:
✅ 新增数据表单字段完整
✅ 提交成功,数据列表刷新

---

【测试场景 7:编辑字典数据】
1. 点击字典数据的「编辑」按钮
2. 验证数据回填
3. 修改标签为「选项A-修改」
4. 提交并验证

预期结果:
✅ 编辑正确回填
✅ 修改生效

---

【测试场景 8:删除字典数据】
1. 点击字典数据的「删除」按钮
2. 验证 Modal.confirm 确认弹窗
3. 确认删除,验证列表刷新

预期结果:
✅ 删除确认正常
✅ 删除后数据列表刷新

---

【测试场景 9:删除字典类型】
1. 点击字典类型的「删除」按钮
2. 验证 Modal.confirm 确认弹窗
3. 确认删除,验证类型列表刷新
4. 如果当前正在查看该类型的数据,验证数据区域消失

预期结果:
✅ 删除确认正常
✅ 类型列表刷新,关联数据区域清空

---

【测试场景 10:返回按钮】
1. 在字典数据区域点击「返回」按钮
2. 验证数据区域消失,回到纯类型列表视图

预期结果:
✅ 返回后数据区域隐藏

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 类型分页 | GET | /admin-api/system/dict-type/page |
| 类型详情 | GET | /admin-api/system/dict-type/get/{id} |
| 创建类型 | POST | /admin-api/system/dict-type/create |
| 更新类型 | PUT | /admin-api/system/dict-type/update |
| 删除类型 | DELETE | /admin-api/system/dict-type/delete?id=X |
| 数据分页 | GET | /admin-api/system/dict-data/page |
| 数据列表 | GET | /admin-api/system/dict-data/list |
| 创建数据 | POST | /admin-api/system/dict-data/create |
| 更新数据 | PUT | /admin-api/system/dict-data/update |
| 删除数据 | DELETE | /admin-api/system/dict-data/delete?id=X |
| 批量删除数据 | DELETE | /admin-api/system/dict-data/delete-list?ids=X |

---

【问题诊断】
- 类型列表不加载 → 检查 dictTypeApi.page() 是否成功
- 数据列表不加载 → 检查 dictDataApi.page() 的 dictType 参数
- DictTag 不显示 → 检查 common_status 字典是否注册
- 删除类型后数据区仍在 → 检查 activeType 清空逻辑
```

#### 5.2.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【字典管理】模块。

【模块信息】
- 服务: System(:8082)
- 页面路径: 左侧菜单「系统管理」→「字典管理」
- 路由: /system/dict
- 权限前缀: system:dict-type: / system:dict-data:

【前端文件清单】
- 主页面: business-web/src/views/system/dict/index.vue
- 类型弹窗: business-web/src/views/system/dict/DictTypeFormModal.vue
- 数据弹窗: business-web/src/views/system/dict/DictDataFormModal.vue
- API 封装: business-web/src/api/system/dict/type.ts, data.ts

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 类型分页 | GET | /admin-api/system/dict-type/page |
| 创建类型 | POST | /admin-api/system/dict-type/create |
| 更新类型 | PUT | /admin-api/system/dict-type/update |
| 删除类型 | DELETE | /admin-api/system/dict-type/delete?id=X |
| 数据分页 | GET | /admin-api/system/dict-data/page |
| 创建数据 | POST | /admin-api/system/dict-data/create |
| 更新数据 | PUT | /admin-api/system/dict-data/update |
| 删除数据 | DELETE | /admin-api/system/dict-data/delete?id=X |

【功能需求】
1. 字典类型列表 + 字典数据列表(主从表结构)
2. 类型搜索:字典名称;数据搜索:无(通过类型切换)
3. 类型 CRUD + 数据 CRUD
4. 状态列使用 DictTag(common_status)
5. 删除类型时,如果正在查看该类型数据,需清空数据区域
6. 数据区域使用 Divider 分隔,标题显示类型名称和编码

【UI 规范】
- UI 库: ant-design-vue
- 类型表格: Table + pagination(默认5条/页)
- 数据表格: Table(无分页,pageSize=100)
- 弹窗: DictTypeFormModal / DictDataFormModal(v-model:open)
- 组件: DictTag(@/components/business/Dict)

【交付物清单】
- [ ] 前端主页面 index.vue
- [ ] 前端 DictTypeFormModal.vue + DictDataFormModal.vue
- [ ] 前端 API 封装 type.ts + data.ts
- [ ] 后端 Controller + Service + DO + Mapper
- [ ] 路由注册 + 菜单注册 + 权限码注册
- [ ] 通过 5.2.2 所有测试场景
```

---

### 5.3 SYS-03 通知公告

**页面路径**: 左侧菜单「系统管理」→「通知公告」  
**源码文件**: `src/views/system/notice/index.vue`, `src/views/system/notice/NoticeFormModal.vue`  
**API 文件**: `src/api/system/notice.ts`  
**权限标识**: `system:notice:create`, `system:notice:update`, `system:notice:delete`

#### 5.3.1 测试场景

#### 5.3.2 测试提示词

```
/browser 或 /open-gstack-browser
打开通知公告页面,执行完整的通知公告 CRUD 测试。

【前置操作】
1. 使用 admin/admin123 登录系统
2. 在左侧菜单点击「系统管理」→「通知公告」
3. 等待页面加载完成

---

【测试场景 1:列表加载与基础显示】
1. 验证公告列表正常加载
2. 检查列:标题、类型、状态、创建时间、更新时间、操作
3. 验证分页功能正常
4. 验证类型列:通知=蓝色Tag,公告=橙色Tag
5. 验证状态列:正常=绿色Tag,关闭=红色Tag

预期结果:
✅ 表格6列完整
✅ 类型/状态 Tag 颜色正确
✅ 分页正常

---

【测试场景 2:搜索功能】
1. 在「公告标题」输入关键词搜索
2. 在「公告类型」下拉选择(通知/公告)搜索
3. 在「状态」下拉选择(正常/关闭)搜索
4. 点击「重置」,验证条件清空

预期结果:
✅ 标题/类型/状态三维过滤正确
✅ 重置清空条件

---

【测试场景 3:新增公告】
1. 点击「新增公告」按钮
2. 验证 NoticeFormModal 弹窗弹出
3. 验证表单字段:标题(必填)、类型(单选:通知/公告)、状态(单选:正常/关闭)、内容(富文本)
4. 填写完整并提交
5. 验证 API POST /admin-api/system/notice/create

预期结果:
✅ 新增弹窗含富文本编辑器
✅ 提交成功,列表刷新

---

【测试场景 4:编辑公告】
1. 点击「编辑」按钮,验证数据回填
2. 修改标题并提交
3. 验证更新时间字段有变化

预期结果:
✅ 编辑回填正确
✅ 更新时间刷新

---

【测试场景 5:删除公告】
1. 点击「删除」按钮
2. 验证 Popconfirm「确定删除该公告？」
3. 确认删除,验证列表刷新

预期结果:
✅ 删除确认+列表刷新

---

【测试场景 6:富文本编辑器】
1. 打开新增弹窗,验证富文本编辑器(wangeditor)正常渲染
2. 输入格式化内容(加粗、列表)
3. 提交后验证内容保存正确

预期结果:
✅ 富文本编辑器正常
✅ 格式化内容保存正确

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表分页 | GET | /admin-api/system/notice/page |
| 公告详情 | GET | /admin-api/system/notice/get?id=X |
| 创建公告 | POST | /admin-api/system/notice/create |
| 更新公告 | PUT | /admin-api/system/notice/update |
| 删除公告 | DELETE | /admin-api/system/notice/delete?id=X |

---

【问题诊断】
- 富文本不渲染 → 检查 wangeditor 是否正确引入
- 类型 Tag 不显示 → 检查 record.type 值是否为 1/2
- 编辑内容不回填 → 检查 NoticeFormModal watch + nextTick
```

#### 5.3.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【通知公告】模块。

【模块信息】
- 服务: System(:8082)
- 页面路径: 左侧菜单「系统管理」→「通知公告」
- 路由: /system/notice
- 权限前缀: system:notice:

【前端文件清单】
- 主页面: business-web/src/views/system/notice/index.vue
- 表单弹窗: business-web/src/views/system/notice/NoticeFormModal.vue
- API 封装: business-web/src/api/system/notice.ts

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | /admin-api/system/notice/page |
| 公告详情 | GET | /admin-api/system/notice/get?id=X |
| 创建公告 | POST | /admin-api/system/notice/create |
| 更新公告 | PUT | /admin-api/system/notice/update |
| 删除公告 | DELETE | /admin-api/system/notice/delete?id=X |

【功能需求】
1. 公告列表,字段:标题、类型(Tag)、状态(Tag)、创建时间、更新时间、操作
2. 搜索:标题(Input)、类型(Select:通知/公告)、状态(Select:正常/关闭)
3. 新增/编辑表单:标题、类型(Radio)、状态(Radio)、内容(富文本编辑器 wangeditor)
4. 删除使用 Popconfirm

【交付物清单】
- [ ] 前端 index.vue + NoticeFormModal.vue
- [ ] 前端 API 封装 notice.ts
- [ ] 后端 Controller + Service + DO + Mapper
- [ ] 路由 + 菜单 + 权限码注册
- [ ] 通过 5.3.2 所有测试场景
```

---

### 5.4 SYS-04 站内信管理

**页面路径**: 左侧菜单「系统管理」→「站内信管理」  
**源码文件**: `src/views/system/notify/index.vue`, `NotifyTemplateFormModal.vue`, `NotifySendModal.vue`, `NotifyMessageDetailModal.vue`  
**API 文件**: `src/api/system/notify.ts`  
**权限标识**: `system:notify-template:create`, `system:notify-template:update`, `system:notify-template:delete`, `system:notify-template:send-notify`

#### 5.4.1 测试场景

#### 5.4.2 测试提示词

```
/browser 或 /open-gstack-browser
打开站内信管理页面(Tab 结构:消息模板 + 站内信),执行完整测试。

【前置操作】
1. 登录系统,进入「系统管理」→「站内信管理」
2. 验证页面包含两个 Tab:「消息模板」和「站内信」

---

【测试场景 1:消息模板列表】
1. 验证默认激活「消息模板」Tab
2. 检查列:模板编码、模板名称、模板类型、发送人、模板内容、状态、创建时间、操作
3. 搜索:模板编码(Input)、模板名称(Input)、状态(Select)
4. 验证分页正常

预期结果:
✅ 模板列表8列完整
✅ Tab 切换正常

---

【测试场景 2:新增/编辑模板】
1. 点击「新增模板」,验证 NotifyTemplateFormModal 弹窗
2. 表单字段:模板名称、模板编码、模板类型、发送人昵称、模板内容、状态、备注
3. 填写并提交
4. 编辑已创建模板,验证回填

预期结果:
✅ 模板 CRUD 正常

---

【测试场景 3:发送站内信】
1. 在模板列表点击「发送」按钮
2. 验证 NotifySendModal 弹窗
3. 填写用户ID、模板参数
4. 提交后切换到「站内信」Tab,验证消息出现

预期结果:
✅ 发送弹窗正常
✅ 消息出现在站内信列表

---

【测试场景 4:站内信列表】
1. 切换到「站内信」Tab
2. 检查列:模板编码、发送人、消息内容、模板类型、是否已读、创建时间、操作
3. 搜索:模板编码(Input)、模板类型(Select)
4. 验证已读状态 Tag(已读=默认色/未读=蓝色)

预期结果:
✅ 站内信列表7列完整
✅ 已读状态 Tag 正确

---

【测试场景 5:查看消息详情】
1. 点击「查看」按钮
2. 验证 NotifyMessageDetailModal 弹窗展示消息详情
3. 验证弹窗关闭后自动标记已读

预期结果:
✅ 详情弹窗正常
✅ 关闭后标记已读

---

【测试场景 6:标记已读】
1. 对未读消息点击「标记已读」按钮
2. 验证调用 updateRead API
3. 验证 Tag 更新为「已读」
4. 验证列表刷新

预期结果:
✅ 标记已读成功
✅ Tag 状态更新

---

【测试场景 7:发送消息(Tab内)】
1. 在站内信 Tab 点击「发送消息」按钮
2. 验证 NotifySendModal 弹窗(无预设模板)
3. 填写并提交

预期结果:
✅ 自由发送消息正常

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 模板分页 | GET | /admin-api/system/notify-template/page |
| 模板详情 | GET | /admin-api/system/notify-template/get?id=X |
| 创建模板 | POST | /admin-api/system/notify-template/create |
| 更新模板 | PUT | /admin-api/system/notify-template/update |
| 删除模板 | DELETE | /admin-api/system/notify-template/delete?id=X |
| 发送站内信 | POST | /admin-api/system/notify-template/send-notify |
| 消息分页 | GET | /admin-api/system/notify-message/page |
| 我的消息 | GET | /admin-api/system/notify-message/my-page |
| 消息详情 | GET | /admin-api/system/notify-message/get?id=X |
| 未读列表 | GET | /admin-api/system/notify-message/get-unread-list?size=X |
| 未读计数 | GET | /admin-api/system/notify-message/get-unread-count |
| 标记已读 | PUT | /admin-api/system/notify-message/update-read |
| 全部已读 | PUT | /admin-api/system/notify-message/update-all-read |

---

【问题诊断】
- Tab 切换不加载数据 → 检查 onMounted 是否只加载了模板数据
- 发送后消息不出现 → 检查 loadMessageData() 调用
- 标记已读不生效 → 检查 updateRead API 的 ids 和 userType 参数
```

#### 5.4.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【站内信管理】模块。

【模块信息】
- 服务: System(:8082)
- 页面路径: 左侧菜单「系统管理」→「站内信管理」
- 路由: /system/notify
- 权限前缀: system:notify-template: / system:notify-message:

【前端文件清单】
- 主页面: business-web/src/views/system/notify/index.vue(Tab 结构)
- 模板弹窗: business-web/src/views/system/notify/NotifyTemplateFormModal.vue
- 发送弹窗: business-web/src/views/system/notify/NotifySendModal.vue
- 详情弹窗: business-web/src/views/system/notify/NotifyMessageDetailModal.vue
- API 封装: business-web/src/api/system/notify.ts

【功能需求】
1. 两个 Tab:消息模板 + 站内信
2. 模板 CRUD + 发送站内信
3. 站内信列表 + 查看详情 + 标记已读
4. DictTag(system_notify_template_type) 展示模板类型

【交付物清单】
- [ ] 前端 index.vue + 三个弹窗组件
- [ ] 前端 API 封装 notify.ts
- [ ] 后端 Controller + Service + DO + Mapper
- [ ] 路由 + 菜单 + 权限码注册
- [ ] 通过 5.4.2 所有测试场景
```

---

### 5.5 SYS-05 文件管理

**页面路径**: 左侧菜单「系统管理」→「文件管理」  
**源码文件**: `src/views/system/file/index.vue`, `FilePreviewModal.vue`, `FileGridView.vue`, `FileStatsCard.vue`, `FileTreeDrawer.vue`, `FileDropOverlay.vue`  
**API 文件**: `src/api/system/file.ts`  
**权限标识**: `system:file:upload`, `system:file:delete`

#### 5.5.1 测试场景

#### 5.5.2 测试提示词

```
/browser 或 /open-gstack-browser
打开文件管理页面,对每种存储器配置分别执行完整的 CRUD 测试,并验证文件上传到对应存储。

【前置操作】
1. 登录系统,进入「系统管理」→「文件管理」
2. 等待页面加载完成
3. 确认系统中已配置多种存储类型(数据库/本地/FTP/SFTP/S3)

---

【存储器类型对照表】

| 存储器 | 存储值 | 上传后文件存储位置 |
|--------|--------|-------------------|
| 数据库 | 1 | 数据库 BLOB 字段 |
| 本地存储 | 10 | 本地文件系统指定目录 |
| FTP | 11 | FTP 服务器指定目录 |
| SFTP | 12 | SFTP 服务器指定目录 |
| S3 对象存储 | 20 | S3 兼容对象存储 Bucket |

---

【测试场景 1:存储统计卡片】
1. 验证页面顶部显示 FileStatsCard 存储统计区域
2. 检查卡片内容:配置名称、存储类型、文件数量、总大小
3. 验证多个存储配置的卡片并排显示
4. 确认每种存储器类型都有对应的统计卡片

预期结果:
✅ 统计卡片正确渲染
✅ 每种存储类型都有独立统计

---

【测试场景 2:文件列表视图】
1. 验证默认列表视图,检查列:文件名、路径、大小、类型、预览、URL、创建时间、操作
2. 验证文件大小格式化显示(如 KB/MB)
3. 验证图片类型显示缩略图(Image 组件 48x48)
4. 验证 PDF 显示「预览」链接,其他显示「下载」链接
5. 验证分页功能

预期结果:
✅ 列表视图列完整
✅ 图片缩略图/PDF预览/下载链接正确

---

【测试场景 3:网格视图切换】
1. 点击「网格视图」按钮(AppstoreOutlined 图标)
2. 验证切换到 FileGridView 组件
3. 验证网格卡片显示文件缩略图/图标
4. 点击「列表视图」按钮切回

预期结果:
✅ 视图切换正常
✅ 网格视图渲染正确

---

【测试场景 4:搜索功能】
1. 输入文件路径关键词搜索
2. 输入文件类型搜索
3. 重置条件

预期结果:
✅ 路径/类型过滤正确
✅ 重置清空

---

【测试场景 5:上传文件到数据库存储配置】
1. 确认系统中有「数据库」存储类型的主配置
2. 点击「上传文件」按钮
3. 选择一个文件上传(如 test-db.txt)
4. 验证上传进度条
5. 验证上传成功后列表刷新
6. 检查文件路径是否包含数据库存储标识

预期结果:
✅ 文件上传成功
✅ 文件存储在数据库中

---

【测试场景 6:上传文件到本地存储配置】
1. 切换或确认本地存储配置为主配置
2. 点击「上传文件」按钮
3. 选择一个文件上传(如 test-local.jpg)
4. 验证上传进度条
5. 验证上传成功后列表刷新
6. 检查文件 URL 是否为本地域名路径

预期结果:
✅ 本地存储配置上传成功
✅ 文件 URL 为本地存储域名

---

【测试场景 7:上传文件到 FTP 存储配置】
1. 确认系统中已配置 FTP 存储
2. 切换 FTP 配置为主配置
3. 点击「上传文件」按钮
4. 选择一个文件上传(如 test-ftp.pdf)
5. 验证上传进度条
6. 验证上传成功后列表刷新
7. 检查文件 URL 是否为 FTP 自定义域名

预期结果:
✅ FTP 存储配置上传成功
✅ 文件 URL 为 FTP 存储域名

---

【测试场景 8:上传文件到 SFTP 存储配置】
1. 确认系统中已配置 SFTP 存储
2. 切换 SFTP 配置为主配置
3. 点击「上传文件」按钮
4. 选择一个文件上传(如 test-sftp.docx)
5. 验证上传进度条
6. 验证上传成功后列表刷新
7. 检查文件 URL 是否为 SFTP 自定义域名

预期结果:
✅ SFTP 存储配置上传成功
✅ 文件 URL 为 SFTP 存储域名

---

【测试场景 9:上传文件到 S3 对象存储配置】
1. 确认系统中已配置 S3 存储
2. 切换 S3 配置为主配置
3. 点击「上传文件」按钮
4. 选择一个文件上传(如 test-s3.png)
5. 验证上传进度条
6. 验证上传成功后列表刷新
7. 检查文件 URL 是否为 S3/Bucket 路径

预期结果:
✅ S3 存储配置上传成功
✅ 文件 URL 为 S3 存储路径

---

【测试场景 10:拖拽上传(多类型)】
1. 确认某存储配置为主配置
2. 拖拽多个文件到页面区域
3. 验证上传触发
4. 验证批量上传成功

预期结果:
✅ 拖拽上传可用
✅ 批量文件上传成功

---

【测试场景 11:文件预览】
1. 对图片文件点击预览
2. 验证 FilePreviewModal 弹窗
3. 验证图片放大/缩小/旋转功能
4. 对 PDF 文件点击预览链接

预期结果:
✅ 图片预览弹窗正常
✅ PDF 预览链接可用

---

【测试场景 12:目录树】
1. 点击「目录」按钮(FolderOpenOutlined)
2. 验证 FileTreeDrawer 侧边抽屉弹出
3. 验证目录树结构

预期结果:
✅ 目录树抽屉正常

---

【测试场景 13:批量选择与删除】
1. 勾选多个文件(row-selection)
2. 点击「批量删除」按钮
3. 验证确认弹窗
4. 确认删除,验证列表刷新

预期结果:
✅ 批量选择正常
✅ 批量删除成功

---

【测试场景 14:文件移动】
1. 选择文件,执行移动操作
2. 指定目标目录
3. 验证移动成功

预期结果:
✅ 文件移动正常

---

【测试场景 15:批量下载】
1. 勾选多个文件
2. 点击批量下载
3. 验证 ZIP 文件下载

预期结果:
✅ 批量下载 ZIP 正常

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 文件分页 | GET | /admin-api/system/file/page |
| 文件详情 | GET | /admin-api/system/file/get?id=X |
| 后端上传 | POST | /admin-api/system/file/upload |
| 预签名URL | GET | /admin-api/system/file/presigned-url |
| 创建记录 | POST | /admin-api/system/file/create |
| 删除文件 | DELETE | /admin-api/system/file/delete?id=X |
| 批量删除 | DELETE | /admin-api/system/file/delete-list?ids=X |
| 目录树 | GET | /admin-api/system/file/directories |
| 存储统计 | GET | /admin-api/system/file-config/statistics |
| 批量下载 | POST | /admin-api/system/file/batch-download |
| 批量移动 | PUT | /admin-api/system/file/batch-move |
| 移动文件 | PUT | /admin-api/system/file/move |

---

【问题诊断】
- 统计卡片不显示 → 检查 getStatistics() 调用
- 图片缩略图加载失败 → 检查 url 拼接
- 上传进度不显示 → 检查 onUploadProgress 回调
- 网格视图异常 → 检查 FileGridView 组件渲染
- 存储配置切换不生效 → 检查后端默认存储配置设置
- 文件 URL 与存储类型不匹配 → 检查后端文件路径生成逻辑
```

#### 5.5.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【文件管理】模块。

【模块信息】
- 服务: System(:8082)
- 页面路径: 左侧菜单「系统管理」→「文件管理」
- 路由: /system/file
- 权限前缀: system:file:

【前端文件清单】
- 主页面: business-web/src/views/system/file/index.vue(718行)
- 预览弹窗: business-web/src/views/system/file/components/FilePreviewModal.vue
- 网格视图: business-web/src/views/system/file/components/FileGridView.vue
- 统计卡片: business-web/src/views/system/file/components/FileStatsCard.vue
- 目录抽屉: business-web/src/views/system/file/components/FileTreeDrawer.vue
- 拖拽覆盖: business-web/src/views/system/file/components/FileDropOverlay.vue
- composables: useFileStats.ts, useFileView.ts
- API 封装: business-web/src/api/system/file.ts

【功能需求】
1. 存储统计卡片(FileStatsCard)
2. 列表/网格双视图切换
3. 文件上传(后端上传 + 预签名直传)
4. 图片预览(FilePreviewModal)
5. 目录树抽屉(FileTreeDrawer)
6. 拖拽上传(FileDropOverlay)
7. 批量删除/移动/下载

【交付物清单】
- [ ] 前端 index.vue + 5个子组件 + 2个composables
- [ ] 前端 API 封装 file.ts
- [ ] 后端 Controller + Service + DO + Mapper
- [ ] 路由 + 菜单 + 权限码注册
- [ ] 通过 5.5.2 所有测试场景
```

---

### 5.6 SYS-06 文件配置管理

**页面路径**: 左侧菜单「系统管理」→「文件管理」→「文件配置」(或独立入口)  
**源码文件**: `src/views/system/file/FileConfigFormModal.vue`  
**API 文件**: `src/api/system/fileConfig.ts`  
**权限标识**: `system:file-config:create`, `system:file-config:update`, `system:file-config:delete`

#### 5.6.1 测试场景

#### 5.6.2 测试提示词

```
/browser 或 /open-gstack-browser
打开文件配置管理页面,对每种存储器类型(数据库/本地/FTP/SFTP/S3)分别执行完整的 CRUD 测试。

【前置操作】
1. 登录系统,进入文件配置管理页面
2. 等待页面加载完成

---

【存储器类型对照表】

| 存储器 | 存储值 | 必填配置参数 |
|--------|--------|--------------|
| 数据库 | 1 | 无 |
| 本地存储 | 10 | 基础路径(basePath)、自定义域名(domain) |
| FTP | 11 | 基础路径、主机地址、主机端口、用户名、密码、连接模式、自定义域名 |
| SFTP | 12 | 基础路径、主机地址、主机端口、用户名、密码、自定义域名 |
| S3 对象存储 | 20 | 节点地址(endpoint)、存储 Bucket、AccessKey、AccessSecret、PathStyle、公开访问 |

---

【测试场景 1:配置列表加载】
1. 验证文件配置列表正常加载
2. 检查列:名称、编码、存储器类型、备注、主配置、创建时间、操作
3. 验证存储器类型显示(数据库/本地存储/FTP/SFTP/S3 对象存储)
4. 验证主配置标识(master=true)

预期结果:
✅ 列表正常加载
✅ 存储器类型标签正确

---

【测试场景 2:搜索功能】
1. 按名称搜索
2. 按存储器类型筛选(分别测试数据库/本地/FTP/SFTP/S3)
3. 重置条件

预期结果:
✅ 搜索过滤正确
✅ 按存储器类型筛选正确

---

【测试场景 3:新增-数据库存储配置】
1. 点击「新增」按钮
2. 验证 FileConfigFormModal 弹窗
3. 选择存储器类型为「数据库」(值=1)
4. 验证数据库类型无需额外配置参数
5. 填写名称、编码、备注
6. 提交并验证

预期结果:
✅ 数据库类型无动态表单
✅ 新增成功

---

【测试场景 4:新增-本地存储配置】
1. 点击「新增」按钮
2. 选择存储器类型为「本地存储」(值=10)
3. 验证动态表单显示:基础路径、自定义域名
4. 填写:名称(如 test-local)、编码(如 local-001)、basePath(如 /home/uploads)、domain(如 https://static.example.com)
5. 提交并验证

预期结果:
✅ 动态表单正确显示
✅ 本地存储配置新增成功

---

【测试场景 5:新增-FTP存储配置】
1. 点击「新增」按钮
2. 选择存储器类型为「FTP」(值=11)
3. 验证动态表单显示:基础路径、主机地址、主机端口、用户名、密码、连接模式、自定义域名
4. 填写完整 FTP 配置参数
5. 提交并验证

预期结果:
✅ FTP 动态表单正确显示
✅ FTP 连接模式选项(主动模式/被动模式)
✅ FTP 配置新增成功

---

【测试场景 6:新增-SFTP存储配置】
1. 点击「新增」按钮
2. 选择存储器类型为「SFTP」(值=12)
3. 验证动态表单显示:基础路径、主机地址、主机端口、用户名、密码、自定义域名
4. 填写完整 SFTP 配置参数
5. 提交并验证

预期结果:
✅ SFTP 动态表单正确显示(无连接模式字段)
✅ SFTP 配置新增成功

---

【测试场景 7:新增-S3对象存储配置】
1. 点击「新增」按钮
2. 选择存储器类型为「S3 对象存储」(值=20)
3. 验证动态表单显示:节点地址、存储 Bucket、AccessKey、AccessSecret、PathStyle、公开访问、区域、自定义域名
4. 填写完整 S3 配置参数
5. 提交并验证

预期结果:
✅ S3 动态表单正确显示
✅ PathStyle 选项(启用/禁用)
✅ 公开访问选项(公开/私有)
✅ S3 配置新增成功

---

【测试场景 8:编辑-切换存储器类型测试】
1. 选择一个已存在的本地存储配置，点击「编辑」
2. 验证数据回填正确
3. 【关键】切换存储器类型为 FTP/SFTP/S3
4. 验证动态表单随存储器类型切换而变化
5. 填写新配置参数并提交

预期结果:
✅ 编辑模式存储器类型不可修改(is disabled)
✅ 动态表单正确切换

---

【测试场景 9:测试连接功能】
1. 对每种存储器类型的配置点击「测试」按钮
2. 验证调用 /admin-api/system/file-config/test?id=X
3. 验证测试结果显示(success/error)

预期结果:
✅ 数据库配置测试直接成功
✅ 其他存储类型测试连接正常返回结果

---

【测试场景 10:删除配置】
1. 选择一个测试配置，点击「删除」按钮
2. 确认删除
3. 验证列表刷新

预期结果:
✅ 删除成功
✅ 列表正确刷新

---

【测试场景 11:批量删除】
1. 勾选多个不同存储器类型的配置
2. 点击「批量删除」
3. 确认并验证

预期结果:
✅ 批量删除成功
✅ 列表正确刷新

---

【测试场景 12:导出/导入配置】
1. 点击「导出」，验证 JSON 文件下载
2. 检查导出 JSON 包含所有存储器类型配置
3. 点击「导入」，选择 JSON 文件上传
4. 验证导入结果(successCount/failCount)

预期结果:
✅ 导出 JSON 正常
✅ 导入 JSON 正常

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 配置分页 | GET | /admin-api/system/file-config/page |
| 配置详情 | GET | /admin-api/system/file-config/get?id=X |
| 创建配置 | POST | /admin-api/system/file-config/create |
| 更新配置 | PUT | /admin-api/system/file-config/update |
| 删除配置 | DELETE | /admin-api/system/file-config/delete?id=X |
| 批量删除 | DELETE | /admin-api/system/file-config/delete-list?ids=X |
| 测试连接 | GET | /admin-api/system/file-config/test?id=X |
| 导出配置 | GET | /admin-api/system/file-config/export |
| 导入配置 | POST | /admin-api/system/file-config/import |

---

【问题诊断】
- 动态表单不渲染 → 检查 FileConfigFormModal 中存储器类型联动逻辑
- 数据库类型显示配置参数 → 数据库存储无需 config，config 字段应为空对象
- 测试连接失败 → 检查后端文件存储服务配置和网络连通性
- 导入失败 → 检查 JSON 格式和 Content-Type
```

#### 5.6.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【文件配置管理】模块。

【模块信息】
- 服务: System(:8082)
- 页面路径: 左侧菜单「系统管理」→「文件管理」→「文件配置」
- 路由: /system/file-config
- 权限前缀: system:file-config:

【前端文件清单】
- 表单弹窗: business-web/src/views/system/file/FileConfigFormModal.vue(355行)
- API 封装: business-web/src/api/system/fileConfig.ts

【功能需求】
1. 配置列表 + 搜索(名称/存储器类型)
2. 新增/编辑:动态表单随存储器类型(DB/LOCAL/FTP/SFTP/S3)切换字段
3. 测试连接功能
4. 批量删除
5. 导出/导入 JSON 配置

【交付物清单】
- [ ] 前端列表页 + FileConfigFormModal.vue
- [ ] 前端 API 封装 fileConfig.ts
- [ ] 后端 Controller + Service + DO + Mapper
- [ ] 路由 + 菜单 + 权限码注册
- [ ] 通过 5.6.2 所有测试场景
```

---

### 5.7 SYS-07 邮件管理

**页面路径**: 左侧菜单「系统管理」→「邮件管理」  
**源码文件**: `src/views/system/mail/index.vue`(三 Tab 结构:邮箱账号/邮件模板/发送日志)  
**API 文件**: `src/api/system/mail.ts`  
**权限标识**: `system:mail-account:create`, `system:mail-account:update`, `system:mail-account:delete`, `system:mail-template:create`, `system:mail-template:update`, `system:mail-template:delete`

#### 5.7.1 测试场景

#### 5.7.2 测试提示词

```
/browser 或 /open-gstack-browser
打开邮件管理页面(三 Tab:邮箱账号/邮件模板/发送日志),执行完整测试。

【前置操作】
1. 登录系统,进入「系统管理」→「邮件管理」
2. 验证页面包含三个 Tab:「邮箱账号」「邮件模板」「发送日志」

---

【测试场景 1:邮箱账号列表】
1. 验证默认激活「邮箱账号」Tab
2. 检查列:邮箱地址、SMTP 服务器、端口、SSL(Tag)、创建时间、操作
3. 搜索:邮箱地址(Input)
4. 验证分页正常

预期结果:
✅ 账号列表6列完整
✅ SSL 状态 Tag 正确(启用=绿色/停用=默认色)

---

【测试场景 2:新增/编辑邮箱账号】
1. 点击「新增」按钮(v-has-permi=['system:mail-account:create'])
2. 验证弹窗表单:邮箱地址(必填)、用户名、密码(SMTP密码)、SMTP服务器(必填)、端口(必填,默认465)、启用SSL(Switch)、启用STARTTLS(Switch)
3. 填写:mail=test@example.com, host=smtp.example.com, port=465
4. 提交并验证列表刷新
5. 编辑已创建账号,验证回填

预期结果:
✅ 表单字段完整(含SSL/STARTTLS开关)
✅ CRUD 正常

---

【测试场景 3:邮件模板列表】
1. 切换到「邮件模板」Tab
2. 检查列:模板名称、模板编码、邮件标题、状态(Tag)、备注、创建时间、操作
3. 搜索:模板名称(Input)

预期结果:
✅ 模板列表7列完整
✅ 状态 Tag(启用=绿色/停用=红色)

---

【测试场景 4:新增/编辑邮件模板】
1. 点击「新增」按钮
2. 表单字段:模板名称(必填)、模板编码(必填)、关联账号(Select,来自账号列表)、发件人昵称、邮件标题、邮件内容(Textarea,{param}占位)、状态(Radio)
3. 填写并提交
4. 编辑验证回填

预期结果:
✅ 关联账号下拉来自 mailApi.account.page
✅ 模板 CRUD 正常

---

【测试场景 5:发送日志列表】
1. 切换到「发送日志」Tab
2. 检查列:发件人、收件人、邮件标题、状态(Tag)、发送时间、操作
3. 搜索:收件人(Input)
4. 点击「详情」查看 Modal.info 弹窗

预期结果:
✅ 日志列表6列完整
✅ 详情弹窗正常

---

【测试场景 6:Tab 切换数据加载】
1. 依次切换三个 Tab
2. 验证每个 Tab 首次激活时加载对应数据
3. 验证 handleTabChange 逻辑

预期结果:
✅ Tab 切换正确触发数据加载

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 账号分页 | GET | /admin-api/system/mail-account/page |
| 创建账号 | POST | /admin-api/system/mail-account/create |
| 更新账号 | PUT | /admin-api/system/mail-account/update |
| 删除账号 | DELETE | /admin-api/system/mail-account/delete?id=X |
| 模板分页 | GET | /admin-api/system/mail-template/page |
| 创建模板 | POST | /admin-api/system/mail-template/create |
| 更新模板 | PUT | /admin-api/system/mail-template/update |
| 删除模板 | DELETE | /admin-api/system/mail-template/delete?id=X |
| 日志分页 | GET | /admin-api/system/mail-log/page |

---

【问题诊断】
- Tab 切换不加载 → 检查 handleTabChange 分支
- 关联账号下拉为空 → 检查 accountOptions 赋值
- SSL Tag 不显示 → 检查 record.sslEnable 字段
```

#### 5.7.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【邮件管理】模块。

【模块信息】
- 服务: System(:8082)
- 页面路径: 左侧菜单「系统管理」→「邮件管理」
- 路由: /system/mail
- 权限前缀: system:mail-account: / system:mail-template:

【前端文件清单】
- 主页面: business-web/src/views/system/mail/index.vue(297行,三 Tab 内嵌表单+弹窗)
- API 封装: business-web/src/api/system/mail.ts

【功能需求】
1. 三个 Tab:邮箱账号 + 邮件模板 + 发送日志
2. 账号 CRUD:邮箱地址、用户名、密码、SMTP服务器、端口、SSL/STARTTLS开关
3. 模板 CRUD:模板名称、编码、关联账号(Select)、昵称、标题、内容(Textarea)、状态
4. 日志查看:发件人、收件人、标题、状态、时间、详情(Modal.info)
5. 权限控制: v-has-permi 控制按钮显隐

【交付物清单】
- [ ] 前端 index.vue(三 Tab 结构)
- [ ] 前端 API 封装 mail.ts
- [ ] 后端 Controller + Service + DO + Mapper
- [ ] 路由 + 菜单 + 权限码注册
- [ ] 通过 5.7.2 所有测试场景
```

---

### 5.8 SYS-08 短信管理

**页面路径**: 左侧菜单「系统管理」→「短信管理」  
**源码文件**: `src/views/system/sms/index.vue`(三 Tab:短信渠道/短信模板/发送日志)  
**API 文件**: `src/api/system/sms.ts`  
**权限标识**: `system:sms-channel:create`, `system:sms-channel:update`, `system:sms-channel:delete`, `system:sms-template:create`, `system:sms-template:update`, `system:sms-template:delete`

#### 5.8.1 测试场景

#### 5.8.2 测试提示词

```
/browser 或 /open-gstack-browser
打开短信管理页面(三 Tab:短信渠道/短信模板/发送日志),执行完整测试。

【前置操作】
1. 登录系统,进入「系统管理」→「短信管理」
2. 验证页面包含三个 Tab:「短信渠道」「短信模板」「发送日志」

---

【测试场景 1:短信渠道列表】
1. 验证默认激活「短信渠道」Tab
2. 检查列:签名、渠道编码、状态(Tag)、备注、创建时间、操作
3. 搜索:渠道编码(Input)
4. 验证状态 Tag(启用=绿色/停用=红色)

预期结果:
✅ 渠道列表6列完整

---

【测试场景 2:新增/编辑短信渠道】
1. 点击「新增」按钮
2. 表单字段:签名(必填)、渠道编码(必填)、API账号(必填)、API密钥、回调URL、备注(Textarea)、状态(Radio)
3. 填写:signature=【测试】, code=TEST_SMS, apiKey=test_key
4. 提交并验证
5. 编辑验证回填

预期结果:
✅ 渠道 CRUD 正常
✅ 表单验证生效

---

【测试场景 3:短信模板列表】
1. 切换到「短信模板」Tab
2. 检查列:模板名称、模板编码、类型(Tag)、内容、状态(Tag)、创建时间、操作
3. 搜索:模板名称(Input)
4. 验证类型 Tag(验证码=蓝色/通知=绿色)

预期结果:
✅ 模板列表7列完整

---

【测试场景 4:新增/编辑短信模板】
1. 点击「新增」按钮
2. 表单字段:模板名称(必填)、模板编码(必填)、模板类型(Radio:验证码/通知)、关联渠道(Select)、模板内容(Textarea,${param}占位,必填)、备注、状态
3. 填写并提交
4. 编辑验证回填

预期结果:
✅ 关联渠道下拉来自 smsApi.channel.page
✅ 模板 CRUD 正常

---

【测试场景 5:发送日志列表】
1. 切换到「发送日志」Tab
2. 检查列:手机号、渠道编码、模板编码、发送内容、状态(Tag)、响应结果、发送时间、操作
3. 搜索:手机号(Input)、发送状态(Select:成功/发送中/失败)
4. 点击「详情」查看 Modal.info 弹窗

预期结果:
✅ 日志列表8列完整
✅ 状态 Tag(成功=绿色/发送中=橙色/失败=红色)

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 渠道分页 | GET | /admin-api/system/sms-channel/page |
| 创建渠道 | POST | /admin-api/system/sms-channel/create |
| 更新渠道 | PUT | /admin-api/system/sms-channel/update |
| 删除渠道 | DELETE | /admin-api/system/sms-channel/delete?id=X |
| 模板分页 | GET | /admin-api/system/sms-template/page |
| 创建模板 | POST | /admin-api/system/sms-template/create |
| 更新模板 | PUT | /admin-api/system/sms-template/update |
| 删除模板 | DELETE | /admin-api/system/sms-template/delete?id=X |
| 日志分页 | GET | /admin-api/system/sms-log/page |

---

【问题诊断】
- 渠道列表不加载 → 检查 smsApi.channel.page() 调用
- 关联渠道下拉为空 → 检查 channelOptions 赋值
- 日志状态 Tag 颜色不对 → 检查 smsStatusMap 映射
```

#### 5.8.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【短信管理】模块。

【模块信息】
- 服务: System(:8082)
- 页面路径: 左侧菜单「系统管理」→「短信管理」
- 路由: /system/sms
- 权限前缀: system:sms-channel: / system:sms-template:

【前端文件清单】
- 主页面: business-web/src/views/system/sms/index.vue(311行,三 Tab 内嵌弹窗)
- API 封装: business-web/src/api/system/sms.ts

【功能需求】
1. 三个 Tab:短信渠道 + 短信模板 + 发送日志
2. 渠道 CRUD:签名、编码、API账号/密钥、回调URL、状态
3. 模板 CRUD:名称、编码、类型(验证码/通知)、关联渠道、内容(${param}占位)、状态
4. 日志查看:手机号、渠道、模板、内容、状态(成功/发送中/失败)、详情
5. 权限控制: v-has-permi

【交付物清单】
- [ ] 前端 index.vue(三 Tab 结构)
- [ ] 前端 API 封装 sms.ts
- [ ] 后端 Controller + Service + DO + Mapper
- [ ] 路由 + 菜单 + 权限码注册
- [ ] 通过 5.8.2 所有测试场景
```

---

### 5.9 SYS-09 操作日志

**页面路径**: 左侧菜单「系统管理」→「操作日志」  
**源码文件**: `src/views/system/operatelog/index.vue`  
**API 文件**: `src/api/system/operatelog.ts`  
**权限标识**: `system:operate-log:query`

#### 5.9.1 测试场景

#### 5.9.2 测试提示词

```
/browser 或 /open-gstack-browser
打开操作日志页面,执行查看与详情测试。

【前置操作】
1. 登录系统,进入「系统管理」→「操作日志」
2. 等待页面加载完成

---

【测试场景 1:日志列表加载】
1. 验证操作日志列表正常加载
2. 检查列:日志编号、操作人、操作模块、操作名称、操作内容、业务编号、操作IP、操作时间、操作
3. 验证分页功能(showSizeChanger, showQuickJumper, pageSizeOptions)

预期结果:
✅ 表格9列完整
✅ 分页功能完善

---

【测试场景 2:搜索功能】
1. 搜索字段:操作模块(Input)、操作名称(Input)、操作内容(Input)、业务编号(Input)
2. 点击「查询」验证过滤
3. 点击「重置」验证清空

预期结果:
✅ 四维搜索正确
✅ 重置清空所有条件

---

【测试场景 3:查看详情】
1. 点击「详情」按钮
2. 验证 Modal 弹窗(宽度720px)
3. 验证 Descriptions 组件展示:日志编号、操作人、操作模块、操作名称、业务编号、操作IP、请求方法、请求URL、操作内容、扩展信息、User-Agent、操作时间、链路追踪
4. 验证 :column=2 bordered size=small

预期结果:
✅ 详情弹窗 Descriptions 完整(12项)
✅ 布局两列带边框

---

【测试场景 4:分页切换】
1. 切换页码
2. 切换 pageSize(10/20/50)
3. 使用快速跳转

预期结果:
✅ 分页切换数据刷新

---

【测试场景 5:空数据状态】
1. 输入不存在的条件搜索
2. 验证显示「暂无数据」

预期结果:
✅ 空状态正确

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 日志分页 | GET | /admin-api/system/operate-log/page |
| 日志详情 | GET | /admin-api/system/operate-log/get?id=X |

---

【问题诊断】
- 列表不加载 → 检查 operateLogApi.page() 调用
- 详情弹窗空白 → 检查 currentRow 赋值和 Descriptions 绑定
- bizId 搜索无效 → 检查 Number() 转换
```

#### 5.9.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【操作日志】模块。

【模块信息】
- 服务: System(:8082)
- 页面路径: 左侧菜单「系统管理」→「操作日志」
- 路由: /system/operatelog
- 权限前缀: system:operate-log:

【前端文件清单】
- 主页面: business-web/src/views/system/operatelog/index.vue(204行)
- API 封装: business-web/src/api/system/operatelog.ts

【功能需求】
1. 日志列表(只读,无增删改)
2. 搜索:操作模块、操作名称、操作内容、业务编号
3. 详情弹窗:Modal + Descriptions(column=2, bordered)
4. 分页:showSizeChanger + showQuickJumper + pageSizeOptions

【交付物清单】
- [ ] 前端 index.vue
- [ ] 前端 API 封装 operatelog.ts
- [ ] 后端 Controller + Service + DO + Mapper
- [ ] 路由 + 菜单 + 权限码注册
- [ ] 通过 5.9.2 所有测试场景
```

---

### 5.10 SYS-10 我的站内信

**页面路径**: 左侧菜单「个人中心」→「我的站内信」(或顶部铃铛入口)  
**源码文件**: `src/views/system/notify/my/index.vue`  
**API 文件**: `src/api/system/notify.ts`(notifyMessageApi)  
**权限标识**: 登录用户即可访问

#### 5.10.1 测试场景

#### 5.10.2 测试提示词

```
/browser 或 /open-gstack-browser
打开我的站内信页面,执行消息查看与标记已读测试。

【前置操作】
1. 登录系统,进入「我的站内信」页面
2. 等待页面加载完成

---

【测试场景 1:消息列表加载】
1. 验证消息列表使用 List + ListItemMeta 组件渲染
2. 验证每项显示:头像(Avatar+Badge未读标记)、发送人名称、消息类型Tag、内容摘要(100字截断)、时间
3. 验证操作按钮:标为已读(仅未读)、查看详情
4. 验证右侧未读消息 Badge(count, overflow-count=99)

预期结果:
✅ List 渲染正常
✅ 未读 Badge 红点正确
✅ 未读消息字体加粗

---

【测试场景 2:筛选与搜索】
1. 在「阅读状态」下拉选择(未读/已读)
2. 验证列表过滤
3. 点击「重置」

预期结果:
✅ 阅读状态筛选正确

---

【测试场景 3:标记已读】
1. 对未读消息点击「标为已读」
2. 验证调用 notifyMessageApi.updateRead
3. 验证消息 Badge 消失
4. 验证未读计数减少

预期结果:
✅ 标记已读成功
✅ 未读计数同步更新

---

【测试场景 4:全部标为已读】
1. 当有未读消息时,验证「全部标为已读」按钮可见
2. 点击按钮
3. 验证调用 notifyMessageApi.updateAllRead
4. 验证所有消息变为已读
5. 验证未读计数归零

预期结果:
✅ 全部已读功能正常
✅ 未读计数归零

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 我的消息 | GET | /admin-api/system/notify-message/my-page |
| 未读计数 | GET | /admin-api/system/notify-message/get-unread-count |
| 标记已读 | PUT | /admin-api/system/notify-message/update-read |
| 全部已读 | PUT | /admin-api/system/notify-message/update-all-read |

---

【问题诊断】
- 未读计数不显示 → 检查 getUnreadCount() 调用
- 标记已读不刷新 → 检查 loadData + loadUnreadCount 调用
- 全部已读按钮不出现 → 检查 unreadCount > 0 条件
```

#### 5.10.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【我的站内信】模块。

【模块信息】
- 服务: System(:8082)
- 页面路径: 顶部铃铛入口 / 个人中心 →「我的站内信」
- 路由: /system/notify/my
- 权限: 登录用户即可访问

【前端文件清单】
- 主页面: business-web/src/views/system/notify/my/index.vue(309行)
- 详情弹窗: business-web/src/views/system/notify/NotifyMessageDetailModal.vue
- API 封装: business-web/src/api/system/notify.ts(notifyMessageApi)

【功能需求】
1. 消息列表(List组件):头像+Badge、发送人、类型Tag、内容摘要、时间
2. 筛选:阅读状态(Select:未读/已读)
3. 标记已读(单条/全部)
4. 未读计数(Badge)
5. 查看详情弹窗(NotifyMessageDetailModal)

【交付物清单】
- [ ] 前端 my/index.vue
- [ ] 前端复用 NotifyMessageDetailModal.vue
- [ ] 路由 + 菜单注册
- [ ] 通过 5.10.2 所有测试场景
```

---

### 5.11 SYS-11 站内信详情(消息模板发送后详情)

**页面路径**: 站内信管理/我的站内信 → 查看详情弹窗  
**源码文件**: `src/views/system/notify/NotifyMessageDetailModal.vue`  
**API 文件**: `src/api/system/notify.ts`(notifyMessageApi)

#### 5.11.1 测试场景

#### 5.11.2 测试提示词

```
/browser 或 /open-gstack-browser
打开站内信详情弹窗,验证详情展示。

---

【测试场景 1:详情弹窗展示】
1. 在站内信列表点击「查看」按钮
2. 验证 NotifyMessageDetailModal 弹窗弹出
3. 验证展示:模板编码、发送人昵称、消息内容(HTML渲染)、模板类型、已读状态、创建时间

预期结果:
✅ 详情弹窗字段完整

---

【测试场景 2:关闭弹窗自动标记已读】
1. 对未读消息打开详情弹窗
2. 关闭弹窗
3. 验证触发 @success 事件,调用标记已读

预期结果:
✅ 关闭弹窗后自动标记已读

---

【测试场景 3:HTML 内容渲染】
1. 验证消息内容中的 HTML 标签正确渲染
2. 检查 XSS 防护

预期结果:
✅ HTML 内容正确渲染
✅ 无 XSS 风险
```

#### 5.11.3 开发提示词

```
请基于以下信息,在 business-platform 仓库中实现【站内信详情弹窗】组件。

【模块信息】
- 组件: NotifyMessageDetailModal
- 复用位置: 站内信管理 Tab + 我的站内信

【前端文件清单】
- 详情弹窗: business-web/src/views/system/notify/NotifyMessageDetailModal.vue(88行)

【功能需求】
1. 弹窗展示消息详情:模板编码、发送人、内容(HTML)、类型、状态、时间
2. 关闭弹窗时触发 @success(id) 事件
3. HTML 内容安全渲染

【交付物清单】
- [ ] NotifyMessageDetailModal.vue
- [ ] 通过 5.11.2 所有测试场景
```

---

### 5.12 SYS-12 短信渠道管理(增强)

**页面路径**: 左侧菜单「系统管理」→「短信管理」→「短信渠道」Tab  
**源码文件**: `src/views/system/sms/index.vue`(channel Tab)  
**API 文件**: `src/api/system/sms.ts`(smsApi.channel)  
**权限标识**: `system:sms-channel:create`, `system:sms-channel:update`, `system:sms-channel:delete`

#### 5.12.1 测试场景

#### 5.12.2 测试提示词

```
/browser 或 /open-gstack-browser
针对短信渠道管理执行完整 CRUD 测试。

【前置操作】
1. 登录系统,进入短信管理,确认在「短信渠道」Tab

---

【测试场景 1:渠道列表】
1. 验证列表列:签名、渠道编码、状态(Tag)、备注、创建时间、操作
2. 搜索:渠道编码
3. 分页正常

预期结果:✅ 列表完整

---

【测试场景 2:新增渠道】
1. 点击「新增」
2. 表单:签名(必填)、渠道编码(必填)、API账号(必填)、API密钥、回调URL、备注(Textarea)、状态(Radio:启用/停用)
3. 填写并提交

预期结果:✅ 新增成功

---

【测试场景 3:编辑渠道】
1. 点击「编辑」,验证回填
2. 修改签名并提交

预期结果:✅ 编辑成功

---

【测试场景 4:渠道编码唯一性】
1. 新增渠道,编码设为已存在的编码
2. 验证后端返回错误

预期结果:✅ 唯一性校验生效

---

【测试场景 5:删除渠道】
1. 点击「删除」,Popconfirm 确认
2. 验证列表刷新

预期结果:✅ 删除成功

---

【测试场景 6:状态切换】
1. 新增渠道默认启用
2. 编辑切换为停用
3. 验证状态 Tag 变化

预期结果:✅ 状态切换正常

---

【测试场景 7:关联模板验证】
1. 删除有关联模板的渠道
2. 验证后端是否保护(外键约束)

预期结果:✅ 有模板关联时删除受保护

---

【测试场景 8:空数据状态】
1. 搜索不存在的编码
2. 验证空状态展示

预期结果:✅ 空状态正确

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 渠道分页 | GET | /admin-api/system/sms-channel/page |
| 创建渠道 | POST | /admin-api/system/sms-channel/create |
| 更新渠道 | PUT | /admin-api/system/sms-channel/update |
| 删除渠道 | DELETE | /admin-api/system/sms-channel/delete?id=X |
```

#### 5.12.3 开发提示词

```
请基于以下信息,实现【短信渠道管理(增强)】完整 CRUD。

【模块信息】
- 服务: System(:8082)
- 页面: sms/index.vue 的 channel Tab
- 权限: system:sms-channel:create/update/delete

【功能需求】
1. 完整 CRUD:签名、编码(唯一)、API账号/密钥、回调URL、备注、状态
2. 权限控制 v-has-permi
3. Popconfirm 删除确认

【交付物清单】
- [ ] sms/index.vue channel Tab
- [ ] API sms.ts channel 部分
- [ ] 后端 Controller + Service
- [ ] 通过 5.12.2 所有测试场景
```

---

### 5.13 SYS-13 短信模板管理

**页面路径**: 左侧菜单「系统管理」→「短信管理」→「短信模板」Tab  
**源码文件**: `src/views/system/sms/index.vue`(template Tab)  
**API 文件**: `src/api/system/sms.ts`(smsApi.template)  
**权限标识**: `system:sms-template:create`, `system:sms-template:update`, `system:sms-template:delete`

#### 5.13.1 测试场景

#### 5.13.2 测试提示词

```
/browser 或 /open-gstack-browser
针对短信模板管理执行完整 CRUD 测试。

【前置操作】
1. 登录系统,进入短信管理,切换到「短信模板」Tab

---

【测试场景 1:模板列表】
1. 验证列:模板名称、模板编码、类型(Tag:验证码=蓝/通知=绿)、内容、状态(Tag)、创建时间、操作
2. 搜索:模板名称

预期结果:✅ 列表7列完整

---

【测试场景 2:新增模板】
1. 点击「新增」
2. 表单:模板名称(必填)、模板编码(必填)、模板类型(Radio:验证码/通知)、关联渠道(Select)、模板内容(Textarea,${param}占位,必填)、备注、状态
3. 填写并提交

预期结果:✅ 新增成功

---

【测试场景 3:编辑模板】
1. 点击「编辑」,验证回填
2. 修改内容并提交

预期结果:✅ 编辑成功

---

【测试场景 4:关联渠道下拉】
1. 验证关联渠道下拉数据来自 smsApi.channel.page
2. 显示格式:签名 (编码)

预期结果:✅ 渠道下拉正确

---

【测试场景 5:模板类型切换】
1. 新增时切换模板类型(验证码/通知)
2. 验证 Radio 切换正常

预期结果:✅ 类型切换正常

---

【测试场景 6:删除模板】
1. 点击「删除」,Popconfirm 确认
2. 验证列表刷新

预期结果:✅ 删除成功

---

【测试场景 7:模板编码唯一性】
1. 新增编码重复的模板
2. 验证后端校验

预期结果:✅ 唯一性校验生效

---

【测试场景 8:状态管理】
1. 创建启用状态模板
2. 编辑切换为停用
3. 验证状态 Tag 变化

预期结果:✅ 状态管理正常

---

【测试场景 9:内容占位符】
1. 模板内容输入 ${code} 等占位符
2. 提交后验证保存正确

预期结果:✅ 占位符保存正确

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 模板分页 | GET | /admin-api/system/sms-template/page |
| 创建模板 | POST | /admin-api/system/sms-template/create |
| 更新模板 | PUT | /admin-api/system/sms-template/update |
| 删除模板 | DELETE | /admin-api/system/sms-template/delete?id=X |
```

#### 5.13.3 开发提示词

```
请基于以下信息,实现【短信模板管理】完整 CRUD。

【模块信息】
- 服务: System(:8082)
- 页面: sms/index.vue 的 template Tab
- 权限: system:sms-template:create/update/delete

【功能需求】
1. 完整 CRUD:名称、编码(唯一)、类型(验证码/通知)、关联渠道(Select)、内容(${param}占位)、备注、状态
2. 渠道下拉: smsApi.channel.page 映射为 {value: id, label: 签名 (编码)}
3. 权限控制 v-has-permi

【交付物清单】
- [ ] sms/index.vue template Tab
- [ ] API sms.ts template 部分
- [ ] 后端 Controller + Service
- [ ] 通过 5.13.2 所有测试场景
```

---

### 5.14 SYS-14 短信日志管理

**页面路径**: 左侧菜单「系统管理」→「短信管理」→「发送日志」Tab  
**源码文件**: `src/views/system/sms/index.vue`(log Tab)  
**API 文件**: `src/api/system/sms.ts`(smsApi.log)  
**权限标识**: `system:sms-log:query`

#### 5.14.1 测试场景

#### 5.14.2 测试提示词

```
/browser 或 /open-gstack-browser
针对短信发送日志执行查看测试。

【前置操作】
1. 登录系统,进入短信管理,切换到「发送日志」Tab

---

【测试场景 1:日志列表】
1. 验证列:手机号、渠道编码、模板编码、发送内容、状态(Tag)、响应结果、发送时间、操作
2. 搜索:手机号(Input)、发送状态(Select:成功/发送中/失败)

预期结果:✅ 列表8列完整

---

【测试场景 2:状态筛选】
1. 选择「成功」筛选
2. 选择「发送中」筛选
3. 选择「失败」筛选
4. 重置

预期结果:✅ 状态筛选正确

---

【测试场景 3:查看详情】
1. 点击「详情」按钮
2. 验证 Modal.info 弹窗展示:手机号、渠道、模板、状态、结果、时间

预期结果:✅ 详情弹窗正常

---

【测试场景 4:状态 Tag 颜色】
1. 成功=绿色
2. 发送中=橙色
3. 失败=红色

预期结果:✅ Tag 颜色正确

---

【测试场景 5:分页】
1. 验证分页功能
2. 切换 pageSize

预期结果:✅ 分页正常

---

【测试场景 6:空数据】
1. 搜索不存在的手机号
2. 验证空状态

预期结果:✅ 空状态正确

---

【测试场景 7:手机号搜索】
1. 输入完整手机号搜索
2. 输入部分手机号搜索

预期结果:✅ 搜索正确

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 日志分页 | GET | /admin-api/system/sms-log/page |
```

#### 5.14.3 开发提示词

```
请基于以下信息,实现【短信日志管理】查看功能。

【模块信息】
- 服务: System(:8082)
- 页面: sms/index.vue 的 log Tab
- 权限: system:sms-log:query

【功能需求】
1. 日志列表(只读):手机号、渠道编码、模板编码、内容、状态(成功/发送中/失败)、响应结果、时间
2. 搜索:手机号 + 发送状态(Select)
3. 详情:Modal.info 弹窗

【交付物清单】
- [ ] sms/index.vue log Tab
- [ ] API sms.ts log 部分
- [ ] 后端 Controller + Service
- [ ] 通过 5.14.2 所有测试场景
```

---

### 5.15 SYS-15 邮件账号管理

**页面路径**: 左侧菜单「系统管理」→「邮件管理」→「邮箱账号」Tab  
**源码文件**: `src/views/system/mail/index.vue`(account Tab)  
**API 文件**: `src/api/system/mail.ts`(mailApi.account)  
**权限标识**: `system:mail-account:create`, `system:mail-account:update`, `system:mail-account:delete`

#### 5.15.1 测试场景

#### 5.15.2 测试提示词

```
/browser 或 /open-gstack-browser
针对邮箱账号管理执行完整 CRUD 测试。

【前置操作】
1. 登录系统,进入邮件管理,确认在「邮箱账号」Tab

---

【测试场景 1:账号列表】
1. 验证列:邮箱地址、SMTP 服务器、端口、SSL(Tag:启用=绿/停用=默认)、创建时间、操作
2. 搜索:邮箱地址
3. 分页正常

预期结果:✅ 列表6列完整

---

【测试场景 2:新增账号】
1. 点击「新增」(v-has-permi=['system:mail-account:create'])
2. 表单:邮箱地址(必填)、用户名、密码(SMTP密码,InputPassword)、SMTP服务器(必填)、端口(必填,InputNumber,默认465)、启用SSL(Switch,默认开)、启用STARTTLS(Switch,默认关)
3. 填写并提交

预期结果:✅ 新增成功

---

【测试场景 3:编辑账号】
1. 点击「编辑」,验证回填
2. 修改 SMTP 服务器并提交

预期结果:✅ 编辑成功

---

【测试场景 4:删除账号】
1. 点击「删除」,Popconfirm 确认
2. 验证列表刷新

预期结果:✅ 删除成功

---

【测试场景 5:SSL/STARTTLS 开关】
1. 新增时 SSL 默认开启
2. 切换 STARTTLS 开关
3. 验证 Switch 状态正确

预期结果:✅ 开关切换正常

---

【测试场景 6:端口默认值】
1. 新增时端口默认 465
2. 修改端口为 587
3. 验证 InputNumber 范围(1-65535)

预期结果:✅ 端口默认值和范围正确

---

【测试场景 7:邮箱格式验证】
1. 输入非法邮箱格式
2. 验证提示

预期结果:✅ 邮箱格式校验

---

【测试场景 8:密码字段安全】
1. 验证密码字段使用 InputPassword(星号显示)
2. 编辑时密码字段不回填明文

预期结果:✅ 密码字段安全

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 账号分页 | GET | /admin-api/system/mail-account/page |
| 创建账号 | POST | /admin-api/system/mail-account/create |
| 更新账号 | PUT | /admin-api/system/mail-account/update |
| 删除账号 | DELETE | /admin-api/system/mail-account/delete?id=X |
```

#### 5.15.3 开发提示词

```
请实现【邮箱账号管理】完整 CRUD。

【模块信息】
- 服务: System(:8082)
- 页面: mail/index.vue 的 account Tab
- 权限: system:mail-account:create/update/delete

【功能需求】
1. CRUD:邮箱地址(必填)、用户名、密码(SMTP)、SMTP服务器(必填)、端口(必填,默认465)、SSL/STARTTLS(Switch)
2. 权限控制 v-has-permi
3. 密码使用 InputPassword

【交付物清单】
- [ ] mail/index.vue account Tab
- [ ] API mail.ts account 部分
- [ ] 后端 Controller + Service
- [ ] 通过 5.15.2 所有测试场景
```

---

### 5.16 SYS-16 邮件模板管理

**页面路径**: 左侧菜单「系统管理」→「邮件管理」→「邮件模板」Tab  
**源码文件**: `src/views/system/mail/index.vue`(template Tab)  
**API 文件**: `src/api/system/mail.ts`(mailApi.template)  
**权限标识**: `system:mail-template:create`, `system:mail-template:update`, `system:mail-template:delete`

#### 5.16.1 测试场景

#### 5.16.2 测试提示词

```
/browser 或 /open-gstack-browser
针对邮件模板管理执行完整 CRUD 测试。

【前置操作】
1. 登录系统,进入邮件管理,切换到「邮件模板」Tab

---

【测试场景 1:模板列表】
1. 验证列:模板名称、模板编码、邮件标题、状态(Tag)、备注、创建时间、操作
2. 搜索:模板名称

预期结果:✅ 列表7列完整

---

【测试场景 2:新增模板】
1. 点击「新增」
2. 表单:模板名称(必填)、模板编码(必填)、关联账号(Select)、发件人昵称、邮件标题、邮件内容(Textarea,{param}占位)、状态(Radio:启用/停用)
3. 填写并提交

预期结果:✅ 新增成功

---

【测试场景 3:编辑模板】
1. 点击「编辑」,验证回填
2. 修改内容并提交

预期结果:✅ 编辑成功

---

【测试场景 4:关联账号下拉】
1. 验证关联账号下拉数据来自 mailApi.account.page
2. 显示格式:邮箱地址

预期结果:✅ 账号下拉正确

---

【测试场景 5:删除模板】
1. 点击「删除」,Popconfirm 确认
2. 验证列表刷新

预期结果:✅ 删除成功

---

【测试场景 6:模板编码唯一性】
1. 新增编码重复的模板
2. 验证后端校验

预期结果:✅ 唯一性校验

---

【测试场景 7:状态管理】
1. 创建启用状态模板
2. 编辑切换为停用

预期结果:✅ 状态切换正常

---

【测试场景 8:内容占位符】
1. 模板内容输入 {name} 等占位符
2. 提交后验证保存正确

预期结果:✅ 占位符保存正确

---

【测试场景 9:发件人昵称】
1. 填写发件人昵称(如:「业务平台」)
2. 验证保存和回填

预期结果:✅ 昵称字段正常

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 模板分页 | GET | /admin-api/system/mail-template/page |
| 创建模板 | POST | /admin-api/system/mail-template/create |
| 更新模板 | PUT | /admin-api/system/mail-template/update |
| 删除模板 | DELETE | /admin-api/system/mail-template/delete?id=X |
```

#### 5.16.3 开发提示词

```
请实现【邮件模板管理】完整 CRUD。

【模块信息】
- 服务: System(:8082)
- 页面: mail/index.vue 的 template Tab
- 权限: system:mail-template:create/update/delete

【功能需求】
1. CRUD:名称(必填)、编码(必填)、关联账号(Select)、昵称、标题、内容(Textarea,{param}占位)、状态
2. 账号下拉: mailApi.account.page 映射为 {value: id, label: mail}
3. 权限控制 v-has-permi

【交付物清单】
- [ ] mail/index.vue template Tab
- [ ] API mail.ts template 部分
- [ ] 后端 Controller + Service
- [ ] 通过 5.16.2 所有测试场景
```

---

### 5.17 SYS-17 邮件日志管理

**页面路径**: 左侧菜单「系统管理」→「邮件管理」→「发送日志」Tab  
**源码文件**: `src/views/system/mail/index.vue`(log Tab)  
**API 文件**: `src/api/system/mail.ts`(mailApi.log)  
**权限标识**: `system:mail-log:query`

#### 5.17.1 测试场景

#### 5.17.2 测试提示词

```
/browser 或 /open-gstack-browser
针对邮件发送日志执行查看测试。

【前置操作】
1. 登录系统,进入邮件管理,切换到「发送日志」Tab

---

【测试场景 1:日志列表】
1. 验证列:发件人、收件人、邮件标题、状态(Tag:成功=绿/失败=红)、发送时间、操作
2. 搜索:收件人(Input)

预期结果:✅ 列表6列完整

---

【测试场景 2:状态 Tag】
1. 成功状态:绿色 Tag,文案「成功」
2. 失败状态:红色 Tag,文案「失败」

预期结果:✅ Tag 颜色正确

---

【测试场景 3:查看详情】
1. 点击「详情」
2. 验证 Modal.info 弹窗展示:发件人、收件人、标题、状态、结果、时间

预期结果:✅ 详情弹窗正常

---

【测试场景 4:分页】
1. 验证分页功能
2. 切换 pageSize

预期结果:✅ 分页正常

---

【测试场景 5:收件人搜索】
1. 输入收件人邮箱搜索
2. 重置

预期结果:✅ 搜索正确

---

【测试场景 6:空数据】
1. 搜索不存在的收件人
2. 验证空状态

预期结果:✅ 空状态正确

---

【测试场景 7:日志详情内容】
1. 查看详情中的结果字段
2. 成功时显示「无」,失败时显示错误信息

预期结果:✅ 结果字段正确

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 日志分页 | GET | /admin-api/system/mail-log/page |
```

#### 5.17.3 开发提示词

```
请实现【邮件日志管理】查看功能。

【模块信息】
- 服务: System(:8082)
- 页面: mail/index.vue 的 log Tab
- 权限: system:mail-log:query

【功能需求】
1. 日志列表(只读):发件人、收件人、标题、状态(成功/失败)、时间、详情
2. 搜索:收件人
3. 详情:Modal.info 弹窗

【交付物清单】
- [ ] mail/index.vue log Tab
- [ ] API mail.ts log 部分
- [ ] 后端 Controller + Service
- [ ] 通过 5.17.2 所有测试场景
```

---

### 5.18 SYS-18 API 访问日志

**页面路径**: 左侧菜单「系统管理」→「API 访问日志」  
**源码文件**: `src/views/system/apiAccessLog/index.vue`, `ApiAccessLogDetailModal.vue`  
**API 文件**: `src/api/system/apiAccessLog.ts`  
**权限标识**: `system:api-access-log:query`, `system:api-access-log:export`

#### 5.18.1 测试场景

#### 5.18.2 测试提示词

```
/browser 或 /open-gstack-browser
打开 API 访问日志页面,执行查看与导出测试。

【前置操作】
1. 登录系统,进入「系统管理」→「API 访问日志」

---

【测试场景 1:日志列表加载】
1. 验证列表列:日志编号、用户编号、用户类型(Tag)、应用名、请求方法、请求地址、请求时间、执行时长(ms)、结果(Tag:成功=绿/失败=红)、操作
2. 验证分页(showSizeChanger, showQuickJumper, pageSizeOptions)

预期结果:✅ 表格10列完整
✅ 执行时长显示「XX ms」
✅ 结果码 Tag(0=成功绿色,其他=失败红色)

---

【测试场景 2:搜索功能】
1. 搜索字段:用户编号、用户类型(Select:系统用户/管理员)、应用名、执行时长(>= ms)、结果码、请求地址、请求时间(RangePicker+show-time)
2. 点击「查询」验证过滤
3. 点击「重置」验证清空

预期结果:✅ 七维搜索正确

---

【测试场景 3:导出 Excel】
1. 点击「导出」按钮
2. 验证 message.loading 显示
3. 验证文件下载(文件名:API访问日志_YYYYMMDD_HHmmss.xls)
4. 验证 message.success

预期结果:✅ 导出功能正常

---

【测试场景 4:查看详情】
1. 点击「详情」按钮
2. 验证 ApiAccessLogDetailModal 弹窗(v-model:visible)
3. 验证详情字段完整

预期结果:✅ 详情弹窗正常

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 日志分页 | GET | /admin-api/system/api-access-log/page |
| 日志详情 | GET | /admin-api/system/api-access-log/get?id=X |
| 导出 Excel | GET | /admin-api/system/api-access-log/export-excel |
```

#### 5.18.3 开发提示词

```
请实现【API 访问日志】查看+导出功能。

【模块信息】
- 服务: System(:8082)
- 页面: apiAccessLog/index.vue + ApiAccessLogDetailModal.vue
- 权限: system:api-access-log:query/export

【功能需求】
1. 日志列表(只读):10列
2. 七维搜索:用户编号、用户类型、应用名、执行时长、结果码、请求地址、请求时间(RangePicker)
3. 导出 Excel
4. 详情弹窗(ApiAccessLogDetailModal)

【交付物清单】
- [ ] index.vue + ApiAccessLogDetailModal.vue
- [ ] API apiAccessLog.ts
- [ ] 后端 Controller + Service
- [ ] 通过 5.18.2 所有测试场景
```

---

### 5.19 SYS-19 API 错误日志

**页面路径**: 左侧菜单「系统管理」→「API 错误日志」  
**源码文件**: `src/views/system/apiErrorLog/index.vue`, `ApiErrorLogDetailModal.vue`  
**API 文件**: `src/api/system/apiErrorLog.ts`  
**权限标识**: `system:api-error-log:query`, `system:api-error-log:update-status`, `system:api-error-log:export`

#### 5.19.1 测试场景

#### 5.19.2 测试提示词

```
/browser 或 /open-gstack-browser
打开 API 错误日志页面,执行查看与处理状态流转测试。

【前置操作】
1. 登录系统,进入「系统管理」→「API 错误日志」

---

【测试场景 1:日志列表加载】
1. 验证列:日志编号、用户编号、应用名、请求方法、请求地址、异常类、异常时间、处理状态(Tag)、操作
2. 处理状态:待处理(蓝)=0,已处理(绿)=1,已忽略(默认)=2

预期结果:✅ 表格9列完整
✅ 处理状态 Tag 颜色正确

---

【测试场景 2:搜索功能】
1. 搜索字段:用户编号、应用名、异常类、请求地址、处理状态(Select:待处理/已处理/已忽略)、异常时间(RangePicker+show-time)
2. 查询/重置

预期结果:✅ 六维搜索正确

---

【测试场景 3:处理状态流转】
1. 对「待处理」状态记录点击「已处理」按钮
2. 验证 Modal.confirm 确认弹窗(文案:「确认标记为「已处理」？」、「标记后无法回退为待处理。」)
3. 确认后状态更新为「已处理」
4. 验证「已处理」记录不再显示「已处理」「已忽略」按钮(v-if="record.processStatus === 0")

预期结果:✅ 状态流转正确
✅ 已处理/已忽略记录无操作按钮

---

【测试场景 4:标记已忽略】
1. 对「待处理」记录点击「已忽略」
2. 确认并验证状态变为「已忽略」

预期结果:✅ 已忽略功能正常

---

【测试场景 5:查看详情】
1. 点击「详情」
2. 验证 ApiErrorLogDetailModal 弹窗
3. 验证展示异常堆栈信息(exceptionStackTrace)

预期结果:✅ 详情含异常堆栈

---

【测试场景 6:导出 Excel】
1. 点击「导出」
2. 验证文件下载

预期结果:✅ 导出正常

---

【测试场景 7:状态不可回退】
1. 验证已处理/已忽略的记录无法回退为待处理
2. 确认操作按钮仅对 processStatus===0 显示

预期结果:✅ 状态不可回退

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 日志分页 | GET | /admin-api/system/api-error-log/page |
| 日志详情 | GET | /admin-api/system/api-error-log/get?id=X |
| 更新状态 | PUT | /admin-api/system/api-error-log/update-status |
| 导出 Excel | GET | /admin-api/system/api-error-log/export-excel |
```

#### 5.19.3 开发提示词

```
请实现【API 错误日志】查看+处理状态流转+导出。

【模块信息】
- 服务: System(:8082)
- 页面: apiErrorLog/index.vue + ApiErrorLogDetailModal.vue
- 权限: system:api-error-log:query/update-status/export

【功能需求】
1. 日志列表(9列),含处理状态流转
2. 状态:待处理(0)→已处理(1)/已忽略(2),不可回退
3. 操作按钮仅对 processStatus===0 显示
4. 详情弹窗含异常堆栈
5. 导出 Excel

【交付物清单】
- [ ] index.vue + ApiErrorLogDetailModal.vue
- [ ] API apiErrorLog.ts
- [ ] 后端 Controller + Service
- [ ] 通过 5.19.2 所有测试场景
```

---

### 5.20 SYS-20 Redis 缓存监控

**页面路径**: 左侧菜单「系统管理」→「Redis 监控」  
**源码文件**: `src/views/system/redis/index.vue`  
**API 文件**: `src/api/system/redis.ts`  
**权限标识**: `system:redis:get-monitor-info`

#### 5.20.1 测试场景

#### 5.20.2 测试提示词

```
/browser 或 /open-gstack-browser
打开 Redis 缓存监控页面,执行仪表盘验证。

【前置操作】
1. 登录系统,进入「系统管理」→「Redis 监控」

---

【测试场景 1:基本信息展示】
1. 验证 Descriptions 组件(column=3, bordered)展示:
   - Redis 版本、运行模式、端口、客户端数、运行时间(秒)
   - 使用内存、使用CPU、内存配置峰值、AOF 开启、RDB最后成功、Key总数、网络入口流量
2. 验证 loading 状态

预期结果:✅ 12 项信息完整展示

---

【测试场景 2:命令统计图表】
1. 验证左侧饼图(SimpleChart + ECharts)渲染
2. 图表类型: pie(南丁格尔玫瑰图 roseType=radius)
3. 展示前 20 个命令的调用次数

预期结果:✅ 命令统计饼图正常渲染

---

【测试场景 3:内存使用率仪表】
1. 验证右侧仪表盘(gauge)渲染
2. 计算内存使用率: used_memory / total_system_memory * 100
3. 验证百分比显示

预期结果:✅ 内存使用率仪表盘正常

---

【测试场景 4:数据加载失败处理】
1. 停止 Redis 服务
2. 刷新页面
3. 验证 message.error 提示

预期结果:✅ 失败提示正常

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 监控信息 | GET | /admin-api/system/redis/get-monitor-info |
```

#### 5.20.3 开发提示词

```
请实现【Redis 缓存监控】仪表盘。

【模块信息】
- 服务: System(:8082)
- 页面: redis/index.vue
- 权限: system:redis:get-monitor-info

【功能需求】
1. Descriptions(column=3)展示 Redis INFO 信息(12项)
2. 命令统计饼图(ECharts pie, roseType=radius, Top 20)
3. 内存使用率仪表盘(ECharts gauge)
4. 依赖 SimpleChart 组件(@/components/SimpleChart)

【交付物清单】
- [ ] redis/index.vue
- [ ] API redis.ts
- [ ] 后端 Controller + Service
- [ ] 通过 5.20.2 所有测试场景
```

---

### 5.21 SYS-21 数据源配置

**页面路径**: 左侧菜单「系统管理」→「数据源配置」  
**源码文件**: `src/views/system/dataSourceConfig/index.vue`, `DataSourceConfigFormModal.vue`  
**API 文件**: `src/api/system/dataSourceConfig.ts`  
**权限标识**: `system:data-source-config:create`, `system:data-source-config:update`, `system:data-source-config:delete`

#### 5.21.1 测试场景

#### 5.21.2 测试提示词

```
/browser 或 /open-gstack-browser
打开数据源配置页面,执行 CRUD + 批量删除测试。

【前置操作】
1. 登录系统,进入「系统管理」→「数据源配置」

---

【测试场景 1:列表加载】
1. 验证列表列:编号、名称、类型(主数据源=橙色Tag/-)、连接地址(ellipsis+Tooltip)、用户名、创建时间、操作
2. 验证无分页(pagination=false)
3. 验证主数据源 primary=true 行显示「主数据源」橙色 Tag

预期结果:✅ 列表7列完整
✅ 主数据源标识正确

---

【测试场景 2:新增数据源】
1. 点击「新增数据源」按钮
2. 验证 DataSourceConfigFormModal 弹窗(v-model:visible)
3. 表单字段:名称(必填)、连接地址(必填)、用户名(必填)、密码(必填)
4. 填写并提交

预期结果:✅ 新增成功

---

【测试场景 3:编辑数据源】
1. 点击「编辑」按钮
2. 验证主数据源(primary=true)的编辑按钮 disabled
3. 非主数据源编辑验证回填
4. 修改并提交

预期结果:✅ 编辑成功
✅ 主数据源编辑禁用

---

【测试场景 4:删除数据源】
1. 点击「删除」按钮
2. 验证 Modal.confirm 确认弹窗(标题:确认删除数据源「xxx」？)
3. 验证主数据源删除按钮 disabled
4. 确认删除,验证列表刷新

预期结果:✅ 删除成功
✅ 主数据源删除禁用

---

【测试场景 5:批量删除】
1. 勾选多个非主数据源(row-selection,主数据源 checkbox disabled)
2. 点击「批量删除（N）」按钮
3. 验证 Modal.confirm(文案:「主数据源不可删除,已自动过滤」)
4. 确认删除,验证列表刷新

预期结果:✅ 批量删除成功
✅ 主数据源自动过滤

---

【测试场景 6:刷新列表】
1. 点击「刷新」按钮
2. 验证列表重新加载

预期结果:✅ 刷新正常

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 列表(无分页) | GET | /admin-api/system/data-source-config/list |
| 详情 | GET | /admin-api/system/data-source-config/get?id=X |
| 创建 | POST | /admin-api/system/data-source-config/create |
| 更新 | PUT | /admin-api/system/data-source-config/update |
| 删除 | DELETE | /admin-api/system/data-source-config/delete?id=X |
| 批量删除 | DELETE | /admin-api/system/data-source-config/delete-list?ids=X |
```

#### 5.21.3 开发提示词

```
请实现【数据源配置】CRUD + 批量删除。

【模块信息】
- 服务: System(:8082)
- 页面: dataSourceConfig/index.vue + DataSourceConfigFormModal.vue
- 权限: system:data-source-config:create/update/delete

【功能需求】
1. 列表(无分页):编号、名称、类型(主/从)、连接地址、用户名、创建时间、操作
2. 主数据源保护:编辑/删除/checkbox disabled
3. 批量删除(主数据源自动过滤)
4. 弹窗表单:名称、连接地址、用户名、密码

【交付物清单】
- [ ] index.vue + DataSourceConfigFormModal.vue
- [ ] API dataSourceConfig.ts
- [ ] 后端 Controller + Service
- [ ] 通过 5.21.2 所有测试场景
```

---

### 5.22 SYS-22 Druid SQL 监控

**页面路径**: 左侧菜单「系统管理」→「Druid SQL 监控」  
**源码文件**: `src/views/system/druid/index.vue`  
**API 文件**: `src/api/system/iframe.ts`  
**权限标识**: `system:druid:view`

#### 5.22.1 测试场景

#### 5.22.2 测试提示词

```
/browser 或 /open-gstack-browser
打开 Druid SQL 监控页面,验证 iframe 加载。

【测试场景 1:iframe 加载】
1. 进入「系统管理」→「Druid SQL 监控」
2. 验证 IframePage 组件渲染
3. 验证 iframe src 来自 /admin-api/system/druid/proxy-url 或 fallback-path /druid/index.html
4. 验证 iframe 内容加载(Druid 监控面板)

预期结果:
✅ iframe 正常加载
✅ Druid 监控面板可见

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 代理URL | GET | /admin-api/system/druid/proxy-url |
```

#### 5.22.3 开发提示词

```
请实现【Druid SQL 监控】iframe 页面。

【模块信息】
- 页面: druid/index.vue(16行)
- 组件: IframePage(@/components/IframePage)
- fallback: /druid/index.html

【功能需求】
1. 使用 IframePage 组件,传入 api-name="druid" 和 fallback-path
2. 后端返回代理 URL,前端拼接 origin 加载 iframe

【交付物清单】
- [ ] druid/index.vue
- [ ] IframePage 组件
- [ ] 通过 5.22.2 测试场景
```

---

### 5.23 SYS-23 服务节点监控

**页面路径**: 左侧菜单「系统管理」→「服务节点监控」  
**源码文件**: `src/views/system/server/index.vue`  
**API 文件**: `src/api/system/iframe.ts`  
**权限标识**: `system:server:view`

#### 5.23.1 测试场景

#### 5.23.2 测试提示词

```
/browser 或 /open-gstack-browser
打开服务节点监控页面,验证 iframe 加载。

【测试场景 1:iframe 加载】
1. 进入「系统管理」→「服务节点监控」
2. 验证 IframePage 组件渲染
3. 验证 fallback-path /admin/applications
4. 验证 Spring Boot Admin 面板加载

预期结果:
✅ iframe 正常加载
✅ Spring Boot Admin 面板可见

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 代理URL | GET | /admin-api/system/server/proxy-url |
```

#### 5.23.3 开发提示词

```
请实现【服务节点监控】iframe 页面。

【模块信息】
- 页面: server/index.vue(16行)
- 组件: IframePage
- fallback: /admin/applications

【功能需求】
1. 使用 IframePage 组件,传入 api-name="server"
2. 后端返回代理 URL,前端加载 Spring Boot Admin 面板

【交付物清单】
- [ ] server/index.vue
- [ ] 通过 5.23.2 测试场景
```

---

### 5.24 SYS-24 Swagger/API 文档

**页面路径**: 左侧菜单「系统管理」→「Swagger/API 文档」  
**源码文件**: `src/views/system/swagger/index.vue`  
**API 文件**: `src/api/system/iframe.ts`  
**权限标识**: `system:swagger:view`

#### 5.24.1 测试场景

#### 5.24.2 测试提示词

```
/browser 或 /open-gstack-browser
打开 Swagger/API 文档页面,验证 iframe 加载。

【测试场景 1:iframe 加载】
1. 进入「系统管理」→「Swagger/API 文档」
2. 验证 IframePage 组件渲染
3. 验证 fallback-path /doc.html
4. 验证 Knife4j/Swagger UI 面板加载

预期结果:
✅ iframe 正常加载
✅ API 文档面板可见

---

【API 端点对照】

| 操作 | API 方法 | 路径 |
|------|---------|------|
| 代理URL | GET | /admin-api/system/swagger/proxy-url |
```

#### 5.24.3 开发提示词

```
请实现【Swagger/API 文档】iframe 页面。

【模块信息】
- 页面: swagger/index.vue(16行)
- 组件: IframePage
- fallback: /doc.html

【功能需求】
1. 使用 IframePage 组件,传入 api-name="swagger"
2. 后端返回代理 URL,前端加载 Knife4j 面板

【交付物清单】
- [ ] swagger/index.vue
- [ ] 通过 5.24.2 测试场景
```

---

## 6.测试结果报告模板

```markdown
# System 服务测试报告

**测试日期**: YYYY-MM-DD  
**测试人员**: AI Agent (gstack /qa)  
**测试环境**: http://localhost:5173  
**System 服务**: http://localhost:8082

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
| SYS-01 参数配置 | 6 | 6 | 0 | 0 | 100% |
| SYS-02 字典管理 | 10 | 10 | 0 | 0 | 100% |
| SYS-03 通知公告 | 6 | 6 | 0 | 0 | 100% |
| SYS-04 站内信管理 | 7 | 7 | 0 | 0 | 100% |
| SYS-05 文件管理 | 11 | 10 | 1 | 0 | 90.9% |
| SYS-06 文件配置管理 | 8 | 8 | 0 | 0 | 100% |
| SYS-07 邮件管理 | 6 | 6 | 0 | 0 | 100% |
| SYS-08 短信管理 | 5 | 5 | 0 | 0 | 100% |
| SYS-09 操作日志 | 5 | 5 | 0 | 0 | 100% |
| **合计** | **N** | **X** | **Y** | **Z** | **X/N%** |

## 失败用例详情

### SYS-05 文件管理 — 测试场景 8:文件预览

- **现象**: 点击预览按钮后弹窗未显示
- **控制台错误**: `Component not found: FilePreviewModal`
- **截图**: [.gstack/qa-reports/screenshots/issue-003.png](.gstack/qa-reports/screenshots/issue-003.png)
- **定位**: `file/index.vue` 未正确导入 FilePreviewModal 组件
- **修复建议**: 在 components 中注册 FilePreviewModal
- **修复文件**: `src/views/system/file/index.vue`
- **状态**: 已修复

## 建议

1. **高优先级**:修复文件预览功能,影响用户体验
2. **中优先级**:增强邮件/短信管理的功能完整性
3. **低优先级**:引入 Infra 模块(API 日志、Redis 监控等)
4. **测试补充**:建议增加文件上传的边界测试(超大文件、非法类型)
```

---

## 7.模块开发对照与补全清单

本章节对照 example-ui(参考项目)与 business-platform(当前项目),梳理 System 模块的实现状态与补全建议。

### System 模块补全对照表

| 模块 | example-ui 路径 | business-platform 路径 | 前端状态 | 后端状态 | 补全建议 |
|------|-----------------|----------------------|---------|---------|--------|
| 参数配置 | system/config/index.vue | system/config/index.vue | ✅ 完整 | ✅ | 对齐:无明显差距 |
| 字典管理 | system/dict/index.vue + dict/data/ | system/dict/index.vue | ✅ 完整(子组件化) | ✅ | 已完成:DictTypeFormModal + DictDataFormModal 子组件 |
| 通知公告 | system/notice/index.vue | system/notice/index.vue | ✅ 完整(富文本) | ✅ | 已完成:NoticeFormModal 使用 wangeditor 富文本编辑器 |
| 站内信(消息模板) | system/notify/template/index.vue | system/notify/index.vue | ✅ 完整(Tab结构) | ✅ | 已完成:Tab 切换(模板/站内信) + NotifyTemplateFormModal |
| 站内信(消息列表) | system/notify/message/index.vue | system/notify/index.vue | ✅ 完整(详情弹窗) | ✅ | 已完成:NotifyMessageDetailModal + 标记已读 |
| 站内信(我的消息) | system/notify/my/index.vue | system/notify/my/index.vue | ✅ **已创建** | ✅ | 已完成:我的消息独立页面 + 铃铛入口 |
| 操作日志 | system/operatelog/index.vue | system/operatelog/index.vue | ✅ 完整(含详情弹窗) | ✅ | 已完成:详情弹窗集成在列表页内 |
| 登录日志 | system/loginlog/index.vue | uaa/loginLog/index.vue | ✅ 完整(含详情弹窗) | ✅ | 已完成:LoginLogDetailModal.vue |
| 邮件账号 | system/mail/account/index.vue | system/mail/index.vue | ✅ 完整(Tab结构) | ✅ | 当前已是完整 Tab 结构(账号/模板/日志) |
| 邮件模板 | system/mail/template/index.vue | system/mail/index.vue | ✅ 完整(Tab结构) | ✅ | 当前已是完整 Tab 结构 |
| 邮件日志 | system/mail/log/index.vue | system/mail/index.vue | ✅ 完整(Tab结构) | ✅ | 当前已是完整 Tab 结构 |
| 短信渠道 | system/sms/channel/index.vue | system/sms/index.vue | ✅ 完整(Tab结构) | ✅ | 当前已是完整 Tab 结构(渠道/模板/日志) |
| 短信模板 | system/sms/template/index.vue | system/sms/index.vue | ✅ 完整(Tab结构) | ✅ | 当前已是完整 Tab 结构 |
| 短信日志 | system/sms/log/index.vue | system/sms/index.vue | ✅ 完整(Tab结构) | ✅ | 当前已是完整 Tab 结构 |
| 文件管理 | infra/file/index.vue | system/file/index.vue | ✅ 完整(列表+网格+预览+拖拽上传) | ✅ | 当前文件管理较 example-ui 更完善 |
| 文件配置 | infra/file/FileForm.vue | system/file/FileConfigFormModal.vue | ✅ 完整(测试连接+主配置+复制) | ✅ | 当前文件配置较 example-ui 更完善 |
| API 访问日志 | infra/apiAccessLog/index.vue | system/apiAccessLog/index.vue | ✅ 完整(含导出) | ✅ | 已完成:ApiAccessLogDetailModal + 导出 |
| API 错误日志 | infra/apiErrorLog/index.vue | system/apiErrorLog/index.vue | ✅ 完整(含状态流转) | ✅ | 已完成:处理状态流转 + 导出 |
| Redis 监控 | infra/redis/index.vue | system/redis/index.vue | ✅ 完整(仪表盘+图表) | ✅ | 已完成:ECharts 饼图 + 仪表盘 |
| 数据源配置 | infra/dataSourceConfig/index.vue | system/dataSourceConfig/index.vue | ✅ 完整(批量删除) | ✅ | 已完成:主数据源保护 + 批量删除 |
| Druid 监控 | infra/druid/index.vue | system/druid/index.vue | ✅ 完整(iframe) | ✅ | IframePage + fallback |
| 服务监控 | infra/server/index.vue | system/server/index.vue | ✅ 完整(iframe) | ✅ | IframePage + fallback |
| Swagger | infra/swagger/index.vue | system/swagger/index.vue | ✅ 完整(iframe) | ✅ | IframePage + fallback |

## 8.文档版本

| 版本 | 日期 | 修改内容 | 作者 |
|------|------|---------|------|
| 1.0.0 | 2026-06-14 | 从 system-uaa-test.md 拆分,仅保留 System 服务测试模块 | AI Agent |
| 2.0.0 | 2026-06-17 | 完整补全 SYS-01 ~ SYS-24 所有模块的测试提示词与开发提示词 | AI Agent |

| 项目 | 值 |
|------|------|
| 适用服务 | System(:8082) |
| 前端入口 | http://localhost:5173 |
| 默认账号 | admin / admin123 |
| 默认租户 ID | 1 |
| 测试 Skill | `/qa`, `/qa-only`, `/open-gstack-browser`, MCP Playwright |

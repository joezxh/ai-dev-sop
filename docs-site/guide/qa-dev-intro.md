# /qa-to-dev 提示词转换方案

> 将 `/qa` 测试场景自动转换为开发提示词的完整方案

---

## 一、问题场景

```
原始场景：使用 /qa 执行 scene-template.md 中的测试场景
痛点：测试场景对应的功能可能尚未开发
期望：自动将测试场景转换为完整的开发任务
```

---

## 二、转换核心逻辑

### 2.1 测试验证点 → 开发任务映射表

| 测试验证点 | 前端开发任务 | 后端开发任务 |
|-----------|-------------|-------------|
| 页面加载 + 表格列显示 | Vue 页面组件 + columns 定义 | - |
| 分页功能 | 分页组件 + 请求参数 | Controller 分页接口 |
| 搜索筛选 | 搜索表单 + 过滤逻辑 | WHERE 条件查询 |
| 新增按钮 + 弹窗 | FormModal + v-model | POST 接口 |
| 编辑按钮 + 数据回填 | watch + 表单回填 | GET 详情接口 |
| 删除按钮 + 确认 | confirm 弹窗 | DELETE 接口 |
| 唯一性校验 | - | 数据库唯一索引 + Service 校验 |
| 字典标签 | DictTag 组件 | 字典数据初始化 |
| 权限控制 | v-has-permi 指令 | 权限码注册 |

### 2.2 文件清单生成模板

```
【前端文件清单】
- 主页面: {module}/index.vue
- 表单弹窗: {module}/XxxFormModal.vue
- API 封装: {module}/index.ts

【后端文件清单】
- Controller: {package}/controller/XxxController.java
- Service: {package}/service/XxxService.java
- DTO: {package}/dto/XxxRequest.java
- DO: {package}/dal/dataobject/XxxDO.java
- Mapper: {package}/dal/mapper/XxxMapper.java
```

---

## 三、完整转换提示词

### 3.1 主提示词（QA 发现未开发时执行）

```
## /qa-to-dev

当 `/qa` 执行测试发现功能尚未开发时，将测试场景自动转换为完整的开发任务。

### 输入信息

- 测试场景文件: @mediation-web/docs/scene/scene-template.md
- 指定模块: {MODULE_ID} (如 SYS-01)
- 指定场景: {SCENARIO_SECTION} (如 5.1.2)

### 转换流程

**Step 1: 分析测试场景**
读取测试场景中所有验证点:
1. 列表显示字段 → 确定表格 columns
2. 搜索条件 → 确定查询参数
3. 表单字段 → 确定 DTO 字段
4. 操作按钮 → 确定权限码
5. API 调用 → 确定后端接口

**Step 2: 生成开发提示词**
为每个测试场景生成对应的开发任务:

```
【模块信息】
- 服务: {SERVICE_NAME}(:{PORT})
- 页面路径: {PAGE_PATH}
- 路由: {ROUTE}
- 权限前缀: {PERMISSION_PREFIX}

【前端文件清单】
- 主页面: {module}/index.vue
- 表单弹窗: {module}/XxxFormModal.vue
- API 封装: {module}/index.ts

【后端文件清单】
- Controller: {package}/controller/XxxController.java
- Service: {package}/service/XxxService.java
- DTO: {package}/dto/XxxRequest.java
- DO: {package}/dal/dataobject/XxxDO.java
- Mapper: {package}/dal/mapper/XxxMapper.java

【API 端点】
| 操作 | 方法 | 路径 |
|------|------|------|
| 列表分页 | GET | /admin-api/{service}/{module}/page |
| 详情 | GET | /admin-api/{service}/{module}/get |
| 创建 | POST | /admin-api/{service}/{module}/create |
| 更新 | PUT | /admin-api/{service}/{module}/update |
| 删除 | DELETE | /admin-api/{service}/{module}/delete |

【功能需求】
1. {需求1}
2. {需求2}
...

【UI 规范】
- UI 库: ant-design-vue
- 表单组件: {form_fields}
- 字典类型: {dict_types}
- 业务组件: DictSelect / DictTag / DictSwitch / FormModal

【参考实现】
请参考以下已实现模块:
- {reference_module_1}
- {reference_module_2}

【交付物清单】
- [ ] 前端主页面 .vue
- [ ] 前端表单弹窗 .vue
- [ ] 前端 API 封装 .ts(含 TypeScript 类型)
- [ ] 后端 Controller(含 Swagger @Operation 注解)
- [ ] 后端 Service 接口 + 实现
- [ ] 后端 DTO / DO / Mapper
- [ ] 路由注册
- [ ] 菜单注册
- [ ] 权限码注册
- [ ] 字典数据初始化 SQL(如有)
- [ ] 通过 {SCENARIO_SECTION} 所有测试场景
```

**Step 3: 生成验证清单**
确保开发完成后满足测试场景的所有验证点:

```
【测试场景验证清单】
| 测试场景 | 验证点 | 开发交付物 |
|---------|-------|-----------|
| 场景1:列表加载 | 表格8列完整 | index.vue columns 定义 |
| 场景1:列表加载 | 内置DictTag正确 | DictTag 组件 + 字典数据 |
| 场景1:列表加载 | 分页正常 | 分页组件 + 后端分页逻辑 |
| 场景2:搜索功能 | 参数名称搜索 | 搜索表单 + WHERE 条件 |
| 场景2:搜索功能 | 重置清空条件 | 重置按钮逻辑 |
| 场景3:新增参数 | 表单字段完整 | FormModal.vue 字段定义 |
| 场景3:新增参数 | POST 提交 | POST 接口 |
| 场景4:编辑参数 | 数据回填 | GET 详情 + watch |
| 场景5:唯一性校验 | 后端校验 | Service 唯一性检查 |
| 场景6:删除参数 | 确认弹窗 | confirm + DELETE 接口 |
```

### 输出文件

生成文件: `@mediation-web/docs/scene/{module}-dev-prompt.md`

文件结构:
```
# {MODULE_NAME} 开发提示词

> 由 {SCENARIO_SECTION} 测试场景自动转换生成
> 生成时间: {TIMESTAMP}

## 1. 模块概述

## 2. 开发任务清单

## 3. 完整开发提示词

## 4. 验证清单

## 5. 参考实现
```

---

## 四、针对 SYS-01 参数配置的完整转换示例

### 4.1 测试场景分析

**原始测试场景 (scene-template.md 246-365):**

| 测试场景 | 验证点 |
|---------|-------|
| 场景1:列表加载 | 8列显示、分页、DictTag、分页 |
| 场景2:搜索功能 | 参数名称搜索、参数键名搜索、重置 |
| 场景3:新增参数 | 弹窗、表单5字段、POST提交 |
| 场景4:编辑参数 | 数据回填、PUT提交 |
| 场景5:唯一性校验 | 后端唯一性检查 |
| 场景6:删除参数 | confirm弹窗、DELETE提交 |

### 4.2 转换后的完整开发提示词

```
# SYS-01 参数配置 — 完整开发提示词

> 由 scene-template.md §5.1.2 测试场景自动转换
> 生成时间: 2026-06-29

---

## 模块信息

| 属性 | 值 |
|------|-----|
| 服务 | System(:8082) |
| 页面路径 | 系统管理 → 参数配置 |
| 路由 | /system/config |
| 权限前缀 | system:config |
| 权限码 | system:config:create, system:config:update, system:config:delete |

---

## 前端文件清单

```
mediation-web/src/views/system/config/
├── index.vue                    # 主页面
├── ConfigFormModal.vue          # 表单弹窗
└── index.ts                     # API 封装

mediation-web/src/api/system/
└── config.ts                    # API 接口
```

---

## 后端文件清单

```
mediation-module-system-server/src/main/java/com/tianque/system/
├── controller/
│   └── ConfigController.java    # 控制器
├── service/
│   ├── ConfigService.java       # 服务接口
│   └── impl/
│       └── ConfigServiceImpl.java
├── dto/
│   ├── ConfigPageRequest.java   # 分页请求
│   ├── ConfigCreateRequest.java # 创建请求
│   └── ConfigUpdateRequest.java # 更新请求
├── dal/
│   ├── dataobject/
│   │   └── ConfigDO.java
│   └── mapper/
│       └── ConfigMapper.java
```

---

## API 端点

| 操作 | 方法 | 路径 | 权限码 |
|------|------|------|--------|
| 列表分页 | GET | /admin-api/system/config/page | system:config:query |
| 参数详情 | GET | /admin-api/system/config/get | system:config:query |
| 创建参数 | POST | /admin-api/system/config/create | system:config:create |
| 更新参数 | PUT | /admin-api/system/config/update | system:config:update |
| 删除参数 | DELETE | /admin-api/system/config/delete | system:config:delete |

---

## 功能需求

### 1. 列表展示
- 字段: 参数名称、参数键名、参数键值、内置、参数类型、备注、创建时间、操作
- 内置列使用 DictTag 标签(是=绿色/否=灰色)
- 分页: 默认每页20条

### 2. 搜索筛选
- 参数名称: 模糊搜索
- 参数键名: 模糊搜索
- 重置: 清空所有搜索条件

### 3. 新增参数
- 弹窗标题: 新增参数
- 表单字段:
  - 参数名称(必填)
  - 参数键名(必填,唯一)
  - 参数键值(必填)
  - 参数类型(单选:系统内置/自定义)
  - 备注(可选)
- 提交: POST /admin-api/system/config/create
- 成功后: 关闭弹窗,刷新列表,显示成功提示

### 4. 编辑参数
- 数据回填: 调用 GET /admin-api/system/config/get?id=X
- 表单字段: 同新增
- 提交: PUT /admin-api/system/config/update
- 成功后: 关闭弹窗,刷新列表

### 5. 唯一性校验
- 后端: 参数键名唯一性检查
- 冲突: 返回错误码,弹窗不关闭

### 6. 删除参数
- 确认弹窗: 确定要删除该参数吗?
- 提交: DELETE /admin-api/system/config/delete?id=X
- 成功后: 刷新列表,显示删除成功

---

## 前端开发要求

### index.vue

```vue
<template>
  <!-- 搜索表单 -->
  <a-form layout="inline">
    <a-form-item label="参数名称">
      <a-input v-model:value="searchParams.configName" placeholder="请输入参数名称" />
    </a-form-item>
    <a-form-item label="参数键名">
      <a-input v-model:value="searchParams.configKey" placeholder="请输入参数键名" />
    </a-form-item>
    <a-form-item>
      <a-space>
        <a-button type="primary" @click="handleSearch">搜索</a-button>
        <a-button @click="handleReset">重置</a-button>
      </a-space>
    </a-form-item>
  </a-form>

  <!-- 操作按钮 -->
  <div class="table-operator">
    <a-button type="primary" v-has-permi="['system:config:create']" @click="handleAdd">
      新增
    </a-button>
  </div>

  <!-- 数据表格 -->
  <a-table
    :columns="columns"
    :data-source="dataSource"
    :pagination="pagination"
    :loading="loading"
    @change="handleTableChange"
  >
    <!-- 内置列 -->
    <template #bodyCell="{ column, record }">
      <template v-if="column.key === 'configType'">
        <dict-tag :value="record.configType" dict-type="sys_config_type" />
      </template>
    </template>
  </a-table>

  <!-- 表单弹窗 -->
  <config-form-modal
    v-model:open="formModalVisible"
    :record="currentRecord"
    @success="handleFormSuccess"
  />
</template>

<script setup lang="ts">
// columns 定义
const columns = [
  { title: '参数名称', dataIndex: 'configName' },
  { title: '参数键名', dataIndex: 'configKey' },
  { title: '参数键值', dataIndex: 'configValue' },
  { title: '内置', dataIndex: 'builtin', customRender: ({ text }) => text ? '是' : '否' },
  { title: '参数类型', key: 'configType' },
  { title: '备注', dataIndex: 'remark' },
  { title: '创建时间', dataIndex: 'createTime' },
  { title: '操作', key: 'action', fixed: 'right', width: 150 },
]
</script>
```

### ConfigFormModal.vue

```vue
<template>
  <a-modal
    v-model:open="open"
    :title="isEdit ? '编辑参数' : '新增参数'"
    @ok="handleSubmit"
  >
    <a-form
      ref="formRef"
      :model="formState"
      :rules="rules"
    >
      <a-form-item label="参数名称" name="configName">
        <a-input v-model:value="formState.configName" />
      </a-form-item>
      <a-form-item label="参数键名" name="configKey">
        <a-input v-model:value="formState.configKey" :disabled="isEdit" />
      </a-form-item>
      <a-form-item label="参数键值" name="configValue">
        <a-input v-model:value="formState.configValue" />
      </a-form-item>
      <a-form-item label="参数类型" name="configType">
        <dict-select v-model:value="formState.configType" dict-type="sys_config_type" />
      </a-form-item>
      <a-form-item label="备注" name="remark">
        <a-textarea v-model:value="formState.remark" />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script setup lang="ts">
// 监听 record prop 变化,回填表单
watch(() => props.record, (val) => {
  if (val) {
    formState.value = { ...val }
  } else {
    formState.value = { configType: 'custom' }
  }
}, { immediate: true })
</script>
```

---

## 后端开发要求

### ConfigDO.java

```java
@TableName("sys_config")
@Data
public class ConfigDO {

    @TableId
    private Long id;

    /** 参数名称 */
    private String configName;

    /** 参数键名 */
    private String configKey;

    /** 参数键值 */
    private String configValue;

    /** 内置(Y/N) */
    private Boolean builtin;

    /** 参数类型: system/自定义 */
    private String configType;

    /** 备注 */
    private String remark;

    /** 创建时间 */
    private LocalDateTime createTime;
}
```

### ConfigServiceImpl.java (唯一性校验)

```java
@Override
public void validateConfigKeyUnique(String configKey, Long excludeId) {
    ConfigDO existing = configMapper.selectByConfigKey(configKey);
    if (existing != null && !existing.getId().equals(excludeId)) {
        throw new BizException("参数键名已存在");
    }
}

@Override
public Long createConfig(ConfigCreateRequest request) {
    // 唯一性校验
    validateConfigKeyUnique(request.getConfigKey(), null);
    // ...
}

@Override
public void updateConfig(ConfigUpdateRequest request) {
    // 唯一性校验(排除自身)
    validateConfigKeyUnique(request.getConfigKey(), request.getId());
    // ...
}
```

### 数据库表结构

```sql
CREATE TABLE `sys_config` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `config_name` varchar(100) NOT NULL COMMENT '参数名称',
  `config_key` varchar(100) NOT NULL COMMENT '参数键名',
  `config_value` varchar(500) NOT NULL COMMENT '参数键值',
  `builtin` char(1) NOT NULL DEFAULT 'N' COMMENT '内置(Y/N)',
  `config_type` varchar(20) NOT NULL DEFAULT 'custom' COMMENT '参数类型: system/custom',
  `remark` varchar(500) DEFAULT NULL COMMENT '备注',
  `create_time` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_config_key` (`config_key`) COMMENT '参数键名唯一'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='参数配置表';
```

---

## 字典数据初始化

```sql
-- 参数类型字典
INSERT INTO `sys_dict_type` (`dict_name`, `dict_type`, `status`) VALUES
('参数类型', 'sys_config_type', '0');

INSERT INTO `sys_dict_data` (`dict_sort`, `dict_label`, `dict_value`, `dict_type`, `css_class`, `list_class`, `remark`) VALUES
(1, '系统内置', 'system', 'sys_config_type', NULL, 'danger', '系统内置参数'),
(2, '自定义', 'custom', 'sys_config_type', NULL, 'default', '用户自定义参数');
```

---

## 菜单权限配置

```sql
-- 菜单
INSERT INTO `sys_menu` (`menu_name`, `parent_id`, `order_num`, `path`, `component`, `menu_type`, `perms`) VALUES
('参数配置', 100, 1, 'config', 'system/config/index', 'C', 'system:config:query');

-- 按钮权限
INSERT INTO `sys_menu` (`menu_name`, `parent_id`, `order_num`, `path`, `component`, `menu_type`, `perms`) VALUES
('参数新增', 1001, 1, '', NULL, 'F', 'system:config:create'),
('参数修改', 1001, 2, '', NULL, 'F', 'system:config:update'),
('参数删除', 1001, 3, '', NULL, 'F', 'system:config:delete');
```

---

## 交付物清单

- [x] 前端主页面 index.vue
- [x] 前端表单弹窗 ConfigFormModal.vue
- [x] 前端 API 封装 config.ts
- [x] 后端 Controller ConfigController.java
- [x] 后端 Service ConfigService.java + Impl
- [x] 后端 DTO ConfigCreateRequest.java + ConfigUpdateRequest.java + ConfigPageRequest.java
- [x] 后端 DO ConfigDO.java
- [x] 后端 Mapper ConfigMapper.java
- [x] 数据库表 sys_config
- [x] 字典数据 sys_config_type
- [x] 菜单权限配置

---

## 测试场景验证清单

| 测试场景 | 验证点 | 交付物 | 状态 |
|---------|-------|--------|-----|
| 场景1:列表加载 | 表格8列完整 | index.vue columns | ✅ |
| 场景1:列表加载 | 内置DictTag | DictTag组件 + 字典 | ✅ |
| 场景1:列表加载 | 分页正常 | 分页组件 | ✅ |
| 场景2:搜索功能 | 参数名称搜索 | 模糊查询 | ✅ |
| 场景2:搜索功能 | 参数键名搜索 | 模糊查询 | ✅ |
| 场景2:搜索功能 | 重置清空 | resetFields | ✅ |
| 场景3:新增参数 | 弹窗表单 | ConfigFormModal | ✅ |
| 场景3:新增参数 | 5个表单字段 | formState | ✅ |
| 场景3:新增参数 | POST提交 | POST接口 | ✅ |
| 场景4:编辑参数 | 数据回填 | watch(record) | ✅ |
| 场景4:编辑参数 | PUT提交 | PUT接口 | ✅ |
| 场景5:唯一性校验 | 后端校验 | validateConfigKeyUnique | ✅ |
| 场景6:删除参数 | 确认弹窗 | confirm | ✅ |
| 场景6:删除参数 | DELETE提交 | DELETE接口 | ✅ |

---

## 参考实现

- `mediation-web/src/views/system/dict/index.vue` - 字典管理(适合作为列表参考)
- `mediation-web/src/views/system/dict/DictTypeFormModal.vue` - 字典类型表单(适合作为FormModal参考)
- `mediation-web/src/api/system/dict.ts` - API封装参考
```

---

## 五、自动化执行流程

### 5.1 触发流程

```
用户执行 /qa 测试
    ↓
测试场景执行
    ↓
发现功能未开发(404/空数据/组件不存在)
    ↓
自动触发 /qa-to-dev
    ↓
读取测试场景文件
    ↓
解析测试验证点
    ↓
生成开发提示词
    ↓
输出到 docs/scene/{module}-dev-prompt.md
    ↓
执行开发任务
    ↓
开发完成后重新执行 /qa
```

### 5.2 快速执行命令

```bash
# 方式1: 使用转换后的提示词开发
@docs/scene/SYS-01-dev-prompt.md

# 方式2: 直接生成提示词
/qa-to-dev --module SYS-01 --scenario "5.1.2"

# 方式3: 批量转换所有未开发模块
/qa-to-dev --batch --file docs/scene/scene-template.md
```

---

## 六、模板复用

将以下内容保存为模板文件:

`docs/scene/qa-to-dev-template.md`

```markdown
# {MODULE_NAME} 开发提示词

> 由 {SCENARIO_SECTION} 测试场景自动转换
> 生成时间: {TIMESTAMP}

## 模块信息

## 前端文件清单

## 后端文件清单

## API 端点

## 功能需求

## 交付物清单

## 测试场景验证清单
```

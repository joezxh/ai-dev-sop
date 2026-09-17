# Mem0 Dashboard 用户管理增强 + API Key 关联合并

- 日期：2026-09-16
- 状态：已确认（用户批准）
- 范围：`mem0/server`（后端）与 `mem0/server/dashboard`（Next.js 前端）
- 分支：`feat/mem0-llm-provider-i18n`

## 1. 背景与目标

现有 `mem0/server/dashboard` 中：

- `users/page.tsx`：管理员视角的用户列表（name/email/role/created_at），仅支持**创建**与**删除**，无编辑入口；与 API Key 完全分离。
- `api-keys/page.tsx`：**自作用**（`require_auth`）界面，只展示/管理**当前登录用户本人**的密钥（`GET/POST/DELETE /api-keys` 均按 `created_by == 当前用户` 过滤）。

目标：

1. 在用户管理界面提供**编辑**能力，可修改用户信息。
2. 新增用户字段 `disp_name`（显示名称）。
3. 将 API Key 管理**合并进用户页面**：管理员可在用户详情卡片中查看/创建/吊销**任意用户**的密钥，实现"用户 ↔ 密钥"关联管理。

## 2. 关键决策（已与用户确认）

| 决策点 | 结论 |
| --- | --- |
| Key 作用域 | 管理员可管理任意用户的密钥（新增按用户的管理接口）。 |
| UI 布局 | 每个用户一张**详情卡片**，下半部分为其 API Key 列表与操作。 |
| 编辑字段 | name / email / role / department / disp_name + 可选密码重置。 |
| 独立 api-keys 页面 | **保留**：供非管理员用户自我管理本人密钥（非管理员看不到 Users 页面）。 |
| 卡片密钥加载 | 默认卡片挂载即调用 `GET /users/{id}/api-keys`（N+1，用户量级小时可接受）。 |

## 3. 后端实现（`mem0/server`）

### 3.1 数据模型与迁移

- `models.py`：`User` 新增 `disp_name: Mapped[str | None]`（`String(255)`，可空）。
- `db.py` 的 `ensure_tenant_schema`（启动幂等迁移，已有 `department_id` 先例）新增：
  ```sql
  ALTER TABLE users ADD COLUMN IF NOT EXISTS disp_name VARCHAR(255) NULL
  ```
- `routers/users.py`：
  - `UserCreate`：新增可选 `disp_name: str | None = None`（创建时一并写入）。
  - `UserResponse`：新增 `disp_name: str | None`。
  - 新增 `UserUpdate`：`name`、`email`、`role`、`department_id`、`disp_name` 全可选；`password: str | None = None`（可选重置，≥8 位）。`model_config = {"from_attributes": True}`。

### 3.2 用户编辑接口

- `PATCH /users/{id}`（`require_admin`）：
  - 解析 `uuid.UUID(user_id)`，非法 → 404。
  - 用户不存在 → 404。
  - 若 `email` 变更且与其它用户冲突 → 409。
  - 若提供 `password` 且长度 < 8 → 400。
  - 更新各非空字段；`password` 提供时 `password_hash = hash_password(...)`。
  - 返回 `UserResponse`。

### 3.3 按用户的 API Key 管理（`routers/users.py`，均 `require_admin`）

- `GET /users/{id}/api-keys`：返回 `select(APIKey).where(created_by == id, revoked_at.is_(None))` 列表，序列化复用现有的 `KeyListItem`（只回显前缀，不含完整 key）。
- `POST /users/{id}/api-keys`（body `{label: str}`）：以 `created_by = 目标用户 id` 调用 `generate_api_key()` 创建，返回 `CreateKeyResponse`（含一次性完整 key）。

### 3.4 放宽密钥吊销（`routers/api_keys.py`）

- `DELETE /api-keys/{id}`：当前仅允许 `created_by == 当前用户`。改为：
  - 若 `_auth.role == "admin"` → 可吊销任意密钥（含他人）。
  - 否则维持 `created_by == 当前用户` 的现有约束。
- 注意：`DELETE /api-keys/{id}` 使用 `require_auth`（非 `require_admin`），需在该路由内部读取 `user.role` 判断，而非依赖依赖项返回值。

### 3.5 错误处理

| 情形 | 状态码 |
| --- | --- |
| 用户 / 密钥不存在、非法 uuid | 404 |
| 邮箱与现有用户冲突 | 409 |
| 密码过短、role 非法 | 400 |
| 非管理员调用按用户接口 / 吊销他人密钥 | 403 |

## 4. 前端实现（`mem0/server/dashboard`）

### 4.1 常量与类型

- `utils/api-endpoints.ts`：`USER_ENDPOINTS` 增加
  ```ts
  API_KEYS: (id: string) => `/users/${id}/api-keys`,
  API_KEYS_CREATE: (id: string) => `/users/${id}/api-keys`,
  ```
  保留 `API_KEY_ENDPOINTS.BASE` + `BY_ID(id)` 用于吊销（现支持 admin）。
- `types/api.ts`：`DashboardUser` 增加 `disp_name: string | null`；新增 `UserUpdate` 类型（字段同 `UserUpdate` schema，password 可选）；复用 `ApiKey` / `ApiKeyCreateResponse`。

### 4.2 `users/page.tsx` 重构为详情卡片

- 顶部保留搜索框 + "New user" 按钮；创建弹窗新增 `disp_name` 输入。
- 列表改为**卡片网格/列表**，每个用户一张 `Card`：
  - 头部：显示名（优先 `disp_name`，否则 `name`）、邮箱、角色徽章、部门（按 `department_id` 反查名称，或仅显示 id 占位）、创建时间；右侧 **Edit** / **Delete** 按钮。
  - 下半部分：**API Keys** 子区：
    - 卡片内独立 `useApiQuery` 调用 `GET /users/{id}/api-keys`（挂载即加载）。
    - 密钥列表：label、前缀…、创建时间、最近使用、吊销按钮（`DeleteConfirmationModal`）。
    - "Create key" 按钮 → 弹窗输入 label → 调 `POST /users/{id}/api-keys` → 展示一次性 key（复制）→ 刷新。
- **Edit 弹窗**：字段 name / email / role（select admin|user）/ department（从 `GET /departments` 拉取下拉）/ disp_name / 可选密码重置（placeholder "留空则不修改"）；提交 `PATCH /users/{id}`。
- Delete 沿用现有 `DeleteConfirmationModal`。

### 4.3 独立 `api-keys` 页面

保持现状（非管理员自我管理本人密钥）。仅当 `DELETE /api-keys/{id}` 后端放宽后，其行为对管理员自然扩展为可吊销任意密钥，但页面本身面向当前登录用户。

### 4.4 i18n

在 `i18n` 资源新增键（zh/en）：`tenant.dispName`、`tenant.apiKeys`、`tenant.createKey`、`tenant.revoke`、`tenant.resetPassword`、`tenant.editUser`、`tenant.newApiKey`、`tenant.copyKeyHint` 等，与现有 `tenant.*` 命名一致。

## 5. 测试与验证

### 后端 pytest（`mem0/server/tests`）
- `PATCH /users/{id}`：修改成功、邮箱冲突 409、密码过短 400、缺失 404、非 admin 403。
- `GET /users/{id}/api-keys` 与 `POST /users/{id}/api-keys`：admin 成功、非 admin 403、用户不存在 404。
- `DELETE /api-keys/{id}`：admin 吊销他人成功、非 admin 吊销他人 403。
- `GET /users` 与 `UserResponse` 含 `disp_name`。

### 前端
- `tsc` 类型检查通过。
- 手动：创建（含 disp_name）→ 列表显示 → 编辑各字段 → 卡片内查看/创建/吊销某用户密钥 → 删除用户。

## 6. 非目标（本轮不做）
- 部门级 / 项目级密钥共享。
- 前端把独立 api-keys 导航项移除（保留给非管理员）。
- 卡片密钥懒加载（默认挂载即加载，后续如需再优化为展开时加载或后端批量接口）。

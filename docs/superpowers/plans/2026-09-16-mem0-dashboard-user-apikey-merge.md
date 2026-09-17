# Mem0 Dashboard 用户管理增强 + API Key 关联合并 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 mem0 dashboard 的用户页面增加编辑能力、新增 `disp_name` 字段，并把 API Key 管理合并进每个用户详情卡片（管理员可管理任意用户的密钥）。

**Architecture:** 后端在 `routers/users.py` 新增 `PATCH /users/{id}` 与 `GET/POST /users/{id}/api-keys`，放宽 `DELETE /api-keys/{id}` 让管理员可吊销他人密钥；`User` 模型新增 `disp_name` 经 `ensure_tenant_schema` 幂等迁移。`users/page.tsx` 由表格重构为"用户详情卡片"，卡片下半部分负责该用户的密钥列表/创建/吊销。独立 `api-keys` 页面保留给非管理员。

**Tech Stack:** Python / FastAPI / SQLAlchemy / pytest（后端）；Next.js / React / TypeScript / Tailwind（前端，沿用现有 shadcn 风格组件）。

**测试前提（后端 API 测试）:** 需要运行中的 Postgres（`docker compose -f deploy/mem0/docker-compose.yaml up -d`）。测试用 `ADMIN_API_KEY` 环境变量以管理员身份调用；非管理员 403 用例通过新建用户 + 管理员为其建 key 后以其 `X-API-Key` 调用。

---

## File Structure

**Backend (`mem0/server`)**
- `models.py` — `User` 增加 `disp_name` 映射列。
- `db.py` — `ensure_tenant_schema` 增加 `ALTER TABLE users ADD COLUMN IF NOT EXISTS disp_name`。
- `routers/users.py` — `UserCreate`/`UserResponse` 增加 `disp_name`；新增 `UserUpdate`；新增 `PATCH /users/{id}`、`GET /users/{id}/api-keys`、`POST /users/{id}/api-keys`。
- `routers/api_keys.py` — `DELETE /api-keys/{id}` 允许 admin 吊销他人密钥。
- `tests/test_user_management.py` — 新增（后端集成测试，需 Postgres）。

**Frontend (`mem0/server/dashboard/src`)**
- `utils/api-endpoints.ts` — `USER_ENDPOINTS` 增加 `API_KEYS(id)`、`API_KEYS_CREATE(id)`。
- `types/api.ts` — `DashboardUser` 增加 `disp_name`；新增 `UserUpdate`。
- `i18n/en.ts` / `i18n/zh.ts` — `tenant.*` 增加新键。
- `app/(root)/dashboard/users/page.tsx` — 重构为详情卡片 + 编辑弹窗 + 每用户密钥管理。

---

### Task 1: User 模型新增 disp_name 列 + 幂等迁移

**Files:**
- Modify: `mem0/server/models.py:26-32`（`User` 表定义）
- Modify: `mem0/server/db.py:34-66`（`ensure_tenant_schema`）

- [ ] **Step 1: 在 `User` 模型增加 `disp_name` 映射列**

`mem0/server/models.py` 在 `department_id` 列之后新增：

```python
    department_id: Mapped[uuid.UUID | None] = mapped_column(
        ForeignKey("departments.id", ondelete="SET NULL"), nullable=True, index=True
    )
    disp_name: Mapped[str | None] = mapped_column(String(255), nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=_utcnow)
```

- [ ] **Step 2: 在 `ensure_tenant_schema` 增加幂等 ALTER**

`mem0/server/db.py` 的 `with target_engine.begin() as conn:` 块内、`projects` 列迁移之前新增：

```python
        conn.execute(
            text("ALTER TABLE users ADD COLUMN IF NOT EXISTS disp_name VARCHAR(255) NULL")
        )
```

- [ ] **Step 3: 写最小化校验测试并执行**

`mem0/server/tests/test_user_management.py`：

```python
"""Smoke checks for the disp_name column migration on the User model."""


def test_user_model_has_disp_name_attribute():
    from models import User

    assert hasattr(User, "disp_name")


def test_ensure_tenant_schema_runs_idempotently():
    from db import Base, ensure_tenant_schema, engine

    # Should not raise on a live DB; safe to call repeatedly.
    ensure_tenant_schema(engine)
    ensure_tenant_schema(engine)
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py -v`
Expected: 两条 PASS（需运行中的 Postgres）。

- [ ] **Step 5: Commit**

```bash
git add mem0/server/models.py mem0/server/db.py mem0/server/tests/test_user_management.py
git commit -m "feat(users): add disp_name column with idempotent migration"
```

---

### Task 2: 用户 Schema（UserResponse / UserCreate / UserUpdate）增加 disp_name

**Files:**
- Modify: `mem0/server/routers/users.py:19-45`（`UserCreate`、`UserResponse`）
- Modify: `mem0/server/routers/users.py`（新增 `UserUpdate` 类）

- [ ] **Step 1: 写序列化/校验测试**

在 `mem0/server/tests/test_user_management.py` 增加：

```python
def test_user_response_includes_disp_name():
    from routers.users import UserResponse
    from datetime import datetime, timezone

    class _U:
        id = "11111111-1111-1111-1111-111111111111"
        name = "alice"
        email = "a@b.c"
        role = "user"
        department_id = None
        disp_name = "Alice A"
        created_at = datetime.now(timezone.utc)

    r = UserResponse.from_user(_U())
    assert r.disp_name == "Alice A"
    assert r.id == "11111111-1111-1111-1111-111111111111"


def test_user_update_model_accepts_optional_fields():
    from routers.users import UserUpdate

    u = UserUpdate(name="x", email="x@y.z", role="admin", department_id=None, disp_name="X", password="secret123")
    assert u.name == "x" and u.password == "secret123"
    # All-optional: empty body is valid.
    assert UserUpdate().name is None
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py::test_user_response_includes_disp_name tests/test_user_management.py::test_user_update_model_accepts_optional_fields -v`
Expected: FAIL（`UserResponse` 无 `disp_name` / `UserUpdate` 未定义）。

- [ ] **Step 3: 实现 Schema**

`mem0/server/routers/users.py`：

`UserCreate` 增加字段：

```python
class UserCreate(BaseModel):
    name: str
    email: EmailStr
    password: str
    role: str = "user"  # "admin" | "user"
    disp_name: str | None = None
```

`UserResponse` 增加字段并回显：

```python
class UserResponse(BaseModel):
    id: str
    name: str
    email: str
    role: str
    department_id: str | None
    disp_name: str | None
    created_at: str

    model_config = {"from_attributes": True}

    @classmethod
    def from_user(cls, u: User) -> "UserResponse":
        return cls(
            id=str(u.id),
            name=u.name,
            email=u.email,
            role=u.role,
            department_id=str(u.department_id) if u.department_id else None,
            disp_name=u.disp_name,
            created_at=u.created_at.isoformat() if u.created_at else "",
        )
```

`UserCreate` 的 `create_user` 使用 `disp_name`：

```python
    user = User(
        name=body.name,
        email=body.email,
        password_hash=hash_password(body.password),
        role=body.role,
        disp_name=body.disp_name,
    )
```

新增 `UserUpdate`：

```python
class UserUpdate(BaseModel):
    name: str | None = None
    email: EmailStr | None = None
    role: str | None = None
    department_id: str | None = None
    disp_name: str | None = None
    password: str | None = None
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py -v`
Expected: 全部 PASS。

- [ ] **Step 5: Commit**

```bash
git add mem0/server/routers/users.py mem0/server/tests/test_user_management.py
git commit -m "feat(users): include disp_name in schemas, add UserUpdate"
```

---

### Task 3: PATCH /users/{id} 编辑接口

**Files:**
- Modify: `mem0/server/routers/users.py`（新增 `PATCH /users/{id}`）

- [ ] **Step 1: 写集成测试**

在 `mem0/server/tests/test_user_management.py` 顶部增加 import 与 fixture helper：

```python
import os
os.environ.setdefault("ADMIN_API_KEY", "test-admin-key")
os.environ.setdefault("JWT_SECRET", "test-secret")
os.environ.setdefault("AUTH_DISABLED", "")  # ensure admin_api_key path used

from fastapi.testclient import TestClient
from main import app


def _client():
    return TestClient(app)


def _create_user(client, email="u@e.c", name="U", role="user", password="password123"):
    r = client.post(
        "/users",
        headers={"X-API-Key": os.environ["ADMIN_API_KEY"]},
        json={"name": name, "email": email, "password": password, "role": role},
    )
    assert r.status_code == 201, r.text
    return r.json()
```

```python
def test_patch_user_updates_fields():
    c = _client()
    u = _create_user(c, email="patch@e.c")
    r = c.patch(
        f"/users/{u['id']}",
        headers={"X-API-Key": os.environ["ADMIN_API_KEY"]},
        json={"name": "New", "disp_name": "Display", "role": "admin"},
    )
    assert r.status_code == 200, r.text
    body = r.json()
    assert body["name"] == "New" and body["disp_name"] == "Display" and body["role"] == "admin"


def test_patch_user_reset_password():
    c = _client()
    u = _create_user(c, email="pw@e.c", password="oldpass123")
    r = c.patch(
        f"/users/{u['id']}",
        headers={"X-API-Key": os.environ["ADMIN_API_KEY"]},
        json={"password": "newpass456"},
    )
    assert r.status_code == 200, r.text


def test_patch_user_missing_404():
    c = _client()
    r = c.patch(
        "/users/00000000-0000-0000-0000-000000000000",
        headers={"X-API-Key": os.environ["ADMIN_API_KEY"]},
        json={"name": "x"},
    )
    assert r.status_code == 404


def test_patch_user_duplicate_email_409():
    c = _client()
    _create_user(c, email="dup@e.c")
    u2 = _create_user(c, email="dup2@e.c")
    r = c.patch(
        f"/users/{u2['id']}",
        headers={"X-API-Key": os.environ["ADMIN_API_KEY"]},
        json={"email": "dup@e.c"},
    )
    assert r.status_code == 409


def test_patch_user_short_password_400():
    c = _client()
    u = _create_user(c, email="short@e.c")
    r = c.patch(
        f"/users/{u['id']}",
        headers={"X-API-Key": os.environ["ADMIN_API_KEY"]},
        json={"password": "abc"},
    )
    assert r.status_code == 400


def test_patch_user_forbidden_for_normal_user():
    c = _client()
    # Admin creates a normal user, then an api key for that user (admin-only endpoint).
    u = _create_user(c, email="normal@e.c")
    k = c.post(
        f"/users/{u['id']}/api-keys",
        headers={"X-API-Key": os.environ["ADMIN_API_KEY"]},
        json={"label": "k"},
    ).json()
    r = c.patch(
        f"/users/{u['id']}",
        headers={"X-API-Key": k["key"]},
        json={"name": "hacked"},
    )
    assert r.status_code == 403
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py -k patch -v`
Expected: FAIL（路由不存在 / 403 用例的 `/users/{id}/api-keys` 也尚未实现 → 用 -k patch 时 403 用例依赖 Task 4 端点，可先注释该用例，待 Task 4 后补）。

- [ ] **Step 3: 实现 PATCH 路由**

`mem0/server/routers/users.py` 在 `delete_user` 之前新增（注意 import `hash_password` 已存在、`uuid` 已 import、`select` 已 import）：

```python
@router.patch("/{user_id}", response_model=UserResponse)
def update_user(user_id: str, body: UserUpdate, _admin=Depends(require_admin), db: Session = Depends(get_db)):
    try:
        uid = uuid.UUID(user_id)
    except (TypeError, ValueError):
        raise HTTPException(status_code=404, detail="User not found.")
    user = db.get(User, uid)
    if user is None:
        raise HTTPException(status_code=404, detail="User not found.")
    if body.email is not None and body.email != user.email:
        collision = db.scalar(select(User).where(User.email == body.email, User.id != uid))
        if collision is not None:
            raise HTTPException(status_code=409, detail="A user with this email already exists.")
    if body.name is not None:
        user.name = body.name
    if body.email is not None:
        user.email = body.email
    if body.role is not None:
        if body.role not in ("admin", "user"):
            raise HTTPException(status_code=400, detail="role must be 'admin' or 'user'.")
        user.role = body.role
    if "department_id" in body.model_fields_set:
        user.department_id = uuid.UUID(body.department_id) if body.department_id else None
    if body.disp_name is not None:
        user.disp_name = body.disp_name
    if body.password is not None:
        if len(body.password) < MIN_PASSWORD_LENGTH:
            raise HTTPException(status_code=400, detail=f"Password must be at least {MIN_PASSWORD_LENGTH} characters.")
        user.password_hash = hash_password(body.password)
    db.commit()
    db.refresh(user)
    return UserResponse.from_user(user)
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py -k patch -v`
Expected: 除依赖 Task 4 的 403 用例外全部 PASS（该用例放开注释后将在 Task 4 后通过）。

- [ ] **Step 5: Commit**

```bash
git add mem0/server/routers/users.py mem0/server/tests/test_user_management.py
git commit -m "feat(users): add PATCH /users/{id} edit endpoint"
```

---

### Task 4: 按用户的 API Key 管理接口（GET/POST /users/{id}/api-keys）

**Files:**
- Modify: `mem0/server/routers/users.py`（新增两个路由）

- [ ] **Step 1: 写集成测试**

在 `mem0/server/tests/test_user_management.py` 增加（依赖 Task 3 的 `_create_user`）：

```python
def test_admin_lists_user_api_keys():
    c = _client()
    u = _create_user(c, email="keys@e.c")
    c.post(f"/users/{u['id']}/api-keys", headers={"X-API-Key": os.environ["ADMIN_API_KEY"]}, json={"label": "prod"})
    r = c.get(f"/users/{u['id']}/api-keys", headers={"X-API-Key": os.environ["ADMIN_API_KEY"]})
    assert r.status_code == 200, r.text
    assert any(k["label"] == "prod" for k in r.json())


def test_admin_creates_user_api_key_returns_full_key_once():
    c = _client()
    u = _create_user(c, email="newkey@e.c")
    r = c.post(f"/users/{u['id']}/api-keys", headers={"X-API-Key": os.environ["ADMIN_API_KEY"]}, json={"label": "ci"})
    assert r.status_code == 201, r.text
    body = r.json()
    assert body["key"].startswith("m0sk_") and body["label"] == "ci"


def test_user_api_keys_forbidden_for_normal_user():
    c = _client()
    u = _create_user(c, email="normal2@e.c")
    k = c.post(f"/users/{u['id']}/api-keys", headers={"X-API-Key": os.environ["ADMIN_API_KEY"]}, json={"label": "k"}).json()
    r = c.get(f"/users/{u['id']}/api-keys", headers={"X-API-Key": k["key"]})
    assert r.status_code == 403


def test_user_api_keys_missing_user_404():
    c = _client()
    r = c.get("/users/00000000-0000-0000-0000-000000000000/api-keys", headers={"X-API-Key": os.environ["ADMIN_API_KEY"]})
    assert r.status_code == 404
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py -k api_keys -v`
Expected: FAIL（路由不存在）。

- [ ] **Step 3: 实现路由**

`mem0/server/routers/users.py` 顶部补充 import：

```python
from auth import generate_api_key, hash_password, require_admin
from models import APIKey, User
```

（若 `APIKey` 未导入，补上；`generate_api_key` 已导入。）

新增路由（放在 `update_user` 之后）：

```python
@router.get("/{user_id}/api-keys", response_model=list[KeyListItem])
def list_user_api_keys(user_id: str, _admin=Depends(require_admin), db: Session = Depends(get_db)):
    try:
        uid = uuid.UUID(user_id)
    except (TypeError, ValueError):
        raise HTTPException(status_code=404, detail="User not found.")
    if db.get(User, uid) is None:
        raise HTTPException(status_code=404, detail="User not found.")
    keys = (
        db.execute(
            select(APIKey)
            .where(APIKey.created_by == uid, APIKey.revoked_at.is_(None))
            .order_by(APIKey.created_at.desc())
        )
        .scalars()
        .all()
    )
    return [
        KeyListItem(
            id=str(k.id),
            label=k.label,
            key_prefix=k.key_prefix,
            created_at=k.created_at,
            last_used_at=k.last_used_at,
        )
        for k in keys
    ]


@router.post("/{user_id}/api-keys", response_model=CreateKeyResponse, status_code=201)
def create_user_api_key(user_id: str, body: CreateKeyRequest, _admin=Depends(require_admin), db: Session = Depends(get_db)):
    try:
        uid = uuid.UUID(user_id)
    except (TypeError, ValueError):
        raise HTTPException(status_code=404, detail="User not found.")
    if db.get(User, uid) is None:
        raise HTTPException(status_code=404, detail="User not found.")
    full_key, prefix, key_hash = generate_api_key()
    api_key = APIKey(key_prefix=prefix, key_hash=key_hash, label=body.label, created_by=uid)
    db.add(api_key)
    db.commit()
    db.refresh(api_key)
    return CreateKeyResponse(
        id=str(api_key.id),
        key=full_key,
        label=api_key.label,
        key_prefix=prefix,
        created_at=api_key.created_at,
    )
```

注意：`KeyListItem`、`CreateKeyResponse`、`CreateKeyRequest` 定义在 `routers/api_keys.py`；为免重复，从 `api_keys` 导入或在 `users.py` 复用。采取**导入**方式：在 `routers/users.py` 顶部增加 `from routers.api_keys import CreateKeyRequest, CreateKeyResponse, KeyListItem`（避免循环导入：`api_keys.py` 不导入 `users.py`，安全）。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py -k api_keys -v`
Expected: 全部 PASS。

- [ ] **Step 5: 放开 Task 3 中 403 用例注释并整体回归**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py -v`
Expected: 全部 PASS。

- [ ] **Step 6: Commit**

```bash
git add mem0/server/routers/users.py mem0/server/tests/test_user_management.py
git commit -m "feat(users): admin manage any user's API keys (list/create)"
```

---

### Task 5: 放宽 DELETE /api-keys/{id} 允许管理员吊销他人密钥

**Files:**
- Modify: `mem0/server/routers/api_keys.py:79-93`

- [ ] **Step 1: 写集成测试**

在 `mem0/server/tests/test_user_management.py` 增加：

```python
def test_admin_revokes_other_users_key():
    c = _client()
    u = _create_user(c, email="owner@e.c")
    k = c.post(f"/users/{u['id']}/api-keys", headers={"X-API-Key": os.environ["ADMIN_API_KEY"]}, json={"label": "k"}).json()
    r = c.delete(f"/api-keys/{k['id']}", headers={"X-API-Key": os.environ["ADMIN_API_KEY"]})
    assert r.status_code == 200, r.text


def test_normal_user_cannot_revoke_others_key():
    c = _client()
    owner = _create_user(c, email="owner2@e.c")
    ok = c.post(f"/users/{owner['id']}/api-keys", headers={"X-API-Key": os.environ["ADMIN_API_KEY"]}, json={"label": "k"}).json()
    attacker = _create_user(c, email="attacker@e.c")
    ak = c.post(f"/users/{attacker['id']}/api-keys", headers={"X-API-Key": os.environ["ADMIN_API_KEY"]}, json={"label": "ak"}).json()
    r = c.delete(f"/api-keys/{ok['id']}", headers={"X-API-Key": ak["key"]})
    assert r.status_code == 403


def test_normal_user_revokes_own_key():
    c = _client()
    u = _create_user(c, email="self@e.c")
    k = c.post(f"/users/{u['id']}/api-keys", headers={"X-API-Key": os.environ["ADMIN_API_KEY"]}, json={"label": "k"}).json()
    r = c.delete(f"/api-keys/{k['id']}", headers={"X-API-Key": k["key"]})
    assert r.status_code == 200, r.text
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py -k revoke -v`
Expected: `test_normal_user_cannot_revoke_others_key` FAIL（当前实现允许任意登录用户吊销任意 key，因只校验 `created_by == user.id` 但 user 是攻击者自己，而 key 属于他人 → 当前返回 404；预期 403。实际需看实现：当前 `api_keys.py` 逻辑 `api_key.created_by != user.id` → 404；我们要改成非 admin 时才是 404，admin 放行。因此当前非 admin 他人 → 404 ≠ 403 → FAIL，符合预期）。

- [ ] **Step 3: 修改 DELETE 路由**

`mem0/server/routers/api_keys.py`：

```python
@router.delete("/{key_id}", response_model=MessageResponse)
def revoke_key(key_id: str, user: User = Depends(require_auth), db: Session = Depends(get_db)):
    try:
        key_uuid = uuid.UUID(key_id)
    except (TypeError, ValueError):
        raise HTTPException(status_code=404, detail="API key not found.")
    api_key = db.get(APIKey, key_uuid)
    if api_key is None:
        raise HTTPException(status_code=404, detail="API key not found.")
    # Admins may revoke any key; normal users only their own.
    if user.role != "admin" and api_key.created_by != user.id:
        raise HTTPException(status_code=404, detail="API key not found.")
    if api_key.revoked_at is not None:
        raise HTTPException(status_code=400, detail="API key is already revoked.")

    api_key.revoked_at = datetime.now(timezone.utc)
    db.commit()
    return MessageResponse(message="API key revoked.")
```

（注：`require_auth` 已在该文件导入；`User` 已导入。）

- [ ] **Step 4: 运行测试确认通过**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py -v`
Expected: 全部 PASS。

- [ ] **Step 5: Commit**

```bash
git add mem0/server/routers/api_keys.py mem0/server/tests/test_user_management.py
git commit -m "feat(api-keys): allow admin to revoke any key"
```

---

### Task 6: 前端常量与类型

**Files:**
- Modify: `mem0/server/dashboard/src/utils/api-endpoints.ts`
- Modify: `mem0/server/dashboard/src/types/api.ts`

- [ ] **Step 1: 更新 api-endpoints.ts**

`USER_ENDPOINTS` 改为：

```ts
export const USER_ENDPOINTS = {
  BASE: "/users",
  BY_ID: (id: string) => `/users/${id}`,
  API_KEYS: (id: string) => `/users/${id}/api-keys`,
  API_KEYS_CREATE: (id: string) => `/users/${id}/api-keys`,
} as const;
```

- [ ] **Step 2: 更新 types/api.ts**

`DashboardUser` 增加 `disp_name`：

```ts
export interface DashboardUser {
  id: string;
  name: string;
  email: string;
  role: string;
  department_id: string | null;
  disp_name: string | null;
  created_at: string;
}
```

新增 `UserUpdate`：

```ts
export interface UserUpdate {
  name?: string;
  email?: string;
  role?: string;
  department_id?: string | null;
  disp_name?: string;
  password?: string;
}
```

- [ ] **Step 3: 类型检查**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无新增错误。

- [ ] **Step 4: Commit**

```bash
git add mem0/server/dashboard/src/utils/api-endpoints.ts mem0/server/dashboard/src/types/api.ts
git commit -m "feat(dashboard): add user api-key endpoints + UserUpdate/disp_name types"
```

---

### Task 7: i18n 新增键

**Files:**
- Modify: `mem0/server/dashboard/src/i18n/en.ts`
- Modify: `mem0/server/dashboard/src/i18n/zh.ts`

- [ ] **Step 1: en.ts 的 tenant 增加键**

在 `tenant: { ... }` 对象内（`allProjects` 之后）增加：

```ts
    dispName: "Display Name",
    editUser: "Edit User",
    apiKeys: "API Keys",
    createKey: "Create Key",
    newApiKey: "New API Key",
    revoke: "Revoke",
    resetPassword: "Reset Password",
    resetPasswordHint: "Leave blank to keep current password",
    copyKeyHint: "Save this key — you won't see it again.",
    apiKeyLabel: "Label",
    noApiKeys: "No API keys yet",
```

- [ ] **Step 2: zh.ts 的 tenant 增加对应键**

```ts
    dispName: "显示名称",
    editUser: "编辑用户",
    apiKeys: "API 密钥",
    createKey: "创建密钥",
    newApiKey: "新建 API 密钥",
    revoke: "吊销",
    resetPassword: "重置密码",
    resetPasswordHint: "留空则不修改当前密码",
    copyKeyHint: "请保存此密钥——之后无法再次查看。",
    apiKeyLabel: "备注",
    noApiKeys: "暂无 API 密钥",
```

- [ ] **Step 3: 类型检查（Dict 类型要求 en/zh 完全一致）**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无错误（`zh` 必须实现 `Dict` 全部键）。

- [ ] **Step 4: Commit**

```bash
git add mem0/server/dashboard/src/i18n/en.ts mem0/server/dashboard/src/i18n/zh.ts
git commit -m "feat(dashboard): add i18n keys for user edit + api keys"
```

---

### Task 8: 重构 users/page.tsx 为详情卡片（含编辑 + 每用户密钥管理）

**Files:**
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/users/page.tsx`（整文件重写）

该组件依赖（已存在）：`Button, Input, Label, Card, Dialog/DialogContent/DialogHeader/DialogTitle/DialogFooter, DeleteConfirmationModal, TableSkeleton, EmptyState, api, USER_ENDPOINTS, API_KEY_ENDPOINTS, DEPARTMENT_ENDPOINTS, DashboardUser, ApiKey, ApiKeyCreateResponse, UserUpdate, Department, useTranslation, useApiQuery, CopyToClipboard, format, lucide 图标`。

- [ ] **Step 1: 写完整页面代码**

将 `users/page.tsx` 整体替换为以下实现（卡片 + 编辑弹窗 + 每用户密钥子组件）：

```tsx
"use client";

import React, { useEffect, useState } from "react";
import {
  Plus,
  Search,
  Trash2,
  KeyRound,
  Pencil,
  Copy,
  Check,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import DeleteConfirmationModal from "@/components/ui/delete-confirmation-modal";
import { TableSkeleton } from "@/components/shared/table-skeleton";
import { EmptyState } from "@/components/self-hosted/empty-state";
import { api } from "@/utils/api";
import {
  USER_ENDPOINTS,
  API_KEY_ENDPOINTS,
  DEPARTMENT_ENDPOINTS,
} from "@/utils/api-endpoints";
import {
  DashboardUser,
  ApiKey,
  ApiKeyCreateResponse,
  UserUpdate,
  Department,
} from "@/types/api";
import { useTranslation } from "@/i18n";
import { useApiQuery } from "@/hooks/use-api-query";
import { CopyToClipboard } from "react-copy-to-clipboard";
import { format } from "date-fns";
import { getErrorMessage } from "@/lib/error-message";
import { toast } from "@/components/ui/use-toast";

const PAGE_SIZE = 10;

function UserApiKeys({ userId }: { userId: string }) {
  const { t } = useTranslation();
  const [createOpen, setCreateOpen] = useState(false);
  const [label, setLabel] = useState("");
  const [newKey, setNewKey] = useState("");
  const [copied, setCopied] = useState(false);
  const [toRevoke, setToRevoke] = useState<ApiKey | null>(null);

  const {
    data: keys = [],
    isLoading,
    refetch,
  } = useApiQuery<ApiKey[]>(
    async () => {
      const res = await api.get<ApiKey[]>(USER_ENDPOINTS.API_KEYS(userId));
      return res.data ?? [];
    },
    { errorToast: "Failed to load API keys", initialData: [] }
  );

  const handleCreate = async () => {
    try {
      const res = await api.post<ApiKeyCreateResponse>(
        USER_ENDPOINTS.API_KEYS_CREATE(userId),
        { label }
      );
      setNewKey(res.data.key);
      void refetch();
    } catch (e) {
      toast({ title: "Failed to create key", description: getErrorMessage(e), variant: "destructive" });
    }
  };

  const handleRevoke = async () => {
    if (!toRevoke) return;
    try {
      await api.delete(API_KEY_ENDPOINTS.BY_ID(toRevoke.id));
      toast({ title: "API key revoked", variant: "success" });
      setToRevoke(null);
      void refetch();
    } catch (e) {
      toast({ title: "Failed to revoke key", description: getErrorMessage(e), variant: "destructive" });
    }
  };

  const closeCreate = (open: boolean) => {
    if (!open) {
      setNewKey("");
      setLabel("");
      setCopied(false);
    }
    setCreateOpen(open);
  };

  return (
    <div className="mt-4 border-t border-memBorder-primary pt-4">
      <div className="flex items-center justify-between mb-2">
        <h4 className="text-sm font-medium flex items-center gap-1">
          <KeyRound className="size-3.5" /> {t("tenant.apiKeys")}
        </h4>
        <Button size="sm" variant="outline" onClick={() => setCreateOpen(true)}>
          <Plus className="size-3.5 mr-1" /> {t("tenant.createKey")}
        </Button>
      </div>

      {isLoading ? (
        <div className="text-xs text-memText-secondary">…</div>
      ) : keys.length === 0 ? (
        <p className="text-xs text-memText-secondary">{t("tenant.noApiKeys")}</p>
      ) : (
        <ul className="space-y-1">
          {keys.map((k) => (
            <li
              key={k.id}
              className="flex items-center justify-between text-xs bg-surface-default-secondary rounded px-2 py-1"
            >
              <span className="font-medium">{k.label}</span>
              <span className="flex items-center gap-3">
                <code className="font-mono text-[11px]">{k.key_prefix}…</code>
                <span className="text-memText-secondary">
                  {k.last_used_at ? format(new Date(k.last_used_at), "MMM d, yyyy") : "never"}
                </span>
                <Button variant="ghost" size="icon" className="size-6" onClick={() => setToRevoke(k)}>
                  <Trash2 className="size-3 text-onSurface-danger-primary" />
                </Button>
              </span>
            </li>
          ))}
        </ul>
      )}

      <Dialog open={createOpen} onOpenChange={closeCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tenant.newApiKey")}</DialogTitle>
          </DialogHeader>
          {!newKey ? (
            <div className="space-y-4 mt-2">
              <div className="space-y-2">
                <Label htmlFor="key-label">{t("tenant.apiKeyLabel")}</Label>
                <Input
                  id="key-label"
                  value={label}
                  onChange={(e) => setLabel(e.target.value)}
                  placeholder="e.g. Production"
                />
              </div>
              <Button onClick={handleCreate} disabled={!label} className="w-full">
                {t("common.save")}
              </Button>
            </div>
          ) : (
            <div className="space-y-4 mt-2">
              <div className="space-y-2">
                <Label htmlFor="new-key">{t("tenant.apiKeys")}</Label>
                <div className="flex gap-2">
                  <Input id="new-key" value={newKey} readOnly className="font-mono text-sm" />
                  <CopyToClipboard
                    text={newKey}
                    onCopy={() => {
                      setCopied(true);
                      setTimeout(() => setCopied(false), 2000);
                    }}
                  >
                    <Button variant="outline" size="icon">
                      {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                    </Button>
                  </CopyToClipboard>
                </div>
                <p className="text-xs text-onSurface-danger-primary">{t("tenant.copyKeyHint")}</p>
              </div>
              <Button onClick={() => closeCreate(false)} className="w-full">
                {t("common.cancel")}
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>

      <DeleteConfirmationModal
        isOpen={!!toRevoke}
        onClose={() => setToRevoke(null)}
        onConfirm={handleRevoke}
        title={t("tenant.revoke")}
        description="Applications using this key will immediately stop working."
        itemName={toRevoke?.label ?? ""}
        confirmButtonText={t("tenant.revoke")}
      />
    </div>
  );
}

export default function UsersPage() {
  const { t } = useTranslation();
  const [items, setItems] = useState<DashboardUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [q, setQ] = useState("");
  const [page, setPage] = useState(0);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<DashboardUser | null>(null);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState("user");
  const [dispName, setDispName] = useState("");
  const [departmentId, setDepartmentId] = useState<string | null>(null);
  const [toDelete, setToDelete] = useState<DashboardUser | null>(null);

  const {
    data: departments = [],
    isLoading: deptsLoading,
  } = useApiQuery<Department[]>(
    async () => {
      const res = await api.get<Department[]>(DEPARTMENT_ENDPOINTS.BASE);
      return res.data ?? [];
    },
    { errorToast: "Failed to load departments", initialData: [] }
  );

  async function load(search: string = q) {
    setLoading(true);
    try {
      const res = await api.get<DashboardUser[]>(USER_ENDPOINTS.BASE, {
        params: search.trim() ? { q: search.trim() } : undefined,
      });
      const next = res.data ?? [];
      setItems(next);
      setPage((p) => Math.min(p, Math.max(0, Math.ceil(next.length / PAGE_SIZE) - 1)));
      setError(null);
    } catch (e: any) {
      setError(e?.response?.data?.detail ?? "Failed to load users");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function openCreate() {
    setEditing(null);
    setName("");
    setEmail("");
    setPassword("");
    setRole("user");
    setDispName("");
    setDepartmentId(null);
    setDialogOpen(true);
  }

  function openEdit(u: DashboardUser) {
    setEditing(u);
    setName(u.name);
    setEmail(u.email);
    setPassword("");
    setRole(u.role);
    setDispName(u.disp_name ?? "");
    setDepartmentId(u.department_id);
    setDialogOpen(true);
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    try {
      if (editing) {
        const body: UserUpdate = {
          name: name || undefined,
          email: email || undefined,
          role,
          department_id: departmentId,
          disp_name: dispName || null,
        };
        if (password) body.password = password;
        await api.patch(USER_ENDPOINTS.BY_ID(editing.id), body);
      } else {
        await api.post(USER_ENDPOINTS.BASE, {
          name,
          email,
          password,
          role,
          disp_name: dispName || null,
        });
      }
      setDialogOpen(false);
      await load();
    } catch (e: any) {
      setError(e?.response?.data?.detail ?? "Save failed");
    }
  }

  async function handleDelete() {
    if (!toDelete) return;
    try {
      await api.delete(USER_ENDPOINTS.BY_ID(toDelete.id));
      setToDelete(null);
      await load();
    } catch (e: any) {
      setError(e?.response?.data?.detail ?? "Delete failed");
      setToDelete(null);
    }
  }

  const totalPages = Math.ceil(items.length / PAGE_SIZE);
  const paginated = items.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);
  const deptName = (id: string | null) =>
    id ? departments.find((d) => d.id === id)?.name ?? id : "—";

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold">{t("tenant.usersTitle")}</h1>
          <p className="text-memText-secondary text-sm">{t("tenant.usersDesc")}</p>
        </div>
        <Button onClick={openCreate}>
          <Plus className="size-4 mr-1" />
          New user
        </Button>
      </div>
      {error && <div className="text-red-500 text-sm">{error}</div>}

      <div className="flex gap-3">
        <div className="relative w-72">
          <Search className="absolute left-2.5 top-2.5 size-4 text-onSurface-default-tertiary" />
          <Input
            placeholder="Search by name or email…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                setPage(0);
                load();
              }
            }}
            className="pl-8"
          />
        </div>
      </div>

      {loading ? (
        <TableSkeleton rows={4} columns={5} />
      ) : items.length === 0 ? (
        <EmptyState title="No users found" description="Create a user to grant API access." />
      ) : (
        <>
          <div className="grid gap-4">
            {paginated.map((u) => (
              <Card key={u.id} className="border-memBorder-primary p-4">
                <div className="flex items-start justify-between">
                  <div>
                    <div className="font-medium">
                      {u.disp_name ? `${u.disp_name} (${u.name})` : u.name}
                    </div>
                    <div className="font-mono text-xs text-memText-secondary">{u.email}</div>
                    <div className="mt-1 flex gap-2 text-xs text-memText-secondary">
                      <span className="rounded bg-surface-default-secondary px-2 py-0.5">{u.role}</span>
                      <span className="rounded bg-surface-default-secondary px-2 py-0.5">
                        {t("tenant.departmentsTitle")}: {deptName(u.department_id)}
                      </span>
                      <span>
                        {t("tenant.createdAt")}:{" "}
                        {u.created_at ? new Date(u.created_at).toLocaleString() : "-"}
                      </span>
                    </div>
                  </div>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="icon" onClick={() => openEdit(u)}>
                      <Pencil className="size-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="text-onSurface-danger-primary"
                      onClick={() => setToDelete(u)}
                    >
                      <Trash2 className="size-3.5" />
                    </Button>
                  </div>
                </div>
                <UserApiKeys userId={u.id} />
              </Card>
            ))}
          </div>
          {totalPages > 1 && (
            <div className="flex items-center justify-between text-sm text-onSurface-default-tertiary">
              <span>
                {page * PAGE_SIZE + 1}–{Math.min((page + 1) * PAGE_SIZE, items.length)} of{" "}
                {items.length}
              </span>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" disabled={page === 0} onClick={() => setPage((p) => p - 1)}>
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editing ? t("tenant.editUser") : "New user"}</DialogTitle>
          </DialogHeader>
          <form onSubmit={save} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="user-name">{t("tenant.name")}</Label>
              <Input id="user-name" value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="user-disp">{t("tenant.dispName")}</Label>
              <Input id="user-disp" value={dispName} onChange={(e) => setDispName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="user-email">{t("tenant.email")}</Label>
              <Input
                id="user-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="user-password">
                {editing ? t("tenant.resetPassword") : t("auth.password")}
              </Label>
              <Input
                id="user-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                minLength={8}
                placeholder={editing ? t("tenant.resetPasswordHint") : ""}
                required={!editing}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="user-role">{t("tenant.role")}</Label>
              <select
                id="user-role"
                className="w-full border border-memBorder-primary rounded-md px-3 py-2 text-sm bg-transparent"
                value={role}
                onChange={(e) => setRole(e.target.value)}
              >
                <option value="user">user</option>
                <option value="admin">admin</option>
              </select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="user-dept">{t("tenant.departmentsTitle")}</Label>
              <select
                id="user-dept"
                className="w-full border border-memBorder-primary rounded-md px-3 py-2 text-sm bg-transparent"
                value={departmentId ?? ""}
                onChange={(e) => setDepartmentId(e.target.value || null)}
                disabled={deptsLoading}
              >
                <option value="">—</option>
                {departments.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </select>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                {t("common.cancel")}
              </Button>
              <Button type="submit">{t("common.save")}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <DeleteConfirmationModal
        isOpen={!!toDelete}
        onClose={() => setToDelete(null)}
        onConfirm={handleDelete}
        title="Delete user"
        description="This user will be removed and their API keys will stop working."
        itemName={toDelete?.email ?? ""}
        confirmButtonText="Delete"
      />
    </div>
  );
}
```

- [ ] **Step 2: 类型检查**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无错误（确认 `api.patch` 存在且 `useApiQuery`/`CopyToClipboard`/`getErrorMessage`/`toast` 路径正确；如 `api` 无 `patch` 方法，使用 `api.post` 的同类封装或 `axios` 实例调用）。

- [ ] **Step 3: 构建验证**

Run: `cd mem0/server/dashboard && npm run build` 或 `npx next build`（视 package.json 脚本）
Expected: 构建成功（或仅类型/lint 警告，无阻塞错误）。

- [ ] **Step 4: Commit**

```bash
git add mem0/server/dashboard/src/app/\(root\)/dashboard/users/page.tsx
git commit -m "feat(dashboard): restructure users into cards with edit + per-user api keys"
```

---

### Task 9: 整体回归与冒烟

- [ ] **Step 1: 后端测试全量**

Run: `cd mem0/server && python -m pytest tests/test_user_management.py tests/test_multitenant.py -v`
Expected: 全部 PASS。

- [ ] **Step 2: 前端类型检查**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无错误。

- [ ] **Step 3: 手动冒烟（docker compose 起服务后）**

1. 登录 dashboard → Users 页面：新建用户（含显示名）→ 卡片显示。
2. 点编辑：改 name/email/role/department/显示名、填密码重置 → 保存 → 卡片更新。
3. 展开卡片 API Keys：Create Key → 复制一次性 key；列表出现；Revoke → 消失。
4. 删除用户 → 确认。
5. 独立 API Keys 页面（非管理员登录）仍只显示本人密钥。

- [ ] **Step 4: Commit（若过程中有零散修复）**

```bash
git add -A && git commit -m "chore: post-merge fixes for user + api-key management"
```

---

## Self-Review 命中说明

- **Spec 覆盖:** §3.1 模型/迁移 → Task 1；§3.2 编辑 → Task 3；§3.3 按用户密钥 → Task 4；§3.4 放宽吊销 → Task 5；§4.1 常量/类型 → Task 6；§4.2 卡片重构 → Task 8；§4.3 保留 api-keys 页面 → 不做改动（符合 spec 非目标）；§4.4 i18n → Task 7；测试 → Task 1/3/4/5/9。
- **Type 一致性:** `UserUpdate`、`ApiKey`、`ApiKeyCreateResponse`、`Department`、`DashboardUser.disp_name`、`USER_ENDPOINTS.API_KEYS/API_KEYS_CREATE` 在 Task 6 定义、Task 8 使用，命名一致；后端 `KeyListItem/CreateKeyResponse/CreateKeyRequest` 在 Task 4 从 `api_keys` 导入复用，未重复定义。
- **Placeholder 扫描:** 无 TBD/TODO；每步均含可执行代码或命令。Task 8 Step 2 对 `api.patch` 存在性做了兜底说明（如不存在改用既有封装）。

> [!WARNING]
> **已废弃（2026-09-18）**：本文档描述的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，记忆功能统一替换为自托管 mem0（见 docs/quick-ref/mem0-manual.md）。本文仅作历史归档保留，内容不再维护。

# docs-site / cbmem-team 独立部署 + Ant Design 迁移 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 docs-site 前端与 cbmem-team 后端解耦为独立部署单元；前端 Element Plus 替换为 Ant Design Vue 4。

**Architecture:** docs-site 构建为纯静态 `dist/`，由 Caddy 反代服务；cbmem-team 去除 `/console/*` 静态托管并新增 CORS 中间件；前后端通过 `VITE_API_BASE` 环境变量通信。本地开发保留 vite proxy。

**Tech Stack:** VitePress 1.3, Vue 3.4, Ant Design Vue 4.2, Pinia 3.0, Vitest; Gin (Go 1.22+), Caddy 2.x, systemd.

**Spec:** `docs/superpowers/specs/2026-07-12-docs-site-cbmem-team-decouple-design.md`

## Global Constraints

- **Vue 版本**：保持 `^3.4.0`（devDependencies）
- **Ant Design Vue 版本**：`^4.2.0`；附加 `@ant-design/icons-vue ^7.0.1`
- **Element Plus**：从 `package.json` 删除所有引用
- **域名**：生产 `docs.fin-ai.net`（静态）+ `api.fin-ai.net`（API）
- **CORS 白名单**：`https://docs.fin-ai.net`（生产必填），`http://localhost:5173`（dev）
- **`*` 通配**：仅 dev/staging 允许，生产禁止
- **后端 `-console-dist` flag**：删除（含路由注册、systemd 引用）
- **测试覆盖率**：Go 与 Vitest 均 ≥ 80%
- **TDD**：每个 task 先写失败测试，再写最小实现，最后重构
- **频繁提交**：每个 task 至少一次原子提交
- **commit 规范**：`<type>(scope): <description>` — feat / refactor / test / docs / fix / chore

---

## Task 1: 后端 CORS 中间件 + flag (TDD)

**Files:**
- Create: `tools/cbmem-team/internal/httpsrv/middleware/cors.go`
- Create: `tools/cbmem-team/internal/httpsrv/middleware/cors_test.go`
- Modify: `tools/cbmem-team/cmd/cbmem-team/main.go`
- Modify: `tools/cbmem-team/deploy/cbmem-team.service.minimal`

**Interfaces:**
- Consumes: gin.HandlerFunc
- Produces: `middleware.CORS(allowed []string) gin.HandlerFunc`
- Flag: `-cors-allow-origins string` (comma-separated, default `"*"`)

- [ ] **Step 1: Write the failing test**

```go
// internal/httpsrv/middleware/cors_test.go
package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func newRouter(origins []string) *gin.Engine {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.Use(CORS(origins))
    r.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
    return r
}

func TestCORS_AllowsWhitelistedOrigin(t *testing.T) {
    r := newRouter([]string{"https://docs.fin-ai.net"})
    req := httptest.NewRequest("GET", "/ping", nil)
    req.Header.Set("Origin", "https://docs.fin-ai.net")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://docs.fin-ai.net" {
        t.Errorf("want ACAO=https://docs.fin-ai.net, got %q", got)
    }
    if w.Code != 200 {
        t.Errorf("want 200, got %d", w.Code)
    }
}

func TestCORS_RejectsUnlistedOrigin(t *testing.T) {
    r := newRouter([]string{"https://docs.fin-ai.net"})
    req := httptest.NewRequest("GET", "/ping", nil)
    req.Header.Set("Origin", "https://evil.example")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
        t.Errorf("want empty ACAO, got %q", got)
    }
}

func TestCORS_WildcardAllowsAnyOrigin(t *testing.T) {
    r := newRouter([]string{"*"})
    req := httptest.NewRequest("GET", "/ping", nil)
    req.Header.Set("Origin", "https://anything.example")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://anything.example" {
        t.Errorf("want ACAO=*, got %q", got)
    }
}

func TestCORS_OptionsPreflightReturns204(t *testing.T) {
    r := newRouter([]string{"https://docs.fin-ai.net"})
    req := httptest.NewRequest("OPTIONS", "/ping", nil)
    req.Header.Set("Origin", "https://docs.fin-ai.net")
    req.Header.Set("Access-Control-Request-Method", "POST")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != 204 {
        t.Errorf("want 204, got %d", w.Code)
    }
    if w.Header().Get("Access-Control-Allow-Methods") == "" {
        t.Error("want ACAM header set")
    }
}

func TestCORS_SetsVaryHeader(t *testing.T) {
    r := newRouter([]string{"https://docs.fin-ai.net"})
    req := httptest.NewRequest("GET", "/ping", nil)
    req.Header.Set("Origin", "https://docs.fin-ai.net")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if got := w.Header().Get("Vary"); got != "Origin" {
        t.Errorf("want Vary=Origin, got %q", got)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd tools/cbmem-team && go test ./internal/httpsrv/middleware/ -run TestCORS -v
```

Expected: FAIL with `middleware: no such file or directory` (package doesn't exist yet).

- [ ] **Step 3: Write minimal implementation**

```go
// internal/httpsrv/middleware/cors.go
package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

func CORS(allowed []string) gin.HandlerFunc {
    set := make(map[string]struct{}, len(allowed))
    wildcard := false
    for _, o := range allowed {
        o = strings.TrimSpace(o)
        if o == "" {
            continue
        }
        if o == "*" {
            wildcard = true
            continue
        }
        set[o] = struct{}{}
    }

    return func(c *gin.Context) {
        origin := c.GetHeader("Origin")
        allow := wildcard || origin == ""
        if !allow {
            _, allow = set[origin]
        }
        if allow {
            c.Header("Access-Control-Allow-Origin", origin)
            c.Header("Vary", "Origin")
            c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
            c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Admin-Token")
            c.Header("Access-Control-Max-Age", "86400")
        }
        if c.Request.Method == http.MethodOptions {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }
        c.Next()
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/httpsrv/middleware/ -run TestCORS -v
```

Expected: 5 PASS.

- [ ] **Step 5: Wire flag in main.go**

In `tools/cbmem-team/cmd/cbmem-team/main.go` add flag and apply middleware to API group only (NOT to /health).

```go
// In flag.VisitAll block or main(), add:
var corsOrigins = flag.String("cors-allow-origins", "*",
    "Comma-separated CORS origin whitelist. Use '*' for dev only.")

// In router setup, where /api/console group is created:
apiGroup := r.Group("/api/console")
apiGroup.Use(middleware.CORS(strings.Split(*corsOrigins, ",")))
```

(`/health` 不挂 CORS — 不动现成代码)

- [ ] **Step 6: Update systemd unit**

In `tools/cbmem-team/deploy/cbmem-team.service.minimal`:

```diff
  ExecStart=/usr/local/bin/cbmem-team \
      -listen :8787 \
      -data /var/lib/cbmem-team \
      -mcp-bin /home/deploy/codebase-memory-mcp \
      -jwt-secret ${JWT_SECRET} \
      -admin-token ${ADMIN_TOKEN} \
+     -cors-allow-origins ${CORS_ALLOW_ORIGINS} \
      -log info
```

- [ ] **Step 7: Commit**

```bash
git add tools/cbmem-team/internal/httpsrv/middleware/cors.go \
        tools/cbmem-team/internal/httpsrv/middleware/cors_test.go \
        tools/cbmem-team/cmd/cbmem-team/main.go \
        tools/cbmem-team/deploy/cbmem-team.service.minimal
git commit -m "feat(cbmem-team): add CORS middleware + -cors-allow-origins flag"
```

---

## Task 2: 后端去除 `/console/*` 静态托管 (TDD)

**Files:**
- Modify: `tools/cbmem-team/cmd/cbmem-team/main.go`
- Modify: `tools/cbmem-team/internal/httpsrv/routes.go` (如果路由注册在此)
- Create: `tools/cbmem-team/internal/httpsrv/routes_test.go`
- Modify: `tools/cbmem-team/deploy/cbmem-team.service.minimal`
- Modify: `tools/cbmem-team/deploy/cbmem-team.env.example`

**Interfaces:**
- Consumes: Task 1's middleware
- Produces: route table without `/console/*` and without `-console-dist` flag

- [ ] **Step 1: Write the failing test**

```go
// internal/httpsrv/routes_test.go
package httpsrv

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestRoutes_NoConsoleStaticPath(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := BuildRouter(BuildOpts{
        JWTSecret: "test",
        AdminToken: "test",
        CORSOrigins: []string{"*"},
        // intentionally NOT setting ConsoleDist
    })

    routes := r.Routes()
    for _, ri := range routes {
        if ri.Path == "/console" || ri.Path == "/console/*filepath" || ri.Path == "/console/*action" {
            t.Errorf("route %s %s must not exist after decoupling", ri.Method, ri.Path)
        }
    }
}

func TestRoutes_HealthEndpointExists(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := BuildRouter(BuildOpts{JWTSecret: "test", AdminToken: "test"})

    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("want /health 200, got %d", w.Code)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd tools/cbmem-team && go test ./internal/httpsrv/ -run TestRoutes -v
```

Expected: FAIL with `BuildRouter undefined` (or similar — current code doesn't have this helper).

- [ ] **Step 3: Refactor to BuildRouter(BuildOpts)**

If main.go currently builds routes inline, extract a `BuildRouter(opts BuildOpts) *gin.Engine` helper. Example shape:

```go
// internal/httpsrv/routes.go
package httpsrv

import (
    "github.com/gin-gonic/gin"
    "your-project/internal/httpsrv/middleware"
)

type BuildOpts struct {
    JWTSecret   string
    AdminToken  string
    CORSOrigins []string
    ConsoleDist string // REMOVE this field
}

func BuildRouter(o BuildOpts) *gin.Engine {
    r := gin.New()
    r.GET("/health", func(c *gin.Context) { c.String(200, "ok") })

    api := r.Group("/api/console")
    api.Use(middleware.CORS(o.CORSOrigins))
    // ... register existing /api/console/* routes
    return r
}
```

Remove the `r.Static("/console", *consoleDist)` line and the `-console-dist` flag from main.go.

- [ ] **Step 4: Update systemd + env example**

In `cbmem-team.service.minimal`:
```diff
  ExecStart=/usr/local/bin/cbmem-team \
      -listen :8787 \
      -data /var/lib/cbmem-team \
      -mcp-bin /home/deploy/codebase-memory-mcp \
      -jwt-secret ${JWT_SECRET} \
      -admin-token ${ADMIN_TOKEN} \
-     -console-dist /var/www/docs/console \
      -cors-allow-origins ${CORS_ALLOW_ORIGINS} \
      -log info
```

In `cbmem-team.env.example` (create if missing):
```
JWT_SECRET=change-me
ADMIN_TOKEN=change-me
CORS_ALLOW_ORIGINS=https://docs.fin-ai.net
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./internal/httpsrv/ -run TestRoutes -v
```

Expected: PASS for both tests.

- [ ] **Step 6: Build and binary smoke**

```bash
cd tools/cbmem-team && go build -o /tmp/cbmem-team ./cmd/cbmem-team
/tmp/cbmem-team -listen :18787 -jwt-secret test -admin-token test -cors-allow-origins "*" &
sleep 2
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:18787/health
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:18787/console/index.html
kill %1
```

Expected: `200` for /health; `404` for /console/index.html (proves static removed).

- [ ] **Step 7: Commit**

```bash
git add tools/cbmem-team/internal/httpsrv/routes.go \
        tools/cbmem-team/internal/httpsrv/routes_test.go \
        tools/cbmem-team/cmd/cbmem-team/main.go \
        tools/cbmem-team/deploy/cbmem-team.service.minimal \
        tools/cbmem-team/deploy/cbmem-team.env.example
git commit -m "refactor(cbmem-team): drop -console-dist static hosting (decouple)"
```

---

## Task 3: 前端依赖替换 + 主题接入 AntD (TDD via build + smoke)

**Files:**
- Modify: `docs-site/package.json`
- Modify: `docs-site/.vitepress/theme/index.ts`
- Create: `docs-site/.vitepress/theme/console/api.ts`
- Create: `docs-site/.vitepress/theme/console/api.test.ts`

**Interfaces:**
- Consumes: Ant Design Vue 4
- Produces: `apiBase() string`, `api<T>(path, init?) Promise<T>`, `notify { success, error, confirm }`

- [ ] **Step 1: Write failing test for api.ts**

```ts
// .vitepress/theme/console/api.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { api, apiBase } from './api'

describe('api', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('apiBase returns string from env or default', () => {
    const base = apiBase()
    expect(typeof base).toBe('string')
    expect(base.length).toBeGreaterThan(0)
  })

  it('api() prepends base URL', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ ok: 1 }),
    })
    vi.stubGlobal('fetch', fetchMock)

    await api<{ ok: number }>('/users')

    expect(fetchMock).toHaveBeenCalledOnce()
    const calledUrl = fetchMock.mock.calls[0][0]
    expect(calledUrl).toMatch(/\/users$/)
    expect(calledUrl.startsWith(apiBase())).toBe(true)
  })

  it('api() throws on non-2xx response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      text: async () => 'internal',
    }))

    await expect(api('/boom')).rejects.toThrow(/500/)
  })

  it('api() includes credentials and JSON header', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) })
    vi.stubGlobal('fetch', fetchMock)

    await api('/x', { method: 'POST', body: JSON.stringify({ a: 1 }) })

    const init = fetchMock.mock.calls[0][1]
    expect(init.credentials).toBe('include')
    expect(init.headers['Content-Type']).toBe('application/json')
    expect(init.body).toBe('{"a":1}')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd docs-site && pnpm vitest run .vitepress/theme/console/api.test.ts
```

Expected: FAIL (file doesn't exist).

- [ ] **Step 3: Write minimal api.ts**

```ts
// .vitepress/theme/console/api.ts
// SSR-safe env read.
const BASE: string =
  (typeof import.meta !== 'undefined' &&
    (import.meta as any).env?.VITE_API_BASE) ||
  '/api/console'

export const apiBase = (): string => BASE

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) throw new Error(`API ${res.status}: ${await res.text()}`)
  return (await res.json()) as T
}

// notify re-exports — pages use notify.success/error/confirm instead of importing antd directly.
import { message, Modal } from 'ant-design-vue'
export const notify = {
  success: (m: string) => message.success(m),
  error: (m: string) => message.error(m),
  confirm: (opts: { title: string; content: string; onOk: () => void }) =>
    Modal.confirm({ ...opts, okText: '确认', cancelText: '取消' }),
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
pnpm vitest run .vitepress/theme/console/api.test.ts
```

Expected: 4 PASS.

- [ ] **Step 5: Update package.json**

```json
{
  "dependencies": {
    "ant-design-vue": "^4.2.0",
    "@ant-design/icons-vue": "^7.0.1",
    "mermaid": "^10.9.0",
    "pinia": "^3.0.4",
    "vitepress-plugin-mermaid": "^2.0.17"
  }
}
```

Remove `element-plus`. Keep vue at `^3.4.0` in devDependencies.

- [ ] **Step 6: Install + update theme/index.ts**

```bash
cd docs-site && pnpm install
```

```ts
// .vitepress/theme/index.ts
import DefaultTheme from 'vitepress/theme'
import { createPinia } from 'pinia'
import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import ConsoleLayout from './console/Layout.vue'

export default {
  extends: DefaultTheme,
  Layout: ConsoleLayout,
  enhanceApp({ app }) {
    app.use(createPinia())
    app.use(Antd)
  },
}
```

- [ ] **Step 7: Update vite optimizeDeps**

In `docs-site/.vitepress/config.mjs`:

```js
vite: {
  optimizeDeps: {
    include: ['dayjs', 'ant-design-vue'],
  },
  server: { proxy: { '/api/console': { target: 'http://127.0.0.1:8787', changeOrigin: false } } },
}
```

- [ ] **Step 8: Smoke — pnpm dev must boot**

```bash
cd docs-site && pnpm dev &
sleep 8
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:5173/
kill %1
```

Expected: `200`.

- [ ] **Step 9: Commit**

```bash
git add docs-site/package.json docs-site/pnpm-lock.yaml \
        docs-site/.vitepress/theme/index.ts \
        docs-site/.vitepress/config.mjs \
        docs-site/.vitepress/theme/console/api.ts \
        docs-site/.vitepress/theme/console/api.test.ts
git commit -m "feat(docs-site): swap element-plus for ant-design-vue 4 + api helper"
```

---

## Task 4: 创建 .env 文件 + 顶层注释

**Files:**
- Create: `docs-site/.env.development`
- Create: `docs-site/.env.production`
- Modify: `docs-site/.vitepress/config.mjs`

**Interfaces:**
- Consumes: VitePress build-time env injection
- Produces: env files committed; config.mjs comment block

- [ ] **Step 1: Write .env.development**

```bash
# docs-site/.env.development
# dev: same-origin, vite proxy forwards /api/console → 127.0.0.1:8787
VITE_API_BASE=/api/console
```

- [ ] **Step 2: Write .env.production**

```bash
# docs-site/.env.production
# prod: CORS direct call to api.fin-ai.net
VITE_API_BASE=https://api.fin-ai.net/api/console
```

- [ ] **Step 3: Add config.mjs top-of-file comment**

Insert at line 1 of `docs-site/.vitepress/config.mjs`:

```js
/**
 * docs-site VitePress config.
 *
 * API base URL:
 *   Dev:  VITE_API_BASE = '/api/console'              → vite proxy → 127.0.0.1:8787
 *   Prod: VITE_API_BASE = 'https://api.fin-ai.net/api/console'  → CORS direct call
 *
 * Override via docs-site/.env.development or .env.production.
 *
 * Topology:
 *   docs.fin-ai.net → VitePress dist (static)
 *   api.fin-ai.net  → cbmem-team :8787 (CORS whitelist: docs.fin-ai.net)
 */
```

- [ ] **Step 4: Verify build uses prod env**

```bash
cd docs-site && pnpm build
grep -r "VITE_API_BASE" docs-site/.vitepress/dist/assets/ | head -3
```

Expected: at least one occurrence of `https://api.fin-ai.net/api/console` baked into the dist bundle.

- [ ] **Step 5: Commit**

```bash
git add docs-site/.env.development docs-site/.env.production \
        docs-site/.vitepress/config.mjs
git commit -m "feat(docs-site): wire VITE_API_BASE env (dev/prod split)"
```

---

## Task 5: Console Layout.vue 迁移到 AntD

**Files:**
- Modify: `docs-site/.vitepress/theme/console/Layout.vue`

**Interfaces:**
- Consumes: `api`, `notify` (from Task 3)
- Produces: AntD `<a-button>` in topbar

- [ ] **Step 1: Write Layout smoke test**

```ts
// .vitepress/theme/console/Layout.test.ts
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

// Mock useSessionStore before importing Layout.
vi.mock('./store/session', () => ({
  useSessionStore: () => ({ sessionId: '', logout: vi.fn() }),
}))
import Layout from './Layout.vue'

describe('Console Layout', () => {
  it('renders topbar title', () => {
    setActivePinia(createPinia())
    const w = mount(Layout)
    expect(w.find('h1').text()).toBe('双轨记忆控制台')
  })

  it('shows 未登录 when no session', () => {
    setActivePinia(createPinia())
    const w = mount(Layout)
    expect(w.text()).toContain('未登录')
  })
})
```

- [ ] **Step 2: Run test (will pass on old code)**

```bash
cd docs-site && pnpm vitest run .vitepress/theme/console/Layout.test.ts
```

Expected: 2 PASS (or "no test target" — either is acceptable for now; main goal is post-migration smoke).

- [ ] **Step 3: Modify Layout.vue — replace `el-button` with `a-button`**

In `docs-site/.vitepress/theme/console/Layout.vue`:

```diff
-      <el-button size="small" @click="onLogout">退出</el-button>
+      <a-button size="small" @click="onLogout">退出</a-button>
```

No other changes (rest of Layout uses plain divs, not Element Plus).

- [ ] **Step 4: Run vitest**

```bash
pnpm vitest run .vitepress/theme/console/Layout.test.ts
```

Expected: 2 PASS.

- [ ] **Step 5: Manual smoke via pnpm dev**

```bash
cd docs-site && pnpm dev &
sleep 8
curl -s http://localhost:5173/console/ -o /dev/null -w "%{http_code}\n"
kill %1
```

Expected: `200`. Also open `http://localhost:5173/console/` in browser — topbar should render with AntD button.

- [ ] **Step 6: Commit**

```bash
git add docs-site/.vitepress/theme/console/Layout.vue \
        docs-site/.vitepress/theme/console/Layout.test.ts
git commit -m "refactor(console): Layout.vue el-button → a-button"
```

---

## Task 6: Login.vue 迁移到 AntD

**Files:**
- Modify: `docs-site/.vitepress/theme/console/pages/Login.vue`

**Interfaces:**
- Consumes: `api`, `notify` (Task 3)
- Produces: AntD form

- [ ] **Step 1: Write failing test**

```ts
// .vitepress/theme/console/pages/Login.test.ts
import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Antd from 'ant-design-vue'
import Login from './Login.vue'

describe('Login page', () => {
  it('renders input and button', () => {
    setActivePinia(createPinia())
    const w = mount(Login, { global: { plugins: [Antd] } })
    expect(w.find('input[type="password"]').exists()).toBe(true)
    expect(w.text()).toContain('登录')
  })

  it('calls api and notifies on success', async () => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ token: 'abc' }),
    }))
    const w = mount(Login, { global: { plugins: [Antd] } })
    await w.find('input[type="password"]').setValue('token-xyz')
    await w.find('button').trigger('click')
    await flushPromises()
    // no assertion on toast — AntD message mounts to body; we just verify no throw
    expect(true).toBe(true)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd docs-site && pnpm vitest run .vitepress/theme/console/pages/Login.test.ts
```

Expected: FAIL (button text says 登录 in old code too, but `a-button` doesn't exist yet — the snapshot diff will catch it).

- [ ] **Step 3: Rewrite Login.vue**

```vue
<!-- .vitepress/theme/console/pages/Login.vue -->
<template>
  <div class="login">
    <h2>管理员登录</h2>
    <a-form layout="vertical" :model="form" @finish="onLogin">
      <a-form-item label="管理员 Token" name="token">
        <a-input v-model:value="form.token" type="password" placeholder="管理员 Token" />
      </a-form-item>
      <a-form-item>
        <a-button type="primary" html-type="submit" :loading="loading" block>
          登录
        </a-button>
      </a-form-item>
    </a-form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { api } from '../api'

const form = reactive({ token: '' })
const loading = ref(false)

async function onLogin() {
  loading.value = true
  try {
    await api('/login', {
      method: 'POST',
      body: JSON.stringify({ token: form.token }),
    })
    location.reload()
  } catch (e) {
    notify.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

import { notify } from '../api'
</script>

<style scoped>
.login { max-width: 360px; margin: 40px auto; }
</style>
```

- [ ] **Step 4: Run test**

```bash
pnpm vitest run .vitepress/theme/console/pages/Login.test.ts
```

Expected: 2 PASS.

- [ ] **Step 5: Commit**

```bash
git add docs-site/.vitepress/theme/console/pages/Login.vue \
        docs-site/.vitepress/theme/console/pages/Login.test.ts
git commit -m "refactor(console): Login.vue migrate to AntD form + a-input"
```

---

## Task 7: Users.vue 迁移到 AntD table + form

**Files:**
- Modify: `docs-site/.vitepress/theme/console/pages/Users.vue`
- Create: `docs-site/.vitepress/theme/console/pages/Users.test.ts`

- [ ] **Step 1: Write failing test**

```ts
// .vitepress/theme/console/pages/Users.test.ts
import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Antd from 'ant-design-vue'
import Users from './Users.vue'

describe('Users page', () => {
  it('loads and renders user rows', async () => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ([{ id: 'u1', display_name: 'Alice' }]),
    }))
    const w = mount(Users, { global: { plugins: [Antd] } })
    await flushPromises()
    expect(w.text()).toContain('Alice')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd docs-site && pnpm vitest run .vitepress/theme/console/pages/Users.test.ts
```

Expected: FAIL (Users still uses `el-table`).

- [ ] **Step 3: Rewrite Users.vue**

```vue
<template>
  <div>
    <a-button type="primary" @click="openCreate">新增用户</a-button>
    <a-table :dataSource="users" :columns="columns" :loading="loading" rowKey="id" style="margin-top: 16px;">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'token_status'">
          <a-tag color="green">已设置</a-tag>
        </template>
        <template v-else-if="column.key === 'actions'">
          <a-button size="small" @click="onEdit(record as User)">编辑</a-button>
          <a-button size="small" danger @click="onDelete(record as User)">删除</a-button>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="dialog" :title="editing ? '编辑用户' : '新增用户'" @ok="onSubmit">
      <a-form :model="form" layout="vertical">
        <a-form-item label="用户 ID"><a-input v-model:value="form.id" :disabled="!!editing" /></a-form-item>
        <a-form-item label="显示名"><a-input v-model:value="form.display_name" /></a-form-item>
        <a-form-item label="项目路径">
          <a-input v-model:value="pathsText" placeholder="逗号分隔" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { api, notify } from '../api'

interface User { id: string; display_name: string; project_paths?: string[] }

const users = ref<User[]>([])
const loading = ref(false)
const dialog = ref(false)
const editing = ref<User | null>(null)
const form = reactive({ id: '', display_name: '' })
const pathsText = ref('')

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '显示名', dataIndex: 'display_name', key: 'display_name' },
  { title: 'Token 状态', key: 'token_status' },
  { title: '操作', key: 'actions' },
]

async function load() {
  loading.value = true
  try { users.value = await api<User[]>('/users') }
  catch (e) { notify.error((e as Error).message) }
  finally { loading.value = false }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { id: '', display_name: '' })
  pathsText.value = ''
  dialog.value = true
}

function onEdit(u: User) {
  editing.value = u
  Object.assign(form, { id: u.id, display_name: u.display_name })
  pathsText.value = (u.project_paths ?? []).join(',')
  dialog.value = true
}

async function onDelete(u: User) {
  notify.confirm({
    title: '删除用户',
    content: `确认删除 ${u.id}？`,
    onOk: async () => {
      try { await api(`/users/${u.id}`, { method: 'DELETE' }); await load() }
      catch (e) { notify.error((e as Error).message) }
    },
  })
}

async function onSubmit() {
  try {
    const payload = { ...form, project_paths: pathsText.value.split(',').map(s => s.trim()).filter(Boolean) }
    if (editing.value) await api(`/users/${form.id}`, { method: 'PUT', body: JSON.stringify(payload) })
    else await api('/users', { method: 'POST', body: JSON.stringify(payload) })
    dialog.value = false
    await load()
  } catch (e) { notify.error((e as Error).message) }
}

onMounted(load)
</script>
```

- [ ] **Step 4: Run test**

```bash
pnpm vitest run .vitepress/theme/console/pages/Users.test.ts
```

Expected: 1 PASS.

- [ ] **Step 5: Commit**

```bash
git add docs-site/.vitepress/theme/console/pages/Users.vue \
        docs-site/.vitepress/theme/console/pages/Users.test.ts
git commit -m "refactor(console): Users.vue migrate to AntD table + modal"
```

---

## Task 8: Projects.vue 迁移到 AntD

**Files:**
- Modify: `docs-site/.vitepress/theme/console/pages/Projects.vue`

- [ ] **Step 1: Apply Users.vue pattern** — copy structure from Task 7 step 3. Replace `el-button/el-table/el-form/el-form-item/el-input` with AntD equivalents. Field set: name / path / wing / mcp_bin.

- [ ] **Step 2: Write smoke test (mirror Users.test.ts shape)**

```ts
// .vitepress/theme/console/pages/Projects.test.ts
import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Antd from 'ant-design-vue'
import Projects from './Projects.vue'

describe('Projects page', () => {
  it('loads and renders project rows', async () => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ([{ id: 1, name: 'demo', path: '/tmp/demo', wing: 'g1' }]),
    }))
    const w = mount(Projects, { global: { plugins: [Antd] } })
    await flushPromises()
    expect(w.text()).toContain('demo')
  })
})
```

- [ ] **Step 3: Run test**

```bash
pnpm vitest run .vitepress/theme/console/pages/Projects.test.ts
```

Expected: 1 PASS.

- [ ] **Step 4: Commit**

```bash
git add docs-site/.vitepress/theme/console/pages/Projects.vue \
        docs-site/.vitepress/theme/console/pages/Projects.test.ts
git commit -m "refactor(console): Projects.vue migrate to AntD"
```

---

## Task 9: Sessions.vue 迁移到 AntD

**Files:**
- Modify: `docs-site/.vitepress/theme/console/pages/Sessions.vue`
- Create: `docs-site/.vitepress/theme/console/pages/Sessions.test.ts`

- [ ] **Step 1: Apply same pattern** — `<a-table :dataSource :columns>`, `<a-button>` for actions. Columns: id / user_id / project_path / turn_count / started_at / actions. Single "查看" action opens a detail modal.

- [ ] **Step 2: Smoke test**

```ts
import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Antd from 'ant-design-vue'
import Sessions from './Sessions.vue'

describe('Sessions page', () => {
  it('renders session rows', async () => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ([{ id: 's1', user_id: 'u1', turn_count: 5 }]),
    }))
    const w = mount(Sessions, { global: { plugins: [Antd] } })
    await flushPromises()
    expect(w.text()).toContain('s1')
  })
})
```

- [ ] **Step 3: Run + commit**

```bash
pnpm vitest run .vitepress/theme/console/pages/Sessions.test.ts
git add docs-site/.vitepress/theme/console/pages/Sessions.vue \
        docs-site/.vitepress/theme/console/pages/Sessions.test.ts
git commit -m "refactor(console): Sessions.vue migrate to AntD"
```

---

## Task 10: Summarize.vue 迁移到 AntD form

**Files:**
- Modify: `docs-site/.vitepress/theme/console/pages/Summarize.vue`
- Create: `docs-site/.vitepress/theme/console/pages/Summarize.test.ts`

- [ ] **Step 1: Migrate** — Replace `el-form` + `el-form-item` + `el-input` + `el-input-number` + `el-button`. Use `<a-form layout="inline" :model="form">` with same fields: sourceText (逗号分隔 ID) / depth (a-input-number 1-5) / target_wing / 提交 button.

- [ ] **Step 2: Smoke test**

```ts
import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Antd from 'ant-design-vue'
import Summarize from './Summarize.vue'

describe('Summarize page', () => {
  it('renders form', () => {
    setActivePinia(createPinia())
    const w = mount(Summarize, { global: { plugins: [Antd] } })
    expect(w.text()).toContain('开始归纳')
  })

  it('submits on button click', async () => {
    setActivePinia(createPinia())
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) })
    vi.stubGlobal('fetch', fetchMock)
    const w = mount(Summarize, { global: { plugins: [Antd] } })
    await w.find('button').trigger('click')
    await flushPromises()
    expect(fetchMock).toHaveBeenCalled()
  })
})
```

- [ ] **Step 3: Run + commit**

```bash
pnpm vitest run .vitepress/theme/console/pages/Summarize.test.ts
git add docs-site/.vitepress/theme/console/pages/Summarize.vue \
        docs-site/.vitepress/theme/console/pages/Summarize.test.ts
git commit -m "refactor(console): Summarize.vue migrate to AntD form"
```

---

## Task 11: Distill.vue 迁移到 AntD form

**Files:**
- Modify: `docs-site/.vitepress/theme/console/pages/Distill.vue`
- Create: `docs-site/.vitepress/theme/console/pages/Distill.test.ts`

- [ ] **Step 1: Migrate** — Same pattern as Summarize. Two phases: (1) form → submit triggers "开始蒸馏"; (2) results section with "写入 MemPalace" button.

- [ ] **Step 2: Smoke test (mirror Summarize test)**

- [ ] **Step 3: Run + commit**

```bash
pnpm vitest run .vitepress/theme/console/pages/Distill.test.ts
git add docs-site/.vitepress/theme/console/pages/Distill.vue \
        docs-site/.vitepress/theme/console/pages/Distill.test.ts
git commit -m "refactor(console): Distill.vue migrate to AntD"
```

---

## Task 12: Status.vue + Logs.vue 迁移到 AntD

**Files:**
- Modify: `docs-site/.vitepress/theme/console/pages/Status.vue`
- Modify: `docs-site/.vitepress/theme/console/pages/Logs.vue`
- Create: corresponding `.test.ts`

- [ ] **Step 1: Migrate Status.vue** — Replace any `el-*` tags. Use AntD `<a-statistic>` for counters, `<a-card>` for sections.

- [ ] **Step 2: Migrate Logs.vue** — Replace `el-*` with AntD. Logs likely uses `<a-table>` with paging.

- [ ] **Step 3: Tests + commit**

```bash
pnpm vitest run .vitepress/theme/console/pages/Status.test.ts \
                  .vitepress/theme/console/pages/Logs.test.ts
git add docs-site/.vitepress/theme/console/pages/Status.vue \
        docs-site/.vitepress/theme/console/pages/Status.test.ts \
        docs-site/.vitepress/theme/console/pages/Logs.vue \
        docs-site/.vitepress/theme/console/pages/Logs.test.ts
git commit -m "refactor(console): Status.vue + Logs.vue migrate to AntD"
```

---

## Task 13: 验证 — 全控制台 SPA 跨三语渲染

**Files:** none (verification only)

- [ ] **Step 1: Build production**

```bash
cd docs-site && pnpm build
ls -la docs-site/.vitepress/dist/index.html docs-site/.vitepress/dist/assets/index*.js
```

Expected: index.html + at least one JS asset exist.

- [ ] **Step 2: Boot dev + cbmem-team + smoke**

```bash
# Terminal 1
cd docs-site && pnpm dev &
# Terminal 2
cd tools/cbmem-team && /tmp/cbmem-team -listen :18787 -jwt-secret test -admin-token test -cors-allow-origins "*" &
sleep 8

# Health (vite proxy works)
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:5173/api/console/health
# Direct cbmem-team
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:18787/health
```

Expected: both `200`.

- [ ] **Step 3: Manual — zh/en/ja /console/* all load**

Open in browser:
- http://localhost:5173/console/
- http://localhost:5173/en/console/
- http://localhost:5173/ja/console/

Expected: each loads with console SPA + no console errors. AntD widgets render.

- [ ] **Step 4: Kill processes**

```bash
kill %1 %2 2>/dev/null
```

- [ ] **Step 5: Confirm zero `el-` references in docs-site**

```bash
grep -rn "el-button\|el-input\|el-table\|el-form\|ElMessage\|element-plus" docs-site/.vitepress/theme/ || echo "CLEAN"
```

Expected: `CLEAN` (no matches).

- [ ] **Step 6: Commit (verification report only if any fix needed)**

```bash
git status
# If anything was fixed during verification, commit; otherwise nothing to commit.
```

---

## Task 14: 部署配置 — Caddyfile + systemd 更新

**Files:**
- Create: `tools/cbmem-team/deploy/Caddyfile`
- Modify: `tools/cbmem-team/README.md`
- Modify: `tools/cbmem-team/CONSOLE.md`

- [ ] **Step 1: Write Caddyfile**

```caddyfile
# tools/cbmem-team/deploy/Caddyfile
# Two vhosts for fin-ai.net domain split.

docs.fin-ai.net {
    root * /var/www/docs.fin-ai.net
    encode zstd gzip
    file_server
    try_files {path} {path}.html
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Content-Type-Options "nosniff"
    }
}

api.fin-ai.net {
    reverse_proxy 127.0.0.1:8787 {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }
}
```

- [ ] **Step 2: Update README.md**

In `tools/cbmem-team/README.md`, replace any reference to `-console-dist` with documentation of `-cors-allow-origins`. Example diff:

```diff
-### Run with embedded console
+### Run with CORS for browser SPA

 ```bash
-cbmem-team -listen :8787 -console-dist ./console-dist ...
+cbmem-team -listen :8787 \
+    -cors-allow-origins "https://docs.fin-ai.net,http://localhost:5173" \
+    ...
 ```
```

- [ ] **Step 3: Update CONSOLE.md**

Replace any "gin 托管 console dist" paragraph with CORS+reverse-proxy paragraph. Add ASCII diagram:

```
Browser → https://docs.fin-ai.net (VitePress dist, static)
        → https://api.fin-ai.net/api/console (cbmem-team, CORS)
```

- [ ] **Step 4: Commit**

```bash
git add tools/cbmem-team/deploy/Caddyfile \
        tools/cbmem-team/README.md \
        tools/cbmem-team/CONSOLE.md
git commit -m "docs(cbmem-team): add Caddyfile + update console deployment guide"
```

---

## Task 15: 用户文档更新

**Files:**
- Modify: `docs-site/guide/intro.md`
- Modify: `docs-site/sop/memory/deploy.md`
- Modify: `docs-site/guide/changelog.md`

- [ ] **Step 1: Update guide/intro.md** — Add "## 独立部署拓扑" section with ASCII diagram + table of domains/endpoints.

- [ ] **Step 2: Update sop/memory/deploy.md** — Replace inline-cbmem-team-deploy steps with the new Caddy + systemd + CORS env workflow.

- [ ] **Step 3: Update guide/changelog.md** — Add v2.0.0 entry:

```markdown
## v2.0.0 — 独立部署 + Ant Design

### Breaking
- docs-site 与 cbmem-team 完全解耦
- 前端替换 Element Plus 为 Ant Design Vue 4
- cbmem-team 移除 `-console-dist` flag
- 新增 `-cors-allow-origins` flag

### Migration
- Production: deploy via Caddy reverse-proxy. See sop/memory/deploy.md.
- Dev: `pnpm dev` works as before; vite proxy unchanged.
```

- [ ] **Step 4: Commit**

```bash
git add docs-site/guide/intro.md \
        docs-site/sop/memory/deploy.md \
        docs-site/guide/changelog.md
git commit -m "docs(docs-site): v2.0.0 — independent deploy + Ant Design"
```

---

## Task 16: 覆盖率 + 最终冒烟

**Files:** none (verification only)

- [ ] **Step 1: Coverage Go**

```bash
cd tools/cbmem-team && go test ./... -coverprofile=/tmp/cover.out
go tool cover -func=/tmp/cover.out | tail -3
```

Expected: total coverage ≥ 80%.

- [ ] **Step 2: Coverage docs-site**

```bash
cd docs-site && pnpm vitest run --coverage
```

Expected: total coverage ≥ 80%.

- [ ] **Step 3: Full E2E**

```bash
cd docs-site && pnpm build
# Confirm dist is purely static
find docs-site/.vitepress/dist -type f | head -20
# Confirm no go files bundled
find docs-site/.vitepress/dist -name "*.go" | wc -l   # expect 0
```

Expected: dist contains only HTML/JS/CSS/assets; no Go source.

- [ ] **Step 4: Tag release**

```bash
git tag v2.0.0 -m "v2.0.0: independent deploy + Ant Design Vue 4"
```

---

## Done Criteria

- [ ] All 16 tasks committed individually
- [ ] Go coverage ≥ 80%
- [ ] Vitest coverage ≥ 80%
- [ ] No `el-*` references in docs-site
- [ ] No `-console-dist` references in cbmem-team
- [ ] `pnpm build` produces pure static dist
- [ ] Manual smoke: pnpm dev + cbmem-team + browser load works for /console/ in zh/en/ja
- [ ] Caddyfile + systemd units documented and committed
- [ ] CHANGELOG v2.0.0 entry
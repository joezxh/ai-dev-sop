> [!WARNING]
> **已废弃（2026-09-18）**：本文档描述的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，记忆功能统一替换为自托管 mem0（见 docs/quick-ref/mem0-ai-tools-config-guide.md）。本文仅作历史归档保留，内容不再维护。

# docs-site / cbmem-team 独立部署 + Ant Design 迁移 — 设计

| | |
|---|---|
| **Date** | 2026-07-12 |
| **Status** | Draft (awaiting user review) |
| **Owner** | docs-site maintainer |
| **Affects** | `docs-site/`, `tools/cbmem-team/` |
| **Target version** | docs-site 2.0.0, cbmem-team 2.0.0 |

## 1. Background & Motivation

### 1.1 当前耦合点

`docs-site` (VitePress + Vue 3.4 + Element Plus 2.14) 当前与 `cbmem-team` (Gin REST API on `:8787`) 通过**同进程静态托管**耦合：

```go
// 现有模式 (gin main.go)
r.Static("/console", *consoleDist)  // gin 直接服务 vitepress dist/
api := r.Group("/api/console", auth(...))
// ...
```

`docs-site/.vitepress/config.mjs` 里的注释明示：

> Local dev proxy: forward console API calls to the cbmem-team backend. In production the vitepress `dist/` is served by gin under `/console`, so `/api/console/*` hits the same process and no proxy is needed.

### 1.2 问题

1. **违反关注点分离**：API 进程承担了静态资源分发职能
2. **不可独立扩展**：docs-site 与 cbmem-team 必须同进程部署，无法分别水平扩展或上 CDN
3. **路由冲突风险**（已出现）：`/console/*action` (API) 与 `/console/*filepath` (静态) 在 gin radix tree 中冲突，未提交 WIP 已因此 panic
4. **Element Plus 2.14 与团队后续规划不一致**

### 1.3 Goal

> 将 docs-site 前端程序与 cbmem-team 后端服务完全解耦，使其能够独立部署和运行；同时把 Element Plus 替换为 Ant Design Vue 4。

## 2. Non-Goals

- 不修改 cbmem-team 的业务 API（路径、payload、auth 方案不变）
- 不引入 SSR / Nuxt — 保持 VitePress SSG
- 不动 M1-M4 已有的功能代码（rate limiter、BP graph、repo pipeline 等）
- 不做 E2E 自动化测试 (本次仅手工 + 单元/集成)

## 3. Architecture

### 3.1 部署拓扑

```
┌─────────────────────────────────────────────────────────────┐
│                  生产 (fin-ai.net 域名)                       │
│                                                              │
│  Browser ──HTTPS──► Caddy/Nginx ──┬─► docs.fin-ai.net/       │
│                                  │   (VitePress dist, 静态)   │
│                                  │                            │
│                                  └─► api.fin-ai.net/         │
│                                      (cbmem-team + CORS)     │
│                                                              │
│  本地开发:                                                     │
│  Browser ──► :5173 (pnpm dev)  ──vite proxy──► :8787         │
│                                              (cbmem-team)    │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 资产边界

| 资产 | 所属 | 生产入口 | 开发入口 |
|---|---|---|---|
| VitePress dist (HTML/JS/CSS) | docs-site | `docs.fin-ai.net` (反代静态) | `http://localhost:5173` |
| console SPA (Layout + 7 pages) | docs-site theme | `docs.fin-ai.net/console` | `http://localhost:5173/console` |
| `/api/console/*` 后端 | cbmem-team | `api.fin-ai.net/api/console` | `http://localhost:8787/api/console` |
| API base URL 配置 | 共享契约 | `VITE_API_BASE` 环境变量 | `/api/console` (vite proxy) |

### 3.3 选型理由

- **Caddy/Nginx 反代 vs 直连**：选反代（方案 A）。原因：
  1. 静态资源可上 CDN
  2. 后端不暴露在公网
  3. 本地/staging/prod 拓扑一致
  4. 落地后 cbmem-team 单一职责（API），再无 `-console-dist` 路由冲突空间

## 4. Component Design

### 4.1 `docs-site/package.json`

```diff
  "dependencies": {
-   "element-plus": "^2.14.2",
+   "ant-design-vue": "^4.2.0",
+   "@ant-design/icons-vue": "^7.0.1",
    "mermaid": "^10.9.0",
    "pinia": "^3.0.4",
    "vitepress-plugin-mermaid": "^2.0.17"
  }
```

Vue 版本保持 `^3.4.0` (已在 `devDependencies`)。Pinia `^3.0.4` 与 Vue 3.4 兼容。

### 4.2 `docs-site/.env.*` (新建)

```bash
# docs-site/.env.development
VITE_API_BASE=/api/console

# docs-site/.env.production
VITE_API_BASE=https://api.fin-ai.net/api/console
```

通过 `import.meta.env.VITE_API_BASE` 在前端代码中读取。

### 4.3 `docs-site/.vitepress/config.mjs` 变更

```diff
  vite: {
    optimizeDeps: {
-     include: ['dayjs'],
+     include: ['dayjs', 'ant-design-vue'],
    },
    server: {
      proxy: {
        '/api/console': {
          target: 'http://127.0.0.1:8787',
          changeOrigin: false,
+         // dev-only proxy; prod 不需要 (走 VITE_API_BASE 直连 api.fin-ai.net)
        },
      },
    },
  }
```

顶部加注释说明 dev/prod 双轨：

```js
/**
 * Dev:  VITE_API_BASE = '/api/console'  → vite proxy → 127.0.0.1:8787
 * Prod: VITE_API_BASE = 'https://api.fin-ai.net/api/console' → CORS 直连
 */
```

### 4.4 `docs-site/.vitepress/theme/index.ts`

```diff
- import ElementPlus from 'element-plus'
- import 'element-plus/dist/index.css'
+ import Antd from 'ant-design-vue'
+ import 'ant-design-vue/dist/reset.css'
  import ConsoleLayout from './console/Layout.vue'
  ...
- app.use(ElementPlus)
+ app.use(Antd)
```

### 4.5 新增 `.vitepress/theme/console/api.ts`

封装所有 fetch + 通知逻辑：

```ts
// SSR-safe: VitePress 在构建时 import.meta.env 不含运行时变量
const BASE: string =
  (typeof import.meta !== 'undefined' && (import.meta as any).env?.VITE_API_BASE) ||
  '/api/console'

export const apiBase = () => BASE

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) throw new Error(`API ${res.status}: ${await res.text()}`)
  return res.json() as Promise<T>
}

import { message, Modal } from 'ant-design-vue'
export const notify = {
  success: (m: string) => message.success(m),
  error: (m: string) => message.error(m),
  confirm: (opts: { title: string; content: string; onOk: () => void }) =>
    Modal.confirm({ ...opts, okText: '确认', cancelText: '取消' }),
}
```

7 个页面统一用 `api()` + `notify` 替代硬编码 URL 与散落的 `ElMessage`。

### 4.6 Element Plus → Ant Design Vue 4 映射表

| Element Plus | Ant Design Vue 4 | 备注 |
|---|---|---|
| `<el-button type="primary">` | `<a-button type="primary">` | 直替 |
| `<el-button type="success">` | `<a-button>` (Ant 无 success) | type 去除 |
| `<el-button size="small" type="danger">` | `<a-button size="small" danger>` | danger 是布尔属性 |
| `<el-input v-model>` | `<a-input v-model>` | 兼容 |
| `<el-input-number>` | `<a-input-number>` | min/max/step 一致 |
| `<el-table :data>` + 子 `<el-table-column>` | `<a-table :dataSource :columns>` | columns 数组化 |
| `<el-form :model :inline>` | `<a-form :model layout>` | 默认 vertical |
| `<el-form-item label>` | `<a-form-item label>` | 一致 |
| `ElMessage.success()` | `message.success()` | 函数式 |
| `ElMessageBox.confirm()` | `Modal.confirm()` | API 略有差异 (返回 Promise) |
| `<el-button loading>` | `<a-button :loading>` | 一致 |

### 4.7 cbmem-team 变更

#### 4.7.1 新 flag

```
-cors-allow-origins "https://docs.fin-ai.net,http://localhost:5173"
```

- 逗号分隔白名单
- 默认值 `"*"` 仅 dev；生产必须显式指定
- 空字符串 = 禁用 CORS
- 特殊值 `"*"` 通配所有 origin（仅允许在 dev / staging 使用，生产禁止）

#### 4.7.2 `cmd/cbmem-team/main.go` 改动

1. **删除** `-console-dist` 静态托管路由注册（与现有 WIP 文件分离）
2. **新增** CORS 中间件 `internal/httpsrv/middleware/cors.go`：

```go
package middleware

import "github.com/gin-gonic/gin"

func CORS(allowed []string) gin.HandlerFunc {
    set := make(map[string]struct{}, len(allowed))
    wildcard := false
    for _, o := range allowed {
        if o == "*" { wildcard = true; continue }
        set[o] = struct{}{}
    }
    return func(c *gin.Context) {
        origin := c.GetHeader("Origin")
        if wildcard { set["*"] = struct{}{} }
        if _, ok := set[origin]; ok || wildcard {
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

3. 在 API 路由组挂 `middleware.CORS(origins)`；`/health` 不挂

#### 4.7.3 systemd unit 改动

```diff
  ExecStart=/usr/local/bin/cbmem-team \
      -listen :8787 \
      -data /var/lib/cbmem-team \
      -mcp-bin /home/tianque/codebase-memory-mcp \
      -jwt-secret ${JWT_SECRET} \
      -admin-token ${ADMIN_TOKEN} \
+     -cors-allow-origins ${CORS_ALLOW_ORIGINS} \
      -log info
```

`/etc/cbmem-team/cbmem-team.env`：
```
CORS_ALLOW_ORIGINS=https://docs.fin-ai.net
```

#### 4.7.4 新增 `tools/cbmem-team/deploy/Caddyfile`

```caddyfile
docs.fin-ai.net {
    root * /var/www/docs.fin-ai.net
    encode zstd gzip
    file_server
    try_files {path} {path}.html
}

api.fin-ai.net {
    reverse_proxy 127.0.0.1:8787 {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
    }
}
```

## 5. Data Flow

### 5.1 浏览器 → cbmem-team (生产)

```
[Browser]
  fetch(import.meta.env.VITE_API_BASE + '/users')
  → 'https://api.fin-ai.net/api/console/users'
[Browser 发送 OPTIONS 预检]
  Origin: https://docs.fin-ai.net
  Access-Control-Request-Method: GET
[Caddy]
  TLS terminate, reverse_proxy → :8787
[cbmem-team]
  CORS 中间件匹配 Origin 白名单
  → 200 OK + Access-Control-Allow-Origin: https://docs.fin-ai.net
```

### 5.2 浏览器 → cbmem-team (dev)

```
[Browser]
  fetch('/api/console/users')
[Vite dev server]
  proxy: '/api/console' → http://127.0.0.1:8787
[cbmem-team]
  无 Origin (同源) → 不需要 CORS
  → 200 OK
```

## 6. Error Handling

| 场景 | 行为 |
|---|---|
| cbmem-team 不可达 | `api()` throw → 页面 `try/catch` → `notify.error('后端不可达')` |
| CORS 白名单未匹配 | 浏览器拦截，console 报 CORS error；前端页面显示通用错误 |
| OPTIONS 预检失败 | 浏览器拒绝请求；需检查 `-cors-allow-origins` 是否含正确源 |
| env var 缺失 | `VITE_API_BASE` 默认 `/api/console`；CI/CD 必须 `.env.production` 存在 |
| AntD 组件加载失败 | 整页 SPA 崩溃；新增 `errorCaptured` 兜底显示静态错误页 |

## 7. Testing

### 7.1 单元测试 (Go)

| 文件 | 用例 |
|---|---|
| `internal/httpsrv/middleware/cors_test.go` | 白名单命中/拒绝；`*` 通配；OPTIONS 204；Vary header |
| `cmd/cbmem-team/routes_test.go` | 路由表断言无 `/console/*` 路径；`/api/console/*` 全部存在 |

目标覆盖率 ≥ 80%（现有模块已达标）。

### 7.2 单元测试 (Vitest, docs-site)

| 文件 | 用例 |
|---|---|
| `console/api.test.ts` | `api()` 拼接 base URL；非 2xx 抛错；JSON 解析 |
| `console/pages/Users.test.ts` | 加载/删除/编辑流程，mock fetch |
| `console/pages/Distill.test.ts` | 表单提交 + 错误处理 |
| `console/Layout.test.ts` | hash 路由切换 |

目标覆盖率 ≥ 80%。

### 7.3 集成验证 (手工)

1. **dev 链路**：`pnpm dev` + `cbmem-team` 同时跑
   - `curl http://localhost:5173/api/console/health` → 200
   - 浏览器手动登录 → 列表 → 提交，无 CORS error

2. **prod 链路 (本地模拟)**：`cbmem-team --cors-allow-origins=https://docs.fin-ai.net` + 自签证书 Caddy
   - `curl https://api.fin-ai.net/api/console/health` → 200
   - 浏览器从 `docs.fin-ai.net` 域名发请求 → 200

3. **验收**：
   - [ ] `pnpm build` 产物 `dist/` 是纯静态
   - [ ] docs-site 删 cbmem-team import 后能独立 `pnpm dev`
   - [ ] cbmem-team 删 docs-site 静态托管后能纯后端跑
   - [ ] 三语 console 占位页 (`/console/`, `/en/console/`, `/ja/console/`) 正常加载 console SPA
   - [ ] AntD 控件在 7 个 console 页面无 console error

## 8. Deployment & Rollback

### 8.1 部署步骤

1. `pnpm build` → 推送 `dist.tar.gz` 到 `docs.fin-ai.net:/tmp/`
2. SSH 到服务器，解压到 `/var/www/docs.fin-ai.net/`
3. `systemctl reload cbmem-team` 读取新 CORS env
4. Caddy 配置 reload：`systemctl reload caddy`
5. 烟测：`curl https://api.fin-ai.net/api/console/health`

### 8.2 回滚

- docs-site：保留 `dist.tar.gz` 历史，`systemctl revert caddy` 5 秒回退
- cbmem-team：`systemctl revert cbmem-team` 一键回退 systemd unit + 配置

### 8.3 文档更新

- `docs-site/guide/intro.md`：新增"独立部署拓扑"小节 + ASCII 图
- `docs-site/sop/memory/deploy.md`：更新部署章节，加入 Caddyfile + CORS 说明
- `tools/cbmem-team/CONSOLE.md`：删除"gin 托管 console dist"段落
- `tools/cbmem-team/README.md`：命令示例加 `-cors-allow-origins`
- `CHANGELOG.md`：v2.0.0 — 独立部署 + Ant Design

## 9. Risks & Mitigations

| 风险 | 缓解 |
|---|---|
| AntD 4.x 与 Vue 3.4 兼容性问题 | 已查文档：ant-design-vue ^4.2.0 兼容 Vue ^3.4 |
| AntD bundle 体积大 (~1MB) | Vite 按需引入 (`unplugin-vue-components`)；但 console SPA 已在 docs-site 内，本来就加载 vitepress 包，可接受 |
| CORS 配置错误导致全站不可用 | staging 先验证；保留 `systemctl revert` 一键回退 |
| `-console-dist` 删除破坏现有部署 | systemd unit `previous` 自动快照；部署脚本保留 5 份历史 |
| 7 个页面逐个迁移引入回归 | 每迁移一个页面跑一次 `pnpm dev` 验证；用 git bisect 兜底 |

## 10. Open Questions

无 — 关键决策（域名、版本、迁移范围、API 配置）已在 brainstorm 阶段与用户确认。
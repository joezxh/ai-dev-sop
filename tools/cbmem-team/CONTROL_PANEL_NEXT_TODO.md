# 控制台集成 · P1-P3 工作清单（接续 fc40b28）

> **入口 spec**：`docs-site/specs/2026-07-12-cbmem-team-console-design.md` (rev 2)
> **本轮已落**：
> - `8b2b6b1` spec rev 2 (设计文档)
> - `9fb9b6c` docs-site pinia SSR + console 路由修复
> - `fc40b28` cbmem-team 后端 `/distill` 挂载 + RBAC 骨架
>
> **本文件目标**：让下一轮会话接续时不丢上下文。

---

## 1. 总览：spec rev 2 §10 五大阶段

| 阶段 | 状态 | 估时 |
|---|---|---|
| A. docs-site 基础设施 | ✅ 框架完成（pinia + console 路由）；设计令牌 + 主题覆盖未做 | 0.5 天 |
| B. 后端补 4 接口 + RBAC | 🟡 `/distill` 已挂；`/config` `/logs` `/prefs` 待做 | 1.5 天 |
| C. 7 模块前端 | 🟡 M0 6 views 已存在；其余 0% | 8 天 |
| D. 测试 + 文档 | ❌ | 2 天 |
| E. 灰度 fallback | ❌ | 1 天 |

---

## 2. P1 · 服务状态模块（首推优先，因为它是零依赖验证整条链路）

### 2.1 数据源

- 现有：`GET /healthz`、`GET /api/console/v2/dashboard/summary`、`GET /api/console/me`
- **需补**：`GET /api/console/v2/status/runtime`（返回 PID / uptime / Go 版本 / 监听端口）

### 2.2 后端 task

新建 `internal/console/status_handler.go`：

```go
package console

import (
    "net/http"
    "os"
    "runtime"
    "time"

    "github.com/gin-gonic/gin"
)

// ProcessInfo captures the runtime state of the cbmem-team
// process. Uptime is computed from process start time on Linux/macOS
// and from binary birthtime on Windows — fallback to OS time minus
// a reasonable offset on systems lacking /proc/self.
type ProcessInfo struct {
    PID       int    `json:"pid"`
    GoVersion string `json:"go_version"`
    NumGoroutine int `json:"goroutine_count"`
    Listen    string `json:"listen"`         // -listen flag value
    SinceISO  string `json:"uptime_started"`
    UptimeSec int64  `json:"uptime_seconds"`
}

func StatusRuntimeHandler(started time.Time, listen string) gin.HandlerFunc {
    return func(c *gin.Context) {
        info := ProcessInfo{
            PID:          os.Getpid(),
            GoVersion:    runtime.Version(),
            NumGoroutine: runtime.NumGoroutine(),
            Listen:       listen,
            SinceISO:     started.UTC().Format(time.RFC3339),
            UptimeSec:    int64(time.Since(started).Seconds()),
        }
        OK(c, info)
    }
}
```

在 `router.go` 中挂载（在 `v2` 路由组下）：

```go
v2.GET("/status/runtime", StatusRuntimeHandler(cfg.StartedAt, cfg.ListenAddr))
```

需要给 `MountConfig` 增加两个字段：`StartedAt time.Time` 和 `ListenAddr string`。

### 2.3 前端 task

新建 `docs-site/.vitepress/theme/console/pages/Status.vue`：

```vue
<template>
  <div>
    <h2>服务状态</h2>
    <el-row :gutter="16">
      <el-col :span="6"><el-card><h3>健康检查</h3><p :class="healthClass">{{ healthText }}</p></el-card></el-col>
      <el-col :span="6"><el-card><h3>进程</h3><p>PID {{ runtime.pid }} · Go {{ runtime.go_version }}</p></el-card></el-col>
      <el-col :span="6"><el-card><h3>Goroutine</h3><p>{{ runtime.goroutine_count }}</p></el-card></el-col>
      <el-col :span="6"><el-card><h3>Uptime</h3><p>{{ uptimeText }}</p></el-card></el-col>
    </el-row>

    <h3 style="margin-top:24px">Dashboard 摘要</h3>
    <el-table :data="[summary]" v-loading="loading">
      <el-table-column prop="total_calls"     label="总调用" />
      <el-table-column prop="errors"          label="错误" />
      <el-table-column prop="active_users"    label="活跃用户" />
      <el-table-column prop="today_calls"     label="今日" />
    </el-table>

    <p>每 10 秒自动刷新</p>
    <el-button @click="onRestart" type="danger" :disabled="!isAdmin">重启后端</el-button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { api } from '../store/api'

const runtime = ref<any>({})
const summary = ref<any>({})
const loading = ref(false)
const isAdmin = ref(true)  // wired from session store once RBAC role is read

let timer: number | undefined

async function reload() {
  loading.value = true
  try {
    const [r, s] = await Promise.all([
      api('/v2/status/runtime'),
      api('/v2/dashboard/summary'),
    ])
    runtime.value = r
    summary.value = s
  } finally {
    loading.value = false
  }
}

const uptimeText = computed(() => {
  const sec = runtime.value.uptime_seconds || 0
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return `${h}h ${m}m`
})

const healthText = computed(() => runtime.value.pid ? '运行中' : '离线')
const healthClass = computed(() => runtime.value.pid ? 'ok' : 'err')

function onRestart() {
  api('/v2/status/restart', { method: 'POST' }).then(() => alert('重启请求已发送'))
}

onMounted(() => { reload(); timer = window.setInterval(reload, 10_000) })
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>
```

需要修改：
- `Layout.vue`：在侧边栏加 `<li><a href="#/status">状态</a></li>` + 路由分流 `if (r.startsWith('/status')) return Status`
- `api.ts`：可加 `StatusAPI = { runtime: () => api('/v2/status/runtime'), restart: () => api('/v2/status/restart', { method: 'POST' }) }`

### 2.4 测试

- 后端：`status_handler_test.go`（用真实 gin engine 验证 PID > 0 / GoVersion 不空 / UptimeSec >= 0）
- 前端：先不做（已经有 RBAC 单元测试 + 现有 e2e 覆盖）

### 2.5 验收

```bash
# 1. 启动 cbmem-team（带 -console-dist 指向 docs-site/.vitepress/dist/）
# 2. 浏览器访问 http://localhost:8787/console/
# 3. 登录 admin token
# 4. 侧边栏点 "状态" → 看到 4 个卡片 + 1 个 dashboard 摘要
# 5. 等 10 秒 → 数据自动刷新
# 6. 点 "重启后端" → 后端实际重启（可选）
```

---

## 3. P2 · 用户权限实装（RBAC 从骨架到真生效）

### 3.1 改动点

- DB schema：`ALTER TABLE users ADD COLUMN role VARCHAR(16) NOT NULL DEFAULT 'admin'`
- `internal/store/user.go`：User struct 增加 `Role string` 字段
- `internal/console/users_handler.go`：userToDTO / create / update 加 role 处理
- `internal/console/auth.go`：RequireSession 改为读 role 注入 context
- 前端 `store/session.ts`：login 后存 role；`store/api.ts`：进 envelope 时附带 role
- 路由 `r.POST("/admin/users", RequireSession(sm), RequireCSRF(), RequireRole(RoleAdmin), ...)` 加 RequireRole

### 3.2 文件清单

新建 1 个：`internal/console/migrate_role.go`（自动 migration）
修改 5 个：users_handler.go / store/user.go / auth.go / router.go（多个路由加 RequireRole）
前端 2 个：store/session.ts + store/api.ts

---

## 4. P2 · 日志查看（logs 后端 + 前端 + SSE）

### 4.1 后端

新建 `internal/console/logs_handler.go`：

- 全局 ring buffer（默认 5000 条）
- `GET /api/console/v2/logs?level=info&since=...&limit=500` —— 返回历史
- `GET /api/console/v2/logs/stream` —— SSE 实时流（Content-Type: text/event-stream）
- Configure via `-log-buffer-size` flag

### 4.2 前端

新建 `docs-site/.vitepress/theme/console/pages/Logs.vue`：

- 顶部筛选条：level 多选 / 时间 / limit / 关键字 / 实时跟踪开关
- el-table 虚拟滚动：1000+ 行不卡
- "导出 CSV" 按钮（前端 blob URL）

---

## 5. P3 · 工作流 + Repo Pipeline

### 5.1 工作流编辑器

- 装 `@vue-flow/core`（vue-flow）
- `pages/WorkflowEditor.vue`：vue-flow 拖拽，7 节点 kind（log/tool_call/bp_check/condition/wait_approval/parallel/loop_guard）右侧栏可拖入
- 已存在后端 `/api/console/v2/workflows/*` 完整 8 路由——直接接

### 5.2 Repo Pipeline

- `pages/Repos.vue` + `pages/RepoDetail.vue` + `pages/Candidates.vue`
- 已存在后端 `/api/console/v2/repos/*` 完整 14 路由——直接接

---

## 6. P3 · 测试 + 文档

### 6.1 vitest 单元测试

新建：
- `docs-site/.vitepress/theme/console/__tests__/api.test.ts`
- `docs-site/.vitepress/theme/console/__tests__/rbac.test.ts`
- `docs-site/.vitepress/theme/console/__tests__/users-page.test.ts`

`pnpm --dir docs-site test` 跑通 + coverage ≥ 60%

### 6.2 Playwright e2e

`docs-site/tests/e2e/`：
- `login.spec.ts`
- `status.spec.ts`
- `users-rbac.spec.ts`

### 6.3 文档更新

- `tools/cbmem-team/manual.md` §5.7：从"前端在 docs-site/.vitepress/dist/"改成实际可部署版本（含 nginx 范例）
- `tools/cbmem-team/BUILD.md`：加 pnpm build docs-site 步骤
- `docs-site/README.md`：加 "控制台接入" 段落
- 新建 `docs-site/console/README.md`

---

## 7. 风险与注意事项

- **降级**：当前 session.ts 写的是 `useStore(useSessionStore)` — 必须在 `enhanceApp` 之前先 `app.use(createPinia())` 才能用。已经修了（9fb9b6c）。
- **dev server vs build**：`pnpm --dir docs-site dev` 时 Vue 会 hot reload，需要保证 Layout.vue 不要每次 mount 都 `setTimeout(import('./router'))`（别在 SSR 时执行）—— 已有 consoleHash computed 用 `typeof window` 守卫。
- **RBAC 真生效不能急**：M0 已有 Users.vue 走 happy path，加 role 后一旦viewer 登录，编辑按钮必须 v-show 隐藏——**避免改了后端而忘了前端 v-if**。
- **MySQL 字符序**：所有 schema 改动要保持 `utf8mb4_unicode_ci`（现有约定）；新加 ALTER TABLE 加 COMMENT 标记新增原因。

---

## 8. 给下一轮会话的 kickoff 提示

如果新会话从本文件起步，第一句应该是：

> "继续 docs-site/specs/2026-07-12-cbmem-team-console-design.md rev 2 的实施。已 commit: 8b2b6b1 / 9fb9b6c / fc40b28。下一步做 P1 服务状态模块——按 `tools/cbmem-team/CONTROL_PANEL_NEXT_TODO.md` §2 步骤：先补后端，再补前端，最后跑 build 验证。"

---

_Last updated: 2026-07-12_

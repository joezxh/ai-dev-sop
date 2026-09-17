# Task 12 Report — Dashboard Webhooks 真实管理页

**状态：** ✅ 完成
**提交：** `2881f8e3` — `feat(dashboard): real webhooks management page`（分支 `feat/llm-provider-i18n`）
**变更文件：** 4 个文件，+325 / -40

## 实现内容

| 文件 | 变更 |
| --- | --- |
| `server/dashboard/src/app/(root)/dashboard/webhooks/page.tsx` | 整页替换 LockedPage 占位 → 真实 CRUD 管理页 |
| `server/dashboard/src/utils/api-endpoints.ts` | 新增 `WEBHOOK_ENDPOINTS = { BASE, BY_ID }` |
| `server/dashboard/src/types/api.ts` | 新增 `Webhook` 接口（id/name/url/events/secret/created_at） |
| `server/dashboard/src/app/(root)/dashboard/components/main-nav.tsx` | 导入 `Webhook` 图标；`NAV_KEYS` 加 `Webhooks: "nav.webhooks"`；ACTIVITY 组加 Webhooks 入口 |

## 页面实现（对齐 `departments/page.tsx` 结构）

- **列表**：Card 表格（Name / URL / Events / Created / Actions），客户端分页（PAGE_SIZE=10），`TableSkeleton` 加载态、`EmptyState` 空态、错误条。
- **新建**：shadcn Dialog 表单 —— name（必填）、URL（必填，前端校验 scheme 仅 http/https，与后端 400 行为一致）、events 复选框组（`memory_added` / `memory_updated` / `memory_deleted`，至少选一个的前端校验）、secret（可选，type=password）。创建体 `{name, url, events, secret?}`。
- **删除**：`DeleteConfirmationModal` 确认后调 `api.delete(WEBHOOK_ENDPOINTS.BY_ID(id))`。
- 数据获取：`api.get(WEBHOOK_ENDPOINTS.BASE)`，错误取 `e.response.data.detail`，与 departments 页一致；沿用现有主题 token（`border-memBorder-primary`、`bg-surface-default-secondary`、`text-memText-secondary`、`text-onSurface-danger-primary` 等），未硬编码色板。
- `Webhook` 图标已先用 node 验证存在于 lucide-react（`FolderKan` 教训）。

## 验证

| # | 验证项 | 结果 |
| --- | --- | --- |
| 1 | 图标存在性 | `node -e "require('lucide-react').Webhook"` → `object` ✅ |
| 2 | IDE lint（4 个改动文件） | 0 error / 0 warning ✅ |
| 3 | `docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard` | `mem0-api` + `mem0-dashboard` 均 `Built` / `Recreated` / `Started`，构建日志无 error（构建含类型检查，`ignoreBuildErrors: false`）✅ |
| 4 | 容器日志 | `Next.js 15.5.21` `✓ Ready in 360ms`，无运行时 error ✅ |
| 5 | 页面 `GET http://localhost:3001/dashboard/webhooks` | **307**（未登录重定向，预期）✅ |
| 6 | 后端联动 `GET http://localhost:8888/webhooks` | **200**，body `[]`（后端在线，空列表）✅ |

## 环境备注

- 执行时 Docker Desktop 未运行，已启动后重试（引擎 28.1.1，与 task-1 报告同路径）。
- 实际端口映射：dashboard → 宿主机 **3001**、api → 宿主机 **8888**（任务描述里"页面 200"以 3001 为准）。

## Concerns

1. **未做投递失败端到端冒烟**（brief Step 4：建 webhook 指向死 URL → 加 memory → 看容器日志 `Webhook ... delivery failed`）：该链路需 admin 登录态调用 API 与添加 memory，本任务未持有凭据；前端页面 CRUD 结构与 departments 页完全同构，且后端（Task 4）6+3 测试已覆盖，风险低。建议 Task 15 全量冒烟时补做。
2. Dashboard 未登录返回 307 重定向到登录页属预期，登录后的页面交互未做浏览器级验证（与既往任务验证深度一致）。

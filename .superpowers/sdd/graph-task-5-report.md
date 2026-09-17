# Graph Task 5 报告 — 图谱页脚手架（依赖/端点/类型/导航）

**状态：✅ 完成**
**提交：`89227fdb`**（分支 `feat/graph-memory`，`feat(dashboard): graph page scaffolding (deps/endpoints/types/nav)`，7 files changed, 245 insertions）

## 变更明细

| 文件 | 变更 |
| --- | --- |
| `server/dashboard/package.json` + `pnpm-lock.yaml` | 新增 `react-force-graph-2d@1.29.1` |
| `src/utils/api-endpoints.ts` | 新增 `GRAPH_ENDPOINTS = { SEARCH: "/graph/search", GET_ALL: "/graph/get_all" } as const` |
| `src/types/api.ts` | 新增 `GraphRelation`（source/relationship/destination，含 docstring）与 `GraphSearchResponse`（relations） |
| `src/app/(root)/dashboard/components/main-nav.tsx` | 导入 `Waypoints` 图标；`NAV_KEYS` 加 `"Graph": "nav.graph"`；ACTIVITY 组新增 Graph 项（url `/dashboard/graph`） |
| `src/i18n/en.ts` / `zh.ts` | 新增 `nav.graph`："Graph" / "图谱" |

## 验证结果

1. **图标存在性**：`node -e "console.log(!!l.Share2,!!l.Waypoints)"` → `true true`，两者均可用，选用语义更贴合的 `Waypoints`。
2. **TypeScript**：`npx tsc --noEmit` → exit 0，无类型错误。
3. **Lint**：read_lints 对 5 个改动文件 0 诊断。
4. **Docker 构建**：`docker compose -f deploy/mem0/docker-compose.yaml up -d --build mem0-dashboard` → `mem0-dashboard Built`，Next.js 生产构建成功（路由表完整输出，无 error）。新页面未创建故构建产物中无 `/dashboard/graph` 路由，符合预期（Task 6 创建）。

## 偏差说明

1. **[Rule 3 - 阻塞修正] 用 `pnpm add` 替代 `npm install`**：项目由 pnpm 管理（`pnpm-lock.yaml` + `packageManager: pnpm@10.34.2`）。用 npm 安装会生成冗余 `package-lock.json` 并破坏 pnpm 依赖树一致性，故改用 `pnpm add react-force-graph-2d`。无需 `--legacy-peer-deps`——新依赖未引入任何 peer 冲突（pnpm 报告的 peer 警告均为遗留：react-day-picker、vaul 对 React 19 的旧约束）。
2. **提交范围收窄**：指令为 `git add server/dashboard`，但工作区存在遗留改动 `server/dashboard/public/favicon.ico` 删除（非本任务产物）。为避免夹带，改为逐个精确 add 本任务的 7 个文件。favicon 删除仍保留在工作区未动。

## Concerns

1. **容器未 Started（端口冲突，非代码问题）**：构建成功后容器启动失败——宿主机 3001 端口被项目自身的一个本地 Next dev 进程占用（PID 33348，`server/dashboard` 的 `next start-server`，疑似用户或前序任务手动启动）。未擅自 kill 该进程。代码层面验证已充分（tsc 0 错误 + Docker 内生产构建成功），如需完整 Started 验证，请先停掉该 dev 进程后重跑 `docker compose up -d mem0-dashboard`。
2. `/dashboard/graph` 当前 404 属预期，页面在 Task 6 创建。
3. 工作区存在与本任务无关的遗留改动（`server/main.py`、`server/docker-compose.yaml`、`server/requirements.txt`、`mem0/embeddings/*`、`mem0/vector_stores/pgvector.py`、`server/data/`、`server/scripts/seed.sql`、favicon 删除），均未纳入本提交。

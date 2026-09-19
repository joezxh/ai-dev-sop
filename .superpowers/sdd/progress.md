# SDD Progress Ledger — mem0 cloud alignment (plan 2026-09-15)

Task 1: complete (commit 1b349988, review clean; deferred: stats time-filter test coverage -> final review)
Task 2: complete (commit 2a989800, review clean; minors: null-clear unavailable, prompt falsy semantics)
Task 3: complete (commit 27f11b4e, review clean; deviation: route merged into existing OSS DELETE /memories with project_id dispatch; minors: AUTH_DISABLED=admin semantics, >10k silent truncation, id=None edge)
Task 4: complete (commits 0b3c23ad+6b9d4099, review clean after fixes: no-throw dispatch guard + URL scheme validation; minors: secret plaintext echo, event payload minimal, per-add thread, log string dup)
Task 5: complete (commit 556cdb4d, review clean; deviations: naive-ts UTC fix + invalid-date 400; minors: format builtin shadow, tz replace vs astimezone, end-date exclusive semantics)
Task 6: complete (commits ab81d8ff+cf63d919, review clean after fixes: nav reorg 4 groups, stubs/banner removed, tooltip i18n, Tags icon; minors: group headings hardcoded EN, nav.dashboard duplicate, flattenKeys dead export)
Task 7: complete (commits b59f934f+bc343225+5e4e34ff, review clean (3 rounds): typed /stats response, race guard, all-time labels, effect-based refetch, error state; minors: no retry button, skeleton is table-shaped, 10k memory cap)
Task 8: complete (commits fa5e95a8+f67a0d52+d4471c49+895c32f4, review clean after fixes: 7 MCP tools aligned with mcp_server.py, placeholder hints for all tabs, python timeout/raise_for_status, Windows cURL note; minors: no i18n, no copy button, server address not injected)
Task 9: complete (commits 8edc8d50+4f644186+322ba5a6, review clean after fixes: admin gating, cURL shq escaping, SEARCH_ENDPOINTS constant, output reset, NEXT_PUBLIC_API_URL; minors: sandbox id SSR/CSR hydration, cURL fallback host)
Task 10: complete (commits 44ffe074+effaedd1+95c0942b+8f413045+4531d1e5+4a4a7196+31fa248e, review clean (4 rounds): exclude_unset nullable clears, delete truncation report, NOT NULL guards for name/git_remotes, trimmed fields, English purge toasts)
Task 11: complete (commits 9556e9dc+c13c25ea+80b9bdd5, review clean (2 rounds): SQL action filter + limit restored, statement-capture tests; FOLLOW-UP for user: memories page setState+refetch stale-closure filter bug (real, out of scope))
Task 12: complete (commit 2881f8e3, review clean; minors inherited from departments pattern: stale-page after delete-last, blank name, formError position, hardcoded EN; plus known backend secret-plaintext echo)
Task 13: complete (commits 23587e3c+484b371a, review clean after fix: blob error detail surfaced; minors: fixed filename vs Content-Disposition, empty-projects EmptyState contradiction, load() catch any)
Task 14: complete (commit 72f7e6e1, review clean; enabled-gated localStorage init verified; follow-up: memories select should write back localStorage, hardcoded "All projects", collapsed-mode fetch)
Task 15: complete (verification only, no commits; 67 passed, smoke 6/6, 5 pages 307-as-expected; webhook E2E blocked by placeholder OPENAI_API_KEY — covered by Task 4 unit tests)
Final review: APPROVE_WITH_NOTES -> 3 must-fix fixed (commits 407598a1 backend, d2f5e426 frontend); 69 passed; branch ready to merge (mem0 HEAD = d2f5e426)
Known-issues fix wave 1 (user-requested): backend da573fda+13f872a8 (secret mask+HMAC, SSRF private-net guard, executor+2xx log+payload id, ALLOWED_EVENTS single-source, ACTION map single-source, SCAN_LIMIT unify, stats SQL counts) -> 92 passed
Known-issues fix wave 2 (frontend): f67709a6+d66922fd (useApiQuery deps option, request-sequence guard follow-up, PROJECT_CONTEXT_KEY single-source, localStorage write-back, allProjects i18n, 4-page pagination clamp, webhook name trim, copy buttons, playground metadata JSON) -> re-review I-01/I-02 fixed in 4260fc1a, Approved
Status: 92 passed; mem0 HEAD = 4260fc1a; remaining accepted-by-design items: X-Webhook-Secret compat header, delivery-time DNS rebinding, cross-language event whitelist sync by comment, overview not on deps, AUTH_DISABLED=admin semantics, caps 10k/1k

== Graph memory plan (feat/graph-memory branch, plan 2026-09-16-mem0-graph-memory) ==
Graph Task 1: complete (commits 5a9cf86b+683151e9+2f3eff57, review clean (3 rounds): GraphMemory core, node identity (name,user,project) with "" materialization, scope-driven predicates, destructive-delete guards; INFO open: project-scope disambiguation pools same-name nodes over time)
Graph Task 2: complete (commits 1661945a+d0455262+1be73f4d, review clean (2 rounds): graph endpoints with scoping, server_state singleton DCL, unscoped-read guards; debug-instrumentation riding changes reverted; minor: get_all dead-code model, GET limit clamp)
Graph Task 3: complete (commit e8e2cc34, review clean; follow-up: single-memory soft-delete scope gap (project-pool memory delete leaves relations — brief-level gap, needs metadata.project_id in Memory.get); entity-delete wired in routers/entities.py with user-type guard)
Graph Task 4: complete (outer commit 852b791; config wiring verified: env live, 116 passed, no degrade log; .env stays untracked by design)
Graph Task 5: complete (commit 89227fdb, review clean; pnpm used correctly; minors: Stats JSDoc split by insertion point, exact-match nav active)
Graph Task 6: complete (commit 580a227d, review clean: tsc 0 err + lint 0 diag; full Next prod build OK except Windows standalone symlink EPERM — environmental, Docker/Linux build unaffected per Task 5). Files: graph/page.tsx + en.ts/zh.ts graph.* keys (title/searchPlaceholder/empty*/stats*/detail*/relations). Force-graph canvas, node degree sizing, select-highlight, drag-pin, zoom-to-fit, rebuild-layout, dark/light palette, detail side-panel with incident edges.
Graph Task 7: complete (commit c9262013, review clean: tsc 0 err + lint 0 diag). memories/page.tsx detail Sheet adds "View related graph" button (Waypoints icon, i18n memories.viewGraph en/zh) → router.push(/dashboard/graph?q=<memory>&project=<project_id>); wires useRouter + useTranslation.
Graph Task 8: verification — container pytest: 24 graph tests pass; full suite 116 passed (92 baseline + 24 graph), no regression. Frontend prod compile: 23/23 static pages generated incl /dashboard/graph (only Windows standalone symlink EPERM at trace-copy, environmental). Backend graph integration (main.py fusion/delete links, routers/graph scoping, graph_memory core) all committed (Tasks 1-5) and covered by tests.

## 扩展需求（再实现 1-4）
> 注意：分支上存在预提交 `c3be02aa`「feat(graph): 添加图谱导出功能和重试机制」——它实际是一笔**打包提交**：把无关的 `mem0-llm-provider-i18n` WIP（embeddings/openai.py、pgvector.py、base.py、main.py、requirements.txt、docker-compose.yaml、favicon、config.json、seed.sql）与"较早的图谱导出/重试尝试"混在一起。其 graph_memory.py 改动已被本轮 Task 11 干净覆盖（graph_memory.py 每个方法仅一份定义，无重复、无死代码），其余 WIP 文件不在本图任务提交范围内。

Graph Task 9: complete (commit 63e70c75, tsc 0 + lint 0). graph page 增加 2D/3D 切换（react-force-graph-3d + three）与 PNG 快照导出（2D `toDataURL()` / 3D `renderer().domElement.toDataURL()`，3D 设 `rendererConfig:{preserveDrawingBuffer:true}`）。i18n 增 view2d/view3d/snapshot/exportData。

Graph Task 10: complete (commit 6295b8e9, py_compile OK + test pass). mcp_server.py 新增 `search_graph`（POST /graph/search）与 `get_all_graph`（GET /graph/get_all）两个 MCP 工具，复用 _client 代理；新增 test_mcp_graph.py。

Graph Task 11: complete (commit 3e1c62e6, container pytest 122 passed = 116 baseline+graph + 6 new). graph_memory.py 增加内存重试队列（add 失败→_enqueue_retry→守护线程 _retry_loop 按指数退避重试 MAX_RETRIES=5，_backoff 上限 30s；提供 retry_pending()/stop()）；新增 export(filters,limit) 导出节点+边 JSON。routers/graph.py 新增 GET /graph/export（scope 契约同 get_all）。dashboard 增"导出数据"按钮（GRAPH_ENDPOINTS.EXPORT 下载 JSON）。新增 test_graph_retry_queue.py、test_graph_export.py。

Graph Task 12: complete (commit 85421872, 单独提交). server/dev.Dockerfile `RUN pip install -e .[graph]` → `RUN pip install -e .`（mem0ai 2.x 无 graph extra，图能力由 graph_memory.py + langchain-neo4j 提供，后者已在 requirements.txt）。requirements.txt 仍含 langchain-neo4j>=0.4，依赖完整。

== mem0 一键接入 plan (branch feat/mem0-llm-provider-i18n, plan 2026-09-19-mem0-one-step-setup) ==
Task 1: complete (commits e248aff+0e278c1, review spec ✅ / quality issues resolved by controller: MI-01 归一化已修；IM-01 由 Task 7 .gitignore 覆盖（已跑 git check-ignore 断言通过）；IM-02 由 Task 7 服务名表述更新覆盖；MI-02 CR 容忍作为 Task 3 约束下发；MI-03 计划校验命令已加 -AllMatches)
Task 2: complete (commits 1348617+ddef51c; 控制器实测修复两处真实缺陷：ConvertFrom-Json -AsHashtable 为 PS6+ 参数在 PS5.1 抛错被误判为 JSON 损坏；-DryRun 仍执行 REST 写入。DryRun 全绿：4 IDE 识别、端点 406 存活、DryRun 跳过 REST)
Task 3: complete (commit 1be5717; bash 版内置 CR 容忍（python3 渲染模板）、python3 做 JSON 合并；bash -n 通过；临时 HOME 下 -i cursor,codex 矩阵验证通过；WSL 无法访问 Windows localhost 服务属环境限制)
Task 4: complete (commit 7973b24; 5 模板 + 5 表格 + 故障表共 11 处跨天条款，验证 '当天不一致'=11)
Task 5: complete (commit 3cd2901; 121 处替换：服务名统一 mem0（保护 ~/mem0-remote/ 部署目录名）、凭证路径统一 .mem0/mem0.config.json、session 路径按 IDE 统一；验证 0/0/0/0/1/0 全绿)
Task 6: complete (commit 含在 3cd2901 之后的提交；§4.2–4.6 各章一键脚本提示 + §5.3.1 校验项)
Task 7: complete (commit 5982da9; CODEBUDDY.md 路径/服务名约定、.gitignore 增 .mem0/（check-ignore 生效）、install-all 双脚本指引、凭证与会话文件迁至 .mem0/)
Task 8: verification (T1 合并写保留 Playwright/sqlbot ✓；T2 幂等 marker 唯一 ✓；T3 cursor .mdc frontmatter + codex TOML ✓；T4/T5 错误分支提示正确 ✓；REST 往返在本机不可验证：REST 实例(8000/8002)与 MCP(8080) 非同一实例，同密钥 MCP 通而 REST 401——环境差异，非脚本缺陷；已加 -RestUrl/-s 覆盖参数 e2d0cd0)
执行方式备注：Task 1-2 用 subagent；自 Task 3 起因 gsd-executor(31KB)/gsd-code-reviewer(15KB) 定义过重、单调用 15-22s 且长任务被 abort(code=10003)，改为内联执行 + grep 断言验证。

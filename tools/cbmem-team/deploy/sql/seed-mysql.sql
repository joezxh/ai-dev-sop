-- =============================================================================
-- cbmem-team MySQL 8.0 初始化数据
-- =============================================================================
-- 内容：
--   1. admin 初始管理员账号（bcrypt hash of "admin123"）
--   2. 默认团队 "Default Team"
--   3. 49工具目录 (tool_directory)
--   4. 3条最佳实践 (best_practices + bp_versions)
--   5. 3条内置工作流 (workflows + workflow_versions)
--   6. 5条内置记忆模板 (memory_templates)
--   7. 内置AI工具 (ai_tools)
--
-- 应用顺序（先创建 admin 用户，再创建团队关联）：
--   1. 创建 admin 用户
--   2. 创建默认团队
--   3. 创建团队成员关联
--   4. 创建 legacy 节点（team_legacy / proj_legacy / mod_legacy）
--   5. 工具目录
--   6. 最佳实践
--   7. 工作流
--   8. 记忆模板
--   9. AI工具
--
-- 密码：bcrypt(cost=12) of "admin123"
--   $2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4.PQDJp.nwfP4H8y

SET autocommit=0;
START TRANSACTION;

-- =============================================================================
-- 1. admin 用户（bcrypt cost=12, "admin123"）
-- =============================================================================
INSERT INTO users (id, username, display_name, email, password_hash, role, must_change_password, disabled, created_at, updated_at)
VALUES (
    'u_admin_default',
    'admin',
    'Administrator',
    'admin@example.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4.PQDJp.nwfP4H8y',
    'admin',
    0,
    0,
    NOW(),
    NOW()
);

-- =============================================================================
-- 2. 默认团队
-- =============================================================================
INSERT INTO teams (id, name, slug, description, owner_id, created_at, updated_at, deleted)
VALUES (
    'team_default',
    'Default Team',
    'default',
    '默认团队，系统自动创建',
    'u_admin_default',
    NOW(),
    NOW(),
    0
);

-- =============================================================================
-- 3. 团队成员关联（admin 加入默认团队，角色为 admin）
-- =============================================================================
INSERT INTO team_members (team_id, user_id, role, joined_at)
VALUES ('team_default', 'u_admin_default', 'admin', NOW());

-- =============================================================================
-- 4. admin 用户的默认团队设为 default
-- =============================================================================
UPDATE users SET default_team_id = 'team_default' WHERE id = 'u_admin_default';

-- =============================================================================
-- 5. Legacy 迁移节点（v1→v2 兼容）
-- =============================================================================
INSERT IGNORE INTO teams (id, name, slug, description, owner_id, created_at, updated_at, deleted)
VALUES ('team_legacy', 'Legacy Team (migrated from v1)', 'legacy', '从 v1 迁移的默认团队', NULL, NOW(), NOW(), 0);

INSERT IGNORE INTO projects (id, team_id, name, slug, description, path, git_url, git_branch, git_commit_sha, status, owner_id, created_at, updated_at, deleted)
VALUES ('proj_legacy', 'team_legacy', 'Legacy Project', 'legacy', '从 v1 迁移的默认项目', '', '', '', '', 'ready', '', NOW(), NOW(), 0);

INSERT IGNORE INTO modules (id, project_id, name, description, path, is_leaf, created_at, updated_at, deleted)
VALUES ('mod_legacy', 'proj_legacy', 'Legacy Module', '从 v1 迁移的默认模块', '', 1, NOW(), NOW(), 0);

-- =============================================================================
-- 6. 49工具目录
-- =============================================================================
INSERT INTO tool_directory (tool_id, track, category, name, signature, status, priority, version, created_at, updated_at)
VALUES
    ('mempalace_status',            'A', 'read',    'Status',             'system_state',      'active', 1, '1.0.0', NOW(), NOW()),
    ('mempalace_list_wings',        'A', 'read',    'List wings',         'scope',             'active', 1, '1.0.0', NOW(), NOW()),
    ('mempalace_list_rooms',        'A', 'read',    'List rooms',         'wing_id',           'active', 2, '1.0.0', NOW(), NOW()),
    ('mempalace_search',            'A', 'read',    'Semantic search',    'query',             'active', 1, '1.0.0', NOW(), NOW()),
    ('mempalace_recall',            'A', 'read',    'Fuzzy recall',       'query',             'active', 2, '1.0.0', NOW(), NOW()),
    ('mempalace_wing_info',         'A', 'read',    'Wing info',          'wing_id',           'active', 3, '1.0.0', NOW(), NOW()),
    ('mempalace_list_halls',        'A', 'read',    'List halls',         'scope',             'active', 2, '1.0.0', NOW(), NOW()),
    ('mempalace_list_drawers',      'A', 'read',    'List drawers',       'hall_id',           'active', 2, '1.0.0', NOW(), NOW()),
    ('mempalace_view_drawer',       'A', 'read',    'View drawer',        'drawer_id',         'active', 1, '1.0.0', NOW(), NOW()),
    ('mempalace_get_context',        'A', 'read',    'Get context bundle', 'wing_ids[]',        'active', 2, '1.0.0', NOW(), NOW()),
    ('mempalace_add_drawer',        'A', 'write',   'Add drawer',         'drawer{original,context}', 'active', 3, '1.0.0', NOW(), NOW()),
    ('mempalace_checkpoint',        'A', 'write',   'Checkpoint session', 'session_id',        'active', 4, '1.0.0', NOW(), NOW()),
    ('mempalace_mine',              'A', 'write',   'Mine project files', 'path,wing_id',      'active', 5, '1.0.0', NOW(), NOW()),
    ('mempalace_update_drawer',     'A', 'write',   'Update drawer',      'drawer_id,body',    'active', 4, '1.0.0', NOW(), NOW()),
    ('mempalace_delete_drawer',     'A', 'write',   'Delete drawer',      'drawer_id',         'active', 6, '1.0.0', NOW(), NOW()),
    ('mempalace_tag_drawer',        'A', 'write',   'Tag drawer',         'drawer_id,tags[]',  'active', 5, '1.0.0', NOW(), NOW()),
    ('mempalace_create_wing',       'A', 'admin',   'Create wing',        'wing_name',         'active', 5, '1.0.0', NOW(), NOW()),
    ('mempalace_create_room',       'A', 'admin',   'Create room',        'wing_id,room_name', 'active', 6, '1.0.0', NOW(), NOW()),
    ('mempalace_request_access',     'A', 'admin',   'Request access',     'wing_id',           'active', 7, '1.0.0', NOW(), NOW()),
    ('mempalace_config',            'A', 'admin',   'Get/set config',     'key,value',         'active', 6, '1.0.0', NOW(), NOW()),
    ('mempalace_stats',             'A', 'admin',   'Stats',              'scope',             'active', 6, '1.0.0', NOW(), NOW()),
    ('mempalace_export',            'A', 'admin',   'Export data',        'wing_id,format',    'active', 7, '1.0.0', NOW(), NOW()),
    ('mempalace_grant_access',       'A', 'admin',   'Grant access',       'wing_id,user_id',   'active', 6, '1.0.0', NOW(), NOW()),
    ('mempalace_revoke_access',     'A', 'admin',   'Revoke access',     'wing_id,user_id',   'active', 6, '1.0.0', NOW(), NOW()),
    ('mempalace_merge_drawer',      'A', 'admin',   'Merge drawer',       'src_id,dst_id',     'active', 7, '1.0.0', NOW(), NOW()),
    ('mempalace_archive_wing',      'A', 'admin',   'Archive wing',       'wing_id',           'active', 8, '1.0.0', NOW(), NOW()),
    ('mempalace_restore_wing',      'A', 'admin',   'Restore wing',       'wing_id',           'active', 8, '1.0.0', NOW(), NOW()),
    ('mempalace_reindex_search',   'A', 'admin',   'Reindex search',     'wing_id',           'active', 7, '1.0.0', NOW(), NOW()),
    ('mempalace_quota_set',         'A', 'admin',   'Set quota',          'wing_id,bytes',     'active', 8, '1.0.0', NOW(), NOW()),
    ('mempalace_audit_log',        'A', 'admin',   'Audit log',          'filter',            'active', 8, '1.0.0', NOW(), NOW()),
    ('mempalace_backup',            'A', 'admin',   'Backup',             'wing_id',           'active', 9, '1.0.0', NOW(), NOW()),
    ('mempalace_restore',           'A', 'admin',   'Restore from backup','backup_id',         'active', 9, '1.0.0', NOW(), NOW()),
    ('mempalace_mcp_initialize',   'A', 'admin',   'MCP initialize',     'client_info',       'active', 7, '1.0.0', NOW(), NOW()),
    ('mempalace_mcp_health',       'A', 'admin',   'MCP health',         '--',               'active', 9, '1.0.0', NOW(), NOW()),
    ('mempalace_mcp_ping',         'A', 'admin',   'MCP ping',           '--',               'active', 9, '1.0.0', NOW(), NOW()),
    ('index_repository',            'B', 'index',   'Index repository',   'repo_path,mode',    'active', 3, '1.0.0', NOW(), NOW()),
    ('list_projects',               'B', 'index',   'List projects',      '--',               'active', 7, '1.0.0', NOW(), NOW()),
    ('index_status',                'B', 'index',   'Index status',       'project_id',        'active', 4, '1.0.0', NOW(), NOW()),
    ('delete_project',              'B', 'index',   'Delete project',     'project_id',        'active', 7, '1.0.0', NOW(), NOW()),
    ('search_graph',                'B', 'query',   'Search graph',       'label,name',        'active', 2, '1.0.0', NOW(), NOW()),
    ('trace_path',                  'B', 'query',   'Trace call path',    'target,depth',      'active', 2, '1.0.0', NOW(), NOW()),
    ('query_graph',                 'B', 'query',   'Cypher query',       'cypher,params',     'active', 3, '1.0.0', NOW(), NOW()),
    ('search_code',                 'B', 'query',   'Code text search',   'query,scope',       'active', 3, '1.0.0', NOW(), NOW()),
    ('get_code_snippet',            'B', 'query',   'Get snippet',        'file,line_range',   'active', 4, '1.0.0', NOW(), NOW()),
    ('get_graph_schema',            'B', 'query',   'Graph schema',       '--',               'active', 6, '1.0.0', NOW(), NOW()),
    ('detect_changes',              'B', 'analysis','Detect changes',     'git_diff',          'active', 1, '1.0.0', NOW(), NOW()),
    ('get_architecture',           'B', 'analysis','Get architecture',   'scope',             'active', 3, '1.0.0', NOW(), NOW()),
    ('manage_adr',                  'B', 'analysis','ADR CRUD',           'op,adr',            'active', 4, '1.0.0', NOW(), NOW()),
    ('ingest_traces',               'B', 'analysis','Ingest traces',      'trace_bundle',      'active', 5, '1.0.0', NOW(), NOW());

-- =============================================================================
-- 7. 最佳实践 (3条)
-- =============================================================================

-- BP1: MemPalace Drawer 命名约定
INSERT INTO best_practices (id, title, category, track, tools, related_halls, related_wings, scenes, priority, status, version, created_by, created_at, updated_at, review_due, source, body)
VALUES (
    'bp-naming-mempalace-drawer',
    'MemPalace Drawer 命名约定',
    'naming',
    'A',
    '["mempalace_add_drawer","mempalace_update_drawer","mempalace_view_drawer"]',
    '["hall_facts"]',
    '["wing-org-policy"]',
    '["design","contribute"]',
    'P2',
    'published',
    1,
    'u_admin_default',
    NOW(),
    NOW(),
    DATE_ADD(NOW(), INTERVAL 90 DAY),
    'manual',
    '## 场景\n新建 drawer 时。\n\n## 工具签名\nmempalace_add_drawer(title, original, context)\n\n## 操作步骤\n1. 标题 ≤ 80 字，使用「动词 + 对象 + 上下文」结构\n2. original 保留客户/会议原话\n3. context 补充决策背景\n\n## 验证\n`mempalace_list_drawers` 能按名称前缀过滤。\n\n## 注意事项\n禁止修改他人的 drawer — 用 `mempalace_update_drawer` 自己的。\n\n## 协同使用\n与 `codebase-mem.manage_adr` 双写。'
);

INSERT INTO bp_versions (bp_id, version, snapshot_json, changed_by, change_note, created_at)
VALUES ('bp-naming-mempalace-drawer', 1, '{"id":"bp-naming-mempalace-drawer","title":"MemPalace Drawer 命名约定","category":"naming","track":"A","status":"published","version":1}', 'system', 'initial seed', NOW());

-- BP2: 大仓首次索引走 fast 模式
INSERT INTO best_practices (id, title, category, track, tools, related_halls, related_wings, scenes, priority, status, version, created_by, created_at, updated_at, review_due, source, body)
VALUES (
    'bp-perf-codebase-index-fast',
    '大仓首次索引走 fast 模式',
    'performance',
    'B',
    '["index_repository","index_status"]',
    '["hall_events"]',
    '["wing-perf"]',
    '["deploy","review"]',
    'P1',
    'published',
    1,
    'u_admin_default',
    NOW(),
    NOW(),
    DATE_ADD(NOW(), INTERVAL 90 DAY),
    'manual',
    '## 场景\n仓库 > 100k LOC 首次进入索引。\n\n## 工具签名\nindex_repository(repo_path, mode=''fast'')\n\n## 操作步骤\n1. 先用 `mode=fast` 拿到 80% 覆盖率\n2. CI 异步补全 `mode=deep`\n\n## 验证\n`index_status.coverage` ≥ 80% in 30s。\n\n## 注意事项\nfast 模式不索引符号引用 — 后续 trace_path 精度下降。\n\n## 协同使用\n完成后调用 `mempalace_checkpoint` 留底。'
);

INSERT INTO bp_versions (bp_id, version, snapshot_json, changed_by, change_note, created_at)
VALUES ('bp-perf-codebase-index-fast', 1, '{"id":"bp-perf-codebase-index-fast","title":"大仓首次索引走 fast 模式","category":"performance","track":"B","status":"published","version":1}', 'system', 'initial seed', NOW());

-- BP3: ADR ↔ Drawer 双写桥
INSERT INTO best_practices (id, title, category, track, tools, related_halls, related_wings, scenes, priority, status, version, created_by, created_at, updated_at, review_due, source, body)
VALUES (
    'bp-collab-adr-drawer-bridge',
    'ADR ↔ Drawer 双写桥',
    'collaboration',
    'J',
    '["manage_adr","mempalace_add_drawer","mempalace_view_drawer"]',
    '["hall_facts"]',
    '["wing-org-policy"]',
    '["design","review","replay"]',
    'P0',
    'published',
    1,
    'u_admin_default',
    NOW(),
    NOW(),
    DATE_ADD(NOW(), INTERVAL 90 DAY),
    'manual',
    '## 场景\n每次新建/修改 ADR 时。\n\n## 工具签名\nmanage_adr(op=''update'', adr); mempalace_add_drawer(title, original=''<adr body>'', context=''ADR auto-mirror'')\n\n## 操作步骤\n1. 先写 ADR\n2. 自动镜像到 MemPalace hall_facts\n3. 关联同一 wing\n\n## 验证\n`mempalace_search <adr title>` 返回镜像 drawer。\n\n## 注意事项\n`original` 字段保留 ADR 全文 markdown。\n\n## 协同使用\n轨道 A 看板 + 轨道 B 架构图共享同一决策源。'
);

INSERT INTO bp_versions (bp_id, version, snapshot_json, changed_by, change_note, created_at)
VALUES ('bp-collab-adr-drawer-bridge', 1, '{"id":"bp-collab-adr-drawer-bridge","title":"ADR ↔ Drawer 双写桥","category":"collaboration","track":"J","status":"published","version":1}', 'system', 'initial seed', NOW());

-- =============================================================================
-- 8. 内置工作流 (3条)
-- =============================================================================

-- WF1: Commit 前置检查
INSERT INTO workflows (workflow_id, name, description, track, category, nodes_json, entry_id, status, version, created_by, created_at, updated_at)
VALUES (
    'wf-commit-precheck',
    'Commit 前置检查',
    '提交前自动跑命名规范 BP 检查 + detect_changes 算爆炸半径，超阈值则 wait_approval。',
    'J',
    'precheck',
    '[{"id":"n_start","kind":"log","label":"commit-precheck begin","next":["n_bp_check_naming"]},{"id":"n_bp_check_naming","kind":"bp_check","label":"naming P0 check","params":{"bp_id":"bp-naming-p0"},"next":["n_detect_changes"]},{"id":"n_detect_changes","kind":"tool_call","label":"blast-radius via detect_changes","params":{"tool":"detect_changes"},"next":["n_threshold"]},{"id":"n_threshold","kind":"condition","label":"blast_radius > 50 ?","params":{"expr":"blast_radius > 50"},"next":["n_wait","n_log_done"]},{"id":"n_wait","kind":"wait_approval","label":"human approval required","next":["n_log_done"]},{"id":"n_log_done","kind":"log","label":"commit-precheck done","next":[]}]',
    'n_start',
    'published',
    1,
    'u_admin_default',
    NOW(),
    NOW()
);

INSERT INTO workflow_versions (workflow_id, version, snapshot_json, changed_by, change_note, created_at)
VALUES ('wf-commit-precheck', 1, '{"id":"wf-commit-precheck","status":"published","version":1}', 'system', 'initial seed', NOW());

-- WF2: ADR 双写
INSERT INTO workflows (workflow_id, name, description, track, category, nodes_json, entry_id, status, version, created_by, created_at, updated_at)
VALUES (
    'wf-adr-doublewrite',
    'ADR 双写',
    'MemPalace add_drawer 写入决策的同时，把同一份摘要 double-write 到 codebase-mem 的 manage_adr 节点。',
    'J',
    'governance',
    '[{"id":"n_start","kind":"log","label":"adr-doublewrite begin","next":["n_parallel"]},{"id":"n_parallel","kind":"parallel","label":"fan-out double-write","next":["n_add_drawer","n_manage_adr"]},{"id":"n_add_drawer","kind":"tool_call","label":"mempalace_add_drawer","params":{"tool":"mempalace_add_drawer"},"next":["n_join"]},{"id":"n_manage_adr","kind":"tool_call","label":"manage_adr create","params":{"tool":"manage_adr"},"next":["n_join"]},{"id":"n_join","kind":"log","label":"fan-in","next":[]}]',
    'n_start',
    'published',
    1,
    'u_admin_default',
    NOW(),
    NOW()
);

INSERT INTO workflow_versions (workflow_id, version, snapshot_json, changed_by, change_note, created_at)
VALUES ('wf-adr-doublewrite', 1, '{"id":"wf-adr-doublewrite","status":"published","version":1}', 'system', 'initial seed', NOW());

-- WF3: 仓库每日同步
INSERT INTO workflows (workflow_id, name, description, track, category, nodes_json, entry_id, status, version, created_by, created_at, updated_at)
VALUES (
    'wf-repo-daily-sync',
    '仓库每日同步',
    '每日触发 ingest_traces 把 runtime traces 写入 wing 内的 drawers；loop_guard 限 100 步。',
    'J',
    'sync',
    '[{"id":"n_start","kind":"log","label":"repo-daily-sync begin","next":["n_loop_guard"]},{"id":"n_loop_guard","kind":"loop_guard","label":"cap 100 iterations","max_iters":100,"next":["n_ingest"]},{"id":"n_ingest","kind":"tool_call","label":"ingest_traces","params":{"tool":"ingest_traces"},"next":["n_log_done"]},{"id":"n_log_done","kind":"log","label":"repo-daily-sync done","next":[]}]',
    'n_start',
    'published',
    1,
    'u_admin_default',
    NOW(),
    NOW()
);

INSERT INTO workflow_versions (workflow_id, version, snapshot_json, changed_by, change_note, created_at)
VALUES ('wf-repo-daily-sync', 1, '{"id":"wf-repo-daily-sync","status":"published","version":1}', 'system', 'initial seed', NOW());

-- =============================================================================
-- 9. 内置记忆模板 (5条)
-- =============================================================================

-- TPL1: ADR
INSERT INTO memory_templates (id, name, description, fields_json, body_template, is_builtin)
VALUES (
    'tpl_adr',
    'Architecture Decision Record (ADR)',
    'Capture one architectural decision with context, options, and consequences.',
    '[{"key":"title","label":"Title","type":"string","required":true},{"key":"status","label":"Status","type":"string","required":true,"help":"proposed | accepted | deprecated | superseded"},{"key":"context","label":"Context","type":"string","required":true,"help":"What is the issue we are addressing?"},{"key":"decision","label":"Decision","type":"string","required":true,"help":"What did we decide?"},{"key":"consequences","label":"Consequences","type":"string","required":false,"help":"What becomes easier / harder?"}]',
    '# {{.title}}\n\n> **Status**: {{.status}}\n> **Date**: {{.date}}\n\n## Context\n\n{{.context}}\n\n## Decision\n\n{{.decision}}\n\n## Consequences\n\n{{.consequences}}',
    1
);

-- TPL2: Lessons Learned
INSERT INTO memory_templates (id, name, description, fields_json, body_template, is_builtin)
VALUES (
    'tpl_lesson',
    'Lessons Learned',
    'Document a problem, root cause, fix, and long-term impact.',
    '[{"key":"problem","label":"Problem","type":"string","required":true},{"key":"root_cause","label":"Root Cause","type":"string","required":true},{"key":"solution","label":"Solution","type":"string","required":true},{"key":"impact","label":"Impact","type":"string","required":false,"help":"severity + blast radius"}]',
    '## Problem\n\n{{.problem}}\n\n## Root Cause\n\n{{.root_cause}}\n\n## Solution\n\n{{.solution}}\n\n## Impact\n\n{{.impact}}',
    1
);

-- TPL3: Code Snippet
INSERT INTO memory_templates (id, name, description, fields_json, body_template, is_builtin)
VALUES (
    'tpl_snippet',
    'Reusable Code Snippet',
    'Save a small, language-tagged code snippet with usage notes.',
    '[{"key":"language","label":"Language","type":"string","required":true,"help":"go | python | ts | sql | ..."},{"key":"description","label":"Description","type":"string","required":true},{"key":"code","label":"Code","type":"string","required":true}]',
    '**{{.language}}** — {{.description}}\n\n```{{.language}}\n{{.code}}\n```',
    1
);

-- TPL4: Runbook
INSERT INTO memory_templates (id, name, description, fields_json, body_template, is_builtin)
VALUES (
    'tpl_runbook',
    'Operational Runbook',
    'A trigger-and-steps manual for an on-call or ops procedure.',
    '[{"key":"trigger","label":"Trigger","type":"string","required":true,"help":"What symptom / alert fires this?"},{"key":"steps","label":"Steps","type":"string","required":true,"help":"Numbered steps to resolve"},{"key":"rollback","label":"Rollback","type":"string","required":false,"help":"How to undo if needed"}]',
    '## Trigger\n\n{{.trigger}}\n\n## Steps\n\n{{.steps}}\n\n## Rollback\n\n{{.rollback}}',
    1
);

-- TPL5: Lightweight Decision
INSERT INTO memory_templates (id, name, description, fields_json, body_template, is_builtin)
VALUES (
    'tpl_decision',
    'Lightweight Decision',
    'Capture a small decision without the full ADR ceremony.',
    '[{"key":"title","label":"Title","type":"string","required":true},{"key":"decision","label":"Decision","type":"string","required":true},{"key":"context","label":"Context","type":"string","required":false}]',
    '**{{.title}}**\n\n{{.decision}}\n\n{{if .context}}_Context_: {{.context}}{{end}}',
    1
);

-- =============================================================================
-- 10. 内置AI工具 (3条)
-- =============================================================================

INSERT INTO ai_tools (id, team_id, name, slug, description, endpoint, protocol, command, args_json, required_role, allowed_projects_json, timeout_seconds, enabled, created_at, updated_at)
VALUES (
    'aitool_mempalace',
    'team_default',
    'MemPalace',
    'mempalace',
    '知识宫殿 — 结构化长期记忆管理，支持 Wing/Room/Hall/Drawer 四级存储，语义搜索与自动归档。',
    '',
    'stdio',
    'mempalace',
    '[]',
    'developer',
    '[]',
    300,
    1,
    NOW(),
    NOW()
);

INSERT INTO ai_tools (id, team_id, name, slug, description, endpoint, protocol, command, args_json, required_role, allowed_projects_json, timeout_seconds, enabled, created_at, updated_at)
VALUES (
    'aitool_codebase_mem',
    'team_default',
    'Codebase Memory MCP',
    'codebase-memory-mcp',
    '代码库记忆 — 基于图数据库的代码索引、符号搜索、调用链追踪与架构分析。',
    '',
    'stdio',
    'codebase-memory-mcp',
    '[]',
    'developer',
    '[]',
    600,
    1,
    NOW(),
    NOW()
);

INSERT INTO ai_tools (id, team_id, name, slug, description, endpoint, protocol, command, args_json, required_role, allowed_projects_json, timeout_seconds, enabled, created_at, updated_at)
VALUES (
    'aitool_deep_research',
    'team_default',
    'Deep Research',
    'deep-research',
    '深度研究 — 多源网络搜索与文档综合分析报告生成。',
    'http://localhost:8090/research',
    'http',
    '',
    '[]',
    'lead',
    '[]',
    900,
    1,
    NOW(),
    NOW()
);

COMMIT;

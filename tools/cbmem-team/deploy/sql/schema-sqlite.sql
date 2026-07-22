-- =============================================================================
-- cbmem-team 完整 SQLite 建表语句 (v4)
-- 覆盖 M1-M6 全量表：legacy 7表 + M1工具目录 + M2团队/最佳实践 + M3工作流 + M4仓库流水线 + M5记忆模板 + M6 AI工具
-- =============================================================================
-- 引擎：SQLite 3.35+（支持 ALTER TABLE DROP COLUMN）
-- 外键：PRAGMA foreign_keys = ON（默认开启）
-- WAL模式：PRAGMA journal_mode = WAL（推荐）
--
-- 注意：
--   - 主键使用 INTEGER PRIMARY KEY AUTOINCREMENT（BIGINT 等价）
--   - 外键关联使用 INTEGER 类型
--   - FOREIGN KEY 引用统一使用 INTEGER
--
-- 应用顺序（严格按此顺序执行）：
--   1. sys_users
--   2. pm_teams
--   3. pm_team_members / pm_projects / pm_modules
--   4. ai_sessions / ai_session_turns / ai_summarize_tasks / ai_distill_tasks / sys_console_sessions
--   5. ai_memories_templates / ai_memories
--   6. ai_tools / ai_tool_invocations
--   7. sys_refresh_tokens / sys_audit_logs
--   8. ai_tool_directory / ai_tool_invocation_logs
--   9. pm_best_practices / pm_bp_versions
--  10. pm_workflows / pm_workflow_versions / pm_workflow_runs / pm_tickets
--  11. pm_repo_pipelines / pm_repo_pipeline_runs / pm_bp_candidates

PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;

BEGIN;

-- =============================================================================
-- 1. sys_users — 平台用户表
-- 主键使用 INTEGER PRIMARY KEY AUTOINCREMENT
-- =============================================================================
CREATE TABLE IF NOT EXISTS sys_users (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    username              TEXT NOT NULL DEFAULT '',
    display_name          TEXT NOT NULL DEFAULT '',
    email                 TEXT NOT NULL DEFAULT '',
    password_hash         TEXT NOT NULL DEFAULT '',
    default_team_id       INTEGER,
    role                  TEXT NOT NULL DEFAULT 'developer',
    must_change_password  INTEGER NOT NULL DEFAULT 1,
    disabled              INTEGER NOT NULL DEFAULT 0,
    created_at            DATETIME NOT NULL,
    updated_at            DATETIME,
    last_login_at         DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_sys_users_username ON sys_users(username);
CREATE INDEX IF NOT EXISTS idx_sys_users_default_team ON sys_users(default_team_id);
CREATE INDEX IF NOT EXISTS idx_sys_users_role ON sys_users(role);
CREATE INDEX IF NOT EXISTS idx_sys_users_disabled ON sys_users(disabled);

-- =============================================================================
-- 2. pm_teams — 团队表
-- 主键使用 INTEGER PRIMARY KEY AUTOINCREMENT
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_teams (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    owner_id    INTEGER,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    deleted     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_pm_teams_slug ON pm_teams(slug);
CREATE INDEX IF NOT EXISTS idx_pm_teams_owner ON pm_teams(owner_id);
CREATE INDEX IF NOT EXISTS idx_pm_teams_deleted ON pm_teams(deleted);

-- =============================================================================
-- 3. pm_team_members — 团队成员关联表
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_team_members (
    team_id   INTEGER NOT NULL,
    user_id   INTEGER NOT NULL,
    role      TEXT NOT NULL DEFAULT 'developer',
    joined_at DATETIME NOT NULL,
    PRIMARY KEY (team_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_pm_team_members_user ON pm_team_members(user_id);

-- =============================================================================
-- 4. pm_projects — 项目表
-- 主键使用 INTEGER PRIMARY KEY AUTOINCREMENT
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    path            TEXT NOT NULL UNIQUE,
    wing            TEXT,
    mcp_bin         TEXT,
    creator_id      INTEGER,
    team_id         INTEGER NOT NULL DEFAULT 0,
    slug            TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    git_url         TEXT NOT NULL DEFAULT '',
    git_branch      TEXT NOT NULL DEFAULT '',
    git_commit_sha  TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'ready',
    owner_id        INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    deleted         INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_pm_projects_creator ON pm_projects(creator_id);
CREATE INDEX IF NOT EXISTS idx_pm_projects_team ON pm_projects(team_id);
CREATE INDEX IF NOT EXISTS idx_pm_projects_status ON pm_projects(status);
CREATE INDEX IF NOT EXISTS idx_pm_projects_owner ON pm_projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_pm_projects_deleted ON pm_projects(deleted);
CREATE UNIQUE INDEX IF NOT EXISTS uk_pm_projects_team_slug ON pm_projects(team_id, slug);

-- =============================================================================
-- 5. pm_modules — 模块表
-- 主键使用 INTEGER PRIMARY KEY AUTOINCREMENT
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_modules (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id  INTEGER NOT NULL,
    parent_id   INTEGER,
    name        TEXT NOT NULL,
    path        TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    "order"     INTEGER NOT NULL DEFAULT 0,
    is_leaf     INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    deleted     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_pm_modules_project ON pm_modules(project_id);
CREATE INDEX IF NOT EXISTS idx_pm_modules_parent ON pm_modules(parent_id);
CREATE INDEX IF NOT EXISTS idx_pm_modules_is_leaf ON pm_modules(is_leaf);
CREATE INDEX IF NOT EXISTS idx_pm_modules_deleted ON pm_modules(deleted);
CREATE UNIQUE INDEX IF NOT EXISTS uk_pm_modules_parent_name ON pm_modules(project_id, parent_id, name);

-- =============================================================================
-- 6. ai_sessions — AI会话记录表
-- 主键使用 INTEGER PRIMARY KEY AUTOINCREMENT
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_sessions (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id                    INTEGER NOT NULL,
    team_id                    INTEGER NOT NULL DEFAULT 0,
    project_id                 INTEGER,
    module_id                  INTEGER NOT NULL DEFAULT 0,
    project_path               TEXT NOT NULL,
    started_at                 DATETIME NOT NULL,
    ended_at                   DATETIME,
    tool_count                 INTEGER NOT NULL DEFAULT 0,
    turn_count                 INTEGER NOT NULL DEFAULT 0,
    summary                    TEXT NOT NULL DEFAULT '',
    mempalace_synced_turns    INTEGER NOT NULL DEFAULT 0,
    mempalace_last_synced_at  DATETIME,
    mempalace_last_error      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_ai_sessions_user ON ai_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_sessions_project ON ai_sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_ai_sessions_started ON ai_sessions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_sessions_team ON ai_sessions(team_id);
CREATE INDEX IF NOT EXISTS idx_ai_sessions_module ON ai_sessions(module_id);
CREATE INDEX IF NOT EXISTS idx_ai_sessions_mempalace_pending ON ai_sessions(mempalace_synced_turns, turn_count);

-- =============================================================================
-- 7. ai_session_turns — 会话轮次详情表
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_session_turns (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  INTEGER NOT NULL,
    turn_no     INTEGER NOT NULL,
    role        TEXT NOT NULL,
    content     TEXT NOT NULL,
    tools_json  TEXT NOT NULL DEFAULT '',
    ts          DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_session_turns_session ON ai_session_turns(session_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_ai_session_turns_no ON ai_session_turns(session_id, turn_no);

-- =============================================================================
-- 8. ai_summarize_tasks — 总结任务表
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_summarize_tasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER,
    source_ids   TEXT NOT NULL DEFAULT '[]',
    depth        TEXT NOT NULL DEFAULT 'deep',
    target_wing  TEXT NOT NULL DEFAULT '',
    admin_id     INTEGER,
    status       TEXT NOT NULL DEFAULT 'pending',
    result_json  TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL,
    finished_at  DATETIME
);

CREATE INDEX IF NOT EXISTS idx_ai_summarize_tasks_user ON ai_summarize_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_summarize_tasks_status ON ai_summarize_tasks(status);

-- =============================================================================
-- 9. ai_distill_tasks — 提炼任务表
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_distill_tasks (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id           INTEGER,
    source_ids        TEXT NOT NULL DEFAULT '[]',
    rules_json        TEXT NOT NULL DEFAULT '{}',
    admin_id          INTEGER,
    status            TEXT NOT NULL DEFAULT 'pending',
    result_json       TEXT NOT NULL DEFAULT '',
    mempalace_synced  INTEGER NOT NULL DEFAULT 0,
    target_wing       TEXT NOT NULL DEFAULT '',
    created_at        DATETIME NOT NULL,
    finished_at       DATETIME
);

CREATE INDEX IF NOT EXISTS idx_ai_distill_tasks_user ON ai_distill_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_distill_tasks_status ON ai_distill_tasks(status);
CREATE INDEX IF NOT EXISTS idx_ai_distill_tasks_synced ON ai_distill_tasks(mempalace_synced);

-- =============================================================================
-- 10. sys_console_sessions — 控制台会话表
-- =============================================================================
CREATE TABLE IF NOT EXISTS sys_console_sessions (
    id           TEXT PRIMARY KEY,
    user_id      INTEGER NOT NULL,
    created_at   DATETIME NOT NULL,
    expires_at   DATETIME NOT NULL,
    last_seen_at DATETIME,
    ip           TEXT NOT NULL DEFAULT '',
    ua           TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sys_console_sessions_expires ON sys_console_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sys_console_sessions_user ON sys_console_sessions(user_id);

-- =============================================================================
-- 11. ai_memories_templates — 记忆模板表
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_memories_templates (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    fields_json   TEXT NOT NULL DEFAULT '[]',
    body_template TEXT NOT NULL DEFAULT '',
    is_builtin    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_memory_templates_builtin ON ai_memories_templates(is_builtin);

-- =============================================================================
-- 12. ai_memories — 记忆条目表
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_memories (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id      INTEGER NOT NULL,
    project_id   INTEGER NOT NULL,
    module_id    INTEGER NOT NULL,
    user_id      INTEGER NOT NULL,
    title        TEXT NOT NULL,
    content      TEXT NOT NULL,
    template_id  INTEGER,
    tags_json    TEXT NOT NULL DEFAULT '[]',
    hall         TEXT NOT NULL DEFAULT 'facts',
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL,
    deleted      INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_memories_team ON ai_memories(team_id);
CREATE INDEX IF NOT EXISTS idx_memories_project ON ai_memories(project_id);
CREATE INDEX IF NOT EXISTS idx_memories_module ON ai_memories(module_id);
CREATE INDEX IF NOT EXISTS idx_memories_user ON ai_memories(user_id);
CREATE INDEX IF NOT EXISTS idx_memories_hall ON ai_memories(hall);
CREATE INDEX IF NOT EXISTS idx_memories_template ON ai_memories(template_id);
CREATE INDEX IF NOT EXISTS idx_memories_updated ON ai_memories(updated_at DESC);

-- =============================================================================
-- 13. sys_refresh_tokens — JWT刷新令牌表
-- =============================================================================
CREATE TABLE IF NOT EXISTS sys_refresh_tokens (
    id          TEXT PRIMARY KEY,
    user_id     INTEGER NOT NULL,
    token_hash  TEXT NOT NULL,
    issued_at   DATETIME NOT NULL,
    expires_at  DATETIME NOT NULL,
    revoked_at  DATETIME
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON sys_refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON sys_refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON sys_refresh_tokens(token_hash);

-- =============================================================================
-- 14. sys_audit_logs — 审计日志表
-- =============================================================================
CREATE TABLE IF NOT EXISTS sys_audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    ts          DATETIME NOT NULL,
    user_id     INTEGER,
    team_id     INTEGER,
    kind        TEXT NOT NULL,
    target_id   TEXT NOT NULL DEFAULT '',
    meta_json   TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_audit_ts ON sys_audit_logs(ts DESC);
CREATE INDEX IF NOT EXISTS idx_audit_user ON sys_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_kind ON sys_audit_logs(kind);

-- =============================================================================
-- 15. ai_tool_directory — 工具目录表 (M1)
-- tool_id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_tool_directory (
    tool_id              TEXT PRIMARY KEY,
    track                TEXT NOT NULL,
    category             TEXT NOT NULL,
    name                 TEXT NOT NULL,
    signature            TEXT,
    status               TEXT NOT NULL DEFAULT 'active',
    priority             INTEGER NOT NULL DEFAULT 5,
    rate_limit_json      TEXT,
    related_halls_json   TEXT,
    related_wings_json   TEXT,
    version              TEXT NOT NULL DEFAULT '1.0.0',
    created_at           DATETIME NOT NULL,
    updated_at           DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_tool_directory_track ON ai_tool_directory(track, category);
CREATE INDEX IF NOT EXISTS idx_ai_tool_directory_status ON ai_tool_directory(status);

-- =============================================================================
-- 16. ai_tool_invocation_logs — 工具调用日志表 (M1)
-- invocation_id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_tool_invocation_logs (
    invocation_id            TEXT PRIMARY KEY,
    user_id                  INTEGER NOT NULL,
    project_id               INTEGER,
    project_path             TEXT,
    tool_id                  TEXT NOT NULL,
    transport                TEXT NOT NULL DEFAULT 'stdio',
    args_json                TEXT,
    response_json            TEXT,
    started_at               DATETIME NOT NULL,
    ended_at                 DATETIME,
    latency_ms               INTEGER,
    error_code               TEXT,
    timeout_flag             INTEGER NOT NULL DEFAULT 0,
    workflow_run_id          TEXT,
    blast_radius_json        TEXT,
    affected_drawers_json    TEXT,
    affected_halls_json      TEXT,
    affected_adrs_json       TEXT,
    wing_unauthorized_json   TEXT,
    client_ip                TEXT,
    jwt_sub                  TEXT
);

CREATE INDEX IF NOT EXISTS idx_inv_user_started ON ai_tool_invocation_logs(user_id, started_at);
CREATE INDEX IF NOT EXISTS idx_inv_tool_started ON ai_tool_invocation_logs(tool_id, started_at);
CREATE INDEX IF NOT EXISTS idx_inv_project_started ON ai_tool_invocation_logs(project_id, started_at);
CREATE INDEX IF NOT EXISTS idx_inv_workflow ON ai_tool_invocation_logs(workflow_run_id);
CREATE INDEX IF NOT EXISTS idx_inv_error ON ai_tool_invocation_logs(error_code);

-- =============================================================================
-- 17. pm_best_practices — 最佳实践表 (M2)
-- id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_best_practices (
    id           TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    category     TEXT NOT NULL,
    track        TEXT NOT NULL CHECK (track IN ('A','B','J')),
    tools        TEXT NOT NULL DEFAULT '[]',
    related_halls TEXT NOT NULL DEFAULT '[]',
    related_wings TEXT NOT NULL DEFAULT '[]',
    scenes       TEXT NOT NULL DEFAULT '[]',
    priority     TEXT NOT NULL DEFAULT 'P3',
    status       TEXT NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','published','deprecated','review_due')),
    version      INTEGER NOT NULL DEFAULT 1,
    created_by   INTEGER NOT NULL,
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL,
    review_due   DATE NOT NULL,
    source       TEXT NOT NULL DEFAULT 'manual',
    body         TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bp_status_priority ON pm_best_practices(status, priority);
CREATE INDEX IF NOT EXISTS idx_bp_track_category ON pm_best_practices(track, category);
CREATE INDEX IF NOT EXISTS idx_bp_tools_first ON pm_best_practices(json_extract(tools, '$[0]'));

-- =============================================================================
-- 18. pm_bp_versions — 最佳实践版本历史表 (M2)
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_bp_versions (
    bp_id         TEXT NOT NULL,
    version       INTEGER NOT NULL,
    snapshot_json TEXT NOT NULL,
    changed_by    INTEGER NOT NULL,
    change_note   TEXT,
    created_at    DATETIME NOT NULL,
    PRIMARY KEY (bp_id, version)
);

CREATE INDEX IF NOT EXISTS idx_bpv_created ON pm_bp_versions(bp_id, created_at);

-- =============================================================================
-- 19. ai_tools — AI工具表 (M6)
-- id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_tools (
    id                     TEXT PRIMARY KEY,
    team_id                INTEGER NOT NULL,
    name                   TEXT NOT NULL,
    slug                   TEXT NOT NULL,
    description            TEXT NOT NULL DEFAULT '',
    endpoint               TEXT NOT NULL,
    protocol               TEXT NOT NULL DEFAULT 'http',
    command                TEXT NOT NULL DEFAULT '',
    args_json              TEXT NOT NULL DEFAULT '[]',
    required_role          TEXT NOT NULL DEFAULT 'developer',
    allowed_projects_json  TEXT NOT NULL DEFAULT '[]',
    timeout_seconds        INTEGER NOT NULL DEFAULT 300,
    enabled                INTEGER NOT NULL DEFAULT 1,
    created_at             DATETIME NOT NULL,
    updated_at             DATETIME NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_ai_tools_team_slug ON ai_tools(team_id, slug);
CREATE INDEX IF NOT EXISTS idx_ai_tools_enabled ON ai_tools(enabled);

-- =============================================================================
-- 20. ai_tool_invocations — AI工具调用记录表 (M6)
-- id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_tool_invocations (
    id           TEXT PRIMARY KEY,
    tool_id      TEXT NOT NULL,
    user_id      INTEGER NOT NULL,
    team_id      INTEGER NOT NULL,
    project_id   INTEGER NOT NULL,
    module_id    INTEGER NOT NULL,
    working_dir  TEXT NOT NULL,
    input_json   TEXT NOT NULL DEFAULT '{}',
    output_json  TEXT NOT NULL DEFAULT '{}',
    status       TEXT NOT NULL DEFAULT 'pending',
    started_at   DATETIME NOT NULL,
    finished_at  DATETIME,
    error        TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_invocations_tool ON ai_tool_invocations(tool_id);
CREATE INDEX IF NOT EXISTS idx_invocations_user ON ai_tool_invocations(user_id);
CREATE INDEX IF NOT EXISTS idx_invocations_team ON ai_tool_invocations(team_id);
CREATE INDEX IF NOT EXISTS idx_invocations_project ON ai_tool_invocations(project_id);
CREATE INDEX IF NOT EXISTS idx_invocations_started ON ai_tool_invocations(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_invocations_status ON ai_tool_invocations(status);

-- =============================================================================
-- 21. pm_workflows — 工作流定义表 (M3)
-- workflow_id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_workflows (
    workflow_id  TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    description  TEXT,
    track        TEXT NOT NULL DEFAULT 'J',
    category     TEXT NOT NULL DEFAULT 'governance',
    nodes_json   TEXT NOT NULL,
    entry_id     TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'draft',
    version      INTEGER NOT NULL DEFAULT 1,
    created_by   INTEGER NOT NULL,
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_status ON pm_workflows(status, category);

-- =============================================================================
-- 22. pm_workflow_versions — 工作流版本历史表 (M3)
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_workflow_versions (
    workflow_id    TEXT NOT NULL,
    version        INTEGER NOT NULL,
    snapshot_json  TEXT NOT NULL,
    changed_by     INTEGER NOT NULL,
    change_note   TEXT,
    created_at     DATETIME NOT NULL,
    PRIMARY KEY (workflow_id, version)
);

CREATE INDEX IF NOT EXISTS idx_wfv_created ON pm_workflow_versions(workflow_id, created_at);

-- =============================================================================
-- 23. pm_workflow_runs — 工作流运行记录表 (M3)
-- run_id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_workflow_runs (
    run_id             TEXT PRIMARY KEY,
    workflow_id        TEXT NOT NULL,
    version            INTEGER NOT NULL,
    triggered_by       INTEGER NOT NULL,
    project_path       TEXT,
    dry_run            INTEGER NOT NULL DEFAULT 0,
    status             TEXT NOT NULL DEFAULT 'pending',
    started_at         DATETIME NOT NULL,
    ended_at          DATETIME,
    step_count        INTEGER NOT NULL DEFAULT 0,
    step_history_json TEXT,
    error              TEXT
);

CREATE INDEX IF NOT EXISTS idx_wfrun_workflow ON pm_workflow_runs(workflow_id, started_at);
CREATE INDEX IF NOT EXISTS idx_wfrun_status ON pm_workflow_runs(status, started_at);
CREATE INDEX IF NOT EXISTS idx_wfrun_trigger ON pm_workflow_runs(triggered_by, started_at);

-- =============================================================================
-- 24. pm_tickets — 问题工单表 (M3)
-- ticket_id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_tickets (
    ticket_id        TEXT PRIMARY KEY,
    source           TEXT NOT NULL DEFAULT 'manual',
    invocation_id    TEXT,
    tool_id          TEXT NOT NULL,
    user_id          INTEGER,
    severity         TEXT NOT NULL DEFAULT 'medium',
    status           TEXT NOT NULL DEFAULT 'open',
    title            TEXT NOT NULL,
    detail_json      TEXT,
    resolution_note  TEXT,
    created_at       DATETIME NOT NULL,
    updated_at       DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ticket_status ON pm_tickets(status, created_at);
CREATE INDEX IF NOT EXISTS idx_ticket_invocation ON pm_tickets(invocation_id);

-- =============================================================================
-- 25. pm_repo_pipelines — 仓库流水线表 (M4)
-- pipeline_id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_repo_pipelines (
    pipeline_id  TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    source      TEXT NOT NULL DEFAULT 'local',
    target      TEXT NOT NULL,
    cron_expr   TEXT,
    threshold   REAL NOT NULL DEFAULT 0.7,
    status      TEXT NOT NULL DEFAULT 'draft',
    created_by  INTEGER NOT NULL,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rp_status ON pm_repo_pipelines(status, source);

-- =============================================================================
-- 26. pm_repo_pipeline_runs — 仓库流水线运行记录表 (M4)
-- run_id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_repo_pipeline_runs (
    run_id           TEXT PRIMARY KEY,
    pipeline_id      TEXT NOT NULL,
    stage_status    TEXT,
    items_ingested   INTEGER NOT NULL DEFAULT 0,
    items_parsed     INTEGER NOT NULL DEFAULT 0,
    items_graded     INTEGER NOT NULL DEFAULT 0,
    items_accepted  INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'pending',
    started_at      DATETIME NOT NULL,
    ended_at       DATETIME,
    error            TEXT
);

CREATE INDEX IF NOT EXISTS idx_rprun_pipeline ON pm_repo_pipeline_runs(pipeline_id, started_at);
CREATE INDEX IF NOT EXISTS idx_rprun_status ON pm_repo_pipeline_runs(status, started_at);

-- =============================================================================
-- 27. pm_bp_candidates — 最佳实践候选表 (M4)
-- candidate_id 使用 TEXT 因为这是业务标识符
-- =============================================================================
CREATE TABLE IF NOT EXISTS pm_bp_candidates (
    candidate_id TEXT PRIMARY KEY,
    pipeline_id  TEXT NOT NULL,
    run_id       TEXT NOT NULL,
    title        TEXT NOT NULL,
    body         TEXT NOT NULL,
    grade_score  REAL NOT NULL DEFAULT 0.0,
    source_url   TEXT,
    tags_json    TEXT,
    status       TEXT NOT NULL DEFAULT 'draft',
    merged_bp_id TEXT,
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bpc_status ON pm_bp_candidates(status, created_at);
CREATE INDEX IF NOT EXISTS idx_bpc_run ON pm_bp_candidates(run_id);
CREATE INDEX IF NOT EXISTS idx_bpc_pipeline ON pm_bp_candidates(pipeline_id);

COMMIT;

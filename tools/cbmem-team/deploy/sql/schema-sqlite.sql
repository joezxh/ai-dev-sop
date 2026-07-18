-- =============================================================================
-- cbmem-team 完整 SQLite 建表语句 (v2)
-- 覆盖 M1-M6 全量表：legacy 7表 + M1工具目录 + M2团队/最佳实践 + M3工作流 + M4仓库流水线 + M5记忆模板 + M6 AI工具
-- =============================================================================
-- 引擎：SQLite 3.35+（支持 ALTER TABLE DROP COLUMN）
-- 外键：PRAGMA foreign_keys = ON（默认开启）
-- WAL模式：PRAGMA journal_mode = WAL（推荐）
--
-- 应用顺序（严格按此顺序执行）：
--   1. legacy 7表 (users/projects/sessions/session_turns/summarize_tasks/distill_tasks/console_sessions)
--   2. M1: tool_directory / tool_invocation_logs
--   3. M2: teams / team_members / modules / best_practices / bp_versions
--   4. M3: workflows / workflow_versions / workflow_runs / tickets
--   5. M4: repo_pipelines / repo_pipeline_runs / bp_candidates
--   6. M5: memory_templates / memories
--   6. M6: ai_tools / ai_tool_invocations
--   7. refresh_tokens / audit_logs

PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;

BEGIN;

-- =============================================================================
-- 1. users — 平台用户表
-- =============================================================================
CREATE TABLE IF NOT EXISTS users (
    id                     TEXT PRIMARY KEY,
    username              TEXT NOT NULL DEFAULT '',
    display_name          TEXT NOT NULL DEFAULT '',
    email                 TEXT NOT NULL DEFAULT '',
    password_hash         TEXT NOT NULL DEFAULT '!v1-legacy-must-reset!',
    default_team_id       TEXT,
    role                  TEXT NOT NULL DEFAULT 'developer',
    must_change_password  INTEGER NOT NULL DEFAULT 1,
    disabled              INTEGER NOT NULL DEFAULT 0,
    created_at            DATETIME NOT NULL,
    updated_at            DATETIME,
    last_login_at         DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_default_team ON users(default_team_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_disabled ON users(disabled);

-- =============================================================================
-- 2. projects — 项目表
-- =============================================================================
CREATE TABLE IF NOT EXISTS projects (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    path            TEXT NOT NULL UNIQUE,
    wing            TEXT,
    mcp_bin         TEXT,
    creator_id      TEXT,
    team_id         TEXT NOT NULL DEFAULT '',
    slug            TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    git_url         TEXT NOT NULL DEFAULT '',
    git_branch      TEXT NOT NULL DEFAULT '',
    git_commit_sha  TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'ready',
    owner_id        TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    deleted         INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_projects_creator ON projects(creator_id);
CREATE INDEX IF NOT EXISTS idx_projects_team ON projects(team_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_projects_deleted ON projects(deleted);
CREATE UNIQUE INDEX IF NOT EXISTS uk_projects_team_slug ON projects(team_id, slug);

-- =============================================================================
-- 3. sessions — AI会话记录表
-- =============================================================================
CREATE TABLE IF NOT EXISTS sessions (
    id                         TEXT PRIMARY KEY,
    user_id                    TEXT NOT NULL,
    team_id                    TEXT NOT NULL DEFAULT '',
    project_id                 TEXT,
    module_id                  TEXT NOT NULL DEFAULT '',
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

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_team ON sessions(team_id);
CREATE INDEX IF NOT EXISTS idx_sessions_module ON sessions(module_id);
CREATE INDEX IF NOT EXISTS idx_sessions_mempalace_pending ON sessions(mempalace_synced_turns, turn_count);

-- =============================================================================
-- 4. session_turns — 会话轮次详情表
-- =============================================================================
CREATE TABLE IF NOT EXISTS session_turns (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL,
    turn_no     INTEGER NOT NULL,
    role        TEXT NOT NULL,
    content     TEXT NOT NULL,
    tools_json  TEXT NOT NULL DEFAULT '',
    ts          DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_session_turns_session ON session_turns(session_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_session_turns_no ON session_turns(session_id, turn_no);

-- =============================================================================
-- 5. summarize_tasks — 总结任务表
-- =============================================================================
CREATE TABLE IF NOT EXISTS summarize_tasks (
    id           TEXT PRIMARY KEY,
    user_id      TEXT,
    source_ids   TEXT NOT NULL DEFAULT '[]',
    depth        TEXT NOT NULL DEFAULT 'deep',
    target_wing  TEXT NOT NULL DEFAULT '',
    admin_id     TEXT,
    status       TEXT NOT NULL DEFAULT 'pending',
    result_json  TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL,
    finished_at  DATETIME
);

CREATE INDEX IF NOT EXISTS idx_summarize_tasks_user ON summarize_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_summarize_tasks_status ON summarize_tasks(status);

-- =============================================================================
-- 6. distill_tasks — 提炼任务表
-- =============================================================================
CREATE TABLE IF NOT EXISTS distill_tasks (
    id                TEXT PRIMARY KEY,
    user_id           TEXT,
    source_ids        TEXT NOT NULL DEFAULT '[]',
    rules_json        TEXT NOT NULL DEFAULT '{}',
    admin_id          TEXT,
    status            TEXT NOT NULL DEFAULT 'pending',
    result_json       TEXT NOT NULL DEFAULT '',
    mempalace_synced  INTEGER NOT NULL DEFAULT 0,
    target_wing       TEXT NOT NULL DEFAULT '',
    created_at        DATETIME NOT NULL,
    finished_at       DATETIME
);

CREATE INDEX IF NOT EXISTS idx_distill_tasks_user ON distill_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_distill_tasks_status ON distill_tasks(status);
CREATE INDEX IF NOT EXISTS idx_distill_tasks_synced ON distill_tasks(mempalace_synced);

-- =============================================================================
-- 7. console_sessions — 控制台会话表
-- =============================================================================
CREATE TABLE IF NOT EXISTS console_sessions (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    created_at   DATETIME NOT NULL,
    expires_at   DATETIME NOT NULL,
    last_seen_at DATETIME,
    ip           TEXT NOT NULL DEFAULT '',
    ua           TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_console_sessions_expires ON console_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_console_sessions_user ON console_sessions(user_id);

-- =============================================================================
-- 8. tool_directory — 工具目录表 (M1)
-- =============================================================================
CREATE TABLE IF NOT EXISTS tool_directory (
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

CREATE INDEX IF NOT EXISTS idx_tool_directory_track ON tool_directory(track, category);
CREATE INDEX IF NOT EXISTS idx_tool_directory_status ON tool_directory(status);

-- =============================================================================
-- 9. tool_invocation_logs — 工具调用日志表 (M1)
-- =============================================================================
CREATE TABLE IF NOT EXISTS tool_invocation_logs (
    invocation_id            TEXT PRIMARY KEY,
    user_id                  TEXT NOT NULL,
    project_id               TEXT,
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

CREATE INDEX IF NOT EXISTS idx_inv_user_started ON tool_invocation_logs(user_id, started_at);
CREATE INDEX IF NOT EXISTS idx_inv_tool_started ON tool_invocation_logs(tool_id, started_at);
CREATE INDEX IF NOT EXISTS idx_inv_project_started ON tool_invocation_logs(project_id, started_at);
CREATE INDEX IF NOT EXISTS idx_inv_workflow ON tool_invocation_logs(workflow_run_id);
CREATE INDEX IF NOT EXISTS idx_inv_error ON tool_invocation_logs(error_code);

-- =============================================================================
-- 10. teams — 团队表 (M2)
-- =============================================================================
CREATE TABLE IF NOT EXISTS teams (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    owner_id    TEXT,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    deleted     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_teams_slug ON teams(slug);
CREATE INDEX IF NOT EXISTS idx_teams_owner ON teams(owner_id);
CREATE INDEX IF NOT EXISTS idx_teams_deleted ON teams(deleted);

-- =============================================================================
-- 11. team_members — 团队成员关联表 (M2)
-- =============================================================================
CREATE TABLE IF NOT EXISTS team_members (
    team_id   TEXT NOT NULL,
    user_id   TEXT NOT NULL,
    role      TEXT NOT NULL DEFAULT 'developer',
    joined_at DATETIME NOT NULL,
    PRIMARY KEY (team_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id);

-- =============================================================================
-- 12. modules — 模块表 (M2)
-- =============================================================================
CREATE TABLE IF NOT EXISTS modules (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL,
    parent_id   TEXT,
    name        TEXT NOT NULL,
    path        TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    "order"     INTEGER NOT NULL DEFAULT 0,
    is_leaf     INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    deleted     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_modules_project ON modules(project_id);
CREATE INDEX IF NOT EXISTS idx_modules_parent ON modules(parent_id);
CREATE INDEX IF NOT EXISTS idx_modules_is_leaf ON modules(is_leaf);
CREATE INDEX IF NOT EXISTS idx_modules_deleted ON modules(deleted);
CREATE UNIQUE INDEX IF NOT EXISTS uk_modules_parent_name ON modules(project_id, parent_id, name);

-- =============================================================================
-- 13. best_practices — 最佳实践表 (M2)
-- =============================================================================
CREATE TABLE IF NOT EXISTS best_practices (
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
    created_by   TEXT NOT NULL,
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL,
    review_due   DATE NOT NULL,
    source       TEXT NOT NULL DEFAULT 'manual',
    body         TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bp_status_priority ON best_practices(status, priority);
CREATE INDEX IF NOT EXISTS idx_bp_track_category ON best_practices(track, category);
CREATE INDEX IF NOT EXISTS idx_bp_tools_first ON best_practices(json_extract(tools, '$[0]'));

-- =============================================================================
-- 14. bp_versions — 最佳实践版本历史表 (M2)
-- =============================================================================
CREATE TABLE IF NOT EXISTS bp_versions (
    bp_id         TEXT NOT NULL,
    version       INTEGER NOT NULL,
    snapshot_json TEXT NOT NULL,
    changed_by   TEXT NOT NULL,
    change_note   TEXT,
    created_at    DATETIME NOT NULL,
    PRIMARY KEY (bp_id, version)
);

CREATE INDEX IF NOT EXISTS idx_bpv_created ON bp_versions(bp_id, created_at);

-- =============================================================================
-- 15. memory_templates — 记忆模板表 (M5)
-- =============================================================================
CREATE TABLE IF NOT EXISTS memory_templates (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    fields_json   TEXT NOT NULL DEFAULT '[]',
    body_template TEXT NOT NULL DEFAULT '',
    is_builtin    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_memory_templates_builtin ON memory_templates(is_builtin);

-- =============================================================================
-- 16. memories — 记忆条目表 (M5)
-- =============================================================================
CREATE TABLE IF NOT EXISTS memories (
    id           TEXT PRIMARY KEY,
    team_id      TEXT NOT NULL,
    project_id   TEXT NOT NULL,
    module_id    TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    title        TEXT NOT NULL,
    content      TEXT NOT NULL,
    template_id  TEXT,
    tags_json    TEXT NOT NULL DEFAULT '[]',
    hall         TEXT NOT NULL DEFAULT 'facts',
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL,
    deleted      INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_memories_team ON memories(team_id);
CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project_id);
CREATE INDEX IF NOT EXISTS idx_memories_module ON memories(module_id);
CREATE INDEX IF NOT EXISTS idx_memories_user ON memories(user_id);
CREATE INDEX IF NOT EXISTS idx_memories_hall ON memories(hall);
CREATE INDEX IF NOT EXISTS idx_memories_template ON memories(template_id);
CREATE INDEX IF NOT EXISTS idx_memories_updated ON memories(updated_at DESC);

-- =============================================================================
-- 17. refresh_tokens — JWT刷新令牌表
-- =============================================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    token_hash  TEXT NOT NULL,
    issued_at   DATETIME NOT NULL,
    expires_at  DATETIME NOT NULL,
    revoked_at  DATETIME
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);

-- =============================================================================
-- 18. audit_logs — 审计日志表
-- =============================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    ts          DATETIME NOT NULL,
    user_id     TEXT,
    team_id     TEXT,
    kind        TEXT NOT NULL,
    target_id   TEXT NOT NULL DEFAULT '',
    meta_json   TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_logs(ts DESC);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_team ON audit_logs(team_id);
CREATE INDEX IF NOT EXISTS idx_audit_kind ON audit_logs(kind);

-- =============================================================================
-- 19. ai_tools — AI工具表 (M6)
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_tools (
    id                     TEXT PRIMARY KEY,
    team_id                TEXT NOT NULL,
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
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_tool_invocations (
    id           TEXT PRIMARY KEY,
    tool_id      TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    team_id      TEXT NOT NULL,
    project_id   TEXT NOT NULL,
    module_id    TEXT NOT NULL,
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
-- 21. workflows — 工作流定义表 (M3)
-- =============================================================================
CREATE TABLE IF NOT EXISTS workflows (
    workflow_id  TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    description  TEXT,
    track        TEXT NOT NULL DEFAULT 'J',
    category     TEXT NOT NULL DEFAULT 'governance',
    nodes_json   TEXT NOT NULL,
    entry_id     TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'draft',
    version      INTEGER NOT NULL DEFAULT 1,
    created_by   TEXT NOT NULL DEFAULT 'anonymous',
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_status ON workflows(status, category);

-- =============================================================================
-- 22. workflow_versions — 工作流版本历史表 (M3)
-- =============================================================================
CREATE TABLE IF NOT EXISTS workflow_versions (
    workflow_id    TEXT NOT NULL,
    version        INTEGER NOT NULL,
    snapshot_json  TEXT NOT NULL,
    changed_by    TEXT NOT NULL DEFAULT 'anonymous',
    change_note   TEXT,
    created_at     DATETIME NOT NULL,
    PRIMARY KEY (workflow_id, version)
);

CREATE INDEX IF NOT EXISTS idx_wfv_created ON workflow_versions(workflow_id, created_at);

-- =============================================================================
-- 23. workflow_runs — 工作流运行记录表 (M3)
-- =============================================================================
CREATE TABLE IF NOT EXISTS workflow_runs (
    run_id             TEXT PRIMARY KEY,
    workflow_id        TEXT NOT NULL,
    version            INTEGER NOT NULL,
    triggered_by       TEXT NOT NULL,
    project_path       TEXT,
    dry_run            INTEGER NOT NULL DEFAULT 0,
    status             TEXT NOT NULL DEFAULT 'pending',
    started_at         DATETIME NOT NULL,
    ended_at          DATETIME,
    step_count        INTEGER NOT NULL DEFAULT 0,
    step_history_json TEXT,
    error              TEXT
);

CREATE INDEX IF NOT EXISTS idx_wfrun_workflow ON workflow_runs(workflow_id, started_at);
CREATE INDEX IF NOT EXISTS idx_wfrun_status ON workflow_runs(status, started_at);
CREATE INDEX IF NOT EXISTS idx_wfrun_trigger ON workflow_runs(triggered_by, started_at);

-- =============================================================================
-- 24. tickets — 问题工单表 (M3)
-- =============================================================================
CREATE TABLE IF NOT EXISTS tickets (
    ticket_id        TEXT PRIMARY KEY,
    source           TEXT NOT NULL DEFAULT 'manual',
    invocation_id    TEXT,
    tool_id          TEXT NOT NULL,
    user_id          TEXT,
    severity         TEXT NOT NULL DEFAULT 'medium',
    status           TEXT NOT NULL DEFAULT 'open',
    title            TEXT NOT NULL,
    detail_json      TEXT,
    resolution_note  TEXT,
    created_at       DATETIME NOT NULL,
    updated_at       DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ticket_status ON tickets(status, created_at);
CREATE INDEX IF NOT EXISTS idx_ticket_invocation ON tickets(invocation_id);

-- =============================================================================
-- 25. repo_pipelines — 仓库流水线表 (M4)
-- =============================================================================
CREATE TABLE IF NOT EXISTS repo_pipelines (
    pipeline_id  TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    source      TEXT NOT NULL DEFAULT 'local',
    target      TEXT NOT NULL,
    cron_expr   TEXT,
    threshold   REAL NOT NULL DEFAULT 0.7,
    status      TEXT NOT NULL DEFAULT 'draft',
    created_by  TEXT NOT NULL DEFAULT 'anonymous',
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rp_status ON repo_pipelines(status, source);

-- =============================================================================
-- 26. repo_pipeline_runs — 仓库流水线运行记录表 (M4)
-- =============================================================================
CREATE TABLE IF NOT EXISTS repo_pipeline_runs (
    run_id           TEXT PRIMARY KEY,
    pipeline_id      TEXT NOT NULL,
    stage_status    TEXT,
    items_ingested   INTEGER NOT NULL DEFAULT 0,
    items_parsed    INTEGER NOT NULL DEFAULT 0,
    items_graded    INTEGER NOT NULL DEFAULT 0,
    items_accepted  INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'pending',
    started_at      DATETIME NOT NULL,
    ended_at       DATETIME,
    error            TEXT
);

CREATE INDEX IF NOT EXISTS idx_rprun_pipeline ON repo_pipeline_runs(pipeline_id, started_at);
CREATE INDEX IF NOT EXISTS idx_rprun_status ON repo_pipeline_runs(status, started_at);

-- =============================================================================
-- 27. bp_candidates — 最佳实践候选表 (M4)
-- =============================================================================
CREATE TABLE IF NOT EXISTS bp_candidates (
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

CREATE INDEX IF NOT EXISTS idx_bpc_status ON bp_candidates(status, created_at);
CREATE INDEX IF NOT EXISTS idx_bpc_run ON bp_candidates(run_id);
CREATE INDEX IF NOT EXISTS idx_bpc_pipeline ON bp_candidates(pipeline_id);

COMMIT;

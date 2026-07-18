-- =============================================================================
-- cbmem-team 完整 MySQL 8.0 建表语句 (v2)
-- 覆盖 M1-M6 全量表：legacy 7表 + M1工具目录 + M2团队/最佳实践 + M3工作流 + M4仓库流水线 + M5记忆模板 + M6 AI工具
-- =============================================================================
-- 编码：utf8mb4_unicode_ci
-- 引擎：InnoDB
--
-- 注意事项：
--   - TEXT/MEDIUMTEXT/LONGTEXT 列不能有 DEFAULT 值，应用层负责提供默认值
--   - VARCHAR(255) 用于短文本，TEXT 用于长文本
--   - JSON 列在 MySQL 5.7+ 支持，DEFAULT 用 NOT NULL DEFAULT (JSON_OBJECT())
--
-- 应用顺序（严格按此顺序执行，因存在外键依赖）：
--   1. users / teams (循环FK延迟到末尾)
--   2. team_members / projects / modules
--   3. sessions / session_turns / summarize_tasks / distill_tasks / console_sessions
--   4. memory_templates / memories
--   5. ai_tools / ai_tool_invocations
--   6. refresh_tokens / audit_logs
--   7. tool_directory / tool_invocation_logs
--   8. best_practices / bp_versions
--   9. workflows / workflow_versions / workflow_runs / tickets
--  10. repo_pipelines / repo_pipeline_runs / bp_candidates
--  11. 外键约束（循环FK在此添加）
--  12. views

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- =============================================================================
-- 1. users — 平台用户表
-- =============================================================================
CREATE TABLE IF NOT EXISTS users (
    id                     VARCHAR(128) NOT NULL PRIMARY KEY,
    username              VARCHAR(128) NOT NULL,
    display_name          VARCHAR(255) NOT NULL DEFAULT '',
    email                 VARCHAR(255) NOT NULL DEFAULT '',
    password_hash         VARCHAR(255) NOT NULL DEFAULT '',
    default_team_id       VARCHAR(128) NULL,
    role                  VARCHAR(32)  NOT NULL DEFAULT 'developer',
    must_change_password  TINYINT(1)   NOT NULL DEFAULT 1,
    disabled              TINYINT(1)   NOT NULL DEFAULT 0,
    created_at            DATETIME(0)  NOT NULL,
    updated_at            DATETIME(0)  NULL,
    last_login_at         DATETIME(0)  NULL,
    UNIQUE KEY uk_users_username (username),
    KEY idx_users_default_team (default_team_id),
    KEY idx_users_role (role),
    KEY idx_users_disabled (disabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 2. teams — 团队表
-- =============================================================================
CREATE TABLE IF NOT EXISTS teams (
    id          VARCHAR(128) NOT NULL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(128) NOT NULL,
    description TEXT         NOT NULL,
    owner_id    VARCHAR(128) NULL,
    created_at  DATETIME(0)  NOT NULL,
    updated_at  DATETIME(0)  NOT NULL,
    deleted     TINYINT(1)   NOT NULL DEFAULT 0,
    UNIQUE KEY uk_teams_slug (slug),
    KEY idx_teams_owner (owner_id),
    KEY idx_teams_deleted (deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 3. team_members — 团队成员关联表
-- =============================================================================
CREATE TABLE IF NOT EXISTS team_members (
    team_id   VARCHAR(128) NOT NULL,
    user_id   VARCHAR(128) NOT NULL,
    role      VARCHAR(32)  NOT NULL DEFAULT 'developer',
    joined_at DATETIME(0)  NOT NULL,
    PRIMARY KEY (team_id, user_id),
    KEY idx_team_members_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 4. projects — 项目表
-- =============================================================================
CREATE TABLE IF NOT EXISTS projects (
    id              VARCHAR(128) NOT NULL PRIMARY KEY,
    team_id         VARCHAR(128) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(128) NOT NULL,
    description     TEXT         NOT NULL,
    path            VARCHAR(512) NOT NULL,
    wing            VARCHAR(128) NOT NULL DEFAULT '',
    mcp_bin         VARCHAR(512) NOT NULL DEFAULT '',
    creator_id      VARCHAR(128) NULL,
    git_url         VARCHAR(512) NOT NULL DEFAULT '',
    git_branch      VARCHAR(128) NOT NULL DEFAULT '',
    git_commit_sha  VARCHAR(64)  NOT NULL DEFAULT '',
    status          VARCHAR(32)  NOT NULL DEFAULT 'ready',
    owner_id        VARCHAR(128) NULL,
    created_at      DATETIME(0)  NOT NULL,
    updated_at      DATETIME(0)  NOT NULL,
    deleted         TINYINT(1)   NOT NULL DEFAULT 0,
    UNIQUE KEY uk_projects_path (path),
    UNIQUE KEY uk_projects_team_slug (team_id, slug),
    KEY idx_projects_creator (creator_id),
    KEY idx_projects_team (team_id),
    KEY idx_projects_owner (owner_id),
    KEY idx_projects_status (status),
    KEY idx_projects_deleted (deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 5. modules — 模块表（树形结构）
-- =============================================================================
CREATE TABLE IF NOT EXISTS modules (
    id          VARCHAR(128) NOT NULL PRIMARY KEY,
    project_id  VARCHAR(128) NOT NULL,
    parent_id   VARCHAR(128) NULL,
    name        VARCHAR(255) NOT NULL,
    path        VARCHAR(512) NOT NULL DEFAULT '',
    description TEXT         NOT NULL,
    `order`     INT          NOT NULL DEFAULT 0,
    is_leaf     TINYINT(1)   NOT NULL DEFAULT 1,
    created_at  DATETIME(0)  NOT NULL,
    updated_at  DATETIME(0)  NOT NULL,
    deleted     TINYINT(1)   NOT NULL DEFAULT 0,
    KEY idx_modules_project (project_id),
    KEY idx_modules_parent (parent_id),
    KEY idx_modules_is_leaf (is_leaf),
    KEY idx_modules_deleted (deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 6. sessions — AI会话记录表
-- =============================================================================
CREATE TABLE IF NOT EXISTS sessions (
    id                        VARCHAR(128) NOT NULL PRIMARY KEY,
    user_id                   VARCHAR(128) NOT NULL,
    team_id                   VARCHAR(128) NOT NULL,
    project_id                VARCHAR(128) NOT NULL,
    module_id                 VARCHAR(128) NOT NULL,
    project_path              VARCHAR(512) NOT NULL,
    started_at                DATETIME(0)  NOT NULL,
    ended_at                  DATETIME(0)  NULL,
    tool_count                INT          NOT NULL DEFAULT 0,
    turn_count                INT          NOT NULL DEFAULT 0,
    summary                   TEXT         NOT NULL,
    mempalace_synced_turns   INT          NOT NULL DEFAULT 0,
    mempalace_last_synced_at DATETIME(0)  NULL,
    mempalace_last_error     TEXT         NOT NULL,
    KEY idx_sessions_user (user_id),
    KEY idx_sessions_team (team_id),
    KEY idx_sessions_project (project_id),
    KEY idx_sessions_module (module_id),
    KEY idx_sessions_started (started_at DESC),
    KEY idx_sessions_mempalace_pending (mempalace_synced_turns, turn_count)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 7. session_turns — 会话轮次详情表
-- =============================================================================
CREATE TABLE IF NOT EXISTS session_turns (
    id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    session_id  VARCHAR(128) NOT NULL,
    turn_no     INT          NOT NULL,
    role        VARCHAR(32)  NOT NULL,
    content     MEDIUMTEXT   NOT NULL,
    tools_json  TEXT         NOT NULL,
    ts          DATETIME(0)  NOT NULL,
    UNIQUE KEY uk_session_turns_no (session_id, turn_no),
    KEY idx_session_turns_session (session_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 8. summarize_tasks — 总结任务表
-- =============================================================================
CREATE TABLE IF NOT EXISTS summarize_tasks (
    id           VARCHAR(128) NOT NULL PRIMARY KEY,
    user_id      VARCHAR(128) NULL,
    source_ids   JSON         NOT NULL,
    depth        VARCHAR(16)  NOT NULL DEFAULT 'deep',
    target_wing  VARCHAR(128) NOT NULL DEFAULT '',
    status       VARCHAR(16)  NOT NULL DEFAULT 'pending',
    result_json  LONGTEXT     NOT NULL,
    created_at   DATETIME(0)  NOT NULL,
    finished_at  DATETIME(0)  NULL,
    KEY idx_summarize_tasks_user (user_id),
    KEY idx_summarize_tasks_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 9. distill_tasks — 提炼任务表
-- =============================================================================
CREATE TABLE IF NOT EXISTS distill_tasks (
    id                VARCHAR(128) NOT NULL PRIMARY KEY,
    user_id           VARCHAR(128) NULL,
    source_ids        JSON         NOT NULL,
    rules_json        JSON         NOT NULL,
    status            VARCHAR(16)  NOT NULL DEFAULT 'pending',
    result_json       LONGTEXT     NOT NULL,
    mempalace_synced  TINYINT(1)   NOT NULL DEFAULT 0,
    target_wing       VARCHAR(128) NOT NULL DEFAULT '',
    created_at        DATETIME(0)  NOT NULL,
    finished_at       DATETIME(0)  NULL,
    KEY idx_distill_tasks_user (user_id),
    KEY idx_distill_tasks_status (status),
    KEY idx_distill_tasks_synced (mempalace_synced)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 10. console_sessions — 控制台会话表（cookie认证）
-- =============================================================================
CREATE TABLE IF NOT EXISTS console_sessions (
    id           VARCHAR(128) NOT NULL PRIMARY KEY,
    user_id      VARCHAR(128) NOT NULL,
    created_at   DATETIME(0)  NOT NULL,
    expires_at   DATETIME(0)  NOT NULL,
    last_seen_at DATETIME(0)  NULL,
    ip           VARCHAR(64)  NOT NULL DEFAULT '',
    ua           VARCHAR(512) NOT NULL DEFAULT '',
    KEY idx_console_sessions_expires (expires_at),
    KEY idx_console_sessions_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 11. memory_templates — 记忆模板表
-- =============================================================================
CREATE TABLE IF NOT EXISTS memory_templates (
    id            VARCHAR(128) NOT NULL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    description   TEXT         NOT NULL,
    fields_json   JSON         NOT NULL,
    body_template MEDIUMTEXT   NOT NULL,
    is_builtin    TINYINT(1)   NOT NULL DEFAULT 0,
    KEY idx_memory_templates_builtin (is_builtin)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 12. memories — 记忆条目表
-- =============================================================================
CREATE TABLE IF NOT EXISTS memories (
    id           VARCHAR(128) NOT NULL PRIMARY KEY,
    team_id      VARCHAR(128) NOT NULL,
    project_id   VARCHAR(128) NOT NULL,
    module_id    VARCHAR(128) NOT NULL,
    user_id      VARCHAR(128) NOT NULL,
    title        VARCHAR(255) NOT NULL,
    content      MEDIUMTEXT   NOT NULL,
    template_id  VARCHAR(128) NULL,
    tags_json    JSON         NOT NULL,
    hall         VARCHAR(32)  NOT NULL DEFAULT 'facts',
    created_at   DATETIME(0)  NOT NULL,
    updated_at   DATETIME(0)  NOT NULL,
    deleted      TINYINT(1)   NOT NULL DEFAULT 0,
    KEY idx_memories_team (team_id),
    KEY idx_memories_project (project_id),
    KEY idx_memories_module (module_id),
    KEY idx_memories_user (user_id),
    KEY idx_memories_hall (hall),
    KEY idx_memories_template (template_id),
    KEY idx_memories_updated (updated_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 13. refresh_tokens — JWT刷新令牌表
-- =============================================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          VARCHAR(128) NOT NULL PRIMARY KEY,
    user_id     VARCHAR(128) NOT NULL,
    token_hash  VARCHAR(255) NOT NULL,
    issued_at   DATETIME(0)  NOT NULL,
    expires_at  DATETIME(0)  NOT NULL,
    revoked_at  DATETIME(0)  NULL,
    KEY idx_refresh_tokens_user (user_id),
    KEY idx_refresh_tokens_expires (expires_at),
    KEY idx_refresh_tokens_hash (token_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 14. audit_logs — 审计日志表
-- =============================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    ts          DATETIME(0)  NOT NULL,
    user_id     VARCHAR(128) NULL,
    team_id     VARCHAR(128) NULL,
    kind        VARCHAR(64)  NOT NULL,
    target_id   VARCHAR(128) NOT NULL DEFAULT '',
    meta_json   JSON         NOT NULL,
    KEY idx_audit_ts (ts DESC),
    KEY idx_audit_user (user_id),
    KEY idx_audit_team (team_id),
    KEY idx_audit_kind (kind)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 15. tool_directory — 工具目录表 (M1)
-- =============================================================================
CREATE TABLE IF NOT EXISTS tool_directory (
    tool_id              VARCHAR(128) NOT NULL PRIMARY KEY,
    track                VARCHAR(64)  NOT NULL,
    category             VARCHAR(64)  NOT NULL,
    name                 VARCHAR(255) NOT NULL,
    signature            TEXT,
    status               VARCHAR(32)  NOT NULL DEFAULT 'active',
    priority             INT          NOT NULL DEFAULT 5,
    rate_limit_json      JSON,
    related_halls_json   JSON,
    related_wings_json   JSON,
    version              VARCHAR(32)  NOT NULL DEFAULT '1.0.0',
    created_at           DATETIME(0)  NOT NULL,
    updated_at           DATETIME(0)  NOT NULL,
    KEY idx_tool_directory_track (track, category),
    KEY idx_tool_directory_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 16. tool_invocation_logs — 工具调用日志表 (M1)
-- =============================================================================
CREATE TABLE IF NOT EXISTS tool_invocation_logs (
    invocation_id            VARCHAR(64)  NOT NULL PRIMARY KEY,
    user_id                  VARCHAR(128) NOT NULL,
    project_id               VARCHAR(128) NULL,
    project_path             VARCHAR(512) NULL,
    tool_id                  VARCHAR(128) NOT NULL,
    transport                VARCHAR(16)  NOT NULL DEFAULT 'stdio',
    args_json                LONGTEXT,
    response_json            LONGTEXT,
    started_at               DATETIME(3)  NOT NULL,
    ended_at                 DATETIME(3)  NULL,
    latency_ms               INT,
    error_code               VARCHAR(64),
    timeout_flag             TINYINT(1)   NOT NULL DEFAULT 0,
    workflow_run_id          VARCHAR(64),
    blast_radius_json        LONGTEXT,
    affected_drawers_json    LONGTEXT,
    affected_halls_json      LONGTEXT,
    affected_adrs_json       LONGTEXT,
    wing_unauthorized_json   LONGTEXT,
    client_ip                VARCHAR(64),
    jwt_sub                  VARCHAR(128),
    KEY idx_inv_user_started (user_id, started_at),
    KEY idx_inv_tool_started (tool_id, started_at),
    KEY idx_inv_project_started (project_id, started_at),
    KEY idx_inv_workflow (workflow_run_id),
    KEY idx_inv_error (error_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 17. best_practices — 最佳实践表 (M2)
-- =============================================================================
CREATE TABLE IF NOT EXISTS best_practices (
    id           VARCHAR(64)  NOT NULL PRIMARY KEY,
    title        VARCHAR(160) NOT NULL,
    category     VARCHAR(32)  NOT NULL,
    track        VARCHAR(8)   NOT NULL,
    tools        JSON         NOT NULL,
    related_halls JSON       NOT NULL,
    related_wings JSON       NOT NULL,
    scenes       JSON         NOT NULL,
    priority     VARCHAR(8)   NOT NULL DEFAULT 'P3',
    status       VARCHAR(16)  NOT NULL DEFAULT 'draft',
    version      INT          NOT NULL DEFAULT 1,
    created_by   VARCHAR(128) NOT NULL,
    created_at   DATETIME(0)  NOT NULL,
    updated_at   DATETIME(0)  NOT NULL,
    review_due   DATE         NOT NULL,
    source       VARCHAR(64)  NOT NULL DEFAULT 'manual',
    body         LONGTEXT     NOT NULL,
    CHECK (track IN ('A','B','J')),
    CHECK (status IN ('draft','published','deprecated','review_due'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MySQL 多值索引：加速 tools JSON 数组的包含查询
CREATE INDEX idx_bp_tools_mv ON best_practices((CAST(tools AS CHAR(64) ARRAY)));

-- =============================================================================
-- 18. bp_versions — 最佳实践版本历史表 (M2)
-- =============================================================================
CREATE TABLE IF NOT EXISTS bp_versions (
    bp_id         VARCHAR(64)  NOT NULL,
    version       INT          NOT NULL,
    snapshot_json LONGTEXT     NOT NULL,
    changed_by    VARCHAR(128) NOT NULL,
    change_note   VARCHAR(512),
    created_at    DATETIME(0) NOT NULL,
    PRIMARY KEY (bp_id, version),
    KEY idx_bpv_created (bp_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 19. ai_tools — AI工具表 (M6)
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_tools (
    id                     VARCHAR(128) NOT NULL PRIMARY KEY,
    team_id                VARCHAR(128) NOT NULL,
    name                   VARCHAR(255) NOT NULL,
    slug                   VARCHAR(128) NOT NULL,
    description            TEXT         NOT NULL,
    endpoint               VARCHAR(512) NOT NULL,
    protocol               VARCHAR(16)  NOT NULL DEFAULT 'http',
    command                VARCHAR(255) NOT NULL DEFAULT '',
    args_json              JSON         NOT NULL,
    required_role          VARCHAR(32)  NOT NULL DEFAULT 'developer',
    allowed_projects_json  JSON         NOT NULL,
    timeout_seconds        INT          NOT NULL DEFAULT 300,
    enabled                TINYINT(1)   NOT NULL DEFAULT 1,
    created_at             DATETIME(0)  NOT NULL,
    updated_at              DATETIME(0)  NOT NULL,
    UNIQUE KEY uk_ai_tools_team_slug (team_id, slug),
    KEY idx_ai_tools_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 20. ai_tool_invocations — AI工具调用记录表 (M6)
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_tool_invocations (
    id           VARCHAR(128) NOT NULL PRIMARY KEY,
    tool_id      VARCHAR(128) NOT NULL,
    user_id      VARCHAR(128) NOT NULL,
    team_id      VARCHAR(128) NOT NULL,
    project_id   VARCHAR(128) NOT NULL,
    module_id    VARCHAR(128) NOT NULL,
    working_dir  VARCHAR(512) NOT NULL,
    input_json   JSON         NOT NULL,
    output_json  JSON         NOT NULL,
    status       VARCHAR(16)  NOT NULL DEFAULT 'pending',
    started_at   DATETIME(0)  NOT NULL,
    finished_at  DATETIME(0)  NULL,
    error        TEXT         NOT NULL,
    KEY idx_invocations_tool (tool_id),
    KEY idx_invocations_user (user_id),
    KEY idx_invocations_team (team_id),
    KEY idx_invocations_project (project_id),
    KEY idx_invocations_started (started_at DESC),
    KEY idx_invocations_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 21. workflows — 工作流定义表 (M3)
-- =============================================================================
CREATE TABLE IF NOT EXISTS workflows (
    workflow_id  VARCHAR(64)  NOT NULL PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    track        VARCHAR(8)   NOT NULL DEFAULT 'J',
    category     VARCHAR(64)  NOT NULL DEFAULT 'governance',
    nodes_json   LONGTEXT     NOT NULL,
    entry_id     VARCHAR(64)  NOT NULL,
    status       VARCHAR(16)  NOT NULL DEFAULT 'draft',
    version      INT          NOT NULL DEFAULT 1,
    created_by   VARCHAR(128) NOT NULL DEFAULT 'anonymous',
    created_at   DATETIME(3) NOT NULL,
    updated_at   DATETIME(3) NOT NULL,
    KEY idx_workflow_status (status, category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 22. workflow_versions — 工作流版本历史表 (M3)
-- =============================================================================
CREATE TABLE IF NOT EXISTS workflow_versions (
    workflow_id    VARCHAR(64)  NOT NULL,
    version        INT          NOT NULL,
    snapshot_json  LONGTEXT     NOT NULL,
    changed_by     VARCHAR(128) NOT NULL DEFAULT 'anonymous',
    change_note    TEXT,
    created_at     DATETIME(3) NOT NULL,
    PRIMARY KEY (workflow_id, version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 23. workflow_runs — 工作流运行记录表 (M3)
-- =============================================================================
CREATE TABLE IF NOT EXISTS workflow_runs (
    run_id             VARCHAR(64)  NOT NULL PRIMARY KEY,
    workflow_id        VARCHAR(64)  NOT NULL,
    version            INT          NOT NULL,
    triggered_by       VARCHAR(128) NOT NULL,
    project_path       VARCHAR(512) NULL,
    dry_run            TINYINT(1)   NOT NULL DEFAULT 0,
    status             VARCHAR(16)  NOT NULL DEFAULT 'pending',
    started_at         DATETIME(3) NOT NULL,
    ended_at          DATETIME(3) NULL,
    step_count        INT          NOT NULL DEFAULT 0,
    step_history_json LONGTEXT,
    error              TEXT,
    KEY idx_wfrun_workflow (workflow_id, started_at),
    KEY idx_wfrun_status (status, started_at),
    KEY idx_wfrun_trigger (triggered_by, started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 24. tickets — 问题工单表 (M3)
-- =============================================================================
CREATE TABLE IF NOT EXISTS tickets (
    ticket_id        VARCHAR(64)  NOT NULL PRIMARY KEY,
    source           VARCHAR(64)  NOT NULL DEFAULT 'manual',
    invocation_id    VARCHAR(64),
    tool_id          VARCHAR(128) NOT NULL,
    user_id          VARCHAR(128),
    severity         VARCHAR(16)  NOT NULL DEFAULT 'medium',
    status           VARCHAR(16)  NOT NULL DEFAULT 'open',
    title            VARCHAR(255) NOT NULL,
    detail_json      LONGTEXT,
    resolution_note  TEXT,
    created_at       DATETIME(3) NOT NULL,
    updated_at       DATETIME(3) NOT NULL,
    KEY idx_ticket_status (status, created_at),
    KEY idx_ticket_invocation (invocation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 25. repo_pipelines — 仓库流水线表 (M4)
-- =============================================================================
CREATE TABLE IF NOT EXISTS repo_pipelines (
    pipeline_id  VARCHAR(64)  NOT NULL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    source      VARCHAR(16)  NOT NULL DEFAULT 'local',
    target      VARCHAR(1024) NOT NULL,
    cron_expr   VARCHAR(64),
    threshold   DOUBLE        NOT NULL DEFAULT 0.7,
    status      VARCHAR(16)  NOT NULL DEFAULT 'draft',
    created_by  VARCHAR(128) NOT NULL DEFAULT 'anonymous',
    created_at  DATETIME(3) NOT NULL,
    updated_at  DATETIME(3) NOT NULL,
    KEY idx_rp_status (status, source)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 26. repo_pipeline_runs — 仓库流水线运行记录表 (M4)
-- =============================================================================
CREATE TABLE IF NOT EXISTS repo_pipeline_runs (
    run_id           VARCHAR(64)  NOT NULL PRIMARY KEY,
    pipeline_id      VARCHAR(64)  NOT NULL,
    stage_status    JSON,
    items_ingested   INT          NOT NULL DEFAULT 0,
    items_parsed    INT          NOT NULL DEFAULT 0,
    items_graded    INT          NOT NULL DEFAULT 0,
    items_accepted  INT          NOT NULL DEFAULT 0,
    status          VARCHAR(16)  NOT NULL DEFAULT 'pending',
    started_at      DATETIME(3) NOT NULL,
    ended_at       DATETIME(3) NULL,
    error            TEXT,
    KEY idx_rprun_pipeline (pipeline_id, started_at),
    KEY idx_rprun_status (status, started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 27. bp_candidates — 最佳实践候选表 (M4)
-- =============================================================================
CREATE TABLE IF NOT EXISTS bp_candidates (
    candidate_id VARCHAR(64)  NOT NULL PRIMARY KEY,
    pipeline_id  VARCHAR(64)  NOT NULL,
    run_id       VARCHAR(64)  NOT NULL,
    title        VARCHAR(255) NOT NULL,
    body         LONGTEXT     NOT NULL,
    grade_score  DOUBLE       NOT NULL DEFAULT 0.0,
    source_url   VARCHAR(1024),
    tags_json    JSON,
    status       VARCHAR(16)  NOT NULL DEFAULT 'draft',
    merged_bp_id VARCHAR(64),
    created_at   DATETIME(3) NOT NULL,
    updated_at   DATETIME(3) NOT NULL,
    KEY idx_bpc_status (status, created_at),
    KEY idx_bpc_run (run_id),
    KEY idx_bpc_pipeline (pipeline_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 循环外键约束（延迟到所有表创建完成后）
-- =============================================================================
ALTER TABLE users ADD CONSTRAINT fk_users_default_team FOREIGN KEY (default_team_id) REFERENCES teams(id) ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE teams ADD CONSTRAINT fk_teams_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE projects ADD CONSTRAINT fk_projects_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE projects ADD CONSTRAINT fk_projects_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE modules ADD CONSTRAINT fk_modules_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE modules ADD CONSTRAINT fk_modules_parent FOREIGN KEY (parent_id) REFERENCES modules(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE sessions ADD CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE sessions ADD CONSTRAINT fk_sessions_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE sessions ADD CONSTRAINT fk_sessions_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE sessions ADD CONSTRAINT fk_sessions_module FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE session_turns ADD CONSTRAINT fk_session_turns_session FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE memories ADD CONSTRAINT fk_memories_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE memories ADD CONSTRAINT fk_memories_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE memories ADD CONSTRAINT fk_memories_module FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE memories ADD CONSTRAINT fk_memories_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE memories ADD CONSTRAINT fk_memories_template FOREIGN KEY (template_id) REFERENCES memory_templates(id) ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE refresh_tokens ADD CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE console_sessions ADD CONSTRAINT fk_console_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE team_members ADD CONSTRAINT fk_team_members_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE team_members ADD CONSTRAINT fk_team_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE summarize_tasks ADD CONSTRAINT fk_summarize_tasks_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE distill_tasks ADD CONSTRAINT fk_distill_tasks_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE bp_versions ADD CONSTRAINT fk_bpv_bp FOREIGN KEY (bp_id) REFERENCES best_practices(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE ai_tools ADD CONSTRAINT fk_ai_tools_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE ai_tool_invocations ADD CONSTRAINT fk_invocations_tool FOREIGN KEY (tool_id) REFERENCES ai_tools(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE ai_tool_invocations ADD CONSTRAINT fk_invocations_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE ai_tool_invocations ADD CONSTRAINT fk_invocations_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE ai_tool_invocations ADD CONSTRAINT fk_invocations_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE ai_tool_invocations ADD CONSTRAINT fk_invocations_module FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE workflow_versions ADD CONSTRAINT fk_wfv_workflow FOREIGN KEY (workflow_id) REFERENCES workflows(workflow_id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE workflow_runs ADD CONSTRAINT fk_wfrun_workflow FOREIGN KEY (workflow_id) REFERENCES workflows(workflow_id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE tickets ADD CONSTRAINT fk_tickets_invocation FOREIGN KEY (invocation_id) REFERENCES tool_invocation_logs(invocation_id) ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE tickets ADD CONSTRAINT fk_tickets_tool FOREIGN KEY (tool_id) REFERENCES tool_directory(tool_id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE tickets ADD CONSTRAINT fk_tickets_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE repo_pipeline_runs ADD CONSTRAINT fk_rprun_pipeline FOREIGN KEY (pipeline_id) REFERENCES repo_pipelines(pipeline_id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE bp_candidates ADD CONSTRAINT fk_bpc_pipeline FOREIGN KEY (pipeline_id) REFERENCES repo_pipelines(pipeline_id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE bp_candidates ADD CONSTRAINT fk_bpc_run FOREIGN KEY (run_id) REFERENCES repo_pipeline_runs(run_id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE bp_candidates ADD CONSTRAINT fk_bpc_merged FOREIGN KEY (merged_bp_id) REFERENCES best_practices(id) ON DELETE SET NULL ON UPDATE CASCADE;

-- =============================================================================
-- Views
-- =============================================================================

DROP VIEW IF EXISTS v_high_risk_invocations_7d;
CREATE VIEW v_high_risk_invocations_7d AS
SELECT
    invocation_id, user_id, project_id, tool_id, transport,
    started_at, latency_ms, error_code,
    affected_halls_json, affected_adrs_json, blast_radius_json
FROM tool_invocation_logs
WHERE started_at >= (NOW() - INTERVAL 7 DAY)
  AND (
    JSON_LENGTH(affected_halls_json) > 0
    OR JSON_LENGTH(affected_adrs_json) > 0
    OR JSON_LENGTH(blast_radius_json) > 0
    OR error_code IN ('UPSTREAM_ERROR','FORBIDDEN','TIMEOUT')
  );

DROP VIEW IF EXISTS v_repo_pipeline_success_rate_7d;
CREATE VIEW v_repo_pipeline_success_rate_7d AS
SELECT
    pipeline_id,
    SUM(CASE WHEN status = 'succeeded' THEN 1 ELSE 0 END) AS ok_runs,
    COUNT(*) AS total_runs,
    ROUND(SUM(CASE WHEN status = 'succeeded' THEN 1 ELSE 0 END) / NULLIF(COUNT(*),0), 3) AS success_rate,
    MAX(started_at) AS last_started_at
FROM repo_pipeline_runs
WHERE started_at >= (NOW() - INTERVAL 7 DAY)
GROUP BY pipeline_id;

SET FOREIGN_KEY_CHECKS = 1;

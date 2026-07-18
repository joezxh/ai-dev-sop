package console

// v2 domain models — direct 1:1 mapping with the ER diagram in
// docs/superpowers/diagrams/cbmem-team-v2-erd.md.
//
// All time fields use time.Time so the dialect layer (SQLite uses
// "YYYY-MM-DD HH:MM:SS" strings, MySQL uses DATETIME) can format them
// consistently on write and parse them with sql.NullTime on read.
//
// JSON tags use snake_case to match the existing envelope convention in
// wrap.go. Bcrypt cost and JWT TTL live in cmd/cbmem-team/main.go and
// are not duplicated here.

import "time"

// ----------------------------------------------------------------------------
// Roles & statuses — kept as typed strings so handlers can `if u.Role != RoleAdmin`
// without juggling string literals. CHECK constraints in the SQL match these.
// ----------------------------------------------------------------------------

// Role type and constants live in rbac.go (RoleAdmin / RoleLead / etc.)
// where the type itself is defined. The string aliases below are
// convenient for handlers that only need to compare against a literal:
//
//     if u.Role != string(RoleAdmin) { ... }
//
// The CHECK constraints in deploy/sql/v2_schema.*.sql match these.

const (
	ProjectStatusCloning = "cloning"
	ProjectStatusIndexing = "indexing"
	ProjectStatusReady   = "ready"
	ProjectStatusError   = "error"

	InvocationStatusPending = "pending"
	InvocationStatusRunning = "running"
	InvocationStatusSuccess = "success"
	InvocationStatusError   = "error"
	InvocationStatusTimeout = "timeout"

	ToolProtocolHTTP   = "http"
	ToolProtocolStdio  = "stdio"
)

// User mirrors the v2 users table. DefaultTeamID is denormalised for
// convenience — the canonical user↔team mapping lives in team_members.
type User struct {
	ID                  string    `json:"id"`
	Username            string    `json:"username"`
	DisplayName         string    `json:"display_name"`
	Email               string    `json:"email"`
	PasswordHash        string    `json:"-"` // never serialise
	DefaultTeamID       string    `json:"default_team_id,omitempty"`
	Role                string    `json:"role"`
	MustChangePassword  bool      `json:"must_change_password"`
	Disabled            bool      `json:"disabled"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
}

// Team is the top-level container that owns projects and members.
type Team struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	OwnerID     string    `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Deleted     bool      `json:"deleted"`

	// Populated by ListTeamsHandler / GetTeamHandler; not stored.
	MemberCount  int `json:"member_count,omitempty"`
	ProjectCount int `json:"project_count,omitempty"`
}

// TeamMember is the many-to-many join row. Role here is the role WITHIN
// this team and may differ from users.role (the platform-level role).
// Username is populated at read time via ListTeamMembers /
// GetTeamMember (LEFT JOIN users); it is not stored on the row.
type TeamMember struct {
	TeamID   string    `json:"team_id"`
	UserID   string    `json:"user_id"`
	Username string    `json:"username,omitempty"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// Project belongs to exactly one team. Path is globally unique so the
// legacy project_paths whitelist can be retired; the git clone happens
// in a goroutine after the row is inserted and Status flips cloning →
// indexing → ready.
type Project struct {
	ID           string    `json:"id"`
	TeamID       string    `json:"team_id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description"`
	Path         string    `json:"path"`
	GitURL       string    `json:"git_url,omitempty"`
	GitBranch    string    `json:"git_branch,omitempty"`
	GitCommitSHA string    `json:"git_commit_sha,omitempty"`
	Status       string    `json:"status"`
	OwnerID      string    `json:"owner_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Deleted      bool      `json:"deleted"`
}

// Module is the recursive tree under a project. ParentID nil means
// "root". IsLeaf is maintained by the application layer on every
// create/delete and is the gate for sessions/memories writes.
type Module struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ParentID    *string   `json:"parent_id,omitempty"`
	Name        string    `json:"name"`
	Path        string    `json:"path,omitempty"`
	Description string    `json:"description,omitempty"`
	Order       int       `json:"order"`
	IsLeaf      bool      `json:"is_leaf"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Deleted     bool      `json:"deleted"`

	// Populated by tree handlers; not stored.
	ChildCount   int `json:"child_count,omitempty"`
	SessionCount int `json:"session_count,omitempty"`
	MemoryCount  int `json:"memory_count,omitempty"`
	Depth        int `json:"depth,omitempty"`
}

// Session captures one rolling MCP conversation. ModuleID is the leaf
// module the conversation belongs to and is required (the v1 capture
// middleware pre-dates modules; the v2 middleware upgrades it).
type Session struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	TeamID          string     `json:"team_id"`
	ProjectID       string     `json:"project_id"`
	ModuleID        string     `json:"module_id"`
	ProjectPath     string     `json:"project_path"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	ToolCount       int        `json:"tool_count"`
	TurnCount       int        `json:"turn_count"`
	Summary         string     `json:"summary,omitempty"`

	// Auto-sync bookkeeping (added 2026-07-15). MemPalaceSyncedTurns is
	// the watermark; anything > this number has been pushed. LastError
	// is cleared on the next successful sync.
	MemPalaceSyncedTurns  int        `json:"mempalace_synced_turns"`
	MemPalaceLastSyncedAt *time.Time `json:"mempalace_last_synced_at,omitempty"`
	MemPalaceLastError    string     `json:"mempalace_last_error,omitempty"`
}

// SessionTurn is one user/assistant/tool exchange inside a Session.
type SessionTurn struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	TurnNo    int       `json:"turn_no"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	ToolsJSON string    `json:"tools_json,omitempty"`
	TS        time.Time `json:"ts"`
}

// MemoryTemplate is a structured form a user fills in before committing
// a memory. FieldsJSON is the list of form fields with their type
// (string|list|date) and whether they're required; BodyTemplate is a
// Go text/template that gets rendered with the field values.
type MemoryTemplate struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	FieldsJSON   string `json:"fields_json"`
	BodyTemplate string `json:"body_template"`
	IsBuiltin    bool   `json:"is_builtin"`
}

// Memory is a piece of long-lived knowledge written to a leaf module.
// Hall tags it for MemPalace-style 5-Hall classification; tags_json is
// a flat string array used by the search index.
type Memory struct {
	ID         string    `json:"id"`
	TeamID     string    `json:"team_id"`
	ProjectID  string    `json:"project_id"`
	ModuleID   string    `json:"module_id"`
	UserID     string    `json:"user_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	TemplateID string    `json:"template_id,omitempty"`
	TagsJSON   string    `json:"tags_json,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Hall       string    `json:"hall"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Deleted    bool      `json:"deleted"`
}

// AITool is a third-party AI capability registered for a team. Two
// protocols: HTTP (forward request to Endpoint) and stdio (spawn
// Command+Args). RequiredRole is the minimum users.role a caller needs;
// AllowedProjects, when non-empty, restricts invocations to those
// project IDs.
type AITool struct {
	ID                string    `json:"id"`
	TeamID            string    `json:"team_id"`
	Name              string    `json:"name"`
	Slug              string    `json:"slug"`
	Description       string    `json:"description"`
	Endpoint          string    `json:"endpoint"`
	Protocol          string    `json:"protocol"`
	Command           string    `json:"command,omitempty"`
	ArgsJSON          string    `json:"args_json,omitempty"`
	RequiredRole      string    `json:"required_role"`
	AllowedProjectsJSON string `json:"allowed_projects_json,omitempty"`
	TimeoutSeconds    int       `json:"timeout_seconds"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// AIToolInvocation records one execution of an AI tool. WorkingDir is
// the project path the tool ran against; Status progresses
// pending → running → success/error/timeout. Error is populated on
// failure and cleared on the next success.
type AIToolInvocation struct {
	ID         string     `json:"id"`
	ToolID     string     `json:"tool_id"`
	UserID     string     `json:"user_id"`
	TeamID     string     `json:"team_id"`
	ProjectID  string     `json:"project_id"`
	ModuleID   string     `json:"module_id"`
	WorkingDir string     `json:"working_dir"`
	InputJSON  string     `json:"input_json,omitempty"`
	OutputJSON string     `json:"output_json,omitempty"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// ConsoleSession is the web login cookie (separate from MCP JWT). Used
// by the /api/console UI to authenticate CSRF-protected calls. v2 keeps
// the same shape as v1; only the user_id → role mapping is enforced.
type ConsoleSession struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastSeenAt time.Time `json:"last_seen_at,omitempty"`
	IP         string    `json:"ip,omitempty"`
	UA         string    `json:"ua,omitempty"`
}

// RefreshToken is the long-lived credential paired with an access JWT.
// Stored on disk so we can revoke them (logout / password change).
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// SummarizeTask / DistillTask — v1 carried these as JSON columns; v2
// keeps the same on-disk shape (status + result_json blob) so existing
// SQL stays valid. They are populated by the manual UI buttons; the
// auto-sync path in capture.go writes Drawers directly and does not
// create tasks here.
type SummarizeTask struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	SourceIDs   string     `json:"source_ids"`
	Depth       string     `json:"depth"`
	TargetWing  string     `json:"target_wing"`
	Status      string     `json:"status"`
	ResultJSON  string     `json:"result_json,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

type DistillTask struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	SourceIDs       string     `json:"source_ids"`
	RulesJSON       string     `json:"rules_json"`
	Status          string     `json:"status"`
	ResultJSON      string     `json:"result_json,omitempty"`
	MemPalaceSynced bool       `json:"mempalace_synced"`
	TargetWing      string     `json:"target_wing"`
	CreatedAt       time.Time  `json:"created_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}

// ----------------------------------------------------------------------------
// Computed types — populated by the tree/stats endpoints, never persisted.
// ----------------------------------------------------------------------------

// ModuleNode is a recursive view of the module tree. The Tree endpoint
// returns []*ModuleNode with Children either nil (leaf) or populated.
type ModuleNode struct {
	Module
	Children []*ModuleNode `json:"children,omitempty"`
}

// SessionStats is the aggregation payload for /api/sessions/stats.
type SessionStats struct {
	Total         int            `json:"total"`
	TurnCount     int            `json:"turn_count"`
	ToolCount     int            `json:"tool_count"`
	ByProject     map[string]int `json:"by_project"`
	ByModule      map[string]int `json:"by_module"`
	ByUser        map[string]int `json:"by_user"`
	TopUsers      []StatEntry    `json:"top_users"`
	TopProjects   []StatEntry    `json:"top_projects"`
}

// StatEntry is a single "label → count" pair for top-N lists.
type StatEntry struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ModuleTreeMoveRequest is the body of POST /api/modules/:id/move.
// NewParentID nil means "promote to root".
type ModuleTreeMoveRequest struct {
	NewParentID *string `json:"new_parent_id"`
}
package console

// Pagination and DTO types shared by Tasks 6, 7, 9, 13, 14.
//
// The v2 DTOs in this file mirror the wire-level payloads the console UI
// sends and receives. They are intentionally decoupled from the DB
// models in models.go so the storage shape can evolve (new columns,
// snake_case vs camelCase, denormalised counters) without breaking
// clients. Request bodies live alongside their DTOs as `*Req` types and
// use pointer fields for partial updates.

import "time"

type PageReq struct {
	PageNo   int `json:"page_no"`
	PageSize int `json:"page_size"`
}

const (
	defaultPageNo   = 1
	defaultPageSize = 20
	maxPageSize     = 200
)

// Normalize clamps PageNo/PageSize to safe bounds.
func (p *PageReq) Normalize() {
	if p.PageNo <= 0 {
		p.PageNo = defaultPageNo
	}
	if p.PageSize <= 0 {
		p.PageSize = defaultPageSize
	}
	if p.PageSize > maxPageSize {
		p.PageSize = maxPageSize
	}
}

type PageResp struct {
	List     any `json:"list"`
	Total    int `json:"total"`
	PageNo   int `json:"page_no"`
	PageSize int `json:"page_size"`
}

// ----------------------------------------------------------------------------
// v1 DTOs (kept unchanged so the existing UI keeps working)
// ----------------------------------------------------------------------------

type UserDTO struct {
	ID           string   `json:"id"`
	DisplayName  string   `json:"display_name"`
	ProjectPaths []string `json:"project_paths"`
	MaxProcs     int      `json:"max_procs"`
	Disabled     bool     `json:"disabled"`
	CreatedAt    string   `json:"created_at"`
}

type ProjectDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Wing       string `json:"wing"`
	MCPBin     string `json:"mcp_bin"`
	CreatorID  string `json:"creator_id"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Deleted    bool   `json:"deleted"`
}

type SessionDTO struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	ProjectID   string  `json:"project_id"`
	ProjectPath string  `json:"project_path"`
	StartedAt   string  `json:"started_at"`
	EndedAt     *string `json:"ended_at"`
	ToolCount   int     `json:"tool_count"`
	TurnCount   int     `json:"turn_count"`
	Summary     string  `json:"summary"`
}

type TurnDTO struct {
	ID        int    `json:"id"`
	SessionID string `json:"session_id"`
	TurnNo    int    `json:"turn_no"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	ToolsJSON string `json:"tools_json"`
	TS        string `json:"ts"`
}

// ----------------------------------------------------------------------------
// v2 DTOs (added 2026-07-15 alongside models.go)
// ----------------------------------------------------------------------------

// AuthDTO is the response to POST /api/auth/login. The `User` field
// is the v2 user payload so the UI can show role / must_change_password
// / default_team_id without a second round-trip.
type AuthDTO struct {
	AccessToken        string    `json:"access_token"`
	RefreshToken       string    `json:"refresh_token"`
	ExpiresIn          int       `json:"expires_in"`
	MustChangePassword bool      `json:"must_change_password"`
	User               UserDTOv2 `json:"user"`
}

// UserDTOv2 is the v2 user payload — supersedes the v1 UserDTO but keeps
// the same JSON id so legacy admin handlers keep returning compatible
// shapes. Pointer fields are used for partial updates.
type UserDTOv2 struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	DisplayName        string     `json:"display_name"`
	Email              string     `json:"email,omitempty"`
	DefaultTeamID      string     `json:"default_team_id,omitempty"`
	Role               string     `json:"role"`
	MustChangePassword bool       `json:"must_change_password"`
	Disabled           bool       `json:"disabled"`
	CreatedAt          string     `json:"created_at"`
	UpdatedAt          string     `json:"updated_at"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
}

type CreateUserReq struct {
	ID            string `json:"id" binding:"required"`
	Username      string `json:"username" binding:"required"`
	DisplayName   string `json:"display_name"`
	Email         string `json:"email"`
	Password      string `json:"password" binding:"required,min=8"`
	Role          string `json:"role"`
	DefaultTeamID string `json:"default_team_id"`
}

type UpdateUserReq struct {
	DisplayName   *string `json:"display_name"`
	Email         *string `json:"email"`
	Role          *string `json:"role"`
	DefaultTeamID *string `json:"default_team_id"`
	Disabled      *bool   `json:"disabled"`
}

type ResetPasswordReq struct {
	NewPassword       string `json:"new_password" binding:"required,min=8"`
	MustChangeOnLogin bool   `json:"must_change_on_login"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ----------------------------------------------------------------------------
// Team DTOs
// ----------------------------------------------------------------------------

type TeamDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description,omitempty"`
	OwnerID      string `json:"owner_id"`
	MemberCount  int    `json:"member_count"`
	ProjectCount int    `json:"project_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Deleted      bool   `json:"deleted"`
}

type CreateTeamReq struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description"`
	// OwnerID is optional. When omitted, the caller (JWT user_id)
	// becomes the team owner. Useful when an admin creates a team on
	// behalf of someone else — then OwnerID should be that someone.
	OwnerID string `json:"owner_id"`
}

type UpdateTeamReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type TeamMemberDTO struct {
	TeamID   string `json:"team_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username,omitempty"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

type AddTeamMemberReq struct {
	Username string `json:"username" binding:"required,min=2,max=128"`
	Password string `json:"password" binding:"required,min=6,max=128"`
	Role     string `json:"role" binding:"required"`
}

type UpdateTeamMemberReq struct {
	Role string `json:"role" binding:"required"`
}

// ----------------------------------------------------------------------------
// Project DTOs
// ----------------------------------------------------------------------------

type ProjectDTOv2 struct {
	ID           string `json:"id"`
	TeamID       string `json:"team_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description,omitempty"`
	Path         string `json:"path"`
	PathRel      string `json:"path_rel,omitempty"`
	GitURL       string `json:"git_url,omitempty"`
	GitBranch    string `json:"git_branch,omitempty"`
	GitCommitSHA string `json:"git_commit_sha,omitempty"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Deleted      bool   `json:"deleted"`
}

type CreateProjectReq struct {
	Name         string `json:"name" binding:"required"`
	Slug         string `json:"slug" binding:"required"`
	Description  string `json:"description"`
	GitURL       string `json:"git_url"`
	GitBranch    string `json:"git_branch"`
	DefaultBranch string `json:"default_branch"`
}

type UpdateProjectReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	GitBranch   *string `json:"git_branch"`
}

type ProjectIndexStatusDTO struct {
	ProjectID   string  `json:"project_id"`
	Indexed     bool    `json:"indexed"`
	FileCount   int     `json:"file_count"`
	LastUpdate  *string `json:"last_update,omitempty"`
	Status      string  `json:"status"`
	ErrorMsg    string  `json:"error,omitempty"`
}

type ProjectGitStatusDTO struct {
	ProjectID  string `json:"project_id"`
	Branch     string `json:"branch"`
	CommitSHA  string `json:"commit_sha"`
	IsClean    bool   `json:"is_clean"`
	Untracked  int    `json:"untracked"`
	Modified   int    `json:"modified"`
	Staged     int    `json:"staged"`
}

// ----------------------------------------------------------------------------
// Module DTOs
// ----------------------------------------------------------------------------

type ModuleDTO struct {
	ID           string  `json:"id"`
	ProjectID    string  `json:"project_id"`
	ParentID     *string `json:"parent_id,omitempty"`
	Name         string  `json:"name"`
	Path         string  `json:"path,omitempty"`
	Description  string  `json:"description,omitempty"`
	Order        int     `json:"order"`
	IsLeaf       bool    `json:"is_leaf"`
	ChildCount   int     `json:"child_count"`
	SessionCount int     `json:"session_count"`
	MemoryCount  int     `json:"memory_count"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type ModuleNodeDTO struct {
	ModuleDTO
	Children []*ModuleNodeDTO `json:"children,omitempty"`
}

type CreateModuleReq struct {
	ParentID    *string `json:"parent_id"`
	Name        string  `json:"name" binding:"required"`
	Path        string  `json:"path"`
	Description string  `json:"description"`
	Order       int     `json:"order"`
}

type UpdateModuleReq struct {
	Name        *string `json:"name"`
	Path        *string `json:"path"`
	Description *string `json:"description"`
	Order       *int    `json:"order"`
}

type MoveModuleReq struct {
	NewParentID *string `json:"new_parent_id"`
}

// ----------------------------------------------------------------------------
// Session DTOs (v2 — module_id is mandatory)
// ----------------------------------------------------------------------------

type SessionDTOv2 struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	TeamID          string  `json:"team_id"`
	ProjectID       string  `json:"project_id"`
	ModuleID        string  `json:"module_id"`
	ProjectPath     string  `json:"project_path"`
	StartedAt       string  `json:"started_at"`
	EndedAt         *string `json:"ended_at,omitempty"`
	ToolCount       int     `json:"tool_count"`
	TurnCount       int     `json:"turn_count"`
	Summary         string  `json:"summary,omitempty"`

	// Auto-sync bookkeeping surfaced to the UI for ops.
	MemPalaceSyncedTurns  int     `json:"mempalace_synced_turns"`
	MemPalaceLastSyncedAt *string `json:"mempalace_last_synced_at,omitempty"`
	MemPalaceLastError    string  `json:"mempalace_last_error,omitempty"`
}

type SessionDetailDTO struct {
	Session SessionDTOv2 `json:"session"`
	Turns   []TurnDTO   `json:"turns"`
}

// ----------------------------------------------------------------------------
// Memory & template DTOs
// ----------------------------------------------------------------------------

type MemoryTemplateDTO struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Fields       []TemplateField `json:"fields"`
	IsBuiltin    bool           `json:"is_builtin"`
}

// TemplateField describes one input in a memory template's form.
type TemplateField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"` // string | list | date
	Required bool   `json:"required"`
	Help     string `json:"help,omitempty"`
}

type MemoryDTO struct {
	ID         string   `json:"id"`
	TeamID     string   `json:"team_id"`
	ProjectID  string   `json:"project_id"`
	ModuleID   string   `json:"module_id"`
	UserID     string   `json:"user_id"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	TemplateID string   `json:"template_id,omitempty"`
	Tags       []string `json:"tags"`
	Hall       string   `json:"hall"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

type CreateMemoryReq struct {
	Title      string   `json:"title" binding:"required"`
	Content    string   `json:"content" binding:"required"`
	TemplateID string   `json:"template_id"`
	Tags       []string `json:"tags"`
	Hall       string   `json:"hall"`
	// ModuleID identifies the leaf module this memory attaches to.
	// Module must exist, not be soft-deleted, and is_leaf=1. The
	// server derives team_id and project_id from this so the UI
	// only has to know the module.
	ModuleID string `json:"module_id" binding:"required"`
}

// UpdateMemoryReq is the body for PUT /api/v2/memories/:id. All
// fields are optional; nil = "leave unchanged". Tags nil = "leave
// unchanged"; empty slice = "clear all tags".
type UpdateMemoryReq struct {
	Title   *string   `json:"title"`
	Content *string   `json:"content"`
	Hall    *string   `json:"hall"`
	Tags    []string  `json:"tags"`
}

// AddTagReq appends a single tag (and the legacy counterpart
// TagMemoryReq right below accepts a full slice — both forms are
// kept because older front-ends emit one shape and new ones the
// other).
type AddTagReq struct {
	Tag string `json:"tag" binding:"required"`
}

// SummarizeTaskReq is the body for POST /api/v2/summarize-tasks.
type SummarizeTaskReq struct {
	SourceIDs  []string `json:"source_ids" binding:"required"`
	Depth      string   `json:"depth"`
	TargetWing string   `json:"target_wing"`
}

// DistillTaskReq is the body for POST /api/v2/distill-tasks.
type DistillTaskReq struct {
	SourceIDs  []string       `json:"source_ids" binding:"required"`
	Rules      map[string]any `json:"rules"`
	TargetWing string         `json:"target_wing"`
}

// SummarizeTaskDTO / DistillTaskDTO already live in the AI Tool
// DTO block below (returned by the v1 admin-cookie handlers).
// The summary/distill v2 handlers reuse those shapes and adapt the
// SourceIDs/Result mapping. No new DTOs needed here.

// ----------------------------------------------------------------------------
// Render template
// ----------------------------------------------------------------------------

type RenderTemplateReq struct {
	Fields map[string]any `json:"fields" binding:"required"`
}

type RenderTemplateResp struct {
	Rendered string `json:"rendered"`
}

// CreateTemplateReq is the body for POST /api/v2/memory-templates.
// All fields except name are optional; an admin can register a
// custom template with just a name + body.
type CreateTemplateReq struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	FieldsJSON   string `json:"fields_json"`
	BodyTemplate string `json:"body_template"`
}

// ----------------------------------------------------------------------------
// AI Tool DTOs
// ----------------------------------------------------------------------------

type AIToolDTO struct {
	ID                  string   `json:"id"`
	TeamID              string   `json:"team_id"`
	Name                string   `json:"name"`
	Slug                string   `json:"slug"`
	Description         string   `json:"description,omitempty"`
	Endpoint            string   `json:"endpoint"`
	Protocol            string   `json:"protocol"`
	Command             string   `json:"command,omitempty"`
	Args                []string `json:"args,omitempty"`
	RequiredRole        string   `json:"required_role"`
	AllowedProjects     []string `json:"allowed_projects,omitempty"`
	TimeoutSeconds      int      `json:"timeout_seconds"`
	Enabled             bool     `json:"enabled"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}

type CreateAIToolReq struct {
	TeamID          string   `json:"team_id" binding:"required"`
	Name            string   `json:"name" binding:"required"`
	Slug            string   `json:"slug" binding:"required"`
	Description     string   `json:"description"`
	Endpoint        string   `json:"endpoint"`
	Protocol        string   `json:"protocol" binding:"required"`
	Command         string   `json:"command"`
	Args            []string `json:"args"`
	RequiredRole    string   `json:"required_role" binding:"required"`
	AllowedProjects []string `json:"allowed_projects"`
	TimeoutSeconds  int      `json:"timeout_seconds"`
}

type UpdateAIToolReq struct {
	Name            *string   `json:"name"`
	Description     *string   `json:"description"`
	Endpoint        *string   `json:"endpoint"`
	Command         *string   `json:"command"`
	Args            *[]string `json:"args"`
	RequiredRole    *string   `json:"required_role"`
	AllowedProjects *[]string `json:"allowed_projects"`
	TimeoutSeconds  *int      `json:"timeout_seconds"`
	Enabled         *bool     `json:"enabled"`
}

type InvokeAIToolReq struct {
	ProjectID  string         `json:"project_id" binding:"required"`
	ModuleID   string         `json:"module_id" binding:"required"`
	Input      map[string]any `json:"input" binding:"required"`
	WorkingDir string         `json:"working_dir"`
}

type AIToolInvocationDTO struct {
	ID         string  `json:"id"`
	ToolID     string  `json:"tool_id"`
	UserID     string  `json:"user_id"`
	TeamID     string  `json:"team_id"`
	ProjectID  string  `json:"project_id"`
	ModuleID   string  `json:"module_id"`
	WorkingDir string  `json:"working_dir"`
	Status     string  `json:"status"`
	StartedAt  string  `json:"started_at"`
	FinishedAt *string `json:"finished_at,omitempty"`
	Error      string  `json:"error,omitempty"`
}

// ----------------------------------------------------------------------------
// Summarize / Distill task DTOs
// ----------------------------------------------------------------------------

type SummarizeTaskDTO struct {
	ID         string  `json:"id"`
	UserID     string  `json:"user_id"`
	SourceIDs  []string `json:"source_ids"`
	Depth      string   `json:"depth"`
	TargetWing string   `json:"target_wing"`
	Status     string   `json:"status"`
	Result     any      `json:"result,omitempty"`
	CreatedAt  string   `json:"created_at"`
	FinishedAt *string  `json:"finished_at,omitempty"`
}

type DistillTaskDTO struct {
	ID              string   `json:"id"`
	UserID          string   `json:"user_id"`
	SourceIDs       []string `json:"source_ids"`
	Rules           any      `json:"rules"`
	Status          string   `json:"status"`
	Result          any      `json:"result,omitempty"`
	MemPalaceSynced bool     `json:"mempalace_synced"`
	TargetWing      string   `json:"target_wing"`
	CreatedAt       string   `json:"created_at"`
	FinishedAt      *string  `json:"finished_at,omitempty"`
}

type CreateSummarizeReq struct {
	SourceIDs  []string `json:"source_ids" binding:"required"`
	Depth      string   `json:"depth"`
	TargetWing string   `json:"target_wing" binding:"required"`
}

type CreateDistillReq struct {
	SourceIDs  []string        `json:"source_ids" binding:"required"`
	Rules      map[string]any  `json:"rules"`
	TargetWing string          `json:"target_wing" binding:"required"`
}
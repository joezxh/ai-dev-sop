package console

// Pagination and DTO types shared by Tasks 6, 7, 9, 13, 14.

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
	UserID     string `json:"user_id"`
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

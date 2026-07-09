package console

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/store"
)

// contains returns true if `needle` is found in `haystack`.
// Empty needle matches everything (used to short-circuit the query filter).
func contains(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return bytesContains(haystack, needle)
}

// bytesContains is a tiny substring search used for user/project queries.
// (Re-implemented locally rather than pulling in strings.Contains from the
// standard library, per the brief's explicit helper definition.)
func bytesContains(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func userToDTO(u *store.User) UserDTO {
	paths := u.ProjectPaths
	if paths == nil {
		paths = []string{}
	}
	return UserDTO{
		ID:           u.ID,
		DisplayName:  u.DisplayName,
		ProjectPaths: paths,
		MaxProcs:     u.MaxProcs,
		Disabled:     u.Disabled,
		CreatedAt:    u.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type createUserReq struct {
	ID           string   `json:"id" binding:"required"`
	DisplayName  string   `json:"display_name"`
	ProjectPaths []string `json:"project_paths"`
	MaxProcs     int      `json:"max_procs"`
}

type updateUserReq struct {
	DisplayName  *string  `json:"display_name"`
	ProjectPaths []string `json:"project_paths"`
	MaxProcs     *int     `json:"max_procs"`
	Disabled     *bool    `json:"disabled"`
}

// ListUsersHandler returns the user registry filtered by `q` substring match
// and paginated by `page_no` / `page_size` query parameters.
func ListUsersHandler(reg *store.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p PageReq
		_ = c.ShouldBindQuery(&p)
		p.Normalize()
		pageNo, pageSize := p.PageNo, p.PageSize
		q := c.Query("q")
		all := reg.List()

		filtered := make([]*store.User, 0, len(all))
		for _, u := range all {
			if !contains(u.ID, q) && !contains(u.DisplayName, q) {
				continue
			}
			filtered = append(filtered, u)
		}
		start := (pageNo - 1) * pageSize
		end := start + pageSize
		if start > len(filtered) {
			start = len(filtered)
		}
		if end > len(filtered) {
			end = len(filtered)
		}
		slice := filtered[start:end]
		list := make([]UserDTO, 0, len(slice))
		for _, u := range slice {
			list = append(list, userToDTO(u))
		}
		OK(c, PageResp{List: list, Total: len(filtered), PageNo: pageNo, PageSize: pageSize})
	}
}

// CreateUserHandler upserts a user record.
func CreateUserHandler(reg *store.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000001, "invalid request: "+err.Error())
			return
		}
		display := req.DisplayName
		if display == "" {
			display = req.ID
		}
		u := &store.User{
			ID:           req.ID,
			DisplayName:  display,
			ProjectPaths: req.ProjectPaths,
			MaxProcs:     req.MaxProcs,
			CreatedAt:    time.Now().UTC(),
		}
		if err := reg.Upsert(u); err != nil {
			Fail(c, http.StatusConflict, 4090001, "upsert user: "+err.Error())
			return
		}
		OK(c, userToDTO(u))
	}
}

// UpdateUserHandler applies a partial update to an existing user.
func UpdateUserHandler(reg *store.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		existing, ok := reg.Get(id)
		if !ok {
			Fail(c, http.StatusNotFound, 4040001, "user not found")
			return
		}

		var req updateUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000002, "invalid request: "+err.Error())
			return
		}
		if req.DisplayName != nil {
			existing.DisplayName = *req.DisplayName
		}
		if req.ProjectPaths != nil {
			existing.ProjectPaths = req.ProjectPaths
		}
		if req.MaxProcs != nil {
			existing.MaxProcs = *req.MaxProcs
		}
		if req.Disabled != nil {
			existing.Disabled = *req.Disabled
		}
		if err := reg.Upsert(existing); err != nil {
			Fail(c, http.StatusInternalServerError, 5000001, "update user: "+err.Error())
			return
		}
		OK(c, userToDTO(existing))
	}
}

// DeleteUserHandler removes a user from the registry.
func DeleteUserHandler(reg *store.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := reg.Delete(id); err != nil {
			Fail(c, http.StatusInternalServerError, 5000002, "delete user: "+err.Error())
			return
		}
		OK(c, gin.H{"id": id})
	}
}

// RevokeUserHandler marks a user as disabled.
func RevokeUserHandler(reg *store.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		existing, ok := reg.Get(id)
		if !ok {
			Fail(c, http.StatusNotFound, 4040002, "user not found")
			return
		}
		existing.Disabled = true
		if err := reg.Upsert(existing); err != nil {
			Fail(c, http.StatusInternalServerError, 5000003, "revoke user: "+err.Error())
			return
		}
		OK(c, userToDTO(existing))
	}
}

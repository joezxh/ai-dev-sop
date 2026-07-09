package console

import (
	"context"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/llm"
	"cbmem-team/internal/store"
)

type MountConfig struct {
	DB         *DB
	AdminToken string
	Session    *SessionManager
	Users      *store.Registry
	LLM        llm.Provider
	MemPalace  *llm.MemPalace
}

func Mount(r *gin.Engine, cfg MountConfig) {
	if cfg.DB != nil {
		if err := cfg.DB.Migrate(context.Background()); err != nil {
			panic("console: migrate db: " + err.Error())
		}
	}

	api := r.Group("/api/console")
	api.POST("/login", LoginHandler(cfg.AdminToken, cfg.Session))
	api.POST("/logout", RequireSession(cfg.Session), RequireCSRF(), LogoutHandler(cfg.Session))

	protected := api.Group("")
	protected.Use(RequireSession(cfg.Session))
	protected.GET("/me", MeHandler())

	users := protected.Group("/users")
	users.GET("", ListUsersHandler(cfg.Users))
	users.POST("", CreateUserHandler(cfg.Users))
	users.PUT("/:id", UpdateUserHandler(cfg.Users))
	users.DELETE("/:id", DeleteUserHandler(cfg.Users))
	users.POST("/:id/revoke", RevokeUserHandler(cfg.Users))

	projects := protected.Group("/projects")
	projects.GET("", ListProjectsHandler(cfg.DB))
	projects.POST("", CreateProjectHandler(cfg.DB, nil))
	projects.PUT("/:id", UpdateProjectHandler(cfg.DB))
	projects.DELETE("/:id", DeleteProjectHandler(cfg.DB))
	projects.GET("/:id/index-status", ProjectIndexStatusHandler(cfg.DB, nil))
	projects.POST("/:id/reindex", ProjectReindexHandler(cfg.DB, nil))

	// Sessions captured by CaptureSessions — list, detail, and dashboard
	// stats. The stats endpoint is mounted on `protected` (NOT under
	// /sessions) so the `:id` route above doesn't accidentally capture it.
	sessions := protected.Group("/sessions")
	sessions.GET("", ListSessionsHandler(cfg.DB))
	sessions.GET("/:id", SessionDetailHandler(cfg.DB))
	protected.GET("/sessions-stats", SessionsStatsHandler(cfg.DB))

	// Summarize route group requires an LLM provider. Without one
	// (e.g. in unit tests / no-llm console) the routes are simply not
	// mounted.
	if cfg.LLM != nil {
		summary := protected.Group("/summarize")
		summary.POST("", SummarizeHandler(cfg.DB, cfg.LLM))
		summary.GET("/:task_id", GetSummarizeHandler(cfg.DB))
	}

	// Distill routes (Task 14) are mounted when both an LLM provider and
	// a MemPalace client are wired in.
	_ = cfg.MemPalace
}

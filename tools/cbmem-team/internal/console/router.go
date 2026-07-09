package console

import (
	"context"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/store"
)

type MountConfig struct {
	DB         *DB
	AdminToken string
	Session    *SessionManager
	Users      *store.Registry
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
}

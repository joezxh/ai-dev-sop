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
}

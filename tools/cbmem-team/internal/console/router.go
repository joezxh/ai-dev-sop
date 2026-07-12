package console

import (
	"context"
	"fmt"
	"net/http"

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
		// M2 schema registration is independent of M1: even if M1's
		// tool_directory / tool_invocation_logs already exist, M2 just
		// adds the two BP tables alongside.
		if err := cfg.DB.Migrate(context.Background(), M1ExtraSchema(), M2ExtraSchema(), M3ExtraSchema(), M4ExtraSchema()); err != nil {
			panic("console: migrate db: " + err.Error())
		}
		if n, err := SeedToolDirectory(context.Background(), cfg.DB); err != nil {
			// best-effort: an empty catalog is still served by the API,
			// it just returns []. Log and continue.
			fmt.Printf("seed tool_directory: %v\n", err)
		} else if n > 0 {
			fmt.Printf("seeded %d tool_directory rows\n", n)
		}
		if n, err := SeedBPs(context.Background(), cfg.DB); err != nil {
			fmt.Printf("seed best_practices: %v\n", err)
		} else if n > 0 {
			fmt.Printf("seeded %d best_practices rows\n", n)
		}
		if n, err := SeedWorkflows(context.Background(), cfg.DB); err != nil {
			fmt.Printf("seed workflows: %v\n", err)
		} else if n > 0 {
			fmt.Printf("seeded %d workflow rows\n", n)
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

	// M1: tool directory + invocation dashboard. Mounted under
	// /api/console/v2/* so existing /api/console/* routes are not disturbed.
	v2 := protected.Group("/v2")
	{
		v2.GET("/tools", ListToolsHandler(cfg.DB))
		v2.GET("/invocations/recent", RecentInvocationsHandler(cfg.DB))
		v2.GET("/dashboard/summary", DashboardSummaryHandler(cfg.DB))

		// M2: per-tool rate-limit admin (T2.2). Six routes:
		//   GET    /v2/rate-limits                     list all configured tools
		//   GET    /v2/tools/:id/rate-limit            fetch single (or defaults)
		//   PUT    /v2/tools/:id/rate-limit            overwrite
		//   DELETE /v2/tools/:id/rate-limit            reset to defaults
		//   POST   /v2/tools/:id/rate-limit/reset      force-clear in-memory state
		v2.GET("/rate-limits", ListRateLimitsHandler(cfg.DB))
		v2.GET("/tools/:id/rate-limit", GetRateLimitHandler(cfg.DB))
		v2.PUT("/tools/:id/rate-limit", PutRateLimitHandler(cfg.DB))
		v2.DELETE("/tools/:id/rate-limit", DeleteRateLimitHandler(cfg.DB))
		v2.POST("/tools/:id/rate-limit/reset", ResetRateLimitHandler())

		// M2: best-practice CRUD + version history + association graph.
		// Mounted under the same v2 namespace; the tool detail UI in M2
		// will hit these via /api/console/v2/bps/*
		bps := v2.Group("/bps")
		{
			bps.GET("", ListBPsHandler(cfg.DB))
			bps.POST("", CreateBPHandler(cfg.DB))
			bps.GET("/:id", GetBPHandler(cfg.DB))
			bps.PUT("/:id", UpdateBPHandler(cfg.DB))
			bps.POST("/:id/publish", PublishBPHandler(cfg.DB))
			bps.GET("/:id/versions", ListBPVersionsHandler(cfg.DB))
			bps.GET("/:id/versions/:n", GetBPVersionHandler(cfg.DB))
			bps.GET("/:id/graph", BPGraphHandler(cfg.DB))
		}

		// M3: workflow CRUD + run/simulate + impact dashboard + tickets.
		// All routes live under the existing v2 namespace; the editor UI
		// (planned for a separate delivery) hits these endpoints.
		wfs := v2.Group("/workflows")
		{
			wfs.GET("", ListWorkflowsHandler(cfg.DB))
			wfs.POST("", CreateWorkflowHandler(cfg.DB))
			wfs.GET("/:id", GetWorkflowHandler(cfg.DB))
			wfs.PUT("/:id", UpdateWorkflowHandler(cfg.DB))
			wfs.POST("/:id/publish", PublishWorkflowHandler(cfg.DB))
			wfs.POST("/:id/archive", ArchiveWorkflowHandler(cfg.DB))
			wfs.POST("/:id/run", RunWorkflowHandler(cfg.DB))
			wfs.GET("/:id/runs", ListWorkflowRunsHandler(cfg.DB))
		}

		v2.GET("/dashboard/high-risk", HighRiskDashboardHandler(cfg.DB))
		v2.GET("/dashboard/high-risk/summary", HighRiskSummaryHandler(cfg.DB))

		tickets := v2.Group("/tickets")
		{
			tickets.GET("", ListTicketsHandler(cfg.DB))
			tickets.POST("", CreateTicketHandler(cfg.DB))
			tickets.POST("/auto-open", AutoOpenTicketHandler(cfg.DB))
			tickets.GET("/:id", GetTicketHandler(cfg.DB))
			tickets.POST("/:id/resolve", ResolveTicketHandler(cfg.DB))
			tickets.POST("/:id/wont-fix", WontFixTicketHandler(cfg.DB))
		}

		// M4: repo pipeline CRUD + run/simulate + bp_candidates review.
		rpRuntime := DefaultRuntime()
		repos := v2.Group("/repos")
		{
			repos.GET("", ListRepoPipelinesHandler(cfg.DB))
			repos.POST("", CreateRepoPipelineHandler(cfg.DB))
			repos.GET("/runs/:run_id", GetRepoPipelineRunHandler(cfg.DB))
			repos.GET("/candidates", ListBPCandidatesHandler(cfg.DB))
			repos.GET("/candidates/:id", GetBPCandidateHandler(cfg.DB))
			repos.POST("/candidates/:id/accept", AcceptBPCandidateHandler(cfg.DB))
			repos.POST("/candidates/:id/reject", RejectBPCandidateHandler(cfg.DB))
			repos.POST("/candidates/:id/merge", MergeBPCandidateHandler(cfg.DB))
			repos.GET("/:id", GetRepoPipelineHandler(cfg.DB))
			repos.PUT("/:id", UpdateRepoPipelineHandler(cfg.DB))
			repos.POST("/:id/activate", ActivateRepoPipelineHandler(cfg.DB))
			repos.POST("/:id/archive", ArchiveRepoPipelineHandler(cfg.DB))
			repos.POST("/:id/run", RunRepoPipelineHandler(cfg.DB, rpRuntime))
			repos.GET("/:id/runs", ListRepoPipelineRunsHandler(cfg.DB))
		}
	}

	// M2 single-file UI. Public so a curl smoke can hit it without a
	// login; the in-page API calls still hit /api/console/* which is
	// session-protected.
	r.GET("/ui/m2", func(c *gin.Context) { c.Redirect(http.StatusFound, "/ui/m2/") })
	r.GET("/ui/m2/*filepath", StaticUIHandler())

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

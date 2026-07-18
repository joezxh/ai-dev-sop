package console

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/auth"
	"cbmem-team/internal/llm"
	"cbmem-team/internal/store"
)

type MountConfig struct {
	DB         *DB
	AdminToken string
	JWTSecret  []byte
	Session    *SessionManager
	Users      *store.Registry
	LLM        llm.Provider
	MemPalace  *llm.MemPalace
	DataDir    string // per-user data root, used to auto-generate project paths
	MCPBinary  string // default codebase-memory-mcp binary path

	// CORS, when non-nil, is mounted on the /api/console group only.
	// /healthz and other root-level routes are intentionally left untouched
	// so internal health checks remain CORS-free.
	CORS gin.HandlerFunc

	// Auth (M2): when non-nil, /api/auth/* routes are mounted under
	// the public (unauthenticated) /api group. JWTSecret must match
	// MountConfig.JWTSecret; InitialAdmin, when non-empty, enables
	// the POST /api/auth/first-admin route for first-boot setup.
	Auth *AuthHandlers

	// JWTVerifier (M3): when non-nil, used by the /api/console/v2
	// team/project routes as the JWT auth middleware. Same secret as
	// Auth.JWTSecret. Kept separate so deployments can mount M3 even
	// when Auth's extra-checks (user-disabled, etc.) aren't desired.
	JWTVerifier *auth.Verifier

	// Teams (M3): when non-nil, mounts /api/v2/teams/* endpoints.
	Teams *TeamHandlers

// Projects (M3): when non-nil, mounts /api/v2/teams/:id/projects/*
// and /api/v2/projects/:pid/* endpoints. RepoRoot is the canonical
// base directory under which every v2 project path lives; passed
// to NewProjectHandlers.
	Projects *ProjectHandlers

	// Modules (M4): when non-nil, mounts /api/v2/projects/:id/modules/*
	// plus /api/v2/modules/:id* endpoints.
	Modules *ModuleHandlers

	// SessionsV2 (M4): when non-nil, mounts /api/v2/sessions*
	// and /api/v2/projects/:id/sessions* endpoints. These are
	// additive to the v1 cookie-protected sessions endpoints under
	// /api/console/sessions; the v2 path is JWT-gated and exposes
	// richer filters (team_id, project_id, module_id).
	SessionsV2 *SessionHandlersV2

	// MemoryTemplates (M5): list + render endpoints under
	// /api/v2/memory-templates/*.
	MemoryTemplates *MemoryTemplateHandlers

	// Memories (M5): CRUD + tags under /api/v2/memories and
	// /api/v2/modules/:id/memories.
	Memories *MemoryHandlers

	// SummarizeDistillV2 (M5): JWT-gated wrappers for summarize
	// and distill task lifecycle. Mounts under
	// /api/v2/summarize-tasks and /api/v2/distill-tasks.
	SummarizeDistillV2 *SummarizeDistillHandlersV2

	// AITools (M6): CRUD + invoke for the AI tool catalogue
	// (cursor / qoder / superpowers / gstack / custom). Mounts
	// under /api/v2/ai-tools/*.
	AITools *AIToolHandlers
}

func Mount(r *gin.Engine, cfg MountConfig) {
	if cfg.DB != nil {
		// M2 schema registration is independent of M1: even if M1's
		// tool_directory / tool_invocation_logs already exist, M2 just
		// adds the two BP tables alongside.
		if err := cfg.DB.Migrate(context.Background(), M1ExtraSchema(), M2ExtraSchema(), M3ExtraSchema(), M4ExtraSchema()); err != nil {
			panic("console: migrate db: " + err.Error())
		}
		// Auto-sync columns on sessions (mempalace_synced_turns,
		// mempalace_last_synced_at, mempalace_last_error) are part of
		// CREATE TABLE for fresh DBs; the ALTER here makes them appear on
		// pre-existing v1 deployments. Idempotent.
		if err := cfg.DB.MigrateSessionsAutoSyncColumns(context.Background()); err != nil {
			// Don't fail boot — capture and summarisation still work
			// without the watermark. Log and continue.
			fmt.Printf("migrate auto-sync columns: %v\n", err)
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

		// v2 schema migration (M2-M4). Runs after the v1 schema so the
		// ALTER TABLE statements have a base to work on. Idempotent.
		if err := cfg.DB.MigrateV2(context.Background(), nil); err != nil {
			panic("console: migrate v2: " + err.Error())
		}

		// Seed the default admin user (admin / admin123) when no admin
		// exists yet. Runs after MigrateV2 so the password_hash column
		// is guaranteed to exist.
		if n, err := SeedDefaultAdmin(context.Background(), cfg.DB); err != nil {
			fmt.Printf("seed default admin: %v\n", err)
		} else if n > 0 {
			fmt.Printf("seeded default admin user (admin / admin123)\n")
		}
	}

	api := r.Group("/api/console")
	if cfg.CORS != nil {
		api.Use(cfg.CORS)
	}

	// /api/auth/* — M2 password + JWT routes. Lives at /api/auth (not
	// /api/console/auth) so the cookie-based /api/console/login can
	// continue to exist for the v1 UI while the v2 UI moves here.
	authGroup := r.Group("/api/auth")
	if cfg.CORS != nil {
		authGroup.Use(cfg.CORS)
	}
	if cfg.Auth != nil {
		cfg.Auth.Mount(authGroup)
	}

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
	users.POST("/:id/token", MintTokenHandler(cfg.Users, cfg.JWTSecret))

	projects := protected.Group("/projects")
	projects.GET("", ListProjectsHandler(cfg.DB))
	projects.POST("", CreateProjectHandler(cfg.DB, cfg.DataDir, cfg.MCPBinary))
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

	// /api/console/v2 — shared prefix for session-auth (M1/M2) and
	// JWT-auth (M3+) routes. The group itself carries NO auth middleware;
	// each sub-group applies its own.
	v2 := api.Group("/v2")
	{
		// Session-authenticated routes (M1/M2). v2sess inherits
		// RequireSession from the protected-style sub-group.
		v2sess := v2.Group("")
		v2sess.Use(RequireSession(cfg.Session))
		{
			// M1: tool directory + invocation dashboard.
			v2sess.GET("/tools", ListToolsHandler(cfg.DB))
			v2sess.GET("/invocations/recent", RecentInvocationsHandler(cfg.DB))
			v2sess.GET("/dashboard/summary", DashboardSummaryHandler(cfg.DB))

			// M2: per-tool rate-limit admin (T2.2).
			v2sess.GET("/rate-limits", ListRateLimitsHandler(cfg.DB))
			v2sess.GET("/tools/:id/rate-limit", GetRateLimitHandler(cfg.DB))
			v2sess.PUT("/tools/:id/rate-limit", PutRateLimitHandler(cfg.DB))
			v2sess.DELETE("/tools/:id/rate-limit", DeleteRateLimitHandler(cfg.DB))
			v2sess.POST("/tools/:id/rate-limit/reset", ResetRateLimitHandler())

			// M2: best-practice CRUD + version history + association graph.
			bps := v2sess.Group("/bps")
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
			wfs := v2sess.Group("/workflows")
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

			v2sess.GET("/dashboard/high-risk", HighRiskDashboardHandler(cfg.DB))
			v2sess.GET("/dashboard/high-risk/summary", HighRiskSummaryHandler(cfg.DB))

			tickets := v2sess.Group("/tickets")
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
			repos := v2sess.Group("/repos")
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

		// JWT-authenticated routes (M3+). Only JWT middleware, no
		// session cookie required. Middleware order:
		//   1. CORS (already on /api/console)
		//   2. JWT (sets user_id / role / jti in ctx)
		//   3. handler-level role checks via RequireRoleAtLeast
		//
		// Routes:
		//   GET    /api/console/v2/teams
		//   POST   /api/console/v2/teams
		//   GET    /api/console/v2/teams/:id
		//   PUT    /api/console/v2/teams/:id
		//   DELETE /api/console/v2/teams/:id
		//   GET    /api/console/v2/teams/:id/members
		//   POST   /api/console/v2/teams/:id/members
		//   PUT    /api/console/v2/teams/:id/members/:uid
		//   DELETE /api/console/v2/teams/:id/members/:uid
		//   GET    /api/console/v2/teams/:id/projects
		//   POST   /api/console/v2/teams/:id/projects
		//   GET    /api/console/v2/projects/:pid
		//   PUT    /api/console/v2/projects/:pid
		//   DELETE /api/console/v2/projects/:pid
		//   POST   /api/console/v2/projects/:pid/clone
		//   GET    /api/console/v2/projects/:pid/index-status
		//   POST   /api/console/v2/projects/:pid/reindex
		// M4 (modules, sessions), M5 (memory templates, memories,
		// summarize/distill tasks) are mounted in the same group so
		// they share the JWT verifier and "user_id / role" context.
		if cfg.Teams != nil || cfg.Projects != nil || cfg.Modules != nil || cfg.SessionsV2 != nil ||
			cfg.MemoryTemplates != nil || cfg.Memories != nil || cfg.SummarizeDistillV2 != nil ||
			cfg.AITools != nil {
			v2jwt := v2.Group("")
			if cfg.JWTVerifier != nil {
				v2jwt.Use(cfg.JWTVerifier.Middleware())
			}
			if cfg.Teams != nil {
				v2jwt.GET("/teams", cfg.Teams.ListTeams())
				v2jwt.POST("/teams", cfg.Teams.CreateTeam())
				v2jwt.GET("/teams/:id", cfg.Teams.GetTeam())
				v2jwt.PUT("/teams/:id", cfg.Teams.UpdateTeam())
				v2jwt.DELETE("/teams/:id", cfg.Teams.DeleteTeam())

				v2jwt.GET("/teams/:id/members", cfg.Teams.ListMembers())
				v2jwt.POST("/teams/:id/members", cfg.Teams.AddMember())
				v2jwt.PUT("/teams/:id/members/:uid", cfg.Teams.UpdateMember())
				v2jwt.DELETE("/teams/:id/members/:uid", cfg.Teams.RemoveMember())

				if cfg.Projects != nil {
					v2jwt.GET("/teams/:id/projects", cfg.Projects.ListProjects())
					v2jwt.POST("/teams/:id/projects", cfg.Projects.CreateProject())
				}
			}
			if cfg.Projects != nil {
				v2jwt.GET("/projects/:pid", cfg.Projects.GetProject())
				v2jwt.PUT("/projects/:pid", cfg.Projects.UpdateProject())
				v2jwt.DELETE("/projects/:pid", cfg.Projects.DeleteProject())
				v2jwt.POST("/projects/:pid/clone", cfg.Projects.CloneProject())
				v2jwt.GET("/projects/:pid/index-status", cfg.Projects.IndexStatus())
				v2jwt.POST("/projects/:pid/reindex", cfg.Projects.Reindex())

				// M4: module tree under a project + per-module endpoints.
				if cfg.Modules != nil {
					v2jwt.GET("/projects/:pid/modules", cfg.Modules.ListModules())
					v2jwt.POST("/projects/:pid/modules", cfg.Modules.CreateModule())
					v2jwt.GET("/projects/:pid/modules/tree", cfg.Modules.Tree())
				}
			}
			if cfg.Modules != nil {
				v2jwt.GET("/modules/:id", cfg.Modules.GetModule())
				v2jwt.PUT("/modules/:id", cfg.Modules.UpdateModule())
				v2jwt.DELETE("/modules/:id", cfg.Modules.DeleteModule())
				v2jwt.POST("/modules/:id/move", cfg.Modules.MoveModule())
			}
			if cfg.SessionsV2 != nil {
				v2jwt.GET("/sessions", cfg.SessionsV2.ListSessions())
				v2jwt.GET("/sessions/stats", cfg.SessionsV2.Stats())
				v2jwt.GET("/sessions/:id", cfg.SessionsV2.GetSession())
				if cfg.Projects != nil {
					v2jwt.GET("/projects/:pid/sessions", cfg.SessionsV2.ProjectSessions())
				}
			}
			// M5 — memory templates.
			if cfg.MemoryTemplates != nil {
				v2jwt.GET("/memory-templates", cfg.MemoryTemplates.List())
				v2jwt.GET("/memory-templates/:id", cfg.MemoryTemplates.Get())
				v2jwt.POST("/memory-templates/:id/render", cfg.MemoryTemplates.Render())
				v2jwt.POST("/memory-templates", cfg.MemoryTemplates.Create())
			}
			// M5 — memories (CRUD + tags). RBAC handled in handler.
			if cfg.Memories != nil {
				v2jwt.GET("/memories", cfg.Memories.List())
				v2jwt.POST("/memories", cfg.Memories.Create())
				v2jwt.GET("/memories/:id", cfg.Memories.Get())
				v2jwt.PUT("/memories/:id", cfg.Memories.Update())
				v2jwt.DELETE("/memories/:id", cfg.Memories.Delete())
				v2jwt.POST("/memories/:id/tags", cfg.Memories.AddTag())
				if cfg.Modules != nil {
					v2jwt.GET("/modules/:id/memories", cfg.Memories.ModuleMemories())
				}
			}
			// M5 — summarize/distill v2 task lifecycle (JWT-wrapped).
			if cfg.SummarizeDistillV2 != nil {
				v2jwt.GET("/summarize-tasks", cfg.SummarizeDistillV2.ListSummarize())
				v2jwt.POST("/summarize-tasks", cfg.SummarizeDistillV2.CreateSummarize())
				v2jwt.GET("/summarize-tasks/:id", cfg.SummarizeDistillV2.GetSummarize())
				v2jwt.GET("/distill-tasks", cfg.SummarizeDistillV2.ListDistill())
				v2jwt.POST("/distill-tasks", cfg.SummarizeDistillV2.CreateDistill())
				v2jwt.GET("/distill-tasks/:id", cfg.SummarizeDistillV2.GetDistill())
			}
			// M6 — AI tool catalogue + invoke + audit.
			if cfg.AITools != nil {
				v2jwt.GET("/ai-tools", cfg.AITools.List())
				v2jwt.POST("/ai-tools", cfg.AITools.Create())
				v2jwt.GET("/ai-tools/:id", cfg.AITools.Get())
				v2jwt.PUT("/ai-tools/:id", cfg.AITools.Update())
				v2jwt.DELETE("/ai-tools/:id", cfg.AITools.Delete())
				v2jwt.POST("/ai-tools/:id/invoke", cfg.AITools.Invoke())
				v2jwt.GET("/ai-tools/:id/invocations", cfg.AITools.ListInvocations())
				v2jwt.GET("/ai-tools/invocations/:inv_id", cfg.AITools.GetInvocation())
			}
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

		// Distill routes (Task 14). Mounted whenever an LLM provider is
		// wired up — the LLM is the only required dependency. The
		// MemPalace client is only used by the commit step (POST .../commit),
		// so absent MemPalace that single endpoint 503s rather than the
		// whole group disappearing.
		distill := protected.Group("/distill")
		distill.POST("", DistillHandler(cfg.DB, cfg.LLM, cfg.MemPalace))
		distill.GET("/:task_id", GetDistillHandler(cfg.DB))
		if cfg.MemPalace != nil {
			distill.POST("/:task_id/commit", CommitDistillHandler(cfg.DB, cfg.MemPalace))
		} else {
			distill.POST("/:task_id/commit", func(c *gin.Context) {
				Fail(c, http.StatusServiceUnavailable, 5000003, "mempalace client not configured")
			})
		}
	}
}

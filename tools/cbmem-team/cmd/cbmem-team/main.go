// Package main is the entry point for cbmem-team.
//
// cbmem-team is an HTTP wrapper around codebase-memory-mcp that enables
// multiple developers to share a single remote instance while keeping each
// user's code-ast index strictly isolated.
//
// Architecture:
//
//	HTTP client (Cursor/Claude/Qoder)
//	   │   POST /mcp   Authorization: Bearer <jwt>
//	   ▼
//	cbmem-team  ──route by user_id──►  per-user stdio process
//	                                       │
//	                                       ▼
//	                              codebase-memory-mcp <index dir>
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/auth"
	"cbmem-team/internal/config"
	"cbmem-team/internal/console"
	"cbmem-team/internal/httpsrv/middleware"
	"cbmem-team/internal/llm"
	"cbmem-team/internal/mcp"
	"cbmem-team/internal/pool"
	"cbmem-team/internal/reload"
	"cbmem-team/internal/repos"
	"cbmem-team/internal/store"
)

// subcommand dispatch: cbmem-team <serve|migrate|mysql-ping> [flags].
//
// Default subcommand (no args) is "serve" so existing systemd units keep
// working unchanged. The new subcommands are designed for M0.5 ETL and
// production ops.
func main() {
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		switch os.Args[1] {
		case "migrate-tables":
			os.Exit(runMigrateTables(os.Args[2:]))
		case "migrate-sqlite-to-mysql":
			os.Exit(runMigrateSQLiteToMySQL(os.Args[2:]))
		case "mysql-ping":
			os.Exit(runMySQLPing(os.Args[2:]))
		case "help", "--help", "-h":
			fmt.Print(config.Usage)
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n%s\n", os.Args[1], config.Usage)
			os.Exit(2)
		}
	}
	os.Exit(runServe(os.Args[1:]))
}

// runServe opens flags, loads config, and runs the HTTP server with hot reload support.
// It is the original `main()` body, refactored into a function so the subcommand
// dispatcher above can call it. The legacy SQLite path stays the default
// unless `-mysql-dsn` is set, in which case `console.OpenEither` switches
// transparently to MySQL 8.0.
func runServe(args []string) int {
	cfg, flagValues, err := config.ParseFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse flags: %v\n", err)
		return 2
	}

	configPath, _ := flagValues["config_file"]

	if err := runWithHotReload(cfg, configPath); err != nil {
		log.Printf("fatal: %v", err)
		return 1
	}
	return 0
}

// subcommand entry: create the 7 console tables on the configured MySQL.
func runMigrateTables(args []string) int {
	fs := flag.NewFlagSet("migrate-tables", flag.ExitOnError)
	mysqlDSN := fs.String("mysql-dsn", "", "MySQL DSN (required)")
	_ = fs.Parse(args)
	if *mysqlDSN == "" {
		fmt.Fprintln(os.Stderr, "migrate-tables: -mysql-dsn required")
		return 2
	}
	db, err := console.OpenMySQL(*mysqlDSN, console.MySQLOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "open mysql:", err)
		return 3
	}
	defer db.Close()
	if err := db.CreateMySQLSchema(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "create schema:", err)
		return 4
	}
	fmt.Println("migrate-tables: 7 tables created ✓")
	return 0
}

// subcommand entry: one-shot ETL from SQLite console DB to MySQL.
func runMigrateSQLiteToMySQL(args []string) int {
	fs := flag.NewFlagSet("migrate-sqlite-to-mysql", flag.ExitOnError)
	sqlitePath := fs.String("sqlite", "/var/lib/cbmem-team/cbmem-team.db", "source SQLite console DB")
	mysqlDSN := fs.String("mysql-dsn", "", "target MySQL DSN (required)")
	_ = fs.Parse(args)
	if *mysqlDSN == "" {
		fmt.Fprintln(os.Stderr, "migrate-sqlite-to-mysql: -mysql-dsn required")
		return 2
	}
	src, err := console.Open(*sqlitePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open sqlite:", err)
		return 3
	}
	defer src.Close()
	dst, err := console.OpenMySQL(*mysqlDSN, console.MySQLOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "open mysql:", err)
		return 4
	}
	defer dst.Close()

	ctx, cancel := context.WithTimeout(context.Background(), store.EtlTimeout)
	defer cancel()
	counts, err := store.MigrateSQLiteToMySQL(ctx, src.DB, dst.DB)
	if err != nil {
		fmt.Fprintln(os.Stderr, "etl:", err)
		return 5
	}
	fmt.Println("migrate-sqlite-to-mysql: ok")
	for t, n := range counts {
		fmt.Printf("  %-20s %d rows\n", t, n)
	}
	return 0
}

// subcommand entry: ping a MySQL DSN and exit.
func runMySQLPing(args []string) int {
	fs := flag.NewFlagSet("mysql-ping", flag.ExitOnError)
	mysqlDSN := fs.String("mysql-dsn", "", "MySQL DSN (required)")
	_ = fs.Parse(args)
	if *mysqlDSN == "" {
		fmt.Fprintln(os.Stderr, "mysql-ping: -mysql-dsn required")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := console.PingMySQL(ctx, *mysqlDSN); err != nil {
		fmt.Fprintln(os.Stderr, "ping:", err)
		return 3
	}
	fmt.Println("mysql-ping: ok ✓")
	return 0
}

// splitCORSOrigins converts the -cors-allow-origins flag into the slice
// shape middleware.CORS expects. The empty string yields a nil slice so
// the middleware treats CORS as disabled.
func splitCORSOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

// runWithHotReload wraps run() with hot reload support for the config file.
func runWithHotReload(cfg *config.Config, configPath string) error {
	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create hot reload manager
	rm := reload.NewManager()
	if configPath != "" {
		if err := rm.Start(ctx, cfg, configPath); err != nil {
			log.Printf("hot reload: failed to start watcher: %v", err)
			// Non-fatal: continue without hot reload
		} else {
			log.Printf("hot reload: enabled, watching %s", configPath)
		}
	}

	err := run(ctx, cfg, rm)
	if err != nil {
		return err
	}

	// Stop hot reload
	if rm != nil {
		rm.Stop()
	}
	return nil
}

// run starts the HTTP server with the given configuration.
func run(ctx context.Context, cfg *config.Config, rm *reload.Manager) error {
	log.Printf("data_dir: %s", cfg.DataDir)
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("mkdir data: %w", err)
	}
	// M3: ensure the git repo root exists so the first project
	// creation doesn't have to.
	if cfg.RepoRoot != "" {
		if err := os.MkdirAll(cfg.RepoRoot, 0o755); err != nil {
			return fmt.Errorf("mkdir repo_root: %w", err)
		}
	}

	users, err := store.LoadUsers(filepath.Join(cfg.DataDir, "users.json"))
	if err != nil {
		return fmt.Errorf("load users: %w", err)
	}

	// Repos manager discovers any pre-existing clones on disk so restarts
	// don't lose track of admin-managed mirrors.
	reposMgr, err := repos.New(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("init repos: %w", err)
	}

	p := pool.New(pool.Config{
		DataDir:      cfg.DataDir,
		MCPBinary:    cfg.MCPBinary,
		IdleTTL:      cfg.IdleTTL,
		MaxProcs:     cfg.MaxProcs,
		StartTimeout: 15 * time.Second,
	})

	jwtv := auth.NewVerifier([]byte(cfg.JWTSecret))
	adminAuth := auth.NewStaticToken(cfg.AdminToken)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "users": len(users.List())})
	})

	// Console DB is needed by both the console routes AND the /mcp capture
	// middleware, so open it before mounting either. Backend is MySQL when
	// cfg.MySQLDSN is set, otherwise SQLite at <data>/cbmem-team.db.
	consoleDB, err := console.OpenEither(
		filepath.Join(cfg.DataDir, "cbmem-team.db"),
		cfg.MySQLDSN,
		console.MySQLOptions{
			MaxOpenConns: cfg.MySQLMaxOpen,
			MaxIdleConns: cfg.MySQLMaxIdle,
			ConnMaxLife:  cfg.MySQLMaxLife,
		},
	)
	if err != nil {
		return fmt.Errorf("open console db: %w", err)
	}
	defer consoleDB.Close()
	log.Printf("console db: driver=%s", consoleDB.Driver())

	// MCP endpoint - per-user isolated stdio (v1) + streamable (v2) + SSE.
	//
	// Middleware order: JWT first (sets `user_id`), then CaptureSessions
	// (writes rolling session_turns), then CaptureInvocations (writes
	// tool_invocation_logs for every call). Both middlewares read the body
	// before c.Next() and replace it so downstream handlers can drain it.
	//
	// CaptureSessions receives MemPalace so newly captured turns are
	// mirrored into MemPalace in the background (bookkeeping lives on
	// sessions.mempalace_synced_turns). See internal/console/capture.go
	// for the watermark / retry policy.
	var mpCapture *llm.MemPalace
	if cfg.MemPalaceBase != "" {
		mpCapture = llm.NewMemPalace(cfg.MemPalaceBase, cfg.MemPalaceToken)
	}
	mcpGroup := r.Group("/mcp")
	mcpGroup.Use(jwtv.Middleware(),
		console.CaptureSessions(console.CaptureConfig{
			DB:        consoleDB,
			MemPalace: mpCapture,
			Wing:      cfg.MemPalaceAutoSyncWing,
			Hall:      cfg.MemPalaceAutoSyncHall,
		}),
		mcp.CaptureInvocations(consoleDB))
	{
		mcpGroup.POST("", mcp.StreamableHandler(p, users, reposMgr, consoleDB))
		mcpGroup.POST("/", mcp.StreamableHandler(p, users, reposMgr, consoleDB))
		mcpGroup.GET("/sse", jwtv.Middleware(), mcp.SSEHandler())
	}
	// GET /mcp — Streamable HTTP transport requires a GET endpoint for
	// server-to-client SSE notifications. Registered separately so it
	// skips the capture middlewares (POST-only).
	mcpGet := r.Group("/mcp")
	mcpGet.Use(jwtv.Middleware())
	mcpGet.GET("", mcp.StreamableGETHandler())

	// Public JWT refresh: lets a long-lived client (e.g. Cursor) swap a
	// still-valid JWT for a fresh one without bothering the admin.
	r.POST("/refresh", refreshTokenHandler([]byte(cfg.JWTSecret)))

	// Admin endpoints - bootstrapping & user lifecycle
	adminGroup := r.Group("/admin")
	adminGroup.Use(adminAuth.Middleware())
	{
		adminGroup.POST("/users", createUserHandler(users, cfg.DataDir))
		adminGroup.GET("/users", listUsersHandler(users))
		adminGroup.PATCH("/users/:id", patchUserHandler(users))
		adminGroup.DELETE("/users/:id", deleteUserHandler(users, p))
		adminGroup.POST("/users/:id/token", mintTokenHandler(users, []byte(cfg.JWTSecret)))
		adminGroup.GET("/stats", statsHandler(p, users))

		// Repo management: admins register a remote URL and the server
		// clones it into the user's repos dir.
		adminGroup.POST("/repos/clone", cloneRepoHandler(reposMgr))
		adminGroup.GET("/repos", listReposHandler(reposMgr))
	}

	sm := console.NewSessionManager(consoleDB, 8*time.Hour)

	var provider llm.Provider
	switch strings.ToLower(cfg.LLMProvider) {
	case "openai":
		provider = llm.NewOpenAI(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
	case "ollama":
		provider = llm.NewOllama(cfg.LLMBaseURL, cfg.LLMModel)
	default:
		provider = llm.NewFake(`{"hall_facts":[],"hall_events":[],"hall_discoveries":[],"hall_preferences":[],"hall_advice":[]}`)
	}
	var mp *llm.MemPalace
	if cfg.MemPalaceBase != "" {
		mp = llm.NewMemPalace(cfg.MemPalaceBase, cfg.MemPalaceToken)
	}

	authHandlers := console.NewAuthHandlersWithCost(
		consoleDB,
		[]byte(cfg.JWTSecret),
		cfg.AuthAccessTTL,
		cfg.AuthRefreshTTL,
		cfg.AuthInitialAdmin,
		cfg.AuthBcryptCost,
	)

	// M3: JWT verifier shared with auth handlers. We build our own
	// instance (rather than re-using authHandlers.Verifier) so that
	// tests and ops can wire M3 routes independently of M2's
	// user-disabled extra check. The secret MUST match.
	jwtVerifier := auth.NewVerifier([]byte(cfg.JWTSecret))

	// M3: teams + projects v2. RepoRoot is the canonical base
	// directory for v2 projects; configurable via -repo-root
	// (cfg.RepoRoot) so dev / staging / prod can co-exist.
	teamHandlers := console.NewTeamHandlers(consoleDB)
	projectHandlers := console.NewProjectHandlers(consoleDB, cfg.RepoRoot, cfg.MCPBinary)
	// M4: modules + v2 sessions read APIs.
	moduleHandlers := console.NewModuleHandlers(consoleDB)
	sessionsV2Handlers := console.NewSessionHandlersV2(consoleDB)
	// M5: memory templates + memories + summarize/distill v2 wrappers.
	memoryTemplateHandlers := console.NewMemoryTemplateHandlers(consoleDB)
	memoryHandlers := console.NewMemoryHandlers(consoleDB)
	summarizeDistillV2 := console.NewSummarizeDistillHandlersV2(consoleDB, provider, mp)
	// M6: AI tool catalogue (cursor / qoder / superpowers / gstack /
	// custom). Invoke execution is HTTP/stdio and doesn't need an
	// LLM provider at registration time, so we always wire this.
	aiToolsHandlers := console.NewAIToolHandlers(consoleDB)

	console.Mount(r, console.MountConfig{
		DB:                  consoleDB,
		AdminToken:          cfg.AdminToken,
		JWTSecret:           []byte(cfg.JWTSecret),
		JWTVerifier:         jwtVerifier,
		Session:             sm,
		Users:               users,
		LLM:                 provider,
		MemPalace:           mp,
		DataDir:             cfg.DataDir,
		MCPBinary:           cfg.MCPBinary,
		CORS:                middleware.CORS(splitCORSOrigins(cfg.CORSOrigins)),
		Auth:                authHandlers,
		Teams:               teamHandlers,
		Projects:            projectHandlers,
		Modules:             moduleHandlers,
		SessionsV2:          sessionsV2Handlers,
		MemoryTemplates:     memoryTemplateHandlers,
		Memories:            memoryHandlers,
		SummarizeDistillV2:  summarizeDistillV2,
		AITools:             aiToolsHandlers,
	})

	if cfg.ConsoleDist != "" {
		// Serve the SPA static files. Use NoRoute as a fallback so that
		// any unmatched /console/* path returns index.html (SPA history
		// fallback). We avoid registering a second wildcard route which
		// would conflict with r.Static's internal /*filepath.
		r.Static("/console", cfg.ConsoleDist)
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/console/") {
				c.File(filepath.Join(cfg.ConsoleDist, "index.html"))
				return
			}
			c.Status(http.StatusNotFound)
		})
	}

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("cbmem-team listening on %s (data=%s mcp=%s repo_root=%s)", cfg.Listen, cfg.DataDir, cfg.MCPBinary, cfg.RepoRoot)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	if err := p.Close(); err != nil {
		log.Printf("pool shutdown: %v", err)
	}
	log.Println("bye")
	return nil
}

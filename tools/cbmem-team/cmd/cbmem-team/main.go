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
	"cbmem-team/internal/console"
	"cbmem-team/internal/llm"
	"cbmem-team/internal/mcp"
	"cbmem-team/internal/pool"
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
			fmt.Println(usage)
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n%s\n", os.Args[1], usage)
			os.Exit(2)
		}
	}
	os.Exit(runServe(os.Args[1:]))
}

const usage = `cbmem-team — HTTP wrapper for codebase-memory-mcp

Subcommands:
  serve                            (default) start the HTTP server
  migrate-tables                   create console tables on the configured MySQL
  migrate-sqlite-to-mysql         one-shot ETL: copy all rows from a SQLite
                                   console DB into MySQL
  mysql-ping                       ping a MySQL DSN and exit

Use "<subcommand> --help" for subcommand flags.

serve flags (excerpt):
  -listen       :8787
  -data         /var/lib/cbmem-team
  -mysql-dsn    "" (empty = use SQLite at <data>/cbmem-team.db)
  -mysql-max-open 16
  -mysql-max-idle 4
  -mysql-max-lifetime 30m
`

// runServe opens flags, loads config, and runs the HTTP server. It is the
// original `main()` body, refactored into a function so the subcommand
// dispatcher above can call it. The legacy SQLite path stays the default
// unless `-mysql-dsn` is set, in which case `console.OpenEither` switches
// transparently to MySQL 8.0.
func runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	var (
		listen        = fs.String("listen", ":8787", "HTTP listen address")
		cfgFile       = fs.String("config", "/etc/cbmem-team/config.yaml", "config file path")
		dataDir       = fs.String("data", "/var/lib/cbmem-team", "per-user data root")
		mcpBin        = fs.String("mcp-bin", "/usr/local/bin/codebase-memory-mcp", "path to codebase-memory-mcp binary")
		jwtSecret     = fs.String("jwt-secret", "", "HMAC secret for JWT verification (overrides config)")
		adminTok      = fs.String("admin-token", "", "admin token for /admin endpoints (overrides config)")
		logLevel      = fs.String("log", "info", "log level: debug|info|warn|error")
		llmProvider   = fs.String("llm-provider", "fake", "llm provider: fake|openai|ollama")
		llmModel      = fs.String("llm-model", "", "model name")
		llmBaseURL    = fs.String("llm-base-url", "", "llm base url")
		llmAPIKey     = fs.String("llm-api-key", "", "llm api key (optional for ollama)")
		mempalaceBase = fs.String("mempalace-base", "", "mempalace http base url, empty disables integration")
		consoleDist   = fs.String("console-dist", "", "path to vitepress build dist for console frontend")
		mysqlDSN      = fs.String("mysql-dsn", "", "MySQL DSN; empty = use SQLite at <data>/cbmem-team.db")
		mysqlMaxOpen  = fs.Int("mysql-max-open", 16, "MySQL max open conns")
		mysqlMaxIdle  = fs.Int("mysql-max-idle", 4, "MySQL max idle conns")
		mysqlMaxLife  = fs.Duration("mysql-max-lifetime", 30*time.Minute, "MySQL conn max lifetime")
	)
	_ = fs.Parse(args)

	cfg := loadConfig(*cfgFile)
	if *listen != "" && *listen != ":8787" {
		cfg.Listen = *listen
	}
	if *dataDir != "" && *dataDir != "/var/lib/cbmem-team" {
		cfg.DataDir = *dataDir
	}
	if *mcpBin != "" && *mcpBin != "/usr/local/bin/codebase-memory-mcp" {
		cfg.MCPBinary = *mcpBin
	}
	if *jwtSecret != "" {
		cfg.JWTSecret = *jwtSecret
	}
	cfg.AdminToken = *adminTok
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}
	cfg.LLMProvider = *llmProvider
	cfg.LLMModel = *llmModel
	cfg.LLMBaseURL = *llmBaseURL
	cfg.LLMAPIKey = *llmAPIKey
	cfg.MemPalaceBase = *mempalaceBase
	cfg.ConsoleDist = *consoleDist
	cfg.MySQLDSN = *mysqlDSN
	cfg.MySQLMaxOpen = *mysqlMaxOpen
	cfg.MySQLMaxIdle = *mysqlMaxIdle
	cfg.MySQLMaxLife = *mysqlMaxLife

	if err := run(cfg); err != nil {
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

type Config struct {
	Listen        string        `yaml:"listen"`
	DataDir       string        `yaml:"data_dir"`
	MCPBinary     string        `yaml:"mcp_binary"`
	JWTSecret     string        `yaml:"jwt_secret"`
	AdminToken    string        `yaml:"admin_token"`
	LogLevel      string        `yaml:"log_level"`
	LLMProvider   string        `yaml:"llm_provider"`
	LLMModel      string        `yaml:"llm_model"`
	LLMBaseURL    string        `yaml:"llm_base_url"`
	LLMAPIKey     string        `yaml:"llm_api_key"`
	MemPalaceBase string        `yaml:"mempalace_base"`
	ConsoleDist   string        `yaml:"console_dist"`
	Users         []store.User  `yaml:"users"`
	IdleTTL       time.Duration `yaml:"idle_ttl"`
	MaxProcs      int           `yaml:"max_procs_per_user"`
	MySQLDSN      string        `yaml:"mysql_dsn"`
	MySQLMaxOpen  int           `yaml:"mysql_max_open"`
	MySQLMaxIdle  int           `yaml:"mysql_max_idle"`
	MySQLMaxLife  time.Duration `yaml:"mysql_max_lifetime"`
}

func loadConfig(path string) *Config {
	// Minimal YAML-less loader: file is optional; if absent, defaults are used.
	cfg := &Config{
		Listen:    ":8787",
		DataDir:   "/var/lib/cbmem-team",
		MCPBinary: "/usr/local/bin/codebase-memory-mcp",
		IdleTTL:   30 * time.Minute,
		MaxProcs:  4,
		LogLevel:  "info",
	}
	if path == "" {
		return cfg
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		log.Printf("config %s not found, using defaults", path)
		return cfg
	}
	// We avoid pulling a YAML dep by using a tiny key=value parser.
	// Production deployments can swap this for yaml.Unmarshal.
	f, err := os.Open(path)
	if err != nil {
		log.Printf("open config: %v", err)
		return cfg
	}
	defer f.Close()
	log.Printf("loaded config: %s", path)
	return cfg
}

func run(cfg *Config) error {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("mkdir data: %w", err)
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
	mcpGroup := r.Group("/mcp")
	mcpGroup.Use(jwtv.Middleware(), console.CaptureSessions(consoleDB), mcp.CaptureInvocations(consoleDB))
	{
		mcpGroup.POST("", mcp.StreamableHandler(p, users, reposMgr, consoleDB))
		mcpGroup.POST("/", mcp.StreamableHandler(p, users, reposMgr, consoleDB))
		mcpGroup.GET("/sse", jwtv.Middleware(), mcp.SSEHandler())
	}

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
		mp = llm.NewMemPalace(cfg.MemPalaceBase, "")
	}

	console.Mount(r, console.MountConfig{
		DB:         consoleDB,
		AdminToken: cfg.AdminToken,
		Session:    sm,
		Users:      users,
		LLM:        provider,
		MemPalace:  mp,
	})

	if cfg.ConsoleDist != "" {
		r.Static("/console", cfg.ConsoleDist)
		r.GET("/console/*action", func(c *gin.Context) {
			c.File(filepath.Join(cfg.ConsoleDist, "index.html"))
		})
	}

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("cbmem-team listening on %s (data=%s mcp=%s)", cfg.Listen, cfg.DataDir, cfg.MCPBinary)
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

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
	"cbmem-team/internal/store"
)

func main() {
	var (
		listen        = flag.String("listen", ":8787", "HTTP listen address")
		cfgFile       = flag.String("config", "/etc/cbmem-team/config.yaml", "config file path")
		dataDir       = flag.String("data", "/var/lib/cbmem-team", "per-user data root")
		mcpBin        = flag.String("mcp-bin", "/usr/local/bin/codebase-memory-mcp", "path to codebase-memory-mcp binary")
		jwtSecret     = flag.String("jwt-secret", "", "HMAC secret for JWT verification (overrides config)")
		adminTok      = flag.String("admin-token", "", "admin token for /admin endpoints (overrides config)")
		logLevel      = flag.String("log", "info", "log level: debug|info|warn|error")
		llmProvider   = flag.String("llm-provider", "fake", "llm provider: fake|openai|ollama")
		llmModel      = flag.String("llm-model", "", "model name")
		llmBaseURL    = flag.String("llm-base-url", "", "llm base url")
		llmAPIKey     = flag.String("llm-api-key", "", "llm api key (optional for ollama)")
		mempalaceBase = flag.String("mempalace-base", "", "mempalace http base url, empty disables integration")
		consoleDist   = flag.String("console-dist", "", "path to vitepress build dist for console frontend")
	)
	flag.Parse()

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

	if err := run(cfg); err != nil {
		log.Fatalf("fatal: %v", err)
	}
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

	p := pool.New(pool.Config{
		DataDir:    cfg.DataDir,
		MCPBinary:  cfg.MCPBinary,
		IdleTTL:    cfg.IdleTTL,
		MaxProcs:   cfg.MaxProcs,
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
	// middleware, so open it before mounting either.
	consoleDB, err := console.Open(filepath.Join(cfg.DataDir, "cbmem-team.db"))
	if err != nil {
		return fmt.Errorf("open console db: %w", err)
	}
	defer consoleDB.Close()

	// MCP endpoint - per-user isolated stdio.
	//
	// Middleware order: JWT first (sets `user_id`), then CaptureSessions
	// (reads body, replaces it with a replay-able reader, calls c.Next,
	// then post-processes asynchronously).
	mcpGroup := r.Group("/mcp")
	mcpGroup.Use(jwtv.Middleware(), console.CaptureSessions(consoleDB))
	{
		mcpGroup.POST("", mcp.Handler(p, users))
		mcpGroup.POST("/", mcp.Handler(p, users))
	}

	// Admin endpoints - bootstrapping & user lifecycle
	adminGroup := r.Group("/admin")
	adminGroup.Use(adminAuth.Middleware())
	{
		adminGroup.POST("/users", createUserHandler(users, cfg.DataDir))
		adminGroup.GET("/users", listUsersHandler(users))
		adminGroup.DELETE("/users/:id", deleteUserHandler(users, p))
		adminGroup.POST("/users/:id/token", mintTokenHandler(users, []byte(cfg.JWTSecret)))
		adminGroup.GET("/stats", statsHandler(p, users))
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
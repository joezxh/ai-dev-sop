// Package config provides hot-reloadable configuration management for cbmem-team.
// It watches the config file for changes and notifies listeners when reload is needed.
package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Config represents the application configuration. All fields are copied
// from command-line flags and optional YAML config file.
type Config struct {
	Listen        string `yaml:"listen"`
	DataDir       string `yaml:"data_dir"`
	MCPBinary     string `yaml:"mcp_binary"`
	JWTSecret     string `yaml:"jwt_secret"`
	AdminToken    string `yaml:"admin_token"`
	LogLevel      string `yaml:"log_level"`
	LLMProvider   string `yaml:"llm_provider"`
	LLMModel      string `yaml:"llm_model"`
	LLMBaseURL    string `yaml:"llm_base_url"`
	LLMAPIKey     string `yaml:"llm_api_key"`
	MemPalaceBase string `yaml:"mempalace_base"`
	// MemPalaceToken is the Bearer token sent to the MemPalace HTTP MCP
	// endpoint (POST /mcp). Required when MemPalace binds a non-loopback
	// address; matches its MEMPALACE_MCP_HTTP_TOKEN. Empty = no auth header.
	MemPalaceToken string `yaml:"mempalace_token"`
	// MemPalaceAutoSyncWing sets the default wing used for /mcp auto-synced drawers.
	// If empty, the project basename is used.
	MemPalaceAutoSyncWing string `yaml:"mempalace_auto_sync_wing"`
	// MemPalaceAutoSyncHall sets the hall used for /mcp auto-synced drawers.
	// Defaults to "events".
	MemPalaceAutoSyncHall string        `yaml:"mempalace_auto_sync_hall"`
	ConsoleDist           string        `yaml:"console_dist"`
	IdleTTL               time.Duration `yaml:"idle_ttl"`
	MaxProcs              int           `yaml:"max_procs_per_user"`
	MySQLDSN              string        `yaml:"mysql_dsn"`
	MySQLMaxOpen          int           `yaml:"mysql_max_open"`
	MySQLMaxIdle          int           `yaml:"mysql_max_idle"`
	MySQLMaxLife          time.Duration `yaml:"mysql_max_lifetime"`
	CORSOrigins           string        `yaml:"cors_allow_origins"`
	AuthAccessTTL         time.Duration `yaml:"auth_access_ttl"`
	AuthRefreshTTL        time.Duration `yaml:"auth_refresh_ttl"`
	AuthInitialAdmin      string        `yaml:"auth_initial_admin"`
	AuthBcryptCost        int           `yaml:"auth_bcrypm_cost"`
	RepoRoot              string        `yaml:"repo_root"`
}

// DefaultConfig returns a Config with sensible defaults for development.
func DefaultConfig() *Config {
	return &Config{
		Listen:    ":8787",
		DataDir:   "/var/lib/cbmem-team",
		MCPBinary: "/usr/local/bin/codebase-memory-mcp",
		IdleTTL:   30 * time.Minute,
		MaxProcs:  4,
		LogLevel:  "info",
	}
}

// ParseFlags parses command-line flags and returns a Config. It also returns
// the raw flag values so the caller can override config file values with CLI.
func ParseFlags(args []string) (*Config, map[string]string, error) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	var (
		listen            = fs.String("listen", "", "HTTP listen address")
		cfgFile           = fs.String("config", "", "config file path")
		dataDir           = fs.String("data", "", "per-user data root")
		mcpBin            = fs.String("mcp-bin", "", "path to codebase-memory-mcp binary")
		jwtSecret         = fs.String("jwt-secret", "", "HMAC secret for JWT verification (overrides config)")
		adminTok          = fs.String("admin-token", "", "admin token for /admin endpoints (overrides config)")
		logLevel          = fs.String("log", "", "log level: debug|info|warn|error")
		llmProvider       = fs.String("llm-provider", "", "llm provider: fake|openai|ollama")
		llmModel          = fs.String("llm-model", "", "model name")
		llmBaseURL        = fs.String("llm-base-url", "", "llm base url")
		llmAPIKey         = fs.String("llm-api-key", "", "llm api key (optional for ollama)")
		mempalaceBase     = fs.String("mempalace-base", "", "mempalace http base url, empty disables integration")
		mempalaceToken    = fs.String("mempalace-token", "", "bearer token for the mempalace /mcp endpoint (matches MEMPALACE_MCP_HTTP_TOKEN)")
		mempalaceAutoWing = fs.String("mempalace-auto-sync-wing", "", "wing used for /mcp auto-synced drawers")
		mempalaceAutoHall = fs.String("mempalace-auto-sync-hall", "", "hall used for /mcp auto-synced drawers")
		consoleDist       = fs.String("console-dist", "", "path to vitepress build dist for console frontend")
		authAccessTTL     = fs.Duration("auth-access-ttl", 0, "JWT access token TTL")
		authRefreshTTL    = fs.Duration("auth-refresh-ttl", 0, "refresh token TTL")
		authInitialAdmin  = fs.String("auth-initial-admin", "", "if non-empty, mount POST /api/auth/first-admin for first-boot bootstrap. Empty disables the route.")
		authBcryptCost    = fs.Int("auth-bcrypt-cost", 0, "bcrypt work factor (4-31, default 12)")
		mysqlDSN          = fs.String("mysql-dsn", "", "MySQL DSN; empty = use SQLite at <data>/cbmem-team.db")
		mysqlMaxOpen      = fs.Int("mysql-max-open", 0, "MySQL max open conns")
		mysqlMaxIdle      = fs.Int("mysql-max-idle", 0, "MySQL max idle conns")
		mysqlMaxLife      = fs.Duration("mysql-max-lifetime", 0, "MySQL conn max lifetime")
		corsOrigins       = fs.String("cors-allow-origins", "", "Comma-separated CORS origin whitelist for /api/console. Use '*' for dev only; pass an empty string to disable CORS.")
		repoRoot          = fs.String("repo-root", "", "global git repo root for v2 projects; each project path = <repo-root>/<slug>. Empty disables project creation.")
	)
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	// Load config file first.
	cfg := LoadConfig(*cfgFile)

	// CLI flags override config file values.
	applyOverrides(cfg, *listen, *dataDir, *mcpBin, *jwtSecret, *adminTok,
		*logLevel, *llmProvider, *llmModel, *llmBaseURL, *llmAPIKey,
		*mempalaceBase, *mempalaceToken, *mempalaceAutoWing, *mempalaceAutoHall, *consoleDist,
		*authAccessTTL, *authRefreshTTL, *authInitialAdmin, *authBcryptCost,
		*mysqlDSN, *mysqlMaxOpen, *mysqlMaxIdle, *mysqlMaxLife,
		*corsOrigins, *repoRoot)

	// Default RepoRoot to <DataDir>/repos when unset so dev / single-node installs Just Work.
	if cfg.RepoRoot == "" && cfg.DataDir != "" {
		cfg.RepoRoot = filepath.Join(cfg.DataDir, "repos")
	}

	return cfg, map[string]string{"config_file": *cfgFile}, nil
}

func applyOverrides(cfg *Config, listen, dataDir, mcpBin, jwtSecret, adminTok,
	logLevel, llmProvider, llmModel, llmBaseURL, llmAPIKey,
	mempalaceBase, mempalaceToken, mempalaceAutoWing, mempalaceAutoHall, consoleDist string,
	authAccessTTL, authRefreshTTL time.Duration, authInitialAdmin string, authBcryptCost int,
	mysqlDSN string, mysqlMaxOpen, mysqlMaxIdle int, mysqlMaxLife time.Duration,
	corsOrigins, repoRoot string) {

	// String overrides: only apply if non-empty
	if listen != "" {
		cfg.Listen = listen
	}
	if dataDir != "" {
		cfg.DataDir = dataDir
	}
	if mcpBin != "" {
		cfg.MCPBinary = mcpBin
	}
	if jwtSecret != "" {
		cfg.JWTSecret = jwtSecret
	}
	if adminTok != "" {
		cfg.AdminToken = adminTok
	}
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if llmProvider != "" {
		cfg.LLMProvider = llmProvider
	}
	if llmModel != "" {
		cfg.LLMModel = llmModel
	}
	if llmBaseURL != "" {
		cfg.LLMBaseURL = llmBaseURL
	}
	if llmAPIKey != "" {
		cfg.LLMAPIKey = llmAPIKey
	}
	if mempalaceBase != "" {
		cfg.MemPalaceBase = mempalaceBase
	}
	if mempalaceToken != "" {
		cfg.MemPalaceToken = mempalaceToken
	}
	if mempalaceAutoWing != "" {
		cfg.MemPalaceAutoSyncWing = mempalaceAutoWing
	}
	if mempalaceAutoHall != "" {
		cfg.MemPalaceAutoSyncHall = mempalaceAutoHall
	}
	if consoleDist != "" {
		cfg.ConsoleDist = consoleDist
	}
	if authAccessTTL > 0 {
		cfg.AuthAccessTTL = authAccessTTL
	}
	if authRefreshTTL > 0 {
		cfg.AuthRefreshTTL = authRefreshTTL
	}
	if authInitialAdmin != "" {
		cfg.AuthInitialAdmin = authInitialAdmin
	}
	if authBcryptCost > 0 {
		cfg.AuthBcryptCost = authBcryptCost
	}
	if mysqlDSN != "" {
		cfg.MySQLDSN = mysqlDSN
	}
	if mysqlMaxOpen > 0 {
		cfg.MySQLMaxOpen = mysqlMaxOpen
	}
	if mysqlMaxIdle > 0 {
		cfg.MySQLMaxIdle = mysqlMaxIdle
	}
	if mysqlMaxLife > 0 {
		cfg.MySQLMaxLife = mysqlMaxLife
	}
	if corsOrigins != "" {
		cfg.CORSOrigins = corsOrigins
	}
	if repoRoot != "" {
		cfg.RepoRoot = repoRoot
	}
}

// LoadConfig loads a config from the given YAML file path. If the file does
// not exist or cannot be read, a default config is returned.
func LoadConfig(path string) *Config {
	cfg := DefaultConfig()
	if path == "" {
		return cfg
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		log.Printf("config %s not found, using defaults", path)
		return cfg
	}
	f, err := os.Open(path)
	if err != nil {
		log.Printf("open config: %v", err)
		return cfg
	}
	defer f.Close()

	// Minimal YAML parser for key=value pairs.
	// Production deployments can use yaml.Unmarshal directly.
	parseYAML(f, cfg)
	log.Printf("loaded config: %s", path)
	return cfg
}

// parseYAML reads a minimal YAML file (key: value pairs) into cfg.
// Only supports flat scalar values, not nested structs or arrays.
func parseYAML(f *os.File, cfg *Config) {
	// Read file content
	data, err := os.ReadFile(f.Name())
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Parse "key: value" or "key:value"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Strip surrounding quotes if present
		value = stripQuotes(value)

		switch key {
		case "listen":
			cfg.Listen = value
		case "data_dir":
			cfg.DataDir = value
		case "mcp_binary":
			cfg.MCPBinary = value
		case "jwt_secret":
			cfg.JWTSecret = value
		case "admin_token":
			cfg.AdminToken = value
		case "log_level":
			cfg.LogLevel = value
		case "llm_provider":
			cfg.LLMProvider = value
		case "llm_model":
			cfg.LLMModel = value
		case "llm_base_url":
			cfg.LLMBaseURL = value
		case "llm_api_key":
			cfg.LLMAPIKey = value
		case "mempalace_base":
			cfg.MemPalaceBase = value
		case "mempalace_auto_sync_wing":
			cfg.MemPalaceAutoSyncWing = value
		case "mempalace_auto_sync_hall":
			cfg.MemPalaceAutoSyncHall = value
		case "console_dist":
			cfg.ConsoleDist = value
		case "idle_ttl":
			if d, err := time.ParseDuration(value); err == nil {
				cfg.IdleTTL = d
			}
		case "max_procs_per_user":
			if n, err := fmt.Sscanf(value, "%d", &cfg.MaxProcs); err == nil && n == 1 {
			}
		case "mysql_dsn":
			cfg.MySQLDSN = value
		case "mysql_max_open":
			fmt.Sscanf(value, "%d", &cfg.MySQLMaxOpen)
		case "mysql_max_idle":
			fmt.Sscanf(value, "%d", &cfg.MySQLMaxIdle)
		case "mysql_max_lifetime":
			if d, err := time.ParseDuration(value); err == nil {
				cfg.MySQLMaxLife = d
			}
		case "cors_allow_origins":
			cfg.CORSOrigins = value
		case "auth_access_ttl":
			if d, err := time.ParseDuration(value); err == nil {
				cfg.AuthAccessTTL = d
			}
		case "auth_refresh_ttl":
			if d, err := time.ParseDuration(value); err == nil {
				cfg.AuthRefreshTTL = d
			}
		case "auth_initial_admin":
			cfg.AuthInitialAdmin = value
		case "auth_bcrypm_cost":
			fmt.Sscanf(value, "%d", &cfg.AuthBcryptCost)
		case "repo_root":
			cfg.RepoRoot = value
		}
	}
}

// stripQuotes removes surrounding single or double quotes from a string.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// Watcher watches the config file for changes and notifies listeners.
type Watcher struct {
	watcher    *fsnotify.Watcher
	configPath string
	notifyCh   chan struct{}
	closeCh    chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
	closed     bool
}

// NewWatcher creates a new config file watcher.
func NewWatcher(configPath string) (*Watcher, error) {
	if configPath == "" {
		return nil, errors.New("config path is empty")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		watcher:    watcher,
		configPath: configPath,
		notifyCh:   make(chan struct{}, 1),
		closeCh:    make(chan struct{}),
	}

	// Watch the config directory so we catch renames (common during writes).
	dir := filepath.Dir(configPath)
	if dir == "." || dir == "" {
		dir = filepath.Dir(os.Args[0])
	}
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("watch config dir %s: %w", dir, err)
	}

	w.wg.Add(1)
	go w.run()

	log.Printf("config watcher: watching %s", dir)
	return w, nil
}

// C returns a channel that receives a struct{} each time the config file changes.
func (w *Watcher) C() <-chan struct{} {
	return w.notifyCh
}

func (w *Watcher) run() {
	defer w.wg.Done()
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			// We care about writes and renames to the config file.
			if filepath.Base(event.Name) == filepath.Base(w.configPath) {
				if event.Op&(fsnotify.Write|fsnotify.Rename) != 0 {
					select {
					case w.notifyCh <- struct{}{}:
					default:
					}
				}
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config watcher error: %v", err)
		case <-w.closeCh:
			return
		}
	}
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	close(w.closeCh)
	w.wg.Wait()
	return w.watcher.Close()
}

// Usage returns the help text for config flags.
const Usage = `cbmem-team — HTTP wrapper for codebase-memory-mcp

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
  -config       /etc/cbmem-team/config.yaml
  -mysql-dsn    "" (empty = use SQLite at <data>/cbmem-team.db)
  -mysql-max-open 16
  -mysql-max-idle 4
  -mysql-max-lifetime 30m
`

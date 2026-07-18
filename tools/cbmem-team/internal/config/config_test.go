package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Listen != ":8787" {
		t.Errorf("expected Listen :8787, got %s", cfg.Listen)
	}
	if cfg.DataDir != "/var/lib/cbmem-team" {
		t.Errorf("expected DataDir /var/lib/cbmem-team, got %s", cfg.DataDir)
	}
	if cfg.MCPBinary != "/usr/local/bin/codebase-memory-mcp" {
		t.Errorf("expected MCPBinary /usr/local/bin/codebase-memory-mcp, got %s", cfg.MCPBinary)
	}
	if cfg.IdleTTL != 30*time.Minute {
		t.Errorf("expected IdleTTL 30m, got %v", cfg.IdleTTL)
	}
	if cfg.MaxProcs != 4 {
		t.Errorf("expected MaxProcs 4, got %d", cfg.MaxProcs)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel info, got %s", cfg.LogLevel)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	cfg := LoadConfig("/nonexistent/path/config.yaml")
	// Should return defaults when file doesn't exist
	if cfg.Listen != ":8787" {
		t.Errorf("expected default Listen, got %s", cfg.Listen)
	}
}

func TestLoadConfig_ValidYAML(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	content := `listen: ":9999"
data_dir: /tmp/test-data
log_level: debug
llm_provider: openai
llm_model: gpt-4
auth_access_ttl: 30m
auth_refresh_ttl: 168h
max_procs_per_user: 8
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg := LoadConfig(cfgPath)

	if cfg.Listen != ":9999" {
		t.Errorf("expected Listen :9999, got %s", cfg.Listen)
	}
	if cfg.DataDir != "/tmp/test-data" {
		t.Errorf("expected DataDir /tmp/test-data, got %s", cfg.DataDir)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel debug, got %s", cfg.LogLevel)
	}
	if cfg.LLMProvider != "openai" {
		t.Errorf("expected LLMProvider openai, got %s", cfg.LLMProvider)
	}
	if cfg.LLMModel != "gpt-4" {
		t.Errorf("expected LLMModel gpt-4, got %s", cfg.LLMModel)
	}
	if cfg.AuthAccessTTL != 30*time.Minute {
		t.Errorf("expected AuthAccessTTL 30m, got %v", cfg.AuthAccessTTL)
	}
	if cfg.AuthRefreshTTL != 168*time.Hour {
		t.Errorf("expected AuthRefreshTTL 168h, got %v", cfg.AuthRefreshTTL)
	}
	if cfg.MaxProcs != 8 {
		t.Errorf("expected MaxProcs 8, got %d", cfg.MaxProcs)
	}
}

func TestLoadConfig_EmptyPath(t *testing.T) {
	cfg := LoadConfig("")
	if cfg.Listen != ":8787" {
		t.Errorf("expected default Listen, got %s", cfg.Listen)
	}
}

func TestParseFlags(t *testing.T) {
	cfg, _, err := ParseFlags([]string{
		"-listen", ":9999",
		"-data", "/tmp/data",
		"-jwt-secret", "test-secret",
		"-log", "debug",
		"-llm-provider", "openai",
		"-llm-model", "gpt-4",
		"-llm-base-url", "https://api.openai.com",
		"-auth-access-ttl", "30m",
		"-auth-refresh-ttl", "168h",
	})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if cfg.Listen != ":9999" {
		t.Errorf("expected Listen :9999, got %s", cfg.Listen)
	}
	if cfg.DataDir != "/tmp/data" {
		t.Errorf("expected DataDir /tmp/data, got %s", cfg.DataDir)
	}
	if cfg.JWTSecret != "test-secret" {
		t.Errorf("expected JWTSecret test-secret, got %s", cfg.JWTSecret)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel debug, got %s", cfg.LogLevel)
	}
	if cfg.LLMProvider != "openai" {
		t.Errorf("expected LLMProvider openai, got %s", cfg.LLMProvider)
	}
	if cfg.LLMModel != "gpt-4" {
		t.Errorf("expected LLMModel gpt-4, got %s", cfg.LLMModel)
	}
	if cfg.LLMBaseURL != "https://api.openai.com" {
		t.Errorf("expected LLMBaseURL https://api.openai.com, got %s", cfg.LLMBaseURL)
	}
	if cfg.AuthAccessTTL != 30*time.Minute {
		t.Errorf("expected AuthAccessTTL 30m, got %v", cfg.AuthAccessTTL)
	}
	if cfg.AuthRefreshTTL != 168*time.Hour {
		t.Errorf("expected AuthRefreshTTL 168h, got %v", cfg.AuthRefreshTTL)
	}
}

func TestParseFlags_WithConfigFile(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	content := `listen: ":8888"
data_dir: "/config-data"
log_level: "warn"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	// CLI flag should override config file value
	cfg, _, err := ParseFlags([]string{
		"-config", cfgPath,
		"-listen", ":9999",
	})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	// CLI flag should take precedence
	if cfg.Listen != ":9999" {
		t.Errorf("expected Listen :9999 (CLI override), got %s", cfg.Listen)
	}
	// Config file value should be loaded
	if cfg.DataDir != "/config-data" {
		t.Errorf("expected DataDir /config-data, got %s", cfg.DataDir)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("expected LogLevel warn, got %s", cfg.LogLevel)
	}
}

func TestParseFlags_RepoRootDefault(t *testing.T) {
	cfg, _, err := ParseFlags([]string{
		"-data", "/tmp/test-data",
	})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	// RepoRoot should default to <DataDir>/repos
	expected := filepath.Join("/tmp/test-data", "repos")
	if cfg.RepoRoot != expected {
		t.Errorf("expected RepoRoot %s, got %s", expected, cfg.RepoRoot)
	}
}

func TestParseFlags_RepoRootExplicit(t *testing.T) {
	cfg, _, err := ParseFlags([]string{
		"-data", "/tmp/test-data",
		"-repo-root", "/explicit/repos",
	})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if cfg.RepoRoot != "/explicit/repos" {
		t.Errorf("expected RepoRoot /explicit/repos, got %s", cfg.RepoRoot)
	}
}

func TestNewWatcher_EmptyPath(t *testing.T) {
	_, err := NewWatcher("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestNewWatcher_DirectoryNotExist(t *testing.T) {
	_, err := NewWatcher("/nonexistent/directory/config.yaml")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestUsage(t *testing.T) {
	if Usage == "" {
		t.Error("Usage should not be empty")
	}
	if len(Usage) < 100 {
		t.Error("Usage should contain substantial help text")
	}
}

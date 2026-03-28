package config

import (
	"errors"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/viper"
)

// Load loads configuration from file and environment
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server_url", "https://agentsmesh.ai")
	v.SetDefault("max_concurrent_pods", 5)
	v.SetDefault("workspace_root", DefaultWorkspaceRoot())
	v.SetDefault("mcp_port", 19000)
	v.SetDefault("health_check_port", 9090)
	v.SetDefault("log_level", "info")
	v.SetDefault("default_agent", "claude-code")

	// Auto-update defaults
	v.SetDefault("auto_update.enabled", true)
	v.SetDefault("auto_update.check_interval", 24*time.Hour)
	v.SetDefault("auto_update.channel", "stable")
	v.SetDefault("auto_update.max_wait_time", 30*time.Minute)
	v.SetDefault("auto_update.auto_apply", true)

	// Read from environment
	v.SetEnvPrefix("AGENTSMESH")
	v.AutomaticEnv()

	// Read from config file if specified
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// Search for config in common locations
		v.SetConfigName("runner")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".agentsmesh"))
		}
		if runtime.GOOS != "windows" {
			v.AddConfigPath("/etc/agentsmesh")
		}
	}

	if err := v.ReadInConfig(); err != nil {
		// Config file not found is okay if we have env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			slog.Error("Failed to read config file", "error", err)
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		slog.Error("Failed to unmarshal config", "error", err)
		return nil, err
	}

	// Generate node ID if not set
	if cfg.NodeID == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "runner"
		}
		cfg.NodeID = hostname
	}

	// Expand workspace root
	if cfg.WorkspaceRoot != "" {
		cfg.WorkspaceRoot = os.ExpandEnv(cfg.WorkspaceRoot)
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ServerURL == "" {
		return errors.New("server_url is required")
	}

	// gRPC/mTLS is required - validate certificate configuration
	if err := c.validateGRPCConfig(); err != nil {
		return err
	}

	if c.MaxConcurrentPods < 1 {
		return errors.New("max_concurrent_pods must be at least 1")
	}

	// Ensure workspace root exists
	if c.WorkspaceRoot != "" {
		if err := os.MkdirAll(c.WorkspaceRoot, 0755); err != nil {
			return errors.New("failed to create workspace root: " + err.Error())
		}
	}

	return nil
}

// RewriteRelayURL replaces the origin (scheme://host:port) of the given relay URL
// with RelayBaseURL, preserving the path and query. Returns the original URL unchanged
// if RelayBaseURL is not configured or parsing fails.
func (c *Config) RewriteRelayURL(relayURL string) string {
	if c.RelayBaseURL == "" {
		return relayURL
	}

	orig, err := url.Parse(relayURL)
	if err != nil {
		return relayURL
	}

	base, err := url.Parse(c.RelayBaseURL)
	if err != nil {
		return relayURL
	}

	// Replace origin, keep path and query from original
	orig.Scheme = base.Scheme
	orig.Host = base.Host
	return orig.String()
}

// DefaultWorkspaceRoot returns a platform-appropriate default workspace root.
// On Windows: %LOCALAPPDATA%\agentsmesh\workspace (fallback to ~/.agentsmesh/workspace).
// On Unix (Docker/server): /workspace (container convention).
// Exported so register.go can use the same logic.
func DefaultWorkspaceRoot() string {
	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "agentsmesh", "workspace")
		}
		// Fallback when LOCALAPPDATA is not set (e.g., containers)
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".agentsmesh", "workspace")
		}
	}
	return "/workspace"
}

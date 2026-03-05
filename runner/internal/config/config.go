package config

import (
	"errors"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all runner configuration
type Config struct {
	// Server connection
	ServerURL string `mapstructure:"server_url"`

	// Runner identification
	NodeID      string `mapstructure:"node_id"`
	Description string `mapstructure:"description"`

	// mTLS Certificate Authentication (gRPC)
	CertFile     string `mapstructure:"cert_file"`     // Path to client certificate
	KeyFile      string `mapstructure:"key_file"`      // Path to client private key
	CAFile       string `mapstructure:"ca_file"`       // Path to CA certificate
	GRPCEndpoint string `mapstructure:"grpc_endpoint"` // gRPC server endpoint (e.g., grpc.example.com:9443)

	// Organization (set during registration, used for org-scoped API paths)
	OrgSlug string `mapstructure:"org_slug"`

	// Capacity
	MaxConcurrentPods int `mapstructure:"max_concurrent_pods"`

	// Workspace settings
	WorkspaceRoot string `mapstructure:"workspace_root"`
	GitConfigPath string `mapstructure:"git_config_path"`

	// Git settings (for ticket-based development)
	RepositoryPath string `mapstructure:"repository_path"` // Path to the main git repository
	BaseBranch     string `mapstructure:"base_branch"`     // Base branch for new git worktrees (default: main)

	// MCP settings
	MCPConfigPath string `mapstructure:"mcp_config_path"` // Path to MCP servers config file
	MCPPort       int    `mapstructure:"mcp_port"`        // MCP HTTP Server port (default: 19000)

	// Sandbox settings
	Workspace string `mapstructure:"workspace"` // Workspace root for sandboxes and repos cache

	// Agent settings
	DefaultAgent string            `mapstructure:"default_agent"`
	DefaultShell string            `mapstructure:"default_shell"` // Default shell for pods
	AgentEnvVars map[string]string `mapstructure:"agent_env_vars"`

	// Plugin settings
	PluginsDir string `mapstructure:"plugins_dir"` // User custom plugins directory (default: ~/.agentsmesh/plugins)

	// Health check
	HealthCheckPort int `mapstructure:"health_check_port"`

	// Logging
	LogLevel string `mapstructure:"log_level"`
	LogFile  string `mapstructure:"log_file"`

	// PTY logging (for debugging)
	LogPTY    bool   `mapstructure:"log_pty"`     // Enable PTY output logging
	LogPTYDir string `mapstructure:"log_pty_dir"` // PTY log directory (default: $TMPDIR/agentsmesh/pty-logs)

	// Auto-update settings
	AutoUpdate AutoUpdateConfig `mapstructure:"auto_update"`

	// Version is set programmatically from build-time ldflags, not from config file
	Version string `yaml:"-" mapstructure:"-"`

	// ConfigFilePath is set programmatically to track where config was loaded from.
	// Not stored in config file.
	ConfigFilePath string `yaml:"-" mapstructure:"-"`

	// ResolvedPATH is the login shell PATH resolved at startup.
	// Used to inject a usable PATH into PTY environments when running as a service.
	ResolvedPATH string `yaml:"-" mapstructure:"-"`
}

// AutoUpdateConfig holds auto-update configuration.
type AutoUpdateConfig struct {
	// Enabled controls whether auto-update is enabled (default: true)
	Enabled bool `mapstructure:"enabled"`

	// CheckInterval is how often to check for updates (default: 24h)
	CheckInterval time.Duration `mapstructure:"check_interval"`

	// Channel is the update channel: "stable" or "beta" (default: "stable")
	// "stable" = only stable releases (v1.0.0)
	// "beta" = includes prereleases (v1.1.0-beta.1, v1.1.0-rc.1)
	Channel string `mapstructure:"channel"`

	// MaxWaitTime is the maximum time to wait for pods to finish before postponing update (default: 30m)
	MaxWaitTime time.Duration `mapstructure:"max_wait_time"`

	// AutoApply controls whether to automatically apply updates (default: true)
	// If false, only check and download, notify user but don't apply
	AutoApply bool `mapstructure:"auto_apply"`
}

// Load loads configuration from file and environment
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server_url", "https://agentsmesh.ai")
	v.SetDefault("max_concurrent_pods", 5)
	v.SetDefault("workspace_root", "/workspace")
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
		v.AddConfigPath("$HOME/.agentsmesh")
		v.AddConfigPath("/etc/agentsmesh")
	}

	if err := v.ReadInConfig(); err != nil {
		// Config file not found is okay if we have env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
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

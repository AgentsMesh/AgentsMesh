package config

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
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

	// Worktree settings (for ticket-based development)
	RepositoryPath string `mapstructure:"repository_path"` // Path to the main git repository
	WorktreesDir   string `mapstructure:"worktrees_dir"`   // Directory for worktrees
	BaseBranch     string `mapstructure:"base_branch"`     // Base branch for new worktrees (default: main)

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

	// Auto-update settings
	AutoUpdate AutoUpdateConfig `mapstructure:"auto_update"`
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
	v.SetDefault("server_url", "https://api.agentsmesh.ai")
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

// SaveOrgSlug saves the organization slug to a file for persistence
func (c *Config) SaveOrgSlug(orgSlug string) error {
	c.OrgSlug = orgSlug

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".agentsmesh")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	orgSlugFile := filepath.Join(configDir, "org_slug")
	return os.WriteFile(orgSlugFile, []byte(orgSlug), 0600)
}

// LoadOrgSlug loads the organization slug from file if not in config
func (c *Config) LoadOrgSlug() error {
	if c.OrgSlug != "" {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil // Not an error
	}

	orgSlugFile := filepath.Join(home, ".agentsmesh", "org_slug")
	data, err := os.ReadFile(orgSlugFile)
	if err != nil {
		return nil // Not an error
	}

	c.OrgSlug = string(data)
	return nil
}

// GetWorkspace returns the workspace directory path.
// Falls back to WorkspaceRoot if Workspace is not set.
func (c *Config) GetWorkspace() string {
	if c.Workspace != "" {
		return c.Workspace
	}
	if c.WorkspaceRoot != "" {
		return c.WorkspaceRoot
	}
	// Default to user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/agentsmesh"
	}
	return filepath.Join(home, ".agentsmesh")
}

// GetSandboxesDir returns the sandboxes directory path.
func (c *Config) GetSandboxesDir() string {
	return filepath.Join(c.GetWorkspace(), "sandboxes")
}

// GetReposDir returns the repository cache directory path.
func (c *Config) GetReposDir() string {
	return filepath.Join(c.GetWorkspace(), "repos")
}

// GetMCPPort returns the MCP HTTP Server port.
func (c *Config) GetMCPPort() int {
	if c.MCPPort > 0 {
		return c.MCPPort
	}
	return 19000 // Default port
}

// GetPluginsDir returns the user plugins directory path.
// Returns empty string if no plugins directory is configured.
func (c *Config) GetPluginsDir() string {
	if c.PluginsDir != "" {
		return os.ExpandEnv(c.PluginsDir)
	}
	// Default to ~/.agentsmesh/plugins
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agentsmesh", "plugins")
}

// GetLogPath returns the log file path.
func (c *Config) GetLogPath() string {
	if c.LogFile != "" {
		return os.ExpandEnv(c.LogFile)
	}
	// Default to system temp directory (can be safely deleted)
	return filepath.Join(os.TempDir(), "agentsmesh", "runner.log")
}

// GetLogConfig returns the logger configuration.
func (c *Config) GetLogConfig() logger.Config {
	return logger.Config{
		Level:       c.LogLevel,
		FilePath:    c.GetLogPath(),
		Format:      "text", // Default to human-readable format
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		MaxBackups:  3,                 // Keep 3 backup files
	}
}

// ==================== gRPC/mTLS Configuration ====================

// UsesGRPC returns true if gRPC mode is configured (certificates present).
func (c *Config) UsesGRPC() bool {
	return c.CertFile != "" && c.KeyFile != "" && c.CAFile != "" && c.GRPCEndpoint != ""
}

// validateGRPCConfig validates gRPC-specific configuration.
func (c *Config) validateGRPCConfig() error {
	if c.GRPCEndpoint == "" {
		return errors.New("grpc_endpoint is required for gRPC mode")
	}
	if c.CertFile == "" {
		return errors.New("cert_file is required for gRPC mode")
	}
	if c.KeyFile == "" {
		return errors.New("key_file is required for gRPC mode")
	}
	if c.CAFile == "" {
		return errors.New("ca_file is required for gRPC mode")
	}

	// Verify certificate files exist
	if _, err := os.Stat(c.CertFile); os.IsNotExist(err) {
		return errors.New("certificate file not found: " + c.CertFile)
	}
	if _, err := os.Stat(c.KeyFile); os.IsNotExist(err) {
		return errors.New("private key file not found: " + c.KeyFile)
	}
	if _, err := os.Stat(c.CAFile); os.IsNotExist(err) {
		return errors.New("CA certificate file not found: " + c.CAFile)
	}

	return nil
}

// GetCertsDir returns the certificates directory path.
func (c *Config) GetCertsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/etc/agentsmesh/certs"
	}
	return filepath.Join(home, ".agentsmesh", "certs")
}

// SaveCertificates saves gRPC certificates to the default location.
func (c *Config) SaveCertificates(certPEM, keyPEM, caCertPEM []byte) error {
	certsDir := c.GetCertsDir()
	if err := os.MkdirAll(certsDir, 0700); err != nil {
		return err
	}

	// Save certificate
	certPath := filepath.Join(certsDir, "runner.crt")
	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return err
	}

	// Save private key
	keyPath := filepath.Join(certsDir, "runner.key")
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return err
	}

	// Save CA certificate
	caPath := filepath.Join(certsDir, "ca.crt")
	if err := os.WriteFile(caPath, caCertPEM, 0644); err != nil {
		return err
	}

	// Update config paths
	c.CertFile = certPath
	c.KeyFile = keyPath
	c.CAFile = caPath

	return nil
}

// SaveGRPCEndpoint saves the gRPC endpoint to config.
func (c *Config) SaveGRPCEndpoint(endpoint string) error {
	c.GRPCEndpoint = endpoint

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".agentsmesh")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	endpointFile := filepath.Join(configDir, "grpc_endpoint")
	return os.WriteFile(endpointFile, []byte(endpoint), 0600)
}

// LoadGRPCConfig loads gRPC configuration from files if not already set.
func (c *Config) LoadGRPCConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil // Not an error
	}

	// Load gRPC endpoint
	if c.GRPCEndpoint == "" {
		endpointFile := filepath.Join(home, ".agentsmesh", "grpc_endpoint")
		if data, err := os.ReadFile(endpointFile); err == nil {
			c.GRPCEndpoint = string(data)
		}
	}

	// Set certificate paths if files exist
	certsDir := filepath.Join(home, ".agentsmesh", "certs")
	if c.CertFile == "" {
		certPath := filepath.Join(certsDir, "runner.crt")
		if _, err := os.Stat(certPath); err == nil {
			c.CertFile = certPath
		}
	}
	if c.KeyFile == "" {
		keyPath := filepath.Join(certsDir, "runner.key")
		if _, err := os.Stat(keyPath); err == nil {
			c.KeyFile = keyPath
		}
	}
	if c.CAFile == "" {
		caPath := filepath.Join(certsDir, "ca.crt")
		if _, err := os.Stat(caPath); err == nil {
			c.CAFile = caPath
		}
	}

	return nil
}

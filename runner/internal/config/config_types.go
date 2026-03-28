package config

import "time"

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

	// Relay settings
	// RelayBaseURL overrides the origin (scheme://host:port) of relay URLs received from Backend.
	// Used in Docker environments where Runner cannot reach the external PRIMARY_DOMAIN.
	// Example: "ws://traefik:80" rewrites "ws://localhost:31650/relay" -> "ws://traefik:80/relay"
	RelayBaseURL string `mapstructure:"relay_base_url"`

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

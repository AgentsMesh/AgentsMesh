package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the relay service
type Config struct {
	// Server settings
	Server ServerConfig `mapstructure:"server"`

	// JWT settings for token validation
	JWT JWTConfig `mapstructure:"jwt"`

	// Backend connection settings
	Backend BackendConfig `mapstructure:"backend"`

	// Session settings
	Session SessionConfig `mapstructure:"session"`

	// Relay identity
	Relay RelayConfig `mapstructure:"relay"`

	// =============================================================================
	// Unified Domain Configuration - Single source of truth for public URLs
	// If PRIMARY_DOMAIN is set, RELAY_URL is derived as ws(s)://{PRIMARY_DOMAIN}/relay
	// =============================================================================
	PrimaryDomain string `mapstructure:"primary_domain"` // e.g., "localhost:10000" or "agentsmesh.ai"
	UseHTTPS      bool   `mapstructure:"use_https"`      // Use wss:// instead of ws://
}

// ServerConfig holds HTTP/WebSocket server configuration
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	TLS          TLSConfig     `mapstructure:"tls"`
}

// TLSConfig holds TLS/SSL configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"` // Path to certificate file (PEM)
	KeyFile  string `mapstructure:"key_file"`  // Path to private key file (PEM)
}

// JWTConfig holds JWT validation configuration
type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Issuer string `mapstructure:"issuer"`
}

// BackendConfig holds backend API configuration
type BackendConfig struct {
	URL               string        `mapstructure:"url"`
	InternalAPISecret string        `mapstructure:"internal_api_secret"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
}

// SessionConfig holds session management configuration
type SessionConfig struct {
	KeepAliveDuration        time.Duration `mapstructure:"keep_alive_duration"`
	MaxBrowsersPerPod        int           `mapstructure:"max_browsers_per_pod"`
	RunnerReconnectTimeout   time.Duration `mapstructure:"runner_reconnect_timeout"`   // How long to wait for runner to reconnect
	BrowserReconnectTimeout  time.Duration `mapstructure:"browser_reconnect_timeout"`  // How long to wait for browser to reconnect
	PendingConnectionTimeout time.Duration `mapstructure:"pending_connection_timeout"` // How long to wait for counterpart connection
	OutputBufferSize         int           `mapstructure:"output_buffer_size"`         // Max bytes for output buffer
	OutputBufferCount        int           `mapstructure:"output_buffer_count"`        // Max messages for output buffer
}

// RelayConfig holds relay identity configuration
type RelayConfig struct {
	ID       string `mapstructure:"id"`
	Name     string `mapstructure:"name"`    // Relay name for DNS auto-registration (e.g., "us-east-1")
	URL      string `mapstructure:"url"`     // Public URL for browsers and runners (auto-generated from PRIMARY_DOMAIN)
	Region   string `mapstructure:"region"`
	Capacity int    `mapstructure:"capacity"`
	AutoIP   bool   `mapstructure:"auto_ip"` // Auto-detect public IP for DNS registration
}

// Load loads configuration from environment variables and config file
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8090)
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)

	v.SetDefault("jwt.issuer", "agentsmesh-relay")

	v.SetDefault("backend.url", "http://backend:8080")
	v.SetDefault("backend.heartbeat_interval", 10*time.Second)

	v.SetDefault("session.keep_alive_duration", 30*time.Second)
	v.SetDefault("session.max_browsers_per_pod", 10)
	v.SetDefault("session.runner_reconnect_timeout", 30*time.Second)
	v.SetDefault("session.browser_reconnect_timeout", 30*time.Second)
	v.SetDefault("session.pending_connection_timeout", 60*time.Second)
	v.SetDefault("session.output_buffer_size", 256*1024) // 256KB
	v.SetDefault("session.output_buffer_count", 200)

	v.SetDefault("relay.capacity", 1000)
	v.SetDefault("relay.region", "default")

	// Enable environment variable reading
	v.SetEnvPrefix("RELAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Map specific environment variables
	envMappings := map[string]string{
		// Unified domain configuration
		"PRIMARY_DOMAIN": "primary_domain",
		"USE_HTTPS":      "use_https",
		// Server
		"SERVER_HOST":  "server.host",
		"SERVER_PORT":  "server.port",
		"TLS_ENABLED":  "server.tls.enabled",
		"TLS_CERT_FILE": "server.tls.cert_file",
		"TLS_KEY_FILE":  "server.tls.key_file",
		// JWT
		"JWT_SECRET": "jwt.secret",
		"JWT_ISSUER": "jwt.issuer",
		// Backend
		"BACKEND_URL":         "backend.url",
		"INTERNAL_API_SECRET": "backend.internal_api_secret",
		"HEARTBEAT_INTERVAL":  "backend.heartbeat_interval",
		// Session
		"KEEP_ALIVE_DURATION":        "session.keep_alive_duration",
		"MAX_BROWSERS_PER_POD":       "session.max_browsers_per_pod",
		"RUNNER_RECONNECT_TIMEOUT":   "session.runner_reconnect_timeout",
		"BROWSER_RECONNECT_TIMEOUT":  "session.browser_reconnect_timeout",
		"PENDING_CONNECTION_TIMEOUT": "session.pending_connection_timeout",
		"OUTPUT_BUFFER_SIZE":         "session.output_buffer_size",
		"OUTPUT_BUFFER_COUNT":        "session.output_buffer_count",
		// Relay identity
		"RELAY_ID":           "relay.id",
		"RELAY_NAME":         "relay.name",
		"RELAY_URL":          "relay.url",
		"RELAY_REGION":       "relay.region",
		"RELAY_CAPACITY":     "relay.capacity",
		"RELAY_AUTO_IP":      "relay.auto_ip",
	}

	for env, key := range envMappings {
		if val := os.Getenv(env); val != "" {
			v.Set(key, val)
		}
		// Also try with RELAY_ prefix, but skip keys that already start with RELAY_
		// to avoid double-prefixed lookups like RELAY_RELAY_ID
		if !strings.HasPrefix(env, "RELAY_") {
			if val := os.Getenv("RELAY_" + env); val != "" {
				v.Set(key, val)
			}
		}
	}

	// Try to read config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/relay")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			slog.Warn("Failed to read config file", "error", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	if cfg.Backend.InternalAPISecret == "" {
		return nil, fmt.Errorf("INTERNAL_API_SECRET is required")
	}

	if cfg.Relay.ID == "" {
		// Generate a default ID based on hostname
		hostname, _ := os.Hostname()
		cfg.Relay.ID = fmt.Sprintf("relay-%s", hostname)
	}

	// Derive RELAY_URL from PRIMARY_DOMAIN if not explicitly set
	if cfg.Relay.URL == "" {
		if cfg.PrimaryDomain != "" {
			// Derive from PRIMARY_DOMAIN (unified domain configuration)
			scheme := "ws"
			if cfg.UseHTTPS {
				scheme = "wss"
			}
			cfg.Relay.URL = fmt.Sprintf("%s://%s/relay", scheme, cfg.PrimaryDomain)
		} else {
			// Fallback to local address
			scheme := "ws"
			if cfg.Server.TLS.Enabled {
				scheme = "wss"
			}
			cfg.Relay.URL = fmt.Sprintf("%s://localhost:%d", scheme, cfg.Server.Port)
		}
	}

	// Note: TLS certificate validation is skipped because we support dynamic certificate loading from Backend via ACME.
	// The server will use GetCertificate callback to load certificate from backend client or fall back to files if available.

	return &cfg, nil
}

// Address returns the server listen address
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

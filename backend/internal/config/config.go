package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	OAuth    OAuthConfig
	Webhook  WebhookConfig
	Log      LogConfig
	Email    EmailConfig
	Storage  StorageConfig
}

// StorageConfig holds object storage configuration (S3-compatible)
type StorageConfig struct {
	Endpoint       string   // S3 endpoint (empty for AWS, set for MinIO/OSS)
	PublicEndpoint string   // Public endpoint for browser access (if different from Endpoint)
	Region         string   // AWS region or equivalent
	Bucket         string   // Bucket name
	AccessKey      string   // Access key ID
	SecretKey      string   // Secret access key
	UseSSL         bool     // Use HTTPS
	UsePathStyle   bool     // Use path-style URLs (required for MinIO)
	MaxFileSize    int64    // Max file size in MB
	AllowedTypes   []string // Allowed MIME types
}

// EmailConfig holds email service configuration
type EmailConfig struct {
	Provider    string // "resend" or "console"
	ResendKey   string
	FromAddress string
	BaseURL     string // Frontend base URL
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level      string // debug, info, warn, error
	Format     string // json, text
	FilePath   string // path to log file, empty means stdout only
	MaxSizeMB  int    // max size in MB before rotation
	MaxBackups int    // max number of backup files
}

// WebhookConfig holds webhook secret configurations
type WebhookConfig struct {
	GitLabSecret string
	GitHubSecret string
	GiteeSecret  string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Address            string
	Debug              bool
	CORSAllowedOrigins []string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	SSLMode     string
	ReplicaDSNs []string // Read replica DSNs for read-write separation
}

// DSN returns the PostgreSQL connection string for the master database
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// HasReplicas returns true if replica DSNs are configured
func (c DatabaseConfig) HasReplicas() bool {
	return len(c.ReplicaDSNs) > 0
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Host     string
	Port     int
	Password string
	DB       int
}

// IsConfigured returns true if Redis is configured
func (c RedisConfig) IsConfigured() bool {
	return c.URL != "" || c.Host != ""
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret          string
	ExpirationHours int
}

// OAuthConfig holds OAuth provider configurations
type OAuthConfig struct {
	DefaultRedirectURL string
	GitHub             OAuthProviderConfig
	Google             OAuthProviderConfig
	GitLab             GitLabOAuthConfig
	Gitee              OAuthProviderConfig
}

// OAuthProviderConfig holds OAuth provider configuration
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// GitLabOAuthConfig holds GitLab OAuth provider configuration
type GitLabOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	BaseURL      string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Address:            getEnv("SERVER_ADDRESS", ":8080"),
			Debug:              getEnvBool("DEBUG", false),
			CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS", []string{"*"}),
		},
		Database: DatabaseConfig{
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        getEnvInt("DB_PORT", 5432),
			User:        getEnv("DB_USER", "agentsmesh"),
			Password:    getEnv("DB_PASSWORD", ""),
			DBName:      getEnv("DB_NAME", "agentsmesh"),
			SSLMode:     getEnv("DB_SSLMODE", "disable"),
			ReplicaDSNs: getEnvList("DB_REPLICA_DSNS", nil),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", ""),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "change-me-in-production"),
			ExpirationHours: getEnvInt("JWT_EXPIRATION_HOURS", 24),
		},
		OAuth: OAuthConfig{
			DefaultRedirectURL: getEnv("OAUTH_DEFAULT_REDIRECT_URL", "/"),
			GitHub: OAuthProviderConfig{
				ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GITHUB_REDIRECT_URL", ""),
			},
			Google: OAuthProviderConfig{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
			},
			GitLab: GitLabOAuthConfig{
				ClientID:     getEnv("GITLAB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITLAB_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GITLAB_REDIRECT_URL", ""),
				BaseURL:      getEnv("GITLAB_BASE_URL", "https://gitlab.com"),
			},
			Gitee: OAuthProviderConfig{
				ClientID:     getEnv("GITEE_CLIENT_ID", ""),
				ClientSecret: getEnv("GITEE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GITEE_REDIRECT_URL", ""),
			},
		},
		Webhook: WebhookConfig{
			GitLabSecret: getEnv("GITLAB_WEBHOOK_SECRET", ""),
			GitHubSecret: getEnv("GITHUB_WEBHOOK_SECRET", ""),
			GiteeSecret:  getEnv("GITEE_WEBHOOK_SECRET", ""),
		},
		Log: LogConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "text"),
			FilePath:   getEnv("LOG_FILE", "logs/agentsmesh.log"),
			MaxSizeMB:  getEnvInt("LOG_MAX_SIZE_MB", 100),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 5),
		},
		Email: EmailConfig{
			Provider:    getEnv("EMAIL_PROVIDER", "console"),
			ResendKey:   getEnv("RESEND_API_KEY", ""),
			FromAddress: getEnv("EMAIL_FROM_ADDRESS", "AgentsMesh <noreply@agentsmesh.dev>"),
			BaseURL:     getEnv("FRONTEND_BASE_URL", "http://localhost:3000"),
		},
		Storage: StorageConfig{
			Endpoint:       getEnv("STORAGE_ENDPOINT", ""),
			PublicEndpoint: getEnv("STORAGE_PUBLIC_ENDPOINT", ""),
			Region:         getEnv("STORAGE_REGION", "us-east-1"),
			Bucket:         getEnv("STORAGE_BUCKET", "agentsmesh"),
			AccessKey:      getEnv("STORAGE_ACCESS_KEY", ""),
			SecretKey:      getEnv("STORAGE_SECRET_KEY", ""),
			UseSSL:         getEnvBool("STORAGE_USE_SSL", true),
			UsePathStyle:   getEnvBool("STORAGE_USE_PATH_STYLE", false),
			MaxFileSize:    int64(getEnvInt("STORAGE_MAX_FILE_SIZE", 10)),
			AllowedTypes:   getEnvList("STORAGE_ALLOWED_TYPES", []string{"image/jpeg", "image/png", "image/gif", "image/webp", "application/pdf"}),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvList(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Split by comma
		parts := []string{}
		for _, part := range splitAndTrim(value, ",") {
			if part != "" {
				parts = append(parts, part)
			}
		}
		if len(parts) > 0 {
			return parts
		}
	}
	return defaultValue
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		result = append(result, trimmed)
	}
	return result
}

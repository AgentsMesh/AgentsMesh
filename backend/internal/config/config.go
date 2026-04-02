package config

import (
	"log/slog"
	"time"
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
	Payment  PaymentConfig
	PKI      PKIConfig
	GRPC     GRPCConfig
	Admin    AdminConfig
	Relay       RelayConfig
	Marketplace MarketplaceConfig

	// Unified domain configuration - all URLs are derived from these two values
	PrimaryDomain string // Primary domain (e.g., "localhost:10000" or "agentsmesh.ai")
	UseHTTPS      bool   // Use HTTPS/WSS protocols
}

// MarketplaceConfig configures the platform-level Marketplace for Skills and MCP servers.
// Platform-level Skill Registries are managed via Admin API (stored in DB).
// MCP servers are synced from the official MCP Registry.
type MarketplaceConfig struct {
	SyncInterval    time.Duration // Interval between marketplace sync cycles (e.g., "1h", "30m")
	RegistryEnabled bool          // Enable MCP Registry sync (default: true)
	RegistryURL     string        // MCP Registry API URL (default: https://registry.modelcontextprotocol.io)
}

// Load loads configuration from environment variables
// All URLs are derived from PRIMARY_DOMAIN and USE_HTTPS
func Load() (*Config, error) {
	return &Config{
		// Unified Domain Configuration
		PrimaryDomain: getEnv("PRIMARY_DOMAIN", "localhost:10000"),
		UseHTTPS:      getEnvBool("USE_HTTPS", false),

		// Server Configuration
		Server: ServerConfig{
			Address:            getEnv("SERVER_ADDRESS", ":8080"),
			Debug:              getEnvBool("DEBUG", false),
			CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS", []string{"*"}),
			InternalAPISecret:  getEnv("INTERNAL_API_SECRET", "change-me-internal-secret"),
		},

		// Database Configuration
		Database: DatabaseConfig{
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        getEnvInt("DB_PORT", 5432),
			User:        getEnv("DB_USER", "agentsmesh"),
			Password:    getEnv("DB_PASSWORD", ""),
			DBName:      getEnv("DB_NAME", "agentsmesh"),
			SSLMode:     getEnv("DB_SSLMODE", "disable"),
			ReplicaDSNs: getEnvList("DB_REPLICA_DSNS", nil),
		},

		// Redis Configuration
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", ""),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},

		// JWT Configuration
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "change-me-in-production"),
			ExpirationHours: getEnvInt("JWT_EXPIRATION_HOURS", 24),
		},

		// OAuth Configuration
		OAuth: OAuthConfig{
			DefaultRedirectURL: getEnv("OAUTH_DEFAULT_REDIRECT_URL", "/"),
			GitHub: OAuthProviderConfig{
				ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
			},
			Google: OAuthProviderConfig{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			},
			GitLab: GitLabOAuthConfig{
				ClientID:     getEnv("GITLAB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITLAB_CLIENT_SECRET", ""),
				BaseURL:      getEnv("GITLAB_BASE_URL", "https://gitlab.com"),
			},
			Gitee: OAuthProviderConfig{
				ClientID:     getEnv("GITEE_CLIENT_ID", ""),
				ClientSecret: getEnv("GITEE_CLIENT_SECRET", ""),
			},
		},

		// Webhook Configuration
		Webhook: WebhookConfig{
			GitLabSecret: getEnv("GITLAB_WEBHOOK_SECRET", ""),
			GitHubSecret: getEnv("GITHUB_WEBHOOK_SECRET", ""),
			GiteeSecret:  getEnv("GITEE_WEBHOOK_SECRET", ""),
		},

		// Logging Configuration
		Log: LogConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "text"),
			FilePath:   getEnv("LOG_FILE", "logs/agentsmesh.log"),
			MaxSizeMB:  getEnvInt("LOG_MAX_SIZE_MB", 100),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 5),
		},

		// Email Configuration
		Email: EmailConfig{
			Provider:    getEnv("EMAIL_PROVIDER", "console"),
			ResendKey:   getEnv("RESEND_API_KEY", ""),
			FromAddress: getEnv("EMAIL_FROM_ADDRESS", "AgentsMesh <noreply@agentsmesh.ai>"),
		},

		// Storage Configuration
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

		// Payment Configuration
		Payment: PaymentConfig{
			DeploymentType: DeploymentType(getEnv("DEPLOYMENT_TYPE", "global")),
			MockEnabled:    getEnvBool("PAYMENT_MOCK", false),
			MockBaseURL:    getEnv("PAYMENT_MOCK_BASE_URL", ""),
			Stripe: StripeConfig{
				SecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
				PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
				WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
			},
			LemonSqueezy: LemonSqueezyConfig{
				APIKey:        getEnv("LEMONSQUEEZY_API_KEY", ""),
				StoreID:       getEnv("LEMONSQUEEZY_STORE_ID", ""),
				WebhookSecret: getEnv("LEMONSQUEEZY_WEBHOOK_SECRET", ""),
			},
			Alipay: AlipayConfig{
				AppID:           getEnv("ALIPAY_APP_ID", ""),
				PrivateKey:      getEnv("ALIPAY_PRIVATE_KEY", ""),
				AlipayPublicKey: getEnv("ALIPAY_PUBLIC_KEY", ""),
				IsSandbox:       getEnvBool("ALIPAY_SANDBOX", false),
			},
			WeChat: WeChatConfig{
				AppID:     getEnv("WECHAT_APP_ID", ""),
				MchID:     getEnv("WECHAT_MCH_ID", ""),
				APIKey:    getEnv("WECHAT_API_KEY", ""),
				APIv3Key:  getEnv("WECHAT_APIV3_KEY", ""),
				CertPath:  getEnv("WECHAT_CERT_PATH", ""),
				KeyPath:   getEnv("WECHAT_KEY_PATH", ""),
				IsSandbox: getEnvBool("WECHAT_SANDBOX", false),
			},
			License: LicenseConfig{
				PublicKeyPath:    getEnv("LICENSE_PUBLIC_KEY_PATH", ""),
				LicenseFilePath:  getEnv("LICENSE_FILE_PATH", ""),
				LicenseServerURL: getEnv("LICENSE_SERVER_URL", ""),
			},
		},

		// PKI Configuration
		PKI: PKIConfig{
			CACertFile:     getEnv("PKI_CA_CERT_FILE", ""),
			CAKeyFile:      getEnv("PKI_CA_KEY_FILE", ""),
			ServerCertFile: getEnv("PKI_SERVER_CERT_FILE", ""),
			ServerKeyFile:  getEnv("PKI_SERVER_KEY_FILE", ""),
			ValidityDays:   getEnvInt("PKI_VALIDITY_DAYS", 365),
		},

		// gRPC Configuration
		GRPC: GRPCConfig{
			Address:  getEnv("GRPC_ADDRESS", ":9090"),
			Endpoint: getEnv("GRPC_PUBLIC_ENDPOINT", ""),
		},

		// Admin Configuration
		Admin: AdminConfig{
			Enabled: getEnvBool("ADMIN_ENABLED", true),
		},

		// Marketplace Configuration
		Marketplace: MarketplaceConfig{
			SyncInterval:    getEnvDuration("MARKETPLACE_SYNC_INTERVAL", 1*time.Hour),
			RegistryEnabled: getEnvBool("MCP_REGISTRY_ENABLED", true),
			RegistryURL:     getEnv("MCP_REGISTRY_URL", "https://registry.modelcontextprotocol.io"),
		},

		// Relay Management Configuration
		Relay: RelayConfig{
			BaseDomain: getEnv("RELAY_BASE_DOMAIN", ""),
			DNS: DNSConfig{
				Provider:              getEnv("DNS_PROVIDER", ""),
				CloudflareAPIToken:    getEnv("CLOUDFLARE_API_TOKEN", ""),
				CloudflareZoneID:      getEnv("CLOUDFLARE_ZONE_ID", ""),
				AliyunAccessKeyID:     getEnv("ALIYUN_ACCESS_KEY_ID", ""),
				AliyunAccessKeySecret: getEnv("ALIYUN_ACCESS_KEY_SECRET", ""),
			},
			ACME: ACMEConfig{
				Enabled:      getEnvBool("ACME_ENABLED", false),
				Email:        getEnv("ACME_EMAIL", ""),
				DirectoryURL: getEnv("ACME_DIRECTORY_URL", ""),
				StorageDir:   getEnv("ACME_STORAGE_DIR", "/var/lib/agentsmesh/acme"),
				Staging:      getEnvBool("ACME_STAGING", false),
			},
		},
	}, nil
}

// WarnInsecureDefaults logs warnings for insecure default configuration values.
// Should be called at startup after loading configuration.
func (c *Config) WarnInsecureDefaults() {
	if c.Server.InternalAPISecret == "change-me-internal-secret" {
		slog.Warn("SECURITY: INTERNAL_API_SECRET is using the default value; set a strong random secret via environment variable")
	}
	if c.JWT.Secret == "change-me-in-production" {
		if c.Server.Debug {
			slog.Warn("SECURITY: JWT_SECRET is using the default value; set a strong random secret via environment variable")
		} else {
			slog.Error("SECURITY: JWT_SECRET is using the default value in non-debug mode; this is a critical security risk — set JWT_SECRET environment variable")
		}
	}
}


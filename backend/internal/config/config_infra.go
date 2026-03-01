package config

// PKIConfig holds PKI (certificate) configuration for Runner mTLS authentication
// Required for Runner communication via gRPC + mTLS
type PKIConfig struct {
	CACertFile     string // Path to CA certificate file (required)
	CAKeyFile      string // Path to CA private key file (required)
	ServerCertFile string // Path to server certificate file (optional, generated if not set)
	ServerKeyFile  string // Path to server private key file (optional)
	ValidityDays   int    // Certificate validity period in days (default: 365)
}

// GRPCConfig holds gRPC server configuration for Runner connections
// gRPC server starts automatically when PKI CA files are configured
type GRPCConfig struct {
	Address  string // gRPC server listen address (default: :9090)
	Endpoint string // Public gRPC endpoint URL for Runners (e.g., grpcs://api.example.com:9443)
}

// AdminConfig holds admin console configuration
// Note: FrontendURL is derived from Config.PrimaryDomain + "/admin"
type AdminConfig struct {
	Enabled bool // Enable admin console
}

// IsEnabled returns true if admin console is enabled
func (c AdminConfig) IsEnabled() bool {
	return c.Enabled
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
// Note: BaseURL is derived from Config.PrimaryDomain
type EmailConfig struct {
	Provider    string // "resend" or "console"
	ResendKey   string
	FromAddress string
}

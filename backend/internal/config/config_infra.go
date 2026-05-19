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

type GRPCConfig struct {
	Address  string // gRPC server listen address (default: :9090)
	Endpoint string // Public gRPC endpoint URL for Runners (e.g., grpcs://api.agentsmesh.cn:9443)
}

type AdminConfig struct {
	Enabled bool // Enable admin console
}

func (c AdminConfig) IsEnabled() bool {
	return c.Enabled
}

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

type EmailConfig struct {
	Provider    string // "resend" or "console"
	ResendKey   string
	FromAddress string
}

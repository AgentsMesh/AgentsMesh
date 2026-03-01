package config

// RelayConfig holds Relay server management configuration
type RelayConfig struct {
	// Base domain for relay subdomains (e.g., "relay.example.com")
	// Used by DNSService to generate A records like "01.relay.example.com → IP"
	BaseDomain string

	// DNS provider configuration
	DNS DNSConfig

	// ACME (Let's Encrypt) configuration for automatic TLS certificates
	ACME ACMEConfig
}

// DNSConfig holds DNS provider configuration
type DNSConfig struct {
	Provider string // "cloudflare" or "aliyun"

	// Cloudflare settings
	CloudflareAPIToken string // API token with DNS edit permissions
	CloudflareZoneID   string // Zone ID for the domain

	// Aliyun DNS settings
	AliyunAccessKeyID     string
	AliyunAccessKeySecret string
}

// ACMEConfig holds ACME (Let's Encrypt) configuration
type ACMEConfig struct {
	Enabled      bool   // Enable ACME certificate management
	Email        string // Email for Let's Encrypt registration
	DirectoryURL string // ACME directory URL (empty for production Let's Encrypt)
	StorageDir   string // Directory to store certificates (default: /var/lib/agentsmesh/acme)
	Staging      bool   // Use Let's Encrypt staging environment
}

// IsEnabled returns true if DNS management is configured
func (c RelayConfig) IsEnabled() bool {
	return c.BaseDomain != "" && c.DNS.Provider != ""
}

// DNSProviderType represents supported DNS providers
type DNSProviderType string

const (
	DNSProviderCloudflare DNSProviderType = "cloudflare"
	DNSProviderAliyun     DNSProviderType = "aliyun"
)

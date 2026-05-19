package config

type RelayConfig struct {
	BaseDomain string

	DNS DNSConfig

	ACME ACMEConfig
}

type DNSConfig struct {
	Provider string // "cloudflare" or "aliyun"

	CloudflareAPIToken string // API token with DNS edit permissions
	CloudflareZoneID   string // Zone ID for the domain

	AliyunAccessKeyID     string
	AliyunAccessKeySecret string
}

type ACMEConfig struct {
	Enabled      bool   // Enable ACME certificate management
	Email        string // Email for Let's Encrypt registration
	DirectoryURL string // ACME directory URL (empty for production Let's Encrypt)
	StorageDir   string // Directory to store certificates (default: /var/lib/agentsmesh/acme)
	Staging      bool   // Use Let's Encrypt staging environment
}

func (c RelayConfig) IsEnabled() bool {
	return c.BaseDomain != "" && c.DNS.Provider != ""
}

type DNSProviderType string

const (
	DNSProviderCloudflare DNSProviderType = "cloudflare"
	DNSProviderAliyun     DNSProviderType = "aliyun"
)

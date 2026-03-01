package dns

import (
	"context"
	"fmt"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
)

// Provider is the interface for DNS management
type Provider interface {
	// CreateRecord creates an A record for the given subdomain
	CreateRecord(ctx context.Context, subdomain, ip string) error

	// DeleteRecord deletes the A record for the given subdomain
	DeleteRecord(ctx context.Context, subdomain string) error

	// GetRecord returns the IP address for the given subdomain, or empty if not found
	GetRecord(ctx context.Context, subdomain string) (string, error)

	// UpdateRecord updates the A record for the given subdomain
	UpdateRecord(ctx context.Context, subdomain, ip string) error
}

// NewProvider creates a DNS provider based on configuration
func NewProvider(cfg config.DNSConfig) (Provider, error) {
	switch cfg.Provider {
	case string(config.DNSProviderCloudflare):
		if cfg.CloudflareAPIToken == "" || cfg.CloudflareZoneID == "" {
			return nil, fmt.Errorf("cloudflare requires API token and zone ID")
		}
		return NewCloudflareProvider(cfg.CloudflareAPIToken, cfg.CloudflareZoneID), nil

	case string(config.DNSProviderAliyun):
		if cfg.AliyunAccessKeyID == "" || cfg.AliyunAccessKeySecret == "" {
			return nil, fmt.Errorf("aliyun requires access key ID and secret")
		}
		return NewAliyunProvider(cfg.AliyunAccessKeyID, cfg.AliyunAccessKeySecret), nil

	case "":
		return nil, fmt.Errorf("DNS provider not configured")

	default:
		return nil, fmt.Errorf("unsupported DNS provider: %s", cfg.Provider)
	}
}

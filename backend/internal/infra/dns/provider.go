package dns

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/config"
)

type Provider interface {
	CreateRecord(ctx context.Context, subdomain, ip string) error

	DeleteRecord(ctx context.Context, subdomain string) error

	GetRecord(ctx context.Context, subdomain string) (string, error)

	UpdateRecord(ctx context.Context, subdomain, ip string) error
}

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

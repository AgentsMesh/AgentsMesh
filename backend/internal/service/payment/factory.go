package payment

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	alipayprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/alipay"
	licenseprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/license"
	mockprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/mock"
	stripeprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/stripe"
	wechatprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/wechat"
)

// Factory creates payment providers based on configuration
type Factory struct {
	config          *config.PaymentConfig
	db              *gorm.DB
	mockProvider    *mockprovider.Provider    // Singleton mock provider instance
	licenseProvider *licenseprovider.Provider // Singleton license provider instance
}

// NewFactoryWithDB creates a new payment provider factory with database support
func NewFactoryWithDB(cfg *config.PaymentConfig, db *gorm.DB) *Factory {
	f := &Factory{config: cfg, db: db}

	// Initialize mock provider if enabled
	if cfg.MockEnabled {
		baseURL := cfg.MockBaseURL
		if baseURL == "" {
			baseURL = "http://localhost:3000" // Default to local frontend
		}
		f.mockProvider = mockprovider.NewProvider(baseURL)
	}

	// Initialize license provider if enabled and db is available
	if cfg.LicenseEnabled() && db != nil {
		licenseProvider, err := licenseprovider.NewProvider(&cfg.License, db)
		if err == nil {
			f.licenseProvider = licenseProvider
		}
	}

	return f
}

// GetProvider returns the appropriate provider for the given provider name
func (f *Factory) GetProvider(providerName string) (Provider, error) {
	// If mock is enabled, always return mock provider
	if f.config.MockEnabled {
		if f.mockProvider == nil {
			return nil, fmt.Errorf("mock provider not initialized")
		}
		return f.mockProvider, nil
	}

	switch providerName {
	case billing.PaymentProviderStripe:
		if !f.config.StripeEnabled() {
			return nil, fmt.Errorf("stripe is not configured")
		}
		return stripeprovider.NewProvider(&f.config.Stripe), nil

	case billing.PaymentProviderAlipay:
		if !f.config.AlipayEnabled() {
			return nil, fmt.Errorf("alipay is not configured")
		}
		return alipayprovider.NewProvider(&f.config.Alipay)

	case billing.PaymentProviderWeChat:
		if !f.config.WeChatEnabled() {
			return nil, fmt.Errorf("wechat is not configured")
		}
		return wechatprovider.NewProvider(&f.config.WeChat)

	case billing.PaymentProviderLicense:
		if !f.config.LicenseEnabled() {
			return nil, fmt.Errorf("license is not configured")
		}
		if f.licenseProvider == nil {
			return nil, fmt.Errorf("license provider not initialized (database required)")
		}
		return f.licenseProvider, nil

	case "mock":
		if f.mockProvider == nil {
			return nil, fmt.Errorf("mock provider not enabled")
		}
		return f.mockProvider, nil

	default:
		return nil, fmt.Errorf("unknown payment provider: %s", providerName)
	}
}

// GetDefaultProvider returns the default provider based on deployment type
func (f *Factory) GetDefaultProvider() (Provider, error) {
	// If mock is enabled, always return mock provider
	if f.config.MockEnabled {
		return f.GetProvider("mock")
	}

	switch f.config.DeploymentType {
	case config.DeploymentGlobal:
		return f.GetProvider(billing.PaymentProviderStripe)
	case config.DeploymentCN:
		// Default to Alipay for China deployment
		return f.GetProvider(billing.PaymentProviderAlipay)
	case config.DeploymentOnPremise:
		return f.GetProvider(billing.PaymentProviderLicense)
	default:
		return nil, fmt.Errorf("unknown deployment type: %s", f.config.DeploymentType)
	}
}

// GetMockProvider returns the mock provider instance (for mock checkout handling)
func (f *Factory) GetMockProvider() *mockprovider.Provider {
	return f.mockProvider
}

// GetLicenseProvider returns the license provider instance (for license-specific operations)
func (f *Factory) GetLicenseProvider() *licenseprovider.Provider {
	return f.licenseProvider
}

// IsMockEnabled returns true if mock provider is enabled
func (f *Factory) IsMockEnabled() bool {
	return f.config.MockEnabled
}

// GetAvailableProviders returns all configured and available providers
func (f *Factory) GetAvailableProviders() []string {
	return f.config.GetAvailableProviders()
}

// IsProviderAvailable checks if a specific provider is available
func (f *Factory) IsProviderAvailable(providerName string) bool {
	for _, p := range f.GetAvailableProviders() {
		if p == providerName {
			return true
		}
	}
	return false
}

// GetDeploymentType returns the current deployment type
func (f *Factory) GetDeploymentType() config.DeploymentType {
	return f.config.DeploymentType
}

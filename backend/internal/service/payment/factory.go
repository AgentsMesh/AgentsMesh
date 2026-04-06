package payment

import (
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	alipayprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/alipay"
	lemonsqueezyprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/lemonsqueezy"
	licenseprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/license"
	mockprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/mock"
	stripeprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/stripe"
	wechatprovider "github.com/anthropics/agentsmesh/backend/internal/service/payment/wechat"
)

// Factory creates payment providers based on configuration
type Factory struct {
	appConfig       *config.Config            // Full app config for URL derivation
	config          *config.PaymentConfig     // Payment config
	licenseRepo     billing.LicenseRepository // Optional, for license provider
	mockProvider    *mockprovider.Provider    // Singleton mock provider instance
	licenseProvider *licenseprovider.Provider // Singleton license provider instance
}

// NewFactoryFromConfig creates a payment provider factory without database support.
// License provider will not be available. Use NewFactoryWithLicenseRepo if license support is needed.
func NewFactoryFromConfig(appConfig *config.Config) *Factory {
	return NewFactoryWithLicenseRepo(appConfig, nil)
}

// NewFactoryWithLicenseRepo creates a new payment provider factory with license repository support
// appConfig is needed for URL derivation (AlipayNotifyURL, WeChatNotifyURL, etc.)
func NewFactoryWithLicenseRepo(appConfig *config.Config, licenseRepo billing.LicenseRepository) *Factory {
	cfg := &appConfig.Payment
	f := &Factory{appConfig: appConfig, config: cfg, licenseRepo: licenseRepo}

	// Initialize mock provider if enabled
	if cfg.MockEnabled {
		baseURL := cfg.MockBaseURL
		if baseURL == "" {
			baseURL = appConfig.FrontendURL() // Use derived frontend URL
		}
		f.mockProvider = mockprovider.NewProvider(baseURL)
	}

	// Initialize license provider if enabled and license repo is available
	if cfg.LicenseEnabled() && licenseRepo != nil {
		licenseProvider, err := licenseprovider.NewProvider(&cfg.License, licenseRepo)
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
	case billing.PaymentProviderLemonSqueezy:
		if !f.config.LemonSqueezyEnabled() {
			return nil, fmt.Errorf("lemonsqueezy is not configured")
		}
		return lemonsqueezyprovider.NewProvider(&f.config.LemonSqueezy), nil

	case billing.PaymentProviderStripe:
		if !f.config.StripeEnabled() {
			return nil, fmt.Errorf("stripe is not configured")
		}
		return stripeprovider.NewProvider(&f.config.Stripe), nil

	case billing.PaymentProviderAlipay:
		if !f.config.AlipayEnabled() {
			return nil, fmt.Errorf("alipay is not configured")
		}
		// URLs are derived from appConfig.PrimaryDomain
		return alipayprovider.NewProvider(&f.config.Alipay,
			f.appConfig.AlipayNotifyURL(),
			f.appConfig.AlipayReturnURL())

	case billing.PaymentProviderWeChat:
		if !f.config.WeChatEnabled() {
			return nil, fmt.Errorf("wechat is not configured")
		}
		// URL is derived from appConfig.PrimaryDomain
		return wechatprovider.NewProvider(&f.config.WeChat,
			f.appConfig.WeChatNotifyURL())

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
		// Use LemonSqueezy as the default provider for global deployment
		return f.GetProvider(billing.PaymentProviderLemonSqueezy)
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

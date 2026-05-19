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

type Factory struct {
	appConfig       *config.Config            // Full app config for URL derivation
	config          *config.PaymentConfig     // Payment config
	licenseRepo     billing.LicenseRepository // Optional, for license provider
	mockProvider    *mockprovider.Provider    // Singleton mock provider instance
	licenseProvider *licenseprovider.Provider // Singleton license provider instance
}

func NewFactoryFromConfig(appConfig *config.Config) *Factory {
	return NewFactoryWithLicenseRepo(appConfig, nil)
}

func NewFactoryWithLicenseRepo(appConfig *config.Config, licenseRepo billing.LicenseRepository) *Factory {
	cfg := &appConfig.Payment
	f := &Factory{appConfig: appConfig, config: cfg, licenseRepo: licenseRepo}

	if cfg.MockEnabled {
		baseURL := cfg.MockBaseURL
		if baseURL == "" {
			baseURL = appConfig.FrontendURL() // Use derived frontend URL
		}
		f.mockProvider = mockprovider.NewProvider(baseURL)
	}

	if cfg.LicenseEnabled() && licenseRepo != nil {
		licenseProvider, err := licenseprovider.NewProvider(&cfg.License, licenseRepo)
		if err == nil {
			f.licenseProvider = licenseProvider
		}
	}

	return f
}

func (f *Factory) GetProvider(providerName string) (Provider, error) {
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
		return alipayprovider.NewProvider(&f.config.Alipay,
			f.appConfig.AlipayNotifyURL(),
			f.appConfig.AlipayReturnURL())

	case billing.PaymentProviderWeChat:
		if !f.config.WeChatEnabled() {
			return nil, fmt.Errorf("wechat is not configured")
		}
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

func (f *Factory) GetDefaultProvider() (Provider, error) {
	if f.config.MockEnabled {
		return f.GetProvider("mock")
	}

	switch f.config.DeploymentType {
	case config.DeploymentGlobal:
		return f.GetProvider(billing.PaymentProviderLemonSqueezy)
	case config.DeploymentCN:
		return f.GetProvider(billing.PaymentProviderAlipay)
	case config.DeploymentOnPremise:
		return f.GetProvider(billing.PaymentProviderLicense)
	default:
		return nil, fmt.Errorf("unknown deployment type: %s", f.config.DeploymentType)
	}
}

func (f *Factory) GetMockProvider() *mockprovider.Provider {
	return f.mockProvider
}

func (f *Factory) GetLicenseProvider() *licenseprovider.Provider {
	return f.licenseProvider
}

func (f *Factory) IsMockEnabled() bool {
	return f.config.MockEnabled
}

func (f *Factory) GetAvailableProviders() []string {
	return f.config.GetAvailableProviders()
}

func (f *Factory) IsProviderAvailable(providerName string) bool {
	for _, p := range f.GetAvailableProviders() {
		if p == providerName {
			return true
		}
	}
	return false
}

func (f *Factory) GetDeploymentType() config.DeploymentType {
	return f.config.DeploymentType
}

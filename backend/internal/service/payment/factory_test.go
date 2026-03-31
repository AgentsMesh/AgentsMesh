package payment

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentFactory_MockProvider(t *testing.T) {
	appCfg := &config.Config{
		Payment: config.PaymentConfig{
			MockEnabled: true,
			MockBaseURL: "http://localhost:3000",
		},
	}

	factory := NewFactoryFromConfig(appCfg)
	require.NotNil(t, factory)

	assert.True(t, factory.IsMockEnabled())
	assert.NotNil(t, factory.GetMockProvider())

	// GetProvider should always return mock when mock is enabled
	p, err := factory.GetProvider("stripe")
	require.NoError(t, err)
	assert.Equal(t, "mock", p.GetProviderName())

	// GetDefaultProvider should also return mock
	dp, err := factory.GetDefaultProvider()
	require.NoError(t, err)
	assert.Equal(t, "mock", dp.GetProviderName())

	// Available providers list should be ["mock"]
	providers := factory.GetAvailableProviders()
	assert.Equal(t, []string{"mock"}, providers)
}

func TestPaymentFactory_EmptyConfig(t *testing.T) {
	appCfg := &config.Config{
		Payment: config.PaymentConfig{
			DeploymentType: config.DeploymentGlobal,
			// No keys configured
		},
	}

	factory := NewFactoryFromConfig(appCfg)
	require.NotNil(t, factory)

	assert.False(t, factory.IsMockEnabled())
	assert.Nil(t, factory.GetMockProvider())
	assert.Nil(t, factory.GetLicenseProvider())

	// Should fail for unconfigured providers
	_, err := factory.GetProvider(billing.PaymentProviderStripe)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")

	_, err = factory.GetProvider(billing.PaymentProviderAlipay)
	assert.Error(t, err)

	_, err = factory.GetProvider(billing.PaymentProviderWeChat)
	assert.Error(t, err)

	_, err = factory.GetProvider(billing.PaymentProviderLicense)
	assert.Error(t, err)

	_, err = factory.GetProvider("unknown-provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown payment provider")

	// Default provider for global deployment requires LemonSqueezy
	_, err = factory.GetDefaultProvider()
	assert.Error(t, err)

	// No providers available
	assert.Empty(t, factory.GetAvailableProviders())
}

func TestPaymentFactory_DeploymentType(t *testing.T) {
	tests := []struct {
		name     string
		deplType config.DeploymentType
	}{
		{"global", config.DeploymentGlobal},
		{"cn", config.DeploymentCN},
		{"onpremise", config.DeploymentOnPremise},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appCfg := &config.Config{
				Payment: config.PaymentConfig{
					DeploymentType: tt.deplType,
				},
			}
			factory := NewFactoryFromConfig(appCfg)
			assert.Equal(t, tt.deplType, factory.GetDeploymentType())
		})
	}
}

func TestPaymentFactory_IsProviderAvailable(t *testing.T) {
	appCfg := &config.Config{
		Payment: config.PaymentConfig{
			MockEnabled: true,
		},
	}
	factory := NewFactoryFromConfig(appCfg)

	assert.True(t, factory.IsProviderAvailable("mock"))
	assert.False(t, factory.IsProviderAvailable("stripe"))
	assert.False(t, factory.IsProviderAvailable("alipay"))
}

func TestPaymentFactory_MockProviderDerivedBaseURL(t *testing.T) {
	// When MockBaseURL is empty, it should derive from FrontendURL
	appCfg := &config.Config{
		Payment: config.PaymentConfig{
			MockEnabled: true,
			// MockBaseURL intentionally empty
		},
	}

	factory := NewFactoryFromConfig(appCfg)
	require.NotNil(t, factory.GetMockProvider())

	p, err := factory.GetProvider("mock")
	require.NoError(t, err)
	assert.Equal(t, "mock", p.GetProviderName())
}

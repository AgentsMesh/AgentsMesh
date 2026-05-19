package config

type DeploymentType string

const (
	DeploymentGlobal    DeploymentType = "global"    // International - Stripe
	DeploymentCN        DeploymentType = "cn"        // China - Alipay + WeChat Pay
	DeploymentOnPremise DeploymentType = "onpremise" // Self-hosted - License file
)

type PaymentConfig struct {
	DeploymentType DeploymentType
	MockEnabled    bool   // Enable mock payment provider for testing
	MockBaseURL    string // Base URL for mock checkout pages
	Stripe         StripeConfig
	LemonSqueezy   LemonSqueezyConfig
	Alipay         AlipayConfig
	WeChat         WeChatConfig
	License        LicenseConfig
}

type StripeConfig struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
}

type AlipayConfig struct {
	AppID           string
	PrivateKey      string
	AlipayPublicKey string
	IsSandbox       bool
}

type WeChatConfig struct {
	AppID     string
	MchID     string
	APIKey    string
	APIv3Key  string
	CertPath  string
	KeyPath   string
	IsSandbox bool
}

type LicenseConfig struct {
	PublicKeyPath    string // Path to public key for license verification
	LicenseFilePath  string // Path to license file
	LicenseServerURL string // Optional: License server URL for online verification
}

type LemonSqueezyConfig struct {
	APIKey        string // LemonSqueezy API key
	StoreID       string // LemonSqueezy Store ID
	WebhookSecret string // Webhook signing secret for signature verification
}

func (c PaymentConfig) IsGlobal() bool {
	return c.DeploymentType == DeploymentGlobal
}

func (c PaymentConfig) IsCN() bool {
	return c.DeploymentType == DeploymentCN
}

func (c PaymentConfig) IsOnPremise() bool {
	return c.DeploymentType == DeploymentOnPremise
}

func (c PaymentConfig) StripeEnabled() bool {
	return c.IsGlobal() && c.Stripe.SecretKey != ""
}

func (c PaymentConfig) AlipayEnabled() bool {
	return c.IsCN() && c.Alipay.AppID != ""
}

func (c PaymentConfig) WeChatEnabled() bool {
	return c.IsCN() && c.WeChat.AppID != "" && c.WeChat.MchID != ""
}

func (c PaymentConfig) LicenseEnabled() bool {
	return c.IsOnPremise() && c.License.PublicKeyPath != ""
}

func (c PaymentConfig) LemonSqueezyEnabled() bool {
	return c.IsGlobal() && c.LemonSqueezy.APIKey != ""
}

func (c PaymentConfig) LemonSqueezyFullyConfigured() bool {
	return c.LemonSqueezyEnabled() &&
		c.LemonSqueezy.StoreID != "" &&
		c.LemonSqueezy.WebhookSecret != ""
}

func (c PaymentConfig) IsMockEnabled() bool {
	return c.MockEnabled
}

func (c PaymentConfig) GetAvailableProviders() []string {
	if c.MockEnabled {
		return []string{"mock"}
	}

	var providers []string
	if c.LemonSqueezyEnabled() {
		providers = append(providers, "lemonsqueezy")
	}
	if c.StripeEnabled() {
		providers = append(providers, "stripe")
	}
	if c.AlipayEnabled() {
		providers = append(providers, "alipay")
	}
	if c.WeChatEnabled() {
		providers = append(providers, "wechat")
	}
	if c.LicenseEnabled() {
		providers = append(providers, "license")
	}
	return providers
}

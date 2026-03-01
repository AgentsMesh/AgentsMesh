package payment

import (
	"context"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
)

// Re-export types for convenience
type (
	CheckoutRequest       = types.CheckoutRequest
	CheckoutResponse      = types.CheckoutResponse
	WebhookEvent          = types.WebhookEvent
	RefundRequest         = types.RefundRequest
	RefundResponse        = types.RefundResponse
	CustomerPortalRequest = types.CustomerPortalRequest
	CustomerPortalResponse = types.CustomerPortalResponse
	SubscriptionDetails   = types.SubscriptionDetails
	AgreementSignRequest  = types.AgreementSignRequest
	AgreementSignResponse = types.AgreementSignResponse
	AgreementPayRequest   = types.AgreementPayRequest
	AgreementPayResponse  = types.AgreementPayResponse
	LicenseStatus         = types.LicenseStatus
)

// Provider defines the interface for payment providers
type Provider interface {
	// GetProviderName returns the provider name (stripe, alipay, wechat, license)
	GetProviderName() string

	// CreateCheckoutSession creates a new checkout/payment session
	CreateCheckoutSession(ctx context.Context, req *CheckoutRequest) (*CheckoutResponse, error)

	// GetCheckoutStatus checks the status of a checkout session
	GetCheckoutStatus(ctx context.Context, sessionID string) (string, error)

	// HandleWebhook parses and validates a webhook payload
	HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)

	// RefundPayment initiates a refund
	RefundPayment(ctx context.Context, req *RefundRequest) (*RefundResponse, error)

	// CancelSubscription cancels a subscription
	// If immediate is true, cancels immediately; otherwise cancels at period end
	CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error
}

// SubscriptionProvider extends Provider with subscription-specific methods
type SubscriptionProvider interface {
	Provider

	// CreateCustomer creates a customer in the payment provider
	CreateCustomer(ctx context.Context, email string, name string, metadata map[string]string) (string, error)

	// GetCustomerPortalURL returns a URL for the customer to manage their billing
	GetCustomerPortalURL(ctx context.Context, req *CustomerPortalRequest) (*CustomerPortalResponse, error)

	// UpdateSubscriptionSeats updates the seat count for a subscription
	UpdateSubscriptionSeats(ctx context.Context, subscriptionID string, seats int) error

	// GetSubscription retrieves subscription details
	GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionDetails, error)
}

// AgreementProvider extends Provider with auto-debit agreement methods (for Alipay/WeChat)
type AgreementProvider interface {
	Provider

	// CreateAgreementSign creates a signing request for auto-debit agreement
	CreateAgreementSign(ctx context.Context, req *AgreementSignRequest) (*AgreementSignResponse, error)

	// ExecuteAgreementPay executes a payment using the agreement
	ExecuteAgreementPay(ctx context.Context, req *AgreementPayRequest) (*AgreementPayResponse, error)

	// CancelAgreement cancels an auto-debit agreement
	CancelAgreement(ctx context.Context, agreementNo string) error

	// GetAgreementStatus checks the status of an agreement
	GetAgreementStatus(ctx context.Context, agreementNo string) (string, error)
}

// LicenseProvider defines the interface for license verification
type LicenseProvider interface {
	// VerifyLicense verifies a license file/key
	VerifyLicense(ctx context.Context, licenseData []byte) (*billing.License, error)

	// GetLicenseStatus returns the current license status
	GetLicenseStatus(ctx context.Context) (*LicenseStatus, error)

	// ActivateLicense activates a license for an organization
	ActivateLicense(ctx context.Context, licenseKey string, orgID int64) error
}

package payment

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

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

type Provider interface {
	GetProviderName() string

	CreateCheckoutSession(ctx context.Context, req *CheckoutRequest) (*CheckoutResponse, error)

	GetCheckoutStatus(ctx context.Context, sessionID string) (string, error)

	HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)

	RefundPayment(ctx context.Context, req *RefundRequest) (*RefundResponse, error)

	// CancelSubscription cancels a subscription
	// If immediate is true, cancels immediately; otherwise cancels at period end
	CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error
}

type SubscriptionProvider interface {
	Provider

	CreateCustomer(ctx context.Context, email string, name string, metadata map[string]string) (string, error)

	GetCustomerPortalURL(ctx context.Context, req *CustomerPortalRequest) (*CustomerPortalResponse, error)

	UpdateSubscriptionSeats(ctx context.Context, subscriptionID string, seats int) error

	UpdateSubscriptionPlan(ctx context.Context, subscriptionID string, newVariantID string) error

	GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionDetails, error)
}

type AgreementProvider interface {
	Provider

	CreateAgreementSign(ctx context.Context, req *AgreementSignRequest) (*AgreementSignResponse, error)

	ExecuteAgreementPay(ctx context.Context, req *AgreementPayRequest) (*AgreementPayResponse, error)

	CancelAgreement(ctx context.Context, agreementNo string) error

	GetAgreementStatus(ctx context.Context, agreementNo string) (string, error)
}

type LicenseProvider interface {
	VerifyLicense(ctx context.Context, licenseData []byte) (*billing.License, error)

	GetLicenseStatus(ctx context.Context) (*LicenseStatus, error)

	ActivateLicense(ctx context.Context, licenseKey string, orgID int64) error
}

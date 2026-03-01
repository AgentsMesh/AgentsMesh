package types

import (
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// CheckoutRequest represents a request to create a checkout session
type CheckoutRequest struct {
	OrganizationID int64
	UserID         int64
	UserEmail      string

	// Order details
	OrderType    string // subscription, seat_purchase, plan_upgrade, renewal
	PlanID       int64
	BillingCycle string // monthly, yearly
	Seats        int

	// Amount
	Currency     string
	Amount       float64
	ActualAmount float64

	// Options
	SuccessURL     string
	CancelURL      string
	IdempotencyKey string

	// Metadata
	Metadata map[string]string
}

// CheckoutResponse represents a checkout session response
type CheckoutResponse struct {
	// Session info
	SessionID  string
	SessionURL string // URL to redirect user for payment

	// Order info
	OrderNo         string
	ExternalOrderNo string

	// For QR code payments
	QRCodeURL  string
	QRCodeData string
	ExpiresAt  time.Time
}

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
	EventID   string
	EventType string
	Provider  string

	// Order identification
	OrderNo         string
	ExternalOrderNo string

	// Payment info
	Amount       float64
	Currency     string
	Status       string // succeeded, failed, refunded
	FailedReason string

	// Subscription info (for recurring payments)
	SubscriptionID string
	CustomerID     string

	// Raw data
	RawPayload map[string]interface{}
}

// RefundRequest represents a refund request
type RefundRequest struct {
	OrderNo         string
	ExternalOrderNo string
	Amount          float64
	Reason          string
	IdempotencyKey  string
}

// RefundResponse represents a refund response
type RefundResponse struct {
	RefundID string
	Status   string
	Amount   float64
	Currency string
}

// CustomerPortalRequest represents a customer portal request
type CustomerPortalRequest struct {
	CustomerID     string
	SubscriptionID string // Optional: used by LemonSqueezy to get portal URL from subscription
	ReturnURL      string
}

// CustomerPortalResponse represents a customer portal response
type CustomerPortalResponse struct {
	URL string
}

// SubscriptionDetails represents subscription details from the provider
type SubscriptionDetails struct {
	ID                 string
	CustomerID         string
	Status             string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
	Seats              int
	PriceID            string
}

// AgreementSignRequest represents a request to sign an auto-debit agreement
type AgreementSignRequest struct {
	OrganizationID int64
	UserID         int64
	UserEmail      string

	// Agreement details
	PlanName     string
	BillingCycle string
	Amount       float64
	Currency     string

	// URLs
	ReturnURL string
	NotifyURL string
}

// AgreementSignResponse represents a response from agreement signing
type AgreementSignResponse struct {
	SignURL   string // URL or QR code data for user to sign
	RequestNo string // Request number for tracking
	ExpiresAt time.Time
}

// AgreementPayRequest represents a request to execute agreement payment
type AgreementPayRequest struct {
	AgreementNo    string
	OrderNo        string
	Amount         float64
	Currency       string
	Description    string
	IdempotencyKey string
}

// AgreementPayResponse represents a response from agreement payment
type AgreementPayResponse struct {
	TransactionID string
	Status        string
	Amount        float64
	PaidAt        *time.Time
}

// LicenseStatus represents the current license status
type LicenseStatus struct {
	IsValid         bool
	License         *billing.License
	DaysUntilExpiry int
	Message         string
}

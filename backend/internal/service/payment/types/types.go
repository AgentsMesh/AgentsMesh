package types

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

type CheckoutRequest struct {
	OrganizationID int64
	UserID         int64
	UserEmail      string

	OrderType    string // subscription, seat_purchase, plan_upgrade, renewal
	PlanID       int64
	BillingCycle string // monthly, yearly
	Seats        int

	Currency     string
	Amount       float64
	ActualAmount float64

	SuccessURL     string
	CancelURL      string
	IdempotencyKey string

	Metadata map[string]string
}

type CheckoutResponse struct {
	SessionID  string
	SessionURL string // URL to redirect user for payment

	OrderNo         string
	ExternalOrderNo string

	QRCodeURL  string
	QRCodeData string
	ExpiresAt  time.Time
}

type WebhookEvent struct {
	EventID   string
	EventType string
	Provider  string

	OrderNo         string
	ExternalOrderNo string

	Amount       float64
	Currency     string
	Status       string // succeeded, failed, refunded
	FailedReason string

	SubscriptionID string
	CustomerID     string

	Seats     int    // Seat quantity from subscription item
	VariantID string // Provider variant/price ID (used to reverse-lookup plan)

	RawPayload map[string]interface{}
}

type RefundRequest struct {
	OrderNo         string
	ExternalOrderNo string
	Amount          float64
	Reason          string
	IdempotencyKey  string
}

type RefundResponse struct {
	RefundID string
	Status   string
	Amount   float64
	Currency string
}

type CustomerPortalRequest struct {
	CustomerID     string
	SubscriptionID string // Optional: used by LemonSqueezy to get portal URL from subscription
	ReturnURL      string
}

type CustomerPortalResponse struct {
	URL string
}

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

type AgreementSignRequest struct {
	OrganizationID int64
	UserID         int64
	UserEmail      string

	PlanName     string
	BillingCycle string
	Amount       float64
	Currency     string

	ReturnURL string
	NotifyURL string
}

type AgreementSignResponse struct {
	SignURL   string // URL or QR code data for user to sign
	RequestNo string // Request number for tracking
	ExpiresAt time.Time
}

type AgreementPayRequest struct {
	AgreementNo    string
	OrderNo        string
	Amount         float64
	Currency       string
	Description    string
	IdempotencyKey string
}

type AgreementPayResponse struct {
	TransactionID string
	Status        string
	Amount        float64
	PaidAt        *time.Time
}

type LicenseStatus struct {
	IsValid         bool
	License         *billing.License
	DaysUntilExpiry int
	Message         string
}

package promocode

import (
	"context"
	"time"
)

// PlanInfo contains basic plan information needed by promocode service
type PlanInfo struct {
	ID          int64
	Name        string
	DisplayName string
	IsActive    bool
}

// SubscriptionInfo contains subscription information needed by promocode service
type SubscriptionInfo struct {
	PlanID             int64
	PlanName           string
	Status             string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
}

// ApplySubscriptionRequest contains the parameters for applying a promo code subscription
type ApplySubscriptionRequest struct {
	OrganizationID int64
	PlanID         int64
	DurationMonths int
}

// ApplySubscriptionResult contains the result of applying a subscription
type ApplySubscriptionResult struct {
	PreviousPlanName  *string
	PreviousPeriodEnd *time.Time
	NewPeriodEnd      time.Time
}

// BillingProvider defines the interface for billing operations needed by promocode
type BillingProvider interface {
	// GetPlanByName returns plan info by name
	GetPlanByName(ctx context.Context, name string) (*PlanInfo, error)

	// GetActivePlanByName returns active plan info by name
	GetActivePlanByName(ctx context.Context, name string) (*PlanInfo, error)

	// GetSubscription returns the current subscription for an organization
	GetSubscription(ctx context.Context, orgID int64) (*SubscriptionInfo, error)

	// ApplyPromoSubscription applies a promo code subscription within a transaction.
	// The tx parameter is an implementation-specific transaction handle (e.g. *gorm.DB).
	ApplyPromoSubscription(ctx context.Context, tx interface{}, req *ApplySubscriptionRequest) (*ApplySubscriptionResult, error)
}

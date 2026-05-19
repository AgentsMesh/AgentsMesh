package promocode

import (
	"context"
	"time"
)

type PlanInfo struct {
	ID          int64
	Name        string
	DisplayName string
	IsActive    bool
}

type SubscriptionInfo struct {
	PlanID             int64
	PlanName           string
	Status             string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
}

type ApplySubscriptionRequest struct {
	OrganizationID int64
	PlanID         int64
	DurationMonths int
}

type ApplySubscriptionResult struct {
	PreviousPlanName  *string
	PreviousPeriodEnd *time.Time
	NewPeriodEnd      time.Time
}

type BillingProvider interface {
	GetPlanByName(ctx context.Context, name string) (*PlanInfo, error)

	GetActivePlanByName(ctx context.Context, name string) (*PlanInfo, error)

	GetSubscription(ctx context.Context, orgID int64) (*SubscriptionInfo, error)

	ApplyPromoSubscription(ctx context.Context, tx interface{}, req *ApplySubscriptionRequest) (*ApplySubscriptionResult, error)
}

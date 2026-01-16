package promocode

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"gorm.io/gorm"
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

	// ApplyPromoSubscription applies a promo code subscription within a transaction
	ApplyPromoSubscription(ctx context.Context, tx *gorm.DB, req *ApplySubscriptionRequest) (*ApplySubscriptionResult, error)
}

// GormBillingProvider implements BillingProvider using GORM
type GormBillingProvider struct {
	db *gorm.DB
}

// NewGormBillingProvider creates a new GormBillingProvider
func NewGormBillingProvider(db *gorm.DB) *GormBillingProvider {
	return &GormBillingProvider{db: db}
}

// GetPlanByName returns plan info by name
func (p *GormBillingProvider) GetPlanByName(ctx context.Context, name string) (*PlanInfo, error) {
	var plan billing.SubscriptionPlan
	if err := p.db.WithContext(ctx).Where("name = ?", name).First(&plan).Error; err != nil {
		return nil, err
	}
	return &PlanInfo{
		ID:          plan.ID,
		Name:        plan.Name,
		DisplayName: plan.DisplayName,
		IsActive:    plan.IsActive,
	}, nil
}

// GetActivePlanByName returns active plan info by name
func (p *GormBillingProvider) GetActivePlanByName(ctx context.Context, name string) (*PlanInfo, error) {
	var plan billing.SubscriptionPlan
	if err := p.db.WithContext(ctx).Where("name = ? AND is_active = ?", name, true).First(&plan).Error; err != nil {
		return nil, err
	}
	return &PlanInfo{
		ID:          plan.ID,
		Name:        plan.Name,
		DisplayName: plan.DisplayName,
		IsActive:    plan.IsActive,
	}, nil
}

// GetSubscription returns the current subscription for an organization
func (p *GormBillingProvider) GetSubscription(ctx context.Context, orgID int64) (*SubscriptionInfo, error) {
	var sub billing.Subscription
	if err := p.db.WithContext(ctx).Where("organization_id = ?", orgID).First(&sub).Error; err != nil {
		return nil, err
	}

	var planName string
	var plan billing.SubscriptionPlan
	if err := p.db.WithContext(ctx).First(&plan, sub.PlanID).Error; err == nil {
		planName = plan.Name
	}

	return &SubscriptionInfo{
		PlanID:             sub.PlanID,
		PlanName:           planName,
		Status:             string(sub.Status),
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
	}, nil
}

// ApplyPromoSubscription applies a promo code subscription within a transaction
func (p *GormBillingProvider) ApplyPromoSubscription(ctx context.Context, tx *gorm.DB, req *ApplySubscriptionRequest) (*ApplySubscriptionResult, error) {
	var currentSub billing.Subscription
	hasSubscription := tx.Where("organization_id = ?", req.OrganizationID).First(&currentSub).Error == nil

	var previousPlanName *string
	var previousPeriodEnd *time.Time

	if hasSubscription {
		var currentPlan billing.SubscriptionPlan
		if err := tx.First(&currentPlan, currentSub.PlanID).Error; err == nil {
			previousPlanName = &currentPlan.Name
		}
		previousPeriodEnd = &currentSub.CurrentPeriodEnd
	}

	now := time.Now()
	var newPeriodEnd time.Time

	if hasSubscription && currentSub.CurrentPeriodEnd.After(now) {
		newPeriodEnd = currentSub.CurrentPeriodEnd.AddDate(0, req.DurationMonths, 0)
	} else {
		newPeriodEnd = now.AddDate(0, req.DurationMonths, 0)
	}

	if hasSubscription {
		currentSub.PlanID = req.PlanID
		currentSub.Status = billing.SubscriptionStatusActive
		if !currentSub.CurrentPeriodEnd.After(now) {
			currentSub.CurrentPeriodStart = now
		}
		currentSub.CurrentPeriodEnd = newPeriodEnd
		if err := tx.Save(&currentSub).Error; err != nil {
			return nil, err
		}
	} else {
		newSub := &billing.Subscription{
			OrganizationID:     req.OrganizationID,
			PlanID:             req.PlanID,
			Status:             billing.SubscriptionStatusActive,
			BillingCycle:       billing.BillingCycleMonthly,
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   newPeriodEnd,
		}
		if err := tx.Create(newSub).Error; err != nil {
			return nil, err
		}
	}

	return &ApplySubscriptionResult{
		PreviousPlanName:  previousPlanName,
		PreviousPeriodEnd: previousPeriodEnd,
		NewPeriodEnd:      newPeriodEnd,
	}, nil
}

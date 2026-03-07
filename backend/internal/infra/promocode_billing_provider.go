package infra

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/promocode"
	"gorm.io/gorm"
)

// GormBillingProvider implements promocode.BillingProvider using GORM
type GormBillingProvider struct {
	db *gorm.DB
}

// NewGormBillingProvider creates a new GormBillingProvider
func NewGormBillingProvider(db *gorm.DB) *GormBillingProvider {
	return &GormBillingProvider{db: db}
}

// GetPlanByName returns plan info by name
func (p *GormBillingProvider) GetPlanByName(ctx context.Context, name string) (*promocode.PlanInfo, error) {
	var plan billing.SubscriptionPlan
	if err := p.db.WithContext(ctx).Where("name = ?", name).First(&plan).Error; err != nil {
		return nil, err
	}
	return &promocode.PlanInfo{
		ID:          plan.ID,
		Name:        plan.Name,
		DisplayName: plan.DisplayName,
		IsActive:    plan.IsActive,
	}, nil
}

// GetActivePlanByName returns active plan info by name
func (p *GormBillingProvider) GetActivePlanByName(ctx context.Context, name string) (*promocode.PlanInfo, error) {
	var plan billing.SubscriptionPlan
	if err := p.db.WithContext(ctx).Where("name = ? AND is_active = ?", name, true).First(&plan).Error; err != nil {
		return nil, err
	}
	return &promocode.PlanInfo{
		ID:          plan.ID,
		Name:        plan.Name,
		DisplayName: plan.DisplayName,
		IsActive:    plan.IsActive,
	}, nil
}

// GetSubscription returns the current subscription for an organization
func (p *GormBillingProvider) GetSubscription(ctx context.Context, orgID int64) (*promocode.SubscriptionInfo, error) {
	var sub billing.Subscription
	if err := p.db.WithContext(ctx).Where("organization_id = ?", orgID).First(&sub).Error; err != nil {
		return nil, err
	}

	var planName string
	var plan billing.SubscriptionPlan
	if err := p.db.WithContext(ctx).First(&plan, sub.PlanID).Error; err == nil {
		planName = plan.Name
	}

	return &promocode.SubscriptionInfo{
		PlanID:             sub.PlanID,
		PlanName:           planName,
		Status:             string(sub.Status),
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
	}, nil
}

// ApplyPromoSubscription applies a promo code subscription within a transaction.
// The tx parameter must be a *gorm.DB transaction handle.
func (p *GormBillingProvider) ApplyPromoSubscription(ctx context.Context, tx interface{}, req *promocode.ApplySubscriptionRequest) (*promocode.ApplySubscriptionResult, error) {
	gormTx, ok := tx.(*gorm.DB)
	if !ok {
		return nil, errors.New("unsupported transaction type")
	}

	var currentSub billing.Subscription
	hasSubscription := gormTx.Where("organization_id = ?", req.OrganizationID).First(&currentSub).Error == nil

	var previousPlanName *string
	var previousPeriodEnd *time.Time

	if hasSubscription {
		var currentPlan billing.SubscriptionPlan
		if err := gormTx.First(&currentPlan, currentSub.PlanID).Error; err == nil {
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
		if err := gormTx.Save(&currentSub).Error; err != nil {
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
		if err := gormTx.Create(newSub).Error; err != nil {
			return nil, err
		}
	}

	return &promocode.ApplySubscriptionResult{
		PreviousPlanName:  previousPlanName,
		PreviousPeriodEnd: previousPeriodEnd,
		NewPeriodEnd:      newPeriodEnd,
	}, nil
}

// Compile-time interface compliance check
var _ promocode.BillingProvider = (*GormBillingProvider)(nil)

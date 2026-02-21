package billing

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// GetSubscription returns subscription for an organization
func (s *Service) GetSubscription(ctx context.Context, orgID int64) (*billing.Subscription, error) {
	var sub billing.Subscription
	if err := s.db.WithContext(ctx).Preload("Plan").Where("organization_id = ?", orgID).First(&sub).Error; err != nil {
		return nil, ErrSubscriptionNotFound
	}
	return &sub, nil
}

// CreateSubscription creates a new subscription
func (s *Service) CreateSubscription(ctx context.Context, orgID int64, planName string) (*billing.Subscription, error) {
	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0) // 1 month

	sub := &billing.Subscription{
		OrganizationID:     orgID,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
	}

	if err := s.db.WithContext(ctx).Create(sub).Error; err != nil {
		return nil, err
	}

	sub.Plan = plan
	return sub, nil
}

// CreateTrialSubscription creates a trial subscription for a new organization
func (s *Service) CreateTrialSubscription(ctx context.Context, orgID int64, planName string, trialDays int) (*billing.Subscription, error) {
	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	if trialDays <= 0 {
		trialDays = billing.DefaultTrialDays
	}

	now := time.Now()
	periodEnd := now.AddDate(0, 0, trialDays)

	sub := &billing.Subscription{
		OrganizationID:     orgID,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusTrialing,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		SeatCount:          1,
	}

	if err := s.db.WithContext(ctx).Create(sub).Error; err != nil {
		return nil, err
	}

	sub.Plan = plan
	return sub, nil
}

// AdminUpdatePlan directly changes the subscription plan without payment checks or downgrade delays.
// This is intended for admin operations where the admin has full authority.
func (s *Service) AdminUpdatePlan(ctx context.Context, orgID int64, planName string) (*billing.Subscription, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	newPlan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	// Direct update: no payment checks, no downgrade delay
	if err := s.db.WithContext(ctx).Model(sub).Updates(map[string]interface{}{
		"plan_id":           newPlan.ID,
		"downgrade_to_plan": nil, // Clear any pending downgrade
	}).Error; err != nil {
		return nil, err
	}

	// Sync organization table redundant fields
	s.db.WithContext(ctx).Table("organizations").
		Where("id = ?", orgID).
		Update("subscription_plan", newPlan.Name)

	sub.PlanID = newPlan.ID
	sub.DowngradeToPlan = nil
	sub.Plan = newPlan
	return sub, nil
}

// AdminRenew extends a subscription by the specified number of months.
// Starts from current_period_end (or now, whichever is later). Reactivates canceled/frozen subscriptions.
func (s *Service) AdminRenew(ctx context.Context, orgID int64, months int) (*billing.Subscription, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Start from current_period_end or now, whichever is later
	start := sub.CurrentPeriodEnd
	now := time.Now()
	if now.After(start) {
		start = now
	}
	end := start.AddDate(0, months, 0)

	updates := map[string]interface{}{
		"status":               billing.SubscriptionStatusActive,
		"current_period_start": start,
		"current_period_end":   end,
		"frozen_at":            nil,
		"canceled_at":          nil,
		"cancel_at_period_end": false,
	}

	if err := s.db.WithContext(ctx).Model(sub).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Sync organization table
	s.db.WithContext(ctx).Table("organizations").
		Where("id = ?", orgID).
		Update("subscription_status", billing.SubscriptionStatusActive)

	// Reload to get fresh data
	return s.GetSubscription(ctx, orgID)
}

// AdminCancelSubscription cancels a subscription without calling external payment APIs (Stripe, etc.).
func (s *Service) AdminCancelSubscription(ctx context.Context, orgID int64) error {
	now := time.Now()
	err := s.db.WithContext(ctx).Model(&billing.Subscription{}).
		Where("organization_id = ?", orgID).
		Updates(map[string]interface{}{
			"status":      billing.SubscriptionStatusCanceled,
			"canceled_at": now,
		}).Error
	if err != nil {
		return err
	}

	// Sync organization table
	s.db.WithContext(ctx).Table("organizations").
		Where("id = ?", orgID).
		Update("subscription_status", billing.SubscriptionStatusCanceled)

	return nil
}

// UpdateSubscription updates subscription plan (handles upgrade/downgrade)
func (s *Service) UpdateSubscription(ctx context.Context, orgID int64, planName string) (*billing.Subscription, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	newPlan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	// Get current plan for comparison
	currentPlan, err := s.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	// Determine if this is an upgrade or downgrade based on price
	isDowngrade := newPlan.PricePerSeatMonthly < currentPlan.PricePerSeatMonthly

	if isDowngrade {
		// Downgrade: schedule for end of billing period
		// Check if current seat count exceeds new plan's max_users
		if newPlan.MaxUsers > 0 && sub.SeatCount > newPlan.MaxUsers {
			return nil, ErrSeatCountExceedsLimit
		}

		// Set downgrade to take effect at period end
		sub.DowngradeToPlan = &planName
		if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
			return nil, err
		}

		// Return current subscription with downgrade scheduled
		sub.Plan = currentPlan
		return sub, nil
	}

	// Upgrade: immediate effect (payment should be handled separately via checkout)
	// For free plan or if no payment required, apply immediately
	if currentPlan.PricePerSeatMonthly == 0 || newPlan.PricePerSeatMonthly == 0 {
		sub.PlanID = newPlan.ID
		sub.DowngradeToPlan = nil // Clear any pending downgrade

		// Update Stripe subscription if enabled
		if s.stripeEnabled && sub.StripeSubscriptionID != nil {
			// In a real implementation, update Stripe subscription here
		}

		if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
			return nil, err
		}

		sub.Plan = newPlan
		return sub, nil
	}

	// For paid upgrades, just return the subscription - payment flow handles the actual upgrade
	sub.Plan = currentPlan
	return sub, nil
}

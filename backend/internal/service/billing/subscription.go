package billing

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// GetSubscription returns subscription for an organization
func (s *Service) GetSubscription(ctx context.Context, orgID int64) (*billing.Subscription, error) {
	sub, err := s.repo.GetSubscriptionByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
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

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	sub.Plan = plan
	return sub, nil
}

// CreateTrialSubscription creates a trial subscription for a new organization.
// NOTE: This uses the service's own repo. If the org was created in a
// transaction that hasn't committed yet, use CreateTrialSubscriptionTx instead.
func (s *Service) CreateTrialSubscription(ctx context.Context, orgID int64, planName string, trialDays int) (*billing.Subscription, error) {
	return s.createTrialSubscription(ctx, s.repo, orgID, planName, trialDays)
}

// CreateTrialSubscriptionTx creates a trial subscription using the provided transaction handle.
// This ensures the subscription insert can see the org record created in the same transaction.
// The rawTx parameter must be the underlying DB transaction type (e.g. *gorm.DB).
func (s *Service) CreateTrialSubscriptionTx(ctx context.Context, rawTx interface{}, orgID int64, planName string, trialDays int) (*billing.Subscription, error) {
	txRepo := s.repo.Scoped(rawTx)
	return s.createTrialSubscription(ctx, txRepo, orgID, planName, trialDays)
}

func (s *Service) createTrialSubscription(ctx context.Context, repo billing.BillingRepository, orgID int64, planName string, trialDays int) (*billing.Subscription, error) {
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

	if err := repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	sub.Plan = plan
	return sub, nil
}

// AdminCreateSubscription creates a new active subscription for an organization that doesn't have one.
func (s *Service) AdminCreateSubscription(ctx context.Context, orgID int64, planName string, months int) (*billing.Subscription, error) {
	_, err := s.GetSubscription(ctx, orgID)
	if err == nil {
		return nil, ErrSubscriptionAlreadyExists
	}

	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	if months <= 0 {
		months = 1
	}

	now := time.Now()
	periodEnd := now.AddDate(0, months, 0)

	sub := &billing.Subscription{
		OrganizationID:     orgID,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		SeatCount:          1,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	// Sync organization table redundant fields
	s.syncOrganizationSubscription(ctx, orgID, &plan.Name, strPtr(billing.SubscriptionStatusActive))

	sub.Plan = plan
	return sub, nil
}

// AdminUpdatePlan directly changes the subscription plan without payment checks or downgrade delays.
func (s *Service) AdminUpdatePlan(ctx context.Context, orgID int64, planName string) (*billing.Subscription, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	newPlan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	log.Printf("[AdminUpdatePlan] orgID=%d, planName=%q, sub.PlanID=%d, newPlan.ID=%d, newPlan.Name=%q",
		orgID, planName, sub.PlanID, newPlan.ID, newPlan.Name)

	if err := s.repo.UpdateSubscriptionFields(ctx, sub.ID, map[string]interface{}{
		"plan_id":           newPlan.ID,
		"downgrade_to_plan": nil,
	}); err != nil {
		return nil, err
	}

	// Sync organization table redundant fields
	s.syncOrganizationSubscription(ctx, orgID, &newPlan.Name, nil)

	sub.PlanID = newPlan.ID
	sub.DowngradeToPlan = nil
	sub.Plan = newPlan
	return sub, nil
}

// AdminRenew extends a subscription by the specified number of months.
func (s *Service) AdminRenew(ctx context.Context, orgID int64, months int) (*billing.Subscription, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	start := sub.CurrentPeriodEnd
	now := time.Now()
	if now.After(start) {
		start = now
	}
	end := start.AddDate(0, months, 0)

	if err := s.repo.UpdateSubscriptionFields(ctx, sub.ID, map[string]interface{}{
		"status":               billing.SubscriptionStatusActive,
		"current_period_start": start,
		"current_period_end":   end,
		"frozen_at":            nil,
		"canceled_at":          nil,
		"cancel_at_period_end": false,
	}); err != nil {
		return nil, err
	}

	// Sync organization table
	s.syncOrganizationSubscription(ctx, orgID, nil, strPtr(billing.SubscriptionStatusActive))

	// Reload to get fresh data
	return s.GetSubscription(ctx, orgID)
}

// AdminCancelSubscription cancels a subscription without calling external payment APIs.
func (s *Service) AdminCancelSubscription(ctx context.Context, orgID int64) error {
	now := time.Now()

	if err := s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"status":      billing.SubscriptionStatusCanceled,
		"canceled_at": now,
	}); err != nil {
		return err
	}

	// Sync organization table
	s.syncOrganizationSubscription(ctx, orgID, nil, strPtr(billing.SubscriptionStatusCanceled))
	return nil
}

// UpgradePlan upgrades the subscription plan via the payment provider API.
func (s *Service) UpgradePlan(ctx context.Context, orgID int64, planName string) (*billing.Subscription, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	if !sub.IsActive() && !sub.IsTrialing() {
		if sub.IsFrozen() {
			return nil, ErrSubscriptionFrozen
		}
		return nil, ErrSubscriptionNotActive
	}

	newPlan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	currentPlan := sub.Plan
	if currentPlan == nil {
		currentPlan, err = s.GetPlanByID(ctx, sub.PlanID)
		if err != nil {
			return nil, err
		}
	}

	if newPlan.PricePerSeatMonthly <= currentPlan.PricePerSeatMonthly {
		return nil, fmt.Errorf("can only upgrade to a higher plan; use downgrade for lower plans")
	}

	currency := billing.CurrencyUSD
	if s.paymentConfig != nil && s.paymentConfig.DeploymentType == "cn" {
		currency = billing.CurrencyCNY
	}
	price, err := s.GetPlanPriceByID(ctx, newPlan.ID, currency)
	if err != nil {
		return nil, fmt.Errorf("price not found for target plan: %w", err)
	}

	var variantID string
	if sub.BillingCycle == billing.BillingCycleYearly {
		if price.LemonSqueezyVariantIDYearly != nil {
			variantID = *price.LemonSqueezyVariantIDYearly
		}
	} else {
		if price.LemonSqueezyVariantIDMonthly != nil {
			variantID = *price.LemonSqueezyVariantIDMonthly
		}
	}

	if sub.LemonSqueezySubscriptionID != nil && *sub.LemonSqueezySubscriptionID != "" {
		if variantID == "" {
			return nil, fmt.Errorf("no variant ID configured for plan %q with billing cycle %q and currency %q", planName, sub.BillingCycle, currency)
		}
		provider, err := s.getSubscriptionProvider()
		if err != nil {
			return nil, fmt.Errorf("payment provider not available: %w", err)
		}
		if err := provider.UpdateSubscriptionPlan(ctx, *sub.LemonSqueezySubscriptionID, variantID); err != nil {
			return nil, fmt.Errorf("failed to update plan with provider: %w", err)
		}
	}

	if err := s.repo.UpdateSubscriptionFields(ctx, sub.ID, map[string]interface{}{
		"plan_id":           newPlan.ID,
		"downgrade_to_plan": nil,
	}); err != nil {
		log.Printf("[WARN] UpgradePlan: provider API succeeded but DB update failed for org=%d, plan=%s: %v", orgID, planName, err)
		return nil, fmt.Errorf("failed to sync plan locally: %w", err)
	}

	// Sync organization table redundant fields
	s.syncOrganizationSubscription(ctx, orgID, &newPlan.Name, nil)

	sub.PlanID = newPlan.ID
	sub.DowngradeToPlan = nil
	sub.Plan = newPlan
	return sub, nil
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

	currentPlan, err := s.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	isDowngrade := newPlan.PricePerSeatMonthly < currentPlan.PricePerSeatMonthly

	if isDowngrade {
		if newPlan.MaxUsers > 0 && sub.SeatCount > newPlan.MaxUsers {
			return nil, ErrSeatCountExceedsLimit
		}

		sub.DowngradeToPlan = &planName
		if err := s.repo.SaveSubscription(ctx, sub); err != nil {
			return nil, err
		}

		sub.Plan = currentPlan
		return sub, nil
	}

	if currentPlan.PricePerSeatMonthly == 0 || newPlan.PricePerSeatMonthly == 0 {
		sub.PlanID = newPlan.ID
		sub.DowngradeToPlan = nil

		if err := s.repo.SaveSubscription(ctx, sub); err != nil {
			return nil, err
		}

		s.syncOrganizationSubscription(ctx, orgID, &newPlan.Name, nil)

		sub.Plan = newPlan
		return sub, nil
	}

	sub.Plan = currentPlan
	return sub, nil
}

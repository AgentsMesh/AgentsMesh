package billing

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

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
		slog.Warn("provider API succeeded but DB update failed for plan upgrade",
			"org_id", orgID, "plan", planName, "error", err)
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

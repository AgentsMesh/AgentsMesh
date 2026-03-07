package billing

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ActivateTrialSubscription converts a trial subscription to active
func (s *Service) ActivateTrialSubscription(ctx context.Context, orgID int64, billingCycle string) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return err
	}

	if sub.Status != billing.SubscriptionStatusTrialing {
		return nil // Already active or other status
	}

	now := time.Now()
	var periodEnd time.Time
	if billingCycle == billing.BillingCycleYearly {
		periodEnd = now.AddDate(1, 0, 0)
	} else {
		billingCycle = billing.BillingCycleMonthly
		periodEnd = now.AddDate(0, 1, 0)
	}

	return s.repo.UpdateSubscriptionFields(ctx, sub.ID, map[string]interface{}{
		"status":               billing.SubscriptionStatusActive,
		"billing_cycle":        billingCycle,
		"current_period_start": now,
		"current_period_end":   periodEnd,
	})
}

// FreezeSubscription freezes a subscription due to non-payment
func (s *Service) FreezeSubscription(ctx context.Context, orgID int64) error {
	now := time.Now()
	return s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"status":    billing.SubscriptionStatusFrozen,
		"frozen_at": now,
	})
}

// UnfreezeSubscription reactivates a frozen subscription
func (s *Service) UnfreezeSubscription(ctx context.Context, orgID int64, billingCycle string) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return err
	}

	now := time.Now()
	var periodEnd time.Time
	if billingCycle == billing.BillingCycleYearly {
		periodEnd = now.AddDate(1, 0, 0)
	} else {
		billingCycle = billing.BillingCycleMonthly
		periodEnd = now.AddDate(0, 1, 0)
	}

	return s.repo.UpdateSubscriptionFields(ctx, sub.ID, map[string]interface{}{
		"status":               billing.SubscriptionStatusActive,
		"billing_cycle":        billingCycle,
		"current_period_start": now,
		"current_period_end":   periodEnd,
		"frozen_at":            nil,
	})
}

// CancelSubscription cancels a subscription
func (s *Service) CancelSubscription(ctx context.Context, orgID int64) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return err
	}

	now := time.Now()
	sub.Status = billing.SubscriptionStatusCanceled
	sub.CanceledAt = &now

	// Cancel Stripe subscription if enabled
	if s.stripeEnabled && s.stripeClient != nil && sub.StripeSubscriptionID != nil {
		_, err := s.stripeClient.CancelSubscription(*sub.StripeSubscriptionID, nil)
		if err != nil {
			return err
		}
	}

	return s.repo.SaveSubscription(ctx, sub)
}

// SetCancelAtPeriodEnd sets or clears the cancel_at_period_end flag
func (s *Service) SetCancelAtPeriodEnd(ctx context.Context, orgID int64, cancel bool) error {
	return s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"cancel_at_period_end": cancel,
	})
}

// SetNextBillingCycle sets the next billing cycle (takes effect on renewal)
func (s *Service) SetNextBillingCycle(ctx context.Context, orgID int64, cycle string) error {
	return s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"next_billing_cycle": cycle,
	})
}

// RenewSubscription renews subscription for next period
func (s *Service) RenewSubscription(ctx context.Context, orgID int64) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return err
	}

	// Set new period
	sub.CurrentPeriodStart = sub.CurrentPeriodEnd
	if sub.BillingCycle == billing.BillingCycleYearly {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(1, 0, 0)
	} else {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(0, 1, 0)
	}

	return s.repo.SaveSubscription(ctx, sub)
}

// SetAutoRenew sets the auto_renew flag for a subscription
func (s *Service) SetAutoRenew(ctx context.Context, orgID int64, autoRenew bool) error {
	return s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"auto_renew": autoRenew,
	})
}

package billing

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ActivateTrialSubscription converts a trial subscription to active
func (s *Service) ActivateTrialSubscription(ctx context.Context, orgID int64, billingCycle string) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		slog.Error("failed to get subscription for trial activation", "org_id", orgID, "error", err)
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

	if err := s.repo.UpdateSubscriptionFields(ctx, sub.ID, map[string]interface{}{
		"status":               billing.SubscriptionStatusActive,
		"billing_cycle":        billingCycle,
		"current_period_start": now,
		"current_period_end":   periodEnd,
	}); err != nil {
		slog.Error("failed to activate trial subscription", "org_id", orgID, "error", err)
		return err
	}
	slog.Info("trial subscription activated", "org_id", orgID, "billing_cycle", billingCycle)
	return nil
}

// FreezeSubscription freezes a subscription due to non-payment
func (s *Service) FreezeSubscription(ctx context.Context, orgID int64) error {
	now := time.Now()
	if err := s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"status":    billing.SubscriptionStatusFrozen,
		"frozen_at": now,
	}); err != nil {
		slog.Error("failed to freeze subscription", "org_id", orgID, "error", err)
		return err
	}
	slog.Warn("subscription frozen due to non-payment", "org_id", orgID)
	return nil
}

// UnfreezeSubscription reactivates a frozen subscription
func (s *Service) UnfreezeSubscription(ctx context.Context, orgID int64, billingCycle string) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		slog.Error("failed to get subscription for unfreeze", "org_id", orgID, "error", err)
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

	if err := s.repo.UpdateSubscriptionFields(ctx, sub.ID, map[string]interface{}{
		"status":               billing.SubscriptionStatusActive,
		"billing_cycle":        billingCycle,
		"current_period_start": now,
		"current_period_end":   periodEnd,
		"frozen_at":            nil,
	}); err != nil {
		slog.Error("failed to unfreeze subscription", "org_id", orgID, "error", err)
		return err
	}
	slog.Info("subscription unfrozen", "org_id", orgID, "billing_cycle", billingCycle)
	return nil
}

// CancelSubscription cancels a subscription
func (s *Service) CancelSubscription(ctx context.Context, orgID int64) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		slog.Error("failed to get subscription for cancellation", "org_id", orgID, "error", err)
		return err
	}

	now := time.Now()
	sub.Status = billing.SubscriptionStatusCanceled
	sub.CanceledAt = &now

	// Cancel Stripe subscription if enabled
	if s.stripeEnabled && s.stripeClient != nil && sub.StripeSubscriptionID != nil {
		_, err := s.stripeClient.CancelSubscription(*sub.StripeSubscriptionID, nil)
		if err != nil {
			slog.Error("failed to cancel stripe subscription", "org_id", orgID, "stripe_subscription_id", *sub.StripeSubscriptionID, "error", err)
			return err
		}
	}

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.Error("failed to save canceled subscription", "org_id", orgID, "error", err)
		return err
	}
	slog.Info("subscription canceled", "org_id", orgID)
	return nil
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
		slog.Error("failed to get subscription for renewal", "org_id", orgID, "error", err)
		return err
	}

	// Set new period
	sub.CurrentPeriodStart = sub.CurrentPeriodEnd
	if sub.BillingCycle == billing.BillingCycleYearly {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(1, 0, 0)
	} else {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(0, 1, 0)
	}

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.Error("failed to renew subscription", "org_id", orgID, "error", err)
		return err
	}
	slog.Info("subscription renewed", "org_id", orgID, "billing_cycle", sub.BillingCycle, "period_end", sub.CurrentPeriodEnd)
	return nil
}

// SetAutoRenew sets the auto_renew flag for a subscription
func (s *Service) SetAutoRenew(ctx context.Context, orgID int64, autoRenew bool) error {
	return s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"auto_renew": autoRenew,
	})
}

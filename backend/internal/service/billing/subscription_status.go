package billing

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

func (s *Service) ActivateTrialSubscription(ctx context.Context, orgID int64, billingCycle string) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get subscription for trial activation", "org_id", orgID, "error", err)
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
		slog.ErrorContext(ctx, "failed to activate trial subscription", "org_id", orgID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "trial subscription activated", "org_id", orgID, "billing_cycle", billingCycle)
	return nil
}

func (s *Service) FreezeSubscription(ctx context.Context, orgID int64) error {
	now := time.Now()
	if err := s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"status":    billing.SubscriptionStatusFrozen,
		"frozen_at": now,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to freeze subscription", "org_id", orgID, "error", err)
		return err
	}
	slog.WarnContext(ctx, "subscription frozen due to non-payment", "org_id", orgID)
	return nil
}

func (s *Service) UnfreezeSubscription(ctx context.Context, orgID int64, billingCycle string) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get subscription for unfreeze", "org_id", orgID, "error", err)
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
		slog.ErrorContext(ctx, "failed to unfreeze subscription", "org_id", orgID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "subscription unfrozen", "org_id", orgID, "billing_cycle", billingCycle)
	return nil
}

func (s *Service) CancelSubscription(ctx context.Context, orgID int64) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get subscription for cancellation", "org_id", orgID, "error", err)
		return err
	}

	now := time.Now()
	sub.Status = billing.SubscriptionStatusCanceled
	sub.CanceledAt = &now

	if s.stripeEnabled && s.stripeClient != nil && sub.StripeSubscriptionID != nil {
		_, err := s.stripeClient.CancelSubscription(*sub.StripeSubscriptionID, nil)
		if err != nil {
			slog.ErrorContext(ctx, "failed to cancel stripe subscription", "org_id", orgID, "stripe_subscription_id", *sub.StripeSubscriptionID, "error", err)
			return err
		}
	}

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.ErrorContext(ctx, "failed to save canceled subscription", "org_id", orgID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "subscription canceled", "org_id", orgID)
	return nil
}

func (s *Service) SetCancelAtPeriodEnd(ctx context.Context, orgID int64, cancel bool) error {
	return s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"cancel_at_period_end": cancel,
	})
}

func (s *Service) SetNextBillingCycle(ctx context.Context, orgID int64, cycle string) error {
	return s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"next_billing_cycle": cycle,
	})
}

func (s *Service) RenewSubscription(ctx context.Context, orgID int64) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get subscription for renewal", "org_id", orgID, "error", err)
		return err
	}

	sub.CurrentPeriodStart = sub.CurrentPeriodEnd
	if sub.BillingCycle == billing.BillingCycleYearly {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(1, 0, 0)
	} else {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(0, 1, 0)
	}

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.ErrorContext(ctx, "failed to renew subscription", "org_id", orgID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "subscription renewed", "org_id", orgID, "billing_cycle", sub.BillingCycle, "period_end", sub.CurrentPeriodEnd)
	return nil
}

func (s *Service) SetAutoRenew(ctx context.Context, orgID int64, autoRenew bool) error {
	return s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"auto_renew": autoRenew,
	})
}

package billing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

func (s *Service) handleRecurringPaymentSuccess(ctx context.Context, event *payment.WebhookEvent) error {
	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found, ignore
	}

	var downgradedPlanName *string
	if sub.DowngradeToPlan != nil {
		plan, err := s.GetPlan(ctx, *sub.DowngradeToPlan)
		if err == nil {
			sub.PlanID = plan.ID
			downgradedPlanName = &plan.Name
		} else {
			slog.WarnContext(ctx, "pending downgrade plan not found on recurring payment, downgrade dropped",
				"plan", *sub.DowngradeToPlan, "org_id", sub.OrganizationID, "error", err)
		}
		sub.DowngradeToPlan = nil
	}
	if sub.NextBillingCycle != nil {
		sub.BillingCycle = *sub.NextBillingCycle
		sub.NextBillingCycle = nil
	}

	if sub.CurrentPeriodEnd.IsZero() {
		sub.CurrentPeriodStart = time.Now()
	} else {
		sub.CurrentPeriodStart = sub.CurrentPeriodEnd
	}
	if sub.BillingCycle == billing.BillingCycleYearly {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(1, 0, 0)
	} else {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(0, 1, 0)
	}

	sub.Status = billing.SubscriptionStatusActive
	sub.FrozenAt = nil

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		return err
	}

	status := billing.SubscriptionStatusActive
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, downgradedPlanName, &status)
	return nil
}

func (s *Service) handleRecurringPaymentFailure(ctx context.Context, event *payment.WebhookEvent) error {
	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found, ignore
	}

	now := time.Now()
	sub.Status = billing.SubscriptionStatusFrozen
	sub.FrozenAt = &now

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		return err
	}

	status := billing.SubscriptionStatusFrozen
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	return nil
}

func (s *Service) activateSubscription(ctx context.Context, order *billing.PaymentOrder, event *payment.WebhookEvent) error {
	if order.PlanID == nil {
		return fmt.Errorf("activateSubscription: order %s has nil PlanID, cannot activate subscription", order.OrderNo)
	}

	seats := order.Seats
	if seats <= 0 {
		seats = 1
	}

	var planName string
	if order.Plan != nil {
		planName = order.Plan.Name
	} else if order.PlanID != nil {
		if p, err := s.GetPlanByID(ctx, *order.PlanID); err == nil {
			planName = p.Name
		}
	}

	sub, err := s.GetSubscription(ctx, order.OrganizationID)
	if err != nil {
		now := time.Now()
		var periodEnd time.Time
		if order.BillingCycle == billing.BillingCycleYearly {
			periodEnd = now.AddDate(1, 0, 0)
		} else {
			periodEnd = now.AddDate(0, 1, 0)
		}

		provider := order.PaymentProvider
		sub = &billing.Subscription{
			OrganizationID:     order.OrganizationID,
			PlanID:             *order.PlanID,
			Status:             billing.SubscriptionStatusActive,
			BillingCycle:       order.BillingCycle,
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   periodEnd,
			PaymentProvider:    &provider,
			PaymentMethod:      order.PaymentMethod,
			AutoRenew:          true,
			SeatCount:          seats,
		}

		setProviderIDs(sub, event)

		if err := s.repo.CreateSubscription(ctx, sub); err != nil {
			return err
		}

		status := billing.SubscriptionStatusActive
		s.syncOrganizationSubscription(ctx, order.OrganizationID, strPtr(planName), &status)
		return nil
	}

	now := time.Now()
	var periodEnd time.Time
	if order.BillingCycle == billing.BillingCycleYearly {
		periodEnd = now.AddDate(1, 0, 0)
	} else {
		periodEnd = now.AddDate(0, 1, 0)
	}

	sub.PlanID = *order.PlanID
	sub.Status = billing.SubscriptionStatusActive
	sub.BillingCycle = order.BillingCycle
	sub.CurrentPeriodStart = now
	sub.CurrentPeriodEnd = periodEnd
	sub.SeatCount = seats
	sub.FrozenAt = nil
	sub.CanceledAt = nil
	sub.CancelAtPeriodEnd = false
	sub.DowngradeToPlan = nil
	sub.NextBillingCycle = nil
	provider := order.PaymentProvider
	sub.PaymentProvider = &provider
	sub.PaymentMethod = order.PaymentMethod

	setProviderIDs(sub, event)

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		return err
	}

	status := billing.SubscriptionStatusActive
	s.syncOrganizationSubscription(ctx, order.OrganizationID, strPtr(planName), &status)
	return nil
}

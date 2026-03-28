package billing

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Webhook Internal Helper Methods
// ===========================================

// syncOrganizationSubscription syncs the redundant subscription fields on the organizations table.
// The organizations table maintains subscription_plan and subscription_status for fast access.
// This method MUST be called whenever subscription status or plan changes to keep them in sync.
func (s *Service) syncOrganizationSubscription(ctx context.Context, orgID int64, planName *string, status *string) {
	updates := map[string]interface{}{}
	if planName != nil {
		updates["subscription_plan"] = *planName
	}
	if status != nil {
		updates["subscription_status"] = *status
	}
	if len(updates) == 0 {
		return
	}
	if err := s.repo.SyncOrganizationSubscription(ctx, orgID, updates); err != nil {
		slog.Error("failed to sync organization subscription fields",
			"org_id", orgID, "updates", updates, "error", err)
	}
}

// findSubscriptionByProviderID finds a subscription by provider-specific ID
func (s *Service) findSubscriptionByProviderID(ctx context.Context, provider string, subscriptionID string) (*billing.Subscription, error) {
	sub, err := s.repo.FindSubscriptionByProviderID(ctx, provider, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}


// setProviderIDs sets provider-specific IDs on a subscription
func setProviderIDs(sub *billing.Subscription, event *payment.WebhookEvent) {
	switch event.Provider {
	case billing.PaymentProviderLemonSqueezy:
		if event.CustomerID != "" {
			sub.LemonSqueezyCustomerID = &event.CustomerID
		}
		if event.SubscriptionID != "" {
			sub.LemonSqueezySubscriptionID = &event.SubscriptionID
		}
	default:
		// Default to Stripe for backward compatibility
		if event.CustomerID != "" {
			sub.StripeCustomerID = &event.CustomerID
		}
		if event.SubscriptionID != "" {
			sub.StripeSubscriptionID = &event.SubscriptionID
		}
	}
}

// findPlanByVariantID looks up a plan by LemonSqueezy variant ID from plan_prices table
func (s *Service) findPlanByVariantID(ctx context.Context, variantID string) (*billing.SubscriptionPlan, error) {
	plan, err := s.repo.FindPlanByVariantID(ctx, variantID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}
	return plan, nil
}

// addSeats adds seats to a subscription after payment
func (s *Service) addSeats(ctx context.Context, order *billing.PaymentOrder) error {
	// Validate against plan max_users limit
	sub, err := s.GetSubscription(ctx, order.OrganizationID)
	if err == nil && sub.Plan != nil && sub.Plan.MaxUsers > 0 {
		if sub.SeatCount+order.Seats > sub.Plan.MaxUsers {
			slog.Warn("seat count would exceed plan max_users limit",
			"current_seats", sub.SeatCount, "additional_seats", order.Seats,
			"max_users", sub.Plan.MaxUsers, "org_id", order.OrganizationID)
			return ErrQuotaExceeded
		}
	}

	return s.repo.AddSeats(ctx, order.OrganizationID, order.Seats)
}

// upgradePlan upgrades a subscription to a new plan
func (s *Service) upgradePlan(ctx context.Context, order *billing.PaymentOrder) error {
	if order.PlanID == nil {
		return ErrInvalidPlan
	}
	if err := s.repo.UpdateSubscriptionFieldsByOrg(ctx, order.OrganizationID, map[string]interface{}{
		"plan_id":           *order.PlanID,
		"downgrade_to_plan": nil,
		"updated_at":        time.Now(),
	}); err != nil {
		return err
	}

	// Sync organization table with new plan name
	var planName string
	if order.Plan != nil {
		planName = order.Plan.Name
	} else {
		if p, err := s.GetPlanByID(ctx, *order.PlanID); err == nil {
			planName = p.Name
		}
	}
	if planName != "" {
		s.syncOrganizationSubscription(ctx, order.OrganizationID, &planName, nil)
	}
	return nil
}

// renewSubscriptionFromOrder renews a subscription from an order
func (s *Service) renewSubscriptionFromOrder(ctx context.Context, order *billing.PaymentOrder) error {
	sub, err := s.GetSubscription(ctx, order.OrganizationID)
	if err != nil {
		return err
	}

	// Apply pending changes BEFORE period calculation (consistent with handleRecurringPaymentSuccess)
	var downgradedPlanName *string
	if sub.DowngradeToPlan != nil {
		plan, err := s.GetPlan(ctx, *sub.DowngradeToPlan)
		if err == nil {
			sub.PlanID = plan.ID
			downgradedPlanName = &plan.Name
		} else {
			slog.Warn("pending downgrade plan not found, downgrade dropped",
				"plan", *sub.DowngradeToPlan, "org_id", sub.OrganizationID, "error", err)
		}
		sub.DowngradeToPlan = nil
	}
	if sub.NextBillingCycle != nil {
		sub.BillingCycle = *sub.NextBillingCycle
		sub.NextBillingCycle = nil
	}

	// Set new period using the (possibly updated) billing cycle
	// Guard against zero-value CurrentPeriodEnd to avoid starting from epoch
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

	// Sync organization table
	status := billing.SubscriptionStatusActive
	s.syncOrganizationSubscription(ctx, order.OrganizationID, downgradedPlanName, &status)
	return nil
}

// strPtr returns a pointer to the string, or nil if empty
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

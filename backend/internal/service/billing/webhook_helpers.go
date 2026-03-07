package billing

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

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
	if err := s.db.WithContext(ctx).Table("organizations").
		Where("id = ?", orgID).
		Updates(updates).Error; err != nil {
		log.Printf("[ERROR] syncOrganizationSubscription: failed to sync organization subscription fields for org_id=%d (updates=%v): %v", orgID, updates, err)
	}
}

// findSubscriptionByProviderID finds a subscription by provider-specific ID
func (s *Service) findSubscriptionByProviderID(ctx context.Context, provider string, subscriptionID string) (*billing.Subscription, error) {
	var sub billing.Subscription
	var err error

	switch provider {
	case billing.PaymentProviderLemonSqueezy:
		err = s.db.WithContext(ctx).Where("lemonsqueezy_subscription_id = ?", subscriptionID).First(&sub).Error
	default:
		// Default to Stripe for backward compatibility
		err = s.db.WithContext(ctx).Where("stripe_subscription_id = ?", subscriptionID).First(&sub).Error
	}

	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// handleRecurringPaymentSuccess handles successful recurring payments
func (s *Service) handleRecurringPaymentSuccess(ctx context.Context, event *payment.WebhookEvent) error {
	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found, ignore
	}

	// Apply pending changes BEFORE period calculation so the new period uses the updated cycle/plan
	var downgradedPlanName *string
	if sub.DowngradeToPlan != nil {
		plan, err := s.GetPlan(ctx, *sub.DowngradeToPlan)
		if err == nil {
			sub.PlanID = plan.ID
			downgradedPlanName = &plan.Name
		} else {
			log.Printf("[WARN] handleRecurringPaymentSuccess: pending downgrade to plan %q not found for org=%d, downgrade dropped: %v",
				*sub.DowngradeToPlan, sub.OrganizationID, err)
		}
		sub.DowngradeToPlan = nil
	}
	if sub.NextBillingCycle != nil {
		sub.BillingCycle = *sub.NextBillingCycle
		sub.NextBillingCycle = nil
	}

	// Renew the subscription period using the (possibly updated) billing cycle
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

	// Unfreeze if was frozen
	sub.Status = billing.SubscriptionStatusActive
	sub.FrozenAt = nil

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
		return err
	}

	// Sync organization table: status always changes to active; plan may have changed via downgrade
	status := billing.SubscriptionStatusActive
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, downgradedPlanName, &status)
	return nil
}

// handleRecurringPaymentFailure handles failed recurring payments
func (s *Service) handleRecurringPaymentFailure(ctx context.Context, event *payment.WebhookEvent) error {
	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found, ignore
	}

	// Freeze the subscription
	now := time.Now()
	sub.Status = billing.SubscriptionStatusFrozen
	sub.FrozenAt = &now

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
		return err
	}

	// Sync organization table
	status := billing.SubscriptionStatusFrozen
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	return nil
}

// activateSubscription activates a new subscription after payment
func (s *Service) activateSubscription(ctx context.Context, order *billing.PaymentOrder, event *payment.WebhookEvent) error {
	// PlanID is required to activate a subscription
	if order.PlanID == nil {
		return fmt.Errorf("activateSubscription: order %s has nil PlanID, cannot activate subscription", order.OrderNo)
	}

	// Ensure at least 1 seat
	seats := order.Seats
	if seats <= 0 {
		seats = 1
	}

	// Resolve plan name for org sync
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
		// Create new subscription
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

		// Set provider-specific IDs based on provider
		setProviderIDs(sub, event)

		if err := s.db.WithContext(ctx).Create(sub).Error; err != nil {
			return err
		}

		// Sync organization table
		status := billing.SubscriptionStatusActive
		s.syncOrganizationSubscription(ctx, order.OrganizationID, strPtr(planName), &status)
		return nil
	}

	// Update existing subscription
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

	// Set provider-specific IDs based on provider
	setProviderIDs(sub, event)

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
		return err
	}

	// Sync organization table
	status := billing.SubscriptionStatusActive
	s.syncOrganizationSubscription(ctx, order.OrganizationID, strPtr(planName), &status)
	return nil
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
	var price billing.PlanPrice
	err := s.db.WithContext(ctx).
		Preload("Plan").
		Where("lemonsqueezy_variant_id_monthly = ? OR lemonsqueezy_variant_id_yearly = ?", variantID, variantID).
		First(&price).Error
	if err != nil {
		return nil, err
	}
	return price.Plan, nil
}

// addSeats adds seats to a subscription after payment
func (s *Service) addSeats(ctx context.Context, order *billing.PaymentOrder) error {
	// Validate against plan max_users limit
	sub, err := s.GetSubscription(ctx, order.OrganizationID)
	if err == nil && sub.Plan != nil && sub.Plan.MaxUsers > 0 {
		if sub.SeatCount+order.Seats > sub.Plan.MaxUsers {
			log.Printf("[WARN] addSeats: seat count %d + %d would exceed plan max_users %d for org=%d",
				sub.SeatCount, order.Seats, sub.Plan.MaxUsers, order.OrganizationID)
			return ErrQuotaExceeded
		}
	}

	return s.db.WithContext(ctx).Model(&billing.Subscription{}).
		Where("organization_id = ?", order.OrganizationID).
		Update("seat_count", gorm.Expr("seat_count + ?", order.Seats)).Error
}

// upgradePlan upgrades a subscription to a new plan
func (s *Service) upgradePlan(ctx context.Context, order *billing.PaymentOrder) error {
	if order.PlanID == nil {
		return ErrInvalidPlan
	}
	if err := s.db.WithContext(ctx).Model(&billing.Subscription{}).
		Where("organization_id = ?", order.OrganizationID).
		Updates(map[string]interface{}{
			"plan_id":           *order.PlanID,
			"downgrade_to_plan": nil,
			"updated_at":        time.Now(),
		}).Error; err != nil {
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
			log.Printf("[WARN] renewSubscriptionFromOrder: pending downgrade to plan %q not found for org=%d, downgrade dropped: %v",
				*sub.DowngradeToPlan, sub.OrganizationID, err)
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

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
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

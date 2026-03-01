package billing

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Webhook Internal Helper Methods
// ===========================================

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

	// Renew the subscription period
	sub.CurrentPeriodStart = sub.CurrentPeriodEnd
	if sub.BillingCycle == billing.BillingCycleYearly {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(1, 0, 0)
	} else {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(0, 1, 0)
	}

	// Unfreeze if was frozen
	sub.Status = billing.SubscriptionStatusActive
	sub.FrozenAt = nil

	// Apply pending changes if any
	if sub.DowngradeToPlan != nil {
		plan, err := s.GetPlan(ctx, *sub.DowngradeToPlan)
		if err == nil {
			sub.PlanID = plan.ID
		}
		sub.DowngradeToPlan = nil
	}
	if sub.NextBillingCycle != nil {
		sub.BillingCycle = *sub.NextBillingCycle
		sub.NextBillingCycle = nil
	}

	return s.db.WithContext(ctx).Save(sub).Error
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

	return s.db.WithContext(ctx).Save(sub).Error
}

// activateSubscription activates a new subscription after payment
func (s *Service) activateSubscription(ctx context.Context, order *billing.PaymentOrder, event *payment.WebhookEvent) error {
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
			SeatCount:          order.Seats,
		}

		// Set provider-specific IDs based on provider
		setProviderIDs(sub, event)

		return s.db.WithContext(ctx).Create(sub).Error
	}

	// Update existing subscription
	sub.PlanID = *order.PlanID
	sub.Status = billing.SubscriptionStatusActive
	sub.BillingCycle = order.BillingCycle
	sub.SeatCount = order.Seats
	sub.FrozenAt = nil
	provider := order.PaymentProvider
	sub.PaymentProvider = &provider
	sub.PaymentMethod = order.PaymentMethod

	// Set provider-specific IDs based on provider
	setProviderIDs(sub, event)

	return s.db.WithContext(ctx).Save(sub).Error
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

// addSeats adds seats to a subscription after payment
func (s *Service) addSeats(ctx context.Context, order *billing.PaymentOrder) error {
	return s.db.WithContext(ctx).Model(&billing.Subscription{}).
		Where("organization_id = ?", order.OrganizationID).
		Update("seat_count", gorm.Expr("seat_count + ?", order.Seats)).Error
}

// upgradePlan upgrades a subscription to a new plan
func (s *Service) upgradePlan(ctx context.Context, order *billing.PaymentOrder) error {
	if order.PlanID == nil {
		return ErrInvalidPlan
	}
	return s.db.WithContext(ctx).Model(&billing.Subscription{}).
		Where("organization_id = ?", order.OrganizationID).
		Updates(map[string]interface{}{
			"plan_id":    *order.PlanID,
			"updated_at": time.Now(),
		}).Error
}

// renewSubscriptionFromOrder renews a subscription from an order
func (s *Service) renewSubscriptionFromOrder(ctx context.Context, order *billing.PaymentOrder) error {
	sub, err := s.GetSubscription(ctx, order.OrganizationID)
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
	sub.Status = billing.SubscriptionStatusActive
	sub.FrozenAt = nil

	return s.db.WithContext(ctx).Save(sub).Error
}

package billing

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// LemonSqueezy Subscription Webhook Handlers
// ===========================================

// HandleSubscriptionCreated handles subscription creation webhook event (mainly for LemonSqueezy)
func (s *Service) HandleSubscriptionCreated(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	if event.SubscriptionID == "" {
		return nil
	}

	// Idempotency check
	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			return nil
		}
		return err
	}
	// Roll back the idempotency mark if the handler fails, so the event
	// can be retried on the next delivery.
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	// Find subscription by organization (the order_created event should have already created it)
	// We need to update it with the LemonSqueezy subscription ID
	if event.Provider == billing.PaymentProviderLemonSqueezy {
		var sub billing.Subscription
		var err error

		// Try to find by customer ID first (set during order_created)
		if event.CustomerID != "" {
			err = s.db.WithContext(ctx).Where("lemonsqueezy_customer_id = ?", event.CustomerID).First(&sub).Error
		}

		// Fallback: try to find by order_no if customer_id lookup failed
		// The order_no is passed in custom_data and stored in payment_orders
		if err != nil && event.OrderNo != "" {
			var order billing.PaymentOrder
			if orderErr := s.db.WithContext(ctx).Where("order_no = ?", event.OrderNo).First(&order).Error; orderErr == nil {
				err = s.db.WithContext(ctx).Where("organization_id = ?", order.OrganizationID).First(&sub).Error
			}
		}

		// Update subscription with LemonSqueezy IDs if found and not already set
		if err == nil && sub.LemonSqueezySubscriptionID == nil {
			sub.LemonSqueezySubscriptionID = &event.SubscriptionID
			// Also set customer_id if not already set
			if sub.LemonSqueezyCustomerID == nil && event.CustomerID != "" {
				sub.LemonSqueezyCustomerID = &event.CustomerID
			}
			return s.db.WithContext(ctx).Save(&sub).Error
		}
	}

	return nil
}

// HandleSubscriptionPaused handles subscription pause webhook event
func (s *Service) HandleSubscriptionPaused(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	if event.SubscriptionID == "" {
		return nil
	}

	// Idempotency check
	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			return nil
		}
		return err
	}
	// Roll back the idempotency mark if the handler fails, so the event
	// can be retried on the next delivery.
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found
	}

	// Pause the subscription
	// NOTE: Paused is user-initiated, different from Frozen (payment failure).
	// Do NOT set FrozenAt here — FrozenAt is reserved for payment failure freezes.
	sub.Status = billing.SubscriptionStatusPaused

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
		return err
	}

	// Sync organization table
	status := billing.SubscriptionStatusPaused
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	return nil
}

// HandleSubscriptionResumed handles subscription resume webhook event
func (s *Service) HandleSubscriptionResumed(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	if event.SubscriptionID == "" {
		return nil
	}

	// Idempotency check
	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			return nil
		}
		return err
	}
	// Roll back the idempotency mark if the handler fails, so the event
	// can be retried on the next delivery.
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found
	}

	// Resume the subscription
	sub.Status = billing.SubscriptionStatusActive
	sub.FrozenAt = nil

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
		return err
	}

	// Sync organization table
	status := billing.SubscriptionStatusActive
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	return nil
}

// HandleSubscriptionExpired handles subscription expiration webhook event
func (s *Service) HandleSubscriptionExpired(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	if event.SubscriptionID == "" {
		return nil
	}

	// Idempotency check
	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			return nil
		}
		return err
	}
	// Roll back the idempotency mark if the handler fails, so the event
	// can be retried on the next delivery.
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found
	}

	// Mark subscription as expired
	// NOTE: Expired is distinct from Canceled. We do NOT set CanceledAt here
	// because this is a natural expiration, not a user-initiated cancellation.
	sub.Status = billing.SubscriptionStatusExpired

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
		return err
	}

	// Sync organization table
	status := billing.SubscriptionStatusExpired
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	return nil
}

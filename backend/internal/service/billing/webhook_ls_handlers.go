package billing

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// LemonSqueezy Subscription Webhook Handlers
// ===========================================

// HandleSubscriptionCreated handles subscription creation webhook event (mainly for LemonSqueezy)
func (s *Service) HandleSubscriptionCreated(c *gin.Context, event *payment.WebhookEvent) error {
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
func (s *Service) HandleSubscriptionPaused(c *gin.Context, event *payment.WebhookEvent) error {
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

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found
	}

	// Pause the subscription
	now := time.Now()
	sub.Status = billing.SubscriptionStatusPaused
	sub.FrozenAt = &now

	return s.db.WithContext(ctx).Save(&sub).Error
}

// HandleSubscriptionResumed handles subscription resume webhook event
func (s *Service) HandleSubscriptionResumed(c *gin.Context, event *payment.WebhookEvent) error {
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

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found
	}

	// Resume the subscription
	sub.Status = billing.SubscriptionStatusActive
	sub.FrozenAt = nil

	return s.db.WithContext(ctx).Save(&sub).Error
}

// HandleSubscriptionExpired handles subscription expiration webhook event
func (s *Service) HandleSubscriptionExpired(c *gin.Context, event *payment.WebhookEvent) error {
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

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found
	}

	// Mark subscription as expired
	now := time.Now()
	sub.Status = billing.SubscriptionStatusExpired
	sub.CanceledAt = &now

	return s.db.WithContext(ctx).Save(&sub).Error
}

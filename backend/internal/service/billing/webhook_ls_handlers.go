package billing

import (
	"errors"
	"log/slog"

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
			slog.Info("webhook already processed, skipping", "event_id", event.EventID, "provider", event.Provider)
			return nil
		}
		slog.Error("failed to check webhook idempotency", "event_id", event.EventID, "provider", event.Provider, "error", err)
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
		var sub *billing.Subscription
		var err error

		// Try to find by customer ID first (set during order_created)
		if event.CustomerID != "" {
			sub, err = s.repo.FindSubscriptionByLSCustomerID(ctx, event.CustomerID)
		}

		// Fallback: try to find by order_no if customer_id lookup failed
		// The order_no is passed in custom_data and stored in payment_orders
		if (err != nil || sub == nil) && event.OrderNo != "" {
			order, orderErr := s.repo.GetPaymentOrderByNo(ctx, event.OrderNo)
			if orderErr == nil && order != nil {
				sub, err = s.repo.GetSubscriptionByOrgID(ctx, order.OrganizationID)
			}
		}

		// Update subscription with LemonSqueezy IDs if found and not already set
		if err == nil && sub != nil && sub.LemonSqueezySubscriptionID == nil {
			sub.LemonSqueezySubscriptionID = &event.SubscriptionID
			// Also set customer_id if not already set
			if sub.LemonSqueezyCustomerID == nil && event.CustomerID != "" {
				sub.LemonSqueezyCustomerID = &event.CustomerID
			}
			if err := s.repo.SaveSubscription(ctx, sub); err != nil {
				slog.Error("failed to save subscription with LS IDs", "org_id", sub.OrganizationID, "subscription_id", event.SubscriptionID, "error", err)
				return err
			}
			slog.Info("subscription linked to LemonSqueezy", "org_id", sub.OrganizationID, "ls_subscription_id", event.SubscriptionID)
			return nil
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
			slog.Info("webhook already processed, skipping", "event_id", event.EventID, "provider", event.Provider)
			return nil
		}
		slog.Error("failed to check webhook idempotency", "event_id", event.EventID, "provider", event.Provider, "error", err)
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
		slog.Warn("subscription not found for pause webhook", "provider", event.Provider, "subscription_id", event.SubscriptionID)
		return nil // Subscription not found
	}

	// Pause the subscription
	// NOTE: Paused is user-initiated, different from Frozen (payment failure).
	// Do NOT set FrozenAt here — FrozenAt is reserved for payment failure freezes.
	sub.Status = billing.SubscriptionStatusPaused

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.Error("failed to save paused subscription", "org_id", sub.OrganizationID, "error", err)
		return err
	}

	// Sync organization table
	status := billing.SubscriptionStatusPaused
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	slog.Info("subscription paused via webhook", "org_id", sub.OrganizationID, "provider", event.Provider)
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
			slog.Info("webhook already processed, skipping", "event_id", event.EventID, "provider", event.Provider)
			return nil
		}
		slog.Error("failed to check webhook idempotency", "event_id", event.EventID, "provider", event.Provider, "error", err)
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
		slog.Warn("subscription not found for resume webhook", "provider", event.Provider, "subscription_id", event.SubscriptionID)
		return nil // Subscription not found
	}

	// Resume the subscription
	sub.Status = billing.SubscriptionStatusActive
	sub.FrozenAt = nil

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.Error("failed to save resumed subscription", "org_id", sub.OrganizationID, "error", err)
		return err
	}

	// Sync organization table
	status := billing.SubscriptionStatusActive
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	slog.Info("subscription resumed via webhook", "org_id", sub.OrganizationID, "provider", event.Provider)
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
			slog.Info("webhook already processed, skipping", "event_id", event.EventID, "provider", event.Provider)
			return nil
		}
		slog.Error("failed to check webhook idempotency", "event_id", event.EventID, "provider", event.Provider, "error", err)
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
		slog.Warn("subscription not found for expiration webhook", "provider", event.Provider, "subscription_id", event.SubscriptionID)
		return nil // Subscription not found
	}

	// Mark subscription as expired
	// NOTE: Expired is distinct from Canceled. We do NOT set CanceledAt here
	// because this is a natural expiration, not a user-initiated cancellation.
	sub.Status = billing.SubscriptionStatusExpired

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.Error("failed to save expired subscription", "org_id", sub.OrganizationID, "error", err)
		return err
	}

	// Sync organization table
	status := billing.SubscriptionStatusExpired
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	slog.Info("subscription expired via webhook", "org_id", sub.OrganizationID, "provider", event.Provider)
	return nil
}

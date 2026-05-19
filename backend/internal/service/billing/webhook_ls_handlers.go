package billing

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

func (s *Service) HandleSubscriptionCreated(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	if event.SubscriptionID == "" {
		return nil
	}

	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			slog.InfoContext(c.Request.Context(), "webhook already processed, skipping", "event_id", event.EventID, "provider", event.Provider)
			return nil
		}
		slog.ErrorContext(c.Request.Context(), "failed to check webhook idempotency", "event_id", event.EventID, "provider", event.Provider, "error", err)
		return err
	}
	// Roll back idempotency mark on handler failure so the event can be retried on next delivery.
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	if event.Provider == billing.PaymentProviderLemonSqueezy {
		var sub *billing.Subscription
		var err error

		if event.CustomerID != "" {
			sub, err = s.repo.FindSubscriptionByLSCustomerID(ctx, event.CustomerID)
		}

		if (err != nil || sub == nil) && event.OrderNo != "" {
			order, orderErr := s.repo.GetPaymentOrderByNo(ctx, event.OrderNo)
			if orderErr == nil && order != nil {
				sub, err = s.repo.GetSubscriptionByOrgID(ctx, order.OrganizationID)
			}
		}

		if err == nil && sub != nil && sub.LemonSqueezySubscriptionID == nil {
			sub.LemonSqueezySubscriptionID = &event.SubscriptionID
			if sub.LemonSqueezyCustomerID == nil && event.CustomerID != "" {
				sub.LemonSqueezyCustomerID = &event.CustomerID
			}
			if err := s.repo.SaveSubscription(ctx, sub); err != nil {
				slog.ErrorContext(c.Request.Context(), "failed to save subscription with LS IDs", "org_id", sub.OrganizationID, "subscription_id", event.SubscriptionID, "error", err)
				return err
			}
			slog.InfoContext(c.Request.Context(), "subscription linked to LemonSqueezy", "org_id", sub.OrganizationID, "ls_subscription_id", event.SubscriptionID)
			return nil
		}
	}

	return nil
}

func (s *Service) HandleSubscriptionPaused(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	if event.SubscriptionID == "" {
		return nil
	}

	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			slog.InfoContext(c.Request.Context(), "webhook already processed, skipping", "event_id", event.EventID, "provider", event.Provider)
			return nil
		}
		slog.ErrorContext(c.Request.Context(), "failed to check webhook idempotency", "event_id", event.EventID, "provider", event.Provider, "error", err)
		return err
	}
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "subscription not found for pause webhook", "provider", event.Provider, "subscription_id", event.SubscriptionID)
		return nil
	}

	sub.Status = billing.SubscriptionStatusPaused

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to save paused subscription", "org_id", sub.OrganizationID, "error", err)
		return err
	}

	status := billing.SubscriptionStatusPaused
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	slog.InfoContext(c.Request.Context(), "subscription paused via webhook", "org_id", sub.OrganizationID, "provider", event.Provider)
	return nil
}

func (s *Service) HandleSubscriptionResumed(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	if event.SubscriptionID == "" {
		return nil
	}

	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			slog.InfoContext(c.Request.Context(), "webhook already processed, skipping", "event_id", event.EventID, "provider", event.Provider)
			return nil
		}
		slog.ErrorContext(c.Request.Context(), "failed to check webhook idempotency", "event_id", event.EventID, "provider", event.Provider, "error", err)
		return err
	}
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "subscription not found for resume webhook", "provider", event.Provider, "subscription_id", event.SubscriptionID)
		return nil
	}

	sub.Status = billing.SubscriptionStatusActive
	sub.FrozenAt = nil

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to save resumed subscription", "org_id", sub.OrganizationID, "error", err)
		return err
	}

	status := billing.SubscriptionStatusActive
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	slog.InfoContext(c.Request.Context(), "subscription resumed via webhook", "org_id", sub.OrganizationID, "provider", event.Provider)
	return nil
}

func (s *Service) HandleSubscriptionExpired(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	if event.SubscriptionID == "" {
		return nil
	}

	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			slog.InfoContext(c.Request.Context(), "webhook already processed, skipping", "event_id", event.EventID, "provider", event.Provider)
			return nil
		}
		slog.ErrorContext(c.Request.Context(), "failed to check webhook idempotency", "event_id", event.EventID, "provider", event.Provider, "error", err)
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
		slog.WarnContext(c.Request.Context(), "subscription not found for expiration webhook", "provider", event.Provider, "subscription_id", event.SubscriptionID)
		return nil
	}

	// Expired ≠ Canceled — natural expiration, do NOT set CanceledAt.
	sub.Status = billing.SubscriptionStatusExpired

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to save expired subscription", "org_id", sub.OrganizationID, "error", err)
		return err
	}

	status := billing.SubscriptionStatusExpired
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	slog.InfoContext(c.Request.Context(), "subscription expired via webhook", "org_id", sub.OrganizationID, "provider", event.Provider)
	return nil
}

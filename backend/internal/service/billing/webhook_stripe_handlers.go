package billing

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Stripe Subscription Webhook Handlers
// ===========================================

// HandleSubscriptionUpdated handles subscription update webhook event
func (s *Service) HandleSubscriptionUpdated(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
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

	// Find subscription by provider subscription ID
	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		// P0 #2: Race condition fallback — if subscription_created webhook was processed
		// before order_created, the LemonSqueezySubscriptionID may not have been set.
		// Try to find by customer_id and set the subscription ID as a safety net.
		if event.Provider == billing.PaymentProviderLemonSqueezy && event.CustomerID != "" {
			sub, err = s.findAndLinkLSSubscription(ctx, event)
			if err != nil {
				slog.Warn("subscription not found for provider",
				"provider", event.Provider, "subscription_id", event.SubscriptionID, "customer_id", event.CustomerID)
				return nil
			}
		} else {
			return nil // Subscription not found
		}
	}

	// Map status to our status using provider-specific mapping
	if event.Provider == billing.PaymentProviderLemonSqueezy {
		mappedStatus := billing.MapLSStatusToInternal(event.Status)
		switch mappedStatus {
		case billing.SubscriptionStatusActive, billing.SubscriptionStatusTrialing,
			billing.SubscriptionStatusPaused, billing.SubscriptionStatusPastDue,
			billing.SubscriptionStatusFrozen, billing.SubscriptionStatusCanceled,
			billing.SubscriptionStatusExpired:
			sub.Status = mappedStatus
		default:
			slog.Warn("unknown LemonSqueezy subscription status",
				"status", event.Status, "subscription_id", event.SubscriptionID)
		}
	} else {
		// Generic status mapping for Stripe and others
		switch event.Status {
		case "active":
			sub.Status = billing.SubscriptionStatusActive
			sub.FrozenAt = nil
		case "past_due":
			sub.Status = billing.SubscriptionStatusPastDue
		case "canceled", "cancelled":
			sub.Status = billing.SubscriptionStatusCanceled
		case "trialing":
			sub.Status = billing.SubscriptionStatusTrialing
		case "paused":
			sub.Status = billing.SubscriptionStatusPaused
		case "expired":
			sub.Status = billing.SubscriptionStatusExpired
		default:
			slog.Warn("unknown subscription status from provider",
				"status", event.Status, "provider", event.Provider, "subscription_id", event.SubscriptionID)
		}
	}

	// Clear frozen timestamp if reactivated
	if sub.Status == billing.SubscriptionStatusActive {
		sub.FrozenAt = nil
	}

	// Set frozen timestamp when transitioning to frozen state (e.g., LS "unpaid" → frozen)
	if sub.Status == billing.SubscriptionStatusFrozen && sub.FrozenAt == nil {
		now := time.Now()
		sub.FrozenAt = &now
	}

	// Sync seat count from provider
	if event.Seats > 0 && event.Seats != sub.SeatCount {
		sub.SeatCount = event.Seats
	}

	// Sync plan via variant_id reverse lookup (LemonSqueezy only)
	var planName *string
	if event.VariantID != "" {
		if plan, err := s.findPlanByVariantID(ctx, event.VariantID); err == nil && plan != nil && plan.ID != sub.PlanID {
			sub.PlanID = plan.ID
			sub.DowngradeToPlan = nil // Clear pending downgrade since plan changed externally
			planName = &plan.Name
		}
	}

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		return err
	}

	// Sync organization table: always sync status, and plan if it changed
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, planName, &sub.Status)
	return nil
}

// findAndLinkLSSubscription finds a subscription by LemonSqueezy customer_id
// and links the subscription_id. This serves as a fallback when the normal
// subscription_created → subscription_updated ordering is disrupted by race conditions.
func (s *Service) findAndLinkLSSubscription(ctx context.Context, event *payment.WebhookEvent) (*billing.Subscription, error) {
	sub, err := s.repo.FindSubscriptionByLSCustomerID(ctx, event.CustomerID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}

	// Link the subscription_id if not set
	if sub.LemonSqueezySubscriptionID == nil {
		sub.LemonSqueezySubscriptionID = &event.SubscriptionID
		slog.Info("linked LS subscription via customer ID (race condition recovery)",
			"subscription_id", event.SubscriptionID, "org_id", sub.OrganizationID, "customer_id", event.CustomerID)
	}

	return sub, nil
}

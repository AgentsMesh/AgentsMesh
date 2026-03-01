package billing

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Stripe Subscription Webhook Handlers
// ===========================================

// HandleSubscriptionUpdated handles subscription update webhook event
func (s *Service) HandleSubscriptionUpdated(c *gin.Context, event *payment.WebhookEvent) error {
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

	// Find subscription by provider subscription ID
	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found
	}

	// Map status to our status using provider-specific mapping
	if event.Provider == billing.PaymentProviderLemonSqueezy {
		sub.Status = billing.MapLSStatusToInternal(event.Status)
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
		}
	}

	// Clear frozen timestamp if reactivated
	if sub.Status == billing.SubscriptionStatusActive {
		sub.FrozenAt = nil
	}

	return s.db.WithContext(ctx).Save(&sub).Error
}

package webhooks

import (
	"io"
	"net/http"

	billingdomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Payment Webhook Handlers - LemonSqueezy
// ===========================================

// handleLemonSqueezyWebhook handles LemonSqueezy webhook events
func (r *WebhookRouter) handleLemonSqueezyWebhook(c *gin.Context) {
	// Check if LemonSqueezy is configured
	if r.paymentFactory == nil || !r.paymentFactory.IsProviderAvailable(billingdomain.PaymentProviderLemonSqueezy) {
		r.logger.Warn("LemonSqueezy webhook received but LemonSqueezy is not configured")
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "LemonSqueezy not configured")
		return
	}

	// Read the request body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		r.logger.Error("failed to read LemonSqueezy webhook body", "error", err)
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "failed to read request body")
		return
	}

	// Get the signature header (X-Signature)
	signature := c.GetHeader("X-Signature")
	if signature == "" {
		r.logger.Warn("missing X-Signature header for LemonSqueezy webhook")
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "missing signature")
		return
	}

	// Get the LemonSqueezy provider
	provider, err := r.paymentFactory.GetProvider(billingdomain.PaymentProviderLemonSqueezy)
	if err != nil {
		r.logger.Error("failed to get LemonSqueezy provider", "error", err)
		apierr.InternalError(c, "provider not available")
		return
	}

	// Parse and validate the webhook
	event, err := provider.HandleWebhook(c.Request.Context(), payload, signature)
	if err != nil {
		r.logger.Error("failed to validate LemonSqueezy webhook", "error", err)
		apierr.InvalidInput(c, "invalid webhook signature")
		return
	}

	r.logger.Info("received LemonSqueezy webhook",
		"event_id", event.EventID,
		"event_type", event.EventType,
		"order_no", event.OrderNo,
		"subscription_id", event.SubscriptionID,
	)

	// Process the event based on type
	processErr := r.processLemonSqueezyEvent(c, event)

	if processErr != nil {
		r.logger.Error("failed to process LemonSqueezy webhook",
			"error", processErr,
			"event_type", event.EventType,
			"event_id", event.EventID,
		)
		apierr.InternalError(c, "failed to process event")
		return
	}

	// Acknowledge receipt
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// processLemonSqueezyEvent processes LemonSqueezy webhook events
func (r *WebhookRouter) processLemonSqueezyEvent(c *gin.Context, event *payment.WebhookEvent) error {
	switch event.EventType {
	case billingdomain.WebhookEventLSOrderCreated:
		event.Status = billingdomain.OrderStatusSucceeded
		return r.billingSvc.HandlePaymentSucceeded(c, event)

	case billingdomain.WebhookEventLSSubscriptionCreated:
		r.logger.Info("LemonSqueezy subscription created",
			"subscription_id", event.SubscriptionID,
			"customer_id", event.CustomerID,
		)
		return r.billingSvc.HandleSubscriptionCreated(c, event)

	case billingdomain.WebhookEventLSSubscriptionUpdated:
		return r.billingSvc.HandleSubscriptionUpdated(c, event)

	case billingdomain.WebhookEventLSSubscriptionCancelled:
		event.Status = billingdomain.SubscriptionStatusCanceled
		return r.billingSvc.HandleSubscriptionCanceled(c, event)

	case billingdomain.WebhookEventLSSubscriptionPaused:
		event.Status = billingdomain.SubscriptionStatusPaused
		return r.billingSvc.HandleSubscriptionPaused(c, event)

	case billingdomain.WebhookEventLSSubscriptionResumed:
		event.Status = billingdomain.SubscriptionStatusActive
		return r.billingSvc.HandleSubscriptionResumed(c, event)

	case billingdomain.WebhookEventLSSubscriptionExpired:
		event.Status = billingdomain.SubscriptionStatusExpired
		return r.billingSvc.HandleSubscriptionExpired(c, event)

	case billingdomain.WebhookEventLSPaymentSuccess:
		event.Status = billingdomain.OrderStatusSucceeded
		return r.billingSvc.HandlePaymentSucceeded(c, event)

	case billingdomain.WebhookEventLSPaymentFailed:
		event.Status = billingdomain.OrderStatusFailed
		return r.billingSvc.HandlePaymentFailed(c, event)

	default:
		r.logger.Debug("ignoring unhandled LemonSqueezy event type", "event_type", event.EventType)
		return nil
	}
}

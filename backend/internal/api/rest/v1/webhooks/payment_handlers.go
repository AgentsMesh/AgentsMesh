package webhooks

import (
	"io"
	"net/http"

	billingdomain "github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (r *WebhookRouter) handleStripeWebhook(c *gin.Context) {
	if r.paymentFactory == nil || !r.paymentFactory.IsProviderAvailable(billingdomain.PaymentProviderStripe) {
		r.logger.Warn("Stripe webhook received but Stripe is not configured")
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Stripe not configured")
		return
	}

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		r.logger.Error("failed to read Stripe webhook body", "error", err)
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "failed to read request body")
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		r.logger.Warn("missing Stripe-Signature header")
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "missing signature")
		return
	}

	provider, err := r.paymentFactory.GetProvider(billingdomain.PaymentProviderStripe)
	if err != nil {
		r.logger.Error("failed to get Stripe provider", "error", err)
		apierr.InternalError(c, "provider not available")
		return
	}

	event, err := provider.HandleWebhook(c.Request.Context(), payload, signature)
	if err != nil {
		r.logger.Error("failed to validate Stripe webhook", "error", err)
		apierr.InvalidInput(c, "invalid webhook signature")
		return
	}

	r.logger.Info("received Stripe webhook",
		"event_id", event.EventID,
		"event_type", event.EventType,
	)

	var processErr error
	switch event.EventType {
	case billingdomain.WebhookEventCheckoutCompleted:
		event.Status = billingdomain.OrderStatusSucceeded
		processErr = r.billingSvc.HandlePaymentSucceeded(c, event)

	case billingdomain.WebhookEventInvoicePaid:
		event.Status = billingdomain.OrderStatusSucceeded
		processErr = r.billingSvc.HandlePaymentSucceeded(c, event)

	case billingdomain.WebhookEventInvoiceFailed:
		event.Status = billingdomain.OrderStatusFailed
		processErr = r.billingSvc.HandlePaymentFailed(c, event)

	case billingdomain.WebhookEventSubscriptionDeleted:
		event.Status = billingdomain.SubscriptionStatusCanceled
		processErr = r.billingSvc.HandleSubscriptionCanceled(c, event)

	case billingdomain.WebhookEventSubscriptionUpdated:
		processErr = r.billingSvc.HandleSubscriptionUpdated(c, event)

	default:
		r.logger.Debug("ignoring unhandled Stripe event type", "event_type", event.EventType)
	}

	if processErr != nil {
		r.logger.Error("failed to process Stripe webhook",
			"error", processErr,
			"event_type", event.EventType,
			"event_id", event.EventID,
		)
		apierr.InternalError(c, "failed to process event")
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

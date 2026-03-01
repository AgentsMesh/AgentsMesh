package webhooks

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// StripeWebhookHandler handles Stripe webhook events
type StripeWebhookHandler struct {
	provider       payment.Provider
	billingService BillingServiceInterface
}

// BillingServiceInterface defines the interface for billing service operations
// needed by the webhook handler
type BillingServiceInterface interface {
	// HandlePaymentSucceeded handles a successful payment
	HandlePaymentSucceeded(ctx *gin.Context, event *payment.WebhookEvent) error
	// HandlePaymentFailed handles a failed payment
	HandlePaymentFailed(ctx *gin.Context, event *payment.WebhookEvent) error
	// HandleSubscriptionCanceled handles subscription cancellation
	HandleSubscriptionCanceled(ctx *gin.Context, event *payment.WebhookEvent) error
	// HandleSubscriptionUpdated handles subscription updates
	HandleSubscriptionUpdated(ctx *gin.Context, event *payment.WebhookEvent) error
}

// NewStripeWebhookHandler creates a new Stripe webhook handler
func NewStripeWebhookHandler(provider payment.Provider, billingSvc BillingServiceInterface) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		provider:       provider,
		billingService: billingSvc,
	}
}

// Handle processes incoming Stripe webhooks
func (h *StripeWebhookHandler) Handle(c *gin.Context) {
	// Read the request body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Error("Failed to read webhook body", "error", err)
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "failed to read request body")
		return
	}

	// Get the Stripe signature header
	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		slog.Warn("Missing Stripe-Signature header")
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "missing signature")
		return
	}

	// Parse and validate the webhook
	event, err := h.provider.HandleWebhook(c.Request.Context(), payload, signature)
	if err != nil {
		slog.Error("Failed to validate webhook", "error", err)
		apierr.InvalidInput(c, "invalid webhook signature")
		return
	}

	slog.Info("Received Stripe webhook",
		"event_id", event.EventID,
		"event_type", event.EventType,
	)

	// Process the event based on type
	switch event.EventType {
	case billing.WebhookEventCheckoutCompleted:
		if err := h.handleCheckoutCompleted(c, event); err != nil {
			slog.Error("Failed to handle checkout.session.completed",
				"error", err,
				"event_id", event.EventID,
			)
			apierr.InternalError(c, "failed to process event")
			return
		}

	case billing.WebhookEventInvoicePaid:
		if err := h.handleInvoicePaid(c, event); err != nil {
			slog.Error("Failed to handle invoice.paid",
				"error", err,
				"event_id", event.EventID,
			)
			apierr.InternalError(c, "failed to process event")
			return
		}

	case billing.WebhookEventInvoiceFailed:
		if err := h.handleInvoicePaymentFailed(c, event); err != nil {
			slog.Error("Failed to handle invoice.payment_failed",
				"error", err,
				"event_id", event.EventID,
			)
			apierr.InternalError(c, "failed to process event")
			return
		}

	case billing.WebhookEventSubscriptionDeleted:
		if err := h.handleSubscriptionDeleted(c, event); err != nil {
			slog.Error("Failed to handle customer.subscription.deleted",
				"error", err,
				"event_id", event.EventID,
			)
			apierr.InternalError(c, "failed to process event")
			return
		}

	case billing.WebhookEventSubscriptionUpdated:
		if err := h.handleSubscriptionUpdated(c, event); err != nil {
			slog.Error("Failed to handle customer.subscription.updated",
				"error", err,
				"event_id", event.EventID,
			)
			apierr.InternalError(c, "failed to process event")
			return
		}

	default:
		slog.Debug("Ignoring unhandled event type",
			"event_type", event.EventType,
		)
	}

	// Acknowledge receipt
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handleCheckoutCompleted processes checkout.session.completed events
func (h *StripeWebhookHandler) handleCheckoutCompleted(c *gin.Context, event *payment.WebhookEvent) error {
	slog.Info("Processing checkout.session.completed",
		"order_no", event.OrderNo,
		"external_order_no", event.ExternalOrderNo,
		"customer_id", event.CustomerID,
		"subscription_id", event.SubscriptionID,
		"amount", event.Amount,
	)

	event.Status = billing.OrderStatusSucceeded
	return h.billingService.HandlePaymentSucceeded(c, event)
}

// handleInvoicePaid processes invoice.paid events (for recurring payments)
func (h *StripeWebhookHandler) handleInvoicePaid(c *gin.Context, event *payment.WebhookEvent) error {
	slog.Info("Processing invoice.paid",
		"external_order_no", event.ExternalOrderNo,
		"subscription_id", event.SubscriptionID,
		"amount", event.Amount,
	)

	event.Status = billing.OrderStatusSucceeded
	return h.billingService.HandlePaymentSucceeded(c, event)
}

// handleInvoicePaymentFailed processes invoice.payment_failed events
func (h *StripeWebhookHandler) handleInvoicePaymentFailed(c *gin.Context, event *payment.WebhookEvent) error {
	slog.Warn("Processing invoice.payment_failed",
		"external_order_no", event.ExternalOrderNo,
		"subscription_id", event.SubscriptionID,
		"reason", event.FailedReason,
	)

	event.Status = billing.OrderStatusFailed
	return h.billingService.HandlePaymentFailed(c, event)
}

// handleSubscriptionDeleted processes customer.subscription.deleted events
func (h *StripeWebhookHandler) handleSubscriptionDeleted(c *gin.Context, event *payment.WebhookEvent) error {
	slog.Info("Processing customer.subscription.deleted",
		"subscription_id", event.SubscriptionID,
		"customer_id", event.CustomerID,
	)

	event.Status = billing.SubscriptionStatusCanceled
	return h.billingService.HandleSubscriptionCanceled(c, event)
}

// handleSubscriptionUpdated processes customer.subscription.updated events
func (h *StripeWebhookHandler) handleSubscriptionUpdated(c *gin.Context, event *payment.WebhookEvent) error {
	slog.Info("Processing customer.subscription.updated",
		"subscription_id", event.SubscriptionID,
		"status", event.Status,
	)

	return h.billingService.HandleSubscriptionUpdated(c, event)
}

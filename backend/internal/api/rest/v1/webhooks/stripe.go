package webhooks

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

type StripeWebhookHandler struct {
	provider       payment.Provider
	billingService BillingServiceInterface
}

type BillingServiceInterface interface {
	HandlePaymentSucceeded(ctx *gin.Context, event *payment.WebhookEvent) error
	HandlePaymentFailed(ctx *gin.Context, event *payment.WebhookEvent) error
	HandleSubscriptionCanceled(ctx *gin.Context, event *payment.WebhookEvent) error
	HandleSubscriptionUpdated(ctx *gin.Context, event *payment.WebhookEvent) error
}

func NewStripeWebhookHandler(provider payment.Provider, billingSvc BillingServiceInterface) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		provider:       provider,
		billingService: billingSvc,
	}
}

func (h *StripeWebhookHandler) Handle(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to read webhook body", "error", err)
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "failed to read request body")
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		slog.WarnContext(c.Request.Context(), "Missing Stripe-Signature header")
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "missing signature")
		return
	}

	event, err := h.provider.HandleWebhook(c.Request.Context(), payload, signature)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to validate webhook", "error", err)
		apierr.InvalidInput(c, "invalid webhook signature")
		return
	}

	slog.InfoContext(c.Request.Context(), "Received Stripe webhook",
		"event_id", event.EventID,
		"event_type", event.EventType,
	)

	switch event.EventType {
	case billing.WebhookEventCheckoutCompleted:
		if err := h.handleCheckoutCompleted(c, event); err != nil {
			slog.ErrorContext(c.Request.Context(), "Failed to handle checkout.session.completed",
				"error", err,
				"event_id", event.EventID,
			)
			apierr.InternalError(c, "failed to process event")
			return
		}

	case billing.WebhookEventInvoicePaid:
		if err := h.handleInvoicePaid(c, event); err != nil {
			slog.ErrorContext(c.Request.Context(), "Failed to handle invoice.paid",
				"error", err,
				"event_id", event.EventID,
			)
			apierr.InternalError(c, "failed to process event")
			return
		}

	case billing.WebhookEventInvoiceFailed:
		if err := h.handleInvoicePaymentFailed(c, event); err != nil {
			slog.ErrorContext(c.Request.Context(), "Failed to handle invoice.payment_failed",
				"error", err,
				"event_id", event.EventID,
			)
			apierr.InternalError(c, "failed to process event")
			return
		}

	case billing.WebhookEventSubscriptionDeleted:
		if err := h.handleSubscriptionDeleted(c, event); err != nil {
			slog.ErrorContext(c.Request.Context(), "Failed to handle customer.subscription.deleted",
				"error", err,
				"event_id", event.EventID,
			)
			apierr.InternalError(c, "failed to process event")
			return
		}

	case billing.WebhookEventSubscriptionUpdated:
		if err := h.handleSubscriptionUpdated(c, event); err != nil {
			slog.ErrorContext(c.Request.Context(), "Failed to handle customer.subscription.updated",
				"error", err,
				"event_id", event.EventID,
			)
			apierr.InternalError(c, "failed to process event")
			return
		}

	default:
		slog.DebugContext(c.Request.Context(), "Ignoring unhandled event type",
			"event_type", event.EventType,
		)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

func (h *StripeWebhookHandler) handleCheckoutCompleted(c *gin.Context, event *payment.WebhookEvent) error {
	slog.InfoContext(c.Request.Context(), "Processing checkout.session.completed",
		"order_no", event.OrderNo,
		"external_order_no", event.ExternalOrderNo,
		"customer_id", event.CustomerID,
		"subscription_id", event.SubscriptionID,
		"amount", event.Amount,
	)

	event.Status = billing.OrderStatusSucceeded
	return h.billingService.HandlePaymentSucceeded(c, event)
}

func (h *StripeWebhookHandler) handleInvoicePaid(c *gin.Context, event *payment.WebhookEvent) error {
	slog.InfoContext(c.Request.Context(), "Processing invoice.paid",
		"external_order_no", event.ExternalOrderNo,
		"subscription_id", event.SubscriptionID,
		"amount", event.Amount,
	)

	event.Status = billing.OrderStatusSucceeded
	return h.billingService.HandlePaymentSucceeded(c, event)
}

func (h *StripeWebhookHandler) handleInvoicePaymentFailed(c *gin.Context, event *payment.WebhookEvent) error {
	slog.WarnContext(c.Request.Context(), "Processing invoice.payment_failed",
		"external_order_no", event.ExternalOrderNo,
		"subscription_id", event.SubscriptionID,
		"reason", event.FailedReason,
	)

	event.Status = billing.OrderStatusFailed
	return h.billingService.HandlePaymentFailed(c, event)
}

func (h *StripeWebhookHandler) handleSubscriptionDeleted(c *gin.Context, event *payment.WebhookEvent) error {
	slog.InfoContext(c.Request.Context(), "Processing customer.subscription.deleted",
		"subscription_id", event.SubscriptionID,
		"customer_id", event.CustomerID,
	)

	event.Status = billing.SubscriptionStatusCanceled
	return h.billingService.HandleSubscriptionCanceled(c, event)
}

func (h *StripeWebhookHandler) handleSubscriptionUpdated(c *gin.Context, event *payment.WebhookEvent) error {
	slog.InfoContext(c.Request.Context(), "Processing customer.subscription.updated",
		"subscription_id", event.SubscriptionID,
		"status", event.Status,
	)

	return h.billingService.HandleSubscriptionUpdated(c, event)
}

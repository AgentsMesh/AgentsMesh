package billing

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

func (s *Service) HandlePaymentSucceeded(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	attrs := []any{"event_id", event.EventID, "provider", event.Provider, "event_type", event.EventType}

	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			return nil
		}
		slog.ErrorContext(c.Request.Context(), "webhook idempotency check failed", append(attrs, "error", err)...)
		return err
	}
	// Roll back the idempotency mark if the handler fails, so the event
	// can be retried on the next delivery.
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	var order *billing.PaymentOrder
	var err error

	if event.OrderNo != "" {
		order, err = s.GetPaymentOrderByNo(ctx, event.OrderNo)
		if err != nil && !errors.Is(err, ErrOrderNotFound) {
			slog.ErrorContext(c.Request.Context(), "failed to lookup order by order_no", append(attrs, "order_no", event.OrderNo, "error", err)...)
			return fmt.Errorf("failed to lookup order by order_no: %w", err)
		}
	}
	if order == nil && event.ExternalOrderNo != "" {
		order, err = s.GetPaymentOrderByExternalNo(ctx, event.ExternalOrderNo)
		if err != nil && !errors.Is(err, ErrOrderNotFound) {
			slog.ErrorContext(c.Request.Context(), "failed to lookup order by external_order_no", append(attrs, "external_order_no", event.ExternalOrderNo, "error", err)...)
			return fmt.Errorf("failed to lookup order by external_order_no: %w", err)
		}
	}

	if order == nil && event.SubscriptionID != "" {
		return s.handleRecurringPaymentSuccess(ctx, event)
	}

	if order == nil {
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "order not found for payment webhook", attrs...)
			return fmt.Errorf("order not found: %w", err)
		}
		return nil
	}

	if err := s.UpdatePaymentOrderStatus(ctx, order.OrderNo, billing.OrderStatusSucceeded, nil); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update order status", append(attrs, "order_no", order.OrderNo, "error", err)...)
		return fmt.Errorf("failed to update order status: %w", err)
	}

	tx := &billing.PaymentTransaction{
		PaymentOrderID:        order.ID,
		TransactionType:       billing.TransactionTypePayment,
		ExternalTransactionID: &event.ExternalOrderNo,
		Amount:                event.Amount,
		Currency:              event.Currency,
		Status:                billing.TransactionStatusSucceeded,
		WebhookEventID:        &event.EventID,
		WebhookEventType:      &event.EventType,
		RawPayload:            billing.RawPayload(event.RawPayload),
	}
	if err := s.CreatePaymentTransaction(ctx, tx); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to create payment transaction", append(attrs, "order_no", order.OrderNo, "error", err)...)
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	slog.InfoContext(c.Request.Context(), "payment succeeded", append(attrs, "order_no", order.OrderNo, "org_id", order.OrganizationID, "order_type", order.OrderType, "amount", event.Amount)...)

	switch order.OrderType {
	case billing.OrderTypeSubscription:
		return s.activateSubscription(ctx, order, event)
	case billing.OrderTypeSeatPurchase:
		return s.addSeats(ctx, order)
	case billing.OrderTypePlanUpgrade:
		return s.upgradePlan(ctx, order)
	case billing.OrderTypeRenewal:
		return s.renewSubscriptionFromOrder(ctx, order)
	}

	return nil
}

func (s *Service) HandlePaymentFailed(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	attrs := []any{"event_id", event.EventID, "provider", event.Provider, "event_type", event.EventType}

	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			return nil
		}
		slog.ErrorContext(c.Request.Context(), "webhook idempotency check failed", append(attrs, "error", err)...)
		return err
	}
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	if event.SubscriptionID != "" {
		slog.WarnContext(c.Request.Context(), "recurring payment failed", append(attrs, "subscription_id", event.SubscriptionID)...)
		return s.handleRecurringPaymentFailure(ctx, event)
	}

	var order *billing.PaymentOrder
	var err error

	if event.OrderNo != "" {
		order, err = s.GetPaymentOrderByNo(ctx, event.OrderNo)
		if err != nil && !errors.Is(err, ErrOrderNotFound) {
			slog.ErrorContext(c.Request.Context(), "failed to lookup order by order_no", append(attrs, "order_no", event.OrderNo, "error", err)...)
			return fmt.Errorf("failed to lookup order by order_no: %w", err)
		}
	}
	if order == nil && event.ExternalOrderNo != "" {
		order, err = s.GetPaymentOrderByExternalNo(ctx, event.ExternalOrderNo)
		if err != nil && !errors.Is(err, ErrOrderNotFound) {
			slog.ErrorContext(c.Request.Context(), "failed to lookup order by external_order_no", append(attrs, "external_order_no", event.ExternalOrderNo, "error", err)...)
			return fmt.Errorf("failed to lookup order by external_order_no: %w", err)
		}
	}

	if err != nil || order == nil {
		return nil
	}

	slog.WarnContext(c.Request.Context(), "payment failed", append(attrs, "order_no", order.OrderNo, "reason", event.FailedReason)...)

	return s.UpdatePaymentOrderStatus(ctx, order.OrderNo, billing.OrderStatusFailed, &event.FailedReason)
}

func (s *Service) HandleSubscriptionCanceled(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	if event.SubscriptionID == "" {
		return nil
	}

	attrs := []any{"event_id", event.EventID, "provider", event.Provider, "subscription_id", event.SubscriptionID}

	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			return nil
		}
		slog.ErrorContext(c.Request.Context(), "webhook idempotency check failed", append(attrs, "error", err)...)
		return err
	}
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "subscription not found for cancellation webhook", append(attrs, "error", err)...)
		return nil
	}

	now := time.Now()
	sub.Status = billing.SubscriptionStatusCanceled
	sub.CanceledAt = &now

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to save canceled subscription", append(attrs, "org_id", sub.OrganizationID, "error", err)...)
		return err
	}

	slog.InfoContext(c.Request.Context(), "subscription canceled", append(attrs, "org_id", sub.OrganizationID)...)

	status := billing.SubscriptionStatusCanceled
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	return nil
}

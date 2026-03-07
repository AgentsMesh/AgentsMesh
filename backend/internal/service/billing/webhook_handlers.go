package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Payment Webhook Event Handlers
// ===========================================

// HandlePaymentSucceeded handles a successful payment webhook event
func (s *Service) HandlePaymentSucceeded(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

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

	// Try to find order by order_no first, then by external_order_no
	var order *billing.PaymentOrder
	var err error

	if event.OrderNo != "" {
		order, err = s.GetPaymentOrderByNo(ctx, event.OrderNo)
		if err != nil && !errors.Is(err, ErrOrderNotFound) {
			return fmt.Errorf("failed to lookup order by order_no: %w", err)
		}
	}
	if order == nil && event.ExternalOrderNo != "" {
		order, err = s.GetPaymentOrderByExternalNo(ctx, event.ExternalOrderNo)
		if err != nil && !errors.Is(err, ErrOrderNotFound) {
			return fmt.Errorf("failed to lookup order by external_order_no: %w", err)
		}
	}

	// For recurring payments (invoice.paid), there may not be an order in our system
	if order == nil && event.SubscriptionID != "" {
		return s.handleRecurringPaymentSuccess(ctx, event)
	}

	if order == nil {
		// No order found and not a recurring payment — nothing to process
		if err != nil {
			return fmt.Errorf("order not found: %w", err)
		}
		return nil
	}

	// Update order status
	if err := s.UpdatePaymentOrderStatus(ctx, order.OrderNo, billing.OrderStatusSucceeded, nil); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Create transaction record
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
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Process based on order type
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

// HandlePaymentFailed handles a failed payment webhook event
func (s *Service) HandlePaymentFailed(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	// Idempotency check
	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			return nil
		}
		return err
	}
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	// For recurring payment failures, freeze the subscription
	if event.SubscriptionID != "" {
		return s.handleRecurringPaymentFailure(ctx, event)
	}

	// Try to find and update the order
	var order *billing.PaymentOrder
	var err error

	if event.OrderNo != "" {
		order, err = s.GetPaymentOrderByNo(ctx, event.OrderNo)
		if err != nil && !errors.Is(err, ErrOrderNotFound) {
			return fmt.Errorf("failed to lookup order by order_no: %w", err)
		}
	}
	if order == nil && event.ExternalOrderNo != "" {
		order, err = s.GetPaymentOrderByExternalNo(ctx, event.ExternalOrderNo)
		if err != nil && !errors.Is(err, ErrOrderNotFound) {
			return fmt.Errorf("failed to lookup order by external_order_no: %w", err)
		}
	}

	if err != nil || order == nil {
		return nil // Order not found, nothing to update
	}

	// Update order status
	return s.UpdatePaymentOrderStatus(ctx, order.OrderNo, billing.OrderStatusFailed, &event.FailedReason)
}

// HandleSubscriptionCanceled handles subscription cancellation webhook event
func (s *Service) HandleSubscriptionCanceled(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
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
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		return nil // Subscription not found
	}

	// Update subscription status
	now := time.Now()
	sub.Status = billing.SubscriptionStatusCanceled
	sub.CanceledAt = &now

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
		return err
	}

	// Sync organization table
	status := billing.SubscriptionStatusCanceled
	s.syncOrganizationSubscription(ctx, sub.OrganizationID, nil, &status)
	return nil
}

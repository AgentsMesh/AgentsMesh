package billing

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Payment Failed Webhook Tests
// ===========================================

func TestHandlePaymentFailed(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-FAIL-001",
		OrderType:       billing.OrderTypeSubscription,
		Amount:          19.99,
		ActualAmount:    19.99,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:      "evt_pay_fail",
		EventType:    "payment_intent.failed",
		OrderNo:      "ORD-FAIL-001",
		Status:       billing.OrderStatusFailed,
		FailedReason: "Card declined",
	}

	err := service.HandlePaymentFailed(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment failed: %v", err)
	}

	updatedOrder, _ := service.GetPaymentOrderByNo(ctx, "ORD-FAIL-001")
	if updatedOrder.Status != billing.OrderStatusFailed {
		t.Errorf("expected order status failed, got %s", updatedOrder.Status)
	}
}

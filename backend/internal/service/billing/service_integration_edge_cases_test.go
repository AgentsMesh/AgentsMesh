package billing

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// TestIntegrationPaymentFailedNoOrder tests payment failed with no order
func TestIntegrationPaymentFailedNoOrder(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	c, _ := createTestGinContext()

	// Event for non-existent order (not a recurring payment)
	event := &payment.WebhookEvent{
		EventID:      "evt_fail_no_order",
		EventType:    "payment_intent.payment_failed",
		Provider:     "mock",
		OrderNo:      "ORD-NONEXISTENT",
		Amount:       19.99,
		Currency:     "USD",
		Status:       billing.OrderStatusFailed,
		FailedReason: "Card declined",
	}

	// Should not error
	err := service.HandlePaymentFailed(c, event)
	if err != nil {
		t.Errorf("expected no error for non-existent order, got %v", err)
	}
}

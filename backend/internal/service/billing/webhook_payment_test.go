package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Webhook Payment Handler Tests
// ===========================================

// TestHandlePaymentSucceeded_WithOrder tests payment succeeded with existing order
func TestHandlePaymentSucceeded_WithOrder(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a payment order
	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)
	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-TEST-001",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &planID,
		Amount:          19.99,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		ExpiresAt:       &expiresAt,
	})

	// Create subscription
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // will be upgraded
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_payment_order",
		EventType: billing.WebhookEventLSPaymentSuccess,
		Provider:  billing.PaymentProviderLemonSqueezy,
		OrderNo:   "ORD-TEST-001",
		Amount:    19.99,
		Currency:  "USD",
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify order status updated
	order, _ := svc.GetPaymentOrderByNo(context.Background(), "ORD-TEST-001")
	if order.Status != billing.OrderStatusSucceeded {
		t.Errorf("expected order status succeeded, got %s", order.Status)
	}
}

// TestHandlePaymentFailed_WithOrder tests payment failed with existing order
func TestHandlePaymentFailed_WithOrder(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)
	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-FAIL-001",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &planID,
		Amount:          19.99,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		ExpiresAt:       &expiresAt,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:      "evt_payment_failed",
		EventType:    billing.WebhookEventLSPaymentFailed,
		Provider:     billing.PaymentProviderLemonSqueezy,
		OrderNo:      "ORD-FAIL-001",
		FailedReason: "Card declined",
	}

	err := svc.HandlePaymentFailed(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify order status updated
	order, _ := svc.GetPaymentOrderByNo(context.Background(), "ORD-FAIL-001")
	if order.Status != billing.OrderStatusFailed {
		t.Errorf("expected order status failed, got %s", order.Status)
	}
}

// TestHandleRecurringPaymentSuccess_Extended tests recurring payment period extension
func TestHandleRecurringPaymentSuccess_Extended(t *testing.T) {
	svc, db := setupTestService(t)

	lsSubID := "ls_sub_recurring"
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:             1,
		PlanID:                     2,
		Status:                     billing.SubscriptionStatusActive,
		BillingCycle:               billing.BillingCycleMonthly,
		CurrentPeriodStart:         now.AddDate(0, -1, 0),
		CurrentPeriodEnd:           now,
		SeatCount:                  1,
		LemonSqueezySubscriptionID: &lsSubID,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_recurring_payment",
		EventType:      billing.WebhookEventLSPaymentSuccess,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
		Amount:         19.99,
		Currency:       "USD",
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify period extended
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.CurrentPeriodEnd.Before(now) {
		t.Error("expected period to be extended")
	}
}

// TestHandleRecurringPaymentFailure_Freeze tests recurring payment failure freezes subscription
func TestHandleRecurringPaymentFailure_Freeze(t *testing.T) {
	svc, db := setupTestService(t)

	lsSubID := "ls_sub_fail_recurring"
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:             1,
		PlanID:                     2,
		Status:                     billing.SubscriptionStatusActive,
		BillingCycle:               billing.BillingCycleMonthly,
		CurrentPeriodStart:         now,
		CurrentPeriodEnd:           now.AddDate(0, 1, 0),
		SeatCount:                  1,
		LemonSqueezySubscriptionID: &lsSubID,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_recurring_fail",
		EventType:      billing.WebhookEventLSPaymentFailed,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
		FailedReason:   "Payment method declined",
	}

	err := svc.HandlePaymentFailed(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify subscription frozen
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.FrozenAt == nil {
		t.Error("expected subscription to be frozen")
	}
}

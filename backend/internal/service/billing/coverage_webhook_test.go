package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Webhook/Payment Tests
// ===========================================

// Helper function for pointer string
func ptrString(s string) *string {
	return &s
}

// TestHandlePaymentSucceeded_WithExternalOrderNo tests finding order by external_order_no
func TestHandlePaymentSucceeded_WithExternalOrderNo(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)

	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-EXT-LOOKUP",
		ExternalOrderNo: ptrString("ext_order_123"),
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &planID,
		Seats:           1,
		BillingCycle:    billing.BillingCycleMonthly,
		Amount:          19.99,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderStripe,
		ExpiresAt:       &expiresAt,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:         "evt_ext_lookup",
		EventType:       "checkout.session.completed",
		Provider:        billing.PaymentProviderStripe,
		ExternalOrderNo: "ext_order_123", // Match by external_order_no
		Amount:          19.99,
		Currency:        "USD",
		RawPayload:      map[string]interface{}{"test": true},
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestHandlePaymentFailed_RecurringWithSubscription tests recurring payment failure
func TestHandlePaymentFailed_RecurringWithSubscription(t *testing.T) {
	svc, db := setupTestService(t)

	lsSubID := "ls_sub_recurring_fail"
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
		SubscriptionID: lsSubID, // Has subscription ID - triggers recurring payment failure path
		FailedReason:   "Card declined",
	}

	err := svc.HandlePaymentFailed(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Subscription should be frozen
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusFrozen {
		t.Errorf("expected frozen status, got %s", sub.Status)
	}
}

// TestHandleSubscriptionCreated_FallbackByOrderNo tests subscription created with order_no fallback
func TestHandleSubscriptionCreated_FallbackByOrderNo(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)

	// Create order first
	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-LS-FALLBACK",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &planID,
		Seats:           1,
		BillingCycle:    billing.BillingCycleMonthly,
		Amount:          19.99,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		ExpiresAt:       &expiresAt,
	})

	// Create subscription (without LemonSqueezy IDs)
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
		// No LemonSqueezy IDs
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_created_fallback",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_fallback",
		CustomerID:     "nonexistent_customer", // Customer lookup will fail
		OrderNo:        "ORD-LS-FALLBACK",      // Fallback to order lookup
	}

	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify subscription was updated via fallback
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_fallback" {
		t.Error("expected subscription ID to be set via order fallback")
	}
}

// TestHandleSubscriptionCreated_BothCustomerAndOrder tests both customer and order lookups
func TestHandleSubscriptionCreated_BothCustomerAndOrderFail(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_created_both_fail",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_both_fail",
		CustomerID:     "nonexistent_customer",
		OrderNo:        "nonexistent_order",
	}

	// Should return nil even when both lookups fail
	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Errorf("expected nil when both lookups fail, got %v", err)
	}
}

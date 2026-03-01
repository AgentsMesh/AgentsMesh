package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Coverage Boost Tests - Payment Handling
// ===========================================

// TestHandlePaymentFailed_NoOrderNoSubID tests failed payment with neither order nor subscription
func TestHandlePaymentFailed_NoOrderNoSubID(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:      "evt_fail_nothing",
		EventType:    billing.WebhookEventLSPaymentFailed,
		Provider:     billing.PaymentProviderLemonSqueezy,
		OrderNo:      "",
		FailedReason: "Card declined",
	}

	err := svc.HandlePaymentFailed(c, event)
	if err != nil {
		t.Errorf("expected nil when nothing found, got %v", err)
	}
}

// TestHandlePaymentSucceeded_DefaultOrderType tests payment with unknown order type
func TestHandlePaymentSucceeded_UnknownOrderType(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)

	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-UNKNOWN-001",
		OrderType:       "unknown_type", // Unknown order type
		PlanID:          &planID,
		Amount:          19.99,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		ExpiresAt:       &expiresAt,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_unknown_type",
		EventType: billing.WebhookEventLSPaymentSuccess,
		Provider:  billing.PaymentProviderLemonSqueezy,
		OrderNo:   "ORD-UNKNOWN-001",
		Amount:    19.99,
		Currency:  "USD",
	}

	// Should complete without error (falls through switch)
	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify order status still updated
	order, _ := svc.GetPaymentOrderByNo(context.Background(), "ORD-UNKNOWN-001")
	if order.Status != billing.OrderStatusSucceeded {
		t.Errorf("expected order status succeeded, got %s", order.Status)
	}
}

// TestHandleRecurringPaymentSuccess_WithPendingChanges tests applying pending changes
func TestHandleRecurringPaymentSuccess_WithPendingChanges(t *testing.T) {
	svc, db := setupTestService(t)

	lsSubID := "ls_sub_pending_changes"
	now := time.Now()
	downgradePlan := "based"
	nextCycle := billing.BillingCycleYearly

	db.Create(&billing.Subscription{
		OrganizationID:             1,
		PlanID:                     2, // pro
		Status:                     billing.SubscriptionStatusActive,
		BillingCycle:               billing.BillingCycleMonthly,
		CurrentPeriodStart:         now.AddDate(0, -1, 0),
		CurrentPeriodEnd:           now,
		SeatCount:                  1,
		LemonSqueezySubscriptionID: &lsSubID,
		DowngradeToPlan:            &downgradePlan,
		NextBillingCycle:           &nextCycle,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_recurring_pending",
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

	sub, _ := svc.GetSubscription(context.Background(), 1)

	// Downgrade should be applied
	if sub.PlanID != 1 { // based plan
		t.Errorf("expected plan ID 1 (based), got %d", sub.PlanID)
	}

	// DowngradeToPlan should be cleared
	if sub.DowngradeToPlan != nil {
		t.Error("expected DowngradeToPlan to be cleared")
	}

	// NextBillingCycle should be applied
	if sub.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", sub.BillingCycle)
	}

	// NextBillingCycle should be cleared
	if sub.NextBillingCycle != nil {
		t.Error("expected NextBillingCycle to be cleared")
	}
}

// TestHandleRecurringPaymentSuccess_NotFound tests recurring payment with non-existent subscription
func TestHandleRecurringPaymentSuccess_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_recurring_not_found",
		EventType:      billing.WebhookEventLSPaymentSuccess,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "nonexistent",
		Amount:         19.99,
		Currency:       "USD",
	}

	// Should return nil (subscription not found is ignored)
	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Errorf("expected nil for non-existent subscription, got %v", err)
	}
}

// TestHandleRecurringPaymentFailure_NotFound tests recurring failure with non-existent subscription
func TestHandleRecurringPaymentFailure_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_recurring_fail_not_found",
		EventType:      billing.WebhookEventLSPaymentFailed,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "nonexistent",
		FailedReason:   "Card declined",
	}

	err := svc.HandlePaymentFailed(c, event)
	if err != nil {
		t.Errorf("expected nil for non-existent subscription, got %v", err)
	}
}

// TestHandlePaymentSucceeded_ExternalOrderNoLookup tests finding order by external order no
func TestHandlePaymentSucceeded_ExternalOrderNoLookup(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)
	externalNo := "ext_order_lookup_123"

	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-INT-001",
		ExternalOrderNo: &externalNo,
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &planID,
		Amount:          19.99,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		ExpiresAt:       &expiresAt,
	})

	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:         "evt_ext_order_lookup",
		EventType:       billing.WebhookEventLSPaymentSuccess,
		Provider:        billing.PaymentProviderLemonSqueezy,
		OrderNo:         "",         // No internal order no
		ExternalOrderNo: externalNo, // Use external order no
		Amount:          19.99,
		Currency:        "USD",
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	order, _ := svc.GetPaymentOrderByNo(context.Background(), "ORD-INT-001")
	if order.Status != billing.OrderStatusSucceeded {
		t.Errorf("expected order status succeeded, got %s", order.Status)
	}
}

// TestHandlePaymentFailed_ExternalOrderNoLookup tests finding order by external order no for failure
func TestHandlePaymentFailed_ExternalOrderNoLookup(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)
	externalNo := "ext_fail_order_456"

	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-FAIL-INT-001",
		ExternalOrderNo: &externalNo,
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
		EventID:         "evt_ext_fail_lookup",
		EventType:       billing.WebhookEventLSPaymentFailed,
		Provider:        billing.PaymentProviderLemonSqueezy,
		OrderNo:         "",         // No internal order no
		ExternalOrderNo: externalNo, // Use external order no
		FailedReason:    "Card declined",
	}

	err := svc.HandlePaymentFailed(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	order, _ := svc.GetPaymentOrderByNo(context.Background(), "ORD-FAIL-INT-001")
	if order.Status != billing.OrderStatusFailed {
		t.Errorf("expected order status failed, got %s", order.Status)
	}
}

// TestRecurringPaymentSuccess_YearlyCycle tests recurring payment with yearly billing
func TestRecurringPaymentSuccess_YearlyCycle(t *testing.T) {
	svc, db := setupTestService(t)

	lsSubID := "ls_sub_yearly_recurring"
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:             1,
		PlanID:                     2,
		Status:                     billing.SubscriptionStatusActive,
		BillingCycle:               billing.BillingCycleYearly,
		CurrentPeriodStart:         now.AddDate(-1, 0, 0),
		CurrentPeriodEnd:           now,
		SeatCount:                  1,
		LemonSqueezySubscriptionID: &lsSubID,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_yearly_recurring",
		EventType:      billing.WebhookEventLSPaymentSuccess,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
		Amount:         199.99,
		Currency:       "USD",
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)

	// Period should be extended by 1 year
	expectedEnd := now.AddDate(1, 0, 0)
	if sub.CurrentPeriodEnd.Before(expectedEnd.AddDate(0, 0, -1)) {
		t.Error("expected period to be extended by 1 year")
	}
}

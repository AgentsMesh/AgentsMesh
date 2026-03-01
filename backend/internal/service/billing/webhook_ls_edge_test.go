package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// TestHandleSubscriptionPaused_NotFound tests pausing non-existent subscription
// Webhook handlers return nil when subscription not found (graceful degradation)
func TestHandleSubscriptionPaused_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_pause_notfound",
		EventType:      billing.WebhookEventLSSubscriptionPaused,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "nonexistent_sub",
		CustomerID:     "nonexistent_cust",
		Status:         billing.SubscriptionStatusPaused,
	}

	// Should return nil (graceful handling) when subscription not found
	err := svc.HandleSubscriptionPaused(c, event)
	if err != nil {
		t.Errorf("expected nil error for graceful handling, got %v", err)
	}
}

// TestHandleSubscriptionPaused_ByCustomerID tests pausing by customer ID
func TestHandleSubscriptionPaused_ByCustomerID(t *testing.T) {
	svc, db := setupTestService(t)

	// Create subscription with LemonSqueezy subscription ID (required for lookup)
	lsSubID := "ls_sub_pause_123"
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
		EventID:        "evt_pause_cust",
		EventType:      billing.WebhookEventLSSubscriptionPaused,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
		Status:         billing.SubscriptionStatusPaused,
	}

	err := svc.HandleSubscriptionPaused(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status changed
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusPaused {
		t.Errorf("expected status paused, got %s", sub.Status)
	}
}

// TestHandleSubscriptionResumed_NotFound tests resuming non-existent subscription
func TestHandleSubscriptionResumed_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_resume_notfound",
		EventType:      billing.WebhookEventLSSubscriptionResumed,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "nonexistent_sub",
		CustomerID:     "nonexistent_cust",
		Status:         billing.SubscriptionStatusActive,
	}

	// Should return nil (graceful handling) when subscription not found
	err := svc.HandleSubscriptionResumed(c, event)
	if err != nil {
		t.Errorf("expected nil error for graceful handling, got %v", err)
	}
}

// TestHandleSubscriptionResumed_BySubscriptionID tests resuming by subscription ID
func TestHandleSubscriptionResumed_BySubscriptionID(t *testing.T) {
	svc, db := setupTestService(t)

	// Create paused subscription with LemonSqueezy subscription ID
	lsSubID := "ls_sub_resume_456"
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:             1,
		PlanID:                     2,
		Status:                     billing.SubscriptionStatusPaused,
		BillingCycle:               billing.BillingCycleMonthly,
		CurrentPeriodStart:         now,
		CurrentPeriodEnd:           now.AddDate(0, 1, 0),
		SeatCount:                  1,
		LemonSqueezySubscriptionID: &lsSubID,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_resume_cust",
		EventType:      billing.WebhookEventLSSubscriptionResumed,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
		Status:         billing.SubscriptionStatusActive,
	}

	err := svc.HandleSubscriptionResumed(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status changed
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
}

// TestHandleSubscriptionExpired_NotFound tests expiring non-existent subscription
func TestHandleSubscriptionExpired_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_expire_notfound",
		EventType:      billing.WebhookEventLSSubscriptionExpired,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "nonexistent_sub",
		CustomerID:     "nonexistent_cust",
		Status:         billing.SubscriptionStatusExpired,
	}

	// Should return nil (graceful handling) when subscription not found
	err := svc.HandleSubscriptionExpired(c, event)
	if err != nil {
		t.Errorf("expected nil error for graceful handling, got %v", err)
	}
}

// TestHandleSubscriptionExpired_BySubscriptionID tests expiring by subscription ID
func TestHandleSubscriptionExpired_BySubscriptionID(t *testing.T) {
	svc, db := setupTestService(t)

	// Create subscription with LemonSqueezy subscription ID
	lsSubID := "ls_sub_expire_789"
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
		EventID:        "evt_expire_cust",
		EventType:      billing.WebhookEventLSSubscriptionExpired,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
		Status:         billing.SubscriptionStatusExpired,
	}

	err := svc.HandleSubscriptionExpired(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status changed
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusExpired {
		t.Errorf("expected status expired, got %s", sub.Status)
	}
}

// TestHandleSubscriptionCreated_NotFoundOrder tests subscription created without matching order/customer
func TestHandleSubscriptionCreated_NotFoundOrder(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_create_no_order",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "sub_new_123",
		CustomerID:     "cust_new_456",
		OrderNo:        "nonexistent_order",
		Status:         billing.SubscriptionStatusActive,
	}

	// Should return nil - graceful handling when no matching subscription found
	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Errorf("expected nil error for graceful handling, got %v", err)
	}
}

// TestHandlePaymentSucceeded_RenewalNotFound tests renewal payment when subscription not found
func TestHandlePaymentSucceeded_RenewalNotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_renew_notfound",
		EventType:      billing.WebhookEventLSPaymentSuccess,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "nonexistent_sub",
		CustomerID:     "nonexistent_cust",
		Status:         billing.OrderStatusSucceeded,
		Amount:         20.00,
		Currency:       "USD",
	}

	// Should return nil - graceful handling for recurring payment without subscription
	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Errorf("expected nil error for graceful handling, got %v", err)
	}
}

// TestHandlePaymentFailed_NotFound tests payment failed when subscription not found
func TestHandlePaymentFailed_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_fail_notfound",
		EventType:      billing.WebhookEventLSPaymentFailed,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "nonexistent_sub",
		CustomerID:     "nonexistent_cust",
		Status:         billing.OrderStatusFailed,
	}

	// Should return nil - graceful handling when subscription not found
	err := svc.HandlePaymentFailed(c, event)
	if err != nil {
		t.Errorf("expected nil error for graceful handling, got %v", err)
	}
}

// TestHandleSubscriptionCanceled_NotFound tests canceling non-existent subscription
func TestHandleSubscriptionCanceled_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_cancel_notfound",
		EventType:      billing.WebhookEventLSSubscriptionCancelled,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "nonexistent_sub",
		CustomerID:     "nonexistent_cust",
		Status:         billing.SubscriptionStatusCanceled,
	}

	// Should return nil - graceful handling when subscription not found
	err := svc.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Errorf("expected nil error for graceful handling, got %v", err)
	}
}

// TestHandleSubscriptionCanceled_Success tests successful subscription cancellation
func TestHandleSubscriptionCanceled_Success(t *testing.T) {
	svc, db := setupTestService(t)

	// Create subscription with LemonSqueezy subscription ID
	lsSubID := "ls_sub_cancel_999"
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
		EventID:        "evt_cancel_success",
		EventType:      billing.WebhookEventLSSubscriptionCancelled,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
		Status:         billing.SubscriptionStatusCanceled,
	}

	err := svc.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status changed
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected status canceled, got %s", sub.Status)
	}
	if sub.CanceledAt == nil {
		t.Error("expected CanceledAt to be set")
	}
}

package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Webhook Subscription Handler Tests
// ===========================================

// TestHandleSubscriptionCanceled_ByStripeID tests successful cancellation via webhook
func TestHandleSubscriptionCanceled_ByStripeID(t *testing.T) {
	svc, db := setupTestService(t)

	stripeSubID := "sub_stripe_cancel"
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:       1,
		PlanID:               2,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleMonthly,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.AddDate(0, 1, 0),
		SeatCount:            1,
		StripeSubscriptionID: &stripeSubID,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_stripe_cancel",
		EventType:      "customer.subscription.deleted",
		Provider:       billing.PaymentProviderStripe,
		SubscriptionID: stripeSubID,
	}

	err := svc.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected canceled status, got %s", sub.Status)
	}
}

// TestHandleSubscriptionCreated_WithCustomerID tests creating subscription with customer ID
func TestHandleSubscriptionCreated_WithCustomerID(t *testing.T) {
	svc, db := setupTestService(t)

	lsCustID := "ls_cust_create"
	now := time.Now()
	// Create subscription with only customer ID set
	db.Create(&billing.Subscription{
		OrganizationID:         1,
		PlanID:                 2,
		Status:                 billing.SubscriptionStatusActive,
		BillingCycle:           billing.BillingCycleMonthly,
		CurrentPeriodStart:     now,
		CurrentPeriodEnd:       now.AddDate(0, 1, 0),
		SeatCount:              1,
		LemonSqueezyCustomerID: &lsCustID,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_create",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_new",
		CustomerID:     lsCustID,
	}

	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify subscription ID was set
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_new" {
		t.Error("expected LemonSqueezy subscription ID to be set")
	}
}

// TestHandleSubscriptionPaused_Success tests successful subscription pause
func TestHandleSubscriptionPaused_Success(t *testing.T) {
	svc, db := setupTestService(t)

	lsSubID := "ls_sub_pause"
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
		EventID:        "evt_pause_success",
		EventType:      billing.WebhookEventLSSubscriptionPaused,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
	}

	err := svc.HandleSubscriptionPaused(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusPaused {
		t.Errorf("expected paused status, got %s", sub.Status)
	}
	if sub.FrozenAt == nil {
		t.Error("expected FrozenAt to be set")
	}
}

// TestHandleSubscriptionResumed_Success tests successful subscription resume
func TestHandleSubscriptionResumed_Success(t *testing.T) {
	svc, db := setupTestService(t)

	lsSubID := "ls_sub_resume"
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
		FrozenAt:                   &now,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_resume_success",
		EventType:      billing.WebhookEventLSSubscriptionResumed,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
	}

	err := svc.HandleSubscriptionResumed(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected active status, got %s", sub.Status)
	}
	if sub.FrozenAt != nil {
		t.Error("expected FrozenAt to be cleared")
	}
}

// TestHandleSubscriptionExpired_Success tests successful subscription expiration
func TestHandleSubscriptionExpired_Success(t *testing.T) {
	svc, db := setupTestService(t)

	lsSubID := "ls_sub_expire"
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
		EventID:        "evt_expire_success",
		EventType:      billing.WebhookEventLSSubscriptionExpired,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
	}

	err := svc.HandleSubscriptionExpired(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusExpired {
		t.Errorf("expected expired status, got %s", sub.Status)
	}
	if sub.CanceledAt == nil {
		t.Error("expected CanceledAt to be set")
	}
}

// TestHandleSubscriptionCreated_WithOrderNo tests subscription created with order number fallback
func TestHandleSubscriptionCreated_WithOrderNo(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)

	// Create payment order
	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-CREATE-001",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &planID,
		Amount:          19.99,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		ExpiresAt:       &expiresAt,
	})

	// Create subscription without customer ID
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	c, _ := createTestGinContext()
	// Use a non-existent customer ID to trigger the order_no fallback path
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_create_order",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_order",
		CustomerID:     "nonexistent_customer", // This will fail lookup, triggering order_no fallback
		OrderNo:        "ORD-CREATE-001",
	}

	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify subscription ID was set via order lookup
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_order" {
		t.Error("expected LemonSqueezy subscription ID to be set")
	}
}

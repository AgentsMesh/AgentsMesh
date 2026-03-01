package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Coverage Boost Tests - Subscription Handling
// ===========================================

// TestActivateSubscription_NewSubscription tests creating new subscription from order
func TestActivateSubscription_NewSubscription(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)

	db.Create(&billing.PaymentOrder{
		OrganizationID:  999, // New org with no subscription
		OrderNo:         "ORD-NEW-SUB-001",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &planID,
		Seats:           5,
		BillingCycle:    billing.BillingCycleYearly,
		Amount:          99.99,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		ExpiresAt:       &expiresAt,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_new_sub",
		EventType:      billing.WebhookEventLSPaymentSuccess,
		Provider:       billing.PaymentProviderLemonSqueezy,
		OrderNo:        "ORD-NEW-SUB-001",
		CustomerID:     "ls_cust_new",
		SubscriptionID: "ls_sub_new",
		Amount:         99.99,
		Currency:       "USD",
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify new subscription created
	sub, err := svc.GetSubscription(context.Background(), 999)
	if err != nil {
		t.Fatalf("subscription not created: %v", err)
	}

	if sub.PlanID != 2 {
		t.Errorf("expected plan ID 2, got %d", sub.PlanID)
	}
	if sub.SeatCount != 5 {
		t.Errorf("expected 5 seats, got %d", sub.SeatCount)
	}
	if sub.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", sub.BillingCycle)
	}
	if sub.LemonSqueezyCustomerID == nil || *sub.LemonSqueezyCustomerID != "ls_cust_new" {
		t.Error("expected LemonSqueezy customer ID to be set")
	}
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_new" {
		t.Error("expected LemonSqueezy subscription ID to be set")
	}
}

// TestActivateSubscription_StripeProvider tests setting Stripe IDs
func TestActivateSubscription_StripeProvider(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)

	db.Create(&billing.PaymentOrder{
		OrganizationID:  998, // New org
		OrderNo:         "ORD-STRIPE-001",
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
		EventID:        "evt_stripe_new_sub",
		EventType:      "checkout.session.completed",
		Provider:       billing.PaymentProviderStripe,
		OrderNo:        "ORD-STRIPE-001",
		CustomerID:     "cus_stripe_new",
		SubscriptionID: "sub_stripe_new",
		Amount:         19.99,
		Currency:       "USD",
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 998)
	if sub.StripeCustomerID == nil || *sub.StripeCustomerID != "cus_stripe_new" {
		t.Error("expected Stripe customer ID to be set")
	}
	if sub.StripeSubscriptionID == nil || *sub.StripeSubscriptionID != "sub_stripe_new" {
		t.Error("expected Stripe subscription ID to be set")
	}
}

// TestUpgradePlan_NilPlanID tests upgrade with nil plan ID
func TestUpgradePlan_NilPlanID(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	expiresAt := now.Add(time.Hour)

	// Create order without plan ID
	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-UPGRADE-NIL",
		OrderType:       billing.OrderTypePlanUpgrade,
		PlanID:          nil, // No plan ID
		Amount:          79.99,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		ExpiresAt:       &expiresAt,
	})

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
	event := &payment.WebhookEvent{
		EventID:   "evt_upgrade_nil",
		EventType: billing.WebhookEventLSPaymentSuccess,
		Provider:  billing.PaymentProviderLemonSqueezy,
		OrderNo:   "ORD-UPGRADE-NIL",
		Amount:    79.99,
		Currency:  "USD",
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != ErrInvalidPlan {
		t.Errorf("expected ErrInvalidPlan, got %v", err)
	}
}

// TestHandleSubscriptionCreated_AlreadySet tests when subscription ID already set
func TestHandleSubscriptionCreated_AlreadySet(t *testing.T) {
	svc, db := setupTestService(t)

	lsCustID := "ls_cust_existing"
	lsSubID := "ls_sub_existing"
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:             1,
		PlanID:                     2,
		Status:                     billing.SubscriptionStatusActive,
		BillingCycle:               billing.BillingCycleMonthly,
		CurrentPeriodStart:         now,
		CurrentPeriodEnd:           now.AddDate(0, 1, 0),
		SeatCount:                  1,
		LemonSqueezyCustomerID:     &lsCustID,
		LemonSqueezySubscriptionID: &lsSubID, // Already set
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_already_set",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_new_attempt",
		CustomerID:     lsCustID,
	}

	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify subscription ID was NOT changed (already set)
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if *sub.LemonSqueezySubscriptionID != lsSubID {
		t.Error("subscription ID should not have been changed")
	}
}

// TestRenewSubscriptionFromOrder_NotFound tests renewal with non-existent subscription
func TestRenewSubscriptionFromOrder_NotFound(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)

	db.Create(&billing.PaymentOrder{
		OrganizationID:  999, // Non-existent org
		OrderNo:         "ORD-RENEW-NOTFOUND",
		OrderType:       billing.OrderTypeRenewal,
		PlanID:          &planID,
		Amount:          19.99,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		ExpiresAt:       &expiresAt,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_renew_notfound",
		EventType: billing.WebhookEventLSPaymentSuccess,
		Provider:  billing.PaymentProviderLemonSqueezy,
		OrderNo:   "ORD-RENEW-NOTFOUND",
		Amount:    19.99,
		Currency:  "USD",
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

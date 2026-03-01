package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Payment Tests from Coverage 95
// ===========================================

// TestHandlePaymentSucceeded_NoOrderFound_95 tests when order is not found
func TestHandlePaymentSucceeded_NoOrderFound_95(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_no_order_no_sub_95",
		EventType: billing.WebhookEventLSPaymentSuccess,
		Provider:  billing.PaymentProviderLemonSqueezy,
		OrderNo:   "nonexistent_order_95",
		Amount:    19.99,
		Currency:  "USD",
	}

	// Should return error since order not found and no subscription_id
	err := svc.HandlePaymentSucceeded(c, event)
	if err == nil {
		t.Error("expected error when order not found")
	}
}

// TestHandlePaymentFailed_WithOrderNoNotFound_95 tests failed payment with order lookup
func TestHandlePaymentFailed_WithOrderNoNotFound_95(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:      "evt_fail_no_order_95",
		EventType:    billing.WebhookEventLSPaymentFailed,
		Provider:     billing.PaymentProviderLemonSqueezy,
		OrderNo:      "nonexistent_95",
		FailedReason: "Card declined",
	}

	// Should return nil when order not found
	err := svc.HandlePaymentFailed(c, event)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestActivateSubscription_UpdateExisting_95 tests updating existing subscription
func TestActivateSubscription_UpdateExisting_95(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(3) // enterprise
	expiresAt := now.Add(time.Hour)

	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-UPDATE-EXIST-95",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &planID,
		Seats:           3,
		BillingCycle:    billing.BillingCycleMonthly,
		Amount:          299.97,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		PaymentMethod:   func() *string { s := "card"; return &s }(),
		ExpiresAt:       &expiresAt,
	})

	// Create existing subscription
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
		SeatCount:          1,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_update_existing_95",
		EventType:      billing.WebhookEventLSPaymentSuccess,
		Provider:       billing.PaymentProviderLemonSqueezy,
		OrderNo:        "ORD-UPDATE-EXIST-95",
		CustomerID:     "ls_cust_update_95",
		SubscriptionID: "ls_sub_update_95",
		Amount:         299.97,
		Currency:       "USD",
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)

	// activateSubscription updates PlanID, BillingCycle, SeatCount and provider IDs
	// Verify the update happened - check LemonSqueezy IDs were set
	if sub.LemonSqueezyCustomerID == nil || *sub.LemonSqueezyCustomerID != "ls_cust_update_95" {
		t.Error("expected LemonSqueezy customer ID to be set")
	}
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_update_95" {
		t.Error("expected LemonSqueezy subscription ID to be set")
	}
}

// TestListPlansWithPrices_MixedAvailability_95 tests listing plans when some don't have prices
func TestListPlansWithPrices_MixedAvailability_95(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a plan with only USD price (no CNY)
	plan := &billing.SubscriptionPlan{
		Name:                "usd_only_95",
		DisplayName:         "USD Only Plan 95",
		PricePerSeatMonthly: 5.99,
		PricePerSeatYearly:  59.99,
		IsActive:            true,
	}
	db.Create(plan)

	// Add only USD price
	db.Create(&billing.PlanPrice{
		PlanID:       plan.ID,
		Currency:     billing.CurrencyUSD,
		PriceMonthly: 5.99,
		PriceYearly:  59.99,
	})

	// List plans with CNY - should exclude the USD-only plan
	plans, err := svc.ListPlansWithPrices(context.Background(), "CNY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that usd_only_95 plan is not included
	for _, p := range plans {
		if p.Plan.Name == "usd_only_95" {
			t.Error("expected USD-only plan to be excluded from CNY list")
		}
	}
}

// TestRenewSubscriptionFromOrder_YearlyCycle_95 tests renewal order with yearly cycle
func TestRenewSubscriptionFromOrder_YearlyCycle_95(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)

	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-RENEW-YEARLY-95",
		OrderType:       billing.OrderTypeRenewal,
		PlanID:          &planID,
		Amount:          199.99,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		ExpiresAt:       &expiresAt,
	})

	periodEnd := now.AddDate(0, 0, -1) // About to expire
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now.AddDate(-1, 0, 0),
		CurrentPeriodEnd:   periodEnd,
		SeatCount:          1,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_renew_yearly_95",
		EventType: billing.WebhookEventLSPaymentSuccess,
		Provider:  billing.PaymentProviderLemonSqueezy,
		OrderNo:   "ORD-RENEW-YEARLY-95",
		Amount:    199.99,
		Currency:  "USD",
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)

	// Should have period extended by 1 year
	expectedEnd := periodEnd.AddDate(1, 0, 0)
	if sub.CurrentPeriodEnd.Before(expectedEnd.AddDate(0, 0, -1)) {
		t.Error("expected period to be extended by 1 year")
	}
}

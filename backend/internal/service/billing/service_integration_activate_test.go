package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// TestIntegrationActivateNewSubscription tests activation of brand new subscription
func TestIntegrationActivateNewSubscription(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Get pro plan (no existing subscription)
	proPlan, _ := service.GetPlan(ctx, "pro")

	// Use org ID 999 to avoid existing subscriptions
	// 2. Create payment order
	paymentMethod := billing.PaymentMethodCard
	order := &billing.PaymentOrder{
		OrganizationID:  999,
		OrderNo:         "ORD-NEW-SUB-001",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &proPlan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           3,
		Amount:          proPlan.PricePerSeatMonthly * 3,
		ActualAmount:    proPlan.PricePerSeatMonthly * 3,
		PaymentProvider: billing.PaymentProviderStripe,
		PaymentMethod:   &paymentMethod,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	// 3. Simulate payment succeeded
	event := &payment.WebhookEvent{
		EventID:         "evt_new_sub",
		EventType:       "checkout.session.completed",
		Provider:        "mock",
		OrderNo:         "ORD-NEW-SUB-001",
		ExternalOrderNo: "mock_cs_new_sub",
		CustomerID:      "cus_new_999",
		SubscriptionID:  "sub_new_999",
		Amount:          proPlan.PricePerSeatMonthly * 3,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to activate new subscription: %v", err)
	}

	// 4. Verify new subscription created
	newSub, err := service.GetSubscription(ctx, 999)
	if err != nil {
		t.Fatalf("expected subscription to be created, got error: %v", err)
	}
	if newSub.PlanID != proPlan.ID {
		t.Errorf("expected plan ID %d, got %d", proPlan.ID, newSub.PlanID)
	}
	if newSub.SeatCount != 3 {
		t.Errorf("expected 3 seats, got %d", newSub.SeatCount)
	}
	if newSub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected active status, got %s", newSub.Status)
	}
	if newSub.StripeCustomerID == nil || *newSub.StripeCustomerID != "cus_new_999" {
		t.Error("expected Stripe customer ID to be set")
	}
	if newSub.StripeSubscriptionID == nil || *newSub.StripeSubscriptionID != "sub_new_999" {
		t.Error("expected Stripe subscription ID to be set")
	}
}

// TestIntegrationActivateNewSubscriptionYearly tests activation of yearly subscription
func TestIntegrationActivateNewSubscriptionYearly(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Get pro plan
	proPlan, _ := service.GetPlan(ctx, "pro")

	// 2. Create payment order for yearly subscription
	order := &billing.PaymentOrder{
		OrganizationID:  998,
		OrderNo:         "ORD-NEW-YEARLY-001",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &proPlan.ID,
		BillingCycle:    billing.BillingCycleYearly,
		Seats:           1,
		Amount:          proPlan.PricePerSeatYearly,
		ActualAmount:    proPlan.PricePerSeatYearly,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	// 3. Simulate payment succeeded
	event := &payment.WebhookEvent{
		EventID:         "evt_new_yearly",
		EventType:       "checkout.session.completed",
		Provider:        "mock",
		OrderNo:         "ORD-NEW-YEARLY-001",
		ExternalOrderNo: "mock_cs_new_yearly",
		Amount:          proPlan.PricePerSeatYearly,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to activate yearly subscription: %v", err)
	}

	// 4. Verify subscription period is 1 year
	newSub, _ := service.GetSubscription(ctx, 998)
	expectedEnd := newSub.CurrentPeriodStart.AddDate(1, 0, 0)
	if !newSub.CurrentPeriodEnd.Truncate(time.Hour).Equal(expectedEnd.Truncate(time.Hour)) {
		t.Error("expected period end to be 1 year from start")
	}
}

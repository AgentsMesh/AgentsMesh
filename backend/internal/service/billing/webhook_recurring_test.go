package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Recurring Payment Webhook Tests
// ===========================================

func TestHandleRecurringPaymentSuccess(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	seedProPlan(t, db)

	plan, _ := service.GetPlan(ctx, "based")
	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	stripeSubID := "sub_recurring"
	downgradePlan := "pro"
	nextCycle := billing.BillingCycleYearly
	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               plan.ID,
		Status:               billing.SubscriptionStatusFrozen,
		FrozenAt:             &now,
		StripeSubscriptionID: &stripeSubID,
		CurrentPeriodStart:   now.AddDate(0, -1, 0),
		CurrentPeriodEnd:     now,
		DowngradeToPlan:      &downgradePlan,
		NextBillingCycle:     &nextCycle,
	}
	db.Create(sub)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_recurring_success",
		EventType:      "invoice.paid",
		SubscriptionID: "sub_recurring",
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle recurring payment: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
	if sub.FrozenAt != nil {
		t.Error("expected FrozenAt to be nil")
	}
	if sub.PlanID != proPlan.ID {
		t.Errorf("expected plan to be upgraded to pro")
	}
	if sub.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected billing cycle yearly")
	}
	if sub.DowngradeToPlan != nil || sub.NextBillingCycle != nil {
		t.Error("expected pending changes to be cleared")
	}
}

func TestHandleRecurringPaymentSuccessYearly(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	stripeSubID := "sub_recurring_yearly"
	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               plan.ID,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleYearly,
		StripeSubscriptionID: &stripeSubID,
		CurrentPeriodStart:   now.AddDate(-1, 0, 0),
		CurrentPeriodEnd:     now,
	}
	db.Create(sub)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_recurring_yearly",
		EventType:      "invoice.paid",
		SubscriptionID: "sub_recurring_yearly",
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle recurring payment: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	expectedEnd := sub.CurrentPeriodStart.AddDate(1, 0, 0)
	if !sub.CurrentPeriodEnd.Truncate(time.Second).Equal(expectedEnd.Truncate(time.Second)) {
		t.Error("expected period to be extended by 1 year")
	}
}

func TestHandleRecurringPaymentFailure(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	stripeSubID := "sub_fail_recurring"
	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               plan.ID,
		Status:               billing.SubscriptionStatusActive,
		StripeSubscriptionID: &stripeSubID,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_recurring_fail",
		EventType:      "invoice.payment_failed",
		SubscriptionID: "sub_fail_recurring",
	}

	err := service.HandlePaymentFailed(c, event)
	if err != nil {
		t.Fatalf("failed to handle recurring payment failure: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusFrozen {
		t.Errorf("expected status frozen, got %s", sub.Status)
	}
	if sub.FrozenAt == nil {
		t.Error("expected FrozenAt to be set")
	}
}

func TestActivateSubscriptionWithStripeIDs(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-STRIPE-IDS",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &plan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          0,
		ActualAmount:    0,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_stripe_ids",
		EventType:      "checkout.session.completed",
		OrderNo:        "ORD-STRIPE-IDS",
		CustomerID:     "cus_test123",
		SubscriptionID: "sub_test456",
		Amount:         0,
		Currency:       "USD",
		Status:         billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.StripeCustomerID == nil || *sub.StripeCustomerID != "cus_test123" {
		t.Error("expected Stripe customer ID to be set")
	}
	if sub.StripeSubscriptionID == nil || *sub.StripeSubscriptionID != "sub_test456" {
		t.Error("expected Stripe subscription ID to be set")
	}
}

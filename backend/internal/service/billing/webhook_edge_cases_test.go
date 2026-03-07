package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Webhook Edge Case Tests
// ===========================================

func TestHandlePaymentSucceededWithYearlyBilling(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db)
	plan, _ := service.GetPlan(ctx, "pro")

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-YEARLY",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &plan.ID,
		BillingCycle:    billing.BillingCycleYearly,
		Seats:           2,
		Amount:          399.80,
		ActualAmount:    399.80,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_pay_yearly",
		EventType: "checkout.session.completed",
		OrderNo:   "ORD-YEARLY",
		Amount:    399.80,
		Currency:  "USD",
		Status:    billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly billing, got %s", sub.BillingCycle)
	}
	expectedEnd := sub.CurrentPeriodStart.AddDate(1, 0, 0)
	if !sub.CurrentPeriodEnd.Truncate(time.Second).Equal(expectedEnd.Truncate(time.Second)) {
		t.Error("expected 1 year period for yearly billing")
	}
}

func TestUpgradePlanWithNilPlanID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-NILPLAN",
		OrderType:       billing.OrderTypePlanUpgrade,
		PlanID:          nil,
		Amount:          19.99,
		ActualAmount:    19.99,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_pay_nilplan",
		EventType: "checkout.session.completed",
		OrderNo:   "ORD-NILPLAN",
		Amount:    19.99,
		Currency:  "USD",
		Status:    billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != ErrInvalidPlan {
		t.Errorf("expected ErrInvalidPlan, got %v", err)
	}
}

func TestHandlePaymentSucceededNoOrder(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_pay_no_order",
		EventType: "checkout.session.completed",
		OrderNo:   "ORD-NONEXISTENT",
		Amount:    19.99,
		Currency:  "USD",
		Status:    billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err == nil {
		t.Error("expected error for missing order")
	}
}

func TestHandlePaymentSucceededAlreadySucceeded(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")

	now := time.Now()
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-ALREADY-DONE",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &plan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          0,
		ActualAmount:    0,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusSucceeded,
		PaidAt:          &now,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_pay_already",
		EventType: "checkout.session.completed",
		OrderNo:   "ORD-ALREADY-DONE",
		Amount:    0,
		Currency:  "USD",
		Status:    billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Errorf("expected no error for already succeeded order, got %v", err)
	}
}

func TestHandlePaymentFailedNoOrder(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:      "evt_pay_fail_no_order",
		EventType:    "payment_intent.failed",
		OrderNo:      "nonexistent",
		Status:       billing.OrderStatusFailed,
		FailedReason: "Card declined",
	}

	err := service.HandlePaymentFailed(c, event)
	if err != nil {
		t.Errorf("expected no error for missing order, got %v", err)
	}
}

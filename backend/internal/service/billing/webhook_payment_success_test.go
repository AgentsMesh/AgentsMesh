package billing

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Payment Success Webhook Tests - Basic
// ===========================================

func TestHandlePaymentSucceeded(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-WH-001",
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
		EventID:   "evt_pay_001",
		EventType: "checkout.session.completed",
		OrderNo:   "ORD-WH-001",
		Amount:    0,
		Currency:  "USD",
		Status:    billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment succeeded: %v", err)
	}

	updatedOrder, _ := service.GetPaymentOrderByNo(ctx, "ORD-WH-001")
	if updatedOrder.Status != billing.OrderStatusSucceeded {
		t.Errorf("expected order status succeeded, got %s", updatedOrder.Status)
	}

	sub, err := service.GetSubscription(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get subscription: %v", err)
	}
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected subscription active, got %s", sub.Status)
	}
}

func TestHandlePaymentSucceededByExternalOrderNo(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")

	extNo := "ext_order_123"
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-WH-002",
		ExternalOrderNo: &extNo,
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
		EventID:         "evt_pay_002",
		EventType:       "checkout.session.completed",
		ExternalOrderNo: "ext_order_123",
		Amount:          0,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment succeeded: %v", err)
	}
}

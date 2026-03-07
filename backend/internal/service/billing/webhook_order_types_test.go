package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Order Type Specific Payment Tests
// ===========================================

func TestHandlePaymentSucceededSeatPurchase(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	seedProPlan(t, db)
	// Use pro plan (MaxUsers=50) instead of based (MaxUsers=1) so seat purchase is allowed
	service.CreateSubscription(ctx, 1, "pro")

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-SEAT-001",
		OrderType:       billing.OrderTypeSeatPurchase,
		Seats:           3,
		Amount:          59.97,
		ActualAmount:    59.97,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_pay_seat",
		EventType: "checkout.session.completed",
		OrderNo:   "ORD-SEAT-001",
		Amount:    59.97,
		Currency:  "USD",
		Status:    billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle seat purchase: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.SeatCount != 4 {
		t.Errorf("expected 4 seats, got %d", sub.SeatCount)
	}
}

func TestHandlePaymentSucceededPlanUpgrade(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	proPlan := seedProPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-UPGRADE-001",
		OrderType:       billing.OrderTypePlanUpgrade,
		PlanID:          &proPlan.ID,
		Amount:          19.99,
		ActualAmount:    19.99,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_pay_upgrade",
		EventType: "checkout.session.completed",
		OrderNo:   "ORD-UPGRADE-001",
		Amount:    19.99,
		Currency:  "USD",
		Status:    billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle plan upgrade: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.PlanID != proPlan.ID {
		t.Errorf("expected plan ID %d, got %d", proPlan.ID, sub.PlanID)
	}
}

func TestHandlePaymentSucceededRenewal(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	sub, _ := service.GetSubscription(ctx, 1)
	originalEnd := sub.CurrentPeriodEnd

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-RENEW-001",
		OrderType:       billing.OrderTypeRenewal,
		Amount:          0,
		ActualAmount:    0,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_pay_renew",
		EventType: "checkout.session.completed",
		OrderNo:   "ORD-RENEW-001",
		Amount:    0,
		Currency:  "USD",
		Status:    billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle renewal: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if !sub.CurrentPeriodStart.Truncate(time.Second).Equal(originalEnd.Truncate(time.Second)) {
		t.Error("expected period to be renewed")
	}
}

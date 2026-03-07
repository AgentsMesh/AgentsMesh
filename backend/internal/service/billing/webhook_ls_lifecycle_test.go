package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// LemonSqueezy Subscription Created Tests
// ===========================================

func TestHandleSubscriptionCreated(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	lsCustomerID := "ls_cust_123"
	sub := &billing.Subscription{
		OrganizationID:         1,
		PlanID:                 plan.ID,
		Status:                 billing.SubscriptionStatusActive,
		LemonSqueezyCustomerID: &lsCustomerID,
		CurrentPeriodStart:     now,
		CurrentPeriodEnd:       now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_ls_sub_created",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_456",
		CustomerID:     "ls_cust_123",
	}

	err := service.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Fatalf("failed to handle subscription created: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_456" {
		t.Error("expected LemonSqueezy subscription ID to be set")
	}
}

func TestHandleSubscriptionCreatedByOrderNo(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-LS-001",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &plan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          9.9,
		ActualAmount:    9.9,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		Status:          billing.OrderStatusSucceeded,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_ls_sub_order",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_789",
		CustomerID:     "ls_cust_new",
		OrderNo:        "ORD-LS-001",
	}

	err := service.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Fatalf("failed to handle subscription created: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_789" {
		t.Error("expected LemonSqueezy subscription ID to be set")
	}
}

func TestHandleSubscriptionCreatedNoSubscriptionID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_ls_no_sub",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "",
	}

	err := service.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Errorf("expected no error for empty subscription ID, got %v", err)
	}
}

package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// LemonSqueezy Activation Tests
// ===========================================

func TestActivateSubscriptionWithLemonSqueezyIDs(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-LS-ACTIVATE",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &plan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          9.9,
		ActualAmount:    9.9,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_ls_activate",
		EventType:      billing.WebhookEventLSOrderCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		OrderNo:        "ORD-LS-ACTIVATE",
		CustomerID:     "ls_cust_activate",
		SubscriptionID: "ls_sub_activate",
		Amount:         9.9,
		Currency:       "USD",
		Status:         billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.LemonSqueezyCustomerID == nil || *sub.LemonSqueezyCustomerID != "ls_cust_activate" {
		t.Error("expected LemonSqueezy customer ID to be set")
	}
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_activate" {
		t.Error("expected LemonSqueezy subscription ID to be set")
	}
}

func TestUpdateExistingSubscriptionWithLemonSqueezyIDs(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")

	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-LS-UPDATE",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &plan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          9.9,
		ActualAmount:    9.9,
		PaymentProvider: billing.PaymentProviderLemonSqueezy,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_ls_update_ids",
		EventType:      billing.WebhookEventLSOrderCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		OrderNo:        "ORD-LS-UPDATE",
		CustomerID:     "ls_cust_update_ids",
		SubscriptionID: "ls_sub_update_ids",
		Amount:         9.9,
		Currency:       "USD",
		Status:         billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.LemonSqueezyCustomerID == nil || *sub.LemonSqueezyCustomerID != "ls_cust_update_ids" {
		t.Error("expected LemonSqueezy customer ID to be set")
	}
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_update_ids" {
		t.Error("expected LemonSqueezy subscription ID to be set")
	}
}

func TestFindSubscriptionByProviderIDLemonSqueezy(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	lsSubID := "ls_sub_find"
	sub := &billing.Subscription{
		OrganizationID:             1,
		PlanID:                     plan.ID,
		Status:                     billing.SubscriptionStatusActive,
		LemonSqueezySubscriptionID: &lsSubID,
		CurrentPeriodStart:         now,
		CurrentPeriodEnd:           now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_ls_find",
		EventType:      billing.WebhookEventLSSubscriptionCancelled,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_find",
	}

	err := service.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Fatalf("failed to handle cancellation: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected status canceled, got %s", sub.Status)
	}
}

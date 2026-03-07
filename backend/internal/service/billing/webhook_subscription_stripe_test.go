package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Stripe Subscription Canceled Tests
// ===========================================

func TestHandleSubscriptionCanceled(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	stripeSubID := "sub_stripe_cancel"
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
		EventID:        "evt_sub_cancel",
		EventType:      "customer.subscription.deleted",
		SubscriptionID: "sub_stripe_cancel",
	}

	err := service.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Fatalf("failed to handle subscription canceled: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected status canceled, got %s", sub.Status)
	}
	if sub.CanceledAt == nil {
		t.Error("expected CanceledAt to be set")
	}
}

func TestHandleSubscriptionCanceledEmptyID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_cancel_empty",
		EventType:      "customer.subscription.deleted",
		SubscriptionID: "",
	}

	err := service.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Errorf("expected no error for empty subscription ID, got %v", err)
	}
}

func TestHandleSubscriptionCanceledNoSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_cancel_none",
		EventType:      "customer.subscription.deleted",
		SubscriptionID: "sub_nonexistent",
	}

	err := service.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Errorf("expected no error for missing subscription, got %v", err)
	}
}

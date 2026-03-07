package billing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Stripe Subscription Updated Tests
// ===========================================

func TestHandleSubscriptionUpdated(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	stripeSubID := "sub_stripe_update"
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

	statuses := []struct {
		stripeStatus   string
		expectedStatus string
	}{
		{"active", billing.SubscriptionStatusActive},
		{"past_due", billing.SubscriptionStatusPastDue},
		{"canceled", billing.SubscriptionStatusCanceled},
		{"trialing", billing.SubscriptionStatusTrialing},
		{"paused", billing.SubscriptionStatusPaused},
		{"expired", billing.SubscriptionStatusExpired},
	}

	for i, test := range statuses {
		event := &payment.WebhookEvent{
			EventID:        fmt.Sprintf("evt_sub_update_%d", i),
			EventType:      "customer.subscription.updated",
			SubscriptionID: "sub_stripe_update",
			Status:         test.stripeStatus,
		}

		err := service.HandleSubscriptionUpdated(c, event)
		if err != nil {
			t.Fatalf("failed to handle subscription updated: %v", err)
		}

		sub, _ = service.GetSubscription(ctx, 1)
		if sub.Status != test.expectedStatus {
			t.Errorf("expected status %s, got %s", test.expectedStatus, sub.Status)
		}
	}
}

func TestHandleSubscriptionUpdatedEmptyID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_update_empty",
		EventType:      "customer.subscription.updated",
		SubscriptionID: "",
	}

	err := service.HandleSubscriptionUpdated(c, event)
	if err != nil {
		t.Errorf("expected no error for empty subscription ID, got %v", err)
	}
}

func TestHandleSubscriptionUpdatedNoSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_update_none",
		EventType:      "customer.subscription.updated",
		SubscriptionID: "sub_nonexistent",
		Status:         "active",
	}

	err := service.HandleSubscriptionUpdated(c, event)
	if err != nil {
		t.Errorf("expected no error for missing subscription, got %v", err)
	}
}

func TestHandleSubscriptionUpdatedClearsFreeze(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	stripeSubID := "sub_freeze_clear"
	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               plan.ID,
		Status:               billing.SubscriptionStatusFrozen,
		FrozenAt:             &now,
		StripeSubscriptionID: &stripeSubID,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_sub_clear_freeze",
		EventType:      "customer.subscription.updated",
		SubscriptionID: "sub_freeze_clear",
		Status:         "active",
	}

	err := service.HandleSubscriptionUpdated(c, event)
	if err != nil {
		t.Fatalf("failed to handle subscription updated: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
	if sub.FrozenAt != nil {
		t.Error("expected FrozenAt to be cleared")
	}
}

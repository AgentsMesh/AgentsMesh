package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// LemonSqueezy Subscription Pause/Resume Tests
// ===========================================

func TestHandleSubscriptionPaused(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	lsSubID := "ls_sub_pause"
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
		EventID:        "evt_ls_paused",
		EventType:      billing.WebhookEventLSSubscriptionPaused,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_pause",
	}

	err := service.HandleSubscriptionPaused(c, event)
	if err != nil {
		t.Fatalf("failed to handle subscription paused: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusPaused {
		t.Errorf("expected status paused, got %s", sub.Status)
	}
	// Paused is user-initiated; FrozenAt should NOT be set (reserved for payment failure)
	if sub.FrozenAt != nil {
		t.Error("expected FrozenAt to be nil for paused subscription (not frozen)")
	}
}

func TestHandleSubscriptionResumed(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	lsSubID := "ls_sub_resume"
	sub := &billing.Subscription{
		OrganizationID:             1,
		PlanID:                     plan.ID,
		Status:                     billing.SubscriptionStatusPaused,
		FrozenAt:                   &now,
		LemonSqueezySubscriptionID: &lsSubID,
		CurrentPeriodStart:         now,
		CurrentPeriodEnd:           now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_ls_resumed",
		EventType:      billing.WebhookEventLSSubscriptionResumed,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_resume",
	}

	err := service.HandleSubscriptionResumed(c, event)
	if err != nil {
		t.Fatalf("failed to handle subscription resumed: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
	if sub.FrozenAt != nil {
		t.Error("expected FrozenAt to be nil")
	}
}

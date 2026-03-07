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
// LemonSqueezy Status Update Tests
// ===========================================

func TestHandleSubscriptionExpired(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	lsSubID := "ls_sub_expire"
	sub := &billing.Subscription{
		OrganizationID:             1,
		PlanID:                     plan.ID,
		Status:                     billing.SubscriptionStatusActive,
		LemonSqueezySubscriptionID: &lsSubID,
		CurrentPeriodStart:         now.AddDate(0, -1, 0),
		CurrentPeriodEnd:           now,
	}
	db.Create(sub)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_ls_expired",
		EventType:      billing.WebhookEventLSSubscriptionExpired,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_expire",
	}

	err := service.HandleSubscriptionExpired(c, event)
	if err != nil {
		t.Fatalf("failed to handle subscription expired: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusExpired {
		t.Errorf("expected status expired, got %s", sub.Status)
	}
	// Expired is a natural end, not a user cancellation; CanceledAt should NOT be set
	if sub.CanceledAt != nil {
		t.Error("expected CanceledAt to be nil for expired subscription (not canceled)")
	}
}

func TestHandleSubscriptionUpdatedLemonSqueezy(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	lsSubID := "ls_sub_update"
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

	// LemonSqueezy status mappings (unpaid -> frozen per MapLSStatusToInternal)
	statuses := []struct {
		lsStatus       string
		expectedStatus string
	}{
		{"active", billing.SubscriptionStatusActive},
		{"on_trial", billing.SubscriptionStatusTrialing},
		{"paused", billing.SubscriptionStatusPaused},
		{"past_due", billing.SubscriptionStatusPastDue},
		{"unpaid", billing.SubscriptionStatusFrozen},
		{"cancelled", billing.SubscriptionStatusCanceled},
		{"expired", billing.SubscriptionStatusExpired},
	}

	for i, test := range statuses {
		event := &payment.WebhookEvent{
			EventID:        fmt.Sprintf("evt_ls_update_%d", i),
			EventType:      billing.WebhookEventLSSubscriptionUpdated,
			Provider:       billing.PaymentProviderLemonSqueezy,
			SubscriptionID: "ls_sub_update",
			Status:         test.lsStatus,
		}

		err := service.HandleSubscriptionUpdated(c, event)
		if err != nil {
			t.Fatalf("failed to handle subscription updated: %v", err)
		}

		sub, _ = service.GetSubscription(ctx, 1)
		if sub.Status != test.expectedStatus {
			t.Errorf("for LS status %s: expected %s, got %s",
				test.lsStatus, test.expectedStatus, sub.Status)
		}
	}
}

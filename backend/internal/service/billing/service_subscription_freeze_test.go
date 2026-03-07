package billing

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Freeze/Unfreeze Subscription Tests
// ===========================================

func TestFreezeSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	err := service.FreezeSubscription(ctx, 1)
	if err != nil {
		t.Fatalf("failed to freeze subscription: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusFrozen {
		t.Errorf("expected status frozen, got %s", sub.Status)
	}
	if sub.FrozenAt == nil {
		t.Error("expected FrozenAt to be set")
	}
}

func TestUnfreezeSubscriptionMonthly(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")
	service.FreezeSubscription(ctx, 1)

	err := service.UnfreezeSubscription(ctx, 1, billing.BillingCycleMonthly)
	if err != nil {
		t.Fatalf("failed to unfreeze subscription: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
	if sub.BillingCycle != billing.BillingCycleMonthly {
		t.Errorf("expected billing cycle monthly, got %s", sub.BillingCycle)
	}
}

func TestUnfreezeSubscriptionYearly(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")
	service.FreezeSubscription(ctx, 1)

	err := service.UnfreezeSubscription(ctx, 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("failed to unfreeze subscription: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
	if sub.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected billing cycle yearly, got %s", sub.BillingCycle)
	}
}

func TestUnfreezeSubscriptionNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	err := service.UnfreezeSubscription(ctx, 999, billing.BillingCycleMonthly)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

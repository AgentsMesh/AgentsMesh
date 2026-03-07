package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Subscription Renewal Tests
// ===========================================

func TestRenewSubscriptionMonthly(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	sub, _ := service.GetSubscription(ctx, 1)
	originalEnd := sub.CurrentPeriodEnd

	err := service.RenewSubscription(ctx, 1)
	if err != nil {
		t.Fatalf("failed to renew subscription: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if !sub.CurrentPeriodStart.Equal(originalEnd) {
		t.Error("expected new period start to equal old period end")
	}
}

func TestRenewSubscriptionYearly(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	now := time.Now()
	plan, _ := service.GetPlan(ctx, "based")
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
	}
	db.Create(sub)

	originalEnd := sub.CurrentPeriodEnd

	err := service.RenewSubscription(ctx, 1)
	if err != nil {
		t.Fatalf("failed to renew subscription: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	expectedEnd := originalEnd.AddDate(1, 0, 0)
	if !sub.CurrentPeriodEnd.Truncate(time.Second).Equal(expectedEnd.Truncate(time.Second)) {
		t.Errorf("expected yearly renewal, got %v vs %v", sub.CurrentPeriodEnd, expectedEnd)
	}
}

func TestRenewSubscriptionNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	err := service.RenewSubscription(ctx, 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

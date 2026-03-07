package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Basic Subscription Tests
// ===========================================

func TestGetSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	plan := seedTestPlan(t, db)

	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	result, err := service.GetSubscription(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get subscription: %v", err)
	}
	if result.OrganizationID != 1 {
		t.Errorf("expected org ID 1, got %d", result.OrganizationID)
	}
}

func TestGetSubscriptionNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.GetSubscription(ctx, 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestCreateSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	sub, err := service.CreateSubscription(ctx, 1, "based")
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}
	if sub.OrganizationID != 1 {
		t.Errorf("expected org ID 1, got %d", sub.OrganizationID)
	}
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
}

func TestCreateSubscriptionPlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.CreateSubscription(ctx, 1, "nonexistent")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestCancelSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	err := service.CancelSubscription(ctx, 1)
	if err != nil {
		t.Fatalf("failed to cancel subscription: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected status canceled, got %s", sub.Status)
	}
}

func TestCancelSubscriptionNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	err := service.CancelSubscription(ctx, 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestSetCancelAtPeriodEnd(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	err := service.SetCancelAtPeriodEnd(ctx, 1, true)
	if err != nil {
		t.Fatalf("failed to set cancel at period end: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if !sub.CancelAtPeriodEnd {
		t.Error("expected CancelAtPeriodEnd to be true")
	}
}

func TestSetNextBillingCycle(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	err := service.SetNextBillingCycle(ctx, 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("failed to set next billing cycle: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.NextBillingCycle == nil || *sub.NextBillingCycle != billing.BillingCycleYearly {
		t.Error("expected NextBillingCycle to be yearly")
	}
}

func TestSetAutoRenew(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	// Enable auto-renew
	err := service.SetAutoRenew(ctx, 1, true)
	if err != nil {
		t.Fatalf("failed to set auto-renew: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if !sub.AutoRenew {
		t.Error("expected AutoRenew to be true")
	}

	// Disable auto-renew
	err = service.SetAutoRenew(ctx, 1, false)
	if err != nil {
		t.Fatalf("failed to disable auto-renew: %v", err)
	}

	sub, _ = service.GetSubscription(ctx, 1)
	if sub.AutoRenew {
		t.Error("expected AutoRenew to be false")
	}
}

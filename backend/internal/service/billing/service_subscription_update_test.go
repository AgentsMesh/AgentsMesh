package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Subscription Update/Upgrade/Downgrade Tests
// ===========================================

func TestUpdateSubscriptionUpgrade(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	seedProPlan(t, db)

	service.CreateSubscription(ctx, 1, "based")

	// Upgrade from based to pro requires payment, so plan should not change immediately
	// UpdateSubscription should return current subscription (payment handled via checkout)
	sub, err := service.UpdateSubscription(ctx, 1, "pro")
	if err != nil {
		t.Fatalf("failed to update subscription: %v", err)
	}
	// Since based is a paid plan ($9.9), upgrading to pro requires payment
	// The subscription should remain on based until payment is completed
	if sub.Plan.Name != "based" {
		t.Errorf("expected plan to remain 'based' until payment, got %s", sub.Plan.Name)
	}
}

func TestUpdateSubscriptionDowngrade(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	seedProPlan(t, db)

	// Create pro subscription first
	plan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          1, // Within based plan limit
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// Downgrade to free - should schedule for period end
	result, err := service.UpdateSubscription(ctx, 1, "based")
	if err != nil {
		t.Fatalf("failed to schedule downgrade: %v", err)
	}
	if result.DowngradeToPlan == nil || *result.DowngradeToPlan != "based" {
		t.Error("expected downgrade to be scheduled")
	}
}

func TestUpdateSubscriptionDowngradeSeatExceeds(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db) // Free plan max_users = 5
	seedProPlan(t, db)

	// Create pro subscription with too many seats
	plan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          10, // Exceeds based plan limit of 5
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// Downgrade to free - should fail
	_, err := service.UpdateSubscription(ctx, 1, "based")
	if err != ErrSeatCountExceedsLimit {
		t.Errorf("expected ErrSeatCountExceedsLimit, got %v", err)
	}
}

func TestUpdateSubscriptionPaidToPaid(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db)
	seedEnterprisePlan(t, db)

	// Create pro subscription
	plan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          1,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// Upgrade from pro to enterprise (paid to paid)
	// Should just return current subscription (payment flow handles upgrade)
	result, err := service.UpdateSubscription(ctx, 1, "enterprise")
	if err != nil {
		t.Fatalf("failed to update subscription: %v", err)
	}
	// Should still be pro - upgrade requires payment flow
	if result.Plan.Name != "pro" {
		t.Errorf("expected plan 'pro', got %s", result.Plan.Name)
	}
}

func TestUpdateSubscriptionNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	_, err := service.UpdateSubscription(ctx, 999, "based")
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestUpdateSubscriptionInvalidNewPlan(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	_, err := service.UpdateSubscription(ctx, 1, "nonexistent")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestUpdateSubscriptionSamePlan(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	// Update to same plan
	sub, err := service.UpdateSubscription(ctx, 1, "based")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.Plan.Name != "based" {
		t.Errorf("expected plan 'based', got %s", sub.Plan.Name)
	}
}

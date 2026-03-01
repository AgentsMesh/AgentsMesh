package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Subscription Update/Cancel Tests
// ===========================================

// TestUpdateSubscription_DowngradeSeatExceedsLimit tests downgrade with seat count exceeding limit
func TestUpdateSubscription_DowngradeSeatExceedsLimit(t *testing.T) {
	svc, db := setupTestService(t)

	// Seed free plan
	freePlan := seedFreePlan(t, db)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro plan
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          10, // More than free plan's max_users (1)
	})

	// Downgrade from pro to free - should fail due to seat count exceeding limit
	_, err := svc.UpdateSubscription(context.Background(), 1, freePlan.Name)
	if err != ErrSeatCountExceedsLimit {
		t.Errorf("expected ErrSeatCountExceedsLimit, got %v", err)
	}
}

// TestUpdateSubscription_DowngradeScheduled tests scheduled downgrade (not exceeding limit)
func TestUpdateSubscription_DowngradeScheduled(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro plan
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1, // Low seat count, won't exceed based plan limit
	})

	// Downgrade from pro to based - should be scheduled
	sub, err := svc.UpdateSubscription(context.Background(), 1, "based")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sub.DowngradeToPlan == nil || *sub.DowngradeToPlan != "based" {
		t.Error("expected downgrade to be scheduled")
	}
}

// TestUpdateSubscription_PaidUpgradeReturnsCurrent tests paid upgrade (returns current plan)
func TestUpdateSubscription_PaidUpgradeReturnsCurrent(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based plan (paid)
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Upgrade from based to pro (both paid) - returns current plan (payment flow handles actual upgrade)
	sub, err := svc.UpdateSubscription(context.Background(), 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// For paid upgrades, plan is NOT changed immediately - payment flow handles it
	// The returned subscription still has the current plan
	if sub.Plan == nil || sub.Plan.Name != "based" {
		t.Errorf("expected current plan (based), got %v", sub.Plan.Name)
	}
}

// TestUpdateSubscription_NotFoundPlan tests update to nonexistent plan
func TestUpdateSubscription_NotFoundPlan(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	_, err := svc.UpdateSubscription(context.Background(), 1, "nonexistent")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

// TestCancelSubscription_NoSub tests cancelling nonexistent subscription
func TestCancelSubscription_NoSub(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.CancelSubscription(context.Background(), 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestCancelSubscription_Active tests cancelling active subscription
func TestCancelSubscription_Active(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	err := svc.CancelSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected canceled status, got %s", sub.Status)
	}
}

// TestUnfreezeSubscription_NotFound tests unfreezing without subscription
func TestUnfreezeSubscription_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.UnfreezeSubscription(context.Background(), 999, billing.BillingCycleMonthly)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestActivateTrialSubscription_NotFound tests activating trial without subscription
func TestActivateTrialSubscription_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.ActivateTrialSubscription(context.Background(), 999, billing.BillingCycleMonthly)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestCreateTrialSubscription_CustomDays tests trial with custom days
func TestCreateTrialSubscription_CustomDays(t *testing.T) {
	svc, _ := setupTestService(t)

	// Create with 30 days trial
	sub, err := svc.CreateTrialSubscription(context.Background(), 1, "pro", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check trial period is ~30 days
	duration := sub.CurrentPeriodEnd.Sub(sub.CurrentPeriodStart)
	if duration.Hours() < 29*24 || duration.Hours() > 31*24 {
		t.Errorf("expected ~30 days trial, got %v", duration)
	}
}

// TestCreateTrialSubscription_ZeroDays tests trial with zero days (should use default 30 days)
func TestCreateTrialSubscription_ZeroDays(t *testing.T) {
	svc, _ := setupTestService(t)

	// Create with 0 days - should use default 30 days
	sub, err := svc.CreateTrialSubscription(context.Background(), 2, "pro", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check trial period is ~30 days (default)
	duration := sub.CurrentPeriodEnd.Sub(sub.CurrentPeriodStart)
	if duration.Hours() < 29*24 || duration.Hours() > 31*24 {
		t.Errorf("expected ~30 days trial (default), got %v", duration)
	}
}

// TestGetBillingOverview_PlanLoadedFromDB tests billing overview when plan is not preloaded
func TestGetBillingOverview_PlanLoadedFromDB(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	// Create subscription without preloading plan
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          3,
	}
	db.Create(sub)

	overview, err := svc.GetBillingOverview(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if overview.Plan == nil {
		t.Error("expected plan to be loaded")
	}
	if overview.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected active status, got %s", overview.Status)
	}
}

// TestGetBillingOverview_NoSubscription tests billing overview when subscription not found
func TestGetBillingOverview_NoSubscription(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.GetBillingOverview(context.Background(), 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

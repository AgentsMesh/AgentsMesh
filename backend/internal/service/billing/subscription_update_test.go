package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// TestUpdateSubscription_Downgrade tests downgrade scheduling
func TestUpdateSubscription_Downgrade(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a subscription on enterprise plan (ID=3)
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             3, // enterprise
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          5,
	})

	// Downgrade to pro (enterprise has higher price)
	sub, err := svc.UpdateSubscription(context.Background(), 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should schedule downgrade, not apply immediately
	if sub.DowngradeToPlan == nil {
		t.Error("expected downgrade to be scheduled")
	}
	if *sub.DowngradeToPlan != "pro" {
		t.Errorf("expected downgrade to pro, got %s", *sub.DowngradeToPlan)
	}
	// Plan should still be enterprise
	if sub.PlanID != 3 {
		t.Errorf("expected plan to still be enterprise (3), got %d", sub.PlanID)
	}
}

// TestUpdateSubscription_DowngradeExceedsSeatLimit tests downgrade when seat count exceeds limit
func TestUpdateSubscription_DowngradeExceedsSeatLimit(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a subscription on enterprise plan with many seats
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             3, // enterprise
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          100, // More than pro's max of 50
	})

	// Try to downgrade to pro
	_, err := svc.UpdateSubscription(context.Background(), 1, "pro")
	if err != ErrSeatCountExceedsLimit {
		t.Errorf("expected ErrSeatCountExceedsLimit, got %v", err)
	}
}

// TestUpdateSubscription_UpgradeFromBasedPlan tests upgrade from based plan
// Note: Based plan has non-zero price in test data, so upgrade doesn't apply immediately
// (requires payment to be processed first)
func TestUpdateSubscription_UpgradeFromBasedPlan(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a subscription on based plan (ID=1)
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based plan
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Upgrade to pro - this is an upgrade from paid to higher paid plan
	sub, err := svc.UpdateSubscription(context.Background(), 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Plan stays the same until payment is processed
	if sub.PlanID != 1 {
		t.Errorf("expected plan to stay based (1) until payment, got %d", sub.PlanID)
	}
}

// TestUpdateSubscription_PaidUpgrade tests upgrade between paid plans
func TestUpdateSubscription_PaidUpgrade(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a subscription on pro plan (ID=2)
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          5,
	})

	// Upgrade to enterprise
	sub, err := svc.UpdateSubscription(context.Background(), 1, "enterprise")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// For paid upgrades, plan stays the same until payment is processed
	if sub.PlanID != 2 {
		t.Errorf("expected plan to still be pro (2), got %d", sub.PlanID)
	}
}

// TestUpdateSubscription_NotFound tests updating non-existent subscription
func TestUpdateSubscription_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	// Try to update non-existent subscription
	_, err := svc.UpdateSubscription(context.Background(), 999, "pro")
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestUpdateSubscription_InvalidPlan tests updating with invalid plan
func TestUpdateSubscription_InvalidPlan(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a subscription
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Try to update with invalid plan
	_, err := svc.UpdateSubscription(context.Background(), 1, "nonexistent")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

// TestCancelSubscription_Success tests successful subscription cancellation
func TestCancelSubscription_Success(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a subscription
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

	// Cancel subscription
	err := svc.CancelSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status changed
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected status to be canceled, got %s", sub.Status)
	}
}

// TestCancelSubscription_NotFound tests canceling non-existent subscription
func TestCancelSubscription_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	// Try to cancel non-existent subscription
	err := svc.CancelSubscription(context.Background(), 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestUpdateSubscription_UpgradeFromFreePlan tests upgrade from free plan path
// Note: Due to GORM caching behavior in tests, we verify the returned Plan object
// instead of re-fetching from database
func TestUpdateSubscription_UpgradeFromFreePlan(t *testing.T) {
	svc, db := setupTestService(t)

	// Seed free plan
	freePlan := seedFreePlan(t, db)

	// Create a subscription on free plan
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             freePlan.ID, // free plan with price = 0
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Upgrade from free to based - should apply immediately
	sub, err := svc.UpdateSubscription(context.Background(), 1, "based")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that Plan reference is set to the new plan (based)
	if sub.Plan == nil {
		t.Error("expected Plan to be set")
	} else if sub.Plan.Name != "based" {
		t.Errorf("expected Plan.Name to be 'based', got %s", sub.Plan.Name)
	}
}

// TestUpdateSubscription_ScheduleDowngradeToFree tests scheduling downgrade to free plan
func TestUpdateSubscription_ScheduleDowngradeToFree(t *testing.T) {
	svc, db := setupTestService(t)

	// Seed free plan
	seedFreePlan(t, db)

	// Create a subscription on based plan
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based plan
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Downgrade to free plan - should schedule downgrade (free plan has lower price)
	sub, err := svc.UpdateSubscription(context.Background(), 1, "free")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should schedule downgrade since free plan has lower price
	if sub.DowngradeToPlan == nil {
		t.Error("expected downgrade to be scheduled")
	}
	if *sub.DowngradeToPlan != "free" {
		t.Errorf("expected downgrade to free, got %s", *sub.DowngradeToPlan)
	}
}

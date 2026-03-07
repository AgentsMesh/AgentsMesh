package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

func TestCalculateUpgradePrice(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)       // based plan ($9.9/month)
	seedProPlan(t, db)        // pro plan ($19.99/month)
	seedEnterprisePlan(t, db) // enterprise plan

	// Create a subscription halfway through the period
	basedPlan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	periodStart := now.Add(-15 * 24 * time.Hour) // Started 15 days ago
	periodEnd := now.Add(15 * 24 * time.Hour)    // Ends in 15 days

	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             basedPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          1,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
	}
	db.Create(sub)

	// Test upgrading from based ($9.9) to pro ($19.99)
	result, err := service.CalculateUpgradePrice(ctx, 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Full price is 19.99, should be approximately half (prorated)
	if !almostEqual(result.Amount, 19.99, 0.001) {
		t.Errorf("expected full amount 19.99, got %f", result.Amount)
	}
	// Price difference is 19.99 - 9.9 = 10.09
	// Prorated for half period: 10.09 * 0.5 = ~5.045
	if result.ActualAmount < 4 || result.ActualAmount > 7 {
		t.Errorf("expected prorated amount around 5, got %f", result.ActualAmount)
	}
}

func TestCalculateUpgradePriceYearlyCycle(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)       // based plan ($99/year)
	seedProPlan(t, db)        // pro plan ($199.90/year)

	basedPlan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	periodStart := now.Add(-180 * 24 * time.Hour) // 6 months ago
	periodEnd := now.Add(185 * 24 * time.Hour)    // 6 months left

	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             basedPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		SeatCount:          1,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
	}
	db.Create(sub)

	// Upgrade from based ($99/year) to pro ($199.90/year)
	result, err := service.CalculateUpgradePrice(ctx, 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Full price is 199.90
	if !almostEqual(result.Amount, 199.90, 0.001) {
		t.Errorf("expected full amount 199.90, got %f", result.Amount)
	}
	// Billing cycle should be yearly
	if result.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly billing cycle, got %s", result.BillingCycle)
	}
}

func TestCalculateUpgradePriceZeroSeats(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	seedProPlan(t, db)

	basedPlan, _ := service.GetPlan(ctx, "based")
	now := time.Now()

	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             basedPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          0, // Zero seats should default to 1
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	result, err := service.CalculateUpgradePrice(ctx, 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Seats != 1 {
		t.Errorf("expected seats to default to 1, got %d", result.Seats)
	}
}

func TestCalculateUpgradePriceDowngrade(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	seedProPlan(t, db)

	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()

	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          1,
		CurrentPeriodStart: now.Add(-15 * 24 * time.Hour),
		CurrentPeriodEnd:   now.Add(15 * 24 * time.Hour),
	}
	db.Create(sub)

	// Downgrade from pro to based
	result, err := service.CalculateUpgradePrice(ctx, 1, "based")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ActualAmount should be 0 for downgrade (handled by different flow)
	if result.ActualAmount != 0 {
		t.Errorf("expected ActualAmount 0 for downgrade, got %f", result.ActualAmount)
	}
}

func TestCalculateUpgradePriceNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.CalculateUpgradePrice(ctx, 999, "pro")
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestCalculateUpgradePricePlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	_, err := service.CalculateUpgradePrice(ctx, 1, "nonexistent")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

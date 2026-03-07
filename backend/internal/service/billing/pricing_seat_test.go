package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

func TestCalculateSeatPurchasePrice(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db)
	proPlan, _ := service.GetPlan(ctx, "pro")

	// Create a subscription with some time remaining
	now := time.Now()
	periodStart := now.Add(-10 * 24 * time.Hour)
	periodEnd := now.Add(20 * 24 * time.Hour) // 2/3 of period remaining

	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          1,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
	}
	db.Create(sub)

	// Test purchasing 2 additional seats
	result, err := service.CalculateSeatPurchasePrice(ctx, 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Full price for 2 seats: 19.99 * 2 = 39.98
	expectedFullPrice := 39.98
	if !almostEqual(result.Amount, expectedFullPrice, 0.001) {
		t.Errorf("expected full amount %f, got %f", expectedFullPrice, result.Amount)
	}

	// Prorated should be about 2/3 of full price
	if result.ActualAmount < 24 || result.ActualAmount > 30 {
		t.Errorf("expected prorated amount around 26.65, got %f", result.ActualAmount)
	}
}

func TestCalculateSeatPurchasePrice_BasedPlan(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	// Should fail for based plan (fixed 1 seat, cannot purchase more)
	_, err := service.CalculateSeatPurchasePrice(ctx, 1, 1)
	if err != ErrInvalidPlan {
		t.Errorf("expected ErrInvalidPlan, got %v", err)
	}
}

func TestCalculateSeatPurchasePrice_ExceedsMax(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db) // max_users = 50
	proPlan, _ := service.GetPlan(ctx, "pro")

	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          49, // Already have 49 seats
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// Try to add 2 seats (would exceed 50)
	_, err := service.CalculateSeatPurchasePrice(ctx, 1, 2)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestCalculateSeatPurchasePriceInvalidSeats(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.CalculateSeatPurchasePrice(ctx, 1, 0)
	if err != ErrInvalidPlan {
		t.Errorf("expected ErrInvalidPlan, got %v", err)
	}

	_, err = service.CalculateSeatPurchasePrice(ctx, 1, -1)
	if err != ErrInvalidPlan {
		t.Errorf("expected ErrInvalidPlan, got %v", err)
	}
}

func TestCalculateSeatPurchasePriceYearlyCycle(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db)
	proPlan, _ := service.GetPlan(ctx, "pro")

	now := time.Now()
	periodStart := now.Add(-182 * 24 * time.Hour)
	periodEnd := now.Add(183 * 24 * time.Hour)

	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		SeatCount:          1,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
	}
	db.Create(sub)

	result, err := service.CalculateSeatPurchasePrice(ctx, 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Full price for 2 seats yearly: 199.90 * 2 = 399.80
	if !almostEqual(result.Amount, 399.80, 0.001) {
		t.Errorf("expected full amount 399.80, got %f", result.Amount)
	}
}

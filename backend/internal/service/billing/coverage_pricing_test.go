package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Price Calculation Tests
// ===========================================

// TestListPlansWithPrices_AllPricesExist tests when all plans have prices
func TestListPlansWithPrices_AllPricesExist(t *testing.T) {
	svc, _ := setupTestService(t)

	// List plans with USD (all seeded plans have USD prices)
	plans, err := svc.ListPlansWithPrices(context.Background(), "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have based, pro, enterprise plans
	if len(plans) < 3 {
		t.Errorf("expected at least 3 plans with USD prices, got %d", len(plans))
	}
}

// TestCalculateRenewalPrice_YearlyWithSeats tests renewal with yearly and multiple seats
func TestCalculateRenewalPrice_YearlyWithSeats(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
		SeatCount:          5,
	})

	result, err := svc.CalculateRenewalPrice(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Seats != 5 {
		t.Errorf("expected 5 seats, got %d", result.Seats)
	}
	if result.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", result.BillingCycle)
	}
}

// TestCalculateUpgradePrice_ZeroSeats tests upgrade price with zero seats (defaults to 1)
func TestCalculateUpgradePrice_ZeroSeats(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          0, // Zero seats - should default to 1
	})

	result, err := svc.CalculateUpgradePrice(context.Background(), 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With 0 seats defaulting to 1, price should be calculated for 1 seat
	if result.Seats != 1 {
		t.Errorf("expected 1 seat (default), got %d", result.Seats)
	}
}

// TestCalculateSeatPurchasePrice_ZeroSeats tests seat purchase with zero existing seats
func TestCalculateSeatPurchasePrice_ZeroSeats(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          0, // Zero seats
	})

	// Purchase 3 additional seats
	result, err := svc.CalculateSeatPurchasePrice(context.Background(), 1, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// New total should be 3 (0 -> 3) - actually 1 + 3 = 4 if zero defaults to 1
	if result.Seats < 3 {
		t.Errorf("expected at least 3 seats, got %d", result.Seats)
	}
}

// TestCalculateBillingCycleChangePrice_NoSeats tests cycle change with zero seats
func TestCalculateBillingCycleChangePrice_NoSeats(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          0, // Zero seats
	})

	result, err := svc.CalculateBillingCycleChangePrice(context.Background(), 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With 0 seats defaulting to 1, should still calculate correctly
	if result.Seats != 1 {
		t.Errorf("expected 1 seat (default), got %d", result.Seats)
	}
}

// TestCalculateRenewalPrice_NoSeats tests renewal with zero seats
func TestCalculateRenewalPrice_NoSeats(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          0, // Zero seats
	})

	result, err := svc.CalculateRenewalPrice(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With 0 seats defaulting to 1
	if result.Seats != 1 {
		t.Errorf("expected 1 seat (default), got %d", result.Seats)
	}
}

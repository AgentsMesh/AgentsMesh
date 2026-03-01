package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Final Coverage Tests - Pricing Calculations
// ===========================================

// TestCalculateRenewalPrice_ZeroSeats tests renewal with zero seats
func TestCalculateRenewalPrice_ZeroSeats(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          0, // Edge case
	})

	result, err := svc.CalculateRenewalPrice(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Seats != 1 {
		t.Errorf("expected 1 seat (default), got %d", result.Seats)
	}
}

// TestCalculateBillingCycleChangePrice_SameCycle tests no change when same cycle
func TestCalculateBillingCycleChangePrice_SameCycle(t *testing.T) {
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

	result, err := svc.CalculateBillingCycleChangePrice(context.Background(), 1, billing.BillingCycleMonthly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Error("expected nil for same billing cycle")
	}
}

// TestCalculateBillingCycleChangePrice_YearlyToMonthly tests changing from yearly to monthly
func TestCalculateBillingCycleChangePrice_YearlyToMonthly(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
		SeatCount:          1,
	})

	result, err := svc.CalculateBillingCycleChangePrice(context.Background(), 1, billing.BillingCycleMonthly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Yearly to monthly is a downgrade, actual amount should be 0
	if result.ActualAmount != 0 {
		t.Errorf("expected 0 actual amount for downgrade, got %f", result.ActualAmount)
	}
}

// TestCalculateBillingCycleChangePrice_ZeroSeats tests cycle change with zero seats
func TestCalculateBillingCycleChangePrice_ZeroSeats(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          0, // Edge case
	})

	result, err := svc.CalculateBillingCycleChangePrice(context.Background(), 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Seats != 1 {
		t.Errorf("expected 1 seat (default), got %d", result.Seats)
	}
}

// TestCalculateRemainingPeriodRatio_ZeroPeriod tests when period is zero
func TestCalculateRemainingPeriodRatio_ZeroPeriod(t *testing.T) {
	now := time.Now()
	// Same start and end time = zero period
	ratio := calculateRemainingPeriodRatio(now, now)
	if ratio != 0 {
		t.Errorf("expected 0 for zero period, got %f", ratio)
	}
}

package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Upgrade Pricing Coverage Tests
// ===========================================

// TestCalculateUpgradePrice_NoSubscription tests upgrade pricing without subscription
func TestCalculateUpgradePrice_NoSubscription(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.CalculateUpgradePrice(context.Background(), 999, "enterprise")
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestCalculateUpgradePrice_InvalidNewPlan tests upgrade to invalid plan
func TestCalculateUpgradePrice_InvalidNewPlan(t *testing.T) {
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

	_, err := svc.CalculateUpgradePrice(context.Background(), 1, "nonexistent")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

// TestCalculateUpgradePrice_YearlyCycle tests upgrade pricing with yearly billing
func TestCalculateUpgradePrice_YearlyCycle(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
		SeatCount:          1,
	})

	price, err := svc.CalculateUpgradePrice(context.Background(), 1, "enterprise")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", price.BillingCycle)
	}
}

// TestCalculateUpgradePrice_WithZeroSeats tests upgrade with zero seats (should default to 1)
func TestCalculateUpgradePrice_WithZeroSeats(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          0, // Edge case
	})

	price, err := svc.CalculateUpgradePrice(context.Background(), 1, "enterprise")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.Seats != 1 {
		t.Errorf("expected 1 seat (default), got %d", price.Seats)
	}
}

// TestCalculateUpgradePrice_Downgrade tests upgrade calculation for downgrade (should return 0 actual amount)
func TestCalculateUpgradePrice_Downgrade(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             3, // enterprise
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	price, err := svc.CalculateUpgradePrice(context.Background(), 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Downgrade should have 0 actual amount
	if price.ActualAmount != 0 {
		t.Errorf("expected 0 actual amount for downgrade, got %f", price.ActualAmount)
	}
}

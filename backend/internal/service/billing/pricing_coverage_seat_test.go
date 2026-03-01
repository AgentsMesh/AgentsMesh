package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Seat Purchase Pricing Coverage Tests
// ===========================================

// TestCalculateSeatPurchasePrice_NoSubscription tests seat purchase without subscription
func TestCalculateSeatPurchasePrice_NoSubscription(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.CalculateSeatPurchasePrice(context.Background(), 999, 5)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestCalculateSeatPurchasePrice_YearlyCycle tests seat purchase with yearly billing
func TestCalculateSeatPurchasePrice_YearlyCycle(t *testing.T) {
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

	price, err := svc.CalculateSeatPurchasePrice(context.Background(), 1, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", price.BillingCycle)
	}
}

// TestCalculateSeatPurchasePrice_InvalidAdditionalSeats tests with zero or negative seats
func TestCalculateSeatPurchasePrice_InvalidAdditionalSeats(t *testing.T) {
	svc, db := setupTestService(t)

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

	_, err := svc.CalculateSeatPurchasePrice(context.Background(), 1, 0)
	if err != ErrInvalidPlan {
		t.Errorf("expected ErrInvalidPlan for zero seats, got %v", err)
	}

	_, err = svc.CalculateSeatPurchasePrice(context.Background(), 1, -1)
	if err != ErrInvalidPlan {
		t.Errorf("expected ErrInvalidPlan for negative seats, got %v", err)
	}
}

// TestCalculateSeatPurchasePrice_ExceedsMaxSeats tests seat purchase exceeding max
func TestCalculateSeatPurchasePrice_ExceedsMaxSeats(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro: max 50 users
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          45,
	})

	// Try to purchase 10 more (would exceed 50)
	_, err := svc.CalculateSeatPurchasePrice(context.Background(), 1, 10)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

package billing

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// almostEqual compares two floats with a tolerance for floating point precision issues
func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func TestCalculateSubscriptionPrice(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)    // based plan: 0/month
	seedProPlan(t, db)     // pro plan: 19.99/month, 199.90/year

	tests := []struct {
		name         string
		planName     string
		billingCycle string
		seats        int
		wantAmount   float64
		wantErr      bool
	}{
		{
			name:         "monthly pro with 1 seat",
			planName:     "pro",
			billingCycle: billing.BillingCycleMonthly,
			seats:        1,
			wantAmount:   19.99,
		},
		{
			name:         "yearly pro with 1 seat",
			planName:     "pro",
			billingCycle: billing.BillingCycleYearly,
			seats:        1,
			wantAmount:   199.90,
		},
		{
			name:         "monthly pro with 5 seats",
			planName:     "pro",
			billingCycle: billing.BillingCycleMonthly,
			seats:        5,
			wantAmount:   99.95,
		},
		{
			name:         "based plan",
			planName:     "based",
			billingCycle: billing.BillingCycleMonthly,
			seats:        1,
			wantAmount:   9.9, // Based plan is now $9.9/month
		},
		{
			name:         "invalid plan",
			planName:     "nonexistent",
			billingCycle: billing.BillingCycleMonthly,
			seats:        1,
			wantErr:      true,
		},
		{
			name:         "zero seats defaults to 1",
			planName:     "pro",
			billingCycle: billing.BillingCycleMonthly,
			seats:        0,
			wantAmount:   19.99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.CalculateSubscriptionPrice(ctx, tt.planName, tt.billingCycle, tt.seats)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !almostEqual(result.Amount, tt.wantAmount, 0.001) {
				t.Errorf("expected amount %f, got %f", tt.wantAmount, result.Amount)
			}
			if !almostEqual(result.Amount, result.ActualAmount, 0.001) {
				t.Errorf("expected ActualAmount to equal Amount for new subscription")
			}
		})
	}
}

func TestCalculateRenewalPrice(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db)
	proPlan, _ := service.GetPlan(ctx, "pro")

	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          3,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// Renew with same cycle
	result, err := service.CalculateRenewalPrice(ctx, 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 19.99 * 3 seats
	expectedAmount := 59.97
	if !almostEqual(result.Amount, expectedAmount, 0.001) {
		t.Errorf("expected amount %f, got %f", expectedAmount, result.Amount)
	}
	if result.BillingCycle != billing.BillingCycleMonthly {
		t.Errorf("expected monthly cycle, got %s", result.BillingCycle)
	}
}

func TestCalculateRenewalPrice_ChangeCycle(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db)
	proPlan, _ := service.GetPlan(ctx, "pro")

	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          2,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// Renew with yearly cycle
	result, err := service.CalculateRenewalPrice(ctx, 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 199.90 * 2 seats
	expectedAmount := 399.80
	if !almostEqual(result.Amount, expectedAmount, 0.001) {
		t.Errorf("expected amount %f, got %f", expectedAmount, result.Amount)
	}
	if result.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", result.BillingCycle)
	}
}

func TestCalculateRenewalPriceNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.CalculateRenewalPrice(ctx, 999, "")
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestCalculateRenewalPriceYearly(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db)
	proPlan, _ := service.GetPlan(ctx, "pro")

	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		SeatCount:          1,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
	}
	db.Create(sub)

	result, err := service.CalculateRenewalPrice(ctx, 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !almostEqual(result.Amount, 199.90, 0.001) {
		t.Errorf("expected yearly amount 199.90, got %f", result.Amount)
	}
}

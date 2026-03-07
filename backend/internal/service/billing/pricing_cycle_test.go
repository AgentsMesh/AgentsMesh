package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Billing Cycle Change Tests
// ===========================================

func TestCalculateBillingCycleChangePrice(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db)
	proPlan, _ := service.GetPlan(ctx, "pro")

	// Create monthly subscription halfway through period
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          2,
		CurrentPeriodStart: now.Add(-15 * 24 * time.Hour),
		CurrentPeriodEnd:   now.Add(15 * 24 * time.Hour),
	}
	db.Create(sub)

	// Change from monthly to yearly (upgrade - needs payment)
	result, err := service.CalculateBillingCycleChangePrice(ctx, 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", result.BillingCycle)
	}
	// Yearly is more expensive, so ActualAmount should be positive
	if result.ActualAmount < 0 {
		t.Error("expected positive ActualAmount for monthly to yearly change")
	}
}

func TestCalculateBillingCycleChangePriceYearlyToMonthly(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db)
	proPlan, _ := service.GetPlan(ctx, "pro")

	// Create yearly subscription
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		SeatCount:          1,
		CurrentPeriodStart: now.Add(-180 * 24 * time.Hour),
		CurrentPeriodEnd:   now.Add(185 * 24 * time.Hour),
	}
	db.Create(sub)

	// Change from yearly to monthly (downgrade - credit applied at renewal)
	result, err := service.CalculateBillingCycleChangePrice(ctx, 1, billing.BillingCycleMonthly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BillingCycle != billing.BillingCycleMonthly {
		t.Errorf("expected monthly cycle, got %s", result.BillingCycle)
	}
	// Monthly is cheaper, ActualAmount should be 0 (credit at renewal)
	if result.ActualAmount != 0 {
		t.Errorf("expected 0 ActualAmount for downgrade, got %f", result.ActualAmount)
	}
}

func TestCalculateBillingCycleChangePriceSameCycle(t *testing.T) {
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
		SeatCount:          1,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// Same cycle - should return nil
	result, err := service.CalculateBillingCycleChangePrice(ctx, 1, billing.BillingCycleMonthly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for same billing cycle")
	}
}

func TestCalculateBillingCycleChangePriceNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.CalculateBillingCycleChangePrice(ctx, 999, billing.BillingCycleYearly)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestCalculateRemainingPeriodRatio(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		periodStart time.Time
		periodEnd   time.Time
		wantRatio   float64
		tolerance   float64
	}{
		{
			name:        "full period remaining",
			periodStart: now,
			periodEnd:   now.Add(30 * 24 * time.Hour),
			wantRatio:   1.0,
			tolerance:   0.01,
		},
		{
			name:        "half period remaining",
			periodStart: now.Add(-15 * 24 * time.Hour),
			periodEnd:   now.Add(15 * 24 * time.Hour),
			wantRatio:   0.5,
			tolerance:   0.01,
		},
		{
			name:        "period ended",
			periodStart: now.Add(-30 * 24 * time.Hour),
			periodEnd:   now.Add(-1 * time.Hour),
			wantRatio:   0,
			tolerance:   0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio := calculateRemainingPeriodRatio(tt.periodStart, tt.periodEnd)
			diff := ratio - tt.wantRatio
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("expected ratio %f (±%f), got %f", tt.wantRatio, tt.tolerance, ratio)
			}
		})
	}
}

func TestGetPricePreview(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db)

	// Test subscription preview (no existing subscription needed)
	result, err := service.GetPricePreview(ctx, 0, billing.OrderTypeSubscription, "pro", billing.BillingCycleMonthly, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !almostEqual(result.Amount, 19.99, 0.001) {
		t.Errorf("expected amount 19.99, got %f", result.Amount)
	}

	// Test invalid order type
	_, err = service.GetPricePreview(ctx, 0, "invalid", "", "", 0)
	if err == nil {
		t.Error("expected error for invalid order type")
	}
}

func TestGetPricePreviewUpgrade(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	seedProPlan(t, db)

	// Create a subscription
	basedPlan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             basedPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		SeatCount:          1,
		CurrentPeriodStart: now.Add(-15 * 24 * time.Hour),
		CurrentPeriodEnd:   now.Add(15 * 24 * time.Hour),
	}
	db.Create(sub)

	// Test upgrade preview
	result, err := service.GetPricePreview(ctx, 1, billing.OrderTypePlanUpgrade, "pro", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Amount <= 0 {
		t.Error("expected positive amount for upgrade")
	}
}

func TestGetPricePreviewSeatPurchase(t *testing.T) {
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
		SeatCount:          1,
		CurrentPeriodStart: now.Add(-10 * 24 * time.Hour),
		CurrentPeriodEnd:   now.Add(20 * 24 * time.Hour),
	}
	db.Create(sub)

	// Test seat purchase preview
	result, err := service.GetPricePreview(ctx, 1, billing.OrderTypeSeatPurchase, "", "", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Seats != 2 {
		t.Errorf("expected 2 seats, got %d", result.Seats)
	}
}

func TestGetPricePreviewRenewal(t *testing.T) {
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

	// Test renewal preview
	result, err := service.GetPricePreview(ctx, 1, billing.OrderTypeRenewal, "", billing.BillingCycleYearly, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly billing cycle, got %s", result.BillingCycle)
	}
}

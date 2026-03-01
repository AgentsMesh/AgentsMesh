package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Pricing Calculation Tests (from additional_coverage_test.go)
// ===========================================

// TestAdditionalCalculateSubscriptionPriceWithCurrency_CNY tests pricing in CNY
func TestAdditionalCalculateSubscriptionPriceWithCurrency_CNY(t *testing.T) {
	svc, _ := setupTestService(t)

	price, err := svc.CalculateSubscriptionPriceWithCurrency(context.Background(), "pro", "CNY", billing.BillingCycleMonthly, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.Currency != "CNY" {
		t.Errorf("expected CNY currency, got %s", price.Currency)
	}
	// Pro plan CNY monthly is 139
	if price.Amount != 139 {
		t.Errorf("expected amount 139, got %f", price.Amount)
	}
}

// TestAdditionalCalculateSubscriptionPriceWithCurrency_ZeroSeats tests pricing with zero seats (defaults to 1)
func TestAdditionalCalculateSubscriptionPriceWithCurrency_ZeroSeats(t *testing.T) {
	svc, _ := setupTestService(t)

	price, err := svc.CalculateSubscriptionPriceWithCurrency(context.Background(), "pro", "USD", billing.BillingCycleMonthly, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.Seats != 1 {
		t.Errorf("expected 1 seat (default), got %d", price.Seats)
	}
}

// TestAdditionalCalculateSubscriptionPriceWithCurrency_NegativeSeats tests pricing with negative seats
func TestAdditionalCalculateSubscriptionPriceWithCurrency_NegativeSeats(t *testing.T) {
	svc, _ := setupTestService(t)

	price, err := svc.CalculateSubscriptionPriceWithCurrency(context.Background(), "pro", "USD", billing.BillingCycleMonthly, -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.Seats != 1 {
		t.Errorf("expected 1 seat (default), got %d", price.Seats)
	}
}

// ===========================================
// Upgrade Price Calculation Tests
// ===========================================

// TestAdditionalCalculateUpgradePrice_CurrentPlanNotFound tests upgrade with invalid current plan
func TestAdditionalCalculateUpgradePrice_CurrentPlanNotFound(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	// Create subscription with non-existent plan ID
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             9999, // non-existent
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	_, err := svc.CalculateUpgradePrice(context.Background(), 1, "enterprise")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

// ===========================================
// Renewal Price Calculation Tests
// ===========================================

// TestAdditionalCalculateRenewalPrice_YearlyCycle tests renewal with yearly cycle
func TestAdditionalCalculateRenewalPrice_YearlyCycle(t *testing.T) {
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

	// Calculate renewal with yearly cycle
	price, err := svc.CalculateRenewalPrice(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use current yearly billing cycle
	if price.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", price.BillingCycle)
	}
}

// ===========================================
// Quota Check Tests
// ===========================================

// TestAdditionalCheckQuota_ZeroLimit tests quota check with 0 limit (disabled quota type)
func TestAdditionalCheckQuota_ZeroLimit(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based plan: max_users=1, but other quotas may be 0
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Check a quota type that isn't explicitly limited
	err := svc.CheckQuota(context.Background(), 1, "nonexistent_quota_type", 100)
	// Should pass because unknown quota types default to no limit
	if err != nil {
		t.Errorf("expected no error for unknown quota type, got %v", err)
	}
}

package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Basic Pricing Coverage Tests
// ===========================================

// TestCalculateSubscriptionPriceWithCurrency_InvalidPlan tests pricing with invalid plan
func TestCalculateSubscriptionPriceWithCurrency_InvalidPlan(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.CalculateSubscriptionPriceWithCurrency(context.Background(), "nonexistent", "USD", billing.BillingCycleMonthly, 1)
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

// TestCalculateSubscriptionPriceWithCurrency_InvalidCurrency tests pricing with invalid currency
func TestCalculateSubscriptionPriceWithCurrency_InvalidCurrency(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.CalculateSubscriptionPriceWithCurrency(context.Background(), "pro", "EUR", billing.BillingCycleMonthly, 1)
	if err != ErrPriceNotFound {
		t.Errorf("expected ErrPriceNotFound, got %v", err)
	}
}

// TestCalculateSubscriptionPriceWithCurrency_YearlyCycle tests yearly pricing
func TestCalculateSubscriptionPriceWithCurrency_YearlyCycle(t *testing.T) {
	svc, _ := setupTestService(t)

	price, err := svc.CalculateSubscriptionPriceWithCurrency(context.Background(), "pro", "USD", billing.BillingCycleYearly, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Pro yearly is 199.90 per seat
	expected := 199.90 * 2
	if price.Amount != expected {
		t.Errorf("expected amount %.2f, got %.2f", expected, price.Amount)
	}
}

// TestCalculateRemainingPeriodRatio_FutureEnd tests ratio with future end date
func TestCalculateRemainingPeriodRatio_FutureEnd(t *testing.T) {
	now := time.Now()
	start := now.AddDate(0, -1, 0) // Started 1 month ago
	end := now.AddDate(0, 0, 15)   // Ends in 15 days

	ratio := calculateRemainingPeriodRatio(start, end)
	if ratio <= 0 || ratio >= 1 {
		t.Errorf("expected ratio between 0 and 1, got %f", ratio)
	}
}

// TestCalculateRemainingPeriodRatio_ExpiredPeriod tests ratio when period has ended
func TestCalculateRemainingPeriodRatio_ExpiredPeriod(t *testing.T) {
	now := time.Now()
	start := now.AddDate(0, -2, 0) // Started 2 months ago
	end := now.AddDate(0, -1, 0)   // Ended 1 month ago

	ratio := calculateRemainingPeriodRatio(start, end)
	if ratio != 0 {
		t.Errorf("expected 0 for expired period, got %f", ratio)
	}
}

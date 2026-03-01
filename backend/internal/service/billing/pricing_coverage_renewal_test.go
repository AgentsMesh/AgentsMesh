package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Renewal and Billing Cycle Pricing Tests
// ===========================================

// TestCalculateRenewalPrice_NoSubscription tests renewal pricing without subscription
func TestCalculateRenewalPrice_NoSubscription(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.CalculateRenewalPrice(context.Background(), 999, "")
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestCalculateRenewalPrice_WithNewCycle tests renewal with new billing cycle
func TestCalculateRenewalPrice_WithNewCycle(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Calculate renewal with yearly cycle
	price, err := svc.CalculateRenewalPrice(context.Background(), 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", price.BillingCycle)
	}
}

// TestCalculateBillingCycleChangePrice_NoSubscription tests cycle change without subscription
func TestCalculateBillingCycleChangePrice_NoSubscription(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.CalculateBillingCycleChangePrice(context.Background(), 999, billing.BillingCycleYearly)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestCalculateBillingCycleChangePrice_Success tests successful cycle change pricing
func TestCalculateBillingCycleChangePrice_Success(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	price, err := svc.CalculateBillingCycleChangePrice(context.Background(), 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", price.BillingCycle)
	}
}

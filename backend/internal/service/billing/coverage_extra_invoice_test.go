package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Extra Tests - Invoice and Plan Coverage
// ===========================================

// TestGetInvoices_WithLimitAndOffset tests getting invoices with pagination
func TestGetInvoices_WithLimitAndOffset(t *testing.T) {
	svc, _ := setupTestService(t)

	invoices, err := svc.GetInvoicesByOrg(context.Background(), 1, 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty (no invoices in test db)
	if len(invoices) != 0 {
		t.Errorf("expected 0 invoices, got %d", len(invoices))
	}
}

// TestCalculateRenewalPrice_PlanNotFound tests renewal with missing plan
func TestCalculateRenewalPrice_PlanNotFound(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             9999, // Non-existent plan
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	_, err := svc.CalculateRenewalPrice(context.Background(), 1, "")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

// TestCalculateBillingCycleChangePrice_PlanNotFound tests cycle change with missing plan
func TestCalculateBillingCycleChangePrice_PlanNotFound(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             9999, // Non-existent plan
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	_, err := svc.CalculateBillingCycleChangePrice(context.Background(), 1, billing.BillingCycleYearly)
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

// TestListPlansWithPrices_NoPlans tests listing plans when query fails
func TestListPlansWithPrices_NoPlans(t *testing.T) {
	svc, db := setupTestService(t)

	// Delete all plans
	db.Exec("DELETE FROM plan_prices")
	db.Exec("DELETE FROM subscription_plans")

	plans, err := svc.ListPlansWithPrices(context.Background(), "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}
}

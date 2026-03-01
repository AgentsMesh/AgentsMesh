package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Additional Service Coverage Tests
// ===========================================

// TestRenewSubscription tests subscription renewal
func TestRenewSubscription(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now.AddDate(0, -1, 0),
		CurrentPeriodEnd:   now,
		SeatCount:          1,
	})

	err := svc.RenewSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.CurrentPeriodEnd.Before(now) {
		t.Error("expected CurrentPeriodEnd to be extended")
	}
}

// TestRenewSubscription_NotFound tests renewing non-existent subscription
func TestRenewSubscription_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.RenewSubscription(context.Background(), 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestRenewSubscription_WithDowngrade tests renewal preserves pending downgrade
// Note: RenewSubscription only extends the period, it does not apply the downgrade
func TestRenewSubscription_WithDowngrade(t *testing.T) {
	svc, db := setupTestService(t)

	downgradePlan := "based"
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now.AddDate(0, -1, 0),
		CurrentPeriodEnd:   now,
		SeatCount:          1,
		DowngradeToPlan:    &downgradePlan,
	})

	err := svc.RenewSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	// RenewSubscription only extends period, doesn't apply downgrade
	if sub.PlanID != 2 {
		t.Errorf("expected plan to remain pro (2), got %d", sub.PlanID)
	}
	// DowngradeToPlan should still be set (downgrade happens via separate process)
	if sub.DowngradeToPlan == nil || *sub.DowngradeToPlan != "based" {
		t.Error("expected DowngradeToPlan to be preserved")
	}
	// Period should be extended
	if sub.CurrentPeriodEnd.Before(now) {
		t.Error("expected CurrentPeriodEnd to be extended")
	}
}

// TestRenewSubscription_WithNextBillingCycle tests renewal preserves next billing cycle
// Note: RenewSubscription only extends the period using current cycle, cycle change happens separately
func TestRenewSubscription_WithNextBillingCycle(t *testing.T) {
	svc, db := setupTestService(t)

	nextCycle := billing.BillingCycleYearly
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now.AddDate(0, -1, 0),
		CurrentPeriodEnd:   now,
		SeatCount:          1,
		NextBillingCycle:   &nextCycle,
	})

	err := svc.RenewSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	// RenewSubscription uses current billing cycle, doesn't apply next cycle change
	if sub.BillingCycle != billing.BillingCycleMonthly {
		t.Errorf("expected billing cycle to remain monthly, got %s", sub.BillingCycle)
	}
	// NextBillingCycle should be preserved (cycle change happens via separate process)
	if sub.NextBillingCycle == nil || *sub.NextBillingCycle != billing.BillingCycleYearly {
		t.Error("expected NextBillingCycle to be preserved")
	}
	// Period should be extended by 1 month (current cycle)
	if sub.CurrentPeriodEnd.Before(now) {
		t.Error("expected CurrentPeriodEnd to be extended")
	}
}

// TestRenewSubscription_YearlyCycle tests renewal with yearly billing cycle
func TestRenewSubscription_YearlyCycle(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now.AddDate(-1, 0, 0),
		CurrentPeriodEnd:   now,
		SeatCount:          1,
	})

	err := svc.RenewSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	// Period should be extended by 1 year
	expectedEnd := now.AddDate(1, 0, 0)
	// Allow 1 second tolerance for time comparison
	if sub.CurrentPeriodEnd.Sub(expectedEnd).Abs() > time.Second {
		t.Errorf("expected CurrentPeriodEnd to be extended by 1 year, got %v", sub.CurrentPeriodEnd)
	}
}

// TestGetBillingOverview_NotFound tests billing overview for non-existent subscription
func TestGetBillingOverview_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.GetBillingOverview(context.Background(), 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestListPlansWithPrices_EmptyResult tests listing plans with prices for unavailable currency
func TestListPlansWithPrices_EmptyResult(t *testing.T) {
	svc, _ := setupTestService(t)

	// EUR prices don't exist in test data
	plans, err := svc.ListPlansWithPrices(context.Background(), "EUR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plans) != 0 {
		t.Errorf("expected empty result for EUR currency, got %d plans", len(plans))
	}
}


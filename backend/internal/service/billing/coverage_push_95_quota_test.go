package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Push 95% Coverage - Quota Tests
// ===========================================

// TestCheckQuota_ConcurrentPods tests quota check for concurrent pods
func TestCheckQuota_ConcurrentPods(t *testing.T) {
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

	err := svc.CheckQuota(context.Background(), 1, "concurrent_pods", 1)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestCheckQuota_Repositories tests quota check for repositories
func TestCheckQuota_Repositories(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro: max 100 repositories
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	err := svc.CheckQuota(context.Background(), 1, "repositories", 1)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestCheckQuota_Runners tests quota check for runners
func TestCheckQuota_Runners(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro: max 10 runners
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	err := svc.CheckQuota(context.Background(), 1, "runners", 1)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestGetSeatUsage_PlanLoadPath tests when Plan is nil and needs loading
func TestGetSeatUsage_PlanLoadPath(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	// Create subscription without preloading plan
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          3,
	}
	db.Create(sub)

	usage, err := svc.GetSeatUsage(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.TotalSeats != 3 {
		t.Errorf("expected 3 total seats, got %d", usage.TotalSeats)
	}
}

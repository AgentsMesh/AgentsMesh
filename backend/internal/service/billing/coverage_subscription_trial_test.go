package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Trial Subscription Tests
// ===========================================

// TestAdditionalCreateTrialSubscription_EnterprisePlan tests creating trial with enterprise plan
func TestAdditionalCreateTrialSubscription_EnterprisePlan(t *testing.T) {
	svc, _ := setupTestService(t)

	sub, err := svc.CreateTrialSubscription(context.Background(), 1, "enterprise", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sub.Status != billing.SubscriptionStatusTrialing {
		t.Errorf("expected trialing status, got %s", sub.Status)
	}
	// Trial plan should be enterprise
	if sub.PlanID != 3 {
		t.Errorf("expected enterprise plan (3), got %d", sub.PlanID)
	}
}

// ===========================================
// Billing Overview Tests
// ===========================================

// TestAdditionalGetBillingOverview_WithPlanLoaded tests billing overview when plan is already loaded
func TestAdditionalGetBillingOverview_WithPlanLoaded(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          3,
	})

	overview, err := svc.GetBillingOverview(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if overview.Plan == nil {
		t.Error("expected plan in overview")
	}
	if overview.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected active status, got %s", overview.Status)
	}
}

// ===========================================
// Seat Usage Tests
// ===========================================

// TestAdditionalGetSeatUsage_NoPlan tests seat usage when plan is not loaded
func TestAdditionalGetSeatUsage_NoPlan(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          5,
	})

	usage, err := svc.GetSeatUsage(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.TotalSeats != 5 {
		t.Errorf("expected 5 total seats, got %d", usage.TotalSeats)
	}
}

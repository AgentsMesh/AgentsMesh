package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Extra Tests - Subscription Operations
// ===========================================

// TestUpdateSubscription_SamePlanNoChange tests update to same plan
func TestUpdateSubscription_SamePlanNoChange(t *testing.T) {
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

	// Update to same plan (pro to pro) - should be treated as paid upgrade
	sub, err := svc.UpdateSubscription(context.Background(), 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return current plan
	if sub.PlanID != 2 {
		t.Errorf("expected plan ID 2, got %d", sub.PlanID)
	}
}

// TestActivateTrialSubscription_AlreadyActive tests trial activation when already active
func TestActivateTrialSubscription_AlreadyActive(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive, // Already active
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Should do nothing and return nil
	err := svc.ActivateTrialSubscription(context.Background(), 1, billing.BillingCycleMonthly)
	if err != nil {
		t.Errorf("expected nil for already active subscription, got %v", err)
	}
}

// TestActivateTrialSubscription_YearlyCycle tests activating trial with yearly cycle
func TestActivateTrialSubscription_YearlyCycle(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusTrialing,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 0, 14),
		SeatCount:          1,
	})

	err := svc.ActivateTrialSubscription(context.Background(), 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", sub.BillingCycle)
	}
}

// TestActivateTrialSubscription_DefaultCycle tests activating trial with default (monthly) cycle
func TestActivateTrialSubscription_DefaultCycle(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusTrialing,
		BillingCycle:       billing.BillingCycleYearly, // Will be changed to monthly
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 0, 14),
		SeatCount:          1,
	})

	// Empty billing cycle = default to monthly
	err := svc.ActivateTrialSubscription(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.BillingCycle != billing.BillingCycleMonthly {
		t.Errorf("expected monthly cycle, got %s", sub.BillingCycle)
	}
}

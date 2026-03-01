package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Extra Tests - Subscription Settings
// ===========================================

// TestSetAutoRenew_Disable tests disabling auto-renew
func TestSetAutoRenew_Disable(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
		AutoRenew:          true,
	})

	// Disable auto-renew
	err := svc.SetAutoRenew(context.Background(), 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.AutoRenew {
		t.Error("expected AutoRenew to be false")
	}
}

// TestSetCancelAtPeriodEnd_Enable tests setting cancel at period end flag
func TestSetCancelAtPeriodEnd_Enable(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	err := svc.SetCancelAtPeriodEnd(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if !sub.CancelAtPeriodEnd {
		t.Error("expected CancelAtPeriodEnd to be true")
	}
}

// TestFreezeSubscription_Active tests freezing an active subscription
func TestFreezeSubscription_Active(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	err := svc.FreezeSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusFrozen {
		t.Errorf("expected frozen status, got %s", sub.Status)
	}
	if sub.FrozenAt == nil {
		t.Error("expected FrozenAt to be set")
	}
}

// TestUnfreezeSubscription_Yearly tests unfreezing with yearly billing
func TestUnfreezeSubscription_Yearly(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusFrozen,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
		FrozenAt:           &now,
	})

	err := svc.UnfreezeSubscription(context.Background(), 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected active status, got %s", sub.Status)
	}
	if sub.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", sub.BillingCycle)
	}
	if sub.FrozenAt != nil {
		t.Error("expected FrozenAt to be nil")
	}
}

// TestUnfreezeSubscription_DefaultCycle tests unfreezing with default (monthly) billing
func TestUnfreezeSubscription_DefaultCycle(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusFrozen,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
		SeatCount:          1,
		FrozenAt:           &now,
	})

	// Empty cycle = default to monthly
	err := svc.UnfreezeSubscription(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.BillingCycle != billing.BillingCycleMonthly {
		t.Errorf("expected monthly cycle, got %s", sub.BillingCycle)
	}
}

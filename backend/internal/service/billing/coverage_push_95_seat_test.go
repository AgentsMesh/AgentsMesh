package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Push 95% Coverage - Seat Tests
// ===========================================

// TestCheckSeatAvailability_WithMembers tests seat check with members
func TestCheckSeatAvailability_WithMembers(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          3, // 3 seats total
	})

	// Add members (uses 2 seats)
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 1, 'owner')")
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 2, 'member')")

	// Should have 3 - 2 = 1 available seat
	err := svc.CheckSeatAvailability(context.Background(), 1, 1)
	if err != nil {
		t.Errorf("expected nil for 1 seat, got %v", err)
	}

	// Should fail when requesting more than available
	err = svc.CheckSeatAvailability(context.Background(), 1, 2)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded for 2 seats, got %v", err)
	}
}

// TestCheckSeatAvailability_NoSubscription tests default behavior without subscription
func TestCheckSeatAvailability_NoSubscription(t *testing.T) {
	svc, _ := setupTestService(t)

	// No subscription = 1 default seat
	err := svc.CheckSeatAvailability(context.Background(), 999, 1)
	if err != nil {
		t.Errorf("expected nil for first seat, got %v", err)
	}

	// More than 1 should fail
	err = svc.CheckSeatAvailability(context.Background(), 999, 2)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

// TestSetCustomQuota tests setting custom quota
func TestSetCustomQuota_Success(t *testing.T) {
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

	err := svc.SetCustomQuota(context.Background(), 1, "users", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify custom quota was set
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.CustomQuotas == nil || sub.CustomQuotas["users"] != float64(100) {
		t.Error("expected custom quota to be set")
	}
}

// TestGetCurrentConcurrentPods_Empty tests getting concurrent pods count when empty
func TestGetCurrentConcurrentPods_Empty(t *testing.T) {
	svc, _ := setupTestService(t)

	count, err := svc.GetCurrentConcurrentPods(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return 0 (no pods)
	if count != 0 {
		t.Errorf("expected 0 pods, got %d", count)
	}
}

// TestRecordUsage tests recording usage
func TestRecordUsage_Success(t *testing.T) {
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

	err := svc.RecordUsage(context.Background(), 1, billing.UsageTypePodMinutes, 50.5, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify usage was recorded
	usage, _ := svc.GetUsage(context.Background(), 1, billing.UsageTypePodMinutes)
	if usage != 50.5 {
		t.Errorf("expected usage 50.5, got %f", usage)
	}
}

// TestCreateTrialSubscription_InvalidPlanName tests trial with invalid plan
func TestCreateTrialSubscription_InvalidPlanName(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.CreateTrialSubscription(context.Background(), 1, "nonexistent", 14)
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

// TestUpdateSubscription_ClearDowngradeOnUpgrade tests clearing downgrade when upgrading
func TestUpdateSubscription_ClearDowngradeOnUpgrade(t *testing.T) {
	svc, db := setupTestService(t)

	// Seed free plan
	freePlan := seedFreePlan(t, db)

	now := time.Now()
	downgradePlan := "free"
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             freePlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
		DowngradeToPlan:    &downgradePlan,
	})

	// Upgrade from free to based - should clear downgrade
	sub, err := svc.UpdateSubscription(context.Background(), 1, "based")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sub.DowngradeToPlan != nil {
		t.Error("expected DowngradeToPlan to be cleared")
	}
}

package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Final Coverage Tests - Quota Checks
// ===========================================

// TestCheckQuota_FrozenSubscription tests quota check with frozen subscription
func TestCheckQuota_FrozenSubscription(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusFrozen,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          5,
		FrozenAt:           &now,
	})

	err := svc.CheckQuota(context.Background(), 1, "users", 1)
	if err != ErrSubscriptionFrozen {
		t.Errorf("expected ErrSubscriptionFrozen, got %v", err)
	}
}

// TestCheckQuota_WithCustomQuota tests quota check with custom quota
func TestCheckQuota_WithCustomQuota(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	customQuotas := billing.CustomQuotas{"users": float64(10)}
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          5,
		CustomQuotas:       customQuotas,
	})

	// Should pass with custom quota
	err := svc.CheckQuota(context.Background(), 1, "users", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fail when exceeding custom quota
	err = svc.CheckQuota(context.Background(), 1, "users", 15)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

// TestCheckQuota_NoSubscriptionNoBasedPlan tests quota when no subscription and no based plan
func TestCheckQuota_DBError(t *testing.T) {
	svc, _ := setupTestService(t)

	// No subscription exists - should try to use Based plan
	// Based plan exists from seed data, so this should pass
	err := svc.CheckQuota(context.Background(), 999, "users", 1)
	if err != nil {
		t.Errorf("expected nil (allows by default when using Based plan), got %v", err)
	}
}

// TestCheckQuota_UnlimitedResources tests quota for resources with -1 limit
func TestCheckQuota_UnlimitedResources(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             3, // enterprise has unlimited resources (-1)
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Should pass even with high amount due to unlimited
	err := svc.CheckQuota(context.Background(), 1, "users", 1000)
	if err != nil {
		t.Errorf("expected nil for unlimited quota, got %v", err)
	}
}

// TestCheckQuota_QuotaExceeded tests when quota is exceeded
func TestCheckQuota_QuotaExceeded(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based plan: max_users = 1
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Create an existing member to use the quota
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 1, 'owner')")

	err := svc.CheckQuota(context.Background(), 1, "users", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

// TestGetUsageHistory_WithUsageType tests getting usage history with specific type
func TestGetUsageHistory_WithUsageType(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now.AddDate(0, -1, 0),
		CurrentPeriodEnd:   now.AddDate(0, 0, 1),
		SeatCount:          1,
	})

	// Create usage record
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      billing.UsageTypePodMinutes,
		Quantity:       100,
		PeriodStart:    now.AddDate(0, -1, 0),
		PeriodEnd:      now,
	})

	// Get history with usage type
	records, err := svc.GetUsageHistory(context.Background(), 1, billing.UsageTypePodMinutes, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) == 0 {
		t.Error("expected at least one record")
	}
}

// TestGetUsageHistory_WithoutUsageType tests getting all usage history
func TestGetUsageHistory_WithoutUsageType(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      billing.UsageTypePodMinutes,
		Quantity:       50,
		PeriodStart:    now.AddDate(0, -1, 0),
		PeriodEnd:      now,
	})

	// Get all history
	records, err := svc.GetUsageHistory(context.Background(), 1, "", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) == 0 {
		t.Error("expected at least one record")
	}
}

// TestGetCurrentResourceCount_AllTypes tests resource counting for all types
func TestGetCurrentResourceCount_AllTypes(t *testing.T) {
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

	resources := []string{"users", "runners", "concurrent_pods", "repositories", "pod_minutes"}
	for _, resource := range resources {
		err := svc.CheckQuota(context.Background(), 1, resource, 1)
		if err != nil && err != ErrQuotaExceeded {
			t.Errorf("unexpected error for resource %s: %v", resource, err)
		}
	}
}

// TestSeatUsage_BasedPlan tests seat usage for based plan (cannot add seats)
func TestSeatUsage_BasedPlan(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based plan
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	usage, err := svc.GetSeatUsage(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.CanAddSeats {
		t.Error("expected CanAddSeats to be false for based plan")
	}
}

// TestCalculateSeatPurchasePrice_BasedPlanFails tests seat purchase fails for based plan
func TestCalculateSeatPurchasePrice_BasedPlanFails(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based plan
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	_, err := svc.CalculateSeatPurchasePrice(context.Background(), 1, 5)
	if err != ErrInvalidPlan {
		t.Errorf("expected ErrInvalidPlan for based plan, got %v", err)
	}
}

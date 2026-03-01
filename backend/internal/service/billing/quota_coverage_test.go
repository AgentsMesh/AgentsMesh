package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Additional Quota and Usage Coverage Tests
// ===========================================

// TestGetUsage_WithUsageRecords tests GetUsage with actual usage records
func TestGetUsage_WithUsageRecords(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	periodStart := now.AddDate(0, 0, -15)
	periodEnd := now.AddDate(0, 0, 15)

	// Create subscription
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		SeatCount:          1,
	})

	// Create usage records within the period
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      "pod_minutes",
		Quantity:       100.5,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	})
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      "pod_minutes",
		Quantity:       50.5,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	})

	usage, err := svc.GetUsage(context.Background(), 1, "pod_minutes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 151.0 // 100.5 + 50.5
	if usage != expected {
		t.Errorf("expected usage %.1f, got %.1f", expected, usage)
	}
}

// TestGetUsageHistory_WithRecords tests GetUsageHistory with records
func TestGetUsageHistory_WithRecords(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()

	// Create usage records for different months
	for i := 0; i < 3; i++ {
		periodStart := now.AddDate(0, -i, 0)
		periodEnd := now.AddDate(0, -i+1, 0)
		db.Create(&billing.UsageRecord{
			OrganizationID: 1,
			UsageType:      "pod_minutes",
			Quantity:       float64(100 * (i + 1)),
			PeriodStart:    periodStart,
			PeriodEnd:      periodEnd,
		})
	}

	history, err := svc.GetUsageHistory(context.Background(), 1, "pod_minutes", 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("expected 3 records, got %d", len(history))
	}
}

// TestGetUsageHistory_AllTypes tests GetUsageHistory without type filter
func TestGetUsageHistory_AllTypes(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()

	// Create usage records for different types
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      "pod_minutes",
		Quantity:       100,
		PeriodStart:    now.AddDate(0, 0, -15),
		PeriodEnd:      now.AddDate(0, 0, 15),
	})
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      "api_calls",
		Quantity:       200,
		PeriodStart:    now.AddDate(0, 0, -15),
		PeriodEnd:      now.AddDate(0, 0, 15),
	})

	// Get all usage types
	history, err := svc.GetUsageHistory(context.Background(), 1, "", 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("expected 2 records for all types, got %d", len(history))
	}
}

// TestGetSeatUsage_WithMemberCount tests GetSeatUsage with actual member count
func TestGetSeatUsage_WithMemberCount(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro: max 50 users
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          10,
	})

	// Need to create org_members table and records for complete test
	// For now, just verify the structure
	usage, err := svc.GetSeatUsage(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.TotalSeats != 10 {
		t.Errorf("expected 10 total seats, got %d", usage.TotalSeats)
	}
	if usage.MaxSeats != 50 {
		t.Errorf("expected 50 max seats, got %d", usage.MaxSeats)
	}
}

// TestCheckQuota_WithExistingUsage tests quota check with existing usage
func TestCheckQuota_WithExistingUsage(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro: 1000 pod minutes
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now.AddDate(0, 0, -15),
		CurrentPeriodEnd:   now.AddDate(0, 0, 15),
		SeatCount:          1,
	})

	// Add some usage
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      "pod_minutes",
		Quantity:       900,
		PeriodStart:    now.AddDate(0, 0, -15),
		PeriodEnd:      now.AddDate(0, 0, 15),
	})

	// Check if we can use 50 more minutes (should pass: 900 + 50 = 950 < 1000)
	err := svc.CheckQuota(context.Background(), 1, "pod_minutes", 50)
	if err != nil {
		t.Errorf("expected quota check to pass, got error: %v", err)
	}

	// Check if we can use 200 more minutes (should fail: 900 + 200 = 1100 > 1000)
	err = svc.CheckQuota(context.Background(), 1, "pod_minutes", 200)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

// TestCheckQuota_UnlimitedPlan tests quota check on unlimited plan
func TestCheckQuota_UnlimitedPlan(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             3, // enterprise: unlimited (-1)
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Even large requests should pass
	err := svc.CheckQuota(context.Background(), 1, "users", 10000)
	if err != nil {
		t.Errorf("expected unlimited quota to pass, got error: %v", err)
	}
}

// TestCheckQuota_PausedSubscription tests quota check on paused subscription
func TestCheckQuota_PausedSubscription(t *testing.T) {
	svc, db := setupTestService(t)

	freezeTime := time.Now()
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusPaused,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
		FrozenAt:           &freezeTime,
	})

	// Frozen/paused subscriptions should fail quota checks
	err := svc.CheckQuota(context.Background(), 1, "users", 1)
	if err != ErrSubscriptionFrozen {
		t.Errorf("expected ErrSubscriptionFrozen, got %v", err)
	}
}

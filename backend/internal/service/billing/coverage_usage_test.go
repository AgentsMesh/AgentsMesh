package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Usage/History/Seats Tests
// ===========================================

// TestGetInvoicesByOrg_NoLimit tests getting all invoices without limit
func TestGetInvoicesByOrg_NoLimit(t *testing.T) {
	svc, _ := setupTestService(t)

	// Without limit (limit=0)
	invoices, err := svc.GetInvoicesByOrg(context.Background(), 1, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty is fine
	if invoices == nil {
		t.Error("expected non-nil slice, got nil")
	}
}

// TestGetUsage_MultipleRecords tests usage aggregation
func TestGetUsage_MultipleRecords(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	periodStart := now.AddDate(0, 0, -15)
	periodEnd := now.AddDate(0, 0, 15)

	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		SeatCount:          1,
	})

	// Create multiple usage records
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      billing.UsageTypePodMinutes,
		Quantity:       100,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	})
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      billing.UsageTypePodMinutes,
		Quantity:       50,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	})

	usage, err := svc.GetUsage(context.Background(), 1, billing.UsageTypePodMinutes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should aggregate: 100 + 50 = 150
	if usage != 150 {
		t.Errorf("expected usage 150, got %f", usage)
	}
}

// TestGetUsageHistory_Multiple tests history with multiple records
func TestGetUsageHistory_Multiple(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	// Create multiple usage records
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      billing.UsageTypePodMinutes,
		Quantity:       100,
		PeriodStart:    now.AddDate(0, -1, 0),
		PeriodEnd:      now,
	})
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      "other_type",
		Quantity:       50,
		PeriodStart:    now.AddDate(0, -1, 0),
		PeriodEnd:      now,
	})

	// Get all history (empty usageType)
	records, err := svc.GetUsageHistory(context.Background(), 1, "", 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

// TestGetUsageHistory_WithFilter tests usage history with type filter
func TestGetUsageHistory_WithFilter(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	// Create multiple usage records of different types
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      billing.UsageTypePodMinutes,
		Quantity:       100,
		PeriodStart:    now.AddDate(0, -1, 0),
		PeriodEnd:      now,
	})
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      "other_type",
		Quantity:       50,
		PeriodStart:    now.AddDate(0, -1, 0),
		PeriodEnd:      now,
	})

	// Get history with specific type filter
	records, err := svc.GetUsageHistory(context.Background(), 1, billing.UsageTypePodMinutes, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only return pod_minutes type
	if len(records) != 1 {
		t.Errorf("expected 1 record with type filter, got %d", len(records))
	}
}

// TestGetSeatUsage_ProPlan tests seat usage with pro plan
func TestGetSeatUsage_ProPlan(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
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
	if !usage.CanAddSeats {
		t.Error("expected CanAddSeats to be true for pro plan")
	}
}

// TestGetSeatUsage_BasedPlan tests seat usage with based plan (fixed seats)
func TestGetSeatUsage_BasedPlan(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based plan - MaxUsers = 1
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

	// Based plan has fixed seats (MaxUsers = 1), so CanAddSeats should be false
	if usage.CanAddSeats {
		t.Error("expected CanAddSeats to be false for based plan")
	}
}

// TestGetSeatUsage_WithPlanNotPreloaded tests seat usage when plan is not preloaded
func TestGetSeatUsage_WithPlanNotPreloaded(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	// Create subscription - Plan will be nil initially
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          5,
		// Plan is NOT set - will be loaded
	}
	db.Create(sub)

	usage, err := svc.GetSeatUsage(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.TotalSeats != 5 {
		t.Errorf("expected 5 total seats, got %d", usage.TotalSeats)
	}
	// Plan should have been loaded
	if usage.MaxSeats <= 0 {
		t.Error("expected MaxSeats to be loaded from plan")
	}
}

// TestRecordUsage_NotFound tests recording usage without subscription
func TestRecordUsage_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.RecordUsage(context.Background(), 999, billing.UsageTypePodMinutes, 10, nil)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

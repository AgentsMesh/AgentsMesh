package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Extra Tests - Quota Coverage
// ===========================================

// TestGetUsage_EmptyResult tests getting usage with empty result
func TestGetUsage_EmptyResult(t *testing.T) {
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

	// No usage records, should return 0
	usage, err := svc.GetUsage(context.Background(), 1, "custom_type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage != 0 {
		t.Errorf("expected 0 usage, got %f", usage)
	}
}

// TestCheckQuota_UsersQuotaExceeded tests users quota exceeded
func TestCheckQuota_UsersQuotaExceeded(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based plan: max 1 user
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Add a member to use the only slot
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 1, 'owner')")

	err := svc.CheckQuota(context.Background(), 1, "users", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

// TestGetSeatUsage_NotFound tests seat usage with no subscription
func TestGetSeatUsage_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.GetSeatUsage(context.Background(), 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestCheckQuota_WithUnlimitedCustomQuota tests custom quota with -1 (unlimited)
func TestCheckQuota_WithUnlimitedCustomQuota(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	customQuotas := billing.CustomQuotas{"users": float64(-1)} // Unlimited custom quota
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             3, // enterprise: unlimited users (-1)
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
		CustomQuotas:       customQuotas,
	})

	// With custom quota -1 and enterprise plan (unlimited), should pass
	err := svc.CheckQuota(context.Background(), 1, "users", 1000)
	if err != nil {
		t.Errorf("expected nil for unlimited quota, got %v", err)
	}
}

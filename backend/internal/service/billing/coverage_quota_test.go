package billing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Quota Check Tests
// ===========================================

// TestCheckQuota_RunnersExceeded tests runners quota exceeded
func TestCheckQuota_RunnersExceeded(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based: max 1 runner
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Add a runner to use the quota (test DB uses simplified schema)
	db.Exec("INSERT INTO runners (organization_id, node_id) VALUES (1, 'runner1')")

	// Should fail when trying to add another runner
	err := svc.CheckQuota(context.Background(), 1, "runners", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

// TestCheckQuota_ReposExceeded tests repositories quota exceeded
func TestCheckQuota_ReposExceeded(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based: max 5 repos
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Add repos to fill quota (test DB uses simplified schema)
	for i := 0; i < 5; i++ {
		db.Exec("INSERT INTO repositories (organization_id, name, slug) VALUES (1, ?, ?)",
			fmt.Sprintf("repo-%d", i), fmt.Sprintf("repo-%d", i))
	}

	// Should fail when trying to add another repo
	err := svc.CheckQuota(context.Background(), 1, "repositories", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

// TestCheckQuota_ConcurrentPodsExceeded tests concurrent pods quota
func TestCheckQuota_ConcurrentPodsExceeded(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based: max 5 concurrent pods
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Add running pods to fill quota (test DB uses simplified schema with 'pods' table)
	for i := 0; i < 5; i++ {
		db.Exec("INSERT INTO pods (organization_id, pod_key, status) VALUES (1, ?, 'running')",
			fmt.Sprintf("pod-%d", i))
	}

	// Should fail when trying to add another pod
	err := svc.CheckQuota(context.Background(), 1, "concurrent_pods", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

// TestCheckQuota_DefaultPlan tests quota when no based plan exists
func TestCheckQuota_NoPlanFound(t *testing.T) {
	svc, db := setupTestService(t)

	// Delete all plans to simulate no plans in database
	db.Exec("DELETE FROM plan_prices")
	db.Exec("DELETE FROM subscription_plans")

	// No subscription and no plans - should allow by default (nil error)
	err := svc.CheckQuota(context.Background(), 999, "users", 1)
	if err != nil {
		t.Errorf("expected nil when no plan found, got %v", err)
	}
}

// TestCheckQuota_WithCustomQuota tests custom quota that is not -1
func TestCheckQuota_WithCustomQuotaLimit(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	customQuotas := billing.CustomQuotas{"users": float64(5)} // Custom limit of 5 users
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1, // based plan - would normally have 1 user limit
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
		CustomQuotas:       customQuotas,
	})

	// Should pass - custom quota allows 5 users
	err := svc.CheckQuota(context.Background(), 1, "users", 3)
	if err != nil {
		t.Errorf("expected nil for custom quota, got %v", err)
	}

	// Add 4 members to use quota
	for i := 0; i < 4; i++ {
		db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, ?, 'member')", i+1)
	}

	// Should fail - custom quota of 5, used 4, requesting 2 more (total 6 > 5)
	err = svc.CheckQuota(context.Background(), 1, "users", 2)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

// TestSetCustomQuota_NotFound tests setting quota without subscription
func TestSetCustomQuota_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.SetCustomQuota(context.Background(), 999, "users", 100)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// TestCheckQuota_SubPlanNilNeedsLoad tests quota check when subscription plan is nil and needs to be loaded
func TestCheckQuota_SubPlanNilNeedsLoad(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	// Create subscription without preloaded plan
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
		// Plan field is nil - will need to be loaded
	}
	db.Create(sub)

	// The subscription's Plan field is nil, so CheckQuota will load it
	err := svc.CheckQuota(context.Background(), 1, "users", 1)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

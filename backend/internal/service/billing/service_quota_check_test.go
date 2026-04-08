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

func TestCheckQuotaWithinLimit(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	err := service.CheckQuota(ctx, 1, "users", 1)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCheckQuotaExceeded(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db) // max_users = 5
	service.CreateSubscription(ctx, 1, "based")

	// Add existing members to exceed quota
	for i := 0; i < 5; i++ {
		db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, ?, 'member')", i+1)
	}

	err := service.CheckQuota(ctx, 1, "users", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestCheckQuotaUnlimited(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedEnterprisePlan(t, db) // max_users = -1 (unlimited)

	plan, _ := service.GetPlan(ctx, "enterprise")
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	err := service.CheckQuota(ctx, 1, "users", 1000)
	if err != nil {
		t.Errorf("expected no error for unlimited quota, got %v", err)
	}
}

func TestCheckQuotaCustomQuota(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")
	service.SetCustomQuota(ctx, 1, "users", 100)

	// Should allow up to 100 users now
	err := service.CheckQuota(ctx, 1, "users", 50)
	if err != nil {
		t.Errorf("expected no error with custom quota, got %v", err)
	}
}

func TestCheckQuotaNoSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	// No subscription - should use based plan defaults
	err := service.CheckQuota(ctx, 999, "users", 1)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCheckQuotaAllResourceTypes(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	resources := []string{"users", "runners", "concurrent_pods", "repositories", "pod_minutes", "unknown"}
	for _, resource := range resources {
		err := service.CheckQuota(ctx, 1, resource, 0)
		if err != nil {
			t.Errorf("unexpected error for resource %s: %v", resource, err)
		}
	}
}

func TestCheckQuotaRunners(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db) // max_runners = 1
	service.CreateSubscription(ctx, 1, "based")

	// Add one runner
	db.Exec("INSERT INTO runners (organization_id, node_id) VALUES (1, 'runner1')")

	// Should fail to add another
	err := service.CheckQuota(ctx, 1, "runners", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestCheckQuotaConcurrentPods(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db) // max_concurrent_pods = 5 (based plan)
	service.CreateSubscription(ctx, 1, "based")

	// Add five running pods to reach the limit
	for i := 1; i <= 5; i++ {
		db.Exec("INSERT INTO pods (organization_id, pod_key, status) VALUES (1, ?, 'running')", fmt.Sprintf("pod-%d", i))
	}

	// Should fail to add another
	err := service.CheckQuota(ctx, 1, "concurrent_pods", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestCheckQuotaRepositories(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db) // max_repositories = 5 (based plan)
	service.CreateSubscription(ctx, 1, "based")

	// Add five repositories to reach the limit
	for i := 1; i <= 5; i++ {
		db.Exec("INSERT INTO repositories (organization_id, name, slug) VALUES (1, ?, ?)", fmt.Sprintf("repo-%d", i), fmt.Sprintf("repo-%d", i))
	}

	// Should fail to add another
	err := service.CheckQuota(ctx, 1, "repositories", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestCheckQuotaWithCustomQuotaExceeded(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")
	service.SetCustomQuota(ctx, 1, "users", 2) // Override to 2 users

	// Add 2 members
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 1, 'owner')")
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 2, 'member')")

	// Should fail to add another (quota is 2)
	err := service.CheckQuota(ctx, 1, "users", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestCheckQuotaPodMinutes(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db) // included_pod_minutes = 100
	service.CreateSubscription(ctx, 1, "based")

	// Record 100 minutes of usage
	service.RecordUsage(ctx, 1, "pod_minutes", 100.0, billing.UsageMetadata{})

	// Should fail to use more
	err := service.CheckQuota(ctx, 1, "pod_minutes", 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestCheckQuotaFrozenSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	plan := seedTestPlan(t, db)

	// Create frozen subscription
	now := time.Now()
	frozenAt := now.Add(-24 * time.Hour)
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusFrozen,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now.Add(-30 * 24 * time.Hour),
		CurrentPeriodEnd:   now.Add(-24 * time.Hour),
		FrozenAt:           &frozenAt,
	}
	db.Create(sub)

	// Should return ErrSubscriptionFrozen
	err := service.CheckQuota(ctx, 1, "users", 1)
	if err != ErrSubscriptionFrozen {
		t.Errorf("expected ErrSubscriptionFrozen, got %v", err)
	}
}

func TestCheckQuotaWithUnlimitedCustomQuota(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedEnterprisePlan(t, db) // Enterprise has unlimited users (-1)

	plan, _ := service.GetPlan(ctx, "enterprise")
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// Set custom quota to -1 (unlimited), which should skip custom check and fall through to plan limit (-1 = unlimited)
	service.SetCustomQuota(ctx, 1, "users", -1)

	// Should allow unlimited users via plan's -1 limit
	err := service.CheckQuota(ctx, 1, "users", 1000)
	if err != nil {
		t.Errorf("expected no error for unlimited quota, got %v", err)
	}
}

func TestSetCustomQuota(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	err := service.SetCustomQuota(ctx, 1, "users", 100)
	if err != nil {
		t.Fatalf("failed to set custom quota: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.CustomQuotas == nil {
		t.Fatal("expected custom quotas to be set")
	}
	if limit, ok := sub.CustomQuotas["users"].(float64); !ok || int(limit) != 100 {
		t.Errorf("expected users quota 100, got %v", sub.CustomQuotas["users"])
	}
}

func TestSetCustomQuotaNoSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	err := service.SetCustomQuota(ctx, 999, "users", 100)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

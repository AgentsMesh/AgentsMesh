package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupQuotaIntegration creates a billing service using testkit.SetupTestDB
// (shared across all packages) instead of the package-local setupTestDB.
func setupQuotaIntegration(t *testing.T) (*Service, context.Context, *gorm.DB, int64, int64) {
	t.Helper()
	db := testkit.SetupTestDB(t)
	repo := infra.NewBillingRepository(db)
	svc := NewService(repo, "")
	ctx := context.Background()

	seedTestPlan(t, db) // "based" plan: maxUsers=1, maxRunners=1, maxPods=5

	userID := testkit.CreateUser(t, db, "quota-user@test.com", "quota-user")
	orgID := testkit.CreateOrg(t, db, "quota-org", userID)
	_, err := svc.CreateSubscription(ctx, orgID, billing.PlanBased)
	require.NoError(t, err)

	return svc, ctx, db, orgID, userID
}

func TestBillingQuotaIntegration_CheckQuotaWithinLimit(t *testing.T) {
	svc, ctx, _, orgID, _ := setupQuotaIntegration(t)

	// "based" plan has maxRunners=1, with zero runners → requesting 0 should pass
	err := svc.CheckQuota(ctx, orgID, "runners", 0)
	require.NoError(t, err)
}

func TestBillingQuotaIntegration_CheckQuotaExceeded(t *testing.T) {
	svc, ctx, db, orgID, userID := setupQuotaIntegration(t)

	// "based" plan: maxUsers=1. The owner already counts. Add 1 member to exceed.
	db.Exec(
		"INSERT INTO organization_members (organization_id, user_id, role) VALUES (?, ?, 'member')",
		orgID, userID,
	)

	err := svc.CheckQuota(ctx, orgID, "users", 1)
	assert.ErrorIs(t, err, ErrQuotaExceeded)
}

func TestBillingQuotaIntegration_FrozenSubscription(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := infra.NewBillingRepository(db)
	svc := NewService(repo, "")
	ctx := context.Background()

	plan := seedTestPlan(t, db)
	userID := testkit.CreateUser(t, db, "frozen@test.com", "frozen")
	orgID := testkit.CreateOrg(t, db, "frozen-org", userID)

	now := time.Now()
	frozenAt := now.Add(-24 * time.Hour)
	sub := &billing.Subscription{
		OrganizationID:     orgID,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusFrozen,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now.Add(-30 * 24 * time.Hour),
		CurrentPeriodEnd:   now.Add(-24 * time.Hour),
		FrozenAt:           &frozenAt,
	}
	db.Create(sub)

	err := svc.CheckQuota(ctx, orgID, "users", 1)
	assert.ErrorIs(t, err, ErrSubscriptionFrozen)
}

func TestBillingQuotaIntegration_NoSubscriptionFallsBack(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := infra.NewBillingRepository(db)
	svc := NewService(repo, "")
	ctx := context.Background()

	seedTestPlan(t, db)

	// Org 999 has no subscription → should fall back to "based" plan
	err := svc.CheckQuota(ctx, 999, "users", 0)
	require.NoError(t, err)
}

func TestBillingQuotaIntegration_ConcurrentPods(t *testing.T) {
	svc, ctx, db, orgID, userID := setupQuotaIntegration(t)

	// "based" plan: max_concurrent_pods = 5. Fill up to the limit.
	for i := 1; i <= 5; i++ {
		db.Exec(
			"INSERT INTO pods (organization_id, pod_key, runner_id, created_by_id, status) VALUES (?, ?, 1, ?, 'running')",
			orgID, "pod-"+string(rune('a'+i)), userID,
		)
	}

	err := svc.CheckQuota(ctx, orgID, "concurrent_pods", 1)
	assert.ErrorIs(t, err, ErrQuotaExceeded)
}

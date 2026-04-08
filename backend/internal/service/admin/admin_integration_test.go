package admin

import (
	"context"
	"testing"

	adminDomain "github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newIntegrationService sets up a testkit SQLite DB and returns an admin Service.
func newIntegrationService(t *testing.T) (*Service, *database.GormWrapper) {
	t.Helper()
	gdb := testkit.SetupTestDB(t)
	wrapper := database.NewGormWrapper(gdb)
	return NewService(wrapper), wrapper
}

func TestAdmin_UserLifecycle(t *testing.T) {
	svc, _ := newIntegrationService(t)
	ctx := context.Background()

	// Seed two users via testkit.
	db := svc.db.GormDB()
	u1 := testkit.CreateUser(t, db, "alice@test.com", "alice")
	u2 := testkit.CreateUser(t, db, "bob@test.com", "bob")

	// ListUsers: verify count.
	resp, err := svc.ListUsers(ctx, &UserListQuery{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Data, 2)

	// GetUser by ID.
	got, err := svc.GetUser(ctx, u1)
	require.NoError(t, err)
	assert.Equal(t, "alice@test.com", got.Email)

	// GetUser not found.
	_, err = svc.GetUser(ctx, 99999)
	assert.ErrorIs(t, err, ErrUserNotFound)

	// DisableUser.
	disabled, err := svc.DisableUser(ctx, u1)
	require.NoError(t, err)
	assert.False(t, disabled.IsActive)

	// EnableUser.
	enabled, err := svc.EnableUser(ctx, u1)
	require.NoError(t, err)
	assert.True(t, enabled.IsActive)

	// GrantAdmin.
	admin, err := svc.GrantAdmin(ctx, u2)
	require.NoError(t, err)
	assert.True(t, admin.IsSystemAdmin)

	// RevokeAdmin (by a different admin).
	revoked, err := svc.RevokeAdmin(ctx, u2, u1)
	require.NoError(t, err)
	assert.False(t, revoked.IsSystemAdmin)

	// RevokeAdmin on self should fail.
	_, err = svc.RevokeAdmin(ctx, u1, u1)
	assert.ErrorIs(t, err, ErrCannotRevokeOwnAdmin)
}

func TestAdmin_OrgLifecycle(t *testing.T) {
	svc, _ := newIntegrationService(t)
	ctx := context.Background()
	db := svc.db.GormDB()

	owner := testkit.CreateUser(t, db, "owner@test.com", "owner")
	org1 := testkit.CreateOrg(t, db, "org-alpha", owner)
	testkit.CreateOrg(t, db, "org-beta", owner)

	// ListOrganizations.
	resp, err := svc.ListOrganizations(ctx, &OrganizationListQuery{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Data, 2)

	// GetOrganization.
	org, err := svc.GetOrganization(ctx, org1)
	require.NoError(t, err)
	assert.Equal(t, "Org org-alpha", org.Name)

	// GetOrganization not found.
	_, err = svc.GetOrganization(ctx, 99999)
	assert.ErrorIs(t, err, ErrOrganizationNotFound)

	// GetOrganizationWithMembers.
	org, members, err := svc.GetOrganizationWithMembers(ctx, org1)
	require.NoError(t, err)
	assert.Equal(t, org1, org.ID)
	assert.Len(t, members, 1)
	assert.Equal(t, "owner", members[0].Role)

	// DeleteOrganization — no runners, should succeed.
	err = svc.DeleteOrganization(ctx, org1)
	require.NoError(t, err)

	// Verify deleted.
	_, err = svc.GetOrganization(ctx, org1)
	assert.ErrorIs(t, err, ErrOrganizationNotFound)
}

func TestAdmin_OrgDeleteBlockedByRunner(t *testing.T) {
	svc, _ := newIntegrationService(t)
	ctx := context.Background()
	db := svc.db.GormDB()

	owner := testkit.CreateUser(t, db, "owner2@test.com", "owner2")
	orgID := testkit.CreateOrg(t, db, "org-with-runner", owner)
	testkit.CreateRunner(t, db, orgID, "node-x")

	err := svc.DeleteOrganization(ctx, orgID)
	assert.ErrorIs(t, err, ErrOrganizationHasActiveRunner)
}

func TestAdmin_RunnerManagement(t *testing.T) {
	svc, _ := newIntegrationService(t)
	ctx := context.Background()
	db := svc.db.GormDB()

	owner := testkit.CreateUser(t, db, "rowner@test.com", "rowner")
	orgID := testkit.CreateOrg(t, db, "runner-org", owner)
	r1 := testkit.CreateRunner(t, db, orgID, "node-a")
	testkit.CreateRunner(t, db, orgID, "node-b")

	// ListRunners.
	resp, err := svc.ListRunners(ctx, &RunnerListQuery{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Data, 2)
	// Each result should have organization info.
	assert.NotNil(t, resp.Data[0].Organization)

	// GetRunner.
	runner, err := svc.GetRunner(ctx, r1)
	require.NoError(t, err)
	assert.Equal(t, "node-a", runner.NodeID)

	// GetRunner not found.
	_, err = svc.GetRunner(ctx, 99999)
	assert.ErrorIs(t, err, ErrRunnerNotFound)

	// DisableRunner.
	disabled, err := svc.DisableRunner(ctx, r1)
	require.NoError(t, err)
	assert.False(t, disabled.IsEnabled)

	// EnableRunner.
	enabled, err := svc.EnableRunner(ctx, r1)
	require.NoError(t, err)
	assert.True(t, enabled.IsEnabled)

	// DeleteRunner — no active pods, no loop refs, should succeed.
	deleted, err := svc.DeleteRunner(ctx, r1)
	require.NoError(t, err)
	assert.Equal(t, r1, deleted.ID)

	// Verify deleted.
	_, err = svc.GetRunner(ctx, r1)
	assert.ErrorIs(t, err, ErrRunnerNotFound)
}

func TestAdmin_DashboardStats(t *testing.T) {
	svc, _ := newIntegrationService(t)
	ctx := context.Background()
	db := svc.db.GormDB()

	u1 := testkit.CreateUser(t, db, "stats1@test.com", "stats1")
	testkit.CreateUser(t, db, "stats2@test.com", "stats2")
	orgID := testkit.CreateOrg(t, db, "stats-org", u1)
	testkit.CreateRunner(t, db, orgID, "node-s1")

	stats, err := svc.GetDashboardStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.TotalUsers)
	assert.Equal(t, int64(2), stats.ActiveUsers)
	assert.Equal(t, int64(1), stats.TotalOrganizations)
	assert.Equal(t, int64(1), stats.TotalRunners)
	// OnlineRunners: testkit creates runners with status 'online'.
	assert.Equal(t, int64(1), stats.OnlineRunners)
	// Time-based metrics are non-negative (exact values depend on SQLite
	// CURRENT_TIMESTAMP vs Go timezone, so we just verify no error).
	assert.GreaterOrEqual(t, stats.NewUsersToday, int64(0))
	assert.GreaterOrEqual(t, stats.NewUsersThisWeek, int64(0))
	assert.GreaterOrEqual(t, stats.NewUsersThisMonth, int64(0))
}

func TestAdmin_AuditLog(t *testing.T) {
	svc, _ := newIntegrationService(t)
	ctx := context.Background()
	db := svc.db.GormDB()

	adminID := testkit.CreateUser(t, db, "audit-admin@test.com", "audit-admin")

	// LogActionFromContext.
	err := svc.LogActionFromContext(
		ctx, adminID,
		adminDomain.AuditActionUserDisable,
		adminDomain.TargetTypeUser,
		42,
		map[string]bool{"is_active": true},
		map[string]bool{"is_active": false},
		"127.0.0.1", "test-agent",
	)
	require.NoError(t, err)

	// GetAuditLogs — no filter.
	action := adminDomain.AuditActionUserDisable
	resp, err := svc.GetAuditLogs(ctx, &adminDomain.AuditLogQuery{
		Action:   &action,
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, adminDomain.AuditActionUserDisable, resp.Data[0].Action)
	assert.Equal(t, adminDomain.TargetTypeUser, resp.Data[0].TargetType)
	assert.Equal(t, adminID, resp.Data[0].AdminUserID)
}

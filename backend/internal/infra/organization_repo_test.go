package infra

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedOrganization(t *testing.T, db *gorm.DB, slug string) int64 {
	t.Helper()
	org := organization.Organization{Name: "Test", Slug: slug}
	require.NoError(t, db.Create(&org).Error)
	return org.ID
}

func countWhereOrg(t *testing.T, db *gorm.DB, table string, orgID int64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Table(table).Where("organization_id = ?", orgID).Count(&n).Error)
	return n
}

func assertOrgDeleted(t *testing.T, db *gorm.DB, id int64) {
	t.Helper()
	var org organization.Organization
	err := db.First(&org, id).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestOrganizationRepo_DeleteWithCleanup_EmptyOrg(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewOrganizationRepository(db)

	id := seedOrganization(t, db, "empty-org")

	require.NoError(t, repo.DeleteWithCleanup(context.Background(), id))

	assertOrgDeleted(t, db, id)
}

func TestOrganizationRepo_DeleteWithCleanup_WithChannels(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewOrganizationRepository(db)

	id := seedOrganization(t, db, "chan-org")

	require.NoError(t, db.Exec(
		"INSERT INTO channels (id, organization_id, name) VALUES (?, ?, ?), (?, ?, ?)",
		1, id, "general", 2, id, "random",
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO channel_messages (channel_id, body) VALUES (1, 'hi'), (1, 'bye'), (2, 'yo')",
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO channel_members (channel_id, user_id) VALUES (1, 100), (2, 100)",
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO channel_read_states (channel_id, user_id) VALUES (1, 100)",
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO channel_pods (channel_id, pod_key) VALUES (1, 'pod-a')",
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO channel_access (channel_id) VALUES (1)",
	).Error)

	require.NoError(t, repo.DeleteWithCleanup(context.Background(), id))

	assertOrgDeleted(t, db, id)
	for _, tbl := range []string{
		"channels", "channel_messages", "channel_members",
		"channel_read_states", "channel_pods", "channel_access",
	} {
		var n int64
		require.NoError(t, db.Table(tbl).Count(&n).Error)
		assert.Zerof(t, n, "%s should be empty after org cleanup", tbl)
	}
}

// Regression guard for the production bug fixed in this PR.
//
// pod_bindings has no channel_id column (see migration 000001) — the old
// DeleteWithCleanup tried `DELETE FROM pod_bindings WHERE channel_id IN (...)`
// and crashed every org deletion with SQLSTATE 42703. We rely on the
// organization_id FK CASCADE to clean the table now.
//
// The testkit schema previously included a spurious `channel_id` column that
// hid the bug; with that fixed, this test would have failed on the old code.
func TestOrganizationRepo_DeleteWithCleanup_WithPodBindings(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewOrganizationRepository(db)

	id := seedOrganization(t, db, "bindings-org")

	require.NoError(t, db.Exec(
		"INSERT INTO pod_bindings (organization_id, initiator_pod, target_pod) VALUES (?, 'pod-a', 'pod-b'), (?, 'pod-c', 'pod-d')",
		id, id,
	).Error)
	require.Equal(t, int64(2), countWhereOrg(t, db, "pod_bindings", id))

	require.NoError(t, repo.DeleteWithCleanup(context.Background(), id))

	assertOrgDeleted(t, db, id)
	// testkit's in-memory SQLite has FKs disabled, so CASCADE doesn't fire.
	// What matters is that DeleteWithCleanup does NOT touch pod_bindings.channel_id —
	// the test passes purely because the buggy SQL is gone. Production Postgres
	// has the real ON DELETE CASCADE on pod_bindings.organization_id (see
	// migration 000001) so the actual cleanup happens there.
}

func TestOrganizationRepo_DeleteWithCleanup_WithLoops(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewOrganizationRepository(db)

	id := seedOrganization(t, db, "loops-org")

	require.NoError(t, db.Exec(
		"INSERT INTO loops (id, organization_id, name, slug, prompt_template) VALUES (1, ?, 'L1', 'l1', 'p')",
		id,
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO loop_runs (organization_id, loop_id, run_number) VALUES (?, 1, 1), (?, 1, 2)",
		id, id,
	).Error)

	require.NoError(t, repo.DeleteWithCleanup(context.Background(), id))

	assertOrgDeleted(t, db, id)
	assert.Zero(t, countWhereOrg(t, db, "loops", id), "loops missing application-level cleanup")
	assert.Zero(t, countWhereOrg(t, db, "loop_runs", id), "loop_runs missing application-level cleanup")
}

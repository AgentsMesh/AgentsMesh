package infra

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The transaction's whole point is that a create failure rolls the claim back,
// so the auth key stays reusable and no orphan runner survives. That only fires
// when tx.Create hits UNIQUE(organization_id, node_id) — a path the service test
// never reaches because its pre-check short-circuits first. Here we call the repo
// directly with a pre-existing conflicting runner to force the in-tx failure.
func TestAuthorizeAndCreateRunner_RollsBackClaimOnDuplicate(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewRunnerRepository(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&runner.Runner{
		OrganizationID: 1, NodeID: "dup-node", Status: runner.RunnerStatusOffline,
	}).Error)

	pa := &runner.PendingAuth{
		AuthKey: "k1", MachineKey: "m1", ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	require.NoError(t, db.Create(pa).Error)

	claimed, err := repo.AuthorizeAndCreateRunner(ctx, pa.ID, 1, &runner.Runner{
		OrganizationID: 1, NodeID: "dup-node", Status: runner.RunnerStatusOffline,
	}, -1)

	require.True(t, errors.Is(err, gorm.ErrDuplicatedKey),
		"a duplicate (org, node_id) must surface as ErrDuplicatedKey so the service maps it to 409")
	assert.Zero(t, claimed)

	var after runner.PendingAuth
	require.NoError(t, db.First(&after, pa.ID).Error)
	assert.False(t, after.Authorized, "the claim must roll back so the auth key stays reusable")
	assert.Nil(t, after.RunnerID)

	var count int64
	db.Model(&runner.Runner{}).Where("organization_id = ? AND node_id = ?", 1, "dup-node").Count(&count)
	assert.Equal(t, int64(1), count, "no orphan runner may survive the rolled-back tx")
}

func TestAuthorizeAndCreateRunner_RejectsAtQuota(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewRunnerRepository(db)
	ctx := context.Background()

	// Org already at its limit of 1.
	require.NoError(t, db.Create(&runner.Runner{
		OrganizationID: 1, NodeID: "existing", Status: runner.RunnerStatusOffline,
	}).Error)
	pa := &runner.PendingAuth{
		AuthKey: "kq", MachineKey: "mq", ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	require.NoError(t, db.Create(pa).Error)

	claimed, err := repo.AuthorizeAndCreateRunner(ctx, pa.ID, 1, &runner.Runner{
		OrganizationID: 1, NodeID: "over-quota", Status: runner.RunnerStatusOffline,
	}, 1)

	require.ErrorIs(t, err, runner.ErrRunnerQuotaExceeded)
	assert.Zero(t, claimed)

	var after runner.PendingAuth
	require.NoError(t, db.First(&after, pa.ID).Error)
	assert.False(t, after.Authorized, "a quota rejection must not burn the auth key")

	var count int64
	db.Model(&runner.Runner{}).Where("organization_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count, "no runner may be created over quota")
}

func TestAuthorizeAndCreateRunner_HappyPathLinks(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewRunnerRepository(db)
	ctx := context.Background()

	pa := &runner.PendingAuth{
		AuthKey: "k2", MachineKey: "m2", ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	require.NoError(t, db.Create(pa).Error)

	rn := &runner.Runner{OrganizationID: 1, NodeID: "fresh-node", Status: runner.RunnerStatusOffline}
	claimed, err := repo.AuthorizeAndCreateRunner(ctx, pa.ID, 1, rn, -1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), claimed)

	var after runner.PendingAuth
	require.NoError(t, db.First(&after, pa.ID).Error)
	assert.True(t, after.Authorized)
	require.NotNil(t, after.RunnerID)
	assert.Equal(t, rn.ID, *after.RunnerID)
}

package infra

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtensionRepo_ClaimSyncLock_FreshClaim(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewExtensionRepository(db)
	ctx := context.Background()

	reg := &extension.SkillRegistry{
		RepositoryURL: "https://example.com/fresh",
		Branch:        "main",
		SyncStatus:    extension.SyncStatusPending,
		IsActive:      true,
	}
	require.NoError(t, repo.CreateSkillRegistry(ctx, reg))

	claimed, wasStale, err := repo.ClaimSyncLock(ctx, reg.ID, 30*time.Minute)
	require.NoError(t, err)
	assert.True(t, claimed, "pending lock must be claimable")
	assert.False(t, wasStale, "pending lock is not stale")

	reloaded, err := repo.GetSkillRegistry(ctx, reg.ID)
	require.NoError(t, err)
	assert.Equal(t, extension.SyncStatusSyncing, reloaded.SyncStatus, "CAS UPDATE must persist syncing status")
}

func TestExtensionRepo_ClaimSyncLock_RejectsFreshPeer(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewExtensionRepository(db)
	ctx := context.Background()

	reg := &extension.SkillRegistry{
		RepositoryURL: "https://example.com/contested",
		Branch:        "main",
		SyncStatus:    extension.SyncStatusPending,
		IsActive:      true,
	}
	require.NoError(t, repo.CreateSkillRegistry(ctx, reg))

	claimed1, _, err := repo.ClaimSyncLock(ctx, reg.ID, 30*time.Minute)
	require.NoError(t, err)
	require.True(t, claimed1)

	claimed2, wasStale, err := repo.ClaimSyncLock(ctx, reg.ID, 30*time.Minute)
	require.NoError(t, err)
	assert.False(t, claimed2, "second caller must be rejected while first holds fresh lock")
	assert.False(t, wasStale)
}

func TestExtensionRepo_ClaimSyncLock_ReclaimsStaleLock(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewExtensionRepository(db)
	ctx := context.Background()

	reg := &extension.SkillRegistry{
		RepositoryURL: "https://example.com/stale",
		Branch:        "main",
		SyncStatus:    extension.SyncStatusSyncing,
		IsActive:      true,
	}
	require.NoError(t, repo.CreateSkillRegistry(ctx, reg))

	// Backdate updated_at past the stale threshold (CreateSkillRegistry just
	// set it to now via the default; we need it old enough to look abandoned).
	require.NoError(t, db.Model(&extension.SkillRegistry{}).
		Where("id = ?", reg.ID).
		Update("updated_at", time.Now().Add(-2*time.Hour)).Error)

	claimed, wasStale, err := repo.ClaimSyncLock(ctx, reg.ID, 30*time.Minute)
	require.NoError(t, err)
	assert.True(t, claimed, "stale lock must be reclaimable")
	assert.True(t, wasStale, "wasStale must flag reclaim path for ops visibility")
}

func TestExtensionRepo_ClaimSyncLock_NotFound(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewExtensionRepository(db)

	claimed, wasStale, err := repo.ClaimSyncLock(context.Background(), 999999, 30*time.Minute)
	assert.Error(t, err, "missing registry must error")
	assert.False(t, claimed)
	assert.False(t, wasStale)
}

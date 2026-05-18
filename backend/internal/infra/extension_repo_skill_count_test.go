package infra

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createRegistry returns a persisted SkillRegistry; createMarketItem inserts
// a row whose registry/slug/is_active drive the SkillCount derivation under test.
func createRegistry(t *testing.T, repo extension.Repository, url string) *extension.SkillRegistry {
	t.Helper()
	reg := &extension.SkillRegistry{
		RepositoryURL: url,
		Branch:        "main",
		SyncStatus:    extension.SyncStatusPending,
		IsActive:      true,
	}
	require.NoError(t, repo.CreateSkillRegistry(context.Background(), reg))
	return reg
}

// createMarketItem inserts a market item then explicitly overrides is_active
// via raw UPDATE because GORM's INSERT skips zero-value bool fields, which
// would otherwise let the column default (`true`) win every time.
func createMarketItem(t *testing.T, db *gorm.DB, repo extension.Repository, registryID int64, slug string, active bool) {
	t.Helper()
	item := &extension.SkillMarketItem{
		RegistryID: registryID,
		Slug:       slug,
		ContentSha: "sha-" + slug,
		StorageKey: "key-" + slug,
		Version:    1,
		IsActive:   active,
	}
	require.NoError(t, repo.CreateSkillMarketItem(context.Background(), item))
	if !active {
		require.NoError(t, db.Exec(
			"UPDATE skill_market_items SET is_active = ? WHERE id = ?", false, item.ID,
		).Error)
	}
}

func TestExtensionRepo_SkillCount_DerivedOnGet(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewExtensionRepository(db)
	ctx := context.Background()

	reg := createRegistry(t, repo, "https://example.com/r1")
	createMarketItem(t, db, repo, reg.ID, "alpha", true)
	createMarketItem(t, db, repo, reg.ID, "beta", true)
	createMarketItem(t, db, repo, reg.ID, "gamma", false) // inactive — must not count

	got, err := repo.GetSkillRegistry(ctx, reg.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.SkillCount, "GetSkillRegistry must derive count from active market items only")
}

func TestExtensionRepo_SkillCount_DerivedOnList(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewExtensionRepository(db)
	ctx := context.Background()

	r1 := createRegistry(t, repo, "https://example.com/list-a")
	r2 := createRegistry(t, repo, "https://example.com/list-b")
	r3 := createRegistry(t, repo, "https://example.com/list-c")

	createMarketItem(t, db, repo, r1.ID, "x", true)
	createMarketItem(t, db, repo, r2.ID, "y", true)
	createMarketItem(t, db, repo, r2.ID, "z", true)
	// r3 intentionally has no items — must come back as 0, not missing.

	all, err := repo.ListAllActiveSkillRegistries(ctx)
	require.NoError(t, err)

	counts := map[int64]int{}
	for _, reg := range all {
		counts[reg.ID] = reg.SkillCount
	}
	assert.Equal(t, 1, counts[r1.ID])
	assert.Equal(t, 2, counts[r2.ID])
	assert.Equal(t, 0, counts[r3.ID], "registry without market items must derive count=0")
}

func TestExtensionRepo_SkillCount_NotPersisted(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewExtensionRepository(db)
	ctx := context.Background()

	reg := createRegistry(t, repo, "https://example.com/transient")
	createMarketItem(t, db, repo, reg.ID, "one", true)

	// Set SkillCount on the in-memory struct and persist via UpdateSkillRegistry.
	// Because the GORM field tag is `-`, the value must not reach the DB.
	got, err := repo.GetSkillRegistry(ctx, reg.ID)
	require.NoError(t, err)
	got.SkillCount = 999
	require.NoError(t, repo.UpdateSkillRegistry(ctx, got))

	// Direct table lookup must show no skill_count column influence.
	reloaded, err := repo.GetSkillRegistry(ctx, reg.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, reloaded.SkillCount, "SkillCount must always reflect live active-item count, never a persisted value")
}

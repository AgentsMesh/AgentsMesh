package infra

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/tokenusage"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTokenUsageTestDB creates an in-memory SQLite database with the token_usages table.
func setupTokenUsageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testkit.SetupTestDB(t)
}

// seedUsers inserts test users for JOIN tests.
func seedUsers(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO users (id, name, username, email) VALUES (1, 'Alice', 'alice', 'alice@test.com')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO users (id, name, username, email) VALUES (2, 'Bob', 'bob', 'bob@test.com')`).Error)
}

func int64Ptr(v int64) *int64 { return &v }

func newTestRecord(orgID, userID, runnerID int64, agentSlug, model string, input, output int64, createdAt time.Time) *tokenusage.TokenUsage {
	return &tokenusage.TokenUsage{
		OrganizationID: orgID,
		PodKey:         "pod-test",
		UserID:         int64Ptr(userID),
		RunnerID:       int64Ptr(runnerID),
		AgentSlug:      agentSlug,
		Model:          model,
		InputTokens:    input,
		OutputTokens:   output,
		CreatedAt:      createdAt,
	}
}

// --- Create / CreateBatch ---

func TestTokenUsageRepo_Create(t *testing.T) {
	db := setupTokenUsageTestDB(t)
	repo := NewTokenUsageRepository(db)
	ctx := context.Background()

	record := newTestRecord(1, 10, 20, "claude", "opus", 100, 50, time.Now())
	require.NoError(t, repo.Create(ctx, record))
	assert.NotZero(t, record.ID)
}

func TestTokenUsageRepo_CreateBatch(t *testing.T) {
	db := setupTokenUsageTestDB(t)
	repo := NewTokenUsageRepository(db)
	ctx := context.Background()

	t.Run("inserts multiple records", func(t *testing.T) {
		now := time.Now()
		records := []*tokenusage.TokenUsage{
			newTestRecord(1, 10, 20, "claude", "opus", 100, 50, now),
			newTestRecord(1, 10, 20, "claude", "sonnet", 200, 100, now),
		}
		require.NoError(t, repo.CreateBatch(ctx, records))
		assert.NotZero(t, records[0].ID)
		assert.NotZero(t, records[1].ID)
	})

	t.Run("empty slice is no-op", func(t *testing.T) {
		require.NoError(t, repo.CreateBatch(ctx, nil))
		require.NoError(t, repo.CreateBatch(ctx, []*tokenusage.TokenUsage{}))
	})
}

// --- Helper function tests ---

func TestValidGranularity(t *testing.T) {
	assert.Equal(t, "day", validGranularity("day"))
	assert.Equal(t, "week", validGranularity("week"))
	assert.Equal(t, "month", validGranularity("month"))
	assert.Equal(t, "day", validGranularity(""))
	assert.Equal(t, "day", validGranularity("hour"))
	assert.Equal(t, "day", validGranularity("year"))
}

func TestEffectiveLimit(t *testing.T) {
	assert.Equal(t, 100, effectiveLimit(0))
	assert.Equal(t, 100, effectiveLimit(-1))
	assert.Equal(t, 50, effectiveLimit(50))
	assert.Equal(t, 1, effectiveLimit(1))
}

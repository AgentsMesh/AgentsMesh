package infra

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/tokenusage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTokenUsageTestDB creates an in-memory SQLite database with the token_usages table.
func setupTokenUsageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS token_usages (
			id                    INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id       INTEGER NOT NULL,
			pod_id                INTEGER,
			pod_key               TEXT NOT NULL,
			user_id               INTEGER NOT NULL,
			runner_id             INTEGER NOT NULL,
			agent_slug       TEXT NOT NULL,
			model                 TEXT,
			input_tokens          INTEGER NOT NULL DEFAULT 0,
			output_tokens         INTEGER NOT NULL DEFAULT 0,
			cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
			cache_read_tokens     INTEGER NOT NULL DEFAULT 0,
			session_started_at    DATETIME,
			session_ended_at      DATETIME,
			created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	// Minimal users table for GetByUser JOIN.
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			name     TEXT,
			username TEXT,
			email    TEXT
		)
	`).Error
	require.NoError(t, err)

	return db
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

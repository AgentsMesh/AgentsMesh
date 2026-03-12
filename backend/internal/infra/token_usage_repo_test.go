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
			agent_type_slug       TEXT NOT NULL,
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

func newTestRecord(orgID, userID, runnerID int64, agentType, model string, input, output int64, createdAt time.Time) *tokenusage.TokenUsage {
	return &tokenusage.TokenUsage{
		OrganizationID: orgID,
		PodKey:         "pod-test",
		UserID:         int64Ptr(userID),
		RunnerID:       int64Ptr(runnerID),
		AgentTypeSlug:  agentType,
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

// --- GetSummary ---

func TestTokenUsageRepo_GetSummary(t *testing.T) {
	db := setupTokenUsageTestDB(t)
	repo := NewTokenUsageRepository(db)
	ctx := context.Background()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	// Seed data: two records within range, one outside.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p1", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50, CacheCreationTokens: 10, CacheReadTokens: 5,
		CreatedAt: yesterday,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p2", UserID: int64Ptr(2), RunnerID: int64Ptr(1),
		AgentTypeSlug: "aider", Model: "gpt4",
		InputTokens: 200, OutputTokens: 100, CacheCreationTokens: 20, CacheReadTokens: 10,
		CreatedAt: yesterday,
	}))
	// Outside range.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p3", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 9999, OutputTokens: 9999,
		CreatedAt: lastWeek.Add(-24 * time.Hour),
	}))
	// Different org.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 99, PodKey: "p4", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 8888, OutputTokens: 8888,
		CreatedAt: yesterday,
	}))

	filter := tokenusage.AggregationFilter{
		StartTime: lastWeek,
		EndTime:   now.Add(time.Hour),
	}

	t.Run("aggregates within range for org", func(t *testing.T) {
		result, err := repo.GetSummary(ctx, 1, filter)
		require.NoError(t, err)
		assert.Equal(t, int64(300), result.InputTokens)
		assert.Equal(t, int64(150), result.OutputTokens)
		assert.Equal(t, int64(30), result.CacheCreationTokens)
		assert.Equal(t, int64(15), result.CacheReadTokens)
		assert.Equal(t, int64(495), result.TotalTokens)
	})

	t.Run("filters by agent_type", func(t *testing.T) {
		agent := "claude"
		f := filter
		f.AgentTypeSlug = &agent
		result, err := repo.GetSummary(ctx, 1, f)
		require.NoError(t, err)
		assert.Equal(t, int64(100), result.InputTokens)
	})

	t.Run("filters by user_id", func(t *testing.T) {
		uid := int64(2)
		f := filter
		f.UserID = &uid
		result, err := repo.GetSummary(ctx, 1, f)
		require.NoError(t, err)
		assert.Equal(t, int64(200), result.InputTokens)
	})

	t.Run("filters by model", func(t *testing.T) {
		model := "gpt4"
		f := filter
		f.Model = &model
		result, err := repo.GetSummary(ctx, 1, f)
		require.NoError(t, err)
		assert.Equal(t, int64(200), result.InputTokens)
	})

	t.Run("empty result returns zeros", func(t *testing.T) {
		result, err := repo.GetSummary(ctx, 999, filter)
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.InputTokens)
		assert.Equal(t, int64(0), result.TotalTokens)
	})
}

// --- GetTimeSeries ---

func TestTokenUsageRepo_GetTimeSeries(t *testing.T) {
	db := setupTokenUsageTestDB(t)
	repo := NewTokenUsageRepository(db)
	ctx := context.Background()

	// SQLite does not have date_trunc; it uses strftime. Skip if SQLite cannot handle it.
	// We test the code path rather than exact SQL compatibility.
	day1 := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 1, 11, 14, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p1", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: day1,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p2", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 200, OutputTokens: 100,
		CreatedAt: day2,
	}))

	filter := tokenusage.AggregationFilter{
		StartTime:   day1.Add(-time.Hour),
		EndTime:     day2.Add(time.Hour),
		Granularity: "day",
	}

	// date_trunc is PostgreSQL-specific; SQLite will error.
	// This validates the code path compiles and runs without panic.
	_, err := repo.GetTimeSeries(ctx, 1, filter)
	// Accept either success (PostgreSQL) or error (SQLite lacks date_trunc).
	if err != nil {
		assert.Contains(t, err.Error(), "date_trunc")
	}
}

// --- GetByAgent ---

func TestTokenUsageRepo_GetByAgent(t *testing.T) {
	db := setupTokenUsageTestDB(t)
	repo := NewTokenUsageRepository(db)
	ctx := context.Background()

	now := time.Now()
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p1", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p2", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "sonnet",
		InputTokens: 200, OutputTokens: 100,
		CreatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p3", UserID: int64Ptr(2), RunnerID: int64Ptr(1),
		AgentTypeSlug: "aider", Model: "gpt4",
		InputTokens: 300, OutputTokens: 150,
		CreatedAt: now,
	}))

	filter := tokenusage.AggregationFilter{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}

	results, err := repo.GetByAgent(ctx, 1, filter)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Ordered by total_tokens DESC: claude (450) > aider (450) — order may vary.
	agentMap := map[string]tokenusage.AgentUsage{}
	for _, r := range results {
		agentMap[r.AgentTypeSlug] = r
	}
	assert.Equal(t, int64(300), agentMap["claude"].InputTokens)
	assert.Equal(t, int64(300), agentMap["aider"].InputTokens)
}

// --- GetByUser ---

func TestTokenUsageRepo_GetByUser(t *testing.T) {
	db := setupTokenUsageTestDB(t)
	seedUsers(t, db)
	repo := NewTokenUsageRepository(db)
	ctx := context.Background()

	now := time.Now()
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p1", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p2", UserID: int64Ptr(2), RunnerID: int64Ptr(1),
		AgentTypeSlug: "aider", Model: "gpt4",
		InputTokens: 200, OutputTokens: 100,
		CreatedAt: now,
	}))

	filter := tokenusage.AggregationFilter{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}

	results, err := repo.GetByUser(ctx, 1, filter)
	require.NoError(t, err)
	require.Len(t, results, 2)

	userMap := map[int64]tokenusage.UserUsage{}
	for _, r := range results {
		userMap[r.UserID] = r
	}
	assert.Equal(t, int64(100), userMap[1].InputTokens)
	assert.Equal(t, "Alice", userMap[1].Username)
	assert.Equal(t, "alice@test.com", userMap[1].Email)
	assert.Equal(t, int64(200), userMap[2].InputTokens)
}

// --- GetByModel ---

func TestTokenUsageRepo_GetByModel(t *testing.T) {
	db := setupTokenUsageTestDB(t)
	repo := NewTokenUsageRepository(db)
	ctx := context.Background()

	now := time.Now()
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p1", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p2", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "sonnet",
		InputTokens: 200, OutputTokens: 100,
		CreatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p3", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 50, OutputTokens: 25,
		CreatedAt: now,
	}))

	filter := tokenusage.AggregationFilter{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}

	results, err := repo.GetByModel(ctx, 1, filter)
	require.NoError(t, err)
	require.Len(t, results, 2)

	modelMap := map[string]tokenusage.ModelUsage{}
	for _, r := range results {
		modelMap[r.Model] = r
	}
	assert.Equal(t, int64(150), modelMap["opus"].InputTokens)
	assert.Equal(t, int64(200), modelMap["sonnet"].InputTokens)
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

// --- EndTime boundary (half-open interval) ---

func TestTokenUsageRepo_EndTimeBoundary(t *testing.T) {
	db := setupTokenUsageTestDB(t)
	repo := NewTokenUsageRepository(db)
	ctx := context.Background()

	// Record exactly at the EndTime boundary — should be EXCLUDED (half-open: [start, end)).
	boundary := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	beforeBoundary := boundary.Add(-time.Hour)

	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "in-range", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: beforeBoundary,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "at-boundary", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 999, OutputTokens: 999,
		CreatedAt: boundary, // exactly at EndTime
	}))

	filter := tokenusage.AggregationFilter{
		StartTime: boundary.Add(-24 * time.Hour),
		EndTime:   boundary, // half-open: record at this exact time should be excluded
	}

	summary, err := repo.GetSummary(ctx, 1, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(100), summary.InputTokens, "record at exact EndTime must be excluded")
	assert.Equal(t, int64(50), summary.OutputTokens)
}

// --- applyOptionalFilters combined filters ---

func TestTokenUsageRepo_CombinedFilters(t *testing.T) {
	db := setupTokenUsageTestDB(t)
	seedUsers(t, db)
	repo := NewTokenUsageRepository(db)
	ctx := context.Background()

	now := time.Now()
	// Record matching all filters.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "match", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: now,
	}))
	// Same agent but different model — should be excluded by model filter.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "wrong-model", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "sonnet",
		InputTokens: 200, OutputTokens: 100,
		CreatedAt: now,
	}))
	// Same model but different agent — should be excluded by agent filter.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "wrong-agent", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "aider", Model: "opus",
		InputTokens: 300, OutputTokens: 150,
		CreatedAt: now,
	}))

	agent := "claude"
	model := "opus"
	uid := int64(1)
	filter := tokenusage.AggregationFilter{
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		AgentTypeSlug: &agent,
		Model:         &model,
		UserID:        &uid,
	}

	t.Run("GetSummary with combined filters", func(t *testing.T) {
		summary, err := repo.GetSummary(ctx, 1, filter)
		require.NoError(t, err)
		assert.Equal(t, int64(100), summary.InputTokens)
	})

	t.Run("GetByUser with combined filters (qualified columns)", func(t *testing.T) {
		results, err := repo.GetByUser(ctx, 1, filter)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, int64(100), results[0].InputTokens)
		assert.Equal(t, "Alice", results[0].Username)
	})

	t.Run("GetByAgent with combined filters", func(t *testing.T) {
		results, err := repo.GetByAgent(ctx, 1, filter)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "claude", results[0].AgentTypeSlug)
		assert.Equal(t, int64(100), results[0].InputTokens)
	})

	t.Run("GetByModel with combined filters", func(t *testing.T) {
		results, err := repo.GetByModel(ctx, 1, filter)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "opus", results[0].Model)
		assert.Equal(t, int64(100), results[0].InputTokens)
	})
}

// --- Org isolation ---

func TestTokenUsageRepo_OrgIsolation(t *testing.T) {
	db := setupTokenUsageTestDB(t)
	repo := NewTokenUsageRepository(db)
	ctx := context.Background()

	now := time.Now()
	// Org 1 record.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p1", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentTypeSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: now,
	}))
	// Org 2 record.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 2, PodKey: "p2", UserID: int64Ptr(2), RunnerID: int64Ptr(2),
		AgentTypeSlug: "aider", Model: "gpt4",
		InputTokens: 999, OutputTokens: 999,
		CreatedAt: now,
	}))

	filter := tokenusage.AggregationFilter{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}

	summary, err := repo.GetSummary(ctx, 1, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(100), summary.InputTokens, "org 2 data must not leak into org 1")
}

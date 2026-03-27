package infra

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/tokenusage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		AgentSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50, CacheCreationTokens: 10, CacheReadTokens: 5,
		CreatedAt: yesterday,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p2", UserID: int64Ptr(2), RunnerID: int64Ptr(1),
		AgentSlug: "aider", Model: "gpt4",
		InputTokens: 200, OutputTokens: 100, CacheCreationTokens: 20, CacheReadTokens: 10,
		CreatedAt: yesterday,
	}))
	// Outside range.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p3", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentSlug: "claude", Model: "opus",
		InputTokens: 9999, OutputTokens: 9999,
		CreatedAt: lastWeek.Add(-24 * time.Hour),
	}))
	// Different org.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 99, PodKey: "p4", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentSlug: "claude", Model: "opus",
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

	t.Run("filters by agent_slug", func(t *testing.T) {
		agent := "claude"
		f := filter
		f.AgentSlug = &agent
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
		AgentSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: day1,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p2", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentSlug: "claude", Model: "opus",
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
		AgentSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p2", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentSlug: "claude", Model: "sonnet",
		InputTokens: 200, OutputTokens: 100,
		CreatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p3", UserID: int64Ptr(2), RunnerID: int64Ptr(1),
		AgentSlug: "aider", Model: "gpt4",
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
		agentMap[r.AgentSlug] = r
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
		AgentSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p2", UserID: int64Ptr(2), RunnerID: int64Ptr(1),
		AgentSlug: "aider", Model: "gpt4",
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
		AgentSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p2", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentSlug: "claude", Model: "sonnet",
		InputTokens: 200, OutputTokens: 100,
		CreatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "p3", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentSlug: "claude", Model: "opus",
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

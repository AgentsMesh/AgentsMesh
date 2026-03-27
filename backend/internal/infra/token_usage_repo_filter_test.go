package infra

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/tokenusage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		AgentSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: beforeBoundary,
	}))
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "at-boundary", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentSlug: "claude", Model: "opus",
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
		AgentSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: now,
	}))
	// Same agent but different model — should be excluded by model filter.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "wrong-model", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentSlug: "claude", Model: "sonnet",
		InputTokens: 200, OutputTokens: 100,
		CreatedAt: now,
	}))
	// Same model but different agent — should be excluded by agent filter.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 1, PodKey: "wrong-agent", UserID: int64Ptr(1), RunnerID: int64Ptr(1),
		AgentSlug: "aider", Model: "opus",
		InputTokens: 300, OutputTokens: 150,
		CreatedAt: now,
	}))

	agent := "claude"
	model := "opus"
	uid := int64(1)
	filter := tokenusage.AggregationFilter{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
		AgentSlug: &agent,
		Model:     &model,
		UserID:    &uid,
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
		assert.Equal(t, "claude", results[0].AgentSlug)
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
		AgentSlug: "claude", Model: "opus",
		InputTokens: 100, OutputTokens: 50,
		CreatedAt: now,
	}))
	// Org 2 record.
	require.NoError(t, repo.Create(ctx, &tokenusage.TokenUsage{
		OrganizationID: 2, PodKey: "p2", UserID: int64Ptr(2), RunnerID: int64Ptr(2),
		AgentSlug: "aider", Model: "gpt4",
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

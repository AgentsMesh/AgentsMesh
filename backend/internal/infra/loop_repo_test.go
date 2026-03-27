package infra

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoopRepository_Create(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	l := &loop.Loop{
		OrganizationID: 1,
		Name:           "Test Loop",
		Slug:           "test-loop",
		PromptTemplate: "Review code in {{branch}}",
		ExecutionMode:  loop.ExecutionModeAutopilot,
		Status:         loop.StatusEnabled,
		SandboxStrategy:   loop.SandboxStrategyPersistent,
		ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1,
		TimeoutMinutes:    60,
		AutopilotConfig:   []byte("{}"),
		ConfigOverrides:   []byte("{}"),
		CreatedByID:       1,
	}

	err := repo.Create(ctx, l)
	require.NoError(t, err)
	assert.NotZero(t, l.ID)
}

func TestLoopRepository_GetByID(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	// Seed
	l := &loop.Loop{
		OrganizationID: 1, Name: "Test", Slug: "test",
		PromptTemplate: "prompt",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, l))

	t.Run("should return loop by ID", func(t *testing.T) {
		got, err := repo.GetByID(ctx, l.ID)
		require.NoError(t, err)
		assert.Equal(t, "test", got.Slug)
		assert.Equal(t, "Test", got.Name)
	})

	t.Run("should return ErrNotFound for non-existent ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.ErrorIs(t, err, loop.ErrNotFound)
	})
}

func TestLoopRepository_GetBySlug(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	l := &loop.Loop{
		OrganizationID: 1, Name: "My Loop", Slug: "my-loop",
		PromptTemplate: "prompt",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, l))

	t.Run("should return loop by org_id and slug", func(t *testing.T) {
		got, err := repo.GetBySlug(ctx, 1, "my-loop")
		require.NoError(t, err)
		assert.Equal(t, "My Loop", got.Name)
	})

	t.Run("should return ErrNotFound for different org", func(t *testing.T) {
		_, err := repo.GetBySlug(ctx, 999, "my-loop")
		assert.ErrorIs(t, err, loop.ErrNotFound)
	})

	t.Run("should return ErrNotFound for non-existent slug", func(t *testing.T) {
		_, err := repo.GetBySlug(ctx, 1, "no-such-loop")
		assert.ErrorIs(t, err, loop.ErrNotFound)
	})
}

func TestLoopRepository_List(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	// Seed multiple loops
	cron := "0 9 * * *"
	loops := []*loop.Loop{
		{OrganizationID: 1, Name: "Loop A", Slug: "loop-a", Status: loop.StatusEnabled,
			ExecutionMode: loop.ExecutionModeAutopilot, CronExpression: &cron,
			PromptTemplate: "p",
			SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
			MaxConcurrentRuns: 1, TimeoutMinutes: 60,
			AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"), CreatedByID: 1},
		{OrganizationID: 1, Name: "Loop B", Slug: "loop-b", Status: loop.StatusEnabled,
			ExecutionMode: loop.ExecutionModeDirect,
			PromptTemplate: "p",
			SandboxStrategy: loop.SandboxStrategyFresh, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
			MaxConcurrentRuns: 1, TimeoutMinutes: 60,
			AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"), CreatedByID: 1},
		{OrganizationID: 1, Name: "Loop C", Slug: "loop-c", Status: loop.StatusDisabled,
			ExecutionMode: loop.ExecutionModeAutopilot,
			PromptTemplate: "p",
			SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
			MaxConcurrentRuns: 1, TimeoutMinutes: 60,
			AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"), CreatedByID: 1},
		{OrganizationID: 1, Name: "Loop D", Slug: "loop-d", Status: loop.StatusArchived,
			ExecutionMode: loop.ExecutionModeDirect,
			PromptTemplate: "p",
			SandboxStrategy: loop.SandboxStrategyFresh, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
			MaxConcurrentRuns: 1, TimeoutMinutes: 60,
			AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"), CreatedByID: 1},
		{OrganizationID: 2, Name: "Other Org Loop", Slug: "other", Status: loop.StatusEnabled,
			ExecutionMode: loop.ExecutionModeAutopilot,
			PromptTemplate: "p",
			SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
			MaxConcurrentRuns: 1, TimeoutMinutes: 60,
			AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"), CreatedByID: 2},
	}
	for _, l := range loops {
		require.NoError(t, repo.Create(ctx, l))
	}

	t.Run("should list non-archived loops by default", func(t *testing.T) {
		result, total, err := repo.List(ctx, &loop.ListFilter{OrganizationID: 1})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total) // A, B, C (not D=archived)
		assert.Len(t, result, 3)
	})

	t.Run("should filter by status", func(t *testing.T) {
		result, total, err := repo.List(ctx, &loop.ListFilter{
			OrganizationID: 1,
			Status:         loop.StatusEnabled,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total) // A, B
		assert.Len(t, result, 2)
	})

	t.Run("should filter by execution mode", func(t *testing.T) {
		result, total, err := repo.List(ctx, &loop.ListFilter{
			OrganizationID: 1,
			ExecutionMode:  loop.ExecutionModeDirect,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total) // B (not D=archived)
		assert.Len(t, result, 1)
		assert.Equal(t, "loop-b", result[0].Slug)
	})

	t.Run("should filter by cron enabled", func(t *testing.T) {
		enabled := true
		result, _, err := repo.List(ctx, &loop.ListFilter{
			OrganizationID: 1,
			CronEnabled:    &enabled,
		})
		require.NoError(t, err)
		assert.Len(t, result, 1) // Only Loop A has cron
		assert.Equal(t, "loop-a", result[0].Slug)
	})

	t.Run("should respect limit and offset", func(t *testing.T) {
		result, total, err := repo.List(ctx, &loop.ListFilter{
			OrganizationID: 1,
			Limit:          2,
			Offset:         0,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total) // total count is unaffected
		assert.Len(t, result, 2)
	})

	t.Run("should isolate by organization", func(t *testing.T) {
		result, total, err := repo.List(ctx, &loop.ListFilter{OrganizationID: 2})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, result, 1)
		assert.Equal(t, "other", result[0].Slug)
	})
}

func TestLoopRepository_Update(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	l := &loop.Loop{
		OrganizationID: 1, Name: "Original", Slug: "original",
		PromptTemplate: "prompt",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, l))

	err := repo.Update(ctx, l.ID, map[string]interface{}{
		"name":            "Updated",
		"status":          loop.StatusDisabled,
		"total_runs":      5,
		"successful_runs": 3,
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)
	assert.Equal(t, loop.StatusDisabled, got.Status)
	assert.Equal(t, 5, got.TotalRuns)
	assert.Equal(t, 3, got.SuccessfulRuns)
}

func TestLoopRepository_Delete(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	l := &loop.Loop{
		OrganizationID: 1, Name: "To Delete", Slug: "to-delete",
		PromptTemplate: "prompt",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, l))

	t.Run("should delete existing loop", func(t *testing.T) {
		affected, err := repo.Delete(ctx, 1, "to-delete")
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)

		_, err = repo.GetBySlug(ctx, 1, "to-delete")
		assert.ErrorIs(t, err, loop.ErrNotFound)
	})

	t.Run("should return 0 affected for non-existent", func(t *testing.T) {
		affected, err := repo.Delete(ctx, 1, "no-such")
		require.NoError(t, err)
		assert.Equal(t, int64(0), affected)
	})
}

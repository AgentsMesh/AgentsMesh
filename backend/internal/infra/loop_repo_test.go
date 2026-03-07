package infra

import (
	"context"
	"testing"
	"time"

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

func TestLoopRepository_GetDueCronLoops(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	cron := "0 9 * * *"
	pastTime := time.Now().Add(-1 * time.Hour)
	futureTime := time.Now().Add(1 * time.Hour)

	// Due cron loop
	due := &loop.Loop{
		OrganizationID: 1, Name: "Due", Slug: "due",
		PromptTemplate: "prompt",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, due))

	// Not yet due
	notDue := &loop.Loop{
		OrganizationID: 1, Name: "Not Due", Slug: "not-due",
		PromptTemplate: "prompt",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		CronExpression: &cron, NextRunAt: &futureTime,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, notDue))

	// Disabled loop
	disabled := &loop.Loop{
		OrganizationID: 1, Name: "Disabled", Slug: "disabled",
		PromptTemplate: "prompt",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusDisabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, disabled))

	result, err := repo.GetDueCronLoops(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "due", result[0].Slug)
}

func TestLoopRepository_FindLoopsNeedingNextRun(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	cron := "0 9 * * *"
	pastTime := time.Now().Add(-1 * time.Hour)

	// Enabled cron loop with next_run_at IS NULL -> should be found
	needsInit := &loop.Loop{
		OrganizationID: 1, Name: "Needs Init", Slug: "needs-init",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		CronExpression: &cron, // next_run_at is nil
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, needsInit))

	// Enabled cron loop with next_run_at set -> should NOT be found
	hasNextRun := &loop.Loop{
		OrganizationID: 1, Name: "Has Next", Slug: "has-next",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, hasNextRun))

	// Disabled cron loop with next_run_at IS NULL -> should NOT be found
	disabled := &loop.Loop{
		OrganizationID: 1, Name: "Disabled", Slug: "disabled",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusDisabled,
		CronExpression: &cron,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, disabled))

	// API-only loop (no cron) -> should NOT be found
	apiOnly := &loop.Loop{
		OrganizationID: 1, Name: "API Only", Slug: "api-only",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, apiOnly))

	result, err := repo.FindLoopsNeedingNextRun(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "needs-init", result[0].Slug)
}

func TestLoopRepository_IncrementRunStats(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	l := &loop.Loop{
		OrganizationID: 1, Name: "Stats Loop", Slug: "stats-loop",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, l))

	now := time.Now()

	t.Run("should increment total and successful for completed", func(t *testing.T) {
		err := repo.IncrementRunStats(ctx, l.ID, loop.RunStatusCompleted, now)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, l.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, got.TotalRuns)
		assert.Equal(t, 1, got.SuccessfulRuns)
		assert.Equal(t, 0, got.FailedRuns)
	})

	t.Run("should increment total and failed for failed", func(t *testing.T) {
		err := repo.IncrementRunStats(ctx, l.ID, loop.RunStatusFailed, now)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, l.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, got.TotalRuns)
		assert.Equal(t, 1, got.SuccessfulRuns)
		assert.Equal(t, 1, got.FailedRuns)
	})

	t.Run("should increment total and failed for timeout", func(t *testing.T) {
		err := repo.IncrementRunStats(ctx, l.ID, loop.RunStatusTimeout, now)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, l.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, got.TotalRuns)
		assert.Equal(t, 1, got.SuccessfulRuns)
		assert.Equal(t, 2, got.FailedRuns)
	})

	t.Run("should only increment total for skipped", func(t *testing.T) {
		err := repo.IncrementRunStats(ctx, l.ID, loop.RunStatusSkipped, now)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, l.ID)
		require.NoError(t, err)
		assert.Equal(t, 4, got.TotalRuns)
		assert.Equal(t, 1, got.SuccessfulRuns)
		assert.Equal(t, 2, got.FailedRuns)
	})
}

// ========== Org-scoped filtering tests ==========

func TestLoopRepository_GetDueCronLoops_WithOrgFilter(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	cron := "0 9 * * *"
	pastTime := time.Now().Add(-1 * time.Hour)

	// Due loop in org 1
	org1Loop := &loop.Loop{
		OrganizationID: 1, Name: "Org1 Due", Slug: "org1-due",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, org1Loop))

	// Due loop in org 2
	org2Loop := &loop.Loop{
		OrganizationID: 2, Name: "Org2 Due", Slug: "org2-due",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 2,
	}
	require.NoError(t, repo.Create(ctx, org2Loop))

	// Due loop in org 3
	org3Loop := &loop.Loop{
		OrganizationID: 3, Name: "Org3 Due", Slug: "org3-due",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 3,
	}
	require.NoError(t, repo.Create(ctx, org3Loop))

	t.Run("nil orgIDs should return all due loops", func(t *testing.T) {
		result, err := repo.GetDueCronLoops(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("should filter to specific org", func(t *testing.T) {
		result, err := repo.GetDueCronLoops(ctx, []int64{1})
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "org1-due", result[0].Slug)
	})

	t.Run("should filter to multiple orgs", func(t *testing.T) {
		result, err := repo.GetDueCronLoops(ctx, []int64{1, 3})
		require.NoError(t, err)
		assert.Len(t, result, 2)
		slugs := []string{result[0].Slug, result[1].Slug}
		assert.ElementsMatch(t, []string{"org1-due", "org3-due"}, slugs)
	})

	t.Run("should return empty for non-matching orgs", func(t *testing.T) {
		result, err := repo.GetDueCronLoops(ctx, []int64{999})
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})
}

func TestLoopRepository_FindLoopsNeedingNextRun_WithOrgFilter(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	cron := "0 9 * * *"

	// Loop needing init in org 1
	org1 := &loop.Loop{
		OrganizationID: 1, Name: "Org1 Init", Slug: "org1-init",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		CronExpression: &cron,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, org1))

	// Loop needing init in org 2
	org2 := &loop.Loop{
		OrganizationID: 2, Name: "Org2 Init", Slug: "org2-init",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		CronExpression: &cron,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 2,
	}
	require.NoError(t, repo.Create(ctx, org2))

	t.Run("nil orgIDs should return all", func(t *testing.T) {
		result, err := repo.FindLoopsNeedingNextRun(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("should filter to specific org", func(t *testing.T) {
		result, err := repo.FindLoopsNeedingNextRun(ctx, []int64{2})
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "org2-init", result[0].Slug)
	})

	t.Run("should return empty for non-matching orgs", func(t *testing.T) {
		result, err := repo.FindLoopsNeedingNextRun(ctx, []int64{999})
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})
}

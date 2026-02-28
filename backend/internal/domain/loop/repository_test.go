package loop

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoopRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	loop := &Loop{
		OrganizationID: 1,
		Name:           "Test Loop",
		Slug:           "test-loop",
		PromptTemplate: "Review code in {{branch}}",
		ExecutionMode:  ExecutionModeAutopilot,
		Status:         StatusEnabled,
		SandboxStrategy:   SandboxStrategyPersistent,
		ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1,
		TimeoutMinutes:    60,
		AutopilotConfig:   []byte("{}"),
		ConfigOverrides:   []byte("{}"),
		CreatedByID:       1,
	}

	err := repo.Create(ctx, loop)
	require.NoError(t, err)
	assert.NotZero(t, loop.ID)
}

func TestLoopRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	// Seed
	loop := &Loop{
		OrganizationID: 1, Name: "Test", Slug: "test",
		PromptTemplate: "prompt",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, loop))

	t.Run("should return loop by ID", func(t *testing.T) {
		got, err := repo.GetByID(ctx, loop.ID)
		require.NoError(t, err)
		assert.Equal(t, "test", got.Slug)
		assert.Equal(t, "Test", got.Name)
	})

	t.Run("should return ErrNotFound for non-existent ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestLoopRepository_GetBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	loop := &Loop{
		OrganizationID: 1, Name: "My Loop", Slug: "my-loop",
		PromptTemplate: "prompt",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, loop))

	t.Run("should return loop by org_id and slug", func(t *testing.T) {
		got, err := repo.GetBySlug(ctx, 1, "my-loop")
		require.NoError(t, err)
		assert.Equal(t, "My Loop", got.Name)
	})

	t.Run("should return ErrNotFound for different org", func(t *testing.T) {
		_, err := repo.GetBySlug(ctx, 999, "my-loop")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("should return ErrNotFound for non-existent slug", func(t *testing.T) {
		_, err := repo.GetBySlug(ctx, 1, "no-such-loop")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestLoopRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	// Seed multiple loops
	cron := "0 9 * * *"
	loops := []*Loop{
		{OrganizationID: 1, Name: "Loop A", Slug: "loop-a", Status: StatusEnabled,
			ExecutionMode: ExecutionModeAutopilot, CronExpression: &cron,
			PromptTemplate: "p",
			SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
			MaxConcurrentRuns: 1, TimeoutMinutes: 60,
			AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"), CreatedByID: 1},
		{OrganizationID: 1, Name: "Loop B", Slug: "loop-b", Status: StatusEnabled,
			ExecutionMode: ExecutionModeDirect,
			PromptTemplate: "p",
			SandboxStrategy: SandboxStrategyFresh, ConcurrencyPolicy: ConcurrencyPolicySkip,
			MaxConcurrentRuns: 1, TimeoutMinutes: 60,
			AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"), CreatedByID: 1},
		{OrganizationID: 1, Name: "Loop C", Slug: "loop-c", Status: StatusDisabled,
			ExecutionMode: ExecutionModeAutopilot,
			PromptTemplate: "p",
			SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
			MaxConcurrentRuns: 1, TimeoutMinutes: 60,
			AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"), CreatedByID: 1},
		{OrganizationID: 1, Name: "Loop D", Slug: "loop-d", Status: StatusArchived,
			ExecutionMode: ExecutionModeDirect,
			PromptTemplate: "p",
			SandboxStrategy: SandboxStrategyFresh, ConcurrencyPolicy: ConcurrencyPolicySkip,
			MaxConcurrentRuns: 1, TimeoutMinutes: 60,
			AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"), CreatedByID: 1},
		{OrganizationID: 2, Name: "Other Org Loop", Slug: "other", Status: StatusEnabled,
			ExecutionMode: ExecutionModeAutopilot,
			PromptTemplate: "p",
			SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
			MaxConcurrentRuns: 1, TimeoutMinutes: 60,
			AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"), CreatedByID: 2},
	}
	for _, l := range loops {
		require.NoError(t, repo.Create(ctx, l))
	}

	t.Run("should list non-archived loops by default", func(t *testing.T) {
		result, total, err := repo.List(ctx, &ListFilter{OrganizationID: 1})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total) // A, B, C (not D=archived)
		assert.Len(t, result, 3)
	})

	t.Run("should filter by status", func(t *testing.T) {
		result, total, err := repo.List(ctx, &ListFilter{
			OrganizationID: 1,
			Status:         StatusEnabled,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total) // A, B
		assert.Len(t, result, 2)
	})

	t.Run("should filter by execution mode", func(t *testing.T) {
		result, total, err := repo.List(ctx, &ListFilter{
			OrganizationID: 1,
			ExecutionMode:  ExecutionModeDirect,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total) // B (not D=archived)
		assert.Len(t, result, 1)
		assert.Equal(t, "loop-b", result[0].Slug)
	})

	t.Run("should filter by cron enabled", func(t *testing.T) {
		enabled := true
		result, _, err := repo.List(ctx, &ListFilter{
			OrganizationID: 1,
			CronEnabled:    &enabled,
		})
		require.NoError(t, err)
		assert.Len(t, result, 1) // Only Loop A has cron
		assert.Equal(t, "loop-a", result[0].Slug)
	})

	t.Run("should respect limit and offset", func(t *testing.T) {
		result, total, err := repo.List(ctx, &ListFilter{
			OrganizationID: 1,
			Limit:          2,
			Offset:         0,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total) // total count is unaffected
		assert.Len(t, result, 2)
	})

	t.Run("should isolate by organization", func(t *testing.T) {
		result, total, err := repo.List(ctx, &ListFilter{OrganizationID: 2})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, result, 1)
		assert.Equal(t, "other", result[0].Slug)
	})
}

func TestLoopRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	loop := &Loop{
		OrganizationID: 1, Name: "Original", Slug: "original",
		PromptTemplate: "prompt",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, loop))

	err := repo.Update(ctx, loop.ID, map[string]interface{}{
		"name":           "Updated",
		"status":         StatusDisabled,
		"total_runs":     5,
		"successful_runs": 3,
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, loop.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)
	assert.Equal(t, StatusDisabled, got.Status)
	assert.Equal(t, 5, got.TotalRuns)
	assert.Equal(t, 3, got.SuccessfulRuns)
}

func TestLoopRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	loop := &Loop{
		OrganizationID: 1, Name: "To Delete", Slug: "to-delete",
		PromptTemplate: "prompt",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, loop))

	t.Run("should delete existing loop", func(t *testing.T) {
		affected, err := repo.Delete(ctx, 1, "to-delete")
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)

		_, err = repo.GetBySlug(ctx, 1, "to-delete")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("should return 0 affected for non-existent", func(t *testing.T) {
		affected, err := repo.Delete(ctx, 1, "no-such")
		require.NoError(t, err)
		assert.Equal(t, int64(0), affected)
	})
}

func TestLoopRepository_GetDueCronLoops(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	cron := "0 9 * * *"
	pastTime := time.Now().Add(-1 * time.Hour)
	futureTime := time.Now().Add(1 * time.Hour)

	// Due cron loop
	due := &Loop{
		OrganizationID: 1, Name: "Due", Slug: "due",
		PromptTemplate: "prompt",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, due))

	// Not yet due
	notDue := &Loop{
		OrganizationID: 1, Name: "Not Due", Slug: "not-due",
		PromptTemplate: "prompt",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		CronExpression: &cron, NextRunAt: &futureTime,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, notDue))

	// Disabled loop
	disabled := &Loop{
		OrganizationID: 1, Name: "Disabled", Slug: "disabled",
		PromptTemplate: "prompt",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusDisabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
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
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	cron := "0 9 * * *"
	pastTime := time.Now().Add(-1 * time.Hour)

	// Enabled cron loop with next_run_at IS NULL → should be found
	needsInit := &Loop{
		OrganizationID: 1, Name: "Needs Init", Slug: "needs-init",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		CronExpression: &cron, // next_run_at is nil
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, needsInit))

	// Enabled cron loop with next_run_at set → should NOT be found
	hasNextRun := &Loop{
		OrganizationID: 1, Name: "Has Next", Slug: "has-next",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, hasNextRun))

	// Disabled cron loop with next_run_at IS NULL → should NOT be found
	disabled := &Loop{
		OrganizationID: 1, Name: "Disabled", Slug: "disabled",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusDisabled,
		CronExpression: &cron,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, disabled))

	// API-only loop (no cron) → should NOT be found
	apiOnly := &Loop{
		OrganizationID: 1, Name: "API Only", Slug: "api-only",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
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
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	loop := &Loop{
		OrganizationID: 1, Name: "Stats Loop", Slug: "stats-loop",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, loop))

	now := time.Now()

	t.Run("should increment total and successful for completed", func(t *testing.T) {
		err := repo.IncrementRunStats(ctx, loop.ID, RunStatusCompleted, now)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, loop.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, got.TotalRuns)
		assert.Equal(t, 1, got.SuccessfulRuns)
		assert.Equal(t, 0, got.FailedRuns)
	})

	t.Run("should increment total and failed for failed", func(t *testing.T) {
		err := repo.IncrementRunStats(ctx, loop.ID, RunStatusFailed, now)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, loop.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, got.TotalRuns)
		assert.Equal(t, 1, got.SuccessfulRuns)
		assert.Equal(t, 1, got.FailedRuns)
	})

	t.Run("should increment total and failed for timeout", func(t *testing.T) {
		err := repo.IncrementRunStats(ctx, loop.ID, RunStatusTimeout, now)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, loop.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, got.TotalRuns)
		assert.Equal(t, 1, got.SuccessfulRuns)
		assert.Equal(t, 2, got.FailedRuns)
	})

	t.Run("should only increment total for skipped", func(t *testing.T) {
		err := repo.IncrementRunStats(ctx, loop.ID, RunStatusSkipped, now)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, loop.ID)
		require.NoError(t, err)
		assert.Equal(t, 4, got.TotalRuns)
		assert.Equal(t, 1, got.SuccessfulRuns)
		assert.Equal(t, 2, got.FailedRuns)
	})
}

// ========== Org-scoped filtering tests ==========

func TestLoopRepository_GetDueCronLoops_WithOrgFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	cron := "0 9 * * *"
	pastTime := time.Now().Add(-1 * time.Hour)

	// Due loop in org 1
	org1Loop := &Loop{
		OrganizationID: 1, Name: "Org1 Due", Slug: "org1-due",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, org1Loop))

	// Due loop in org 2
	org2Loop := &Loop{
		OrganizationID: 2, Name: "Org2 Due", Slug: "org2-due",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 2,
	}
	require.NoError(t, repo.Create(ctx, org2Loop))

	// Due loop in org 3
	org3Loop := &Loop{
		OrganizationID: 3, Name: "Org3 Due", Slug: "org3-due",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		CronExpression: &cron, NextRunAt: &pastTime,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
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
	db := setupTestDB(t)
	repo := NewLoopRepository(db)
	ctx := context.Background()

	cron := "0 9 * * *"

	// Loop needing init in org 1
	org1 := &Loop{
		OrganizationID: 1, Name: "Org1 Init", Slug: "org1-init",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		CronExpression: &cron,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, repo.Create(ctx, org1))

	// Loop needing init in org 2
	org2 := &Loop{
		OrganizationID: 2, Name: "Org2 Init", Slug: "org2-init",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		CronExpression: &cron,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
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

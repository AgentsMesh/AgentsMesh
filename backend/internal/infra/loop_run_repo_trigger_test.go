package infra

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRepository_BatchGetPodStatuses(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	now := time.Now()
	db.Exec(`INSERT INTO pods (pod_key, status, finished_at) VALUES (?, ?, ?)`, "bp-1", "completed", now)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES (?, ?)`, "bp-2", "running")

	t.Run("should return statuses for known pod keys", func(t *testing.T) {
		results, err := repo.BatchGetPodStatuses(ctx, []string{"bp-1", "bp-2", "bp-unknown"})
		require.NoError(t, err)
		assert.Len(t, results, 2)

		statusMap := make(map[string]string)
		for _, r := range results {
			statusMap[r.PodKey] = r.Status
		}
		assert.Equal(t, "completed", statusMap["bp-1"])
		assert.Equal(t, "running", statusMap["bp-2"])
	})

	t.Run("should return nil for empty keys", func(t *testing.T) {
		results, err := repo.BatchGetPodStatuses(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, results)
	})
}

func TestRunRepository_BatchGetAutopilotPhases(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	db.Exec(`INSERT INTO autopilot_controllers (autopilot_controller_key, phase) VALUES ('ba-1', 'completed')`)
	db.Exec(`INSERT INTO autopilot_controllers (autopilot_controller_key, phase) VALUES ('ba-2', 'running')`)

	t.Run("should return phases for known autopilot keys", func(t *testing.T) {
		result, err := repo.BatchGetAutopilotPhases(ctx, []string{"ba-1", "ba-2", "ba-unknown"})
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "completed", result["ba-1"])
		assert.Equal(t, "running", result["ba-2"])
	})

	t.Run("should return nil for empty keys", func(t *testing.T) {
		result, err := repo.BatchGetAutopilotPhases(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

// TestRunRepository_TriggerRunAtomic tests the atomic run creation with concurrency check.
func TestRunRepository_TriggerRunAtomic(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRunRepository(db)
	loopRepo := NewLoopRepository(db)
	ctx := context.Background()

	l := &loop.Loop{
		OrganizationID: 1, Name: "Atomic Loop", Slug: "atomic-loop",
		PromptTemplate: "Do the thing",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, loopRepo.Create(ctx, l))

	t.Run("should create run atomically", func(t *testing.T) {
		result, err := repo.TriggerRunAtomic(ctx, &loop.TriggerRunAtomicParams{
			LoopID:        l.ID,
			TriggerType:   loop.RunTriggerManual,
			TriggerSource: "test",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Skipped)
		assert.NotNil(t, result.Run)
		assert.Equal(t, 1, result.Run.RunNumber)
		assert.Equal(t, loop.RunStatusPending, result.Run.Status)
		assert.Equal(t, loop.RunTriggerManual, result.Run.TriggerType)
		assert.NotNil(t, result.Run.ResolvedPrompt)
		assert.Equal(t, "Do the thing", *result.Run.ResolvedPrompt)
		assert.NotNil(t, result.Run.StartedAt)
		assert.NotNil(t, result.Loop)
		assert.Equal(t, l.ID, result.Loop.ID)
	})

	t.Run("should increment run number", func(t *testing.T) {
		result, err := repo.TriggerRunAtomic(ctx, &loop.TriggerRunAtomicParams{
			LoopID:        l.ID,
			TriggerType:   loop.RunTriggerCron,
			TriggerSource: "cron",
		})
		require.NoError(t, err)
		assert.Equal(t, 2, result.Run.RunNumber)
	})

	t.Run("should return ErrNotFound for non-existent loop", func(t *testing.T) {
		_, err := repo.TriggerRunAtomic(ctx, &loop.TriggerRunAtomicParams{
			LoopID:      99999,
			TriggerType: loop.RunTriggerManual,
		})
		assert.ErrorIs(t, err, loop.ErrNotFound)
	})

	t.Run("should return error for disabled loop", func(t *testing.T) {
		require.NoError(t, loopRepo.Update(ctx, l.ID, map[string]interface{}{
			"status": loop.StatusDisabled,
		}))

		_, err := repo.TriggerRunAtomic(ctx, &loop.TriggerRunAtomicParams{
			LoopID:      l.ID,
			TriggerType: loop.RunTriggerManual,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")

		require.NoError(t, loopRepo.Update(ctx, l.ID, map[string]interface{}{
			"status": loop.StatusEnabled,
		}))
	})
}

// TestRunRepository_TriggerRunAtomic_ConcurrencySkip tests the skip concurrency policy.
func TestRunRepository_TriggerRunAtomic_ConcurrencySkip(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRunRepository(db)
	loopRepo := NewLoopRepository(db)
	ctx := context.Background()

	l := &loop.Loop{
		OrganizationID: 1, Name: "Skip Loop", Slug: "skip-loop",
		PromptTemplate: "prompt",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, loopRepo.Create(ctx, l))

	pendingRun := &loop.LoopRun{
		OrganizationID: 1, LoopID: l.ID, RunNumber: 1,
		Status: loop.RunStatusPending, TriggerType: loop.RunTriggerManual,
	}
	require.NoError(t, repo.Create(ctx, pendingRun))

	result, err := repo.TriggerRunAtomic(ctx, &loop.TriggerRunAtomicParams{
		LoopID:        l.ID,
		TriggerType:   loop.RunTriggerCron,
		TriggerSource: "cron",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Skipped)
	assert.Equal(t, "max concurrent runs reached", result.Reason)
	assert.NotNil(t, result.Run)
	assert.Equal(t, loop.RunStatusSkipped, result.Run.Status)
	assert.Equal(t, 2, result.Run.RunNumber)
}

// TestRunRepository_GetTimedOutRuns_OrgFilter tests org filtering for timed-out runs.
func TestRunRepository_GetTimedOutRuns_OrgFilter(t *testing.T) {
	t.Skip("Requires PostgreSQL (uses ::INTERVAL cast syntax). Org filtering tested via GetDueCronLoops_WithOrgFilter.")
}

// TestRunRepository_FinishRun tests the atomic finish with optimistic locking.
func TestRunRepository_FinishRun(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRunRepository(db)
	loopRepo := NewLoopRepository(db)
	ctx := context.Background()

	l := &loop.Loop{
		OrganizationID: 1, Name: "Finish Loop", Slug: "finish-loop",
		PromptTemplate: "p",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, loopRepo.Create(ctx, l))

	now := time.Now()

	t.Run("should finish an unfinished run", func(t *testing.T) {
		run := &loop.LoopRun{
			OrganizationID: 1, LoopID: l.ID, RunNumber: 100,
			Status: loop.RunStatusRunning, TriggerType: loop.RunTriggerManual,
			PodKey: loopStrPtr("finish-pod-1"),
		}
		require.NoError(t, repo.Create(ctx, run))

		updated, err := repo.FinishRun(ctx, run.ID, map[string]interface{}{
			"status":      loop.RunStatusCompleted,
			"finished_at": now,
		})
		require.NoError(t, err)
		assert.True(t, updated, "should update unfinished run")

		fetched, err := repo.GetByID(ctx, run.ID)
		require.NoError(t, err)
		assert.Equal(t, loop.RunStatusCompleted, fetched.Status)
		assert.NotNil(t, fetched.FinishedAt)
	})

	t.Run("should not finish an already-finished run (idempotency)", func(t *testing.T) {
		run := &loop.LoopRun{
			OrganizationID: 1, LoopID: l.ID, RunNumber: 101,
			Status: loop.RunStatusCompleted, TriggerType: loop.RunTriggerManual,
			PodKey:     loopStrPtr("finish-pod-2"),
			FinishedAt: &now,
		}
		require.NoError(t, repo.Create(ctx, run))

		updated, err := repo.FinishRun(ctx, run.ID, map[string]interface{}{
			"status":      loop.RunStatusFailed,
			"finished_at": now,
		})
		require.NoError(t, err)
		assert.False(t, updated, "should not update already-finished run")

		fetched, err := repo.GetByID(ctx, run.ID)
		require.NoError(t, err)
		assert.Equal(t, loop.RunStatusCompleted, fetched.Status, "status should remain completed")
	})

	t.Run("should return false for non-existent run", func(t *testing.T) {
		updated, err := repo.FinishRun(ctx, 99999, map[string]interface{}{
			"status":      loop.RunStatusFailed,
			"finished_at": now,
		})
		require.NoError(t, err)
		assert.False(t, updated, "should return false for non-existent run")
	})
}

// TestRunRepository_TriggerRunAtomic_TerminatedPodFreesSlot tests that terminated pods
// don't count as active.
func TestRunRepository_TriggerRunAtomic_TerminatedPodFreesSlot(t *testing.T) {
	db := setupLoopTestDB(t)
	repo := NewLoopRunRepository(db)
	loopRepo := NewLoopRepository(db)
	ctx := context.Background()

	l := &loop.Loop{
		OrganizationID: 1, Name: "Free Slot", Slug: "free-slot",
		PromptTemplate: "prompt",
		ExecutionMode: loop.ExecutionModeAutopilot, Status: loop.StatusEnabled,
		SandboxStrategy: loop.SandboxStrategyPersistent, ConcurrencyPolicy: loop.ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, loopRepo.Create(ctx, l))

	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('term-pod', 'terminated')`)

	run := &loop.LoopRun{
		OrganizationID: 1, LoopID: l.ID, RunNumber: 1,
		Status: loop.RunStatusRunning, TriggerType: loop.RunTriggerManual,
		PodKey: loopStrPtr("term-pod"),
	}
	require.NoError(t, repo.Create(ctx, run))

	result, err := repo.TriggerRunAtomic(ctx, &loop.TriggerRunAtomicParams{
		LoopID:        l.ID,
		TriggerType:   loop.RunTriggerManual,
		TriggerSource: "test",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Skipped, "terminated pod should free the concurrency slot")
	assert.Equal(t, 2, result.Run.RunNumber)
}

package loop

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	// Seed a parent loop
	loopRepo := NewLoopRepository(db)
	loop := &Loop{
		OrganizationID: 1, Name: "Parent", Slug: "parent",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, loopRepo.Create(ctx, loop))

	run := &LoopRun{
		OrganizationID: 1,
		LoopID:         loop.ID,
		RunNumber:      1,
		Status:         RunStatusPending,
		TriggerType:    RunTriggerManual,
	}
	err := repo.Create(ctx, run)
	require.NoError(t, err)
	assert.NotZero(t, run.ID)
}

func TestRunRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	run := &LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status: RunStatusPending, TriggerType: RunTriggerManual,
	}
	require.NoError(t, repo.Create(ctx, run))

	t.Run("should return run by ID", func(t *testing.T) {
		got, err := repo.GetByID(ctx, run.ID)
		require.NoError(t, err)
		assert.Equal(t, RunStatusPending, got.Status)
		assert.Equal(t, 1, got.RunNumber)
	})

	t.Run("should return ErrNotFound for non-existent", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestRunRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	// Seed runs
	for i := 1; i <= 5; i++ {
		run := &LoopRun{
			OrganizationID: 1, LoopID: 1, RunNumber: i,
			Status: RunStatusCompleted, TriggerType: RunTriggerCron,
		}
		require.NoError(t, repo.Create(ctx, run))
	}
	// Different loop
	run := &LoopRun{
		OrganizationID: 1, LoopID: 2, RunNumber: 1,
		Status: RunStatusPending, TriggerType: RunTriggerAPI,
	}
	require.NoError(t, repo.Create(ctx, run))

	t.Run("should list runs for specific loop", func(t *testing.T) {
		result, total, err := repo.List(ctx, &RunListFilter{LoopID: 1})
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, result, 5)
	})

	t.Run("should respect limit", func(t *testing.T) {
		result, total, err := repo.List(ctx, &RunListFilter{LoopID: 1, Limit: 2})
		require.NoError(t, err)
		assert.Equal(t, int64(5), total) // total unaffected
		assert.Len(t, result, 2)
	})

	t.Run("should isolate by loop_id", func(t *testing.T) {
		result, total, err := repo.List(ctx, &RunListFilter{LoopID: 2})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, result, 1)
	})
}

func TestRunRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	run := &LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status: RunStatusPending, TriggerType: RunTriggerManual,
	}
	require.NoError(t, repo.Create(ctx, run))

	podKey := "pod-123"
	err := repo.Update(ctx, run.ID, map[string]interface{}{
		"status":  RunStatusRunning,
		"pod_key": podKey,
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, RunStatusRunning, got.Status)
	assert.Equal(t, &podKey, got.PodKey)
}

func TestRunRepository_GetMaxRunNumber(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	t.Run("should return 0 for no runs", func(t *testing.T) {
		max, err := repo.GetMaxRunNumber(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, 0, max)
	})

	// Seed runs
	for i := 1; i <= 3; i++ {
		run := &LoopRun{
			OrganizationID: 1, LoopID: 1, RunNumber: i,
			Status: RunStatusCompleted, TriggerType: RunTriggerCron,
		}
		require.NoError(t, repo.Create(ctx, run))
	}

	t.Run("should return max run number", func(t *testing.T) {
		max, err := repo.GetMaxRunNumber(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, 3, max)
	})

	t.Run("should be scoped to loop_id", func(t *testing.T) {
		max, err := repo.GetMaxRunNumber(ctx, 999)
		require.NoError(t, err)
		assert.Equal(t, 0, max)
	})
}

func TestRunRepository_GetByAutopilotKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	apKey := "ap-ctrl-123"
	run := &LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status: RunStatusRunning, TriggerType: RunTriggerManual,
		AutopilotControllerKey: &apKey,
	}
	require.NoError(t, repo.Create(ctx, run))

	t.Run("should find run by autopilot key", func(t *testing.T) {
		got, err := repo.GetByAutopilotKey(ctx, "ap-ctrl-123")
		require.NoError(t, err)
		assert.Equal(t, run.ID, got.ID)
	})

	t.Run("should return ErrNotFound for unknown key", func(t *testing.T) {
		_, err := repo.GetByAutopilotKey(ctx, "unknown-key")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

// TestRunRepository_CountActiveRuns tests the SSOT-based active run counting.
// Active runs are determined by Pod status (JOIN with pods table).
func TestRunRepository_CountActiveRuns(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	// Seed pods with different statuses
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('pod-running', 'running')`)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('pod-init', 'initializing')`)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('pod-done', 'completed')`)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('pod-err', 'error')`)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('pod-term', 'terminated')`)

	// Runs with Pods
	runs := []LoopRun{
		{OrganizationID: 1, LoopID: 1, RunNumber: 1, Status: RunStatusRunning,
			TriggerType: RunTriggerManual, PodKey: strPtr("pod-running")},
		{OrganizationID: 1, LoopID: 1, RunNumber: 2, Status: RunStatusRunning,
			TriggerType: RunTriggerManual, PodKey: strPtr("pod-init")},
		{OrganizationID: 1, LoopID: 1, RunNumber: 3, Status: RunStatusRunning,
			TriggerType: RunTriggerManual, PodKey: strPtr("pod-done")},
		{OrganizationID: 1, LoopID: 1, RunNumber: 4, Status: RunStatusRunning,
			TriggerType: RunTriggerManual, PodKey: strPtr("pod-err")},
		{OrganizationID: 1, LoopID: 1, RunNumber: 5, Status: RunStatusRunning,
			TriggerType: RunTriggerManual, PodKey: strPtr("pod-term")},
		// Pending run (no Pod yet) — counts as active
		{OrganizationID: 1, LoopID: 1, RunNumber: 6, Status: RunStatusPending,
			TriggerType: RunTriggerManual},
		// Skipped run (no Pod) — does NOT count as active
		{OrganizationID: 1, LoopID: 1, RunNumber: 7, Status: RunStatusSkipped,
			TriggerType: RunTriggerManual},
	}
	for i := range runs {
		require.NoError(t, repo.Create(ctx, &runs[i]))
	}

	count, err := repo.CountActiveRuns(ctx, 1)
	require.NoError(t, err)
	// Active: pod-running, pod-init (active pods) + pending (no pod) = 3
	// Inactive: pod-done (completed), pod-err (error), pod-term (terminated), skipped
	assert.Equal(t, int64(3), count)
}

// TestRunRepository_GetActiveRunByPodKey tests finding active runs by pod key via SSOT JOIN.
func TestRunRepository_GetActiveRunByPodKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('active-pod', 'running')`)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('done-pod', 'completed')`)

	run1 := &LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status: RunStatusRunning, TriggerType: RunTriggerManual,
		PodKey: strPtr("active-pod"),
	}
	// GetActiveRunByPodKey uses "finished_at IS NULL" as the guard (not Pod status).
	// Set finished_at on the "done" run so the method correctly excludes it.
	finishedAt := time.Now()
	run2 := &LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 2,
		Status: RunStatusCompleted, TriggerType: RunTriggerManual,
		PodKey: strPtr("done-pod"),
		FinishedAt: &finishedAt,
	}
	require.NoError(t, repo.Create(ctx, run1))
	require.NoError(t, repo.Create(ctx, run2))

	t.Run("should find run with active pod", func(t *testing.T) {
		got, err := repo.GetActiveRunByPodKey(ctx, "active-pod")
		require.NoError(t, err)
		assert.Equal(t, run1.ID, got.ID)
	})

	t.Run("should not find run with completed pod", func(t *testing.T) {
		_, err := repo.GetActiveRunByPodKey(ctx, "done-pod")
		assert.Error(t, err)
	})
}

// TestRunRepository_ComputeLoopStats tests the SSOT statistics computation.
// Stats are derived from Pod/Autopilot status, not from the run's own status field.
func TestRunRepository_ComputeLoopStats(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	// Seed pods
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('stat-completed', 'completed')`)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('stat-terminated', 'terminated')`)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('stat-error', 'error')`)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('stat-running', 'running')`)

	// Seed autopilot controllers
	db.Exec(`INSERT INTO autopilot_controllers (autopilot_controller_key, phase) VALUES ('ap-completed', 'completed')`)
	db.Exec(`INSERT INTO autopilot_controllers (autopilot_controller_key, phase) VALUES ('ap-failed', 'failed')`)
	db.Exec(`INSERT INTO autopilot_controllers (autopilot_controller_key, phase) VALUES ('ap-stopped', 'stopped')`)

	runs := []LoopRun{
		// Direct mode: completed pod → completed
		{OrganizationID: 1, LoopID: 1, RunNumber: 1, Status: RunStatusRunning,
			TriggerType: RunTriggerCron, PodKey: strPtr("stat-completed")},
		// Direct mode: terminated pod → cancelled (killed)
		{OrganizationID: 1, LoopID: 1, RunNumber: 2, Status: RunStatusRunning,
			TriggerType: RunTriggerCron, PodKey: strPtr("stat-terminated")},
		// Direct mode: error pod → failed
		{OrganizationID: 1, LoopID: 1, RunNumber: 3, Status: RunStatusRunning,
			TriggerType: RunTriggerCron, PodKey: strPtr("stat-error")},
		// Direct mode: running pod → running (not counted as success/fail)
		{OrganizationID: 1, LoopID: 1, RunNumber: 4, Status: RunStatusRunning,
			TriggerType: RunTriggerCron, PodKey: strPtr("stat-running")},
		// No pod: skipped → skipped (counted in total, not success/fail)
		{OrganizationID: 1, LoopID: 1, RunNumber: 5, Status: RunStatusSkipped,
			TriggerType: RunTriggerCron},
		// Autopilot mode: ap completed → completed
		{OrganizationID: 1, LoopID: 1, RunNumber: 6, Status: RunStatusRunning,
			TriggerType: RunTriggerCron, PodKey: strPtr("stat-running"),
			AutopilotControllerKey: strPtr("ap-completed")},
		// Autopilot mode: ap failed → failed
		{OrganizationID: 1, LoopID: 1, RunNumber: 7, Status: RunStatusRunning,
			TriggerType: RunTriggerCron, PodKey: strPtr("stat-running"),
			AutopilotControllerKey: strPtr("ap-failed")},
		// Autopilot mode: ap stopped → cancelled
		{OrganizationID: 1, LoopID: 1, RunNumber: 8, Status: RunStatusRunning,
			TriggerType: RunTriggerCron, PodKey: strPtr("stat-running"),
			AutopilotControllerKey: strPtr("ap-stopped")},
	}
	for i := range runs {
		require.NoError(t, repo.Create(ctx, &runs[i]))
	}

	total, successful, failed, err := repo.ComputeLoopStats(ctx, 1)
	require.NoError(t, err)

	// Total: 8
	assert.Equal(t, 8, total)
	// Successful: completed(1) + ap-completed(6) = 2
	assert.Equal(t, 2, successful)
	// Failed (includes cancelled): terminated→cancelled(2) + error(3) + ap-failed(7) + ap-stopped→cancelled(8) = 4
	assert.Equal(t, 4, failed)
}

// TestRunRepository_ComputeLoopStats_PodWinsOverAutopilot tests the edge case
// where Pod is terminal but autopilot phase is still active.
func TestRunRepository_ComputeLoopStats_PodWinsOverAutopilot(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	// Pod is completed, but autopilot is still in "running" phase
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('pod-wins', 'completed')`)
	db.Exec(`INSERT INTO autopilot_controllers (autopilot_controller_key, phase) VALUES ('ap-stale', 'running')`)

	run := &LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status: RunStatusRunning, TriggerType: RunTriggerManual,
		PodKey:                 strPtr("pod-wins"),
		AutopilotControllerKey: strPtr("ap-stale"),
	}
	require.NoError(t, repo.Create(ctx, run))

	total, successful, failed, err := repo.ComputeLoopStats(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, successful, "Pod terminal (completed) should win over autopilot active (running)")
	assert.Equal(t, 0, failed)
}

func TestRunRepository_GetLatestPodKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	ctx := context.Background()

	t.Run("should return nil when no runs exist", func(t *testing.T) {
		result := repo.GetLatestPodKey(ctx, 1)
		assert.Nil(t, result)
	})

	t.Run("should return nil when runs have no pod_key", func(t *testing.T) {
		run := &LoopRun{
			OrganizationID: 1, LoopID: 1, RunNumber: 1,
			Status: RunStatusSkipped, TriggerType: RunTriggerCron,
		}
		require.NoError(t, repo.Create(ctx, run))

		result := repo.GetLatestPodKey(ctx, 1)
		assert.Nil(t, result)
	})

	t.Run("should return latest pod_key", func(t *testing.T) {
		run1 := &LoopRun{
			OrganizationID: 1, LoopID: 2, RunNumber: 1,
			Status: RunStatusCompleted, TriggerType: RunTriggerManual,
			PodKey: strPtr("old-pod"),
		}
		run2 := &LoopRun{
			OrganizationID: 1, LoopID: 2, RunNumber: 2,
			Status: RunStatusCompleted, TriggerType: RunTriggerManual,
			PodKey: strPtr("latest-pod"),
		}
		require.NoError(t, repo.Create(ctx, run1))
		require.NoError(t, repo.Create(ctx, run2))

		result := repo.GetLatestPodKey(ctx, 2)
		require.NotNil(t, result)
		assert.Equal(t, "latest-pod", *result)
	})
}

func TestRunRepository_BatchGetPodStatuses(t *testing.T) {
	db := setupTestDB(t)
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
	db := setupTestDB(t)
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
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	loopRepo := NewLoopRepository(db)
	ctx := context.Background()

	// Seed a loop
	loop := &Loop{
		OrganizationID: 1, Name: "Atomic Loop", Slug: "atomic-loop",
		PromptTemplate: "Do the thing",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, loopRepo.Create(ctx, loop))

	t.Run("should create run atomically", func(t *testing.T) {
		result, err := repo.TriggerRunAtomic(ctx, &TriggerRunAtomicParams{
			LoopID:        loop.ID,
			TriggerType:   RunTriggerManual,
			TriggerSource: "test",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Skipped)
		assert.NotNil(t, result.Run)
		assert.Equal(t, 1, result.Run.RunNumber)
		assert.Equal(t, RunStatusPending, result.Run.Status)
		assert.Equal(t, RunTriggerManual, result.Run.TriggerType)
		assert.NotNil(t, result.Run.ResolvedPrompt)
		assert.Equal(t, "Do the thing", *result.Run.ResolvedPrompt)
		assert.NotNil(t, result.Run.StartedAt)
		assert.NotNil(t, result.Loop)
		assert.Equal(t, loop.ID, result.Loop.ID)
	})

	t.Run("should increment run number", func(t *testing.T) {
		result, err := repo.TriggerRunAtomic(ctx, &TriggerRunAtomicParams{
			LoopID:        loop.ID,
			TriggerType:   RunTriggerCron,
			TriggerSource: "cron",
		})
		require.NoError(t, err)
		assert.Equal(t, 2, result.Run.RunNumber)
	})

	t.Run("should return ErrNotFound for non-existent loop", func(t *testing.T) {
		_, err := repo.TriggerRunAtomic(ctx, &TriggerRunAtomicParams{
			LoopID:      99999,
			TriggerType: RunTriggerManual,
		})
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("should return error for disabled loop", func(t *testing.T) {
		// Disable the loop
		require.NoError(t, loopRepo.Update(ctx, loop.ID, map[string]interface{}{
			"status": StatusDisabled,
		}))

		_, err := repo.TriggerRunAtomic(ctx, &TriggerRunAtomicParams{
			LoopID:      loop.ID,
			TriggerType: RunTriggerManual,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")

		// Re-enable for subsequent tests
		require.NoError(t, loopRepo.Update(ctx, loop.ID, map[string]interface{}{
			"status": StatusEnabled,
		}))
	})
}

// TestRunRepository_TriggerRunAtomic_ConcurrencySkip tests the skip concurrency policy.
func TestRunRepository_TriggerRunAtomic_ConcurrencySkip(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	loopRepo := NewLoopRepository(db)
	ctx := context.Background()

	// Seed a loop with max_concurrent_runs=1, policy=skip
	loop := &Loop{
		OrganizationID: 1, Name: "Skip Loop", Slug: "skip-loop",
		PromptTemplate: "prompt",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, loopRepo.Create(ctx, loop))

	// Create a pending run (active, no pod_key)
	pendingRun := &LoopRun{
		OrganizationID: 1, LoopID: loop.ID, RunNumber: 1,
		Status: RunStatusPending, TriggerType: RunTriggerManual,
	}
	require.NoError(t, repo.Create(ctx, pendingRun))

	// Trigger should be skipped
	result, err := repo.TriggerRunAtomic(ctx, &TriggerRunAtomicParams{
		LoopID:        loop.ID,
		TriggerType:   RunTriggerCron,
		TriggerSource: "cron",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Skipped)
	assert.Equal(t, "max concurrent runs reached", result.Reason)
	assert.NotNil(t, result.Run)
	assert.Equal(t, RunStatusSkipped, result.Run.Status)
	assert.Equal(t, 2, result.Run.RunNumber)
}

// TestRunRepository_GetTimedOutRuns_OrgFilter tests the org filtering for timed-out runs.
// NOTE: The full GetTimedOutRuns query uses PostgreSQL-specific syntax
// (NOW() - (timeout_minutes || ' minutes')::INTERVAL) which is not supported by SQLite.
// The org filtering pattern (WHERE organization_id IN ?) is identical to
// GetDueCronLoops/FindLoopsNeedingNextRun and is thoroughly tested there.
// Full integration tests should use PostgreSQL.
func TestRunRepository_GetTimedOutRuns_OrgFilter(t *testing.T) {
	t.Skip("Requires PostgreSQL (uses ::INTERVAL cast syntax). Org filtering tested via GetDueCronLoops_WithOrgFilter.")
}

// TestRunRepository_FinishRun tests the atomic finish with optimistic locking.
func TestRunRepository_FinishRun(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	loopRepo := NewLoopRepository(db)
	ctx := context.Background()

	// Seed a loop
	loop := &Loop{
		OrganizationID: 1, Name: "Finish Loop", Slug: "finish-loop",
		PromptTemplate: "p",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, loopRepo.Create(ctx, loop))

	now := time.Now()

	t.Run("should finish an unfinished run", func(t *testing.T) {
		run := &LoopRun{
			OrganizationID: 1, LoopID: loop.ID, RunNumber: 100,
			Status: RunStatusRunning, TriggerType: RunTriggerManual,
			PodKey: strPtr("finish-pod-1"),
		}
		require.NoError(t, repo.Create(ctx, run))

		updated, err := repo.FinishRun(ctx, run.ID, map[string]interface{}{
			"status":      RunStatusCompleted,
			"finished_at": now,
		})
		require.NoError(t, err)
		assert.True(t, updated, "should update unfinished run")

		// Verify the row was updated
		fetched, err := repo.GetByID(ctx, run.ID)
		require.NoError(t, err)
		assert.Equal(t, RunStatusCompleted, fetched.Status)
		assert.NotNil(t, fetched.FinishedAt)
	})

	t.Run("should not finish an already-finished run (idempotency)", func(t *testing.T) {
		run := &LoopRun{
			OrganizationID: 1, LoopID: loop.ID, RunNumber: 101,
			Status: RunStatusCompleted, TriggerType: RunTriggerManual,
			PodKey:     strPtr("finish-pod-2"),
			FinishedAt: &now,
		}
		require.NoError(t, repo.Create(ctx, run))

		updated, err := repo.FinishRun(ctx, run.ID, map[string]interface{}{
			"status":      RunStatusFailed,
			"finished_at": now,
		})
		require.NoError(t, err)
		assert.False(t, updated, "should not update already-finished run")

		// Verify the row was NOT changed
		fetched, err := repo.GetByID(ctx, run.ID)
		require.NoError(t, err)
		assert.Equal(t, RunStatusCompleted, fetched.Status, "status should remain completed")
	})

	t.Run("should return false for non-existent run", func(t *testing.T) {
		updated, err := repo.FinishRun(ctx, 99999, map[string]interface{}{
			"status":      RunStatusFailed,
			"finished_at": now,
		})
		require.NoError(t, err)
		assert.False(t, updated, "should return false for non-existent run")
	})
}

// TestRunRepository_TriggerRunAtomic_TerminatedPodFreesSlot tests that terminated pods
// don't count as active, allowing new runs even when max_concurrent_runs is reached
// based on stored status.
func TestRunRepository_TriggerRunAtomic_TerminatedPodFreesSlot(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLoopRunRepository(db)
	loopRepo := NewLoopRepository(db)
	ctx := context.Background()

	// Seed a loop with max_concurrent_runs=1
	loop := &Loop{
		OrganizationID: 1, Name: "Free Slot", Slug: "free-slot",
		PromptTemplate: "prompt",
		ExecutionMode: ExecutionModeAutopilot, Status: StatusEnabled,
		SandboxStrategy: SandboxStrategyPersistent, ConcurrencyPolicy: ConcurrencyPolicySkip,
		MaxConcurrentRuns: 1, TimeoutMinutes: 60,
		AutopilotConfig: []byte("{}"), ConfigOverrides: []byte("{}"),
		CreatedByID: 1,
	}
	require.NoError(t, loopRepo.Create(ctx, loop))

	// Create a terminated pod
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('term-pod', 'terminated')`)

	// Create a "running" run that points to a terminated pod
	run := &LoopRun{
		OrganizationID: 1, LoopID: loop.ID, RunNumber: 1,
		Status: RunStatusRunning, TriggerType: RunTriggerManual,
		PodKey: strPtr("term-pod"),
	}
	require.NoError(t, repo.Create(ctx, run))

	// Should NOT be skipped — the terminated pod frees the slot (SSOT)
	result, err := repo.TriggerRunAtomic(ctx, &TriggerRunAtomicParams{
		LoopID:        loop.ID,
		TriggerType:   RunTriggerManual,
		TriggerSource: "test",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Skipped, "terminated pod should free the concurrency slot")
	assert.Equal(t, 2, result.Run.RunNumber)
}

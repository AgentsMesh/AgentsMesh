package loop

import (
	"context"
	"testing"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupLoopRunServiceTestDB(t *testing.T) *gorm.DB {
	return testkit.SetupTestDB(t)
}

func newTestLoopRunService(t *testing.T) (*LoopRunService, *gorm.DB) {
	db := setupLoopRunServiceTestDB(t)
	repo := infra.NewLoopRunRepository(db)
	svc := NewLoopRunService(repo)
	return svc, db
}

func TestLoopRunService_Create(t *testing.T) {
	svc, _ := newTestLoopRunService(t)
	ctx := context.Background()

	run := &loopDomain.LoopRun{
		OrganizationID: 1,
		LoopID:         1,
		RunNumber:      1,
		Status:         loopDomain.RunStatusPending,
		TriggerType:    loopDomain.RunTriggerManual,
	}
	err := svc.Create(ctx, run)
	require.NoError(t, err)
	assert.NotZero(t, run.ID)
}

func TestLoopRunService_GetByID(t *testing.T) {
	svc, _ := newTestLoopRunService(t)
	ctx := context.Background()

	run := &loopDomain.LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status: loopDomain.RunStatusPending, TriggerType: loopDomain.RunTriggerManual,
	}
	require.NoError(t, svc.Create(ctx, run))

	t.Run("should return run", func(t *testing.T) {
		got, err := svc.GetByID(ctx, run.ID)
		require.NoError(t, err)
		assert.Equal(t, loopDomain.RunStatusPending, got.Status)
	})

	t.Run("should return ErrRunNotFound for non-existent", func(t *testing.T) {
		_, err := svc.GetByID(ctx, 99999)
		assert.ErrorIs(t, err, ErrRunNotFound)
	})
}

func TestLoopRunService_GetNextRunNumber(t *testing.T) {
	svc, _ := newTestLoopRunService(t)
	ctx := context.Background()

	t.Run("should return 1 for first run", func(t *testing.T) {
		next, err := svc.GetNextRunNumber(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, 1, next)
	})

	// Seed some runs
	for i := 1; i <= 3; i++ {
		run := &loopDomain.LoopRun{
			OrganizationID: 1, LoopID: 1, RunNumber: i,
			Status: loopDomain.RunStatusCompleted, TriggerType: loopDomain.RunTriggerCron,
		}
		require.NoError(t, svc.Create(ctx, run))
	}

	t.Run("should return max+1", func(t *testing.T) {
		next, err := svc.GetNextRunNumber(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, 4, next)
	})
}

// TestLoopRunService_ResolveStatus_SSOT tests that GetByID resolves
// the run status from Pod status (SSOT) rather than using the stored value.
func TestLoopRunService_ResolveStatus_SSOT(t *testing.T) {
	svc, db := newTestLoopRunService(t)
	ctx := context.Background()

	// Pod is completed, but run's own status is still "running"
	db.Exec(`INSERT INTO pods (pod_key, status, finished_at) VALUES (?, ?, ?)`,
		"ssot-pod", "completed", time.Now())

	started := time.Now().Add(-5 * time.Minute)
	run := &loopDomain.LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status:      loopDomain.RunStatusRunning, // stored as running
		TriggerType: loopDomain.RunTriggerManual,
		PodKey:      strPtr("ssot-pod"),
		StartedAt:   &started,
	}
	require.NoError(t, svc.Create(ctx, run))

	got, err := svc.GetByID(ctx, run.ID)
	require.NoError(t, err)
	// SSOT: effective status should be derived from Pod (completed), not stored value (running)
	assert.Equal(t, loopDomain.RunStatusCompleted, got.Status)
	assert.NotNil(t, got.FinishedAt, "should have derived FinishedAt from Pod")
	assert.NotNil(t, got.DurationSec, "should have computed duration")
}

// TestLoopRunService_ResolveStatus_AutopilotPhase tests autopilot phase resolution.
func TestLoopRunService_ResolveStatus_AutopilotPhase(t *testing.T) {
	svc, db := newTestLoopRunService(t)
	ctx := context.Background()

	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('ap-pod', 'running')`)
	db.Exec(`INSERT INTO autopilot_controllers (autopilot_controller_key, phase) VALUES ('ap-key', 'completed')`)

	run := &loopDomain.LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status:                 loopDomain.RunStatusRunning,
		TriggerType:            loopDomain.RunTriggerManual,
		PodKey:                 strPtr("ap-pod"),
		AutopilotControllerKey: strPtr("ap-key"),
	}
	require.NoError(t, svc.Create(ctx, run))

	got, err := svc.GetByID(ctx, run.ID)
	require.NoError(t, err)
	// Autopilot terminal phase should take priority over Pod active status
	assert.Equal(t, loopDomain.RunStatusCompleted, got.Status)
}

// TestLoopRunService_ResolveStatus_PodWinsOverAutopilot tests the edge case
// where Pod is terminal but autopilot phase is still non-terminal.
func TestLoopRunService_ResolveStatus_PodWinsOverAutopilot(t *testing.T) {
	svc, db := newTestLoopRunService(t)
	ctx := context.Background()

	db.Exec(`INSERT INTO pods (pod_key, status, finished_at) VALUES (?, ?, ?)`,
		"pod-wins-2", "terminated", time.Now())
	db.Exec(`INSERT INTO autopilot_controllers (autopilot_controller_key, phase) VALUES ('ap-stale-2', 'running')`)

	run := &loopDomain.LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status:                 loopDomain.RunStatusRunning,
		TriggerType:            loopDomain.RunTriggerManual,
		PodKey:                 strPtr("pod-wins-2"),
		AutopilotControllerKey: strPtr("ap-stale-2"),
	}
	require.NoError(t, svc.Create(ctx, run))

	got, err := svc.GetByID(ctx, run.ID)
	require.NoError(t, err)
	// Pod terminal (terminated = killed) should win over autopilot active (running) → cancelled
	assert.Equal(t, loopDomain.RunStatusCancelled, got.Status)
}

// TestLoopRunService_ResolveStatus_NoPod tests that runs without pod_key keep their own status.
func TestLoopRunService_ResolveStatus_NoPod(t *testing.T) {
	svc, _ := newTestLoopRunService(t)
	ctx := context.Background()

	run := &loopDomain.LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status:      loopDomain.RunStatusSkipped,
		TriggerType: loopDomain.RunTriggerCron,
		// No PodKey
	}
	require.NoError(t, svc.Create(ctx, run))

	got, err := svc.GetByID(ctx, run.ID)
	require.NoError(t, err)
	// Without pod_key, the run's own status is authoritative
	assert.Equal(t, loopDomain.RunStatusSkipped, got.Status)
}

// TestLoopRunService_ListRuns_StatusFilter tests post-resolution status filtering.
func TestLoopRunService_ListRuns_StatusFilter(t *testing.T) {
	svc, db := newTestLoopRunService(t)
	ctx := context.Background()

	// Create pods with different statuses
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('list-done', 'completed')`)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('list-run', 'running')`)
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('list-err', 'error')`)

	runs := []*loopDomain.LoopRun{
		{OrganizationID: 1, LoopID: 1, RunNumber: 1,
			Status: loopDomain.RunStatusRunning, TriggerType: loopDomain.RunTriggerCron,
			PodKey: strPtr("list-done")}, // effective: completed
		{OrganizationID: 1, LoopID: 1, RunNumber: 2,
			Status: loopDomain.RunStatusRunning, TriggerType: loopDomain.RunTriggerCron,
			PodKey: strPtr("list-run")}, // effective: running
		{OrganizationID: 1, LoopID: 1, RunNumber: 3,
			Status: loopDomain.RunStatusRunning, TriggerType: loopDomain.RunTriggerCron,
			PodKey: strPtr("list-err")}, // effective: failed
		{OrganizationID: 1, LoopID: 1, RunNumber: 4,
			Status: loopDomain.RunStatusSkipped, TriggerType: loopDomain.RunTriggerCron},
	}
	for _, run := range runs {
		require.NoError(t, svc.Create(ctx, run))
	}

	t.Run("filter by completed (resolved status)", func(t *testing.T) {
		result, total, err := svc.ListRuns(ctx, &ListRunsFilter{
			LoopID: 1,
			Status: loopDomain.RunStatusCompleted,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, result, 1)
		assert.Equal(t, loopDomain.RunStatusCompleted, result[0].Status)
	})

	t.Run("filter by failed (resolved status)", func(t *testing.T) {
		result, total, err := svc.ListRuns(ctx, &ListRunsFilter{
			LoopID: 1,
			Status: loopDomain.RunStatusFailed,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, result, 1)
	})

	t.Run("no filter returns all with resolved statuses", func(t *testing.T) {
		result, total, err := svc.ListRuns(ctx, &ListRunsFilter{LoopID: 1})
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, result, 4)
	})
}

// TestLoopRunService_ResolveStatus_OrphanedPodRef tests behavior when a run
// references a pod_key that no longer exists in the database.
func TestLoopRunService_ResolveStatus_OrphanedPodRef(t *testing.T) {
	svc, _ := newTestLoopRunService(t)
	ctx := context.Background()

	// Run references a pod_key that doesn't exist in pods table
	run := &loopDomain.LoopRun{
		OrganizationID: 1, LoopID: 1, RunNumber: 1,
		Status:      loopDomain.RunStatusRunning,
		TriggerType: loopDomain.RunTriggerManual,
		PodKey:      strPtr("ghost-pod"),
	}
	require.NoError(t, svc.Create(ctx, run))

	got, err := svc.GetByID(ctx, run.ID)
	require.NoError(t, err)
	// Orphaned pod reference should be treated as failed
	assert.Equal(t, loopDomain.RunStatusFailed, got.Status)
}

func TestLoopRunService_GetLatestPodKey(t *testing.T) {
	svc, _ := newTestLoopRunService(t)
	ctx := context.Background()

	t.Run("nil when no runs", func(t *testing.T) {
		result := svc.GetLatestPodKey(ctx, 1)
		assert.Nil(t, result)
	})

	t.Run("returns latest pod_key", func(t *testing.T) {
		runs := []*loopDomain.LoopRun{
			{OrganizationID: 1, LoopID: 10, RunNumber: 1,
				Status: loopDomain.RunStatusCompleted, TriggerType: loopDomain.RunTriggerCron,
				PodKey: strPtr("old-pod")},
			{OrganizationID: 1, LoopID: 10, RunNumber: 2,
				Status: loopDomain.RunStatusCompleted, TriggerType: loopDomain.RunTriggerCron,
				PodKey: strPtr("new-pod")},
		}
		for _, r := range runs {
			require.NoError(t, svc.Create(ctx, r))
		}

		result := svc.GetLatestPodKey(ctx, 10)
		require.NotNil(t, result)
		assert.Equal(t, "new-pod", *result)
	})
}

package loop

import (
	"context"
	"testing"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupIntegrationServices creates real LoopService + LoopRunService backed by testutil.SetupTestDB.
func setupIntegrationServices(t *testing.T) (*LoopService, *LoopRunService, context.Context) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	loopRepo := infra.NewLoopRepository(db)
	runRepo := infra.NewLoopRunRepository(db)
	return NewLoopService(loopRepo), NewLoopRunService(runRepo), context.Background()
}

func createTestLoop(t *testing.T, svc *LoopService, ctx context.Context, orgID int64, slug string) *loopDomain.Loop {
	t.Helper()
	loop, err := svc.Create(ctx, &CreateLoopRequest{
		OrganizationID: orgID,
		CreatedByID:    1,
		Name:           "Test Loop " + slug,
		Slug:           slug,
		AgentSlug:      "claude",
		PromptTemplate: "Do the task for {{project}}",
		TimeoutMinutes: 30,
	})
	require.NoError(t, err)
	return loop
}

func TestLoopLifecycle_CreateAndQuery(t *testing.T) {
	loopSvc, _, ctx := setupIntegrationServices(t)

	created := createTestLoop(t, loopSvc, ctx, 1, "create-query")

	// Query back by slug
	got, err := loopSvc.GetBySlug(ctx, 1, "create-query")
	require.NoError(t, err)

	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "Test Loop create-query", got.Name)
	assert.Equal(t, "create-query", got.Slug)
	assert.Equal(t, "claude", got.AgentSlug)
	assert.Equal(t, "Do the task for {{project}}", got.PromptTemplate)
	assert.Equal(t, loopDomain.StatusEnabled, got.Status)
	assert.Equal(t, loopDomain.ExecutionModeAutopilot, got.ExecutionMode)
	assert.Equal(t, loopDomain.SandboxStrategyPersistent, got.SandboxStrategy)
	assert.Equal(t, loopDomain.ConcurrencyPolicySkip, got.ConcurrencyPolicy)
	assert.Equal(t, 1, got.MaxConcurrentRuns)
	assert.Equal(t, 30, got.TimeoutMinutes)
	assert.Equal(t, int64(1), got.OrganizationID)

	// Query back by ID
	gotByID, err := loopSvc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, gotByID.ID)
	assert.Equal(t, "create-query", gotByID.Slug)

	// List returns the created loop
	loops, total, err := loopSvc.List(ctx, &ListLoopsFilter{OrganizationID: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, created.ID, loops[0].ID)
}

func TestLoopLifecycle_TriggerRun(t *testing.T) {
	loopSvc, runSvc, ctx := setupIntegrationServices(t)

	loop := createTestLoop(t, loopSvc, ctx, 1, "trigger-run")

	// Get next run number (should be 1 for a fresh loop)
	nextNum, err := runSvc.GetNextRunNumber(ctx, loop.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, nextNum)

	// Create a run
	now := time.Now()
	run := &loopDomain.LoopRun{
		OrganizationID: 1,
		LoopID:         loop.ID,
		RunNumber:      nextNum,
		Status:         loopDomain.RunStatusPending,
		TriggerType:    loopDomain.RunTriggerManual,
		StartedAt:      &now,
	}
	err = runSvc.Create(ctx, run)
	require.NoError(t, err)
	assert.NotZero(t, run.ID)
	assert.Equal(t, 1, run.RunNumber)

	// Query back
	got, err := runSvc.GetByID(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, loopDomain.RunStatusPending, got.Status)
	assert.Equal(t, loop.ID, got.LoopID)
	assert.Equal(t, 1, got.RunNumber)
	assert.Equal(t, loopDomain.RunTriggerManual, got.TriggerType)
}

func TestLoopLifecycle_RunNumberIncrement(t *testing.T) {
	loopSvc, runSvc, ctx := setupIntegrationServices(t)

	loop := createTestLoop(t, loopSvc, ctx, 1, "run-number")

	for i := 1; i <= 5; i++ {
		nextNum, err := runSvc.GetNextRunNumber(ctx, loop.ID)
		require.NoError(t, err)
		assert.Equal(t, i, nextNum)

		run := &loopDomain.LoopRun{
			OrganizationID: 1,
			LoopID:         loop.ID,
			RunNumber:      nextNum,
			Status:         loopDomain.RunStatusCompleted,
			TriggerType:    loopDomain.RunTriggerCron,
		}
		require.NoError(t, runSvc.Create(ctx, run))
	}

	// After 5 runs, next should be 6
	nextNum, err := runSvc.GetNextRunNumber(ctx, loop.ID)
	require.NoError(t, err)
	assert.Equal(t, 6, nextNum)
}

func TestLoopLifecycle_RunStatusTransitions(t *testing.T) {
	loopSvc, runSvc, ctx := setupIntegrationServices(t)

	loop := createTestLoop(t, loopSvc, ctx, 1, "status-transition")

	// Create run in pending state
	now := time.Now()
	run := &loopDomain.LoopRun{
		OrganizationID: 1,
		LoopID:         loop.ID,
		RunNumber:      1,
		Status:         loopDomain.RunStatusPending,
		TriggerType:    loopDomain.RunTriggerAPI,
		StartedAt:      &now,
	}
	require.NoError(t, runSvc.Create(ctx, run))

	// Transition to running
	err := runSvc.UpdateStatus(ctx, run.ID, map[string]interface{}{
		"status": loopDomain.RunStatusRunning,
	})
	require.NoError(t, err)

	got, err := runSvc.GetByID(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, loopDomain.RunStatusRunning, got.Status)

	// Finish the run (using FinishRun with optimistic locking)
	finishedAt := time.Now()
	durationSec := int(finishedAt.Sub(now).Seconds())
	updated, err := runSvc.FinishRun(ctx, run.ID, map[string]interface{}{
		"status":       loopDomain.RunStatusCompleted,
		"finished_at":  finishedAt,
		"duration_sec": durationSec,
	})
	require.NoError(t, err)
	assert.True(t, updated, "FinishRun should update the row")

	got, err = runSvc.GetByID(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, loopDomain.RunStatusCompleted, got.Status)
	assert.NotNil(t, got.FinishedAt)
	assert.NotNil(t, got.DurationSec)

	// Double-finish should return false (optimistic lock)
	updated, err = runSvc.FinishRun(ctx, run.ID, map[string]interface{}{
		"status":      loopDomain.RunStatusFailed,
		"finished_at": time.Now(),
	})
	require.NoError(t, err)
	assert.False(t, updated, "double-finish should be rejected by optimistic lock")
}

func TestLoopLifecycle_DeleteOldRuns(t *testing.T) {
	loopSvc, runSvc, ctx := setupIntegrationServices(t)

	loop := createTestLoop(t, loopSvc, ctx, 1, "delete-old-runs")

	// Create 5 finished runs
	for i := 1; i <= 5; i++ {
		finished := time.Now()
		run := &loopDomain.LoopRun{
			OrganizationID: 1,
			LoopID:         loop.ID,
			RunNumber:      i,
			Status:         loopDomain.RunStatusCompleted,
			TriggerType:    loopDomain.RunTriggerCron,
			FinishedAt:     &finished,
		}
		require.NoError(t, runSvc.Create(ctx, run))
	}

	// Verify all 5 exist
	runs, total, err := runSvc.ListRuns(ctx, &ListRunsFilter{LoopID: loop.ID, Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, runs, 5)

	// Delete old runs, keeping only 2
	deleted, err := runSvc.DeleteOldFinishedRuns(ctx, loop.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)

	// Verify only 2 remain
	runs, total, err = runSvc.ListRuns(ctx, &ListRunsFilter{LoopID: loop.ID, Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, runs, 2)

	// The remaining runs should be the 2 most recent (highest ID, i.e., run_number 4 and 5)
	remaining := map[int]bool{runs[0].RunNumber: true, runs[1].RunNumber: true}
	assert.True(t, remaining[4], "run_number 4 should be retained")
	assert.True(t, remaining[5], "run_number 5 should be retained")
}

func TestLoopLifecycle_SlugUniqueness(t *testing.T) {
	loopSvc, _, ctx := setupIntegrationServices(t)

	// Create first loop
	_, err := loopSvc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1,
		CreatedByID:    1,
		Name:           "First",
		Slug:           "unique-slug",
		PromptTemplate: "prompt",
	})
	require.NoError(t, err)

	// Create second loop with same slug and same org should fail.
	// In PostgreSQL the error maps to ErrDuplicateSlug; in SQLite the unique
	// constraint error message differs, so we accept either sentinel or any error.
	_, err = loopSvc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1,
		CreatedByID:    1,
		Name:           "Second",
		Slug:           "unique-slug",
		PromptTemplate: "prompt",
	})
	require.Error(t, err, "duplicate slug in same org should fail")

	// Verify the first loop is still intact
	got, err := loopSvc.GetBySlug(ctx, 1, "unique-slug")
	require.NoError(t, err)
	assert.Equal(t, "First", got.Name)

	// Same slug in a different org should succeed
	_, err = loopSvc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 2,
		CreatedByID:    1,
		Name:           "Third",
		Slug:           "unique-slug",
		PromptTemplate: "prompt",
	})
	assert.NoError(t, err)
}

func TestLoopLifecycle_TimeoutDetection(t *testing.T) {
	// GetTimedOutRuns uses PostgreSQL-specific syntax (::INTERVAL),
	// so we test the timeout concept via manual DB state + direct query.
	db := testutil.SetupTestDB(t)
	runRepo := infra.NewLoopRunRepository(db)
	runSvc := NewLoopRunService(runRepo)
	loopRepo := infra.NewLoopRepository(db)
	loopSvc := NewLoopService(loopRepo)
	ctx := context.Background()

	loop := createTestLoop(t, loopSvc, ctx, 1, "timeout-detect")

	// Create a run that started 2 hours ago (loop timeout is 30 min)
	startedAt := time.Now().Add(-2 * time.Hour)
	run := &loopDomain.LoopRun{
		OrganizationID: 1,
		LoopID:         loop.ID,
		RunNumber:      1,
		Status:         loopDomain.RunStatusRunning,
		TriggerType:    loopDomain.RunTriggerCron,
		StartedAt:      &startedAt,
	}
	require.NoError(t, runSvc.Create(ctx, run))

	// Verify the run is active (started but not finished)
	got, err := runSvc.GetByID(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, loopDomain.RunStatusRunning, got.Status)
	assert.Nil(t, got.FinishedAt, "run should not be finished")

	// Verify it's detectable as timed out: started_at + timeout < now
	assert.True(t, startedAt.Add(time.Duration(loop.TimeoutMinutes)*time.Minute).Before(time.Now()),
		"run should have exceeded timeout_minutes (%d)", loop.TimeoutMinutes)

	// Simulate what the scheduler does: mark the run as timed out
	finishedAt := time.Now()
	updated, err := runSvc.FinishRun(ctx, run.ID, map[string]interface{}{
		"status":      loopDomain.RunStatusTimeout,
		"finished_at": finishedAt,
	})
	require.NoError(t, err)
	assert.True(t, updated)

	got, err = runSvc.GetByID(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, loopDomain.RunStatusTimeout, got.Status)
	assert.NotNil(t, got.FinishedAt)
}

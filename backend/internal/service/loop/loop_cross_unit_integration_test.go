package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	ticketSvc "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// crossUnitEnv holds objects for cross-unit integration tests that exercise
// the full EventBus -> LoopOrchestrator event-driven chain.
type crossUnitEnv struct {
	orchestrator *LoopOrchestrator
	loopSvc      *LoopService
	runSvc       *LoopRunService
	eventBus     *eventbus.EventBus
	loop         *loopDomain.Loop
	ctx          context.Context
}

// setupCrossUnit creates a real EventBus wired to a real DB-backed LoopOrchestrator,
// replicating the production subscription pattern from setupLoopEventSubscriptions.
func setupCrossUnit(t *testing.T, opts ...func(*loopDomain.Loop)) crossUnitEnv {
	t.Helper()
	db := testutil.SetupTestDB(t)

	// SQLite `:memory:` creates a separate DB per connection. Force single connection
	// so async EventBus goroutines share the same in-memory DB.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	loopRepo := infra.NewLoopRepository(db)
	runRepo := infra.NewLoopRunRepository(db)
	loopSvc := NewLoopService(loopRepo)
	runSvc := NewLoopRunService(runRepo)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, slog.Default())
	t.Cleanup(func() { eb.Close() })

	orchestrator := NewLoopOrchestrator(loopSvc, runSvc, eb, slog.Default())
	orchestrator.SetPodDependencies(nil, nil, &mockPodTerminatorForLoop{}, nil, nil)

	// Wire subscriptions exactly like production (cmd/server/eventbus_loop.go)
	wireLoopSubscriptions(eb, orchestrator)

	slug := fmt.Sprintf("cu-test-%d", time.Now().UnixNano()%100000)
	loop, err := loopSvc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1,
		CreatedByID:    1,
		Name:           "Cross Unit Test",
		Slug:           slug,
		AgentSlug:      "claude",
		PromptTemplate: "do the task",
		ExecutionMode:  loopDomain.ExecutionModeDirect,
		TimeoutMinutes: 30,
	})
	require.NoError(t, err)

	for _, opt := range opts {
		opt(loop)
	}
	if len(opts) > 0 {
		updates := map[string]interface{}{
			"sandbox_strategy":    loop.SandboxStrategy,
			"session_persistence": loop.SessionPersistence,
			"execution_mode":      loop.ExecutionMode,
			"timeout_minutes":     loop.TimeoutMinutes,
			"ticket_id":           loop.TicketID,
		}
		require.NoError(t, loopRepo.Update(ctx, loop.ID, updates))
		loop, err = loopSvc.GetByID(ctx, loop.ID)
		require.NoError(t, err)
	}

	return crossUnitEnv{
		orchestrator: orchestrator,
		loopSvc:      loopSvc,
		runSvc:       runSvc,
		eventBus:     eb,
		loop:         loop,
		ctx:          ctx,
	}
}

// wireLoopSubscriptions replicates production subscription wiring.
func wireLoopSubscriptions(eb *eventbus.EventBus, orchestrator *LoopOrchestrator) {
	eb.Subscribe(eventbus.EventPodTerminated, func(event *eventbus.Event) {
		var data eventbus.PodStatusChangedData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return
		}
		now := time.Now()
		orchestrator.HandlePodTerminated(context.Background(), data.PodKey, data.Status, &now)
	})

	eb.Subscribe(eventbus.EventPodStatusChanged, func(event *eventbus.Event) {
		var data eventbus.PodStatusChangedData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return
		}
		switch data.Status {
		case podDomain.StatusCompleted, podDomain.StatusError:
			now := time.Now()
			orchestrator.HandlePodTerminated(context.Background(), data.PodKey, data.Status, &now)
		}
	})

	eb.Subscribe(eventbus.EventAutopilotStatusChanged, func(event *eventbus.Event) {
		var data eventbus.AutopilotStatusChangedData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return
		}
		switch data.Phase {
		case podDomain.AutopilotPhaseCompleted, podDomain.AutopilotPhaseFailed, podDomain.AutopilotPhaseStopped:
			orchestrator.HandleAutopilotTerminated(context.Background(), data.AutopilotControllerKey, data.Phase)
		}
	})
}

// createRunWithPodKey triggers a run and associates a pod key.
func createRunWithPodKey(t *testing.T, env crossUnitEnv, podKey, autopilotKey string) *loopDomain.LoopRun {
	t.Helper()
	result, err := env.orchestrator.TriggerRun(env.ctx, &TriggerRunRequest{
		LoopID:      env.loop.ID,
		TriggerType: loopDomain.RunTriggerManual,
	})
	require.NoError(t, err)
	require.False(t, result.Skipped)
	require.NoError(t, env.orchestrator.SetRunPodKey(env.ctx, result.Run.ID, podKey, autopilotKey))
	return result.Run
}

// publishPodTerminated publishes an EventPodTerminated event.
func publishPodTerminated(t *testing.T, eb *eventbus.EventBus, podKey, status string, orgID int64) {
	t.Helper()
	event, err := eventbus.NewEntityEvent(eventbus.EventPodTerminated, orgID, "pod", podKey,
		eventbus.PodStatusChangedData{PodKey: podKey, Status: status})
	require.NoError(t, err)
	require.NoError(t, eb.Publish(context.Background(), event))
}

// publishAutopilotStatusChanged publishes an EventAutopilotStatusChanged event.
func publishAutopilotStatusChanged(t *testing.T, eb *eventbus.EventBus, autopilotKey, phase, podKey string, orgID int64) {
	t.Helper()
	event, err := eventbus.NewEntityEvent(eventbus.EventAutopilotStatusChanged, orgID, "autopilot", autopilotKey,
		eventbus.AutopilotStatusChangedData{AutopilotControllerKey: autopilotKey, Phase: phase, PodKey: podKey})
	require.NoError(t, err)
	require.NoError(t, eb.Publish(context.Background(), event))
}

// waitForRunStatus polls the DB until the run reaches the expected status with FinishedAt set.
// We check FinishedAt because the SSOT resolver can derive a status before FinishRun persists it.
func waitForRunStatus(t *testing.T, runSvc *LoopRunService, runID int64, expected string) *loopDomain.LoopRun {
	t.Helper()
	var run *loopDomain.LoopRun
	require.Eventually(t, func() bool {
		r, err := runSvc.GetByID(context.Background(), runID)
		if err != nil {
			return false
		}
		run = r
		return r.Status == expected && r.FinishedAt != nil
	}, 3*time.Second, 50*time.Millisecond, "run %d did not reach status %q with FinishedAt", runID, expected)
	return run
}

// --- Test 1: EventBus PodTerminated -> Loop Completion ---

func TestCrossUnit_EventBusPodTerminated_TriggersLoopCompletion(t *testing.T) {
	env := setupCrossUnit(t)
	podKey := fmt.Sprintf("cu-pod-%d", time.Now().UnixNano())

	run := createRunWithPodKey(t, env, podKey, "")

	// Publish EventPodTerminated with status=completed (like production)
	publishPodTerminated(t, env.eventBus, podKey, podDomain.StatusCompleted, 1)

	// Wait for async handler to process and update DB
	completed := waitForRunStatus(t, env.runSvc, run.ID, loopDomain.RunStatusCompleted)
	assert.NotNil(t, completed.FinishedAt)

	// Verify loop stats: total_runs and successful_runs incremented
	loop, err := env.loopSvc.GetByID(env.ctx, env.loop.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, loop.TotalRuns)
	assert.Equal(t, 1, loop.SuccessfulRuns)
	assert.Equal(t, 0, loop.FailedRuns)
}

// --- Test 2: EventBus PodError -> Loop Failure ---

func TestCrossUnit_EventBusPodError_TriggersLoopFailure(t *testing.T) {
	env := setupCrossUnit(t)
	podKey := fmt.Sprintf("cu-pod-err-%d", time.Now().UnixNano())

	run := createRunWithPodKey(t, env, podKey, "")

	// Publish EventPodTerminated with status=error
	publishPodTerminated(t, env.eventBus, podKey, podDomain.StatusError, 1)

	// Wait for async handler
	failed := waitForRunStatus(t, env.runSvc, run.ID, loopDomain.RunStatusFailed)
	assert.NotNil(t, failed.FinishedAt)

	// Verify loop stats: failed_runs incremented
	loop, err := env.loopSvc.GetByID(env.ctx, env.loop.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, loop.TotalRuns)
	assert.Equal(t, 0, loop.SuccessfulRuns)
	assert.Equal(t, 1, loop.FailedRuns)
}

// --- Test 3: EventBus AutopilotStatusChanged -> Loop Completion ---

func TestCrossUnit_EventBusAutopilotTerminated_TriggersLoopCompletion(t *testing.T) {
	env := setupCrossUnit(t, func(l *loopDomain.Loop) {
		l.ExecutionMode = loopDomain.ExecutionModeAutopilot
	})

	podKey := fmt.Sprintf("cu-pod-ap-%d", time.Now().UnixNano())
	autopilotKey := fmt.Sprintf("cu-ap-%d", time.Now().UnixNano())

	run := createRunWithPodKey(t, env, podKey, autopilotKey)

	// Publish EventAutopilotStatusChanged with phase=completed
	publishAutopilotStatusChanged(t, env.eventBus, autopilotKey, podDomain.AutopilotPhaseCompleted, podKey, 1)

	// Wait for async handler
	completed := waitForRunStatus(t, env.runSvc, run.ID, loopDomain.RunStatusCompleted)
	assert.NotNil(t, completed.FinishedAt)

	// Verify loop stats
	loop, err := env.loopSvc.GetByID(env.ctx, env.loop.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, loop.TotalRuns)
	assert.Equal(t, 1, loop.SuccessfulRuns)
}

// --- Test 4: Loop Completion -> Posts Ticket Comment ---

func TestCrossUnit_LoopCompletion_PostsTicketComment(t *testing.T) {
	mockRepo := &mockTicketRepoForCrossUnit{}
	tktSvc := ticketSvc.NewService(mockRepo)

	ticketID := int64(42)
	env := setupCrossUnit(t, func(l *loopDomain.Loop) {
		l.TicketID = &ticketID
	})
	env.orchestrator.SetPodDependencies(nil, nil, &mockPodTerminatorForLoop{}, tktSvc, nil)

	podKey := fmt.Sprintf("cu-pod-tkt-%d", time.Now().UnixNano())
	run := createRunWithPodKey(t, env, podKey, "")

	// Publish EventPodTerminated with status=completed
	publishPodTerminated(t, env.eventBus, podKey, podDomain.StatusCompleted, 1)

	// Wait for run completion
	waitForRunStatus(t, env.runSvc, run.ID, loopDomain.RunStatusCompleted)

	// postTicketComment is called via goroutine, wait for it
	require.Eventually(t, func() bool {
		return len(mockRepo.getComments()) > 0
	}, 3*time.Second, 50*time.Millisecond, "ticket comment was not posted")

	comments := mockRepo.getComments()
	require.Len(t, comments, 1)
	assert.Equal(t, ticketID, comments[0].TicketID)
	assert.Equal(t, int64(1), comments[0].UserID) // Loop.CreatedByID
	assert.Contains(t, comments[0].Content, "Loop Run #1")
	assert.Contains(t, comments[0].Content, "completed")
	// Success emoji
	assert.Contains(t, comments[0].Content, "\u2705") // checkmark emoji
}

// mockTicketRepoForCrossUnit is in mock_ticket_repo_test.go

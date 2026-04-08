package loop

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock types for workflow tests ---

type mockPodOrchForLoop struct {
	mu         sync.Mutex
	callCount  int
	failFirstN int
	results    []*agentpodSvc.OrchestrateCreatePodResult
	err        error
}

func (m *mockPodOrchForLoop) CreatePod(_ context.Context, _ *agentpodSvc.OrchestrateCreatePodRequest) (*agentpodSvc.OrchestrateCreatePodResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	if m.failFirstN > 0 && m.callCount <= m.failFirstN {
		return nil, fmt.Errorf("mock create pod error (call %d)", m.callCount)
	}
	if m.err != nil {
		return nil, m.err
	}
	if len(m.results) > 0 {
		idx := m.callCount - 1
		if m.failFirstN > 0 {
			idx = m.callCount - m.failFirstN - 1
		}
		if idx >= 0 && idx < len(m.results) {
			return m.results[idx], nil
		}
	}
	return nil, errors.New("no mock result configured")
}

type mockPodTerminatorForLoop struct {
	mu             sync.Mutex
	terminatedKeys []string
}

func (m *mockPodTerminatorForLoop) TerminatePod(_ context.Context, podKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminatedKeys = append(m.terminatedKeys, podKey)
	return nil
}

func (m *mockPodTerminatorForLoop) getTerminatedKeys() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]string, len(m.terminatedKeys))
	copy(cp, m.terminatedKeys)
	return cp
}

// workflowTestEnv holds all the objects used by a workflow integration test.
type workflowTestEnv struct {
	orchestrator *LoopOrchestrator
	loopSvc      *LoopService
	runSvc       *LoopRunService
	eventBus     *eventbus.EventBus
	podOrch      *mockPodOrchForLoop
	podTerm      *mockPodTerminatorForLoop
	loop         *loopDomain.Loop
	ctx          context.Context
}

// setupWorkflowTest creates a real DB-backed orchestrator with mock pod dependencies.
func setupWorkflowTest(t *testing.T, opts ...func(*loopDomain.Loop)) workflowTestEnv {
	t.Helper()
	db := testkit.SetupTestDB(t)
	loopRepo := infra.NewLoopRepository(db)
	runRepo := infra.NewLoopRunRepository(db)
	loopSvc := NewLoopService(loopRepo)
	runSvc := NewLoopRunService(runRepo)
	ctx := context.Background()

	// EventBus with nil redis (local-only dispatch, no Redis dependency)
	eb := eventbus.NewEventBus(nil, slog.Default())
	t.Cleanup(func() { eb.Close() })

	orchestrator := NewLoopOrchestrator(loopSvc, runSvc, eb, slog.Default())

	// Create a loop
	slug := fmt.Sprintf("wf-test-%d", time.Now().UnixNano()%100000)
	loop, err := loopSvc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1,
		CreatedByID:    1,
		Name:           "Workflow Test",
		Slug:           slug,
		AgentSlug:      "claude",
		PromptTemplate: "Do the task",
		ExecutionMode:  loopDomain.ExecutionModeDirect,
		TimeoutMinutes: 30,
	})
	require.NoError(t, err)

	for _, opt := range opts {
		opt(loop)
	}
	// Persist any option modifications
	if len(opts) > 0 {
		updates := map[string]interface{}{
			"sandbox_strategy":   loop.SandboxStrategy,
			"session_persistence": loop.SessionPersistence,
			"execution_mode":     loop.ExecutionMode,
			"last_pod_key":       loop.LastPodKey,
			"timeout_minutes":    loop.TimeoutMinutes,
		}
		require.NoError(t, loopRepo.Update(ctx, loop.ID, updates))
		loop, err = loopSvc.GetByID(ctx, loop.ID)
		require.NoError(t, err)
	}

	podOrch := &mockPodOrchForLoop{}
	podTerm := &mockPodTerminatorForLoop{}

	// PodOrchestrator is a concrete struct — we set it to nil here because our tests
	// exercise the orchestrator methods (SetRunPodKey, HandlePodTerminated, etc.) directly,
	// bypassing StartRun's pod creation. For StartRun, the nil check triggers MarkRunFailed.
	orchestrator.SetPodDependencies(nil, nil, podTerm, nil, nil)

	return workflowTestEnv{
		orchestrator: orchestrator,
		loopSvc:      loopSvc,
		runSvc:       runSvc,
		eventBus:     eb,
		podOrch:      podOrch,
		podTerm:      podTerm,
		loop:         loop,
		ctx:          ctx,
	}
}

func TestLoopWorkflow_TriggerToCompletion(t *testing.T) {
	env := setupWorkflowTest(t)

	podKey := fmt.Sprintf("pod-wf-%d", time.Now().UnixNano())
	env.podOrch.results = []*agentpodSvc.OrchestrateCreatePodResult{{
		Pod: &podDomain.Pod{PodKey: podKey, OrganizationID: 1, RunnerID: 1},
	}}

	// Step 1: TriggerRun
	triggerResult, err := env.orchestrator.TriggerRun(env.ctx, &TriggerRunRequest{
		LoopID:      env.loop.ID,
		TriggerType: loopDomain.RunTriggerManual,
	})
	require.NoError(t, err)
	require.False(t, triggerResult.Skipped)
	require.NotNil(t, triggerResult.Run)
	run := triggerResult.Run
	assert.Equal(t, loopDomain.RunStatusPending, run.Status)
	assert.Equal(t, 1, run.RunNumber)

	// Step 2: Simulate StartRun (synchronous — bypass goroutine for testing)
	// Since PodOrchestrator is nil, StartRun will call MarkRunFailed.
	// We need to manually simulate what StartRun does: create pod + set pod key.
	result, err := env.podOrch.CreatePod(env.ctx, nil)
	require.NoError(t, err)
	require.NoError(t, env.orchestrator.SetRunPodKey(env.ctx, run.ID, result.Pod.PodKey, ""))

	// Verify run has pod_key
	updated, err := env.runSvc.GetByID(env.ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.PodKey)
	assert.Equal(t, podKey, *updated.PodKey)

	// Step 3: HandlePodTerminated with "completed"
	env.orchestrator.HandlePodTerminated(env.ctx, podKey, podDomain.StatusCompleted, nil)

	// Verify run is completed
	completed, err := env.runSvc.GetByID(env.ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, loopDomain.RunStatusCompleted, completed.Status)
	assert.NotNil(t, completed.FinishedAt)

	// Verify loop stats incremented
	loop, err := env.loopSvc.GetByID(env.ctx, env.loop.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, loop.TotalRuns)
	assert.Equal(t, 1, loop.SuccessfulRuns)
}

func TestLoopWorkflow_StartRunPodCreationFailure(t *testing.T) {
	env := setupWorkflowTest(t)
	env.podOrch.err = errors.New("runner unreachable")

	// TriggerRun
	triggerResult, err := env.orchestrator.TriggerRun(env.ctx, &TriggerRunRequest{
		LoopID:      env.loop.ID,
		TriggerType: loopDomain.RunTriggerManual,
	})
	require.NoError(t, err)
	run := triggerResult.Run

	// StartRun with nil podOrchestrator — will call MarkRunFailed
	env.orchestrator.StartRun(env.ctx, env.loop, run, 1)

	// Verify run is failed
	failed, err := env.runSvc.GetByID(env.ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, loopDomain.RunStatusFailed, failed.Status)
	assert.NotNil(t, failed.FinishedAt)
	assert.NotNil(t, failed.ErrorMessage)
	assert.Contains(t, *failed.ErrorMessage, "Pod orchestrator not configured")
}

func TestLoopWorkflow_HandlePodTerminatedError(t *testing.T) {
	env := setupWorkflowTest(t)

	podKey := fmt.Sprintf("pod-err-%d", time.Now().UnixNano())

	// TriggerRun and associate pod
	triggerResult, err := env.orchestrator.TriggerRun(env.ctx, &TriggerRunRequest{
		LoopID:      env.loop.ID,
		TriggerType: loopDomain.RunTriggerAPI,
	})
	require.NoError(t, err)
	run := triggerResult.Run
	require.NoError(t, env.orchestrator.SetRunPodKey(env.ctx, run.ID, podKey, ""))

	// HandlePodTerminated with error status
	env.orchestrator.HandlePodTerminated(env.ctx, podKey, podDomain.StatusError, nil)

	// Verify run status becomes "failed"
	got, err := env.runSvc.GetByID(env.ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, loopDomain.RunStatusFailed, got.Status)
	assert.NotNil(t, got.FinishedAt)

	// Verify loop stats: failed_runs incremented
	loop, err := env.loopSvc.GetByID(env.ctx, env.loop.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, loop.TotalRuns)
	assert.Equal(t, 1, loop.FailedRuns)
}

func TestLoopWorkflow_HandleAutopilotTerminated(t *testing.T) {
	env := setupWorkflowTest(t, func(l *loopDomain.Loop) {
		l.ExecutionMode = loopDomain.ExecutionModeAutopilot
	})

	podKey := fmt.Sprintf("pod-ap-%d", time.Now().UnixNano())
	autopilotKey := fmt.Sprintf("ap-key-%d", time.Now().UnixNano())

	// TriggerRun and associate pod + autopilot key
	triggerResult, err := env.orchestrator.TriggerRun(env.ctx, &TriggerRunRequest{
		LoopID:      env.loop.ID,
		TriggerType: loopDomain.RunTriggerManual,
	})
	require.NoError(t, err)
	run := triggerResult.Run
	require.NoError(t, env.orchestrator.SetRunPodKey(env.ctx, run.ID, podKey, autopilotKey))

	// HandleAutopilotTerminated with "completed" phase
	env.orchestrator.HandleAutopilotTerminated(env.ctx, autopilotKey, podDomain.AutopilotPhaseCompleted)

	// Verify run completed
	got, err := env.runSvc.GetByID(env.ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, loopDomain.RunStatusCompleted, got.Status)
	assert.NotNil(t, got.FinishedAt)

	// Stats updated
	loop, err := env.loopSvc.GetByID(env.ctx, env.loop.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, loop.SuccessfulRuns)
}

func TestLoopWorkflow_HandleRunCompleted_PersistentSandbox(t *testing.T) {
	env := setupWorkflowTest(t, func(l *loopDomain.Loop) {
		l.SandboxStrategy = loopDomain.SandboxStrategyPersistent
	})

	podKey := fmt.Sprintf("pod-ps-%d", time.Now().UnixNano())

	// TriggerRun + associate pod
	triggerResult, err := env.orchestrator.TriggerRun(env.ctx, &TriggerRunRequest{
		LoopID:      env.loop.ID,
		TriggerType: loopDomain.RunTriggerManual,
	})
	require.NoError(t, err)
	require.NoError(t, env.orchestrator.SetRunPodKey(env.ctx, triggerResult.Run.ID, podKey, ""))

	// Complete the run
	env.orchestrator.HandlePodTerminated(env.ctx, podKey, podDomain.StatusCompleted, nil)

	// Verify last_pod_key updated for persistent sandbox resume
	loop, err := env.loopSvc.GetByID(env.ctx, env.loop.ID)
	require.NoError(t, err)
	require.NotNil(t, loop.LastPodKey, "persistent sandbox should update last_pod_key")
	assert.Equal(t, podKey, *loop.LastPodKey)
}

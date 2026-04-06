//go:build integration

package autopilot

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sequentialMockProcess returns different decisions per call.
type sequentialMockProcess struct {
	decisions []*ControlDecision
	errs      []error
	callCount atomic.Int32
}

func (m *sequentialMockProcess) RunControlProcess(_ context.Context, _ int) (*ControlDecision, error) {
	idx := int(m.callCount.Add(1)) - 1
	if idx < len(m.errs) && m.errs[idx] != nil {
		return nil, m.errs[idx]
	}
	if idx < len(m.decisions) {
		return m.decisions[idx], nil
	}
	// Fallback: return last decision
	return m.decisions[len(m.decisions)-1], nil
}

func (m *sequentialMockProcess) SetSessionID(_ string) {}
func (m *sequentialMockProcess) GetSessionID() string  { return "" }
func (m *sequentialMockProcess) Stop()                 {}

func newTestController(t *testing.T, maxIter int32, ctrl TargetPodController, cp ControlProcess,
) (*AutopilotController, *MockEventReporter) {
	t.Helper()
	reporter := &MockEventReporter{}
	ac := NewAutopilotController(Config{
		AutopilotKey: "ap-test",
		PodKey:       "pod-test",
		ProtoConfig: &runnerv1.AutopilotConfig{
			Prompt:           "do the task",
			MaxIterations:           maxIter,
			IterationTimeoutSeconds: 10,
		},
		PodCtrl:        ctrl,
		Reporter:       reporter,
		ControlProcess: cp,
	})
	return ac, reporter
}

// TestAutopilot_CompleteCycle_SendInput_Integration tests a full cycle:
// start -> control returns CONTINUE -> state change -> next iteration.
func TestAutopilot_CompleteCycle_SendInput_Integration(t *testing.T) {
	podCtrl := NewMockPodControllerWithStateChange()
	podCtrl.workDir = t.TempDir()
	podCtrl.podKey = "pod-test"
	podCtrl.agentStatus = "waiting"

	cp := &sequentialMockProcess{
		decisions: []*ControlDecision{
			{Type: DecisionContinue, Summary: "sent ls -la", Action: &DecisionAction{Type: "send_input", Content: "ls -la"}},
			{Type: DecisionCompleted, Summary: "All done"},
		},
	}

	ac, reporter := newTestController(t, 10, podCtrl, cp)
	defer ac.Stop()

	require.NoError(t, ac.Start())

	// Wait for first decision (CONTINUE) to be processed
	require.True(t, waitForCondition(t, 5*time.Second, func() bool {
		return ac.GetStatus().CurrentIteration >= 1 && ac.GetStatus().LastDecision == "CONTINUE"
	}), "first iteration should complete with CONTINUE")

	// Reset trigger dedup timer so next OnPodWaiting is not throttled
	resetTriggerDedup(ac)

	// Simulate pod state: executing -> waiting (triggers next iteration)
	podCtrl.agentStatus = "executing"
	podCtrl.SimulateStateChange("executing")
	time.Sleep(100 * time.Millisecond)
	podCtrl.agentStatus = "waiting"
	podCtrl.SimulateStateChange("waiting")

	// Wait for completed phase
	require.True(t, waitForPhase(ac, PhaseCompleted, 10*time.Second),
		"should reach completed phase")

	// Verify: iteration events reported, thinking events sent
	iterEvents := reporter.GetIterationEvents()
	assert.GreaterOrEqual(t, len(iterEvents), 2, "at least 2 iteration events expected")
	assert.Equal(t, int32(2), cp.callCount.Load(), "control process called twice")

	hasCompleted := false
	for _, e := range reporter.GetTerminatedEvents() {
		if e.Reason == "completed" {
			hasCompleted = true
		}
	}
	assert.True(t, hasCompleted, "terminated event with reason=completed expected")
}

// TestAutopilot_CompleteCycle_Finish_Integration tests that a finish decision
// stops the controller and reports the termination event.
func TestAutopilot_CompleteCycle_Finish_Integration(t *testing.T) {
	podCtrl := &MockPodController{
		workDir: t.TempDir(), podKey: "pod-test", agentStatus: "waiting",
	}
	cp := &MockControlProcess{
		Decision: &ControlDecision{Type: DecisionCompleted, Summary: "Task complete"},
	}

	ac, reporter := newTestController(t, 10, podCtrl, cp)
	defer ac.Stop()

	require.NoError(t, ac.Start())
	require.True(t, waitForPhase(ac, PhaseCompleted, 10*time.Second),
		"should reach completed phase")

	terminated := reporter.GetTerminatedEvents()
	require.NotEmpty(t, terminated)
	assert.Equal(t, "completed", terminated[0].Reason)

	status := ac.GetStatus()
	assert.Equal(t, PhaseCompleted, status.Phase)
	assert.Equal(t, 1, status.CurrentIteration)
}

// TestAutopilot_MaxIterations_Integration tests that the controller stops
// after reaching max iterations and reports the correct termination reason.
func TestAutopilot_MaxIterations_Integration(t *testing.T) {
	podCtrl := NewMockPodControllerWithStateChange()
	podCtrl.workDir = t.TempDir()
	podCtrl.podKey = "pod-test"
	podCtrl.agentStatus = "waiting"

	cp := &MockControlProcess{} // Returns DecisionContinue by default

	ac, reporter := newTestController(t, 3, podCtrl, cp)
	defer ac.Stop()

	require.NoError(t, ac.Start())

	// Drive iterations via state changes: after each CONTINUE, simulate executing->waiting
	for i := 0; i < 4; i++ {
		// Wait for either the next iteration or a terminal phase
		require.True(t, waitForCondition(t, 5*time.Second, func() bool {
			s := ac.GetStatus()
			return s.CurrentIteration >= i+1 || s.Phase == PhaseMaxIterations
		}), "iteration %d should complete or max reached", i+1)

		if ac.GetStatus().Phase == PhaseMaxIterations {
			break
		}

		resetTriggerDedup(ac)
		podCtrl.agentStatus = "executing"
		podCtrl.SimulateStateChange("executing")
		time.Sleep(100 * time.Millisecond)
		podCtrl.agentStatus = "waiting"
		podCtrl.SimulateStateChange("waiting")
	}

	require.True(t, waitForPhase(ac, PhaseMaxIterations, 10*time.Second),
		"should reach max_iterations phase")

	assert.LessOrEqual(t, ac.GetStatus().CurrentIteration, 3)

	hasMaxIter := false
	for _, e := range reporter.GetTerminatedEvents() {
		if e.Reason == "max_iterations" {
			hasMaxIter = true
		}
	}
	assert.True(t, hasMaxIter, "terminated event with reason=max_iterations expected")
}

// TestAutopilot_ErrorRecovery_Integration tests that the controller retries
// on error and succeeds on the subsequent attempt.
func TestAutopilot_ErrorRecovery_Integration(t *testing.T) {
	podCtrl := &MockPodController{
		workDir: t.TempDir(), podKey: "pod-test", agentStatus: "waiting",
	}

	cp := &sequentialMockProcess{
		decisions: []*ControlDecision{
			nil, // placeholder for error path
			{Type: DecisionCompleted, Summary: "Recovered"},
		},
		errs: []error{
			errors.New("transient failure"),
			nil,
		},
	}

	ac, reporter := newTestController(t, 10, podCtrl, cp)
	defer ac.Stop()

	require.NoError(t, ac.Start())

	require.True(t, waitForPhase(ac, PhaseCompleted, 15*time.Second),
		"should recover and reach completed phase")

	// Verify error was reported
	hasError := false
	for _, e := range reporter.GetIterationEvents() {
		if e.Phase == "error" {
			hasError = true
		}
	}
	assert.True(t, hasError, "error iteration event expected before recovery")

	hasCompleted := false
	for _, e := range reporter.GetTerminatedEvents() {
		if e.Reason == "completed" {
			hasCompleted = true
		}
	}
	assert.True(t, hasCompleted, "terminated event with reason=completed expected after recovery")
}

// waitForCondition polls until fn returns true or timeout expires.
func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// resetTriggerDedup resets the trigger dedup timer so the next OnPodWaiting is not throttled.
func resetTriggerDedup(ac *AutopilotController) {
	ac.iterCtrl.triggerMu.Lock()
	ac.iterCtrl.lastTriggerTime = time.Time{}
	ac.iterCtrl.triggerMu.Unlock()
}

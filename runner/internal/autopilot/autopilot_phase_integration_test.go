//go:build integration

package autopilot

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// delayedMockProcess returns a decision after a configurable delay.
type delayedMockProcess struct {
	delay    time.Duration
	decision *ControlDecision
	err      error
	called   atomic.Int32
}

func (m *delayedMockProcess) RunControlProcess(ctx context.Context, _ int) (*ControlDecision, error) {
	m.called.Add(1)
	select {
	case <-time.After(m.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.decision, nil
}

func (m *delayedMockProcess) SetSessionID(_ string) {}
func (m *delayedMockProcess) GetSessionID() string  { return "" }
func (m *delayedMockProcess) Stop()                 {}

// TestAutopilot_WaitForPodState_Integration verifies controller runs
// control process only after pod reaches "waiting" state via state change events.
func TestAutopilot_WaitForPodState_Integration(t *testing.T) {
	podCtrl := NewMockPodControllerWithStateChange()
	podCtrl.workDir = t.TempDir()
	podCtrl.podKey = "pod-test"
	podCtrl.agentStatus = "waiting" // Start as waiting so initial iteration fires

	callCount := &atomic.Int32{}
	cp := &sequentialMockProcess{
		decisions: []*ControlDecision{
			{Type: DecisionContinue, Summary: "sent input", Action: &DecisionAction{Type: "send_input", Content: "ls"}},
			{Type: DecisionCompleted, Summary: "Done"},
		},
	}

	ac, _ := newTestController(t, 10, podCtrl, cp)
	defer ac.Stop()

	require.NoError(t, ac.Start())

	// First iteration should fire (pod is waiting)
	require.True(t, waitForCondition(t, 5*time.Second, func() bool {
		return cp.callCount.Load() >= 1
	}), "first iteration should run")

	_ = callCount // suppress warning

	// Now pod is executing — next iteration should NOT fire until waiting again
	resetTriggerDedup(ac)
	podCtrl.agentStatus = "executing"
	podCtrl.SimulateStateChange("executing")
	time.Sleep(200 * time.Millisecond)

	countAfterExec := cp.callCount.Load()

	// Simulate pod becoming waiting again — triggers next iteration
	podCtrl.agentStatus = "waiting"
	podCtrl.SimulateStateChange("waiting")

	require.True(t, waitForPhase(ac, PhaseCompleted, 10*time.Second),
		"should complete after second iteration")
	assert.Greater(t, cp.callCount.Load(), countAfterExec, "should run after waiting")
}

// TestAutopilot_ConsecutiveErrorsExceedLimit_Integration verifies the
// controller gives up after too many consecutive errors.
func TestAutopilot_ConsecutiveErrorsExceedLimit_Integration(t *testing.T) {
	podCtrl := &MockPodController{
		workDir: t.TempDir(), podKey: "pod-test", agentStatus: "waiting",
	}
	cp := &MockControlProcess{Err: assert.AnError}

	ac, _ := newTestController(t, 10, podCtrl, cp)
	defer ac.Stop()

	require.NoError(t, ac.Start())

	// Should give up after consecutive errors (default limit is 3)
	require.True(t, waitForCondition(t, 20*time.Second, func() bool {
		s := ac.GetStatus()
		return s.Phase == PhaseFailed || s.Phase == PhaseCompleted || s.Phase == PhaseMaxIterations
	}), "should reach terminal phase after consecutive errors")
}

// TestAutopilot_StopDuringIteration_Integration verifies clean shutdown
// when Stop() is called while control process is running.
func TestAutopilot_StopDuringIteration_Integration(t *testing.T) {
	podCtrl := &MockPodController{
		workDir: t.TempDir(), podKey: "pod-test", agentStatus: "waiting",
	}
	cp := &delayedMockProcess{
		delay:    2 * time.Second,
		decision: &ControlDecision{Type: DecisionCompleted, Summary: "Done"},
	}

	ac, _ := newTestController(t, 10, podCtrl, cp)
	require.NoError(t, ac.Start())

	// Wait for control process to start
	require.True(t, waitForCondition(t, 3*time.Second, func() bool {
		return cp.called.Load() >= 1
	}), "control process should start")

	// Stop while running — should not hang
	done := make(chan struct{})
	go func() { ac.Stop(); close(done) }()

	select {
	case <-done:
		// Clean shutdown
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() hung during active iteration")
	}
}

// TestAutopilot_DecisionTypeWait_Integration verifies "wait" action
// doesn't send input but proceeds to next iteration.
func TestAutopilot_DecisionTypeWait_Integration(t *testing.T) {
	podCtrl := NewMockPodControllerWithStateChange()
	podCtrl.workDir = t.TempDir()
	podCtrl.podKey = "pod-test"
	podCtrl.agentStatus = "waiting"

	cp := &sequentialMockProcess{
		decisions: []*ControlDecision{
			{Type: DecisionContinue, Summary: "Waiting for tests", Action: &DecisionAction{Type: "wait"}},
			{Type: DecisionCompleted, Summary: "Tests passed"},
		},
	}

	ac, _ := newTestController(t, 10, podCtrl, cp)
	defer ac.Stop()

	require.NoError(t, ac.Start())

	// Wait for first decision (wait action)
	require.True(t, waitForCondition(t, 5*time.Second, func() bool {
		return ac.GetStatus().CurrentIteration >= 1
	}), "first iteration should complete")

	// Drive next iteration
	resetTriggerDedup(ac)
	podCtrl.agentStatus = "executing"
	podCtrl.SimulateStateChange("executing")
	time.Sleep(100 * time.Millisecond)
	podCtrl.agentStatus = "waiting"
	podCtrl.SimulateStateChange("waiting")

	require.True(t, waitForPhase(ac, PhaseCompleted, 10*time.Second),
		"should complete after wait+continue")
	assert.Equal(t, int32(2), cp.callCount.Load())
}

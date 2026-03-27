package autopilot

import (
	"os"
	"runtime"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForPhase polls until the autopilot reaches the expected phase or timeout.
// Returns true if the expected phase was reached, false on timeout.
func waitForPhase(rp *AutopilotController, expectedPhase Phase, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if rp.GetStatus().Phase == expectedPhase {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// waitForTerminalPhase polls until the autopilot reaches a terminal phase or timeout.
// Returns the phase reached, or empty string on timeout.
func waitForTerminalPhase(rp *AutopilotController, timeout time.Duration) Phase {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		phase := rp.GetStatus().Phase
		switch phase {
		case PhaseCompleted, PhaseFailed, PhaseStopped, PhaseMaxIterations, PhaseWaitingApproval:
			return phase
		}
		time.Sleep(100 * time.Millisecond)
	}
	return ""
}

func TestAutopilotController_HandleDecision_Completed(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping: shell-based test scripts use Unix echo semantics")
	}
	tmpDir, err := os.MkdirTemp("", "autopilot_test")
	require.NoError(t, err)

	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent",
		"echo TASK_COMPLETED\necho All tasks done.")

	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:    "Test",
		MaxIterations:    10,
		ControlAgentSlug: scriptPath,
	}

	workerCtrl := &MockPodController{
		workDir: tmpDir, podKey: "worker-123", agentStatus: "waiting",
	}
	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:   "autopilot-123",
		PodKey:         "worker-123",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       reporter,
		ControlProcess: &MockControlProcess{Decision: &ControlDecision{Type: DecisionCompleted, Summary: "All tasks done."}},
		MCPPort:        19000,
	})

	defer func() {
		rp.Stop()
		os.RemoveAll(tmpDir)
	}()

	err = rp.Start()
	require.NoError(t, err)

	reached := waitForPhase(rp, PhaseCompleted, 10*time.Second)
	require.True(t, reached, "Expected phase to reach 'completed' within timeout")

	hasTerminated := false
	for _, e := range reporter.GetTerminatedEvents() {
		if e.Reason == "completed" {
			hasTerminated = true
			break
		}
	}
	assert.True(t, hasTerminated)
}

func TestAutopilotController_HandleDecision_NeedHumanHelp(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping: shell-based test scripts use Unix echo semantics")
	}
	tmpDir, err := os.MkdirTemp("", "autopilot_test")
	require.NoError(t, err)

	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent",
		"echo NEED_HUMAN_HELP\necho Need credentials.")

	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:    "Test",
		MaxIterations:    10,
		ControlAgentSlug: scriptPath,
	}

	workerCtrl := &MockPodController{
		workDir: tmpDir, podKey: "worker-123", agentStatus: "waiting",
	}
	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:   "autopilot-123",
		PodKey:         "worker-123",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       reporter,
		ControlProcess: &MockControlProcess{Decision: &ControlDecision{Type: DecisionNeedHumanHelp, Summary: "Need credentials."}},
		MCPPort:        19000,
	})

	defer func() {
		rp.Stop()
		os.RemoveAll(tmpDir)
	}()

	err = rp.Start()
	require.NoError(t, err)

	reached := waitForPhase(rp, PhaseWaitingApproval, 10*time.Second)
	require.True(t, reached, "Expected phase to reach 'waiting_approval' within timeout")
}

func TestAutopilotController_HandleDecision_GiveUp(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping: shell-based test scripts use Unix echo semantics")
	}
	tmpDir, err := os.MkdirTemp("", "autopilot_test")
	require.NoError(t, err)

	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent",
		"echo GIVE_UP\necho Cannot complete.")

	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:    "Test",
		MaxIterations:    10,
		ControlAgentSlug: scriptPath,
	}

	workerCtrl := &MockPodController{
		workDir: tmpDir, podKey: "worker-123", agentStatus: "waiting",
	}
	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:   "autopilot-123",
		PodKey:         "worker-123",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       reporter,
		ControlProcess: &MockControlProcess{Decision: &ControlDecision{Type: DecisionGiveUp, Summary: "Cannot proceed."}},
		MCPPort:        19000,
	})

	defer func() {
		rp.Stop()
		os.RemoveAll(tmpDir)
	}()

	err = rp.Start()
	require.NoError(t, err)

	reached := waitForPhase(rp, PhaseFailed, 10*time.Second)
	require.True(t, reached, "Expected phase to reach 'failed' within timeout")

	hasTerminated := false
	for _, e := range reporter.GetTerminatedEvents() {
		if e.Reason == "failed" {
			hasTerminated = true
			break
		}
	}
	assert.True(t, hasTerminated)
}

func TestAutopilotController_HandleDecision_Continue(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping: shell-based test scripts use Unix echo semantics")
	}
	tmpDir, err := os.MkdirTemp("", "autopilot_test")
	require.NoError(t, err)

	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent",
		"echo CONTINUE\necho Working on it.")

	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:    "Test",
		MaxIterations:    10,
		ControlAgentSlug: scriptPath,
	}

	workerCtrl := &MockPodController{
		workDir: tmpDir, podKey: "worker-123", agentStatus: "waiting",
	}
	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:   "autopilot-123",
		PodKey:         "worker-123",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       reporter,
		ControlProcess: &MockControlProcess{},
		MCPPort:        19000,
	})

	defer func() {
		rp.Stop()
		os.RemoveAll(tmpDir)
	}()

	err = rp.Start()
	require.NoError(t, err)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if rp.GetStatus().LastDecision == "CONTINUE" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	status := rp.GetStatus()
	assert.Equal(t, PhaseRunning, status.Phase)
	assert.Equal(t, "CONTINUE", status.LastDecision)
}

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
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autopilot_test")
	require.NoError(t, err)

	// Create mock agent that returns TASK_COMPLETED
	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent",
		"echo TASK_COMPLETED\necho All tasks done.")

	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:    "Test",
		MaxIterations:    10,
		ControlAgentType: scriptPath,
	}

	workerCtrl := &MockPodController{
		workDir:     tmpDir,
		podKey:      "worker-123",
		agentStatus: "waiting",
	}

	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:  "autopilot-123",
		PodKey:        "worker-123",
		ProtoConfig:   protoConfig,
		PodCtrl:       workerCtrl,
		Reporter:      reporter,
		MCPPort:       19000,
	})

	// Stop must be called before removing tmpDir to avoid "no such file" errors
	defer func() {
		rp.Stop()
		os.RemoveAll(tmpDir)
	}()

	err = rp.Start()
	require.NoError(t, err)

	// Wait for phase to reach completed (with timeout)
	reached := waitForPhase(rp, PhaseCompleted, 10*time.Second)
	require.True(t, reached, "Expected phase to reach 'completed' within timeout")

	// Check terminated event
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
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autopilot_test")
	require.NoError(t, err)

	// Create mock agent that returns NEED_HUMAN_HELP
	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent",
		"echo NEED_HUMAN_HELP\necho Need credentials.")

	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:    "Test",
		MaxIterations:    10,
		ControlAgentType: scriptPath,
	}

	workerCtrl := &MockPodController{
		workDir:     tmpDir,
		podKey:      "worker-123",
		agentStatus: "waiting",
	}

	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:  "autopilot-123",
		PodKey:        "worker-123",
		ProtoConfig:   protoConfig,
		PodCtrl:       workerCtrl,
		Reporter:      reporter,
		MCPPort:       19000,
	})

	// Stop must be called before removing tmpDir to avoid "no such file" errors
	defer func() {
		rp.Stop()
		os.RemoveAll(tmpDir)
	}()

	err = rp.Start()
	require.NoError(t, err)

	// Wait for phase to reach waiting_approval (with timeout)
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
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autopilot_test")
	require.NoError(t, err)

	// Create mock agent that returns GIVE_UP
	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent",
		"echo GIVE_UP\necho Cannot complete.")

	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:    "Test",
		MaxIterations:    10,
		ControlAgentType: scriptPath,
	}

	workerCtrl := &MockPodController{
		workDir:     tmpDir,
		podKey:      "worker-123",
		agentStatus: "waiting",
	}

	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:  "autopilot-123",
		PodKey:        "worker-123",
		ProtoConfig:   protoConfig,
		PodCtrl:       workerCtrl,
		Reporter:      reporter,
		MCPPort:       19000,
	})

	// Stop must be called before removing tmpDir to avoid "no such file" errors
	defer func() {
		rp.Stop()
		os.RemoveAll(tmpDir)
	}()

	err = rp.Start()
	require.NoError(t, err)

	// Wait for phase to reach failed (with timeout)
	reached := waitForPhase(rp, PhaseFailed, 10*time.Second)
	require.True(t, reached, "Expected phase to reach 'failed' within timeout")

	// Check terminated event
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
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autopilot_test")
	require.NoError(t, err)

	// Create mock agent that returns CONTINUE
	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent",
		"echo CONTINUE\necho Working on it.")

	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:    "Test",
		MaxIterations:    10,
		ControlAgentType: scriptPath,
	}

	workerCtrl := &MockPodController{
		workDir:     tmpDir,
		podKey:      "worker-123",
		agentStatus: "waiting",
	}

	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:  "autopilot-123",
		PodKey:        "worker-123",
		ProtoConfig:   protoConfig,
		PodCtrl:       workerCtrl,
		Reporter:      reporter,
		MCPPort:       19000,
	})

	// Stop must be called before removing tmpDir to avoid "no such file" errors
	defer func() {
		rp.Stop()
		os.RemoveAll(tmpDir)
	}()

	err = rp.Start()
	require.NoError(t, err)

	// Wait for LastDecision to be set (polling with timeout)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if rp.GetStatus().LastDecision == "CONTINUE" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	status := rp.GetStatus()
	// Should remain running after CONTINUE
	assert.Equal(t, PhaseRunning, status.Phase)
	assert.Equal(t, "CONTINUE", status.LastDecision)
}

func TestAutopilotController_OnPodWaiting_IncrementAfterMaxReached(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test",
		MaxIterations: 1,
	}

	workDir := t.TempDir()
	workerCtrl := &MockPodController{
		workDir:     workDir,
		podKey:      "worker-123",
		agentStatus: "executing",
	}

	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:  "autopilot-123",
		PodKey: "worker-123",
		ProtoConfig:  protoConfig,
		PodCtrl:   workerCtrl,
		Reporter:     reporter,
	})
	_ = rp.Start()
	defer rp.Stop()

	// First call - should increment to 1
	rp.OnPodWaiting()
	assert.Equal(t, 1, rp.GetStatus().CurrentIteration)

	// Wait for trigger dedup
	time.Sleep(6 * time.Second)

	// Second call - should hit max iterations
	rp.OnPodWaiting()

	status := rp.GetStatus()
	assert.Equal(t, PhaseMaxIterations, status.Phase)
}

func TestAutopilotController_RunSingleDecision_ControlFailureRetry(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping: shell-based test scripts use Unix echo semantics")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autopilot_test")
	require.NoError(t, err)

	// Create mock agent that fails
	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent", "exit 1")

	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:           "Test",
		MaxIterations:           10,
		ControlAgentType:        scriptPath,
		IterationTimeoutSeconds: 5,
	}

	// Worker returns waiting status to trigger retry
	workerCtrl := &MockPodController{
		workDir:     tmpDir,
		podKey:      "worker-123",
		agentStatus: "waiting",
	}

	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:  "autopilot-123",
		PodKey:        "worker-123",
		ProtoConfig:   protoConfig,
		PodCtrl:       workerCtrl,
		Reporter:      reporter,
		MCPPort:       19000,
	})

	// Stop must be called before removing tmpDir to avoid "no such file" errors
	defer func() {
		rp.Stop()
		os.RemoveAll(tmpDir)
	}()

	err = rp.Start()
	require.NoError(t, err)

	// Wait for error event (polling with timeout)
	deadline := time.Now().Add(10 * time.Second)
	hasError := false
	for time.Now().Before(deadline) {
		for _, e := range reporter.GetIterationEvents() {
			if e.Phase == "error" {
				hasError = true
				break
			}
		}
		if hasError {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.True(t, hasError, "Expected error event within timeout")
}

func TestAutopilotController_RunSingleDecision_WorkerNotWaitingAfterFailure(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping: shell-based test scripts use Unix echo semantics")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autopilot_test")
	require.NoError(t, err)

	// Create mock agent that fails
	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent", "exit 1")

	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:           "Test",
		MaxIterations:           10,
		ControlAgentType:        scriptPath,
		IterationTimeoutSeconds: 5,
	}

	// Worker returns executing status - should NOT retry
	workerCtrl := &MockPodController{
		workDir:     tmpDir,
		podKey:      "worker-123",
		agentStatus: "executing",
	}

	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:  "autopilot-123",
		PodKey:        "worker-123",
		ProtoConfig:   protoConfig,
		PodCtrl:       workerCtrl,
		Reporter:      reporter,
		MCPPort:       19000,
	})

	// Stop must be called before removing tmpDir to avoid "no such file" errors
	defer func() {
		rp.Stop()
		os.RemoveAll(tmpDir)
	}()

	// Manually trigger OnPodWaiting
	rp.OnPodWaiting()

	// Wait for error event (polling with timeout)
	deadline := time.Now().Add(10 * time.Second)
	hasError := false
	for time.Now().Before(deadline) {
		for _, e := range reporter.GetIterationEvents() {
			if e.Phase == "error" {
				hasError = true
				break
			}
		}
		if hasError {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Should only have 1 iteration attempt (no retry because worker is executing)
	assert.Equal(t, 1, rp.GetStatus().CurrentIteration)
}

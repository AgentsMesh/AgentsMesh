package autopilot

import (
	"errors"
	"os"
	"runtime"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		AutopilotKey:   "autopilot-123",
		PodKey:         "worker-123",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       reporter,
		ControlProcess: &MockControlProcess{},
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
		ControlAgentSlug:        scriptPath,
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
		AutopilotKey:   "autopilot-123",
		PodKey:         "worker-123",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       reporter,
		ControlProcess: &MockControlProcess{Err: errors.New("mock control failure")},
		MCPPort:        19000,
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
		ControlAgentSlug:        scriptPath,
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
		AutopilotKey:   "autopilot-123",
		PodKey:         "worker-123",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       reporter,
		ControlProcess: &MockControlProcess{},
		MCPPort:        19000,
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

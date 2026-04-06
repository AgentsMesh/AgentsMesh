package autopilot

import (
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
)

func TestAutopilotController_OnPodWaiting_UserTakeover(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		Prompt: "Test",
		MaxIterations: 10,
	}

	// Start with "executing" status so Start() doesn't trigger prompt
	workerCtrl := &MockPodController{
		workDir:     t.TempDir(),
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

	// User takes over
	rp.Takeover()

	initialIterationCount := len(reporter.GetIterationEvents())

	// OnPodWaiting should be skipped when user has taken over
	rp.OnPodWaiting()

	// No new iteration events
	assert.Equal(t, initialIterationCount, len(reporter.GetIterationEvents()))
}

func TestAutopilotController_OnPodWaiting_Paused(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		Prompt: "Test",
		MaxIterations: 10,
	}

	// Start with "executing" status so Start() doesn't trigger prompt
	workerCtrl := &MockPodController{
		workDir:     t.TempDir(),
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

	// Pause
	rp.Pause()

	initialIterationCount := len(reporter.GetIterationEvents())

	// OnPodWaiting should be skipped when paused
	rp.OnPodWaiting()

	// No new iteration events
	assert.Equal(t, initialIterationCount, len(reporter.GetIterationEvents()))
}

func TestAutopilotController_MaxIterations(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		Prompt: "Test",
		MaxIterations: 2, // Very low for testing
	}

	// Start with "executing" status so Start() doesn't trigger prompt
	workerCtrl := &MockPodController{
		workDir:     t.TempDir(),
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

	// Manually set iteration count to max
	rp.setIterationForTest(2)

	// OnPodWaiting should trigger max iterations phase
	rp.OnPodWaiting()

	status := rp.GetStatus()
	assert.Equal(t, PhaseMaxIterations, status.Phase)
}

func TestAutopilotController_OnPodWaiting_Completed(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		Prompt: "Test",
		MaxIterations: 10,
	}

	// Start with "executing" status so Start() doesn't trigger prompt
	workerCtrl := &MockPodController{
		workDir:     t.TempDir(),
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

	// Set phase to completed
	rp.setPhaseForTest(PhaseCompleted)

	initialIterationCount := len(reporter.GetIterationEvents())

	// OnPodWaiting should be skipped when completed
	rp.OnPodWaiting()

	// No new iteration events
	assert.Equal(t, initialIterationCount, len(reporter.GetIterationEvents()))
}

func TestAutopilotController_OnPodWaiting_Failed(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		Prompt: "Test",
		MaxIterations: 10,
	}

	// Start with "executing" status so Start() doesn't trigger prompt
	workerCtrl := &MockPodController{
		workDir:     t.TempDir(),
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

	// Set phase to failed
	rp.setPhaseForTest(PhaseFailed)

	initialIterationCount := len(reporter.GetIterationEvents())

	// OnPodWaiting should be skipped when failed
	rp.OnPodWaiting()

	// No new iteration events
	assert.Equal(t, initialIterationCount, len(reporter.GetIterationEvents()))
}

func TestAutopilotController_OnPodWaiting_RunsDecision(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		Prompt: "Test",
		MaxIterations: 10,
	}

	// Start with "executing" status so Start() doesn't trigger prompt
	// This allows us to test OnPodWaiting in isolation
	workerCtrl := &MockPodController{
		workDir:     t.TempDir(),
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

	initialIterationCount := len(reporter.GetIterationEvents())

	// OnPodWaiting should start decision process
	rp.OnPodWaiting()

	// Should report iteration started
	assert.Greater(t, len(reporter.GetIterationEvents()), initialIterationCount)

	// Iteration should have been incremented
	status := rp.GetStatus()
	assert.Equal(t, 1, status.CurrentIteration)
}

func TestAutopilotController_OnPodWaiting_MaxIterationsWithTerminatedEvent(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		Prompt: "Test",
		MaxIterations: 2,
	}

	// Start with "executing" status so Start() doesn't trigger prompt
	workerCtrl := &MockPodController{
		workDir:     t.TempDir(),
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

	// Manually set iteration count to max
	rp.setIterationForTest(2)

	// OnPodWaiting should trigger max iterations phase
	rp.OnPodWaiting()

	status := rp.GetStatus()
	assert.Equal(t, PhaseMaxIterations, status.Phase)

	// Should report terminated event with max_iterations reason
	terminatedEvents := reporter.GetTerminatedEvents()
	assert.Len(t, terminatedEvents, 1)
	assert.Equal(t, "max_iterations", terminatedEvents[0].Reason)
}

func TestAutopilotController_OnPodWaiting_Stopped(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		Prompt: "Test",
		MaxIterations: 10,
	}

	// Start with "executing" status so Start() doesn't trigger prompt
	workerCtrl := &MockPodController{
		workDir:     t.TempDir(),
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
	defer rp.Stop() // Ensure cleanup even if test fails early

	// Stop the pod
	rp.Stop()

	initialIterationCount := len(reporter.GetIterationEvents())

	// OnPodWaiting should be skipped when stopped
	rp.OnPodWaiting()

	// No new iteration events
	assert.Equal(t, initialIterationCount, len(reporter.GetIterationEvents()))
}

func TestAutopilotController_OnPodWaiting_WaitingApproval(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		Prompt: "Test",
		MaxIterations: 10,
	}

	workerCtrl := &MockPodController{
		workDir:     t.TempDir(),
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

	// Set phase to waiting_approval (Control requested human help)
	rp.setPhaseForTest(PhaseWaitingApproval)

	initialIterationCount := len(reporter.GetIterationEvents())

	// OnPodWaiting should be skipped when waiting for approval
	rp.OnPodWaiting()

	// No new iteration events
	assert.Equal(t, initialIterationCount, len(reporter.GetIterationEvents()))
}

func TestAutopilotController_Start_WithWaitingWorker(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		Prompt: "Test",
		MaxIterations: 10,
	}

	workerCtrl := &MockPodController{
		workDir:     t.TempDir(),
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
		ControlProcess: &MockControlProcess{},
	})

	// Start should succeed and initiate first iteration when worker is waiting
	err := rp.Start()
	assert.NoError(t, err)
	defer rp.Stop()

	// Should report created
	assert.Len(t, reporter.GetCreatedEvents(), 1)

	// Should be in running phase
	status := rp.GetStatus()
	assert.Equal(t, PhaseRunning, status.Phase)

	// Give goroutine time to start iteration (runSingleDecision runs in goroutine)
	time.Sleep(10 * time.Millisecond)

	// First iteration should be running
	assert.Equal(t, 1, rp.GetStatus().CurrentIteration)
}

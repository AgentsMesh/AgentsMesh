package autopilot

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
)

func TestAutopilotController_Pause_Resume(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test",
		MaxIterations: 10,
	}

	workDir := t.TempDir()
	workerCtrl := &MockPodController{
		workDir:     workDir,
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
	_ = rp.Start()
	defer rp.Stop()

	// Pause
	rp.Pause()
	status := rp.GetStatus()
	assert.Equal(t, PhasePaused, status.Phase)

	// Resume
	rp.Resume()
	status = rp.GetStatus()
	assert.Equal(t, PhaseRunning, status.Phase)
}

func TestAutopilotController_Takeover_Handback(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test",
		MaxIterations: 10,
	}

	workDir := t.TempDir()
	workerCtrl := &MockPodController{
		workDir:     workDir,
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
	_ = rp.Start()
	defer rp.Stop()

	// Takeover
	rp.Takeover()
	status := rp.GetStatus()
	assert.Equal(t, PhaseUserTakeover, status.Phase)

	// Handback
	rp.Handback()
	status = rp.GetStatus()
	assert.Equal(t, PhaseRunning, status.Phase)
}

func TestAutopilotController_Approve(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test",
		MaxIterations: 5,
	}

	workDir := t.TempDir()
	workerCtrl := &MockPodController{
		workDir:     workDir,
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
	_ = rp.Start()
	defer rp.Stop()

	// Set to waiting approval state (e.g., Control requested human help)
	rp.setPhaseForTest(PhaseWaitingApproval)

	// Approve with additional iterations
	rp.Approve(true, 5)

	status := rp.GetStatus()
	assert.Equal(t, PhaseRunning, status.Phase)
	assert.Equal(t, 10, status.MaxIterations) // 5 + 5
}

func TestAutopilotController_Approve_Stop(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test",
		MaxIterations: 5,
	}

	workDir := t.TempDir()
	workerCtrl := &MockPodController{
		workDir:     workDir,
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
	_ = rp.Start()
	defer rp.Stop()

	// Set to waiting approval state
	rp.setPhaseForTest(PhaseWaitingApproval)

	// Approve with continue=false should stop
	rp.Approve(false, 0)

	status := rp.GetStatus()
	assert.Equal(t, PhaseStopped, status.Phase)
}

func TestAutopilotController_Approve_NotWaitingApproval(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test",
		MaxIterations: 5,
	}

	workDir := t.TempDir()
	workerCtrl := &MockPodController{
		workDir:     workDir,
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
	_ = rp.Start()
	defer rp.Stop()

	// Phase is running, not waiting_approval
	statusBefore := rp.GetStatus()
	maxIterationsBefore := statusBefore.MaxIterations

	// Approve should be a no-op when not in waiting_approval phase
	rp.Approve(true, 5)

	statusAfter := rp.GetStatus()
	// Phase and max iterations should not change
	assert.Equal(t, PhaseRunning, statusAfter.Phase)
	assert.Equal(t, maxIterationsBefore, statusAfter.MaxIterations)
}

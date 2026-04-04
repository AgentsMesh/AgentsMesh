package autopilot

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutopilotController_NewAutopilotController(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:           "Test prompt",
		MaxIterations:           10,
		IterationTimeoutSeconds: 300,
		ApprovalTimeoutMinutes:  30,
	}

	workDir := t.TempDir()
	workerCtrl := &MockPodController{
		workDir:     workDir,
		podKey:      "worker-pod-123",
		agentStatus: "waiting",
	}

	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:   "autopilot-123",
		PodKey:         "worker-pod-123",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       reporter,
		ControlProcess: &MockControlProcess{},
	})

	assert.NotNil(t, rp)
	assert.Equal(t, "autopilot-123", rp.Key())
	assert.Equal(t, "worker-pod-123", rp.PodKey())

	status := rp.GetStatus()
	assert.Equal(t, PhaseInitializing, status.Phase)
	assert.Equal(t, 0, status.CurrentIteration)
}

func TestAutopilotController_Start(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt:           "Implement feature X",
		MaxIterations:           10,
		IterationTimeoutSeconds: 300,
	}

	workDir := t.TempDir()
	workerCtrl := &MockPodController{
		workDir:     workDir,
		podKey:      "worker-pod-123",
		agentStatus: "waiting",
	}

	reporter := &MockEventReporter{}

	rp := NewAutopilotController(Config{
		AutopilotKey:   "autopilot-123",
		PodKey:         "worker-pod-123",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       reporter,
		ControlProcess: &MockControlProcess{},
	})

	err := rp.Start()
	require.NoError(t, err)
	defer rp.Stop()

	// Should report created event
	createdEvents := reporter.GetCreatedEvents()
	assert.Len(t, createdEvents, 1)
	assert.Equal(t, "autopilot-123", createdEvents[0].AutopilotKey)

	// Phase should be running
	status := rp.GetStatus()
	assert.Equal(t, PhaseRunning, status.Phase)
}

func TestAutopilotController_Stop(t *testing.T) {
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

	rp.Stop()

	status := rp.GetStatus()
	assert.Equal(t, PhaseStopped, status.Phase)
	terminatedEvents := reporter.GetTerminatedEvents()
	assert.Len(t, terminatedEvents, 1)
	assert.Equal(t, "stopped", terminatedEvents[0].Reason)
}

func TestAutopilotController_Stop_AlreadyStopped(t *testing.T) {
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

	// Stop first time
	rp.Stop()
	terminatedCount := len(reporter.GetTerminatedEvents())

	// Stop again - should be no-op
	rp.Stop()

	// Should not report another terminated event
	assert.Equal(t, terminatedCount, len(reporter.GetTerminatedEvents()))
}

func TestAutopilotController_GetStatus(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test prompt",
		MaxIterations: 10,
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

	status := rp.GetStatus()

	assert.Equal(t, PhaseRunning, status.Phase)
	assert.Equal(t, 0, status.CurrentIteration)
	assert.Equal(t, 10, status.MaxIterations)
}

func TestAutopilotController_DefaultMaxIterations(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test",
		MaxIterations: 0, // Should use default of 10
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

	status := rp.GetStatus()
	assert.Equal(t, 10, status.MaxIterations) // Default value
}

func TestAutopilotController_Key_PodKey(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test",
	}

	workerCtrl := &MockPodController{}

	rp := NewAutopilotController(Config{
		AutopilotKey:   "autopilot-test-key",
		PodKey:         "worker-test-key",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       &MockEventReporter{},
		ControlProcess: &MockControlProcess{},
	})

	assert.Equal(t, "autopilot-test-key", rp.Key())
	assert.Equal(t, "worker-test-key", rp.PodKey())
}

func TestAutopilotController_NilReporter(t *testing.T) {
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

	// Create AutopilotController with nil reporter
	rp := NewAutopilotController(Config{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		ProtoConfig:  protoConfig,
		PodCtrl:      workerCtrl,
		Reporter:     nil, // nil reporter
	})

	// These operations should not panic with nil reporter
	err := rp.Start()
	assert.NoError(t, err)
	defer rp.Stop()

	rp.Pause()
	rp.Resume()
	rp.Takeover()
	rp.Handback()
}

func TestAutopilotController_Start_ExecutingStatus(t *testing.T) {
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test",
		MaxIterations: 10,
	}

	workDir := t.TempDir()
	workerCtrl := &MockPodController{
		workDir:     workDir,
		podKey:      "worker-123",
		agentStatus: "executing", // Not waiting
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

	err := rp.Start()
	assert.NoError(t, err)
	defer rp.Stop()

	// Should not have sent initial prompt when executing
	assert.Len(t, workerCtrl.sendTextCalls, 0)

	// Should still report created
	assert.Len(t, reporter.GetCreatedEvents(), 1)
}

func TestAutopilotController_SessionID(t *testing.T) {
	t.Skip("SessionID extraction is specific to exec mode; ACP manages sessions internally")
	protoConfig := &runnerv1.AutopilotConfig{
		InitialPrompt: "Test",
		MaxIterations: 10,
	}

	workDir := t.TempDir()
	workerCtrl := &MockPodController{
		workDir:     workDir,
		podKey:      "worker-123",
		agentStatus: "executing",
	}

	rp := NewAutopilotController(Config{
		AutopilotKey:   "autopilot-123",
		PodKey:         "worker-123",
		ProtoConfig:    protoConfig,
		PodCtrl:        workerCtrl,
		Reporter:       &MockEventReporter{},
		ControlProcess: &MockControlProcess{},
	})

	// Session ID should be empty initially
	assert.Empty(t, rp.getSessionID())

	// After extracting from output, it should be set
	rp.extractSessionID(`{"session_id": "test-session-123", "result": "ok"}`)
	assert.Equal(t, "test-session-123", rp.getSessionID())
}

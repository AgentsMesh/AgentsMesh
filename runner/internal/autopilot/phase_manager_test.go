package autopilot

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
)

func TestPhaseManager_NewPhaseManager(t *testing.T) {
	statusGetter := func() *runnerv1.AutopilotStatus {
		return &runnerv1.AutopilotStatus{
			CurrentIteration: 1,
			MaxIterations:    10,
		}
	}

	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		Reporter:     &MockEventReporter{},
		StatusGetter: statusGetter,
	})

	assert.NotNil(t, pm)
	assert.Equal(t, PhaseInitializing, pm.GetPhase())
}

func TestPhaseManager_SetPhase(t *testing.T) {
	reporter := &MockEventReporter{}
	statusGetter := func() *runnerv1.AutopilotStatus {
		return &runnerv1.AutopilotStatus{
			CurrentIteration: 1,
			MaxIterations:    10,
		}
	}

	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		Reporter:     reporter,
		StatusGetter: statusGetter,
	})

	// Set to a new phase
	changed := pm.SetPhase(PhaseRunning)
	assert.True(t, changed)
	assert.Equal(t, PhaseRunning, pm.GetPhase())
	assert.Len(t, reporter.GetStatusEvents(), 1)

	// Set to the same phase - should return false
	changed = pm.SetPhase(PhaseRunning)
	assert.False(t, changed)
	assert.Len(t, reporter.GetStatusEvents(), 1) // No new event
}

func TestPhaseManager_SetPhaseWithoutReport(t *testing.T) {
	reporter := &MockEventReporter{}
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		Reporter:     reporter,
	})

	// Set phase without reporting
	changed := pm.SetPhaseWithoutReport(PhaseRunning)
	assert.True(t, changed)
	assert.Equal(t, PhaseRunning, pm.GetPhase())
	assert.Len(t, reporter.GetStatusEvents(), 0) // No event

	// Set to same phase
	changed = pm.SetPhaseWithoutReport(PhaseRunning)
	assert.False(t, changed)
}

func TestPhaseManager_IsTerminalPhase(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})

	// Non-terminal phases
	pm.SetPhaseWithoutReport(PhaseInitializing)
	assert.False(t, pm.IsTerminalPhase())

	pm.SetPhaseWithoutReport(PhaseRunning)
	assert.False(t, pm.IsTerminalPhase())

	pm.SetPhaseWithoutReport(PhasePaused)
	assert.False(t, pm.IsTerminalPhase())

	pm.SetPhaseWithoutReport(PhaseUserTakeover)
	assert.False(t, pm.IsTerminalPhase())

	pm.SetPhaseWithoutReport(PhaseWaitingApproval)
	assert.False(t, pm.IsTerminalPhase())

	// Terminal phases
	pm.SetPhaseWithoutReport(PhaseCompleted)
	assert.True(t, pm.IsTerminalPhase())

	pm.SetPhaseWithoutReport(PhaseFailed)
	assert.True(t, pm.IsTerminalPhase())

	pm.SetPhaseWithoutReport(PhaseStopped)
	assert.True(t, pm.IsTerminalPhase())

	pm.SetPhaseWithoutReport(PhaseMaxIterations)
	assert.True(t, pm.IsTerminalPhase())
}

func TestPhaseManager_CanProcessIteration(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})

	// Phases that can process iterations
	pm.SetPhaseWithoutReport(PhaseInitializing)
	assert.True(t, pm.CanProcessIteration())

	pm.SetPhaseWithoutReport(PhaseRunning)
	assert.True(t, pm.CanProcessIteration())

	pm.SetPhaseWithoutReport(PhaseUserTakeover)
	assert.True(t, pm.CanProcessIteration())

	// Phases that cannot process iterations
	pm.SetPhaseWithoutReport(PhasePaused)
	assert.False(t, pm.CanProcessIteration())

	pm.SetPhaseWithoutReport(PhaseStopped)
	assert.False(t, pm.CanProcessIteration())

	pm.SetPhaseWithoutReport(PhaseCompleted)
	assert.False(t, pm.CanProcessIteration())

	pm.SetPhaseWithoutReport(PhaseFailed)
	assert.False(t, pm.CanProcessIteration())

	pm.SetPhaseWithoutReport(PhaseWaitingApproval)
	assert.False(t, pm.CanProcessIteration())

	pm.SetPhaseWithoutReport(PhaseMaxIterations)
	assert.False(t, pm.CanProcessIteration())
}

func TestPhaseManager_TransitionToRunning(t *testing.T) {
	reporter := &MockEventReporter{}
	statusGetter := func() *runnerv1.AutopilotStatus {
		return &runnerv1.AutopilotStatus{}
	}

	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		Reporter:     reporter,
		StatusGetter: statusGetter,
	})

	// From initializing
	pm.SetPhaseWithoutReport(PhaseInitializing)
	changed := pm.TransitionToRunning()
	assert.True(t, changed)
	assert.Equal(t, PhaseRunning, pm.GetPhase())

	// From paused
	pm.SetPhaseWithoutReport(PhasePaused)
	changed = pm.TransitionToRunning()
	assert.True(t, changed)
	assert.Equal(t, PhaseRunning, pm.GetPhase())

	// From other phase - should not transition
	pm.SetPhaseWithoutReport(PhaseCompleted)
	changed = pm.TransitionToRunning()
	assert.False(t, changed)
	assert.Equal(t, PhaseCompleted, pm.GetPhase())
}

func TestPhaseManager_ReportStatus(t *testing.T) {
	reporter := &MockEventReporter{}
	statusGetter := func() *runnerv1.AutopilotStatus {
		return &runnerv1.AutopilotStatus{
			CurrentIteration: 5,
			MaxIterations:    10,
		}
	}

	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		Reporter:     reporter,
		StatusGetter: statusGetter,
	})

	pm.SetPhaseWithoutReport(PhaseRunning)
	pm.ReportStatus()

	statusEvents := reporter.GetStatusEvents()
	assert.Len(t, statusEvents, 1)
	assert.Equal(t, "running", statusEvents[0].Status.Phase)
}

func TestPhaseManager_ReportStatus_NilReporter(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		Reporter:     nil,
	})

	// Should not panic
	pm.ReportStatus()
}

func TestPhaseManager_ReportStatus_NilStatusGetter(t *testing.T) {
	reporter := &MockEventReporter{}
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		Reporter:     reporter,
		StatusGetter: nil,
	})

	// Should not panic and not report
	pm.ReportStatus()
	assert.Len(t, reporter.GetStatusEvents(), 0)
}

func TestPhaseManager_ReportStatus_StatusGetterReturnsNil(t *testing.T) {
	reporter := &MockEventReporter{}
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		Reporter:     reporter,
		StatusGetter: func() *runnerv1.AutopilotStatus { return nil },
	})

	// Should not panic and not report
	pm.ReportStatus()
	assert.Len(t, reporter.GetStatusEvents(), 0)
}

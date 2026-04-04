package autopilot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserInteractionHandler_NewUserInteractionHandler(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})

	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
	})

	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager:        pm,
		IterationController: ic,
		Logger:              nil,
	})

	assert.NotNil(t, uih)
	assert.False(t, uih.IsUserTakeover())
}

func TestUserInteractionHandler_Takeover(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})
	pm.SetPhaseWithoutReport(PhaseRunning)

	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager: pm,
	})

	uih.Takeover()

	assert.True(t, uih.IsUserTakeover())
	assert.Equal(t, PhaseUserTakeover, pm.GetPhase())
}

func TestUserInteractionHandler_Handback(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})

	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager: pm,
	})

	// First takeover
	uih.Takeover()
	assert.True(t, uih.IsUserTakeover())

	// Then handback
	uih.Handback()
	assert.False(t, uih.IsUserTakeover())
	assert.Equal(t, PhaseRunning, pm.GetPhase())
}

func TestUserInteractionHandler_Approve_Continue(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})
	pm.SetPhaseWithoutReport(PhaseWaitingApproval)

	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 5,
	})

	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager:        pm,
		IterationController: ic,
	})

	uih.Approve(true, 5)

	assert.Equal(t, PhaseRunning, pm.GetPhase())
	assert.Equal(t, 10, ic.GetMaxIterations()) // 5 + 5
}

func TestUserInteractionHandler_Approve_Stop(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})
	pm.SetPhaseWithoutReport(PhaseWaitingApproval)

	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager: pm,
	})

	uih.Approve(false, 0)

	assert.Equal(t, PhaseStopped, pm.GetPhase())
}

func TestUserInteractionHandler_Approve_NotWaitingApproval(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})
	pm.SetPhaseWithoutReport(PhaseRunning)

	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 5,
	})

	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager:        pm,
		IterationController: ic,
	})

	// Approve when not in waiting_approval phase - should be no-op
	uih.Approve(true, 5)

	assert.Equal(t, PhaseRunning, pm.GetPhase())
	assert.Equal(t, 5, ic.GetMaxIterations()) // Unchanged
}

func TestUserInteractionHandler_Approve_ZeroAdditionalIterations(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})
	pm.SetPhaseWithoutReport(PhaseWaitingApproval)

	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 5,
	})

	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager:        pm,
		IterationController: ic,
	})

	// Approve with zero additional iterations
	uih.Approve(true, 0)

	assert.Equal(t, PhaseRunning, pm.GetPhase())
	assert.Equal(t, 5, ic.GetMaxIterations()) // Unchanged
}

func TestUserInteractionHandler_Approve_NilIterationController(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})
	pm.SetPhaseWithoutReport(PhaseWaitingApproval)

	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager:        pm,
		IterationController: nil, // No iteration controller
	})

	// Should not panic
	uih.Approve(true, 5)

	assert.Equal(t, PhaseRunning, pm.GetPhase())
}

func TestUserInteractionHandler_TakeoverChannel(t *testing.T) {
	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager: NewPhaseManager(PhaseManagerConfig{
			AutopilotKey: "autopilot-123",
			PodKey:       "worker-123",
		}),
	})

	ch := uih.TakeoverChannel()
	assert.NotNil(t, ch)

	// Trigger takeover
	uih.Takeover()

	// Should receive signal
	select {
	case <-ch:
		// OK
	default:
		t.Fatal("Expected to receive from takeover channel")
	}
}

func TestUserInteractionHandler_HandbackChannel(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})

	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager: pm,
	})

	ch := uih.HandbackChannel()
	assert.NotNil(t, ch)

	// First takeover then handback
	uih.Takeover()
	uih.Handback()

	// Should receive signal
	select {
	case <-ch:
		// OK
	default:
		t.Fatal("Expected to receive from handback channel")
	}
}

func TestUserInteractionHandler_TakeoverChannel_NonBlocking(t *testing.T) {
	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager: NewPhaseManager(PhaseManagerConfig{
			AutopilotKey: "autopilot-123",
			PodKey:       "worker-123",
		}),
	})

	// Multiple takeovers should not block (channel is buffered)
	uih.Takeover()
	uih.Takeover() // Should not block even though channel has 1 item
}

func TestUserInteractionHandler_HandbackChannel_NonBlocking(t *testing.T) {
	pm := NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
	})

	uih := NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager: pm,
	})

	// Multiple handbacks should not block
	uih.Takeover()
	uih.Handback()
	uih.Takeover()
	uih.Handback() // Should not block
}

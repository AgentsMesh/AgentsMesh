package autopilot

import (
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/terminal/detector"
	"github.com/stretchr/testify/assert"
)

func TestStateDetectorCoordinator_NewStateDetectorCoordinator(t *testing.T) {
	mockDetector := NewMockStateDetector()
	var calledWith detector.AgentState

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector: mockDetector,
		OnWaiting: func() {
			calledWith = detector.StateWaiting
		},
		CheckPeriod: 100 * time.Millisecond,
		Logger:      nil,
		AutopilotKey: "autopilot-123",
	})

	assert.NotNil(t, sdc)

	// Simulate state change from executing to waiting
	mockDetector.SetState(detector.StateExecuting)
	mockDetector.SetState(detector.StateWaiting)

	// Callback should have been triggered
	assert.Equal(t, detector.StateWaiting, calledWith)
}

func TestStateDetectorCoordinator_DefaultCheckPeriod(t *testing.T) {
	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector:    nil,
		CheckPeriod: 0, // Should use default
	})

	assert.NotNil(t, sdc)
	assert.Equal(t, 500*time.Millisecond, sdc.checkPeriod)
}

func TestStateDetectorCoordinator_NilDetector(t *testing.T) {
	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector: nil,
	})

	// Start should not panic with nil detector
	sdc.Start()

	// Stop should not panic
	sdc.Stop()
}

func TestStateDetectorCoordinator_Start(t *testing.T) {
	mockDetector := NewMockStateDetector()

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector:    mockDetector,
		CheckPeriod: 50 * time.Millisecond,
	})

	sdc.Start()

	// Wait for a few detection cycles
	time.Sleep(150 * time.Millisecond)

	sdc.Stop()
}

func TestStateDetectorCoordinator_Stop(t *testing.T) {
	mockDetector := NewMockStateDetector()

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector:    mockDetector,
		CheckPeriod: 50 * time.Millisecond,
	})

	sdc.Start()
	time.Sleep(60 * time.Millisecond)

	sdc.Stop()

	// Context should be done
	select {
	case <-sdc.GetContext().Done():
		// OK
	default:
		t.Fatal("Context should be done after stop")
	}
}

func TestStateDetectorCoordinator_GetContext(t *testing.T) {
	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector: nil,
	})

	ctx := sdc.GetContext()
	assert.NotNil(t, ctx)

	// Context should not be done initially
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be done yet")
	default:
		// OK
	}
}

func TestStateDetectorCoordinator_CallbackOnlyOnExecutingToWaiting(t *testing.T) {
	mockDetector := NewMockStateDetector()
	var callCount atomic.Int32

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector: mockDetector,
		OnWaiting: func() {
			callCount.Add(1)
		},
	})

	// Start from not_running
	mockDetector.SetState(detector.StateNotRunning)
	assert.Equal(t, int32(0), callCount.Load())

	// Transition to waiting (not from executing) - should NOT trigger
	mockDetector.SetState(detector.StateWaiting)
	assert.Equal(t, int32(0), callCount.Load())

	// Transition to executing
	mockDetector.SetState(detector.StateExecuting)
	assert.Equal(t, int32(0), callCount.Load())

	// Transition from executing to waiting - SHOULD trigger
	mockDetector.SetState(detector.StateWaiting)
	// Note: callback is triggered in a goroutine, wait a bit
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, int32(1), callCount.Load())

	// Stop coordinator
	sdc.Stop()
}

func TestStateDetectorCoordinator_RunStateDetection(t *testing.T) {
	mockDetector := NewMockStateDetector()

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector:    mockDetector,
		CheckPeriod: 30 * time.Millisecond,
	})

	sdc.Start()

	// Wait for multiple detection cycles
	time.Sleep(100 * time.Millisecond)

	sdc.Stop()

	// Should have detected multiple times (using the counter in MockStateDetector)
	assert.GreaterOrEqual(t, mockDetector.GetDetectCallCount(), 2)
}

func TestStateDetectorCoordinator_NilCallback(t *testing.T) {
	mockDetector := NewMockStateDetector()

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector:  mockDetector,
		OnWaiting: nil, // No callback
	})

	// Should not panic when state changes
	mockDetector.SetState(detector.StateExecuting)
	mockDetector.SetState(detector.StateWaiting)

	sdc.Stop()
}

// Test AgentState constants
func TestAgentState_Constants(t *testing.T) {
	assert.Equal(t, detector.AgentState("not_running"), detector.StateNotRunning)
	assert.Equal(t, detector.AgentState("executing"), detector.StateExecuting)
	assert.Equal(t, detector.AgentState("waiting"), detector.StateWaiting)
}

func TestStateDetectorCoordinator_WithLogger(t *testing.T) {
	mockDetector := NewMockStateDetector()
	var calledWith detector.AgentState

	// Create a logger to cover the log branch
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector: mockDetector,
		OnWaiting: func() {
			calledWith = detector.StateWaiting
		},
		CheckPeriod: 100 * time.Millisecond,
		Logger:      logger,
		AutopilotKey: "autopilot-123",
	})

	assert.NotNil(t, sdc)

	// Simulate state change from executing to waiting - should trigger log
	mockDetector.SetState(detector.StateExecuting)
	mockDetector.SetState(detector.StateWaiting)

	// Callback should have been triggered
	assert.Equal(t, detector.StateWaiting, calledWith)

	sdc.Stop()
}

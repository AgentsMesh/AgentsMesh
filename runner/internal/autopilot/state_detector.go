// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// AgentState represents the detected state of an AI agent.
// This mirrors terminal.AgentState to avoid direct dependency.
type AgentState string

const (
	// StateNotRunning indicates the agent is not running.
	StateNotRunning AgentState = "not_running"
	// StateExecuting indicates the agent is actively executing.
	StateExecuting AgentState = "executing"
	// StateWaiting indicates the agent is waiting for user input.
	StateWaiting AgentState = "waiting"
)

// StateChangeCallback is called when the agent state changes.
type StateChangeCallback func(newState, prevState AgentState)

// StateDetector is an interface for detecting terminal/agent state.
// This abstraction allows AutopilotController to be decoupled from the concrete
// terminal.TerminalStateDetector implementation.
type StateDetector interface {
	// DetectState analyzes and returns the current agent state.
	DetectState() AgentState
	// GetState returns the current state without performing detection.
	GetState() AgentState
	// SetCallback sets the state change callback.
	SetCallback(cb StateChangeCallback)
	// Reset resets the detector state.
	Reset()
	// OnOutput should be called when terminal output is received.
	OnOutput(bytes int)
	// OnScreenUpdate should be called with current screen lines after each Feed.
	// This enables single-direction data flow without reverse lock acquisition.
	OnScreenUpdate(lines []string)
}

// StateDetectorCoordinator coordinates state detection and triggers callbacks.
// It runs periodic state detection and converts terminal states to AutopilotController callbacks.
type StateDetectorCoordinator struct {
	detector     StateDetector
	onWaiting    func() // Called when pod transitions to waiting
	ctx          context.Context
	cancel       context.CancelFunc
	log          *slog.Logger
	checkPeriod  time.Duration
	autopilotKey string
}

// StateDetectorCoordinatorConfig contains configuration for StateDetectorCoordinator.
type StateDetectorCoordinatorConfig struct {
	Detector     StateDetector
	OnWaiting    func()
	CheckPeriod  time.Duration
	Logger       *slog.Logger
	AutopilotKey string
}

// NewStateDetectorCoordinator creates a new StateDetectorCoordinator.
func NewStateDetectorCoordinator(cfg StateDetectorCoordinatorConfig) *StateDetectorCoordinator {
	checkPeriod := cfg.CheckPeriod
	if checkPeriod == 0 {
		checkPeriod = DefaultStateCheckPeriod
	}

	ctx, cancel := context.WithCancel(context.Background())

	sdc := &StateDetectorCoordinator{
		detector:     cfg.Detector,
		onWaiting:    cfg.OnWaiting,
		ctx:          ctx,
		cancel:       cancel,
		log:          cfg.Logger,
		checkPeriod:  checkPeriod,
		autopilotKey: cfg.AutopilotKey,
	}

	// Setup callback if detector is provided
	if cfg.Detector != nil {
		cfg.Detector.SetCallback(func(newState, prevState AgentState) {
			// Only trigger when transitioning from executing to waiting
			if newState == StateWaiting && prevState == StateExecuting {
				if sdc.log != nil {
					sdc.log.Debug("StateDetector: Pod transitioned to waiting",
						"autopilot_key", sdc.autopilotKey,
						"prev_state", prevState,
						"new_state", newState)
				}
				if sdc.onWaiting != nil {
					sdc.onWaiting()
				}
			}
		})
	}

	return sdc
}

// Start begins the periodic state detection loop.
func (sdc *StateDetectorCoordinator) Start() {
	if sdc.detector == nil {
		if sdc.log != nil {
			sdc.log.Warn("StateDetector not available, state detection disabled",
				"autopilot_key", sdc.autopilotKey)
		}
		return
	}

	go sdc.runStateDetection()
}

// Stop stops the state detection loop.
func (sdc *StateDetectorCoordinator) Stop() {
	sdc.cancel()
	if sdc.log != nil {
		sdc.log.Info("StateDetectorCoordinator stopped", "autopilot_key", sdc.autopilotKey)
	}
}

// runStateDetection runs the periodic state detection loop.
func (sdc *StateDetectorCoordinator) runStateDetection() {
	ticker := time.NewTicker(sdc.checkPeriod)
	defer ticker.Stop()

	if sdc.log != nil {
		sdc.log.Info("StateDetectorCoordinator started periodic detection",
			"autopilot_key", sdc.autopilotKey,
			"check_period", sdc.checkPeriod)
	}

	for {
		select {
		case <-sdc.ctx.Done():
			if sdc.log != nil {
				sdc.log.Debug("StateDetectorCoordinator stopped",
					"autopilot_key", sdc.autopilotKey)
			}
			return
		case <-ticker.C:
			if sdc.detector != nil {
				state := sdc.detector.DetectState()
				logger.AutopilotTrace().Trace("StateDetectorCoordinator tick",
					"autopilot_key", sdc.autopilotKey,
					"detected_state", state)
			}
		}
	}
}

// GetContext returns the coordinator's context.
func (sdc *StateDetectorCoordinator) GetContext() context.Context {
	return sdc.ctx
}

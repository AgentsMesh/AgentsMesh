// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"context"
	"log/slog"
	"time"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/logger"
	"github.com/AgentsMesh/AgentsMesh/runner/internal/terminal/detector"
)

// StateDetectorCoordinator coordinates state detection and triggers callbacks.
// It runs periodic state detection and converts terminal states to AutopilotController callbacks.
// The coordinator uses detector.StateDetector interface from the terminal/detector package,
// which is a foundational service independent of Autopilot.
type StateDetectorCoordinator struct {
	detector     detector.StateDetector
	onWaiting    func() // Called when pod transitions to waiting
	ctx          context.Context
	cancel       context.CancelFunc
	log          *slog.Logger
	checkPeriod  time.Duration
	autopilotKey string
}

// StateDetectorCoordinatorConfig contains configuration for StateDetectorCoordinator.
type StateDetectorCoordinatorConfig struct {
	Detector     detector.StateDetector
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
		cfg.Detector.SetCallback(func(newState, prevState detector.AgentState) {
			// Only trigger when transitioning from executing to waiting
			if newState == detector.StateWaiting && prevState == detector.StateExecuting {
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

// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"log/slog"
	"sync"
)

// StateDetectorCoordinator listens for Pod state changes and triggers
// the OnPodWaiting callback when the agent transitions from executing to waiting.
//
// It is mode-agnostic: PTY state changes arrive via ManagedStateDetector → PodIO,
// and ACP state changes arrive via ACPClient.OnStateChange → ACPPodIO.
type StateDetectorCoordinator struct {
	podCtrl      TargetPodController
	onWaiting    func()
	subscribeID  string
	log          *slog.Logger
	autopilotKey string

	mu         sync.Mutex
	lastStatus string
	started    bool
}

// StateDetectorCoordinatorConfig contains configuration for StateDetectorCoordinator.
type StateDetectorCoordinatorConfig struct {
	PodCtrl      TargetPodController
	OnWaiting    func()
	Logger       *slog.Logger
	AutopilotKey string
}

// NewStateDetectorCoordinator creates a new event-driven state coordinator.
func NewStateDetectorCoordinator(cfg StateDetectorCoordinatorConfig) *StateDetectorCoordinator {
	return &StateDetectorCoordinator{
		podCtrl:      cfg.PodCtrl,
		onWaiting:    cfg.OnWaiting,
		subscribeID:  "autopilot-state-" + cfg.AutopilotKey,
		log:          cfg.Logger,
		autopilotKey: cfg.AutopilotKey,
	}
}

// Start subscribes to Pod state change events via PodIO.
func (sdc *StateDetectorCoordinator) Start() {
	sdc.mu.Lock()
	if sdc.started {
		sdc.mu.Unlock()
		return
	}
	sdc.started = true
	sdc.mu.Unlock()

	if sdc.podCtrl == nil {
		if sdc.log != nil {
			sdc.log.Warn("PodController not available, state detection disabled",
				"autopilot_key", sdc.autopilotKey)
		}
		return
	}

	sdc.podCtrl.SubscribeStateChange(sdc.subscribeID, func(newStatus string) {
		sdc.mu.Lock()
		oldStatus := sdc.lastStatus
		sdc.lastStatus = newStatus
		sdc.mu.Unlock()

		// Trigger when agent transitions from executing → waiting (idle).
		// "waiting" means the PTY agent is waiting for input.
		// "idle" means the ACP agent finished processing and is ready for next prompt.
		if oldStatus == "executing" && (newStatus == "waiting" || newStatus == "idle") {
			if sdc.log != nil {
				sdc.log.Debug("Pod transitioned to waiting/idle",
					"autopilot_key", sdc.autopilotKey,
					"prev_status", oldStatus,
					"new_status", newStatus)
			}
			if sdc.onWaiting != nil {
				sdc.onWaiting()
			}
		}
	})

	if sdc.log != nil {
		sdc.log.Info("StateDetectorCoordinator started (event-driven)",
			"autopilot_key", sdc.autopilotKey)
	}
}

// Stop unsubscribes from Pod state change events.
func (sdc *StateDetectorCoordinator) Stop() {
	sdc.mu.Lock()
	if !sdc.started {
		sdc.mu.Unlock()
		return
	}
	sdc.started = false
	sdc.mu.Unlock()

	if sdc.podCtrl != nil {
		sdc.podCtrl.UnsubscribeStateChange(sdc.subscribeID)
	}

	if sdc.log != nil {
		sdc.log.Info("StateDetectorCoordinator stopped", "autopilot_key", sdc.autopilotKey)
	}
}

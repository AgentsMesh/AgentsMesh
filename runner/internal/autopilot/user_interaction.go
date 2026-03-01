// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"log/slog"
	"sync"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/safego"
)

// UserInteractionHandler manages user takeover, handback, and approval interactions.
type UserInteractionHandler struct {
	mu           sync.RWMutex
	userTakeover bool
	takeoverCh   chan struct{}
	handbackCh   chan struct{}

	// Dependencies
	phaseMgr *PhaseManager
	iterCtrl *IterationController
	log      *slog.Logger

	// Callback to check Pod status and trigger iteration after resume
	onResumeCallback func()
}

// UserInteractionConfig contains configuration for creating a UserInteractionHandler.
type UserInteractionConfig struct {
	PhaseManager        *PhaseManager
	IterationController *IterationController
	Logger              *slog.Logger
	OnResumeCallback    func() // Called after handback/approve to check if iteration needed
}

// NewUserInteractionHandler creates a new UserInteractionHandler instance.
func NewUserInteractionHandler(cfg UserInteractionConfig) *UserInteractionHandler {
	return &UserInteractionHandler{
		takeoverCh:       make(chan struct{}, 1),
		handbackCh:       make(chan struct{}, 1),
		phaseMgr:         cfg.PhaseManager,
		iterCtrl:         cfg.IterationController,
		log:              cfg.Logger,
		onResumeCallback: cfg.OnResumeCallback,
	}
}

// IsUserTakeover returns true if user has taken over control.
func (uih *UserInteractionHandler) IsUserTakeover() bool {
	uih.mu.RLock()
	defer uih.mu.RUnlock()
	return uih.userTakeover
}

// Takeover allows the user to take control from AutopilotController.
func (uih *UserInteractionHandler) Takeover() {
	uih.mu.Lock()
	uih.userTakeover = true
	uih.mu.Unlock()

	uih.phaseMgr.SetPhase(PhaseUserTakeover)

	select {
	case uih.takeoverCh <- struct{}{}:
	default:
	}

	if uih.log != nil {
		uih.log.Info("User takeover")
	}
}

// Handback returns control from user to AutopilotController.
func (uih *UserInteractionHandler) Handback() {
	uih.mu.Lock()
	uih.userTakeover = false
	uih.mu.Unlock()

	uih.phaseMgr.SetPhase(PhaseRunning)

	select {
	case uih.handbackCh <- struct{}{}:
	default:
	}

	if uih.log != nil {
		uih.log.Info("User handback")
	}

	// Trigger resume callback to check if Pod is waiting and needs iteration
	if uih.onResumeCallback != nil {
		safego.Go("resume-callback", uih.onResumeCallback)
	}
}

// Approve handles approval when Control requests human help (NEED_HUMAN_HELP).
// continueExecution: if true, continue execution; if false, stop the AutopilotController.
// additionalIterations: number of additional iterations to allow.
func (uih *UserInteractionHandler) Approve(continueExecution bool, additionalIterations int32) {
	if uih.phaseMgr.GetPhase() != PhaseWaitingApproval {
		return
	}

	if !continueExecution {
		uih.phaseMgr.SetPhase(PhaseStopped)
		if uih.log != nil {
			uih.log.Info("AutopilotController stopped by user decision")
		}
		return
	}

	// Add additional iterations
	if additionalIterations > 0 && uih.iterCtrl != nil {
		newMax := uih.iterCtrl.AddMaxIterations(int(additionalIterations))
		if uih.log != nil {
			uih.log.Info("AutopilotController approved to continue",
				"additional_iterations", additionalIterations,
				"new_max", newMax)
		}
	}

	uih.phaseMgr.SetPhase(PhaseRunning)

	// Trigger resume callback to check if Pod is waiting and needs iteration
	if uih.onResumeCallback != nil {
		safego.Go("resume-callback", uih.onResumeCallback)
	}
}

// TakeoverChannel returns the takeover notification channel.
func (uih *UserInteractionHandler) TakeoverChannel() <-chan struct{} {
	return uih.takeoverCh
}

// HandbackChannel returns the handback notification channel.
func (uih *UserInteractionHandler) HandbackChannel() <-chan struct{} {
	return uih.handbackCh
}

// SetOnResumeCallback sets the callback for resume events.
// Used to resolve circular initialization dependency.
func (uih *UserInteractionHandler) SetOnResumeCallback(callback func()) {
	uih.mu.Lock()
	defer uih.mu.Unlock()
	uih.onResumeCallback = callback
}

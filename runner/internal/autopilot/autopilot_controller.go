// Package autopilot implements the AutopilotController for supervised Pod automation.
// AutopilotController orchestrates Pod execution by detecting when the controlled pod
// is waiting for input and automatically providing the next instruction.
package autopilot

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// AutopilotController is a supervised automation controller that orchestrates
// a Pod to complete tasks autonomously.
//
// This is a thin coordinator that delegates to specialized components:
// - PhaseManager: lifecycle phase transitions
// - IterationController: iteration counting and max iteration protection
// - UserInteractionHandler: takeover/handback/approve handling
// - StateDetectorCoordinator: terminal state detection
// - ControlRunner: control process execution
// - ProgressTracker: file/git change tracking
type AutopilotController struct {
	key    string
	podKey string
	config *runnerv1.AutopilotConfig

	// Pod controller
	podCtrl PodController

	// MCP port for control process to connect to
	mcpPort int

	// MCP config file path for control process
	mcpConfigPath string

	// Component delegates (SRP compliance)
	phaseMgr         *PhaseManager
	iterCtrl         *IterationController
	userHandler      *UserInteractionHandler
	stateCoordinator *StateDetectorCoordinator
	controlRunner    *ControlRunner
	promptBuilder    *PromptBuilder
	progressTracker  *ProgressTracker

	// Status mutex for LastDecision fields
	decisionMu      sync.RWMutex
	lastDecision    string
	lastDecisionMsg string

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	// Event reporting
	reporter EventReporter

	// Logger
	log *slog.Logger
}

// Config contains configuration for creating an AutopilotController.
type Config struct {
	AutopilotKey string
	PodKey       string
	ProtoConfig  *runnerv1.AutopilotConfig
	PodCtrl      PodController
	Reporter     EventReporter
	MCPPort      int // MCP HTTP Server port for control process
}

// NewAutopilotController creates a new AutopilotController instance.
func NewAutopilotController(cfg Config) *AutopilotController {
	ctx, cancel := context.WithCancel(context.Background())
	log := logger.Autopilot()

	mcpPort := cfg.MCPPort
	if mcpPort == 0 {
		mcpPort = DefaultMCPPort
	}

	// Create MCP config file for Control Agent
	mcpConfigPath, err := createMCPConfigFile(cfg.PodCtrl.GetWorkDir(), cfg.PodKey, mcpPort)
	if err != nil {
		log.Warn("Failed to create MCP config file, Control will use curl fallback",
			"error", err)
	}

	ac := &AutopilotController{
		key:           cfg.AutopilotKey,
		podKey:        cfg.PodKey,
		config:        cfg.ProtoConfig,
		podCtrl:       cfg.PodCtrl,
		mcpPort:       mcpPort,
		mcpConfigPath: mcpConfigPath,
		ctx:           ctx,
		cancel:        cancel,
		reporter:      cfg.Reporter,
		log:           log,
	}

	// Initialize IterationController
	ac.iterCtrl = NewIterationController(IterationControllerConfig{
		MaxIterations: int(cfg.ProtoConfig.MaxIterations),
		MinTriggerGap: 5 * time.Second,
		Reporter:      cfg.Reporter,
		AutopilotKey:  cfg.AutopilotKey,
		PodKey:        cfg.PodKey,
		Logger:        log,
	})

	// Initialize PhaseManager with status getter callback
	ac.phaseMgr = NewPhaseManager(PhaseManagerConfig{
		AutopilotKey: cfg.AutopilotKey,
		PodKey:       cfg.PodKey,
		Reporter:     cfg.Reporter,
		StatusGetter: ac.buildAutopilotStatus,
	})

	// Initialize UserInteractionHandler
	// Note: OnResumeCallback set after creation due to circular reference
	ac.userHandler = NewUserInteractionHandler(UserInteractionConfig{
		PhaseManager:        ac.phaseMgr,
		IterationController: ac.iterCtrl,
		Logger:              log,
		OnResumeCallback:    nil, // Set below
	})

	// Initialize PromptBuilder
	ac.promptBuilder = NewPromptBuilder(PromptBuilderConfig{
		InitialPrompt:       cfg.ProtoConfig.InitialPrompt,
		CustomTemplate:      cfg.ProtoConfig.ControlPromptTemplate,
		MCPPort:             mcpPort,
		PodKey:              cfg.PodKey,
		GetMaxIterations:    ac.iterCtrl.GetMaxIterations,
		GetCurrentIteration: ac.iterCtrl.GetCurrentIteration,
	})

	// Initialize ControlRunner
	ac.controlRunner = NewControlRunner(ControlRunnerConfig{
		WorkDir:        cfg.PodCtrl.GetWorkDir(),
		AgentType:      cfg.ProtoConfig.ControlAgentType,
		MCPConfigPath:  mcpConfigPath,
		PromptBuilder:  ac.promptBuilder,
		DecisionParser: NewDecisionParser(),
		Logger:         log,
	})

	// Initialize ProgressTracker
	ac.progressTracker = NewProgressTracker(ProgressTrackerConfig{
		WorkDir: cfg.PodCtrl.GetWorkDir(),
		Logger:  log,
	})

	// Initialize StateDetectorCoordinator
	var detector StateDetector
	if cfg.PodCtrl != nil {
		detector = cfg.PodCtrl.GetStateDetector()
	}
	ac.stateCoordinator = NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		Detector:     detector,
		OnWaiting:    ac.OnPodWaiting,
		CheckPeriod:  500 * time.Millisecond,
		Logger:       log,
		AutopilotKey: cfg.AutopilotKey,
	})

	// Start state detection
	ac.stateCoordinator.Start()

	// Set up resume callback now that all components are initialized
	ac.userHandler.SetOnResumeCallback(ac.onResumeFromUserInteraction)

	log.Info("AutopilotController created",
		"autopilot_key", cfg.AutopilotKey,
		"pod_key", cfg.PodKey,
		"max_iterations", cfg.ProtoConfig.MaxIterations)

	return ac
}

// Key returns the AutopilotController's key.
func (ac *AutopilotController) Key() string {
	return ac.key
}

// PodKey returns the associated Pod's key.
func (ac *AutopilotController) PodKey() string {
	return ac.podKey
}

// GetStatus returns a copy of the current status.
func (ac *AutopilotController) GetStatus() Status {
	ac.decisionMu.RLock()
	lastDecision := ac.lastDecision
	lastDecisionMsg := ac.lastDecisionMsg
	ac.decisionMu.RUnlock()

	return Status{
		Phase:            ac.phaseMgr.GetPhase(),
		CurrentIteration: ac.iterCtrl.GetCurrentIteration(),
		MaxIterations:    ac.iterCtrl.GetMaxIterations(),
		PodStatus:        ac.podCtrl.GetAgentStatus(),
		StartedAt:        ac.iterCtrl.GetStartedAt(),
		LastIterationAt:  ac.iterCtrl.GetLastIterationAt(),
		LastDecision:     lastDecision,
		LastDecisionMsg:  lastDecisionMsg,
	}
}

// OnPodWaiting is called when the Pod transitions to waiting state.
// This is the main event-driven entry point triggered by StateDetectorCoordinator.
// Includes deduplication to prevent rapid re-triggering.
func (ac *AutopilotController) OnPodWaiting() {
	ac.log.Debug("OnPodWaiting triggered", "autopilot_key", ac.key)

	// Check trigger deduplication
	if !ac.iterCtrl.CheckTriggerDedup() {
		ac.log.Debug("Skipping iteration - deduplication", "autopilot_key", ac.key)
		return
	}

	// Check if user has taken over
	if ac.userHandler.IsUserTakeover() {
		ac.log.Debug("Skipping iteration - user takeover", "autopilot_key", ac.key)
		return
	}

	// Check if phase allows iteration
	if !ac.phaseMgr.CanProcessIteration() {
		ac.log.Debug("Skipping iteration - phase not ready", "autopilot_key", ac.key, "phase", ac.phaseMgr.GetPhase())
		return
	}

	// Check max iterations (the only hard protection)
	if ac.iterCtrl.HasReachedMaxIterations() {
		ac.phaseMgr.SetPhase(PhaseMaxIterations)
		ac.log.Info("Max iterations reached", "autopilot_key", ac.key)
		if ac.reporter != nil {
			ac.reporter.ReportAutopilotTerminated(&runnerv1.AutopilotTerminatedEvent{
				AutopilotKey: ac.key,
				Reason:       "max_iterations",
			})
		}
		return
	}

	// Increment iteration
	iteration, ok := ac.iterCtrl.IncrementIteration()
	if !ok {
		// Max iterations reached during increment
		ac.phaseMgr.SetPhase(PhaseMaxIterations)
		ac.log.Info("Max iterations reached", "autopilot_key", ac.key)
		if ac.reporter != nil {
			ac.reporter.ReportAutopilotTerminated(&runnerv1.AutopilotTerminatedEvent{
				AutopilotKey: ac.key,
				Reason:       "max_iterations",
			})
		}
		return
	}

	// Report iteration started
	ac.iterCtrl.ReportIterationEvent(iteration, "started", "", nil)

	// Run single decision in a goroutine
	go ac.runSingleDecision(iteration)
}

// sendInitialPrompt starts the first iteration when Pod is waiting.
// This launches the control process which will use MCP tools to interact with Pod.
func (ac *AutopilotController) sendInitialPrompt() {
	ac.log.Info("Starting initial iteration", "autopilot_key", ac.key)

	// Update trigger time to prevent OnPodWaiting from double-triggering
	ac.iterCtrl.UpdateTriggerTime()

	// Set initial iteration
	iteration := ac.iterCtrl.SetInitialIteration()

	// Report iteration started
	ac.iterCtrl.ReportIterationEvent(iteration, "started", "", nil)

	// Run the control process
	go ac.runSingleDecision(iteration)
}

// =============================================================================
// Test Helper Methods (unexported)
// These methods provide access to internal components for testing purposes.
// They delegate to the appropriate component and are not part of the public API.
// =============================================================================

func (ac *AutopilotController) extractSessionID(output string) {
	if sessionID := ExtractSessionID(output); sessionID != "" {
		ac.controlRunner.SetSessionID(sessionID)
	}
}

func (ac *AutopilotController) getSessionID() string {
	return ac.controlRunner.GetSessionID()
}

func (ac *AutopilotController) buildInitialPrompt() string {
	return ac.promptBuilder.BuildInitialPrompt()
}

func (ac *AutopilotController) buildResumePrompt(iteration int) string {
	return ac.promptBuilder.BuildResumePrompt(iteration)
}

func (ac *AutopilotController) parseDecision(output string) *ControlDecision {
	return NewDecisionParser().ParseDecision(output)
}

func (ac *AutopilotController) setPhaseForTest(phase Phase) {
	ac.phaseMgr.SetPhaseWithoutReport(phase)
}

func (ac *AutopilotController) setIterationForTest(iteration int) {
	ac.iterCtrl.mu.Lock()
	ac.iterCtrl.currentIter = iteration
	ac.iterCtrl.mu.Unlock()
}

// GetProgressSummary returns a human-readable summary of task progress.
func (ac *AutopilotController) GetProgressSummary() string {
	if ac.progressTracker == nil {
		return "No progress tracking available"
	}
	return ac.progressTracker.GenerateSummary()
}

// IsStuck checks if no progress has been made for the specified duration.
func (ac *AutopilotController) IsStuck(threshold time.Duration) bool {
	if ac.progressTracker == nil {
		return false
	}
	return ac.progressTracker.IsStuck(threshold)
}

// GetChangedFiles returns files changed during the autopilot session.
func (ac *AutopilotController) GetChangedFiles() []string {
	if ac.progressTracker == nil {
		return nil
	}
	startedAt := ac.iterCtrl.GetStartedAt()
	return ac.progressTracker.GetChangedFilesSince(startedAt)
}

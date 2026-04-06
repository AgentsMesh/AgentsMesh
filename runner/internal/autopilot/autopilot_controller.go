// Package autopilot implements the AutopilotController for supervised Pod automation.
// AutopilotController orchestrates Pod execution by detecting when the controlled pod
// is waiting for input and automatically providing the next instruction.
package autopilot

import (
	"context"
	"log/slog"
	"sync"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// AutopilotController is a supervised automation controller that orchestrates
// a Pod to complete tasks autonomously.
//
// This is a thin coordinator that delegates to specialized components:
// - PhaseManager: lifecycle phase transitions
// - IterationController: iteration counting and max iteration protection
// - UserInteractionHandler: takeover/handback/approve handling
// - StateDetectorCoordinator: pod state change detection (event-driven via PodIO)
// - AcpControlProcess: control agent execution (long-lived ACP session)
// - ProgressTracker: file/git change tracking
type AutopilotController struct {
	key    string
	podKey string
	config *runnerv1.AutopilotConfig

	// Pod controller
	podCtrl TargetPodController

	// MCP port for control process to connect to
	mcpPort int

	// MCP config file path for control process
	mcpConfigPath string

	// Component delegates (SRP compliance)
	phaseMgr         *PhaseManager
	iterCtrl         *IterationController
	userHandler      *UserInteractionHandler
	stateCoordinator *StateDetectorCoordinator
	controlRunner    ControlProcess
	promptBuilder    *PromptBuilder
	progressTracker  *ProgressTracker

	// Status mutex for LastDecision fields
	decisionMu      sync.RWMutex
	lastDecision    string
	lastDecisionMsg string

	// Lifecycle
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup // Tracks running goroutines for clean shutdown
	wgMu     sync.Mutex     // Protects wg.Add to ensure atomicity with stopped check
	stopped  bool           // Set to true when Stop() is called, guarded by wgMu
	stopOnce sync.Once      // Ensures cleanup runs only once

	// Event reporting
	reporter EventReporter

	// Logger
	log *slog.Logger
}

// Config contains configuration for creating an AutopilotController.
type Config struct {
	AutopilotKey   string
	PodKey         string
	ProtoConfig    *runnerv1.AutopilotConfig
	PodCtrl        TargetPodController
	Reporter       EventReporter
	MCPPort        int            // MCP HTTP Server port for control process
	ControlProcess ControlProcess // Optional: inject custom ControlProcess (for testing)
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
		Prompt:              cfg.ProtoConfig.Prompt,
		CustomTemplate:      cfg.ProtoConfig.ControlPromptTemplate,
		MCPPort:             mcpPort,
		PodKey:              cfg.PodKey,
		GetMaxIterations:    ac.iterCtrl.GetMaxIterations,
		GetCurrentIteration: ac.iterCtrl.GetCurrentIteration,
	})

	// Initialize ControlProcess (use injected or default ACP)
	if cfg.ControlProcess != nil {
		ac.controlRunner = cfg.ControlProcess
	} else {
		ac.controlRunner = NewAcpControlProcess(AcpControlProcessConfig{
			Command:        cfg.ProtoConfig.ControlAgentSlug,
			WorkDir:        cfg.PodCtrl.GetWorkDir(),
			MCPConfigPath:  mcpConfigPath,
			PromptBuilder:  ac.promptBuilder,
			DecisionParser: NewDecisionParser(),
			Logger:         log,
		})
	}

	// Initialize ProgressTracker
	ac.progressTracker = NewProgressTracker(ProgressTrackerConfig{
		WorkDir: cfg.PodCtrl.GetWorkDir(),
		Logger:  log,
	})

	// Initialize StateDetectorCoordinator (event-driven, mode-agnostic)
	ac.stateCoordinator = NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		PodCtrl:      cfg.PodCtrl,
		OnWaiting:    ac.OnPodWaiting,
		Logger:       log,
		AutopilotKey: cfg.AutopilotKey,
	})

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

// Note: OnPodWaiting, sendPrompt are in autopilot_controller_logic.go
// Note: Test helpers and progress methods are in autopilot_controller_helpers.go

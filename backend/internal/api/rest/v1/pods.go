package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// PodHandler handles pod-related requests.
// Pod creation is delegated to PodOrchestrator (service layer).
// This handler remains responsible for CRUD and HTTP protocol adaptation.
type PodHandler struct {
	podService     PodServiceForHandler            // Pod CRUD operations (ListPods, GetPod, TerminatePod, etc.)
	runnerService  *runner.Service                 // Runner management
	runnerConnMgr  *runner.RunnerConnectionManager // Runner gRPC connections
	podCoordinator *runner.PodCoordinator          // Pod coordination (TerminatePod, terminal routing)
	orchestrator         *agentpod.PodOrchestrator       // Unified Pod creation logic
	eventBus             *eventbus.EventBus              // Event bus for real-time events
	commandSender        runner.RunnerCommandSender      // Unified command sender (PTY + ACP)
}

// PodHandlerOption is a functional option for configuring PodHandler
type PodHandlerOption func(*PodHandler)

// WithRunnerConnectionManager sets the runner connection manager
func WithRunnerConnectionManager(cm *runner.RunnerConnectionManager) PodHandlerOption {
	return func(h *PodHandler) {
		h.runnerConnMgr = cm
	}
}

// WithPodCoordinator sets the pod coordinator
func WithPodCoordinator(pc *runner.PodCoordinator) PodHandlerOption {
	return func(h *PodHandler) {
		h.podCoordinator = pc
	}
}

// WithPodService sets the pod service (for testing with mock implementations)
func WithPodService(ps PodServiceForHandler) PodHandlerOption {
	return func(h *PodHandler) {
		h.podService = ps
	}
}

// WithEventBus sets the event bus for publishing real-time events
func WithEventBus(eb *eventbus.EventBus) PodHandlerOption {
	return func(h *PodHandler) {
		h.eventBus = eb
	}
}

// WithCommandSender sets the unified command sender for PTY and ACP commands
func WithCommandSender(sender runner.RunnerCommandSender) PodHandlerOption {
	return func(h *PodHandler) {
		h.commandSender = sender
	}
}

// NewPodHandler creates a new pod handler with required dependencies and optional configurations.
func NewPodHandler(
	podService *agentpod.PodService,
	runnerService *runner.Service,
	orchestrator *agentpod.PodOrchestrator,
	opts ...PodHandlerOption,
) *PodHandler {
	h := &PodHandler{
		podService:    podService,
		runnerService: runnerService,
		orchestrator:  orchestrator,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

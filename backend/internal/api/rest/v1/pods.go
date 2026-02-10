package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// PodHandler handles pod-related requests.
// Pod creation is delegated to PodOrchestrator (service layer).
// This handler remains responsible for CRUD, terminal, and HTTP protocol adaptation.
type PodHandler struct {
	podService     PodServiceForHandler            // Pod CRUD operations (ListPods, GetPod, TerminatePod, etc.)
	runnerService  *runner.Service                 // Runner management
	runnerConnMgr  *runner.RunnerConnectionManager // Runner gRPC connections
	podCoordinator *runner.PodCoordinator          // Pod coordination (TerminatePod, terminal routing)
	terminalRouter interface{}                     // *runner.TerminalRouter, optional
	orchestrator   *agentpod.PodOrchestrator       // Unified Pod creation logic
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

// WithTerminalRouter sets the terminal router
func WithTerminalRouter(tr interface{}) PodHandlerOption {
	return func(h *PodHandler) {
		h.terminalRouter = tr
	}
}

// WithPodService sets the pod service (for testing with mock implementations)
func WithPodService(ps PodServiceForHandler) PodHandlerOption {
	return func(h *PodHandler) {
		h.podService = ps
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

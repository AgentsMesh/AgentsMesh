package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	grantservice "github.com/anthropics/agentsmesh/backend/internal/service/grant"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

type PodHandler struct {
	podService     PodServiceForHandler
	runnerService  *runner.Service
	runnerConnMgr  *runner.RunnerConnectionManager
	podCoordinator *runner.PodCoordinator
	orchestrator         *agentpod.PodOrchestrator
	eventBus             *eventbus.EventBus
	commandSender        runner.RunnerCommandSender
	grantService         *grantservice.Service
}

type PodHandlerOption func(*PodHandler)

func WithRunnerConnectionManager(cm *runner.RunnerConnectionManager) PodHandlerOption {
	return func(h *PodHandler) {
		h.runnerConnMgr = cm
	}
}

func WithPodCoordinator(pc *runner.PodCoordinator) PodHandlerOption {
	return func(h *PodHandler) {
		h.podCoordinator = pc
	}
}

func WithPodService(ps PodServiceForHandler) PodHandlerOption {
	return func(h *PodHandler) {
		h.podService = ps
	}
}

func WithEventBus(eb *eventbus.EventBus) PodHandlerOption {
	return func(h *PodHandler) {
		h.eventBus = eb
	}
}

func WithCommandSender(sender runner.RunnerCommandSender) PodHandlerOption {
	return func(h *PodHandler) {
		h.commandSender = sender
	}
}

func WithGrantServiceForPod(gs *grantservice.Service) PodHandlerOption {
	return func(h *PodHandler) {
		h.grantService = gs
	}
}

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

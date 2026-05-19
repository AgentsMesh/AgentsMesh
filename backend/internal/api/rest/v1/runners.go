package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	grantservice "github.com/anthropics/agentsmesh/backend/internal/service/grant"
	runner "github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerlogservice "github.com/anthropics/agentsmesh/backend/internal/service/runnerlog"
)

type RunnerHandler struct {
	runnerService        *runner.Service
	podService           *agentpod.PodService
	sandboxQueryService  *runner.SandboxQueryService
	podCoordinator       *runner.PodCoordinator
	versionChecker       *runner.VersionChecker
	upgradeCommandSender runner.UpgradeCommandSender
	logUploadSender      runner.LogUploadCommandSender
	logUploadService     *runnerlogservice.Service
	grantService         *grantservice.Service
}

func NewRunnerHandler(runnerService *runner.Service, opts ...RunnerHandlerOption) *RunnerHandler {
	h := &RunnerHandler{
		runnerService: runnerService,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

type RunnerHandlerOption func(*RunnerHandler)

func WithPodServiceForRunner(ps *agentpod.PodService) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.podService = ps
	}
}

func WithSandboxQueryService(sqs *runner.SandboxQueryService) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.sandboxQueryService = sqs
	}
}

func WithPodCoordinatorForRunner(pc *runner.PodCoordinator) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.podCoordinator = pc
	}
}

func WithVersionChecker(vc *runner.VersionChecker) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.versionChecker = vc
	}
}

func WithUpgradeCommandSender(ucs runner.UpgradeCommandSender) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.upgradeCommandSender = ucs
	}
}

func WithLogUploadSender(sender runner.LogUploadCommandSender) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.logUploadSender = sender
	}
}

func WithLogUploadService(svc *runnerlogservice.Service) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.logUploadService = svc
	}
}

func WithGrantServiceForRunner(gs *grantservice.Service) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.grantService = gs
	}
}

type UpdateRunnerRequest struct {
	Description       *string  `json:"description"`
	MaxConcurrentPods *int     `json:"max_concurrent_pods"`
	IsEnabled         *bool    `json:"is_enabled"`
	Visibility        *string  `json:"visibility"`
	Tags              []string `json:"tags"`
}

type ListRunnerPodsRequest struct {
	Status string `form:"status"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

type QuerySandboxesRequest struct {
	PodKeys []string `json:"pod_keys" binding:"required,min=1,max=100"`
}

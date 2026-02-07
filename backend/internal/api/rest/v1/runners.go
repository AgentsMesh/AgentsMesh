package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	runner "github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// RunnerHandler handles runner-related requests
type RunnerHandler struct {
	runnerService       *runner.Service
	podService          *agentpod.PodService
	sandboxQueryService *runner.SandboxQueryService
	sandboxQuerySender  runner.SandboxQuerySender
	podCoordinator      *runner.PodCoordinator
}

// NewRunnerHandler creates a new runner handler
func NewRunnerHandler(runnerService *runner.Service, opts ...RunnerHandlerOption) *RunnerHandler {
	h := &RunnerHandler{
		runnerService: runnerService,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// RunnerHandlerOption is a functional option for configuring RunnerHandler
type RunnerHandlerOption func(*RunnerHandler)

// WithPodServiceForRunner sets the pod service for runner handler
func WithPodServiceForRunner(ps *agentpod.PodService) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.podService = ps
	}
}

// WithSandboxQueryService sets the sandbox query service for runner handler
func WithSandboxQueryService(sqs *runner.SandboxQueryService) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.sandboxQueryService = sqs
	}
}

// WithSandboxQuerySender sets the sandbox query sender for runner handler
func WithSandboxQuerySender(sqs runner.SandboxQuerySender) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.sandboxQuerySender = sqs
	}
}

// WithPodCoordinatorForRunner sets the pod coordinator for runner handler
func WithPodCoordinatorForRunner(pc *runner.PodCoordinator) RunnerHandlerOption {
	return func(h *RunnerHandler) {
		h.podCoordinator = pc
	}
}

// UpdateRunnerRequest represents runner update request
type UpdateRunnerRequest struct {
	Description       *string `json:"description"`
	MaxConcurrentPods *int    `json:"max_concurrent_pods"`
	IsEnabled         *bool   `json:"is_enabled"`
}

// ListRunnerPodsRequest represents request for listing runner pods
type ListRunnerPodsRequest struct {
	Status string `form:"status"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

// QuerySandboxesRequest represents request for querying sandbox status
type QuerySandboxesRequest struct {
	PodKeys []string `json:"pod_keys" binding:"required,min=1,max=100"`
}

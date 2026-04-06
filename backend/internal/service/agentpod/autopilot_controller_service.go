package agentpod

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

var (
	ErrAutopilotControllerNotFound = errors.New("autopilot pod not found")
)

// AutopilotCommandSender defines the interface for sending autopilot commands to runners.
type AutopilotCommandSender interface {
	SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error
}

// AutopilotControllerService handles AutopilotController operations.
type AutopilotControllerService struct {
	repo          agentpod.AutopilotRepository
	commandSender AutopilotCommandSender
}

// NewAutopilotControllerService creates a new AutopilotController service
func NewAutopilotControllerService(repo agentpod.AutopilotRepository) *AutopilotControllerService {
	return &AutopilotControllerService{repo: repo}
}

// SetCommandSender injects the command sender for gRPC communication with Runners.
func (s *AutopilotControllerService) SetCommandSender(sender AutopilotCommandSender) {
	s.commandSender = sender
}

// ========== CreateAndStart (encapsulated Autopilot creation) ==========

// CreateAndStartRequest contains all parameters for creating and starting an AutopilotController.
type CreateAndStartRequest struct {
	OrganizationID int64
	Pod            *agentpod.Pod
	Prompt         string

	MaxIterations         int32
	IterationTimeoutSec   int32
	NoProgressThreshold   int32
	SameErrorThreshold    int32
	ApprovalTimeoutMin    int32
	ControlAgentSlug      string
	ControlPromptTemplate string
	MCPConfigJSON         string
	KeyPrefix             string
}

// CreateAndStart creates an AutopilotController record, applies domain defaults,
// and sends the creation command to the Runner via gRPC.
func (s *AutopilotControllerService) CreateAndStart(ctx context.Context, req *CreateAndStartRequest) (*agentpod.AutopilotController, error) {
	if req.Pod == nil {
		return nil, fmt.Errorf("target pod is required")
	}

	prefix := req.KeyPrefix
	if prefix == "" {
		prefix = "autopilot"
	}
	autopilotKey := fmt.Sprintf("%s-%s-%d", prefix, req.Pod.PodKey, time.Now().UnixNano())

	maxIter, iterTimeout, noProg, sameErr, approvalTimeout := agentpod.ApplyDefaults(
		req.MaxIterations, req.IterationTimeoutSec, req.NoProgressThreshold,
		req.SameErrorThreshold, req.ApprovalTimeoutMin,
	)

	controller := &agentpod.AutopilotController{
		OrganizationID:         req.OrganizationID,
		AutopilotControllerKey: autopilotKey,
		PodKey:                 req.Pod.PodKey,
		PodID:                  req.Pod.ID,
		RunnerID:               req.Pod.RunnerID,
		Prompt:                 req.Prompt,
		Phase:                  agentpod.AutopilotPhaseInitializing,
		MaxIterations:          maxIter,
		IterationTimeoutSec:    iterTimeout,
		NoProgressThreshold:    noProg,
		SameErrorThreshold:     sameErr,
		ApprovalTimeoutMin:     approvalTimeout,
		CircuitBreakerState:    agentpod.CircuitBreakerClosed,
	}

	if req.ControlAgentSlug != "" {
		controller.ControlAgentSlug = &req.ControlAgentSlug
	}
	if req.ControlPromptTemplate != "" {
		controller.ControlPromptTemplate = &req.ControlPromptTemplate
	}
	if req.MCPConfigJSON != "" {
		controller.MCPConfigJSON = &req.MCPConfigJSON
	}

	if err := s.repo.Create(ctx, controller); err != nil {
		slog.Error("failed to create autopilot controller", "autopilot_key", autopilotKey, "pod_key", req.Pod.PodKey, "error", err)
		return nil, fmt.Errorf("failed to create autopilot controller: %w", err)
	}

	slog.Info("autopilot controller created", "autopilot_key", autopilotKey, "pod_key", req.Pod.PodKey, "org_id", req.OrganizationID)

	if s.commandSender != nil {
		cmd := &runnerv1.CreateAutopilotCommand{
			AutopilotKey: autopilotKey,
			PodKey:       req.Pod.PodKey,
			Config: &runnerv1.AutopilotConfig{
				Prompt:                  req.Prompt,
				MaxIterations:           maxIter,
				IterationTimeoutSeconds: iterTimeout,
				NoProgressThreshold:     noProg,
				SameErrorThreshold:      sameErr,
				ApprovalTimeoutMinutes:  approvalTimeout,
				ControlAgentSlug:        req.ControlAgentSlug,
				ControlPromptTemplate:   req.ControlPromptTemplate,
				McpConfigJson:           req.MCPConfigJSON,
			},
		}
		if err := s.commandSender.SendCreateAutopilot(req.Pod.RunnerID, cmd); err != nil {
			slog.Error("failed to send autopilot command to runner",
				"autopilot_key", autopilotKey, "pod_key", req.Pod.PodKey, "runner_id", req.Pod.RunnerID, "error", err)
			return controller, fmt.Errorf("autopilot created in DB but failed to send command to runner: %w", err)
		}
	}

	return controller, nil
}

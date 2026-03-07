package agentpod

import (
	"context"
	"errors"
	"fmt"
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
	InitialPrompt  string

	MaxIterations         int32
	IterationTimeoutSec   int32
	NoProgressThreshold   int32
	SameErrorThreshold    int32
	ApprovalTimeoutMin    int32
	ControlAgentType      string
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
		InitialPrompt:          req.InitialPrompt,
		Phase:                  agentpod.AutopilotPhaseInitializing,
		MaxIterations:          maxIter,
		IterationTimeoutSec:    iterTimeout,
		NoProgressThreshold:    noProg,
		SameErrorThreshold:     sameErr,
		ApprovalTimeoutMin:     approvalTimeout,
		CircuitBreakerState:    agentpod.CircuitBreakerClosed,
	}

	if req.ControlAgentType != "" {
		controller.ControlAgentType = &req.ControlAgentType
	}
	if req.ControlPromptTemplate != "" {
		controller.ControlPromptTemplate = &req.ControlPromptTemplate
	}
	if req.MCPConfigJSON != "" {
		controller.MCPConfigJSON = &req.MCPConfigJSON
	}

	if err := s.repo.Create(ctx, controller); err != nil {
		return nil, fmt.Errorf("failed to create autopilot controller: %w", err)
	}

	if s.commandSender != nil {
		cmd := &runnerv1.CreateAutopilotCommand{
			AutopilotKey: autopilotKey,
			PodKey:       req.Pod.PodKey,
			Config: &runnerv1.AutopilotConfig{
				InitialPrompt:           req.InitialPrompt,
				MaxIterations:           maxIter,
				IterationTimeoutSeconds: iterTimeout,
				NoProgressThreshold:     noProg,
				SameErrorThreshold:      sameErr,
				ApprovalTimeoutMinutes:  approvalTimeout,
				ControlAgentType:        req.ControlAgentType,
				ControlPromptTemplate:   req.ControlPromptTemplate,
				McpConfigJson:           req.MCPConfigJSON,
			},
		}
		if err := s.commandSender.SendCreateAutopilot(req.Pod.RunnerID, cmd); err != nil {
			return controller, fmt.Errorf("autopilot created in DB but failed to send command to runner: %w", err)
		}
	}

	return controller, nil
}

// ========== CRUD Operations ==========

// GetAutopilotController retrieves an AutopilotController by organization ID and key
func (s *AutopilotControllerService) GetAutopilotController(ctx context.Context, orgID int64, autopilotPodKey string) (*agentpod.AutopilotController, error) {
	controller, err := s.repo.GetByOrgAndKey(ctx, orgID, autopilotPodKey)
	if err != nil {
		return nil, err
	}
	if controller == nil {
		return nil, ErrAutopilotControllerNotFound
	}
	return controller, nil
}

// ListAutopilotControllers lists all AutopilotControllers for an organization
func (s *AutopilotControllerService) ListAutopilotControllers(ctx context.Context, orgID int64) ([]*agentpod.AutopilotController, error) {
	return s.repo.ListByOrg(ctx, orgID)
}

// CreateAutopilotController creates a new AutopilotController record.
func (s *AutopilotControllerService) CreateAutopilotController(ctx context.Context, pod *agentpod.AutopilotController) error {
	return s.repo.Create(ctx, pod)
}

// UpdateAutopilotController updates an existing AutopilotController
func (s *AutopilotControllerService) UpdateAutopilotController(ctx context.Context, pod *agentpod.AutopilotController) error {
	return s.repo.Save(ctx, pod)
}

// UpdateAutopilotControllerStatus updates the status fields of an AutopilotController
func (s *AutopilotControllerService) UpdateAutopilotControllerStatus(ctx context.Context, autopilotPodKey string, updates map[string]interface{}) error {
	return s.repo.UpdateStatusByKey(ctx, autopilotPodKey, updates)
}

// GetIterations retrieves all iterations for an AutopilotController
func (s *AutopilotControllerService) GetIterations(ctx context.Context, autopilotPodID int64) ([]*agentpod.AutopilotIteration, error) {
	return s.repo.ListIterations(ctx, autopilotPodID)
}

// CreateIteration creates a new iteration record
func (s *AutopilotControllerService) CreateIteration(ctx context.Context, iteration *agentpod.AutopilotIteration) error {
	return s.repo.CreateIteration(ctx, iteration)
}

// GetAutopilotControllerByKey retrieves an AutopilotController by key only
func (s *AutopilotControllerService) GetAutopilotControllerByKey(ctx context.Context, autopilotPodKey string) (*agentpod.AutopilotController, error) {
	controller, err := s.repo.GetByKey(ctx, autopilotPodKey)
	if err != nil {
		return nil, err
	}
	if controller == nil {
		return nil, ErrAutopilotControllerNotFound
	}
	return controller, nil
}

// GetActiveAutopilotControllerForPod retrieves active AutopilotController for a pod
func (s *AutopilotControllerService) GetActiveAutopilotControllerForPod(ctx context.Context, podKey string) (*agentpod.AutopilotController, error) {
	controller, err := s.repo.GetActiveForPod(ctx, podKey)
	if err != nil {
		return nil, err
	}
	if controller == nil {
		return nil, ErrAutopilotControllerNotFound
	}
	return controller, nil
}

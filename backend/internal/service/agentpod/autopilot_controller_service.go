package agentpod

import (
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"gorm.io/gorm"
)

var (
	ErrAutopilotControllerNotFound = errors.New("autopilot pod not found")
)

// AutopilotCommandSender defines the interface for sending autopilot commands to runners.
// Defined here to avoid circular dependency with service/runner.
type AutopilotCommandSender interface {
	SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error
}

// AutopilotControllerService handles AutopilotController operations.
//
// Responsibilities:
//   - CRUD operations for AutopilotController records
//   - CreateAndStart: full lifecycle of creating a controller record + sending gRPC command to Runner
type AutopilotControllerService struct {
	db            *gorm.DB
	commandSender AutopilotCommandSender
}

// NewAutopilotControllerService creates a new AutopilotController service
func NewAutopilotControllerService(db *gorm.DB) *AutopilotControllerService {
	return &AutopilotControllerService{db: db}
}

// SetCommandSender injects the command sender for gRPC communication with Runners.
// Must be called after PodCoordinator has its command sender configured.
func (s *AutopilotControllerService) SetCommandSender(sender AutopilotCommandSender) {
	s.commandSender = sender
}

// ========== CreateAndStart (encapsulated Autopilot creation) ==========

// CreateAndStartRequest contains all parameters for creating and starting an AutopilotController.
type CreateAndStartRequest struct {
	OrganizationID int64
	Pod            *agentpod.Pod // the validated target Pod (must be active)
	InitialPrompt  string

	// Optional configuration (zero values use domain defaults from agentpod package)
	MaxIterations         int32
	IterationTimeoutSec   int32
	NoProgressThreshold   int32
	SameErrorThreshold    int32
	ApprovalTimeoutMin    int32
	ControlAgentType      string
	ControlPromptTemplate string
	MCPConfigJSON         string

	// KeyPrefix customizes the generated autopilot controller key.
	// Examples: "autopilot" (REST), "loop-daily-review-run3" (Loop).
	// Defaults to "autopilot" if empty.
	KeyPrefix string
}

// CreateAndStart creates an AutopilotController record, applies domain defaults,
// and sends the creation command to the Runner via gRPC.
//
// This is the single entry point for Autopilot creation, used by both:
//   - REST API handler (user-initiated)
//   - LoopOrchestrator (automated Loop runs)
//
// Returns the created controller and any error.
func (s *AutopilotControllerService) CreateAndStart(req *CreateAndStartRequest) (*agentpod.AutopilotController, error) {
	if req.Pod == nil {
		return nil, fmt.Errorf("target pod is required")
	}

	// 1. Generate key
	prefix := req.KeyPrefix
	if prefix == "" {
		prefix = "autopilot"
	}
	autopilotKey := fmt.Sprintf("%s-%s-%d", prefix, req.Pod.PodKey, time.Now().UnixNano())

	// 2. Apply domain defaults for zero-valued config fields
	maxIter, iterTimeout, noProg, sameErr, approvalTimeout := agentpod.ApplyDefaults(
		req.MaxIterations,
		req.IterationTimeoutSec,
		req.NoProgressThreshold,
		req.SameErrorThreshold,
		req.ApprovalTimeoutMin,
	)

	// 3. Build domain model
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

	// 4. Persist to database
	if err := s.db.Create(controller).Error; err != nil {
		return nil, fmt.Errorf("failed to create autopilot controller: %w", err)
	}

	// 5. Send command to Runner
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

// GetAutopilotController retrieves a AutopilotController by organization ID and key
func (s *AutopilotControllerService) GetAutopilotController(orgID int64, autopilotPodKey string) (*agentpod.AutopilotController, error) {
	var pod agentpod.AutopilotController
	err := s.db.Where("organization_id = ? AND autopilot_controller_key = ?", orgID, autopilotPodKey).First(&pod).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAutopilotControllerNotFound
		}
		return nil, err
	}
	return &pod, nil
}

// ListAutopilotControllers lists all AutopilotControllers for an organization
func (s *AutopilotControllerService) ListAutopilotControllers(orgID int64) ([]*agentpod.AutopilotController, error) {
	var pods []*agentpod.AutopilotController
	err := s.db.Where("organization_id = ?", orgID).Order("created_at DESC").Find(&pods).Error
	if err != nil {
		return nil, err
	}
	return pods, nil
}

// CreateAutopilotController creates a new AutopilotController record.
// Prefer CreateAndStart for the full lifecycle (record + gRPC command).
func (s *AutopilotControllerService) CreateAutopilotController(pod *agentpod.AutopilotController) error {
	return s.db.Create(pod).Error
}

// UpdateAutopilotController updates an existing AutopilotController
func (s *AutopilotControllerService) UpdateAutopilotController(pod *agentpod.AutopilotController) error {
	return s.db.Save(pod).Error
}

// UpdateAutopilotControllerStatus updates the status fields of a AutopilotController
func (s *AutopilotControllerService) UpdateAutopilotControllerStatus(autopilotPodKey string, updates map[string]interface{}) error {
	return s.db.Model(&agentpod.AutopilotController{}).
		Where("autopilot_controller_key = ?", autopilotPodKey).
		Updates(updates).Error
}

// GetIterations retrieves all iterations for a AutopilotController
func (s *AutopilotControllerService) GetIterations(autopilotPodID int64) ([]*agentpod.AutopilotIteration, error) {
	var iterations []*agentpod.AutopilotIteration
	err := s.db.Where("autopilot_controller_id = ?", autopilotPodID).Order("iteration ASC").Find(&iterations).Error
	if err != nil {
		return nil, err
	}
	return iterations, nil
}

// CreateIteration creates a new iteration record
func (s *AutopilotControllerService) CreateIteration(iteration *agentpod.AutopilotIteration) error {
	return s.db.Create(iteration).Error
}

// GetAutopilotControllerByKey retrieves a AutopilotController by key only (for internal use)
func (s *AutopilotControllerService) GetAutopilotControllerByKey(autopilotPodKey string) (*agentpod.AutopilotController, error) {
	var pod agentpod.AutopilotController
	err := s.db.Where("autopilot_controller_key = ?", autopilotPodKey).First(&pod).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAutopilotControllerNotFound
		}
		return nil, err
	}
	return &pod, nil
}

// GetActiveAutopilotControllerForPod retrieves active AutopilotController for a pod
func (s *AutopilotControllerService) GetActiveAutopilotControllerForPod(podKey string) (*agentpod.AutopilotController, error) {
	var pod agentpod.AutopilotController
	err := s.db.Where("pod_key = ? AND phase NOT IN ?",
		podKey,
		agentpod.TerminalPhases(),
	).First(&pod).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAutopilotControllerNotFound
		}
		return nil, err
	}
	return &pod, nil
}

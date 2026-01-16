package agentpod

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

var (
	ErrPodNotFound       = errors.New("pod not found")
	ErrNoAvailableRunner = errors.New("no available runner")
	ErrPodTerminated     = errors.New("pod already terminated")
	ErrRunnerNotFound    = errors.New("runner not found")
	ErrRunnerOffline     = errors.New("runner is offline")
)

// PodService handles pod operations
type PodService struct {
	db             *gorm.DB
	eventPublisher EventPublisher
}

// SetEventPublisher sets the event publisher for the service
func (s *PodService) SetEventPublisher(publisher EventPublisher) {
	s.eventPublisher = publisher
}

// NewPodService creates a new pod service
func NewPodService(db *gorm.DB) *PodService {
	return &PodService{db: db}
}

// CreatePodRequest represents a pod creation request
type CreatePodRequest struct {
	OrganizationID    int64
	RunnerID          int64
	AgentTypeID       *int64
	CustomAgentTypeID *int64
	RepositoryID      *int64
	TicketID          *int64
	CreatedByID       int64
	InitialPrompt     string
	BranchName        *string
	Model             string
	PermissionMode    string
	SkipPermissions   bool
	ThinkLevel        string
	PreparationConfig *agentpod.PreparationConfig
	EnvVars           map[string]string
}

// CreatePod creates a new pod
func (s *PodService) CreatePod(ctx context.Context, req *CreatePodRequest) (*agentpod.Pod, error) {
	keyBytes := make([]byte, 4)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, err
	}
	randomSuffix := hex.EncodeToString(keyBytes)

	ticketPart := "standalone"
	if req.TicketID != nil {
		ticketPart = fmt.Sprintf("%d", *req.TicketID)
	}
	podKey := fmt.Sprintf("%d-%s-%s", req.CreatedByID, ticketPart, randomSuffix)

	model := req.Model
	if model == "" {
		model = "opus"
	}
	permissionMode := req.PermissionMode
	if permissionMode == "" {
		permissionMode = agentpod.PermissionModePlan
	}
	thinkLevel := req.ThinkLevel
	if thinkLevel == "" {
		thinkLevel = agentpod.ThinkLevelUltrathink
	}

	pod := &agentpod.Pod{
		OrganizationID:    req.OrganizationID,
		PodKey:            podKey,
		RunnerID:          req.RunnerID,
		AgentTypeID:       req.AgentTypeID,
		CustomAgentTypeID: req.CustomAgentTypeID,
		RepositoryID:      req.RepositoryID,
		TicketID:          req.TicketID,
		CreatedByID:       req.CreatedByID,
		Status:            agentpod.PodStatusInitializing,
		AgentStatus:       agentpod.AgentStatusUnknown,
		InitialPrompt:     req.InitialPrompt,
		BranchName:        req.BranchName,
		Model:             &model,
		PermissionMode:    &permissionMode,
		ThinkLevel:        &thinkLevel,
	}

	if err := s.db.WithContext(ctx).Create(pod).Error; err != nil {
		return nil, err
	}

	s.db.WithContext(ctx).Exec("UPDATE runners SET current_pods = current_pods + 1 WHERE id = ?", req.RunnerID)

	return pod, nil
}

// CreatePodForTicket creates a pod with ticket context
func (s *PodService) CreatePodForTicket(ctx context.Context, req *CreatePodRequest) (*agentpod.Pod, error) {
	if req.TicketID == nil {
		return nil, errors.New("ticket_id is required")
	}

	var t ticket.Ticket
	if err := s.db.WithContext(ctx).First(&t, *req.TicketID).Error; err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	if req.InitialPrompt == "" {
		req.InitialPrompt = BuildTicketPrompt(&t)
	}

	return s.CreatePod(ctx, req)
}

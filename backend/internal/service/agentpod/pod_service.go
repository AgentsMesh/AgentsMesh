package agentpod

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/agentmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentmesh/backend/internal/domain/ticket"
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
// This is called during initialization to inject the EventBus publisher
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

	// Enhanced fields (from Mainline)
	Model             string                     // opus/sonnet/haiku
	PermissionMode    string                     // plan/default/bypassPermissions
	SkipPermissions   bool                       // Whether to skip permission checks
	ThinkLevel        string                     // ultrathink/megathink
	PreparationConfig *agentpod.PreparationConfig // Preparation script config
	EnvVars           map[string]string           // Environment variables (AI credentials)
}

// CreatePod creates a new pod
func (s *PodService) CreatePod(ctx context.Context, req *CreatePodRequest) (*agentpod.Pod, error) {
	// Generate pod key with user and ticket context
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

	// Set default values
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

	// Increment runner pod count
	s.db.WithContext(ctx).Exec("UPDATE runners SET current_pods = current_pods + 1 WHERE id = ?", req.RunnerID)

	return pod, nil
}

// CreatePodForTicket creates a pod with ticket context
func (s *PodService) CreatePodForTicket(ctx context.Context, req *CreatePodRequest) (*agentpod.Pod, error) {
	if req.TicketID == nil {
		return nil, errors.New("ticket_id is required")
	}

	// Get ticket for identifier and context
	var t ticket.Ticket
	if err := s.db.WithContext(ctx).First(&t, *req.TicketID).Error; err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	// Build prompt with ticket context if not provided
	if req.InitialPrompt == "" {
		req.InitialPrompt = BuildTicketPrompt(&t)
	}

	return s.CreatePod(ctx, req)
}

// BuildTicketPrompt builds an initial prompt from ticket context
func BuildTicketPrompt(t *ticket.Ticket) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Working on ticket: %s", t.Identifier))
	parts = append(parts, fmt.Sprintf("Title: %s", t.Title))
	if t.Description != nil && *t.Description != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", *t.Description))
	}
	return strings.Join(parts, "\n")
}

// BuildAgentCommand builds the agent startup command (e.g., claude command)
func BuildAgentCommand(model, permissionMode string, skipPermissions bool) string {
	cmdParts := []string{"claude"}
	if skipPermissions {
		cmdParts = append(cmdParts, "--dangerously-skip-permissions")
	}
	if permissionMode != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--permission-mode %s", permissionMode))
	}
	if model != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--model %s", model))
	}
	return strings.Join(cmdParts, " ")
}

// BuildInitialPrompt builds the initial prompt with think level appended
func BuildInitialPrompt(prompt, thinkLevel string) string {
	if thinkLevel != "" && thinkLevel != agentpod.ThinkLevelNone {
		return fmt.Sprintf("%s\n\n%s", prompt, thinkLevel)
	}
	return prompt
}

// GetCreatePodCommand returns the command to send to runner
func (s *PodService) GetCreatePodCommand(ctx context.Context, pod *agentpod.Pod, req *CreatePodRequest) (*agentpod.CreatePodCommand, error) {
	model := "opus"
	if pod.Model != nil {
		model = *pod.Model
	}
	permissionMode := agentpod.PermissionModePlan
	if pod.PermissionMode != nil {
		permissionMode = *pod.PermissionMode
	}
	thinkLevel := agentpod.ThinkLevelUltrathink
	if pod.ThinkLevel != nil {
		thinkLevel = *pod.ThinkLevel
	}

	// Build command
	initialCommand := BuildAgentCommand(model, permissionMode, req.SkipPermissions)

	// Build prompt
	var formattedPrompt string
	if pod.InitialPrompt != "" {
		formattedPrompt = BuildInitialPrompt(pod.InitialPrompt, thinkLevel)
	}

	// Get ticket identifier for worktree
	var ticketIdentifier string
	if pod.TicketID != nil {
		var t ticket.Ticket
		if err := s.db.WithContext(ctx).First(&t, *pod.TicketID).Error; err == nil {
			ticketIdentifier = t.Identifier
		}
	}

	// Extract worktree suffix from pod key
	parts := strings.Split(pod.PodKey, "-")
	worktreeSuffix := parts[len(parts)-1]

	return &agentpod.CreatePodCommand{
		PodKey:            pod.PodKey,
		InitialCommand:    initialCommand,
		InitialPrompt:     formattedPrompt,
		PermissionMode:    permissionMode,
		TicketIdentifier:  ticketIdentifier,
		WorktreeSuffix:    worktreeSuffix,
		EnvVars:           req.EnvVars,
		PreparationConfig: req.PreparationConfig,
	}, nil
}

// GetPod returns a pod by key
func (s *PodService) GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	var pod agentpod.Pod
	if err := s.db.WithContext(ctx).
		Preload("Runner").
		Preload("AgentType").
		Where("pod_key = ?", podKey).
		First(&pod).Error; err != nil {
		return nil, ErrPodNotFound
	}
	return &pod, nil
}

// GetPodByID returns a pod by ID
func (s *PodService) GetPodByID(ctx context.Context, podID int64) (*agentpod.Pod, error) {
	var pod agentpod.Pod
	if err := s.db.WithContext(ctx).
		Preload("Runner").
		Preload("AgentType").
		First(&pod, podID).Error; err != nil {
		return nil, ErrPodNotFound
	}
	return &pod, nil
}

// GetPodByKey returns a pod by key (implements middleware.PodService)
// This is used by PodAuthMiddleware to lookup pods by X-Pod-Key header
func (s *PodService) GetPodByKey(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	return s.GetPod(ctx, podKey)
}

// GetPodInfo returns pod info for binding policy evaluation
// This implements the binding.PodQuerier interface
func (s *PodService) GetPodInfo(ctx context.Context, podKey string) (map[string]interface{}, error) {
	pod, err := s.GetPod(ctx, podKey)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"user_id":         pod.CreatedByID,
		"organization_id": pod.OrganizationID,
		"ticket_id":       pod.TicketID,
		"status":          pod.Status,
	}, nil
}

// GetPodOrganizationAndCreator returns the organization ID and creator ID for a pod
// This implements the runner.PodInfoGetter interface for terminal notifications
func (s *PodService) GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error) {
	var pod agentpod.Pod
	if err := s.db.WithContext(ctx).
		Select("organization_id", "created_by_id").
		Where("pod_key = ?", podKey).
		First(&pod).Error; err != nil {
		return 0, 0, ErrPodNotFound
	}
	return pod.OrganizationID, pod.CreatedByID, nil
}

// ListPods returns pods for an organization
// Note: teamID parameter is deprecated and ignored - all pods visible to org members
func (s *PodService) ListPods(ctx context.Context, orgID int64, _ *int64, status string, limit, offset int) ([]*agentpod.Pod, int64, error) {
	query := s.db.WithContext(ctx).Model(&agentpod.Pod{}).Where("organization_id = ?", orgID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var pods []*agentpod.Pod
	if err := query.
		Preload("Runner").
		Preload("AgentType").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&pods).Error; err != nil {
		return nil, 0, err
	}

	return pods, total, nil
}

// ListActivePods returns active pods for a runner
func (s *PodService) ListActivePods(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	if err := s.db.WithContext(ctx).
		Where("runner_id = ? AND status IN ?", runnerID, []string{
			agentpod.PodStatusInitializing,
			agentpod.PodStatusRunning,
			agentpod.PodStatusPaused,
			agentpod.PodStatusDisconnected,
		}).
		Find(&pods).Error; err != nil {
		return nil, err
	}
	return pods, nil
}

// UpdatePodStatus updates pod status
func (s *PodService) UpdatePodStatus(ctx context.Context, podKey, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == agentpod.PodStatusRunning {
		now := time.Now()
		updates["started_at"] = now
	} else if status == agentpod.PodStatusTerminated || status == agentpod.PodStatusOrphaned {
		now := time.Now()
		updates["finished_at"] = now
	}

	result := s.db.WithContext(ctx).Model(&agentpod.Pod{}).Where("pod_key = ?", podKey).Updates(updates)
	if result.RowsAffected == 0 {
		return ErrPodNotFound
	}

	// If terminated, decrement runner pod count
	if status == agentpod.PodStatusTerminated || status == agentpod.PodStatusOrphaned {
		var pod agentpod.Pod
		s.db.WithContext(ctx).Where("pod_key = ?", podKey).First(&pod)
		s.db.WithContext(ctx).Exec("UPDATE runners SET current_pods = GREATEST(current_pods - 1, 0) WHERE id = ?", pod.RunnerID)
	}

	return nil
}

// UpdateAgentStatus updates agent status
func (s *PodService) UpdateAgentStatus(ctx context.Context, podKey, agentStatus string, agentPID *int) error {
	now := time.Now()
	updates := map[string]interface{}{
		"agent_status":  agentStatus,
		"last_activity": now,
	}
	if agentPID != nil {
		updates["agent_pid"] = *agentPID
	}
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Updates(updates).Error
}

// UpdatePodPTY updates pod PTY PID
func (s *PodService) UpdatePodPTY(ctx context.Context, podKey string, ptyPID int) error {
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Update("pty_pid", ptyPID).Error
}

// UpdateWorktreePath updates pod worktree path and branch
func (s *PodService) UpdateWorktreePath(ctx context.Context, podKey, worktreePath, branchName string) error {
	updates := map[string]interface{}{
		"worktree_path": worktreePath,
	}
	if branchName != "" {
		updates["branch_name"] = branchName
	}
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Updates(updates).Error
}

// HandlePodCreated handles the pod_created event from runner
func (s *PodService) HandlePodCreated(ctx context.Context, podKey string, ptyPID int, worktreePath, branchName string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"pty_pid":       ptyPID,
		"status":        agentpod.PodStatusRunning,
		"started_at":    now,
		"last_activity": now,
	}
	if worktreePath != "" {
		updates["worktree_path"] = worktreePath
	}
	if branchName != "" {
		updates["branch_name"] = branchName
	}
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Updates(updates).Error
}

// HandlePodTerminated handles the pod_terminated event from runner
func (s *PodService) HandlePodTerminated(ctx context.Context, podKey string, exitCode *int) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Updates(map[string]interface{}{
			"status":      agentpod.PodStatusTerminated,
			"finished_at": now,
			"pty_pid":     nil,
		}).Error
}

// TerminatePod terminates a pod
func (s *PodService) TerminatePod(ctx context.Context, podKey string) error {
	pod, err := s.GetPod(ctx, podKey)
	if err != nil {
		return err
	}

	if !pod.IsActive() {
		return ErrPodTerminated
	}

	previousStatus := pod.Status
	if err := s.UpdatePodStatus(ctx, podKey, agentpod.PodStatusTerminated); err != nil {
		return err
	}

	// Publish pod:terminated event (Information Expert: Service knows when state changes)
	if s.eventPublisher != nil {
		s.eventPublisher.PublishPodEvent(
			ctx,
			PodEventTerminated,
			pod.OrganizationID,
			podKey,
			agentpod.PodStatusTerminated,
			previousStatus,
			"",
		)
	}

	return nil
}

// MarkDisconnected marks a pod as disconnected (user closed browser)
func (s *PodService) MarkDisconnected(ctx context.Context, podKey string) error {
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", podKey, agentpod.PodStatusRunning).
		Update("status", agentpod.PodStatusDisconnected).Error
}

// MarkReconnected marks a pod as running again (user reconnected)
func (s *PodService) MarkReconnected(ctx context.Context, podKey string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", podKey, agentpod.PodStatusDisconnected).
		Updates(map[string]interface{}{
			"status":        agentpod.PodStatusRunning,
			"last_activity": now,
		}).Error
}

// RecordActivity records pod activity
func (s *PodService) RecordActivity(ctx context.Context, podKey string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Update("last_activity", now).Error
}

// ReconcilePods marks orphaned pods that are not reported by runner
func (s *PodService) ReconcilePods(ctx context.Context, runnerID int64, reportedPodKeys []string) error {
	// Get active pods for this runner from database
	var dbPods []*agentpod.Pod
	err := s.db.WithContext(ctx).
		Where("runner_id = ? AND status IN ?", runnerID, []string{
			agentpod.PodStatusRunning,
			agentpod.PodStatusInitializing,
		}).
		Find(&dbPods).Error
	if err != nil {
		return err
	}

	// Create a set of reported pod keys
	reportedSet := make(map[string]bool)
	for _, key := range reportedPodKeys {
		reportedSet[key] = true
	}

	// Mark pods not in heartbeat as orphaned
	now := time.Now()
	for _, pod := range dbPods {
		if !reportedSet[pod.PodKey] {
			s.db.WithContext(ctx).Model(pod).Updates(map[string]interface{}{
				"status":      agentpod.PodStatusOrphaned,
				"finished_at": now,
			})
		}
	}

	return nil
}

// GetPodsByTicket returns pods for a ticket
func (s *PodService) GetPodsByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	if err := s.db.WithContext(ctx).
		Preload("Runner").
		Preload("AgentType").
		Where("ticket_id = ?", ticketID).
		Order("created_at DESC").
		Find(&pods).Error; err != nil {
		return nil, err
	}
	return pods, nil
}

// CleanupStalePods marks stale pods as terminated
func (s *PodService) CleanupStalePods(ctx context.Context, maxIdleHours int) (int64, error) {
	threshold := time.Now().Add(-time.Duration(maxIdleHours) * time.Hour)
	now := time.Now()

	result := s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("status IN ? AND last_activity < ?", []string{
			agentpod.PodStatusDisconnected,
		}, threshold).
		Updates(map[string]interface{}{
			"status":      agentpod.PodStatusTerminated,
			"finished_at": now,
		})

	return result.RowsAffected, result.Error
}

// ListByRunner returns pods for a runner with optional status filter
func (s *PodService) ListByRunner(ctx context.Context, runnerID int64, status string) ([]*agentpod.Pod, error) {
	query := s.db.WithContext(ctx).Where("runner_id = ?", runnerID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var pods []*agentpod.Pod
	if err := query.
		Preload("Runner").
		Preload("AgentType").
		Order("created_at DESC").
		Find(&pods).Error; err != nil {
		return nil, err
	}
	return pods, nil
}

// ListByTicket returns pods for a ticket
func (s *PodService) ListByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	if err := s.db.WithContext(ctx).
		Preload("Runner").
		Preload("AgentType").
		Where("ticket_id = ?", ticketID).
		Order("created_at DESC").
		Find(&pods).Error; err != nil {
		return nil, err
	}
	return pods, nil
}

// PodUpdateFunc is a callback for pod updates
type PodUpdateFunc func(*agentpod.Pod)

// Subscribe subscribes to pod updates and returns an unsubscribe function
func (s *PodService) Subscribe(ctx context.Context, podKey string, callback PodUpdateFunc) (func(), error) {
	// In a real implementation, this would use Redis pub/sub or similar
	// For now, return a simple unsubscribe function
	return func() {}, nil
}

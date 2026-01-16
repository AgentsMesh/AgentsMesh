package binding

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

var (
	ErrBindingNotFound      = errors.New("binding not found")
	ErrBindingExists        = errors.New("binding already exists")
	ErrSelfBinding          = errors.New("cannot bind a pod to itself")
	ErrInvalidScope         = errors.New("invalid scope")
	ErrNotAuthorized        = errors.New("not authorized for this operation")
	ErrBindingNotPending    = errors.New("binding is not pending")
	ErrBindingNotActive     = errors.New("binding is not active")
	ErrNoValidPendingScopes = errors.New("no valid pending scopes to approve")
)

// Default expiry for pending bindings (24 hours)
const PendingExpiryHours = 24

// Service handles pod binding operations
type Service struct {
	db         *gorm.DB
	podQuerier PodQuerier
}

// PodQuerier provides pod information for policy evaluation
type PodQuerier interface {
	GetPodInfo(ctx context.Context, podKey string) (map[string]interface{}, error)
}

// NewService creates a new binding service
func NewService(db *gorm.DB, podQuerier PodQuerier) *Service {
	return &Service{
		db:         db,
		podQuerier: podQuerier,
	}
}

// validateScopes validates that all scopes are valid
func (s *Service) validateScopes(scopes []string) error {
	for _, scope := range scopes {
		if !channel.ValidBindingScopes[scope] {
			return ErrInvalidScope
		}
	}
	return nil
}

// RequestBinding creates a binding request between two pods
func (s *Service) RequestBinding(ctx context.Context, orgID int64, initiatorPod, targetPod string, scopes []string, policy string) (*channel.PodBinding, error) {
	// Validate scopes
	if err := s.validateScopes(scopes); err != nil {
		return nil, err
	}

	// Prevent self-binding
	if initiatorPod == targetPod {
		return nil, ErrSelfBinding
	}

	// Check for existing binding (active or pending)
	existing, err := s.GetExistingBinding(ctx, initiatorPod, targetPod)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Evaluate policy to determine if auto-approve
	autoApprove, initialStatus := s.evaluatePolicy(ctx, initiatorPod, targetPod, policy)

	// Calculate expiry for pending bindings
	var expiresAt *time.Time
	if initialStatus == channel.BindingStatusPending {
		t := time.Now().Add(time.Duration(PendingExpiryHours) * time.Hour)
		expiresAt = &t
	}

	// Determine granted vs pending scopes based on policy
	var grantedScopes, pendingScopes []string
	if autoApprove {
		grantedScopes = scopes
		pendingScopes = []string{}
	} else {
		grantedScopes = []string{}
		pendingScopes = scopes
	}

	now := time.Now()
	binding := &channel.PodBinding{
		OrganizationID:   orgID,
		InitiatorPod: initiatorPod,
		TargetPod:    targetPod,
		GrantedScopes:    pq.StringArray(grantedScopes),
		PendingScopes:    pq.StringArray(pendingScopes),
		Status:           initialStatus,
		RequestedAt:      &now,
		ExpiresAt:        expiresAt,
	}

	if autoApprove {
		binding.RespondedAt = &now
	}

	if err := s.db.WithContext(ctx).Create(binding).Error; err != nil {
		return nil, err
	}

	return binding, nil
}

// RequestScopes requests additional scopes on an existing binding
func (s *Service) RequestScopes(ctx context.Context, bindingID int64, requesterPod string, scopes []string) (*channel.PodBinding, error) {
	if err := s.validateScopes(scopes); err != nil {
		return nil, err
	}

	binding, err := s.GetBinding(ctx, bindingID)
	if err != nil {
		return nil, err
	}

	if binding.InitiatorPod != requesterPod {
		return nil, ErrNotAuthorized
	}

	if !binding.IsActive() {
		return nil, ErrBindingNotActive
	}

	// Filter out already granted or pending scopes
	var newScopes []string
	for _, scope := range scopes {
		if !binding.HasScope(scope) && !binding.HasPendingScope(scope) {
			newScopes = append(newScopes, scope)
		}
	}

	if len(newScopes) == 0 {
		return binding, nil // No new scopes to request
	}

	// Check if we can auto-approve
	autoApprove, _ := s.evaluatePolicy(ctx, binding.InitiatorPod, binding.TargetPod, "")

	if autoApprove {
		binding.GrantedScopes = append(binding.GrantedScopes, newScopes...)
	} else {
		binding.PendingScopes = append(binding.PendingScopes, newScopes...)
	}

	if err := s.db.WithContext(ctx).Save(binding).Error; err != nil {
		return nil, err
	}

	return binding, nil
}

// ApproveScopes approves pending scope requests
func (s *Service) ApproveScopes(ctx context.Context, bindingID int64, approverPod string, scopes []string) (*channel.PodBinding, error) {
	binding, err := s.GetBinding(ctx, bindingID)
	if err != nil {
		return nil, err
	}

	if binding.TargetPod != approverPod {
		return nil, ErrNotAuthorized
	}

	// Only approve scopes that are actually pending
	var approved []string
	for _, scope := range scopes {
		if binding.HasPendingScope(scope) {
			approved = append(approved, scope)
		}
	}

	if len(approved) == 0 {
		return nil, ErrNoValidPendingScopes
	}

	// Move from pending to granted
	newGranted := append([]string{}, binding.GrantedScopes...)
	var newPending []string
	for _, s := range binding.PendingScopes {
		isApproved := false
		for _, a := range approved {
			if s == a {
				isApproved = true
				break
			}
		}
		if isApproved {
			newGranted = append(newGranted, s)
		} else {
			newPending = append(newPending, s)
		}
	}

	binding.GrantedScopes = pq.StringArray(newGranted)
	binding.PendingScopes = pq.StringArray(newPending)

	if err := s.db.WithContext(ctx).Save(binding).Error; err != nil {
		return nil, err
	}

	return binding, nil
}

// AcceptBinding accepts a pending binding request (moves all pending scopes to granted)
func (s *Service) AcceptBinding(ctx context.Context, bindingID int64, targetPod string) (*channel.PodBinding, error) {
	binding, err := s.GetBinding(ctx, bindingID)
	if err != nil {
		return nil, err
	}

	if binding.TargetPod != targetPod {
		return nil, ErrNotAuthorized
	}

	if !binding.IsPending() {
		return nil, ErrBindingNotPending
	}

	// Move pending scopes to granted
	newGranted := append([]string{}, binding.GrantedScopes...)
	newGranted = append(newGranted, binding.PendingScopes...)

	now := time.Now()
	binding.GrantedScopes = pq.StringArray(newGranted)
	binding.PendingScopes = pq.StringArray([]string{})
	binding.Status = channel.BindingStatusActive
	binding.RespondedAt = &now
	binding.ExpiresAt = nil // Active bindings don't expire

	if err := s.db.WithContext(ctx).Save(binding).Error; err != nil {
		return nil, err
	}

	return binding, nil
}

// RejectBinding rejects a pending binding request
func (s *Service) RejectBinding(ctx context.Context, bindingID int64, targetPod string, reason string) (*channel.PodBinding, error) {
	binding, err := s.GetBinding(ctx, bindingID)
	if err != nil {
		return nil, err
	}

	if binding.TargetPod != targetPod {
		return nil, ErrNotAuthorized
	}

	if !binding.IsPending() {
		return nil, ErrBindingNotPending
	}

	now := time.Now()
	binding.Status = channel.BindingStatusRejected
	binding.RespondedAt = &now
	if reason != "" {
		binding.RejectionReason = &reason
	}

	if err := s.db.WithContext(ctx).Save(binding).Error; err != nil {
		return nil, err
	}

	return binding, nil
}

// Unbind removes an active binding between two pods
func (s *Service) Unbind(ctx context.Context, initiatorPod, targetPod string) (bool, error) {
	// Try both directions
	binding, err := s.GetActiveBinding(ctx, initiatorPod, targetPod)
	if err != nil {
		binding, err = s.GetActiveBinding(ctx, targetPod, initiatorPod)
		if err != nil {
			return false, nil
		}
	}

	binding.Status = channel.BindingStatusInactive
	if err := s.db.WithContext(ctx).Save(binding).Error; err != nil {
		return false, err
	}

	return true, nil
}

// CreateAutoBinding creates a binding that is immediately active without approval
func (s *Service) CreateAutoBinding(ctx context.Context, orgID int64, initiatorPod, targetPod string, scopes []string) (*channel.PodBinding, error) {
	if err := s.validateScopes(scopes); err != nil {
		return nil, err
	}

	if initiatorPod == targetPod {
		return nil, ErrSelfBinding
	}

	// Check for existing binding
	existing, err := s.GetExistingBinding(ctx, initiatorPod, targetPod)
	if err == nil && existing != nil {
		return existing, nil
	}

	now := time.Now()
	binding := &channel.PodBinding{
		OrganizationID:   orgID,
		InitiatorPod: initiatorPod,
		TargetPod:    targetPod,
		GrantedScopes:    pq.StringArray(scopes),
		PendingScopes:    pq.StringArray([]string{}),
		Status:           channel.BindingStatusActive,
		RequestedAt:      &now,
		RespondedAt:      &now,
		ExpiresAt:        nil, // Active bindings don't expire
	}

	if err := s.db.WithContext(ctx).Create(binding).Error; err != nil {
		return nil, err
	}

	return binding, nil
}

// GetBinding returns a binding by ID
func (s *Service) GetBinding(ctx context.Context, bindingID int64) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	if err := s.db.WithContext(ctx).First(&binding, bindingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBindingNotFound
		}
		return nil, err
	}
	return &binding, nil
}

// GetActiveBinding returns an active binding between two pods
func (s *Service) GetActiveBinding(ctx context.Context, initiatorPod, targetPod string) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	if err := s.db.WithContext(ctx).
		Where("initiator_pod = ? AND target_pod = ? AND status = ?",
			initiatorPod, targetPod, channel.BindingStatusActive).
		First(&binding).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBindingNotFound
		}
		return nil, err
	}
	return &binding, nil
}

// GetExistingBinding returns any existing binding (active or pending) between two pods
func (s *Service) GetExistingBinding(ctx context.Context, initiatorPod, targetPod string) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	if err := s.db.WithContext(ctx).
		Where("initiator_pod = ? AND target_pod = ? AND status IN ?",
			initiatorPod, targetPod, []string{channel.BindingStatusActive, channel.BindingStatusPending}).
		First(&binding).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBindingNotFound
		}
		return nil, err
	}
	return &binding, nil
}

// GetBindingsForPod returns all bindings for a pod (as initiator or target)
func (s *Service) GetBindingsForPod(ctx context.Context, podKey string, status *string) ([]*channel.PodBinding, error) {
	query := s.db.WithContext(ctx).
		Where("initiator_pod = ? OR target_pod = ?", podKey, podKey)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var bindings []*channel.PodBinding
	if err := query.Order("created_at DESC").Find(&bindings).Error; err != nil {
		return nil, err
	}
	return bindings, nil
}

// GetBoundPods returns pod keys that are bound to a pod
func (s *Service) GetBoundPods(ctx context.Context, podKey string) ([]string, error) {
	active := channel.BindingStatusActive
	bindings, err := s.GetBindingsForPod(ctx, podKey, &active)
	if err != nil {
		return nil, err
	}

	var boundPods []string
	for _, binding := range bindings {
		if binding.InitiatorPod == podKey {
			boundPods = append(boundPods, binding.TargetPod)
		} else {
			boundPods = append(boundPods, binding.InitiatorPod)
		}
	}

	return boundPods, nil
}

// IsBound checks if two pods are bound
func (s *Service) IsBound(ctx context.Context, podA, podB string) (bool, error) {
	_, err := s.GetActiveBinding(ctx, podA, podB)
	if err == nil {
		return true, nil
	}

	_, err = s.GetActiveBinding(ctx, podB, podA)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, ErrBindingNotFound) {
		return false, nil
	}

	return false, err
}

// GetPendingRequests returns pending binding requests for a target pod
func (s *Service) GetPendingRequests(ctx context.Context, targetPod string) ([]*channel.PodBinding, error) {
	var bindings []*channel.PodBinding
	if err := s.db.WithContext(ctx).
		Where("target_pod = ? AND status = ?", targetPod, channel.BindingStatusPending).
		Order("created_at ASC").
		Find(&bindings).Error; err != nil {
		return nil, err
	}
	return bindings, nil
}

// CleanupExpiredBindings marks expired pending bindings as expired
func (s *Service) CleanupExpiredBindings(ctx context.Context) (int64, error) {
	result := s.db.WithContext(ctx).
		Model(&channel.PodBinding{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?",
			channel.BindingStatusPending, time.Now()).
		Update("status", channel.BindingStatusExpired)

	return result.RowsAffected, result.Error
}

// evaluatePolicy evaluates the binding policy to determine if auto-approve
func (s *Service) evaluatePolicy(ctx context.Context, initiatorPod, targetPod, policy string) (bool, string) {
	// If explicit policy is set, use it
	if policy == channel.BindingPolicyExplicitOnly {
		return false, channel.BindingStatusPending
	}

	// If no pod querier available, require explicit confirmation
	if s.podQuerier == nil {
		return false, channel.BindingStatusPending
	}

	// Get pod info
	initiatorInfo, err := s.podQuerier.GetPodInfo(ctx, initiatorPod)
	if err != nil {
		return false, channel.BindingStatusPending
	}

	targetInfo, err := s.podQuerier.GetPodInfo(ctx, targetPod)
	if err != nil {
		return false, channel.BindingStatusPending
	}

	// Same user - auto approve
	initiatorUserID, ok1 := initiatorInfo["user_id"]
	targetUserID, ok2 := targetInfo["user_id"]
	if ok1 && ok2 && initiatorUserID == targetUserID {
		return true, channel.BindingStatusActive
	}

	// Same project - check policy
	if policy == channel.BindingPolicySameProjectAuto {
		initiatorProjectID, ok1 := initiatorInfo["project_id"]
		targetProjectID, ok2 := targetInfo["project_id"]
		if ok1 && ok2 && initiatorProjectID == targetProjectID {
			return true, channel.BindingStatusActive
		}
	}

	// Default: require explicit confirmation
	return false, channel.BindingStatusPending
}

// HasScope checks if initiator has a specific scope on target
func (s *Service) HasScope(ctx context.Context, initiatorPod, targetPod, scope string) (bool, error) {
	binding, err := s.GetActiveBinding(ctx, initiatorPod, targetPod)
	if err != nil {
		if errors.Is(err, ErrBindingNotFound) {
			return false, nil
		}
		return false, err
	}
	return binding.HasScope(scope), nil
}

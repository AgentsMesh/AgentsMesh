package binding

import (
	"context"
	"errors"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/channel"
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

package channel

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// CreateBinding creates a pod binding request
func (s *Service) CreateBinding(ctx context.Context, orgID int64, initiatorPod, targetPod string, scopes []string) (*channel.PodBinding, error) {
	binding := &channel.PodBinding{
		OrganizationID: orgID,
		InitiatorPod:   initiatorPod,
		TargetPod:      targetPod,
		GrantedScopes:  scopes,
		Status:         channel.BindingStatusPending,
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
		return nil, err
	}
	return &binding, nil
}

// GetBindingByPods returns a binding between two pods
func (s *Service) GetBindingByPods(ctx context.Context, initiator, target string) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	if err := s.db.WithContext(ctx).
		Where("initiator_pod = ? AND target_pod = ?", initiator, target).
		First(&binding).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

// ListBindingsForPod returns all bindings for a pod (as initiator or target)
func (s *Service) ListBindingsForPod(ctx context.Context, podKey string) ([]*channel.PodBinding, error) {
	var bindings []*channel.PodBinding
	if err := s.db.WithContext(ctx).
		Where("initiator_pod = ? OR target_pod = ?", podKey, podKey).
		Find(&bindings).Error; err != nil {
		return nil, err
	}
	return bindings, nil
}

// ApproveBinding approves a binding request
func (s *Service) ApproveBinding(ctx context.Context, bindingID int64, scopes []string) error {
	return s.db.WithContext(ctx).Model(&channel.PodBinding{}).
		Where("id = ?", bindingID).
		Updates(map[string]interface{}{
			"status":         channel.BindingStatusActive,
			"granted_scopes": scopes,
		}).Error
}

// RejectBinding rejects a binding request
func (s *Service) RejectBinding(ctx context.Context, bindingID int64) error {
	return s.db.WithContext(ctx).Model(&channel.PodBinding{}).
		Where("id = ?", bindingID).
		Update("status", channel.BindingStatusRejected).Error
}

// RevokeBinding revokes an approved binding
func (s *Service) RevokeBinding(ctx context.Context, bindingID int64) error {
	return s.db.WithContext(ctx).Model(&channel.PodBinding{}).
		Where("id = ?", bindingID).
		Update("status", channel.BindingStatusInactive).Error
}

package channel

import (
	"context"
	"log/slog"

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

	if err := s.repo.CreateBinding(ctx, binding); err != nil {
		slog.Error("failed to create binding", "org_id", orgID, "initiator", initiatorPod, "target", targetPod, "error", err)
		return nil, err
	}

	slog.Info("binding created", "binding_id", binding.ID, "initiator", initiatorPod, "target", targetPod)
	return binding, nil
}

// GetBinding returns a binding by ID
func (s *Service) GetBinding(ctx context.Context, bindingID int64) (*channel.PodBinding, error) {
	binding, err := s.repo.GetBindingByID(ctx, bindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, ErrChannelNotFound
	}
	return binding, nil
}

// GetBindingByPods returns a binding between two pods
func (s *Service) GetBindingByPods(ctx context.Context, initiator, target string) (*channel.PodBinding, error) {
	binding, err := s.repo.GetBindingByPods(ctx, initiator, target)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, ErrChannelNotFound
	}
	return binding, nil
}

// ListBindingsForPod returns all bindings for a pod (as initiator or target)
func (s *Service) ListBindingsForPod(ctx context.Context, podKey string) ([]*channel.PodBinding, error) {
	return s.repo.ListBindingsForPod(ctx, podKey)
}

// ApproveBinding approves a binding request
func (s *Service) ApproveBinding(ctx context.Context, bindingID int64, scopes []string) error {
	if err := s.repo.UpdateBindingFields(ctx, bindingID, map[string]interface{}{
		"status":         channel.BindingStatusActive,
		"granted_scopes": scopes,
	}); err != nil {
		slog.Error("failed to approve binding", "binding_id", bindingID, "error", err)
		return err
	}
	slog.Info("binding approved", "binding_id", bindingID)
	return nil
}

// RejectBinding rejects a binding request
func (s *Service) RejectBinding(ctx context.Context, bindingID int64) error {
	if err := s.repo.UpdateBindingFields(ctx, bindingID, map[string]interface{}{
		"status": channel.BindingStatusRejected,
	}); err != nil {
		slog.Error("failed to reject binding", "binding_id", bindingID, "error", err)
		return err
	}
	slog.Info("binding rejected", "binding_id", bindingID)
	return nil
}

// RevokeBinding revokes an approved binding
func (s *Service) RevokeBinding(ctx context.Context, bindingID int64) error {
	if err := s.repo.UpdateBindingFields(ctx, bindingID, map[string]interface{}{
		"status": channel.BindingStatusInactive,
	}); err != nil {
		slog.Error("failed to revoke binding", "binding_id", bindingID, "error", err)
		return err
	}
	slog.Info("binding revoked", "binding_id", bindingID)
	return nil
}

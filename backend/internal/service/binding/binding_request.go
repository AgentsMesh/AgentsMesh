package binding

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/lib/pq"
)

func (s *Service) RequestBinding(ctx context.Context, orgID int64, initiatorPod, targetPod string, scopes []string, policy string) (*channel.PodBinding, error) {
	if err := s.validateScopes(scopes); err != nil {
		return nil, err
	}

	if initiatorPod == targetPod {
		return nil, ErrSelfBinding
	}

	existing, err := s.GetExistingBinding(ctx, initiatorPod, targetPod)
	if err == nil && existing != nil {
		return existing, nil
	}

	autoApprove, initialStatus := s.evaluatePolicy(ctx, initiatorPod, targetPod, policy)

	var expiresAt *time.Time
	if initialStatus == channel.BindingStatusPending {
		t := time.Now().Add(time.Duration(PendingExpiryHours) * time.Hour)
		expiresAt = &t
	}

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
		OrganizationID: orgID,
		InitiatorPod:   initiatorPod,
		TargetPod:      targetPod,
		GrantedScopes:  pq.StringArray(grantedScopes),
		PendingScopes:  pq.StringArray(pendingScopes),
		Status:         initialStatus,
		RequestedAt:    &now,
		ExpiresAt:      expiresAt,
	}

	if autoApprove {
		binding.RespondedAt = &now
	}

	if err := s.repo.Create(ctx, binding); err != nil {
		slog.ErrorContext(ctx, "failed to create binding request", "initiator", initiatorPod, "target", targetPod, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "binding requested", "binding_id", binding.ID, "initiator", initiatorPod, "target", targetPod, "status", initialStatus)
	return binding, nil
}

func (s *Service) CreateAutoBinding(ctx context.Context, orgID int64, initiatorPod, targetPod string, scopes []string) (*channel.PodBinding, error) {
	if err := s.validateScopes(scopes); err != nil {
		return nil, err
	}

	if initiatorPod == targetPod {
		return nil, ErrSelfBinding
	}

	existing, err := s.GetExistingBinding(ctx, initiatorPod, targetPod)
	if err == nil && existing != nil {
		return existing, nil
	}

	now := time.Now()
	binding := &channel.PodBinding{
		OrganizationID: orgID,
		InitiatorPod:   initiatorPod,
		TargetPod:      targetPod,
		GrantedScopes:  pq.StringArray(scopes),
		PendingScopes:  pq.StringArray([]string{}),
		Status:         channel.BindingStatusActive,
		RequestedAt:    &now,
		RespondedAt:    &now,
		ExpiresAt:      nil, // Active bindings don't expire
	}

	if err := s.repo.Create(ctx, binding); err != nil {
		slog.ErrorContext(ctx, "failed to create auto-binding", "initiator", initiatorPod, "target", targetPod, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "auto-binding created", "binding_id", binding.ID, "initiator", initiatorPod, "target", targetPod)
	return binding, nil
}

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

	newGranted := append([]string{}, binding.GrantedScopes...)
	newGranted = append(newGranted, binding.PendingScopes...)

	now := time.Now()
	binding.GrantedScopes = pq.StringArray(newGranted)
	binding.PendingScopes = pq.StringArray([]string{})
	binding.Status = channel.BindingStatusActive
	binding.RespondedAt = &now
	binding.ExpiresAt = nil // Active bindings don't expire

	if err := s.repo.Save(ctx, binding); err != nil {
		slog.ErrorContext(ctx, "failed to accept binding", "binding_id", bindingID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "binding accepted", "binding_id", bindingID, "target_pod", targetPod)
	return binding, nil
}

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

	if err := s.repo.Save(ctx, binding); err != nil {
		slog.ErrorContext(ctx, "failed to reject binding", "binding_id", bindingID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "binding rejected", "binding_id", bindingID, "target_pod", targetPod, "reason", reason)
	return binding, nil
}

func (s *Service) Unbind(ctx context.Context, initiatorPod, targetPod string) (bool, error) {
	binding, err := s.GetActiveBinding(ctx, initiatorPod, targetPod)
	if err != nil {
		binding, err = s.GetActiveBinding(ctx, targetPod, initiatorPod)
		if err != nil {
			return false, nil
		}
	}

	binding.Status = channel.BindingStatusInactive
	if err := s.repo.Save(ctx, binding); err != nil {
		slog.ErrorContext(ctx, "failed to unbind", "initiator", initiatorPod, "target", targetPod, "error", err)
		return false, err
	}

	slog.InfoContext(ctx, "pods unbound", "binding_id", binding.ID, "initiator", binding.InitiatorPod, "target", binding.TargetPod)
	return true, nil
}

func (s *Service) CleanupExpiredBindings(ctx context.Context) (int64, error) {
	count, err := s.repo.MarkExpired(ctx, time.Now())
	if err != nil {
		slog.ErrorContext(ctx, "failed to cleanup expired bindings", "error", err)
		return 0, err
	}
	if count > 0 {
		slog.InfoContext(ctx, "expired bindings cleaned up", "count", count)
	}
	return count, nil
}

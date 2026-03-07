package binding

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/lib/pq"
)

// RequestBinding creates a binding request between two pods
func (s *Service) RequestBinding(ctx context.Context, orgID int64, initiatorPod, targetPod string, scopes []string, policy string) (*channel.PodBinding, error) {
	if err := s.validateScopes(scopes); err != nil {
		return nil, err
	}

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
		return nil, err
	}

	return binding, nil
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

	if err := s.repo.Save(ctx, binding); err != nil {
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

	if err := s.repo.Save(ctx, binding); err != nil {
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
	if err := s.repo.Save(ctx, binding); err != nil {
		return false, err
	}

	return true, nil
}

// CleanupExpiredBindings marks expired pending bindings as expired
func (s *Service) CleanupExpiredBindings(ctx context.Context) (int64, error) {
	return s.repo.MarkExpired(ctx, time.Now())
}

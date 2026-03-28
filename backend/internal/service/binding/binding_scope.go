package binding

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/lib/pq"
)

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

	if err := s.repo.Save(ctx, binding); err != nil {
		slog.Error("failed to save requested scopes", "binding_id", bindingID, "error", err)
		return nil, err
	}

	slog.Info("scopes requested", "binding_id", bindingID, "new_scopes", newScopes, "auto_approved", autoApprove)
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
	for _, scopeItem := range binding.PendingScopes {
		isApproved := false
		for _, a := range approved {
			if scopeItem == a {
				isApproved = true
				break
			}
		}
		if isApproved {
			newGranted = append(newGranted, scopeItem)
		} else {
			newPending = append(newPending, scopeItem)
		}
	}

	binding.GrantedScopes = pq.StringArray(newGranted)
	binding.PendingScopes = pq.StringArray(newPending)

	if err := s.repo.Save(ctx, binding); err != nil {
		slog.Error("failed to save approved scopes", "binding_id", bindingID, "error", err)
		return nil, err
	}

	slog.Info("scopes approved", "binding_id", bindingID, "approved_scopes", approved, "approver_pod", approverPod)
	return binding, nil
}

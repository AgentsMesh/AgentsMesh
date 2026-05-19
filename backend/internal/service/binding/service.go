package binding

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
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

type Service struct {
	repo       channel.BindingRepository
	podQuerier PodQuerier
}

type PodQuerier interface {
	GetPodInfo(ctx context.Context, podKey string) (map[string]interface{}, error)
}

func NewService(repo channel.BindingRepository, podQuerier PodQuerier) *Service {
	return &Service{
		repo:       repo,
		podQuerier: podQuerier,
	}
}

func (s *Service) validateScopes(scopes []string) error {
	for _, scope := range scopes {
		if !channel.ValidBindingScopes[scope] {
			return ErrInvalidScope
		}
	}
	return nil
}

func (s *Service) GetBinding(ctx context.Context, bindingID int64) (*channel.PodBinding, error) {
	binding, err := s.repo.GetByID(ctx, bindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, ErrBindingNotFound
	}
	return binding, nil
}

func (s *Service) GetActiveBinding(ctx context.Context, initiatorPod, targetPod string) (*channel.PodBinding, error) {
	binding, err := s.repo.GetActive(ctx, initiatorPod, targetPod)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, ErrBindingNotFound
	}
	return binding, nil
}

func (s *Service) GetExistingBinding(ctx context.Context, initiatorPod, targetPod string) (*channel.PodBinding, error) {
	binding, err := s.repo.GetExisting(ctx, initiatorPod, targetPod)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, ErrBindingNotFound
	}
	return binding, nil
}

func (s *Service) GetBindingsForPod(ctx context.Context, podKey string, status *string) ([]*channel.PodBinding, error) {
	return s.repo.ListForPod(ctx, podKey, status)
}

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

	slog.ErrorContext(ctx, "failed to check binding", "pod_a", podA, "pod_b", podB, "error", err)
	return false, err
}

func (s *Service) GetPendingRequests(ctx context.Context, targetPod string) ([]*channel.PodBinding, error) {
	return s.repo.ListPending(ctx, targetPod)
}

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

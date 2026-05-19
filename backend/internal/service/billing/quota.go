package billing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

func (s *Service) CheckQuota(ctx context.Context, orgID int64, resource string, requestedAmount int) error {
	sub, err := s.GetSubscription(ctx, orgID)

	var plan *billing.SubscriptionPlan
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			plan, _ = s.GetPlan(ctx, billing.PlanBased)
			if plan == nil {
				slog.WarnContext(ctx, "no based plan found in database, allowing by default", "org_id", orgID, "resource", resource)
				return nil
			}
		} else {
			return err
		}
	} else {
		if sub.IsFrozen() {
			slog.WarnContext(ctx, "quota check denied: subscription frozen", "org_id", orgID, "resource", resource)
			return ErrSubscriptionFrozen
		}

		plan = sub.Plan
		if plan == nil {
			plan, _ = s.GetPlanByID(ctx, sub.PlanID)
		}
	}

	if plan == nil {
		return ErrPlanNotFound
	}

	if sub != nil && sub.CustomQuotas != nil {
		if customLimit, ok := sub.CustomQuotas[resource]; ok {
			if limit, ok := customLimit.(float64); ok && int(limit) != -1 {
				current, err := s.getCurrentResourceCount(ctx, orgID, resource)
				if err != nil {
					return fmt.Errorf("failed to get current resource count: %w", err)
				}
				if current+requestedAmount > int(limit) {
					return ErrQuotaExceeded
				}
				return nil
			}
		}
	}

	var limit int
	switch resource {
	case "users":
		limit = plan.MaxUsers
	case "runners":
		limit = plan.MaxRunners
	case "concurrent_pods":
		limit = plan.MaxConcurrentPods
	case "repositories":
		limit = plan.MaxRepositories
	case "pod_minutes":
		limit = plan.IncludedPodMinutes
	default:
		return nil
	}

	if limit == -1 {
		return nil
	}

	current, err := s.getCurrentResourceCount(ctx, orgID, resource)
	if err != nil {
		return fmt.Errorf("failed to get current resource count: %w", err)
	}
	if current+requestedAmount > limit {
		slog.WarnContext(ctx, "quota exceeded", "org_id", orgID, "resource", resource, "current", current, "requested", requestedAmount, "limit", limit)
		return ErrQuotaExceeded
	}

	return nil
}

// CheckSeatAvailability gates member invitations against purchased seats (not plan limits).
func (s *Service) CheckSeatAvailability(ctx context.Context, orgID int64, requestedSeats int) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		sub = &billing.Subscription{
			SeatCount: 1,
		}
	}

	usedSeats, _ := s.repo.CountOrgMembers(ctx, orgID)

	pendingInvitations, _ := s.repo.CountPendingInvitations(ctx, orgID)

	availableSeats := sub.SeatCount - int(usedSeats) - int(pendingInvitations)

	if availableSeats < requestedSeats {
		slog.WarnContext(ctx, "seat quota exceeded", "org_id", orgID, "available", availableSeats, "requested", requestedSeats)
		return ErrQuotaExceeded
	}

	return nil
}

func (s *Service) getCurrentResourceCount(ctx context.Context, orgID int64, resource string) (int, error) {
	var count int64
	var err error

	switch resource {
	case "users":
		count, err = s.repo.CountOrgMembers(ctx, orgID)
	case "runners":
		count, err = s.repo.CountRunners(ctx, orgID)
	case "concurrent_pods":
		count, err = s.repo.CountActivePods(ctx, orgID)
	case "repositories":
		count, err = s.repo.CountRepositories(ctx, orgID)
	case "pod_minutes":
		usage, err := s.GetUsage(ctx, orgID, billing.UsageTypePodMinutes)
		return int(usage), err
	}

	return int(count), err
}

func (s *Service) GetCurrentConcurrentPods(ctx context.Context, orgID int64) (int, error) {
	return s.getCurrentResourceCount(ctx, orgID, "concurrent_pods")
}

func (s *Service) SetCustomQuota(ctx context.Context, orgID int64, resource string, limit int) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return err
	}

	if sub.CustomQuotas == nil {
		sub.CustomQuotas = make(billing.CustomQuotas)
	}

	sub.CustomQuotas[resource] = limit

	if err = s.repo.SaveSubscription(ctx, sub); err != nil {
		slog.ErrorContext(ctx, "failed to save custom quota", "org_id", orgID, "resource", resource, "limit", limit, "error", err)
		return err
	}
	slog.InfoContext(ctx, "custom quota set", "org_id", orgID, "resource", resource, "limit", limit)
	return nil
}

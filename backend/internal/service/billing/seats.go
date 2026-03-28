package billing

import (
	"context"
	"fmt"
	"log/slog"

	billingdomain "github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// GetSeatUsage returns seat usage information for an organization
func (s *Service) GetSeatUsage(ctx context.Context, orgID int64) (*SeatUsage, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Count current members
	memberCount, err := s.repo.CountOrgMembers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to count organization members: %w", err)
	}

	plan := sub.Plan
	if plan == nil {
		plan, err = s.GetPlanByID(ctx, sub.PlanID)
		if err != nil {
			return nil, fmt.Errorf("failed to get plan: %w", err)
		}
	}

	return &SeatUsage{
		TotalSeats:     sub.SeatCount,
		UsedSeats:      int(memberCount),
		AvailableSeats: sub.SeatCount - int(memberCount),
		MaxSeats:       plan.MaxUsers,
		CanAddSeats:    plan.Name != billingdomain.PlanBased, // Based plan has fixed 1 seat
	}, nil
}

// UpdateSeats updates the seat count for a subscription via the payment provider.
// It validates the request, calls the provider API to update quantity, then syncs local DB.
func (s *Service) UpdateSeats(ctx context.Context, orgID int64, additionalSeats int) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return ErrSubscriptionNotFound
	}

	// Only active or trialing subscriptions can add seats
	if !sub.IsActive() && !sub.IsTrialing() {
		if sub.IsFrozen() {
			return ErrSubscriptionFrozen
		}
		return ErrSubscriptionNotActive
	}

	plan := sub.Plan
	if plan == nil {
		plan, err = s.GetPlanByID(ctx, sub.PlanID)
		if err != nil {
			return ErrPlanNotFound
		}
	}

	// Based plan cannot add seats
	if plan.Name == billingdomain.PlanBased {
		return ErrInvalidPlan
	}

	newTotalSeats := sub.SeatCount + additionalSeats

	// Validate against plan max_users limit
	if plan.MaxUsers > 0 && newTotalSeats > plan.MaxUsers {
		return ErrQuotaExceeded
	}

	// Get subscription provider and update via API
	if sub.LemonSqueezySubscriptionID != nil && *sub.LemonSqueezySubscriptionID != "" {
		provider, err := s.getSubscriptionProvider()
		if err != nil {
			return fmt.Errorf("payment provider not available: %w", err)
		}
		if err := provider.UpdateSubscriptionSeats(ctx, *sub.LemonSqueezySubscriptionID, newTotalSeats); err != nil {
			return fmt.Errorf("failed to update seats with provider: %w", err)
		}
	} else if sub.StripeSubscriptionID != nil && *sub.StripeSubscriptionID != "" {
		// Stripe seat update would go here in the future
		// For now, just update locally
	}

	// Sync local DB
	// NOTE: If this fails after provider API succeeded, the webhook sync (Phase 1)
	// will reconcile the data on the next subscription_updated event.
	if err := s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"seat_count": newTotalSeats,
	}); err != nil {
		slog.Warn("provider API succeeded but DB update failed for seat change",
			"org_id", orgID, "new_seats", newTotalSeats, "error", err)
		return fmt.Errorf("failed to sync seat count locally: %w", err)
	}
	return nil
}

// getSubscriptionProvider returns the SubscriptionProvider from the payment factory
func (s *Service) getSubscriptionProvider() (payment.SubscriptionProvider, error) {
	if s.paymentFactory == nil {
		return nil, fmt.Errorf("payment factory not configured")
	}
	provider, err := s.paymentFactory.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	subProvider, ok := provider.(payment.SubscriptionProvider)
	if !ok {
		return nil, fmt.Errorf("provider does not support subscription management")
	}
	return subProvider, nil
}

// AdminSetSeatCount directly sets the seat count for a subscription without payment validation.
func (s *Service) AdminSetSeatCount(ctx context.Context, orgID int64, seatCount int) error {
	return s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"seat_count": seatCount,
	})
}

// SeatUsage represents seat usage information
type SeatUsage struct {
	TotalSeats     int  `json:"total_seats"`
	UsedSeats      int  `json:"used_seats"`
	AvailableSeats int  `json:"available_seats"`
	MaxSeats       int  `json:"max_seats"`
	CanAddSeats    bool `json:"can_add_seats"`
}

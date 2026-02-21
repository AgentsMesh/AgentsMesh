package billing

import (
	"context"

	billingdomain "github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// GetSeatUsage returns seat usage information for an organization
func (s *Service) GetSeatUsage(ctx context.Context, orgID int64) (*SeatUsage, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Count current members
	var memberCount int64
	s.db.WithContext(ctx).Table("organization_members").
		Where("organization_id = ?", orgID).
		Count(&memberCount)

	plan := sub.Plan
	if plan == nil {
		plan, _ = s.GetPlanByID(ctx, sub.PlanID)
	}

	return &SeatUsage{
		TotalSeats:     sub.SeatCount,
		UsedSeats:      int(memberCount),
		AvailableSeats: sub.SeatCount - int(memberCount),
		MaxSeats:       plan.MaxUsers,
		CanAddSeats:    plan.Name != billingdomain.PlanBased, // Based plan has fixed 1 seat
	}, nil
}

// AdminSetSeatCount directly sets the seat count for a subscription without payment validation.
func (s *Service) AdminSetSeatCount(ctx context.Context, orgID int64, seatCount int) error {
	return s.db.WithContext(ctx).Model(&billingdomain.Subscription{}).
		Where("organization_id = ?", orgID).
		Update("seat_count", seatCount).Error
}

// SeatUsage represents seat usage information
type SeatUsage struct {
	TotalSeats     int  `json:"total_seats"`
	UsedSeats      int  `json:"used_seats"`
	AvailableSeats int  `json:"available_seats"`
	MaxSeats       int  `json:"max_seats"`
	CanAddSeats    bool `json:"can_add_seats"`
}

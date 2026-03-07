package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

// ========== Assignee Operations ==========

// UpdateAssignees updates ticket assignees.
func (s *Service) UpdateAssignees(ctx context.Context, ticketID int64, userIDs []int64) error {
	return s.repo.ReplaceAssignees(ctx, ticketID, userIDs)
}

// AddAssignee adds an assignee to a ticket.
func (s *Service) AddAssignee(ctx context.Context, ticketID, userID int64) error {
	return s.repo.AddAssignee(ctx, ticketID, userID)
}

// RemoveAssignee removes an assignee from a ticket.
func (s *Service) RemoveAssignee(ctx context.Context, ticketID, userID int64) error {
	return s.repo.RemoveAssignee(ctx, ticketID, userID)
}

// GetAssignees returns assignees for a ticket.
func (s *Service) GetAssignees(ctx context.Context, ticketID int64) ([]*user.User, error) {
	return s.repo.GetAssigneeUsers(ctx, ticketID)
}

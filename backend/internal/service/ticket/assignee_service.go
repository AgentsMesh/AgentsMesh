package ticket

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

// ========== Assignee Operations ==========

// UpdateAssignees updates ticket assignees.
func (s *Service) UpdateAssignees(ctx context.Context, ticketID int64, userIDs []int64) error {
	if err := s.repo.ReplaceAssignees(ctx, ticketID, userIDs); err != nil {
		slog.Error("failed to update assignees", "ticket_id", ticketID, "error", err)
		return err
	}
	slog.Info("assignees updated", "ticket_id", ticketID, "user_ids", userIDs)
	return nil
}

// AddAssignee adds an assignee to a ticket.
func (s *Service) AddAssignee(ctx context.Context, ticketID, userID int64) error {
	if err := s.repo.AddAssignee(ctx, ticketID, userID); err != nil {
		slog.Error("failed to add assignee", "ticket_id", ticketID, "user_id", userID, "error", err)
		return err
	}
	slog.Info("assignee added", "ticket_id", ticketID, "user_id", userID)
	return nil
}

// RemoveAssignee removes an assignee from a ticket.
func (s *Service) RemoveAssignee(ctx context.Context, ticketID, userID int64) error {
	if err := s.repo.RemoveAssignee(ctx, ticketID, userID); err != nil {
		slog.Error("failed to remove assignee", "ticket_id", ticketID, "user_id", userID, "error", err)
		return err
	}
	slog.Info("assignee removed", "ticket_id", ticketID, "user_id", userID)
	return nil
}

// GetAssignees returns assignees for a ticket.
func (s *Service) GetAssignees(ctx context.Context, ticketID int64) ([]*user.User, error) {
	return s.repo.GetAssigneeUsers(ctx, ticketID)
}

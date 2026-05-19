package ticket

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

func (s *Service) UpdateAssignees(ctx context.Context, ticketID int64, userIDs []int64) error {
	if err := s.repo.ReplaceAssignees(ctx, ticketID, userIDs); err != nil {
		slog.ErrorContext(ctx, "failed to update assignees", "ticket_id", ticketID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "assignees updated", "ticket_id", ticketID, "user_ids", userIDs)
	return nil
}

func (s *Service) AddAssignee(ctx context.Context, ticketID, userID int64) error {
	if err := s.repo.AddAssignee(ctx, ticketID, userID); err != nil {
		slog.ErrorContext(ctx, "failed to add assignee", "ticket_id", ticketID, "user_id", userID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "assignee added", "ticket_id", ticketID, "user_id", userID)
	return nil
}

func (s *Service) RemoveAssignee(ctx context.Context, ticketID, userID int64) error {
	if err := s.repo.RemoveAssignee(ctx, ticketID, userID); err != nil {
		slog.ErrorContext(ctx, "failed to remove assignee", "ticket_id", ticketID, "user_id", userID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "assignee removed", "ticket_id", ticketID, "user_id", userID)
	return nil
}

func (s *Service) GetAssignees(ctx context.Context, ticketID int64) ([]*user.User, error) {
	return s.repo.GetAssigneeUsers(ctx, ticketID)
}

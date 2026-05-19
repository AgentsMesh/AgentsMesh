package ticket

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

func (s *Service) CreateLabel(ctx context.Context, orgID int64, repoID *int64, name, color string) (*ticket.Label, error) {
	existing, err := s.repo.GetLabelByOrgNameRepo(ctx, orgID, name, repoID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateLabel
	}

	label := &ticket.Label{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Name:           name,
		Color:          color,
	}
	if err := s.repo.CreateLabel(ctx, label); err != nil {
		slog.ErrorContext(ctx, "failed to create label", "org_id", orgID, "name", name, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "label created", "label_id", label.ID, "org_id", orgID, "name", name)
	return label, nil
}

func (s *Service) GetLabel(ctx context.Context, labelID int64) (*ticket.Label, error) {
	label, err := s.repo.GetLabel(ctx, labelID)
	if err != nil {
		return nil, err
	}
	if label == nil {
		return nil, ErrLabelNotFound
	}
	return label, nil
}

func (s *Service) ListLabels(ctx context.Context, orgID int64, repoID *int64) ([]*ticket.Label, error) {
	return s.repo.ListLabels(ctx, orgID, repoID)
}

func (s *Service) UpdateLabel(ctx context.Context, orgID, labelID int64, updates map[string]interface{}) (*ticket.Label, error) {
	if len(updates) > 0 {
		if err := s.repo.UpdateLabelFields(ctx, orgID, labelID, updates); err != nil {
			slog.ErrorContext(ctx, "failed to update label", "label_id", labelID, "org_id", orgID, "error", err)
			return nil, err
		}
		slog.InfoContext(ctx, "label updated", "label_id", labelID, "org_id", orgID)
	}
	return s.GetLabel(ctx, labelID)
}

func (s *Service) DeleteLabel(ctx context.Context, orgID, labelID int64) error {
	if err := s.repo.DeleteLabelAtomic(ctx, orgID, labelID); err != nil {
		slog.ErrorContext(ctx, "failed to delete label", "label_id", labelID, "org_id", orgID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "label deleted", "label_id", labelID, "org_id", orgID)
	return nil
}

func (s *Service) GetTicketLabels(ctx context.Context, ticketID int64) ([]*ticket.Label, error) {
	return s.repo.GetTicketLabels(ctx, ticketID)
}

func (s *Service) AddLabel(ctx context.Context, ticketID, labelID int64) error {
	if err := s.repo.AddTicketLabel(ctx, ticketID, labelID); err != nil {
		slog.ErrorContext(ctx, "failed to add label to ticket", "ticket_id", ticketID, "label_id", labelID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "label added to ticket", "ticket_id", ticketID, "label_id", labelID)
	return nil
}

func (s *Service) RemoveLabel(ctx context.Context, ticketID, labelID int64) error {
	if err := s.repo.RemoveTicketLabel(ctx, ticketID, labelID); err != nil {
		slog.ErrorContext(ctx, "failed to remove label from ticket", "ticket_id", ticketID, "label_id", labelID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "label removed from ticket", "ticket_id", ticketID, "label_id", labelID)
	return nil
}

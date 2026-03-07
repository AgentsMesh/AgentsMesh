package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// ========== Label Operations ==========

// CreateLabel creates a new label.
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
		return nil, err
	}
	return label, nil
}

// GetLabel returns a label by ID.
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

// ListLabels returns labels for an organization/repository.
func (s *Service) ListLabels(ctx context.Context, orgID int64, repoID *int64) ([]*ticket.Label, error) {
	return s.repo.ListLabels(ctx, orgID, repoID)
}

// UpdateLabel updates a label.
func (s *Service) UpdateLabel(ctx context.Context, orgID, labelID int64, updates map[string]interface{}) (*ticket.Label, error) {
	if len(updates) > 0 {
		if err := s.repo.UpdateLabelFields(ctx, orgID, labelID, updates); err != nil {
			return nil, err
		}
	}
	return s.GetLabel(ctx, labelID)
}

// DeleteLabel deletes a label.
func (s *Service) DeleteLabel(ctx context.Context, orgID, labelID int64) error {
	return s.repo.DeleteLabelAtomic(ctx, orgID, labelID)
}

// GetTicketLabels returns labels for a ticket.
func (s *Service) GetTicketLabels(ctx context.Context, ticketID int64) ([]*ticket.Label, error) {
	return s.repo.GetTicketLabels(ctx, ticketID)
}

// AddLabel adds a label to a ticket.
func (s *Service) AddLabel(ctx context.Context, ticketID, labelID int64) error {
	return s.repo.AddTicketLabel(ctx, ticketID, labelID)
}

// RemoveLabel removes a label from a ticket.
func (s *Service) RemoveLabel(ctx context.Context, ticketID, labelID int64) error {
	return s.repo.RemoveTicketLabel(ctx, ticketID, labelID)
}

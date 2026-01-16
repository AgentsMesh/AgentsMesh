package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// ========== Label Operations ==========

// CreateLabel creates a new label
func (s *Service) CreateLabel(ctx context.Context, orgID int64, repoID *int64, name, color string) (*ticket.Label, error) {
	// Check for duplicate
	var existing ticket.Label
	query := s.db.WithContext(ctx).Where("organization_id = ? AND name = ?", orgID, name)
	if repoID != nil {
		query = query.Where("repository_id = ?", *repoID)
	} else {
		query = query.Where("repository_id IS NULL")
	}
	if err := query.First(&existing).Error; err == nil {
		return nil, ErrDuplicateLabel
	}

	label := &ticket.Label{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Name:           name,
		Color:          color,
	}

	if err := s.db.WithContext(ctx).Create(label).Error; err != nil {
		return nil, err
	}

	return label, nil
}

// GetLabel returns a label by ID
func (s *Service) GetLabel(ctx context.Context, labelID int64) (*ticket.Label, error) {
	var label ticket.Label
	if err := s.db.WithContext(ctx).First(&label, labelID).Error; err != nil {
		return nil, ErrLabelNotFound
	}
	return &label, nil
}

// ListLabels returns labels for an organization/repository
func (s *Service) ListLabels(ctx context.Context, orgID int64, repoID *int64) ([]*ticket.Label, error) {
	query := s.db.WithContext(ctx).Where("organization_id = ?", orgID)

	if repoID != nil {
		// Include both org-level and repo-level labels
		query = query.Where("repository_id IS NULL OR repository_id = ?", *repoID)
	} else {
		// Only org-level labels
		query = query.Where("repository_id IS NULL")
	}

	var labels []*ticket.Label
	if err := query.Order("name ASC").Find(&labels).Error; err != nil {
		return nil, err
	}

	return labels, nil
}

// UpdateLabel updates a label
func (s *Service) UpdateLabel(ctx context.Context, orgID, labelID int64, updates map[string]interface{}) (*ticket.Label, error) {
	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&ticket.Label{}).Where("id = ? AND organization_id = ?", labelID, orgID).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.GetLabel(ctx, labelID)
}

// DeleteLabel deletes a label
func (s *Service) DeleteLabel(ctx context.Context, orgID, labelID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Remove label from all tickets
		if err := tx.Where("label_id = ?", labelID).Delete(&ticket.TicketLabel{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND organization_id = ?", labelID, orgID).Delete(&ticket.Label{}).Error
	})
}

// GetTicketLabels returns labels for a ticket
func (s *Service) GetTicketLabels(ctx context.Context, ticketID int64) ([]*ticket.Label, error) {
	var ticketLabels []ticket.TicketLabel
	if err := s.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Find(&ticketLabels).Error; err != nil {
		return nil, err
	}

	if len(ticketLabels) == 0 {
		return []*ticket.Label{}, nil
	}

	labelIDs := make([]int64, len(ticketLabels))
	for i, tl := range ticketLabels {
		labelIDs[i] = tl.LabelID
	}

	var labels []*ticket.Label
	if err := s.db.WithContext(ctx).Where("id IN ?", labelIDs).Find(&labels).Error; err != nil {
		return nil, err
	}

	return labels, nil
}

// AddLabel adds a label to a ticket
func (s *Service) AddLabel(ctx context.Context, ticketID, labelID int64) error {
	ticketLabel := &ticket.TicketLabel{
		TicketID: ticketID,
		LabelID:  labelID,
	}
	return s.db.WithContext(ctx).Create(ticketLabel).Error
}

// RemoveLabel removes a label from a ticket
func (s *Service) RemoveLabel(ctx context.Context, ticketID, labelID int64) error {
	return s.db.WithContext(ctx).Where("ticket_id = ? AND label_id = ?", ticketID, labelID).Delete(&ticket.TicketLabel{}).Error
}

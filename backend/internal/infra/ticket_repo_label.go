package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// --- Label methods for ticketRepository ---

func (r *ticketRepository) GetLabelByOrgNameRepo(ctx context.Context, orgID int64, name string, repoID *int64) (*ticket.Label, error) {
	query := r.db.WithContext(ctx).Where("organization_id = ? AND name = ?", orgID, name)
	if repoID != nil {
		query = query.Where("repository_id = ?", *repoID)
	} else {
		query = query.Where("repository_id IS NULL")
	}

	var label ticket.Label
	if err := query.First(&label).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &label, nil
}

func (r *ticketRepository) CreateLabel(ctx context.Context, label *ticket.Label) error {
	return r.db.WithContext(ctx).Create(label).Error
}

func (r *ticketRepository) GetLabel(ctx context.Context, labelID int64) (*ticket.Label, error) {
	var label ticket.Label
	if err := r.db.WithContext(ctx).First(&label, labelID).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &label, nil
}

func (r *ticketRepository) ListLabels(ctx context.Context, orgID int64, repoID *int64) ([]*ticket.Label, error) {
	query := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if repoID != nil {
		query = query.Where("repository_id IS NULL OR repository_id = ?", *repoID)
	} else {
		query = query.Where("repository_id IS NULL")
	}

	var labels []*ticket.Label
	if err := query.Order("name ASC").Find(&labels).Error; err != nil {
		return nil, err
	}
	return labels, nil
}

func (r *ticketRepository) UpdateLabelFields(ctx context.Context, orgID, labelID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&ticket.Label{}).
		Where("id = ? AND organization_id = ?", labelID, orgID).
		Updates(updates).Error
}

func (r *ticketRepository) DeleteLabelAtomic(ctx context.Context, orgID, labelID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("label_id = ?", labelID).Delete(&ticket.TicketLabel{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND organization_id = ?", labelID, orgID).Delete(&ticket.Label{}).Error
	})
}

func (r *ticketRepository) GetTicketLabels(ctx context.Context, ticketID int64) ([]*ticket.Label, error) {
	var ticketLabels []ticket.TicketLabel
	if err := r.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Find(&ticketLabels).Error; err != nil {
		return nil, err
	}
	if len(ticketLabels) == 0 {
		return []*ticket.Label{}, nil
	}

	ids := make([]int64, len(ticketLabels))
	for i, tl := range ticketLabels {
		ids[i] = tl.LabelID
	}

	var labels []*ticket.Label
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&labels).Error; err != nil {
		return nil, err
	}
	return labels, nil
}

func (r *ticketRepository) AddTicketLabel(ctx context.Context, ticketID, labelID int64) error {
	return r.db.WithContext(ctx).Create(&ticket.TicketLabel{TicketID: ticketID, LabelID: labelID}).Error
}

func (r *ticketRepository) RemoveTicketLabel(ctx context.Context, ticketID, labelID int64) error {
	return r.db.WithContext(ctx).
		Where("ticket_id = ? AND label_id = ?", ticketID, labelID).
		Delete(&ticket.TicketLabel{}).Error
}

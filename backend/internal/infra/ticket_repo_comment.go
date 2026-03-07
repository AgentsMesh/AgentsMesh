package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// --- Comment methods for ticketRepository ---

func (r *ticketRepository) GetCommentByIDAndTicket(ctx context.Context, commentID, ticketID int64) (*ticket.Comment, error) {
	var c ticket.Comment
	if err := r.db.WithContext(ctx).
		Where("id = ? AND ticket_id = ?", commentID, ticketID).
		First(&c).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *ticketRepository) CreateComment(ctx context.Context, comment *ticket.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *ticketRepository) GetCommentWithUser(ctx context.Context, commentID int64) (*ticket.Comment, error) {
	var c ticket.Comment
	if err := r.db.WithContext(ctx).
		Preload("User").
		First(&c, commentID).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *ticketRepository) ListComments(ctx context.Context, ticketID int64, limit, offset int) ([]*ticket.Comment, int64, error) {
	var total int64
	r.db.WithContext(ctx).
		Model(&ticket.Comment{}).
		Where("ticket_id = ? AND parent_id IS NULL", ticketID).
		Count(&total)

	query := r.db.WithContext(ctx).
		Where("ticket_id = ? AND parent_id IS NULL", ticketID).
		Preload("User").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC").Preload("User")
		}).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var comments []*ticket.Comment
	if err := query.Find(&comments).Error; err != nil {
		return nil, 0, err
	}
	return comments, total, nil
}

func (r *ticketRepository) GetComment(ctx context.Context, commentID int64) (*ticket.Comment, error) {
	var c ticket.Comment
	if err := r.db.WithContext(ctx).First(&c, commentID).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *ticketRepository) UpdateComment(ctx context.Context, comment *ticket.Comment) error {
	return r.db.WithContext(ctx).
		Model(comment).
		Select("Content", "Mentions").
		Updates(comment).Error
}

func (r *ticketRepository) DeleteCommentAtomic(ctx context.Context, commentID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("parent_id = ?", commentID).Delete(&ticket.Comment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&ticket.Comment{}, commentID).Error
	})
}

func (r *ticketRepository) DeleteCommentsByTicket(ctx context.Context, ticketID int64) error {
	return r.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Delete(&ticket.Comment{}).Error
}

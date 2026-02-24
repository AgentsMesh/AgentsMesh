package ticket

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// ========== Ticket Comments ==========

var (
	ErrCommentNotFound     = errors.New("comment not found")
	ErrUnauthorizedComment = errors.New("unauthorized to modify this comment")
)

// CreateComment creates a new comment on a ticket
func (s *Service) CreateComment(ctx context.Context, ticketID, userID int64, content string, parentID *int64, mentions []ticket.CommentMention) (*ticket.Comment, error) {
	// Validate parent comment belongs to the same ticket
	if parentID != nil {
		var parent ticket.Comment
		if err := s.db.WithContext(ctx).
			Where("id = ? AND ticket_id = ?", *parentID, ticketID).
			First(&parent).Error; err != nil {
			return nil, ErrCommentNotFound
		}
	}

	comment := &ticket.Comment{
		TicketID: ticketID,
		UserID:   userID,
		Content:  content,
		ParentID: parentID,
		Mentions: mentions,
	}

	if err := s.db.WithContext(ctx).Create(comment).Error; err != nil {
		return nil, err
	}

	// Reload with user association
	if err := s.db.WithContext(ctx).
		Preload("User").
		First(comment, comment.ID).Error; err != nil {
		return nil, err
	}

	return comment, nil
}

// ListComments returns top-level comments for a ticket, ordered by created_at ASC.
// Includes user info and first-level replies with their user info.
// The total count reflects only top-level comments (excludes replies).
func (s *Service) ListComments(ctx context.Context, ticketID int64, limit, offset int) ([]*ticket.Comment, int64, error) {
	var total int64
	s.db.WithContext(ctx).
		Model(&ticket.Comment{}).
		Where("ticket_id = ? AND parent_id IS NULL", ticketID).
		Count(&total)

	var comments []*ticket.Comment
	query := s.db.WithContext(ctx).
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

	if err := query.Find(&comments).Error; err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// UpdateComment updates a comment (only the author can update).
// ticketID is used to verify the comment belongs to the specified ticket.
func (s *Service) UpdateComment(ctx context.Context, ticketID, commentID, userID int64, content string, mentions []ticket.CommentMention) (*ticket.Comment, error) {
	var comment ticket.Comment
	if err := s.db.WithContext(ctx).First(&comment, commentID).Error; err != nil {
		return nil, ErrCommentNotFound
	}

	if comment.TicketID != ticketID {
		return nil, ErrCommentNotFound
	}

	if comment.UserID != userID {
		return nil, ErrUnauthorizedComment
	}

	updates := map[string]interface{}{
		"content":  content,
		"mentions": mentions,
	}

	if err := s.db.WithContext(ctx).Model(&comment).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload with user association
	if err := s.db.WithContext(ctx).
		Preload("User").
		First(&comment, commentID).Error; err != nil {
		return nil, err
	}

	return &comment, nil
}

// DeleteComment deletes a comment and its replies (only the author can delete).
// ticketID is used to verify the comment belongs to the specified ticket.
// Since no DB foreign keys are used, cascade is handled at application level.
func (s *Service) DeleteComment(ctx context.Context, ticketID, commentID, userID int64) error {
	var comment ticket.Comment
	if err := s.db.WithContext(ctx).First(&comment, commentID).Error; err != nil {
		return ErrCommentNotFound
	}

	if comment.TicketID != ticketID {
		return ErrCommentNotFound
	}

	if comment.UserID != userID {
		return ErrUnauthorizedComment
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete replies first (application-level cascade)
		if err := tx.Where("parent_id = ?", commentID).Delete(&ticket.Comment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&comment).Error
	})
}

// DeleteCommentsByTicket deletes all comments for a ticket.
// Call this before deleting a ticket to maintain referential integrity
// without DB-level foreign keys.
func (s *Service) DeleteCommentsByTicket(ctx context.Context, ticketID int64) error {
	return s.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Delete(&ticket.Comment{}).Error
}

package ticket

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// ========== Ticket Comments ==========

var (
	ErrCommentNotFound     = errors.New("comment not found")
	ErrUnauthorizedComment = errors.New("unauthorized to modify this comment")
)

// CreateComment creates a new comment on a ticket.
func (s *Service) CreateComment(ctx context.Context, ticketID, userID int64, content string, parentID *int64, mentions []ticket.CommentMention) (*ticket.Comment, error) {
	// Validate parent comment belongs to the same ticket
	if parentID != nil {
		parent, err := s.repo.GetCommentByIDAndTicket(ctx, *parentID, ticketID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
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

	if err := s.repo.CreateComment(ctx, comment); err != nil {
		slog.Error("failed to create comment", "ticket_id", ticketID, "user_id", userID, "error", err)
		return nil, err
	}

	slog.Info("comment created", "comment_id", comment.ID, "ticket_id", ticketID, "user_id", userID)
	// Reload with user association
	loaded, err := s.repo.GetCommentWithUser(ctx, comment.ID)
	if err != nil {
		return nil, err
	}
	return loaded, nil
}

// ListComments returns top-level comments for a ticket, ordered by created_at ASC.
func (s *Service) ListComments(ctx context.Context, ticketID int64, limit, offset int) ([]*ticket.Comment, int64, error) {
	return s.repo.ListComments(ctx, ticketID, limit, offset)
}

// UpdateComment updates a comment (only the author can update).
func (s *Service) UpdateComment(ctx context.Context, ticketID, commentID, userID int64, content string, mentions []ticket.CommentMention) (*ticket.Comment, error) {
	comment, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, ErrCommentNotFound
	}
	if comment.TicketID != ticketID {
		return nil, ErrCommentNotFound
	}
	if comment.UserID != userID {
		return nil, ErrUnauthorizedComment
	}

	comment.Content = content
	comment.Mentions = mentions

	if err := s.repo.UpdateComment(ctx, comment); err != nil {
		slog.Error("failed to update comment", "comment_id", commentID, "ticket_id", ticketID, "error", err)
		return nil, err
	}

	slog.Info("comment updated", "comment_id", commentID, "ticket_id", ticketID, "user_id", userID)
	loaded, err := s.repo.GetCommentWithUser(ctx, commentID)
	if err != nil {
		return nil, err
	}
	return loaded, nil
}

// DeleteComment deletes a comment and its replies (only the author can delete).
func (s *Service) DeleteComment(ctx context.Context, ticketID, commentID, userID int64) error {
	comment, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return ErrCommentNotFound
	}
	if comment.TicketID != ticketID {
		return ErrCommentNotFound
	}
	if comment.UserID != userID {
		return ErrUnauthorizedComment
	}

	return s.repo.DeleteCommentAtomic(ctx, commentID)
}

// DeleteCommentsByTicket deletes all comments for a ticket.
func (s *Service) DeleteCommentsByTicket(ctx context.Context, ticketID int64) error {
	if err := s.repo.DeleteCommentsByTicket(ctx, ticketID); err != nil {
		slog.Error("failed to delete comments by ticket", "ticket_id", ticketID, "error", err)
		return err
	}
	slog.Info("comments deleted for ticket", "ticket_id", ticketID)
	return nil
}

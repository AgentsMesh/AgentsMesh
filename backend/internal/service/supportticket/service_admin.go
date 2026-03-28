package supportticket

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
)

// --- Admin-side methods ---

// AdminList returns paginated tickets for admin (all users)
func (s *Service) AdminList(ctx context.Context, query *AdminListQuery) (*ListResponse, error) {
	page, pageSize := normalizePagination(query.Page, query.PageSize)
	offset := (page - 1) * pageSize

	tickets, total, err := s.repo.AdminList(ctx, query.Search, query.Status, query.Category, query.Priority, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}

	return &ListResponse{
		Data:       tickets,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	}, nil
}

// AdminGetByID returns a ticket by ID (no ownership check)
func (s *Service) AdminGetByID(ctx context.Context, id int64) (*supportticket.SupportTicket, error) {
	ticket, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, ErrTicketNotFound
	}
	return ticket, nil
}

// AdminListMessages returns all messages for a ticket (admin, no ownership check)
func (s *Service) AdminListMessages(ctx context.Context, ticketID int64) ([]supportticket.SupportTicketMessage, error) {
	return s.repo.ListMessagesByTicketID(ctx, ticketID)
}

// AdminAddReply adds an admin reply to a ticket
func (s *Service) AdminAddReply(ctx context.Context, ticketID, adminUserID int64, req *AddMessageRequest) (*supportticket.SupportTicketMessage, error) {
	if _, err := s.AdminGetByID(ctx, ticketID); err != nil {
		return nil, err
	}

	msg := &supportticket.SupportTicketMessage{
		TicketID:     ticketID,
		UserID:       adminUserID,
		Content:      req.Content,
		IsAdminReply: true,
	}

	if err := s.repo.AddAdminReplyAndTransition(ctx, msg, ticketID); err != nil {
		slog.Error("failed to add admin reply", "ticket_id", ticketID, "admin_user_id", adminUserID, "error", err)
		return nil, fmt.Errorf("failed to create admin reply: %w", err)
	}
	slog.Info("admin reply added", "ticket_id", ticketID, "admin_user_id", adminUserID)
	return msg, nil
}

// AdminUpdateStatus updates the status of a ticket with transition validation
func (s *Service) AdminUpdateStatus(ctx context.Context, ticketID int64, status string) error {
	if !supportticket.ValidStatuses[status] {
		return ErrInvalidStatus
	}

	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}
	if ticket == nil {
		return ErrTicketNotFound
	}

	if ticket.Status == status {
		return nil
	}

	allowed, ok := supportticket.ValidTransitions[ticket.Status]
	if !ok || !allowed[status] {
		return ErrInvalidTransition
	}

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if status == supportticket.StatusResolved && ticket.ResolvedAt == nil {
		now := time.Now()
		updates["resolved_at"] = &now
	}

	rowsAffected, err := s.repo.UpdateStatus(ctx, ticketID, ticket.Status, status, updates)
	if err != nil {
		return fmt.Errorf("failed to update ticket status: %w", err)
	}
	if rowsAffected == 0 {
		return ErrInvalidTransition
	}
	slog.Info("support ticket status updated", "ticket_id", ticketID, "old_status", ticket.Status, "new_status", status)
	return nil
}

// AdminAssign assigns a ticket to an admin
func (s *Service) AdminAssign(ctx context.Context, ticketID, adminUserID int64) error {
	rowsAffected, err := s.repo.AssignAdmin(ctx, ticketID, adminUserID)
	if err != nil {
		slog.Error("failed to assign support ticket", "ticket_id", ticketID, "admin_user_id", adminUserID, "error", err)
		return fmt.Errorf("failed to assign ticket: %w", err)
	}
	if rowsAffected == 0 {
		return ErrTicketNotFound
	}
	slog.Info("support ticket assigned", "ticket_id", ticketID, "admin_user_id", adminUserID)
	return nil
}

// AdminGetStats returns ticket statistics
func (s *Service) AdminGetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{}

	var err error
	if stats.Total, err = s.repo.CountByStatus(ctx, ""); err != nil {
		return nil, err
	}
	if stats.Open, err = s.repo.CountByStatus(ctx, supportticket.StatusOpen); err != nil {
		return nil, err
	}
	if stats.InProgress, err = s.repo.CountByStatus(ctx, supportticket.StatusInProgress); err != nil {
		return nil, err
	}
	if stats.Resolved, err = s.repo.CountByStatus(ctx, supportticket.StatusResolved); err != nil {
		return nil, err
	}
	if stats.Closed, err = s.repo.CountByStatus(ctx, supportticket.StatusClosed); err != nil {
		return nil, err
	}

	return stats, nil
}

// AdminGetAttachmentURL returns a presigned URL for downloading an attachment (admin, no ownership check)
func (s *Service) AdminGetAttachmentURL(ctx context.Context, attachmentID int64) (string, error) {
	if s.storage == nil {
		return "", ErrStorageError
	}

	attachment, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return "", err
	}
	if attachment == nil {
		return "", ErrAttachmentNotFound
	}

	return s.storage.GetURL(ctx, attachment.StorageKey, 1*time.Hour)
}

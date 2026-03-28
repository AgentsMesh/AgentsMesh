package ticket

import (
	"context"
	"log/slog"
	"time"

	domainTicket "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// UpdateTicket updates a ticket.
func (s *Service) UpdateTicket(ctx context.Context, ticketID int64, updates map[string]interface{}) (*domainTicket.Ticket, error) {
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	previousStatus := oldTicket.Status

	if len(updates) > 0 {
		if err := s.repo.UpdateFields(ctx, ticketID, updates); err != nil {
			slog.Error("failed to update ticket fields", "ticket_id", ticketID, "org_id", oldTicket.OrganizationID, "error", err)
			return nil, err
		}
	}

	updatedTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	if newStatus, ok := updates["status"].(string); ok && newStatus != previousStatus {
		slog.Info("ticket status changed", "ticket_id", ticketID, "slug", updatedTicket.Slug, "from", previousStatus, "to", newStatus)
		s.publishEvent(ctx, TicketEventStatusChanged, oldTicket.OrganizationID, updatedTicket.Slug, updatedTicket.Status, previousStatus)
	} else {
		s.publishEvent(ctx, TicketEventUpdated, oldTicket.OrganizationID, updatedTicket.Slug, updatedTicket.Status, previousStatus)
	}

	return updatedTicket, nil
}

// UpdateStatus updates a ticket's status.
func (s *Service) UpdateStatus(ctx context.Context, ticketID int64, status string) error {
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	previousStatus := oldTicket.Status

	updates := map[string]interface{}{"status": status}
	now := time.Now()
	switch status {
	case domainTicket.TicketStatusInProgress:
		updates["started_at"] = now
	case domainTicket.TicketStatusDone:
		updates["completed_at"] = now
	}

	if err := s.repo.UpdateFields(ctx, ticketID, updates); err != nil {
		slog.Error("failed to update ticket status", "ticket_id", ticketID, "org_id", oldTicket.OrganizationID, "status", status, "error", err)
		return err
	}

	slog.Info("ticket status updated", "ticket_id", ticketID, "slug", oldTicket.Slug, "from", previousStatus, "to", status)
	s.publishEvent(ctx, TicketEventStatusChanged, oldTicket.OrganizationID, oldTicket.Slug, status, previousStatus)
	return nil
}

// DeleteTicket deletes a ticket and its associated comments within a transaction.
func (s *Service) DeleteTicket(ctx context.Context, ticketID int64) error {
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteTicketAtomic(ctx, ticketID); err != nil {
		slog.Error("failed to delete ticket", "ticket_id", ticketID, "org_id", oldTicket.OrganizationID, "slug", oldTicket.Slug, "error", err)
		return err
	}

	slog.Info("ticket deleted", "ticket_id", ticketID, "slug", oldTicket.Slug, "org_id", oldTicket.OrganizationID)
	s.publishEvent(ctx, TicketEventDeleted, oldTicket.OrganizationID, oldTicket.Slug, "deleted", oldTicket.Status)
	return nil
}

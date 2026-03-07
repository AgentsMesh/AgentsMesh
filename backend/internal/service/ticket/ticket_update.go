package ticket

import (
	"context"
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
			return nil, err
		}
	}

	updatedTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	if newStatus, ok := updates["status"].(string); ok && newStatus != previousStatus {
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
		return err
	}

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
		return err
	}

	s.publishEvent(ctx, TicketEventDeleted, oldTicket.OrganizationID, oldTicket.Slug, "deleted", oldTicket.Status)
	return nil
}

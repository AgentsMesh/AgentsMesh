package ticket

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// UpdateTicket updates a ticket
func (s *Service) UpdateTicket(ctx context.Context, ticketID int64, updates map[string]interface{}) (*ticket.Ticket, error) {
	// Get the ticket before update to capture previous status
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	previousStatus := oldTicket.Status

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&ticket.Ticket{}).Where("id = ?", ticketID).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	updatedTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	// Publish appropriate event based on what changed
	if newStatus, ok := updates["status"].(string); ok && newStatus != previousStatus {
		s.publishEvent(ctx, TicketEventStatusChanged, oldTicket.OrganizationID, updatedTicket.Slug, updatedTicket.Status, previousStatus)
	} else {
		s.publishEvent(ctx, TicketEventUpdated, oldTicket.OrganizationID, updatedTicket.Slug, updatedTicket.Status, previousStatus)
	}

	return updatedTicket, nil
}

// UpdateStatus updates a ticket's status
func (s *Service) UpdateStatus(ctx context.Context, ticketID int64, status string) error {
	// Get the ticket before update to capture previous status and org ID
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	previousStatus := oldTicket.Status

	updates := map[string]interface{}{
		"status": status,
	}

	now := time.Now()
	switch status {
	case ticket.TicketStatusInProgress:
		updates["started_at"] = now
	case ticket.TicketStatusDone:
		updates["completed_at"] = now
	}

	if err := s.db.WithContext(ctx).Model(&ticket.Ticket{}).Where("id = ?", ticketID).Updates(updates).Error; err != nil {
		return err
	}

	// Publish status changed event (for kanban board real-time updates)
	s.publishEvent(ctx, TicketEventStatusChanged, oldTicket.OrganizationID, oldTicket.Slug, status, previousStatus)

	return nil
}

// DeleteTicket deletes a ticket and its associated comments within a transaction.
func (s *Service) DeleteTicket(ctx context.Context, ticketID int64) error {
	// Get the ticket before deletion to capture info for event
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clean up comments first (application-level cascade, no DB foreign keys)
		if err := tx.Where("ticket_id = ?", ticketID).Delete(&ticket.Comment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&ticket.Ticket{}, ticketID).Error
	}); err != nil {
		return err
	}

	// Publish ticket deleted event (outside transaction — fire-and-forget)
	s.publishEvent(ctx, TicketEventDeleted, oldTicket.OrganizationID, oldTicket.Slug, "deleted", oldTicket.Status)

	return nil
}

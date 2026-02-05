package ticket

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// CreateTicket creates a new ticket
func (s *Service) CreateTicket(ctx context.Context, req *CreateTicketRequest) (*ticket.Ticket, error) {
	var number int
	var identifier string

	// Check if repository has a ticket_prefix
	var prefix sql.NullString
	if req.RepositoryID != nil {
		s.db.WithContext(ctx).Table("repositories").
			Where("id = ?", *req.RepositoryID).
			Select("ticket_prefix").
			Scan(&prefix)
	}

	if prefix.Valid && prefix.String != "" {
		// Repository has a prefix: generate number scoped to repository
		var maxNumber int
		s.db.WithContext(ctx).Model(&ticket.Ticket{}).
			Where("repository_id = ?", req.RepositoryID).
			Select("COALESCE(MAX(number), 0)").
			Scan(&maxNumber)
		number = maxNumber + 1
		identifier = fmt.Sprintf("%s-%d", prefix.String, number)
	} else {
		// No prefix: generate number scoped to organization with TICKET- prefix
		var maxNumber int
		s.db.WithContext(ctx).Model(&ticket.Ticket{}).
			Where("organization_id = ? AND identifier LIKE 'TICKET-%'", req.OrganizationID).
			Select("COALESCE(MAX(number), 0)").
			Scan(&maxNumber)
		number = maxNumber + 1
		identifier = fmt.Sprintf("TICKET-%d", number)
	}

	status := req.Status
	if status == "" {
		status = ticket.TicketStatusBacklog
	}

	t := &ticket.Ticket{
		OrganizationID: req.OrganizationID,
		Number:         number,
		Identifier:     identifier,
		Type:           req.Type,
		Title:          req.Title,
		Description:    req.Description,
		Content:        req.Content,
		Status:         status,
		Priority:       req.Priority,
		DueDate:        req.DueDate,
		RepositoryID:   req.RepositoryID,
		ReporterID:     req.ReporterID,
		ParentTicketID: req.ParentTicketID,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(t).Error; err != nil {
			return err
		}

		// Add assignees
		for _, userID := range req.AssigneeIDs {
			assignee := &ticket.Assignee{
				TicketID: t.ID,
				UserID:   userID,
			}
			if err := tx.Create(assignee).Error; err != nil {
				return err
			}
		}

		// Add labels by ID
		for _, labelID := range req.LabelIDs {
			ticketLabel := &ticket.TicketLabel{
				TicketID: t.ID,
				LabelID:  labelID,
			}
			if err := tx.Create(ticketLabel).Error; err != nil {
				return err
			}
		}

		// Add labels by name (if provided)
		for _, labelName := range req.Labels {
			var label ticket.Label
			if err := tx.Where("organization_id = ? AND name = ?", req.OrganizationID, labelName).First(&label).Error; err != nil {
				continue // Skip if label not found
			}
			ticketLabel := &ticket.TicketLabel{
				TicketID: t.ID,
				LabelID:  label.ID,
			}
			if err := tx.Create(ticketLabel).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Get the created ticket with full details
	createdTicket, err := s.GetTicket(ctx, t.ID)
	if err != nil {
		return nil, err
	}

	// Publish ticket created event (Service layer - Information Expert)
	s.publishEvent(ctx, TicketEventCreated, req.OrganizationID, createdTicket.Identifier, createdTicket.Status, "")

	return createdTicket, nil
}

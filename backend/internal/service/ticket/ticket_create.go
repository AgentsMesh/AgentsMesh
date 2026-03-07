package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// CreateTicket creates a new ticket.
func (s *Service) CreateTicket(ctx context.Context, req *CreateTicketRequest) (*ticket.Ticket, error) {
	// Determine ticket prefix from repository (cross-domain)
	ticketPrefix := "TICKET"
	if req.RepositoryID != nil {
		prefix, err := s.repo.GetRepoTicketPrefix(ctx, *req.RepositoryID)
		if err == nil && prefix != "" {
			ticketPrefix = prefix
		}
	}

	status := req.Status
	if status == "" {
		status = ticket.TicketStatusBacklog
	}

	t := &ticket.Ticket{
		OrganizationID: req.OrganizationID,
		Title:          req.Title,
		Content:        req.Content,
		Status:         status,
		Priority:       req.Priority,
		DueDate:        req.DueDate,
		RepositoryID:   req.RepositoryID,
		ReporterID:     req.ReporterID,
		ParentTicketID: req.ParentTicketID,
	}

	params := &ticket.CreateTicketParams{
		Ticket:      t,
		Prefix:      ticketPrefix,
		AssigneeIDs: req.AssigneeIDs,
		LabelIDs:    req.LabelIDs,
		LabelNames:  req.Labels,
	}

	if err := s.repo.CreateTicketAtomic(ctx, params); err != nil {
		return nil, err
	}

	// Get the created ticket with full details
	createdTicket, err := s.GetTicket(ctx, t.ID)
	if err != nil {
		return nil, err
	}

	s.publishEvent(ctx, TicketEventCreated, req.OrganizationID, createdTicket.Slug, createdTicket.Status, "")
	return createdTicket, nil
}

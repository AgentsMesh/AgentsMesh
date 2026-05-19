package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/google/uuid"
)

func (s *Service) CreateTicket(ctx context.Context, req *CreateTicketRequest) (*ticket.Ticket, error) {
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

	var (
		contentBlockID *uuid.UUID
		inlineContent  *string
	)
	if s.blockstore != nil && req.Content != nil {
		id, err := s.writeContentBlock(ctx, req.OrganizationID, req.ReporterID, *req.Content)
		if err != nil {
			return nil, err
		}
		if id != uuid.Nil {
			contentBlockID = &id
		}
	} else {
		inlineContent = req.Content
	}

	t := &ticket.Ticket{
		OrganizationID: req.OrganizationID,
		Title:          req.Title,
		Content:        inlineContent,
		ContentBlockID: contentBlockID,
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

	createdTicket, err := s.GetTicket(ctx, t.ID)
	if err != nil {
		return nil, err
	}

	s.publishEvent(ctx, TicketEventCreated, req.OrganizationID, createdTicket.Slug, createdTicket.Status, "")
	return createdTicket, nil
}

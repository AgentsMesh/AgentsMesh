package ticket

import (
	"context"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// GetTicket returns a ticket by ID.
func (s *Service) GetTicket(ctx context.Context, ticketID int64) (*ticket.Ticket, error) {
	t, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTicketNotFound
	}
	return t, nil
}

// GetTicketBySlug returns a ticket by slug scoped to an organization.
func (s *Service) GetTicketBySlug(ctx context.Context, organizationID int64, slug string) (*ticket.Ticket, error) {
	t, err := s.repo.GetByOrgAndSlug(ctx, organizationID, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTicketNotFound
	}
	return t, nil
}

// GetTicketByIDOrSlug returns a ticket by numeric ID or string slug,
// scoped to an organization. It first tries slug lookup; if the input is
// a pure numeric string, it falls back to primary-key lookup with org validation.
func (s *Service) GetTicketByIDOrSlug(ctx context.Context, organizationID int64, idOrSlug string) (*ticket.Ticket, error) {
	t, err := s.GetTicketBySlug(ctx, organizationID, idOrSlug)
	if err == nil {
		return t, nil
	}

	if numericID, parseErr := strconv.ParseInt(idOrSlug, 10, 64); parseErr == nil {
		t, err = s.GetTicket(ctx, numericID)
		if err != nil {
			return nil, ErrTicketNotFound
		}
		if t.OrganizationID != organizationID {
			return nil, ErrTicketNotFound
		}
		return t, nil
	}

	return nil, ErrTicketNotFound
}

// ListTickets returns tickets based on filters.
func (s *Service) ListTickets(ctx context.Context, filter *ticket.TicketListFilter) ([]*ticket.Ticket, int64, error) {
	return s.repo.List(ctx, filter)
}

package ticket

import (
	"context"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

func (s *Service) GetTicket(ctx context.Context, ticketID int64) (*ticket.Ticket, error) {
	t, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTicketNotFound
	}
	s.hydrateContentFromBlock(ctx, t)
	return t, nil
}

func (s *Service) GetTicketBySlug(ctx context.Context, organizationID int64, slug string) (*ticket.Ticket, error) {
	t, err := s.repo.GetByOrgAndSlug(ctx, organizationID, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTicketNotFound
	}
	s.hydrateContentFromBlock(ctx, t)
	return t, nil
}

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

func (s *Service) ListTickets(ctx context.Context, filter *ticket.TicketListFilter) ([]*ticket.Ticket, int64, error) {
	return s.repo.List(ctx, filter)
}

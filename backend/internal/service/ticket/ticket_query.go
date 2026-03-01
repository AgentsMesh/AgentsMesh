package ticket

import (
	"context"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/ticket"
)

// GetTicket returns a ticket by ID
func (s *Service) GetTicket(ctx context.Context, ticketID int64) (*ticket.Ticket, error) {
	var t ticket.Ticket
	if err := s.db.WithContext(ctx).
		Preload("Assignees.User").
		Preload("Labels").
		Preload("MergeRequests").
		Preload("SubTickets").
		First(&t, ticketID).Error; err != nil {
		return nil, ErrTicketNotFound
	}
	return &t, nil
}

// GetTicketBySlug returns a ticket by slug scoped to an organization.
// Since slug uniqueness is per-organization, organizationID is required.
func (s *Service) GetTicketBySlug(ctx context.Context, organizationID int64, slug string) (*ticket.Ticket, error) {
	var t ticket.Ticket
	if err := s.db.WithContext(ctx).
		Preload("Assignees.User").
		Preload("Labels").
		Preload("MergeRequests").
		Preload("SubTickets").
		Where("organization_id = ? AND slug = ?", organizationID, slug).
		First(&t).Error; err != nil {
		return nil, ErrTicketNotFound
	}
	return &t, nil
}

// GetTicketByIDOrSlug returns a ticket by numeric ID or string slug,
// scoped to an organization. It first tries slug lookup; if the input is
// a pure numeric string, it falls back to primary-key lookup with org validation.
func (s *Service) GetTicketByIDOrSlug(ctx context.Context, organizationID int64, idOrSlug string) (*ticket.Ticket, error) {
	// Try slug lookup first (covers both "AM-123" and numeric strings that happen to match a slug)
	t, err := s.GetTicketBySlug(ctx, organizationID, idOrSlug)
	if err == nil {
		return t, nil
	}

	// If the input is a numeric string, fall back to primary-key lookup
	if numericID, parseErr := strconv.ParseInt(idOrSlug, 10, 64); parseErr == nil {
		t, err = s.GetTicket(ctx, numericID)
		if err != nil {
			return nil, ErrTicketNotFound
		}
		// Verify organization ownership
		if t.OrganizationID != organizationID {
			return nil, ErrTicketNotFound
		}
		return t, nil
	}

	return nil, ErrTicketNotFound
}

// ListTickets returns tickets based on filters
func (s *Service) ListTickets(ctx context.Context, filter *ListTicketsFilter) ([]*ticket.Ticket, int64, error) {
	query := s.db.WithContext(ctx).Model(&ticket.Ticket{}).Where("organization_id = ?", filter.OrganizationID)

	if filter.RepositoryID != nil {
		query = query.Where("repository_id = ?", *filter.RepositoryID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Priority != "" {
		query = query.Where("priority = ?", filter.Priority)
	}
	if filter.ReporterID != nil {
		query = query.Where("reporter_id = ?", *filter.ReporterID)
	}
	if filter.ParentTicketID != nil {
		query = query.Where("parent_ticket_id = ?", *filter.ParentTicketID)
	}
	if filter.Query != "" {
		query = query.Where("title ILIKE ? OR slug ILIKE ?", "%"+filter.Query+"%", "%"+filter.Query+"%")
	}
	if filter.AssigneeID != nil {
		query = query.Joins("JOIN ticket_assignees ON ticket_assignees.ticket_id = tickets.id").
			Where("ticket_assignees.user_id = ?", *filter.AssigneeID)
	}
	if len(filter.LabelIDs) > 0 {
		query = query.Joins("JOIN ticket_labels ON ticket_labels.ticket_id = tickets.id").
			Where("ticket_labels.label_id IN ?", filter.LabelIDs)
	}

	var total int64
	query.Count(&total)

	var tickets []*ticket.Ticket
	if err := query.
		Preload("Assignees.User").
		Preload("Labels").
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&tickets).Error; err != nil {
		return nil, 0, err
	}

	return tickets, total, nil
}

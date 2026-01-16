package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// ========== Board View (Kanban) ==========

// GetBoard returns a kanban board view of tickets
func (s *Service) GetBoard(ctx context.Context, filter *ListTicketsFilter) (*ticket.Board, error) {
	// Define board columns (ordered)
	columnStatuses := []string{
		ticket.TicketStatusBacklog,
		ticket.TicketStatusTodo,
		ticket.TicketStatusInProgress,
		ticket.TicketStatusInReview,
		ticket.TicketStatusDone,
	}

	board := &ticket.Board{
		Columns: make([]ticket.BoardColumn, len(columnStatuses)),
	}

	for i, status := range columnStatuses {
		filter.Status = status
		tickets, count, err := s.ListTickets(ctx, filter)
		if err != nil {
			return nil, err
		}

		column := ticket.BoardColumn{
			Status:  status,
			Count:   int(count),
			Tickets: make([]ticket.Ticket, len(tickets)),
		}
		for j, t := range tickets {
			column.Tickets[j] = *t
		}
		board.Columns[i] = column
	}

	return board, nil
}

// GetActiveTickets returns active (non-completed) tickets
func (s *Service) GetActiveTickets(ctx context.Context, orgID int64, repoID *int64, limit int) ([]*ticket.Ticket, error) {
	query := s.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Where("status NOT IN ?", []string{ticket.TicketStatusDone, ticket.TicketStatusCancelled})

	if repoID != nil {
		query = query.Where("repository_id = ?", *repoID)
	}

	var tickets []*ticket.Ticket
	if err := query.
		Preload("Assignees").
		Preload("Labels").
		Order("updated_at DESC").
		Limit(limit).
		Find(&tickets).Error; err != nil {
		return nil, err
	}

	return tickets, nil
}

// GetChildTickets returns child tickets for a parent ticket
func (s *Service) GetChildTickets(ctx context.Context, parentTicketID int64) ([]*ticket.Ticket, error) {
	var tickets []*ticket.Ticket
	if err := s.db.WithContext(ctx).
		Preload("Assignees").
		Preload("Labels").
		Where("parent_ticket_id = ?", parentTicketID).
		Order("created_at ASC").
		Find(&tickets).Error; err != nil {
		return nil, err
	}
	return tickets, nil
}

// GetSubTicketCounts returns sub-ticket counts for multiple parent tickets
func (s *Service) GetSubTicketCounts(ctx context.Context, parentTicketIDs []int64) (map[int64]map[string]int64, error) {
	type countResult struct {
		ParentTicketID int64
		Status         string
		Count          int64
	}

	var results []countResult
	if err := s.db.WithContext(ctx).
		Model(&ticket.Ticket{}).
		Select("parent_ticket_id, status, COUNT(*) as count").
		Where("parent_ticket_id IN ?", parentTicketIDs).
		Group("parent_ticket_id, status").
		Find(&results).Error; err != nil {
		return nil, err
	}

	counts := make(map[int64]map[string]int64)
	for _, r := range results {
		if counts[r.ParentTicketID] == nil {
			counts[r.ParentTicketID] = make(map[string]int64)
		}
		counts[r.ParentTicketID][r.Status] = r.Count
	}

	return counts, nil
}

// ========== Statistics ==========

// GetTicketStats returns ticket statistics for a repository
func (s *Service) GetTicketStats(ctx context.Context, orgID int64, repoID *int64) (map[string]int64, error) {
	stats := make(map[string]int64)

	query := s.db.WithContext(ctx).Model(&ticket.Ticket{}).Where("organization_id = ?", orgID)
	if repoID != nil {
		query = query.Where("repository_id = ?", *repoID)
	}

	statuses := []string{
		ticket.TicketStatusBacklog,
		ticket.TicketStatusTodo,
		ticket.TicketStatusInProgress,
		ticket.TicketStatusInReview,
		ticket.TicketStatusDone,
		ticket.TicketStatusCancelled,
	}

	for _, status := range statuses {
		var count int64
		query.Where("status = ?", status).Count(&count)
		stats[status] = count
	}

	return stats, nil
}

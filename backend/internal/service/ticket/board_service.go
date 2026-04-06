package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// ========== Board View (Kanban) ==========

// GetBoard returns a kanban board view of tickets.
func (s *Service) GetBoard(ctx context.Context, filter *ticket.TicketListFilter) (*ticket.Board, error) {
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

// GetActiveTickets returns active (non-completed) tickets.
func (s *Service) GetActiveTickets(ctx context.Context, orgID int64, repoID *int64, limit int) ([]*ticket.Ticket, error) {
	return s.repo.GetActiveTickets(ctx, orgID, repoID, limit)
}

// GetChildTickets returns child tickets for a parent ticket.
func (s *Service) GetChildTickets(ctx context.Context, parentTicketID int64) ([]*ticket.Ticket, error) {
	return s.repo.GetChildTickets(ctx, parentTicketID)
}

// GetSubTicketCounts returns sub-ticket counts for multiple parent tickets.
func (s *Service) GetSubTicketCounts(ctx context.Context, parentTicketIDs []int64) (map[int64]map[string]int64, error) {
	return s.repo.GetSubTicketCounts(ctx, parentTicketIDs)
}

// ========== Statistics ==========

// GetTicketStats returns ticket statistics for a repository.
func (s *Service) GetTicketStats(ctx context.Context, orgID int64, repoID *int64) (map[string]int64, error) {
	return s.repo.GetTicketStats(ctx, orgID, repoID)
}

package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

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
		colFilter := *filter
		colFilter.Status = status
		tickets, count, err := s.ListTickets(ctx, &colFilter)
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

	priorityCounts, err := s.repo.GetPriorityCounts(ctx, filter.OrganizationID, filter.RepositoryID)
	if err != nil {
		return nil, err
	}
	board.PriorityCounts = priorityCounts

	return board, nil
}

func (s *Service) GetActiveTickets(ctx context.Context, orgID int64, repoID *int64, limit int) ([]*ticket.Ticket, error) {
	return s.repo.GetActiveTickets(ctx, orgID, repoID, limit)
}

func (s *Service) GetChildTickets(ctx context.Context, parentTicketID int64) ([]*ticket.Ticket, error) {
	return s.repo.GetChildTickets(ctx, parentTicketID)
}

func (s *Service) GetSubTicketCounts(ctx context.Context, parentTicketIDs []int64) (map[int64]map[string]int64, error) {
	return s.repo.GetSubTicketCounts(ctx, parentTicketIDs)
}

func (s *Service) GetTicketStats(ctx context.Context, orgID int64, repoID *int64) (map[string]int64, error) {
	return s.repo.GetTicketStats(ctx, orgID, repoID)
}

package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// --- Board / Statistics methods for ticketRepository ---

func (r *ticketRepository) GetActiveTickets(ctx context.Context, orgID int64, repoID *int64, limit int) ([]*ticket.Ticket, error) {
	query := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Where("status != ?", ticket.TicketStatusDone)

	if repoID != nil {
		query = query.Where("repository_id = ?", *repoID)
	}

	var tickets []*ticket.Ticket
	if err := query.
		Preload("Assignees.User").
		Preload("Labels").
		Order("updated_at DESC").
		Limit(limit).
		Find(&tickets).Error; err != nil {
		return nil, err
	}
	return tickets, nil
}

func (r *ticketRepository) GetChildTickets(ctx context.Context, parentTicketID int64) ([]*ticket.Ticket, error) {
	var tickets []*ticket.Ticket
	if err := r.db.WithContext(ctx).
		Preload("Assignees.User").
		Preload("Labels").
		Where("parent_ticket_id = ?", parentTicketID).
		Order("created_at ASC").
		Find(&tickets).Error; err != nil {
		return nil, err
	}
	return tickets, nil
}

func (r *ticketRepository) GetSubTicketCounts(ctx context.Context, parentIDs []int64) (map[int64]map[string]int64, error) {
	type countResult struct {
		ParentTicketID int64
		Status         string
		Count          int64
	}

	var results []countResult
	if err := r.db.WithContext(ctx).
		Model(&ticket.Ticket{}).
		Select("parent_ticket_id, status, COUNT(*) as count").
		Where("parent_ticket_id IN ?", parentIDs).
		Group("parent_ticket_id, status").
		Find(&results).Error; err != nil {
		return nil, err
	}

	counts := make(map[int64]map[string]int64)
	for _, row := range results {
		if counts[row.ParentTicketID] == nil {
			counts[row.ParentTicketID] = make(map[string]int64)
		}
		counts[row.ParentTicketID][row.Status] = row.Count
	}
	return counts, nil
}

func (r *ticketRepository) GetTicketStats(ctx context.Context, orgID int64, repoID *int64) (map[string]int64, error) {
	base := r.db.WithContext(ctx).Model(&ticket.Ticket{}).Where("organization_id = ?", orgID)
	if repoID != nil {
		base = base.Where("repository_id = ?", *repoID)
	}

	statuses := []string{
		ticket.TicketStatusBacklog,
		ticket.TicketStatusTodo,
		ticket.TicketStatusInProgress,
		ticket.TicketStatusInReview,
		ticket.TicketStatusDone,
	}

	stats := make(map[string]int64, len(statuses))
	for _, status := range statuses {
		var count int64
		base.Where("status = ?", status).Count(&count)
		stats[status] = count
	}
	return stats, nil
}

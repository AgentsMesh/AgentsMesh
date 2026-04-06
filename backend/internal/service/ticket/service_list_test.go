package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

func TestListTickets(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create multiple tickets
	for i := 1; i <= 5; i++ {
		req := &CreateTicketRequest{
			OrganizationID: 1,
			ReporterID:     1,
			Title:          "Test Ticket",
			Priority:       "medium",
		}
		service.CreateTicket(ctx, req)
	}

	// List tickets
	filter := &ticket.TicketListFilter{
		OrganizationID: 1,
		Limit:          10,
		Offset:         0,
	}

	tickets, total, err := service.ListTickets(ctx, filter)
	if err != nil {
		t.Fatalf("failed to list tickets: %v", err)
	}

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(tickets) != 5 {
		t.Errorf("expected 5 tickets, got %d", len(tickets))
	}
}

func TestListTicketsWithFilter(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create tickets with different statuses
	statuses := []string{
		ticket.TicketStatusBacklog,
		ticket.TicketStatusTodo,
		ticket.TicketStatusInProgress,
		ticket.TicketStatusInProgress,
		ticket.TicketStatusDone,
	}

	for _, status := range statuses {
		req := &CreateTicketRequest{
			OrganizationID: 1,
			ReporterID:     1,
			Title:          "Test Ticket",
			Priority:       "medium",
		}
		tkt, _ := service.CreateTicket(ctx, req)

		// Update status using UpdateStatus method
		service.UpdateStatus(ctx, tkt.ID, status)
	}

	// Filter by status
	filter := &ticket.TicketListFilter{
		OrganizationID: 1,
		Status:         ticket.TicketStatusInProgress,
		Limit:          10,
		Offset:         0,
	}

	tickets, total, err := service.ListTickets(ctx, filter)
	if err != nil {
		t.Fatalf("failed to list tickets: %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(tickets) != 2 {
		t.Errorf("expected 2 tickets, got %d", len(tickets))
	}
}

// TestListTickets_Filters covers various filter combinations
func TestListTickets_Filters(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Setup: Create tickets with varied properties
	repoID := int64(1)

	ticketData := []struct {
		repoID     *int64
		priority   string
		reporterID int64
	}{
		{&repoID, "high", 1},
		{nil, "low", 2},
		{nil, "medium", 1},
	}

	for _, tc := range ticketData {
		req := &CreateTicketRequest{
			OrganizationID: 1,
			ReporterID:     tc.reporterID,
			RepositoryID:   tc.repoID,
			Title:          "Test",
			Priority:       tc.priority,
		}
		service.CreateTicket(ctx, req)
	}

	tests := []struct {
		name      string
		filter    *ticket.TicketListFilter
		wantCount int64
	}{
		{
			name:      "filter by repository",
			filter:    &ticket.TicketListFilter{OrganizationID: 1, RepositoryID: &repoID, Limit: 10},
			wantCount: 1,
		},
		{
			name:      "filter by priority",
			filter:    &ticket.TicketListFilter{OrganizationID: 1, Priority: "high", Limit: 10},
			wantCount: 1,
		},
		{
			name:      "filter by reporter",
			filter:    &ticket.TicketListFilter{OrganizationID: 1, ReporterID: func() *int64 { v := int64(1); return &v }(), Limit: 10},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, total, err := service.ListTickets(ctx, tt.filter)
			if err != nil {
				t.Fatalf("ListTickets() error = %v", err)
			}
			if total != tt.wantCount {
				t.Errorf("total = %d, want %d", total, tt.wantCount)
			}
		})
	}
}

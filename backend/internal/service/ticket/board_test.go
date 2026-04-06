package ticket

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Uses setupTestDB from service_test.go

func TestGetBoard(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	service := newTestService(db)

	// Create tickets in different statuses
	statuses := []string{
		ticket.TicketStatusBacklog,
		ticket.TicketStatusTodo,
		ticket.TicketStatusInProgress,
		ticket.TicketStatusInReview,
		ticket.TicketStatusDone,
	}

	for i, status := range statuses {
		for j := 0; j < 2; j++ {
			tkt := &ticket.Ticket{
				OrganizationID: 1,
				Slug:     fmt.Sprintf("BRD-%d", i*10+j+1),
				Title:          "Ticket " + status,
				Status:         status,
				Priority:       ticket.TicketPriorityMedium,
			}
			db.Create(tkt)
		}
	}

	t.Run("returns board with all columns", func(t *testing.T) {
		filter := &ticket.TicketListFilter{
			OrganizationID: 1,
			Limit:          50,
			Offset:         0,
		}
		board, err := service.GetBoard(ctx, filter)
		require.NoError(t, err)
		assert.NotNil(t, board)
		assert.Len(t, board.Columns, 5)

		// Verify each column
		for i, col := range board.Columns {
			assert.Equal(t, statuses[i], col.Status)
			assert.Equal(t, 2, col.Count)
			assert.Len(t, col.Tickets, 2)
		}
	})

	t.Run("returns empty columns for new organization", func(t *testing.T) {
		filter := &ticket.TicketListFilter{
			OrganizationID: 999,
			Limit:          50,
			Offset:         0,
		}
		board, err := service.GetBoard(ctx, filter)
		require.NoError(t, err)
		assert.NotNil(t, board)
		assert.Len(t, board.Columns, 5)

		for _, col := range board.Columns {
			assert.Equal(t, 0, col.Count)
			assert.Empty(t, col.Tickets)
		}
	})
}

func TestGetSubTicketCounts(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	service := newTestService(db)

	// Create parent tickets
	parent1 := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "CNT-1",
		Title:          "Parent 1",
		Status:         ticket.TicketStatusInProgress,
		Priority:       ticket.TicketPriorityMedium,
	}
	parent2 := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "CNT-2",
		Title:          "Parent 2",
		Status:         ticket.TicketStatusTodo,
		Priority:       ticket.TicketPriorityMedium,
	}
	db.Create(parent1)
	db.Create(parent2)

	// Create child tickets for parent1
	children := []struct {
		parentID int64
		status   string
	}{
		{parent1.ID, ticket.TicketStatusTodo},
		{parent1.ID, ticket.TicketStatusTodo},
		{parent1.ID, ticket.TicketStatusInProgress},
		{parent1.ID, ticket.TicketStatusDone},
		{parent2.ID, ticket.TicketStatusBacklog},
	}

	for i, c := range children {
		child := &ticket.Ticket{
			OrganizationID: 1,
			ParentTicketID: &c.parentID,
			Slug:     fmt.Sprintf("CNT-%d", 100+i),
			Title:          "Child",
			Status:         c.status,
			Priority:       ticket.TicketPriorityMedium,
		}
		db.Create(child)
	}

	t.Run("returns sub-ticket counts by status", func(t *testing.T) {
		counts, err := service.GetSubTicketCounts(ctx, []int64{parent1.ID, parent2.ID})
		require.NoError(t, err)
		assert.NotNil(t, counts)

		// Check parent1 counts
		assert.Equal(t, int64(2), counts[parent1.ID][ticket.TicketStatusTodo])
		assert.Equal(t, int64(1), counts[parent1.ID][ticket.TicketStatusInProgress])
		assert.Equal(t, int64(1), counts[parent1.ID][ticket.TicketStatusDone])

		// Check parent2 counts
		assert.Equal(t, int64(1), counts[parent2.ID][ticket.TicketStatusBacklog])
	})

	t.Run("returns empty map for non-existent parents", func(t *testing.T) {
		counts, err := service.GetSubTicketCounts(ctx, []int64{9999})
		require.NoError(t, err)
		assert.Empty(t, counts)
	})
}

// Note: TestGetActiveTickets is defined in service_extended_test.go

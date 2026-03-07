package mesh

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestBatchGetTicketPods(t *testing.T) {
	repo, db := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	// Create pods
	ticketID1 := int64(100)
	ticketID2 := int64(200)

	db.Create(&agentpod.Pod{PodKey: "pod-1", OrganizationID: 1, TicketID: &ticketID1, Status: "running", CreatedByID: 1})
	db.Create(&agentpod.Pod{PodKey: "pod-2", OrganizationID: 1, TicketID: &ticketID1, Status: "terminated", CreatedByID: 1})
	db.Create(&agentpod.Pod{PodKey: "pod-3", OrganizationID: 1, TicketID: &ticketID2, Status: "running", CreatedByID: 1})

	result, err := service.BatchGetTicketPods(ctx, []int64{100, 200, 300})
	if err != nil {
		t.Fatalf("BatchGetTicketPods() error = %v", err)
	}

	if len(result.TicketPods[100]) != 2 {
		t.Errorf("ticket 100 pods = %d, want 2", len(result.TicketPods[100]))
	}
	if len(result.TicketPods[200]) != 1 {
		t.Errorf("ticket 200 pods = %d, want 1", len(result.TicketPods[200]))
	}
	if len(result.TicketPods[300]) != 0 {
		t.Errorf("ticket 300 pods = %d, want 0", len(result.TicketPods[300]))
	}
}

func TestBatchGetTicketPods_Empty(t *testing.T) {
	repo, _ := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	result, err := service.BatchGetTicketPods(ctx, []int64{})
	if err != nil {
		t.Fatalf("BatchGetTicketPods() error = %v", err)
	}
	if len(result.TicketPods) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result.TicketPods))
	}
}

func TestBatchGetTicketPods_NoPods(t *testing.T) {
	repo, _ := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	result, err := service.BatchGetTicketPods(ctx, []int64{1, 2, 3})
	if err != nil {
		t.Fatalf("BatchGetTicketPods() error = %v", err)
	}
	// Should have entries for all requested IDs even if empty
	for _, id := range []int64{1, 2, 3} {
		if _, ok := result.TicketPods[id]; !ok {
			t.Errorf("expected entry for ticket ID %d", id)
		}
		if len(result.TicketPods[id]) != 0 {
			t.Errorf("expected 0 pods for ticket %d, got %d", id, len(result.TicketPods[id]))
		}
	}
}

func TestBatchGetTicketPods_PodWithNilTicket(t *testing.T) {
	repo, db := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	// Create pod with nil ticket_id
	db.Create(&agentpod.Pod{PodKey: "no-ticket", OrganizationID: 1, Status: "running", CreatedByID: 1})

	result, err := service.BatchGetTicketPods(ctx, []int64{100})
	if err != nil {
		t.Fatalf("BatchGetTicketPods() error = %v", err)
	}
	// Pod with nil ticket_id should not be included
	if len(result.TicketPods[100]) != 0 {
		t.Errorf("expected 0 pods for ticket 100, got %d", len(result.TicketPods[100]))
	}
}

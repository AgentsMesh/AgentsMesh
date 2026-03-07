package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

func TestCreateLabel(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	label, err := service.CreateLabel(ctx, 1, nil, "bug", "#FF0000")
	if err != nil {
		t.Fatalf("failed to create label: %v", err)
	}

	if label.Name != "bug" {
		t.Errorf("expected Name 'bug', got %s", label.Name)
	}
	if label.Color != "#FF0000" {
		t.Errorf("expected Color '#FF0000', got %s", label.Color)
	}
}

func TestCreateDuplicateLabel(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create first label
	service.CreateLabel(ctx, 1, nil, "bug", "#FF0000")

	// Try to create duplicate
	_, err := service.CreateLabel(ctx, 1, nil, "bug", "#00FF00")
	if err != ErrDuplicateLabel {
		t.Errorf("expected ErrDuplicateLabel, got %v", err)
	}
}

func TestListLabels(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create labels
	service.CreateLabel(ctx, 1, nil, "bug", "#FF0000")
	service.CreateLabel(ctx, 1, nil, "feature", "#00FF00")
	service.CreateLabel(ctx, 1, nil, "enhancement", "#0000FF")

	// List labels
	labels, err := service.ListLabels(ctx, 1, nil)
	if err != nil {
		t.Fatalf("failed to list labels: %v", err)
	}

	if len(labels) != 3 {
		t.Errorf("expected 3 labels, got %d", len(labels))
	}
}

func TestTicketWithAssignees(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create ticket with assignees
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test Ticket",
		Priority:       "medium",
		AssigneeIDs:    []int64{1, 2, 3},
	}

	tkt, err := service.CreateTicket(ctx, req)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	if len(tkt.Assignees) != 3 {
		t.Errorf("expected 3 assignees, got %d", len(tkt.Assignees))
	}
}

func TestTicketWithLabels(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create labels
	label1, _ := service.CreateLabel(ctx, 1, nil, "bug", "#FF0000")
	label2, _ := service.CreateLabel(ctx, 1, nil, "urgent", "#FF6600")

	// Create ticket with labels
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test Ticket",
		Priority:       "medium",
		LabelIDs:       []int64{label1.ID, label2.ID},
	}

	tkt, err := service.CreateTicket(ctx, req)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	if len(tkt.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(tkt.Labels))
	}
}

// TestUpdateStatus_Transitions tests status change side effects
func TestUpdateStatus_Transitions(t *testing.T) {
	tests := []struct {
		name            string
		toStatus        string
		wantStartedAt   bool
		wantCompletedAt bool
	}{
		{"to in_progress", ticket.TicketStatusInProgress, true, false},
		{"to done", ticket.TicketStatusDone, false, true},
		{"to backlog", ticket.TicketStatusBacklog, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			service := newTestService(db)
			ctx := context.Background()

			tkt, _ := service.CreateTicket(ctx, &CreateTicketRequest{
				OrganizationID: 1,
				ReporterID:     1,
		
				Title:          "Test",
				Priority:       "medium",
			})

			service.UpdateStatus(ctx, tkt.ID, tt.toStatus)
			updated, _ := service.GetTicket(ctx, tkt.ID)

			if tt.wantStartedAt && updated.StartedAt == nil {
				t.Error("expected StartedAt to be set")
			}
			if tt.wantCompletedAt && updated.CompletedAt == nil {
				t.Error("expected CompletedAt to be set")
			}
		})
	}
}

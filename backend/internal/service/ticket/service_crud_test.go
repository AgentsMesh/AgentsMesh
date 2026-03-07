package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

func TestCreateTicket(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test Ticket",
		Priority:       "medium",
	}

	tkt, err := service.CreateTicket(ctx, req)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	if tkt == nil {
		t.Fatal("expected non-nil ticket")
	}
	if tkt.Title != "Test Ticket" {
		t.Errorf("expected Title 'Test Ticket', got %s", tkt.Title)
	}
	if tkt.Status != ticket.TicketStatusBacklog {
		t.Errorf("expected Status '%s', got %s", ticket.TicketStatusBacklog, tkt.Status)
	}
	if tkt.Number != 1 {
		t.Errorf("expected Number 1, got %d", tkt.Number)
	}
	if tkt.Slug != "TICKET-1" {
		t.Errorf("expected Slug 'TICKET-1', got %s", tkt.Slug)
	}
}

func TestGetTicket(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create a ticket
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test Ticket",
		Priority:       "medium",
	}
	created, _ := service.CreateTicket(ctx, req)

	// Get the ticket
	tkt, err := service.GetTicket(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get ticket: %v", err)
	}
	if tkt.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, tkt.ID)
	}
}

func TestGetTicketNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	_, err := service.GetTicket(ctx, 99999)
	if err != ErrTicketNotFound {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestGetTicketBySlug(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create a ticket
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test Ticket",
		Priority:       "medium",
	}
	created, _ := service.CreateTicket(ctx, req)

	// Get by slug
	tkt, err := service.GetTicketBySlug(ctx, int64(1), created.Slug)
	if err != nil {
		t.Fatalf("failed to get ticket by slug: %v", err)
	}
	if tkt.Slug != created.Slug {
		t.Errorf("expected Slug %s, got %s", created.Slug, tkt.Slug)
	}
}

func TestGetTicketBySlug_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	_, err := service.GetTicketBySlug(ctx, int64(1), "NONEXISTENT-999")
	if err != ErrTicketNotFound {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestUpdateTicket(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create a ticket
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test Ticket",
		Priority:       "medium",
	}
	created, _ := service.CreateTicket(ctx, req)

	// Update the ticket
	updates := map[string]interface{}{
		"title": "Updated Title",
	}

	updated, err := service.UpdateTicket(ctx, created.ID, updates)
	if err != nil {
		t.Fatalf("failed to update ticket: %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Errorf("expected Title 'Updated Title', got %s", updated.Title)
	}

	// Test status update separately using UpdateStatus
	err = service.UpdateStatus(ctx, created.ID, ticket.TicketStatusInProgress)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	updated, _ = service.GetTicket(ctx, created.ID)
	if updated.Status != ticket.TicketStatusInProgress {
		t.Errorf("expected Status '%s', got %s", ticket.TicketStatusInProgress, updated.Status)
	}
	if updated.StartedAt == nil {
		t.Error("expected StartedAt to be set when status changed to in_progress")
	}
}

func TestUpdateTicket_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	_, err := service.UpdateTicket(ctx, 99999, map[string]interface{}{"title": "test"})
	if err != ErrTicketNotFound {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestDeleteTicket(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create a ticket
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test Ticket",
		Priority:       "medium",
	}
	created, _ := service.CreateTicket(ctx, req)

	// Delete the ticket
	err := service.DeleteTicket(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to delete ticket: %v", err)
	}

	// Verify deletion
	_, err = service.GetTicket(ctx, created.ID)
	if err != ErrTicketNotFound {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

// TestCreateTicket_TableDriven covers various CreateTicket scenarios
func TestCreateTicket_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupDB        func(*gorm.DB) // Additional DB setup
		req            *CreateTicketRequest
		wantSlug string
		wantStatus     string
		wantErr        bool
	}{
		{
			name: "with custom status",
			req: &CreateTicketRequest{
				OrganizationID: 1,
				ReporterID:     1,
		
				Title:          "Custom Status",
				Priority:       "medium",
				Status:         ticket.TicketStatusTodo,
			},
			wantSlug: "TICKET-1",
			wantStatus:     ticket.TicketStatusTodo,
		},
		{
			name: "with repository prefix",
			setupDB: func(db *gorm.DB) {
				db.Exec(`CREATE TABLE IF NOT EXISTS repositories (id INTEGER PRIMARY KEY, ticket_prefix TEXT)`)
				db.Exec(`INSERT INTO repositories (id, ticket_prefix) VALUES (1, 'PROJ')`)
			},
			req: &CreateTicketRequest{
				OrganizationID: 1,
				ReporterID:     1,
		
				Title:          "With Prefix",
				Priority:       "medium",
				RepositoryID:   func() *int64 { v := int64(1); return &v }(),
			},
			wantSlug: "PROJ-1",
			wantStatus:     ticket.TicketStatusBacklog,
		},
		{
			name: "with label names",
			setupDB: func(db *gorm.DB) {
				db.Exec(`INSERT INTO labels (organization_id, name, color) VALUES (1, 'bug', '#FF0000')`)
			},
			req: &CreateTicketRequest{
				OrganizationID: 1,
				ReporterID:     1,
		
				Title:          "With Labels",
				Priority:       "medium",
				Labels:         []string{"bug"},
			},
			wantSlug: "TICKET-1",
			wantStatus:     ticket.TicketStatusBacklog,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			if tt.setupDB != nil {
				tt.setupDB(db)
			}
			service := newTestService(db)
			ctx := context.Background()

			tkt, err := service.CreateTicket(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTicket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if tkt.Slug != tt.wantSlug {
				t.Errorf("Slug = %s, want %s", tkt.Slug, tt.wantSlug)
			}
			if tkt.Status != tt.wantStatus {
				t.Errorf("Status = %s, want %s", tkt.Status, tt.wantStatus)
			}
		})
	}
}

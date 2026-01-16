package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			number INTEGER NOT NULL,
			identifier TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'task',
			title TEXT NOT NULL,
			description TEXT,
			content TEXT,
			status TEXT NOT NULL DEFAULT 'backlog',
			priority TEXT NOT NULL DEFAULT 'none',
			severity TEXT,
			estimate INTEGER,
			due_date DATETIME,
			started_at DATETIME,
			completed_at DATETIME,
			repository_id INTEGER,
			reporter_id INTEGER NOT NULL,
			parent_ticket_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create tickets table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			repository_id INTEGER,
			name TEXT NOT NULL,
			color TEXT NOT NULL DEFAULT '#808080',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create labels table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ticket_labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			label_id INTEGER NOT NULL
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create ticket_labels table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ticket_assignees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create ticket_assignees table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ticket_merge_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			ticket_id INTEGER NOT NULL,
			pod_id INTEGER,
			mri_id INTEGER NOT NULL,
			mr_url TEXT NOT NULL UNIQUE,
			source_branch TEXT NOT NULL,
			target_branch TEXT NOT NULL DEFAULT 'main',
			title TEXT,
			state TEXT NOT NULL DEFAULT 'opened',
			pipeline_status TEXT,
			pipeline_id INTEGER,
			pipeline_url TEXT,
			merge_commit_sha TEXT,
			merged_at DATETIME,
			merged_by_id INTEGER,
			last_synced_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create ticket_merge_requests table: %v", err)
	}

	return db
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestCreateTicket(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	description := "Test description"
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,
		Type:           "task",
		Title:          "Test Ticket",
		Description:    &description,
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
	if tkt.Identifier != "TICKET-1" {
		t.Errorf("expected Identifier 'TICKET-1', got %s", tkt.Identifier)
	}
}

func TestGetTicket(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create a ticket
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,
		Type:           "task",
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
	service := NewService(db)
	ctx := context.Background()

	_, err := service.GetTicket(ctx, 99999)
	if err != ErrTicketNotFound {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestGetTicketByIdentifier(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create a ticket
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,
		Type:           "task",
		Title:          "Test Ticket",
		Priority:       "medium",
	}
	created, _ := service.CreateTicket(ctx, req)

	// Get by identifier
	tkt, err := service.GetTicketByIdentifier(ctx, created.Identifier)
	if err != nil {
		t.Fatalf("failed to get ticket by identifier: %v", err)
	}
	if tkt.Identifier != created.Identifier {
		t.Errorf("expected Identifier %s, got %s", created.Identifier, tkt.Identifier)
	}
}

func TestUpdateTicket(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create a ticket
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,
		Type:           "task",
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

func TestListTickets(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create multiple tickets
	for i := 1; i <= 5; i++ {
		req := &CreateTicketRequest{
			OrganizationID: 1,
			ReporterID:     1,
			Type:           "task",
			Title:          "Test Ticket",
			Priority:       "medium",
		}
		service.CreateTicket(ctx, req)
	}

	// List tickets
	filter := &ListTicketsFilter{
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
	service := NewService(db)
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
			Type:           "task",
			Title:          "Test Ticket",
			Priority:       "medium",
		}
		tkt, _ := service.CreateTicket(ctx, req)

		// Update status using UpdateStatus method
		service.UpdateStatus(ctx, tkt.ID, status)
	}

	// Filter by status
	filter := &ListTicketsFilter{
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

func TestDeleteTicket(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create a ticket
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,
		Type:           "task",
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

func TestCreateLabel(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
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
	service := NewService(db)
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
	service := NewService(db)
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
	service := NewService(db)
	ctx := context.Background()

	// Create ticket with assignees
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,
		Type:           "task",
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
	service := NewService(db)
	ctx := context.Background()

	// Create labels
	label1, _ := service.CreateLabel(ctx, 1, nil, "bug", "#FF0000")
	label2, _ := service.CreateLabel(ctx, 1, nil, "urgent", "#FF6600")

	// Create ticket with labels
	req := &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,
		Type:           "task",
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

// TestCreateTicket_TableDriven covers various CreateTicket scenarios
func TestCreateTicket_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupDB        func(*gorm.DB) // Additional DB setup
		req            *CreateTicketRequest
		wantIdentifier string
		wantStatus     string
		wantErr        bool
	}{
		{
			name: "with custom status",
			req: &CreateTicketRequest{
				OrganizationID: 1,
				ReporterID:     1,
				Type:           "task",
				Title:          "Custom Status",
				Priority:       "medium",
				Status:         ticket.TicketStatusTodo,
			},
			wantIdentifier: "TICKET-1",
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
				Type:           "task",
				Title:          "With Prefix",
				Priority:       "medium",
				RepositoryID:   func() *int64 { v := int64(1); return &v }(),
			},
			wantIdentifier: "PROJ-1",
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
				Type:           "task",
				Title:          "With Labels",
				Priority:       "medium",
				Labels:         []string{"bug"},
			},
			wantIdentifier: "TICKET-1",
			wantStatus:     ticket.TicketStatusBacklog,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			if tt.setupDB != nil {
				tt.setupDB(db)
			}
			service := NewService(db)
			ctx := context.Background()

			tkt, err := service.CreateTicket(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTicket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if tkt.Identifier != tt.wantIdentifier {
				t.Errorf("Identifier = %s, want %s", tkt.Identifier, tt.wantIdentifier)
			}
			if tkt.Status != tt.wantStatus {
				t.Errorf("Status = %s, want %s", tkt.Status, tt.wantStatus)
			}
		})
	}
}

// TestListTickets_Filters covers various filter combinations
func TestListTickets_Filters(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Setup: Create tickets with varied properties
	repoID := int64(1)

	tickets := []struct {
		repoID     *int64
		typ        string
		priority   string
		reporterID int64
	}{
		{&repoID, "bug", "high", 1},
		{nil, "task", "low", 2},
		{nil, "feature", "medium", 1},
	}

	for _, tc := range tickets {
		req := &CreateTicketRequest{
			OrganizationID: 1,
			ReporterID:     tc.reporterID,
			RepositoryID:   tc.repoID,
			Type:           tc.typ,
			Title:          "Test",
			Priority:       tc.priority,
		}
		service.CreateTicket(ctx, req)
	}

	tests := []struct {
		name      string
		filter    *ListTicketsFilter
		wantCount int64
	}{
		{
			name:      "filter by repository",
			filter:    &ListTicketsFilter{OrganizationID: 1, RepositoryID: &repoID, Limit: 10},
			wantCount: 1,
		},
		{
			name:      "filter by type",
			filter:    &ListTicketsFilter{OrganizationID: 1, Type: "bug", Limit: 10},
			wantCount: 1,
		},
		{
			name:      "filter by priority",
			filter:    &ListTicketsFilter{OrganizationID: 1, Priority: "high", Limit: 10},
			wantCount: 1,
		},
		{
			name:      "filter by reporter",
			filter:    &ListTicketsFilter{OrganizationID: 1, ReporterID: func() *int64 { v := int64(1); return &v }(), Limit: 10},
			wantCount: 2,
		},
		// Team access control test removed - all organization members can now access all resources
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

// TestUpdateStatus_Transitions tests status change side effects
func TestUpdateStatus_Transitions(t *testing.T) {
	tests := []struct {
		name             string
		toStatus         string
		wantStartedAt    bool
		wantCompletedAt  bool
	}{
		{"to in_progress", ticket.TicketStatusInProgress, true, false},
		{"to done", ticket.TicketStatusDone, false, true},
		{"to backlog", ticket.TicketStatusBacklog, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			service := NewService(db)
			ctx := context.Background()

			tkt, _ := service.CreateTicket(ctx, &CreateTicketRequest{
				OrganizationID: 1,
				ReporterID:     1,
				Type:           "task",
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

// TestGetTicketByIdentifier_NotFound tests error case
func TestGetTicketByIdentifier_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	_, err := service.GetTicketByIdentifier(ctx, "NONEXISTENT-999")
	if err != ErrTicketNotFound {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

// TestUpdateTicket_NotFound tests error case
func TestUpdateTicket_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	_, err := service.UpdateTicket(ctx, 99999, map[string]interface{}{"title": "test"})
	if err != ErrTicketNotFound {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

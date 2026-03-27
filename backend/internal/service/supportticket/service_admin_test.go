package supportticket

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
)

// ============================================================
// Admin methods
// ============================================================

func TestAdminList(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	// Create tickets from different users
	svc.Create(ctx, 1, &CreateRequest{Title: "User 1 ticket", Category: domain.CategoryBug})
	svc.Create(ctx, 2, &CreateRequest{Title: "User 2 ticket", Category: domain.CategoryAccount})
	svc.Create(ctx, 3, &CreateRequest{Title: "User 3 ticket", Category: domain.CategoryOther})

	resp, err := svc.AdminList(ctx, &AdminListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Total)
	}
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 tickets, got %d", len(resp.Data))
	}
}

func TestAdminList_SearchFilter(t *testing.T) {
	// SQLite does not support ILIKE; the AdminList method uses ILIKE which is
	// PostgreSQL-specific. We skip this test in SQLite-backed test runs.
	t.Skip("Skipping: AdminList search uses ILIKE which is not supported by SQLite")
}

func TestAdminList_StatusFilter(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	svc.Create(ctx, 1, &CreateRequest{Title: "Open", Category: domain.CategoryOther})
	ticket2, _ := svc.Create(ctx, 1, &CreateRequest{Title: "Closed", Category: domain.CategoryOther})
	db.Model(&domain.SupportTicket{}).Where("id = ?", ticket2.ID).Update("status", domain.StatusClosed)

	resp, err := svc.AdminList(ctx, &AdminListQuery{Status: domain.StatusOpen, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 open ticket, got %d", resp.Total)
	}
	if len(resp.Data) > 0 && resp.Data[0].Title != "Open" {
		t.Errorf("expected title 'Open', got %q", resp.Data[0].Title)
	}
}

func TestAdminList_CategoryFilter(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	svc.Create(ctx, 1, &CreateRequest{Title: "Bug report", Category: domain.CategoryBug})
	svc.Create(ctx, 1, &CreateRequest{Title: "Feature request", Category: domain.CategoryFeatureRequest})

	resp, err := svc.AdminList(ctx, &AdminListQuery{Category: domain.CategoryBug, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 bug ticket, got %d", resp.Total)
	}
}

func TestAdminList_PriorityFilter(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	svc.Create(ctx, 1, &CreateRequest{Title: "High prio", Category: domain.CategoryOther, Priority: domain.PriorityHigh})
	svc.Create(ctx, 1, &CreateRequest{Title: "Low prio", Category: domain.CategoryOther, Priority: domain.PriorityLow})
	svc.Create(ctx, 1, &CreateRequest{Title: "Default prio", Category: domain.CategoryOther}) // defaults to medium

	resp, err := svc.AdminList(ctx, &AdminListQuery{Priority: domain.PriorityHigh, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 high-priority ticket, got %d", resp.Total)
	}
}

func TestAdminGetByID(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Any user ticket",
		Category: domain.CategoryOther,
	})

	// Admin can access any ticket regardless of ownership
	ticket, err := svc.AdminGetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, ticket.ID)
	}
	if ticket.Title != "Any user ticket" {
		t.Errorf("expected title 'Any user ticket', got %q", ticket.Title)
	}
}

func TestAdminGetByID_NotFound(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	_, err := svc.AdminGetByID(ctx, 99999)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestAdminAddReply(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()
	createTestUser(t, db, 100, "admin@test.com")

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "User ticket",
		Category: domain.CategoryOther,
	})

	msg, err := svc.AdminAddReply(ctx, ticket.ID, 100, &AddMessageRequest{
		Content: "We are looking into this.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if !msg.IsAdminReply {
		t.Error("expected message to be an admin reply")
	}
	if msg.Content != "We are looking into this." {
		t.Errorf("expected content 'We are looking into this.', got %q", msg.Content)
	}
	if msg.UserID != 100 {
		t.Errorf("expected user_id 100, got %d", msg.UserID)
	}
}

func TestAdminAddReply_AutoTransition(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Open ticket",
		Category: domain.CategoryOther,
	})

	// Verify ticket is open
	var status string
	db.Model(&domain.SupportTicket{}).Where("id = ?", ticket.ID).Pluck("status", &status)
	if status != domain.StatusOpen {
		t.Fatalf("expected initial status %q, got %q", domain.StatusOpen, status)
	}

	// Admin replies
	_, err := svc.AdminAddReply(ctx, ticket.ID, 100, &AddMessageRequest{Content: "Looking into it"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status transitioned to in_progress
	db.Model(&domain.SupportTicket{}).Where("id = ?", ticket.ID).Pluck("status", &status)
	if status != domain.StatusInProgress {
		t.Errorf("expected status %q after admin reply, got %q", domain.StatusInProgress, status)
	}
}

func TestAdminAddReply_NotFound(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	_, err := svc.AdminAddReply(ctx, 99999, 100, &AddMessageRequest{Content: "Reply"})
	if !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestAdminUpdateStatus(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Status test",
		Category: domain.CategoryOther,
	})

	err := svc.AdminUpdateStatus(ctx, ticket.ID, domain.StatusInProgress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var status string
	db.Model(&domain.SupportTicket{}).Where("id = ?", ticket.ID).Pluck("status", &status)
	if status != domain.StatusInProgress {
		t.Errorf("expected status %q, got %q", domain.StatusInProgress, status)
	}
}

func TestAdminUpdateStatus_Resolved(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Resolve test",
		Category: domain.CategoryOther,
	})

	err := svc.AdminUpdateStatus(ctx, ticket.ID, domain.StatusResolved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated domain.SupportTicket
	db.First(&updated, ticket.ID)
	if updated.Status != domain.StatusResolved {
		t.Errorf("expected status %q, got %q", domain.StatusResolved, updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Error("expected resolved_at to be set")
	} else {
		// Verify resolved_at is recent (within last 5 seconds)
		if time.Since(*updated.ResolvedAt) > 5*time.Second {
			t.Error("expected resolved_at to be recent")
		}
	}
}

func TestAdminUpdateStatus_InvalidStatus(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Status test",
		Category: domain.CategoryOther,
	})

	err := svc.AdminUpdateStatus(ctx, ticket.ID, "invalid_status")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestAdminUpdateStatus_NotFound(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	err := svc.AdminUpdateStatus(ctx, 99999, domain.StatusClosed)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestAdminUpdateStatus_InvalidTransition(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Transition test",
		Category: domain.CategoryOther,
	})

	// Close the ticket first (open -> closed is valid)
	_ = svc.AdminUpdateStatus(ctx, ticket.ID, domain.StatusClosed)

	// closed -> in_progress is NOT a valid transition
	err := svc.AdminUpdateStatus(ctx, ticket.ID, domain.StatusInProgress)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestAdminUpdateStatus_SameStatusNoop(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Noop test",
		Category: domain.CategoryOther,
	})

	// open -> open should be a no-op (no error)
	err := svc.AdminUpdateStatus(ctx, ticket.ID, domain.StatusOpen)
	if err != nil {
		t.Fatalf("expected no error for same status, got %v", err)
	}
}

func TestAdminUpdateStatus_ResolvedAtNotOverwritten(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "ResolvedAt test",
		Category: domain.CategoryOther,
	})

	// First resolve: open -> resolved
	_ = svc.AdminUpdateStatus(ctx, ticket.ID, domain.StatusResolved)

	var first domain.SupportTicket
	db.First(&first, ticket.ID)
	if first.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set after first resolve")
	}
	firstResolvedAt := *first.ResolvedAt

	// Reopen: resolved -> open
	_ = svc.AdminUpdateStatus(ctx, ticket.ID, domain.StatusOpen)

	// Re-resolve: open -> resolved
	_ = svc.AdminUpdateStatus(ctx, ticket.ID, domain.StatusResolved)

	var second domain.SupportTicket
	db.First(&second, ticket.ID)
	if second.ResolvedAt == nil {
		t.Fatal("expected resolved_at to still be set")
	}
	// ResolvedAt should NOT have been overwritten
	if !firstResolvedAt.Equal(*second.ResolvedAt) {
		t.Errorf("resolved_at was overwritten: first=%v, second=%v", firstResolvedAt, *second.ResolvedAt)
	}
}

func TestAdminAssign(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Assign test",
		Category: domain.CategoryOther,
	})

	err := svc.AdminAssign(ctx, ticket.ID, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated domain.SupportTicket
	db.First(&updated, ticket.ID)
	if updated.AssignedAdminID == nil {
		t.Fatal("expected assigned_admin_id to be set")
	}
	if *updated.AssignedAdminID != 42 {
		t.Errorf("expected assigned_admin_id 42, got %d", *updated.AssignedAdminID)
	}
}

func TestAdminAssign_NotFound(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	err := svc.AdminAssign(ctx, 99999, 42)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestAdminGetStats(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	// Create tickets with various statuses
	svc.Create(ctx, 1, &CreateRequest{Title: "Open 1", Category: domain.CategoryOther})
	svc.Create(ctx, 1, &CreateRequest{Title: "Open 2", Category: domain.CategoryOther})

	t3, _ := svc.Create(ctx, 1, &CreateRequest{Title: "In Progress", Category: domain.CategoryOther})
	db.Model(&domain.SupportTicket{}).Where("id = ?", t3.ID).Update("status", domain.StatusInProgress)

	t4, _ := svc.Create(ctx, 1, &CreateRequest{Title: "Resolved", Category: domain.CategoryOther})
	db.Model(&domain.SupportTicket{}).Where("id = ?", t4.ID).Update("status", domain.StatusResolved)

	t5, _ := svc.Create(ctx, 1, &CreateRequest{Title: "Closed 1", Category: domain.CategoryOther})
	db.Model(&domain.SupportTicket{}).Where("id = ?", t5.ID).Update("status", domain.StatusClosed)

	t6, _ := svc.Create(ctx, 1, &CreateRequest{Title: "Closed 2", Category: domain.CategoryOther})
	db.Model(&domain.SupportTicket{}).Where("id = ?", t6.ID).Update("status", domain.StatusClosed)

	stats, err := svc.AdminGetStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Total != 6 {
		t.Errorf("expected total 6, got %d", stats.Total)
	}
	if stats.Open != 2 {
		t.Errorf("expected open 2, got %d", stats.Open)
	}
	if stats.InProgress != 1 {
		t.Errorf("expected in_progress 1, got %d", stats.InProgress)
	}
	if stats.Resolved != 1 {
		t.Errorf("expected resolved 1, got %d", stats.Resolved)
	}
	if stats.Closed != 2 {
		t.Errorf("expected closed 2, got %d", stats.Closed)
	}
}

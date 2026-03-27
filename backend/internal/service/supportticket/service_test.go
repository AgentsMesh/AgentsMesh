package supportticket

import (
	"context"
	"errors"
	"testing"

	domain "github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
)

// ============================================================
// User methods
// ============================================================

func TestCreate(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	req := &CreateRequest{
		Title:    "Login page broken",
		Category: domain.CategoryBug,
		Priority: domain.PriorityHigh,
	}

	ticket, err := svc.Create(ctx, 1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket == nil {
		t.Fatal("expected non-nil ticket")
	}
	if ticket.Title != "Login page broken" {
		t.Errorf("expected title 'Login page broken', got %q", ticket.Title)
	}
	if ticket.Category != domain.CategoryBug {
		t.Errorf("expected category %q, got %q", domain.CategoryBug, ticket.Category)
	}
	if ticket.Priority != domain.PriorityHigh {
		t.Errorf("expected priority %q, got %q", domain.PriorityHigh, ticket.Priority)
	}
	if ticket.Status != domain.StatusOpen {
		t.Errorf("expected status %q, got %q", domain.StatusOpen, ticket.Status)
	}
	if ticket.UserID != 1 {
		t.Errorf("expected user_id 1, got %d", ticket.UserID)
	}
	if ticket.ID == 0 {
		t.Error("expected non-zero ticket ID")
	}
}

func TestCreate_DefaultCategoryAndPriority(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	req := &CreateRequest{
		Title: "General question",
	}

	ticket, err := svc.Create(ctx, 1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.Category != domain.CategoryOther {
		t.Errorf("expected default category %q, got %q", domain.CategoryOther, ticket.Category)
	}
	if ticket.Priority != domain.PriorityMedium {
		t.Errorf("expected default priority %q, got %q", domain.PriorityMedium, ticket.Priority)
	}
}

func TestCreate_InvalidCategory(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	req := &CreateRequest{
		Title:    "Bad category",
		Category: "nonexistent_category",
	}

	_, err := svc.Create(ctx, 1, req)
	if !errors.Is(err, ErrInvalidCategory) {
		t.Fatalf("expected ErrInvalidCategory, got %v", err)
	}
}

func TestCreate_InvalidPriority(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	req := &CreateRequest{
		Title:    "Bad priority",
		Category: domain.CategoryBug,
		Priority: "critical",
	}

	_, err := svc.Create(ctx, 1, req)
	if !errors.Is(err, ErrInvalidPriority) {
		t.Fatalf("expected ErrInvalidPriority, got %v", err)
	}
}

func TestCreate_WithContent(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	req := &CreateRequest{
		Title:    "Need help",
		Category: domain.CategoryUsageQuestion,
		Content:  "How do I configure the runner?",
	}

	ticket, err := svc.Create(ctx, 1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the initial message was created
	var messages []domain.SupportTicketMessage
	if err := db.Where("ticket_id = ?", ticket.ID).Find(&messages).Error; err != nil {
		t.Fatalf("failed to query messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Content != "How do I configure the runner?" {
		t.Errorf("expected message content 'How do I configure the runner?', got %q", messages[0].Content)
	}
	if messages[0].IsAdminReply {
		t.Error("expected initial message to not be an admin reply")
	}
	if messages[0].UserID != 1 {
		t.Errorf("expected message user_id 1, got %d", messages[0].UserID)
	}
}

func TestListByUser(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	// Create tickets with a small delay to ensure ordering
	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, 1, &CreateRequest{
			Title:    "Ticket " + string(rune('A'+i)),
			Category: domain.CategoryOther,
		})
		if err != nil {
			t.Fatalf("failed to create ticket %d: %v", i, err)
		}
	}
	// Create a ticket for a different user
	_, err := svc.Create(ctx, 2, &CreateRequest{
		Title:    "Other user ticket",
		Category: domain.CategoryOther,
	})
	if err != nil {
		t.Fatalf("failed to create other user ticket: %v", err)
	}

	resp, err := svc.ListByUser(ctx, 1, &ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Total)
	}
	if len(resp.Data) != 3 {
		t.Fatalf("expected 3 tickets, got %d", len(resp.Data))
	}

	// Verify DESC ordering (newest first)
	for i := 1; i < len(resp.Data); i++ {
		if resp.Data[i-1].CreatedAt.Before(resp.Data[i].CreatedAt) {
			t.Errorf("expected descending order by created_at, but ticket at index %d is older than ticket at index %d", i-1, i)
		}
	}
}

func TestListByUser_StatusFilter(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	// Create two open tickets and one resolved
	for i := 0; i < 2; i++ {
		svc.Create(ctx, 1, &CreateRequest{Title: "Open ticket", Category: domain.CategoryOther})
	}
	ticket, _ := svc.Create(ctx, 1, &CreateRequest{Title: "Resolved ticket", Category: domain.CategoryOther})
	db.Model(&domain.SupportTicket{}).Where("id = ?", ticket.ID).Update("status", domain.StatusResolved)

	resp, err := svc.ListByUser(ctx, 1, &ListQuery{Status: domain.StatusOpen, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected 2 open tickets, got %d", resp.Total)
	}

	resp, err = svc.ListByUser(ctx, 1, &ListQuery{Status: domain.StatusResolved, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 resolved ticket, got %d", resp.Total)
	}
}

func TestListByUser_Pagination(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	// Create 5 tickets
	for i := 0; i < 5; i++ {
		svc.Create(ctx, 1, &CreateRequest{Title: "Ticket", Category: domain.CategoryOther})
	}

	// Page 1, size 2
	resp, err := svc.ListByUser(ctx, 1, &ListQuery{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 5 {
		t.Errorf("expected total 5, got %d", resp.Total)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 items on page 1, got %d", len(resp.Data))
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
	if resp.PageSize != 2 {
		t.Errorf("expected pageSize 2, got %d", resp.PageSize)
	}
	if resp.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", resp.TotalPages)
	}

	// Page 3, size 2 (should have 1 item)
	resp, err = svc.ListByUser(ctx, 1, &ListQuery{Page: 3, PageSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 item on last page, got %d", len(resp.Data))
	}
}

func TestListByUser_EmptyResult(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	resp, err := svc.ListByUser(ctx, 999, &ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
	if resp.Data == nil {
		t.Error("expected non-nil data slice, got nil")
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected empty data slice, got %d items", len(resp.Data))
	}
}

func TestGetByID(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "My ticket",
		Category: domain.CategoryAccount,
		Priority: domain.PriorityLow,
	})

	ticket, err := svc.GetByID(ctx, created.ID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, ticket.ID)
	}
	if ticket.Title != "My ticket" {
		t.Errorf("expected title 'My ticket', got %q", ticket.Title)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999, 1)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestGetByID_WrongUser(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "User 1 ticket",
		Category: domain.CategoryOther,
	})

	_, err := svc.GetByID(ctx, created.ID, 2)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound for wrong user, got %v", err)
	}
}


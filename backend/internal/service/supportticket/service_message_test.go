package supportticket

import (
	"context"
	"errors"
	"testing"

	domain "github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
)

func TestAddMessage(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Test ticket",
		Category: domain.CategoryOther,
	})

	msg, err := svc.AddMessage(ctx, ticket.ID, 1, &AddMessageRequest{
		Content: "Here is more info",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Content != "Here is more info" {
		t.Errorf("expected content 'Here is more info', got %q", msg.Content)
	}
	if msg.IsAdminReply {
		t.Error("expected message to not be an admin reply")
	}
	if msg.TicketID != ticket.ID {
		t.Errorf("expected ticket_id %d, got %d", ticket.ID, msg.TicketID)
	}
	if msg.UserID != 1 {
		t.Errorf("expected user_id 1, got %d", msg.UserID)
	}
}

func TestAddMessage_ReopensResolved(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Resolved ticket",
		Category: domain.CategoryOther,
	})

	// Manually set status to resolved
	db.Model(&domain.SupportTicket{}).Where("id = ?", ticket.ID).Update("status", domain.StatusResolved)

	// User adds a message
	_, err := svc.AddMessage(ctx, ticket.ID, 1, &AddMessageRequest{Content: "Still having issues"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify ticket is reopened
	updated, err := svc.GetByID(ctx, ticket.ID, 1)
	if err != nil {
		t.Fatalf("failed to get ticket: %v", err)
	}
	if updated.Status != domain.StatusOpen {
		t.Errorf("expected status %q after reopen, got %q", domain.StatusOpen, updated.Status)
	}
}

func TestAddMessage_WrongUser(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "User 1 ticket",
		Category: domain.CategoryOther,
	})

	_, err := svc.AddMessage(ctx, ticket.ID, 2, &AddMessageRequest{Content: "I shouldn't be able to do this"})
	if !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound for wrong user, got %v", err)
	}
}

func TestListMessages(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "Ticket with messages",
		Category: domain.CategoryOther,
		Content:  "Initial message",
	})

	svc.AddMessage(ctx, ticket.ID, 1, &AddMessageRequest{Content: "Follow-up"})

	messages, err := svc.ListMessages(ctx, ticket.ID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Content != "Initial message" {
		t.Errorf("expected first message 'Initial message', got %q", messages[0].Content)
	}
	if messages[1].Content != "Follow-up" {
		t.Errorf("expected second message 'Follow-up', got %q", messages[1].Content)
	}
}

func TestListMessages_WrongUser(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	ticket, _ := svc.Create(ctx, 1, &CreateRequest{
		Title:    "User 1 ticket",
		Category: domain.CategoryOther,
		Content:  "Hello",
	})

	_, err := svc.ListMessages(ctx, ticket.ID, 2)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound for wrong user, got %v", err)
	}
}

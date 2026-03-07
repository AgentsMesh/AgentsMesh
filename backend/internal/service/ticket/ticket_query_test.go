package ticket

import (
	"context"
	"fmt"
	"testing"
)

func TestGetTicketByIDOrSlug(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create tickets in org 1
	tkt1, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Org1 First",
		Priority:       "medium",
	})
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	tkt2, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Org1 Second",
		Priority:       "medium",
	})
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	// Create ticket in org 2 (same slug "TICKET-1")
	tktOrg2, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 2,
		ReporterID:     2,

		Title:          "Org2 First",
		Priority:       "medium",
	})
	if err != nil {
		t.Fatalf("failed to create ticket in org 2: %v", err)
	}

	t.Run("lookup by slug string", func(t *testing.T) {
		got, err := service.GetTicketByIDOrSlug(ctx, 1, "TICKET-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.ID != tkt1.ID {
			t.Errorf("expected ticket ID %d, got %d", tkt1.ID, got.ID)
		}
		if got.Title != "Org1 First" {
			t.Errorf("expected title 'Org1 First', got %s", got.Title)
		}
	})

	t.Run("lookup by slug string second ticket", func(t *testing.T) {
		got, err := service.GetTicketByIDOrSlug(ctx, 1, "TICKET-2")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.ID != tkt2.ID {
			t.Errorf("expected ticket ID %d, got %d", tkt2.ID, got.ID)
		}
	})

	t.Run("lookup by numeric ID string", func(t *testing.T) {
		got, err := service.GetTicketByIDOrSlug(ctx, 1, "1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.ID != tkt1.ID {
			t.Errorf("expected ticket ID %d, got %d", tkt1.ID, got.ID)
		}
	})

	t.Run("lookup by numeric ID string second ticket", func(t *testing.T) {
		got, err := service.GetTicketByIDOrSlug(ctx, 1, "2")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.ID != tkt2.ID {
			t.Errorf("expected ticket ID %d, got %d", tkt2.ID, got.ID)
		}
	})

	t.Run("numeric ID respects organization boundary", func(t *testing.T) {
		// tktOrg2 belongs to org 2; querying with org 1 should fail
		idStr := idToString(tktOrg2.ID)
		_, err := service.GetTicketByIDOrSlug(ctx, 1, idStr)
		if err == nil {
			t.Fatal("expected error for cross-org access by numeric ID, got nil")
		}
	})

	t.Run("slug scoped to organization", func(t *testing.T) {
		// Both orgs have "TICKET-1" but they are different tickets
		got, err := service.GetTicketByIDOrSlug(ctx, 2, "TICKET-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.ID != tktOrg2.ID {
			t.Errorf("expected ticket ID %d (org 2), got %d", tktOrg2.ID, got.ID)
		}
	})

	t.Run("nonexistent slug returns error", func(t *testing.T) {
		_, err := service.GetTicketByIDOrSlug(ctx, 1, "NONEXIST-999")
		if err == nil {
			t.Fatal("expected error for nonexistent slug, got nil")
		}
	})

	t.Run("nonexistent numeric ID returns error", func(t *testing.T) {
		_, err := service.GetTicketByIDOrSlug(ctx, 1, "99999")
		if err == nil {
			t.Fatal("expected error for nonexistent numeric ID, got nil")
		}
	})

	t.Run("empty string returns error", func(t *testing.T) {
		_, err := service.GetTicketByIDOrSlug(ctx, 1, "")
		if err == nil {
			t.Fatal("expected error for empty string, got nil")
		}
	})
}

// idToString converts int64 to string for test purposes.
func idToString(id int64) string {
	return fmt.Sprintf("%d", id)
}

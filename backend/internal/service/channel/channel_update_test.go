package channel

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func TestListChannels(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: string(rune('a' + i))})
	}

	channels, _, _ := svc.ListChannels(ctx, 1, &channel.ChannelListFilter{IncludeArchived: true, Limit: 10, Offset: 0})
	if len(channels) > 0 {
		svc.ArchiveChannel(ctx, channels[0].ID)
	}

	t.Run("active only", func(t *testing.T) {
		channels, total, _ := svc.ListChannels(ctx, 1, &channel.ChannelListFilter{IncludeArchived: false, Limit: 10, Offset: 0})
		if total != 4 || len(channels) != 4 {
			t.Errorf("Expected 4 active channels, got %d", total)
		}
	})

	t.Run("including archived", func(t *testing.T) {
		_, total, _ := svc.ListChannels(ctx, 1, &channel.ChannelListFilter{IncludeArchived: true, Limit: 10, Offset: 0})
		if total != 5 {
			t.Errorf("Expected 5 total channels, got %d", total)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		channels, total, _ := svc.ListChannels(ctx, 1, &channel.ChannelListFilter{IncludeArchived: true, Limit: 2, Offset: 0})
		if total != 5 || len(channels) != 2 {
			t.Errorf("Pagination failed: total=%d, len=%d", total, len(channels))
		}
	})
}

func TestListChannels_FilterByRepositoryID(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	repoA := intPtr(10)
	repoB := intPtr(20)
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "repo-a-1", RepositoryID: repoA})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "repo-a-2", RepositoryID: repoA})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "repo-b-1", RepositoryID: repoB})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "no-repo"})

	t.Run("filter by repo A", func(t *testing.T) {
		channels, total, err := svc.ListChannels(ctx, 1, &channel.ChannelListFilter{
			IncludeArchived: true, RepositoryID: repoA, Limit: 10, Offset: 0,
		})
		if err != nil {
			t.Fatalf("ListChannels failed: %v", err)
		}
		if total != 2 || len(channels) != 2 {
			t.Errorf("Expected 2 channels for repo A, got total=%d len=%d", total, len(channels))
		}
	})

	t.Run("filter by repo B", func(t *testing.T) {
		channels, total, err := svc.ListChannels(ctx, 1, &channel.ChannelListFilter{
			IncludeArchived: true, RepositoryID: repoB, Limit: 10, Offset: 0,
		})
		if err != nil {
			t.Fatalf("ListChannels failed: %v", err)
		}
		if total != 1 || len(channels) != 1 {
			t.Errorf("Expected 1 channel for repo B, got total=%d len=%d", total, len(channels))
		}
	})

	t.Run("no filter returns all", func(t *testing.T) {
		_, total, _ := svc.ListChannels(ctx, 1, &channel.ChannelListFilter{
			IncludeArchived: true, Limit: 10, Offset: 0,
		})
		if total != 4 {
			t.Errorf("Expected 4 total channels, got %d", total)
		}
	})
}

func TestListChannels_FilterByTicketID(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ticketA := intPtr(100)
	ticketB := intPtr(200)
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "ticket-a-1", TicketID: ticketA})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "ticket-b-1", TicketID: ticketB})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "ticket-b-2", TicketID: ticketB})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "no-ticket"})

	t.Run("filter by ticket A", func(t *testing.T) {
		channels, total, err := svc.ListChannels(ctx, 1, &channel.ChannelListFilter{
			IncludeArchived: true, TicketID: ticketA, Limit: 10, Offset: 0,
		})
		if err != nil {
			t.Fatalf("ListChannels failed: %v", err)
		}
		if total != 1 || len(channels) != 1 {
			t.Errorf("Expected 1 channel for ticket A, got total=%d len=%d", total, len(channels))
		}
	})

	t.Run("filter by ticket B", func(t *testing.T) {
		channels, total, err := svc.ListChannels(ctx, 1, &channel.ChannelListFilter{
			IncludeArchived: true, TicketID: ticketB, Limit: 10, Offset: 0,
		})
		if err != nil {
			t.Fatalf("ListChannels failed: %v", err)
		}
		if total != 2 || len(channels) != 2 {
			t.Errorf("Expected 2 channels for ticket B, got total=%d len=%d", total, len(channels))
		}
	})
}

func TestListChannels_CombinedFilters(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	repo := intPtr(10)
	ticketA := intPtr(100)
	ticketB := intPtr(200)
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "match", RepositoryID: repo, TicketID: ticketA})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "repo-only", RepositoryID: repo})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "ticket-only", TicketID: ticketA})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "diff-ticket", RepositoryID: repo, TicketID: ticketB})

	channels, total, err := svc.ListChannels(ctx, 1, &channel.ChannelListFilter{
		IncludeArchived: true, RepositoryID: repo, TicketID: ticketA, Limit: 10, Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListChannels failed: %v", err)
	}
	if total != 1 || len(channels) != 1 {
		t.Errorf("Expected 1 channel matching both filters, got total=%d len=%d", total, len(channels))
	}
	if len(channels) == 1 && channels[0].Name != "match" {
		t.Errorf("Expected channel 'match', got '%s'", channels[0].Name)
	}
}

func TestUpdateChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "original"})

	t.Run("update name", func(t *testing.T) {
		newName := "updated"
		updated, err := svc.UpdateChannel(ctx, created.ID, &newName, nil, nil)
		if err != nil || updated.Name != newName {
			t.Errorf("UpdateChannel failed: %v", err)
		}
	})

	t.Run("update description", func(t *testing.T) {
		desc := "New desc"
		updated, err := svc.UpdateChannel(ctx, created.ID, nil, &desc, nil)
		if err != nil || updated.Description == nil || *updated.Description != desc {
			t.Error("Description not updated")
		}
	})

	t.Run("update archived channel", func(t *testing.T) {
		svc.ArchiveChannel(ctx, created.ID)
		newName := "fail"
		if _, err := svc.UpdateChannel(ctx, created.ID, &newName, nil, nil); err != ErrChannelArchived {
			t.Errorf("Expected ErrChannelArchived, got %v", err)
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		name := "test"
		if _, err := svc.UpdateChannel(ctx, 99999, &name, nil, nil); err == nil {
			t.Error("Expected error for non-existent channel")
		}
	})
}

func TestArchiveUnarchiveChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "archive-test"})

	t.Run("archive", func(t *testing.T) {
		if err := svc.ArchiveChannel(ctx, created.ID); err != nil {
			t.Errorf("ArchiveChannel failed: %v", err)
		}
		ch, _ := svc.GetChannel(ctx, created.ID)
		if !ch.IsArchived {
			t.Error("Channel should be archived")
		}
	})

	t.Run("unarchive", func(t *testing.T) {
		if err := svc.UnarchiveChannel(ctx, created.ID); err != nil {
			t.Errorf("UnarchiveChannel failed: %v", err)
		}
		ch, _ := svc.GetChannel(ctx, created.ID)
		if ch.IsArchived {
			t.Error("Channel should not be archived")
		}
	})
}

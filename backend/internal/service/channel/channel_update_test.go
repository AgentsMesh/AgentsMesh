package channel

import (
	"context"
	"testing"
)

func TestListChannels(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: string(rune('a' + i))})
	}

	channels, _, _ := svc.ListChannels(ctx, 1, true, 10, 0)
	if len(channels) > 0 {
		svc.ArchiveChannel(ctx, channels[0].ID)
	}

	t.Run("active only", func(t *testing.T) {
		channels, total, _ := svc.ListChannels(ctx, 1, false, 10, 0)
		if total != 4 || len(channels) != 4 {
			t.Errorf("Expected 4 active channels, got %d", total)
		}
	})

	t.Run("including archived", func(t *testing.T) {
		_, total, _ := svc.ListChannels(ctx, 1, true, 10, 0)
		if total != 5 {
			t.Errorf("Expected 5 total channels, got %d", total)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		channels, total, _ := svc.ListChannels(ctx, 1, true, 2, 0)
		if total != 5 || len(channels) != 2 {
			t.Errorf("Pagination failed: total=%d, len=%d", total, len(channels))
		}
	})
}

func TestUpdateChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
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
	svc := NewService(db)
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

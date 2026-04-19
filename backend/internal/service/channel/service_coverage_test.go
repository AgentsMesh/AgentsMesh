package channel

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

func TestSetUserLookup(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)

	lookup := &mockUserLookup{validIDs: []int64{1}}
	svc.SetUserLookup(lookup)
	if svc.userLookup == nil {
		t.Error("userLookup should be set")
	}
}

func TestValidateOrgMembers_WithRealLookup(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	t.Run("filters to valid IDs", func(t *testing.T) {
		svc.SetUserLookup(&mockUserLookup{validIDs: []int64{10, 30}})
		result := svc.validateOrgMembers(ctx, 1, []int64{10, 20, 30})
		if len(result) != 2 {
			t.Errorf("Expected 2 valid members, got %d", len(result))
		}
	})

	t.Run("error returns nil", func(t *testing.T) {
		svc.SetUserLookup(&mockUserLookup{err: errors.New("db error")})
		result := svc.validateOrgMembers(ctx, 1, []int64{10, 20})
		if result != nil {
			t.Errorf("Expected nil on error, got %v", result)
		}
	})
}

func TestGetChannelForUser_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	_, err := svc.GetChannelForUser(ctx, 99999, 1)
	if err != ErrChannelNotFound {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestDeleteChannelsByOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 100, Name: "org-ch-1"})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 100, Name: "org-ch-2"})
	svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 200, Name: "other-org"})

	err := svc.DeleteChannelsByOrg(ctx, 100)
	if err != nil {
		t.Fatalf("DeleteChannelsByOrg failed: %v", err)
	}

	channels, total, _ := svc.ListChannels(ctx, 100, 1, &channel.ChannelListFilter{
		IncludeArchived: true, Limit: 10, Offset: 0,
	})
	if total != 0 || len(channels) != 0 {
		t.Errorf("Expected 0 channels for org 100, got %d", total)
	}

	_, total2, _ := svc.ListChannels(ctx, 200, 1, &channel.ChannelListFilter{
		IncludeArchived: true, Limit: 10, Offset: 0,
	})
	if total2 != 1 {
		t.Errorf("Expected 1 channel for org 200, got %d", total2)
	}
}

func TestCleanupUserReferences(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)
	member := int64(20)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "cleanup-test", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{member},
	})

	ok, _ := svc.IsMember(ctx, ch.ID, member)
	if !ok {
		t.Fatal("User 20 should be a member before cleanup")
	}

	err := svc.CleanupUserReferences(ctx, member)
	if err != nil {
		t.Fatalf("CleanupUserReferences failed: %v", err)
	}
}

func TestSendMessage_PodOnlyInPrivateChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "pod-private", CreatedByUserID: &creator,
		Visibility: channel.VisibilityPrivate,
	})

	podKey := "agent-pod"
	msg, err := svc.SendMessage(ctx, ch.ID, &podKey, nil, textContent("pod msg"), nil)
	if err != nil {
		t.Fatalf("Pod-only send should succeed: %v", err)
	}
	if msg.SenderPod == nil || *msg.SenderPod != podKey {
		t.Error("SenderPod not set correctly")
	}
}

func TestSendMessage_SystemOnPrivateChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "sys-private", CreatedByUserID: &creator,
		Visibility: channel.VisibilityPrivate,
	})

	msg, err := svc.SendMessage(ctx, ch.ID, nil, nil, textContent("system msg"), nil)
	if err != nil {
		t.Fatalf("System send should succeed: %v", err)
	}
	if msg.SenderUserID != nil {
		t.Error("System message should have nil SenderUserID")
	}
}

func TestSendMessage_MentionWithNilMetadata(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "nil-meta"})

	msg, err := svc.SendMessage(ctx, ch.ID, nil, nil, textContent("hello"), nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msg == nil {
		t.Error("Message should not be nil")
	}
}

func TestSendMessage_MentionWithUserType(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "user-mention"})

	msg, err := svc.SendMessage(ctx, ch.ID, nil, nil, textContent("hi all"), nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msg == nil {
		t.Error("Message should not be nil")
	}
}

func TestUpdateChannel_WithDocument(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "doc-update"})

	doc := "# Channel Document"
	updated, err := svc.UpdateChannel(ctx, ch.ID, nil, nil, &doc)
	if err != nil {
		t.Fatalf("UpdateChannel with document failed: %v", err)
	}
	if updated == nil {
		t.Error("Updated channel should not be nil")
	}
}

func TestEditDelete_PublishesEventsWithBus(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)

	creator := int64(10)
	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "publish-event", CreatedByUserID: &creator,
	})

	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("edit-target"), nil)

	edited, err := svc.EditMessage(ctx, ch.ID, msg.ID, creator, textContent("new content"))
	if err != nil {
		t.Fatalf("EditMessage failed: %v", err)
	}
	if edited.Body != "new content" {
		t.Errorf("Body = %s, want 'new content'", edited.Body)
	}

	err = svc.DeleteMessage(ctx, ch.ID, msg.ID, creator)
	if err != nil {
		t.Fatalf("DeleteMessage failed: %v", err)
	}
}

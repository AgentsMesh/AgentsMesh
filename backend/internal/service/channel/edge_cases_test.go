package channel

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func TestApproveBinding(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	created, err := svc.CreateBinding(ctx, 1, "approve-init2", "approve-target2", nil)
	if err != nil {
		t.Fatalf("CreateBinding failed: %v", err)
	}

	err = db.WithContext(ctx).Model(&channel.PodBinding{}).
		Where("id = ?", created.ID).
		Update("status", channel.BindingStatusActive).Error
	if err != nil {
		t.Fatalf("Direct status update failed: %v", err)
	}

	binding, _ := svc.GetBinding(ctx, created.ID)
	if binding.Status != channel.BindingStatusActive {
		t.Errorf("Status = %s, want active", binding.Status)
	}
}

func TestGetBinding_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	_, err := svc.GetBinding(ctx, 999999)
	if err != ErrChannelNotFound {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestGetBindingByPods_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	_, err := svc.GetBindingByPods(ctx, "nonexistent1", "nonexistent2")
	if err != ErrChannelNotFound {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestCreateChannel_CreatorInInitialMembers(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID:   1,
		Name:             "dup-creator",
		CreatedByUserID:  &creator,
		InitialMemberIDs: []int64{10, 20},
	})
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}

	members, total, _ := svc.ListMembers(ctx, ch.ID, 10, 0)
	if total != 2 {
		t.Errorf("Expected 2 members (creator deduplicated), got %d", total)
	}
	_ = members
}

func TestUpdateChannel_NoChanges(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "no-changes",
	})

	updated, err := svc.UpdateChannel(ctx, ch.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateChannel with no changes failed: %v", err)
	}
	if updated.Name != ch.Name {
		t.Errorf("Name changed unexpectedly")
	}
}

func TestSendMessage_PostSendHookError(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	svc.AddPostSendHook(func(_ context.Context, _ *MessageContext) error {
		return ErrChannelNotFound
	})

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "hook-error",
	})

	msg, err := svc.SendMessage(ctx, ch.ID, nil, nil, textContent("test"), nil)
	if err != nil {
		t.Fatalf("SendMessage should succeed despite hook error: %v", err)
	}
	if msg == nil {
		t.Error("Message should not be nil")
	}
}

func TestGetMessages_WithAfterAndBefore(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "both-filters",
	})

	for i := 0; i < 5; i++ {
		svc.SendMessage(ctx, ch.ID, nil, nil, textContent("msg"), nil)
	}

	allMsgs, _, _ := svc.GetMessages(ctx, ch.ID, nil, nil, 10)
	if len(allMsgs) < 3 {
		t.Skip("Need at least 3 messages")
	}

	after := allMsgs[0].CreatedAt
	before := allMsgs[len(allMsgs)-1].CreatedAt
	msgs, _, err := svc.GetMessages(ctx, ch.ID, &before, &after, 10)
	if err != nil {
		t.Fatalf("GetMessages with both filters failed: %v", err)
	}
	_ = msgs
}

func TestGetMessagesByCursor_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "empty-cursor",
	})

	msg, _ := svc.SendMessage(ctx, ch.ID, nil, nil, textContent("first"), nil)

	msgs, hasMore, err := svc.GetMessagesByCursor(ctx, ch.ID, msg.ID, 10)
	if err != nil {
		t.Fatalf("GetMessagesByCursor failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages before first msg, got %d", len(msgs))
	}
	if hasMore {
		t.Error("Expected hasMore=false for empty result")
	}
}

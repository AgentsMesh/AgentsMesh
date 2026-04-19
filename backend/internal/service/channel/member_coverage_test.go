package channel

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestGetNonMutedMemberUserIDs(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "nonmuted-test", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20, 30},
	})

	svc.SetMemberMuted(ctx, ch.ID, creator, true)

	ids, err := svc.GetNonMutedMemberUserIDs(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetNonMutedMemberUserIDs failed: %v", err)
	}
	for _, id := range ids {
		if id == creator {
			t.Error("Muted creator should not be in non-muted list")
		}
	}
}

func TestListMembers(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "list-members", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20, 30},
	})

	members, total, err := svc.ListMembers(ctx, ch.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	if total != 3 {
		t.Errorf("Expected 3 members, got %d", total)
	}
	if len(members) != 3 {
		t.Errorf("Expected 3 members in slice, got %d", len(members))
	}
}

func TestListMembers_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "list-page", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20, 30, 40},
	})

	members, total, err := svc.ListMembers(ctx, ch.ID, 2, 0)
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	if total != 4 {
		t.Errorf("Expected 4 total, got %d", total)
	}
	if len(members) != 2 {
		t.Errorf("Expected 2 members in page, got %d", len(members))
	}
}

func TestJoinPublicChannel_NonExistentChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	err := svc.JoinPublicChannel(ctx, 99999, 20)
	if err != ErrChannelNotFound {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestInviteMembers_NonExistentChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "inv-err", CreatedByUserID: &creator,
	})

	// Creator is a member, but after delete the GetChannel call inside InviteMembers fails
	svc.DeleteChannel(ctx, ch.ID)
	err := svc.InviteMembers(ctx, ch.ID, creator, []int64{20})
	if err == nil {
		t.Error("Expected error when channel is deleted")
	}
}

func TestLeaveUserChannel_NonExistentChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	// Non-creator role means it won't hit the creator check, goes straight to GetChannel
	err := svc.LeaveUserChannel(ctx, 99999, 20)
	if err != ErrChannelNotFound {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestRemoveMember_NonExistentChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	// Self-removal bypasses creator check, hits GetChannel
	err := svc.RemoveMember(ctx, 99999, 20, 20)
	if err != ErrChannelNotFound {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestMarkRead_NonExistentChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	err := svc.MarkRead(ctx, 99999, 10, 1)
	if err != ErrChannelNotFound {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestJoinChannel_PodNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	svc.SetPodCreatorResolver(&mockPodCreatorResolver{
		pods: map[string]*agentpod.Pod{},
	})

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "pod-not-found",
	})

	err := svc.JoinChannel(ctx, ch.ID, "nonexistent-pod")
	if err != nil {
		t.Fatalf("JoinChannel should succeed even if pod not found: %v", err)
	}
}

func TestJoinChannel_NilResolver(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "nil-resolver",
	})

	err := svc.JoinChannel(ctx, ch.ID, "any-pod")
	if err != nil {
		t.Fatalf("JoinChannel with nil resolver should succeed: %v", err)
	}
}

func TestGetMessages_AfterOnly(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "after-test"})

	for i := 0; i < 5; i++ {
		svc.SendMessage(ctx, ch.ID, nil, nil, textContent("msg"), nil)
	}

	allMsgs, _, _ := svc.GetMessages(ctx, ch.ID, nil, nil, 10)
	if len(allMsgs) < 3 {
		t.Skip("Need at least 3 messages")
	}

	after := allMsgs[1].CreatedAt
	msgs, hasMore, err := svc.GetMessages(ctx, ch.ID, nil, &after, 10)
	if err != nil {
		t.Fatalf("GetMessages with after failed: %v", err)
	}
	if hasMore {
		t.Error("Expected hasMore=false with large limit")
	}
	for _, msg := range msgs {
		if !msg.CreatedAt.After(after) {
			t.Errorf("Message %d created_at %v should be after %v", msg.ID, msg.CreatedAt, after)
		}
	}
}

func TestGetMessagesMentioning_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "empty-mention"})

	msgs, hasMore, err := svc.GetMessagesMentioning(ctx, ch.ID, "nonexistent-pod", 10)
	if err != nil {
		t.Fatalf("GetMessagesMentioning failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(msgs))
	}
	if hasMore {
		t.Error("Expected hasMore=false for empty result")
	}
}

func TestGetRecentMessages_NonExistentChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	msgs, err := svc.GetRecentMessages(ctx, 99999, 10)
	if err != nil {
		t.Fatalf("GetRecentMessages should not error for empty result: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(msgs))
	}
}

func TestPublishMemberEvent_NilEventBus(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "nil-eb", CreatedByUserID: &creator,
	})

	// No eventBus set — should not panic
	err := svc.JoinPublicChannel(ctx, ch.ID, 20)
	if err != nil {
		t.Fatalf("JoinPublicChannel failed: %v", err)
	}
}

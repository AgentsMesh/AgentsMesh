package channel

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func TestMarkRead_PublicChannelAutoJoins(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)
	newUser := int64(20)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "markread-pub", CreatedByUserID: &creator,
	})

	// Send a message so there's something to mark as read
	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("hello"), nil)

	// newUser marks read on public channel → auto-joins
	err := svc.MarkRead(ctx, ch.ID, newUser, msg.ID)
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	ok, _ := svc.IsMember(ctx, ch.ID, newUser)
	if !ok {
		t.Error("User should be auto-joined after MarkRead on public channel")
	}
}

func TestMarkRead_PrivateChannelRejectsNonMember(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "markread-priv", CreatedByUserID: &creator,
		Visibility: channel.VisibilityPrivate,
	})

	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("hello"), nil)

	err := svc.MarkRead(ctx, ch.ID, 99, msg.ID)
	if err != ErrNotMember {
		t.Errorf("Expected ErrNotMember for non-member MarkRead on private channel, got %v", err)
	}
}

func TestMarkRead_CursorOnlyForward(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "cursor-fwd", CreatedByUserID: &creator,
	})

	msg1, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("first"), nil)
	msg2, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("second"), nil)

	// Mark to msg2
	svc.MarkRead(ctx, ch.ID, creator, msg2.ID)

	// Try to move cursor backward to msg1
	svc.MarkRead(ctx, ch.ID, creator, msg1.ID)

	// Unread count should be 0 (cursor stayed at msg2, not regressed to msg1)
	counts, _ := svc.GetUnreadCounts(ctx, creator)
	if count, exists := counts[ch.ID]; exists && count > 0 {
		t.Errorf("Expected 0 unread after forward mark, got %d (cursor may have regressed)", count)
	}
}

func TestGetChannelForUser(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "for-user", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20},
	})

	t.Run("member gets is_member=true", func(t *testing.T) {
		result, err := svc.GetChannelForUser(ctx, ch.ID, creator)
		if err != nil {
			t.Fatalf("GetChannelForUser failed: %v", err)
		}
		if !result.IsMember {
			t.Error("Creator should have is_member=true")
		}
		if result.MemberCount != 2 {
			t.Errorf("Expected member_count=2, got %d", result.MemberCount)
		}
		if result.AgentCount != 0 {
			t.Errorf("Expected agent_count=0 (no pods joined), got %d", result.AgentCount)
		}
	})

	t.Run("non-member gets is_member=false", func(t *testing.T) {
		result, err := svc.GetChannelForUser(ctx, ch.ID, 99)
		if err != nil {
			t.Fatalf("GetChannelForUser failed: %v", err)
		}
		if result.IsMember {
			t.Error("Non-member should have is_member=false")
		}
	})
}

func TestGetChannelForUser_AgentCount(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "agent-count", CreatedByUserID: &creator,
	})

	// Channel starts with 1 member (creator) and 0 agents.
	result, _ := svc.GetChannelForUser(ctx, ch.ID, creator)
	if result.MemberCount != 1 {
		t.Errorf("Expected member_count=1, got %d", result.MemberCount)
	}
	if result.AgentCount != 0 {
		t.Errorf("Expected agent_count=0, got %d", result.AgentCount)
	}

	// Join two pods (pod table absent is fine; channel_pods row gets created
	// regardless — ensurePodCreatorIsMember silently no-ops on missing pods).
	if err := svc.JoinChannel(ctx, ch.ID, "pod-1"); err != nil {
		t.Fatalf("JoinChannel pod-1 failed: %v", err)
	}
	if err := svc.JoinChannel(ctx, ch.ID, "pod-2"); err != nil {
		t.Fatalf("JoinChannel pod-2 failed: %v", err)
	}

	result, _ = svc.GetChannelForUser(ctx, ch.ID, creator)
	if result.AgentCount != 2 {
		t.Errorf("After 2 JoinChannel calls, expected agent_count=2, got %d", result.AgentCount)
	}
	if result.MemberCount != 1 {
		t.Errorf("member_count must stay user-only (=1) after agent joins, got %d", result.MemberCount)
	}

	// Leave one pod, agent_count should drop to 1.
	if err := svc.LeaveChannel(ctx, ch.ID, "pod-1"); err != nil {
		t.Fatalf("LeaveChannel pod-1 failed: %v", err)
	}
	result, _ = svc.GetChannelForUser(ctx, ch.ID, creator)
	if result.AgentCount != 1 {
		t.Errorf("After leave, expected agent_count=1, got %d", result.AgentCount)
	}
}

func TestGetUnreadCounts(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)
	member := int64(20)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "unread-test", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{member},
	})

	// New member should have 0 unread (cursor initialized to latest)
	counts, _ := svc.GetUnreadCounts(ctx, member)
	if count := counts[ch.ID]; count != 0 {
		t.Errorf("New member should have 0 unread, got %d", count)
	}

	// Creator sends 3 messages
	for i := 0; i < 3; i++ {
		svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("msg"), nil)
	}

	// Member should have 3 unread
	counts, _ = svc.GetUnreadCounts(ctx, member)
	if counts[ch.ID] != 3 {
		t.Errorf("Expected 3 unread, got %d", counts[ch.ID])
	}

	// Non-member should have 0
	counts, _ = svc.GetUnreadCounts(ctx, 99)
	if len(counts) != 0 {
		t.Errorf("Non-member should have 0 channels in unread counts, got %d", len(counts))
	}
}

func TestEditMessage(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "edit-test", CreatedByUserID: &creator,
	})

	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("original"), nil)

	t.Run("sender can edit", func(t *testing.T) {
		edited, err := svc.EditMessage(ctx, ch.ID, msg.ID, creator, textContent("updated"))
		if err != nil {
			t.Fatalf("EditMessage failed: %v", err)
		}
		if edited.Body != "updated" {
			t.Errorf("Content = %s, want updated", edited.Body)
		}
	})

	t.Run("non-sender cannot edit", func(t *testing.T) {
		_, err := svc.EditMessage(ctx, ch.ID, msg.ID, 99, textContent("hack"))
		if err != ErrNotMessageSender {
			t.Errorf("Expected ErrNotMessageSender, got %v", err)
		}
	})

	t.Run("archived channel rejects edit", func(t *testing.T) {
		svc.ArchiveChannel(ctx, ch.ID)
		_, err := svc.EditMessage(ctx, ch.ID, msg.ID, creator, textContent("fail"))
		if err != ErrChannelArchived {
			t.Errorf("Expected ErrChannelArchived, got %v", err)
		}
		svc.UnarchiveChannel(ctx, ch.ID)
	})
}

func TestDeleteMessage(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "delete-test", CreatedByUserID: &creator,
	})

	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("to-delete"), nil)

	t.Run("sender can delete", func(t *testing.T) {
		err := svc.DeleteMessage(ctx, ch.ID, msg.ID, creator)
		if err != nil {
			t.Fatalf("DeleteMessage failed: %v", err)
		}
	})

	t.Run("non-sender cannot delete", func(t *testing.T) {
		msg2, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("another"), nil)
		err := svc.DeleteMessage(ctx, ch.ID, msg2.ID, 99)
		if err != ErrNotMessageSender {
			t.Errorf("Expected ErrNotMessageSender, got %v", err)
		}
	})
}

func TestValidateOrgMembers_WithNilLookup(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	// Without UserLookup, validateOrgMembers returns input as-is
	result := svc.validateOrgMembers(ctx, 1, []int64{10, 20, 30})
	if len(result) != 3 {
		t.Errorf("Expected 3 members (graceful degradation), got %d", len(result))
	}
}

func TestValidateOrgMembers_EmptyInput(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	result := svc.validateOrgMembers(ctx, 1, nil)
	if result != nil {
		t.Errorf("Expected nil for empty input, got %v", result)
	}

	result = svc.validateOrgMembers(ctx, 1, []int64{})
	if len(result) != 0 {
		t.Errorf("Expected empty for empty input, got %v", result)
	}
}

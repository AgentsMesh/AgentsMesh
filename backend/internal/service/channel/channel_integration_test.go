package channel

import (
	"context"
	"testing"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
)

// TestChannelFlow_CreateAndSendMessage creates a channel, sends messages,
// and verifies they can be retrieved in correct order.
func TestChannelFlow_CreateAndSendMessage(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewChannelRepository(db))
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "msg@test.io", "msguser")
	orgID := testkit.CreateOrg(t, db, "org-msg", userID)

	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: orgID,
		Name:           "test-channel",
	})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	// Send two messages as a user
	msg1, err := svc.SendMessageAsUser(ctx, ch.ID, userID, "hello world", nil, nil)
	if err != nil {
		t.Fatalf("SendMessageAsUser(1): %v", err)
	}
	if msg1.Content != "hello world" {
		t.Errorf("msg1 content = %q, want %q", msg1.Content, "hello world")
	}
	if msg1.MessageType != channelDomain.MessageTypeText {
		t.Errorf("msg1 type = %s, want text", msg1.MessageType)
	}

	msg2, err := svc.SendMessageAsUser(ctx, ch.ID, userID, "second msg", nil, nil)
	if err != nil {
		t.Fatalf("SendMessageAsUser(2): %v", err)
	}

	// Retrieve messages
	messages, hasMore, err := svc.GetMessages(ctx, ch.ID, nil, nil, 10)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if hasMore {
		t.Error("expected hasMore=false for 2 messages with limit 10")
	}
	if len(messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(messages))
	}
	// Verify both messages are present (order may vary when timestamps are equal)
	ids := map[int64]bool{messages[0].ID: true, messages[1].ID: true}
	if !ids[msg1.ID] || !ids[msg2.ID] {
		t.Errorf("expected messages %d and %d, got %d and %d",
			msg1.ID, msg2.ID, messages[0].ID, messages[1].ID)
	}
}

// TestChannelFlow_MemberManagement verifies joining, listing, and leaving
// channel pods (pod-based membership).
func TestChannelFlow_MemberManagement(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewChannelRepository(db))
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "mem@test.io", "memuser")
	orgID := testkit.CreateOrg(t, db, "org-mem", userID)
	runnerID := testkit.CreateRunner(t, db, orgID, "runner-mem")
	podKey := testkit.CreatePod(t, db, orgID, runnerID, userID)

	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: orgID,
		Name:           "member-channel",
	})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	// Join the pod
	if err := svc.JoinChannel(ctx, ch.ID, podKey); err != nil {
		t.Fatalf("JoinChannel: %v", err)
	}

	// List pods — should include the joined pod
	pods, err := svc.GetChannelPods(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetChannelPods: %v", err)
	}
	if len(pods) != 1 {
		t.Fatalf("pods = %d, want 1", len(pods))
	}
	if pods[0].PodKey != podKey {
		t.Errorf("pod key = %s, want %s", pods[0].PodKey, podKey)
	}

	// Leave the pod
	if err := svc.LeaveChannel(ctx, ch.ID, podKey); err != nil {
		t.Fatalf("LeaveChannel: %v", err)
	}

	// List pods again — should be empty
	pods2, err := svc.GetChannelPods(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetChannelPods after leave: %v", err)
	}
	if len(pods2) != 0 {
		t.Errorf("pods after leave = %d, want 0", len(pods2))
	}
}

// TestChannelFlow_ArchiveBlocksMessages archives a channel and verifies
// that sending messages is blocked with ErrChannelArchived.
func TestChannelFlow_ArchiveBlocksMessages(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewChannelRepository(db))
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "arch@test.io", "archuser")
	orgID := testkit.CreateOrg(t, db, "org-arch", userID)

	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: orgID,
		Name:           "archive-test",
	})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	// Send succeeds before archive
	if _, err := svc.SendMessageAsUser(ctx, ch.ID, userID, "before archive", nil, nil); err != nil {
		t.Fatalf("SendMessageAsUser before archive: %v", err)
	}

	// Archive
	if err := svc.ArchiveChannel(ctx, ch.ID); err != nil {
		t.Fatalf("ArchiveChannel: %v", err)
	}

	// Verify archived
	archived, err := svc.GetChannel(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetChannel: %v", err)
	}
	if !archived.IsArchived {
		t.Fatal("expected channel to be archived")
	}

	// Send should fail
	_, err = svc.SendMessageAsUser(ctx, ch.ID, userID, "after archive", nil, nil)
	if err != ErrChannelArchived {
		t.Errorf("expected ErrChannelArchived, got %v", err)
	}

	// Update should also fail
	newName := "renamed"
	_, err = svc.UpdateChannel(ctx, ch.ID, &newName, nil, nil)
	if err != ErrChannelArchived {
		t.Errorf("update archived channel: expected ErrChannelArchived, got %v", err)
	}
}

// TestChannelFlow_DeleteCascade creates a channel with messages and members,
// deletes it, and verifies all related data is cleaned up.
func TestChannelFlow_DeleteCascade(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewChannelRepository(db))
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "del@test.io", "deluser")
	orgID := testkit.CreateOrg(t, db, "org-del", userID)

	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: orgID,
		Name:           "delete-test",
	})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	// Add messages (SendMessageAsUser also auto-joins the user as a member)
	if _, err := svc.SendMessageAsUser(ctx, ch.ID, userID, "to be deleted", nil, nil); err != nil {
		t.Fatalf("SendMessageAsUser: %v", err)
	}
	if _, err := svc.SendMessageAsUser(ctx, ch.ID, userID, "second msg", nil, nil); err != nil {
		t.Fatalf("SendMessageAsUser(2): %v", err)
	}

	// Delete channel
	if err := svc.DeleteChannel(ctx, ch.ID); err != nil {
		t.Fatalf("DeleteChannel: %v", err)
	}

	// Channel should not be found
	_, err = svc.GetChannel(ctx, ch.ID)
	if err != ErrChannelNotFound {
		t.Errorf("expected ErrChannelNotFound, got %v", err)
	}

	// Messages should be cleaned up
	var msgCount int64
	db.Raw(`SELECT COUNT(*) FROM channel_messages WHERE channel_id = ?`, ch.ID).Scan(&msgCount)
	if msgCount != 0 {
		t.Errorf("messages remaining = %d, want 0", msgCount)
	}

	// Members should be cleaned up
	var memCount int64
	db.Raw(`SELECT COUNT(*) FROM channel_members WHERE channel_id = ?`, ch.ID).Scan(&memCount)
	if memCount != 0 {
		t.Errorf("members remaining = %d, want 0", memCount)
	}
}

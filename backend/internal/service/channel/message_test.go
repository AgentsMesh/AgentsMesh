package channel

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func TestSendMessage(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "msg-test"})

	t.Run("send text message", func(t *testing.T) {
		podKey := "test-pod"
		msg, err := svc.SendMessage(ctx, created.ID, &podKey, nil, channel.MessageTypeText, "Hello", channel.MessageMetadata{})
		if err != nil || msg.Content != "Hello" || msg.MessageType != channel.MessageTypeText {
			t.Errorf("SendMessage failed: %v", err)
		}
	})

	t.Run("send to archived channel", func(t *testing.T) {
		svc.ArchiveChannel(ctx, created.ID)
		_, err := svc.SendMessage(ctx, created.ID, nil, nil, channel.MessageTypeText, "Fail", channel.MessageMetadata{})
		if err != ErrChannelArchived {
			t.Errorf("Expected ErrChannelArchived, got %v", err)
		}
	})

	t.Run("send to non-existent channel", func(t *testing.T) {
		if _, err := svc.SendMessage(ctx, 99999, nil, nil, channel.MessageTypeText, "Fail", channel.MessageMetadata{}); err == nil {
			t.Error("Expected error for non-existent channel")
		}
	})
}

func TestGetMessages(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "msgs-test"})

	for i := 0; i < 5; i++ {
		svc.SendMessage(ctx, ch.ID, nil, nil, channel.MessageTypeText, "Msg"+string(rune('0'+i)), channel.MessageMetadata{})
		time.Sleep(10 * time.Millisecond)
	}

	t.Run("get all messages", func(t *testing.T) {
		messages, err := svc.GetMessages(ctx, ch.ID, nil, 10)
		if err != nil || len(messages) != 5 {
			t.Errorf("GetMessages failed: %v, count=%d", err, len(messages))
		}
	})

	t.Run("with limit", func(t *testing.T) {
		messages, _ := svc.GetMessages(ctx, ch.ID, nil, 3)
		if len(messages) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(messages))
		}
	})

	t.Run("with before filter", func(t *testing.T) {
		allMsgs, _ := svc.GetMessages(ctx, ch.ID, nil, 10)
		if len(allMsgs) >= 3 {
			before := allMsgs[2].CreatedAt
			msgs, err := svc.GetMessages(ctx, ch.ID, &before, 10)
			if err != nil {
				t.Fatalf("GetMessages failed: %v", err)
			}
			t.Logf("All: %d, before filter: %d", len(allMsgs), len(msgs))
		}
	})
}

func TestEnhancedMessageService(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "enhanced"})

	t.Run("send system message", func(t *testing.T) {
		msg, err := svc.SendSystemMessage(ctx, ch.ID, "System notification")
		if err != nil || msg.MessageType != channel.MessageTypeSystem {
			t.Errorf("SendSystemMessage failed: %v", err)
		}
	})

	t.Run("send message as user", func(t *testing.T) {
		msg, err := svc.SendMessageAsUser(ctx, ch.ID, 1, "User message", channel.MessageMetadata{})
		if err != nil || msg.SenderUserID == nil || *msg.SenderUserID != 1 {
			t.Error("SenderUserID not set correctly")
		}
	})

	t.Run("send message as pod", func(t *testing.T) {
		msg, err := svc.SendMessageAsPod(ctx, ch.ID, "test-pod", "Agent message", channel.MessageMetadata{})
		if err != nil || msg.SenderPod == nil || *msg.SenderPod != "test-pod" {
			t.Error("SenderPod not set correctly")
		}
	})

	t.Run("get messages mentioning", func(t *testing.T) {
		svc.SendMessage(ctx, ch.ID, nil, nil, channel.MessageTypeText, "@mention-pod hello", channel.MessageMetadata{})
		messages, err := svc.GetMessagesMentioning(ctx, ch.ID, "mention-pod", 10)
		if err != nil || len(messages) != 1 {
			t.Errorf("GetMessagesMentioning failed: %v, count=%d", err, len(messages))
		}
	})

	t.Run("get recent messages", func(t *testing.T) {
		messages, err := svc.GetRecentMessages(ctx, ch.ID, 5)
		if err != nil || len(messages) == 0 {
			t.Errorf("GetRecentMessages failed: %v", err)
		}
	})
}

func TestSendMessage_WithEventBus(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "test-channel"})
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}

	// Test sending message without eventBus (should still work)
	senderPod := "test-pod"
	msg, err := svc.SendMessage(ctx, ch.ID, &senderPod, nil, "text", "Test message", nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msg.Content != "Test message" {
		t.Errorf("Content = %s, want Test message", msg.Content)
	}
}

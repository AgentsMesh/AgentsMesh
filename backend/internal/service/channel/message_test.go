package channel

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func textContent(text string) channel.MessageContent {
	return channel.MessageContent{
		Kind: "text",
		Blocks: []channel.Block{{
			Type:     "paragraph",
			Elements: []channel.InlineElement{{Type: channel.InlineText, Text: text}},
		}},
	}
}

func TestSendMessage(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "msg-test"})

	t.Run("send text message", func(t *testing.T) {
		podKey := "test-pod"
		msg, err := svc.SendMessage(ctx, created.ID, &podKey, nil, textContent("Hello"), nil)
		if err != nil || msg.Body != "Hello" || msg.MessageType != channel.MessageTypeText {
			t.Errorf("SendMessage failed: %v", err)
		}
	})

	t.Run("send to archived channel", func(t *testing.T) {
		svc.ArchiveChannel(ctx, created.ID)
		_, err := svc.SendMessage(ctx, created.ID, nil, nil, textContent("Fail"), nil)
		if err != ErrChannelArchived {
			t.Errorf("Expected ErrChannelArchived, got %v", err)
		}
	})

	t.Run("send to non-existent channel", func(t *testing.T) {
		if _, err := svc.SendMessage(ctx, 99999, nil, nil, textContent("Fail"), nil); err == nil {
			t.Error("Expected error for non-existent channel")
		}
	})
}

func TestGetMessages(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "msgs-test"})

	for i := 0; i < 5; i++ {
		svc.SendMessage(ctx, ch.ID, nil, nil, textContent("Msg"+string(rune('0'+i))), nil)
		time.Sleep(10 * time.Millisecond)
	}

	t.Run("get all messages", func(t *testing.T) {
		messages, hasMore, err := svc.GetMessages(ctx, ch.ID, nil, nil, 10)
		if err != nil || len(messages) != 5 {
			t.Errorf("GetMessages failed: %v, count=%d", err, len(messages))
		}
		if hasMore {
			t.Error("Expected hasMore=false for 5 messages with limit=10")
		}
	})

	t.Run("with limit", func(t *testing.T) {
		messages, hasMore, _ := svc.GetMessages(ctx, ch.ID, nil, nil, 3)
		if len(messages) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(messages))
		}
		if !hasMore {
			t.Error("Expected hasMore=true for 5 messages with limit=3")
		}
	})

	t.Run("exact limit boundary", func(t *testing.T) {
		// 5 messages with limit=5 → hasMore=false (no extra message returned)
		messages, hasMore, err := svc.GetMessages(ctx, ch.ID, nil, nil, 5)
		if err != nil {
			t.Fatalf("GetMessages failed: %v", err)
		}
		if len(messages) != 5 {
			t.Errorf("Expected 5 messages, got %d", len(messages))
		}
		if hasMore {
			t.Error("Expected hasMore=false when exactly limit messages exist")
		}
	})

	t.Run("limit=1", func(t *testing.T) {
		messages, hasMore, err := svc.GetMessages(ctx, ch.ID, nil, nil, 1)
		if err != nil {
			t.Fatalf("GetMessages failed: %v", err)
		}
		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}
		if !hasMore {
			t.Error("Expected hasMore=true for 5 messages with limit=1")
		}
	})

	t.Run("empty channel", func(t *testing.T) {
		emptyCh, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "empty-test"})
		messages, hasMore, err := svc.GetMessages(ctx, emptyCh.ID, nil, nil, 10)
		if err != nil {
			t.Fatalf("GetMessages failed: %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("Expected 0 messages, got %d", len(messages))
		}
		if hasMore {
			t.Error("Expected hasMore=false for empty channel")
		}
	})

	t.Run("cursor-based pagination hasMore", func(t *testing.T) {
		allMsgs, _, _ := svc.GetMessages(ctx, ch.ID, nil, nil, 10)
		if len(allMsgs) < 3 {
			t.Skip("Need at least 3 messages")
		}
		// Use the 3rd message as cursor — should find 2 older messages
		msgs, hasMore, err := svc.GetMessagesByCursor(ctx, ch.ID, allMsgs[2].ID, 10)
		if err != nil {
			t.Fatalf("GetMessagesByCursor failed: %v", err)
		}
		if len(msgs) != 2 {
			t.Errorf("Expected 2 messages before cursor, got %d", len(msgs))
		}
		if hasMore {
			t.Error("Expected hasMore=false for 2 messages with limit=10")
		}

		// With limit=1 — should find 1 and hasMore=true
		msgs2, hasMore2, _ := svc.GetMessagesByCursor(ctx, ch.ID, allMsgs[2].ID, 1)
		if len(msgs2) != 1 {
			t.Errorf("Expected 1 message, got %d", len(msgs2))
		}
		if !hasMore2 {
			t.Error("Expected hasMore=true for 2 messages with limit=1")
		}
	})

	t.Run("with before filter", func(t *testing.T) {
		allMsgs, _, _ := svc.GetMessages(ctx, ch.ID, nil, nil, 10)
		if len(allMsgs) >= 3 {
			before := allMsgs[2].CreatedAt
			msgs, _, err := svc.GetMessages(ctx, ch.ID, &before, nil, 10)
			if err != nil {
				t.Fatalf("GetMessages failed: %v", err)
			}
			t.Logf("All: %d, before filter: %d", len(allMsgs), len(msgs))
		}
	})
}

func TestEnhancedMessageService(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "enhanced"})

	t.Run("send system message", func(t *testing.T) {
		msg, err := svc.SendSystemMessage(ctx, ch.ID, "System notification")
		if err != nil || msg.MessageType != channel.MessageTypeSystem {
			t.Errorf("SendSystemMessage failed: %v", err)
		}
	})

	t.Run("send message as user", func(t *testing.T) {
		msg, err := svc.SendMessageAsUser(ctx, ch.ID, 1, textContent("User message"))
		if err != nil || msg.SenderUserID == nil || *msg.SenderUserID != 1 {
			t.Error("SenderUserID not set correctly")
		}
	})

	t.Run("send message as pod", func(t *testing.T) {
		msg, err := svc.SendMessageAsPod(ctx, ch.ID, "test-pod", textContent("Agent message"))
		if err != nil || msg.SenderPod == nil || *msg.SenderPod != "test-pod" {
			t.Error("SenderPod not set correctly")
		}
	})

	t.Run("get messages mentioning", func(t *testing.T) {
		content := textContent("hello structured")
		content.Blocks[0].Elements = append(content.Blocks[0].Elements, channel.InlineElement{
			Type:       channel.InlineMention,
			EntityType: channel.EntityPod,
			EntityKey:  "mention-pod",
		})
		svc.SendMessage(ctx, ch.ID, nil, nil, content, nil)
		messages, _, err := svc.GetMessagesMentioning(ctx, ch.ID, "mention-pod", 10)
		if err != nil || len(messages) < 1 {
			t.Errorf("GetMessagesMentioning failed: %v, count=%d (want >=1)", err, len(messages))
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
	svc := newTestService(db)
	ctx := context.Background()

	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "test-channel"})
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}

	senderPod := "test-pod"
	msg, err := svc.SendMessage(ctx, ch.ID, &senderPod, nil, textContent("Test message"), nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msg.Body != "Test message" {
		t.Errorf("Body = %s, want Test message", msg.Body)
	}
}

package agent

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

func TestGetMessages(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	// Send multiple messages to the same receiver
	for i := 0; i < 5; i++ {
		content := agent.MessageContent{"index": i}
		svc.SendMessage(ctx, "sender", "receiver", "text", content, nil, nil)
	}

	t.Run("get all messages for receiver", func(t *testing.T) {
		messages, err := svc.GetMessages(ctx, "receiver", false, nil, 10, 0)
		if err != nil {
			t.Fatalf("GetMessages failed: %v", err)
		}
		if len(messages) != 5 {
			t.Errorf("Messages count = %d, want 5", len(messages))
		}
	})

	t.Run("get messages with limit", func(t *testing.T) {
		messages, err := svc.GetMessages(ctx, "receiver", false, nil, 3, 0)
		if err != nil {
			t.Fatalf("GetMessages failed: %v", err)
		}
		if len(messages) != 3 {
			t.Errorf("Messages count = %d, want 3", len(messages))
		}
	})

	t.Run("get messages with offset", func(t *testing.T) {
		messages, err := svc.GetMessages(ctx, "receiver", false, nil, 10, 2)
		if err != nil {
			t.Fatalf("GetMessages failed: %v", err)
		}
		if len(messages) != 3 {
			t.Errorf("Messages count = %d, want 3", len(messages))
		}
	})

	t.Run("get unread messages only", func(t *testing.T) {
		messages, err := svc.GetMessages(ctx, "receiver", true, nil, 10, 0)
		if err != nil {
			t.Fatalf("GetMessages failed: %v", err)
		}
		// All should be pending (unread)
		if len(messages) != 5 {
			t.Errorf("Messages count = %d, want 5", len(messages))
		}
	})

	t.Run("get messages with type filter", func(t *testing.T) {
		// Send a different type message
		svc.SendMessage(ctx, "sender", "receiver", "command", agent.MessageContent{}, nil, nil)

		messages, err := svc.GetMessages(ctx, "receiver", false, []string{"command"}, 10, 0)
		if err != nil {
			t.Fatalf("GetMessages failed: %v", err)
		}
		if len(messages) != 1 {
			t.Errorf("Messages count = %d, want 1", len(messages))
		}
	})
}

func TestGetUnreadMessages(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	// Send messages
	for i := 0; i < 3; i++ {
		svc.SendMessage(ctx, "sender", "receiver", "text", agent.MessageContent{}, nil, nil)
	}

	messages, err := svc.GetUnreadMessages(ctx, "receiver", 10)
	if err != nil {
		t.Fatalf("GetUnreadMessages failed: %v", err)
	}
	if len(messages) != 3 {
		t.Errorf("Messages count = %d, want 3", len(messages))
	}
}

func TestGetUnreadCount(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		svc.SendMessage(ctx, "sender", "receiver", "text", agent.MessageContent{}, nil, nil)
	}

	count, err := svc.GetUnreadCount(ctx, "receiver")
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}
}

func TestGetConversation(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	correlationID := "conv-123"

	// Send messages with same correlation ID
	for i := 0; i < 4; i++ {
		svc.SendMessage(ctx, "s1", "s2", "text", agent.MessageContent{}, &correlationID, nil)
	}

	// Send message with different correlation ID
	otherCorr := "other-conv"
	svc.SendMessage(ctx, "s1", "s2", "text", agent.MessageContent{}, &otherCorr, nil)

	messages, err := svc.GetConversation(ctx, correlationID, 10)
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}
	if len(messages) != 4 {
		t.Errorf("Messages count = %d, want 4", len(messages))
	}
}

func TestGetThread(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	// Send root message
	root, _ := svc.SendMessage(ctx, "s1", "s2", "text", agent.MessageContent{}, nil, nil)

	// Send replies
	for i := 0; i < 3; i++ {
		svc.SendMessage(ctx, "s2", "s1", "text", agent.MessageContent{}, nil, &root.ID)
	}

	t.Run("get thread", func(t *testing.T) {
		thread, err := svc.GetThread(ctx, root.ID)
		if err != nil {
			t.Fatalf("GetThread failed: %v", err)
		}
		// Should have root + 3 replies
		if len(thread) != 4 {
			t.Errorf("Thread length = %d, want 4", len(thread))
		}
	})

	t.Run("get thread for non-existent message", func(t *testing.T) {
		_, err := svc.GetThread(ctx, 99999)
		if err == nil {
			t.Error("Expected error for non-existent message")
		}
	})
}

func TestGetSentMessages(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	// Send multiple messages from same sender
	for i := 0; i < 4; i++ {
		svc.SendMessage(ctx, "sender", "receiver", "text", agent.MessageContent{}, nil, nil)
	}

	messages, err := svc.GetSentMessages(ctx, "sender", 10, 0)
	if err != nil {
		t.Fatalf("GetSentMessages failed: %v", err)
	}
	if len(messages) != 4 {
		t.Errorf("Messages count = %d, want 4", len(messages))
	}
}

func TestGetMessagesBetween(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	// Send messages in both directions
	svc.SendMessage(ctx, "alice", "bob", "text", agent.MessageContent{}, nil, nil)
	svc.SendMessage(ctx, "bob", "alice", "text", agent.MessageContent{}, nil, nil)
	svc.SendMessage(ctx, "alice", "bob", "text", agent.MessageContent{}, nil, nil)

	// Send message to another user
	svc.SendMessage(ctx, "alice", "charlie", "text", agent.MessageContent{}, nil, nil)

	messages, err := svc.GetMessagesBetween(ctx, "alice", "bob", 10)
	if err != nil {
		t.Fatalf("GetMessagesBetween failed: %v", err)
	}
	if len(messages) != 3 {
		t.Errorf("Messages count = %d, want 3", len(messages))
	}
}

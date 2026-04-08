package agent

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

func TestNewMessageService(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)

	if svc == nil {
		t.Error("NewMessageService returned nil")
	}
}

func TestSendMessage(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	t.Run("send basic message", func(t *testing.T) {
		content := agent.MessageContent{
			"text": "Hello, World!",
		}

		msg, err := svc.SendMessage(ctx, "pod-sender", "pod-receiver", "text", content, nil, nil)
		if err != nil {
			t.Fatalf("SendMessage failed: %v", err)
		}

		if msg.SenderPod != "pod-sender" {
			t.Errorf("SenderPod = %s, want pod-sender", msg.SenderPod)
		}
		if msg.ReceiverPod != "pod-receiver" {
			t.Errorf("ReceiverPod = %s, want pod-receiver", msg.ReceiverPod)
		}
		if msg.MessageType != "text" {
			t.Errorf("MessageType = %s, want text", msg.MessageType)
		}
		if msg.Status != agent.MessageStatusPending {
			t.Errorf("Status = %s, want pending", msg.Status)
		}
		if msg.MaxRetries != 3 {
			t.Errorf("MaxRetries = %d, want 3", msg.MaxRetries)
		}
	})

	t.Run("send message with correlation ID", func(t *testing.T) {
		correlationID := "corr-123"
		content := agent.MessageContent{"data": "test"}

		msg, err := svc.SendMessage(ctx, "s1", "s2", "request", content, &correlationID, nil)
		if err != nil {
			t.Fatalf("SendMessage failed: %v", err)
		}

		if msg.CorrelationID == nil || *msg.CorrelationID != correlationID {
			t.Error("CorrelationID not set correctly")
		}
	})

	t.Run("send reply message", func(t *testing.T) {
		// Send original message
		content := agent.MessageContent{"text": "original"}
		original, _ := svc.SendMessage(ctx, "s1", "s2", "text", content, nil, nil)

		// Send reply
		replyContent := agent.MessageContent{"text": "reply"}
		reply, err := svc.SendMessage(ctx, "s2", "s1", "text", replyContent, nil, &original.ID)
		if err != nil {
			t.Fatalf("SendMessage reply failed: %v", err)
		}

		if reply.ParentMessageID == nil || *reply.ParentMessageID != original.ID {
			t.Error("ParentMessageID not set correctly")
		}
	})
}

func TestGetMessage(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	content := agent.MessageContent{"text": "test"}
	created, _ := svc.SendMessage(ctx, "s1", "s2", "text", content, nil, nil)

	t.Run("get existing message", func(t *testing.T) {
		msg, err := svc.GetMessage(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetMessage failed: %v", err)
		}
		if msg.ID != created.ID {
			t.Errorf("ID = %d, want %d", msg.ID, created.ID)
		}
	})

	t.Run("get non-existent message", func(t *testing.T) {
		_, err := svc.GetMessage(ctx, 99999)
		if err == nil {
			t.Error("Expected error for non-existent message")
		}
		if err != ErrMessageNotFound {
			t.Errorf("Expected ErrMessageNotFound, got %v", err)
		}
	})
}

func TestMessageServiceErrors(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrMessageNotFound, "message not found"},
		{ErrNotAuthorized, "not authorized to access this message"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("Error message = %s, want %s", tt.err.Error(), tt.expected)
		}
	}
}

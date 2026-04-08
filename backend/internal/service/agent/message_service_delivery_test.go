package agent

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

func TestMarkRead(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	msg, _ := svc.SendMessage(ctx, "sender", "receiver", "text", agent.MessageContent{}, nil, nil)

	t.Run("mark message as read", func(t *testing.T) {
		err := svc.MarkRead(ctx, msg.ID, "receiver")
		if err != nil {
			t.Fatalf("MarkRead failed: %v", err)
		}

		updated, _ := svc.GetMessage(ctx, msg.ID)
		if updated.Status != agent.MessageStatusRead {
			t.Errorf("Status = %s, want read", updated.Status)
		}
		if updated.ReadAt == nil {
			t.Error("ReadAt should be set")
		}
	})

	t.Run("unauthorized mark read", func(t *testing.T) {
		msg2, _ := svc.SendMessage(ctx, "sender", "receiver", "text", agent.MessageContent{}, nil, nil)
		err := svc.MarkRead(ctx, msg2.ID, "other-pod")
		if err == nil {
			t.Error("Expected error for unauthorized mark read")
		}
		if err != ErrNotAuthorized {
			t.Errorf("Expected ErrNotAuthorized, got %v", err)
		}
	})

	t.Run("mark non-existent message", func(t *testing.T) {
		err := svc.MarkRead(ctx, 99999, "receiver")
		if err == nil {
			t.Error("Expected error for non-existent message")
		}
	})
}

func TestMarkDelivered(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	msg, _ := svc.SendMessage(ctx, "sender", "receiver", "text", agent.MessageContent{}, nil, nil)

	err := svc.MarkDelivered(ctx, msg.ID)
	if err != nil {
		t.Fatalf("MarkDelivered failed: %v", err)
	}

	updated, _ := svc.GetMessage(ctx, msg.ID)
	if updated.Status != agent.MessageStatusDelivered {
		t.Errorf("Status = %s, want delivered", updated.Status)
	}
	if updated.DeliveredAt == nil {
		t.Error("DeliveredAt should be set")
	}
}

func TestMarkAllRead(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	// Send multiple messages
	for i := 0; i < 5; i++ {
		svc.SendMessage(ctx, "sender", "receiver", "text", agent.MessageContent{}, nil, nil)
	}

	affected, err := svc.MarkAllRead(ctx, "receiver")
	if err != nil {
		t.Fatalf("MarkAllRead failed: %v", err)
	}
	if affected != 5 {
		t.Errorf("Affected rows = %d, want 5", affected)
	}

	// Verify all are read
	messages, _ := svc.GetUnreadMessages(ctx, "receiver", 10)
	if len(messages) != 0 {
		t.Errorf("Unread count = %d, want 0", len(messages))
	}
}

func TestDeleteMessage(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	msg, _ := svc.SendMessage(ctx, "sender", "receiver", "text", agent.MessageContent{}, nil, nil)

	t.Run("sender can delete", func(t *testing.T) {
		err := svc.DeleteMessage(ctx, msg.ID, "sender")
		if err != nil {
			t.Fatalf("DeleteMessage failed: %v", err)
		}

		// Verify deleted
		_, err = svc.GetMessage(ctx, msg.ID)
		if err == nil {
			t.Error("Message should be deleted")
		}
	})

	t.Run("receiver cannot delete", func(t *testing.T) {
		msg2, _ := svc.SendMessage(ctx, "sender", "receiver", "text", agent.MessageContent{}, nil, nil)
		err := svc.DeleteMessage(ctx, msg2.ID, "receiver")
		if err == nil {
			t.Error("Expected error for unauthorized delete")
		}
		if err != ErrNotAuthorized {
			t.Errorf("Expected ErrNotAuthorized, got %v", err)
		}
	})

	t.Run("delete non-existent message", func(t *testing.T) {
		err := svc.DeleteMessage(ctx, 99999, "sender")
		if err == nil {
			t.Error("Expected error for non-existent message")
		}
	})
}

func TestGetPendingRetries(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	// Create a message with failed status and next_retry_at
	msg, _ := svc.SendMessage(ctx, "s1", "s2", "text", agent.MessageContent{}, nil, nil)

	// Update to failed status with next_retry_at
	nextRetry := time.Now().Add(-1 * time.Hour)
	db.Model(&agent.AgentMessage{}).Where("id = ?", msg.ID).Updates(map[string]interface{}{
		"status":        agent.MessageStatusFailed,
		"next_retry_at": nextRetry,
	})

	messages, err := svc.GetPendingRetries(ctx, time.Now(), 10)
	if err != nil {
		t.Fatalf("GetPendingRetries failed: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(messages))
	}
}

func TestRecordDeliveryFailure(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	t.Run("first failure schedules retry", func(t *testing.T) {
		msg, _ := svc.SendMessage(ctx, "s1", "s2", "text", agent.MessageContent{}, nil, nil)

		err := svc.RecordDeliveryFailure(ctx, msg.ID, "Connection refused")
		if err != nil {
			t.Fatalf("RecordDeliveryFailure failed: %v", err)
		}

		updated, _ := svc.GetMessage(ctx, msg.ID)
		if updated.Status != agent.MessageStatusFailed {
			t.Errorf("Status = %s, want failed", updated.Status)
		}
		if updated.DeliveryAttempts != 1 {
			t.Errorf("DeliveryAttempts = %d, want 1", updated.DeliveryAttempts)
		}
		if updated.NextRetryAt == nil {
			t.Error("NextRetryAt should be set")
		}
	})

	t.Run("max retries moves to dead letter", func(t *testing.T) {
		msg, _ := svc.SendMessage(ctx, "s1", "s2", "text", agent.MessageContent{}, nil, nil)
		// Set attempts to just below max
		db.Model(&agent.AgentMessage{}).Where("id = ?", msg.ID).Update("delivery_attempts", 2)

		err := svc.RecordDeliveryFailure(ctx, msg.ID, "Max retries exceeded")
		if err != nil {
			t.Fatalf("RecordDeliveryFailure failed: %v", err)
		}

		updated, _ := svc.GetMessage(ctx, msg.ID)
		if updated.Status != agent.MessageStatusDeadLetter {
			t.Errorf("Status = %s, want dead_letter", updated.Status)
		}
		if updated.NextRetryAt != nil {
			t.Error("NextRetryAt should be nil for dead letter")
		}

		// Verify dead letter entry created
		var deadLetter agent.DeadLetterEntry
		err = db.Where("original_message_id = ?", msg.ID).First(&deadLetter).Error
		if err != nil {
			t.Error("Dead letter entry should be created")
		}
	})

	t.Run("non-existent message", func(t *testing.T) {
		err := svc.RecordDeliveryFailure(ctx, 99999, "Error")
		if err == nil {
			t.Error("Expected error for non-existent message")
		}
	})
}

func TestGetDeadLetters(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	// Create messages and move to dead letter
	for i := 0; i < 3; i++ {
		msg, _ := svc.SendMessage(ctx, "s1", "s2", "text", agent.MessageContent{}, nil, nil)
		db.Model(&agent.AgentMessage{}).Where("id = ?", msg.ID).Update("delivery_attempts", 2)
		svc.RecordDeliveryFailure(ctx, msg.ID, "Failed")
	}

	entries, err := svc.GetDeadLetters(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetDeadLetters failed: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("Entries count = %d, want 3", len(entries))
	}
}

func TestReplayDeadLetter(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	// Create message and move to dead letter
	msg, _ := svc.SendMessage(ctx, "s1", "s2", "text", agent.MessageContent{}, nil, nil)
	db.Model(&agent.AgentMessage{}).Where("id = ?", msg.ID).Update("delivery_attempts", 2)
	svc.RecordDeliveryFailure(ctx, msg.ID, "Failed")

	// Get the dead letter entry
	var entry agent.DeadLetterEntry
	db.Where("original_message_id = ?", msg.ID).First(&entry)

	t.Run("replay dead letter", func(t *testing.T) {
		replayed, err := svc.ReplayDeadLetter(ctx, entry.ID)
		if err != nil {
			t.Fatalf("ReplayDeadLetter failed: %v", err)
		}

		if replayed.Status != agent.MessageStatusPending {
			t.Errorf("Status = %s, want pending", replayed.Status)
		}
		if replayed.DeliveryAttempts != 0 {
			t.Errorf("DeliveryAttempts = %d, want 0", replayed.DeliveryAttempts)
		}

		// Verify entry updated
		var updatedEntry agent.DeadLetterEntry
		db.First(&updatedEntry, entry.ID)
		if updatedEntry.ReplayedAt == nil {
			t.Error("ReplayedAt should be set")
		}
		if updatedEntry.ReplayResult == nil || *updatedEntry.ReplayResult != "Replayed successfully" {
			t.Error("ReplayResult should be set")
		}
	})

	t.Run("replay non-existent entry", func(t *testing.T) {
		_, err := svc.ReplayDeadLetter(ctx, 99999)
		if err == nil {
			t.Error("Expected error for non-existent entry")
		}
	})
}

func TestCleanupExpiredMessages(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestMessageService(db)
	ctx := context.Background()

	// Create dead letter entries
	for i := 0; i < 3; i++ {
		msg, _ := svc.SendMessage(ctx, "s1", "s2", "text", agent.MessageContent{}, nil, nil)
		db.Model(&agent.AgentMessage{}).Where("id = ?", msg.ID).Update("delivery_attempts", 2)
		svc.RecordDeliveryFailure(ctx, msg.ID, "Failed")
	}

	// Set all to old date
	oldDate := time.Now().Add(-30 * 24 * time.Hour)
	db.Model(&agent.DeadLetterEntry{}).Where("1=1").Update("moved_at", oldDate)

	affected, err := svc.CleanupExpiredMessages(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		t.Fatalf("CleanupExpiredMessages failed: %v", err)
	}
	if affected != 3 {
		t.Errorf("Affected rows = %d, want 3", affected)
	}

	// Verify all cleaned up
	entries, _ := svc.GetDeadLetters(ctx, 10, 0)
	if len(entries) != 0 {
		t.Errorf("Entries remaining = %d, want 0", len(entries))
	}
}

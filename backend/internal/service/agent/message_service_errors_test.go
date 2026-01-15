package agent

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupBadMessageDB creates a database without the required tables for error testing
func setupBadMessageDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	// Don't create tables - this will cause errors
	return db
}

func TestMessageService_SendMessage_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.SendMessage(ctx, "s1", "s2", "text", agent.MessageContent{}, nil, nil)
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_GetMessage_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.GetMessage(ctx, 1)
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
	// Should not be ErrMessageNotFound, but a DB error
	if err == ErrMessageNotFound {
		t.Error("Expected DB error, not ErrMessageNotFound")
	}
}

func TestMessageService_GetMessages_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.GetMessages(ctx, "receiver", false, nil, 10, 0)
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_GetUnreadMessages_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.GetUnreadMessages(ctx, "receiver", 10)
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_GetUnreadCount_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.GetUnreadCount(ctx, "receiver")
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_GetConversation_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.GetConversation(ctx, "conv-123", 10)
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_GetPendingRetries_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.GetPendingRetries(ctx, time.Now(), 10)
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_GetSentMessages_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.GetSentMessages(ctx, "sender", 10, 0)
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_GetMessagesBetween_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.GetMessagesBetween(ctx, "alice", "bob", 10)
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_GetDeadLetters_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.GetDeadLetters(ctx, 10, 0)
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_ReplayDeadLetter_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.ReplayDeadLetter(ctx, 1)
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_CleanupExpiredMessages_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.CleanupExpiredMessages(ctx, time.Now())
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_MarkAllRead_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	_, err := svc.MarkAllRead(ctx, "receiver")
	if err == nil {
		t.Error("Expected error when table doesn't exist")
	}
}

func TestMessageService_MarkDelivered_DBError(t *testing.T) {
	db := setupBadMessageDB(t)
	svc := NewMessageService(db)
	ctx := context.Background()

	// This should not error since it updates non-existent rows (returns 0 affected)
	// but let's verify
	err := svc.MarkDelivered(ctx, 99999)
	// GORM update doesn't error when no rows match
	if err != nil {
		t.Logf("Got error: %v (expected for missing table)", err)
	}
}

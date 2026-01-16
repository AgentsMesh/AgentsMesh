package agent

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrNotAuthorized   = errors.New("not authorized to access this message")
)

// MessageService handles agent message operations
type MessageService struct {
	db *gorm.DB
}

// NewMessageService creates a new message service
func NewMessageService(db *gorm.DB) *MessageService {
	return &MessageService{db: db}
}

// SendMessage creates and sends a message from one agent to another
func (s *MessageService) SendMessage(ctx context.Context, senderPod, receiverPod, messageType string, content agent.MessageContent, correlationID *string, parentMessageID *int64) (*agent.AgentMessage, error) {
	message := &agent.AgentMessage{
		SenderPod:       senderPod,
		ReceiverPod:     receiverPod,
		MessageType:     messageType,
		Content:         content,
		Status:          agent.MessageStatusPending,
		CorrelationID:   correlationID,
		ParentMessageID: parentMessageID,
		MaxRetries:      3,
	}

	if err := s.db.WithContext(ctx).Create(message).Error; err != nil {
		return nil, err
	}

	return message, nil
}

// GetMessage returns a message by ID
func (s *MessageService) GetMessage(ctx context.Context, messageID int64) (*agent.AgentMessage, error) {
	var message agent.AgentMessage
	if err := s.db.WithContext(ctx).First(&message, messageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	return &message, nil
}

// GetMessages returns messages for a pod
func (s *MessageService) GetMessages(ctx context.Context, podKey string, unreadOnly bool, messageTypes []string, limit, offset int) ([]*agent.AgentMessage, error) {
	query := s.db.WithContext(ctx).Where("receiver_pod = ?", podKey)

	if unreadOnly {
		query = query.Where("status IN ?", []string{agent.MessageStatusPending, agent.MessageStatusDelivered})
	}

	if len(messageTypes) > 0 {
		query = query.Where("message_type IN ?", messageTypes)
	}

	var messages []*agent.AgentMessage
	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

// GetUnreadMessages returns unread messages for a pod
func (s *MessageService) GetUnreadMessages(ctx context.Context, podKey string, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("receiver_pod = ? AND status IN ?", podKey,
			[]string{agent.MessageStatusPending, agent.MessageStatusDelivered}).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// MarkRead marks a message as read
func (s *MessageService) MarkRead(ctx context.Context, messageID int64, podKey string) error {
	message, err := s.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	if message.ReceiverPod != podKey {
		return ErrNotAuthorized
	}

	now := time.Now()
	return s.db.WithContext(ctx).Model(message).Updates(map[string]interface{}{
		"status":  agent.MessageStatusRead,
		"read_at": now,
	}).Error
}

// MarkDelivered marks a message as delivered
func (s *MessageService) MarkDelivered(ctx context.Context, messageID int64) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&agent.AgentMessage{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"status":       agent.MessageStatusDelivered,
			"delivered_at": now,
		}).Error
}

// MarkAllRead marks all messages for a pod as read
func (s *MessageService) MarkAllRead(ctx context.Context, podKey string) (int64, error) {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&agent.AgentMessage{}).
		Where("receiver_pod = ? AND status IN ?", podKey,
			[]string{agent.MessageStatusPending, agent.MessageStatusDelivered}).
		Updates(map[string]interface{}{
			"status":  agent.MessageStatusRead,
			"read_at": now,
		})
	return result.RowsAffected, result.Error
}

// GetUnreadCount returns the count of unread messages for a pod
func (s *MessageService) GetUnreadCount(ctx context.Context, podKey string) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&agent.AgentMessage{}).
		Where("receiver_pod = ? AND status IN ?", podKey,
			[]string{agent.MessageStatusPending, agent.MessageStatusDelivered}).
		Count(&count).Error
	return count, err
}

// GetConversation returns all messages with a correlation ID
func (s *MessageService) GetConversation(ctx context.Context, correlationID string, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("correlation_id = ?", correlationID).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetThread returns a message thread (original + all replies)
func (s *MessageService) GetThread(ctx context.Context, messageID int64) ([]*agent.AgentMessage, error) {
	// Get root message
	root, err := s.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	messages := []*agent.AgentMessage{root}

	// Get replies
	var replies []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("parent_message_id = ?", messageID).
		Order("created_at ASC").
		Find(&replies).Error; err != nil {
		return nil, err
	}

	messages = append(messages, replies...)
	return messages, nil
}

// DeleteMessage soft deletes a message (only sender can delete)
func (s *MessageService) DeleteMessage(ctx context.Context, messageID int64, podKey string) error {
	message, err := s.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	if message.SenderPod != podKey {
		return ErrNotAuthorized
	}

	return s.db.WithContext(ctx).Delete(message).Error
}

// GetPendingRetries returns messages that need retry
func (s *MessageService) GetPendingRetries(ctx context.Context, before time.Time, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?",
			agent.MessageStatusFailed, before).
		Order("next_retry_at ASC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// RecordDeliveryFailure records a delivery failure and schedules retry
func (s *MessageService) RecordDeliveryFailure(ctx context.Context, messageID int64, errorMsg string) error {
	message, err := s.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	now := time.Now()
	message.DeliveryAttempts++
	message.LastDeliveryAttempt = &now
	message.DeliveryError = &errorMsg

	if message.DeliveryAttempts >= message.MaxRetries {
		message.Status = agent.MessageStatusDeadLetter
		message.NextRetryAt = nil

		// Create dead letter entry
		deadLetter := &agent.DeadLetterEntry{
			OriginalMessageID: message.ID,
			Reason:            errorMsg,
			FinalAttempt:      message.DeliveryAttempts,
			MovedAt:           now,
		}
		if err := s.db.WithContext(ctx).Create(deadLetter).Error; err != nil {
			return err
		}
	} else {
		message.Status = agent.MessageStatusFailed
		// Exponential backoff: 1min, 2min, 4min, etc.
		backoff := time.Duration(1<<uint(message.DeliveryAttempts)) * time.Minute
		nextRetry := now.Add(backoff)
		message.NextRetryAt = &nextRetry
	}

	return s.db.WithContext(ctx).Save(message).Error
}

// GetSentMessages returns messages sent by a pod
func (s *MessageService) GetSentMessages(ctx context.Context, podKey string, limit, offset int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("sender_pod = ?", podKey).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetMessagesBetween returns messages between two pods
func (s *MessageService) GetMessagesBetween(ctx context.Context, podA, podB string, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("(sender_pod = ? AND receiver_pod = ?) OR (sender_pod = ? AND receiver_pod = ?)",
			podA, podB, podB, podA).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetDeadLetters returns dead letter entries for review
func (s *MessageService) GetDeadLetters(ctx context.Context, limit, offset int) ([]*agent.DeadLetterEntry, error) {
	var entries []*agent.DeadLetterEntry
	if err := s.db.WithContext(ctx).
		Preload("OriginalMessage").
		Order("moved_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// ReplayDeadLetter attempts to replay a dead letter message
func (s *MessageService) ReplayDeadLetter(ctx context.Context, entryID int64) (*agent.AgentMessage, error) {
	var entry agent.DeadLetterEntry
	if err := s.db.WithContext(ctx).
		Preload("OriginalMessage").
		First(&entry, entryID).Error; err != nil {
		return nil, err
	}

	// Reset the original message for retry
	now := time.Now()
	entry.OriginalMessage.Status = agent.MessageStatusPending
	entry.OriginalMessage.DeliveryAttempts = 0
	entry.OriginalMessage.NextRetryAt = nil
	entry.OriginalMessage.DeliveryError = nil

	if err := s.db.WithContext(ctx).Save(entry.OriginalMessage).Error; err != nil {
		return nil, err
	}

	// Update dead letter entry
	entry.ReplayedAt = &now
	result := "Replayed successfully"
	entry.ReplayResult = &result
	if err := s.db.WithContext(ctx).Save(&entry).Error; err != nil {
		return nil, err
	}

	return entry.OriginalMessage, nil
}

// CleanupExpiredMessages removes old dead letter entries
func (s *MessageService) CleanupExpiredMessages(ctx context.Context, olderThan time.Time) (int64, error) {
	result := s.db.WithContext(ctx).
		Where("moved_at < ?", olderThan).
		Delete(&agent.DeadLetterEntry{})
	return result.RowsAffected, result.Error
}

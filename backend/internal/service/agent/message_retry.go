package agent

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

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

package agent

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// GetPendingRetries returns messages that need retry
func (s *MessageService) GetPendingRetries(ctx context.Context, before time.Time, limit int) ([]*agent.AgentMessage, error) {
	return s.repo.GetPendingRetries(ctx, before, limit)
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

		slog.Warn("message moved to dead letter", "message_id", messageID, "attempts", message.DeliveryAttempts, "error", errorMsg)

		// Create dead letter entry
		deadLetter := &agent.DeadLetterEntry{
			OriginalMessageID: message.ID,
			Reason:            errorMsg,
			FinalAttempt:      message.DeliveryAttempts,
			MovedAt:           now,
		}
		if err := s.repo.CreateDeadLetter(ctx, deadLetter); err != nil {
			return err
		}
	} else {
		message.Status = agent.MessageStatusFailed
		// Exponential backoff: 1min, 2min, 4min, etc.
		backoff := time.Duration(1<<uint(message.DeliveryAttempts)) * time.Minute
		nextRetry := now.Add(backoff)
		message.NextRetryAt = &nextRetry
	}

	return s.repo.Save(ctx, message)
}

// GetDeadLetters returns dead letter entries for review
func (s *MessageService) GetDeadLetters(ctx context.Context, limit, offset int) ([]*agent.DeadLetterEntry, error) {
	return s.repo.GetDeadLetters(ctx, limit, offset)
}

// ReplayDeadLetter attempts to replay a dead letter message
func (s *MessageService) ReplayDeadLetter(ctx context.Context, entryID int64) (*agent.AgentMessage, error) {
	entry, err := s.repo.GetDeadLetterWithMessage(ctx, entryID)
	if err != nil {
		return nil, err
	}

	// Reset the original message for retry
	now := time.Now()
	entry.OriginalMessage.Status = agent.MessageStatusPending
	entry.OriginalMessage.DeliveryAttempts = 0
	entry.OriginalMessage.NextRetryAt = nil
	entry.OriginalMessage.DeliveryError = nil

	if err := s.repo.Save(ctx, entry.OriginalMessage); err != nil {
		return nil, err
	}

	// Update dead letter entry
	entry.ReplayedAt = &now
	result := "Replayed successfully"
	entry.ReplayResult = &result
	if err := s.repo.SaveDeadLetter(ctx, entry); err != nil {
		return nil, err
	}

	slog.Info("dead letter replayed", "entry_id", entryID, "message_id", entry.OriginalMessage.ID)
	return entry.OriginalMessage, nil
}

// CleanupExpiredMessages removes old dead letter entries
func (s *MessageService) CleanupExpiredMessages(ctx context.Context, olderThan time.Time) (int64, error) {
	return s.repo.CleanupExpiredDeadLetters(ctx, olderThan)
}

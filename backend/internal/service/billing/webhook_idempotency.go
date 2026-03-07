package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Webhook Idempotency
// ===========================================

// ErrWebhookAlreadyProcessed is returned when a webhook event has already been processed
var ErrWebhookAlreadyProcessed = errors.New("webhook event already processed")

// CheckAndMarkWebhookProcessed checks if a webhook event has already been processed.
// If not, it marks it as processed and returns nil.
// If already processed, it returns ErrWebhookAlreadyProcessed.
func (s *Service) CheckAndMarkWebhookProcessed(ctx context.Context, eventID, provider, eventType string) error {
	webhookEvent := &billing.WebhookEvent{
		EventID:     eventID,
		Provider:    provider,
		EventType:   eventType,
		ProcessedAt: time.Now(),
	}

	// Try to insert - if duplicate, will fail due to unique constraint
	err := s.db.WithContext(ctx).Create(webhookEvent).Error
	if err != nil {
		// Check if it's a duplicate key error
		if isDuplicateKeyError(err) {
			return ErrWebhookAlreadyProcessed
		}
		return fmt.Errorf("failed to mark webhook as processed: %w", err)
	}

	return nil
}

// isDuplicateKeyError checks if the error is a duplicate key violation
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// PostgreSQL / SQLite / MySQL duplicate key patterns
	return strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "UNIQUE constraint failed") ||
		strings.Contains(errStr, "Duplicate entry")
}

// DeleteWebhookProcessedMark removes the idempotency record for a webhook event,
// allowing it to be reprocessed. This is used to roll back the mark when the
// handler fails after the mark was written.
func (s *Service) DeleteWebhookProcessedMark(ctx context.Context, eventID, provider string) {
	s.db.WithContext(ctx).
		Where("event_id = ? AND provider = ?", eventID, provider).
		Delete(&billing.WebhookEvent{})
}

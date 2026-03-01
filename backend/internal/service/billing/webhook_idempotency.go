package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
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
	// PostgreSQL duplicate key error
	return contains(errStr, "duplicate key") ||
		contains(errStr, "UNIQUE constraint failed") || // SQLite
		contains(errStr, "Duplicate entry") // MySQL
}

// contains is a simple string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

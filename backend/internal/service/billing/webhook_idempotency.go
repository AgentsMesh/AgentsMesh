package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

var ErrWebhookAlreadyProcessed = fmt.Errorf("webhook event already processed")

func (s *Service) CheckAndMarkWebhookProcessed(ctx context.Context, eventID, provider, eventType string) error {
	webhookEvent := &billing.WebhookEvent{
		EventID:     eventID,
		Provider:    provider,
		EventType:   eventType,
		ProcessedAt: time.Now(),
	}

	err := s.repo.CreateWebhookEvent(ctx, webhookEvent)
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrWebhookAlreadyProcessed
		}
		return fmt.Errorf("failed to mark webhook as processed: %w", err)
	}

	return nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "UNIQUE constraint failed") ||
		strings.Contains(errStr, "Duplicate entry")
}

// DeleteWebhookProcessedMark rolls back the idempotency record so the handler can retry on next delivery.
func (s *Service) DeleteWebhookProcessedMark(ctx context.Context, eventID, provider string) {
	_ = s.repo.DeleteWebhookEvent(ctx, eventID, provider)
}

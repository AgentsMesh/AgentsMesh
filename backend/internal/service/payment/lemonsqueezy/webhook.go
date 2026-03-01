package lemonsqueezy

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
)

// ErrWebhookSecretNotConfigured indicates webhook secret is not configured
var ErrWebhookSecretNotConfigured = fmt.Errorf("webhook secret is not configured - this is required for security")

// HandleWebhook parses and validates a LemonSqueezy webhook
func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*types.WebhookEvent, error) {
	// Verify signature using HMAC-SHA256
	if err := p.verifySignature(payload, signature); err != nil {
		return nil, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	// Parse the webhook payload
	var webhookPayload WebhookPayload
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Use Data.ID as the unique event identifier for idempotency
	// This ensures each webhook event has a truly unique ID
	eventID := webhookPayload.Data.ID
	if eventID == "" {
		// Fallback to event name + timestamp if ID is missing (shouldn't happen)
		eventID = webhookPayload.Meta.EventName + "_" + time.Now().Format("20060102150405.000000")
	}

	result := &types.WebhookEvent{
		EventID:   eventID,
		EventType: webhookPayload.Meta.EventName,
		Provider:  billing.PaymentProviderLemonSqueezy,
	}

	// Extract custom data for order_no
	if webhookPayload.Meta.CustomData != nil {
		if orderNo, ok := webhookPayload.Meta.CustomData["order_no"].(string); ok {
			result.OrderNo = orderNo
		}
	}

	// Parse event data based on type
	switch webhookPayload.Meta.EventName {
	case billing.WebhookEventLSOrderCreated:
		p.parseOrderEvent(&webhookPayload, result)

	case billing.WebhookEventLSSubscriptionCreated:
		p.parseSubscriptionCreatedEvent(&webhookPayload, result)

	case billing.WebhookEventLSSubscriptionUpdated:
		p.parseSubscriptionUpdatedEvent(&webhookPayload, result)

	case billing.WebhookEventLSSubscriptionCancelled:
		p.parseSubscriptionCancelledEvent(&webhookPayload, result)

	case billing.WebhookEventLSSubscriptionPaused:
		p.parseSubscriptionPausedEvent(&webhookPayload, result)

	case billing.WebhookEventLSSubscriptionResumed:
		p.parseSubscriptionResumedEvent(&webhookPayload, result)

	case billing.WebhookEventLSSubscriptionExpired:
		p.parseSubscriptionExpiredEvent(&webhookPayload, result)

	case billing.WebhookEventLSPaymentSuccess:
		p.parsePaymentSuccessEvent(&webhookPayload, result)

	case billing.WebhookEventLSPaymentFailed:
		p.parsePaymentFailedEvent(&webhookPayload, result)
	}

	// Store raw payload
	result.RawPayload = make(map[string]interface{})
	_ = json.Unmarshal(payload, &result.RawPayload)

	return result, nil
}

// verifySignature verifies the webhook signature using HMAC-SHA256
// SECURITY: Webhook secret MUST be configured for signature verification
func (p *Provider) verifySignature(payload []byte, signature string) error {
	if p.webhookSecret == "" {
		// SECURITY: Never skip signature verification
		// This prevents processing webhooks without proper authentication
		return ErrWebhookSecretNotConfigured
	}

	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
		return fmt.Errorf("invalid webhook signature")
	}

	return nil
}

// parseOrderEvent parses an order_created event
func (p *Provider) parseOrderEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.ExternalOrderNo = payload.Data.ID
	result.Amount = float64(payload.Data.Attributes.Total) / 100.0 // Convert from cents to dollars
	result.Currency = payload.Data.Attributes.Currency
	result.Status = billing.OrderStatusSucceeded

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

// parseSubscriptionCreatedEvent parses a subscription_created event
func (p *Provider) parseSubscriptionCreatedEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = billing.SubscriptionStatusActive

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

// parseSubscriptionUpdatedEvent parses a subscription_updated event
func (p *Provider) parseSubscriptionUpdatedEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = payload.Data.Attributes.Status

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

// parseSubscriptionCancelledEvent parses a subscription_cancelled event
func (p *Provider) parseSubscriptionCancelledEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = billing.SubscriptionStatusCanceled

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

// parsePaymentSuccessEvent parses a subscription_payment_success event
func (p *Provider) parsePaymentSuccessEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.ExternalOrderNo = payload.Data.ID
	result.Amount = float64(payload.Data.Attributes.Total) / 100.0 // Convert from cents to dollars
	result.Currency = payload.Data.Attributes.Currency
	result.Status = billing.OrderStatusSucceeded

	if payload.Data.Attributes.SubscriptionID != 0 {
		result.SubscriptionID = strconv.Itoa(payload.Data.Attributes.SubscriptionID)
	}
	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

// parsePaymentFailedEvent parses a subscription_payment_failed event
func (p *Provider) parsePaymentFailedEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.ExternalOrderNo = payload.Data.ID
	result.Status = billing.OrderStatusFailed

	if payload.Data.Attributes.SubscriptionID != 0 {
		result.SubscriptionID = strconv.Itoa(payload.Data.Attributes.SubscriptionID)
	}
	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

// parseSubscriptionPausedEvent parses a subscription_paused event
func (p *Provider) parseSubscriptionPausedEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = billing.SubscriptionStatusPaused

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

// parseSubscriptionResumedEvent parses a subscription_resumed event
func (p *Provider) parseSubscriptionResumedEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = billing.SubscriptionStatusActive

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

// parseSubscriptionExpiredEvent parses a subscription_expired event
func (p *Provider) parseSubscriptionExpiredEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = billing.SubscriptionStatusExpired

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

package lemonsqueezy

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

var ErrWebhookSecretNotConfigured = fmt.Errorf("webhook secret is not configured - this is required for security")

func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*types.WebhookEvent, error) {
	if err := p.verifySignature(payload, signature); err != nil {
		return nil, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	var webhookPayload WebhookPayload
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// data.id alone is the resource ID (same across repeated updates) — combine with
	// event_name + updated_at/created_at to get a per-delivery idempotency key.
	eventID := webhookPayload.Data.ID
	if eventID == "" {
		eventID = "unknown"
	}
	eventID = eventID + "_" + webhookPayload.Meta.EventName
	if webhookPayload.Data.Attributes.UpdatedAt != nil {
		eventID = eventID + "_" + webhookPayload.Data.Attributes.UpdatedAt.Format("20060102150405.000000000")
	} else if webhookPayload.Data.Attributes.CreatedAt != nil {
		eventID = eventID + "_" + webhookPayload.Data.Attributes.CreatedAt.Format("20060102150405.000000000")
	}

	result := &types.WebhookEvent{
		EventID:   eventID,
		EventType: webhookPayload.Meta.EventName,
		Provider:  billing.PaymentProviderLemonSqueezy,
	}

	if webhookPayload.Meta.CustomData != nil {
		if orderNo, ok := webhookPayload.Meta.CustomData["order_no"].(string); ok {
			result.OrderNo = orderNo
		}
	}

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

	result.RawPayload = make(map[string]interface{})
	_ = json.Unmarshal(payload, &result.RawPayload)

	return result, nil
}

// verifySignature — webhook secret MUST be configured; HMAC-SHA256 over payload.
func (p *Provider) verifySignature(payload []byte, signature string) error {
	if p.webhookSecret == "" {
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

func (p *Provider) parseOrderEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.ExternalOrderNo = payload.Data.ID
	result.Amount = float64(payload.Data.Attributes.Total) / 100.0
	result.Currency = payload.Data.Attributes.Currency
	result.Status = billing.OrderStatusSucceeded

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

func (p *Provider) parseSubscriptionCreatedEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = billing.SubscriptionStatusActive

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}

	if payload.Data.Attributes.FirstSubscriptionItem != nil {
		result.Seats = payload.Data.Attributes.FirstSubscriptionItem.Quantity
	}

	if payload.Data.Attributes.VariantID != 0 {
		result.VariantID = strconv.Itoa(payload.Data.Attributes.VariantID)
	}
}

func (p *Provider) parseSubscriptionUpdatedEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = payload.Data.Attributes.Status

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}

	if payload.Data.Attributes.FirstSubscriptionItem != nil {
		result.Seats = payload.Data.Attributes.FirstSubscriptionItem.Quantity
	}

	if payload.Data.Attributes.VariantID != 0 {
		result.VariantID = strconv.Itoa(payload.Data.Attributes.VariantID)
	}
}

func (p *Provider) parseSubscriptionCancelledEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = billing.SubscriptionStatusCanceled

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

func (p *Provider) parsePaymentSuccessEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.ExternalOrderNo = payload.Data.ID
	result.Amount = float64(payload.Data.Attributes.Total) / 100.0
	result.Currency = payload.Data.Attributes.Currency
	result.Status = billing.OrderStatusSucceeded

	if payload.Data.Attributes.SubscriptionID != 0 {
		result.SubscriptionID = strconv.Itoa(payload.Data.Attributes.SubscriptionID)
	}
	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

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

func (p *Provider) parseSubscriptionPausedEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = billing.SubscriptionStatusPaused

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

func (p *Provider) parseSubscriptionResumedEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = billing.SubscriptionStatusActive

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

func (p *Provider) parseSubscriptionExpiredEvent(payload *WebhookPayload, result *types.WebhookEvent) {
	result.SubscriptionID = payload.Data.ID
	result.Status = billing.SubscriptionStatusExpired

	if payload.Data.Attributes.CustomerID != 0 {
		result.CustomerID = strconv.Itoa(payload.Data.Attributes.CustomerID)
	}
}

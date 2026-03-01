package lemonsqueezy

import (
	"encoding/json"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

func TestHandleWebhook_InvalidSignature(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{
		APIKey:        "test_api_key",
		StoreID:       "12345",
		WebhookSecret: "test_secret",
	}

	provider := NewProvider(cfg)

	payload := []byte(`{"meta":{"event_name":"order_created"}}`)
	_, err := provider.HandleWebhook(nil, payload, "invalid_signature")
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestHandleWebhook_ValidPayload(t *testing.T) {
	secret := "test_webhook_secret"
	cfg := &config.LemonSqueezyConfig{
		APIKey:        "test_api_key",
		StoreID:       "12345",
		WebhookSecret: secret,
	}

	provider := NewProvider(cfg)

	payload := WebhookPayload{
		Meta: WebhookMeta{
			EventName: billing.WebhookEventLSOrderCreated,
			CustomData: map[string]interface{}{
				"order_no": "ORD-123",
			},
		},
		Data: WebhookData{
			ID:   "order_12345",
			Type: "orders",
			Attributes: WebhookDataAttributes{
				Total:      2999,
				Currency:   "USD",
				CustomerID: 67890,
			},
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	signature := generateHMAC(string(payloadBytes), secret)

	event, err := provider.HandleWebhook(nil, payloadBytes, signature)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.EventType != billing.WebhookEventLSOrderCreated {
		t.Errorf("expected event type %s, got %s", billing.WebhookEventLSOrderCreated, event.EventType)
	}
	if event.Provider != billing.PaymentProviderLemonSqueezy {
		t.Errorf("expected provider %s, got %s", billing.PaymentProviderLemonSqueezy, event.Provider)
	}
	if event.OrderNo != "ORD-123" {
		t.Errorf("expected order no 'ORD-123', got %s", event.OrderNo)
	}
}

func TestHandleWebhook_AllEventTypes(t *testing.T) {
	secret := "test_webhook_secret"
	cfg := &config.LemonSqueezyConfig{
		APIKey:        "test_api_key",
		StoreID:       "12345",
		WebhookSecret: secret,
	}

	provider := NewProvider(cfg)

	eventTypes := []string{
		billing.WebhookEventLSOrderCreated,
		billing.WebhookEventLSSubscriptionCreated,
		billing.WebhookEventLSSubscriptionUpdated,
		billing.WebhookEventLSSubscriptionCancelled,
		billing.WebhookEventLSSubscriptionPaused,
		billing.WebhookEventLSSubscriptionResumed,
		billing.WebhookEventLSSubscriptionExpired,
		billing.WebhookEventLSPaymentSuccess,
		billing.WebhookEventLSPaymentFailed,
	}

	for _, eventType := range eventTypes {
		t.Run(eventType, func(t *testing.T) {
			payload := WebhookPayload{
				Meta: WebhookMeta{
					EventName: eventType,
				},
				Data: WebhookData{
					ID:   "test_12345",
					Type: "test",
					Attributes: WebhookDataAttributes{
						Total:          1999,
						Currency:       "USD",
						CustomerID:     67890,
						SubscriptionID: 54321,
					},
				},
			}

			payloadBytes, _ := json.Marshal(payload)
			signature := generateHMAC(string(payloadBytes), secret)

			event, err := provider.HandleWebhook(nil, payloadBytes, signature)
			if err != nil {
				t.Fatalf("unexpected error for event %s: %v", eventType, err)
			}

			if event.EventType != eventType {
				t.Errorf("expected event type %s, got %s", eventType, event.EventType)
			}
			if event.RawPayload == nil {
				t.Error("expected raw payload to be stored")
			}
		})
	}
}

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{
		APIKey:        "test_api_key",
		StoreID:       "12345",
		WebhookSecret: "test_secret",
	}

	provider := NewProvider(cfg)

	// Test with invalid JSON
	payload := []byte(`{invalid json}`)
	signature := generateHMAC(string(payload), "test_secret")

	_, err := provider.HandleWebhook(nil, payload, signature)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestHandleWebhook_EmptyDataID(t *testing.T) {
	secret := "test_webhook_secret"
	cfg := &config.LemonSqueezyConfig{
		APIKey:        "test_api_key",
		StoreID:       "12345",
		WebhookSecret: secret,
	}

	provider := NewProvider(cfg)

	// Test with empty Data.ID - should use fallback event ID
	payload := WebhookPayload{
		Meta: WebhookMeta{
			EventName: billing.WebhookEventLSOrderCreated,
		},
		Data: WebhookData{
			ID:   "", // Empty ID
			Type: "orders",
			Attributes: WebhookDataAttributes{
				Total:      1000,
				Currency:   "USD",
				CustomerID: 12345,
			},
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	signature := generateHMAC(string(payloadBytes), secret)

	event, err := provider.HandleWebhook(nil, payloadBytes, signature)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Event ID should contain event name (fallback format)
	if event.EventID == "" {
		t.Error("expected non-empty event ID")
	}
}

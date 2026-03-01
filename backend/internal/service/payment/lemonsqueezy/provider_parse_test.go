package lemonsqueezy

import (
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
)

func TestParseOrderEvent(t *testing.T) {
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

	cfg := &config.LemonSqueezyConfig{}
	provider := NewProvider(cfg)

	result := &types.WebhookEvent{}
	provider.parseOrderEvent(&payload, result)

	if result.ExternalOrderNo != "order_12345" {
		t.Errorf("expected external order no 'order_12345', got %s", result.ExternalOrderNo)
	}
	if result.Amount != 29.99 {
		t.Errorf("expected amount 29.99, got %f", result.Amount)
	}
	if result.Currency != "USD" {
		t.Errorf("expected currency USD, got %s", result.Currency)
	}
	if result.CustomerID != "67890" {
		t.Errorf("expected customer ID '67890', got %s", result.CustomerID)
	}
	if result.Status != billing.OrderStatusSucceeded {
		t.Errorf("expected status succeeded, got %s", result.Status)
	}
}

func TestParseSubscriptionEvents(t *testing.T) {
	tests := []struct {
		name          string
		eventName     string
		parseFunc     func(*Provider, *WebhookPayload, *types.WebhookEvent)
		expectedState string
	}{
		{
			name:      "subscription_created",
			eventName: billing.WebhookEventLSSubscriptionCreated,
			parseFunc: func(p *Provider, payload *WebhookPayload, result *types.WebhookEvent) {
				p.parseSubscriptionCreatedEvent(payload, result)
			},
			expectedState: billing.SubscriptionStatusActive,
		},
		{
			name:      "subscription_cancelled",
			eventName: billing.WebhookEventLSSubscriptionCancelled,
			parseFunc: func(p *Provider, payload *WebhookPayload, result *types.WebhookEvent) {
				p.parseSubscriptionCancelledEvent(payload, result)
			},
			expectedState: billing.SubscriptionStatusCanceled,
		},
		{
			name:      "subscription_paused",
			eventName: billing.WebhookEventLSSubscriptionPaused,
			parseFunc: func(p *Provider, payload *WebhookPayload, result *types.WebhookEvent) {
				p.parseSubscriptionPausedEvent(payload, result)
			},
			expectedState: billing.SubscriptionStatusPaused,
		},
		{
			name:      "subscription_resumed",
			eventName: billing.WebhookEventLSSubscriptionResumed,
			parseFunc: func(p *Provider, payload *WebhookPayload, result *types.WebhookEvent) {
				p.parseSubscriptionResumedEvent(payload, result)
			},
			expectedState: billing.SubscriptionStatusActive,
		},
		{
			name:      "subscription_expired",
			eventName: billing.WebhookEventLSSubscriptionExpired,
			parseFunc: func(p *Provider, payload *WebhookPayload, result *types.WebhookEvent) {
				p.parseSubscriptionExpiredEvent(payload, result)
			},
			expectedState: billing.SubscriptionStatusExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := WebhookPayload{
				Meta: WebhookMeta{
					EventName: tt.eventName,
				},
				Data: WebhookData{
					ID:   "sub_12345",
					Type: "subscriptions",
					Attributes: WebhookDataAttributes{
						CustomerID: 67890,
					},
				},
			}

			cfg := &config.LemonSqueezyConfig{}
			provider := NewProvider(cfg)

			result := &types.WebhookEvent{}
			tt.parseFunc(provider, &payload, result)

			if result.SubscriptionID != "sub_12345" {
				t.Errorf("expected subscription ID 'sub_12345', got %s", result.SubscriptionID)
			}
			if result.CustomerID != "67890" {
				t.Errorf("expected customer ID '67890', got %s", result.CustomerID)
			}
			if result.Status != tt.expectedState {
				t.Errorf("expected status %s, got %s", tt.expectedState, result.Status)
			}
		})
	}
}

func TestParsePaymentEvents(t *testing.T) {
	tests := []struct {
		name           string
		eventName      string
		parseFunc      func(*Provider, *WebhookPayload, *types.WebhookEvent)
		expectedStatus string
	}{
		{
			name:      "payment_success",
			eventName: billing.WebhookEventLSPaymentSuccess,
			parseFunc: func(p *Provider, payload *WebhookPayload, result *types.WebhookEvent) {
				p.parsePaymentSuccessEvent(payload, result)
			},
			expectedStatus: billing.OrderStatusSucceeded,
		},
		{
			name:      "payment_failed",
			eventName: billing.WebhookEventLSPaymentFailed,
			parseFunc: func(p *Provider, payload *WebhookPayload, result *types.WebhookEvent) {
				p.parsePaymentFailedEvent(payload, result)
			},
			expectedStatus: billing.OrderStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := WebhookPayload{
				Meta: WebhookMeta{
					EventName: tt.eventName,
				},
				Data: WebhookData{
					ID:   "inv_12345",
					Type: "subscription-invoices",
					Attributes: WebhookDataAttributes{
						Total:          1999,
						Currency:       "USD",
						SubscriptionID: 54321,
						CustomerID:     67890,
					},
				},
			}

			cfg := &config.LemonSqueezyConfig{}
			provider := NewProvider(cfg)

			result := &types.WebhookEvent{}
			tt.parseFunc(provider, &payload, result)

			if result.SubscriptionID != "54321" {
				t.Errorf("expected subscription ID '54321', got %s", result.SubscriptionID)
			}
			if result.CustomerID != "67890" {
				t.Errorf("expected customer ID '67890', got %s", result.CustomerID)
			}
			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}
		})
	}
}

func TestParseSubscriptionUpdatedEvent(t *testing.T) {
	payload := WebhookPayload{
		Meta: WebhookMeta{
			EventName: billing.WebhookEventLSSubscriptionUpdated,
		},
		Data: WebhookData{
			ID:   "sub_12345",
			Type: "subscriptions",
			Attributes: WebhookDataAttributes{
				CustomerID: 67890,
				Status:     "active",
			},
		},
	}

	cfg := &config.LemonSqueezyConfig{}
	provider := NewProvider(cfg)

	result := &types.WebhookEvent{}
	provider.parseSubscriptionUpdatedEvent(&payload, result)

	if result.SubscriptionID != "sub_12345" {
		t.Errorf("expected subscription ID 'sub_12345', got %s", result.SubscriptionID)
	}
	if result.CustomerID != "67890" {
		t.Errorf("expected customer ID '67890', got %s", result.CustomerID)
	}
	if result.Status != "active" {
		t.Errorf("expected status 'active', got %s", result.Status)
	}
}

func TestParseOrderEvent_ZeroCustomerID(t *testing.T) {
	payload := WebhookPayload{
		Meta: WebhookMeta{
			EventName: billing.WebhookEventLSOrderCreated,
		},
		Data: WebhookData{
			ID:   "order_12345",
			Type: "orders",
			Attributes: WebhookDataAttributes{
				Total:      2999,
				Currency:   "USD",
				CustomerID: 0, // Zero customer ID
			},
		},
	}

	cfg := &config.LemonSqueezyConfig{}
	provider := NewProvider(cfg)

	result := &types.WebhookEvent{}
	provider.parseOrderEvent(&payload, result)

	// CustomerID should remain empty when source is 0
	if result.CustomerID != "" {
		t.Errorf("expected empty customer ID for zero value, got %s", result.CustomerID)
	}
}

func TestParsePaymentEvents_ZeroIDs(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{}
	provider := NewProvider(cfg)

	// Test payment success with zero subscription ID
	payload := WebhookPayload{
		Data: WebhookData{
			ID: "inv_12345",
			Attributes: WebhookDataAttributes{
				Total:          1000,
				Currency:       "USD",
				SubscriptionID: 0, // Zero
				CustomerID:     0, // Zero
			},
		},
	}

	result := &types.WebhookEvent{}
	provider.parsePaymentSuccessEvent(&payload, result)

	if result.SubscriptionID != "" {
		t.Errorf("expected empty subscription ID for zero value, got %s", result.SubscriptionID)
	}
	if result.CustomerID != "" {
		t.Errorf("expected empty customer ID for zero value, got %s", result.CustomerID)
	}

	// Test payment failed
	result2 := &types.WebhookEvent{}
	provider.parsePaymentFailedEvent(&payload, result2)

	if result2.SubscriptionID != "" {
		t.Errorf("expected empty subscription ID for zero value, got %s", result2.SubscriptionID)
	}
}

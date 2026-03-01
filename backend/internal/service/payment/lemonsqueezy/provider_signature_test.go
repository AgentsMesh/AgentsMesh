package lemonsqueezy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

func TestNewProvider(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{
		APIKey:        "test_api_key",
		StoreID:       "12345",
		WebhookSecret: "test_secret",
	}

	provider := NewProvider(cfg)
	if provider.GetProviderName() != billing.PaymentProviderLemonSqueezy {
		t.Errorf("expected provider name %s, got %s", billing.PaymentProviderLemonSqueezy, provider.GetProviderName())
	}
}

func TestVerifySignature(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{
		APIKey:        "test_api_key",
		StoreID:       "12345",
		WebhookSecret: "test_webhook_secret",
	}

	provider := NewProvider(cfg)

	tests := []struct {
		name      string
		payload   string
		signature string
		wantErr   bool
	}{
		{
			name:      "valid signature",
			payload:   `{"meta":{"event_name":"order_created"}}`,
			signature: generateHMAC(`{"meta":{"event_name":"order_created"}}`, "test_webhook_secret"),
			wantErr:   false,
		},
		{
			name:      "invalid signature",
			payload:   `{"meta":{"event_name":"order_created"}}`,
			signature: "invalid_signature",
			wantErr:   true,
		},
		{
			name:      "empty signature",
			payload:   `{"meta":{"event_name":"order_created"}}`,
			signature: "",
			wantErr:   true,
		},
		{
			name:      "tampered payload",
			payload:   `{"meta":{"event_name":"order_created","tampered":true}}`,
			signature: generateHMAC(`{"meta":{"event_name":"order_created"}}`, "test_webhook_secret"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.verifySignature([]byte(tt.payload), tt.signature)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifySignature() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifySignature_NoSecret(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{
		APIKey:        "test_api_key",
		StoreID:       "12345",
		WebhookSecret: "", // No secret configured
	}

	provider := NewProvider(cfg)

	// Should return error when no secret is configured (security requirement)
	err := provider.verifySignature([]byte(`{"test":"data"}`), "any_signature")
	if err == nil {
		t.Error("verifySignature() should return error when no secret is configured")
	}
	if err != ErrWebhookSecretNotConfigured {
		t.Errorf("verifySignature() should return ErrWebhookSecretNotConfigured, got %v", err)
	}
}

// Helper function to generate HMAC signature
func generateHMAC(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

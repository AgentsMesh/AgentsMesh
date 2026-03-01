package lemonsqueezy

import (
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
)

func TestStringToInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"123", 123},
		{"0", 0},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := stringToInt(tt.input)
		if got != tt.expected {
			t.Errorf("stringToInt(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestGetCheckoutStatus(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{
		APIKey:  "test_api_key",
		StoreID: "12345",
	}

	provider := NewProvider(cfg)

	// GetCheckoutStatus always returns pending for LemonSqueezy
	status, err := provider.GetCheckoutStatus(nil, "any_session_id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != billing.OrderStatusPending {
		t.Errorf("expected status %s, got %s", billing.OrderStatusPending, status)
	}
}

func TestRefundPayment(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{
		APIKey:  "test_api_key",
		StoreID: "12345",
	}

	provider := NewProvider(cfg)

	// RefundPayment always returns error for LemonSqueezy (must be done via dashboard)
	_, err := provider.RefundPayment(nil, &types.RefundRequest{})
	if err == nil {
		t.Error("expected error for RefundPayment")
	}
	if err.Error() != "refunds must be processed through the LemonSqueezy dashboard" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCreateCustomer(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{
		APIKey:  "test_api_key",
		StoreID: "12345",
	}

	provider := NewProvider(cfg)

	// CreateCustomer returns empty string for LemonSqueezy (customers are created during checkout)
	customerID, err := provider.CreateCustomer(nil, "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if customerID != "" {
		t.Errorf("expected empty customer ID, got %s", customerID)
	}
}

func TestCreateCheckoutSession_MissingVariantID(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{
		APIKey:  "test_api_key",
		StoreID: "12345",
	}

	provider := NewProvider(cfg)

	// CreateCheckoutSession should fail when variant_id is missing
	req := &types.CheckoutRequest{
		OrganizationID: 1,
		UserID:         1,
		OrderType:      "subscription",
		Metadata:       nil, // No metadata
	}

	_, err := provider.CreateCheckoutSession(nil, req)
	if err == nil {
		t.Error("expected error for missing variant_id")
	}
	if err.Error() != "variant_id is required in metadata for LemonSqueezy checkout" {
		t.Errorf("unexpected error message: %v", err)
	}

	// Also test with empty metadata
	req.Metadata = map[string]string{}
	_, err = provider.CreateCheckoutSession(nil, req)
	if err == nil {
		t.Error("expected error for empty variant_id")
	}
}

func TestGetCustomerPortalURL_MissingSubscriptionID(t *testing.T) {
	cfg := &config.LemonSqueezyConfig{
		APIKey:  "test_api_key",
		StoreID: "12345",
	}

	provider := NewProvider(cfg)

	// GetCustomerPortalURL should fail when subscription_id is missing
	req := &types.CustomerPortalRequest{
		CustomerID:     "",
		SubscriptionID: "",
	}

	_, err := provider.GetCustomerPortalURL(nil, req)
	if err == nil {
		t.Error("expected error for missing subscription_id")
	}
	if err.Error() != "subscription_id is required for LemonSqueezy customer portal" {
		t.Errorf("unexpected error message: %v", err)
	}
}

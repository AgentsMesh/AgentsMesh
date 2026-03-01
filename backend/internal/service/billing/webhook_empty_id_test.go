package billing

import (
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Webhook Empty ID Edge Case Tests
// ===========================================

// TestHandleSubscriptionCanceled_EmptySubID tests with empty subscription ID
func TestHandleSubscriptionCanceled_EmptySubID(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_empty_sub",
		EventType:      billing.WebhookEventLSSubscriptionCancelled,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "", // Empty
	}

	// Should return nil (graceful handling)
	err := svc.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Errorf("expected nil for empty subscription ID, got %v", err)
	}
}

// TestHandleSubscriptionPaused_EmptySubID tests pause with empty subscription ID
func TestHandleSubscriptionPaused_EmptySubID(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_pause_empty",
		EventType:      billing.WebhookEventLSSubscriptionPaused,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "",
	}

	err := svc.HandleSubscriptionPaused(c, event)
	if err != nil {
		t.Errorf("expected nil for empty subscription ID, got %v", err)
	}
}

// TestHandleSubscriptionResumed_EmptySubID tests resume with empty subscription ID
func TestHandleSubscriptionResumed_EmptySubID(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_resume_empty",
		EventType:      billing.WebhookEventLSSubscriptionResumed,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "",
	}

	err := svc.HandleSubscriptionResumed(c, event)
	if err != nil {
		t.Errorf("expected nil for empty subscription ID, got %v", err)
	}
}

// TestHandleSubscriptionExpired_EmptySubID tests expire with empty subscription ID
func TestHandleSubscriptionExpired_EmptySubID(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_expire_empty",
		EventType:      billing.WebhookEventLSSubscriptionExpired,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "",
	}

	err := svc.HandleSubscriptionExpired(c, event)
	if err != nil {
		t.Errorf("expected nil for empty subscription ID, got %v", err)
	}
}

// TestHandleSubscriptionCreated_EmptySubID tests create with empty subscription ID
func TestHandleSubscriptionCreated_EmptySubID(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_create_empty",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "",
	}

	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Errorf("expected nil for empty subscription ID, got %v", err)
	}
}

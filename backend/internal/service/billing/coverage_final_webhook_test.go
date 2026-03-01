package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Final Coverage Tests - Webhook Idempotency
// ===========================================

// TestHandlePaymentSucceeded_IdempotencyDuplicate tests idempotency check
func TestHandlePaymentSucceeded_IdempotencyDuplicate(t *testing.T) {
	svc, db := setupTestService(t)

	// Pre-create the webhook event record
	db.Create(&billing.WebhookEvent{
		EventID:     "evt_duplicate",
		Provider:    billing.PaymentProviderLemonSqueezy,
		EventType:   billing.WebhookEventLSPaymentSuccess,
		ProcessedAt: time.Now(),
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_duplicate",
		EventType: billing.WebhookEventLSPaymentSuccess,
		Provider:  billing.PaymentProviderLemonSqueezy,
	}

	// Should return nil (duplicate is silently ignored)
	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Errorf("expected nil for duplicate event, got %v", err)
	}
}

// TestHandlePaymentFailed_IdempotencyDuplicate tests idempotency check for failed payments
func TestHandlePaymentFailed_IdempotencyDuplicate(t *testing.T) {
	svc, db := setupTestService(t)

	// Pre-create the webhook event record
	db.Create(&billing.WebhookEvent{
		EventID:     "evt_fail_duplicate",
		Provider:    billing.PaymentProviderLemonSqueezy,
		EventType:   billing.WebhookEventLSPaymentFailed,
		ProcessedAt: time.Now(),
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:      "evt_fail_duplicate",
		EventType:    billing.WebhookEventLSPaymentFailed,
		Provider:     billing.PaymentProviderLemonSqueezy,
		FailedReason: "Card declined",
	}

	// Should return nil (duplicate is silently ignored)
	err := svc.HandlePaymentFailed(c, event)
	if err != nil {
		t.Errorf("expected nil for duplicate event, got %v", err)
	}
}

// TestHandleSubscriptionCanceled_IdempotencyDuplicate tests idempotency check for cancellation
func TestHandleSubscriptionCanceled_IdempotencyDuplicate(t *testing.T) {
	svc, db := setupTestService(t)

	// Pre-create the webhook event record
	db.Create(&billing.WebhookEvent{
		EventID:     "evt_cancel_duplicate",
		Provider:    billing.PaymentProviderStripe,
		EventType:   "customer.subscription.deleted",
		ProcessedAt: time.Now(),
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_cancel_duplicate",
		EventType:      "customer.subscription.deleted",
		Provider:       billing.PaymentProviderStripe,
		SubscriptionID: "sub_any",
	}

	// Should return nil (duplicate is silently ignored)
	err := svc.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Errorf("expected nil for duplicate event, got %v", err)
	}
}

// TestHandleSubscriptionUpdated_IdempotencyDuplicate tests idempotency for subscription updated
func TestHandleSubscriptionUpdated_IdempotencyDuplicate(t *testing.T) {
	svc, db := setupTestService(t)

	// Pre-create the webhook event record
	db.Create(&billing.WebhookEvent{
		EventID:     "evt_updated_duplicate",
		Provider:    billing.PaymentProviderStripe,
		EventType:   "customer.subscription.updated",
		ProcessedAt: time.Now(),
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_updated_duplicate",
		EventType:      "customer.subscription.updated",
		Provider:       billing.PaymentProviderStripe,
		SubscriptionID: "sub_any",
		Status:         "active",
	}

	err := svc.HandleSubscriptionUpdated(c, event)
	if err != nil {
		t.Errorf("expected nil for duplicate event, got %v", err)
	}
}

// TestHandleSubscriptionCreated_IdempotencyDuplicate tests idempotency for subscription created
func TestHandleSubscriptionCreated_IdempotencyDuplicate(t *testing.T) {
	svc, db := setupTestService(t)

	// Pre-create the webhook event record
	db.Create(&billing.WebhookEvent{
		EventID:     "evt_created_duplicate",
		Provider:    billing.PaymentProviderLemonSqueezy,
		EventType:   billing.WebhookEventLSSubscriptionCreated,
		ProcessedAt: time.Now(),
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_created_duplicate",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_any",
	}

	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Errorf("expected nil for duplicate event, got %v", err)
	}
}

// TestHandleSubscriptionPaused_IdempotencyDuplicate tests idempotency for subscription paused
func TestHandleSubscriptionPaused_IdempotencyDuplicate(t *testing.T) {
	svc, db := setupTestService(t)

	// Pre-create the webhook event record
	db.Create(&billing.WebhookEvent{
		EventID:     "evt_paused_duplicate",
		Provider:    billing.PaymentProviderLemonSqueezy,
		EventType:   billing.WebhookEventLSSubscriptionPaused,
		ProcessedAt: time.Now(),
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_paused_duplicate",
		EventType:      billing.WebhookEventLSSubscriptionPaused,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_any",
	}

	err := svc.HandleSubscriptionPaused(c, event)
	if err != nil {
		t.Errorf("expected nil for duplicate event, got %v", err)
	}
}

// TestHandleSubscriptionResumed_IdempotencyDuplicate tests idempotency for subscription resumed
func TestHandleSubscriptionResumed_IdempotencyDuplicate(t *testing.T) {
	svc, db := setupTestService(t)

	// Pre-create the webhook event record
	db.Create(&billing.WebhookEvent{
		EventID:     "evt_resumed_duplicate",
		Provider:    billing.PaymentProviderLemonSqueezy,
		EventType:   billing.WebhookEventLSSubscriptionResumed,
		ProcessedAt: time.Now(),
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_resumed_duplicate",
		EventType:      billing.WebhookEventLSSubscriptionResumed,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_any",
	}

	err := svc.HandleSubscriptionResumed(c, event)
	if err != nil {
		t.Errorf("expected nil for duplicate event, got %v", err)
	}
}

// TestHandleSubscriptionExpired_IdempotencyDuplicate tests idempotency for subscription expired
func TestHandleSubscriptionExpired_IdempotencyDuplicate(t *testing.T) {
	svc, db := setupTestService(t)

	// Pre-create the webhook event record
	db.Create(&billing.WebhookEvent{
		EventID:     "evt_expired_duplicate",
		Provider:    billing.PaymentProviderLemonSqueezy,
		EventType:   billing.WebhookEventLSSubscriptionExpired,
		ProcessedAt: time.Now(),
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_expired_duplicate",
		EventType:      billing.WebhookEventLSSubscriptionExpired,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_any",
	}

	err := svc.HandleSubscriptionExpired(c, event)
	if err != nil {
		t.Errorf("expected nil for duplicate event, got %v", err)
	}
}

// ===========================================
// Final Coverage Tests - Trial & Overview
// ===========================================

// TestCreateTrialSubscription_DefaultTrialDays tests trial with default days
func TestCreateTrialSubscription_DefaultTrialDays(t *testing.T) {
	svc, _ := setupTestService(t)

	sub, err := svc.CreateTrialSubscription(context.Background(), 1, "pro", 0) // 0 triggers default
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sub.Status != billing.SubscriptionStatusTrialing {
		t.Errorf("expected trialing status, got %s", sub.Status)
	}
}

// TestCreateTrialSubscription_NegativeTrialDays tests trial with negative days
func TestCreateTrialSubscription_NegativeTrialDays(t *testing.T) {
	svc, _ := setupTestService(t)

	sub, err := svc.CreateTrialSubscription(context.Background(), 2, "pro", -5) // Negative triggers default
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sub.Status != billing.SubscriptionStatusTrialing {
		t.Errorf("expected trialing status, got %s", sub.Status)
	}
}

// TestGetBillingOverview_WithNilPlan tests overview when plan is nil
func TestGetBillingOverview_WithNilPlan(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	// Create subscription without preloading plan
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          3,
	}
	db.Create(sub)

	overview, err := svc.GetBillingOverview(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if overview.Plan == nil {
		t.Error("expected plan to be loaded")
	}
}

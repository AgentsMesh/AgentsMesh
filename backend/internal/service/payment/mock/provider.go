package mock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
)

// Provider implements payment.SubscriptionProvider for testing/mocking
// It simulates payment flows without calling real payment providers
type Provider struct {
	mu       sync.RWMutex
	sessions map[string]*mockSession // sessionID -> session data
	baseURL  string                  // Base URL for mock checkout page
}

// mockSession stores mock checkout session data
type mockSession struct {
	ID             string
	Status         string
	Request        *types.CheckoutRequest
	CreatedAt      time.Time
	ExpiresAt      time.Time
	CompletedAt    *time.Time
	CustomerID     string
	SubscriptionID string
}

// NewProvider creates a new mock payment provider
func NewProvider(baseURL string) *Provider {
	return &Provider{
		sessions: make(map[string]*mockSession),
		baseURL:  baseURL,
	}
}

// GetProviderName returns the provider name
func (p *Provider) GetProviderName() string {
	return "mock"
}

// generateID generates a random ID with a prefix
func generateID(prefix string) string {
	bytes := make([]byte, 12)
	_, _ = rand.Read(bytes)
	return prefix + "_" + hex.EncodeToString(bytes)
}

// CreateCheckoutSession creates a mock checkout session
func (p *Provider) CreateCheckoutSession(ctx context.Context, req *types.CheckoutRequest) (*types.CheckoutResponse, error) {
	sessionID := generateID("mock_cs")
	customerID := generateID("mock_cus")
	subscriptionID := generateID("mock_sub")

	session := &mockSession{
		ID:             sessionID,
		Status:         billing.OrderStatusPending,
		Request:        req,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(30 * time.Minute),
		CustomerID:     customerID,
		SubscriptionID: subscriptionID,
	}

	p.mu.Lock()
	p.sessions[sessionID] = session
	p.mu.Unlock()

	// Build mock checkout URL
	// The URL points to a mock checkout page that auto-completes
	checkoutURL := fmt.Sprintf("%s/mock-checkout?session_id=%s&order_no=%s", p.baseURL, sessionID, req.IdempotencyKey)

	return &types.CheckoutResponse{
		SessionID:       sessionID,
		SessionURL:      checkoutURL,
		OrderNo:         req.IdempotencyKey,
		ExternalOrderNo: sessionID,
		ExpiresAt:       session.ExpiresAt,
	}, nil
}

// GetCheckoutStatus checks the status of a checkout session
func (p *Provider) GetCheckoutStatus(ctx context.Context, sessionID string) (string, error) {
	p.mu.RLock()
	session, ok := p.sessions[sessionID]
	p.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) && session.Status == billing.OrderStatusPending {
		return billing.OrderStatusCanceled, nil
	}

	return session.Status, nil
}

// CompleteSession marks a session as completed (called by mock checkout handler)
func (p *Provider) CompleteSession(sessionID string) (*mockSession, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	session, ok := p.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Status != billing.OrderStatusPending {
		return nil, fmt.Errorf("session already processed: %s", session.Status)
	}

	now := time.Now()
	session.Status = billing.OrderStatusSucceeded
	session.CompletedAt = &now

	return session, nil
}

// GetSession retrieves a session by ID
func (p *Provider) GetSession(sessionID string) (*mockSession, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	session, ok := p.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// HandleWebhook handles mock webhook events
// In mock mode, webhooks are triggered directly via API
func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*types.WebhookEvent, error) {
	// Parse the mock webhook payload
	var mockEvent struct {
		EventType string `json:"event_type"`
		SessionID string `json:"session_id"`
		OrderNo   string `json:"order_no"`
	}

	if err := json.Unmarshal(payload, &mockEvent); err != nil {
		return nil, fmt.Errorf("failed to parse mock webhook: %w", err)
	}

	// Get session data
	session, err := p.GetSession(mockEvent.SessionID)
	if err != nil {
		return nil, err
	}

	event := &types.WebhookEvent{
		EventID:         generateID("mock_evt"),
		EventType:       mockEvent.EventType,
		Provider:        "mock",
		OrderNo:         mockEvent.OrderNo,
		ExternalOrderNo: mockEvent.SessionID,
		CustomerID:      session.CustomerID,
		SubscriptionID:  session.SubscriptionID,
		Currency:        session.Request.Currency,
		Amount:          session.Request.ActualAmount,
		RawPayload:      make(map[string]interface{}),
	}

	switch mockEvent.EventType {
	case billing.WebhookEventCheckoutCompleted:
		event.Status = billing.OrderStatusSucceeded
	case billing.WebhookEventInvoicePaid:
		event.Status = billing.OrderStatusSucceeded
	case billing.WebhookEventInvoiceFailed:
		event.Status = billing.OrderStatusFailed
		event.FailedReason = "mock payment failed"
	case billing.WebhookEventSubscriptionDeleted:
		event.Status = billing.SubscriptionStatusCanceled
	default:
		event.Status = billing.OrderStatusPending
	}

	return event, nil
}

// RefundPayment simulates a refund
func (p *Provider) RefundPayment(ctx context.Context, req *types.RefundRequest) (*types.RefundResponse, error) {
	return &types.RefundResponse{
		RefundID: generateID("mock_re"),
		Status:   "succeeded",
		Amount:   req.Amount,
		Currency: "usd",
	}, nil
}

// CancelSubscription simulates subscription cancellation
func (p *Provider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	// In mock mode, just return success
	return nil
}

// CreateCustomer creates a mock customer
func (p *Provider) CreateCustomer(ctx context.Context, email string, name string, metadata map[string]string) (string, error) {
	return generateID("mock_cus"), nil
}

// GetCustomerPortalURL returns a mock customer portal URL
func (p *Provider) GetCustomerPortalURL(ctx context.Context, req *types.CustomerPortalRequest) (*types.CustomerPortalResponse, error) {
	return &types.CustomerPortalResponse{
		URL: fmt.Sprintf("%s/mock-portal?customer_id=%s", p.baseURL, req.CustomerID),
	}, nil
}

// UpdateSubscriptionSeats simulates updating subscription seats
func (p *Provider) UpdateSubscriptionSeats(ctx context.Context, subscriptionID string, seats int) error {
	return nil
}

// GetSubscription returns mock subscription details
func (p *Provider) GetSubscription(ctx context.Context, subscriptionID string) (*types.SubscriptionDetails, error) {
	return &types.SubscriptionDetails{
		ID:                 subscriptionID,
		CustomerID:         generateID("mock_cus"),
		Status:             "active",
		CurrentPeriodStart: time.Now().AddDate(0, -1, 0),
		CurrentPeriodEnd:   time.Now().AddDate(0, 0, 30),
		CancelAtPeriodEnd:  false,
		Seats:              1,
	}, nil
}

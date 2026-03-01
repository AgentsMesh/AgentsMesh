package webhooks

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockProvider implements payment.Provider for testing
type mockProvider struct {
	handleWebhookFunc func(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error)
}

func (m *mockProvider) GetProviderName() string {
	return "mock"
}

func (m *mockProvider) CreateCheckoutSession(ctx context.Context, req *payment.CheckoutRequest) (*payment.CheckoutResponse, error) {
	return nil, nil
}

func (m *mockProvider) GetCheckoutStatus(ctx context.Context, sessionID string) (string, error) {
	return "complete", nil
}

func (m *mockProvider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error) {
	if m.handleWebhookFunc != nil {
		return m.handleWebhookFunc(ctx, payload, signature)
	}
	return &payment.WebhookEvent{
		EventID:   "evt_test",
		EventType: "checkout.session.completed",
		Provider:  "mock",
		Status:    billing.OrderStatusSucceeded,
	}, nil
}

func (m *mockProvider) RefundPayment(ctx context.Context, req *payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, nil
}

func (m *mockProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	return nil
}

func (m *mockProvider) GetCustomerPortalURL(ctx context.Context, customerID, returnURL string) (string, error) {
	return "", nil
}

// mockBillingService implements BillingServiceInterface for testing
type mockBillingService struct {
	paymentSucceededErr   error
	paymentFailedErr      error
	subscriptionCanceled  error
	subscriptionUpdatedErr error
}

func (m *mockBillingService) HandlePaymentSucceeded(ctx *gin.Context, event *payment.WebhookEvent) error {
	return m.paymentSucceededErr
}

func (m *mockBillingService) HandlePaymentFailed(ctx *gin.Context, event *payment.WebhookEvent) error {
	return m.paymentFailedErr
}

func (m *mockBillingService) HandleSubscriptionCanceled(ctx *gin.Context, event *payment.WebhookEvent) error {
	return m.subscriptionCanceled
}

func (m *mockBillingService) HandleSubscriptionUpdated(ctx *gin.Context, event *payment.WebhookEvent) error {
	return m.subscriptionUpdatedErr
}

// ===========================================
// StripeWebhookHandler Tests
// ===========================================

func TestNewStripeWebhookHandler(t *testing.T) {
	provider := &mockProvider{}
	billingSvc := &mockBillingService{}

	handler := NewStripeWebhookHandler(provider, billingSvc)
	if handler == nil {
		t.Error("expected non-nil handler")
	}
	if handler.provider != provider {
		t.Error("expected provider to be set")
	}
	if handler.billingService != billingSvc {
		t.Error("expected billing service to be set")
	}
}

func createTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestStripeWebhookHandlerHandle_MissingSignature(t *testing.T) {
	provider := &mockProvider{}
	billingSvc := &mockBillingService{}
	handler := NewStripeWebhookHandler(provider, billingSvc)

	c, w := createTestContext()
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	// No Stripe-Signature header

	handler.Handle(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestStripeWebhookHandlerHandle_InvalidSignature(t *testing.T) {
	provider := &mockProvider{
		handleWebhookFunc: func(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error) {
			return nil, context.Canceled
		},
	}
	billingSvc := &mockBillingService{}
	handler := NewStripeWebhookHandler(provider, billingSvc)

	c, w := createTestContext()
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Stripe-Signature", "invalid")

	handler.Handle(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestStripeWebhookHandlerHandle_CheckoutCompleted(t *testing.T) {
	provider := &mockProvider{
		handleWebhookFunc: func(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:   "evt_123",
				EventType: "checkout.session.completed",
				Provider:  "stripe",
			}, nil
		},
	}
	billingSvc := &mockBillingService{}
	handler := NewStripeWebhookHandler(provider, billingSvc)

	c, w := createTestContext()
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Stripe-Signature", "valid_signature")

	handler.Handle(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestStripeWebhookHandlerHandle_InvoicePaid(t *testing.T) {
	provider := &mockProvider{
		handleWebhookFunc: func(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:   "evt_123",
				EventType: "invoice.paid",
				Provider:  "stripe",
			}, nil
		},
	}
	billingSvc := &mockBillingService{}
	handler := NewStripeWebhookHandler(provider, billingSvc)

	c, w := createTestContext()
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Stripe-Signature", "valid_signature")

	handler.Handle(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestStripeWebhookHandlerHandle_InvoicePaymentFailed(t *testing.T) {
	provider := &mockProvider{
		handleWebhookFunc: func(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:   "evt_123",
				EventType: "invoice.payment_failed",
				Provider:  "stripe",
			}, nil
		},
	}
	billingSvc := &mockBillingService{}
	handler := NewStripeWebhookHandler(provider, billingSvc)

	c, w := createTestContext()
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Stripe-Signature", "valid_signature")

	handler.Handle(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestStripeWebhookHandlerHandle_SubscriptionDeleted(t *testing.T) {
	provider := &mockProvider{
		handleWebhookFunc: func(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:   "evt_123",
				EventType: "customer.subscription.deleted",
				Provider:  "stripe",
			}, nil
		},
	}
	billingSvc := &mockBillingService{}
	handler := NewStripeWebhookHandler(provider, billingSvc)

	c, w := createTestContext()
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Stripe-Signature", "valid_signature")

	handler.Handle(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestStripeWebhookHandlerHandle_SubscriptionUpdated(t *testing.T) {
	provider := &mockProvider{
		handleWebhookFunc: func(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:   "evt_123",
				EventType: "customer.subscription.updated",
				Provider:  "stripe",
			}, nil
		},
	}
	billingSvc := &mockBillingService{}
	handler := NewStripeWebhookHandler(provider, billingSvc)

	c, w := createTestContext()
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Stripe-Signature", "valid_signature")

	handler.Handle(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestStripeWebhookHandlerHandle_UnhandledEvent(t *testing.T) {
	provider := &mockProvider{
		handleWebhookFunc: func(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:   "evt_123",
				EventType: "some.unknown.event",
				Provider:  "stripe",
			}, nil
		},
	}
	billingSvc := &mockBillingService{}
	handler := NewStripeWebhookHandler(provider, billingSvc)

	c, w := createTestContext()
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Stripe-Signature", "valid_signature")

	handler.Handle(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestStripeWebhookHandlerHandle_ReadBodyError(t *testing.T) {
	provider := &mockProvider{}
	billingSvc := &mockBillingService{}
	handler := NewStripeWebhookHandler(provider, billingSvc)

	c, w := createTestContext()
	// Create a request with a body that returns an error on read
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", &errorReader{})
	c.Request.Header.Set("Stripe-Signature", "valid_signature")

	handler.Handle(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// errorReader is an io.Reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestStripeWebhookHandlerHandle_BillingServiceError(t *testing.T) {
	provider := &mockProvider{
		handleWebhookFunc: func(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:   "evt_123",
				EventType: "checkout.session.completed",
				Provider:  "stripe",
			}, nil
		},
	}
	billingSvc := &mockBillingService{
		paymentSucceededErr: context.Canceled,
	}
	handler := NewStripeWebhookHandler(provider, billingSvc)

	c, w := createTestContext()
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Stripe-Signature", "valid_signature")

	handler.Handle(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// ===========================================
// Helper function tests
// ===========================================

func testLoggerForStripe() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

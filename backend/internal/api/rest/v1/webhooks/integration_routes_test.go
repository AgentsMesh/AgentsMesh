package webhooks

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	billingService "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Handler Registry Tests
// ===========================================

func TestHandlerRegistryWithMockRouter(t *testing.T) {
	router, _, _ := createMockRouter(t)

	// Verify registry is set up
	if router.registry == nil {
		t.Error("expected registry to be set")
	}

	// Verify payment factory is set
	if router.paymentFactory == nil {
		t.Error("expected payment factory to be set")
	}

	// Verify mock is enabled
	if !router.paymentFactory.IsMockEnabled() {
		t.Error("expected mock to be enabled")
	}
}

// ===========================================
// Routes Registration Tests
// ===========================================

func TestRegisterRoutesWithMock(t *testing.T) {
	router, _, _ := createMockRouter(t)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	rg := engine.Group("/webhooks")

	// Should not panic when registering routes
	router.RegisterRoutes(rg)

	// Verify routes are registered by testing GitLab webhook (new format with org_slug/repo_id)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/webhooks/test-org/gitlab/123", bytes.NewReader([]byte(`{"object_kind": "push"}`)))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)

	// Should not be 404 (route exists)
	if w.Code == http.StatusNotFound {
		t.Error("expected route to be registered")
	}
}

// ===========================================
// Webhook Processing Tests
// ===========================================

func TestProcessWebhookWithMockProvider(t *testing.T) {
	router, _, factory := createMockRouter(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/", nil)

	// Create checkout session
	provider, _ := factory.GetDefaultProvider()
	checkoutReq := &payment.CheckoutRequest{
		OrganizationID: 1,
		OrderType:      billing.OrderTypeSubscription,
		PlanID:         1,
		BillingCycle:   billing.BillingCycleMonthly,
		Seats:          1,
		Currency:       "USD",
		Amount:         19.99,
		ActualAmount:   19.99,
		SuccessURL:     "http://localhost:3000/success",
		CancelURL:      "http://localhost:3000/cancel",
		IdempotencyKey: "ORD-PROCESS-001",
	}

	resp, err := provider.CreateCheckoutSession(ctx.Request.Context(), checkoutReq)
	if err != nil {
		t.Fatalf("failed to create checkout session: %v", err)
	}

	// Complete the session
	mockProvider := factory.GetMockProvider()
	_, err = mockProvider.CompleteSession(resp.SessionID)
	if err != nil {
		t.Fatalf("failed to complete session: %v", err)
	}

	// Handle webhook
	webhookPayload := []byte(`{"event_type": "checkout.session.completed", "session_id": "` + resp.SessionID + `", "order_no": "ORD-PROCESS-001"}`)
	event, err := provider.HandleWebhook(ctx.Request.Context(), webhookPayload, "")
	if err != nil {
		t.Fatalf("failed to handle webhook: %v", err)
	}

	// Process with billing service - will fail due to no order, but tests the flow
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	_ = router.billingSvc.HandlePaymentSucceeded(c, event) // Error expected

	// Verify event was parsed correctly
	if event.EventType != "checkout.session.completed" {
		t.Errorf("expected event type checkout.session.completed, got %s", event.EventType)
	}
}

// ===========================================
// Edge Cases
// ===========================================

func TestMockRouterWithNilFactory(t *testing.T) {
	db := setupIntegrationDB(t)
	logger := testLogger()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	// Create billing service without mock
	billingSvc := billingService.NewService(infra.NewBillingRepository(db), "")

	cfg := &config.Config{}

	router := &WebhookRouter{
		db:             db,
		cfg:            cfg,
		logger:         logger,
		registry:       registry,
		billingSvc:     billingSvc,
		paymentFactory: nil, // No factory
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"session_id": "test123"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/mock/complete", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleMockCheckoutComplete(c)

	// Should return 403 when factory is nil
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

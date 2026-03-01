package webhooks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Mock Payment Handler Integration Tests
// ===========================================

func TestMockCheckoutComplete_Enabled(t *testing.T) {
	router, _, factory := createMockRouter(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/", nil)

	// Create a checkout session
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
		IdempotencyKey: "ORD-MOCK-INT-001",
	}

	resp, err := provider.CreateCheckoutSession(ctx.Request.Context(), checkoutReq)
	if err != nil {
		t.Fatalf("failed to create checkout session: %v", err)
	}

	// Call mock checkout complete (without order - tests the flow)
	w2 := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w2)

	payload := MockCheckoutCompleteRequest{
		SessionID: resp.SessionID,
		OrderNo:   "", // No order - tests error path
	}
	body, _ := json.Marshal(payload)
	c.Request = httptest.NewRequest("POST", "/webhooks/mock/complete", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleMockCheckoutComplete(c)

	// May return error due to no order, but this tests the code path
	// We're testing that the handler executes without panicking
	if w2.Code == http.StatusServiceUnavailable {
		t.Error("mock should be enabled")
	}
}

func TestMockCheckoutComplete_InvalidSession(t *testing.T) {
	router, _, _ := createMockRouter(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := MockCheckoutCompleteRequest{
		SessionID: "nonexistent_session",
		OrderNo:   "ORD-NONEXISTENT",
	}
	body, _ := json.Marshal(payload)
	c.Request = httptest.NewRequest("POST", "/webhooks/mock/complete", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleMockCheckoutComplete(c)

	// Should return 400 for invalid session
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestMockCheckoutComplete_InvalidJSON(t *testing.T) {
	router, _, _ := createMockRouter(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/mock/complete", bytes.NewReader([]byte(`{invalid json`)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleMockCheckoutComplete(c)

	// Should return 400 for invalid JSON
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetMockSession_Enabled(t *testing.T) {
	router, _, factory := createMockRouter(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/", nil)

	// Create a checkout session
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
		IdempotencyKey: "ORD-GET-SESSION-001",
	}

	resp, err := provider.CreateCheckoutSession(ctx.Request.Context(), checkoutReq)
	if err != nil {
		t.Fatalf("failed to create checkout session: %v", err)
	}

	// Get session info
	w2 := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w2)

	c.Request = httptest.NewRequest("GET", "/webhooks/mock/session/"+resp.SessionID, nil)
	c.Params = []gin.Param{{Key: "session_id", Value: resp.SessionID}}

	router.getMockSession(c)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w2.Code, w2.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &result)
	if result["session_id"] != resp.SessionID {
		t.Errorf("expected session_id %s, got %s", resp.SessionID, result["session_id"])
	}
}

func TestGetMockSession_NotFound(t *testing.T) {
	router, _, _ := createMockRouter(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("GET", "/webhooks/mock/session/nonexistent", nil)
	c.Params = []gin.Param{{Key: "session_id", Value: "nonexistent"}}

	router.getMockSession(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetMockSession_EmptyID(t *testing.T) {
	router, _, _ := createMockRouter(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("GET", "/webhooks/mock/session/", nil)
	c.Params = []gin.Param{{Key: "session_id", Value: ""}}

	router.getMockSession(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

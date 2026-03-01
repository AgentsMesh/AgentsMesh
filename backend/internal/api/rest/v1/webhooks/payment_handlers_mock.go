package webhooks

import (
	"net/http"

	billingdomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Mock Payment Handlers (for testing)
// ===========================================

// MockCheckoutCompleteRequest represents a request to complete a mock checkout
type MockCheckoutCompleteRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	OrderNo   string `json:"order_no"`
}

// handleMockCheckoutComplete handles mock checkout completion
func (r *WebhookRouter) handleMockCheckoutComplete(c *gin.Context) {
	// Check if mock is enabled
	if r.paymentFactory == nil || !r.paymentFactory.IsMockEnabled() {
		r.logger.Warn("mock checkout complete requested but mock is not enabled")
		apierr.Forbidden(c, apierr.MOCK_PAYMENT_DISABLED, "Mock payment is not enabled")
		return
	}

	var req MockCheckoutCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Get the mock provider
	mockProvider := r.paymentFactory.GetMockProvider()
	if mockProvider == nil {
		apierr.InternalError(c, "mock provider not available")
		return
	}

	// Complete the session
	session, err := mockProvider.CompleteSession(req.SessionID)
	if err != nil {
		r.logger.Error("failed to complete mock session", "error", err, "session_id", req.SessionID)
		apierr.ValidationError(c, err.Error())
		return
	}

	r.logger.Info("mock checkout completed",
		"session_id", req.SessionID,
		"order_no", req.OrderNo,
	)

	// Create a webhook event to process
	event := &payment.WebhookEvent{
		EventID:         "mock_evt_" + req.SessionID,
		EventType:       billingdomain.WebhookEventCheckoutCompleted,
		Provider:        "mock",
		OrderNo:         req.OrderNo,
		ExternalOrderNo: req.SessionID,
		CustomerID:      session.CustomerID,
		SubscriptionID:  session.SubscriptionID,
		Amount:          session.Request.ActualAmount,
		Currency:        session.Request.Currency,
		Status:          billingdomain.OrderStatusSucceeded,
	}

	// Process the payment success
	if err := r.billingSvc.HandlePaymentSucceeded(c, event); err != nil {
		r.logger.Error("failed to process mock payment",
			"error", err,
			"session_id", req.SessionID,
		)
		apierr.InternalError(c, "failed to process payment")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"session_id":      req.SessionID,
		"order_no":        req.OrderNo,
		"customer_id":     session.CustomerID,
		"subscription_id": session.SubscriptionID,
	})
}

// getMockSession retrieves mock session information
func (r *WebhookRouter) getMockSession(c *gin.Context) {
	// Check if mock is enabled
	if r.paymentFactory == nil || !r.paymentFactory.IsMockEnabled() {
		apierr.Forbidden(c, apierr.MOCK_PAYMENT_DISABLED, "Mock payment is not enabled")
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "session_id is required")
		return
	}

	// Get the mock provider
	mockProvider := r.paymentFactory.GetMockProvider()
	if mockProvider == nil {
		apierr.InternalError(c, "mock provider not available")
		return
	}

	// Get session info
	session, err := mockProvider.GetSession(sessionID)
	if err != nil {
		apierr.ResourceNotFound(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":      session.ID,
		"status":          session.Status,
		"created_at":      session.CreatedAt,
		"expires_at":      session.ExpiresAt,
		"completed_at":    session.CompletedAt,
		"customer_id":     session.CustomerID,
		"subscription_id": session.SubscriptionID,
		"order_type":      session.Request.OrderType,
		"amount":          session.Request.ActualAmount,
		"currency":        session.Request.Currency,
	})
}

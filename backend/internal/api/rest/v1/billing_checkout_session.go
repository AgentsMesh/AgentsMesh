package v1

import (
	"fmt"
	"log/slog"
	"net/http"

	billingdomain "github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	billingService "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// createCheckoutSession creates the checkout session and stores the order
func (h *BillingHandler) createCheckoutSession(c *gin.Context, tenant *middleware.TenantContext, req *CreateCheckoutRequest, priceCalc *billingService.PriceCalculation, providerName string, providerInterface interface{}) {
	provider := providerInterface.(payment.Provider)

	// Extract values from price calculation
	amount := priceCalc.Amount
	actualAmount := priceCalc.ActualAmount
	seats := priceCalc.Seats
	var planID *int64
	if priceCalc.PlanID > 0 {
		planID = &priceCalc.PlanID
	}

	// Generate order number
	orderNo := fmt.Sprintf("ORD-%d-%s", tenant.OrganizationID, uuid.New().String()[:8])

	// Build metadata with provider-specific IDs
	metadata := map[string]string{
		"order_no": orderNo,
	}
	if priceCalc.LemonSqueezyVariantID != "" {
		metadata["variant_id"] = priceCalc.LemonSqueezyVariantID
	}
	if priceCalc.StripePrice != "" {
		metadata["stripe_price_id"] = priceCalc.StripePrice
	}

	// Get user email from JWT claims (set by auth middleware)
	userEmail, _ := c.Get("email")
	userEmailStr, _ := userEmail.(string)

	// Create checkout session
	checkoutReq := &payment.CheckoutRequest{
		OrganizationID: tenant.OrganizationID,
		UserID:         tenant.UserID,
		UserEmail:      userEmailStr,
		OrderType:      req.OrderType,
		PlanID:         0,
		BillingCycle:   req.BillingCycle,
		Seats:          seats,
		Currency:       "usd",
		Amount:         amount,
		ActualAmount:   actualAmount,
		SuccessURL:     req.SuccessURL,
		CancelURL:      req.CancelURL,
		IdempotencyKey: orderNo,
		Metadata:       metadata,
	}
	if planID != nil {
		checkoutReq.PlanID = *planID
	}

	resp, err := provider.CreateCheckoutSession(c.Request.Context(), checkoutReq)
	if err != nil {
		apierr.InternalError(c, fmt.Sprintf("failed to create checkout: %v", err))
		return
	}

	// Store order in database
	order := &billingdomain.PaymentOrder{
		OrganizationID:  tenant.OrganizationID,
		OrderNo:         orderNo,
		ExternalOrderNo: &resp.ExternalOrderNo,
		OrderType:       req.OrderType,
		PlanID:          planID,
		BillingCycle:    req.BillingCycle,
		Seats:           seats,
		Amount:          amount,
		ActualAmount:    actualAmount,
		Currency:        "usd",
		Status:          billingdomain.OrderStatusPending,
		PaymentProvider: providerName,
		ExpiresAt:       &resp.ExpiresAt,
		CreatedByID:     tenant.UserID,
	}
	if err := h.billingService.CreatePaymentOrder(c.Request.Context(), order); err != nil {
		// Order must be persisted before returning the checkout URL to the user.
		// Without a local order record, webhook reconciliation will be unreliable.
		slog.Error("failed to save payment order",
			"order_no", orderNo, "error", err)
		apierr.InternalError(c, "failed to create payment order")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_no":    orderNo,
		"session_id":  resp.SessionID,
		"session_url": resp.SessionURL,
		"qr_code_url": resp.QRCodeURL,
		"expires_at":  resp.ExpiresAt,
		"provider":    providerName,
	})
}

// GetCheckoutStatus returns the status of a checkout/order
func (h *BillingHandler) GetCheckoutStatus(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)
	orderNo := c.Param("order_no")

	order, err := h.billingService.GetPaymentOrderByNo(c.Request.Context(), orderNo)
	if err != nil {
		apierr.ResourceNotFound(c, "order not found")
		return
	}

	// Verify ownership
	if order.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_no":   order.OrderNo,
		"status":     order.Status,
		"order_type": order.OrderType,
		"amount":     order.ActualAmount,
		"currency":   order.Currency,
		"created_at": order.CreatedAt,
		"paid_at":    order.PaidAt,
	})
}

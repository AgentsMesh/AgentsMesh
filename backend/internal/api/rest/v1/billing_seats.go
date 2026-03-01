package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	billingsvc "github.com/AgentsMesh/AgentsMesh/backend/internal/service/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// GetSeatUsage returns seat usage information
func (h *BillingHandler) GetSeatUsage(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	usage, err := h.billingService.GetSeatUsage(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		if err == billingsvc.ErrSubscriptionNotFound {
			// Return default for free plan
			c.JSON(http.StatusOK, gin.H{
				"total_seats":     1,
				"used_seats":      1,
				"available_seats": 0,
				"max_seats":       1,
				"can_add_seats":   false,
			})
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, usage)
}

// PurchaseSeatsRequest represents a seat purchase request
type PurchaseSeatsRequest struct {
	Seats      int    `json:"seats" binding:"required,min=1"`
	SuccessURL string `json:"success_url" binding:"required"`
	CancelURL  string `json:"cancel_url" binding:"required"`
}

// PurchaseSeats initiates a seat purchase checkout
func (h *BillingHandler) PurchaseSeats(c *gin.Context) {
	var req PurchaseSeatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Redirect to CreateCheckout with seat_purchase type
	checkoutReq := CreateCheckoutRequest{
		OrderType:  billing.OrderTypeSeatPurchase,
		Seats:      req.Seats,
		SuccessURL: req.SuccessURL,
		CancelURL:  req.CancelURL,
	}

	// Marshal and re-bind
	c.Set("checkout_request", checkoutReq)
	h.CreateCheckout(c)
}

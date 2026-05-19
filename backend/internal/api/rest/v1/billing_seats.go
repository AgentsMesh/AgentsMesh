package v1

import (
	"errors"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	billingsvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *BillingHandler) GetSeatUsage(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	usage, err := h.billingService.GetSeatUsage(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		if errors.Is(err, billingsvc.ErrSubscriptionNotFound) {
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

type PurchaseSeatsRequest struct {
	Seats int `json:"seats" binding:"required,min=1"`
}

func (h *BillingHandler) PurchaseSeats(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	var req PurchaseSeatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if err := h.billingService.UpdateSeats(c.Request.Context(), tenant.OrganizationID, req.Seats); err != nil {
		switch {
		case errors.Is(err, billingsvc.ErrSubscriptionNotFound):
			apierr.ResourceNotFound(c, "no active subscription")
		case errors.Is(err, billingsvc.ErrSubscriptionNotActive):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "subscription is not active")
		case errors.Is(err, billingsvc.ErrSubscriptionFrozen):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "subscription is frozen, please renew first")
		case errors.Is(err, billingsvc.ErrInvalidPlan):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "cannot add seats to based plan, please upgrade first")
		case errors.Is(err, billingsvc.ErrQuotaExceeded):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "exceeds maximum seats for this plan")
		default:
			apierr.InternalError(c, err.Error())
		}
		return
	}

	usage, err := h.billingService.GetSeatUsage(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "seats updated successfully"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "seats updated successfully",
		"seats":   usage,
	})
}

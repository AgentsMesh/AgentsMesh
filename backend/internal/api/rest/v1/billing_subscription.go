package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	billingsvc "github.com/AgentsMesh/AgentsMesh/backend/internal/service/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Basic Subscription Operations (CRUD)
// ===========================================

// CreateSubscriptionRequest represents the subscription creation request
type CreateSubscriptionRequest struct {
	PlanName     string `json:"plan_name" binding:"required"`
	BillingCycle string `json:"billing_cycle"` // monthly or yearly
}

// GetSubscription returns the current subscription
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		if err == billingsvc.ErrSubscriptionNotFound {
			apierr.ResourceNotFound(c, "no active subscription")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscription": sub})
}

// CreateSubscription creates a new subscription
func (h *BillingHandler) CreateSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	sub, err := h.billingService.CreateSubscription(c.Request.Context(), tenant.OrganizationID, req.PlanName)
	if err != nil {
		if err == billingsvc.ErrPlanNotFound {
			apierr.InvalidInput(c, "invalid plan")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"subscription": sub})
}

// UpdateSubscriptionRequest represents the subscription update request
type UpdateSubscriptionRequest struct {
	PlanName string `json:"plan_name" binding:"required"`
}

// UpdateSubscription updates the subscription plan
func (h *BillingHandler) UpdateSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	var req UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	sub, err := h.billingService.UpdateSubscription(c.Request.Context(), tenant.OrganizationID, req.PlanName)
	if err != nil {
		if err == billingsvc.ErrSubscriptionNotFound {
			apierr.ResourceNotFound(c, "no active subscription")
			return
		}
		if err == billingsvc.ErrPlanNotFound {
			apierr.InvalidInput(c, "invalid plan")
			return
		}
		if err == billingsvc.ErrSeatCountExceedsLimit {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "current seat count exceeds target plan limit, please reduce seats first")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	// Check if downgrade is scheduled
	response := gin.H{"subscription": sub}
	if sub.DowngradeToPlan != nil {
		response["message"] = "downgrade scheduled for end of billing period"
		response["downgrade_to"] = *sub.DowngradeToPlan
	}

	c.JSON(http.StatusOK, response)
}

// CancelSubscription cancels the current subscription
func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if err := h.billingService.CancelSubscription(c.Request.Context(), tenant.OrganizationID); err != nil {
		if err == billingsvc.ErrSubscriptionNotFound {
			apierr.ResourceNotFound(c, "no active subscription")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription cancelled"})
}

package admin

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	billingservice "github.com/anthropics/agentsmesh/backend/internal/service/billing"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// AdminCreateSubscription creates a new subscription for an organization that doesn't have one
func (h *SubscriptionHandler) AdminCreateSubscription(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	var req struct {
		PlanName string `json:"plan_name" binding:"required"`
		Months   int    `json:"months"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "plan_name is required")
		return
	}

	if req.Months <= 0 {
		req.Months = 1
	}

	newSub, err := h.billingService.AdminCreateSubscription(c.Request.Context(), orgID, req.PlanName, req.Months)
	if err != nil {
		if err == billingservice.ErrPlanNotFound {
			apierr.ResourceNotFound(c, "Plan not found")
			return
		}
		if err == billingservice.ErrSubscriptionAlreadyExists {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Subscription already exists for this organization")
			return
		}
		apierr.InternalError(c, "Failed to create subscription")
		return
	}

	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID, nil, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

// AdminUpdatePlan changes the subscription plan directly without payment checks
func (h *SubscriptionHandler) AdminUpdatePlan(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	var req struct {
		PlanName string `json:"plan_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "plan_name is required")
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	newSub, err := h.billingService.AdminUpdatePlan(c.Request.Context(), orgID, req.PlanName)
	if err != nil {
		if err == billingservice.ErrPlanNotFound {
			apierr.ResourceNotFound(c, "Plan not found")
			return
		}
		if err == billingservice.ErrSubscriptionNotFound {
			apierr.ResourceNotFound(c, "Subscription not found")
			return
		}
		apierr.InternalError(c, "Failed to update plan")
		return
	}

	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

// AdminUpdateSeats changes the seat count directly
func (h *SubscriptionHandler) AdminUpdateSeats(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	var req struct {
		SeatCount int `json:"seat_count" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "seat_count must be a positive integer")
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.AdminSetSeatCount(c.Request.Context(), orgID, req.SeatCount); err != nil {
		apierr.InternalError(c, "Failed to update seat count")
		return
	}

	newSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)
	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

// AdminUpdateCycle changes the billing cycle
func (h *SubscriptionHandler) AdminUpdateCycle(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	var req struct {
		BillingCycle string `json:"billing_cycle" binding:"required,oneof=monthly yearly"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "billing_cycle must be 'monthly' or 'yearly'")
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.SetNextBillingCycle(c.Request.Context(), orgID, req.BillingCycle); err != nil {
		apierr.InternalError(c, "Failed to update billing cycle")
		return
	}

	newSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)
	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

package admin

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	billingservice "github.com/anthropics/agentsmesh/backend/internal/service/billing"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// Freeze freezes a subscription
func (h *SubscriptionHandler) Freeze(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.FreezeSubscription(c.Request.Context(), orgID); err != nil {
		apierr.InternalError(c, "Failed to freeze subscription")
		return
	}

	// Sync organization table
	h.syncOrgStatus(c, orgID, billing.SubscriptionStatusFrozen)

	newSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)
	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubFreeze, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

// Unfreeze reactivates a frozen subscription
func (h *SubscriptionHandler) Unfreeze(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	// Default to monthly if no cycle specified
	cycle := billing.BillingCycleMonthly
	if oldSub != nil && oldSub.BillingCycle != "" {
		cycle = oldSub.BillingCycle
	}

	if err := h.billingService.UnfreezeSubscription(c.Request.Context(), orgID, cycle); err != nil {
		apierr.InternalError(c, "Failed to unfreeze subscription")
		return
	}

	// Sync organization table
	h.syncOrgStatus(c, orgID, billing.SubscriptionStatusActive)

	newSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)
	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubUnfreeze, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

// Cancel cancels a subscription without calling external payment APIs
func (h *SubscriptionHandler) Cancel(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.AdminCancelSubscription(c.Request.Context(), orgID); err != nil {
		apierr.InternalError(c, "Failed to cancel subscription")
		return
	}

	newSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)
	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubCancel, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

// AdminRenew extends the subscription by the specified number of months
func (h *SubscriptionHandler) AdminRenew(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	var req struct {
		Months int `json:"months" binding:"required,min=1,max=120"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "months must be between 1 and 120")
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	newSub, err := h.billingService.AdminRenew(c.Request.Context(), orgID, req.Months)
	if err != nil {
		if err == billingservice.ErrSubscriptionNotFound {
			apierr.ResourceNotFound(c, "Subscription not found")
			return
		}
		apierr.InternalError(c, "Failed to renew subscription")
		return
	}

	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubRenew, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

// SetAutoRenew toggles auto-renewal
func (h *SubscriptionHandler) SetAutoRenew(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	var req struct {
		AutoRenew bool `json:"auto_renew"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "auto_renew is required")
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.SetAutoRenew(c.Request.Context(), orgID, req.AutoRenew); err != nil {
		apierr.InternalError(c, "Failed to update auto-renew")
		return
	}

	newSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)
	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

// SetCustomQuota sets a custom quota override for a resource
func (h *SubscriptionHandler) SetCustomQuota(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	var req struct {
		Resource string `json:"resource" binding:"required"`
		Limit    int    `json:"limit" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "resource and limit are required")
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.SetCustomQuota(c.Request.Context(), orgID, req.Resource, req.Limit); err != nil {
		if err == billingservice.ErrSubscriptionNotFound {
			apierr.ResourceNotFound(c, "Subscription not found")
			return
		}
		apierr.InternalError(c, "Failed to set custom quota")
		return
	}

	newSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)
	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubQuota, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

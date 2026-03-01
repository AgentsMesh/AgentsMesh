package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	billingsvc "github.com/AgentsMesh/AgentsMesh/backend/internal/service/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Subscription Plan Changes (Downgrade, Billing Cycle)
// ===========================================

// DowngradeSubscriptionRequest represents a downgrade request
type DowngradeSubscriptionRequest struct {
	PlanName string `json:"plan_name" binding:"required"`
}

// DowngradeSubscription schedules a downgrade to a lower plan at period end
func (h *BillingHandler) DowngradeSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	var req DowngradeSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	ctx := c.Request.Context()

	// Get current subscription
	sub, err := h.billingService.GetSubscription(ctx, tenant.OrganizationID)
	if err != nil {
		apierr.ResourceNotFound(c, "no active subscription")
		return
	}

	// Get target plan
	targetPlan, err := h.billingService.GetPlan(ctx, req.PlanName)
	if err != nil {
		apierr.InvalidInput(c, "invalid plan")
		return
	}

	// Get current plan
	currentPlan := sub.Plan
	if currentPlan == nil {
		currentPlan, _ = h.billingService.GetPlanByID(ctx, sub.PlanID)
	}

	// Verify this is actually a downgrade
	if targetPlan.PricePerSeatMonthly >= currentPlan.PricePerSeatMonthly {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "use upgrade endpoint for higher tier plans")
		return
	}

	// Check if current seat count exceeds target plan limit
	if targetPlan.MaxUsers > 0 && sub.SeatCount > targetPlan.MaxUsers {
		apierr.RespondWithExtra(c, http.StatusBadRequest, apierr.VALIDATION_FAILED, "current seat count exceeds target plan limit", gin.H{
			"current_seats":     sub.SeatCount,
			"target_plan_limit": targetPlan.MaxUsers,
			"action_required":   "reduce seats before downgrading",
		})
		return
	}

	// Schedule downgrade via UpdateSubscription (handles downgrade logic)
	_, err = h.billingService.UpdateSubscription(ctx, tenant.OrganizationID, req.PlanName)
	if err != nil {
		if err == billingsvc.ErrSeatCountExceedsLimit {
			apierr.RespondWithExtra(c, http.StatusBadRequest, apierr.VALIDATION_FAILED, "current seat count exceeds target plan limit", gin.H{
				"action_required": "reduce seats before downgrading",
			})
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "downgrade scheduled for end of billing period",
		"downgrade_to_plan": req.PlanName,
		"effective_date":    sub.CurrentPeriodEnd,
	})
}

// ChangeBillingCycleRequest represents a billing cycle change request
type ChangeBillingCycleRequest struct {
	BillingCycle string `json:"billing_cycle" binding:"required,oneof=monthly yearly"`
}

// ChangeBillingCycle changes the billing cycle for next renewal
func (h *BillingHandler) ChangeBillingCycle(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	var req ChangeBillingCycleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		apierr.ResourceNotFound(c, "no active subscription")
		return
	}

	if sub.BillingCycle == req.BillingCycle {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "already on this billing cycle")
		return
	}

	// Set next billing cycle (takes effect on renewal)
	if err := h.billingService.SetNextBillingCycle(c.Request.Context(), tenant.OrganizationID, req.BillingCycle); err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "billing cycle will change on next renewal",
		"current_cycle":  sub.BillingCycle,
		"next_cycle":     req.BillingCycle,
		"effective_date": sub.CurrentPeriodEnd,
	})
}

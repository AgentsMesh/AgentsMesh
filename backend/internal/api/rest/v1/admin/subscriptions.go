package admin

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	billingservice "github.com/anthropics/agentsmesh/backend/internal/service/billing"

	"github.com/gin-gonic/gin"
)

// SubscriptionHandler handles subscription management requests
type SubscriptionHandler struct {
	adminService  *adminservice.Service
	billingService *billingservice.Service
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(adminSvc *adminservice.Service, billingSvc *billingservice.Service) *SubscriptionHandler {
	return &SubscriptionHandler{
		adminService:  adminSvc,
		billingService: billingSvc,
	}
}

// RegisterRoutes registers subscription management routes under /organizations/:id/subscription
func (h *SubscriptionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	subGroup := rg.Group("/organizations/:id/subscription")
	{
		subGroup.GET("", h.GetSubscription)
		subGroup.GET("/plans", h.ListPlans)
		subGroup.PUT("/plan", h.AdminUpdatePlan)
		subGroup.PUT("/seats", h.AdminUpdateSeats)
		subGroup.PUT("/cycle", h.AdminUpdateCycle)
		subGroup.POST("/freeze", h.Freeze)
		subGroup.POST("/unfreeze", h.Unfreeze)
		subGroup.POST("/cancel", h.Cancel)
		subGroup.POST("/renew", h.AdminRenew)
		subGroup.PUT("/auto-renew", h.SetAutoRenew)
		subGroup.PUT("/quotas", h.SetCustomQuota)
	}
}

// GetSubscription returns the full subscription details for an organization
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), orgID)
	if err != nil {
		if err == billingservice.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subscription"})
		return
	}

	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)

	h.logAction(c, admin.AuditActionSubView, admin.TargetTypeSubscription, orgID, nil, nil)

	c.JSON(http.StatusOK, subscriptionResponse(sub, seatUsage))
}

// ListPlans returns all available subscription plans
func (h *SubscriptionHandler) ListPlans(c *gin.Context) {
	plans, err := h.billingService.ListPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list plans"})
		return
	}

	planList := make([]gin.H, len(plans))
	for i, p := range plans {
		planList[i] = planResponse(p)
	}

	c.JSON(http.StatusOK, gin.H{"data": planList})
}

// AdminUpdatePlan changes the subscription plan directly without payment checks
func (h *SubscriptionHandler) AdminUpdatePlan(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	var req struct {
		PlanName string `json:"plan_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan_name is required"})
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	newSub, err := h.billingService.AdminUpdatePlan(c.Request.Context(), orgID, req.PlanName)
	if err != nil {
		if err == billingservice.ErrPlanNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
			return
		}
		if err == billingservice.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plan"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	var req struct {
		SeatCount int `json:"seat_count" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "seat_count must be a positive integer"})
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.AdminSetSeatCount(c.Request.Context(), orgID, req.SeatCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update seat count"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	var req struct {
		BillingCycle string `json:"billing_cycle" binding:"required,oneof=monthly yearly"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "billing_cycle must be 'monthly' or 'yearly'"})
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.SetNextBillingCycle(c.Request.Context(), orgID, req.BillingCycle); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update billing cycle"})
		return
	}

	newSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)
	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

// Freeze freezes a subscription
func (h *SubscriptionHandler) Freeze(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.FreezeSubscription(c.Request.Context(), orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to freeze subscription"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	// Default to monthly if no cycle specified
	cycle := billing.BillingCycleMonthly
	if oldSub != nil && oldSub.BillingCycle != "" {
		cycle = oldSub.BillingCycle
	}

	if err := h.billingService.UnfreezeSubscription(c.Request.Context(), orgID, cycle); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unfreeze subscription"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.AdminCancelSubscription(c.Request.Context(), orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel subscription"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	var req struct {
		Months int `json:"months" binding:"required,min=1,max=120"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "months must be between 1 and 120"})
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	newSub, err := h.billingService.AdminRenew(c.Request.Context(), orgID, req.Months)
	if err != nil {
		if err == billingservice.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to renew subscription"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	var req struct {
		AutoRenew bool `json:"auto_renew"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auto_renew is required"})
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.SetAutoRenew(c.Request.Context(), orgID, req.AutoRenew); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update auto-renew"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	var req struct {
		Resource string `json:"resource" binding:"required"`
		Limit    int    `json:"limit" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource and limit are required"})
		return
	}

	oldSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)

	if err := h.billingService.SetCustomQuota(c.Request.Context(), orgID, req.Resource, req.Limit); err != nil {
		if err == billingservice.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set custom quota"})
		return
	}

	newSub, _ := h.billingService.GetSubscription(c.Request.Context(), orgID)
	seatUsage, _ := h.billingService.GetSeatUsage(c.Request.Context(), orgID)
	h.logAction(c, admin.AuditActionSubQuota, admin.TargetTypeSubscription, orgID, oldSub, newSub)

	c.JSON(http.StatusOK, subscriptionResponse(newSub, seatUsage))
}

// logAction is a helper method that delegates to the shared LogAdminAction function
func (h *SubscriptionHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

// syncOrgStatus syncs the subscription_status field in the organizations table
func (h *SubscriptionHandler) syncOrgStatus(c *gin.Context, orgID int64, status string) {
	// Use adminService to update organization status directly via DB
	_ = h.adminService.UpdateOrganizationSubscriptionStatus(c.Request.Context(), orgID, status)
}

// subscriptionResponse creates a comprehensive subscription response
func subscriptionResponse(sub *billing.Subscription, seatUsage *billingservice.SeatUsage) gin.H {
	resp := gin.H{
		"id":                   sub.ID,
		"organization_id":      sub.OrganizationID,
		"plan_id":              sub.PlanID,
		"status":               sub.Status,
		"billing_cycle":        sub.BillingCycle,
		"current_period_start": sub.CurrentPeriodStart,
		"current_period_end":   sub.CurrentPeriodEnd,
		"auto_renew":           sub.AutoRenew,
		"seat_count":           sub.SeatCount,
		"cancel_at_period_end": sub.CancelAtPeriodEnd,
		"custom_quotas":        sub.CustomQuotas,
		"created_at":           sub.CreatedAt,
		"updated_at":           sub.UpdatedAt,
	}

	// Payment info (reference only, does not restrict operations)
	if sub.PaymentProvider != nil {
		resp["payment_provider"] = *sub.PaymentProvider
	}
	if sub.PaymentMethod != nil {
		resp["payment_method"] = *sub.PaymentMethod
	}

	// Payment provider flags
	resp["has_stripe"] = sub.StripeSubscriptionID != nil
	resp["has_alipay"] = sub.AlipayAgreementNo != nil
	resp["has_wechat"] = sub.WeChatContractID != nil
	resp["has_lemonsqueezy"] = sub.LemonSqueezySubscriptionID != nil

	// Optional fields
	if sub.CanceledAt != nil {
		resp["canceled_at"] = sub.CanceledAt
	}
	if sub.FrozenAt != nil {
		resp["frozen_at"] = sub.FrozenAt
	}
	if sub.DowngradeToPlan != nil {
		resp["downgrade_to_plan"] = *sub.DowngradeToPlan
	}
	if sub.NextBillingCycle != nil {
		resp["next_billing_cycle"] = *sub.NextBillingCycle
	}

	// Plan details
	if sub.Plan != nil {
		resp["plan"] = planResponse(sub.Plan)
	}

	// Seat usage
	if seatUsage != nil {
		resp["seat_usage"] = seatUsage
	}

	return resp
}

// planResponse creates a plan response
func planResponse(p *billing.SubscriptionPlan) gin.H {
	return gin.H{
		"id":                    p.ID,
		"name":                  p.Name,
		"display_name":          p.DisplayName,
		"price_per_seat_monthly": p.PricePerSeatMonthly,
		"price_per_seat_yearly":  p.PricePerSeatYearly,
		"included_pod_minutes":   p.IncludedPodMinutes,
		"max_users":             p.MaxUsers,
		"max_runners":           p.MaxRunners,
		"max_concurrent_pods":   p.MaxConcurrentPods,
		"max_repositories":      p.MaxRepositories,
		"features":              p.Features,
		"is_active":             p.IsActive,
	}
}

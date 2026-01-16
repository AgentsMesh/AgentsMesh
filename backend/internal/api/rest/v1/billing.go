package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/gin-gonic/gin"
)

// BillingHandler handles billing-related HTTP requests
type BillingHandler struct {
	billingService *billing.Service
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(billingService *billing.Service) *BillingHandler {
	return &BillingHandler{billingService: billingService}
}

// GetOverview returns the billing overview for the organization
func (h *BillingHandler) GetOverview(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	overview, err := h.billingService.GetBillingOverview(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"overview": overview})
}

// GetSubscription returns the current subscription
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		if err == billing.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscription": sub})
}

// CreateSubscriptionRequest represents the subscription creation request
type CreateSubscriptionRequest struct {
	PlanName     string `json:"plan_name" binding:"required"`
	BillingCycle string `json:"billing_cycle"` // monthly or yearly
}

// CreateSubscription creates a new subscription
func (h *BillingHandler) CreateSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.billingService.CreateSubscription(c.Request.Context(), tenant.OrganizationID, req.PlanName)
	if err != nil {
		if err == billing.ErrPlanNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.billingService.UpdateSubscription(c.Request.Context(), tenant.OrganizationID, req.PlanName)
	if err != nil {
		if err == billing.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
			return
		}
		if err == billing.ErrPlanNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscription": sub})
}

// CancelSubscription cancels the current subscription
func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if err := h.billingService.CancelSubscription(c.Request.Context(), tenant.OrganizationID); err != nil {
		if err == billing.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription cancelled"})
}

// ListPlans returns all available subscription plans
func (h *BillingHandler) ListPlans(c *gin.Context) {
	plans, err := h.billingService.ListPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

// GetUsage returns usage statistics for the current period
func (h *BillingHandler) GetUsage(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)
	usageType := c.Query("type")

	if usageType != "" {
		usage, err := h.billingService.GetUsage(c.Request.Context(), tenant.OrganizationID, usageType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"usage": usage, "type": usageType})
		return
	}

	// Return billing overview with all usage data
	overview, err := h.billingService.GetBillingOverview(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"usage": overview.Usage})
}

// GetUsageHistory returns usage history
func (h *BillingHandler) GetUsageHistory(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)
	usageType := c.Query("type")
	monthsStr := c.DefaultQuery("months", "3")

	months, err := strconv.Atoi(monthsStr)
	if err != nil || months < 1 || months > 12 {
		months = 3
	}

	records, err := h.billingService.GetUsageHistory(c.Request.Context(), tenant.OrganizationID, usageType, months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": records})
}

// SetCustomQuotaRequest represents the custom quota setting request
type SetCustomQuotaRequest struct {
	Resource string `json:"resource" binding:"required"`
	Limit    int    `json:"limit" binding:"required"`
}

// SetCustomQuota sets a custom quota for the organization (admin only)
func (h *BillingHandler) SetCustomQuota(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	// Only owners and admins can set custom quotas
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req SetCustomQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.billingService.SetCustomQuota(c.Request.Context(), tenant.OrganizationID, req.Resource, req.Limit); err != nil {
		if err == billing.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "custom quota updated"})
}

// CheckQuota checks if the organization has quota available
func (h *BillingHandler) CheckQuota(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)
	resource := c.Query("resource")
	amountStr := c.DefaultQuery("amount", "1")

	if resource == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource parameter required"})
		return
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount < 1 {
		amount = 1
	}

	if err := h.billingService.CheckQuota(c.Request.Context(), tenant.OrganizationID, resource, amount); err != nil {
		if err == billing.ErrQuotaExceeded {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "quota exceeded", "available": false})
			return
		}
		if err == billing.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"available": true})
}

// CreateStripeCustomerRequest represents the Stripe customer creation request
type CreateStripeCustomerRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required"`
}

// CreateStripeCustomer creates a Stripe customer for the organization
func (h *BillingHandler) CreateStripeCustomer(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	// Only owners can create Stripe customers
	if tenant.UserRole != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req CreateStripeCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customerID, err := h.billingService.CreateStripeCustomer(c.Request.Context(), tenant.OrganizationID, req.Email, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"customer_id": customerID})
}

// RegisterBillingHandlers registers billing routes with actual handlers
func RegisterBillingHandlers(rg *gin.RouterGroup, billingService *billing.Service) {
	handler := NewBillingHandler(billingService)

	rg.GET("/overview", handler.GetOverview)
	rg.GET("/subscription", handler.GetSubscription)
	rg.POST("/subscription", handler.CreateSubscription)
	rg.PUT("/subscription", handler.UpdateSubscription)
	rg.DELETE("/subscription", handler.CancelSubscription)
	rg.GET("/plans", handler.ListPlans)
	rg.GET("/usage", handler.GetUsage)
	rg.GET("/usage/history", handler.GetUsageHistory)
	rg.POST("/quota", handler.SetCustomQuota)
	rg.GET("/quota/check", handler.CheckQuota)
	rg.POST("/stripe/customer", handler.CreateStripeCustomer)
}

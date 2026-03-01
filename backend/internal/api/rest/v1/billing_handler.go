package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	billingsvc "github.com/AgentsMesh/AgentsMesh/backend/internal/service/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// BillingHandler handles billing-related HTTP requests
type BillingHandler struct {
	billingService *billingsvc.Service
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(billingService *billingsvc.Service) *BillingHandler {
	return &BillingHandler{billingService: billingService}
}

// GetOverview returns the billing overview for the organization
func (h *BillingHandler) GetOverview(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	overview, err := h.billingService.GetBillingOverview(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"overview": overview})
}

// ListPlans returns all available subscription plans
func (h *BillingHandler) ListPlans(c *gin.Context) {
	plans, err := h.billingService.ListPlans(c.Request.Context())
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

// ListPlansWithPrices returns all available subscription plans with prices in specified currency
// GET /api/v1/billing/plans/prices?currency=USD
func (h *BillingHandler) ListPlansWithPrices(c *gin.Context) {
	currency := c.DefaultQuery("currency", "USD")

	plans, err := h.billingService.ListPlansWithPrices(c.Request.Context(), currency)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"plans": plans, "currency": currency})
}

// GetPlanPrices returns prices for a specific plan in specified currency
// GET /api/v1/billing/plans/:name/prices?currency=USD
func (h *BillingHandler) GetPlanPrices(c *gin.Context) {
	planName := c.Param("name")
	currency := c.DefaultQuery("currency", "USD")

	price, err := h.billingService.GetPlanPrice(c.Request.Context(), planName, currency)
	if err != nil {
		if err == billingsvc.ErrPlanNotFound {
			apierr.ResourceNotFound(c, "plan not found")
			return
		}
		if err == billingsvc.ErrPriceNotFound {
			apierr.ResourceNotFound(c, "price not found for currency")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"price": price, "currency": currency})
}

// GetAllPlanPrices returns all prices for a specific plan (all currencies)
// GET /api/v1/billing/plans/:name/all-prices
func (h *BillingHandler) GetAllPlanPrices(c *gin.Context) {
	planName := c.Param("name")

	prices, err := h.billingService.GetPlanPrices(c.Request.Context(), planName)
	if err != nil {
		if err == billingsvc.ErrPlanNotFound {
			apierr.ResourceNotFound(c, "plan not found")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"prices": prices})
}

// GetUsage returns usage statistics for the current period
func (h *BillingHandler) GetUsage(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)
	usageType := c.Query("type")

	if usageType != "" {
		usage, err := h.billingService.GetUsage(c.Request.Context(), tenant.OrganizationID, usageType)
		if err != nil {
			apierr.InternalError(c, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"usage": usage, "type": usageType})
		return
	}

	// Return billing overview with all usage data
	overview, err := h.billingService.GetBillingOverview(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		apierr.InternalError(c, err.Error())
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
		apierr.InternalError(c, err.Error())
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
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	var req SetCustomQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if err := h.billingService.SetCustomQuota(c.Request.Context(), tenant.OrganizationID, req.Resource, req.Limit); err != nil {
		if err == billingsvc.ErrSubscriptionNotFound {
			apierr.ResourceNotFound(c, "no active subscription")
			return
		}
		apierr.InternalError(c, err.Error())
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
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "resource parameter required")
		return
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount < 1 {
		amount = 1
	}

	if err := h.billingService.CheckQuota(c.Request.Context(), tenant.OrganizationID, resource, amount); err != nil {
		if err == billingsvc.ErrQuotaExceeded {
			apierr.PaymentRequiredWithExtra(c, apierr.QUOTA_EXCEEDED, "quota exceeded", gin.H{"available": false})
			return
		}
		if err == billingsvc.ErrSubscriptionFrozen {
			apierr.RespondWithExtra(c, http.StatusPaymentRequired, apierr.SUBSCRIPTION_FROZEN, "subscription is frozen, please renew to continue", gin.H{"available": false})
			return
		}
		if err == billingsvc.ErrSubscriptionNotFound {
			apierr.ResourceNotFound(c, "no active subscription")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"available": true})
}

// ListInvoices returns invoice history
func (h *BillingHandler) ListInvoices(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit < 1 || limit > 100 {
		limit = 20
	}

	invoices, err := h.billingService.GetInvoicesByOrg(c.Request.Context(), tenant.OrganizationID, limit, offset)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}

// GetDeploymentInfo returns deployment type and available payment providers
func (h *BillingHandler) GetDeploymentInfo(c *gin.Context) {
	info := h.billingService.GetDeploymentInfo()
	c.JSON(http.StatusOK, info)
}

// PublicPricingResponse represents pricing data for public display
type PublicPricingResponse struct {
	DeploymentType string              `json:"deployment_type"`
	Currency       string              `json:"currency"`
	Plans          []PublicPlanPricing `json:"plans"`
}

// PublicPlanPricing represents a plan's pricing for public display
type PublicPlanPricing struct {
	Name              string  `json:"name"`
	DisplayName       string  `json:"display_name"`
	PriceMonthly      float64 `json:"price_monthly"`
	PriceYearly       float64 `json:"price_yearly"`
	MaxUsers          int     `json:"max_users"`
	MaxRunners        int     `json:"max_runners"`
	MaxRepositories   int     `json:"max_repositories"`
	MaxConcurrentPods int     `json:"max_concurrent_pods"`
}

// GetPublicPricing returns pricing information for public display (no auth required)
// This endpoint serves as the Single Source of Truth for pricing on the landing page
// GET /api/v1/config/pricing
func (h *BillingHandler) GetPublicPricing(c *gin.Context) {
	// Get deployment info to determine currency
	info := h.billingService.GetDeploymentInfo()

	// Determine currency based on deployment type
	currency := billing.CurrencyUSD
	if info.DeploymentType == "cn" {
		currency = billing.CurrencyCNY
	}

	// Get all plans with prices for the appropriate currency
	plansWithPrices, err := h.billingService.ListPlansWithPrices(c.Request.Context(), currency)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	// Transform to public pricing format
	plans := make([]PublicPlanPricing, 0, len(plansWithPrices))
	for _, pwp := range plansWithPrices {
		plans = append(plans, PublicPlanPricing{
			Name:              pwp.Plan.Name,
			DisplayName:       pwp.Plan.DisplayName,
			PriceMonthly:      pwp.Price.PriceMonthly,
			PriceYearly:       pwp.Price.PriceYearly,
			MaxUsers:          pwp.Plan.MaxUsers,
			MaxRunners:        pwp.Plan.MaxRunners,
			MaxRepositories:   pwp.Plan.MaxRepositories,
			MaxConcurrentPods: pwp.Plan.MaxConcurrentPods,
		})
	}

	c.JSON(http.StatusOK, PublicPricingResponse{
		DeploymentType: info.DeploymentType,
		Currency:       currency,
		Plans:          plans,
	})
}

package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	billingsvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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
// GET /api/v1/config/pricing
func (h *BillingHandler) GetPublicPricing(c *gin.Context) {
	info := h.billingService.GetDeploymentInfo()

	currency := billing.CurrencyUSD
	if info.DeploymentType == "cn" {
		currency = billing.CurrencyCNY
	}

	plansWithPrices, err := h.billingService.ListPlansWithPrices(c.Request.Context(), currency)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

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

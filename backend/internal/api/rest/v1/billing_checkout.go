package v1

import (
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	billingService "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// CreateCheckoutRequest represents a checkout request
type CreateCheckoutRequest struct {
	OrderType    string `json:"order_type" binding:"required,oneof=subscription seat_purchase plan_upgrade renewal"`
	PlanName     string `json:"plan_name"`     // Required for subscription/plan_upgrade
	BillingCycle string `json:"billing_cycle"` // monthly or yearly
	Seats        int    `json:"seats"`         // Required for seat_purchase
	Provider     string `json:"provider"`      // stripe, alipay, wechat (auto-selected if not provided)
	SuccessURL   string `json:"success_url" binding:"required"`
	CancelURL    string `json:"cancel_url" binding:"required"`
}

// CreateCheckout creates a payment checkout session
func (h *BillingHandler) CreateCheckout(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	// Only owners can create checkouts
	if tenant.UserRole != "owner" {
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	var req CreateCheckoutRequest
	// Check if request was passed from another handler (e.g., PurchaseSeats)
	if passedReq, exists := c.Get("checkout_request"); exists {
		req = passedReq.(CreateCheckoutRequest)
	} else if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Validate and calculate price
	priceCalc, providerName, provider, err := h.validateAndCalculateCheckout(c, tenant, &req)
	if err != nil {
		return // Error already sent to client
	}

	// Build checkout request and create session
	h.createCheckoutSession(c, tenant, &req, priceCalc, providerName, provider)
}

// validateAndCalculateCheckout validates the request and calculates the price
func (h *BillingHandler) validateAndCalculateCheckout(c *gin.Context, tenant *middleware.TenantContext, req *CreateCheckoutRequest) (*billingService.PriceCalculation, string, interface{}, error) {
	// Validate request based on order type
	if (req.OrderType == billing.OrderTypeSubscription || req.OrderType == billing.OrderTypePlanUpgrade) && req.PlanName == "" {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "plan_name is required for subscription/plan_upgrade")
		return nil, "", nil, fmt.Errorf("validation failed")
	}
	if req.OrderType == billing.OrderTypeSeatPurchase && req.Seats <= 0 {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "seats must be positive for seat_purchase")
		return nil, "", nil, fmt.Errorf("validation failed")
	}

	// Default billing cycle
	if req.BillingCycle == "" {
		req.BillingCycle = billing.BillingCycleMonthly
	}

	// Get payment factory
	factory := h.billingService.GetPaymentFactory()
	if factory == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "payment service not configured")
		return nil, "", nil, fmt.Errorf("no payment factory")
	}

	// Determine provider
	var provider interface{}
	var err error
	var providerName string

	if req.Provider != "" {
		providerName = req.Provider
		provider, err = factory.GetProvider(providerName)
	} else {
		p, e := factory.GetDefaultProvider()
		provider = p
		err = e
		if p != nil {
			providerName = p.GetProviderName()
		}
	}

	if err != nil {
		apierr.ValidationError(c, err.Error())
		return nil, "", nil, err
	}

	// Calculate price
	priceCalc, err := h.calculateCheckoutPrice(c, tenant, req)
	if err != nil {
		return nil, "", nil, err
	}

	return priceCalc, providerName, provider, nil
}

// calculateCheckoutPrice calculates the price based on order type
func (h *BillingHandler) calculateCheckoutPrice(c *gin.Context, tenant *middleware.TenantContext, req *CreateCheckoutRequest) (*billingService.PriceCalculation, error) {
	ctx := c.Request.Context()

	switch req.OrderType {
	case billing.OrderTypeSubscription:
		seats := req.Seats
		if seats <= 0 {
			seats = 1 // Default to 1 seat
		}
		priceCalc, err := h.billingService.CalculateSubscriptionPrice(ctx, req.PlanName, req.BillingCycle, seats)
		if err != nil {
			apierr.InvalidInput(c, "invalid plan or billing cycle")
			return nil, err
		}
		return priceCalc, nil

	case billing.OrderTypePlanUpgrade:
		priceCalc, err := h.billingService.CalculateUpgradePrice(ctx, tenant.OrganizationID, req.PlanName)
		if err != nil {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, fmt.Sprintf("upgrade calculation failed: %v", err))
			return nil, err
		}
		return priceCalc, nil

	case billing.OrderTypeSeatPurchase:
		priceCalc, err := h.billingService.CalculateSeatPurchasePrice(ctx, tenant.OrganizationID, req.Seats)
		if err != nil {
			errMsg := "seat purchase calculation failed"
			switch err {
			case billingService.ErrInvalidPlan:
				errMsg = "cannot add seats to based plan, please upgrade first"
			case billingService.ErrQuotaExceeded:
				errMsg = "exceeds maximum seats for this plan"
			}
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, errMsg)
			return nil, err
		}
		return priceCalc, nil

	case billing.OrderTypeRenewal:
		priceCalc, err := h.billingService.CalculateRenewalPrice(ctx, tenant.OrganizationID, req.BillingCycle)
		if err != nil {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "no subscription to renew")
			return nil, err
		}
		return priceCalc, nil

	default:
		apierr.InvalidInput(c, "invalid order type")
		return nil, fmt.Errorf("invalid order type")
	}
}

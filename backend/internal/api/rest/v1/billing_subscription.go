package v1

import (
	"fmt"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	billingsvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	"github.com/gin-gonic/gin"
)

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
			c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscription": sub})
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
		if err == billingsvc.ErrPlanNotFound {
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
		if err == billingsvc.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
			return
		}
		if err == billingsvc.ErrPlanNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan"})
			return
		}
		if err == billingsvc.ErrSeatCountExceedsLimit {
			c.JSON(http.StatusBadRequest, gin.H{"error": "current seat count exceeds target plan limit, please reduce seats first"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription cancelled"})
}

// RequestCancelSubscriptionRequest represents a cancel subscription request
type RequestCancelSubscriptionRequest struct {
	Immediate bool `json:"immediate"` // If true, cancel immediately; if false, cancel at period end
}

// RequestCancelSubscription cancels the subscription
func (h *BillingHandler) RequestCancelSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req RequestCancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to cancel at period end
		req.Immediate = false
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}

	// If using Stripe, cancel via Stripe
	factory := h.billingService.GetPaymentFactory()
	if factory != nil && sub.StripeSubscriptionID != nil {
		provider, err := factory.GetProvider(billing.PaymentProviderStripe)
		if err == nil {
			if err := provider.CancelSubscription(c.Request.Context(), *sub.StripeSubscriptionID, req.Immediate); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to cancel subscription: %v", err)})
				return
			}
		}
	}

	// Update local subscription
	if req.Immediate {
		if err := h.billingService.CancelSubscription(c.Request.Context(), tenant.OrganizationID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "subscription cancelled"})
	} else {
		// Mark to cancel at period end and save to database
		if err := h.billingService.SetCancelAtPeriodEnd(c.Request.Context(), tenant.OrganizationID, true); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":            "subscription will be cancelled at period end",
			"current_period_end": sub.CurrentPeriodEnd,
		})
	}
}

// ReactivateSubscription undoes a pending cancellation
func (h *BillingHandler) ReactivateSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}

	// Check if subscription is set to cancel at period end
	if !sub.CancelAtPeriodEnd {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription is not pending cancellation"})
		return
	}

	// If using Stripe, reactivate via Stripe API
	factory := h.billingService.GetPaymentFactory()
	if factory != nil && sub.StripeSubscriptionID != nil {
		provider, err := factory.GetProvider(billing.PaymentProviderStripe)
		if err == nil {
			// Stripe: setting cancel_at_period_end to false reactivates the subscription
			if err := provider.CancelSubscription(c.Request.Context(), *sub.StripeSubscriptionID, false); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to reactivate subscription: %v", err)})
				return
			}
		}
	}

	// Update local subscription to remove cancel_at_period_end
	if err := h.billingService.SetCancelAtPeriodEnd(c.Request.Context(), tenant.OrganizationID, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "subscription reactivated",
		"current_period_end": sub.CurrentPeriodEnd,
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
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req ChangeBillingCycleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}

	if sub.BillingCycle == req.BillingCycle {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already on this billing cycle"})
		return
	}

	// Set next billing cycle (takes effect on renewal)
	if err := h.billingService.SetNextBillingCycle(c.Request.Context(), tenant.OrganizationID, req.BillingCycle); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "billing cycle will change on next renewal",
		"current_cycle":  sub.BillingCycle,
		"next_cycle":     req.BillingCycle,
		"effective_date": sub.CurrentPeriodEnd,
	})
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

// DowngradeSubscriptionRequest represents a downgrade request
type DowngradeSubscriptionRequest struct {
	PlanName string `json:"plan_name" binding:"required"`
}

// DowngradeSubscription schedules a downgrade to a lower plan at period end
func (h *BillingHandler) DowngradeSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req DowngradeSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Get current subscription
	sub, err := h.billingService.GetSubscription(ctx, tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}

	// Get target plan
	targetPlan, err := h.billingService.GetPlan(ctx, req.PlanName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan"})
		return
	}

	// Get current plan
	currentPlan := sub.Plan
	if currentPlan == nil {
		currentPlan, _ = h.billingService.GetPlanByID(ctx, sub.PlanID)
	}

	// Verify this is actually a downgrade
	if targetPlan.PricePerSeatMonthly >= currentPlan.PricePerSeatMonthly {
		c.JSON(http.StatusBadRequest, gin.H{"error": "use upgrade endpoint for higher tier plans"})
		return
	}

	// Check if current seat count exceeds target plan limit
	if targetPlan.MaxUsers > 0 && sub.SeatCount > targetPlan.MaxUsers {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":              "current seat count exceeds target plan limit",
			"current_seats":      sub.SeatCount,
			"target_plan_limit":  targetPlan.MaxUsers,
			"action_required":    "reduce seats before downgrading",
		})
		return
	}

	// Schedule downgrade via UpdateSubscription (handles downgrade logic)
	_, err = h.billingService.UpdateSubscription(ctx, tenant.OrganizationID, req.PlanName)
	if err != nil {
		if err == billingsvc.ErrSeatCountExceedsLimit {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":           "current seat count exceeds target plan limit",
				"action_required": "reduce seats before downgrading",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "downgrade scheduled for end of billing period",
		"downgrade_to_plan":  req.PlanName,
		"effective_date":     sub.CurrentPeriodEnd,
	})
}

// CustomerPortalRequest represents a customer portal request
type CustomerPortalRequest struct {
	ReturnURL string `json:"return_url" binding:"required"`
}

// GetCustomerPortal returns a Stripe customer portal URL
func (h *BillingHandler) GetCustomerPortal(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req CustomerPortalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}

	if sub.StripeCustomerID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no Stripe customer associated with this subscription"})
		return
	}

	factory := h.billingService.GetPaymentFactory()
	if factory == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment service not configured"})
		return
	}

	provider, err := factory.GetProvider(billing.PaymentProviderStripe)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Cast to SubscriptionProvider to access GetCustomerPortalURL
	subProvider, ok := provider.(payment.SubscriptionProvider)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "provider does not support customer portal"})
		return
	}

	portalReq := &payment.CustomerPortalRequest{
		CustomerID: *sub.StripeCustomerID,
		ReturnURL:  req.ReturnURL,
	}

	resp, err := subProvider.GetCustomerPortalURL(c.Request.Context(), portalReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create portal session: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": resp.URL})
}

// UpdateAutoRenewRequest represents an auto-renew update request
type UpdateAutoRenewRequest struct {
	AutoRenew bool `json:"auto_renew"`
}

// UpdateAutoRenew updates the auto-renew setting for a subscription
func (h *BillingHandler) UpdateAutoRenew(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req UpdateAutoRenewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current subscription to verify it exists
	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}

	// Update auto_renew setting
	if err := h.billingService.SetAutoRenew(c.Request.Context(), tenant.OrganizationID, req.AutoRenew); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return updated subscription
	sub.AutoRenew = req.AutoRenew
	c.JSON(http.StatusOK, gin.H{
		"subscription": sub,
		"auto_renew":   req.AutoRenew,
	})
}

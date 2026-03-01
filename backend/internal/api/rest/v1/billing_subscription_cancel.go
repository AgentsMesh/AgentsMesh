package v1

import (
	"fmt"
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Subscription Cancellation and Reactivation
// ===========================================

// RequestCancelSubscriptionRequest represents a cancel subscription request
type RequestCancelSubscriptionRequest struct {
	Immediate bool `json:"immediate"` // If true, cancel immediately; if false, cancel at period end
}

// RequestCancelSubscription cancels the subscription
func (h *BillingHandler) RequestCancelSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	var req RequestCancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to cancel at period end
		req.Immediate = false
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		apierr.ResourceNotFound(c, "no active subscription")
		return
	}

	// Cancel via payment provider
	factory := h.billingService.GetPaymentFactory()
	if factory != nil {
		var provider payment.Provider
		var subscriptionID string
		var providerErr error

		// Determine which provider to use based on subscription IDs
		if sub.LemonSqueezySubscriptionID != nil {
			provider, providerErr = factory.GetProvider(billing.PaymentProviderLemonSqueezy)
			subscriptionID = *sub.LemonSqueezySubscriptionID
		} else if sub.StripeSubscriptionID != nil {
			provider, providerErr = factory.GetProvider(billing.PaymentProviderStripe)
			subscriptionID = *sub.StripeSubscriptionID
		}

		if providerErr == nil && provider != nil && subscriptionID != "" {
			if err := provider.CancelSubscription(c.Request.Context(), subscriptionID, req.Immediate); err != nil {
				apierr.InternalError(c, fmt.Sprintf("failed to cancel subscription: %v", err))
				return
			}
		}
	}

	// Update local subscription
	if req.Immediate {
		if err := h.billingService.CancelSubscription(c.Request.Context(), tenant.OrganizationID); err != nil {
			apierr.InternalError(c, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "subscription cancelled"})
	} else {
		// Mark to cancel at period end and save to database
		if err := h.billingService.SetCancelAtPeriodEnd(c.Request.Context(), tenant.OrganizationID, true); err != nil {
			apierr.InternalError(c, err.Error())
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
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		apierr.ResourceNotFound(c, "no active subscription")
		return
	}

	// Check if subscription is set to cancel at period end
	if !sub.CancelAtPeriodEnd {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "subscription is not pending cancellation")
		return
	}

	// Reactivate via payment provider API
	factory := h.billingService.GetPaymentFactory()
	if factory != nil {
		var provider payment.Provider
		var subscriptionID string
		var providerErr error

		// Determine which provider to use based on subscription IDs
		if sub.LemonSqueezySubscriptionID != nil {
			provider, providerErr = factory.GetProvider(billing.PaymentProviderLemonSqueezy)
			subscriptionID = *sub.LemonSqueezySubscriptionID
		} else if sub.StripeSubscriptionID != nil {
			provider, providerErr = factory.GetProvider(billing.PaymentProviderStripe)
			subscriptionID = *sub.StripeSubscriptionID
		}

		if providerErr == nil && provider != nil && subscriptionID != "" {
			// For both Stripe and LemonSqueezy: setting cancel_at_period_end to false reactivates
			if err := provider.CancelSubscription(c.Request.Context(), subscriptionID, false); err != nil {
				apierr.InternalError(c, fmt.Sprintf("failed to reactivate subscription: %v", err))
				return
			}
		}
	}

	// Update local subscription to remove cancel_at_period_end
	if err := h.billingService.SetCancelAtPeriodEnd(c.Request.Context(), tenant.OrganizationID, false); err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "subscription reactivated",
		"current_period_end": sub.CurrentPeriodEnd,
	})
}

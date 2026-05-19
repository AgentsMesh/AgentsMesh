package v1

import (
	"fmt"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type RequestCancelSubscriptionRequest struct {
	Immediate bool `json:"immediate"` // If true, cancel immediately; if false, cancel at period end
}

func (h *BillingHandler) RequestCancelSubscription(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	var req RequestCancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Immediate = false
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		apierr.ResourceNotFound(c, "no active subscription")
		return
	}

	factory := h.billingService.GetPaymentFactory()
	if factory != nil {
		var provider payment.Provider
		var subscriptionID string
		var providerErr error

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

	if req.Immediate {
		if err := h.billingService.CancelSubscription(c.Request.Context(), tenant.OrganizationID); err != nil {
			apierr.InternalError(c, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "subscription cancelled"})
	} else {
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

	if !sub.CancelAtPeriodEnd {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "subscription is not pending cancellation")
		return
	}

	factory := h.billingService.GetPaymentFactory()
	if factory != nil {
		var provider payment.Provider
		var subscriptionID string
		var providerErr error

		if sub.LemonSqueezySubscriptionID != nil {
			provider, providerErr = factory.GetProvider(billing.PaymentProviderLemonSqueezy)
			subscriptionID = *sub.LemonSqueezySubscriptionID
		} else if sub.StripeSubscriptionID != nil {
			provider, providerErr = factory.GetProvider(billing.PaymentProviderStripe)
			subscriptionID = *sub.StripeSubscriptionID
		}

		if providerErr == nil && provider != nil && subscriptionID != "" {
			if err := provider.CancelSubscription(c.Request.Context(), subscriptionID, false); err != nil {
				apierr.InternalError(c, fmt.Sprintf("failed to reactivate subscription: %v", err))
				return
			}
		}
	}

	if err := h.billingService.SetCancelAtPeriodEnd(c.Request.Context(), tenant.OrganizationID, false); err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "subscription reactivated",
		"current_period_end": sub.CurrentPeriodEnd,
	})
}

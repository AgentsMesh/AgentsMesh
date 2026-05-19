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

type CreateStripeCustomerRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required"`
}

func (h *BillingHandler) CreateStripeCustomer(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	var req CreateStripeCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	customerID, err := h.billingService.CreateStripeCustomer(c.Request.Context(), tenant.OrganizationID, req.Email, req.Name)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"customer_id": customerID})
}

type CustomerPortalRequest struct {
	ReturnURL string `json:"return_url" binding:"required"`
}

func (h *BillingHandler) GetCustomerPortal(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	var req CustomerPortalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		apierr.ResourceNotFound(c, "no active subscription")
		return
	}

	factory := h.billingService.GetPaymentFactory()
	if factory == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "payment service not configured")
		return
	}

	var provider payment.Provider
	var customerID string
	var subscriptionID string

	if sub.LemonSqueezyCustomerID != nil {
		provider, err = factory.GetProvider(billing.PaymentProviderLemonSqueezy)
		if err != nil {
			apierr.ValidationError(c, err.Error())
			return
		}
		customerID = *sub.LemonSqueezyCustomerID
		if sub.LemonSqueezySubscriptionID != nil {
			subscriptionID = *sub.LemonSqueezySubscriptionID
		}
	} else if sub.StripeCustomerID != nil {
		provider, err = factory.GetProvider(billing.PaymentProviderStripe)
		if err != nil {
			apierr.ValidationError(c, err.Error())
			return
		}
		customerID = *sub.StripeCustomerID
		if sub.StripeSubscriptionID != nil {
			subscriptionID = *sub.StripeSubscriptionID
		}
	} else {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "no payment provider associated with this subscription")
		return
	}

	subProvider, ok := provider.(payment.SubscriptionProvider)
	if !ok {
		apierr.InternalError(c, "provider does not support customer portal")
		return
	}

	portalReq := &payment.CustomerPortalRequest{
		CustomerID:     customerID,
		SubscriptionID: subscriptionID,
		ReturnURL:      req.ReturnURL,
	}

	resp, err := subProvider.GetCustomerPortalURL(c.Request.Context(), portalReq)
	if err != nil {
		apierr.InternalError(c, fmt.Sprintf("failed to create portal session: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": resp.URL})
}

type UpdateAutoRenewRequest struct {
	AutoRenew bool `json:"auto_renew"`
}

func (h *BillingHandler) UpdateAutoRenew(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	if tenant.UserRole != "owner" {
		apierr.Forbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "insufficient permissions")
		return
	}

	var req UpdateAutoRenewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		apierr.ResourceNotFound(c, "no active subscription")
		return
	}

	if err := h.billingService.SetAutoRenew(c.Request.Context(), tenant.OrganizationID, req.AutoRenew); err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	sub.AutoRenew = req.AutoRenew
	c.JSON(http.StatusOK, gin.H{
		"subscription": sub,
		"auto_renew":   req.AutoRenew,
	})
}

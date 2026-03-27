package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	billingsvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
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
		handleQuotaError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"available": true})
}

func handleQuotaError(c *gin.Context, err error) {
	switch err {
	case billingsvc.ErrQuotaExceeded:
		apierr.PaymentRequiredWithExtra(c, apierr.QUOTA_EXCEEDED, "quota exceeded", gin.H{"available": false})
	case billingsvc.ErrSubscriptionFrozen:
		apierr.RespondWithExtra(c, http.StatusPaymentRequired, apierr.SUBSCRIPTION_FROZEN, "subscription is frozen, please renew to continue", gin.H{"available": false})
	case billingsvc.ErrSubscriptionNotFound:
		apierr.ResourceNotFound(c, "no active subscription")
	default:
		apierr.InternalError(c, err.Error())
	}
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

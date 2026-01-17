package v1

import (
	billingsvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/gin-gonic/gin"
)

// RegisterBillingHandlers registers billing routes with actual handlers
func RegisterBillingHandlers(rg *gin.RouterGroup, billingService *billingsvc.Service) {
	handler := NewBillingHandler(billingService)

	// Basic billing info
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

	// Payment checkout
	rg.POST("/checkout", handler.CreateCheckout)
	rg.GET("/checkout/:order_no", handler.GetCheckoutStatus)

	// Subscription management
	rg.POST("/subscription/cancel", handler.RequestCancelSubscription)
	rg.POST("/subscription/reactivate", handler.ReactivateSubscription)
	rg.POST("/subscription/change-cycle", handler.ChangeBillingCycle)
	rg.POST("/subscription/downgrade", handler.DowngradeSubscription)
	rg.PUT("/subscription/auto-renew", handler.UpdateAutoRenew)

	// Seat management
	rg.GET("/seats", handler.GetSeatUsage)
	rg.POST("/seats/purchase", handler.PurchaseSeats)

	// Invoice history
	rg.GET("/invoices", handler.ListInvoices)

	// Customer portal (Stripe only)
	rg.POST("/customer-portal", handler.GetCustomerPortal)

	// Deployment info
	rg.GET("/deployment", handler.GetDeploymentInfo)
}

// RegisterPublicConfigRoutes registers public config routes that don't require authentication
// These endpoints provide deployment configuration information for the frontend
func RegisterPublicConfigRoutes(rg *gin.RouterGroup, billingService *billingsvc.Service) {
	handler := NewBillingHandler(billingService)

	// Deployment info - returns deployment type (global/cn/onpremise) and available payment providers
	rg.GET("/deployment", handler.GetDeploymentInfo)
}

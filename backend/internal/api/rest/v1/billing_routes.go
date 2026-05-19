package v1

import (
	billingsvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/gin-gonic/gin"
)

func RegisterBillingHandlers(rg *gin.RouterGroup, billingService *billingsvc.Service) {
	handler := NewBillingHandler(billingService)

	rg.GET("/overview", handler.GetOverview)
	rg.GET("/subscription", handler.GetSubscription)
	rg.POST("/subscription", handler.CreateSubscription)
	rg.PUT("/subscription", handler.UpdateSubscription)
	rg.DELETE("/subscription", handler.CancelSubscription)
	rg.GET("/plans", handler.ListPlans)
	rg.GET("/plans/prices", handler.ListPlansWithPrices)
	rg.GET("/plans/:name/prices", handler.GetPlanPrices)
	rg.GET("/plans/:name/all-prices", handler.GetAllPlanPrices)
	rg.GET("/usage", handler.GetUsage)
	rg.GET("/usage/history", handler.GetUsageHistory)
	rg.POST("/quota", handler.SetCustomQuota)
	rg.GET("/quota/check", handler.CheckQuota)
	rg.POST("/stripe/customer", handler.CreateStripeCustomer)

	rg.POST("/checkout", handler.CreateCheckout)
	rg.GET("/checkout/:order_no", handler.GetCheckoutStatus)

	rg.POST("/subscription/cancel", handler.RequestCancelSubscription)
	rg.POST("/subscription/reactivate", handler.ReactivateSubscription)
	rg.POST("/subscription/change-cycle", handler.ChangeBillingCycle)
	rg.POST("/subscription/upgrade", handler.UpgradeSubscription)
	rg.POST("/subscription/downgrade", handler.DowngradeSubscription)
	rg.PUT("/subscription/auto-renew", handler.UpdateAutoRenew)

	rg.GET("/seats", handler.GetSeatUsage)
	rg.POST("/seats/purchase", handler.PurchaseSeats)

	rg.GET("/invoices", handler.ListInvoices)

	rg.POST("/customer-portal", handler.GetCustomerPortal)

	rg.GET("/deployment", handler.GetDeploymentInfo)
}

func RegisterPublicConfigRoutes(rg *gin.RouterGroup, billingService *billingsvc.Service) {
	handler := NewBillingHandler(billingService)

	rg.GET("/deployment", handler.GetDeploymentInfo)

	rg.GET("/pricing", handler.GetPublicPricing)
}

package v1

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func registerBillingRoutes(rg *gin.RouterGroup, svc *Services) {
	RegisterBillingHandlers(rg.Group("/billing"), svc.Billing)

	if svc.PromoCode != nil {
		RegisterPromoCodeRoutes(rg.Group("/billing/promo-codes"), svc.PromoCode)
	}
}

func registerBindingRoutes(rg *gin.RouterGroup, svc *Services) {
	bindingHandler := NewBindingHandler(svc.Binding)
	bindings := rg.Group("/bindings")
	{
		bindings.POST("", bindingHandler.RequestBinding)
		bindings.GET("", bindingHandler.ListBindings)
		bindings.POST("/accept", bindingHandler.AcceptBinding)
		bindings.POST("/reject", bindingHandler.RejectBinding)
		bindings.POST("/unbind", bindingHandler.Unbind)
		bindings.GET("/pending", bindingHandler.GetPendingBindings)
		bindings.GET("/pods", bindingHandler.GetBoundPods)
		bindings.GET("/check/:target_pod", bindingHandler.CheckBinding)
		bindings.POST("/:id/scopes", bindingHandler.RequestScopes)
		bindings.POST("/:id/scopes/approve", bindingHandler.ApproveScopes)
	}
}

func registerInvitationRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.Invitation != nil {
		invitationHandler := NewInvitationHandler(svc.Invitation, svc.Org, svc.User, svc.Billing)
		invitationHandler.RegisterOrgRoutes(rg)
	}
}

func registerFileRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.File == nil {
		slog.Warn("File service is nil, file routes not registered")
		return
	}
	slog.Info("Registering file routes", "service", "file")
	fileHandler := NewFileHandler(svc.File)
	files := rg.Group("/files")
	{
		files.POST("/presign", fileHandler.PresignUpload)
	}
}

func registerNotificationRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.NotificationPrefStore == nil {
		return
	}
	handler := NewNotificationHandler(svc.NotificationPrefStore)
	notifications := rg.Group("/notifications")
	{
		notifications.GET("/preferences", handler.GetPreferences)
		notifications.PUT("/preferences", handler.SetPreference)
	}
}

func registerTokenUsageRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.TokenUsage == nil {
		return
	}
	RegisterTokenUsageRoutes(rg, svc.TokenUsage)
}

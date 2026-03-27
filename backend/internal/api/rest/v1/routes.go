package v1

import (
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/gin-gonic/gin"
)

// RegisterAllRoutes registers all API v1 routes with proper handlers.
func RegisterAllRoutes(rg *gin.RouterGroup, cfg *config.Config, svc *Services) {
	authHandler := NewAuthHandler(svc.Auth, svc.User, svc.Email, cfg)
	authHandler.RegisterRoutes(rg.Group("/auth"))

	RegisterUserRoutes(rg.Group("/users"), svc.User, svc.Org, svc.AgentSvc, svc.CredentialProfile, svc.UserConfig, svc.AgentPodSettings, svc.AgentPodAIProvider)
	RegisterOrganizationRoutes(rg.Group("/orgs"), svc.Org, svc.User)
	RegisterAdminRoutes(rg.Group("/admin"), svc)
	RegisterLicenseHandlers(rg.Group("/license"), svc.License)

	if svc.GRPCRunnerHandler != nil {
		RegisterGRPCRunnerRoutes(rg, svc.GRPCRunnerHandler)
	}
}

// RegisterAdminRoutes registers admin-only routes.
func RegisterAdminRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.PromoCode != nil {
		RegisterAdminPromoCodeRoutes(rg.Group("/promo-codes"), svc.PromoCode)
	}
}

// RegisterOrgScopedRoutes registers organization-scoped routes (require tenant context).
func RegisterOrgScopedRoutes(rg *gin.RouterGroup, svc *Services) {
	slog.Info("RegisterOrgScopedRoutes called", "file_svc_nil", svc.File == nil)

	registerAgentRoutes(rg, svc)
	registerRepositoryRoutes(rg, svc)
	registerRunnerRoutes(rg, svc)
	registerPodRoutes(rg, svc)
	registerChannelRoutes(rg, svc)
	registerTicketRoutes(rg, svc)
	registerBillingRoutes(rg, svc)
	registerBindingRoutes(rg, svc)
	registerMessageRoutes(rg, svc)
	registerInvitationRoutes(rg, svc)
	registerFileRoutes(rg, svc)
	registerAPIKeyManagementRoutes(rg, svc)
	registerExtensionRoutes(rg, svc)
	registerLoopRoutes(rg, svc)
	registerNotificationRoutes(rg, svc)
	registerTokenUsageRoutes(rg, svc)
}

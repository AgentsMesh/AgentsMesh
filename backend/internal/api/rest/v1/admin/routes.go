package admin

import (
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/internal/service/supportticket"

	"github.com/gin-gonic/gin"
)

type Services struct {
	Auth              *auth.Service
	Admin             *admin.Service
	Billing           *billing.Service
	SSO               *ssoservice.Service
	RelayManager      *relay.Manager
	ExtensionRepo     extension.Repository
	MarketplaceWorker *extensionservice.MarketplaceWorker
	SupportTicket     *supportticket.Service
}

func RegisterRoutes(router *gin.Engine, cfg *config.Config, db database.DB, svc *Services) {
	adminAPI := router.Group("/api/v1/admin")

	authHandler := NewAuthHandler(svc.Auth, cfg)
	authHandler.RegisterRoutes(adminAPI)

	protected := adminAPI.Group("")
	protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	protected.Use(middleware.AdminMiddleware(db))

	protected.GET("/me", authHandler.GetMe)

	dashboardHandler := NewDashboardHandler(svc.Admin)
	dashboardHandler.RegisterRoutes(protected)

	userHandler := NewUserHandler(svc.Admin)
	userHandler.RegisterRoutes(protected)

	orgHandler := NewOrganizationHandler(svc.Admin)
	orgHandler.RegisterRoutes(protected)

	runnerHandler := NewRunnerHandler(svc.Admin)
	runnerHandler.RegisterRoutes(protected)

	auditLogHandler := NewAuditLogHandler(svc.Admin)
	auditLogHandler.RegisterRoutes(protected)

	promoCodeHandler := NewPromoCodeHandler(svc.Admin)
	promoCodeHandler.RegisterRoutes(protected)

	if svc.Billing != nil {
		subscriptionHandler := NewSubscriptionHandler(svc.Admin, svc.Billing)
		subscriptionHandler.RegisterRoutes(protected)
	}

	if svc.RelayManager != nil {
		relayHandler := NewRelayHandler(svc.Admin, svc.RelayManager)
		relayHandler.RegisterRoutes(protected)
	}

	if svc.ExtensionRepo != nil {
		skillRegistryHandler := NewSkillRegistryHandler(svc.ExtensionRepo, svc.MarketplaceWorker)
		skillRegistryHandler.RegisterRoutes(protected)
	}

	if svc.SSO != nil {
		ssoHandler := NewSSOHandler(svc.SSO, svc.Admin)
		ssoHandler.RegisterRoutes(protected)
	}

	if svc.SupportTicket != nil {
		supportTicketHandler := NewSupportTicketHandler(svc.SupportTicket, svc.Admin)
		supportTicketHandler.RegisterRoutes(protected)
	}
}

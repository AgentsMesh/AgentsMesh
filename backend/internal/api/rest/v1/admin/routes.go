package admin

import (
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"

	"github.com/gin-gonic/gin"
)

// Services contains all admin-related services
type Services struct {
	Auth              *auth.Service
	Admin             *admin.Service
	Billing           *billing.Service
	RelayManager      *relay.Manager
	ExtensionRepo     extension.Repository
	MarketplaceWorker *extensionservice.MarketplaceWorker
}

// RegisterRoutes registers all admin console routes
func RegisterRoutes(router *gin.Engine, cfg *config.Config, db database.DB, svc *Services) {
	// Admin API v1 routes
	adminAPI := router.Group("/api/v1/admin")

	// Auth routes (public - no middleware)
	authHandler := NewAuthHandler(svc.Auth, cfg)
	authHandler.RegisterRoutes(adminAPI)

	// Protected routes (require auth + admin privileges)
	protected := adminAPI.Group("")
	protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	protected.Use(middleware.AdminMiddleware(db))

	// Get current admin user
	protected.GET("/me", authHandler.GetMe)

	// Dashboard
	dashboardHandler := NewDashboardHandler(svc.Admin)
	dashboardHandler.RegisterRoutes(protected)

	// Users
	userHandler := NewUserHandler(svc.Admin)
	userHandler.RegisterRoutes(protected)

	// Organizations
	orgHandler := NewOrganizationHandler(svc.Admin)
	orgHandler.RegisterRoutes(protected)

	// Runners
	runnerHandler := NewRunnerHandler(svc.Admin)
	runnerHandler.RegisterRoutes(protected)

	// Audit Logs
	auditLogHandler := NewAuditLogHandler(svc.Admin)
	auditLogHandler.RegisterRoutes(protected)

	// Promo Codes
	promoCodeHandler := NewPromoCodeHandler(svc.Admin)
	promoCodeHandler.RegisterRoutes(protected)

	// Subscriptions (optional - only if billing service is available)
	if svc.Billing != nil {
		subscriptionHandler := NewSubscriptionHandler(svc.Admin, svc.Billing)
		subscriptionHandler.RegisterRoutes(protected)
	}

	// Relays (optional - only if relay manager is available)
	if svc.RelayManager != nil {
		relayHandler := NewRelayHandler(svc.Admin, svc.RelayManager)
		relayHandler.RegisterRoutes(protected)
	}

	// Skill Registries (optional - only if extension repo is available)
	if svc.ExtensionRepo != nil {
		skillRegistryHandler := NewSkillRegistryHandler(svc.ExtensionRepo, svc.MarketplaceWorker)
		skillRegistryHandler.RegisterRoutes(protected)
	}
}

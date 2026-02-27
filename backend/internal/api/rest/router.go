package rest

import (
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/api/rest/internal"
	"github.com/anthropics/agentsmesh/backend/internal/api/rest/v1"
	"github.com/anthropics/agentsmesh/backend/internal/api/rest/v1/admin"
	"github.com/anthropics/agentsmesh/backend/internal/api/rest/v1/webhooks"
	"github.com/anthropics/agentsmesh/backend/internal/api/rest/ws"
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NewRouter creates and configures the Gin router
func NewRouter(cfg *config.Config, svc *v1.Services, db *gorm.DB, logger *slog.Logger) *gin.Engine {
	if !cfg.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		slog.Error("Panic recovered in handler",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"error", recovered,
		)
		c.AbortWithStatusJSON(500, apierr.ErrorResponse{Error: "Internal server error", Code: apierr.INTERNAL_ERROR})
	}))

	// CORS configuration
	corsConfig := cors.Config{
		AllowOrigins:     cfg.Server.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Organization-Slug", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}
	if len(corsConfig.AllowOrigins) == 0 {
		corsConfig.AllowOrigins = []string{"*"}
	}
	r.Use(cors.New(corsConfig))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "agentsmesh-api",
		})
	})

	r.GET("/health/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ready",
		})
	})

	// Initialize email service
	// BaseURL is derived from PrimaryDomain
	emailSvc := email.NewService(email.Config{
		Provider:    cfg.Email.Provider,
		ResendKey:   cfg.Email.ResendKey,
		FromAddress: cfg.Email.FromAddress,
		BaseURL:     cfg.FrontendURL(), // Derived from PrimaryDomain
	})

	// API v1
	apiV1 := r.Group("/api/v1")
	{
		// Public routes (no auth required)
		authHandler := v1.NewAuthHandler(svc.Auth, svc.User, emailSvc, cfg)
		authHandler.RegisterRoutes(apiV1.Group("/auth"))

		// Public config endpoints (deployment info for frontend)
		v1.RegisterPublicConfigRoutes(apiV1.Group("/config"), svc.Billing)

		// gRPC Runner routes (public, for Runner CLI registration with mTLS)
		if svc.GRPCRunnerHandler != nil {
			v1.RegisterGRPCRunnerRoutes(apiV1, svc.GRPCRunnerHandler)
		}

		// Webhook endpoints (no auth required, use token verification)
		// Use shared billing service to ensure mock provider sessions are shared
		// Inject services for MR/Pipeline event processing
		webhookOpts := []webhooks.WebhookRouterOption{}
		if svc.Repository != nil {
			webhookOpts = append(webhookOpts, webhooks.WithRepositoryService(svc.Repository))
		}
		if svc.Webhook != nil {
			webhookOpts = append(webhookOpts, webhooks.WithWebhookService(svc.Webhook))
		}
		if svc.MRSync != nil {
			webhookOpts = append(webhookOpts, webhooks.WithMRSyncService(svc.MRSync))
		}
		if svc.Pod != nil {
			webhookOpts = append(webhookOpts, webhooks.WithPodService(svc.Pod))
		}
		if svc.EventBus != nil {
			webhookOpts = append(webhookOpts, webhooks.WithEventBus(svc.EventBus))
		}
		webhookRouter := webhooks.NewWebhookRouterWithBillingSvc(db, cfg, logger, svc.Billing, webhookOpts...)
		webhookRouter.RegisterRoutes(apiV1.Group("/webhooks"))

		// Public invitation routes (token-based access)
		if svc.Invitation != nil {
			invitationHandler := v1.NewInvitationHandler(svc.Invitation, svc.Org, svc.User, svc.Billing)
			invitationHandler.RegisterRoutes(apiV1, middleware.AuthMiddleware(cfg.JWT.Secret))
		}

		// Protected routes (auth required)
		protected := apiV1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			// User-level routes (no tenant context required)
			v1.RegisterUserRoutes(protected.Group("/users"), svc.User, svc.Org, svc.AgentType, svc.CredentialProfile, svc.UserConfig, svc.AgentPodSettings, svc.AgentPodAIProvider)

			// Organization routes (authenticated, some require org context)
			// Path changed: /organizations → /orgs
			v1.RegisterOrganizationRoutes(protected.Group("/orgs"), svc.Org, svc.User)

			// Organization-scoped routes (require tenant context)
			// Path changed: /organizations/:slug → /orgs/:slug
			orgScoped := protected.Group("/orgs/:slug")
			orgScoped.Use(middleware.TenantMiddleware(svc.Org))
			{
				slog.Info("About to call RegisterOrgScopedRoutes")
				v1.RegisterOrgScopedRoutes(orgScoped, svc)
				slog.Info("RegisterOrgScopedRoutes completed")

				// WebSocket endpoints for real-time events
				// Note: Terminal WebSocket has been moved to Relay architecture
				// Use GET /pods/:key/terminal/connect to get Relay URL and token
				wsGroup := orgScoped.Group("/ws")
				{
					eventHandler := ws.NewEventsHandler(svc.Hub)
					wsGroup.GET("/events", eventHandler.HandleEvents)
				}
			}

			// Note: /org alias route removed - all org-scoped requests must use /orgs/:slug/*
		}

		// Note: Runner communication is now via gRPC/mTLS (see internal/api/grpc/)
		// MCP tool communication has been migrated to gRPC bidirectional stream.
		// The Pod REST API (/api/v1/orgs/:slug/pod/*) has been removed.
	}

	// External API (API key authenticated, for third-party service access)
	if svc.APIKeyAdapter != nil {
		extScoped := apiV1.Group("/ext/orgs/:slug")
		extScoped.Use(middleware.APIKeyAuthMiddleware(svc.APIKeyAdapter, svc.Org))
		{
			v1.RegisterExtRoutes(extScoped, svc)
		}
	}

	// Admin Console routes
	if cfg.Admin.IsEnabled() {
		dbWrapper := database.NewGormWrapper(db)
		adminSvc := adminservice.NewService(dbWrapper)
		admin.RegisterRoutes(r, cfg, dbWrapper, &admin.Services{
			Auth:              svc.Auth,
			Admin:             adminSvc,
			Billing:           svc.Billing,
			RelayManager:      svc.RelayManager,
			ExtensionRepo:     svc.ExtensionRepo,
			MarketplaceWorker: svc.MarketplaceWorker,
		})
	}

	// Internal API routes (Relay communication)
	if svc.RelayManager != nil {
		internal.RegisterRelayRoutes(r.Group("/api/internal/relays"), &internal.RelayRouterDeps{
			RelayManager:   svc.RelayManager,
			DNSService:     svc.RelayDNSService,
			ACMEManager:    svc.RelayACMEManager,
			InternalSecret: cfg.Server.InternalAPISecret,
		})
	}

	return r
}

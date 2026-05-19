package rest

import (
	"log/slog"
	"strings"
	"time"

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
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/gorm"
)

func NewRouter(cfg *config.Config, svc *v1.Services, db *gorm.DB, logger *slog.Logger, redisClient *redis.Client) *gin.Engine {
	if !cfg.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(otelgin.Middleware("agentsmesh-backend"))
	r.Use(gin.Logger())
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		slog.ErrorContext(c.Request.Context(), "Panic recovered in handler",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"error", recovered,
		)
		c.AbortWithStatusJSON(500, apierr.ErrorResponse{Error: "Internal server error", Code: apierr.INTERNAL_ERROR})
	}))

	allowed := cfg.Server.CORSAllowedOrigins
	if len(allowed) == 0 {
		allowed = []string{"*"}
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	wildcardAll := false
	for _, o := range allowed {
		if o == "*" {
			wildcardAll = true
		}
		allowedSet[o] = struct{}{}
	}
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Organization-Slug", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			if wildcardAll {
				return true
			}
			if _, ok := allowedSet[origin]; ok {
				return true
			}
			if origin == "null" || strings.HasPrefix(origin, "file://") {
				return true
			}
			return false
		},
	}
	r.Use(cors.New(corsConfig))

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

	emailSvc := email.NewService(email.Config{
		Provider:    cfg.Email.Provider,
		ResendKey:   cfg.Email.ResendKey,
		FromAddress: cfg.Email.FromAddress,
		BaseURL:     cfg.FrontendURL(), // Derived from PrimaryDomain
	})

	apiV1 := r.Group("/api/v1")
	{
		authHandler := v1.NewAuthHandler(svc.Auth, svc.User, emailSvc, cfg)
		authGroup := apiV1.Group("/auth")
		authGroup.Use(middleware.IPRateLimiter(redisClient, "auth", 20, time.Minute))
		authHandler.RegisterRoutes(authGroup)

		if svc.SSO != nil {
			ssoAuthHandler := v1.NewSSOAuthHandler(svc.SSO, svc.Auth, cfg)
			ssoAuthHandler.RegisterRoutes(authGroup.Group("/sso"))
		}

		v1.RegisterPublicConfigRoutes(apiV1.Group("/config"), svc.Billing)

		v1.RegisterRunnerReleaseRoutes(apiV1)

		// Public mTLS endpoints for Runner CLI registration.
		if svc.GRPCRunnerHandler != nil {
			v1.RegisterGRPCRunnerRoutes(apiV1, svc.GRPCRunnerHandler)
		}

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

		if svc.Invitation != nil {
			invitationHandler := v1.NewInvitationHandler(svc.Invitation, svc.Org, svc.User, svc.Billing)
			invitationHandler.RegisterRoutes(apiV1, middleware.AuthMiddleware(cfg.JWT.Secret))
		}

		protected := apiV1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			v1.RegisterUserRoutes(protected.Group("/users"), svc.User, svc.Org, svc.AgentSvc, svc.CredentialProfile, svc.UserConfig, svc.AgentPodSettings, svc.AgentPodAIProvider)

			v1.RegisterOrganizationRoutes(protected.Group("/orgs"), svc.Org, svc.User, redisClient)

			if svc.SupportTicket != nil {
				supportTicketHandler := v1.NewSupportTicketHandler(svc.SupportTicket)
				supportTicketHandler.RegisterRoutes(protected.Group("/support-tickets"))
			}

			orgScoped := protected.Group("/orgs/:slug")
			orgScoped.Use(middleware.TenantMiddleware(svc.Org))
			{
				v1.RegisterOrgScopedRoutes(orgScoped, svc)

				wsGroup := orgScoped.Group("/ws")
				{
					eventHandler := ws.NewEventsHandler(svc.Hub)
					wsGroup.GET("/events", eventHandler.HandleEvents)
				}
			}

		}
	}

	if svc.APIKeyAdapter != nil {
		extScoped := apiV1.Group("/ext/orgs/:slug")
		extScoped.Use(middleware.APIKeyAuthMiddleware(svc.APIKeyAdapter, svc.Org))
		{
			v1.RegisterExtRoutes(extScoped, svc)
		}
	}

	if cfg.Admin.IsEnabled() {
		dbWrapper := database.NewGormWrapper(db)
		adminSvc := adminservice.NewService(dbWrapper)
		admin.RegisterRoutes(r, cfg, dbWrapper, &admin.Services{
			Auth:              svc.Auth,
			Admin:             adminSvc,
			Billing:           svc.Billing,
			SSO:               svc.SSO,
			RelayManager:      svc.RelayManager,
			ExtensionRepo:     svc.ExtensionRepo,
			MarketplaceWorker: svc.MarketplaceWorker,
			SupportTicket:     svc.SupportTicket,
		})
	}

	if svc.RelayManager != nil {
		internal.RegisterRelayRoutes(r.Group("/api/internal/relays"), &internal.RelayRouterDeps{
			RelayManager:   svc.RelayManager,
			DNSService:     svc.RelayDNSService,
			ACMEManager:    svc.RelayACMEManager,
			GeoResolver:    svc.GeoResolver,
			InternalSecret: cfg.Server.InternalAPISecret,
		})
	}

	return r
}

package rest

import (
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/api/rest/v1"
	"github.com/anthropics/agentsmesh/backend/internal/api/rest/v1/webhooks"
	"github.com/anthropics/agentsmesh/backend/internal/api/rest/ws"
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NewRouter creates and configures the Gin router
func NewRouter(cfg *config.Config, svc *v1.Services, db *gorm.DB, logger *slog.Logger) *gin.Engine {
	if !cfg.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS configuration
	corsConfig := cors.Config{
		AllowOrigins:     cfg.Server.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Organization-Slug"},
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
	emailSvc := email.NewService(email.Config{
		Provider:    cfg.Email.Provider,
		ResendKey:   cfg.Email.ResendKey,
		FromAddress: cfg.Email.FromAddress,
		BaseURL:     cfg.Email.BaseURL,
	})

	// API v1
	apiV1 := r.Group("/api/v1")
	{
		// Public routes (no auth required)
		v1.RegisterAuthRoutes(apiV1.Group("/auth"), cfg, svc.Auth, svc.User, emailSvc)

		// Public config endpoints (deployment info for frontend)
		v1.RegisterPublicConfigRoutes(apiV1.Group("/config"), svc.Billing)

		// Runner registration (uses token-based auth, not JWT)
		RegisterRunnerAuthRoutes(apiV1.Group("/runners"), svc)

		// Webhook endpoints (no auth required, use token verification)
		// Use shared billing service to ensure mock provider sessions are shared
		webhookRouter := webhooks.NewWebhookRouterWithBillingSvc(db, cfg, logger, svc.Billing)
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
			v1.RegisterOrganizationRoutes(protected.Group("/orgs"), svc.Org)

			// Organization-scoped routes (require tenant context)
			// Path changed: /organizations/:slug → /orgs/:slug
			orgScoped := protected.Group("/orgs/:slug")
			orgScoped.Use(middleware.TenantMiddleware(svc.Org))
			{
				v1.RegisterOrgScopedRoutes(orgScoped, svc)

				// WebSocket endpoints (moved from /ws to /orgs/:slug/ws)
				wsGroup := orgScoped.Group("/ws")
				{
					terminalHandler := ws.NewTerminalHandler(svc.Hub, svc.Pod)
					if svc.TerminalRouter != nil {
						terminalHandler.SetTerminalRouter(svc.TerminalRouter)
					}
					wsGroup.GET("/terminal/:pod_key", terminalHandler.HandleTerminal)

					eventHandler := ws.NewEventsHandler(svc.Hub)
					wsGroup.GET("/events", eventHandler.HandleEvents)
				}
			}

			// Note: /org alias route removed - all org-scoped requests must use /orgs/:slug/*
		}

		// Note: Runner communication is now via gRPC/mTLS (see internal/api/grpc/)
		// The WebSocket endpoint /api/v1/orgs/:slug/ws/runners has been removed.

		// Pod-based API routes (for MCP tools) - moved under org-scoped
		// Path changed: /api/v1/pod → /api/v1/orgs/:slug/pod
		podOrgScoped := apiV1.Group("/orgs/:slug/pod")
		podOrgScoped.Use(middleware.PodAuthMiddleware(svc.Pod, svc.Org))
		{
			// Channel routes for MCP tools
			channelHandler := v1.NewChannelHandler(svc.Channel)
			podOrgScoped.GET("/channels", channelHandler.ListChannels)
			podOrgScoped.POST("/channels", channelHandler.CreateChannel)
			podOrgScoped.GET("/channels/:id", channelHandler.GetChannel)
			podOrgScoped.GET("/channels/:id/messages", channelHandler.ListMessages)
			podOrgScoped.POST("/channels/:id/messages", channelHandler.SendMessage)
			podOrgScoped.POST("/channels/:id/pods", channelHandler.JoinPod)
			podOrgScoped.GET("/channels/:id/document", channelHandler.GetDocument)
			podOrgScoped.PUT("/channels/:id/document", channelHandler.UpdateDocument)

			// Pod routes for MCP tools - using functional options
			var mcpPodOpts []v1.PodHandlerOption
			if svc.PodCoordinator != nil {
				mcpPodOpts = append(mcpPodOpts, v1.WithPodCoordinator(svc.PodCoordinator))
			}
			if svc.TerminalRouter != nil {
				mcpPodOpts = append(mcpPodOpts, v1.WithTerminalRouter(svc.TerminalRouter))
			}
			if svc.Repository != nil {
				mcpPodOpts = append(mcpPodOpts, v1.WithRepositoryService(svc.Repository))
			}
			if svc.Ticket != nil {
				mcpPodOpts = append(mcpPodOpts, v1.WithTicketService(svc.Ticket))
			}
			if svc.User != nil {
				mcpPodOpts = append(mcpPodOpts, v1.WithUserService(svc.User))
			}
			if svc.Billing != nil {
				mcpPodOpts = append(mcpPodOpts, v1.WithBillingService(svc.Billing))
			}
			podHandler := v1.NewPodHandler(svc.Pod, svc.Runner, svc.AgentType, svc.CredentialProfile, svc.UserConfig, mcpPodOpts...)
			podOrgScoped.GET("/pods", podHandler.ListPods)
			podOrgScoped.POST("/pods", podHandler.CreatePod)
			podOrgScoped.GET("/pods/:key/terminal/observe", podHandler.ObserveTerminal)
			podOrgScoped.POST("/pods/:key/terminal/input", podHandler.SendTerminalInput)

			// Ticket routes for MCP tools
			ticketHandler := v1.NewTicketHandler(svc.Ticket)
			podOrgScoped.GET("/tickets", ticketHandler.ListTickets)
			podOrgScoped.GET("/tickets/:identifier", ticketHandler.GetTicket)
			podOrgScoped.POST("/tickets", ticketHandler.CreateTicket)
			podOrgScoped.PUT("/tickets/:identifier", ticketHandler.UpdateTicket)

			// Binding routes for MCP tools
			bindingHandler := v1.NewBindingHandler(svc.Binding)
			podOrgScoped.POST("/bindings", bindingHandler.RequestBinding)
			podOrgScoped.GET("/bindings", bindingHandler.ListBindings)
			podOrgScoped.POST("/bindings/accept", bindingHandler.AcceptBinding)
			podOrgScoped.POST("/bindings/reject", bindingHandler.RejectBinding)
			podOrgScoped.POST("/bindings/unbind", bindingHandler.Unbind)
			podOrgScoped.GET("/bindings/pods", bindingHandler.GetBoundPods)

			// MCP Discovery routes (runners with nested agent info)
			mcpDiscoveryHandler := v1.NewMCPDiscoveryHandler(svc.Runner, svc.AgentType, svc.UserConfig)
			podOrgScoped.GET("/runners", mcpDiscoveryHandler.ListRunnersForMCP)

			// Repository routes for MCP tools (discovery)
			repositoryHandler := v1.NewRepositoryHandler(svc.Repository, svc.Billing)
			podOrgScoped.GET("/repositories", repositoryHandler.ListRepositories)
		}
	}

	return r
}

// RegisterRunnerAuthRoutes registers runner-specific authentication routes
// Note: Only registration is here. Heartbeat and WebSocket are now org-scoped at:
//   - POST /api/v1/orgs/:slug/runners/heartbeat
//   - GET  /api/v1/orgs/:slug/ws/runners
func RegisterRunnerAuthRoutes(rg *gin.RouterGroup, svc *v1.Services) {
	runnerHandler := v1.NewRunnerHandler(svc.Runner)
	runnerHandler.SetOrgService(svc.Org)
	rg.POST("/register", runnerHandler.RegisterRunner)
}

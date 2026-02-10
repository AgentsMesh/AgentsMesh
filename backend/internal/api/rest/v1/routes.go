package v1

import (
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/gin-gonic/gin"
)

// RegisterAllRoutes registers all API v1 routes with proper handlers
func RegisterAllRoutes(rg *gin.RouterGroup, cfg *config.Config, svc *Services) {
	// Auth routes (public)
	authHandler := NewAuthHandler(svc.Auth, svc.User, svc.Email, cfg)
	authHandler.RegisterRoutes(rg.Group("/auth"))

	// User routes (authenticated, but not org-scoped)
	RegisterUserRoutes(rg.Group("/users"), svc.User, svc.Org, svc.AgentType, svc.CredentialProfile, svc.UserConfig, svc.AgentPodSettings, svc.AgentPodAIProvider)

	// Organization routes (authenticated, some require org context)
	// Path changed: /organizations -> /orgs
	RegisterOrganizationRoutes(rg.Group("/orgs"), svc.Org)

	// Admin routes (require admin role)
	RegisterAdminRoutes(rg.Group("/admin"), svc)

	// License routes (for OnPremise deployments)
	RegisterLicenseHandlers(rg.Group("/license"), svc.License)

	// gRPC Runner routes (public, for Runner CLI registration)
	if svc.GRPCRunnerHandler != nil {
		RegisterGRPCRunnerRoutes(rg, svc.GRPCRunnerHandler)
	}
}

// RegisterAdminRoutes registers admin-only routes
func RegisterAdminRoutes(rg *gin.RouterGroup, svc *Services) {
	// Promo Codes admin
	if svc.PromoCode != nil {
		RegisterAdminPromoCodeRoutes(rg.Group("/promo-codes"), svc.PromoCode)
	}
}

// RegisterOrgScopedRoutes registers organization-scoped routes (require tenant context)
func RegisterOrgScopedRoutes(rg *gin.RouterGroup, svc *Services) {
	slog.Info("RegisterOrgScopedRoutes called", "file_svc_nil", svc.File == nil)

	// Register agent routes
	registerAgentRoutes(rg, svc)

	// Register repository routes
	registerRepositoryRoutes(rg, svc)

	// Register runner routes
	registerRunnerRoutes(rg, svc)

	// Register pod routes
	registerPodRoutes(rg, svc)

	// Register channel routes
	registerChannelRoutes(rg, svc)

	// Register ticket routes
	registerTicketRoutes(rg, svc)

	// Register billing and other routes
	registerBillingRoutes(rg, svc)

	// Register binding routes
	registerBindingRoutes(rg, svc)

	// Register message routes
	registerMessageRoutes(rg, svc)

	// Register invitation routes
	registerInvitationRoutes(rg, svc)

	// Register file routes
	registerFileRoutes(rg, svc)
}

func registerAgentRoutes(rg *gin.RouterGroup, svc *Services) {
	agentHandler := NewAgentHandler(svc.AgentType, svc.CredentialProfile, svc.UserConfig)
	agents := rg.Group("/agents")
	{
		agents.GET("/types", agentHandler.ListAgentTypes)
		agents.GET("/types/:agent_type_id", agentHandler.GetAgentType)
		agents.POST("/custom", agentHandler.CreateCustomAgent)
		agents.PUT("/custom/:id", agentHandler.UpdateCustomAgent)
		agents.DELETE("/custom/:id", agentHandler.DeleteCustomAgent)
		agents.GET("/:agent_type_id/config-schema", agentHandler.GetAgentTypeConfigSchema)
	}
}

func registerRepositoryRoutes(rg *gin.RouterGroup, svc *Services) {
	repositoryHandler := NewRepositoryHandler(svc.Repository, svc.Billing)
	repositories := rg.Group("/repositories")
	{
		repositories.GET("", repositoryHandler.ListRepositories)
		repositories.POST("", repositoryHandler.CreateRepository)
		repositories.GET("/:id", repositoryHandler.GetRepository)
		repositories.PUT("/:id", repositoryHandler.UpdateRepository)
		repositories.DELETE("/:id", repositoryHandler.DeleteRepository)
		repositories.GET("/:id/branches", repositoryHandler.ListBranches)
		repositories.POST("/:id/sync-branches", repositoryHandler.SyncBranches)

		// Webhook management routes
		repositories.POST("/:id/webhook", repositoryHandler.RegisterRepositoryWebhook)
		repositories.DELETE("/:id/webhook", repositoryHandler.DeleteRepositoryWebhook)
		repositories.GET("/:id/webhook/status", repositoryHandler.GetRepositoryWebhookStatus)
		repositories.GET("/:id/webhook/secret", repositoryHandler.GetRepositoryWebhookSecret)
		repositories.POST("/:id/webhook/configured", repositoryHandler.MarkRepositoryWebhookConfigured)

		// Merge requests route
		repositories.GET("/:id/merge-requests", repositoryHandler.ListRepositoryMergeRequests)
	}
}

func registerRunnerRoutes(rg *gin.RouterGroup, svc *Services) {
	var runnerOpts []RunnerHandlerOption
	if svc.Pod != nil {
		runnerOpts = append(runnerOpts, WithPodServiceForRunner(svc.Pod))
	}
	if svc.SandboxQueryService != nil {
		runnerOpts = append(runnerOpts, WithSandboxQueryService(svc.SandboxQueryService))
	}
	if svc.SandboxQuerySender != nil {
		runnerOpts = append(runnerOpts, WithSandboxQuerySender(svc.SandboxQuerySender))
	}
	if svc.PodCoordinator != nil {
		runnerOpts = append(runnerOpts, WithPodCoordinatorForRunner(svc.PodCoordinator))
	}
	runnerHandler := NewRunnerHandler(svc.Runner, runnerOpts...)
	runners := rg.Group("/runners")
	{
		runners.GET("", runnerHandler.ListRunners)
		runners.GET("/available", runnerHandler.ListAvailableRunners)
		runners.GET("/:id", runnerHandler.GetRunner)
		runners.PUT("/:id", runnerHandler.UpdateRunner)
		runners.DELETE("/:id", runnerHandler.DeleteRunner)
		runners.GET("/:id/pods", runnerHandler.ListRunnerPods)
		runners.POST("/:id/sandboxes/query", runnerHandler.QuerySandboxes)

		if svc.GRPCRunnerHandler != nil {
			RegisterOrgGRPCRunnerRoutes(runners, svc.GRPCRunnerHandler)
		}
	}
}

func registerPodRoutes(rg *gin.RouterGroup, svc *Services) {
	var podOpts []PodHandlerOption
	if svc.PodCoordinator != nil {
		podOpts = append(podOpts, WithPodCoordinator(svc.PodCoordinator))
	}
	if svc.TerminalRouter != nil {
		podOpts = append(podOpts, WithTerminalRouter(svc.TerminalRouter))
	}
	podHandler := NewPodHandler(svc.Pod, svc.Runner, svc.PodOrchestrator, podOpts...)
	pods := rg.Group("/pods")
	{
		pods.GET("", podHandler.ListPods)
		pods.POST("", podHandler.CreatePod)
		pods.GET("/:key", podHandler.GetPod)
		pods.POST("/:key/terminate", podHandler.TerminatePod)
		pods.GET("/:key/connect", podHandler.GetConnectionInfo)
		pods.POST("/:key/send-prompt", podHandler.SendPrompt)
		pods.GET("/:key/terminal/observe", podHandler.ObserveTerminal)
		pods.POST("/:key/terminal/input", podHandler.SendTerminalInput)
		pods.POST("/:key/terminal/resize", podHandler.ResizeTerminal)
	}

	// Terminal Relay connection endpoint
	if svc.RelayManager != nil && svc.RelayTokenGenerator != nil {
		var commandSender runner.RunnerCommandSender
		if svc.PodCoordinator != nil {
			commandSender = svc.PodCoordinator.GetCommandSender()
		}
		RegisterTerminalConnectRoutes(rg, svc.Pod, svc.RelayManager, svc.RelayTokenGenerator, commandSender)
	}

	// AutopilotControllers
	var autopilotOpts []AutopilotControllerHandlerOption
	if svc.Pod != nil {
		autopilotOpts = append(autopilotOpts, WithPodServiceForAutopilot(svc.Pod))
	}
	if svc.Autopilot != nil {
		autopilotOpts = append(autopilotOpts, WithAutopilotControllerService(svc.Autopilot))
	}
	if svc.PodCoordinator != nil {
		autopilotOpts = append(autopilotOpts, WithAutopilotCommandSender(svc.PodCoordinator))
	}
	autopilotHandler := NewAutopilotControllerHandler(autopilotOpts...)
	RegisterAutopilotControllerRoutes(rg, autopilotHandler)
}

func registerChannelRoutes(rg *gin.RouterGroup, svc *Services) {
	channelHandler := NewChannelHandler(svc.Channel)
	channels := rg.Group("/channels")
	{
		channels.GET("", channelHandler.ListChannels)
		channels.POST("", channelHandler.CreateChannel)
		channels.GET("/:id", channelHandler.GetChannel)
		channels.PUT("/:id", channelHandler.UpdateChannel)
		channels.POST("/:id/archive", channelHandler.ArchiveChannel)
		channels.POST("/:id/unarchive", channelHandler.UnarchiveChannel)
		channels.GET("/:id/messages", channelHandler.ListMessages)
		channels.POST("/:id/messages", channelHandler.SendMessage)
		channels.GET("/:id/document", channelHandler.GetDocument)
		channels.PUT("/:id/document", channelHandler.UpdateDocument)
		channels.GET("/:id/pods", channelHandler.ListChannelPods)
		channels.POST("/:id/pods", channelHandler.JoinPod)
		channels.DELETE("/:id/pods/:pod_key", channelHandler.LeavePod)
	}
}

func registerTicketRoutes(rg *gin.RouterGroup, svc *Services) {
	ticketHandler := NewTicketHandler(svc.Ticket)
	meshHandler := NewMeshHandler(svc.Mesh, svc.Ticket)
	tickets := rg.Group("/tickets")
	{
		tickets.GET("", ticketHandler.ListTickets)
		tickets.POST("", ticketHandler.CreateTicket)
		tickets.GET("/active", ticketHandler.GetActiveTickets)
		tickets.GET("/board", ticketHandler.GetBoard)
		tickets.POST("/batch-pods", meshHandler.BatchGetTicketPods)
		tickets.GET("/:identifier", ticketHandler.GetTicket)
		tickets.PUT("/:identifier", ticketHandler.UpdateTicket)
		tickets.DELETE("/:identifier", ticketHandler.DeleteTicket)
		tickets.PATCH("/:identifier/status", ticketHandler.UpdateTicketStatus)
		tickets.POST("/:identifier/assignees", ticketHandler.AddAssignee)
		tickets.DELETE("/:identifier/assignees/:user_id", ticketHandler.RemoveAssignee)
		tickets.POST("/:identifier/labels", ticketHandler.AddLabel)
		tickets.DELETE("/:identifier/labels/:label_id", ticketHandler.RemoveLabel)
		tickets.GET("/:identifier/merge-requests", ticketHandler.ListMergeRequests)
		tickets.GET("/:identifier/sub-tickets", ticketHandler.GetSubTickets)
		tickets.GET("/:identifier/relations", ticketHandler.ListRelations)
		tickets.POST("/:identifier/relations", ticketHandler.CreateRelation)
		tickets.DELETE("/:identifier/relations/:relation_id", ticketHandler.DeleteRelation)
		tickets.GET("/:identifier/commits", ticketHandler.ListCommits)
		tickets.POST("/:identifier/commits", ticketHandler.LinkCommit)
		tickets.DELETE("/:identifier/commits/:commit_id", ticketHandler.UnlinkCommit)
		tickets.GET("/:identifier/pods", meshHandler.GetTicketPods)
		tickets.POST("/:identifier/pods", meshHandler.CreatePodForTicket)
	}

	labels := rg.Group("/labels")
	{
		labels.GET("", ticketHandler.ListLabels)
		labels.POST("", ticketHandler.CreateLabel)
		labels.PUT("/:id", ticketHandler.UpdateLabel)
		labels.DELETE("/:id", ticketHandler.DeleteLabel)
	}

	meshGroup := rg.Group("/mesh")
	{
		meshGroup.GET("/topology", meshHandler.GetTopology)
	}
}

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

func registerMessageRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.Message != nil {
		messageHandler := NewMessageHandler(svc.Message)
		messages := rg.Group("/messages")
		{
			messages.POST("", messageHandler.SendMessage)
			messages.GET("", messageHandler.GetMessages)
			messages.GET("/unread-count", messageHandler.GetUnreadCount)
			messages.GET("/sent", messageHandler.GetSentMessages)
			messages.POST("/mark-read", messageHandler.MarkRead)
			messages.POST("/mark-all-read", messageHandler.MarkAllRead)
			messages.GET("/conversation/:correlation_id", messageHandler.GetConversation)
			messages.GET("/dlq", messageHandler.GetDeadLetters)
			messages.POST("/dlq/:id/replay", messageHandler.ReplayDeadLetter)
			messages.GET("/:id", messageHandler.GetMessage)
		}
	}
}

func registerInvitationRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.Invitation != nil {
		invitationHandler := NewInvitationHandler(svc.Invitation, svc.Org, svc.User, svc.Billing)
		invitationHandler.RegisterOrgRoutes(rg)
	}
}

func registerFileRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.File != nil {
		slog.Info("Registering file upload routes", "service", "file")
		fileHandler := NewFileHandler(svc.File)
		files := rg.Group("/files")
		{
			files.POST("/upload", fileHandler.UploadFile)
			files.DELETE("/:id", fileHandler.DeleteFile)
		}
	} else {
		slog.Warn("File service is nil, file upload routes not registered")
	}
}

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
	RegisterOrganizationRoutes(rg.Group("/orgs"), svc.Org, svc.User)

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

	// Register API key management routes (owner/admin only)
	registerAPIKeyManagementRoutes(rg, svc)

	// Register extension routes (Skills marketplace, MCP servers)
	registerExtensionRoutes(rg, svc)

	// Register loop routes
	registerLoopRoutes(rg, svc)

	// Register notification preference routes
	registerNotificationRoutes(rg, svc)

	// Register token usage routes
	registerTokenUsageRoutes(rg, svc)
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
	if svc.VersionChecker != nil {
		runnerOpts = append(runnerOpts, WithVersionChecker(svc.VersionChecker))
	}
	if svc.UpgradeCommandSender != nil {
		runnerOpts = append(runnerOpts, WithUpgradeCommandSender(svc.UpgradeCommandSender))
	}
	if svc.LogUploadSender != nil {
		runnerOpts = append(runnerOpts, WithLogUploadSender(svc.LogUploadSender))
	}
	if svc.LogUploadService != nil {
		runnerOpts = append(runnerOpts, WithLogUploadService(svc.LogUploadService))
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
		runners.POST("/:id/upgrade", runnerHandler.UpgradeRunner)
		runners.POST("/:id/logs/upload", runnerHandler.RequestLogUpload)
		runners.GET("/:id/logs", runnerHandler.ListRunnerLogs)

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
		RegisterTerminalConnectRoutes(rg, svc.Pod, svc.RelayManager, svc.RelayTokenGenerator, commandSender, svc.GeoResolver)
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
	channelHandler := NewChannelHandler(svc.Channel, svc.Ticket)
	channels := rg.Group("/channels")
	{
		channels.GET("", channelHandler.ListChannels)
		channels.POST("", channelHandler.CreateChannel)
		channels.GET("/unread", channelHandler.GetUnreadCounts)
		channels.GET("/:id", channelHandler.GetChannel)
		channels.PUT("/:id", channelHandler.UpdateChannel)
		channels.POST("/:id/archive", channelHandler.ArchiveChannel)
		channels.POST("/:id/unarchive", channelHandler.UnarchiveChannel)
		channels.GET("/:id/messages", channelHandler.ListMessages)
		channels.POST("/:id/messages", channelHandler.SendMessage)
		channels.PUT("/:id/messages/:msg_id", channelHandler.EditMessage)
		channels.DELETE("/:id/messages/:msg_id", channelHandler.DeleteMessage)
		channels.POST("/:id/read", channelHandler.MarkRead)
		channels.POST("/:id/mute", channelHandler.MuteChannel)
		channels.GET("/:id/members", channelHandler.ListMembers)
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
		tickets.GET("/:ticket_slug", ticketHandler.GetTicket)
		tickets.PUT("/:ticket_slug", ticketHandler.UpdateTicket)
		tickets.DELETE("/:ticket_slug", ticketHandler.DeleteTicket)
		tickets.PATCH("/:ticket_slug/status", ticketHandler.UpdateTicketStatus)
		tickets.POST("/:ticket_slug/assignees", ticketHandler.AddAssignee)
		tickets.DELETE("/:ticket_slug/assignees/:user_id", ticketHandler.RemoveAssignee)
		tickets.POST("/:ticket_slug/labels", ticketHandler.AddLabel)
		tickets.DELETE("/:ticket_slug/labels/:label_id", ticketHandler.RemoveLabel)
		tickets.GET("/:ticket_slug/merge-requests", ticketHandler.ListMergeRequests)
		tickets.GET("/:ticket_slug/sub-tickets", ticketHandler.GetSubTickets)
		tickets.GET("/:ticket_slug/relations", ticketHandler.ListRelations)
		tickets.POST("/:ticket_slug/relations", ticketHandler.CreateRelation)
		tickets.DELETE("/:ticket_slug/relations/:relation_id", ticketHandler.DeleteRelation)
		tickets.GET("/:ticket_slug/commits", ticketHandler.ListCommits)
		tickets.POST("/:ticket_slug/commits", ticketHandler.LinkCommit)
		tickets.DELETE("/:ticket_slug/commits/:commit_id", ticketHandler.UnlinkCommit)
		tickets.GET("/:ticket_slug/comments", ticketHandler.ListComments)
		tickets.POST("/:ticket_slug/comments", ticketHandler.CreateComment)
		tickets.PUT("/:ticket_slug/comments/:id", ticketHandler.UpdateComment)
		tickets.DELETE("/:ticket_slug/comments/:id", ticketHandler.DeleteComment)
		tickets.GET("/:ticket_slug/pods", meshHandler.GetTicketPods)
		tickets.POST("/:ticket_slug/pods", meshHandler.CreatePodForTicket)
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

func registerExtensionRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.Extension == nil {
		slog.Warn("Extension services not configured, extension routes not registered")
		return
	}

	handler := NewExtensionHandler(svc.Extension)

	// Skill Registries (org admin only)
	skillRegistries := rg.Group("/skill-registries")
	{
		skillRegistries.GET("", handler.ListSkillRegistries)
		skillRegistries.POST("", handler.CreateSkillRegistry)
		skillRegistries.POST("/:id/sync", handler.SyncSkillRegistry)
		skillRegistries.DELETE("/:id", handler.DeleteSkillRegistry)
		skillRegistries.PUT("/:id/toggle", handler.TogglePlatformRegistry)
	}

	// Skill Registry Overrides
	rg.GET("/skill-registry-overrides", handler.ListSkillRegistryOverrides)

	// Marketplace
	market := rg.Group("/market")
	{
		market.GET("/skills", handler.ListMarketSkills)
		market.GET("/mcp-servers", handler.ListMarketMcpServers)
	}

	// Repository-scoped skills
	repoSkills := rg.Group("/repositories/:id/skills")
	{
		repoSkills.GET("", handler.ListRepoSkills)
		repoSkills.POST("/install-from-market", handler.InstallSkillFromMarket)
		repoSkills.POST("/install-from-github", handler.InstallSkillFromGitHub)
		repoSkills.POST("/install-from-upload", handler.InstallSkillFromUpload)
		repoSkills.PUT("/:installId", handler.UpdateSkill)
		repoSkills.DELETE("/:installId", handler.UninstallSkill)
	}

	// Repository-scoped MCP servers
	repoMcp := rg.Group("/repositories/:id/mcp-servers")
	{
		repoMcp.GET("", handler.ListRepoMcpServers)
		repoMcp.POST("/install-from-market", handler.InstallMcpFromMarket)
		repoMcp.POST("/install-custom", handler.InstallCustomMcpServer)
		repoMcp.PUT("/:installId", handler.UpdateMcpServer)
		repoMcp.DELETE("/:installId", handler.UninstallMcpServer)
	}

	slog.Info("Extension routes registered")
}

func registerLoopRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.Loop == nil {
		return
	}
	loopHandler := NewLoopHandler(svc.Loop, svc.LoopRun, svc.LoopOrchestrator, svc.PodCoordinator)
	loops := rg.Group("/loops")
	{
		loops.GET("", loopHandler.ListLoops)
		loops.POST("", loopHandler.CreateLoop)
		loops.GET("/:loop_slug", loopHandler.GetLoop)
		loops.PUT("/:loop_slug", loopHandler.UpdateLoop)
		loops.DELETE("/:loop_slug", loopHandler.DeleteLoop)
		loops.POST("/:loop_slug/enable", loopHandler.EnableLoop)
		loops.POST("/:loop_slug/disable", loopHandler.DisableLoop)
		loops.POST("/:loop_slug/trigger", loopHandler.TriggerLoop)
		loops.GET("/:loop_slug/runs", loopHandler.ListRuns)
		loops.GET("/:loop_slug/runs/:run_id", loopHandler.GetRun)
		loops.POST("/:loop_slug/runs/:run_id/cancel", loopHandler.CancelRun)
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

package v1

import (
	"github.com/anthropics/agentmesh/backend/internal/config"
	"github.com/anthropics/agentmesh/backend/internal/infra/email"
	"github.com/anthropics/agentmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentmesh/backend/internal/service/agent"
	"github.com/anthropics/agentmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentmesh/backend/internal/service/auth"
	"github.com/anthropics/agentmesh/backend/internal/service/billing"
	"github.com/anthropics/agentmesh/backend/internal/service/binding"
	"github.com/anthropics/agentmesh/backend/internal/service/channel"
	"github.com/anthropics/agentmesh/backend/internal/service/devmesh"
	fileservice "github.com/anthropics/agentmesh/backend/internal/service/file"
	"github.com/anthropics/agentmesh/backend/internal/service/invitation"
	"github.com/anthropics/agentmesh/backend/internal/service/organization"
	"github.com/anthropics/agentmesh/backend/internal/service/promocode"
	"github.com/anthropics/agentmesh/backend/internal/service/repository"
	"github.com/anthropics/agentmesh/backend/internal/service/runner"
	"github.com/anthropics/agentmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// MessageService is a type alias for agent.MessageService
type MessageService = agent.MessageService

// Services holds all service dependencies for API handlers
type Services struct {
	Auth              *auth.Service
	User              *user.Service
	Org               *organization.Service
	// Agent services (split by responsibility)
	AgentType         *agent.AgentTypeService
	CredentialProfile *agent.CredentialProfileService
	UserConfig        *agent.UserConfigService
	Repository        *repository.Service
	Runner            *runner.Service
	RunnerConnMgr     *runner.ConnectionManager // Runner WebSocket connection manager
	PodCoordinator    *runner.PodCoordinator    // Pod lifecycle coordinator
	TerminalRouter    *runner.TerminalRouter    // Terminal data router
	Pod               *agentpod.PodService
	Channel           *channel.Service
	Binding           *binding.Service
	Ticket            *ticket.Service
	DevMesh           *devmesh.Service
	AgentPodSettings  *agentpod.SettingsService   // AgentPod user settings
	AgentPodAIProvider *agentpod.AIProviderService // AgentPod AI provider management
	Billing           *billing.Service
	Message           *MessageService    // Agent-to-agent messaging
	Hub               *websocket.Hub     // WebSocket hub for real-time communication
	EventBus          *eventbus.EventBus // Event bus for real-time events
	Email             email.Service        // Email service
	Invitation        *invitation.Service  // Organization invitations
	File              *fileservice.Service // File storage service
	PromoCode         *promocode.Service   // Promo code management
	// NOTE: GitProvider and SSHKey services have been removed (moved to user-level settings)
}

// RegisterAllRoutes registers all API v1 routes with proper handlers
func RegisterAllRoutes(rg *gin.RouterGroup, cfg *config.Config, svc *Services) {
	// Auth routes (public)
	RegisterAuthRoutes(rg.Group("/auth"), cfg, svc.Auth, svc.User, svc.Email)

	// User routes (authenticated, but not org-scoped)
	RegisterUserRoutes(rg.Group("/users"), svc.User, svc.Org, svc.AgentType, svc.CredentialProfile, svc.UserConfig, svc.AgentPodSettings, svc.AgentPodAIProvider)

	// Organization routes (authenticated, some require org context)
	// Path changed: /organizations → /orgs
	RegisterOrganizationRoutes(rg.Group("/orgs"), svc.Org)

	// Admin routes (require admin role)
	RegisterAdminRoutes(rg.Group("/admin"), svc)
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
	// Agents
	agentHandler := NewAgentHandler(svc.AgentType, svc.CredentialProfile, svc.UserConfig)
	agents := rg.Group("/agents")
	{
		agents.GET("/types", agentHandler.ListAgentTypes)
		agents.GET("/types/:agent_type_id", agentHandler.GetAgentType)
		agents.POST("/custom", agentHandler.CreateCustomAgent)
		agents.PUT("/custom/:id", agentHandler.UpdateCustomAgent)
		agents.DELETE("/custom/:id", agentHandler.DeleteCustomAgent)
		// Config schema (for frontend dynamic form rendering)
		agents.GET("/:agent_type_id/config-schema", agentHandler.GetAgentTypeConfigSchema)
	}

	// NOTE: Git Providers and SSH Keys have been moved to user-level settings
	// Use /api/v1/user/repository-providers and /api/v1/user/git-credentials instead

	// Repositories
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
		repositories.POST("/:id/webhook", repositoryHandler.SetupWebhook)
	}

	// Runners
	runnerHandler := NewRunnerHandler(svc.Runner)
	runners := rg.Group("/runners")
	{
		runners.GET("", runnerHandler.ListRunners)
		runners.GET("/available", runnerHandler.ListAvailableRunners)
		runners.GET("/tokens", runnerHandler.ListRegistrationTokens)
		runners.POST("/tokens", runnerHandler.CreateRegistrationToken)
		runners.DELETE("/tokens/:id", runnerHandler.RevokeRegistrationToken)
		runners.GET("/:id", runnerHandler.GetRunner)
		runners.PUT("/:id", runnerHandler.UpdateRunner)
		runners.DELETE("/:id", runnerHandler.DeleteRunner)
		runners.POST("/:id/regenerate-token", runnerHandler.RegenerateAuthToken)
	}

	// Pods - using functional options for cleaner dependency injection
	var podOpts []PodHandlerOption
	if svc.PodCoordinator != nil {
		podOpts = append(podOpts, WithPodCoordinator(svc.PodCoordinator))
	}
	if svc.TerminalRouter != nil {
		podOpts = append(podOpts, WithTerminalRouter(svc.TerminalRouter))
	}
	if svc.Repository != nil {
		podOpts = append(podOpts, WithRepositoryService(svc.Repository))
	}
	if svc.Ticket != nil {
		podOpts = append(podOpts, WithTicketService(svc.Ticket))
	}
	if svc.User != nil {
		podOpts = append(podOpts, WithUserService(svc.User))
	}
	if svc.Billing != nil {
		podOpts = append(podOpts, WithBillingService(svc.Billing))
	}
	podHandler := NewPodHandler(svc.Pod, svc.Runner, svc.AgentType, svc.CredentialProfile, svc.UserConfig, podOpts...)
	pods := rg.Group("/pods")
	{
		pods.GET("", podHandler.ListPods)
		pods.POST("", podHandler.CreatePod)
		pods.GET("/:key", podHandler.GetPod)
		pods.POST("/:key/terminate", podHandler.TerminatePod)
		pods.GET("/:key/connect", podHandler.GetConnectionInfo)
		pods.POST("/:key/send-prompt", podHandler.SendPrompt)
		// Terminal control endpoints
		pods.GET("/:key/terminal/observe", podHandler.ObserveTerminal)
		pods.POST("/:key/terminal/input", podHandler.SendTerminalInput)
		pods.POST("/:key/terminal/resize", podHandler.ResizeTerminal)
	}

	// Channels
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
		channels.GET("/:id/pods", channelHandler.ListChannelPods)
		channels.POST("/:id/pods", channelHandler.JoinPod)
		channels.DELETE("/:id/pods/:pod_key", channelHandler.LeavePod)
	}

	// Tickets
	// Note: Event publishing is handled in ticket.Service layer (Information Expert principle)
	ticketHandler := NewTicketHandler(svc.Ticket)
	devmeshHandler := NewDevMeshHandler(svc.DevMesh, svc.Ticket)
	tickets := rg.Group("/tickets")
	{
		tickets.GET("", ticketHandler.ListTickets)
		tickets.POST("", ticketHandler.CreateTicket)
		tickets.GET("/active", ticketHandler.GetActiveTickets)         // New: active tickets
		tickets.GET("/board", ticketHandler.GetBoard)                  // New: kanban board
		tickets.POST("/batch-pods", devmeshHandler.BatchGetTicketPods) // Batch get pods for tickets
		tickets.GET("/:identifier", ticketHandler.GetTicket)
		tickets.PUT("/:identifier", ticketHandler.UpdateTicket)
		tickets.DELETE("/:identifier", ticketHandler.DeleteTicket)
		tickets.PATCH("/:identifier/status", ticketHandler.UpdateTicketStatus)
		tickets.POST("/:identifier/assignees", ticketHandler.AddAssignee)
		tickets.DELETE("/:identifier/assignees/:user_id", ticketHandler.RemoveAssignee)
		tickets.POST("/:identifier/labels", ticketHandler.AddLabel)
		tickets.DELETE("/:identifier/labels/:label_id", ticketHandler.RemoveLabel)
		tickets.GET("/:identifier/merge-requests", ticketHandler.ListMergeRequests)
		tickets.GET("/:identifier/sub-tickets", ticketHandler.GetSubTickets)   // New: sub-tickets
		tickets.GET("/:identifier/relations", ticketHandler.ListRelations)     // New: relations
		tickets.POST("/:identifier/relations", ticketHandler.CreateRelation)   // New: create relation
		tickets.DELETE("/:identifier/relations/:relation_id", ticketHandler.DeleteRelation) // New: delete relation
		tickets.GET("/:identifier/commits", ticketHandler.ListCommits)         // New: commits
		tickets.POST("/:identifier/commits", ticketHandler.LinkCommit)         // New: link commit
		tickets.DELETE("/:identifier/commits/:commit_id", ticketHandler.UnlinkCommit) // New: unlink commit
		tickets.GET("/:identifier/pods", devmeshHandler.GetTicketPods) // Get pods for ticket
		tickets.POST("/:identifier/pods", devmeshHandler.CreatePodForTicket) // Create pod for ticket
	}

	// Labels (organization-level)
	labels := rg.Group("/labels")
	{
		labels.GET("", ticketHandler.ListLabels)
		labels.POST("", ticketHandler.CreateLabel)
		labels.PUT("/:id", ticketHandler.UpdateLabel)
		labels.DELETE("/:id", ticketHandler.DeleteLabel)
	}

	// Billing
	RegisterBillingHandlers(rg.Group("/billing"), svc.Billing)

	// Promo Codes (under billing)
	if svc.PromoCode != nil {
		RegisterPromoCodeRoutes(rg.Group("/billing/promo-codes"), svc.PromoCode)
	}

	// DevMesh (topology visualization)
	devmesh := rg.Group("/devmesh")
	{
		devmesh.GET("/topology", devmeshHandler.GetTopology)
	}

	// Bindings (pod collaboration)
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

	// Messages (agent-to-agent communication)
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

	// Invitations (organization-scoped)
	if svc.Invitation != nil {
		invitationHandler := NewInvitationHandler(svc.Invitation, svc.Org, svc.User, svc.Billing)
		invitationHandler.RegisterOrgRoutes(rg)
	}

	// Files (storage)
	if svc.File != nil {
		fileHandler := NewFileHandler(svc.File)
		files := rg.Group("/files")
		{
			files.POST("/upload", fileHandler.UploadFile)
			files.DELETE("/:id", fileHandler.DeleteFile)
		}
	}
}


// RegisterUserRoutes registers user routes
func RegisterUserRoutes(rg *gin.RouterGroup, userSvc *user.Service, orgSvc *organization.Service, agentTypeSvc *agent.AgentTypeService, credentialSvc *agent.CredentialProfileService, userConfigSvc *agent.UserConfigService, agentpodSettingsSvc *agentpod.SettingsService, agentpodAIProviderSvc *agentpod.AIProviderService) {
	userHandler := NewUserHandler(userSvc, orgSvc)
	agentHandler := NewAgentHandler(agentTypeSvc, credentialSvc, userConfigSvc)

	// Profile routes
	rg.GET("/me", userHandler.GetCurrentUser)
	rg.PUT("/me", userHandler.UpdateCurrentUser)
	rg.POST("/me/password", userHandler.ChangePassword)
	rg.GET("/me/organizations", userHandler.ListUserOrganizations)
	rg.GET("/me/identities", userHandler.ListIdentities)
	rg.DELETE("/me/identities/:provider", userHandler.DeleteIdentity)

	// User agent credentials (not org-scoped)
	rg.GET("/me/agents/credentials", agentHandler.GetUserCredentials)
	rg.PUT("/me/agents/credentials/:agent_type_id", agentHandler.SetUserCredentials)
	rg.DELETE("/me/agents/credentials/:agent_type_id", agentHandler.DeleteUserCredentials)

	// User agent configs (personal runtime configuration)
	rg.GET("/me/agent-configs", agentHandler.ListUserAgentConfigs)
	rg.GET("/me/agent-configs/:agent_type_id", agentHandler.GetUserAgentConfig)
	rg.PUT("/me/agent-configs/:agent_type_id", agentHandler.SetUserAgentConfig)
	rg.DELETE("/me/agent-configs/:agent_type_id", agentHandler.DeleteUserAgentConfig)

	// AgentPod settings routes
	if agentpodSettingsSvc != nil && agentpodAIProviderSvc != nil {
		agentpodHandler := NewAgentPodHandler(agentpodSettingsSvc, agentpodAIProviderSvc)
		agentpodGroup := rg.Group("/me/agentpod")
		{
			// Settings
			agentpodGroup.GET("/settings", agentpodHandler.GetSettings)
			agentpodGroup.PUT("/settings", agentpodHandler.UpdateSettings)

			// AI Providers
			providers := agentpodGroup.Group("/providers")
			{
				providers.GET("", agentpodHandler.ListProviders)
				providers.POST("", agentpodHandler.CreateProvider)
				providers.PUT("/:id", agentpodHandler.UpdateProvider)
				providers.DELETE("/:id", agentpodHandler.DeleteProvider)
				providers.POST("/:id/default", agentpodHandler.SetDefaultProvider)
			}
		}
	}

	// User Repository Providers (for importing repositories)
	repositoryProviderHandler := NewUserRepositoryProviderHandler(userSvc)
	repositoryProviderHandler.RegisterRoutes(rg)

	// User Git Credentials (for Git operations)
	gitCredentialHandler := NewUserGitCredentialHandler(userSvc)
	gitCredentialHandler.RegisterRoutes(rg)

	// User Agent Credential Profiles (for agent API credentials)
	agentCredentialHandler := NewUserAgentCredentialHandler(credentialSvc)
	agentCredentialHandler.RegisterRoutes(rg)

	// User Git Connections (legacy, for backward compatibility)
	gitConnectionHandler := NewUserGitConnectionHandler(userSvc)
	gitConnectionHandler.RegisterRoutes(rg)

	// User search
	rg.GET("/search", userHandler.SearchUsers)
}

// RegisterOrganizationRoutes registers organization routes
func RegisterOrganizationRoutes(rg *gin.RouterGroup, orgSvc *organization.Service) {
	handler := NewOrganizationHandler(orgSvc)

	// Organization CRUD
	rg.GET("", handler.ListOrganizations)
	rg.POST("", handler.CreateOrganization)
	rg.GET("/:slug", handler.GetOrganization)
	rg.PUT("/:slug", handler.UpdateOrganization)
	rg.DELETE("/:slug", handler.DeleteOrganization)

	// Member management
	rg.GET("/:slug/members", handler.ListMembers)
	rg.POST("/:slug/members", handler.InviteMember)
	rg.PUT("/:slug/members/:user_id", handler.UpdateMemberRole)
	rg.DELETE("/:slug/members/:user_id", handler.RemoveMember)
}

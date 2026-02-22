package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// registerAPIKeyManagementRoutes registers API key CRUD routes (JWT auth, owner/admin only)
func registerAPIKeyManagementRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.APIKey == nil {
		return
	}

	apiKeyHandler := NewAPIKeyHandler(svc.APIKey)
	apiKeys := rg.Group("/api-keys")
	apiKeys.Use(middleware.RequireAdmin())
	{
		apiKeys.POST("", apiKeyHandler.CreateAPIKey)
		apiKeys.GET("", apiKeyHandler.ListAPIKeys)
		apiKeys.GET("/:id", apiKeyHandler.GetAPIKey)
		apiKeys.PUT("/:id", apiKeyHandler.UpdateAPIKey)
		apiKeys.DELETE("/:id", apiKeyHandler.DeleteAPIKey)
		apiKeys.POST("/:id/revoke", apiKeyHandler.RevokeAPIKey)
	}
}

// RegisterExtRoutes registers third-party API key-authenticated routes.
// These routes reuse existing handler instances with scope-based access control.
func RegisterExtRoutes(rg *gin.RouterGroup, svc *Services) {
	// Pod routes
	var podOpts []PodHandlerOption
	if svc.PodCoordinator != nil {
		podOpts = append(podOpts, WithPodCoordinator(svc.PodCoordinator))
	}
	if svc.TerminalRouter != nil {
		podOpts = append(podOpts, WithTerminalRouter(svc.TerminalRouter))
	}
	podHandler := NewPodHandler(svc.Pod, svc.Runner, svc.PodOrchestrator, podOpts...)

	podsRead := rg.Group("/pods")
	podsRead.Use(middleware.RequireScope("pods:read", "pods:write"))
	{
		podsRead.GET("", podHandler.ListPods)
		podsRead.GET("/:key", podHandler.GetPod)
	}
	podsWrite := rg.Group("/pods")
	podsWrite.Use(middleware.RequireScope("pods:write"))
	{
		podsWrite.POST("", podHandler.CreatePod)
		podsWrite.POST("/:key/terminate", podHandler.TerminatePod)
	}

	// Ticket routes
	ticketHandler := NewTicketHandler(svc.Ticket)

	ticketsRead := rg.Group("/tickets")
	ticketsRead.Use(middleware.RequireScope("tickets:read", "tickets:write"))
	{
		ticketsRead.GET("", ticketHandler.ListTickets)
		ticketsRead.GET("/board", ticketHandler.GetBoard)
		ticketsRead.GET("/:slug", ticketHandler.GetTicket)
	}
	ticketsWrite := rg.Group("/tickets")
	ticketsWrite.Use(middleware.RequireScope("tickets:write"))
	{
		ticketsWrite.POST("", ticketHandler.CreateTicket)
		ticketsWrite.PUT("/:slug", ticketHandler.UpdateTicket)
		ticketsWrite.PATCH("/:slug/status", ticketHandler.UpdateTicketStatus)
		ticketsWrite.DELETE("/:slug", ticketHandler.DeleteTicket)
	}

	// Channel routes
	channelHandler := NewChannelHandler(svc.Channel, svc.Ticket)

	channelsRead := rg.Group("/channels")
	channelsRead.Use(middleware.RequireScope("channels:read", "channels:write"))
	{
		channelsRead.GET("", channelHandler.ListChannels)
		channelsRead.GET("/:id", channelHandler.GetChannel)
		channelsRead.GET("/:id/messages", channelHandler.ListMessages)
	}
	channelsWrite := rg.Group("/channels")
	channelsWrite.Use(middleware.RequireScope("channels:write"))
	{
		channelsWrite.POST("", channelHandler.CreateChannel)
		channelsWrite.PUT("/:id", channelHandler.UpdateChannel)
		channelsWrite.POST("/:id/messages", channelHandler.SendMessage)
	}

	// Runner routes (read-only)
	var runnerOpts []RunnerHandlerOption
	if svc.SandboxQueryService != nil {
		runnerOpts = append(runnerOpts, WithSandboxQueryService(svc.SandboxQueryService))
	}
	if svc.SandboxQuerySender != nil {
		runnerOpts = append(runnerOpts, WithSandboxQuerySender(svc.SandboxQuerySender))
	}
	if svc.Pod != nil {
		runnerOpts = append(runnerOpts, WithPodServiceForRunner(svc.Pod))
	}
	if svc.PodCoordinator != nil {
		runnerOpts = append(runnerOpts, WithPodCoordinatorForRunner(svc.PodCoordinator))
	}
	runnerHandler := NewRunnerHandler(svc.Runner, runnerOpts...)

	runnersRead := rg.Group("/runners")
	runnersRead.Use(middleware.RequireScope("runners:read"))
	{
		runnersRead.GET("", runnerHandler.ListRunners)
		runnersRead.GET("/:id", runnerHandler.GetRunner)
		runnersRead.GET("/available", runnerHandler.ListAvailableRunners)
		runnersRead.GET("/:id/pods", runnerHandler.ListRunnerPods)
	}

	// Repository routes (read-only)
	repositoryHandler := NewRepositoryHandler(svc.Repository, svc.Billing)

	reposRead := rg.Group("/repositories")
	reposRead.Use(middleware.RequireScope("repos:read"))
	{
		reposRead.GET("", repositoryHandler.ListRepositories)
		reposRead.GET("/:id", repositoryHandler.GetRepository)
		reposRead.GET("/:id/branches", repositoryHandler.ListBranches)
		reposRead.GET("/:id/merge-requests", repositoryHandler.ListRepositoryMergeRequests)
	}
}

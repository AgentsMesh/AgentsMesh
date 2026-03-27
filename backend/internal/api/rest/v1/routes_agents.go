package v1

import "github.com/gin-gonic/gin"

func registerAgentRoutes(rg *gin.RouterGroup, svc *Services) {
	agentHandler := NewAgentHandler(svc.AgentSvc, svc.CredentialProfile, svc.UserConfig)
	agents := rg.Group("/agents")
	{
		agents.GET("", agentHandler.ListAgents)
		agents.GET("/:agent_slug", agentHandler.GetAgent)
		agents.POST("/custom", agentHandler.CreateCustomAgent)
		agents.PUT("/custom/:agent_slug", agentHandler.UpdateCustomAgent)
		agents.DELETE("/custom/:agent_slug", agentHandler.DeleteCustomAgent)
		agents.GET("/:agent_slug/config-schema", agentHandler.GetAgentConfigSchema)
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

		repositories.POST("/:id/webhook", repositoryHandler.RegisterRepositoryWebhook)
		repositories.DELETE("/:id/webhook", repositoryHandler.DeleteRepositoryWebhook)
		repositories.GET("/:id/webhook/status", repositoryHandler.GetRepositoryWebhookStatus)
		repositories.GET("/:id/webhook/secret", repositoryHandler.GetRepositoryWebhookSecret)
		repositories.POST("/:id/webhook/configured", repositoryHandler.MarkRepositoryWebhookConfigured)

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

package v1

import "github.com/gin-gonic/gin"

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
	if svc.Grant != nil {
		runnerOpts = append(runnerOpts, WithGrantServiceForRunner(svc.Grant))
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

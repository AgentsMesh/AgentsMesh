package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/gin-gonic/gin"
)

func registerPodRoutes(rg *gin.RouterGroup, svc *Services) {
	var podOpts []PodHandlerOption
	if svc.PodCoordinator != nil {
		podOpts = append(podOpts, WithPodCoordinator(svc.PodCoordinator))
	}
	if svc.EventBus != nil {
		podOpts = append(podOpts, WithEventBus(svc.EventBus))
	}
	if svc.PodCoordinator != nil {
		podOpts = append(podOpts, WithCommandSender(svc.PodCoordinator.GetCommandSender()))
	}
	if svc.Grant != nil {
		podOpts = append(podOpts, WithGrantServiceForPod(svc.Grant))
	}
	podHandler := NewPodHandler(svc.Pod, svc.Runner, svc.PodOrchestrator, podOpts...)
	pods := rg.Group("/pods")
	{
		pods.GET("", podHandler.ListPods)
		pods.POST("", podHandler.CreatePod)
		pods.GET("/:key", podHandler.GetPod)
		pods.POST("/:key/terminate", podHandler.TerminatePod)
		pods.PATCH("/:key/alias", podHandler.UpdatePodAlias)
		pods.PATCH("/:key/perpetual", podHandler.UpdatePodPerpetual)
		pods.GET("/:key/connect", podHandler.GetConnectionInfo)
		pods.POST("/:key/prompt", podHandler.SendPodPrompt)
		pods.GET("/:key/grants", podHandler.ListPodGrants)
		pods.POST("/:key/grants", podHandler.GrantPodAccess)
		pods.DELETE("/:key/grants/:grant_id", podHandler.RevokePodGrant)
	}

	if svc.RelayManager != nil && svc.RelayTokenGenerator != nil {
		var commandSender runner.RunnerCommandSender
		var stateReader runner.RunnerStateReader
		if svc.PodCoordinator != nil {
			cs := svc.PodCoordinator.GetCommandSender()
			commandSender = cs
			if sr, ok := cs.(runner.RunnerStateReader); ok {
				stateReader = sr
			}
		}
		RegisterPodConnectRoutes(rg, svc.Pod, svc.RelayManager, svc.RelayTokenGenerator, commandSender, stateReader, svc.GeoResolver, svc.Grant)
	}

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

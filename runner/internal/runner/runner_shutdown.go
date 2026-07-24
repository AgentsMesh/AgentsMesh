package runner

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Pods stop before the supervisor so final status and token usage can still use gRPC.
func (r *Runner) stopAllPods() {
	log := logger.Runner()
	var autopilotsCopy []*autopilot.AutopilotController
	if r.autopilotStore != nil {
		autopilotsCopy = r.autopilotStore.DrainAll()
	}
	if len(autopilotsCopy) > 0 {
		log.Info("Stopping all autopilots in parallel", "count", len(autopilotsCopy))
		var autopilotWg sync.WaitGroup
		for _, controller := range autopilotsCopy {
			autopilotWg.Add(1)
			go func(controller *autopilot.AutopilotController) {
				defer autopilotWg.Done()
				controller.Stop()
			}(controller)
		}
		autopilotWg.Wait()
	}

	pods := r.podStore.All()
	if len(pods) == 0 {
		return
	}
	log.Info("Stopping all pods in parallel", "count", len(pods))

	var wg sync.WaitGroup
	for _, pod := range pods {
		wg.Add(1)
		go func(pod *Pod) {
			defer wg.Done()
			pod.lifecycleMu.Lock()
			defer pod.lifecycleMu.Unlock()
			// Claim cleanup before Detach can invoke the old perpetual exit handler.
			if !r.podStore.DeleteIf(pod.PodKey, pod) {
				return
			}
			transition := pod.BeginRelayRuntimeTransition()
			runtime := pod.installedRuntime()
			pod.SetStatus(PodStatusStopped)
			log.Debug("Detaching pod (daemon stays alive)", "pod_key", pod.PodKey)

			podKey := pod.PodKey
			agent := pod.Agent
			sandboxPath := pod.SandboxPath
			podStartedAt := pod.StartedAt

			pod.StopStateDetector()
			if runtime.IO != nil {
				if pod.IsACPMode() {
					runtime.IO.Stop()
				} else {
					runtime.IO.Detach()
				}
				runtime.IO.Teardown()
			}
			pod.TeardownRelayTransports(r.GetLocalRelayServer())
			pod.clearRuntimeDuringTransition(transition)
			if r.messageHandler != nil {
				r.messageHandler.collectAndSendTokenUsage(podKey, agent, sandboxPath, podStartedAt)
			}
		}(pod)
	}

	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		log.Info("All pods stopped successfully")
	case <-timer.C:
		log.Warn("Timeout waiting for pods to stop; some token usage data may be lost", "count", len(pods))
	}
}

package runner

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

func (h *RunnerMessageHandler) finalizePodExit(
	podKey string,
	pod *Pod,
	exitCode int,
	stopIO bool,
) bool {
	pod.lifecycleMu.Lock()
	defer pod.lifecycleMu.Unlock()

	current, ok := h.podStore.Get(podKey)
	if !ok || current != pod {
		return false
	}
	transition := pod.BeginRelayRuntimeTransition()
	runtime := pod.installedRuntime()
	if !h.podStore.DeleteIf(podKey, pod) {
		return false
	}
	pod.SetStatus(PodStatusStopped)

	h.stopPodDependents(podKey, pod)
	earlyOutput := h.drainPodRuntime(podKey, pod, runtime, stopIO)
	pod.clearRuntimeDuringTransition(transition)
	h.removePodRegistrations(podKey, pod)
	h.sendPodTermination(podKey, pod, exitCode, earlyOutput)
	return true
}

func (h *RunnerMessageHandler) stopPodDependents(podKey string, pod *Pod) {
	if ac := h.runner.GetAutopilotByPodKey(podKey); ac != nil {
		ac.Stop()
		if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
			agentMon.Unsubscribe("autopilot-" + ac.Key())
		}
		h.runner.RemoveAutopilot(ac.Key())
	}
	pod.StopStateDetector()
}

func (h *RunnerMessageHandler) drainPodRuntime(
	podKey string,
	pod *Pod,
	runtime PodRuntime,
	stopIO bool,
) string {
	var earlyOutput string
	if runtime.IO != nil {
		if stopIO {
			runtime.IO.Stop()
		}
		earlyOutput = runtime.IO.Teardown()
	}
	pod.TeardownRelayTransports(h.runner.GetLocalRelayServer())
	return earlyOutput
}

func (h *RunnerMessageHandler) removePodRegistrations(podKey string, pod *Pod) {
	if pod.SandboxPath != "" {
		_ = poddaemon.DeleteState(pod.SandboxPath)
	}
	if mcpSrv := h.runner.GetMCPServer(); mcpSrv != nil {
		mcpSrv.UnregisterPod(podKey)
	}
	if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
		agentMon.UnregisterPod(podKey)
	}
}

func (h *RunnerMessageHandler) sendPodTermination(
	podKey string,
	pod *Pod,
	exitCode int,
	earlyOutput string,
) {
	podStatus, errorMsg := "completed", earlyOutput
	if exitCode > 0 && exitCode < 128 {
		podStatus = "error"
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("process exited with code %d", exitCode)
		}
	}
	if ptyErr := pod.GetPTYError(); ptyErr != "" {
		podStatus, errorMsg = "error", ptyErr
	}
	if h.conn != nil {
		if err := h.conn.SendPodTerminated(podKey, int32(exitCode), errorMsg, podStatus); err != nil {
			logger.Pod().Error("Failed to send pod terminated event", "error", err)
		}
	}
	agent, sandboxPath, startedAt := pod.Agent, pod.SandboxPath, pod.StartedAt
	safego.Go("token-usage-exit", func() {
		h.collectAndSendTokenUsage(podKey, agent, sandboxPath, startedAt)
	})
}

func (h *RunnerMessageHandler) cleanupPodExitFinal(pod *Pod, exitCode int) {
	h.finalizePodExit(pod.PodKey, pod, exitCode, false)
}

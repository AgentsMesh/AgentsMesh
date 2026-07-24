package runner

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
)

const defaultVTHistoryLimit = 100

type perpetualSessionFactory func(
	manager *poddaemon.PodDaemonManager,
	opts poddaemon.CreateOpts,
) (terminal.PtyProcess, error)

type perpetualRuntimeFactory func(
	pod *Pod,
	process terminal.PtyProcess,
	cols, rows int,
) (PodRuntime, error)

func defaultPerpetualSessionFactory(
	manager *poddaemon.PodDaemonManager,
	opts poddaemon.CreateOpts,
) (terminal.PtyProcess, error) {
	process, _, err := manager.CreateSession(opts)
	return process, err
}

func isCleanExit(exitCode int) bool {
	return exitCode == 0 || exitCode >= 128 || exitCode == -1
}

// restartPerpetualPod restarts a perpetual pod's process in the same sandbox.
// Called from cleanupPodExit when a perpetual pod exits cleanly.
func (h *RunnerMessageHandler) restartPerpetualPod(
	pod *Pod,
	exitCode int,
	transition RelayRuntimeTransition,
) {
	log := logger.Pod()
	log.Info("Perpetual pod restarting",
		"pod_key", pod.PodKey, "exit_code", exitCode, "restart_count", pod.RestartCount)

	// Collect token usage from the completed session before restart
	h.collectTokenUsageAsync(pod)

	mgr := h.runner.GetPodDaemonManager()
	if mgr == nil {
		log.Error("PodDaemonManager unavailable, falling back to normal exit", "pod_key", pod.PodKey)
		h.cleanupPodExitFinal(pod, exitCode)
		return
	}

	const defaultCols, defaultRows = 80, 24

	createSession := h.perpetualSessionFactory
	if createSession == nil {
		createSession = defaultPerpetualSessionFactory
	}
	dpty, err := createSession(mgr, poddaemon.CreateOpts{
		PodKey:         pod.PodKey,
		Agent:          pod.Agent,
		Command:        pod.LaunchCommand,
		Args:           pod.LaunchArgs,
		WorkDir:        pod.WorkDir,
		Env:            pod.LaunchEnv,
		Cols:           defaultCols,
		Rows:           defaultRows,
		SandboxPath:    pod.SandboxPath,
		RepositoryURL:  pod.RepositoryURL,
		Branch:         pod.Branch,
		TicketSlug:     pod.TicketSlug,
		VTHistoryLimit: defaultVTHistoryLimit,
		Perpetual:      true,
	})
	if err != nil {
		log.Error("Failed to restart perpetual pod, falling back to normal exit",
			"pod_key", pod.PodKey, "error", err)
		h.cleanupPodExitFinal(pod, exitCode)
		return
	}

	if !h.ownsPodRuntimeTransition(pod, transition) {
		_ = dpty.Kill()
		_ = dpty.Close()
		return
	}

	buildRuntime := h.perpetualRuntimeFactory
	if buildRuntime == nil {
		buildRuntime = h.buildPerpetualPTYRuntime
	}
	runtime, err := buildRuntime(pod, dpty, defaultCols, defaultRows)
	if err != nil {
		log.Error("Failed to build replacement runtime for perpetual pod", "pod_key", pod.PodKey, "error", err)
		_ = dpty.Close()
		if !h.ownsPodRuntimeTransition(pod, transition) {
			return
		}
		h.cleanupPodExitFinal(pod, exitCode)
		return
	}

	runtime.IO.SetExitHandler(h.createExitHandler(pod.PodKey))
	runtime.IO.SetIOErrorHandler(h.createPTYErrorHandler(pod.PodKey, pod))
	pod.lifecycleMu.Lock()
	if !h.ownsPodRuntimeTransition(pod, transition) ||
		!pod.installRuntimeDuringTransition(transition, runtime) {
		pod.lifecycleMu.Unlock()
		discardPerpetualRuntime(runtime, dpty, false)
		return
	}
	if err := runtime.IO.Start(); err != nil {
		pod.clearRuntimeDuringTransition(transition)
		pod.lifecycleMu.Unlock()
		discardPerpetualRuntime(runtime, dpty, false)
		log.Error("Failed to start replacement runtime for perpetual pod", "pod_key", pod.PodKey, "error", err)
		h.cleanupPodExitFinal(pod, exitCode)
		return
	}
	if h.conn != nil {
		pod.SubscribeAgentStatusBridge(h.conn.SendAgentStatus)
	}
	if !pod.EndRelayRuntimeTransition(transition) {
		pod.clearRuntimeDuringTransition(transition)
		pod.lifecycleMu.Unlock()
		discardPerpetualRuntime(runtime, dpty, true)
		h.cleanupPodExitFinal(pod, exitCode)
		return
	}
	pod.StartedAt = time.Now()
	newPID := int32(runtime.IO.GetPID())
	pod.lifecycleMu.Unlock()

	// Notify Backend with new PID (sent AFTER restart succeeds)
	if h.conn != nil {
		if err := h.conn.SendPodRestarting(pod.PodKey, int32(exitCode), int32(pod.RestartCount), newPID); err != nil {
			log.Error("Failed to send pod restarting event", "pod_key", pod.PodKey, "error", err)
		}
	}

	log.Info("Perpetual pod restarted", "pod_key", pod.PodKey, "restart_count", pod.RestartCount)
}

func discardPerpetualRuntime(runtime PodRuntime, dpty terminal.PtyProcess, started bool) {
	if runtime.IO != nil {
		if started {
			runtime.IO.Stop()
		}
		runtime.IO.Teardown()
	}
	if !started {
		if err := dpty.Kill(); err != nil {
			logger.Pod().Debug("Failed to kill discarded perpetual runtime", "error", err)
		}
		_ = dpty.Close()
	}
}

// collectTokenUsageAsync collects token usage from a completed session.
func (h *RunnerMessageHandler) collectTokenUsageAsync(pod *Pod) {
	agent := pod.Agent
	sandboxPath := pod.SandboxPath
	podStartedAt := pod.StartedAt
	podKey := pod.PodKey
	safego.Go("token-usage-perpetual", func() {
		h.collectAndSendTokenUsage(podKey, agent, sandboxPath, podStartedAt)
	})
}

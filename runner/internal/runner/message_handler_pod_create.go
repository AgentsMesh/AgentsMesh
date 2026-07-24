package runner

import (
	"fmt"
	"os"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

func (h *RunnerMessageHandler) OnCreatePod(cmd *runnerv1.CreatePodCommand) error {
	log := logger.Pod()
	log.Info("Creating pod", "pod_key", cmd.PodKey, "command", cmd.LaunchCommand,
		"args", cmd.LaunchArgs)

	ctx := h.runner.GetRunContext()
	pending := &Pod{PodKey: cmd.PodKey, Status: PodStatusInitializing}
	h.podStore.Put(cmd.PodKey, pending)
	_ = h.conn.SendPodInitProgress(cmd.PodKey, "received", 1, "Pod command received by runner")

	cols, rows := int(cmd.Cols), int(cmd.Rows)
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	cfg := h.runner.GetConfig()
	builder := h.runner.NewPodBuilder().
		WithCommand(cmd).
		WithPtySize(cols, rows).
		WithOSCHandler(h.createOSCHandler(cmd.PodKey))
	if cfg.LogPTY {
		builder.WithPTYLogging(cfg.GetLogPTYDir())
	}

	pod, err := builder.Build(ctx)
	if err != nil {
		h.podStore.DeleteIf(cmd.PodKey, pending)
		if podErr, ok := err.(*client.PodError); ok {
			h.sendPodErrorWithCode(cmd.PodKey, podErr)
		} else {
			h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to build pod: %v", err))
		}
		return fmt.Errorf("failed to build pod: %w", err)
	}

	if !h.podStore.ReplaceIf(cmd.PodKey, pending, pod) {
		log.InfoContext(ctx, "Pod was terminated during build, cleaning up", "pod_key", cmd.PodKey)
		h.cleanupFailedPodRuntime(pod)
		if pod.SandboxPath != "" {
			os.RemoveAll(pod.SandboxPath)
		}
		return fmt.Errorf("pod %s was terminated during build", cmd.PodKey)
	}

	if pod.IsACPMode() {
		if err := h.wireAndStartACPPod(pod, cmd, cols, rows); err != nil {
			return err
		}
	} else if err := h.wireAndStartPTYPod(pod, cmd, cols, rows); err != nil {
		return err
	}
	h.registerMCPIfCurrent(cmd.PodKey, pod, cmd.LaunchCommand)
	return nil
}

func (h *RunnerMessageHandler) registerMCPIfCurrent(podKey string, pod *Pod, agent string) {
	pod.lifecycleMu.Lock()
	defer pod.lifecycleMu.Unlock()
	current, ok := h.podStore.Get(podKey)
	if !ok || current != pod || pod.GetStatus() != PodStatusRunning {
		return
	}
	if mcpSrv := h.runner.GetMCPServer(); mcpSrv != nil {
		mcpSrv.RegisterPod(podKey, h.conn.GetOrgSlug(), nil, nil, agent)
	}
}

func (h *RunnerMessageHandler) wireAndStartPTYPod(
	pod *Pod,
	cmd *runnerv1.CreatePodCommand,
	cols, rows int,
) error {
	log := logger.Pod()
	pod.lifecycleMu.Lock()
	current, owned := h.podStore.Get(cmd.PodKey)
	if !owned || current != pod {
		pod.lifecycleMu.Unlock()
		return fmt.Errorf("pod %s was terminated before terminal start", cmd.PodKey)
	}

	runtime := pod.installedRuntime()
	if !runtime.valid() {
		pod.lifecycleMu.Unlock()
		return fmt.Errorf("pod runtime not available: %s", cmd.PodKey)
	}
	runtime.IO.SetExitHandler(h.createExitHandler(cmd.PodKey))
	runtime.IO.SetIOErrorHandler(h.createPTYErrorHandler(cmd.PodKey, pod))
	if err := runtime.IO.Start(); err != nil {
		h.podStore.DeleteIf(cmd.PodKey, pod)
		pod.lifecycleMu.Unlock()
		h.cleanupFailedPodRuntime(pod)
		if pod.SandboxPath != "" {
			os.RemoveAll(pod.SandboxPath)
		}
		h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to start terminal: %v", err))
		return fmt.Errorf("failed to start terminal: %w", err)
	}

	pod.SetStatus(PodStatusRunning)
	pid := runtime.IO.GetPID()
	if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
		agentMon.RegisterPod(cmd.PodKey, pid)
	}
	pod.SubscribeAgentStatusBridge(h.conn.SendAgentStatus)
	h.sendPodCreated(cmd.PodKey, pid, pod.SandboxPath, pod.Branch, uint16(cols), uint16(rows))
	log.Info("Pod created (PTY)", "pod_key", cmd.PodKey, "pid", pid, "sandbox", pod.SandboxPath)
	pod.lifecycleMu.Unlock()
	return nil
}

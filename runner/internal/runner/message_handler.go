package runner

import (
	"fmt"
	"log/slog"
	"os"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// RunnerMessageHandler implements client.MessageHandler interface.
type RunnerMessageHandler struct {
	runner             MessageHandlerContext
	podStore           PodStore
	conn               client.Connection
	relayClientFactory func(url, podKey, token string, logger *slog.Logger) relay.RelayClient
}

// NewRunnerMessageHandler creates a new message handler.
func NewRunnerMessageHandler(runner MessageHandlerContext, store PodStore, conn client.Connection) *RunnerMessageHandler {
	logger.Runner().Debug("Creating message handler")
	return &RunnerMessageHandler{
		runner:   runner,
		podStore: store,
		conn:     conn,
		relayClientFactory: func(url, podKey, token string, logger *slog.Logger) relay.RelayClient {
			return relay.NewClient(url, podKey, token, logger)
		},
	}
}

// OnCreatePod handles create pod requests from server.
func (h *RunnerMessageHandler) OnCreatePod(cmd *runnerv1.CreatePodCommand) error {
	log := logger.Pod()
	log.Info("Creating pod", "pod_key", cmd.PodKey, "command", cmd.LaunchCommand,
		"args", cmd.LaunchArgs)

	ctx := h.runner.GetRunContext()

	// Register pending placeholder to prevent race with TerminatePod during Build
	h.podStore.Put(cmd.PodKey, &Pod{
		PodKey: cmd.PodKey,
		Status: PodStatusInitializing,
	})

	// ACK: tell Backend we received the command before heavy work
	_ = h.conn.SendPodInitProgress(cmd.PodKey, "received", 1, "Pod command received by runner")

	cols := int(cmd.Cols)
	rows := int(cmd.Rows)
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
		h.podStore.Delete(cmd.PodKey)
		if podErr, ok := err.(*client.PodError); ok {
			h.sendPodErrorWithCode(cmd.PodKey, podErr)
		} else {
			h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to build pod: %v", err))
		}
		return fmt.Errorf("failed to build pod: %w", err)
	}

	// Check if pod was terminated during Build
	if _, ok := h.podStore.Get(cmd.PodKey); !ok {
		log.Info("Pod was terminated during build, cleaning up", "pod_key", cmd.PodKey)
		if pod.IO != nil {
			pod.IO.Teardown()
			pod.IO.Stop()
		}
		if pod.SandboxPath != "" {
			os.RemoveAll(pod.SandboxPath)
		}
		return fmt.Errorf("pod %s was terminated during build", cmd.PodKey)
	}

	h.podStore.Put(cmd.PodKey, pod)

	if pod.IsACPMode() {
		if err := h.wireAndStartACPPod(pod, cmd, cols, rows); err != nil {
			return err
		}
	} else {
		if err := h.wireAndStartPTYPod(pod, cmd, cols, rows); err != nil {
			return err
		}
	}

	if mcpSrv := h.runner.GetMCPServer(); mcpSrv != nil {
		orgSlug := h.conn.GetOrgSlug()
		mcpSrv.RegisterPod(cmd.PodKey, orgSlug, nil, nil, cmd.LaunchCommand)
	}

	return nil
}

// wireAndStartPTYPod wires up PTY-specific handlers and starts the terminal.
func (h *RunnerMessageHandler) wireAndStartPTYPod(pod *Pod, cmd *runnerv1.CreatePodCommand, cols, rows int) error {
	log := logger.Pod()

	pod.IO.SetExitHandler(h.createExitHandler(cmd.PodKey))
	pod.IO.SetIOErrorHandler(h.createPTYErrorHandler(cmd.PodKey, pod))

	if err := pod.IO.Start(); err != nil {
		h.podStore.Delete(cmd.PodKey)
		if pod.IO != nil {
			pod.IO.Teardown()
		}
		if pod.SandboxPath != "" {
			os.RemoveAll(pod.SandboxPath)
		}
		h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to start terminal: %v", err))
		return fmt.Errorf("failed to start terminal: %w", err)
	}

	pod.SetStatus(PodStatusRunning)

	if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
		agentMon.RegisterPod(cmd.PodKey, pod.IO.GetPID())
	}

	pod.SubscribeAgentStatusBridge(h.conn.SendAgentStatus)

	h.sendPodCreated(cmd.PodKey, pod.IO.GetPID(), pod.SandboxPath, pod.Branch, uint16(cols), uint16(rows))
	log.Info("Pod created (PTY)", "pod_key", cmd.PodKey, "pid", pod.IO.GetPID(), "sandbox", pod.SandboxPath)
	return nil
}

// OnTerminatePod handles terminate pod requests from server.
func (h *RunnerMessageHandler) OnTerminatePod(req client.TerminatePodRequest) error {
	log := logger.Pod()
	log.Info("Terminating pod", "pod_key", req.PodKey)

	if _, ok := h.podStore.Get(req.PodKey); !ok {
		log.Warn("Pod not found for termination", "pod_key", req.PodKey)
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	h.cleanupPodExit(req.PodKey, -1, true)
	return nil
}

// OnListPods returns current pods.
func (h *RunnerMessageHandler) OnListPods() []client.PodInfo {
	pods := h.podStore.All()
	result := make([]client.PodInfo, 0, len(pods))

	for _, s := range pods {
		info := client.PodInfo{
			PodKey:      s.PodKey,
			Status:      s.GetStatus(),
			AgentStatus: h.getAgentStatusFromDetector(s),
		}
		if s.IO != nil {
			info.Pid = s.IO.GetPID()
		}
		result = append(result, info)
	}

	return result
}

func (h *RunnerMessageHandler) getAgentStatusFromDetector(pod *Pod) string {
	if pod.IO != nil {
		return pod.IO.GetAgentStatus()
	}
	return "idle"
}

var _ client.MessageHandler = (*RunnerMessageHandler)(nil)

// Note: OnSubscribePod, setupRelayClientHandlers, OnUnsubscribePod are in message_handler_relay.go
// Note: OnListRelayConnections, OnPodInput, OnQuerySandboxes, OnObservePod, OnSendPrompt are in message_handler_ops.go

package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/monitor"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
	"github.com/anthropics/agentsmesh/runner/internal/updater"
)

// --- Role interfaces (ISP: split by consumer usage clusters) ---

// CoreContext provides base capabilities needed by almost all handlers.
type CoreContext interface {
	// GetRunContext returns the runner lifecycle context for cancellable operations.
	GetRunContext() context.Context

	// GetConfig returns the runner configuration.
	GetConfig() *config.Config
}

// PodComponentContext provides Pod lifecycle capabilities
// (used by OnCreatePod, OnTerminatePod, recovery, sandbox queries).
type PodComponentContext interface {
	// NewPodBuilder creates a new PodBuilder with the runner's dependencies.
	NewPodBuilder() *PodBuilder

	// NewPodController creates a new PodController for the given pod.
	NewPodController(pod *Pod) *PodControllerImpl

	// GetMCPServer returns the MCP server (may be nil).
	GetMCPServer() MCPServer

	// GetAgentMonitor returns the agent monitor (may be nil).
	GetAgentMonitor() AgentMonitor

	// GetSandboxStatus returns sandbox status for a pod.
	GetSandboxStatus(podKey string) *client.SandboxStatusInfo
}

// AutopilotRegistry manages AutopilotController instances
// (used by OnCreateAutopilot, OnAutopilotControl, OnTerminatePod).
type AutopilotRegistry interface {
	// GetAutopilot returns an AutopilotController by key.
	GetAutopilot(key string) *autopilot.AutopilotController

	// AddAutopilot registers an AutopilotController.
	AddAutopilot(ac *autopilot.AutopilotController)

	// RemoveAutopilot removes an AutopilotController by key.
	RemoveAutopilot(key string)

	// GetAutopilotByPodKey returns an AutopilotController by its associated pod key.
	GetAutopilotByPodKey(podKey string) *autopilot.AutopilotController
}

// UpgradeController manages upgrade/draining state machine
// (used only by OnUpgradeRunner).
type UpgradeController interface {
	// GetUpdater returns the updater instance (may be nil).
	GetUpdater() *updater.Updater

	// TryStartUpgrade atomically acquires the upgrade lock.
	TryStartUpgrade() bool

	// FinishUpgrade releases the upgrade lock.
	FinishUpgrade()

	// GetActivePodCount returns the number of active pods.
	GetActivePodCount() int

	// SetDraining sets the draining mode flag.
	SetDraining(draining bool)

	// GetRestartFunc returns the restart function (may be nil).
	GetRestartFunc() func() (int, error)
}

// MessageHandlerContext is the composite interface for backward compatibility.
// Runner implements this; individual handlers only consume the role interfaces they need.
type MessageHandlerContext interface {
	CoreContext
	PodComponentContext
	AutopilotRegistry
	UpgradeController
}

// MCPServer defines the MCP server operations needed by message handlers.
type MCPServer interface {
	RegisterPod(podKey, orgSlug string, ticketID, projectID *int, agentType string)
	UnregisterPod(podKey string)
}

// AgentMonitor defines the monitor operations needed by message handlers.
type AgentMonitor interface {
	RegisterPod(podID string, pid int)
	UnregisterPod(podID string)
	Subscribe(id string, callback func(monitor.PodStatus))
	Unsubscribe(id string)
}

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
	log.Info("Creating pod", "pod_key", cmd.PodKey, "command", cmd.LaunchCommand, "args", cmd.LaunchArgs)

	// Use runner's lifecycle context so long operations (git clone) can be
	// cancelled on shutdown, instead of blocking with context.Background().
	ctx := h.runner.GetRunContext()

	// Register a pending pod placeholder to prevent race conditions:
	// - TerminatePod arriving during Build can find and remove the placeholder
	// - Exit handler after Start can find the pod in store
	h.podStore.Put(cmd.PodKey, &Pod{
		PodKey: cmd.PodKey,
		Status: PodStatusInitializing,
	})

	// ACK: immediately tell Backend we received the command, before any heavy work.
	// This lets Backend distinguish "Runner got the command" from "Runner never saw it".
	_ = h.conn.SendPodInitProgress(cmd.PodKey, "received", 1, "Pod command received by runner")

	// Build pod with all components (SRP: PodBuilder handles all component creation)
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
		WithTerminalSize(cols, rows).
		WithOSCHandler(h.createOSCHandler(cmd.PodKey))

	// Enable PTY logging if configured
	if cfg.LogPTY {
		builder.WithPTYLogging(cfg.GetLogPTYDir())
	}

	pod, err := builder.Build(ctx)
	if err != nil {
		h.podStore.Delete(cmd.PodKey) // Remove pending placeholder
		if podErr, ok := err.(*client.PodError); ok {
			h.sendPodErrorWithCode(cmd.PodKey, podErr)
		} else {
			h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to build pod: %v", err))
		}
		return fmt.Errorf("failed to build pod: %w", err)
	}

	// Check if pod was terminated during Build (TerminatePod removed the placeholder)
	if _, ok := h.podStore.Get(cmd.PodKey); !ok {
		log.Info("Pod was terminated during build, cleaning up", "pod_key", cmd.PodKey)
		if pod.Aggregator != nil {
			pod.Aggregator.Stop()
		}
		if pod.PTYLogger != nil {
			pod.PTYLogger.Close()
		}
		if pod.SandboxPath != "" {
			os.RemoveAll(pod.SandboxPath)
		}
		return fmt.Errorf("pod %s was terminated during build", cmd.PodKey)
	}

	// Set exit handler (callback to MessageHandler for lifecycle events)
	pod.Terminal.SetExitHandler(h.createExitHandler(cmd.PodKey))

	// Set PTY error handler to notify frontend when terminal I/O fails.
	// Without this, a PTY read error (e.g., disk full) causes a frozen terminal
	// because the relay stays connected but no data flows through it.
	pod.Terminal.SetPTYErrorHandler(h.createPTYErrorHandler(cmd.PodKey, pod))

	// Replace pending placeholder with fully built pod BEFORE starting terminal.
	// This ensures the exit handler can find the pod if the process exits immediately.
	h.podStore.Put(cmd.PodKey, pod)

	// Start terminal
	if err := pod.Terminal.Start(); err != nil {
		h.podStore.Delete(cmd.PodKey) // Remove from store on failure
		// Clean up resources that Build() created
		if pod.Aggregator != nil {
			pod.Aggregator.Stop()
		}
		if pod.PTYLogger != nil {
			pod.PTYLogger.Close()
		}
		if pod.SandboxPath != "" {
			os.RemoveAll(pod.SandboxPath)
		}
		h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to start terminal: %v", err))
		return fmt.Errorf("failed to start terminal: %w", err)
	}

	pod.SetStatus(PodStatusRunning)

	// Register with MCP server and Claude monitor
	if mcpSrv := h.runner.GetMCPServer(); mcpSrv != nil {
		orgSlug := h.conn.GetOrgSlug()
		mcpSrv.RegisterPod(cmd.PodKey, orgSlug, nil, nil, cmd.LaunchCommand)
	}
	if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
		agentMon.RegisterPod(cmd.PodKey, pod.Terminal.PID())
	}

	// Subscribe to VT state detection events, bridge to gRPC.
	// Uses shared implementation with deduplication (same as session recovery).
	pod.SubscribeAgentStatusBridge(h.conn.SendAgentStatus)

	h.sendPodCreated(cmd.PodKey, pod.Terminal.PID(), pod.SandboxPath, pod.Branch, uint16(cols), uint16(rows))

	log.Info("Pod created", "pod_key", cmd.PodKey, "pid", pod.Terminal.PID(), "sandbox", pod.SandboxPath)
	return nil
}

// OnTerminatePod handles terminate pod requests from server.
func (h *RunnerMessageHandler) OnTerminatePod(req client.TerminatePodRequest) error {
	log := logger.Pod()
	log.Info("Terminating pod", "pod_key", req.PodKey)

	pod := h.podStore.Delete(req.PodKey)
	if pod == nil {
		log.Warn("Pod not found for termination", "pod_key", req.PodKey)
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	// Clean up associated Autopilot if any (before terminal teardown)
	if ac := h.runner.GetAutopilotByPodKey(req.PodKey); ac != nil {
		ac.Stop()
		if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
			agentMon.Unsubscribe("autopilot-" + ac.Key())
		}
		h.runner.RemoveAutopilot(ac.Key())
	}

	pod.StopStateDetector()
	// Stop aggregator BEFORE disconnecting relay, so the final flush
	// can still be sent through the relay (matches createExitHandler order).
	// Close PTYLogger AFTER Aggregator.Stop() to avoid silent data loss.
	var earlyOutput string
	if pod.Aggregator != nil {
		pod.Aggregator.Stop()

		// Drain early output buffered before relay connected (matches createExitHandler).
		// Without this, error messages from fast-exiting processes are silently lost.
		if buf := pod.Aggregator.DrainEarlyBuffer(); len(buf) > 0 {
			earlyOutput = string(buf)
			log.Info("Drained early output on terminate",
				"pod_key", req.PodKey, "bytes", len(buf))
		}
	}
	if pod.PTYLogger != nil {
		pod.PTYLogger.Close()
	}
	pod.DisconnectRelay()
	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}

	// Clean up Pod Daemon state (triggers daemon self-exit when it detects file deletion)
	if pod.SandboxPath != "" {
		_ = poddaemon.DeleteState(pod.SandboxPath)
	}

	if mcpSrv := h.runner.GetMCPServer(); mcpSrv != nil {
		mcpSrv.UnregisterPod(req.PodKey)
	}
	if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
		agentMon.UnregisterPod(req.PodKey)
	}

	// Include early output in the termination event so the backend can
	// display why the process failed (matches createExitHandler behavior).
	if h.conn != nil {
		if err := h.conn.SendPodTerminated(req.PodKey, 0, earlyOutput); err != nil {
			log.Error("Failed to send pod terminated event", "error", err)
		}
	}

	// Async token usage collection — same as createExitHandler.
	// Capture values before goroutine to avoid race.
	agentType := pod.AgentType
	sandboxPath := pod.SandboxPath
	podStartedAt := pod.StartedAt
	safego.Go("token-usage-terminate", func() {
		h.collectAndSendTokenUsage(req.PodKey, agentType, sandboxPath, podStartedAt)
	})

	log.Info("Pod terminated", "pod_key", req.PodKey)
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
		if s.Terminal != nil {
			info.Pid = s.Terminal.PID()
		}
		result = append(result, info)
	}

	return result
}

// getAgentStatusFromDetector maps the detector's AgentState to backend status string.
func (h *RunnerMessageHandler) getAgentStatusFromDetector(pod *Pod) string {
	if pod.VirtualTerminal == nil {
		return "idle"
	}
	d := pod.GetOrCreateStateDetector()
	if d == nil {
		return "idle"
	}
	switch d.GetState() {
	case detector.StateExecuting:
		return "executing"
	case detector.StateWaiting:
		return "waiting"
	case detector.StateNotRunning:
		return "idle"
	default:
		return "idle"
	}
}

// Ensure RunnerMessageHandler implements client.MessageHandler
var _ client.MessageHandler = (*RunnerMessageHandler)(nil)

// Note: OnSubscribeTerminal, setupRelayClientHandlers, OnUnsubscribeTerminal are in message_handler_relay.go
// Note: OnListRelayConnections, OnTerminalInput/Resize/Redraw, OnQuerySandboxes are in message_handler_ops.go

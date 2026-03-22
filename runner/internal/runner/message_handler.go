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
	"github.com/anthropics/agentsmesh/runner/internal/relay"
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
	log.Info("Creating pod", "pod_key", cmd.PodKey, "command", cmd.LaunchCommand,
		"args", cmd.LaunchArgs, "interaction_mode", cmd.GetInteractionMode())

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
		WithPtySize(cols, rows).
		WithOSCHandler(h.createOSCHandler(cmd.PodKey))

	// Enable PTY logging if configured (PTY mode only)
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
		if pod.IO != nil {
			pod.IO.Teardown()
			pod.IO.Stop()
		}
		if pod.SandboxPath != "" {
			os.RemoveAll(pod.SandboxPath)
		}
		return fmt.Errorf("pod %s was terminated during build", cmd.PodKey)
	}

	// Replace pending placeholder with fully built pod BEFORE starting.
	h.podStore.Put(cmd.PodKey, pod)

	// Mode-specific wiring and start
	if pod.IsACPMode() {
		if err := h.wireAndStartACPPod(pod, cmd, cols, rows); err != nil {
			return err
		}
	} else {
		if err := h.wireAndStartPTYPod(pod, cmd, cols, rows); err != nil {
			return err
		}
	}

	// Shared post-start: register with MCP server
	if mcpSrv := h.runner.GetMCPServer(); mcpSrv != nil {
		orgSlug := h.conn.GetOrgSlug()
		mcpSrv.RegisterPod(cmd.PodKey, orgSlug, nil, nil, cmd.LaunchCommand)
	}

	return nil
}

// wireAndStartPTYPod wires up PTY-specific handlers and starts the terminal.
func (h *RunnerMessageHandler) wireAndStartPTYPod(pod *Pod, cmd *runnerv1.CreatePodCommand, cols, rows int) error {
	log := logger.Pod()

	// Set exit handler (callback to MessageHandler for lifecycle events)
	pod.Terminal.SetExitHandler(h.createExitHandler(cmd.PodKey))

	// Set PTY error handler to notify frontend when terminal I/O fails.
	pod.Terminal.SetPTYErrorHandler(h.createPTYErrorHandler(cmd.PodKey, pod))

	// Start terminal
	if err := pod.Terminal.Start(); err != nil {
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

	// Register with agent monitor
	if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
		agentMon.RegisterPod(cmd.PodKey, pod.IO.GetPID())
	}

	// Subscribe to VT state detection events, bridge to gRPC.
	pod.SubscribeAgentStatusBridge(h.conn.SendAgentStatus)

	h.sendPodCreated(cmd.PodKey, pod.IO.GetPID(), pod.SandboxPath, pod.Branch, uint16(cols), uint16(rows))
	log.Info("Pod created (PTY)", "pod_key", cmd.PodKey, "pid", pod.IO.GetPID(), "sandbox", pod.SandboxPath)
	return nil
}

// OnTerminatePod handles terminate pod requests from server.
func (h *RunnerMessageHandler) OnTerminatePod(req client.TerminatePodRequest) error {
	log := logger.Pod()
	log.Info("Terminating pod", "pod_key", req.PodKey)

	// Quick check: return error if the pod doesn't exist at all.
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

// getAgentStatusFromDetector maps the agent state to a backend status string.
// Supports both PTY (StateDetector) and ACP (PodIO) modes.
func (h *RunnerMessageHandler) getAgentStatusFromDetector(pod *Pod) string {
	if pod.IO != nil {
		return pod.IO.GetAgentStatus()
	}
	return "idle"
}

// Ensure RunnerMessageHandler implements client.MessageHandler
var _ client.MessageHandler = (*RunnerMessageHandler)(nil)

// Note: OnSubscribePod, setupRelayClientHandlers, OnUnsubscribePod are in message_handler_relay.go
// Note: OnListRelayConnections, OnPodInput, OnQuerySandboxes, OnObservePod, OnSendPrompt are in message_handler_ops.go

package runner

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/monitor"
	"github.com/anthropics/agentsmesh/runner/internal/updater"
)

// --- Role interfaces (ISP: split by consumer usage clusters) ---

// CoreContext provides base capabilities needed by almost all handlers.
type CoreContext interface {
	GetRunContext() context.Context
	GetConfig() *config.Config
}

// PodComponentContext provides Pod lifecycle capabilities.
type PodComponentContext interface {
	NewPodBuilder() *PodBuilder
	NewPodController(pod *Pod) *PodController
	GetMCPServer() MCPServer
	GetAgentMonitor() AgentMonitor
	GetSandboxStatus(podKey string) *client.SandboxStatusInfo
}

// AutopilotRegistry manages AutopilotController instances.
type AutopilotRegistry interface {
	GetAutopilot(key string) *autopilot.AutopilotController
	AddAutopilot(ac *autopilot.AutopilotController)
	RemoveAutopilot(key string)
	GetAutopilotByPodKey(podKey string) *autopilot.AutopilotController
}

// UpgradeController manages upgrade/draining state machine.
type UpgradeController interface {
	GetUpdater() *updater.Updater
	TryStartUpgrade() bool
	FinishUpgrade()
	GetActivePodCount() int
	SetDraining(draining bool)
	GetRestartFunc() func() (int, error)
}

// MessageHandlerContext is the composite interface for backward compatibility.
type MessageHandlerContext interface {
	CoreContext
	PodComponentContext
	AutopilotRegistry
	UpgradeController
}

// MCPServer defines the MCP server operations needed by message handlers.
type MCPServer interface {
	RegisterPod(podKey, orgSlug string, ticketID, projectID *int, agent string)
	UnregisterPod(podKey string)
}

// AgentMonitor defines the monitor operations needed by message handlers.
type AgentMonitor interface {
	RegisterPod(podID string, pid int)
	UnregisterPod(podID string)
	Subscribe(id string, callback func(monitor.PodStatus))
	Unsubscribe(id string)
}

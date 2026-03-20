package runner

import (
	"context"
	"fmt"

	"github.com/thejerf/suture/v4"

	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/updater"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// Compile-time check: Runner implements MessageHandlerContext.
var _ MessageHandlerContext = (*Runner)(nil)

// Runner is the main runner instance
type Runner struct {
	cfg       *config.Config
	conn      client.Connection
	workspace workspace.WorkspaceManagerInterface

	// Pod management
	podStore         PodStore
	messageHandler   *RunnerMessageHandler
	podDaemonManager *poddaemon.PodDaemonManager

	// Enhanced components (MCP + Monitor)
	components *EnhancedComponents

	// Autopilot management
	autopilotStore *AutopilotStore

	// Upgrade/draining state machine
	upgradeCoord *UpgradeCoordinator

	// Run lifecycle context (set by Run, used by message handlers)
	runCtx context.Context

	// Supervisor services (registered before Run)
	additionalServices []suture.Service

	// Channels for coordination
	stopChan chan struct{}
}

// RunnerDeps holds all external dependencies needed to create a Runner.
// This separates I/O-heavy creation (gRPC, workspace, certs) from pure assembly.
type RunnerDeps struct {
	Config           *config.Config
	Connection       client.Connection
	Workspace        workspace.WorkspaceManagerInterface // optional
	PodStore         PodStore                            // optional, defaults to InMemoryPodStore
	PodDaemonManager *poddaemon.PodDaemonManager         // optional
}

// New creates a new runner instance from pre-created dependencies.
// All I/O operations (gRPC connection, certificate checks, workspace creation)
// should be done by the caller before invoking New().
func New(deps RunnerDeps) (*Runner, error) {
	log := logger.Runner()

	if deps.Config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if deps.Connection == nil {
		return nil, fmt.Errorf("connection is required")
	}
	if deps.PodStore == nil {
		deps.PodStore = NewInMemoryPodStore()
	}

	log.Info("Creating runner instance", "node_id", deps.Config.NodeID)

	r := &Runner{
		cfg:              deps.Config,
		conn:             deps.Connection,
		workspace:        deps.Workspace,
		podStore:         deps.PodStore,
		podDaemonManager: deps.PodDaemonManager,
		autopilotStore:   NewAutopilotStore(),
		upgradeCoord:     NewUpgradeCoordinator(deps.PodStore.Count),
		stopChan:         make(chan struct{}),
	}

	// Create message handler and set it on connection
	r.messageHandler = NewRunnerMessageHandler(r, deps.PodStore, deps.Connection)
	deps.Connection.SetHandler(r.messageHandler)

	// Initialize optional enhanced components (MCP, Monitor)
	r.components = NewEnhancedComponents(deps.Config, deps.Connection)
	r.components.SetProviders(r, r)

	log.Info("Runner instance created successfully")
	return r, nil
}

// GetRunContext returns the runner's lifecycle context.
// Returns context.Background() if Run() has not been called yet.
func (r *Runner) GetRunContext() context.Context {
	if r.runCtx != nil {
		return r.runCtx
	}
	return context.Background()
}

// GetConfig returns the runner configuration (implements MessageHandlerContext).
func (r *Runner) GetConfig() *config.Config {
	return r.cfg
}

// GetMCPServer returns the MCP server (implements MessageHandlerContext).
func (r *Runner) GetMCPServer() MCPServer {
	return r.components.MCPServer()
}

// GetAgentMonitor returns the agent monitor (implements MessageHandlerContext).
func (r *Runner) GetAgentMonitor() AgentMonitor {
	return r.components.AgentMonitor()
}

// NewPodBuilder creates a new PodBuilder with the runner's dependencies (implements MessageHandlerContext).
func (r *Runner) NewPodBuilder() *PodBuilder {
	return NewPodBuilderFromRunner(r)
}

// NewPodController creates a new PodController for the given pod (implements MessageHandlerContext).
func (r *Runner) NewPodController(pod *Pod) *PodControllerImpl {
	return NewPodController(pod, r)
}

// GetConnection returns the gRPC connection.
func (r *Runner) GetConnection() client.Connection {
	return r.conn
}

// GetPodDaemonManager returns the Pod Daemon manager (may be nil).
func (r *Runner) GetPodDaemonManager() *poddaemon.PodDaemonManager {
	return r.podDaemonManager
}

// --- UpgradeController delegation (delegates to UpgradeCoordinator) ---

func (r *Runner) TryStartUpgrade() bool {
	if r.upgradeCoord == nil {
		return false
	}
	return r.upgradeCoord.TryStartUpgrade()
}

func (r *Runner) FinishUpgrade() {
	if r.upgradeCoord == nil {
		return
	}
	r.upgradeCoord.FinishUpgrade()
}

func (r *Runner) GetUpdater() *updater.Updater {
	if r.upgradeCoord == nil {
		return nil
	}
	return r.upgradeCoord.GetUpdater()
}

// SetUpdater sets the updater instance for remote upgrade support.
func (r *Runner) SetUpdater(u *updater.Updater) {
	if r.upgradeCoord == nil {
		return
	}
	r.upgradeCoord.SetUpdater(u)
}

func (r *Runner) GetRestartFunc() func() (int, error) {
	if r.upgradeCoord == nil {
		return nil
	}
	return r.upgradeCoord.GetRestartFunc()
}

// SetRestartFunc sets the restart function for post-upgrade restart.
func (r *Runner) SetRestartFunc(fn func() (int, error)) {
	if r.upgradeCoord == nil {
		return
	}
	r.upgradeCoord.SetRestartFunc(fn)
}

// --- AutopilotRegistry delegation (delegates to AutopilotStore) ---

func (r *Runner) GetAutopilot(key string) *autopilot.AutopilotController {
	if r.autopilotStore == nil {
		return nil
	}
	return r.autopilotStore.GetAutopilot(key)
}

func (r *Runner) AddAutopilot(ac *autopilot.AutopilotController) {
	if r.autopilotStore == nil {
		return
	}
	r.autopilotStore.AddAutopilot(ac)
}

func (r *Runner) RemoveAutopilot(key string) {
	if r.autopilotStore == nil {
		return
	}
	r.autopilotStore.RemoveAutopilot(key)
}

func (r *Runner) GetAutopilotByPodKey(podKey string) *autopilot.AutopilotController {
	if r.autopilotStore == nil {
		return nil
	}
	return r.autopilotStore.GetAutopilotByPodKey(podKey)
}

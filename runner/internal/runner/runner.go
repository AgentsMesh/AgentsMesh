package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/thejerf/suture/v4"

	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/clipboard"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/lifecycle"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/mcp"
	"github.com/anthropics/agentsmesh/runner/internal/monitor"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// Runner is the main runner instance
type Runner struct {
	cfg       *config.Config
	conn      client.Connection
	workspace *workspace.Manager
	clipboard clipboard.Backend

	// Pod management
	podStore       PodStore
	messageHandler *RunnerMessageHandler

	// Enhanced components
	mcpManager    *mcp.Manager
	mcpServer     *mcp.HTTPServer
	agentMonitor *monitor.Monitor

	// Autopilot management
	autopilots   map[string]*autopilot.AutopilotController
	autopilotsMu sync.RWMutex

	// Update management
	draining   bool
	drainingMu sync.RWMutex

	// Run lifecycle context (set by Run, used by message handlers)
	runCtx context.Context

	// Supervisor services (registered before Run)
	additionalServices []suture.Service

	// Channels for coordination
	stopChan chan struct{}
}

// New creates a new runner instance
func New(cfg *config.Config) (*Runner, error) {
	log := logger.Runner()
	log.Info("Creating runner instance", "node_id", cfg.NodeID)

	// Load gRPC config (certificates)
	if err := cfg.LoadGRPCConfig(); err != nil {
		return nil, fmt.Errorf("failed to load gRPC config: %w - please register the runner first using 'runner register'", err)
	}

	// Validate required configuration
	if cfg.OrgSlug == "" {
		return nil, fmt.Errorf("org_slug is required - please re-register the runner")
	}

	if !cfg.UsesGRPC() {
		return nil, fmt.Errorf("gRPC configuration is required - please re-register the runner using 'runner register'")
	}

	// Create workspace manager
	ws, err := workspace.NewManager(cfg.WorkspaceRoot, cfg.GitConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Create gRPC/mTLS connection
	logger.Runner().Info("Using gRPC/mTLS connection", "endpoint", cfg.GRPCEndpoint)

	grpcConn := client.NewGRPCConnection(
		cfg.GRPCEndpoint,
		cfg.NodeID,
		cfg.OrgSlug,
		cfg.CertFile,
		cfg.KeyFile,
		cfg.CAFile,
		client.WithGRPCServerURL(cfg.ServerURL),
		client.WithGRPCRunnerVersion(cfg.Version),
	)

	// Check certificate validity before connecting
	certInfo, err := grpcConn.GetCertificateExpiryInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to check certificate: %w", err)
	}

	if certInfo.IsExpired {
		return nil, fmt.Errorf("certificate has expired on %s. Please reactivate the runner using:\n  runner reactivate --token <token>\nGet a reactivation token from the web UI", certInfo.ExpiresAt.Format("2006-01-02"))
	}

	if certInfo.NeedsRenewal {
		logger.Runner().Warn("Certificate expires soon",
			"days_until_expiry", certInfo.DaysUntilExpiry,
			"expires_at", certInfo.ExpiresAt.Format("2006-01-02"))
	}

	// Detect clipboard backend
	cb := clipboard.Detect()
	log.Info("Clipboard backend detected", "backend", cb.Name())

	// Create pod store
	podStore := NewInMemoryPodStore()

	r := &Runner{
		cfg:        cfg,
		conn:       grpcConn,
		workspace:  ws,
		clipboard:  cb,
		podStore:   podStore,
		autopilots: make(map[string]*autopilot.AutopilotController),
		stopChan:   make(chan struct{}),
	}

	// Create message handler and set it on connection
	r.messageHandler = NewRunnerMessageHandler(r, podStore, grpcConn)
	grpcConn.SetHandler(r.messageHandler)

	// Initialize optional enhanced components
	r.initEnhancedComponents(cfg)

	log.Info("Runner instance created successfully")
	return r, nil
}

// WithConnection sets a custom connection implementation (useful for testing).
func (r *Runner) WithConnection(conn client.Connection) *Runner {
	r.conn = conn
	r.messageHandler = NewRunnerMessageHandler(r, r.podStore, conn)
	conn.SetHandler(r.messageHandler)
	return r
}

// initEnhancedComponents initializes optional enhanced components based on config.
func (r *Runner) initEnhancedComponents(cfg *config.Config) {
	log := logger.Runner()
	log.Debug("Initializing enhanced components")

	// Initialize MCP manager
	r.mcpManager = mcp.NewManager()
	if cfg.MCPConfigPath != "" {
		if err := r.mcpManager.LoadConfig(cfg.MCPConfigPath); err != nil {
			log.Warn("Failed to load MCP config", "error", err)
		} else {
			log.Debug("MCP config loaded", "path", cfg.MCPConfigPath)
		}
	}

	// Initialize RPCClient for MCP over gRPC
	rpcClient := client.NewRPCClient(r.conn)
	if grpcConn, ok := r.conn.(*client.GRPCConnection); ok {
		grpcConn.SetRPCClient(rpcClient)
	}

	// Initialize MCP HTTP Server (started by Supervisor in Run())
	mcpPort := cfg.GetMCPPort()
	r.mcpServer = mcp.NewHTTPServer(rpcClient, mcpPort)
	r.mcpServer.SetStatusProvider(r)
	r.mcpServer.SetTerminalProvider(r)

	// Initialize Monitor (started by Supervisor in Run())
	r.agentMonitor = monitor.NewMonitor(5 * time.Second)
	log.Debug("Enhanced components initialized")
}

// AddService registers an additional suture.Service to be managed by the Supervisor.
// Must be called before Run().
func (r *Runner) AddService(svc suture.Service) {
	r.additionalServices = append(r.additionalServices, svc)
}

// Run starts the runner with a suture Supervisor tree and blocks until context is cancelled.
// All core components (gRPC connection, MCP server, Monitor, etc.) are managed by the Supervisor,
// which automatically restarts them on failure.
func (r *Runner) Run(ctx context.Context) error {
	log := logger.Runner()
	log.Info("Runner starting", "node_id", r.cfg.NodeID, "org", r.cfg.OrgSlug)

	// Store lifecycle context so message handlers can derive cancellable contexts
	// for long-running operations (e.g., git clone in OnCreatePod).
	r.runCtx = ctx

	// Create top-level Supervisor
	supervisor := suture.New("runner", suture.Spec{
		EventHook: func(e suture.Event) {
			log.Warn("Supervisor event", "event", e.String())
		},
		FailureThreshold: 5,
		FailureDecay:     60,
		FailureBackoff:   5 * time.Second,
	})

	// Register core services
	supervisor.Add(&lifecycle.ConnectionService{Conn: r.conn})

	if r.mcpServer != nil {
		supervisor.Add(&lifecycle.MCPServerService{Server: r.mcpServer})
	}
	if r.agentMonitor != nil {
		supervisor.Add(&lifecycle.MonitorService{Monitor: r.agentMonitor})
	}

	// Register Watchdog health monitor
	watchdogCfg := lifecycle.WatchdogConfig{
		Interval: 15 * time.Second,
	}
	// Wire up connection activity monitoring if GRPCConnection supports it
	if am, ok := r.conn.(lifecycle.ActivityMonitor); ok {
		watchdogCfg.ConnMonitor = am
	}
	supervisor.Add(lifecycle.NewWatchdogService(watchdogCfg))

	// Register additional services (Console, etc.)
	for _, svc := range r.additionalServices {
		supervisor.Add(svc)
	}

	// Supervisor.Serve() blocks until ctx is cancelled
	err := supervisor.Serve(ctx)

	log.Info("Shutting down runner...")
	r.stopAllPods()

	return err
}

// stopAllPods stops all active autopilots and pods during shutdown.
func (r *Runner) stopAllPods() {
	log := logger.Runner()

	// Stop all autopilot controllers first (they depend on pods)
	r.autopilotsMu.Lock()
	if len(r.autopilots) > 0 {
		log.Info("Stopping all autopilots", "count", len(r.autopilots))
	}
	for key, ac := range r.autopilots {
		log.Debug("Stopping autopilot", "autopilot_key", key)
		ac.Stop()
	}
	r.autopilots = make(map[string]*autopilot.AutopilotController)
	r.autopilotsMu.Unlock()

	// Stop all pods
	pods := r.podStore.All()
	if len(pods) > 0 {
		log.Info("Stopping all pods", "count", len(pods))
	}
	for _, pod := range pods {
		log.Debug("Stopping pod", "pod_key", pod.PodKey)
		pod.DisconnectRelay()
		pod.StopStateDetector()
		if pod.Aggregator != nil {
			pod.Aggregator.Stop()
		}
		if pod.Terminal != nil {
			pod.Terminal.Stop()
		}
		r.podStore.Delete(pod.PodKey)
	}
}

// IsDraining returns true if the runner is waiting for pods to finish before update.
func (r *Runner) IsDraining() bool {
	r.drainingMu.RLock()
	defer r.drainingMu.RUnlock()
	return r.draining
}

// SetDraining sets the draining state.
func (r *Runner) SetDraining(draining bool) {
	r.drainingMu.Lock()
	defer r.drainingMu.Unlock()
	r.draining = draining
	if draining {
		logger.Runner().Info("Entering draining mode - no new pods will be accepted")
	} else {
		logger.Runner().Info("Exiting draining mode - accepting pods again")
	}
}

// CanAcceptPod returns true if the runner can accept new pods.
func (r *Runner) CanAcceptPod() bool {
	r.drainingMu.RLock()
	draining := r.draining
	r.drainingMu.RUnlock()

	if draining {
		logger.Runner().Debug("Cannot accept pod: runner is draining")
		return false
	}

	currentCount := r.GetActivePodCount()
	if currentCount >= r.cfg.MaxConcurrentPods {
		logger.Runner().Debug("Cannot accept pod: max capacity reached",
			"current", currentCount, "max", r.cfg.MaxConcurrentPods)
		return false
	}

	return true
}

// GetActivePodCount returns the number of currently active pods.
func (r *Runner) GetActivePodCount() int {
	return r.podStore.Count()
}

// GetPodCounter returns a function that counts active pods.
func (r *Runner) GetPodCounter() func() int {
	return func() int {
		return r.GetActivePodCount()
	}
}

// GetRunContext returns the runner's lifecycle context.
// Returns context.Background() if Run() has not been called yet.
func (r *Runner) GetRunContext() context.Context {
	if r.runCtx != nil {
		return r.runCtx
	}
	return context.Background()
}

// Config returns the runner configuration.
func (r *Runner) Config() *config.Config {
	return r.cfg
}

// GetConnection returns the gRPC connection.
func (r *Runner) GetConnection() client.Connection {
	return r.conn
}

// Autopilot management methods

// GetAutopilot returns an AutopilotController by key.
func (r *Runner) GetAutopilot(key string) *autopilot.AutopilotController {
	r.autopilotsMu.RLock()
	defer r.autopilotsMu.RUnlock()
	return r.autopilots[key]
}

// AddAutopilot adds an AutopilotController.
func (r *Runner) AddAutopilot(ac *autopilot.AutopilotController) {
	r.autopilotsMu.Lock()
	defer r.autopilotsMu.Unlock()
	r.autopilots[ac.Key()] = ac
	logger.Runner().Debug("Autopilot added", "autopilot_key", ac.Key(), "pod_key", ac.PodKey())
}

// RemoveAutopilot removes an AutopilotController by key.
func (r *Runner) RemoveAutopilot(key string) {
	r.autopilotsMu.Lock()
	defer r.autopilotsMu.Unlock()
	delete(r.autopilots, key)
	logger.Runner().Debug("Autopilot removed", "autopilot_key", key)
}

// GetAutopilotByPodKey returns an AutopilotController by its associated pod key.
func (r *Runner) GetAutopilotByPodKey(podKey string) *autopilot.AutopilotController {
	r.autopilotsMu.RLock()
	defer r.autopilotsMu.RUnlock()
	for _, ac := range r.autopilots {
		if ac.PodKey() == podKey {
			return ac
		}
	}
	return nil
}

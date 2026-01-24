package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/mcp"
	"github.com/anthropics/agentsmesh/runner/internal/monitor"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// Runner is the main runner instance
type Runner struct {
	cfg       *config.Config
	conn      client.Connection         // gRPC connection interface
	workspace *workspace.Manager
	pods      map[string]*Pod
	mu        sync.RWMutex

	// Pod management
	podStore       PodStore               // Pod state management
	messageHandler *RunnerMessageHandler  // Message handler implementing client.MessageHandler

	// Enhanced components
	mcpManager      *mcp.Manager          // MCP server management
	mcpServer       *mcp.HTTPServer       // MCP HTTP Server for Claude Code
	claudeMonitor   *monitor.Monitor      // Claude CLI status monitoring

	// Update management
	draining       bool                   // True when waiting for pods to finish before update
	drainingMu     sync.RWMutex           // Protects draining flag

	// Channels for coordination
	stopChan chan struct{}
}

// Pod represents an active terminal pod
type Pod struct {
	ID               string
	PodKey           string
	AgentType        string
	RepositoryURL    string
	Branch           string
	WorktreePath     string
	Terminal         *terminal.Terminal
	VirtualTerminal  *terminal.VirtualTerminal  // Virtual terminal for state management and snapshots
	Aggregator       *terminal.SmartAggregator  // Output aggregator for adaptive frame rate
	RelayClient      *relay.Client              // WebSocket client for Relay connection
	relayMu          sync.RWMutex               // Protects RelayClient field
	StartedAt        time.Time
	Status           string              // Pod status - use statusMu for thread-safe access
	statusMu         sync.RWMutex        // Protects Status field
	TicketIdentifier string              // Ticket ID for worktree-based pods
	OnOutput         func([]byte)        // Output callback
	OnExit           func(int)           // Exit callback
	PTYLogger        *terminal.PTYLogger // PTY logger for debugging (optional)
}

// NewVirtualTerminal creates a new VirtualTerminal.
// This is a wrapper for terminal.NewVirtualTerminal to avoid importing terminal package in message_handler.
func NewVirtualTerminal(cols, rows, historyLimit int) *terminal.VirtualTerminal {
	return terminal.NewVirtualTerminal(cols, rows, historyLimit)
}

// SetStatus sets the pod status in a thread-safe manner
func (p *Pod) SetStatus(status string) {
	p.statusMu.Lock()
	defer p.statusMu.Unlock()
	p.Status = status
}

// GetStatus returns the pod status in a thread-safe manner
func (p *Pod) GetStatus() string {
	p.statusMu.RLock()
	defer p.statusMu.RUnlock()
	return p.Status
}

// SetRelayClient sets the relay client in a thread-safe manner
func (p *Pod) SetRelayClient(client *relay.Client) {
	p.relayMu.Lock()
	defer p.relayMu.Unlock()
	p.RelayClient = client
}

// GetRelayClient returns the relay client in a thread-safe manner
func (p *Pod) GetRelayClient() *relay.Client {
	p.relayMu.RLock()
	defer p.relayMu.RUnlock()
	return p.RelayClient
}

// HasRelayClient returns whether a relay client is connected
func (p *Pod) HasRelayClient() bool {
	p.relayMu.RLock()
	defer p.relayMu.RUnlock()
	return p.RelayClient != nil && p.RelayClient.IsConnected()
}

// DisconnectRelay disconnects and clears the relay client
func (p *Pod) DisconnectRelay() {
	p.relayMu.Lock()
	defer p.relayMu.Unlock()
	if p.RelayClient != nil {
		p.RelayClient.Stop()
		p.RelayClient = nil
	}
	// Clear aggregator relay output - will fall back to gRPC
	if p.Aggregator != nil {
		p.Aggregator.SetRelayOutput(nil)
	}
}

// PodStatus constants
const (
	PodStatusInitializing = "initializing"
	PodStatusRunning      = "running"
	PodStatusStopped      = "stopped"
	PodStatusFailed       = "failed"
)

// New creates a new runner instance
func New(cfg *config.Config) (*Runner, error) {
	// Load gRPC config (certificates)
	if err := cfg.LoadGRPCConfig(); err != nil {
		return nil, fmt.Errorf("failed to load gRPC config: %w - please register the runner first using 'runner register'", err)
	}

	// Load org slug from file if not in config
	if err := cfg.LoadOrgSlug(); err != nil {
		logger.Runner().Warn("Failed to load org slug", "error", err)
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
		client.WithGRPCServerURL(cfg.ServerURL), // For certificate renewal API calls
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

	// Create pod store
	podStore := NewInMemoryPodStore()

	r := &Runner{
		cfg:       cfg,
		conn:      grpcConn,
		workspace: ws,
		pods:      make(map[string]*Pod),
		podStore:  podStore,
		stopChan:  make(chan struct{}),
	}

	// Create message handler and set it on connection
	r.messageHandler = NewRunnerMessageHandler(r, podStore, grpcConn)
	grpcConn.SetHandler(r.messageHandler)

	// Initialize optional enhanced components
	r.initEnhancedComponents(cfg)

	return r, nil
}

// WithConnection sets a custom connection implementation (useful for testing).
func (r *Runner) WithConnection(conn client.Connection) *Runner {
	r.conn = conn
	// Re-create message handler with new connection
	r.messageHandler = NewRunnerMessageHandler(r, r.podStore, conn)
	conn.SetHandler(r.messageHandler)
	return r
}

// initEnhancedComponents initializes optional enhanced components based on config.
func (r *Runner) initEnhancedComponents(cfg *config.Config) {
	log := logger.Runner()

	// Initialize MCP manager (for legacy MCP config)
	r.mcpManager = mcp.NewManager()
	if cfg.MCPConfigPath != "" {
		if err := r.mcpManager.LoadConfig(cfg.MCPConfigPath); err != nil {
			log.Warn("Failed to load MCP config", "error", err)
		}
	}

	// Initialize and start MCP HTTP Server
	mcpPort := cfg.GetMCPPort()
	r.mcpServer = mcp.NewHTTPServer(cfg.ServerURL, mcpPort)
	go func() {
		log.Info("Starting MCP HTTP Server", "port", mcpPort)
		if err := r.mcpServer.Start(); err != nil {
			log.Warn("MCP HTTP Server failed", "error", err)
		}
	}()

	// Initialize Claude monitor for status tracking
	r.claudeMonitor = monitor.NewMonitor(5 * time.Second)
}

// Run starts the runner and blocks until context is cancelled
func (r *Runner) Run(ctx context.Context) error {
	log := logger.Runner()
	log.Info("Runner starting", "node_id", r.cfg.NodeID, "org", r.cfg.OrgSlug)

	// Start connection (includes connect, heartbeat, reconnect loop)
	r.conn.Start()
	defer r.conn.Stop()

	// Wait for shutdown
	<-ctx.Done()
	log.Info("Shutting down runner...")

	// Stop all pods
	r.stopAllPods()

	return nil
}

// stopAllPods stops all active pods
func (r *Runner) stopAllPods() {
	pods := r.podStore.All()
	for _, pod := range pods {
		// 1. Disconnect Relay first
		pod.DisconnectRelay()

		// 2. Stop aggregator to flush remaining output
		if pod.Aggregator != nil {
			pod.Aggregator.Stop()
		}

		// 3. Stop terminal
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
// Returns false if draining or at max capacity.
func (r *Runner) CanAcceptPod() bool {
	r.drainingMu.RLock()
	draining := r.draining
	r.drainingMu.RUnlock()

	if draining {
		return false
	}

	activePods := r.GetActivePodCount()
	return activePods < r.cfg.MaxConcurrentPods
}

// GetActivePodCount returns the number of currently active pods.
func (r *Runner) GetActivePodCount() int {
	return r.podStore.Count()
}

// GetPodCounter returns a function that counts active pods.
// This is used by the updater for graceful updates.
func (r *Runner) GetPodCounter() func() int {
	return func() int {
		return r.GetActivePodCount()
	}
}

// Config returns the runner configuration.
func (r *Runner) Config() *config.Config {
	return r.cfg
}

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
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/mcp"
	"github.com/anthropics/agentsmesh/runner/internal/monitor"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// Compile-time check: Runner implements MessageHandlerContext.
var _ MessageHandlerContext = (*Runner)(nil)

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
	mcpManager   *mcp.Manager
	mcpServer    *mcp.HTTPServer
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
		return nil, fmt.Errorf("failed to load gRPC config: %w - please register the runner first using 'agentsmesh-runner register'", err)
	}

	// Validate required configuration
	if cfg.OrgSlug == "" {
		return nil, fmt.Errorf("org_slug is required - please re-register the runner")
	}

	if !cfg.UsesGRPC() {
		return nil, fmt.Errorf("gRPC configuration is required - please re-register the runner using 'agentsmesh-runner register'")
	}

	// Create workspace manager
	ws, err := workspace.NewManager(cfg.WorkspaceRoot, cfg.GitConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Create gRPC/mTLS connection
	logger.Runner().Info("Using gRPC/mTLS connection", "endpoint", cfg.GRPCEndpoint)

	connOpts := []client.GRPCConnectionOption{
		client.WithGRPCServerURL(cfg.ServerURL),
		client.WithGRPCRunnerVersion(cfg.Version),
	}
	// Wire endpoint auto-discovery: when the runner detects a new gRPC endpoint
	// via the discovery API, persist it to the config file for future restarts.
	if cfg.ConfigFilePath != "" {
		cfgFile := cfg.ConfigFilePath
		connOpts = append(connOpts, client.WithGRPCEndpointChanged(func(newEndpoint string) error {
			return config.UpdateGRPCEndpointInFile(cfgFile, newEndpoint)
		}))
	}

	grpcConn := client.NewGRPCConnection(
		cfg.GRPCEndpoint,
		cfg.NodeID,
		cfg.OrgSlug,
		cfg.CertFile,
		cfg.KeyFile,
		cfg.CAFile,
		connOpts...,
	)

	// Check certificate validity before connecting
	certInfo, err := grpcConn.GetCertificateExpiryInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to check certificate: %w", err)
	}

	if certInfo.IsExpired {
		return nil, fmt.Errorf("certificate has expired on %s. Please reactivate the runner using:\n  agentsmesh-runner reactivate --token <token>\nGet a reactivation token from the web UI", certInfo.ExpiresAt.Format("2006-01-02"))
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

// GetConfig returns the runner configuration (implements MessageHandlerContext).
func (r *Runner) GetConfig() *config.Config {
	return r.cfg
}

// GetMCPServer returns the MCP server (implements MessageHandlerContext).
func (r *Runner) GetMCPServer() MCPServer {
	if r.mcpServer == nil {
		return nil
	}
	return r.mcpServer
}

// GetAgentMonitor returns the agent monitor (implements MessageHandlerContext).
func (r *Runner) GetAgentMonitor() AgentMonitor {
	if r.agentMonitor == nil {
		return nil
	}
	return r.agentMonitor
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

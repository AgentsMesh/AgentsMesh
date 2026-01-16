package runner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/mcp"
	"github.com/anthropics/agentsmesh/runner/internal/monitor"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// Runner is the main runner instance
type Runner struct {
	cfg       *config.Config
	conn      client.Connection         // New unified connection interface
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
	termManager     *terminal.Manager     // Enhanced terminal session management

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
	InitialPrompt    string
	Terminal         *terminal.Terminal
	StartedAt        time.Time
	Status           string              // Pod status - use statusMu for thread-safe access
	statusMu         sync.RWMutex        // Protects Status field
	TicketIdentifier string              // Ticket ID for worktree-based pods
	OnOutput         func([]byte)        // Output callback
	OnExit           func(int)           // Exit callback
	Forwarder        *PTYForwarder       // Output forwarder with backpressure
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

// PodStatus constants
const (
	PodStatusInitializing = "initializing"
	PodStatusRunning      = "running"
	PodStatusStopped      = "stopped"
	PodStatusFailed       = "failed"
)

// New creates a new runner instance
func New(cfg *config.Config) (*Runner, error) {
	// Load auth token from file if not in config
	if err := cfg.LoadAuthToken(); err != nil {
		log.Printf("Warning: failed to load auth token: %v", err)
	}

	// Load org slug from file if not in config
	if err := cfg.LoadOrgSlug(); err != nil {
		log.Printf("Warning: failed to load org slug: %v", err)
	}

	// Validate org slug is present
	if cfg.OrgSlug == "" {
		return nil, fmt.Errorf("org_slug is required - please re-register the runner")
	}

	// Create workspace manager
	ws, err := workspace.NewManager(cfg.WorkspaceRoot, cfg.GitConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Build WebSocket base URL from server URL (just convert http to ws, no path)
	wsURL := buildWebSocketBaseURL(cfg.ServerURL)

	// Create new ServerConnection with org slug
	conn := client.NewServerConnection(wsURL, cfg.NodeID, cfg.AuthToken, cfg.OrgSlug)

	// Create pod store
	podStore := NewInMemoryPodStore()

	r := &Runner{
		cfg:       cfg,
		conn:      conn,
		workspace: ws,
		pods:      make(map[string]*Pod),
		podStore:  podStore,
		stopChan:  make(chan struct{}),
	}

	// Create message handler and set it on connection
	r.messageHandler = NewRunnerMessageHandler(r, podStore, conn)
	conn.SetHandler(r.messageHandler)

	// Initialize optional enhanced components
	r.initEnhancedComponents(cfg)

	return r, nil
}

// buildWebSocketBaseURL converts HTTP URL to WebSocket base URL (no path).
// The ServerConnection will append the org-scoped path.
func buildWebSocketBaseURL(serverURL string) string {
	// Parse and convert http(s) to ws(s)
	if len(serverURL) > 5 && serverURL[:5] == "https" {
		return "wss" + serverURL[5:]
	}
	if len(serverURL) > 4 && serverURL[:4] == "http" {
		return "ws" + serverURL[4:]
	}
	return serverURL
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
	// Initialize MCP manager (for legacy MCP config)
	r.mcpManager = mcp.NewManager()
	if cfg.MCPConfigPath != "" {
		if err := r.mcpManager.LoadConfig(cfg.MCPConfigPath); err != nil {
			log.Printf("[runner] Warning: failed to load MCP config: %v", err)
		}
	}

	// Initialize and start MCP HTTP Server
	mcpPort := cfg.GetMCPPort()
	r.mcpServer = mcp.NewHTTPServer(cfg.ServerURL, mcpPort)
	go func() {
		log.Printf("[runner] Starting MCP HTTP Server on port %d", mcpPort)
		if err := r.mcpServer.Start(); err != nil {
			log.Printf("[runner] Warning: MCP HTTP Server failed: %v", err)
		}
	}()

	// Initialize Claude monitor for status tracking
	r.claudeMonitor = monitor.NewMonitor(5 * time.Second)

	// Initialize enhanced terminal manager
	defaultShell := cfg.DefaultShell
	if defaultShell == "" {
		defaultShell = "/bin/sh"
	}
	r.termManager = terminal.NewManager(defaultShell, cfg.WorkspaceRoot)
}

// Run starts the runner and blocks until context is cancelled
func (r *Runner) Run(ctx context.Context) error {
	log.Printf("Runner starting with node_id: %s (org: %s)", r.cfg.NodeID, r.cfg.OrgSlug)

	// Register with server if needed
	if r.cfg.AuthToken == "" {
		if r.cfg.RegistrationToken == "" {
			return fmt.Errorf("no auth_token or registration_token provided")
		}

		log.Println("Registering runner with server...")
		resp, err := r.register(ctx)
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}

		// Update connection with new auth token and org slug
		r.conn.SetAuthToken(resp.AuthToken)
		r.conn.SetOrgSlug(resp.OrgSlug)
		r.cfg.AuthToken = resp.AuthToken
		r.cfg.OrgSlug = resp.OrgSlug

		if err := r.cfg.SaveAuthToken(resp.AuthToken); err != nil {
			log.Printf("Warning: failed to save auth token: %v", err)
		}
		if err := r.cfg.SaveOrgSlug(resp.OrgSlug); err != nil {
			log.Printf("Warning: failed to save org slug: %v", err)
		}

		log.Printf("Registration successful (org: %s)", resp.OrgSlug)
	}

	// Start connection (includes connect, heartbeat, reconnect loop)
	r.conn.Start()
	defer r.conn.Stop()

	// Wait for shutdown
	<-ctx.Done()
	log.Println("Shutting down runner...")

	// Stop all pods
	r.stopAllPods()

	return nil
}

// register registers this runner with the server
func (r *Runner) register(ctx context.Context) (*client.RegistrationResponse, error) {
	req := client.RegistrationRequest{
		ServerURL:         r.cfg.ServerURL,
		NodeID:            r.cfg.NodeID,
		RegistrationToken: r.cfg.RegistrationToken,
		Description:       r.cfg.Description,
		MaxPods:           r.cfg.MaxConcurrentPods,
	}
	return client.Register(ctx, req)
}


// stopAllPods stops all active pods
func (r *Runner) stopAllPods() {
	pods := r.podStore.All()
	for _, pod := range pods {
		if pod.Terminal != nil {
			pod.Terminal.Stop()
		}
		r.podStore.Delete(pod.PodKey)
	}
}

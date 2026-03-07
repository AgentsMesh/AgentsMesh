// Package client provides gRPC connection management for Runner.
package client

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/tls/certprovider"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// GRPCProtocolVersion is the current gRPC protocol version.
const GRPCProtocolVersion = 2

// GRPCConnection manages the gRPC connection to the server with mTLS.
// Responsibilities: mTLS setup, bidirectional streaming, reconnection, message routing.
type GRPCConnection struct {
	// Connection configuration
	endpoint  string
	serverURL string // HTTP server URL for REST API calls (certificate renewal)
	nodeID    string
	orgSlug   string

	// mTLS certificate paths
	certFile string
	keyFile  string
	caFile   string

	// gRPC components
	conn   *grpc.ClientConn
	creds  credentials.TransportCredentials                                            // advancedtls credentials for hot-reload
	client runnerv1.RunnerServiceClient                                                // gRPC service client
	stream grpc.BidiStreamingClient[runnerv1.RunnerMessage, runnerv1.ServerMessage] // Bidirectional stream
	mu     sync.Mutex

	// Certificate providers for cleanup (prevent goroutine leaks)
	identityProvider certprovider.Provider
	rootProvider     certprovider.Provider

	// Message handling
	handler MessageHandler

	// Reconnection strategy
	reconnectStrategy *ReconnectStrategy

	// Heartbeat
	heartbeatInterval time.Duration

	// Initialization
	initTimeout     time.Duration
	initialized     bool
	availableAgents []string
	initResultCh    chan *runnerv1.InitializeResult

	// Runner info
	runnerVersion string
	mcpPort       int

	// Lifecycle - Priority-based channels for message sending
	// Control messages (heartbeat, pod events, OSC) have higher priority than agent status
	controlCh     chan *runnerv1.RunnerMessage // High priority: heartbeat, pod_created, pod_terminated, OSC, etc.
	terminalCh    chan *runnerv1.RunnerMessage // Low priority: agent_status (terminal output via Relay)
	stopCh        chan struct{}
	stopOnce    sync.Once
	reconnectCh chan struct{} // Signal to trigger reconnection

	// Stuck detection for writeLoop
	lastSendTime atomic.Int64

	// Recv liveness tracking — updated by readLoop on every successful Recv().
	// recvWatchdog triggers reconnect when no message arrives for 3× heartbeatInterval,
	// handling the half-dead connection case where the server closed the downstream
	// but stream.Recv() keeps blocking on the runner side.
	lastRecvTime atomic.Int64

	// Rate limiting for terminal output (bytes per second)
	// Default: 100KB/s to avoid overwhelming slow server connections
	terminalRateLimiter *rate.Limiter
	terminalRateLimit   int // bytes per second

	// Certificate renewal
	certRenewalCheckInterval time.Duration
	certExpiryWarningDays    int
	certRenewalDays          int // Days before expiry to trigger renewal (default 30)
	certUrgentDays           int // Days before expiry for urgent reconnection (default 7)

	// Heartbeat monitor for upstream liveness detection.
	// Created per-connection in runConnection(); nil before first connection.
	heartbeatMonitor *HeartbeatMonitor

	// RPCClient for MCP request-response over gRPC stream
	rpcClient *RPCClient

	// Agent probe for version detection and change tracking
	agentProbe *AgentProbe

	// Fatal error tracking - when set, connectionLoop should stop retrying
	fatalErr   error
	fatalErrMu sync.Mutex

	// onEndpointChanged is called when auto-discovery detects a new gRPC endpoint.
	// Implementations should persist the new endpoint to the config file.
	onEndpointChanged func(newEndpoint string) error

	// handlerWg tracks async handler goroutines (handleCreatePod, etc.)
	// so Stop() can wait for in-flight handlers to finish.
	handlerWg sync.WaitGroup

	// loopWg tracks the connectionLoop goroutine for clean shutdown.
	loopWg sync.WaitGroup
}

// NewGRPCConnection creates a new gRPC connection with mTLS.
func NewGRPCConnection(endpoint, nodeID, orgSlug, certFile, keyFile, caFile string, opts ...GRPCConnectionOption) *GRPCConnection {
	c := &GRPCConnection{
		endpoint:                 endpoint,
		nodeID:                   nodeID,
		orgSlug:                  orgSlug,
		certFile:                 certFile,
		keyFile:                  keyFile,
		caFile:                   caFile,
		heartbeatInterval:        30 * time.Second,
		initTimeout:              30 * time.Second,
		reconnectStrategy:        NewReconnectStrategy(5*time.Second, 5*time.Minute),
		controlCh:                make(chan *runnerv1.RunnerMessage, 100),  // Small buffer for control messages
		terminalCh:               make(chan *runnerv1.RunnerMessage, 1000), // Large buffer for terminal output
		stopCh:                   make(chan struct{}),
		reconnectCh:              make(chan struct{}, 1),
		initResultCh:             make(chan *runnerv1.InitializeResult, 1),
		runnerVersion:            "dev",
		mcpPort:                  19000,
		certRenewalCheckInterval: 24 * time.Hour,
		certExpiryWarningDays:    30,
		certRenewalDays:          30, // Renew 30 days before expiry
		certUrgentDays:           7,  // Urgent reconnection 7 days before expiry
		terminalRateLimit:        50 * 1024, // Default: 50KB/s (conservative for shared bandwidth)
		agentProbe:               NewAgentProbe(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize rate limiter if rate limit is set
	if c.terminalRateLimit > 0 {
		// rate.Limit is tokens per second, burst allows one maxSize message
		c.terminalRateLimiter = rate.NewLimiter(rate.Limit(c.terminalRateLimit), c.terminalRateLimit)
		logger.GRPC().Info("Terminal output rate limiting enabled",
			"rate_limit", fmt.Sprintf("%dKB/s", c.terminalRateLimit/1024))
	}

	return c
}

// SetHandler sets the message handler.
func (c *GRPCConnection) SetHandler(handler MessageHandler) {
	c.handler = handler
}

// SetOrgSlug sets the organization slug.
func (c *GRPCConnection) SetOrgSlug(orgSlug string) {
	c.mu.Lock()
	c.orgSlug = orgSlug
	c.mu.Unlock()
}

// GetOrgSlug returns the organization slug.
func (c *GRPCConnection) GetOrgSlug() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.orgSlug
}

// IsInitialized returns whether the connection has completed initialization.
func (c *GRPCConnection) IsInitialized() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.initialized
}

// GetAvailableAgents returns the list of available agents.
func (c *GRPCConnection) GetAvailableAgents() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.availableAgents
}

// Connect establishes a gRPC connection with mTLS using advancedtls for certificate hot-reloading.
func (c *GRPCConnection) Connect() error {
	// Close existing connection and certificate providers (important for reconnection)
	// This prevents resource leaks, TLS session conflicts, and goroutine leaks from pemfile providers
	c.mu.Lock()
	if c.conn != nil {
		logger.GRPC().Debug("Closing existing gRPC connection before reconnect")
		c.conn.Close()
		c.conn = nil
	}
	// Close old certificate providers to prevent goroutine leaks
	// Each pemfile.Provider runs background goroutines for file watching
	if c.identityProvider != nil {
		c.identityProvider.Close()
		c.identityProvider = nil
	}
	if c.rootProvider != nil {
		c.rootProvider.Close()
		c.rootProvider = nil
	}
	c.mu.Unlock()

	// Parse endpoint to extract host:port (remove scheme like grpcs://)
	dialTarget, err := parseGRPCEndpoint(c.endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse gRPC endpoint: %w", err)
	}

	// Create advancedtls credentials with file-based certificate reloading
	creds, err := c.createAdvancedTLSCredentials()
	if err != nil {
		return fmt.Errorf("failed to create TLS credentials: %w", err)
	}

	// gRPC dial options
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	// Connect to server
	conn, err := grpc.Dial(dialTarget, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	// Create gRPC service client
	client := runnerv1.NewRunnerServiceClient(conn)

	c.mu.Lock()
	c.conn = conn
	c.client = client
	c.creds = creds
	c.initialized = false
	c.mu.Unlock()

	logger.GRPC().Info("Connected to server with advancedtls", "endpoint", c.endpoint, "org", c.orgSlug)
	return nil
}

// Start starts the connection management loop.
func (c *GRPCConnection) Start() {
	logger.GRPC().Info("gRPC connection manager starting", "endpoint", c.endpoint)
	c.loopWg.Add(1)
	safego.Go("grpc-connection-loop", func() {
		defer c.loopWg.Done()
		c.connectionLoop()
	})
}

// Stop stops the connection and releases resources.
func (c *GRPCConnection) Stop() {
	c.stopOnce.Do(func() {
		logger.GRPC().Info("gRPC connection stopping")
		close(c.stopCh)

		// Wait for connectionLoop to exit before cleaning up resources
		c.loopWg.Wait()

		// Wait for in-flight async handlers (handleCreatePod, etc.)
		c.handlerWg.Wait()

		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		// Close certificate providers to prevent goroutine leaks
		if c.identityProvider != nil {
			c.identityProvider.Close()
			c.identityProvider = nil
		}
		if c.rootProvider != nil {
			c.rootProvider.Close()
			c.rootProvider = nil
		}
		c.mu.Unlock()
		logger.GRPC().Info("gRPC connection stopped")
	})
}

// Note: connectionLoop and runConnection are in grpc_connection_loop.go

// ==================== gRPC Error Handling ====================

// setFatalError records a fatal error that should stop reconnection attempts.
func (c *GRPCConnection) setFatalError(err error) {
	c.fatalErrMu.Lock()
	c.fatalErr = err
	c.fatalErrMu.Unlock()
}

// getFatalError returns the fatal error if one has been recorded.
func (c *GRPCConnection) getFatalError() error {
	c.fatalErrMu.Lock()
	defer c.fatalErrMu.Unlock()
	return c.fatalErr
}

// isFatalStreamError checks if a gRPC stream error is fatal (should not retry).
// Returns true and a user-friendly message if the error is fatal.
func isFatalStreamError(err error) (bool, string) {
	st, ok := status.FromError(err)
	if !ok {
		return false, ""
	}

	switch st.Code() {
	case codes.Unauthenticated:
		msg := st.Message()
		if strings.Contains(msg, "runner not found") {
			return true, "This runner has been deleted from the server. Please re-register with: agentsmesh-runner register --server <SERVER> --token <TOKEN> --force"
		}
		return true, "Authentication failed: " + msg + ". Please re-register with: agentsmesh-runner register --server <SERVER> --token <TOKEN> --force"

	case codes.PermissionDenied:
		msg := st.Message()
		if strings.Contains(msg, "runner is disabled") {
			return true, "This runner has been disabled by an administrator. Please contact your organization admin to re-enable it."
		}
		return true, "Permission denied: " + msg

	default:
		return false, ""
	}
}

// LastActivityTime returns the most recent send or recv timestamp.
// Used by the Watchdog health checker to detect stuck connections.
func (c *GRPCConnection) LastActivityTime() time.Time {
	sendNs := c.lastSendTime.Load()
	recvNs := c.lastRecvTime.Load()
	latest := max(sendNs, recvNs)
	if latest == 0 {
		return time.Time{}
	}
	return time.Unix(0, latest)
}

// Ensure GRPCConnection implements Connection interface.
var _ Connection = (*GRPCConnection)(nil)

// ==================== Helper Functions ====================

// parseGRPCEndpoint parses a gRPC endpoint URL and returns the host:port for dialing.
// Supports formats:
//   - grpcs://host:port -> host:port (TLS)
//   - grpc://host:port  -> host:port (plain)
//   - host:port         -> host:port (as-is)
func parseGRPCEndpoint(endpoint string) (string, error) {
	log := logger.GRPC()

	// If it doesn't contain a scheme, assume it's already host:port
	if !strings.Contains(endpoint, "://") {
		return endpoint, nil
	}

	// Parse as URL
	u, err := url.Parse(endpoint)
	if err != nil {
		log.Error("Invalid endpoint URL", "endpoint", endpoint, "error", err)
		return "", err
	}

	// Validate scheme
	switch u.Scheme {
	case "grpc", "grpcs":
		// Valid gRPC schemes
	default:
		log.Error("Unsupported gRPC scheme", "scheme", u.Scheme, "endpoint", endpoint)
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}

	// Return host:port
	if u.Host == "" {
		log.Error("Missing host in endpoint URL", "endpoint", endpoint)
		return "", fmt.Errorf("missing host in endpoint")
	}

	logger.GRPCTrace().Trace("Parsed gRPC endpoint", "endpoint", endpoint, "dial_target", u.Host)
	return u.Host, nil
}

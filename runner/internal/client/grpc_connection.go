// Package client provides gRPC connection management for Runner.
package client

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/tls/certprovider"
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

// Note: Connect, Start, Stop are in grpc_connection_connect.go
// Note: connectionLoop and runConnection are in grpc_connection_loop.go
// Note: State query methods are in grpc_connection_state.go
// Note: Error handling methods are in grpc_connection_error.go

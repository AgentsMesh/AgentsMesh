// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/tls/certprovider"
	"google.golang.org/grpc/credentials/tls/certprovider/pemfile"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/security/advancedtls"
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
	creds  credentials.TransportCredentials                            // advancedtls credentials for hot-reload
	client runnerv1.RunnerServiceClient                                // gRPC service client
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
	initTimeout       time.Duration
	initialized       bool
	availableAgents   []string
	initResultCh      chan *runnerv1.InitializeResult

	// Runner info
	runnerVersion string
	mcpPort       int

	// Lifecycle - Priority-based channels for message sending
	// Control messages (heartbeat, pod events) have higher priority than terminal output
	controlCh     chan *runnerv1.RunnerMessage // High priority: heartbeat, pod_created, pod_terminated, etc.
	terminalCh    chan *runnerv1.RunnerMessage // Low priority: terminal_output, agent_status
	stopCh        chan struct{}
	stopOnce      sync.Once
	reconnectOnce sync.Once      // Ensures only one reconnection attempt
	reconnectCh   chan struct{}  // Signal to trigger reconnection

	// Stuck detection for writeLoop
	lastSendTime atomic.Int64

	// Rate limiting for terminal output (bytes per second)
	// Default: 100KB/s to avoid overwhelming slow server connections
	terminalRateLimiter *rate.Limiter
	terminalRateLimit   int // bytes per second

	// Certificate renewal
	certRenewalCheckInterval time.Duration
	certExpiryWarningDays    int
	certRenewalDays          int  // Days before expiry to trigger renewal (default 30)
	certUrgentDays           int  // Days before expiry for urgent reconnection (default 7)
}

// GRPCConnectionOption is a functional option for GRPCConnection.
type GRPCConnectionOption func(*GRPCConnection)

// WithGRPCHeartbeatInterval sets the heartbeat interval.
func WithGRPCHeartbeatInterval(d time.Duration) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.heartbeatInterval = d
	}
}

// WithGRPCInitTimeout sets the initialization timeout.
func WithGRPCInitTimeout(d time.Duration) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.initTimeout = d
	}
}

// WithGRPCRunnerVersion sets the runner version.
func WithGRPCRunnerVersion(version string) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.runnerVersion = version
	}
}

// WithGRPCMCPPort sets the MCP port.
func WithGRPCMCPPort(port int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.mcpPort = port
	}
}

// WithGRPCTerminalRateLimit sets the terminal output rate limit in bytes per second.
// Default is 100KB/s. Set to 0 to disable rate limiting.
// Recommended: Set to ~80% of server upload bandwidth to leave room for control messages.
func WithGRPCTerminalRateLimit(bytesPerSecond int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.terminalRateLimit = bytesPerSecond
	}
}

// WithGRPCServerURL sets the HTTP server URL for REST API calls.
func WithGRPCServerURL(serverURL string) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.serverURL = serverURL
	}
}

// WithGRPCCertRenewalDays sets the days before expiry to trigger renewal.
func WithGRPCCertRenewalDays(days int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.certRenewalDays = days
	}
}

// WithGRPCCertUrgentDays sets the days before expiry for urgent reconnection.
func WithGRPCCertUrgentDays(days int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.certUrgentDays = days
	}
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
		runnerVersion:            "1.0.0",
		mcpPort:                  19000,
		certRenewalCheckInterval: 24 * time.Hour,
		certExpiryWarningDays:    30,
		certRenewalDays:          30,  // Renew 30 days before expiry
		certUrgentDays:           7,   // Urgent reconnection 7 days before expiry
		terminalRateLimit:        50 * 1024, // Default: 50KB/s (conservative for shared bandwidth)
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
	// Close existing connection if any (important for reconnection)
	// This prevents resource leaks and TLS session conflicts
	c.mu.Lock()
	if c.conn != nil {
		logger.GRPC().Debug("Closing existing gRPC connection before reconnect")
		c.conn.Close()
		c.conn = nil
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

// createAdvancedTLSCredentials creates TLS credentials using advancedtls package.
// This enables automatic certificate hot-reloading when certificate files are updated.
func (c *GRPCConnection) createAdvancedTLSCredentials() (credentials.TransportCredentials, error) {
	// Create identity certificate provider with file watching
	// This provider will automatically reload certificates when files change
	identityProvider, err := pemfile.NewProvider(pemfile.Options{
		CertFile:        c.certFile,
		KeyFile:         c.keyFile,
		RefreshDuration: 1 * time.Hour, // Check for file changes every hour
	})
	if err != nil {
		logger.GRPC().Warn("Failed to create pemfile identity provider, using fallback", "error", err)
		return c.createFallbackTLSCredentials()
	}

	// Create root certificate provider with file watching for CA
	// This allows CA certificate rotation if needed
	rootProvider, err := pemfile.NewProvider(pemfile.Options{
		RootFile:        c.caFile,
		RefreshDuration: 24 * time.Hour, // CA changes are rare, check daily
	})
	if err != nil {
		logger.GRPC().Warn("Failed to create pemfile root provider, using static CA", "error", err)
		// Fall back to static CA if file watching fails
		caCert, readErr := os.ReadFile(c.caFile)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", readErr)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		// Save identity provider for cleanup, rootProvider is nil in this path
		c.identityProvider = identityProvider
		c.rootProvider = nil

		// Use static root certificates
		options := &advancedtls.Options{
			IdentityOptions: advancedtls.IdentityCertificateOptions{
				IdentityProvider: identityProvider,
			},
			RootOptions: advancedtls.RootCertificateOptions{
				RootCertificates: caPool,
			},
			MinTLSVersion: tls.VersionTLS13,
			MaxTLSVersion: tls.VersionTLS13,
			// Only verify certificate chain, not hostname (server may use IP)
			VerificationType: advancedtls.CertVerification,
		}
		return advancedtls.NewClientCreds(options)
	}

	// Save providers for cleanup to prevent goroutine leaks
	c.identityProvider = identityProvider
	c.rootProvider = rootProvider

	// Create advancedtls client options with both providers
	options := &advancedtls.Options{
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			IdentityProvider: identityProvider,
		},
		RootOptions: advancedtls.RootCertificateOptions{
			RootProvider: rootProvider,
		},
		MinTLSVersion: tls.VersionTLS13,
		MaxTLSVersion: tls.VersionTLS13,
		// Only verify certificate chain, not hostname (server may use IP)
		VerificationType: advancedtls.CertVerification,
	}

	return advancedtls.NewClientCreds(options)
}

// createFallbackTLSCredentials creates standard TLS credentials as fallback.
// Used when advancedtls is not available or fails to initialize.
func (c *GRPCConnection) createFallbackTLSCredentials() (credentials.TransportCredentials, error) {
	// Load client certificate
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate (only trust AgentMesh CA, not system CAs)
	caCert, err := os.ReadFile(c.caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// TLS config - only trust our private CA
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}

	return credentials.NewTLS(tlsConfig), nil
}

// Start starts the connection management loop.
func (c *GRPCConnection) Start() {
	go c.connectionLoop()
}

// Stop stops the connection and releases resources.
func (c *GRPCConnection) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
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
	})
}

// sendControl queues a control message (high priority).
// Control messages include: heartbeat, pod_created, pod_terminated, pty_resized, error.
// These messages should never be blocked by terminal output.
// Returns error if connection is closed, stopped, or channel is full.
func (c *GRPCConnection) sendControl(msg *runnerv1.RunnerMessage) error {
	c.mu.Lock()
	if c.stream == nil {
		c.mu.Unlock()
		return fmt.Errorf("stream not connected")
	}
	c.mu.Unlock()

	select {
	case c.controlCh <- msg:
		return nil
	case <-c.stopCh:
		return fmt.Errorf("connection stopped")
	default:
		return fmt.Errorf("control buffer full")
	}
}

// sendTerminal queues a terminal message (low priority).
// Terminal messages include: terminal_output, agent_status.
// These messages are dropped silently if buffer is full (TUI frames are expendable).
// Returns nil even when dropped to avoid blocking callers.
//
// IMPORTANT: Messages are rejected before initialization completes.
// This prevents queue buildup during reconnection handshake, which could
// cause gRPC flow control to block the initialize_result response.
func (c *GRPCConnection) sendTerminal(msg *runnerv1.RunnerMessage) error {
	c.mu.Lock()
	stream := c.stream
	initialized := c.initialized
	c.mu.Unlock()

	// Reject messages before initialization completes
	// During reconnection, old Pods may still produce output, but sending it
	// before handshake completes can block the gRPC stream and cause deadlock
	if !initialized {
		logger.Terminal().Debug("sendTerminal: not initialized, dropping message")
		return nil // Silent drop, not an error
	}

	if stream == nil {
		logger.Terminal().Debug("sendTerminal: stream not connected")
		return fmt.Errorf("stream not connected")
	}

	select {
	case c.terminalCh <- msg:
		logger.Terminal().Debug("sendTerminal: message queued",
			"queue_len", len(c.terminalCh))
		return nil
	case <-c.stopCh:
		logger.Terminal().Debug("sendTerminal: connection stopped")
		return fmt.Errorf("connection stopped")
	default:
		// TUI frames are expendable - drop silently
		logger.GRPC().Debug("Terminal output dropped (queue full)",
			"queue_usage", c.QueueUsage())
		return nil
	}
}

// SendPodCreated sends a pod_created event to the server (control message).
func (c *GRPCConnection) SendPodCreated(podKey string, pid int32) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PodCreated{
			PodCreated: &runnerv1.PodCreatedEvent{
				PodKey: podKey,
				Pid:    pid,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendPodTerminated sends a pod_terminated event to the server (control message).
func (c *GRPCConnection) SendPodTerminated(podKey string, exitCode int32, errorMsg string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PodTerminated{
			PodTerminated: &runnerv1.PodTerminatedEvent{
				PodKey:       podKey,
				ExitCode:     exitCode,
				ErrorMessage: errorMsg,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendTerminalOutput sends terminal output to the server (terminal message).
// Non-blocking: drops silently if buffer is full (TUI frames are expendable).
func (c *GRPCConnection) SendTerminalOutput(podKey string, data []byte) error {
	// Apply rate limiting if enabled
	// This prevents overwhelming servers with limited bandwidth
	if c.terminalRateLimiter != nil {
		// WaitN blocks until we have enough tokens for this message
		// Use a context with timeout to avoid blocking forever
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := c.terminalRateLimiter.WaitN(ctx, len(data))
		cancel()
		if err != nil {
			// Rate limit timeout - drop the message (TUI frames are expendable)
			logger.Terminal().Debug("SendTerminalOutput: rate limit timeout, dropping",
				"pod_key", podKey, "data_len", len(data))
			return nil
		}
	}

	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_TerminalOutput{
			TerminalOutput: &runnerv1.TerminalOutputEvent{
				PodKey: podKey,
				Data:   data,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}

	logger.Terminal().Debug("SendTerminalOutput called",
		"pod_key", podKey, "data_len", len(data),
		"queue_len", len(c.terminalCh), "queue_cap", cap(c.terminalCh))

	return c.sendTerminal(msg)
}

// SendAgentStatus sends an agent status change event to the server (terminal message).
func (c *GRPCConnection) SendAgentStatus(podKey string, status string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_AgentStatus{
			AgentStatus: &runnerv1.AgentStatusEvent{
				PodKey: podKey,
				Status: status,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendTerminal(msg)
}

// SendPtyResized sends a PTY resize event to the server (control message).
func (c *GRPCConnection) SendPtyResized(podKey string, cols, rows int32) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PtyResized{
			PtyResized: &runnerv1.PtyResizedEvent{
				PodKey: podKey,
				Cols:   cols,
				Rows:   rows,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendError sends an error event to the server (control message).
func (c *GRPCConnection) SendError(podKey, code, message string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Error{
			Error: &runnerv1.ErrorEvent{
				PodKey:  podKey,
				Code:    code,
				Message: message,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendPodInitProgress sends a pod initialization progress event to the server (control message).
func (c *GRPCConnection) SendPodInitProgress(podKey, phase string, progress int32, message string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PodInitProgress{
			PodInitProgress: &runnerv1.PodInitProgressEvent{
				PodKey:   podKey,
				Phase:    phase,
				Progress: progress,
				Message:  message,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendRequestRelayToken sends a request for a new relay token to the server.
// This is called when the relay connection fails due to token expiration.
func (c *GRPCConnection) SendRequestRelayToken(podKey, sessionID, relayURL string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_RequestRelayToken{
			RequestRelayToken: &runnerv1.RequestRelayTokenEvent{
				PodKey:    podKey,
				SessionId: sessionID,
				RelayUrl:  relayURL,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// QueueLength returns the current terminal send queue length.
func (c *GRPCConnection) QueueLength() int {
	return len(c.terminalCh)
}

// QueueCapacity returns the terminal send queue capacity.
func (c *GRPCConnection) QueueCapacity() int {
	return cap(c.terminalCh)
}

// QueueUsage returns the terminal queue usage ratio (0.0 to 1.0).
// Used for monitoring queue pressure.
func (c *GRPCConnection) QueueUsage() float64 {
	return float64(len(c.terminalCh)) / float64(cap(c.terminalCh))
}

// drainTerminalQueue clears all pending messages in the terminal queue.
// Called before reconnection to discard stale terminal output.
// TUI frames are expendable - old frames are irrelevant after reconnection.
func (c *GRPCConnection) drainTerminalQueue() {
	drained := 0
	for {
		select {
		case <-c.terminalCh:
			drained++
		default:
			if drained > 0 {
				logger.GRPC().Info("Drained stale terminal queue before reconnection",
					"messages_dropped", drained)
			}
			return
		}
	}
}

// connectionLoop manages the connection lifecycle with auto-reconnection.
func (c *GRPCConnection) connectionLoop() {
	for {
		select {
		case <-c.stopCh:
			logger.GRPC().Info("Connection loop stopped")
			return
		default:
		}

		// Try to connect
		if err := c.Connect(); err != nil {
			delay := c.reconnectStrategy.NextDelay()
			logger.GRPC().Warn("Failed to connect, will retry",
				"attempt", c.reconnectStrategy.AttemptCount(),
				"error", err,
				"retry_in", delay)

			select {
			case <-c.stopCh:
				return
			case <-time.After(delay):
			}
			continue
		}

		// Reset reconnect strategy on successful connection
		c.reconnectStrategy.Reset()

		// Run the connection (blocks until disconnected)
		c.runConnection()

		// Check if we should stop
		select {
		case <-c.stopCh:
			return
		default:
		}

		// Wait before reconnecting
		logger.GRPC().Info("Connection closed, will attempt to reconnect")
		select {
		case <-c.stopCh:
			return
		case <-time.After(c.reconnectStrategy.CurrentInterval()):
		}
	}
}

// runConnection establishes the bidirectional stream and handles messages.
func (c *GRPCConnection) runConnection() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Clear terminal queue before establishing new connection
	// Old terminal output is stale after reconnection and would:
	// 1. Delay initialization by flooding the new stream
	// 2. Potentially cause immediate timeout if backend is slow
	// TUI frames are expendable - users will see fresh output after reconnection
	c.drainTerminalQueue()

	// Add org_slug to metadata for organization routing
	ctx = metadata.AppendToOutgoingContext(ctx, "x-org-slug", c.orgSlug)

	logger.GRPC().Debug("Establishing bidirectional stream", "org", c.orgSlug)

	// Create bidirectional stream
	stream, err := c.client.Connect(ctx)
	if err != nil {
		logger.GRPC().Error("Failed to establish stream", "error", err)
		return
	}

	c.mu.Lock()
	c.stream = stream
	c.mu.Unlock()

	logger.GRPC().Info("Bidirectional stream established")

	done := make(chan struct{})
	readLoopDone := make(chan struct{}) // Signal when readLoop exits

	// Start write loop
	go c.writeLoop(ctx, done)

	// IMPORTANT: Start read loop BEFORE initialization
	// The read loop must be running to receive the initialize_result response
	go c.readLoop(ctx, readLoopDone)

	// Perform initialization (blocks until handshake completes or times out)
	if err := c.performInitialization(ctx); err != nil {
		logger.GRPC().Error("Initialization failed", "error", err)
		close(done)
		return
	}

	// Start heartbeat loop (only after successful initialization)
	go c.heartbeatLoop(ctx, done)

	// Start certificate renewal checker
	go c.certRenewalChecker(ctx, done)

	// Monitor for reconnection signal (certificate renewal)
	go func() {
		select {
		case <-c.reconnectCh:
			logger.GRPC().Info("Reconnection requested for certificate renewal")
			cancel() // Cancel context to trigger reconnection
		case <-done:
			return
		case <-c.stopCh:
			return
		}
	}()

	// Wait for context cancellation, stop signal, or readLoop exit
	select {
	case <-ctx.Done():
		logger.GRPC().Debug("Context cancelled, closing connection")
	case <-c.stopCh:
		logger.GRPC().Debug("Stop signal received, closing connection")
	case <-readLoopDone:
		logger.GRPC().Debug("Read loop exited, closing connection")
	}

	// Clear stream to prevent sending to disconnected stream
	// This ensures sendTerminal/sendControl will reject new messages during reconnect
	c.mu.Lock()
	c.stream = nil
	c.mu.Unlock()

	// Signal other goroutines to stop
	close(done)
}

// sendWithTimeout sends a message with a timeout to prevent blocking forever.
// This is used for critical messages like initialization where we can't afford to block.
func (c *GRPCConnection) sendWithTimeout(msg *runnerv1.RunnerMessage, timeout time.Duration) error {
	c.mu.Lock()
	stream := c.stream
	c.mu.Unlock()

	if stream == nil {
		return fmt.Errorf("stream not connected")
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- stream.Send(msg)
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("send timed out after %v", timeout)
	}
}

// performInitialization performs the three-phase initialization handshake.
func (c *GRPCConnection) performInitialization(ctx context.Context) error {
	logger.GRPC().Debug("Starting initialization handshake...")

	// Use a shorter timeout for initialization messages (5s)
	// This ensures we fail fast if stream.Send() is blocking
	const initSendTimeout = 5 * time.Second

	// Phase 1: Send initialize request
	hostname, _ := os.Hostname()
	initReq := &runnerv1.InitializeRequest{
		ProtocolVersion: GRPCProtocolVersion,
		RunnerInfo: &runnerv1.RunnerInfo{
			Version:  c.runnerVersion,
			NodeId:   c.nodeID,
			McpPort:  int32(c.mcpPort),
			Os:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			Hostname: hostname,
		},
	}

	// Send initialize request via stream (with timeout)
	msg := &runnerv1.RunnerMessage{
		Payload:   &runnerv1.RunnerMessage_Initialize{Initialize: initReq},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := c.sendWithTimeout(msg, initSendTimeout); err != nil {
		return fmt.Errorf("failed to send initialize: %w", err)
	}
	logger.GRPC().Debug("Sent initialize request", "version", c.runnerVersion, "mcp_port", c.mcpPort)

	// Phase 2: Wait for initialize_result
	select {
	case result := <-c.initResultCh:
		logger.GRPC().Debug("Received initialize_result",
			"server_version", result.ServerInfo.Version,
			"agent_types", len(result.AgentTypes))

		// Phase 3: Check available agents and send initialized
		availableAgents := c.checkAvailableAgents(result.AgentTypes)
		c.mu.Lock()
		c.availableAgents = availableAgents
		c.mu.Unlock()

		// Send initialized confirmation via stream (with timeout)
		confirmMsg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Initialized{
				Initialized: &runnerv1.InitializedConfirm{
					AvailableAgents: availableAgents,
				},
			},
			Timestamp: time.Now().UnixMilli(),
		}
		if err := c.sendWithTimeout(confirmMsg, initSendTimeout); err != nil {
			return fmt.Errorf("failed to send initialized: %w", err)
		}
		logger.GRPC().Debug("Sent initialized", "available_agents", availableAgents)

		c.mu.Lock()
		c.initialized = true
		c.mu.Unlock()

		logger.GRPC().Info("Initialization completed successfully")
		return nil

	case <-time.After(c.initTimeout):
		return fmt.Errorf("timeout waiting for initialize_result after %v", c.initTimeout)

	case <-ctx.Done():
		return fmt.Errorf("context cancelled during initialization")

	case <-c.stopCh:
		return fmt.Errorf("connection stopped during initialization")
	}
}

// checkAvailableAgents checks which agents are available on this runner.
func (c *GRPCConnection) checkAvailableAgents(agentTypes []*runnerv1.AgentTypeInfo) []string {
	var available []string
	log := logger.GRPC()

	for _, agent := range agentTypes {
		if agent.Command == "" {
			log.Debug("Agent has no command defined, skipping", "agent", agent.Slug)
			continue
		}

		// Check if executable exists in PATH
		path, err := exec.LookPath(agent.Command)
		if err != nil {
			log.Debug("Agent command not found in PATH", "agent", agent.Slug, "command", agent.Command)
			continue
		}

		log.Debug("Agent command found", "agent", agent.Slug, "path", path)
		available = append(available, agent.Slug)
	}

	return available
}

// readLoop reads messages from the gRPC stream.
// The done channel is closed when the loop exits to notify other goroutines.
func (c *GRPCConnection) readLoop(ctx context.Context, done chan<- struct{}) {
	defer close(done) // Signal exit to other goroutines
	log := logger.GRPC()
	for {
		msg, err := c.stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Info("Stream ended (EOF)")
				return
			}
			if status.Code(err) == codes.Canceled {
				log.Debug("Stream cancelled")
			} else {
				log.Error("Stream error", "error", err)
			}
			return
		}
		c.handleServerMessage(msg)
	}
}

// handleServerMessage dispatches received server messages to appropriate handlers.
func (c *GRPCConnection) handleServerMessage(msg *runnerv1.ServerMessage) {
	switch payload := msg.Payload.(type) {
	case *runnerv1.ServerMessage_InitializeResult:
		c.handleInitializeResult(payload.InitializeResult)

	case *runnerv1.ServerMessage_CreatePod:
		c.handleCreatePod(payload.CreatePod)

	case *runnerv1.ServerMessage_TerminatePod:
		c.handleTerminatePod(payload.TerminatePod)

	case *runnerv1.ServerMessage_TerminalInput:
		c.handleTerminalInput(payload.TerminalInput)

	case *runnerv1.ServerMessage_TerminalResize:
		c.handleTerminalResize(payload.TerminalResize)

	case *runnerv1.ServerMessage_TerminalRedraw:
		c.handleTerminalRedraw(payload.TerminalRedraw)

	case *runnerv1.ServerMessage_SendPrompt:
		c.handleSendPrompt(payload.SendPrompt)

	case *runnerv1.ServerMessage_SubscribeTerminal:
		c.handleSubscribeTerminal(payload.SubscribeTerminal)

	case *runnerv1.ServerMessage_UnsubscribeTerminal:
		c.handleUnsubscribeTerminal(payload.UnsubscribeTerminal)

	default:
		logger.GRPC().Warn("Unknown server message type")
	}
}

// handleInitializeResult handles initialize_result from server.
func (c *GRPCConnection) handleInitializeResult(result *runnerv1.InitializeResult) {
	logger.GRPC().Debug("Received initialize_result", "version", result.ServerInfo.Version)
	// Convert to internal type and send to channel
	select {
	case c.initResultCh <- result:
	default:
		logger.GRPC().Warn("Initialize result channel full, dropping")
	}
}

// handleCreatePod handles create_pod command from server.
// Passes Proto type directly to handler for zero-copy message passing.
func (c *GRPCConnection) handleCreatePod(cmd *runnerv1.CreatePodCommand) {
	log := logger.GRPC()
	log.Info("Received create_pod", "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring create_pod")
		return
	}

	// Pass Proto type directly - no conversion needed
	if err := c.handler.OnCreatePod(cmd); err != nil {
		log.Error("Failed to create pod", "pod_key", cmd.PodKey, "error", err)
		c.sendError(cmd.PodKey, "create_pod_failed", err.Error())
	}
}

// handleTerminatePod handles terminate_pod command from server.
func (c *GRPCConnection) handleTerminatePod(cmd *runnerv1.TerminatePodCommand) {
	log := logger.GRPC()
	log.Info("Received terminate_pod", "pod_key", cmd.PodKey, "force", cmd.Force)
	if c.handler == nil {
		log.Warn("No handler set, ignoring terminate_pod")
		return
	}

	req := TerminatePodRequest{PodKey: cmd.PodKey}
	if err := c.handler.OnTerminatePod(req); err != nil {
		log.Error("Failed to terminate pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleTerminalInput handles terminal_input command from server.
func (c *GRPCConnection) handleTerminalInput(cmd *runnerv1.TerminalInputCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalInputRequest{
		PodKey: cmd.PodKey,
		Data:   cmd.Data, // gRPC uses native bytes, no encoding needed
	}
	if err := c.handler.OnTerminalInput(req); err != nil {
		logger.GRPC().Error("Failed to send terminal input to pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleTerminalResize handles terminal_resize command from server.
func (c *GRPCConnection) handleTerminalResize(cmd *runnerv1.TerminalResizeCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalResizeRequest{
		PodKey: cmd.PodKey,
		Cols:   uint16(cmd.Cols),
		Rows:   uint16(cmd.Rows),
	}
	if err := c.handler.OnTerminalResize(req); err != nil {
		logger.GRPC().Error("Failed to resize terminal for pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleTerminalRedraw handles terminal_redraw command from server.
// Uses resize +1/-1 trick to trigger terminal redraw for state recovery.
func (c *GRPCConnection) handleTerminalRedraw(cmd *runnerv1.TerminalRedrawCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalRedrawRequest{
		PodKey: cmd.PodKey,
	}
	if err := c.handler.OnTerminalRedraw(req); err != nil {
		logger.GRPC().Error("Failed to redraw terminal for pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleSendPrompt handles send_prompt command from server.
func (c *GRPCConnection) handleSendPrompt(cmd *runnerv1.SendPromptCommand) {
	logger.GRPC().Debug("Received send_prompt", "pod_key", cmd.PodKey)
	// TODO: Implement prompt sending when handler supports it
}

// handleSubscribeTerminal handles subscribe_terminal command from server.
// This notifies the Runner that a browser wants to observe the terminal via Relay.
func (c *GRPCConnection) handleSubscribeTerminal(cmd *runnerv1.SubscribeTerminalCommand) {
	log := logger.GRPC()
	log.Info("Received subscribe_terminal", "pod_key", cmd.PodKey, "relay_url", cmd.RelayUrl, "session_id", cmd.SessionId)
	if c.handler == nil {
		log.Warn("No handler set, ignoring subscribe_terminal")
		return
	}

	req := SubscribeTerminalRequest{
		PodKey:          cmd.PodKey,
		RelayURL:        cmd.RelayUrl,
		SessionID:       cmd.SessionId,
		RunnerToken:     cmd.RunnerToken,
		IncludeSnapshot: cmd.IncludeSnapshot,
		SnapshotHistory: cmd.SnapshotHistory,
	}
	if err := c.handler.OnSubscribeTerminal(req); err != nil {
		log.Error("Failed to subscribe terminal", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleUnsubscribeTerminal handles unsubscribe_terminal command from server.
// This notifies the Runner that all browsers have disconnected.
func (c *GRPCConnection) handleUnsubscribeTerminal(cmd *runnerv1.UnsubscribeTerminalCommand) {
	log := logger.GRPC()
	log.Info("Received unsubscribe_terminal", "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring unsubscribe_terminal")
		return
	}

	req := UnsubscribeTerminalRequest{
		PodKey: cmd.PodKey,
	}
	if err := c.handler.OnUnsubscribeTerminal(req); err != nil {
		log.Error("Failed to unsubscribe terminal", "pod_key", cmd.PodKey, "error", err)
	}
}

// sendError sends an error event back to the server (internal use, control message).
func (c *GRPCConnection) sendError(podKey, code, message string) {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Error{
			Error: &runnerv1.ErrorEvent{
				PodKey:  podKey,
				Code:    code,
				Message: message,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := c.sendControl(msg); err != nil {
		logger.GRPC().Error("Failed to send error", "error", err)
	}
}

// writeLoop sends messages to the gRPC stream with priority scheduling.
// Control messages (heartbeat, pod events) have higher priority than terminal output.
// This is the ONLY goroutine that calls stream.Send() to ensure thread-safety.
// Includes stuck detection: triggers reconnect if no successful send for 30 seconds.
func (c *GRPCConnection) writeLoop(ctx context.Context, done <-chan struct{}) {
	log := logger.GRPC()
	stuckTicker := time.NewTicker(10 * time.Second)
	defer stuckTicker.Stop()

	// Initialize last send time
	c.lastSendTime.Store(time.Now().UnixNano())

	for {
		select {
		case <-c.stopCh:
			return
		case <-done:
			return
		case <-ctx.Done():
			return

		case <-stuckTicker.C:
			// Stuck detection: if no successful send for 30 seconds, trigger reconnect
			lastSend := time.Unix(0, c.lastSendTime.Load())
			if time.Since(lastSend) > 30*time.Second {
				log.Error("WriteLoop stuck for 30s, triggering reconnect")
				c.triggerReconnect()
				return
			}

		case msg := <-c.controlCh:
			// Control messages have highest priority
			c.sendAndRecord(msg)

		default:
			// No control messages pending - use nested select for priority
			select {
			case <-c.stopCh:
				return
			case <-done:
				return
			case <-ctx.Done():
				return
			case msg := <-c.controlCh:
				// Double-check for control messages (priority)
				c.sendAndRecord(msg)
			case msg := <-c.terminalCh:
				// Process terminal messages when no control messages pending
				log.Debug("writeLoop: sending terminal message",
					"queue_len", len(c.terminalCh))
				c.sendAndRecord(msg)
			}
		}
	}
}

// sendAndRecord sends a message with a hard timeout to prevent writeLoop from blocking forever.
// If stream.Send() doesn't complete within sendTimeout, the message is abandoned and we continue.
// This prevents a slow/stuck stream.Send() from blocking all message processing.
//
// Key insight: gRPC stream.Send() can block indefinitely due to flow control.
// We cannot cancel it, but we can stop waiting and move on.
func (c *GRPCConnection) sendAndRecord(msg *runnerv1.RunnerMessage) {
	c.mu.Lock()
	stream := c.stream
	c.mu.Unlock()

	if stream == nil {
		logger.GRPC().Warn("sendAndRecord: stream is nil, dropping message")
		return
	}

	// Use a goroutine with timeout to prevent blocking forever
	// The send operation runs in a goroutine, and we wait with a timeout
	const sendTimeout = 5 * time.Second

	type sendResult struct {
		err     error
		elapsed time.Duration
	}

	resultCh := make(chan sendResult, 1)
	start := time.Now()

	go func() {
		err := stream.Send(msg)
		resultCh <- sendResult{err: err, elapsed: time.Since(start)}
	}()

	select {
	case result := <-resultCh:
		// Send completed (success or failure)
		if result.err != nil {
			logger.GRPC().Error("Failed to send message", "error", result.err, "elapsed", result.elapsed)
			return
		}

		// Log slow sends for diagnosis
		if result.elapsed > 100*time.Millisecond {
			logger.GRPC().Warn("Slow stream.Send()", "elapsed", result.elapsed,
				"terminal_queue", len(c.terminalCh))
		}

		// Update last successful send time
		c.lastSendTime.Store(time.Now().UnixNano())

	case <-time.After(sendTimeout):
		// Send timed out - the goroutine is still running but we move on
		// This prevents writeLoop from being blocked forever
		logger.GRPC().Error("stream.Send() timed out, abandoning message and triggering reconnect",
			"timeout", sendTimeout, "terminal_queue", len(c.terminalCh))

		// Trigger reconnect to recover from degraded connection
		// The stuck goroutine will eventually complete or error when stream is closed
		c.triggerReconnect()
	}
}

// heartbeatLoop sends periodic heartbeats.
func (c *GRPCConnection) heartbeatLoop(ctx context.Context, done <-chan struct{}) {
	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()

	// Send initial heartbeat
	c.sendHeartbeat()

	for {
		select {
		case <-c.stopCh:
			return
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.sendHeartbeat()
		}
	}
}

// sendHeartbeat sends a heartbeat message (control message - never blocked by terminal output).
func (c *GRPCConnection) sendHeartbeat() {
	var pods []*runnerv1.PodInfo

	if c.handler != nil {
		// Convert from internal PodInfo to proto PodInfo
		internalPods := c.handler.OnListPods()
		for _, p := range internalPods {
			pods = append(pods, &runnerv1.PodInfo{
				PodKey:      p.PodKey,
				Status:      p.Status,
				AgentStatus: p.ClaudeStatus,
			})
		}
	}

	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Heartbeat{
			Heartbeat: &runnerv1.HeartbeatData{
				NodeId: c.nodeID,
				Pods:   pods,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}

	logger.GRPC().Debug("Sending heartbeat", "pods", len(pods))

	if err := c.sendControl(msg); err != nil {
		logger.GRPC().Error("Failed to send heartbeat", "error", err)
	}
}

// certRenewalChecker periodically checks certificate expiry and triggers renewal.
func (c *GRPCConnection) certRenewalChecker(ctx context.Context, done <-chan struct{}) {
	ticker := time.NewTicker(c.certRenewalCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkCertificateExpiry()
		}
	}
}

// checkCertificateExpiry checks if the certificate needs renewal.
func (c *GRPCConnection) checkCertificateExpiry() {
	log := logger.GRPC()
	daysUntilExpiry, err := c.getCertDaysUntilExpiry()
	if err != nil {
		log.Error("Failed to check certificate expiry", "error", err)
		return
	}

	log.Debug("Certificate expiry check", "days_until_expiry", daysUntilExpiry)

	// Check if renewal is needed (30 days before expiry by default)
	if daysUntilExpiry <= float64(c.certRenewalDays) {
		log.Info("Certificate expires soon, triggering renewal", "days_until_expiry", daysUntilExpiry)

		// Attempt to renew certificate via REST API
		if err := c.renewCertificate(); err != nil {
			log.Error("Certificate renewal failed", "error", err)
			// Don't return here - still check for urgent reconnection
		} else {
			log.Info("Certificate renewed successfully, advancedtls will auto-reload")
		}
	}

	// Check if urgent reconnection is needed (7 days before expiry by default)
	// This ensures long-lived connections use the new certificate
	if daysUntilExpiry <= float64(c.certUrgentDays) {
		log.Warn("Certificate expiring urgently, triggering reconnection", "days_until_expiry", daysUntilExpiry)
		c.triggerReconnect()
	}
}

// getCertDaysUntilExpiry returns the number of days until the certificate expires.
func (c *GRPCConnection) getCertDaysUntilExpiry() (float64, error) {
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return 0, fmt.Errorf("failed to load certificate: %w", err)
	}

	if len(cert.Certificate) == 0 {
		return 0, fmt.Errorf("no certificate found")
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return 0, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return time.Until(x509Cert.NotAfter).Hours() / 24, nil
}

// IsCertificateExpired checks if the certificate has expired.
// Returns true if the certificate is expired, along with error details if any.
func (c *GRPCConnection) IsCertificateExpired() (bool, error) {
	daysUntilExpiry, err := c.getCertDaysUntilExpiry()
	if err != nil {
		return false, err
	}
	return daysUntilExpiry <= 0, nil
}

// CertificateExpiryInfo returns detailed information about certificate expiry.
type CertificateExpiryInfo struct {
	DaysUntilExpiry float64
	ExpiresAt       time.Time
	IsExpired       bool
	NeedsRenewal    bool
	NeedsUrgent     bool
}

// GetCertificateExpiryInfo returns detailed certificate expiry information.
func (c *GRPCConnection) GetCertificateExpiryInfo() (*CertificateExpiryInfo, error) {
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("no certificate found")
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	daysUntilExpiry := time.Until(x509Cert.NotAfter).Hours() / 24

	return &CertificateExpiryInfo{
		DaysUntilExpiry: daysUntilExpiry,
		ExpiresAt:       x509Cert.NotAfter,
		IsExpired:       daysUntilExpiry <= 0,
		NeedsRenewal:    daysUntilExpiry <= float64(c.certRenewalDays),
		NeedsUrgent:     daysUntilExpiry <= float64(c.certUrgentDays),
	}, nil
}

// renewCertificate calls the Backend REST API to renew the certificate.
// The new certificate is saved to files, and advancedtls will automatically reload them.
func (c *GRPCConnection) renewCertificate() error {
	if c.serverURL == "" {
		return fmt.Errorf("server URL not configured, cannot renew certificate")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call the renewal API using mTLS
	// Note: This requires a working TLS connection with the current certificate
	result, err := RenewCertificate(ctx, RenewalRequest{
		ServerURL: c.serverURL,
		CertFile:  c.certFile,
		KeyFile:   c.keyFile,
		CAFile:    c.caFile,
	})
	if err != nil {
		return fmt.Errorf("renewal API call failed: %w", err)
	}

	// Save new certificate to files
	// advancedtls FileWatcherCertificateProvider will detect the change and reload
	if err := os.WriteFile(c.certFile, []byte(result.Certificate), 0600); err != nil {
		return fmt.Errorf("failed to save new certificate: %w", err)
	}

	if err := os.WriteFile(c.keyFile, []byte(result.PrivateKey), 0600); err != nil {
		return fmt.Errorf("failed to save new private key: %w", err)
	}

	logger.GRPC().Info("New certificate saved",
		"expires_at", time.Unix(result.ExpiresAt, 0).Format(time.RFC3339))

	return nil
}

// triggerReconnect signals the connection loop to reconnect.
// This is used to ensure long-lived connections use the new certificate.
func (c *GRPCConnection) triggerReconnect() {
	select {
	case c.reconnectCh <- struct{}{}:
		logger.GRPC().Info("Reconnection triggered")
	default:
		// Reconnection already pending
	}
}

// ==================== gRPC Error Handling ====================

// isRetryableError returns true if the gRPC error is retryable.
func isRetryableError(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.Unavailable, codes.ResourceExhausted, codes.Aborted:
		return true
	default:
		return false
	}
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

	log.Debug("Parsed gRPC endpoint", "endpoint", endpoint, "dial_target", u.Host)
	return u.Host, nil
}
